import type { Page, Locator } from '@playwright/test';

const resolvePluginId = () =>
  process.env.POWERX_PLUGIN_ID ||
  process.env.NUXT_PUBLIC_POWERX_PLUGIN_ID ||
  'com.powerx.plugins.base';

const normalizePath = (path: string) => (path.startsWith('/') ? path : `/${path}`);

export const pluginAdminBasePath = () => `/_p/${resolvePluginId()}/admin`;

const resolveBaseUrl = () => process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:3000';

const ensureZhLocale = async (page: Page) => {
  const base = resolveBaseUrl();
  let url: URL | null = null;
  try {
    url = new URL(base);
  } catch {
    try {
      url = new URL(`http://${base}`);
    } catch {
      url = null;
    }
  }
  if (!url) return;
  await page.context().addCookies([
    {
      name: 'px_lang',
      value: 'zh',
      url: url.origin,
    },
  ]);
};

export async function gotoWithFallback(page: Page, path: string, ready?: Locator) {
  await ensureZhLocale(page);
  const normalized = normalizePath(path);
  const candidates = [normalized, `${pluginAdminBasePath()}${normalized}`];
  const failures: string[] = [];

  for (const candidate of candidates) {
    await page.goto(candidate);
    if (!ready) return;
    try {
      await ready.first().waitFor({ state: 'visible', timeout: 10_000 });
      return;
    } catch {
      failures.push(candidate);
      // try next candidate
    }
  }

  if (ready) {
    throw new Error(
      `gotoWithFallback failed: ready locator not visible after candidates: ${failures.join(', ')}`
    );
  }
}

export async function seedAuthStorage(page: Page, opts?: { insidePowerX?: boolean }) {
  const now = Date.now();
  const expiresAt = now + 3600 * 1000;
  const refreshToken = 'e2e-refresh-token';

  await page.addInitScript(
    ({ expiresAt, insidePowerX, refreshToken }) => {
      window.localStorage.setItem('access_token', 'e2e-access-token');
      // delegated/insidePowerX 模式必须有 refresh_token 才会通过 auth middleware
      window.localStorage.setItem('refresh_token', refreshToken);
      window.localStorage.setItem('token_type', 'Bearer');
      window.localStorage.setItem('expires_in', '3600');
      window.localStorage.setItem('scope', 'access');
      window.localStorage.setItem('expires_at', String(expiresAt));
      document.cookie = `token=e2e-access-token; path=/`;
    },
    { expiresAt, insidePowerX: opts?.insidePowerX === true, refreshToken }
  );
}
