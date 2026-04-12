#!/usr/bin/env node
import { existsSync, statSync, readdirSync } from 'node:fs'
import { join } from 'node:path'

const root = process.cwd()
const outputDir = join(root, '.output')

const checks = [
  { name: '.output directory', pass: existsSync(outputDir), detail: outputDir },
  { name: '.output/BUILD_ID', pass: existsSync(join(outputDir, 'BUILD_ID')), detail: join(outputDir, 'BUILD_ID') },
  { name: '.output/static', pass: existsSync(join(outputDir, 'static')), detail: join(outputDir, 'static') },
  { name: '.output/server', pass: existsSync(join(outputDir, 'server')), detail: join(outputDir, 'server') },
]

let tree = []
if (existsSync(outputDir)) {
  tree = readdirSync(outputDir).sort()
}

console.log('[verify-artifacts] checks:')
for (const item of checks) {
  console.log(`- ${item.pass ? 'PASS' : 'FAIL'} | ${item.name} | ${item.detail}`)
}

if (tree.length > 0) {
  console.log('[verify-artifacts] .output top-level entries:')
  for (const entry of tree) {
    const p = join(outputDir, entry)
    const kind = statSync(p).isDirectory() ? 'dir' : 'file'
    console.log(`  - ${entry} (${kind})`)
  }
}

if (checks.some((item) => !item.pass)) {
  process.exitCode = 1
  console.error('[verify-artifacts] artifact validation failed')
} else {
  console.log('[verify-artifacts] artifact validation passed')
}
