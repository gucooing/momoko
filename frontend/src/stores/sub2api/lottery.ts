import { defineStore } from 'pinia'
import { lotteryContext } from '@/api/lottery-context'
import {
  getLotteryStatus,
  registerLottery,
  listLotteryHistoryPublic,
} from '@/api/sub2api-lottery-public'
import type {
  GetLotteryStatusResponse,
  LotteryHistoryItem,
} from '@/types/v1/sub2api'

export const useLotteryStore = defineStore('sub2api-lottery', () => {
  const token = ref('')
  const srcHost = ref('')
  const theme = ref<'light' | 'dark' | ''>('')

  const status = ref<GetLotteryStatusResponse | null>(null)
  const history = ref<LotteryHistoryItem[]>([])
  const historyPage = ref(1)
  const loading = ref(false)
  const registering = ref(false)

  const bootstrap = (query: Record<string, unknown>) => {
    token.value = String(query.token || '')
    srcHost.value = String(query.src_host || query.srcHost || '')
    const t = String(query.theme || '')
    if (t === 'light' || t === 'dark') theme.value = t

    if (!token.value) {
      const target = srcHost.value || '/public/sub2api/home'
      ;(window.top ?? window).location.href = target
      return false
    }
    lotteryContext.token = token.value
    lotteryContext.srcHost = srcHost.value
    return true
  }

  const loadStatus = async () => {
    loading.value = true
    try {
      status.value = await getLotteryStatus()
    } finally {
      loading.value = false
    }
  }

  const loadHistory = async (page = historyPage.value) => {
    historyPage.value = page
    const data = await listLotteryHistoryPublic({
      page: historyPage.value,
      pageSize: 10,
    })
    history.value = data.items || []
  }

  const register = async () => {
    registering.value = true
    try {
      status.value = await registerLottery()
      await loadHistory(1)
    } finally {
      registering.value = false
    }
  }

  return {
    token,
    srcHost,
    theme,
    status,
    history,
    historyPage,
    loading,
    registering,
    bootstrap,
    loadStatus,
    loadHistory,
    register,
  }
})
