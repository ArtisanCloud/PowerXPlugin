import importlib.util
import io
import json
from pathlib import Path
import unittest
from unittest.mock import patch
import uuid

spec = importlib.util.spec_from_file_location('regression_live_check', Path(__file__).with_name('regression-live-check.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def environment():
    values = {key: str(uuid.uuid4()) for key in module.UUID_KEYS}
    values.update(REGRESSION_API_BASE='http://127.0.0.1:18078/api/v1',
                  REGRESSION_PLUGIN_ID='com.powerx.fixture', REGRESSION_LIVE_RESOURCE_TYPE='fixture.resource',
                  REGRESSION_TOKEN_ENV='FIXTURE_TOKEN', FIXTURE_TOKEN='fixture.private.token')
    return values


class LiveReadiness(unittest.TestCase):
    def test_missing_environment_is_blocked_without_requests(self):
        with patch.object(module.urllib.request, 'urlopen') as request:
            result = module.check({})
        self.assertEqual(result['status'], 'blocked')
        self.assertEqual(result['missing_fields'], list(module.KEYS))
        request.assert_not_called()

    def test_rejects_invalid_identity_and_embedded_credentials(self):
        for key, value in [('REGRESSION_TENANT_UUID', '1'),
                           ('REGRESSION_API_BASE', 'https://user:private@fixture.invalid/api/v1')]:
            values = environment()
            values[key] = value
            config, result = module.configuration(values)
            self.assertIsNone(config)
            self.assertEqual(result['invalid_field'], key)
            self.assertNotIn('private', json.dumps(result))

    def test_catalog_check_is_get_only_and_does_not_claim_grant(self):
        values = environment()
        replies = [
            {'success': True, 'data': {'items': [{'uuid': values['REGRESSION_LIVE_PROFILE_UUID'], 'kind': 'live'}]}},
            {'success': True, 'data': {'items': [{'uuid': str(uuid.uuid4()), 'key': 'metadata.tag_bindings'}]}}
        ]
        methods = []
        def response(request, **options):
            methods.append(request.get_method())
            self.assertIsNone(request.data)
            self.assertEqual(request.get_header('Authorization'), 'Bearer ' + values['FIXTURE_TOKEN'])
            return io.BytesIO(json.dumps(replies.pop(0)).encode())
        with patch.object(module.urllib.request, 'urlopen', side_effect=response):
            result = module.check(values)
        self.assertEqual(methods, ['GET', 'GET'])
        self.assertEqual(result['status'], 'ready_for_execution')
        self.assertFalse(result['core_grant_verified'])
        self.assertFalse(result['business_execution_verified'])
        self.assertNotIn(values['FIXTURE_TOKEN'], json.dumps(result))


if __name__ == '__main__':
    unittest.main()
