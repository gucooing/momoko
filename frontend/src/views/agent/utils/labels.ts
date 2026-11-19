// Agent 页面里给人看的文字都在这里拼：后端只给原始字段（种类、厂商、名称、事件），不拼显示用的字符串。
import { translate } from '@/locales'
import type { AgentSessionExport } from '@/types/v1/agent'
import type { AgentItem } from '@/views/agent/code/types'
import { fmtDuration, fmtTokens } from '@/views/agent/utils/format'

type ProviderLike = { kind?: string; vendor?: string; name?: string }

const known = (key: string) => {
  const text = translate(key)
  return text === key ? '' : text
}

/** 供应商的显示名：第三方接口用它的名称，服务器上的登录用检查时 Agent 报的账号。 */
export const providerLabel = (p?: ProviderLike | null): string => {
  if (!p) return ''
  if (p.kind === 'system') return p.name || translate('agent.common.systemLogin')
  return p.name || ''
}

// 权限模式由 Agent 提供：认得的 id 按当前语言显示，认不出的用 Agent 给的原文。
// id 里的「-」换成「_」当文案的键（Codex 的 read-only 等）。
const modeKey = (id?: string) => `agent.modes.${(id || '').replace(/-/g, '_')}`

/** 权限模式的名字 */
export const modeName = (id?: string, fallback?: string) => (id && known(`${modeKey(id)}.name`)) || fallback || id || ''

/** 权限模式的说明 */
export const modeDesc = (id?: string, fallback?: string) => (id && known(`${modeKey(id)}.desc`)) || fallback || ''

/** 会话标题；还没有标题时是「新会话」。 */
export const sessionTitle = (title?: string) => title || translate('agent.common.newSession')

type OptionLike = { kind?: string; name?: string }

/** 批准选项的文字：按种类显示；同一种类有几个选项时（Agent 自己区分了范围）用 Agent 给的原文。 */
export const permOptionLabel = (opt: OptionLike, all: OptionLike[] = []): string => {
  const kind = opt.kind || ''
  const same = all.filter((o) => o.kind === kind).length
  if (same <= 1) {
    const text = known(`agent.perm.${kind}`)
    if (text) return text
  }
  return opt.name || kind
}

/** 拒绝类的选项（按钮放右边、用描边样式）。 */
export const isRejectOption = (opt: OptionLike) => (opt.kind || '').startsWith('reject')

const fence = (text: string, lang = '') => {
  const ticks = text.includes('```') ? '````' : '```'
  return `${ticks}${lang}\n${text.replace(/\n$/, '')}\n${ticks}`
}

const quote = (text: string) =>
  text
    .split('\n')
    .map((line) => `> ${line}`)
    .join('\n')

/** 一条事件写成 Markdown；不需要导出的（子 Agent 内部的过程）返回空。 */
const itemMarkdown = (item: AgentItem): string => {
  if (item.sub) return ''
  const tr = (key: string) => translate(`agent.code.export.${key}`)
  switch (item.type) {
    case 'user': {
      const files = (item.attachments ?? []).map((a) => `\`${a.name}\``).join(' ')
      return `### ${tr('you')}\n\n${item.text || ''}${files ? `\n\n${tr('attachments')}${files}` : ''}`
    }
    case 'assistant':
      return item.text || ''
    case 'thought':
      return item.text ? `${quote(`${tr('thought')}\n\n${item.text}`)}` : ''
    case 'tool': {
      const tool = item.tool
      if (!tool) return ''
      const lines = [`- ${tool.title || tool.kind || ''}${tool.status === 'failed' ? ` (${tr('failed')})` : ''}`]
      if (tool.command) lines.push('', fence(tool.command, 'sh'))
      // 改动原样来自 ACP：新建的给全文，修改的给改前、改后两段
      for (const d of tool.diffs ?? []) {
        if (d.oldText == null) {
          lines.push('', `\`${d.path}\` ${tr('created')}`, '', fence(d.newText))
        } else {
          lines.push('', `\`${d.path}\``, '', tr('before'), '', fence(d.oldText), '', tr('after'), '', fence(d.newText))
        }
      }
      if (tool.output) lines.push('', fence(tool.output))
      return lines.join('\n')
    }
    case 'permission': {
      const perm = item.permission
      if (!perm) return ''
      const chosen = perm.options.find((o) => o.id === perm.optionId)
      const outcome =
        perm.outcome === 'selected' && chosen
          ? permOptionLabel(chosen, perm.options)
          : perm.outcome === 'cancelled'
            ? tr('cancelled')
            : tr('pending')
      return `**${tr('approval')}** ${perm.title}${perm.command ? ` \`${perm.command}\`` : ''} — ${outcome}`
    }
    case 'turn': {
      const turn = item.turn
      if (!turn) return ''
      const parts = [translate(`agent.code.stop.${turn.stop}`) || turn.stop, fmtDuration(turn.ms || 0)]
      if (turn.usage?.total) parts.push(`${fmtTokens(turn.usage.total)} token`)
      const error = turn.code ? translate(`agent.code.turnError.${turn.code}`, { reason: turn.error || '' }) : turn.error
      return `_${parts.join(' · ')}${error ? `：${error}` : ''}_`
    }
    case 'notice':
      return quote(
        item.code
          ? translate(`agent.code.notice.${item.code}`, { reason: item.text || '' })
          : [item.title, item.text].filter(Boolean).join(' · '),
      )
    case 'subagent':
      return item.subagent ? `- ${tr('subagent')}${item.subagent.title || item.subagent.description || ''}` : ''
    default:
      return ''
  }
}

/** 把导出的会话排版成一份 Markdown。 */
export const exportMarkdown = (list: AgentSessionExport[]): string => {
  const out: string[] = []
  for (const entry of list) {
    const s = entry.session
    if (!s) continue
    const meta = [
      s.runtimeName,
      providerLabel({ kind: s.providerKind, vendor: s.providerVendor, name: s.providerName }),
      s.model,
      s.workspaceName || translate('agent.common.noWorkspace'),
      s.createTime ? new Date(s.createTime).toLocaleString() : '',
    ].filter(Boolean)
    out.push(`# ${sessionTitle(s.title)}`, '', meta.join(' · '), '')
    let items: AgentItem[] = []
    try {
      items = JSON.parse(entry.events || '[]') as AgentItem[]
    } catch {
      items = []
    }
    for (const item of items) {
      const text = itemMarkdown(item)
      if (text) out.push(text, '')
    }
  }
  return out.join('\n')
}

/** 把一段文字存成文件下载。 */
export const downloadText = (text: string, filename: string, mime = 'text/markdown;charset=utf-8') => {
  const url = URL.createObjectURL(new Blob([text], { type: mime }))
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

/** 文件名里不能有的字符换掉。 */
export const safeFileName = (name: string) => name.replace(/[\\/:*?"<>|\r\n]+/g, '-').trim() || 'session'
