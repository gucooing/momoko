// 服务器时间：以 MGP status.now 为准（与本机时钟的差值保存下来），每 30 秒走一次；时区来自 GM 上下文。
import { computed, onScopeDispose, ref, watch, type ComputedRef, type Ref } from 'vue'
import type { GmContext } from '../../../context'
import { resolveZone } from './time'

export interface ServerClock {
  /** 当前服务器时间（Unix 毫秒） */
  now: Ref<number>
  /** 服务器时区（IANA） */
  zone: ComputedRef<string>
}

export function useServerClock(ctx: GmContext): ServerClock {
  const offset = ref(0)
  const now = ref(Date.now())
  watch(
    () => ctx.status.value?.now,
    (serverNow) => {
      if (serverNow) offset.value = serverNow - Date.now()
      now.value = Date.now() + offset.value
    },
    { immediate: true },
  )
  const timer = setInterval(() => {
    now.value = Date.now() + offset.value
  }, 30_000)
  onScopeDispose(() => clearInterval(timer))
  const zone = computed(() => resolveZone(ctx.timezone.value))
  return { now, zone }
}
