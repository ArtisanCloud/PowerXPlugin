import { promises as fs } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawn } from "node:child_process";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const DEFAULT_TEMPLATE_ROOT = path.resolve(__dirname, "../../../..", "scaffold", "templates");
const DEFAULT_VERSION = "0.1.0";
const DEFAULT_GO_VERSION = "1.24";
const pluginIdPattern = /^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$/;

type TemplateData = Record<string, string>;

interface ResolvedOptions {
  pluginId: string;
  targetDir: string;
  templateRoot: string;
  modulePath: string;
  preset?: string;
  force: boolean;
  installDeps: boolean;
  sbomPath: string;
  publishManifestPath: string;
  version: string;
  goVersion: string;
}

export interface ScaffoldExecutorOptions {
  pluginId: string;
  targetDir?: string;
  modulePath?: string;
  preset?: string;
  templateRoot?: string;
  force?: boolean;
  installDeps?: boolean;
  sbomPath?: string;
  publishManifestPath?: string;
  version?: string;
  goVersion?: string;
}

export interface ScaffoldResult {
  pluginId: string;
  targetDir: string;
  filesCreated: string[];
  manifestPath: string;
  publishManifestPath: string;
  sbomPath?: string;
  preset?: string;
  instructions: string[];
}

export class ScaffoldExecutor {
  private readonly options: ResolvedOptions;
  private readonly templateData: TemplateData;
  private readonly filesCreated = new Set<string>();

  constructor(options: ScaffoldExecutorOptions) {
    this.options = this.resolveOptions(options);
    this.templateData = this.buildTemplateData();
  }

  async run(): Promise<ScaffoldResult> {
    await this.ensureTemplateRoot();
    await this.prepareTargetDirectory();
    await this.copyTemplates();
    const manifestPath = path.join(this.options.targetDir, "plugin.yaml");
    if (!(await pathExists(manifestPath))) {
      throw new Error(`expected manifest at ${manifestPath}, but it was not generated`);
    }
    const publishManifestPath = await this.ensurePublishManifest();
    let sbomPath: string | undefined;
    if (this.options.sbomPath) {
      sbomPath = await this.writeSbom();
    }
    if (this.options.installDeps) {
      await this.installDependencies();
    }
    return {
      pluginId: this.options.pluginId,
      targetDir: this.options.targetDir,
      filesCreated: Array.from(this.filesCreated).sort(),
      manifestPath,
      publishManifestPath,
      sbomPath,
      preset: this.options.preset,
      instructions: this.buildNextSteps(),
    };
  }

  private resolveOptions(options: ScaffoldExecutorOptions): ResolvedOptions {
    const pluginId = options.pluginId?.trim();
    if (!pluginId) {
      throw new Error("pluginId is required");
    }
    if (!pluginIdPattern.test(pluginId)) {
      throw new Error(`invalid plugin id: ${pluginId} (expected pattern ${pluginIdPattern.source})`);
    }
    const targetDir = path.resolve(options.targetDir ?? path.join(process.cwd(), pluginId));
    const templateRoot = path.resolve(options.templateRoot ?? DEFAULT_TEMPLATE_ROOT);
    const pluginSlug = derivePluginSlug(pluginId);
    const modulePath = options.modulePath?.trim() || `github.com/ArtisanCloud/PowerXPlugin/plugins/${pluginSlug}`;
    const sbomPath = path.resolve(options.sbomPath ?? path.join(targetDir, "reports", "sbom.json"));
    const publishManifestPath = path.resolve(options.publishManifestPath ?? path.join(targetDir, "publish.yml"));

    return {
      pluginId,
      targetDir,
      templateRoot,
      modulePath,
      preset: options.preset,
      force: options.force ?? false,
      installDeps: options.installDeps ?? false,
      sbomPath,
      publishManifestPath,
      version: options.version ?? DEFAULT_VERSION,
      goVersion: options.goVersion ?? DEFAULT_GO_VERSION,
    };
  }

  private buildTemplateData(): TemplateData {
    const pluginName = derivePluginName(this.options.pluginId);
    const pluginSlug = derivePluginSlug(this.options.pluginId);
    return {
      PluginID: this.options.pluginId,
      PluginName: pluginName,
      PluginSlug: pluginSlug,
      BackendModulePath: `${this.options.modulePath}/backend`,
      GoVersion: this.options.goVersion,
      Version: this.options.version,
    };
  }

  private async ensureTemplateRoot(): Promise<void> {
    if (!(await pathExists(this.options.templateRoot))) {
      throw new Error(`template root ${this.options.templateRoot} does not exist. Run npm run sync:templates first.`);
    }
  }

  private async prepareTargetDirectory(): Promise<void> {
    try {
      const stat = await fs.stat(this.options.targetDir);
      if (!stat.isDirectory()) {
        throw new Error(`target ${this.options.targetDir} exists and is not a directory`);
      }
      const entries = await fs.readdir(this.options.targetDir);
      if (entries.length > 0 && !this.options.force) {
        throw new Error(`target directory ${this.options.targetDir} is not empty. Use force=true to overwrite.`);
      }
    } catch (err) {
      if ((err as NodeJS.ErrnoException).code === "ENOENT") {
        await fs.mkdir(this.options.targetDir, { recursive: true });
        return;
      }
      if (!this.options.force) {
        throw err;
      }
      await fs.mkdir(this.options.targetDir, { recursive: true });
    }
  }

  private async copyTemplates(): Promise<void> {
    await this.copyDirectory(this.options.templateRoot, this.options.targetDir);
  }

  private async copyDirectory(source: string, destination: string): Promise<void> {
    const entries = await fs.readdir(source, { withFileTypes: true });
    for (const entry of entries) {
      if (entry.name === ".DS_Store" || entry.name === "Thumbs.db") {
        continue;
      }
      const sourcePath = path.join(source, entry.name);
      const destinationName = entry.name.endsWith(".tmpl") ? entry.name.slice(0, -5) : entry.name;
      const destinationPath = path.join(destination, destinationName);

      if (entry.isDirectory()) {
        await fs.mkdir(destinationPath, { recursive: true });
        await this.copyDirectory(sourcePath, destinationPath);
        continue;
      }

      if (entry.name.endsWith(".tmpl")) {
        const content = await fs.readFile(sourcePath, "utf-8");
        const rendered = renderTemplate(content, this.templateData);
        await this.writeFile(destinationPath, rendered);
      } else {
        await this.copyFile(sourcePath, destinationPath);
      }
    }
  }

  private async writeFile(target: string, contents: string): Promise<void> {
    await fs.mkdir(path.dirname(target), { recursive: true });
    if (!this.options.force && (await pathExists(target))) {
      throw new Error(`refusing to overwrite existing file ${target}. Enable force to continue.`);
    }
    await fs.writeFile(target, contents, "utf-8");
    this.filesCreated.add(this.relativeToTarget(target));
  }

  private async copyFile(source: string, destination: string): Promise<void> {
    await fs.mkdir(path.dirname(destination), { recursive: true });
    if (!this.options.force && (await pathExists(destination))) {
      throw new Error(`refusing to overwrite existing file ${destination}. Enable force to continue.`);
    }
    await fs.copyFile(source, destination);
    this.filesCreated.add(this.relativeToTarget(destination));
  }

  private async ensurePublishManifest(): Promise<string> {
    if (await pathExists(this.options.publishManifestPath)) {
      return this.options.publishManifestPath;
    }
    const content = [
      "# Generated by px-plugin init",
      `plugin: ${this.options.pluginId}`,
      "channels:",
      "  - stable",
      "  - beta",
      "rollout:",
      "  strategy: canary",
      "  batches:",
      "    - percentage: 20",
      "      wait: 10m",
      "    - percentage: 80",
      "      wait: 20m",
      "rollback:",
      "  strategy: automatic",
      "  maxFailingTenants: 5",
      "",
    ].join("\n");
    await this.writeFile(this.options.publishManifestPath, content);
    return this.options.publishManifestPath;
  }

  private async writeSbom(): Promise<string> {
    const payload = {
      schema: "powerx.plugin.sbom@v1",
      generatedAt: new Date().toISOString(),
      plugin: {
        id: this.options.pluginId,
        version: this.options.version,
        module: this.options.modulePath,
      },
      templatePreset: this.options.preset ?? "fullstack-go-nuxt",
      files: Array.from(this.filesCreated).sort(),
    };
    await fs.mkdir(path.dirname(this.options.sbomPath), { recursive: true });
    await fs.writeFile(this.options.sbomPath, JSON.stringify(payload, null, 2), "utf-8");
    this.filesCreated.add(this.relativeToTarget(this.options.sbomPath));
    return this.options.sbomPath;
  }

  private async installDependencies(): Promise<void> {
    const backendDir = path.join(this.options.targetDir, "backend");
    const webAdminDir = path.join(this.options.targetDir, "web-admin");
    if (await pathExists(backendDir)) {
      await runCommand("go", ["mod", "tidy"], backendDir);
    }
    if (await pathExists(webAdminDir)) {
      await runCommand("npm", ["install"], webAdminDir);
    }
  }

  private buildNextSteps(): string[] {
    return [
      `cd ${this.options.targetDir}`,
      "go work sync",
      "cd backend && go run ./cmd/plugin",
      "cd ../web-admin && npm run dev",
    ];
  }

  private relativeToTarget(target: string): string {
    return path.relative(this.options.targetDir, target) || ".";
  }
}

function derivePluginName(pluginId: string): string {
  const parts = pluginId
    .split(/[.\-_]/)
    .filter(Boolean)
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1));
  if (parts.length === 0) {
    return pluginId;
  }
  if (parts[0].toLowerCase() === "com" && parts.length > 1) {
    return parts.slice(1).join(" ");
  }
  return parts.join(" ");
}

function derivePluginSlug(pluginId: string): string {
  return pluginId
    .split(/[.\s]+/)
    .filter(Boolean)
    .map((segment) => segment.replace(/[^a-z0-9]+/gi, "-"))
    .join("-")
    .toLowerCase();
}

function renderTemplate(template: string, data: TemplateData): string {
  return template.replace(/\{\{\s*\.([A-Za-z0-9_]+)\s*\}\}/g, (_match, key: string) => {
    if (data[key]) {
      return data[key];
    }
    return _match;
  });
}

async function pathExists(target: string): Promise<boolean> {
  try {
    await fs.access(target);
    return true;
  } catch {
    return false;
  }
}

async function runCommand(command: string, args: string[], cwd: string): Promise<void> {
  await new Promise<void>((resolve, reject) => {
    const child = spawn(command, args, { cwd, stdio: "inherit" });
    child.on("close", (code) => {
      if (code === 0) {
        resolve();
      } else {
        reject(new Error(`${command} ${args.join(" ")} exited with code ${code}`));
      }
    });
    child.on("error", reject);
  });
}
