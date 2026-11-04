import { onBeforeUnmount, ref } from 'vue'

/** 验证码重发间隔（秒，与后端 biz.EmailCodeResendInterval 一致）。 */
export const CODE_RESEND_SECONDS = 60
const DEFAULT_TTL_SECONDS = 300

/**
 * 邮箱验证码发送：倒计时 + 发送中 + 「已发送，N 分钟内有效」。
 * 接口失败时请求层已提示错误，这里只负责状态。
 */
export function useEmailCode(
  send: (email: string) => Promise<{ data?: { expiresInSeconds?: number | string } | null }>,
) {
  const countdown = ref(0)
  const sending = ref(false)
  /** 已发送时为验证码有效分钟数；0 表示还没发送 */
  const sentMinutes = ref(0)
  let timer: ReturnType<typeof setInterval> | null = null

  const clear = () => {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  const start = () => {
    countdown.value = CODE_RESEND_SECONDS
    clear()
    timer = setInterval(() => {
      if (countdown.value <= 1) {
        countdown.value = 0
        clear()
        return
      }
      countdown.value -= 1
    }, 1000)
  }

  const request = async (email: string) => {
    if (sending.value || countdown.value > 0) return false
    sending.value = true
    try {
      const { data } = await send(email)
      const ttl = Number(data?.expiresInSeconds) || DEFAULT_TTL_SECONDS
      sentMinutes.value = Math.max(1, Math.round(ttl / 60))
      start()
      return true
    } catch {
      return false
    } finally {
      sending.value = false
    }
  }

  /** 邮箱改了：清掉「已发送」与倒计时（后端按邮箱节流，换邮箱可以立即发送） */
  const reset = () => {
    sentMinutes.value = 0
    countdown.value = 0
    clear()
  }

  onBeforeUnmount(clear)

  return { countdown, sending, sentMinutes, request, reset }
}
