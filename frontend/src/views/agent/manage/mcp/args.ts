// MCP 表单里「参数（用空格分开）」那一格和 args 数组之间的转换。
// 引号（"a b" 或 'a b'）里的空格不拆；紧挨着的几段算同一个参数，和 shell 的习惯一致。

/** 按空白拆开，带引号的一段保持完整（引号本身去掉） */
export const splitArgs = (input: string): string[] => {
  const out: string[] = []
  let cur = ''
  let quote = ''
  let open = false
  for (const ch of input) {
    if (quote) {
      if (ch === quote) quote = ''
      else cur += ch
    } else if (ch === '"' || ch === "'") {
      quote = ch
      open = true
    } else if (/\s/.test(ch)) {
      if (open) out.push(cur)
      cur = ''
      open = false
    } else {
      cur += ch
      open = true
    }
  }
  if (open) out.push(cur)
  return out
}

const quoteArg = (arg: string): string => {
  if (arg !== '' && !/[\s"']/.test(arg)) return arg
  if (!arg.includes('"')) return `"${arg}"`
  if (!arg.includes("'")) return `'${arg}'`
  // 两种引号都有：双引号包住，里面的双引号换成单引号包着的一段
  return `"${arg.split('"').join(`"'"'"`)}"`
}

/** 拼回一行：带空格或引号的参数加上引号，splitArgs 能原样拆回来 */
export const joinArgs = (args: string[]): string => args.map(quoteArg).join(' ')
