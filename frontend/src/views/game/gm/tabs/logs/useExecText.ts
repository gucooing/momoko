// GM 记录的展示文本：状态、执行者类型、来源、对象。只用记录里可展示的字段（targets、summary、source），从不涉及请求参数。
import { useI18n } from 'vue-i18n'
import type { GmExecution } from '@/types/v1/game_gm'

export type Pill = 'success' | 'warning' | 'error' | 'info' | 'neutral' | 'primary'

export function useExecText() {
  const { t } = useI18n()

  const statusPill = (s: string): Pill => {
    switch (s) {
      case 'success':
        return 'success'
      case 'partial':
      case 'pending':
        return 'warning'
      case 'failed':
      case 'rejected':
        return 'error'
      case 'running':
        return 'primary'
      default:
        return 'neutral'
    }
  }
  const statusText = (s: string) => t(`gmOps.logs.status.${s || 'unknown'}`)
  const kindText = (k: string) => t(`gmOps.logs.kind.${k === 'player' ? 'player' : 'admin'}`)
  const channelText = (c: string) => t(`gmOps.logs.channel.${['admin', 'passport', 'game'].includes(c) ? c : 'admin'}`)

  /** 来源对象：邮件、兑换码、配置发布、玩家自助、直接执行。 */
  const sourceText = (e: GmExecution) => {
    const s = e.source || ''
    const sum = e.summary || {}
    const unnamed = t('game.unnamed')
    if (s.startsWith('mail:')) return t('gmOps.logs.source.mail', { title: sum.title || unnamed })
    if (s.startsWith('code:')) return t('gmOps.logs.source.code', { batch: sum.batch || unnamed })
    if (s.startsWith('config:')) return t('gmOps.logs.source.config', { domain: sum.title || unnamed })
    if (s === 'self-service') return t('gmOps.logs.source.self')
    return t('gmOps.logs.source.direct')
  }

  /** 列表里「指令 · 对象」的第二行：玩家名单，或服务器指令的标题 / 配置 / 兑换码。 */
  const objectText = (e: GmExecution) => {
    const sum = e.summary || {}
    const parts: string[] = []
    const targets = e.targets || []
    if (targets.length) {
      const names = targets.slice(0, 3).map((x) => x.name || t('gm.player.noName'))
      parts.push(targets.length > 3 ? t('gmOps.logs.targetsMore', { list: names.join('、'), n: targets.length }) : names.join('、'))
    }
    if (e.source?.startsWith('code:')) {
      if (sum.code) parts.push(sum.code)
      if (sum.batch) parts.push(sum.batch)
    } else if (e.source?.startsWith('config:')) {
      parts.push(sum.toSeq ? t('gmOps.logs.configSeq', { title: sum.title || '', from: sum.fromSeq || '0', to: sum.toSeq }) : sum.title || '')
    } else if (e.source?.startsWith('mail:')) {
      if (sum.title) parts.push(sum.title)
      if (!targets.length && sum.target) parts.push(t(`gmOps.mail.target.${sum.target}`))
    } else if (sum.title && !targets.length) {
      parts.push(sum.title)
    }
    if (e.status === 'pending' && e.reason) parts.push(t('gmOps.logs.applyReason', { reason: e.reason }))
    if (!parts.length && e.targetKind === 'server') parts.push(t('gmOps.logs.serverTarget'))
    return parts.filter(Boolean).join(' · ')
  }

  /** 执行者名称（没有时为「未命名」，不显示账号 ID）。 */
  const actorName = (e: { actorName?: string }) => e.actorName || t('game.unnamed')
  /** 指令标题（记录里存的是执行时 manifest 里的标题）。 */
  const commandTitle = (e: { commandTitle?: string }) => e.commandTitle || t('game.unnamed')

  return { statusPill, statusText, kindText, channelText, sourceText, objectText, actorName, commandTitle }
}
