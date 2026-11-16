// 学マス 运营报表（dashboard 接口，近 14 天）：工作台、卡池、玩家三个标签共用。
// 同一台服务器 1 分钟内复用同一个请求（切标签不重复拉）；工作台的定时刷新用 force 跳过缓存。
import { nextTick, onMounted, ref, watch, type Ref } from 'vue'
import { getDashboard } from '@/api/gakumas'
import { usePaletteColors } from '@/composables/usePaletteColors'
import type { Dashboard } from '@/games/gakumas/types'

export const DASHBOARD_DAYS = 14
const TTL = 60_000
const cache = new Map<string, { at: number; p: Promise<Dashboard> }>()

/** 读报表；同一台服务器 TTL 内复用请求，force 时重新请求。 */
export function fetchDashboard(serverId: string, force = false): Promise<Dashboard> {
  const hit = cache.get(serverId)
  if (!force && hit && Date.now() - hit.at < TTL) return hit.p
  const p = getDashboard({ serverId, days: DASHBOARD_DAYS })
  cache.set(serverId, { at: Date.now(), p })
  p.catch(() => cache.delete(serverId))
  return p
}

/**
 * 图表颜色：游戏皮肤的 --m-gInk（从页面根元素读，弹窗挂在 body 上读不到游戏色，所以由页面传进去）；
 * 主题、深浅色变化时重新读取，读不到时用主题的 pInk。
 */
export function useGameChartColor(root: Ref<HTMLElement | undefined>) {
  const palette = usePaletteColors()
  const color = ref('')
  const read = () => {
    const el = root.value
    color.value = (el && getComputedStyle(el).getPropertyValue('--m-gInk').trim()) || palette.value.pInk
  }
  watch(palette, () => nextTick(read))
  onMounted(read)
  return color
}
