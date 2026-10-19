#!/usr/bin/env python3
"""Generate two isolated consumers and verify their own backend, UI and CLI."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import tarfile

ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS = ROOT / 'artifacts/regression'


def run(command, directory, env, log_name):
    with (ARTIFACTS / log_name).open('w') as output:
        result = subprocess.run(command, cwd=directory, env=env, stdout=output, stderr=subprocess.STDOUT)
    if result.returncode:
        raise SystemExit('REGRESSION_GENERATED_COMMAND_FAILED:' + log_name)


def main():
    ARTIFACTS.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='powerx-regression-consumers-') as temporary:
        workspace = Path(temporary)
        packed_tool = subprocess.run(['npm', 'pack', 'npm@11.6.2', '--pack-destination', str(workspace), '--json'],
                                     cwd=ROOT, text=True, capture_output=True, check=True)
        tool_archive = workspace / json.loads(packed_tool.stdout)[0]['filename']
        tool_directory = workspace / 'npm-toolchain'
        tool_directory.mkdir()
        with tarfile.open(tool_archive) as archive:
            archive.extractall(tool_directory, filter='data')
        npm = ['node', str(tool_directory / 'package/bin/npm-cli.js')]
        environment = dict(os.environ, REGRESSION_GENERATED_TEST='1', REGRESSION_GENERATED_OUTPUT=str(workspace))
        run(['go', 'test', './internal/templates', '-run', '^TestRegressionGeneratedPlugins$', '-count=1', '-timeout=4m'],
            ROOT / 'tools/cli', environment, 'generated-validation.log')
        archives = {}
        for name in ('framework-admin', 'framework-client'):
            packed = subprocess.run(npm + ['pack', './framework/frontend/nuxt/' + name,
                                     '--pack-destination', str(workspace), '--json'],
                                    cwd=ROOT, text=True, capture_output=True, check=True)
            item = json.loads(packed.stdout)[0]
            archives[item['name']] = 'file:' + str(workspace / item['filename'])
            files = {file['path'] for file in item['files']}
            required = 'regression.ts' if name == 'framework-client' else 'layer/app/components/powerx/PXRegressionWorkbench.vue'
            if required not in files:
                raise SystemExit('REGRESSION_GENERATED_PACKAGE_INCOMPLETE')
        for suffix in ('alpha', 'beta'):
            consumer = workspace / suffix
            frontend = consumer / 'web-admin'
            manifest = frontend / 'package.json'
            data = json.loads(manifest.read_text())
            for name, path in archives.items():
                data['dependencies'][name] = path
            manifest.write_text(json.dumps(data, indent=2) + '\n')
            tests = frontend / 'tests/e2e'
            tests.mkdir(parents=True, exist_ok=True)
            for name in ('regression-workbench.spec.ts', '_utils.ts'):
                (tests / name).write_bytes((ROOT / 'skeleton/web-admin/nuxt/tests/e2e' / name).read_bytes())
            (frontend / 'playwright.config.ts').write_bytes((ROOT / 'skeleton/web-admin/nuxt/playwright.config.ts').read_bytes())
            run(npm + ['install', '--ignore-scripts', '--no-audit', '--no-fund'], frontend, environment,
                'consumer-' + suffix + '-install.log')
            run(['node', 'node_modules/nuxi/bin/nuxi.mjs', 'build'], frontend, environment,
                'consumer-' + suffix + '-build.log')
            scenario_environment = dict(environment, REGRESSION_CONSUMER_ROOT=str(consumer),
                                        REGRESSION_CONSUMER_PLUGIN_ID='com.powerx.regression.' + suffix)
            run(['python3', 'scripts/testing/regression-browser.py'], ROOT, scenario_environment,
                'consumer-' + suffix + '-browser.log')
        print(json.dumps({'generated_consumers': ['alpha', 'beta'], 'status': 'passed',
                          'dependency_boundary': 'unpublished_local_framework_archives'}))


if __name__ == '__main__':
    main()
