/**
 * i18n audit: unused keys, missing keys, hardcoded Chinese in Vue/TS.
 * Run: node frontend/scripts/i18n-audit.mjs
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(__dirname, '../src')
const messagesPath = path.join(root, 'locales/messages.ts')
const src = fs.readFileSync(messagesPath, 'utf8')

function extractObjectLiteral(name) {
  // allow optional TypeScript type annotation: const foo: Type = {
  const re = new RegExp(`(?:const|export const)\\s+${name}\\b[^=]*=\\s*`)
  const m = re.exec(src)
  if (!m) return null
  const startIdx = m.index + m[0].length
  let i = src.indexOf('{', startIdx)
  if (i < 0) return null
  let depth = 0
  for (let j = i; j < src.length; j++) {
    const c = src[j]
    if (c === '{') depth++
    else if (c === '}') {
      depth--
      if (depth === 0) return src.slice(i, j + 1)
    }
  }
  return null
}

function flatten(obj, prefix = '', out = new Set()) {
  if (obj == null || typeof obj !== 'object') return out
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k
    if (v != null && typeof v === 'object' && !Array.isArray(v)) flatten(v, key, out)
    else out.add(key)
  }
  return out
}

// --- collect defined keys from export const messages ---
// messages nests systemMessages/toolsMessages/... so evaluate all blocks in one scope.
// 块名自动发现（按源码声明顺序），避免重命名模块后脚本失效。
function discoverBlockNames() {
  const re = /^(?:export\s+)?const\s+(\w+)\s*(?::[^=\n]+)?=\s*\{/gm
  const names = []
  let m
  while ((m = re.exec(src))) names.push(m[1])
  if (!names.includes('messages')) throw new Error('cannot find `const messages` in messages.ts')
  return names
}

function evalAllMessages() {
  const names = discoverBlockNames()
  const parts = []
  for (const n of names) {
    const lit = extractObjectLiteral(n)
    if (!lit) throw new Error(`cannot extract ${n}`)
    parts.push(`const ${n} = ${lit};`)
  }
  parts.push('return { messages, knownTextMessages: typeof knownTextMessages === "undefined" ? {} : knownTextMessages };')
  try {
    return new Function(parts.join('\n'))()
  } catch (e) {
    throw new Error(`eval all: ${e.message}`)
  }
}

const { messages, knownTextMessages: knownText } = evalAllMessages()
const locales = Object.keys(messages)
const byLocale = Object.fromEntries(locales.map((l) => [l, flatten(messages[l])]))
const definedKeys = byLocale['zh-CN'] || new Set()

// --- walk source files ---
function walk(dir, acc = []) {
  for (const ent of fs.readdirSync(dir, { withFileTypes: true })) {
    if (ent.name === 'node_modules' || ent.name === 'dist' || ent.name.startsWith('.')) continue
    const p = path.join(dir, ent.name)
    if (ent.isDirectory()) walk(p, acc)
    else if (/\.(vue|ts|tsx|js|jsx)$/.test(ent.name)) acc.push(p)
  }
  return acc
}

const files = walk(root).filter((f) => !f.includes(`${path.sep}locales${path.sep}`))

// Collect t() / translate() keys
// Patterns:
//   t('a.b.c')
//   t("a.b.c")
//   t(`a.b.c`)
//   t('a.b.c', {...})
//   translate('a.b.c')
//   i18n.global.t('...')
// Dynamic: t(`foo.${x}`) — mark as dynamic prefix
const usedKeys = new Set()
const dynamicPrefixes = new Set()
const usedKeyLocations = new Map() // key -> locations (path + line)

// also labelKey: 'language.xxx' style string props that look like keys
const labelKeyRe = /\b(?:labelKey|titleKey|placeholderKey|messageKey|i18nKey)\s*:\s*['"]([a-zA-Z][\w.]+)['"]/g

// t(someVar) — skip
// t(`ns.${x}`) dynamic
const dynamicTRe = /\b(?:t|translate)\s*\(\s*`([^`$]*)\$\{/g

for (const file of files) {
  const text = fs.readFileSync(file, 'utf8')
  const rel = path.relative(root, file).replace(/\\/g, '/')
  const lines = text.split(/\n/)

  for (let li = 0; li < lines.length; li++) {
    const line = lines[li]
    // skip pure comments lines for hardcoded scan later
    let m
    const keyRe = /\b(?:t|translate|te|tm)\s*\(\s*(['"`])([^'"`]+)\1/g
    while ((m = keyRe.exec(line))) {
      const key = m[2]
      if (key.includes('${')) {
        dynamicPrefixes.add(key.split('${')[0])
        continue
      }
      // ignore single-word non-dot that aren't keys (unlikely)
      usedKeys.add(key)
      const loc = `${rel}:${li + 1}`
      if (!usedKeyLocations.has(key)) usedKeyLocations.set(key, [])
      usedKeyLocations.get(key).push(loc)
    }
    const i18nRe = /\bi18n\.global\.t\s*\(\s*(['"`])([^'"`]+)\1/g
    while ((m = i18nRe.exec(line))) {
      usedKeys.add(m[2])
    }
    while ((m = labelKeyRe.exec(line))) {
      usedKeys.add(m[1])
    }
    while ((m = dynamicTRe.exec(line))) {
      dynamicPrefixes.add(m[1])
    }
  }
}

// Also collect string-literal keys assigned to *Key variables / maps (dynamic t(labelKey))
// e.g. 'system.operation.types.authLogin' or "system.operation.types.xxx"
// 命名空间根从 messages 实际结构派生，不再硬编码（否则新增/重命名模块后会漏判为未使用）。
const nsRoots = new Set([...definedKeys].map((k) => k.split('.')[0]))
// 单引号/反引号 = 仓库里的 JS 字符串（prettier singleQuote: true）；双引号绝大多数是 Vue 绑定
// 表达式（:key="instance.id"），那是属性访问不是消息键。后者仍并入 usedKeys（防误报未使用），
// 但不进 keyLikeLiterals，因此不会被当成「缺失的键」。
const keyLikeLiterals = new Set()
const bareKeyStringRe = /(['"`])((?:[a-zA-Z][\w]*\.){1,}[a-zA-Z][\w]*)\1/g
for (const file of files) {
  const text = fs.readFileSync(file, 'utf8')
  let m
  while ((m = bareKeyStringRe.exec(text))) {
    const key = m[2]
    // only count if it looks like a known namespace root
    if (!nsRoots.has(key.split('.')[0])) continue
    usedKeys.add(key)
    if (m[1] !== '"') keyLikeLiterals.add(key)
  }
}

// Keys referenced via dynamic prefix: if defined key starts with prefix, count as used
function isDynamicallyUsed(key) {
  for (const p of dynamicPrefixes) {
    if (p && key.startsWith(p)) return true
  }
  return false
}

// --- unused defined keys ---
const unused = [...definedKeys]
  .filter((k) => !usedKeys.has(k) && !isDynamicallyUsed(k))
  .sort()

// --- missing keys (used but not defined) ---
// 只报「确有 t()/translate()/*Key 出处」的键。裸字符串启发式（bareKeyStringRe）会把
// node.name / instance.id 这类 JS 属性访问误当成键，混进来会把真正的缺失淹掉。
const missing = [...usedKeys]
  .filter((k) => {
    if (k.includes('${')) return false
    if (definedKeys.has(k) || isDynamicallyUsed(k)) return false
    return usedKeyLocations.has(k) || keyLikeLiterals.has(k)
  })
  .sort()

// --- locale parity ---
const parity = {}
const base = 'zh-CN'
for (const loc of locales) {
  if (loc === base) continue
  const miss = [...byLocale[base]].filter((k) => !byLocale[loc].has(k)).sort()
  const extra = [...byLocale[loc]].filter((k) => !byLocale[base].has(k)).sort()
  parity[loc] = { missing: miss, extra }
}

// --- hardcoded Chinese in templates/script (user-visible heuristics) ---
// Match Chinese runs of 2+ chars in .vue template and string literals in script
const CJK = /[一-鿿㐀-䶿]/

// exclude: comments, console, import paths, already-i18n messages file
const hardcodes = []

function stripComments(code) {
  // remove // and /* */ and HTML comments roughly
  return code
    .replace(/<!--[\s\S]*?-->/g, '')
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/(^|[^:])\/\/.*$/gm, '$1')
}

for (const file of files) {
  const rel = path.relative(root, file).replace(/\\/g, '/')
  // skip pure type files if no vue
  let text = fs.readFileSync(file, 'utf8')
  text = stripComments(text)

  // For .vue split template/script
  const isVue = file.endsWith('.vue')
  const chunks = []
  if (isVue) {
    const tm = text.match(/<template[\s\S]*?<\/template>/i)
    const sm = text.match(/<script[\s\S]*?<\/script>/gi) || []
    if (tm) chunks.push({ part: 'template', text: tm[0] })
    for (const s of sm) chunks.push({ part: 'script', text: s })
  } else {
    chunks.push({ part: 'ts', text })
  }

  for (const chunk of chunks) {
    // skip if line has t( or translate( nearby for same string — hard
    // Find string literals with CJK
    const strRe = /(['"`])((?:\\.|(?!\1)[\s\S])*?)\1/g
    let m
    while ((m = strRe.exec(chunk.text))) {
      const raw = m[2]
      if (!CJK.test(raw)) continue
      // skip if looks like pure key? no
      // skip import/require paths
      if (/^[@.\/\\\w-]+$/.test(raw)) continue
      // skip CSS selectors / classes unlikely
      // skip console
      const before = chunk.text.slice(Math.max(0, m.index - 80), m.index)
      if (/\bconsole\.(log|warn|error|debug|info)\s*$/.test(before)) continue
      if (/\b(?:t|translate|te)\s*\(\s*$/.test(before)) continue // already i18n key arg — but keys are ascii
      // skip schema / regex
      if (raw.length > 200) continue
      // find line number
      const upto = chunk.text.slice(0, m.index)
      const lineInChunk = upto.split(/\n/).length
      // approximate absolute line: search first line of match in file
      const fileLines = fs.readFileSync(file, 'utf8').split(/\n/)
      let absLine = 0
      const snippet = raw.slice(0, 30)
      for (let i = 0; i < fileLines.length; i++) {
        if (fileLines[i].includes(snippet) || fileLines[i].includes(raw.slice(0, 15))) {
          absLine = i + 1
          break
        }
      }
      hardcodes.push({
        file: rel,
        line: absLine || lineInChunk,
        part: chunk.part,
        text: raw.replace(/\s+/g, ' ').slice(0, 80),
      })
    }

    // template text nodes: >中文<
    if (chunk.part === 'template') {
      const textNodeRe = />([^<>{}]*[一-鿿][^<>{}]*)</g
      while ((m = textNodeRe.exec(chunk.text))) {
        const raw = m[1].trim()
        if (!raw || raw.length < 2) continue
        if (!CJK.test(raw)) continue
        // skip if only interpolation leftovers
        if (/^\s*$/.test(raw)) continue
        const upto = chunk.text.slice(0, m.index)
        const lineInChunk = upto.split(/\n/).length
        hardcodes.push({
          file: rel,
          line: lineInChunk,
          part: 'template-text',
          text: raw.replace(/\s+/g, ' ').slice(0, 80),
        })
      }
      // attributes: title="中文" label="中文" placeholder="中文" aria-label="中文" alt="中文"
      const attrRe =
        /\b(?:title|label|placeholder|aria-label|alt|description|hint|tooltip|header|message|content|empty-text|confirm-text|cancel-text)\s*=\s*(['"])([^'"]*[一-鿿][^'"]*)\1/gi
      while ((m = attrRe.exec(chunk.text))) {
        hardcodes.push({
          file: rel,
          line: chunk.text.slice(0, m.index).split(/\n/).length,
          part: 'attr',
          text: `${m[0].slice(0, 60)}`,
        })
      }
    }
  }
}

// Dedupe hardcodes by file+text
const seenH = new Set()
const hardcodesUnique = []
for (const h of hardcodes) {
  const k = `${h.file}|${h.text}`
  if (seenH.has(k)) continue
  seenH.add(k)
  hardcodesUnique.push(h)
}
hardcodesUnique.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line)

// namespaces
const ns = new Set([...definedKeys].map((k) => k.split('.')[0]))

// Report
const report = {
  summary: {
    locales,
    definedKeys: definedKeys.size,
    usedKeys: usedKeys.size,
    unusedKeys: unused.length,
    missingKeys: missing.length,
    dynamicPrefixes: [...dynamicPrefixes],
    hardcodedChinese: hardcodesUnique.length,
    namespaces: [...ns].sort(),
  },
  parity: Object.fromEntries(
    Object.entries(parity).map(([loc, v]) => [
      loc,
      { missingCount: v.missing.length, extraCount: v.extra.length, missing: v.missing, extra: v.extra },
    ]),
  ),
  unused,
  missing: missing.map((k) => ({ key: k, locations: (usedKeyLocations.get(k) || []).slice(0, 5) })),
  hardcodes: hardcodesUnique,
  knownTextCounts: Object.fromEntries(
    Object.entries(knownText).map(([loc, map]) => [loc, Object.keys(map || {}).length]),
  ),
}

const outPath = path.resolve(__dirname, 'i18n-audit-report.json')
fs.writeFileSync(outPath, JSON.stringify(report, null, 2), 'utf8')

console.log('=== i18n audit summary ===')
console.log(JSON.stringify(report.summary, null, 2))
console.log('\n=== locale parity ===')
for (const [loc, v] of Object.entries(report.parity)) {
  console.log(`${loc}: missing=${v.missingCount} extra=${v.extraCount}`)
  if (v.missing.length) console.log('  missing sample:', v.missing.slice(0, 15).join(', '))
  if (v.extra.length) console.log('  extra sample:', v.extra.slice(0, 15).join(', '))
}
console.log('\n=== missing keys (used but not in zh-CN messages) ===')
for (const m of report.missing.slice(0, 80)) {
  console.log(`  ${m.key}  @ ${(m.locations || []).join(', ')}`)
}
if (report.missing.length > 80) console.log(`  ... +${report.missing.length - 80} more`)
console.log('\n=== unused keys (defined but never referenced) sample ===')
console.log(unused.slice(0, 100).join('\n'))
if (unused.length > 100) console.log(`... +${unused.length - 100} more`)
console.log('\n=== hardcoded Chinese sample (first 120) ===')
for (const h of hardcodesUnique.slice(0, 120)) {
  console.log(`  ${h.file}:${h.line} [${h.part}] ${h.text}`)
}
if (hardcodesUnique.length > 120) console.log(`... +${hardcodesUnique.length - 120} more`)
console.log(`\nFull report: ${outPath}`)
