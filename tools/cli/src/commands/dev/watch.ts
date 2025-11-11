import path from "node:path";
import fs from "node:fs";
import { SessionClient } from "../../runtime/hotreload/session";

interface WatchOptions {
  entry: string;
  tenant?: string;
  ignore?: string[];
  devApiBaseUrl: string;
}

export async function runDevWatch(options: WatchOptions) {
  const session = new SessionClient({ baseUrl: options.devApiBaseUrl });
  const manifest = await loadManifest(options.entry);
  const registerResponse = await session.register({ manifest, tenant: options.tenant });
  console.log(`Dev session ready: ${registerResponse.sessionId}`);
  await watchFiles(options.entry, async (changedFiles) => {
    await session.reload({
      sessionId: registerResponse.sessionId,
      reloadToken: registerResponse.reloadToken,
      changedFiles,
    });
  });
}

async function loadManifest(entry: string) {
  const manifestPath = path.resolve(entry, "manifest.json");
  const data = await fs.promises.readFile(manifestPath, "utf-8");
  return JSON.parse(data);
}

async function watchFiles(entry: string, onChange: (files: string[]) => Promise<void>) {
  const watcher = fs.watch(entry, { recursive: true });
  const pending = new Set<string>();
  let timer: NodeJS.Timeout | undefined;
  watcher.on("change", (event, fileName) => {
    if (!fileName) {
      return;
    }
    pending.add(fileName.toString());
    if (timer) {
      clearTimeout(timer);
    }
    timer = setTimeout(async () => {
      const files = Array.from(pending).map((f) => path.join(entry, f));
      pending.clear();
      await onChange(files);
    }, 250);
  });
}
