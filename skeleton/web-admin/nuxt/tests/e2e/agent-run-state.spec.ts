import { expect, test } from '@playwright/test';
import { setupServiceSession } from './_agent-session';

// The retired token/agent_run projection is not the service-session protocol.
for (const outcome of ['failed','interrupted'] as const) {
  test('service session reports '+outcome+' without a successful final',async({page})=>{
    const seen=await setupServiceSession(page,outcome);
    await page.getByTestId('agent-chat-input').fill('test.input');
    await page.getByTestId('agent-chat-send').click();
    await expect(page.getByTestId('agent-chat-status')).toContainText('error');
    expect(seen.filter(r=>r.method==='POST'&&r.path.endsWith('/invocations'))).toHaveLength(1);
    expect(seen.some(r=>r.path.endsWith('/cancel'))).toBe(false);
  });
}
