import { expect, test, type Page } from '@playwright/test';
import { gotoWithFallback, seedAuthStorage } from './_utils';

const loginAsLocalAdmin = async (page: Page) => {
  await page.goto('/users/login?redirect=/intro');
  await page.locator('#identifier').fill('admin@local.test');
  await page.locator('#password').fill('S3cret!!');
  await page.getByRole('button', { name: /登录|Sign in|Sign In/ }).click();
  await expect(page).toHaveURL(/\/intro$/);
};

const mockPowerXAgents = async (page: Page) => {
  let sessionSeq = 100;
  await page.route('**/api/v1/plugin/agent/agents**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          items: [
            {
              id: 1,
              uuid: '00000000-0000-4000-8000-000000000001',
              key: 'system.default',
              name: 'System Default Agent',
              status: 'active',
            },
            {
              id: 2,
              uuid: '00000000-0000-4000-8000-000000000002',
              key: 'mediax.skill',
              name: 'MediaX Skill Agent',
              status: 'active',
            },
          ],
        },
      }),
    });
  });
  await page.route('**/api/v1/plugin/agent/sessions', async (route) => {
    const requestBody = route.request().postDataJSON();
    const agentUUID = String(requestBody.agent_uuid || '');
    sessionSeq += 1;
    const sessionID = agentUUID.endsWith('0002')
      ? `10000000-0000-4000-8000-000000000${String(200 + sessionSeq).slice(-3)}`
      : `10000000-0000-4000-8000-000000000${String(sessionSeq).slice(-3)}`;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          id: sessionSeq,
          uuid: sessionID,
          session_id: sessionID,
          status: 'active',
        },
      }),
    });
  });
};

test.describe('Agent Skill Bridge local chat', () => {
  test('is reachable from the sidebar menu', async ({ page }) => {
    await loginAsLocalAdmin(page);

    await expect(page.getByText('PowerX底座能力')).toBeVisible();
    await expect(page.getByRole('link', { name: /framework功能测试|Framework Function Test/ })).toBeVisible();
    await expect(page.getByRole('link', { name: /PowerX 能力调试|PowerX Capability Lab/ })).toBeVisible();
    await page.getByRole('link', { name: /Agent Chat 调试|Agent Chat Debug/ }).click();

    await expect(page).toHaveURL(/\/agent-skill-bridge$/);
    await expect(page.getByTestId('agent-chat-send')).toBeVisible();
  });

  test('sends local chat requests through the plugin Agent SSE proxy', async ({ page }) => {
    await seedAuthStorage(page);
    await mockPowerXAgents(page);
    const seen: string[] = [];
    const forbidden: string[] = [];
    // Negative assertions: the plugin web UI must not call PowerX core endpoints directly.
    await page.route('**/api/v1/admin/agents**', async (route) => {
      forbidden.push(route.request().url());
      await route.abort();
    });
    await page.route('**/api/v1/agents/sessions**', async (route) => {
      forbidden.push(route.request().url());
      await route.abort();
    });
    await page.route('**/api/v1/agents/stream/sse**', async (route) => {
      forbidden.push(route.request().url());
      await route.abort();
    });
    await page.route('**/api/v1/plugin/agent/stream/sse**', async (route) => {
      seen.push(route.request().url());
      const url = new URL(route.request().url());
      expect(url.searchParams.get('agent_id')).toBe('00000000-0000-4000-8000-000000000001');
      expect(url.searchParams.get('session_id')).toMatch(/^10000000-0000-4000-8000-000000000\d{3}$/);
      expect(url.searchParams.get('q')).toContain('篮球模板');
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: [
          'event: intent',
          'data: {"type":"intent","tasks":[{"task_id":"skill-1","node_kind":"skill","node_ref":"mediax.video_rebuilder.cn"}]}',
          '',
          'event: plan',
          'data: {"type":"plan","plan":{"tasks":[{"task_id":"skill-1","node_kind":"skill","node_ref":"mediax.video_rebuilder.cn"}]}}',
          '',
          'event: node_start',
          'data: {"type":"node_start","node_kind":"skill","node_ref":"mediax.video_rebuilder.cn","status":"running"}',
          '',
          'event: node_end',
          'data: {"type":"node_end","node_kind":"skill","node_ref":"mediax.video_rebuilder.cn","status":"completed"}',
          '',
          'event: final',
          'data: {"type":"final","trace_id":"trace_local_debug","payload":{"message":"已创建视频重构任务"}}',
          '',
        ].join('\n')
      });
    });

    await gotoWithFallback(
      page,
      '/agent-skill-bridge',
      page.getByTestId('agent-chat-send')
    );
    await expect(page.getByTestId('agent-chat-agent')).toHaveValue('00000000-0000-4000-8000-000000000001');
    await expect(page.getByTestId('agent-chat-session')).toHaveValue(/^10000000-0000-4000-8000-000000000\d{3}$/);
    await expect(page.getByTestId('agent-chat-proxy')).toHaveValue('/api/v1/plugin/agent/stream/sse');
    await page.getByTestId('agent-chat-send').click();

    await expect(page.getByTestId('agent-chat-status')).toHaveText(/ended/);
    await expect(page.getByTestId('agent-chat-timeline')).toContainText('intent');
    await expect(page.getByTestId('agent-chat-timeline')).toContainText('node_end');
    await expect(page.getByTestId('agent-chat-final')).toContainText('已创建视频重构任务');
    await expect(page.getByTestId('agent-chat-events')).toContainText('final');
    expect(seen).toHaveLength(1);
    expect(forbidden).toEqual([]);
  });

  test('can switch the active PowerX agent', async ({ page }) => {
    await seedAuthStorage(page);
    await mockPowerXAgents(page);
    const seen: string[] = [];
    await page.route('**/api/v1/plugin/agent/stream/sse**', async (route) => {
      seen.push(route.request().url());
      const url = new URL(route.request().url());
      expect(url.searchParams.get('agent_id')).toBe('00000000-0000-4000-8000-000000000002');
      expect(url.searchParams.get('session_id')).toMatch(/^10000000-0000-4000-8000-000000000\d{3}$/);
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: 'event: final\ndata: {"type":"final","payload":{"message":"MediaX agent ready"}}\n\n'
      });
    });

    await gotoWithFallback(
      page,
      '/agent-skill-bridge',
      page.getByTestId('agent-chat-send')
    );
    await page.getByTestId('agent-chat-agent-select').click();
    await page.getByRole('option', { name: 'MediaX Skill Agent' }).click();
    await expect(page.getByTestId('agent-chat-agent')).toHaveValue('00000000-0000-4000-8000-000000000002');
    await expect(page.getByTestId('agent-chat-session')).toHaveValue(/^10000000-0000-4000-8000-000000000\d{3}$/);
    await expect(page.getByText('MediaX Skill Agent').first()).toBeVisible();

    await page.getByTestId('agent-chat-send').click();

    await expect(page.getByTestId('agent-chat-final')).toContainText('MediaX agent ready');
    expect(seen).toHaveLength(1);
  });

  test('does not call plugin business APIs for smart task simulation', async ({ page }) => {
    await seedAuthStorage(page);
    await mockPowerXAgents(page);
    const forbidden: string[] = [];
    // Negative assertions: smart task simulation must enter PowerX Agent, not plugin executors or core SSE directly.
    await page.route('**/api/v1/agents/stream/sse**', async (route) => {
      forbidden.push(route.request().url());
      await route.abort();
    });
    await page.route('**/api/v1/plugin/agent/stream/sse**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: 'event: end\ndata: {"type":"end","trace_id":"trace_local_debug"}\n\n'
      });
    });

    await gotoWithFallback(
      page,
      '/agent-skill-bridge',
      page.getByTestId('agent-chat-send')
    );
    await page.getByTestId('agent-chat-send').click();

    await expect(page.getByTestId('agent-chat-status')).toHaveText(/ended/);
    expect(forbidden).toEqual([]);
  });

  test('creates a real PowerX agent session through the plugin backend proxy', async ({ page }) => {
    await seedAuthStorage(page);
    await mockPowerXAgents(page);
    const created: string[] = [];
    const forbidden: string[] = [];

    await page.route('**/api/v1/agents/sessions**', async (route) => {
      forbidden.push(route.request().url());
      await route.abort();
    });
    await page.route('**/api/v1/plugin/agent/sessions', async (route) => {
      const requestBody = route.request().postDataJSON();
      created.push(String(requestBody.agent_uuid || requestBody.agent_id || ''));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 303,
            uuid: '10000000-0000-4000-8000-000000000303',
            session_id: '10000000-0000-4000-8000-000000000303',
            status: 'active',
          },
        }),
      });
    });

    await gotoWithFallback(
      page,
      '/agent-skill-bridge',
      page.getByTestId('agent-chat-send')
    );
    await page.getByRole('button', { name: '新建会话' }).click();

    await expect(page.getByTestId('agent-chat-session')).toHaveValue('10000000-0000-4000-8000-000000000303');
    expect(created).toContain('00000000-0000-4000-8000-000000000001');
    expect(forbidden).toEqual([]);
  });

  test('keeps newly created sessions in the recent session list', async ({ page }) => {
    await seedAuthStorage(page);
    await mockPowerXAgents(page);

    await gotoWithFallback(
      page,
      '/agent-skill-bridge',
      page.getByTestId('agent-chat-send')
    );
    await page.getByRole('button', { name: '新建会话' }).click();
    await page.getByRole('button', { name: '新建会话' }).click();
    await page.getByRole('button', { name: '新建会话' }).click();

    await expect(page.getByTestId('agent-chat-session-item')).toHaveCount(4);
  });

  test('filters recent sessions by the active PowerX agent', async ({ page }) => {
    await seedAuthStorage(page);
    await mockPowerXAgents(page);

    await gotoWithFallback(
      page,
      '/agent-skill-bridge',
      page.getByTestId('agent-chat-send')
    );
    await page.getByRole('button', { name: '新建会话' }).click();
    await page.getByTestId('agent-chat-agent-select').click();
    await page.getByRole('option', { name: 'MediaX Skill Agent' }).click();
    await page.getByRole('button', { name: '新建会话' }).click();

    await expect(page.getByTestId('agent-chat-session-item')).toHaveCount(2);
    await expect(page.getByTestId('agent-chat-session-item').first()).toContainText('MediaX Skill Agent 会话');
    await expect(page.getByTestId('agent-chat-session-item')).not.toContainText('System Default Agent 会话');

    await page.getByTestId('agent-chat-agent-select').click();
    await page.getByRole('option', { name: 'System Default Agent' }).click();

    await expect(page.getByTestId('agent-chat-session-item')).toHaveCount(3);
    await expect(page.getByTestId('agent-chat-session-item').first()).toContainText('System Default Agent 会话');
    await expect(page.getByTestId('agent-chat-session-item')).not.toContainText('MediaX Skill Agent 会话');
  });
});
