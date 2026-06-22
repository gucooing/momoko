import dayjs from 'dayjs'
import { defineStore } from 'pinia'
import { imagineContext } from '@/api/imagine-context'
import {
  createImageGenGeneration,
  deleteImageGenGeneration,
  getImageGenGeneration,
  imageGenImageUrl,
  listImageGenApiKeys,
  listImageGenGenerations,
  listImageGenModels,
} from '@/api/sub2api-imagine'
import { showRequestError } from '@/utils/request'
import type {
  ImageGenApiKeySummary,
  ImageGenGeneration,
  ImageGenModel,
} from '@/types/v1/sub2api_imagine'

type Mode = 'text2image' | 'image2image'

export interface Params {
  shape: string // 1:1 | 4:3 | 3:4 | 16:9 | 9:16 | 3:2 | 2:3 | auto | custom
  scale: string // 1K | 2K | 4K
  customW: number
  customH: number
  n: number
  quality: string
  outputFormat: string
}

// 形状 → 宽高比（宽/高）。横向 >1，纵向 <1。
const SHAPE_RATIO: Record<string, number> = {
  '1:1': 1,
  '4:3': 4 / 3,
  '3:4': 3 / 4,
  '16:9': 16 / 9,
  '9:16': 9 / 16,
  '3:2': 3 / 2,
  '2:3': 2 / 3,
}
// 清晰度 → 长边像素。
const SCALE_LONG: Record<string, number> = { '1K': 1024, '2K': 2048, '4K': 4096 }

// computeSize 由形状参数解析出上游所需的 `宽x高` 字符串（或 auto）。store 与视图共用，避免重复实现。
export const computeSize = (p: Params): string => {
  if (p.shape === 'auto') return 'auto'
  if (p.shape === 'custom') return `${p.customW}x${p.customH}`
  const long = SCALE_LONG[p.scale] || 2048
  const ratio = SHAPE_RATIO[p.shape]
  if (!ratio) return '1024x1024'
  let w: number
  let h: number
  if (ratio >= 1) {
    w = long
    h = Math.round(long / ratio)
  } else {
    h = long
    w = Math.round(long * ratio)
  }
  w -= w % 2
  h -= h % 2
  return `${w}x${h}`
}

// toDataURL 把 Blob 读为 base64 data URL（data:image/png;base64,...），供整图提交。
const toDataURL = (blob: Blob): Promise<string> =>
  new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(blob)
  })

const POLL_INTERVAL = 1500
const POLL_MAX = 240 // ~6 分钟

export const useImagineStore = defineStore('sub2api-imagine', () => {
  const token = ref('')
  const srcHost = ref('')
  const userId = ref('')
  const theme = ref<'light' | 'dark'>('light')
  const embedded = ref(false)

  const apiKeys = ref<ImageGenApiKeySummary[]>([])
  const apiKeyId = ref('')
  const models = ref<ImageGenModel[]>([])
  const modelId = ref('')

  // 任务列表（轻量，不含图片字节）与当前详情
  const generations = ref<ImageGenGeneration[]>([])
  const currentId = ref('')
  const current = ref<ImageGenGeneration | null>(null)

  const mode = ref<Mode>('text2image')
  // 图生图源图：整张图片的 base64 data URL（自定义上传或已有结果转码），直接提交给后端。
  const sourceImage = ref('')
  const params = ref<Params>({
    shape: '1:1',
    scale: '2K',
    customW: 1024,
    customH: 1024,
    n: 1,
    quality: 'auto',
    outputFormat: 'png',
  })
  const prompt = ref('')
  const busy = ref(false)

  let pollHandle: ReturnType<typeof setInterval> | null = null

  const selectedApiKey = computed(() => apiKeys.value.find((k) => k.id === apiKeyId.value))
  const selectedModel = computed(() => models.value.find((m) => m.id === modelId.value))
  const hasPending = computed(() => generations.value.some((g) => g.status === 'pending'))

  const resolveSize = () => computeSize(params.value)

  // ---------- bootstrap ----------

  const bootstrap = (query: Record<string, unknown>): boolean => {
    token.value = String(query.token || '')
    srcHost.value = String(query.src_host || query.srcHost || '')
    userId.value = String(query.user_id || query.userId || '')
    const t = String(query.theme || '')
    if (t === 'light' || t === 'dark') theme.value = t
    embedded.value = String(query.ui_mode || '') === 'embedded'

    if (!token.value) {
      const target = srcHost.value || '/public/sub2api/home'
      ;(window.top ?? window).location.href = target
      return false
    }
    imagineContext.token = token.value
    imagineContext.srcHost = srcHost.value
    return true
  }

  const loadApiKeys = async () => {
    try {
      const res = await listImageGenApiKeys()
      apiKeys.value = res.apiKeys || []
      if (!apiKeyId.value && apiKeys.value.length) {
        const first = apiKeys.value[0]
        if (first) apiKeyId.value = first.id
      }
    } catch (e) {
      showRequestError(e)
    }
  }

  const loadModels = async () => {
    if (!apiKeyId.value) {
      models.value = []
      return
    }
    try {
      const res = await listImageGenModels({ apiKeyId: apiKeyId.value })
      models.value = res.models || []
      const stillValid = models.value.some((m) => m.id === modelId.value)
      if (!stillValid) {
        const first = models.value[0]
        modelId.value = first?.id || ''
      }
    } catch (e) {
      showRequestError(e)
    }
  }

  const selectApiKey = async (id: string) => {
    apiKeyId.value = id
    modelId.value = ''
    await loadModels()
  }

  // ---------- 任务列表 ----------

  const loadGenerations = async () => {
    try {
      const res = await listImageGenGenerations({ limit: 50 })
      generations.value = res.generations || []
    } catch (e) {
      // 列表加载静默失败，不打扰
    }
  }

  const selectGeneration = async (id: string) => {
    currentId.value = id
    current.value = null
    if (!id) return
    try {
      const res = await getImageGenGeneration(id)
      current.value = res.generation || null
    } catch (e) {
      showRequestError(e)
    }
  }

  const removeGeneration = async (id: string) => {
    try {
      await deleteImageGenGeneration(id)
      generations.value = generations.value.filter((g) => g.id !== id)
      if (currentId.value === id) {
        currentId.value = ''
        current.value = null
      }
    } catch (e) {
      showRequestError(e)
    }
  }

  // ---------- 生成 ----------

  const submitGeneration = async () => {
    if (!prompt.value.trim() || !apiKeyId.value || !modelId.value) return
    if (mode.value === 'image2image' && !sourceImage.value) return
    busy.value = true
    try {
      // 不再提交 mode：后端按是否带 sourceImage 自行判定文生图 / 图生图
      const res = await createImageGenGeneration({
        model: modelId.value,
        prompt: prompt.value.trim(),
        size: resolveSize(),
        n: Math.max(1, params.value.n || 1),
        quality: params.value.quality,
        outputFormat: params.value.outputFormat,
        sourceImage: mode.value === 'image2image' ? sourceImage.value : '',
        apiKeyId: apiKeyId.value,
      })
      generations.value.unshift(res.generation!)
      prompt.value = ''
      sourceImage.value = ''
      if (mode.value === 'image2image') mode.value = 'text2image'
      // 选中刚提交的任务看详情
      await selectGeneration(res.generation!.id)
      startPolling()
    } catch (e) {
      showRequestError(e)
    } finally {
      busy.value = false
    }
  }

  // setCustomSource 设置自定义上传的源图：读为 data URL 并切到图生图。
  const setCustomSource = async (file: File): Promise<boolean> => {
    try {
      sourceImage.value = await toDataURL(file)
      mode.value = 'image2image'
      return true
    } catch (e) {
      showRequestError(e)
      return false
    }
  }

  // clearSource 清除已选源图。
  const clearSource = () => {
    sourceImage.value = ''
  }

  // modifyImage 以某张已有结果图为源图：拉取图片字节转 data URL 后切到图生图。
  const modifyImage = async (imageId: string): Promise<boolean> => {
    try {
      const blob = await fetch(imageGenImageUrl(imageId)).then((r) => {
        if (!r.ok) throw new Error(`加载源图失败 (${r.status})`)
        return r.blob()
      })
      sourceImage.value = await toDataURL(blob)
      mode.value = 'image2image'
      prompt.value = ''
      return true
    } catch (e) {
      showRequestError(e)
      return false
    }
  }

  // ---------- 轮询：刷新列表 + 当前详情 ----------

  const startPolling = () => {
    if (pollHandle) return
    let ticks = 0
    pollHandle = setInterval(async () => {
      // 仅在有生成中任务时持续轮询；全部完成则停止
      if (!hasPending.value) {
        stopPolling()
        return
      }
      if (++ticks > POLL_MAX) {
        stopPolling()
        return
      }
      await refreshAll()
    }, POLL_INTERVAL)
  }

  const refreshAll = async () => {
    await loadGenerations()
    // 若当前详情是 pending，则刷新详情；否则用列表里同 id 的轻量信息更新
    if (currentId.value) {
      const inList = generations.value.find((g) => g.id === currentId.value)
      if (inList?.status === 'pending') {
        try {
          const res = await getImageGenGeneration(currentId.value)
          current.value = res.generation || null
        } catch {
          // ignore
        }
      } else if (current.value) {
        current.value = { ...current.value, status: inList?.status || current.value.status }
      }
    }
  }

  const stopPolling = () => {
    if (pollHandle) {
      clearInterval(pollHandle)
      pollHandle = null
    }
  }

  const formatDateTime = (value?: string) => {
    if (!value) return '-'
    const t = dayjs(value)
    return t.isValid() ? t.format('YYYY-MM-DD HH:mm') : '-'
  }

  const dispose = () => stopPolling()

  return {
    token,
    srcHost,
    userId,
    theme,
    embedded,
    apiKeys,
    apiKeyId,
    models,
    modelId,
    generations,
    currentId,
    current,
    selectedApiKey,
    selectedModel,
    mode,
    sourceImage,
    params,
    prompt,
    busy,
    hasPending,
    resolveSize,
    bootstrap,
    loadApiKeys,
    loadModels,
    selectApiKey,
    loadGenerations,
    selectGeneration,
    removeGeneration,
    submitGeneration,
    setCustomSource,
    clearSource,
    modifyImage,
    startPolling,
    stopPolling,
    refreshAll,
    formatDateTime,
    dispose,
  }
})
