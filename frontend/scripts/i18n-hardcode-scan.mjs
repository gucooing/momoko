/**
 * Broader hardcoded Chinese scan for user-visible strings in Vue/TS.
 * Complements i18n-audit.mjs with template text nodes + attrs + feedback calls.
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(__dirname, '../src')

function walk(dir, acc = []) {
  for (const ent of fs.readdirSync(dir, { withFileTypes: true })) {
    if (ent.name === 'node_modules' || ent.name === 'dist') continue
    const p = path.join(dir, ent.name)
    if (ent.isDirectory()) walk(p, acc)
    else if (/\.(vue|ts|tsx)$/.test(ent.name)) acc.push(p)
  }
  return acc
}

const CJK = /[一-鿿]/
const files = walk(root).filter((f) => !f.includes(`${path.sep}locales${path.sep}`))
const hits = []

for (const file of files) {
  const rel = path.relative(root, file).replace(/\\/g, '/')
  const text = fs.readFileSync(file, 'utf8')
  const lines = text.split(/\n/)

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const trimmed = line.trim()
    if (!CJK.test(line)) continue
    // skip comments
    if (/^\s*(\/\/|\/\*|\*|<!--)/.test(trimmed)) continue
    // skip pure import
    if (/^\s*import\s/.test(trimmed)) continue
    // already i18n
    if (/\b(?:t|translate|translateKnownText)\s*\(/.test(line) && !/['"`][^'"`]*[一-鿿]/.test(line.replace(/\b(?:t|translate)\s*\(\s*['"`][^'"`]*['"`]/g, ''))) {
      // if Chinese only appears inside t('...') key path (won't) or as param — still flag string args with CJK that aren't keys
    }
    // detect Chinese string literals not passed as i18n keys
    const strRe = /(['"`])([^'"`]*[一-鿿][^'"`]*)\1/g
    let m
    while ((m = strRe.exec(line))) {
      const raw = m[2]
      const before = line.slice(0, m.index)
      if (/\b(?:t|translate|te)\s*\(\s*$/.test(before)) continue
      if (/\bconsole\./.test(line)) continue
      // skip knownText map keys in non-locale files unlikely
      hits.push({ file: rel, line: i + 1, text: raw.slice(0, 100), kind: 'string' })
    }
    // template text between tags
    if (file.endsWith('.vue')) {
      const tn = />([^<>{}]*[一-鿿][^<>{}]*)</g
      while ((m = tn.exec(line))) {
        hits.push({ file: rel, line: i + 1, text: m[1].trim().slice(0, 100), kind: 'text' })
      }
    }
  }
}

// dedupe
const seen = new Set()
const out = []
for (const h of hits) {
  const k = `${h.file}:${h.line}:${h.text}`
  if (seen.has(k)) continue
  seen.add(k)
  out.push(h)
}

out.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line)
console.log(`hardcoded Chinese hits: ${out.length}`)
for (const h of out) {
  console.log(`${h.file}:${h.line} [${h.kind}] ${h.text}`)
}
