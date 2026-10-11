import test, { after } from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'
import { execFile } from 'node:child_process'
import { parse } from 'yaml'
import {
  validateTarget,
  writeDistTarget,
  verifyDistTarget,
  readGoTarget,
  protectSource
} from '../../skeleton/scripts/manifest/dist-target.mjs'

const execute = promisify(execFile)
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'powerx-dist-target-'))
after(() => fs.rmSync(temporary, { recursive: true, force: true }))
const native = (await execute('go', ['env', 'GOOS', 'GOARCH'])).stdout.trim().split(/\r?\n/)
const cleanEnvironment = { ...process.env }
for (const key of [
  'GOOS',
  'GOARCH',
  'GOFLAGS',
  'MAKEFLAGS',
  'MFLAGS',
  'BACKEND',
  'PLATFORM',
  'TARGET_ARCH',
  'DIST_OS',
  'DIST_ARCH',
  'DIST_DIR',
  'DIST_ROOT',
  'VERSION',
  'DIST_VERSION'
])
  delete cleanEnvironment[key]
cleanEnvironment.GOWORK = 'off'
cleanEnvironment.GOCACHE = path.join(temporary, 'go-cache')
cleanEnvironment.CGO_ENABLED = '0'

function fixture(name, sourceTarget = '') {
  const directory = path.join(temporary, name)
  fs.mkdirSync(directory, { recursive: true })
  fs.cpSync(path.join(root, 'skeleton/make-files/build.mk'), path.join(directory, 'build.mk'))
  fs.mkdirSync(path.join(directory, 'scripts/manifest'), { recursive: true })
  for (const file of ['dist-target.mjs', 'dist-target.locales.json'])
    fs.copyFileSync(
      path.join(root, 'skeleton/scripts/manifest', file),
      path.join(directory, 'scripts/manifest', file)
    )
  fs.writeFileSync(
    path.join(directory, 'plugin.yaml'),
    `id: fixture.plugin\nversion: 1.0.0\nruntime:\n  entry: backend/bin/plugin\n  env:\n    POWERX_BIND_ADDR: ":__POWERX_DYNAMIC_PORT__"\n${sourceTarget}`
  )
  fs.mkdirSync(path.join(directory, 'backend/cmd/plugin'), { recursive: true })
  fs.mkdirSync(path.join(directory, 'backend/cmd/database'), { recursive: true })
  fs.writeFileSync(
    path.join(directory, 'backend/go.mod'),
    'module example.com/distfixture\n\ngo 1.24\n'
  )
  for (const name of ['plugin', 'database'])
    fs.writeFileSync(
      path.join(directory, `backend/cmd/${name}/main.go`),
      'package main\nfunc main() { println("fixture-dist") }\n'
    )
  fs.mkdirSync(path.join(directory, 'web-admin'), { recursive: true })
  fs.symlinkSync(
    path.join(root, 'node_modules'),
    path.join(directory, 'web-admin/node_modules'),
    'dir'
  )
  fs.writeFileSync(
    path.join(directory, 'web-admin/package.json'),
    JSON.stringify({ private: true, scripts: { build: 'node build.mjs' } })
  )
  fs.writeFileSync(
    path.join(directory, 'web-admin/build.mjs'),
    "import fs from 'node:fs';fs.mkdirSync('.output/server',{recursive:true});fs.writeFileSync('.output/server/index.mjs','export default {}');if(process.env.FIXTURE_SWAP_BINARY)fs.copyFileSync(process.env.FIXTURE_SWAP_BINARY,'../backend/bin/plugin');\n"
  )
  fs.writeFileSync(
    path.join(directory, 'Makefile'),
    'BACKEND ?= gin\nPLUGIN_MANIFEST := plugin.yaml\nBACKEND_DIR := backend\nBUILD_DIR := backend/bin\nFRONTEND_DIR := web-admin\nFRONTEND_OUTPUT := web-admin/.output\nDIST_DIR ?= out\nDIST_BACKEND_BIN = $(DIST_DIR)/backend/bin\nDIST_WEBADMIN_DIR = $(DIST_DIR)/web-admin\nDIST_WEBADMIN_OUTPUT = $(DIST_WEBADMIN_DIR)/.output\nVERSION ?= 2.0.0\ninclude build.mk\n.PHONY: plugin-yaml-check plugin-permission-declaration-check\nplugin-yaml-check plugin-permission-declaration-check:\n\t@true\n'
  )
  return directory
}
async function make(directory, variables, environment = {}) {
  return execute('make', ['dist', 'DIST_VERIFY=0', ...variables], {
    cwd: directory,
    env: { ...cleanEnvironment, ...environment },
    maxBuffer: 5 * 1024 * 1024
  })
}
function failWith(code) {
  return (error) => error.code === code
}

test('accepts only one complete, supported target pair', () => {
  for (const system of ['linux', 'darwin', 'windows'])
    for (const arch of ['amd64', 'arm64'])
      assert.deepEqual(validateTarget(system, arch), { os: system, arch })
  for (const pair of [
    ['', ''],
    ['linux', ''],
    ['', 'arm64']
  ])
    assert.throws(() => validateTarget(...pair), failWith('DIST_TARGET_PAIR_REQUIRED'))
  for (const system of ['mac', 'macos', 'freebsd', 'LINUX'])
    assert.throws(() => validateTarget(system, 'arm64'), failWith('DIST_TARGET_INVALID_OS'))
  for (const arch of ['x86_64', 'aarch64', '386', 'ARM64'])
    assert.throws(() => validateTarget('linux', arch), failWith('DIST_TARGET_INVALID_ARCH'))
})
test('overwrites only artifact target and protects source, including symlink aliases', () => {
  const directory = fixture('writer')
  const source = path.join(directory, 'plugin.yaml'),
    manifest = path.join(directory, 'artifact.yaml')
  const original = fs.readFileSync(source, 'utf8')
  fs.writeFileSync(manifest, original + 'target:\n  os: windows\n  arch: amd64\n  obsolete: true\n')
  writeDistTarget({ manifest, source, os: 'darwin', arch: 'arm64', frontendDir: root })
  assert.deepEqual(parse(fs.readFileSync(manifest, 'utf8')).target, { os: 'darwin', arch: 'arm64' })
  assert.equal(fs.readFileSync(source, 'utf8'), original)
  assert.throws(
    () =>
      writeDistTarget({ manifest: source, source, os: 'darwin', arch: 'arm64', frontendDir: root }),
    failWith('DIST_TARGET_SOURCE_OVERWRITE')
  )
  assert.throws(
    () => protectSource(source, directory, true),
    failWith('DIST_TARGET_SOURCE_OVERWRITE')
  )
  const alias = path.join(directory, 'source-alias.yaml')
  fs.symlinkSync(source, alias)
  assert.throws(
    () =>
      writeDistTarget({ manifest: alias, source, os: 'linux', arch: 'amd64', frontendDir: root }),
    failWith('DIST_TARGET_SOURCE_OVERWRITE')
  )
})
let nativeBinary
for (const target of [
  { name: 'host', os: native[0], arch: native[1], variables: [] },
  {
    name: 'linux-amd64',
    os: 'linux',
    arch: 'amd64',
    variables: ['PLATFORM=linux', 'TARGET_ARCH=amd64']
  },
  {
    name: 'linux-arm64',
    os: 'linux',
    arch: 'arm64',
    variables: ['PLATFORM=linux', 'TARGET_ARCH=arm64']
  },
  {
    name: 'windows-amd64',
    os: 'windows',
    arch: 'amd64',
    variables: [],
    environment: { GOOS: 'windows', GOARCH: 'amd64' }
  }
]) {
  test(`make dist publishes ${target.name} with actual Go build metadata and unchanged source`, async () => {
    const directory = fixture(target.name, 'target:\n  os: windows\n  arch: arm64\n')
    const original = fs.readFileSync(path.join(directory, 'plugin.yaml'), 'utf8')
    const result = await make(
      directory,
      ['DIST_DIR=dist/' + target.name, ...target.variables],
      target.environment || (target.name === 'host' ? { GOOS: target.os, GOARCH: target.arch } : {})
    )
    assert.match(result.stdout, /DIST_TARGET_VERIFIED/)
    const artifact = path.join(directory, 'dist', target.name)
    const manifest = parse(fs.readFileSync(path.join(artifact, 'plugin.yaml'), 'utf8'))
    assert.deepEqual(manifest.target, { os: target.os, arch: target.arch })
    assert.equal(manifest.version, '2.0.0')
    assert.equal(fs.readFileSync(path.join(directory, 'plugin.yaml'), 'utf8'), original)
    for (const binary of ['plugin', 'migrate'])
      assert.deepEqual(readGoTarget(path.join(artifact, 'backend/bin', binary)), manifest.target)
    assert.deepEqual(fs.readdirSync(path.join(directory, 'dist')), [target.name])
    if (target.name === 'host') nativeBinary = path.join(artifact, 'backend/bin/plugin')
  })
}
test('make dist fails on a binary mismatch even when DIST_VERIFY=0', async () => {
  assert.ok(nativeBinary)
  const directory = fixture('mismatch')
  const mismatchOS = native[0] === 'linux' ? 'windows' : 'linux'
  const mismatch = path.join(directory, 'different-target')
  await execute('go', ['build', '-o', mismatch, './cmd/plugin'], {
    cwd: path.join(directory, 'backend'),
    env: { ...cleanEnvironment, GOOS: mismatchOS, GOARCH: 'amd64' }
  })
  await assert.rejects(
    make(directory, [], { GOOS: native[0], GOARCH: native[1], FIXTURE_SWAP_BINARY: mismatch }),
    (error) => /DIST_TARGET_BINARY_MISMATCH/.test(error.stderr)
  )
})
test('checks migrate metadata and rejects fake binaries instead of trusting directory names', () => {
  const manifest = path.join(temporary, 'host/dist/host/plugin.yaml')
  const linux = path.join(temporary, 'linux-arm64/dist/linux-arm64/backend/bin/plugin')
  const target = parse(fs.readFileSync(manifest, 'utf8')).target
  const different =
    target.os !== 'linux' || target.arch !== 'arm64'
      ? linux
      : path.join(temporary, 'linux-amd64/dist/linux-amd64/backend/bin/plugin')
  assert.throws(
    () =>
      verifyDistTarget({
        manifest,
        ...target,
        binaries: [nativeBinary, different],
        frontendDir: root
      }),
    failWith('DIST_TARGET_BINARY_MISMATCH')
  )
  const fake = path.join(temporary, 'fake-binary')
  fs.writeFileSync(fake, 'fixture-not-go')
  assert.throws(
    () => verifyDistTarget({ manifest, ...target, binaries: [fake], frontendDir: root }),
    failWith('DIST_TARGET_BINARY_METADATA')
  )
})
test('non-Go make dist requires explicit paired targets and packages Python without a Go binary', async () => {
  const directory = fixture('python')
  fs.mkdirSync(path.join(directory, 'backend/app'), { recursive: true })
  fs.writeFileSync(path.join(directory, 'backend/app/main.py'), 'fixture = True\n')
  fs.writeFileSync(path.join(directory, 'backend/requirements.txt'), 'fastapi\n')
  fs.writeFileSync(
    path.join(directory, 'plugin.yaml'),
    'id: fixture.plugin\nversion: 1.0.0\nruntime:\n  kind: process\n  entry: python3\n  args: [\"-m\", \"uvicorn\", \"app.main:app\"]\n'
  )
  for (const values of [
    [],
    ['DIST_OS=linux'],
    ['DIST_ARCH=arm64'],
    ['DIST_OS=mac', 'DIST_ARCH=arm64'],
    ['DIST_OS=linux', 'DIST_ARCH=aarch64']
  ]) {
    await assert.rejects(make(directory, ['BACKEND=fastapi', ...values]), (error) =>
      /DIST_TARGET_(PAIR_REQUIRED|INVALID_OS|INVALID_ARCH)/.test(error.stderr)
    )
  }
  assert.equal(fs.existsSync(path.join(directory, 'out')), false)
  const original = fs.readFileSync(path.join(directory, 'plugin.yaml'), 'utf8')
  await make(directory, ['BACKEND=fastapi', 'DIST_OS=linux', 'DIST_ARCH=arm64'])
  assert.deepEqual(parse(fs.readFileSync(path.join(directory, 'out/plugin.yaml'), 'utf8')).target, {
    os: 'linux',
    arch: 'arm64'
  })
  assert.equal(fs.existsSync(path.join(directory, 'out/backend/bin/plugin')), false)
  assert.equal(
    fs.readFileSync(path.join(directory, 'out/backend/app/main.py'), 'utf8'),
    'fixture = True\n'
  )
  assert.equal(fs.readFileSync(path.join(directory, 'plugin.yaml'), 'utf8'), original)
})
test('root make dist forwards exactly one artifact directory without dist/mac', async () => {
  const directory = path.join(temporary, 'root-forward')
  fs.mkdirSync(directory, { recursive: true })
  fs.copyFileSync(path.join(root, 'Makefile'), path.join(directory, 'Makefile'))
  const child = fixture('root-child')
  fs.renameSync(child, path.join(directory, 'skeleton'))
  await make(directory, ['DIST_DIR=dist/platform-artifact'], { GOOS: native[0], GOARCH: native[1] })
  assert.deepEqual(fs.readdirSync(path.join(directory, 'skeleton/dist')), ['platform-artifact'])
  const manifest = parse(
    fs.readFileSync(path.join(directory, 'skeleton/dist/platform-artifact/plugin.yaml'), 'utf8')
  )
  assert.deepEqual(manifest.target, { os: native[0], arch: native[1] })
})
