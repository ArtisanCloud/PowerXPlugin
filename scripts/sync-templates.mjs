#!/usr/bin/env node
import fs from 'fs'
import path from 'path'
import process from 'process'
import fg from 'fast-glob'
import micromatch from 'micromatch'
import YAML from 'yaml'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const rootDir = path.resolve(__dirname, '..')

const configPath = path.resolve(__dirname, 'template-sync-config.yaml')
if (!fs.existsSync(configPath)) {
  console.error(`[sync-templates] Config not found: ${configPath}`)
  process.exit(1)
}

const config = YAML.parse(fs.readFileSync(configPath, 'utf8'))
if (!config?.mappings || !Array.isArray(config.mappings)) {
  console.error('[sync-templates] Invalid config: missing mappings array')
  process.exit(1)
}

const args = process.argv.slice(2)
const checkMode = args.includes('--check')
let diffCount = 0
const diffFiles = []

for (const mapping of config.mappings) {
  const sourceRoot = path.resolve(rootDir, mapping.source)
  if (!fs.existsSync(sourceRoot)) {
    console.warn(`[sync-templates] Source not found, skip: ${mapping.source}`)
    continue
  }
  const include = Array.isArray(mapping.include) && mapping.include.length ? mapping.include : ['**/*']
  const exclude = Array.isArray(mapping.exclude) ? mapping.exclude : []
  const files = fg.sync(include, {
    cwd: sourceRoot,
    ignore: exclude,
    dot: true,
    onlyFiles: true,
  })

  for (const relative of files) {
    const srcPath = path.join(sourceRoot, relative)
    let content = fs.readFileSync(srcPath, 'utf8')
    content = applyReplacements(content, relative, mapping.replacements || [])

    for (const target of mapping.targets || []) {
      const targetRoot = path.resolve(rootDir, target.path)
      let targetRel = relative
      if (target.prefix) {
        targetRel = path.join(target.prefix, targetRel)
      }
      if (target.suffix) {
        targetRel = `${targetRel}${target.suffix}`
      }
      const outPath = path.join(targetRoot, targetRel)

      if (checkMode) {
        if (!fs.existsSync(outPath)) {
          diffCount++
          diffFiles.push(outPath)
          continue
        }
        const existing = fs.readFileSync(outPath, 'utf8')
        if (existing !== content) {
          diffCount++
          diffFiles.push(outPath)
        }
      } else {
        fs.mkdirSync(path.dirname(outPath), { recursive: true })
        fs.writeFileSync(outPath, content)
      }
    }
  }
}

if (checkMode) {
  if (diffCount > 0) {
    console.error(`[sync-templates] Found ${diffCount} out-of-sync file(s):`)
    diffFiles.slice(0, 20).forEach((file) => console.error(`  - ${path.relative(rootDir, file)}`))
    if (diffFiles.length > 20) {
      console.error('  ...')
    }
    process.exit(1)
  } else {
    console.log('[sync-templates] All templates up to date.')
  }
} else {
  console.log('[sync-templates] Templates synchronized successfully.')
}

function applyReplacements(input, relativePath, replacements) {
  let result = input
  if (!Array.isArray(replacements)) return result
  for (const entry of replacements) {
    const globs = entry.globs && entry.globs.length ? entry.globs : ['**/*']
    if (!micromatch.isMatch(relativePath, globs, { dot: true })) {
      continue
    }
    if (entry.literal !== undefined) {
      result = result.split(entry.literal).join(entry.replace ?? '')
      continue
    }
    if (entry.regex) {
      const flags = entry.flags || 'g'
      const re = new RegExp(entry.regex, flags)
      result = result.replace(re, entry.replace ?? '')
    }
  }
  return result
}
