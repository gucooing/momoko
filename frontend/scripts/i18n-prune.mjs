/**
 * i18n prune: 按 i18n-audit-report.json 的 unused 列表，从 messages.ts 删除未引用的键。
 *
 *   node frontend/scripts/i18n-prune.mjs            # dry-run，只报告
 *   node frontend/scripts/i18n-prune.mjs --write    # 实际写回
 *   ... --messages <path> --report <path>           # 指定输入（自测用）
 *
 * 说明：messages.ts 由若干 <name>Messages 块 + export const messages 组装，命名空间映射写在
 * messages 里（system: systemMessages[locale] …）。本脚本按块逐行做「引号感知的花括号深度」
 * walk，还原每一行的 i18n 键路径后再删，并回收因此变空的父对象。
 * 只处理参与 messages 组装的块；knownTextMessages 不是消息键表，跳过。
 *
 * 注意：内置校验（每个键必须恰好命中 3 个语言）只能证明键行都找到了，证明不了删干净了 ——
 * prettier 折行的条目曾漏删值那一行、留下孤儿字符串。删完务必再跑一次
 * `npx eslint src/locales/messages.ts` 和 `npx vue-tsc --build`。
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const argOf = (name, fallback) => {
  const i = process.argv.indexOf(name)
  return i >= 0 && process.argv[i + 1] ? path.resolve(process.argv[i + 1]) : fallback
}
const messagesPath = argOf('--messages', path.resolve(__dirname, '../src/locales/messages.ts'))
const reportPath = argOf('--report', path.resolve(__dirname, 'i18n-audit-report.json'))
const WRITE = process.argv.includes('--write')

const report = JSON.parse(fs.readFileSync(reportPath, 'utf8'))
const doomed = new Set(report.unused)
if (!doomed.size) {
  console.log('unused 列表为空，无需处理')
  process.exit(0)
}

const raw = fs.readFileSync(messagesPath, 'utf8')
const EOL = raw.includes('\r\n') ? '\r\n' : '\n'
let lines = raw.split(EOL)

// --- 命名空间映射：<name>Messages -> messages 里的键 ---
const nsOfBlock = new Map()
for (const m of raw.matchAll(/^ {4}([a-zA-Z]\w*): (\w+Messages)\['[\w-]+'\],$/gm)) {
  nsOfBlock.set(m[2], m[1])
}

// --- 顶层块的行区间 ---
function blockRanges() {
  const out = []
  for (let i = 0; i < lines.length; i++) {
    const m = /^(?:export\s+)?const\s+(\w+)\s*(?::[^=]+)?=\s*\{\s*$/.exec(lines[i])
    if (!m) continue
    const name = m[1]
    let depth = 1
    let j = i + 1
    for (; j < lines.length && depth > 0; j++) depth += braceDelta(lines[j])
    out.push({ name, start: i, end: j - 1 })
  }
  return out
}

/** 引号外的 { } 净增量（消息文案里有 {name} 之类占位符，必须跳过字符串内部） */
function braceDelta(line) {
  let d = 0
  let q = null
  for (let i = 0; i < line.length; i++) {
    const c = line[i]
    if (q) {
      if (c === '\\') i++
      else if (c === q) q = null
      continue
    }
    if (c === "'" || c === '"' || c === '`') q = c
    else if (c === '{') d++
    else if (c === '}') d--
  }
  return d
}

const KEY_RE = /^\s*(?:(['"])([\w-]+)\1|([A-Za-z_$][\w$]*))\s*:/

function keyOf(line) {
  const m = KEY_RE.exec(line)
  if (!m) return null
  return m[2] ?? m[3]
}

const stats = { deleted: 0, continuations: 0, emptied: 0, perKey: new Map() }

function pruneBlock(block) {
  // 只处理参与 messages 组装的块
  const isMessages = block.name === 'messages'
  const ns = nsOfBlock.get(block.name)
  if (!isMessages && !ns) return

  const stack = []
  for (let i = block.start + 1; i < block.end; i++) {
    const line = lines[i]
    if (line === null) continue
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('//') || trimmed.startsWith('/*') || trimmed.startsWith('*')) continue

    const delta = braceDelta(line)
    const key = keyOf(line)

    if (delta > 0) {
      // 打开一个（或多个）对象；只有 key: { 形式才推进路径
      stack.push(key ?? '?')
      continue
    }
    if (delta < 0) {
      for (let n = 0; n < -delta; n++) stack.pop()
      continue
    }
    // delta === 0：单行条目（叶子，或写在一行里的完整对象）
    if (!key) continue
    const full = [...stack, key]
    // stack[0] 是 locale
    const i18nKey = isMessages ? full.slice(1).join('.') : `${ns}.${full.slice(1).join('.')}`
    if (!doomed.has(i18nKey)) continue

    lines[i] = null
    stats.deleted++
    stats.perKey.set(i18nKey, (stats.perKey.get(i18nKey) || 0) + 1)

    // prettier 会把过长的值折到下一行（`key:` 单独一行，值在后面）。
    // 这种条目必须连着续行一起删，否则会留下一个孤立的字符串把文件搞成语法错误。
    if (/:\s*$/.test(line)) {
      for (let j = i + 1; j < block.end; j++) {
        if (lines[j] === null) continue
        const cont = lines[j]
        lines[j] = null
        stats.continuations++
        if (/,\s*$/.test(cont)) break
      }
    }
  }
}

for (const b of blockRanges()) pruneBlock(b)

// --- 回收变空的对象：`x: {` 紧跟 `},` ---
function collapseEmpties() {
  let changed = false
  const idx = []
  for (let i = 0; i < lines.length; i++) if (lines[i] !== null) idx.push(i)
  for (let n = 0; n < idx.length - 1; n++) {
    const a = lines[idx[n]]
    const b = lines[idx[n + 1]]
    if (!a || !b) continue
    if (/^\s*(?:(['"])[\w-]+\1|[A-Za-z_$][\w$]*)\s*:\s*\{\s*$/.test(a) && /^\s*\}\s*,?\s*$/.test(b)) {
      lines[idx[n]] = null
      lines[idx[n + 1]] = null
      stats.emptied++
      changed = true
    }
  }
  return changed
}
while (collapseEmpties());

const kept = lines.filter((l) => l !== null)
const next = kept.join(EOL)

// --- 校验 ---
const expected = doomed.size * 3 // 三个语言
const perKeyCounts = [...stats.perKey.values()]
const notThrice = [...stats.perKey.entries()].filter(([, n]) => n !== 3)
const untouched = [...doomed].filter((k) => !stats.perKey.has(k))

console.log('=== i18n prune' + (WRITE ? ' (write)' : ' (dry-run)') + ' ===')
console.log('unused keys        :', doomed.size)
console.log('lines deleted      :', stats.deleted, '(expected', expected + ')')
console.log('wrapped-value lines:', stats.continuations)
console.log('empty objects gone :', stats.emptied)
console.log('lines              :', lines.length, '->', kept.length)
if (untouched.length) {
  console.log('\n!! 未匹配到的键（映射有问题，已中止）:', untouched.length)
  console.log('  ' + untouched.slice(0, 30).join('\n  '))
  process.exit(1)
}
if (notThrice.length) {
  console.log('\n!! 不是恰好命中 3 个语言的键:', notThrice.length)
  for (const [k, n] of notThrice.slice(0, 30)) console.log(`  ${k}  x${n}`)
  process.exit(1)
}
if (perKeyCounts.length !== doomed.size) {
  console.log('\n!! 命中键数与 unused 数不符，已中止')
  process.exit(1)
}

if (WRITE) {
  fs.writeFileSync(messagesPath, next)
  console.log('\n已写回', path.relative(process.cwd(), messagesPath))
} else {
  console.log('\ndry-run，未写文件。加 --write 生效。')
}
