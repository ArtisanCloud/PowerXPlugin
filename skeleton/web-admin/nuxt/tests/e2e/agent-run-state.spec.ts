import { expect, test, type Page } from '@playwright/test';
import { gotoWithFallback, seedAuthStorage } from './_utils';

const mockAgentShell = async (page: Page) => {
  await page.route('**/api/v1/plugin/agent-registry/agents/runnable**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          items: [
            {
              id: 1,
              powerx_agent_uuid: '00000000-0000-4000-8000-000000000001',
              plugin_agent_id: 'powerxplugin.template.agent',
              name: 'Template Agent',
              sync_status: 'synced',
              powerx_skill_ids: ['powerxplugin.template.basic'],
            },
          ],
        },
      }),
    });
  });
  await page.route(/\/api\/v1\/plugin\/agent\/agents\/[^/]+\/effective-permissions(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { can_use_agent: true, actions: [] } }),
    });
  });
  await page.route('**/api/v1/plugin/agent/sessions?**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          items: [
            {
              id: 101,
              uuid: '10000000-0000-4000-8000-000000000101',
              session_id: '10000000-0000-4000-8000-000000000101',
              title: 'Template Agent 会话',
              status: 'active',
            },
          ],
        },
      }),
    });
  });
  await page.route('**/api/v1/plugin/agent/sessions', async (route) => {
    if (route.request().method() !== 'POST') return route.fallback();
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          id: 101,
          uuid: '10000000-0000-4000-8000-000000000101',
          session_id: '10000000-0000-4000-8000-000000000101',
        },
      }),
    });
  });
  await page.route('**/api/v1/plugin/agent/sessions/*/messages**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: { items: [] } }),
    });
  });
};

const prepareReadyChat = async (page: Page, message: string) => {
  await gotoWithFallback(page, '/agent-skill-bridge', page.getByTestId('agent-chat-send'));
  await page.getByTestId('agent-chat-agent').fill('00000000-0000-4000-8000-000000000001');
  await page.getByTestId('agent-chat-session').fill('10000000-0000-4000-8000-000000000101');
  await page.getByTestId('agent-chat-trace').fill(`trace_e2e_${Date.now()}`);
  await page.getByTestId('agent-chat-input').fill(message);
  await expect(page.getByTestId('agent-chat-send')).toBeEnabled();
};

test.describe('Agent Run State panel', () => {
  test('shows awaiting params for missing template fields', async ({ page }) => {
    await seedAuthStorage(page);
    await mockAgentShell(page);
    await page.route('**/api/v1/plugin/agent/stream/sse**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: [
          'event: agent_run.awaiting_params',
          'data: {"type":"agent_run.awaiting_params","task_id":"task-create","node_kind":"skill","node_ref":"powerxplugin.template.basic","skill_id":"powerxplugin.template.basic","action":"create","missing_fields":["template.description","template.content"]}',
          '',
          'event: agent_run.final',
          'data: {"type":"agent_run.final","payload":{"data":{"content":"还需要补充描述和内容。"}}}',
          '',
          'event: agent_run.ended',
          'data: {"type":"agent_run.ended","success":true}',
          '',
        ].join('\n'),
      });
    });

    await prepareReadyChat(page, '帮我创建一个标题为测试模板的模板');
    await page.getByTestId('agent-chat-send').click();

    await expect(page.getByTestId('agent-run-state-panel')).toContainText('awaiting_params');
    await expect(page.getByTestId('agent-run-state-panel')).toContainText('template.description');
    await expect(page.getByTestId('agent-run-state-panel')).toContainText('template.content');
    await expect(page.getByTestId('agent-run-state-status')).toContainText('awaiting_params');
  });

  test('shows completed result only when Core emits task_completed with result', async ({ page }) => {
    await seedAuthStorage(page);
    await mockAgentShell(page);
    await page.route('**/api/v1/plugin/agent/stream/sse**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: [
          'event: agent_run.task_started',
          'data: {"type":"agent_run.task_started","task_id":"task-create","node_kind":"skill","node_ref":"powerxplugin.template.basic","action":"create","status":"running"}',
          '',
          'event: agent_run.task_completed',
          'data: {"type":"agent_run.task_completed","task_id":"task-create","node_kind":"skill","node_ref":"powerxplugin.template.basic","action":"create","status":"completed","result":{"content":"模板已创建","template_id":"tpl_001"},"links":[{"label":"查看模板","url":"/templates/tpl_001"}]}',
          '',
          'event: agent_run.final',
          'data: {"type":"agent_run.final","payload":{"data":{"content":"模板已创建。"}}}',
          '',
        ].join('\n'),
      });
    });

    await prepareReadyChat(page, '创建标题、描述、内容都完整的模板');
    await page.getByTestId('agent-chat-send').click();

    await expect(page.getByTestId('agent-run-state-panel')).toContainText('completed');
    await expect(page.getByTestId('agent-run-state-panel')).toContainText('模板已创建');
    await expect(page.getByTestId('agent-run-state-panel')).toContainText('查看模板');
  });

  test('does not show success result without task_completed result', async ({ page }) => {
    await seedAuthStorage(page);
    await mockAgentShell(page);
    await page.route('**/api/v1/plugin/agent/stream/sse**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: [
          'event: agent_run.task_started',
          'data: {"type":"agent_run.task_started","task_id":"task-create","node_kind":"skill","node_ref":"powerxplugin.template.basic","action":"create","status":"running"}',
          '',
          'event: agent_run.final',
          'data: {"type":"agent_run.final","payload":{"data":{"content":"已收到请求。"}}}',
          '',
        ].join('\n'),
      });
    });

    await prepareReadyChat(page, '创建模板');
    await page.getByTestId('agent-chat-send').click();

    await expect(page.getByTestId('agent-run-state-panel')).toContainText('running');
    await expect(page.getByTestId('agent-run-state-panel')).not.toContainText('模板已创建');
    await expect(page.getByTestId('agent-run-state-status')).not.toContainText('completed');
  });

  test('shows protocol error when run ends without agent_run.final', async ({ page }) => {
    await seedAuthStorage(page);
    await mockAgentShell(page);
    await page.route('**/api/v1/plugin/agent/stream/sse**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: [
          'event: agent_run.started',
          'data: {"type":"agent_run.started","payload":{"event":"start"}}',
          '',
          'event: agent_run.ended',
          'data: {"type":"agent_run.ended","success":true}',
          '',
        ].join('\n'),
      });
    });

    await prepareReadyChat(page, '你好');
    await page.getByTestId('agent-chat-send').click();

    await expect(page.getByText('Agent Run State 协议错误：运行已结束但未收到 agent_run.final。')).toBeVisible();
  });

});
