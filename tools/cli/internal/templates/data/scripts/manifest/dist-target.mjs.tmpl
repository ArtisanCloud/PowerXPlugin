#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'
import { createRequire } from 'node:module'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const operatingSystems = new Set(['linux', 'darwin', 'windows'])
const architectures = new Set(['amd64', 'arm64'])
const locales = JSON.parse(
  fs.readFileSync(new URL('./dist-target.locales.json', import.meta.url), 'utf8')
)
export class DistTargetError extends Error {
  constructor(code, params = {}) {
    super(code)
    this.code = code
    this.params = params
  }
}
function fail(code, params) {
  throw new DistTargetError(code, params)
}
export function validateTarget(os, arch) {
  if (!os || !arch) fail('DIST_TARGET_PAIR_REQUIRED')
  if (!operatingSystems.has(os)) fail('DIST_TARGET_INVALID_OS', { value: os })
  if (!architectures.has(arch)) fail('DIST_TARGET_INVALID_ARCH', { value: arch })
  return { os, arch }
}
function canonicalPath(value) {
  const resolved = path.resolve(value)
  if (fs.existsSync(resolved)) return fs.realpathSync(resolved)
  const parent = path.dirname(resolved)
  return parent === resolved ? resolved : path.join(canonicalPath(parent), path.basename(resolved))
}
export function protectSource(source, destination, directory = false) {
  const sourcePath = canonicalPath(source),
    outputPath = canonicalPath(destination)
  if (sourcePath === outputPath || (directory && sourcePath.startsWith(outputPath + path.sep))) {
    fail('DIST_TARGET_SOURCE_OVERWRITE', { path: destination })
  }
}
function loadYaml(frontendDir) {
  try {
    return createRequire(path.resolve(frontendDir, 'package.json'))('yaml')
  } catch {
    fail('DIST_TARGET_YAML_DEPENDENCY', { path: frontendDir })
  }
}
function readManifest(manifest, frontendDir) {
  let input
  try {
    input = fs.readFileSync(manifest, 'utf8')
  } catch {
    fail('DIST_TARGET_IO_ERROR', { path: manifest })
  }
  const yaml = loadYaml(frontendDir)
  const document = yaml.parseDocument(input, { uniqueKeys: true })
  if (document.errors.length || !yaml.isMap(document.contents))
    fail('DIST_TARGET_INVALID_MANIFEST', { path: manifest })
  return document
}
export function writeDistTarget({ manifest, source, os, arch, frontendDir }) {
  const target = validateTarget(os, arch)
  protectSource(source, manifest)
  const document = readManifest(manifest, frontendDir)
  document.set('target', target)
  const temporary = `${manifest}.target-${process.pid}.tmp`
  let descriptor,
    created = false
  try {
    descriptor = fs.openSync(temporary, 'wx', fs.statSync(manifest).mode)
    created = true
    fs.writeFileSync(descriptor, document.toString())
    fs.closeSync(descriptor)
    descriptor = undefined
    fs.renameSync(temporary, manifest)
  } catch {
    fail('DIST_TARGET_IO_ERROR', { path: manifest })
  } finally {
    if (descriptor !== undefined) fs.closeSync(descriptor)
    if (created && fs.existsSync(temporary)) fs.unlinkSync(temporary)
  }
  return target
}
export function readGoTarget(binary) {
  let metadata
  try {
    metadata = execFileSync('go', ['version', '-m', binary], {
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe']
    })
  } catch {
    fail('DIST_TARGET_BINARY_METADATA', { path: binary })
  }
  const values = {}
  for (const line of metadata.split(/\r?\n/)) {
    const match = line.match(/^\s*build\s+(GOOS|GOARCH)=([^\s]+)\s*$/)
    if (match) {
      if (values[match[1]] !== undefined) fail('DIST_TARGET_BINARY_METADATA', { path: binary })
      values[match[1]] = match[2]
    }
  }
  if (!values.GOOS || !values.GOARCH) fail('DIST_TARGET_BINARY_METADATA', { path: binary })
  return { os: values.GOOS, arch: values.GOARCH }
}
export function verifyDistTarget({
  manifest,
  os,
  arch,
  binaries = [],
  backend = 'go',
  frontendDir
}) {
  validateTarget(os, arch)
  const target = readManifest(manifest, frontendDir).toJS().target
  if (!target || Object.keys(target).length !== 2 || target.os !== os || target.arch !== arch) {
    fail('DIST_TARGET_MANIFEST_MISMATCH', { path: manifest })
  }
  if (backend !== 'non-go' && binaries.length === 0) fail('DIST_TARGET_BINARY_REQUIRED')
  for (const binary of binaries) {
    const actual = readGoTarget(binary)
    if (actual.os !== target.os || actual.arch !== target.arch)
      fail('DIST_TARGET_BINARY_MISMATCH', {
        path: binary,
        actual: `${actual.os}/${actual.arch}`,
        expected: `${target.os}/${target.arch}`
      })
  }
  return target
}
function parseArguments(args) {
  const options = { binaries: [] },
    allowed = new Set([
      '--manifest',
      '--source',
      '--os',
      '--arch',
      '--frontend-dir',
      '--backend',
      '--platform',
      '--dist-dir'
    ])
  const command = args.shift()
  if (!['validate', 'write', 'verify'].includes(command)) fail('DIST_TARGET_INVALID_ARGUMENT')
  for (let index = 0; index < args.length; index += 2) {
    const key = args[index],
      value = args[index + 1]
    if (typeof value !== 'string' || value.startsWith('--')) fail('DIST_TARGET_INVALID_ARGUMENT')
    if (key === '--binary') {
      options.binaries.push(value)
      continue
    }
    if (!allowed.has(key) || Object.hasOwn(options, key.slice(2)))
      fail('DIST_TARGET_INVALID_ARGUMENT')
    options[key.slice(2)] = value
  }
  return { command, options }
}
function printMessage(code, params = {}, error = false) {
  const language = String(process.env.LC_ALL || process.env.LANG || 'en')
    .toLowerCase()
    .startsWith('zh')
    ? 'zh'
    : 'en'
  const template = locales[language][code]
  const message = template.replace(/\{(\w+)\}/g, (_, key) => String(params[key] ?? ''))
  ;(error ? console.error : console.log)(`[${code}] ${message}`)
}
function run() {
  try {
    const { command, options } = parseArguments(process.argv.slice(2))
    const config = {
      manifest: options.manifest,
      source: options.source,
      os: options.os,
      arch: options.arch,
      frontendDir: options['frontend-dir'],
      binaries: options.binaries,
      backend: options.backend || 'go'
    }
    const target = validateTarget(config.os, config.arch)
    if (!['go', 'non-go'].includes(config.backend)) fail('DIST_TARGET_INVALID_ARGUMENT')
    if (command === 'validate') {
      if (config.backend !== 'non-go' && !['host', 'linux'].includes(options.platform))
        fail('DIST_TARGET_INVALID_PLATFORM', { value: options.platform })
      if (!config.source || !options['dist-dir']) fail('DIST_TARGET_INVALID_ARGUMENT')
      protectSource(config.source, options['dist-dir'], true)
      printMessage('DIST_TARGET_VALID', { target: `${target.os}/${target.arch}` })
    } else if (command === 'write') {
      if (!config.manifest || !config.source || !config.frontendDir)
        fail('DIST_TARGET_INVALID_ARGUMENT')
      writeDistTarget(config)
      printMessage('DIST_TARGET_WRITTEN', {
        target: `${target.os}/${target.arch}`,
        path: config.manifest
      })
    } else {
      if (!config.manifest || !config.frontendDir) fail('DIST_TARGET_INVALID_ARGUMENT')
      verifyDistTarget(config)
      printMessage('DIST_TARGET_VERIFIED', { target: `${target.os}/${target.arch}` })
    }
  } catch (error) {
    if (error instanceof DistTargetError) printMessage(error.code, error.params, true)
    else printMessage('DIST_TARGET_INVALID_ARGUMENT', {}, true)
    process.exitCode = 1
  }
}
if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) run()
