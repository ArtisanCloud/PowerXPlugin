import fs from 'node:fs';
import path from 'node:path';
import http from 'node:http';
import net from 'node:net';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const bundle = path.resolve(process.argv[2] || '');
assert(fs.existsSync(path.join(bundle, 'references.json')), 'SESSION_BUNDLE_REQUIRED');
const require = createRequire(path.join(root, 'skeleton/web-admin/nuxt/package.json'));
const { chromium } = require('playwright'); const { build, stop } = require('esbuild');
const pluginId = 'com.example.browser-session'; const invalid = 'IAM_CONTEXT_USER_SESSION_INVALID';
const built = await build({ stdin: { resolveDir: bundle, sourcefile: 'browser-entry.ts', contents: `
import { initPowerXBridge, getPowerXHostSession, createPowerXApiFetchClient, createPluginSSEClient } from '@artisan-cloud/plugin-framework-client';
const pluginId=${JSON.stringify(pluginId)};const session=getPowerXHostSession(pluginId);
const state=window.harness={session,calls:[],draft:null,pending:[],result:null};
const transport=async(path,options)=>{state.calls.push({path,method:options.method});
 if(options.hostFailure)throw {response:{status:options.hostFailure,_data:{error_code:'IAM_CONTEXT_USER_SESSION_INVALID'}}};
 if(options.hold)return new Promise((resolve,reject)=>state.pending.push(()=>reject({response:{status:401,_data:{error_code:'IAM_CONTEXT_USER_SESSION_INVALID'}}})));
 if(options.code)throw {response:{status:options.status||401,_data:{error_code:options.code}}};return {};
};
const api=createPowerXApiFetchClient({pluginId,hostSession:session,insidePowerX:true,baseURL:'/_p/'+pluginId+'/api/v1',fetcher:{create:()=>transport}}).client;
const initFailure=Number(new URL(location.href).searchParams.get('init'));
initPowerXBridge({pluginId,onAuthToken:payload=>session.apply(payload,{
 readContext:(snapshot,signal)=>initFailure?api.get('/admin/user/auth/me/context',{hostSnapshot:snapshot,signal,hostFailure:initFailure}):Promise.resolve({}),
 validateContext:()=>({tenantUuid:null,memberUuid:null}),commitContext:()=>{},clearContext:scope=>{if(scope==='session')state.draft=null;}
})});
state.fail=(status,code)=>api.post('/orders',{fixture:true},{status,code}).catch(error=>({code:error.code,status:error.response?.status}));
state.hold=()=>{state.result=null;api.get('/records',{hold:true}).then(()=>state.result='applied',error=>state.result=error.code);};
state.sse=()=>createPluginSSEClient({pluginId,insidePowerX:true}).stream({path:'/events'}).catch(error=>({code:error.reasonCode||error.code}));
window.fixtureReady=true;
` }, bundle: true, format: 'iife', platform: 'browser', write: false });
stop(); // Shut down esbuild's worker before recording the owned browser process tree.
const fixture = fs.readFileSync(path.join(root, 'skeleton/web-admin/nuxt/tests/fixtures/core-bridge-v2.js'));
const sockets = new Set(); const requests = [];
const server = http.createServer((req, res) => {
  const url = new URL(req.url, 'http://fixture'); requests.push(url.pathname);
  if (url.pathname === '/entry.js') { res.setHeader('Content-Type','text/javascript'); return res.end(built.outputFiles[0].text); }
  if (url.pathname === '/core.js') { res.setHeader('Content-Type','text/javascript'); return res.end(fixture); }
  if (url.pathname.endsWith('/api/v1/events')) { res.writeHead(401,{'Content-Type':'application/json'});return res.end(JSON.stringify({error_code:invalid})); }
  res.setHeader('Content-Type','text/html');
  if(url.pathname.startsWith('/_p/')) return res.end('<!doctype html><html><body><script src="/core.js"></script><script src="/entry.js"></script></body></html>');
  res.end('<!doctype html><html><body><iframe id="plugin" style="width:100%;height:80vh;border:0"></iframe><script>window.packets=[];window.addEventListener("message",event=>{if(event.source===document.querySelector("iframe").contentWindow&&event.origin===location.origin)packets.push(event.data);});</script></body></html>');
});
server.on('connection',socket=>{sockets.add(socket);socket.on('close',()=>sockets.delete(socket));});
await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
const port=server.address().port; const origin=`http://127.0.0.1:${port}`;
let browser;let children=[];let childrenClosed=false;const results=[];
const processTree=()=>execFileSync('ps',['-axo','pid=,ppid=,comm='],{encoding:'utf8'}).trim().split('\n').map(line=>{const [,pid,ppid]=line.match(/^\s*(\d+)\s+(\d+)/)||[];return {pid:Number(pid),ppid:Number(ppid),comm:line.replace(/^\s*\d+\s+\d+\s+/, '')};});
const childPids=()=>{const entries=processTree();const ids=new Set([process.pid]);let prior=0;while(prior!==ids.size){prior=ids.size;for(const p of entries)if(ids.has(p.ppid))ids.add(p.pid);}ids.delete(process.pid);return [...ids];};
const auth=(sequence=1,sessionId='login-1')=>({type:'auth-token',sequence,sessionId,revision:1,accessToken:'fixture-'+sequence,tokenType:'Bearer',expiresAt:Date.now()+60000,expiresIn:60,scope:'powerx',tenantUuid:null});
const send=(page,payload)=>page.evaluate(({pluginId,payload})=>document.querySelector('iframe').contentWindow.postMessage({source:'powerx',protocolVersion:2,pluginId,instanceId:'frame',...payload},location.origin),{pluginId,payload});
const packets=(page,type)=>page.evaluate(type=>packets.filter(p=>p.type===type),type);
const waitReady=(frame,sequence=1)=>frame.waitForFunction(sequence=>harness.session.getReadySnapshot()?.sequence===sequence,sequence);
try {
  browser=await chromium.launch({headless:true});children=childPids();
  for(const width of [1440,390])for(const name of ['parallel_active','stale_new_login','initial_401','initial_503','non_terminal','missing_method','auth_cleared','sse_invalid']){
    const context=await browser.newContext({viewport:{width,height:900}});const page=await context.newPage();
    try{
      await page.goto(origin);await page.evaluate(({pluginId,name})=>{localStorage.setItem('host-fixture','preserved');document.querySelector('iframe').src=`/_p/${pluginId}/admin/orders?draft=local${name.startsWith('initial_')?'&init='+name.slice(8):''}#editor`;},{pluginId,name});
      const frame=page.frames().find(f=>f!==page.mainFrame());await frame.waitForFunction(()=>window.fixtureReady===true);
      await send(page,{type:'sync',locale:'en',theme:'dark',hostOrigin:origin});await send(page,auth());
      if(name.startsWith('initial_')){
        await page.waitForFunction(()=>packets.some(p=>p.type==='auth:rejected'));
        const rejected=await packets(page,'auth:rejected');assert.equal(rejected[0].code,name==='initial_401'?invalid:undefined);
        assert.equal((await packets(page,'auth:applied')).length,0);assert.equal((await packets(page,'auth:required')).length,0);
      }else{
        await waitReady(frame);await frame.evaluate(()=>harness.draft='bound-fixture');
        if(name==='parallel_active'){
          await frame.evaluate(async invalid=>Promise.all([harness.fail(401,invalid),harness.fail(401,invalid),harness.fail(401,invalid)]),invalid);
          await page.waitForFunction(()=>packets.some(p=>p.type==='auth:required'));
          const reports=await packets(page,'auth:required');assert.equal(reports.length,1);assert.deepEqual(Object.keys(reports[0]).sort(),['type','sequence','sessionId','revision','code','source','protocolVersion','pluginId','instanceId'].sort());
          assert.equal(await frame.evaluate(()=>harness.draft),'bound-fixture');await send(page,auth(2));await waitReady(frame,2);
          assert.equal(await frame.evaluate(()=>harness.draft),'bound-fixture');assert.equal(await frame.evaluate(()=>harness.calls.length),3);
        }
        if(name==='stale_new_login'){
          await frame.evaluate(()=>harness.hold());await frame.waitForFunction(()=>harness.pending.length===1);
          await send(page,auth(2,'login-2'));await waitReady(frame,2);await frame.evaluate(()=>harness.pending[0]());
          await frame.waitForFunction(()=>harness.result!==null);assert.equal(await frame.evaluate(()=>harness.result),'AUTH_SESSION_CHANGED');assert.equal((await packets(page,'auth:required')).length,0);
        }
        if(name==='non_terminal'){
          for(const [status,code] of [[403,invalid],[503,invalid],[401,'IAM_CONTEXT_PROOF_INVALID'],[401,'AUTH_STS_EXPIRED'],[401,'IAM_UNAUTHORIZED']])await frame.evaluate(([s,c])=>harness.fail(s,c),[status,code]);
          assert.equal((await packets(page,'auth:required')).length,0);assert.equal(await frame.evaluate(()=>harness.session.getState().status),'ready');assert.equal(await frame.evaluate(()=>harness.draft),'bound-fixture');
        }
        if(name==='missing_method'){
          await frame.evaluate(()=>delete PowerXBridgeClient.reportSessionInvalid);const result=await frame.evaluate(code=>harness.fail(401,code),invalid);
          assert.equal(result.code,'PLUGIN_BRIDGE_SESSION_TERMINATION_REQUIRED');assert.equal((await packets(page,'auth:required')).length,0);
        }
        if(name==='auth_cleared'){
          await frame.evaluate(()=>harness.hold());await frame.waitForFunction(()=>harness.pending.length===1);
          await send(page,{type:'auth-cleared',sequence:2});await frame.waitForFunction(()=>harness.session.getState().status==='cleared');
          await send(page,auth(3,'login-2'));await waitReady(frame,3);await frame.evaluate(()=>harness.pending[0]());await frame.waitForFunction(()=>harness.result!==null);
          assert.equal(await frame.evaluate(()=>harness.calls.length),1);assert.equal(await frame.evaluate(()=>harness.draft),null);assert.equal((await packets(page,'auth:required')).length,0);
        }
        if(name==='sse_invalid'){
          assert.equal((await frame.evaluate(()=>harness.sse())).code,invalid);await page.waitForFunction(()=>packets.some(p=>p.type==='auth:required'));assert.equal((await packets(page,'auth:required')).length,1);
        }
      }
      assert.equal(await page.evaluate(()=>localStorage.getItem('host-fixture')),'preserved');assert.equal(page.url(),origin+'/');assert(frame.url().endsWith('#editor'));assert(frame.url().includes('draft=local'));results.push({width,name,passed:true});
    }finally{await context.close();}
  }
}finally{
  await browser?.close();
  for(let attempt=0;attempt<60;attempt++){const live=new Set(processTree().map(p=>p.pid));childrenClosed=children.every(pid=>!live.has(pid));if(childrenClosed)break;await new Promise(resolve=>setTimeout(resolve,100));}
  if(!childrenClosed)console.error(JSON.stringify({remaining_children:processTree().filter(p=>children.includes(p.pid))}));
  for(const socket of sockets)socket.destroy();await new Promise(resolve=>server.close(resolve));
  await new Promise((resolve,reject)=>{const socket=net.connect(port,'127.0.0.1');socket.on('connect',()=>{socket.destroy();reject(new Error('FIXTURE_PORT_STILL_OPEN'));});socket.on('error',resolve);});
  fs.writeFileSync(path.join(bundle,'browser-validation.json'),JSON.stringify({scope:'installed_framework_and_real_core_consumer_with_controlled_parent_not_deployed_core',tests:results,server:{command:'node scripts/contracts/test-session-termination-browser.mjs '+bundle,pid:process.pid,port,closed:true,browser_child_pids:children,browser_children_closed:childrenClosed},requests},null,2)+'\n');
}
assert(childrenClosed,'BROWSER_CHILDREN_STILL_RUNNING');assert.equal(results.length,16);console.log(JSON.stringify({tests:results.length,passed:true,port_closed:port}));
