// 一个配置域（MGP /configs/{domain}）的编辑状态：服务端内容、草稿、自动保存、校验、发布与冲突处理。
// 内容以 JSON 文本往返：JSON.parse 后直接改对象，再 JSON.stringify 同一个对象发回，未改动的字段和键顺序原样保留。
import { computed, onBeforeUnmount, ref, shallowRef, toRaw, watch, type ComputedRef, type Ref, type ShallowRef } from 'vue'
import {
  discardGmConfigDraft,
  getGmConfig,
  publishGmConfig,
  saveGmConfigDraft,
  validateGmConfig,
} from '@/api/gameGm'
import type { GetGmConfigResponse, GmConfigDomain } from '@/types/v1/game_gm'
import { getRequestErrorMessage } from '@/utils/request'
import { useGmContext } from '../../../context'
import { diffJson, type DiffEntry } from './diff'

export interface ConfigValidation {
  valid: boolean
  error: string
  /** 校验时的内容（与当前内容不同说明校验结果已过期） */
  text: string
  /** 校验请求本身失败（服务端连不上等），不是内容无效 */
  failed: boolean
}

export interface ConfigDomainState<T> {
  domain: string
  serverId: string
  info: ShallowRef<GmConfigDomain | undefined>
  /** 服务端当前内容 */
  published: ShallowRef<T | undefined>
  /** 正在编辑的内容（草稿；没有草稿时是服务端内容的副本） */
  working: Ref<T | undefined>
  /** 编辑所基于的服务端 revision */
  baseRevision: Ref<string>
  loading: Ref<boolean>
  error: Ref<string>
  saving: Ref<boolean>
  saveError: Ref<string>
  savedAt: Ref<number>
  /** 与服务端内容不同 */
  dirty: ComputedRef<boolean>
  /** 草稿基于的 revision 已不是服务端当前的（别人发布过），直接发布会冲突 */
  stale: ComputedRef<boolean>
  /** 与服务端内容的改动 */
  changes: Ref<DiffEntry[]>
  validation: Ref<ConfigValidation | undefined>
  validating: Ref<boolean>
  /** 校验结果对应当前内容 */
  validationFresh: ComputedRef<boolean>
  /** 当前内容的紧凑 JSON 文本 */
  text: ComputedRef<string>
  load: () => Promise<void>
  flush: () => Promise<void>
  discard: () => Promise<void>
  validate: () => Promise<ConfigValidation | undefined>
  publish: (comment: string) => Promise<number | 'conflict'>
  rebase: () => Promise<void>
  restore: (content: string) => void
}

const AUTOSAVE_MS = 1200

const pretty = (v: unknown) => JSON.stringify(v, null, 2)
const errorCode = (e: unknown) => (e && typeof e === 'object' && 'code' in e ? (e as { code?: number }).code : undefined)

export function useConfigDomain<T>(domain: string): ConfigDomainState<T> {
  const ctx = useGmContext()
  const serverId = ctx.serverId.value

  const info = shallowRef<GmConfigDomain>()
  const published = shallowRef<T>()
  const working = ref() as Ref<T | undefined>
  const baseRevision = ref('')
  const loading = ref(false)
  const error = ref('')
  const saving = ref(false)
  const saveError = ref('')
  const savedAt = ref(0)
  const changes = ref<DiffEntry[]>([])
  const validation = ref<ConfigValidation>()
  const validating = ref(false)

  let publishedText = ''
  let lastSaved = ''
  let timer: ReturnType<typeof setTimeout> | undefined
  let inflight: Promise<void> | undefined

  const text = computed(() => (working.value === undefined ? '' : JSON.stringify(working.value)))
  const dirty = computed(() => !!text.value && text.value !== publishedText)
  const stale = computed(() => !!info.value && !!baseRevision.value && baseRevision.value !== info.value.revision)
  const validationFresh = computed(() => !!validation.value && validation.value.text === text.value)

  const recomputeChanges = () => {
    changes.value = dirty.value ? diffJson(published.value, toRaw(working.value)) : []
  }
  let diffTimer: ReturnType<typeof setTimeout> | undefined

  const apply = (data: GetGmConfigResponse, keepWorking: boolean) => {
    const parsed = JSON.parse(data.content || 'null') as T
    info.value = data.domain
    published.value = parsed
    publishedText = JSON.stringify(parsed)
    if (keepWorking) return
    const draft = data.domain?.draft
    if (draft?.content) {
      working.value = JSON.parse(draft.content) as T
      baseRevision.value = draft.baseRevision || data.domain?.revision || ''
      lastSaved = JSON.stringify(working.value)
    } else {
      working.value = JSON.parse(data.content || 'null') as T
      baseRevision.value = data.domain?.revision || ''
      lastSaved = publishedText
    }
  }

  const load = async () => {
    loading.value = true
    error.value = ''
    try {
      const { data } = await getGmConfig(serverId, domain)
      apply(data, false)
      validation.value = undefined
      recomputeChanges()
    } catch (e) {
      error.value = getRequestErrorMessage(e)
    } finally {
      loading.value = false
    }
  }

  const save = async () => {
    timer = undefined
    const current = text.value
    if (!current || current === lastSaved || working.value === undefined) return
    saving.value = true
    try {
      if (!dirty.value) {
        // 改回了与服务端一致：草稿没有意义了
        if (info.value?.draft) {
          await discardGmConfigDraft(serverId, domain)
          info.value = { ...info.value, draft: undefined }
        }
      } else {
        const { data } = await saveGmConfigDraft({
          serverId,
          domain,
          content: pretty(working.value),
          baseRevision: baseRevision.value,
          comment: '',
        })
        if (info.value) info.value = { ...info.value, draft: { ...data, content: '' } }
      }
      lastSaved = current
      savedAt.value = Date.now()
      saveError.value = ''
    } catch (e) {
      saveError.value = getRequestErrorMessage(e)
    } finally {
      saving.value = false
    }
  }

  const runSave = () => {
    const p = (inflight || Promise.resolve()).then(save)
    inflight = p.finally(() => {
      if (inflight === p) inflight = undefined
    })
    return p
  }

  watch(text, () => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => void runSave(), AUTOSAVE_MS)
    if (diffTimer) clearTimeout(diffTimer)
    diffTimer = setTimeout(recomputeChanges, 250)
  })

  const flush = async () => {
    if (timer) {
      clearTimeout(timer)
      timer = undefined
    }
    if (inflight) await inflight
    await runSave()
  }

  const discard = async () => {
    if (timer) clearTimeout(timer)
    timer = undefined
    if (inflight) await inflight
    if (info.value?.draft) await discardGmConfigDraft(serverId, domain)
    if (info.value) info.value = { ...info.value, draft: undefined }
    working.value = JSON.parse(publishedText || 'null') as T
    baseRevision.value = info.value?.revision || ''
    lastSaved = publishedText
    validation.value = undefined
    recomputeChanges()
  }

  const validate = async () => {
    if (working.value === undefined) return undefined
    const snapshot = text.value
    validating.value = true
    try {
      const { data } = await validateGmConfig(serverId, domain, pretty(working.value))
      validation.value = { valid: data.valid, error: data.error || '', text: snapshot, failed: false }
    } catch (e) {
      validation.value = { valid: false, error: getRequestErrorMessage(e), text: snapshot, failed: true }
    } finally {
      validating.value = false
    }
    return validation.value
  }

  // 服务端内容有变化（发布冲突）：更新服务端内容与 revision，保留正在编辑的内容
  const refreshServer = async () => {
    const { data } = await getGmConfig(serverId, domain)
    apply(data, true)
    recomputeChanges()
  }

  const publish = async (comment: string): Promise<number | 'conflict'> => {
    await flush()
    if (working.value === undefined) throw new Error('nothing to publish')
    try {
      const { data } = await publishGmConfig({
        serverId,
        domain,
        content: pretty(working.value),
        baseRevision: baseRevision.value,
        comment,
      })
      await load()
      return data.seq
    } catch (e) {
      if (errorCode(e) === 409) {
        await refreshServer()
        return 'conflict'
      }
      throw e
    }
  }

  // 以服务端当前版本为基础保留草稿（覆盖别人在此期间发布的改动）
  const rebase = async () => {
    if (!info.value) return
    baseRevision.value = info.value.revision
    lastSaved = ''
    await flush()
  }

  const restore = (content: string) => {
    working.value = JSON.parse(content) as T
    baseRevision.value = info.value?.revision || baseRevision.value
  }

  onBeforeUnmount(() => {
    if (diffTimer) clearTimeout(diffTimer)
    if (timer) {
      clearTimeout(timer)
      void runSave()
    }
  })

  return {
    domain,
    serverId,
    info,
    published,
    working,
    baseRevision,
    loading,
    error,
    saving,
    saveError,
    savedAt,
    dirty,
    stale,
    changes,
    validation,
    validating,
    validationFresh,
    text,
    load,
    flush,
    discard,
    validate,
    publish,
    rebase,
    restore,
  }
}
