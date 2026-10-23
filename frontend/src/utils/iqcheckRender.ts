import MarkdownIt from 'markdown-it'

// 智商检测回复渲染：模型回复可能是 Markdown、HTML 片段或 SVG，这里统一拼成一份独立的 HTML 文档，
// 交给 sandbox iframe 展示（不允许执行脚本，图片/样式照常加载），所以 Markdown 里的内联 HTML 也可以放行。
const markdown = new MarkdownIt({ html: true, linkify: true, breaks: false })

// 回复里带 ```html / ```svg / ```xml 代码块时，优先渲染代码块本身（常见于“画一张图”类提示词）。
const FENCE_RE = /```(?:html|svg|xml)[^\n]*\n([\s\S]*?)```/i
// 整段回复本身就是 HTML/SVG 文档时直接渲染。
const DOCUMENT_RE = /^\s*(<!doctype\s|<html[\s>]|<svg[\s>])/i

export type ReplyMode = 'html' | 'markdown'

/** 从回复文本里取出用于渲染的内容及其类型。 */
export const extractReply = (text: string): { body: string; mode: ReplyMode } => {
  const source = (text || '').trim()
  const fence = FENCE_RE.exec(source)
  if (fence?.[1]?.trim()) return { body: fence[1], mode: 'html' }
  if (DOCUMENT_RE.test(source)) return { body: source, mode: 'html' }
  return { body: markdown.render(source), mode: 'markdown' }
}

const baseStyle = (dark: boolean, preview: boolean) => {
  const text = dark ? '#e6edf3' : '#1f2328'
  const bg = dark ? '#1b1f24' : '#ffffff'
  const subtle = dark ? '#22272e' : '#f6f8fa'
  const border = dark ? '#30363d' : '#e4e8ee'
  const link = dark ? '#2dd4bf' : '#0d9488'
  return [
    `:root{color-scheme:${dark ? 'dark' : 'light'}}`,
    `html,body{margin:0;background:${bg};color:${text}}`,
    `body{padding:16px;font:14px/1.6 system-ui,-apple-system,"Segoe UI",Roboto,"PingFang SC","Microsoft YaHei",sans-serif;word-break:break-word}`,
    'svg,img,video,canvas{max-width:100%;height:auto}',
    `pre{overflow:auto;padding:12px;border-radius:8px;background:${subtle}}`,
    'code{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:13px}',
    `table{border-collapse:collapse}td,th{border:1px solid ${border};padding:4px 8px}`,
    `a{color:${link}}`,
    `blockquote{margin:0;padding-left:12px;border-left:3px solid ${border};color:${dark ? '#9aa4ae' : '#57606a'}}`,
    preview ? 'body{overflow:hidden}' : '',
  ].join('')
}

/** 把回复文本拼成可直接放进 iframe srcdoc 的完整 HTML 文档。 */
export const buildReplyDocument = (text: string, options: { dark?: boolean; preview?: boolean } = {}) => {
  const dark = !!options.dark
  const preview = !!options.preview
  const { body, mode } = extractReply(text)
  const head = `<meta charset="utf-8"><base target="_blank"><style>${baseStyle(dark, preview)}</style>`
  if (mode === 'html' && /<html[\s>]/i.test(body)) {
    // 完整文档：在 head 里补基础样式；没有 head 时原样展示。
    return /<head[^>]*>/i.test(body)
      ? body.replace(/<head([^>]*)>/i, (_, attrs: string) => `<head${attrs}>${head}`)
      : body
  }
  const bodyClass = `${mode}${preview ? ' preview' : ''}`
  return `<!doctype html><html><head>${head}</head><body class="${bodyClass}">${body}</body></html>`
}
