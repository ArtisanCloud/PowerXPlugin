import { expect, type Page } from '@playwright/test';
import { gotoWithFallback, seedAuthStorage } from './_utils';
export const agentUUID='00000000-0000-4000-8000-000000000001';
export const sessionUUID='10000000-0000-4000-8000-000000000001';
export const messageUUID='20000000-0000-4000-8000-000000000001';
export const invocationUUID='30000000-0000-4000-8000-000000000001';
export async function setupServiceSession(page: Page, outcome: 'succeeded' | 'failed' | 'interrupted' = 'succeeded') {
  const pageErrors: string[] = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  const seen: { method: string; path: string; body: any; key: string | undefined }[] = [];
  let executed=false;
  const now=new Date().toISOString();
  const session={session_uuid:sessionUUID,agent_uuid:agentUUID,title:'test.session',status:'active',revision:1,created_at:now,updated_at:now};
  const invocation={invocation_uuid:invocationUUID,session_uuid:sessionUUID,message_uuid:messageUUID,trace_uuid:agentUUID,status:outcome==='succeeded'?'succeeded':'failed',output:'test.output',reason_code:outcome==='succeeded'?'':'AGENT_SESSION_UPSTREAM_DEPENDENCY'};
  await seedAuthStorage(page);
  // Keep unrelated APIs away from a real backend; explicit fixtures below win.
  await page.route('**/api/v1/**', route => route.fulfill({status: 503, json: {success: false, error: {reason_code: 'E2E_UNMOCKED_ENDPOINT'}}}));
  await page.route('**/admin/user/auth/me/context', route => route.fulfill({json: {success: true, data: {
    is_root: true,
    user: {user_uuid: agentUUID, display_name: 'test.user', status: 1},
    current_tenant_uuid: '40000000-0000-4000-8000-000000000001',
    tenants: [{tenant_uuid: '40000000-0000-4000-8000-000000000001', name: 'test.tenant'}],
    members: [], roles: [], permissions: [],
  }}}));
  await page.route('**/api/v1/plugin/agent-registry/agents/runnable**',route=>route.fulfill({json:{data:{items:[{uuid:agentUUID,powerx_agent_uuid:agentUUID,key:'test.agent',name:'test.agent',status:'active'}]}}}));
  await page.route('**/api/v1/plugin/agent/**',async route=>{
    const req=route.request(),url=new URL(req.url()),path=url.pathname,method=req.method();
    const body=req.postData()?req.postDataJSON():undefined;
    seen.push({method,path,body,key:req.headers()['idempotency-key']});
    if(path.endsWith('/effective-permissions')){await route.fulfill({json:{data:{can_use_agent:true,user:{is_root:true,roles:[],permissions:[]},agent:{powerx_agent_uuid:agentUUID,plugin_agent_id:agentUUID,plugin_id:'com.powerx.plugins.base',agent_key:'test.agent',name:'test.agent',sync_status:'synced'},actions:[]}}});return;}
    const reply=async(code:number,data:any)=>route.fulfill({status:code,json:{success:true,data}});
    if(path.endsWith('/sessions')){if(method==='POST')await reply(201,session);else await reply(200,{items:[session],total:1,page:Number(url.searchParams.get('page')),page_size:Number(url.searchParams.get('page_size'))});return;}
    if(path.endsWith('/messages')){
      if(method==='POST')await reply(201,{message_uuid:messageUUID,session_uuid:sessionUUID,role:'user',content:'test.input',sequence:1,created_at:now});
      else await reply(200,{items:executed&&outcome==='succeeded'?[{message_uuid:messageUUID,session_uuid:sessionUUID,role:'assistant',content:'test.output',sequence:2,created_at:now}]:[],total:executed&&outcome==='succeeded'?1:0,page:Number(url.searchParams.get('page')),page_size:Number(url.searchParams.get('page_size'))});return;
    }
    if(path.endsWith('/invocations')){executed=true;await reply(202,invocation);return;}
    if(path.endsWith('/cancel')){await reply(202,{...invocation,status:'cancelling'});return;}
    if(path.endsWith('/events')){
      const event=(name:string,payload:any)=>`event: ${name}\ndata: ${JSON.stringify(payload)}\n\n`;
      let body=event('state',invocation);
      if(outcome==='succeeded')body+=event('final',invocation)+event('end',{status:'succeeded'});
      if(outcome==='failed')body+=event('error',{reason_code:'AGENT_SESSION_UPSTREAM_DEPENDENCY'})+event('end',{status:'failed'});
      await route.fulfill({status:200,contentType:'text/event-stream',body});return;
    }
    await route.fulfill({status:404,json:{reason_code:'AGENT_SESSION_NOT_FOUND'}});
  });
  await gotoWithFallback(page,'/agent-skill-bridge',page.getByTestId('agent-chat-send'));
  await expect(page.getByTestId('agent-chat-session')).toHaveValue(sessionUUID);
  return Object.assign(seen, {pageErrors});
}
