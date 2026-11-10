#!/usr/bin/env python3
"""Build and consume an unpublished, immutable Event Fabric SDK/CLI snapshot."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'scripts/release'))
from framework_archive import source_files, write_module_zip, module_checksums, check_embedded_inputs

MODULE = 'github.com/ArtisanCloud/PowerXPlugin/framework/backend/go'
ESCAPED = 'github.com/!artisan!cloud/!power!x!plugin/framework/backend/go'
CONTRACTS = [
    'event-fabric-service-v1.md', 'event-fabric-service-v1.openapi.yaml',
    'event-fabric-service-v1.proto', 'event-fabric-service-source.json',
    'plugin-event-fabric-install.md',
    'event-fabric-framework.md', 'fixtures/event-fabric-service.v1.json',
]

def digest(files):
    h = hashlib.sha256()
    for name, raw in sorted(files.items()):
        h.update(name.encode() + b'\0' + raw + b'\0')
    return h.hexdigest()

def main():
    run = subprocess.run
    run(['node', 'scripts/contracts/validate-event-fabric.mjs'], cwd=ROOT, check=True)
    run(['node', 'scripts/sync-templates.mjs', '--check'], cwd=ROOT, check=True)
    check_embedded_inputs(ROOT / 'tools/cli/internal/contracts/data')
    check_embedded_inputs(ROOT / 'tools/cli/internal/templates/data')
    sources = source_files(ROOT / 'framework/backend/go')
    hashes = {'go': digest(sources), 'generator': digest(source_files(ROOT / 'tools/cli')),
              'contracts': digest({name: (ROOT / 'docs/contracts' / name).read_bytes() for name in CONTRACTS})}
    identity = hashlib.sha256(json.dumps(hashes, sort_keys=True).encode()).hexdigest()[:16]
    bundle = ROOT / 'artifacts' / ('framework-release-event-fabric-tenant-20261105-' + identity)
    if bundle.exists():
        raise SystemExit('EVENT_FABRIC_BUNDLE_ALREADY_EXISTS:' + str(bundle))
    bundle.mkdir(parents=True)
    version = 'v0.0.30-dev.eventfabric.' + hashes['go'][:16]
    cli_version = 'v0.0.0-eventfabric.' + hashes['generator'][:16]
    proxy = bundle / 'go-proxy' / ESCAPED / '@v'
    proxy.mkdir(parents=True)
    (proxy / (version + '.mod')).write_bytes(sources['go.mod'])
    (proxy / (version + '.info')).write_text(json.dumps({'Version': version, 'Time': '2026-11-05T00:00:00Z'}))
    (proxy / 'list').write_text(version + '\n')
    write_module_zip(proxy / (version + '.zip'), MODULE, version, sources)
    sums = module_checksums(bundle / 'go-proxy', MODULE, version)
    main_file = bundle / 'build-main.go'
    main_file.write_text('''package main
import ("fmt"; "os"; "github.com/powerx-plugin/cli/cmd")
func main(){if err:=cmd.Execute(os.Args[1:]);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}
''')
    binary = bundle / 'px-plugin'
    run(['go', 'build', '-ldflags', '-X github.com/powerx-plugin/cli/cmd.Version=' + cli_version,
         '-o', str(binary), str(main_file)], cwd=ROOT, check=True)
    for name in CONTRACTS:
        target = bundle / 'contracts' / name
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(ROOT / 'docs/contracts' / name, target)
    consumer = bundle / 'generated-consumer'
    env = dict(os.environ, GOWORK='off', GOPROXY=(bundle / 'go-proxy').as_uri() + ',' + os.environ.get('GOPROXY', 'https://proxy.golang.org,direct'), GONOSUMDB=MODULE)
    with (bundle / 'consumer-validation.log').open('w') as log:
        run([str(binary), 'init', 'com.powerx.plugins.event-fabric-fixture', '--directory', str(consumer),
             '--backend', 'go-gin', '--admin', 'nuxt', '--app', 'nuxt', '--git-init=false', '--init-config=false',
             '--framework-version', version], cwd=ROOT, env=env, stdout=log, stderr=subprocess.STDOUT, check=True)
        go_mod = (consumer / 'backend/go.mod').read_text()
        assert version in go_mod and 'replace ' + MODULE not in go_mod
        for name in CONTRACTS:
            assert (consumer / 'docs/contracts' / name).read_bytes() == (ROOT / 'docs/contracts' / name).read_bytes(), name
        assert 'topic_bindings' in (consumer / 'backend/internal/config/config.go').read_text()
        assert 'source_plugin: ""' in (consumer / 'backend/etc/config.example.yaml').read_text()
        assert 'ReliableEvents:' in (consumer / 'backend/cmd/plugin/main.go').read_text()
        assert 'PublishTopicKeys:' in (consumer / 'backend/cmd/plugin/taskbus_provider.go').read_text()
        assert 'publish_topic_keys' in (consumer / 'backend/internal/config/config.go').read_text()
        (consumer / 'backend/eventfabric_generated_test.go').write_text('''package generated
import (
 "context"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "testing"
 fabric "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/eventfabric"
)
func TestGeneratedTenantEventFabric(t *testing.T){
 const tenant="22222222-2222-4222-8222-222222222222"
 const plugin="com.powerx.plugins.event-fabric-fixture"
 const key="fixture.generated.changed"
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Header.Get("Authorization")!="ApiKey generated-tenant-key"{t.Error("SERVICE_CREDENTIAL_MISSING")}
  var input struct{Payload struct{Body struct{Operation string `json:"operation"`;Tenant string `json:"tenant_uuid"`;Plugin string `json:"plugin_id"`;Keys []string `json:"topic_keys"`} `json:"body"`} `json:"payload"`}
  if err:=json.NewDecoder(r.Body).Decode(&input);err!=nil{t.Error(err);return}
  body:=input.Payload.Body
  if body.Operation!="resolve_topics" || body.Tenant!=tenant || body.Plugin!=plugin || len(body.Keys)!=1 || body.Keys[0]!=key{t.Error("RESOLVE_REQUEST_INVALID")}
  _=json.NewEncoder(w).Encode(map[string]any{"code":200,"data":map[string]any{"status":"completed","protocol_used":"core_internal","fallback_used":false,"result":fabric.ResolvedTopics{ContractVersion:fabric.Contract,TenantUUID:tenant,PluginID:plugin,TopicBindings:map[string]string{key:tenant+".powerx.fixture.generated"}}}})
 }));defer server.Close()
 var slot fabric.RuntimeSlot;defer slot.Close()
 runtime,err:=slot.Switch(context.Background(),fabric.TenantRuntimeConfig{Connection:fabric.Config{BaseURL:server.URL,AuthScheme:"apikey",APIKey:"generated-tenant-key"},HTTPClient:server.Client(),TenantUUID:tenant,PluginID:plugin,PublishTopicKeys:[]string{key}})
 if err!=nil{t.Fatal(err)}
 if runtime.TopicBindings()[key]!=tenant+".powerx.fixture.generated"{t.Fatal("RESOLVE_RESPONSE_INVALID")}
 if err:=slot.Close();err!=nil{t.Fatal(err)}
}
''')
        run(['go', 'test', '-mod=mod', MODULE + '/runtime/eventfabric/...', MODULE + '/runtime/taskbus',
             '.', './cmd/plugin', './internal/config'], cwd=consumer / 'backend', env=env,
            stdout=log, stderr=subprocess.STDOUT, check=True)
        run(['go', 'build', '-mod=mod', '-o', str(bundle / 'generated-plugin'), './cmd/plugin'],
            cwd=consumer / 'backend', env=env, stdout=log, stderr=subprocess.STDOUT, check=True)
    assert digest(source_files(ROOT / 'framework/backend/go')) == hashes['go'], 'EVENT_FABRIC_SDK_SOURCE_CHANGED_DURING_PACKAGING'
    assert digest(source_files(ROOT / 'tools/cli')) == hashes['generator'], 'EVENT_FABRIC_GENERATOR_SOURCE_CHANGED_DURING_PACKAGING'
    references = {
        'application_contract': 'event_fabric.service.v1', 'published': False, 'changes_committed': False,
        'base_commit': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
        'source_sha256': hashes, 'go_module': MODULE, 'go_sdk_version': version,
        'go_proxy': (bundle / 'go-proxy').as_uri(), 'go_checksums': sums,
        'generator_version': cli_version, 'generator_binary': str(binary),
        'bundle': str(bundle),
        'consumer': {'path': str(consumer), 'go_work': 'off', 'local_replace': False,
                     'shared_contracts': 'passed', 'sdk_tests': 'passed', 'tenant_resolve_runtime': 'passed', 'backend_build': 'passed'},
        'upgrade_command': 'GOWORK=off GOPROXY=' + (bundle / 'go-proxy').as_uri() + ',https://proxy.golang.org,direct GONOSUMDB=' + MODULE + ' go get ' + MODULE + '@' + version,
        'artifact_sha256': {str(p.relative_to(bundle)): hashlib.sha256(p.read_bytes()).hexdigest()
                            for p in [binary, proxy / (version + '.zip'), bundle / 'generated-plugin']},
        'acceptance': {'controlled_http_sse': True, 'live_core': False, 'live_scrm': False,
                       'migration_applied': False, 'grants_changed': False},
    }
    references['upgrade_steps'] = ['go mod edit -dropreplace=' + MODULE, references['upgrade_command'], 'GOWORK=off go list -m -json ' + MODULE]
    raw = json.dumps(references, ensure_ascii=False, indent=2) + '\n'
    (bundle / 'references.json').write_text(raw)
    (ROOT / 'artifacts/event-fabric-latest.json').write_text(raw)
    print(raw)

if __name__ == '__main__':
    main()
