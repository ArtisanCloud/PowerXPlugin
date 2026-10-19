#!/usr/bin/env python3
"""Read-only readiness check for explicit live regression configuration."""
import argparse
import json
import os
from pathlib import Path
import urllib.error
import urllib.parse
import urllib.request
import uuid

KEYS = ('REGRESSION_API_BASE', 'REGRESSION_PLUGIN_ID', 'REGRESSION_TENANT_UUID',
        'REGRESSION_LIVE_PROFILE_UUID', 'REGRESSION_LIVE_RESOURCE_TYPE',
        'REGRESSION_LIVE_RESOURCE_UUID', 'REGRESSION_LIVE_TAG_UUID', 'REGRESSION_TOKEN_ENV')
UUID_KEYS = ('REGRESSION_TENANT_UUID', 'REGRESSION_LIVE_PROFILE_UUID',
             'REGRESSION_LIVE_RESOURCE_UUID', 'REGRESSION_LIVE_TAG_UUID')


def configuration(environment):
    missing = [key for key in KEYS if not environment.get(key, '').strip()]
    token_name = environment.get('REGRESSION_TOKEN_ENV', '')
    if token_name and not environment.get(token_name):
        missing.append('REGRESSION_TOKEN_ENV_VALUE')
    if missing:
        return None, {'status': 'blocked', 'reason_code': 'REGRESSION_LIVE_ENV_REQUIRED', 'missing_fields': missing}
    for key in UUID_KEYS:
        try:
            parsed = uuid.UUID(environment[key])
            if not parsed.int or str(parsed) != environment[key]:
                raise ValueError(key)
        except ValueError:
            return None, {'status': 'blocked', 'reason_code': 'REGRESSION_INVALID_REQUEST', 'invalid_field': key}
    endpoint = urllib.parse.urlsplit(environment['REGRESSION_API_BASE'])
    if endpoint.scheme not in ('http', 'https') or not endpoint.hostname or endpoint.username or endpoint.query or endpoint.fragment:
        return None, {'status': 'blocked', 'reason_code': 'REGRESSION_INVALID_REQUEST', 'invalid_field': 'REGRESSION_API_BASE'}
    config = {key: environment[key] for key in KEYS}
    config['token'] = environment[token_name]
    return config, None


def fetch(config, path):
    request = urllib.request.Request(config['REGRESSION_API_BASE'].rstrip('/') + '/admin/regression/' + path,
                                     headers={'Authorization': 'Bearer ' + config['token'],
                                              'X-Tenant-UUID': config['REGRESSION_TENANT_UUID']})
    with urllib.request.urlopen(request, timeout=10) as response:
        value = json.load(response)
    if value.get('success') is not True:
        raise ValueError('REGRESSION_INVALID_RESPONSE')
    return value['data']['items']


def check(environment):
    config, error = configuration(environment)
    if error:
        return error
    try:
        profiles = fetch(config, 'profiles')
        cases = fetch(config, 'cases?profile_kind=live')
    except (urllib.error.URLError, ValueError, KeyError):
        return {'status': 'blocked', 'reason_code': 'REGRESSION_LIVE_API_UNAVAILABLE'}
    profile = next((item for item in profiles if item['uuid'] == config['REGRESSION_LIVE_PROFILE_UUID']), None)
    scenario = next((item for item in cases if item['key'] == 'metadata.tag_bindings'), None)
    if not profile or profile.get('kind') != 'live' or not scenario:
        return {'status': 'blocked', 'reason_code': 'REGRESSION_LIVE_BINDING_UNAVAILABLE'}
    return {'status': 'ready_for_execution', 'profile_uuid': profile['uuid'], 'case_uuid': scenario['uuid'],
            'verification_boundary': 'plugin_catalog_only', 'core_grant_verified': False,
            'business_execution_verified': False}


def main():
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument('--output', type=Path)
    options = parser.parse_args()
    report = {'schema_version': 'v1', **check(os.environ)}
    encoded = json.dumps(report, ensure_ascii=False, indent=2) + '\n'
    if options.output:
        options.output.parent.mkdir(parents=True, exist_ok=True)
        options.output.write_text(encoded)
        options.output.chmod(0o600)
    print(encoded, end='')
    raise SystemExit(0 if report['status'] == 'ready_for_execution' else 2)


if __name__ == '__main__':
    main()
