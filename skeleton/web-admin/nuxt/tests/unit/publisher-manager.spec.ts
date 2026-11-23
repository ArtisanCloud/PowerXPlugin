import { afterEach, expect, it } from 'vitest'
import { mkdtemp, mkdir, writeFile, readFile, rm, realpath } from 'node:fs/promises'
import { join } from 'node:path'
import { tmpdir } from 'node:os'
import { createPublisherManager, managedPublisherOrigin, publisherWorkspace } from '../../../../../framework/frontend/nuxt/framework-admin/publisher/manager'
import { directBuildFeature, publisherIntent } from '../../../../../framework/frontend/nuxt/framework-admin/publisher/types'

const cleanup: Array<() => Promise<unknown>> = []
afterEach(async () => { for (const close of cleanup.splice(0).reverse()) await close() })
const request = () => ({ path: '/connect', method: 'POST', remoteAddress: '127.0.0.1', localPort: 3131, headers: new Headers({ host: '127.0.0.1:3131', origin: 'http://127.0.0.1:3131', 'x-powerx-publisher-origin': 'http://127.0.0.1:3131', 'x-powerx-publisher-intent': publisherIntent }) })

it.each(['host', 'port', 'peer', 'origin', 'intent', 'business_auth'])('managed_origin_rejects_%s_before_launch', kind => {
  const r = request()
  if (kind === 'host') r.headers.set('host', 'evil.example:3131')
  if (kind === 'port') r.localPort = 4000
  if (kind === 'peer') r.remoteAddress = '10.0.0.2'
  if (kind === 'origin') r.headers.set('origin', 'https://evil.example')
  if (kind === 'intent') r.headers.delete('x-powerx-publisher-intent')
  if (kind === 'business_auth') r.headers.set('authorization', 'Bearer fixture')
  expect(() => managedPublisherOrigin(r)).toThrowError('PUBLISHER_FORBIDDEN')
})

it('managed_executor_starts_once_and_cleans_its_owned_process_and_bootstrap', async () => {
  const root = await mkdtemp(join(tmpdir(), 'publisher-manager-unit-'))
  cleanup.push(() => rm(root, { recursive: true, force: true }))
  await writeFile(join(root, 'plugin.yaml'), 'id: com.example.publisher\nversion: 0.1.0\n')
  await mkdir(join(root, 'web-admin/nuxt'), { recursive: true })
  expect(await publisherWorkspace(join(root, 'web-admin/nuxt'))).toBe(await realpath(root))
  const executable = join(root, 'executor')
  await writeFile(executable, `#!/usr/bin/env node
const fs=require('node:fs'),http=require('node:http'),path=require('node:path');
const root=process.argv[process.argv.indexOf('--entry')+1],state=path.join(root,'.px-plugin/publisher');
fs.mkdirSync(state,{recursive:true,mode:0o700});
const protocol='powerx.plugin-publisher-local/v1';
const features=fs.existsSync(path.join(root,'updated'))?[${JSON.stringify(directBuildFeature)}]:[];
const server=http.createServer((req,res)=>{
  if(fs.existsSync(path.join(root,'expire'))&&req.url!=='/v1/session'){res.writeHead(401,{'Content-Type':'application/json'});res.end(JSON.stringify({error:{code:'PUBLISHER_UNAUTHORIZED'}}));return;}
  const buildList=req.url.startsWith('/v1/builds?');
  const kind=buildList?'builds':req.url.split('/').pop();
  const stages={builds:'building',uploads:'uploading',submissions:'sending'};
  const items=stages[kind]&&fs.existsSync(path.join(root,kind))?[{stage:stages[kind]}]:[];
  let data;
  if(req.url==='/v1/session') data={session_token:'s'.repeat(43),expires_at:new Date(Date.now()+3600000).toISOString()};
  else if(buildList) data={items,total:items.length,page:1,size:100};
  else if(kind==='uploads'||kind==='submissions') data=items;
  else data={protocol,plugin:{plugin_id:'com.example.publisher',name:'fixture',source_version:'0.1.0'},features,build_targets:[],build_environment:{native_available:false,container_available:false,container_ready:false,can_start:false}};
  res.setHeader('Content-Type','application/json');res.end(JSON.stringify({data}));
});
server.listen(0,'127.0.0.1',()=>{const info={protocol,pid:process.pid,endpoint:'http://127.0.0.1:'+server.address().port,instance_id:'00000000-0000-4000-8000-000000000001',session_file:'.px-plugin/publisher/tool-session.json',bootstrap_expires_at:new Date(Date.now()+300000).toISOString()};fs.writeFileSync(path.join(state,'tool-session.json'),JSON.stringify({...info,bootstrap_token:'b'.repeat(43)}),{mode:0o600});fs.appendFileSync(path.join(root,'starts'),process.pid+'\\n');console.log(JSON.stringify({data:info}));});
process.on('SIGUSR1',()=>{const file=path.join(state,'tool-session.json');const value=JSON.parse(fs.readFileSync(file));value.bootstrap_token='c'.repeat(43);fs.writeFileSync(file+'.next',JSON.stringify(value),{mode:0o600});fs.renameSync(file+'.next',file);fs.rmSync(path.join(root,'expire'),{force:true});});
process.on('SIGTERM',()=>{fs.rmSync(path.join(state,'tool-session.json'));server.closeAllConnections();server.close(()=>process.exit(0));});
`, { mode: 0o700 })
  const manager = createPublisherManager({ rootDir: join(root, 'web-admin/nuxt'), executable })
  cleanup.push(() => manager.close())
  expect(await manager.request({ ...request(), path: '/status', method: 'GET' })).toMatchObject({ body: { data: { connected: false } } })
  const results = await Promise.all([manager.request(request()), manager.request(request())])
  expect(results.every(result => result.status === 200)).toBe(true)
  expect((await readFile(join(root, 'starts'), 'utf8')).trim().split('\n')).toHaveLength(1)
  expect(JSON.stringify(results)).not.toContain('b'.repeat(43))
  expect(JSON.stringify(results)).not.toContain('s'.repeat(43))
  await writeFile(join(root, 'expire'), '')
  await expect(manager.request({ ...request(), path: '/info', method: 'GET' })).rejects.toMatchObject({ code: 'PUBLISHER_UNAUTHORIZED' })
  expect(await manager.request(request())).toMatchObject({ status: 200 })
  expect((await readFile(join(root, 'starts'), 'utf8')).trim().split('\n')).toHaveLength(1)
  const ownedPID = Number((await readFile(join(root, 'starts'), 'utf8')).trim())
  const reconnect = { ...request(), path: '/reconnect' }
  await expect(manager.request({ ...reconnect, method: 'GET' })).rejects.toMatchObject({ status: 405 })
  await expect(manager.request({ ...reconnect, body: new Uint8Array([1]) })).rejects.toMatchObject({ status: 400 })
  for (const kind of ['builds', 'uploads', 'submissions']) {
    const active = join(root, kind)
    await writeFile(active, '')
    await expect(manager.request(reconnect)).rejects.toMatchObject({ code: 'PUBLISHER_EXECUTOR_BUSY' })
    expect(() => process.kill(ownedPID, 0)).not.toThrow()
    await rm(active)
  }
  await writeFile(join(root, 'updated'), '')
  const upgraded = await Promise.all([manager.request(reconnect), manager.request(reconnect)])
  for (const result of upgraded)
    expect(result).toMatchObject({ status: 200, body: { data: { info: { features: [directBuildFeature] } } } })
  expect(() => process.kill(ownedPID, 0)).toThrow()
  await expect(manager.request(reconnect)).resolves.toMatchObject({ status: 200 })
  const starts = (await readFile(join(root, 'starts'), 'utf8')).trim().split('\n')
  expect(starts).toHaveLength(2)
  const pid = Number(starts[1])
  const replacement = createPublisherManager({ rootDir: join(root, 'web-admin/nuxt'), executable })
  cleanup.push(() => replacement.close())
  expect(await replacement.request(request())).toMatchObject({ status: 200 })
  expect(() => process.kill(pid, 0)).toThrow()
  expect((await readFile(join(root, 'starts'), 'utf8')).trim().split('\n')).toHaveLength(3)
  await replacement.close()
  await expect(readFile(join(root, '.px-plugin/publisher/tool-session.json'))).rejects.toMatchObject({ code: 'ENOENT' })
})
