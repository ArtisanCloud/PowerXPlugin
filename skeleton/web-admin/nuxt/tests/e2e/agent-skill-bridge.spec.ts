import { expect, test } from '@playwright/test';
import { setupServiceSession, agentUUID, messageUUID } from './_agent-session';

test('Agent chat uses append, invoke and read-only events', async ({page})=>{
  const seen=await setupServiceSession(page);
  await page.getByTestId('agent-chat-input').fill('test.input');
  await expect(page.getByTestId('agent-chat-send')).toBeEnabled();
  await page.getByTestId('agent-chat-send').click();
  await expect(page.getByTestId('agent-chat-final').last()).toContainText('test.output');
  const operations=seen.filter(r=>r.method==='POST'||r.path.endsWith('/events'));
  expect(operations.map(r=>[r.method,r.path.split('/').at(-1)])).toEqual([['POST','messages'],['POST','invocations'],['GET','events']]);
  expect(operations[0].body).toEqual({role:'user',content:'test.input'});
  expect(operations[1].body).toEqual({message_uuid:messageUUID});
  expect(operations[0].key).toBeTruthy();expect(operations[1].key).toBeTruthy();
  expect(seen.some(r=>r.path.endsWith('/stream/sse'))).toBe(false);
});

test('session creation only sends the formal UUID DTO', async ({page})=>{
  const seen=await setupServiceSession(page);
  await page.getByTestId('agent-chat-create-session').click();
  await expect.poll(()=>seen.filter(r=>r.method==='POST'&&r.path.endsWith('/sessions')).length).toBe(1);
  const body=seen.find(r=>r.method==='POST'&&r.path.endsWith('/sessions'))!.body;
  expect(Object.keys(body).sort()).toEqual(['agent_uuid','title']);
  expect(body.agent_uuid).toBe(agentUUID);
});
