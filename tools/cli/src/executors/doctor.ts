import { promises as fs } from "node:fs";
import path from "node:path";
import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { spawn } from "node:child_process";

const execFileAsync = promisify(execFile);

type CheckStatus = "pass" | "fail" | "warn";

interface DoctorCheck {
  id: string;
  title: string;
  status: CheckStatus;
  details: string;
  remediation?: string;
  autoFixed?: boolean;
}

interface ResolvedDoctorOptions {
  projectDir: string;
  outputPath: string;
  fix: boolean;
  requiredFlags: string[];
}

export interface DoctorExecutorOptions {
  projectDir?: string;
  outputPath?: string;
  fix?: boolean;
  requiredFlags?: string[];
}

export interface DoctorReport {
  generatedAt: string;
  projectDir: string;
  summary: {
    pass: number;
    fail: number;
    warn: number;
  };
  checks: DoctorCheck[];
}

export class DoctorExecutor {
  private readonly options: ResolvedDoctorOptions;

  constructor(options: DoctorExecutorOptions) {
    const projectDir = path.resolve(options.projectDir ?? process.cwd());
    const outputPath = path.resolve(
      options.outputPath ?? path.join(projectDir, ".doctor", "report.json"),
    );
    const requiredFlags =
      options.requiredFlags ?? ["PX_PLUGIN_SCAFFOLD_V2", "plugin-import-audit", "gitops-bootstrap"];

    this.options = {
      projectDir,
      outputPath,
      fix: options.fix ?? false,
      requiredFlags,
    };
  }

  async run(): Promise<DoctorReport> {
    const checks: DoctorCheck[] = [];
    checks.push(this.checkNodeVersion());
    checks.push(await this.checkGoVersion());
    checks.push(await this.checkBackendModule());
    checks.push(await this.checkWebAdminDependencies());
    checks.push(await this.checkGatewayCredentials());
    checks.push(this.checkFeatureFlags());

    const summary = this.buildSummary(checks);
    const report: DoctorReport = {
      generatedAt: new Date().toISOString(),
      projectDir: this.options.projectDir,
      summary,
      checks,
    };

    await fs.mkdir(path.dirname(this.options.outputPath), { recursive: true });
    await fs.writeFile(this.options.outputPath, JSON.stringify(report, null, 2), "utf-8");

    this.printConsoleSummary(report);
    return report;
  }

  private checkNodeVersion(): DoctorCheck {
    const version = process.version.replace(/^v/, "");
    const [major] = version.split(".").map((part) => Number.parseInt(part, 10));
    if (!Number.isFinite(major)) {
      return {
        id: "node-version",
        title: "Node.js version",
        status: "warn",
        details: `无法解析 Node 版本：${process.version}`,
        remediation: "请安装 Node.js 18 或更高版本。",
      };
    }
    if (major < 18) {
      return {
        id: "node-version",
        title: "Node.js version",
        status: "fail",
        details: `当前版本 ${process.version}，需要 >= 18`,
        remediation: "升级 Node.js 至 18+，或使用 nvm 切换。",
      };
    }
    return {
      id: "node-version",
      title: "Node.js version",
      status: "pass",
      details: `Node ${process.version}`,
    };
  }

  private async checkGoVersion(): Promise<DoctorCheck> {
    try {
      const { stdout } = await execFileAsync("go", ["version"]);
      const match = stdout.match(/go(\d+\.\d+)/);
      if (!match) {
        return {
          id: "go-version",
          title: "Go version",
          status: "warn",
          details: `无法解析 go version 输出：${stdout.trim()}`,
          remediation: "确认已安装 Go 1.24+ 并配置在 PATH。",
        };
      }
      const numeric = Number.parseFloat(match[1]);
      if (Number.isNaN(numeric) || numeric < 1.24) {
        return {
          id: "go-version",
          title: "Go version",
          status: "fail",
          details: `检测到 Go ${match[1]}，需要 >=1.24`,
          remediation: "下载 Go 1.24+ 并重新运行 doctor。",
        };
      }
      return {
        id: "go-version",
        title: "Go version",
        status: "pass",
        details: stdout.trim(),
      };
    } catch (error) {
      return {
        id: "go-version",
        title: "Go version",
        status: "fail",
        details: `执行 go version 失败：${(error as Error).message}`,
        remediation: "确认已安装 Go 并添加到 PATH。",
      };
    }
  }

  private async checkBackendModule(): Promise<DoctorCheck> {
    const goMod = path.join(this.options.projectDir, "backend", "go.mod");
    if (!(await pathExists(goMod))) {
      return {
        id: "backend-module",
        title: "Backend module",
        status: "warn",
        details: `未找到 ${goMod}，若此模板无后端可忽略。`,
      };
    }
    if (!this.options.fix) {
      return {
        id: "backend-module",
        title: "Backend module",
        status: "pass",
        details: "检测到 backend/go.mod",
      };
    }
    try {
      await this.runCommand("go", ["mod", "tidy"], path.dirname(goMod));
      return {
        id: "backend-module",
        title: "Backend module",
        status: "pass",
        details: "go mod tidy 执行完成",
        autoFixed: true,
      };
    } catch (error) {
      return {
        id: "backend-module",
        title: "Backend module",
        status: "fail",
        details: `go mod tidy 失败：${(error as Error).message}`,
        remediation: "手动运行 go mod tidy 并检查依赖。",
      };
    }
  }

  private async checkWebAdminDependencies(): Promise<DoctorCheck> {
    const webAdminDir = path.join(this.options.projectDir, "web-admin");
    const packageJSON = path.join(webAdminDir, "package.json");
    if (!(await pathExists(packageJSON))) {
      return {
        id: "web-admin-deps",
        title: "Web Admin dependencies",
        status: "warn",
        details: "未检测到 web-admin/package.json，若模板未包含前端可忽略。",
      };
    }

    const nodeModules = path.join(webAdminDir, "node_modules");
    const hasNodeModules = await pathExists(nodeModules);
    if (hasNodeModules) {
      return {
        id: "web-admin-deps",
        title: "Web Admin dependencies",
        status: "pass",
        details: "web-admin 已安装依赖",
      };
    }

    if (!this.options.fix) {
      return {
        id: "web-admin-deps",
        title: "Web Admin dependencies",
        status: "fail",
        details: "未运行 npm install",
        remediation: "在 web-admin 目录执行 npm install，或使用 px-plugin doctor --fix 自动安装。",
      };
    }

    try {
      await this.runCommand("npm", ["install"], webAdminDir);
      return {
        id: "web-admin-deps",
        title: "Web Admin dependencies",
        status: "pass",
        details: "npm install 已执行",
        autoFixed: true,
      };
    } catch (error) {
      return {
        id: "web-admin-deps",
        title: "Web Admin dependencies",
        status: "fail",
        details: `npm install 失败：${(error as Error).message}`,
        remediation: "手动进入 web-admin 目录执行 npm install。",
      };
    }
  }

  private checkFeatureFlags(): DoctorCheck {
    const missing = this.options.requiredFlags.filter((flag) => !process.env[flag]);
    if (missing.length === 0) {
      return {
        id: "feature-flags",
        title: "Feature flags",
        status: "pass",
        details: `检测到必需 Flag：${this.options.requiredFlags.join(", ")}`,
      };
    }
    return {
      id: "feature-flags",
      title: "Feature flags",
      status: "warn",
      details: `以下 Flag 未设置：${missing.join(", ")}`,
      remediation: "在 CLI 或 CI 环境中设置相应的 Feature Flag。",
    };
  }

  private buildSummary(checks: DoctorCheck[]) {
    return checks.reduce(
      (acc, check) => {
        acc[check.status] += 1;
        return acc;
      },
      { pass: 0, fail: 0, warn: 0 },
    );
  }

  private async runCommand(command: string, args: string[], cwd: string) {
    await new Promise<void>((resolve, reject) => {
      const child = spawn(command, args, { cwd, stdio: "inherit", env: process.env });
      child.on("error", reject);
      child.on("close", (code) => {
        if (code === 0) {
          resolve();
        } else {
          reject(new Error(`${command} ${args.join(" ")} exited with code ${code}`));
        }
      });
    });
  }

  private printConsoleSummary(report: DoctorReport) {
    console.log(`px-plugin doctor 在 ${report.projectDir} 生成报告：${this.options.outputPath}`);
    report.checks.forEach((check) => {
      const icon = check.status === "pass" ? "✅" : check.status === "warn" ? "⚠️" : "❌";
      console.log(`${icon} ${check.title} — ${check.details}`);
    });
  }

  private async checkGatewayCredentials(): Promise<DoctorCheck> {
    const skeletonEnvPath = path.join(this.options.projectDir, "skeleton", ".env.local");
    const envFile = await this.loadEnvFile(skeletonEnvPath);
    const envLabel = envFile.exists ? skeletonEnvPath : undefined;

    const gatewayBase = this.resolveConfigValue(
      ["PX_GATEWAY_BASE_URL"],
      envFile.values,
      envLabel,
    );
    const toolToken = this.resolveConfigValue(
      ["PX_TOOL_TOKEN", "PX_PLUGIN_TOOL_TOKEN"],
      envFile.values,
      envLabel,
    );
    const missing: string[] = [];
    if (!gatewayBase.value) missing.push("PX_GATEWAY_BASE_URL");
    if (!toolToken.value) missing.push("PX_TOOL_TOKEN/PX_PLUGIN_TOOL_TOKEN");

    if (missing.length > 0) {
      return {
        id: "gateway-credentials",
        title: "Gateway credentials",
        status: "fail",
        details: `缺少以下变量：${missing.join(", ")}`,
        remediation:
          "运行 `px-plugin login --manifest ./skeleton/plugin.yaml` 并将输出写入 skeleton/.env.local，或在宿主部署中注入对应环境变量。",
      };
    }

    const detailParts = [
      `Base=${gatewayBase.value} (${gatewayBase.source})`,
      "Tenant=from PX_TOOL_TOKEN.tid",
    ];
    let status: CheckStatus = "pass";
    let remediation: string | undefined;

    const expiry = parseTokenExpiry(toolToken.value);
    if (!expiry) {
      status = "warn";
      detailParts.push("无法解析 Token 过期时间");
      remediation = "确认当前 PX_TOOL_TOKEN 为 JWT，并重新执行 `px-plugin login` 获取最新凭证。";
    } else {
      const now = Date.now();
      const expiresAt = expiry.getTime();
      const hoursLeft = (expiresAt - now) / (1000 * 60 * 60);
      detailParts.push(`TokenExpires=${expiry.toISOString()}`);
      if (expiresAt <= now) {
        status = "fail";
        detailParts.push("Token 已过期");
        remediation =
          "重新执行 `px-plugin login --manifest ./skeleton/plugin.yaml` 刷新 PX_TOOL_TOKEN，并更新 skeleton/.env.local。";
      } else if (hoursLeft < 24 && status !== "fail") {
        status = "warn";
        detailParts.push("Token <24h 将过期");
        remediation =
          "PX_TOOL_TOKEN 将在 24 小时内过期，建议提前运行 `px-plugin login` 刷新凭证。";
      }
    }

    if (!envFile.exists && status === "pass") {
      status = "warn";
      detailParts.push("skeleton/.env.local 未找到");
      remediation =
        "建议创建 skeleton/.env.local 并写入 PX_GATEWAY_BASE_URL、PX_TOOL_TOKEN 以便团队共享配置。";
    }

    return {
      id: "gateway-credentials",
      title: "Gateway credentials",
      status,
      details: detailParts.join("; "),
      remediation,
    };
  }

  private async loadEnvFile(
    filePath: string,
  ): Promise<{ exists: boolean; values: Record<string, string> }> {
    if (!(await pathExists(filePath))) {
      return { exists: false, values: {} };
    }
    const content = await fs.readFile(filePath, "utf-8");
    const values: Record<string, string> = {};
    for (const rawLine of content.split(/\r?\n/)) {
      let line = rawLine.trim();
      if (!line || line.startsWith("#")) {
        continue;
      }
      if (line.startsWith("export ")) {
        line = line.slice(7).trim();
      }
      const idx = line.indexOf("=");
      if (idx === -1) {
        continue;
      }
      const key = line.slice(0, idx).trim();
      if (!key) {
        continue;
      }
      let value = line.slice(idx + 1).trim();
      if (!value) {
        values[key] = "";
        continue;
      }
      if (
        (value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'"))
      ) {
        value = value.slice(1, -1);
      }
      values[key] = value;
    }
    return { exists: true, values };
  }

  private resolveConfigValue(
    keys: string[],
    envFileValues: Record<string, string>,
    envFileLabel?: string,
  ): { value: string; source: string } {
    for (const key of keys) {
      const raw = process.env[key];
      if (typeof raw === "string" && raw.trim() !== "") {
        return { value: raw.trim(), source: `env:${key}` };
      }
    }
    if (envFileLabel) {
      for (const key of keys) {
        const raw = envFileValues[key];
        if (typeof raw === "string" && raw.trim() !== "") {
          return { value: raw.trim(), source: `${envFileLabel}:${key}` };
        }
      }
    }
    return { value: "", source: "unset" };
  }
}

async function pathExists(target: string): Promise<boolean> {
  try {
    await fs.access(target);
    return true;
  } catch {
    return false;
  }
}

function parseTokenExpiry(token: string | undefined): Date | null {
  if (!token) {
    return null;
  }
  const parts = token.split(".");
  if (parts.length < 2) {
    return null;
  }
  try {
    const payload = Buffer.from(parts[1], "base64url").toString("utf-8");
    const data = JSON.parse(payload) as Record<string, unknown>;
    const fromNumeric = parseNumericTimestamp(data["exp"] ?? data["expires"]);
    if (fromNumeric) {
      return fromNumeric;
    }
    const fromString =
      parseStringTimestamp(data["expires_at"]) ?? parseStringTimestamp(data["expiresAt"]);
    if (fromString) {
      return fromString;
    }
  } catch {
    return null;
  }
  return null;
}

function parseNumericTimestamp(value: unknown): Date | null {
  if (value === null || value === undefined) {
    return null;
  }
  let num: number | null = null;
  if (typeof value === "number") {
    num = value;
  } else if (typeof value === "string") {
    const parsed = Number.parseFloat(value);
    if (!Number.isNaN(parsed)) {
      num = parsed;
    }
  }
  if (num === null || Number.isNaN(num)) {
    return null;
  }
  if (num > 1_000_000_000_000) {
    return new Date(num);
  }
  return new Date(num * 1000);
}

function parseStringTimestamp(value: unknown): Date | null {
  if (typeof value !== "string") {
    return null;
  }
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }
  const iso = Date.parse(trimmed);
  if (!Number.isNaN(iso)) {
    return new Date(iso);
  }
  return parseNumericTimestamp(trimmed);
}
