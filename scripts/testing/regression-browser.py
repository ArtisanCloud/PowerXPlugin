#!/usr/bin/env python3
"""Start owned test fixtures, run real browser acceptance, and release every process."""
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import tempfile
import time
from urllib.parse import urlparse
root=Path(__file__).resolve().parents[2]
consumer=os.environ.get('REGRESSION_CONSUMER_ROOT')
plugin_id=os.environ.get('REGRESSION_CONSUMER_PLUGIN_ID','com.powerx.plugins.base')
artifacts=root/'artifacts/regression'/(('consumer-'+plugin_id.rsplit('.',1)[-1]) if consumer else '.')
artifacts.mkdir(parents=True,exist_ok=True)
nuxt=Path(consumer)/'web-admin' if consumer else root/'skeleton/web-admin/nuxt'
backend_cwd=Path(consumer)/'backend' if consumer else root/'skeleton/backend/go-gin'
with tempfile.TemporaryDirectory(prefix='powerx-regression-browser-') as directory:
    ready=Path(directory)/'fixture.json';env=dict(os.environ,REGRESSION_BROWSER_FIXTURE=str(ready),GOWORK='off' if consumer else 'auto')
    backend_log=open(artifacts/'browser-backend.log','w');ui_log=open(artifacts/'browser-ui.log','w');backend=None;ui=None
    try:
        command=['go','test','./internal/transport/http/admin/regression','-run','^TestRegressionBrowserFixture$','-count=1','-timeout','5m']
        backend=subprocess.Popen(command,cwd=backend_cwd,env=env,stdout=backend_log,stderr=subprocess.STDOUT,start_new_session=True)
        for _ in range(300):
            if ready.exists():break
            if backend.poll() is not None:raise RuntimeError('REGRESSION_BROWSER_BACKEND_FAILED')
            time.sleep(.1)
        if not ready.exists():raise RuntimeError('REGRESSION_BROWSER_BACKEND_TIMEOUT')
        fixture=json.loads(ready.read_text())
        with socket.socket() as reservation:reservation.bind(('127.0.0.1',0));port=reservation.getsockname()[1]
        ui_env=dict(env,PORT=str(port),HOST='127.0.0.1',NUXT_PUBLIC_REGRESSION_ENABLED='true',NUXT_PUBLIC_API_BASE_URL='/api/v1')
        ui=subprocess.Popen(['node','.output/server/index.mjs'],cwd=nuxt,env=ui_env,stdout=ui_log,stderr=subprocess.STDOUT,start_new_session=True)
        for _ in range(200):
            with socket.socket() as probe:
                if probe.connect_ex(('127.0.0.1',port))==0:break
            if ui.poll() is not None:raise RuntimeError('REGRESSION_BROWSER_UI_FAILED')
            time.sleep(.1)
        print(json.dumps({'backend_parent_pid':backend.pid,'backend_pid':fixture['pid'],'backend_api':fixture['api_base'],'backend_command':command,'ui_pid':ui.pid,'ui_port':port,'ui_command':['node','.output/server/index.mjs']}),flush=True)
        result=subprocess.run(['node','node_modules/@playwright/test/cli.js','test','tests/e2e/regression-workbench.spec.ts','--workers=1','--output',str(artifacts/'playwright')],cwd=nuxt,env=dict(ui_env,PLAYWRIGHT_BASE_URL=f'http://127.0.0.1:{port}',NUXT_PUBLIC_POWERX_PLUGIN_ID=plugin_id))
        if result.returncode == 0:
            import urllib.request
            # Release the test-only cancellation gate before the independent CLI run.
            urllib.request.urlopen(urllib.request.Request(urlparse(fixture['api_base'])._replace(path='/_fixture/reset').geturl(),method='POST')).close()
            locale=json.loads((nuxt/'i18n/locales/en.json').read_text())['regression']['fixtures']
            catalog=json.load(urllib.request.urlopen(fixture['api_base']+'/admin/regression/cases'))['data']['items']
            cases=[case for case in catalog if case['supported_profile_kinds']==['controlled']]
            if len(cases)!=8:raise RuntimeError('REGRESSION_TEMPLATE_CATALOG_INCOMPLETE')
            inputs=Path(directory)/'inputs.json'
            values={}
            for case in cases:
                fields={'name':locale['name'],'description':locale['description'],'content':locale['validationContent'] if case['key']=='template.validation' else locale['content'],'updated_content':locale['updatedContent']}
                values[case['uuid']]={key:fields[key] for key in case['input_schema']['properties']}
            inputs.write_text(json.dumps(values))
            cli_command=['go','run','./cmd/px-plugin','regression','run','--api-base',fixture['api_base'],'--token-env','REGRESSION_ACCEPTANCE_TOKEN','--plugin-id',plugin_id,'--profile','44444444-4444-4444-8444-444444444444']
            for case in cases:cli_command+=['--case',case['uuid']]
            cli_command+=['--inputs',str(inputs),'--wait','--report',str(artifacts/'cli-business-report.json')]
            with open(artifacts/'cli-browser-parity.log','w') as output:
                cli=subprocess.run(cli_command,cwd=root/'tools/cli',env=dict(env,REGRESSION_ACCEPTANCE_TOKEN='fixture.api.credential',GOWORK='auto'),stdout=output,stderr=subprocess.STDOUT)
            print(json.dumps({'cli_exit':cli.returncode,'report':str(artifacts/'cli-business-report.json')}),flush=True)
            if cli.returncode:raise SystemExit(cli.returncode)
        raise SystemExit(result.returncode)
    finally:
        Path(str(ready)+'.stop').touch()
        if backend:
            try:backend.wait(timeout=15)
            except subprocess.TimeoutExpired:os.killpg(backend.pid,signal.SIGTERM);backend.wait(timeout=10)
        if ready.exists():
            backend_port=urlparse(json.loads(ready.read_text())['api_base']).port
            with socket.socket() as probe:backend_released=probe.connect_ex(('127.0.0.1',backend_port))!=0
            print(json.dumps({'backend_port':backend_port,'released':backend_released}),flush=True)
            if not backend_released:raise RuntimeError('REGRESSION_TEMP_SERVER_NOT_RELEASED')
        evidence=Path(directory)/'browser-business-evidence.json'
        if evidence.exists():(artifacts/evidence.name).write_bytes(evidence.read_bytes())
        if ui:
            os.killpg(ui.pid,signal.SIGTERM)
            try:ui.wait(timeout=10)
            except subprocess.TimeoutExpired:os.killpg(ui.pid,signal.SIGKILL);ui.wait()
            with socket.socket() as probe:released=probe.connect_ex(('127.0.0.1',port))!=0
            print(json.dumps({'ui_port':port,'released':released,'backend_exit':backend.returncode if backend else None}),flush=True)
            if not released:raise RuntimeError('REGRESSION_TEMP_SERVER_NOT_RELEASED')
        backend_log.close();ui_log.close()
