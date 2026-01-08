import { defineConfig, devices } from '@playwright/test';
import os from 'node:os';
import { chromium } from 'playwright';

const ensureMacArm64HostPlatform = () => {
  if (process.env.PLAYWRIGHT_HOST_PLATFORM_OVERRIDE) return;
  if (os.platform() !== 'darwin' || os.arch() !== 'arm64') return;

  // Work around Playwright hostPlatform detection relying on CPU model string
  // that can miss Apple Silicon on some systems, which then makes it look for
  // x64 browser bundles that don't exist.
  const ver = os.release().split('.').map((a) => parseInt(a, 10));
  const darwinMajor = ver[0] ?? 0;
  const LAST_STABLE_MACOS_MAJOR_VERSION = 15;
  const macMajor = Math.min(darwinMajor - 9, LAST_STABLE_MACOS_MAJOR_VERSION);
  if (!Number.isFinite(macMajor) || macMajor <= 0) return;

  process.env.PLAYWRIGHT_HOST_PLATFORM_OVERRIDE = `mac${macMajor}-arm64`;
};

ensureMacArm64HostPlatform();

const useMacArm64Chromium = os.platform() === 'darwin' && os.arch() === 'arm64' && !process.env.CI;
const macArm64ExecutablePath = useMacArm64Chromium ? chromium.executablePath() : undefined;

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  retries: process.env.CI ? 2 : 0,
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:3031',
    locale: 'zh-CN',
    trace: 'on-first-retry',
    launchOptions: macArm64ExecutablePath ? { executablePath: macArm64ExecutablePath } : undefined,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
