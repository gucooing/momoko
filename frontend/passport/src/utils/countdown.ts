// 验证码重发倒计时。
import { onScopeDispose, ref } from 'vue'

export function useCountdown() {
  const left = ref(0)
  let timer: ReturnType<typeof setInterval> | null = null
  const stop = () => {
    if (timer) clearInterval(timer)
    timer = null
  }
  const start = (seconds: number) => {
    stop()
    left.value = seconds
    timer = setInterval(() => {
      left.value -= 1
      if (left.value <= 0) stop()
    }, 1000)
  }
  onScopeDispose(stop)
  return { left, start }
}
