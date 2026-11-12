// 名称的模糊搜索：管理端与用户通行证共用。
// 规范化（全角半角、大小写、平假名 / 片假名、空格与标点都不区分）后，关键字的每一段（空格分隔）
// 按顺序出现在某个名称里即命中（字之间可以隔着别的字，例如「训笔」命中「训练笔记」）。
// 名称传当前语言的译名与游戏原文名（gameNameVariants），中文、日文、英文输入都能搜到。

const PUNCT = /[\s　・·•\-_.,，。、:：;；!！?？'"‘’“”()（）[\]【】「」『』〈〉《》<>~～/\\|]+/g

/** 规范化：NFKC（全角转半角）、小写、平假名转片假名、去掉空格与标点。 */
export function normalizeName(s: string): string {
  return s
    .normalize('NFKC')
    .toLowerCase()
    .replace(/[ぁ-ゖ]/g, (c) => String.fromCharCode(c.charCodeAt(0) + 0x60))
    .replace(PUNCT, '')
}

const isSubsequence = (needle: string, hay: string) => {
  const chars = [...needle]
  let i = 0
  for (const c of hay) {
    if (c === chars[i]) i++
    if (i === chars.length) return true
  }
  return chars.length === 0
}

/** 关键字（可空格分隔多段）是否模糊命中这些名称中的任意一个；关键字为空时总是命中。 */
export function fuzzyMatch(query: string, names: (string | undefined | null)[]): boolean {
  const tokens = query
    .split(/[\s　]+/)
    .map(normalizeName)
    .filter(Boolean)
  if (!tokens.length) return true
  const hays = names.filter((n): n is string => !!n).map(normalizeName)
  return tokens.every((tk) => hays.some((h) => isSubsequence(tk, h)))
}
