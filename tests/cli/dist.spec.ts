import test from 'node:test'
import assert from 'node:assert/strict'
import { promises as fs } from 'node:fs'
import path from 'node:path'
import os from 'node:os'

import { packageOfflineBuild } from '../../tools/cli/src/lib/dist/offlinePackager'

const samplePem = `-----BEGIN RSA PUBLIC KEY-----
MIIBCgKCAQEAwJm3Qa56sIBB5PEBzBZlaVAiwi5P5w0siJmq9hw3rroWAtxEvsCj
N5qRTniYwmY0jk3GN/PTLptTr2YhVdXDxI5mA7WHHo2EvsCl6mqHZe/uzhGbjZcS
4z9sjY6et3YUJDOT7YjudZvzbpfhryiqYh79pRkzIOSdA8CMh3TTFvW55vQPypsP
fNnUspHH8xO6unRPRlexJ3cF7w1bz+M9NyWSskEUQPbEoCclzUU2tsL3WFnDYPXj
MjMCC0adyNPCtPw8XXAZu8VRK39V8fmwKMtvRPYbfQWeF6x9N6poSXZXgS6Rt+oT
6Xu44qov9BwqkR/1v0qZ9Cxi/FohiAEz0QIDAQAB
-----END RSA PUBLIC KEY-----`

test('packageOfflineBuild emits expected artefacts', async () => {
  const tmp = await fs.mkdtemp(path.join(os.tmpdir(), 'pxp-test-'))
  const artefactFile = path.join(tmp, 'bundle.js')
  await fs.writeFile(artefactFile, 'console.log("hello")')
  const manifestPath = path.join(tmp, 'manifest.json')
  await fs.writeFile(manifestPath, JSON.stringify({ id: 'demo', version: '1.0.0' }))

  const output = await packageOfflineBuild({
    manifestPath,
    artefactPaths: [artefactFile],
    outputDir: path.join(tmp, 'out'),
    marketplacePublicKeyPem: samplePem,
    keyId: 'test-key',
  })

  const pxpExists = await fileExists(output.pxpPath)
  const integrityExists = await fileExists(output.integrityPath)
  assert.equal(pxpExists, true)
  assert.equal(integrityExists, true)
})

async function fileExists(target: string) {
  try {
    await fs.access(target)
    return true
  } catch {
    return false
  }
}
