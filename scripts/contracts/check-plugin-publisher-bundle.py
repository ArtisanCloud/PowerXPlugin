#!/usr/bin/env python3
"""Verify the unchanged Marketplace rc.2 snapshot and generated Go consumer."""
import hashlib
import json
from pathlib import Path
root = Path(__file__).resolve().parents[2]
bundle = root / 'contracts/plugin-publisher/v1'
wire = root / 'tools/cli/internal/publisher/wire'
manifest = json.loads((bundle / 'bundle.sha256.json').read_text())
assert manifest['contract_version'] == '1.0.0-rc.2'
assert hashlib.sha256(json.dumps(manifest['files'],sort_keys=True,separators=(',',':')).encode()).hexdigest() == manifest['bundle_sha256']
actual={p.name for p in bundle.iterdir() if p.is_file() and p.name!='bundle.sha256.json'}
assert actual == set(manifest['files']), 'UNEXPECTED_PORTABLE_FILES'
assert not any(p.is_symlink() for p in bundle.iterdir()), 'PORTABLE_CONTRACT_SYMLINK'
market_source=(root/'tools/cli/internal/publisher/market.go').read_text()
assert 'MarketContractVersion = "'+manifest['contract_version']+'"' in market_source
assert 'MarketBundleSHA256 = "'+manifest['bundle_sha256']+'"' in market_source
for name, expected in manifest['files'].items():
    assert hashlib.sha256((bundle / name).read_bytes()).hexdigest() == expected, name
for source in bundle.glob('*.json'):
    assert (wire / source.name).read_bytes() == source.read_bytes(), source.name
assert (wire / 'conformance.go').read_text() == (bundle / 'conformance.go').read_text().replace('package pluginpublisher', 'package wire', 1)
print(json.dumps({'contract_version':manifest['contract_version'],'bundle_sha256':manifest['bundle_sha256'],'files':len(manifest['files']),'generated_consumer':'verified'}))
