#!/usr/bin/env python3
"""Pack unpublished content-addressed frontend SDKs; test installed bytes and CLI output."""
import hashlib
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[2]
NUXT = ROOT / 'skeleton/web-admin/nuxt'
TESTS = ['session-termination-v2', 'session-termination-admin', 'session-termination-adapter', 'session-termination-router', 'bridge-v2-interoperability', 'host-session-v2', 'api-auth-errors', 'framework-realtime']
CONTRACTS = ['plugin-session-termination-v2.md', 'plugin-session-termination-source.json', 'plugin-session-termination-framework.md']


def run(args, **kwargs):
    return subprocess.run(args, check=True, **kwargs)


def inputs(directory):
    package = json.loads((directory/'package.json').read_text())
    files = {'package.json': (directory/'package.json').read_bytes()}
    for entry in package['files']:
        source = directory / entry
        for path in ([source] if source.is_file() else sorted(source.rglob('*'))):
            if path.is_file() and not any(part in {'node_modules','.nuxt','.output','.DS_Store'} for part in path.relative_to(directory).parts):
                files[path.relative_to(directory).as_posix()] = path.read_bytes()
    return package, files


def digest(files):
    value = hashlib.sha256()
    for name, raw in sorted(files.items()):
        value.update(name.encode()+b'\0'+raw)
    return value.hexdigest()


def validate_bundle(data, fingerprint, bundle, work, browser):
    versions = {kind: data[kind][0]['version']+'-dev.session.'+fingerprint[kind][:16] for kind in data}
    refs = {}; tarballs = {}; hashes = {}
    with (bundle/'validation.log').open('w') as log:
        for kind,(package,files) in data.items():
            staging=work/kind; staging.mkdir(exist_ok=True)
            for name,raw in files.items():
                target=staging/name; target.parent.mkdir(parents=True,exist_ok=True);target.write_bytes(raw)
            package['version']=versions[kind]
            if kind=='admin': package['peerDependencies']['@artisan-cloud/plugin-framework-client']=versions['client']
            (staging/'package.json').write_text(json.dumps(package,indent=2)+'\n')
            packed=run(['npm','pack','--json','--ignore-scripts','--pack-destination',str(bundle)],cwd=staging,capture_output=True,text=True)
            tarball=bundle/json.loads(packed.stdout)[0]['filename'];tarballs[kind]=str(tarball)
            refs[kind]='file:'+str(tarball);hashes[kind]=hashlib.sha256(tarball.read_bytes()).hexdigest()
        binary=work/'px-plugin'
        run(['go','build','-o',str(binary),'./tools/cli/cmd/px-plugin'],cwd=ROOT,stdout=log,stderr=subprocess.STDOUT)
        generated=work/'generated-consumer'
        run([str(binary),'init','com.powerx.plugins.session-fixture','--directory',str(generated),'--backend','go-gin','--admin','nuxt','--app','nuxt','--git-init=false','--init-config=false','--framework-client-ref',refs['client'],'--framework-admin-ref',refs['admin']],cwd=ROOT,stdout=log,stderr=subprocess.STDOUT)
        package=json.loads((generated/'web-admin/package.json').read_text())
        for kind in data:assert package['dependencies'][f'@artisan-cloud/plugin-framework-{kind}']==refs[kind]
        for name in CONTRACTS:assert (generated/'docs/contracts'/name).read_bytes()==(ROOT/'docs/contracts'/name).read_bytes()
        for name in ['useHostBridgeAdapter','useEmbeddedAuth']:
            assert 'HostSession' in (generated/f'web-admin/app/composables/{name}.ts').read_text()
        (work/'package.json').write_text(json.dumps({'private':True,'type':'module','dependencies':{f'@artisan-cloud/plugin-framework-{kind}':refs[kind] for kind in data}},indent=2)+'\n')
        run(['npm','install','--ignore-scripts','--legacy-peer-deps','--package-lock=false','--no-audit','--no-fund'],cwd=work,stdout=log,stderr=subprocess.STDOUT)
        for kind,(_,files) in data.items():
            installed=work/f'node_modules/@artisan-cloud/plugin-framework-{kind}'
            assert not installed.is_symlink()
            for name,raw in files.items():
                if name!='package.json':assert (installed/name).read_bytes()==raw,name
        tests=work/'tests/unit';tests.mkdir(parents=True,exist_ok=True)
        fixture=work/'tests/fixtures';fixture.mkdir(exist_ok=True)
        shutil.copyfile(NUXT/'tests/fixtures/core-bridge-v2.js',fixture/'core-bridge-v2.js')
        for name in TESTS:
            text=(NUXT/f'tests/unit/{name}.spec.ts').read_text()
            text=text.replace('../../../../../framework/frontend/nuxt/framework-client','@artisan-cloud/plugin-framework-client')
            text=text.replace('../../../../../framework/frontend/nuxt/framework-admin/layer/app/plugins/powerx-capability.client','@artisan-cloud/plugin-framework-admin/layer/app/plugins/powerx-capability.client.ts')
            text=text.replace("../../app/", "../../generated-consumer/web-admin/app/")
            (tests/f'{name}.spec.ts').write_text(text)
        run(['node',str(NUXT/'node_modules/vitest/dist/cli.js'),'run','--maxWorkers=2','--config',str(NUXT/'vitest.config.ts'),'--root',str(work),'tests/unit'],cwd=ROOT,stdout=log,stderr=subprocess.STDOUT)
        types=work/'consumer-types.ts'
        types.write_text('''import { reportPowerXHostSessionInvalid, PowerXHostContextError, type PowerXFrameBridgeClient, type HostSessionClearScope } from '@artisan-cloud/plugin-framework-client';
declare const bridge: PowerXFrameBridgeClient;
const report = { sessionId: 'fixture', revision: 1, code: 'IAM_CONTEXT_USER_SESSION_INVALID' } as const;
bridge.reportSessionInvalid(report); reportPowerXHostSessionInvalid(report);
const clear: HostSessionClearScope = 'identity'; const failure = new PowerXHostContextError({ status: 401, code: report.code });
void clear; void failure;
''')
        run(['node',str(NUXT/'node_modules/typescript/bin/tsc'),'--noEmit','--strict','--skipLibCheck','--target','ES2022','--module','ESNext','--moduleResolution','bundler','--lib','ES2022,DOM',str(types)],cwd=work,stdout=log,stderr=subprocess.STDOUT)
    result={'scope':'unpublished_local_bundle','fingerprints':fingerprint,'versions':versions,'npm_refs':refs,'tarball_sha256':hashes,'cli_sha256':hashlib.sha256(binary.read_bytes()).hexdigest(),'generated':str(generated),'workspace_lifecycle':'temporary','unit_tests':len(TESTS),'validation':'passed'}
    (bundle/'references.json').write_text(json.dumps(result,indent=2)+'\n')
    if browser:
        (work/'references.json').write_text(json.dumps(result,indent=2)+'\n')
        try:
            run(['node',str(ROOT/'scripts/contracts/test-session-termination-browser.mjs'),str(work)],cwd=ROOT)
        finally:
            report=work/'browser-validation.json'
            if report.exists():shutil.copyfile(report,bundle/report.name)


def main():
    parser=argparse.ArgumentParser();parser.add_argument('--browser',action='store_true');options=parser.parse_args()
    data = {kind: inputs(ROOT/f'framework/frontend/nuxt/framework-{kind}') for kind in ['client','admin']}
    fingerprint = {kind: digest(files) for kind, (_,files) in data.items()}
    generator = run(['git','ls-files','--cached','--others','--exclude-standard','--','tools/cli/cmd','tools/cli/internal'], cwd=ROOT, capture_output=True, text=True)
    fingerprint['generator'] = digest({name:(ROOT/name).read_bytes() for name in sorted(set(generator.stdout.splitlines())) if (ROOT/name).is_file() and name != 'tools/cli/cmd/px-plugin/px-plugin'})
    fingerprint['builder'] = hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
    fingerprint['validation'] = digest({**{name:(NUXT/f'tests/unit/{name}.spec.ts').read_bytes() for name in TESTS}, 'browser':(ROOT/'scripts/contracts/test-session-termination-browser.mjs').read_bytes(), 'provenance':(ROOT/'scripts/contracts/validate-session-termination.mjs').read_bytes()})
    fingerprint['contracts'] = digest({name:(ROOT/'docs/contracts'/name).read_bytes() for name in CONTRACTS})
    identity = hashlib.sha256(json.dumps(fingerprint,sort_keys=True).encode()).hexdigest()[:16]
    bundle = ROOT/'artifacts'/f'framework-session-termination-20261107-{identity}'
    bundle.mkdir(parents=True,exist_ok=True)
    # Keep reports and package archives; disposable consumers stay in ignored tmp/.
    for name in ['references.json','browser-validation.json','cleanup.json']:
        (bundle/name).unlink(missing_ok=True)
    temporary_root=ROOT/'tmp';temporary_root.mkdir(exist_ok=True)
    work=None;passed=False
    try:
        with tempfile.TemporaryDirectory(prefix='session-termination-',dir=temporary_root) as directory:
            work=Path(directory)
            validate_bundle(data,fingerprint,bundle,work,options.browser)
            passed=True
    finally:
        cleanup={'workspace':str(work) if work else None,'temporary_workspace_removed':work is not None and not work.exists(),'validation':'passed' if passed else 'failed'}
        (bundle/'cleanup.json').write_text(json.dumps(cleanup,indent=2)+'\n')
    print(bundle)

if __name__=='__main__':main()
