import { promises as fs } from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import { parse } from "yaml";

type ImportStatus = "pass" | "manual_review" | "blocked";

interface PolicySource {
  type: string;
  domains?: string[];
  maxRepoSizeMb?: number;
  maxSizeMb?: number;
  checksumRequired?: boolean;
}

interface Policy {
  version: string;
  updatedAt?: string;
  sources: { allow: PolicySource[] };
  licenses: { allow: string[]; deny: string[] };
  approvals?: { requiredFor?: string[]; contacts?: string[] };
  audit?: { webhook?: string };
}

interface ImportIssue {
  code: string;
  severity: "warn" | "error";
  message: string;
}

export interface ImportGuardOptions {
  sourcePath: string;
  type?: string;
  provider?: string;
  license?: string;
  projectDir?: string;
  policyPath?: string;
  outputPath?: string;
}

interface ResolvedImportOptions {
  sourcePath: string;
  type: string;
  provider: string;
  license?: string;
  projectDir: string;
  policyPath: string;
  outputPath: string;
}

export interface ImportReport {
  generatedAt: string;
  status: ImportStatus;
  issues: ImportIssue[];
  policyVersion: string;
  source: {
    path: string;
    type: string;
    provider: string;
    sizeBytes?: number;
    checksum?: string;
  };
  nextSteps: string[];
  auditWebhook?: string;
}

export class ImportGuardExecutor {
  private readonly options: ResolvedImportOptions;

  constructor(options: ImportGuardOptions) {
    if (!options.sourcePath) {
      throw new Error("sourcePath is required");
    }
    const projectDir = path.resolve(options.projectDir ?? process.cwd());
    const policyPath = path.resolve(
      options.policyPath ?? path.join(projectDir, "config", "compliance", "external_source_policy.yaml"),
    );
    const outputPath = path.resolve(
      options.outputPath ?? path.join(projectDir, ".compliance", "import-report.json"),
    );

    this.options = {
      sourcePath: path.resolve(options.sourcePath),
      type: options.type ?? "tarball",
      provider: options.provider ?? "github.com",
      license: options.license,
      projectDir,
      policyPath,
      outputPath,
    };
  }

  async run(): Promise<ImportReport> {
    const policy = await this.loadPolicy();
    const stats = await this.describeSource();
    const evaluation = this.evaluatePolicy(policy, stats);

    const report: ImportReport = {
      generatedAt: new Date().toISOString(),
      status: evaluation.status,
      issues: evaluation.issues,
      policyVersion: policy.version,
      source: stats,
      nextSteps: this.buildNextSteps(evaluation.status, policy),
      auditWebhook: policy.audit?.webhook,
    };

    await fs.mkdir(path.dirname(this.options.outputPath), { recursive: true });
    await fs.writeFile(this.options.outputPath, JSON.stringify(report, null, 2), "utf-8");
    return report;
  }

  private async loadPolicy(): Promise<Policy> {
    const raw = await fs.readFile(this.options.policyPath, "utf-8");
    const parsed = parse(raw) as Policy;
    if (!parsed?.version) {
      throw new Error("invalid external source policy: missing version");
    }
    return parsed;
  }

  private async describeSource() {
    const source: ImportReport["source"] = {
      path: this.options.sourcePath,
      type: this.options.type,
      provider: this.options.provider,
    };
    try {
      const stat = await fs.stat(this.options.sourcePath);
      if (stat.isFile()) {
        source.sizeBytes = stat.size;
        source.checksum = await this.computeChecksum(this.options.sourcePath);
      }
    } catch {
      // ignore if source is remote git (no local file yet)
    }
    return source;
  }

  private evaluatePolicy(policy: Policy, source: ImportReport["source"]) {
    const issues: ImportIssue[] = [];
    let status: ImportStatus = "pass";

    const license = (this.options.license ?? "").trim();
    if (license && policy.licenses?.deny?.map((l) => l.toLowerCase()).includes(license.toLowerCase())) {
      issues.push({
        code: "LICENSE_DENYLIST",
        severity: "error",
        message: `许可证 ${license} 在禁止列表中`,
      });
      status = "blocked";
    } else if (!license && policy.approvals?.requiredFor?.includes("unknown_license")) {
      issues.push({
        code: "UNKNOWN_LICENSE",
        severity: "warn",
        message: "未提供第三方源码许可证，需要人工审核",
      });
      status = status === "blocked" ? status : "manual_review";
    }

    const matchingSource = policy.sources?.allow?.find((entry) => entry.type == this.options.type);
    if (!matchingSource) {
      issues.push({
        code: "UNSUPPORTED_SOURCE",
        severity: "error",
        message: `策略不允许导入 ${this.options.type} 类型源码`,
      });
      status = "blocked";
    } else {
      if (matchingSource.domains && !matchingSource.domains.includes(this.options.provider)) {
        issues.push({
          code: "DOMAIN_NOT_ALLOWED",
          severity: "error",
          message: `域名 ${this.options.provider} 未在 allowlist 中`,
        });
        status = "blocked";
      }
      if (matchingSource.maxSizeMb && source.sizeBytes) {
        const sizeMb = source.sizeBytes / (1024 * 1024);
        if (sizeMb > matchingSource.maxSizeMb) {
          issues.push({
            code: "SIZE_EXCEEDED",
            severity: "warn",
            message: `包体 ${sizeMb.toFixed(2)}MB 超过上限 ${matchingSource.maxSizeMb}MB`,
          });
          if (policy.approvals?.requiredFor?.includes("exceeds_max_size")) {
            status = status === "blocked" ? status : "manual_review";
          }
        }
      }
      if (matchingSource.checksumRequired && !source.checksum) {
        issues.push({
          code: "MISSING_CHECKSUM",
          severity: "warn",
          message: "未生成校验和，无法证明包体完整性",
        });
        if (policy.approvals?.requiredFor?.includes("missing_checksum")) {
          status = status === "blocked" ? status : "manual_review";
        }
      }
    }

    return { status, issues };
  }

  private buildNextSteps(status: ImportStatus, policy: Policy): string[] {
    if (status === "pass") {
      return ["记录 import-report.json 并继续执行 px-plugin init / doctor 流程。"];
    }
    if (status === "manual_review") {
      const contacts = policy.approvals?.contacts?.join(", ") ?? "compliance 团队";
      return [
        "提交审核：附上 import-report.json 与 SBOM。",
        `发送邮件至 ${contacts} 申请第三方导入豁免。`,
      ];
    }
    return [
      "阻断导入：修复许可证/域名/包体大小问题。",
      "确保外部依赖满足策略后重新运行 px-plugin import。",
    ];
  }

  private async computeChecksum(target: string) {
    const hash = crypto.createHash("sha256");
    const data = await fs.readFile(target);
    hash.update(data);
    return hash.digest("hex");
  }
}
