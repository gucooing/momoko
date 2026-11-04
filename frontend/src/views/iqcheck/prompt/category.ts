// 提示词分类：图标与文案键（iqcheck.ui.category.<key>）。「其他」按内容猜一个图标。
import { IQCheckPromptCategory, type IQCheckPromptInfo } from '@/types/v1/iqcheck'

export interface PromptCategoryDef {
  value: IQCheckPromptCategory
  key: string
  icon: string
}

export const PROMPT_CATEGORIES: PromptCategoryDef[] = [
  { value: IQCheckPromptCategory.IQ_CHECK_PROMPT_CATEGORY_LOGIC, key: 'logic', icon: 'i-lucide-brain' },
  { value: IQCheckPromptCategory.IQ_CHECK_PROMPT_CATEGORY_MATH, key: 'math', icon: 'i-lucide-sigma' },
  { value: IQCheckPromptCategory.IQ_CHECK_PROMPT_CATEGORY_CODING, key: 'coding', icon: 'i-lucide-code-xml' },
  { value: IQCheckPromptCategory.IQ_CHECK_PROMPT_CATEGORY_DRAWING, key: 'drawing', icon: 'i-lucide-pen-tool' },
  { value: IQCheckPromptCategory.IQ_CHECK_PROMPT_CATEGORY_WRITING, key: 'writing', icon: 'i-lucide-feather' },
  { value: IQCheckPromptCategory.IQ_CHECK_PROMPT_CATEGORY_OTHER, key: 'other', icon: 'i-lucide-message-square-text' },
]

const OTHER = PROMPT_CATEGORIES[PROMPT_CATEGORIES.length - 1]!

export const promptCategoryOf = (value: IQCheckPromptCategory | undefined): PromptCategoryDef =>
  PROMPT_CATEGORIES.find((c) => c.value === value) || OTHER

/** 分类图标；未分类时按内容猜：画图（svg/画）/ 编程（代码/函数）/ 其余用通用图标。 */
export const promptIconOf = (p: IQCheckPromptInfo): string => {
  const def = promptCategoryOf(p.category)
  if (def !== OTHER) return def.icon
  const text = `${p.title}\n${p.content}`.toLowerCase()
  if (/svg|html|画|draw|canvas/.test(text)) return 'i-lucide-pen-tool'
  if (/代码|函数|编程|function|code|typescript|python|sql/.test(text)) return 'i-lucide-code-xml'
  return OTHER.icon
}

/** 粗略估算提示词的 token 数：CJK 字符约 1 个 token，其余约 4 个字符 1 个 token。 */
export const estimateTokens = (text: string): number => {
  const cjk = (text.match(/[　-鿿가-힯＀-￯]/g) || []).length
  const rest = text.replace(/[　-鿿가-힯＀-￯]/g, '').replace(/\s+/g, ' ').trim().length
  return Math.max(1, cjk + Math.ceil(rest / 4))
}
