// 四种通知方式（顺序和后端一样）和它们的说明行：Telegram 写 Chat ID，QQBot 写 AppID 和发给谁，其它写 Webhook 地址的后几位。
// 自己的通知页和 Agent 管理页里所有人的通知列表共用。
export type ChannelKind = 'discord' | 'telegram' | 'feishu' | 'qq'

export const CHANNEL_KINDS: ChannelKind[] = ['discord', 'telegram', 'feishu', 'qq']

type Translate = (key: string, named?: Record<string, unknown>) => string

export const channelSub = (t: Translate, kind: string, fields: Record<string, string>) => {
  if (kind === 'telegram') return t('agent.notify.sub.chat', { chat: fields.chat || '' })
  if (kind === 'qq') return t('agent.notify.sub.qq', { appid: fields.appid || '', to: fields.to || '' })
  return t('agent.notify.sub.webhook', { tail: (fields.url || '').slice(-4) })
}
