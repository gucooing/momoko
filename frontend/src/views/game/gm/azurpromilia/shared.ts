// 蓝色星原 GM 页面共用：按当前语言的名称、时间格式、游标分页、操作结果的转换。
import { computed, ref, shallowRef, type Ref } from 'vue'
import dayjs from 'dayjs'
import { getCurrentLocale, i18n } from '@/locales'
import { azurNames, kindOf } from '@/games/azurpromilia/names'
import type { OperationOutcome, OperationRow } from '@/games/mgp'
import type { GmExecution } from '@/types/v1/game_gm'
import type { OperationResponse } from '@/types/v1/azurpromilia/gm'

/** 按管理端当前语言显示的奖励与条目名称。 */
export const { kindName, rewardTypeName, itemName, rewardText, rewardsText, searchItems } = azurNames(getCurrentLocale)

/** 调整道具与货币时能扣除的种类：只有货币与普通道具（服务端规则）。 */
export const canTake = (type: number) => ['currency', 'common_item'].includes(kindOf(type))

/** Unix 毫秒 → 本地时间；0 为「—」。 */
export const fmtTime = (ms: number | undefined) => (ms ? dayjs(ms).format('YYYY-MM-DD HH:mm') : '—')

/** Unix 毫秒 → 日期时间输入框的值（datetime-local）。 */
export const toInput = (ms: number) => (ms ? dayjs(ms).format('YYYY-MM-DDTHH:mm') : '')
export const fromInput = (v: string) => (v ? dayjs(v).valueOf() : 0)

/** 逗号、空白或换行分隔的 pid。 */
export const parsePids = (text: string) => [...new Set(text.split(/[\s,，]+/).map((s) => s.trim()).filter((s) => /^\d+$/.test(s)))]

export const errorMessage = (e: unknown) => (e instanceof Error ? e.message : String(e))

/**
 * 游标分页：pages 里记着每一页的起始游标，只能逐页前后翻；load 读第 page 页。
 * fetch 返回 items、nextCursor、total。
 */
export function useCursorPages<T>(fetch: (cursor: string) => Promise<{ items: T[]; nextCursor: string; total: number }>) {
  const items = shallowRef<T[]>([])
  const total = ref(0)
  const page = ref(1)
  const loading = ref(false)
  const error = ref('')
  const cursors: Ref<string[]> = ref([''])
  const next = ref('')
  let seq = 0

  const load = async (n = page.value) => {
    const my = ++seq
    loading.value = true
    error.value = ''
    try {
      const res = await fetch(cursors.value[n - 1] || '')
      if (my !== seq) return
      items.value = res.items
      total.value = res.total
      next.value = res.nextCursor
      page.value = n
      if (res.nextCursor) cursors.value[n] = res.nextCursor
    } catch (e) {
      if (my === seq) error.value = errorMessage(e)
    } finally {
      if (my === seq) loading.value = false
    }
  }
  const reset = () => {
    cursors.value = ['']
    return load(1)
  }
  return {
    items,
    total,
    page,
    loading,
    error,
    hasPrev: computed(() => page.value > 1),
    hasNext: computed(() => !!next.value),
    load,
    reset,
    prev: () => load(page.value - 1),
    next: () => load(page.value + 1),
  }
}

/** 一次操作的响应 → GM 页面通用的操作结果（变化按奖励种类的表名与 ID 显示名称）。 */
export function outcomeOf(execution: GmExecution | undefined, res: OperationResponse['results'], dryRun: boolean): OperationOutcome {
  return {
    executionId: execution?.id || '',
    status: execution?.status || '',
    errorCode: execution?.errorCode,
    errorMessage: execution?.errorMessage,
    dryRun,
    backupRef: execution?.backupRef || undefined,
    results: res.map((r): OperationRow => {
      const row: OperationRow = {
        pid: r.target?.pid || '',
        name: r.target?.name || '',
        status: r.target?.status || '',
        code: r.target?.code,
        message: r.target?.message,
      }
      const d = r.data
      if (d?.changes?.length) {
        row.changes = d.changes.map((c) => ({
          kind: c.kind,
          resourceId: c.resourceId,
          before: c.counted ? c.before : 0,
          after: c.counted ? c.after : c.delta,
        }))
      } else if (d) {
        const detail: Record<string, unknown> = {}
        if (d.kicked) detail.kicked = true
        if (d.reason || d.until) {
          detail.until = d.until ? fmtTime(d.until) : i18n.global.t('azurGm.common.permanent')
          if (d.reason) detail.reason = d.reason
        }
        if (d.removed) detail.removed = true
        if (d.name) detail.name = d.name
        if (Object.keys(detail).length) row.detail = detail
      }
      return row
    }),
  }
}
