// 邮件补偿、兑换码、玩家自助、GM 记录四个标签共用的小工具：时间格式、CSV 导出、奖励名称与图片。
import dayjs from 'dayjs'
import { resourceRef } from '@/games/gakumas/catalog'
import { i18n } from '@/locales'
import type { GameReward } from '@/types/v1/game_common'
import { useGmContext } from '../../context'
import { catalogLookupFor } from '../../player/shared/lookup'

/** 接口里的时间（JSON 是 RFC3339 字符串，类型标成 Date）转成 Date；没有值时返回 undefined。 */
export function toDate(value: Date | string | number | undefined | null): Date | undefined {
  if (value === undefined || value === null || value === '') return undefined
  const d = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(d.getTime()) || d.getTime() <= 0) return undefined
  return d
}

/** 完整时间：2026-11-12 10:00。 */
export function fmtDateTime(value: Date | string | number | undefined | null, withSeconds = false): string {
  const d = toDate(value)
  if (!d) return ''
  return dayjs(d).format(withSeconds ? 'YYYY-MM-DD HH:mm:ss' : 'YYYY-MM-DD HH:mm')
}

/** 列表里的短时间：今天只显示时刻，今年显示月日，更早显示日期。 */
export function fmtShort(value: Date | string | number | undefined | null, todayLabel: string): string {
  const d = toDate(value)
  if (!d) return ''
  const v = dayjs(d)
  const now = dayjs()
  if (v.isSame(now, 'day')) return `${todayLabel} ${v.format('HH:mm')}`
  if (v.isSame(now, 'year')) return v.format('MM-DD HH:mm')
  return v.format('YYYY-MM-DD')
}

/** datetime-local 输入框的值（本地时间）。 */
export function toLocalInput(value: Date | string | undefined | null): string {
  const d = toDate(value)
  return d ? dayjs(d).format('YYYY-MM-DDTHH:mm') : ''
}

/** datetime-local 输入框的值转成 Date。 */
export function fromLocalInput(value: string): Date | undefined {
  if (!value) return undefined
  const d = dayjs(value)
  return d.isValid() ? d.toDate() : undefined
}

/** 本地时区名（输入时间旁的提示）。 */
export function localZone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || ''
  } catch {
    return ''
  }
}

const csvCell = (v: unknown) => {
  const s = v === undefined || v === null ? '' : String(v)
  return /[",\r\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

/** 下载 CSV（带 BOM，Excel 打开不乱码）。 */
export function downloadCsv(fileName: string, header: string[], rows: unknown[][]) {
  const text = [header, ...rows].map((row) => row.map(csvCell).join(',')).join('\r\n')
  const blob = new Blob(['﻿', text], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = fileName.endsWith('.csv') ? fileName : `${fileName}.csv`
  a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

/** 按页拉完全部数据（导出用）；pageSize 由调用方按接口上限给。 */
export async function fetchAllPages<T>(
  fetchPage: (page: number) => Promise<{ items: T[]; total: number }>,
  onProgress?: (done: number, total: number) => void,
): Promise<T[]> {
  const out: T[] = []
  for (let page = 1; ; page++) {
    const { items, total } = await fetchPage(page)
    out.push(...items)
    onProgress?.(out.length, total)
    if (!items.length || out.length >= total) break
  }
  return out
}

export interface RewardLabel {
  name: string
  image: string
}

/**
 * 奖励的名称与图片：资源类型按 games/gakumas/catalog.ts 对应到目录，名称先查前端名称表（当前语言 → 原文），
 * 再查目录（只查列表里实际出现的奖励，同一类成批查询并缓存）；从不显示 ID。
 */
export function useRewardLabels() {
  const ctx = useGmContext()
  const lookup = catalogLookupFor(ctx)

  const label = (r: GameReward): RewardLabel => {
    const ref = resourceRef(r.resourceType, r.resourceId)
    if (!ref) return { name: i18n.global.t('game.unnamed'), image: '' }
    return { name: lookup.name(ref.kind, ref.id) || i18n.global.t('game.unnamed'), image: lookup.image(ref.kind, ref.id) }
  }

  /** 「钻石 ×300、AP 回复药 ×3」 */
  const summary = (rewards: GameReward[] | undefined, sep = '、') =>
    (rewards || []).map((r) => `${label(r).name} ×${Number(r.quantity).toLocaleString()}`).join(sep)

  return { label, summary }
}
