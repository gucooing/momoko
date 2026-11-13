<!-- 专用设置：整个服务端配置文件。配置里的每一项都按它的值显示、都能改，也能新增或删除；
     适配器认识的字段补充说明、类型、选项与取值范围，不认识的按值的类型编辑（文本 / 数字 / 开关 / 列表 / JSON）。
     保存带上读取时的版本（内容哈希），被别处改过就拒绝；可「仅保存」或「保存并重启」，也可以直接编辑 JSON 原文。
     Momoko 以配置文件为准：保存后（或在文件管理里改了配置）接入信息（MGP 地址与密钥）按新配置更新。 -->
<template>
  <div class="st">
    <EmptyState v-if="loadError && !settings" icon="HOutline:ExclamationTriangleIcon" :title="loadError" />
    <template v-else-if="settings">
      <nav v-if="mode === 'form' && !menuStore.isMobile" class="st__nav" :aria-label="t('gameServers.detail.tabs.settings')">
        <a v-for="sec in sections" :key="sec.key" :href="`#gss-${sec.key}`" class="st__nav-item" @click.prevent="scrollTo(sec.key)">
          <span>{{ sec.label }}</span>
          <span v-if="changedIn(sec)" class="st__chg m-num">{{ changedIn(sec) }}</span>
        </a>
      </nav>
      <div class="st__main">
        <div class="st__banner">
          <UIcon name="i-lucide-info" />
          <span class="st__banner-text">
            {{ t('gameServers.settings.banner', { file: settings.configFile }) }}
            <small v-if="settings.modifiedAt">{{ t('gameServers.settings.modifiedAt', { t: fmtTime(settings.modifiedAt, true) }) }}</small>
          </span>
          <div class="st__seg" role="radiogroup">
            <button type="button" role="radio" :aria-checked="mode === 'form'" :class="{ 'is-on': mode === 'form' }" @click="setMode('form')">
              {{ t('gameServers.settings.formMode') }}
            </button>
            <button type="button" role="radio" :aria-checked="mode === 'raw'" :class="{ 'is-on': mode === 'raw' }" @click="setMode('raw')">
              {{ t('gameServers.settings.rawMode') }}
            </button>
          </div>
        </div>

        <template v-if="mode === 'form'">
          <section v-for="sec in sections" :id="`gss-${sec.key}`" :key="sec.key" class="st__card">
            <h2>{{ sec.label }}</h2>
            <div v-for="f in sec.fields" :key="f.key" class="st__row" :class="{ 'is-removed': removed.has(f.key) }">
              <span class="st__label">
                {{ fieldLabel(f.key) }}
                <small class="m-mono">{{ f.key }}</small>
              </span>
              <div class="st__ctl">
                <template v-if="removed.has(f.key)">
                  <span class="st__muted">{{ t('gameServers.settings.removed') }}</span>
                  <UButton size="xs" color="neutral" variant="outline" @click="removed.delete(f.key)">{{ t('gameServers.settings.undo') }}</UButton>
                </template>
                <template v-else>
                  <AppSwitch v-if="f.type === 'bool'" :model-value="draft[f.key] === true" :aria-label="fieldLabel(f.key)" @update:model-value="(v: boolean) => (draft[f.key] = v)" />
                  <div v-else-if="f.type === 'enum'" class="st__seg" role="radiogroup" :aria-label="fieldLabel(f.key)">
                    <button
                      v-for="o in f.hint?.options || []"
                      :key="o"
                      type="button"
                      role="radio"
                      :aria-checked="draft[f.key] === o"
                      :class="{ 'is-on': draft[f.key] === o }"
                      @click="draft[f.key] = o"
                    >
                      {{ o }}
                    </button>
                  </div>
                  <UTextarea
                    v-else-if="f.type === 'list'"
                    class="st__input m-mono"
                    :model-value="listText(f.key)"
                    :rows="3"
                    :placeholder="t('gameServers.settings.listPh')"
                    @update:model-value="(v: string) => setList(f, v)"
                  />
                  <UTextarea
                    v-else-if="f.type === 'json'"
                    class="st__input m-mono"
                    :model-value="jsonText[f.key] ?? ''"
                    :rows="4"
                    @update:model-value="(v: string) => (jsonText[f.key] = v)"
                  />
                  <UInput
                    v-else-if="f.type === 'int' || f.type === 'number'"
                    class="st__input st__input--num m-num"
                    type="number"
                    :model-value="numText(f.key)"
                    :min="f.hint?.hasMin ? f.hint.min : undefined"
                    :max="f.hint?.hasMax ? f.hint.max : undefined"
                    :placeholder="draft[f.key] === undefined ? t('gameServers.settings.unset') : undefined"
                    @update:model-value="(v) => setNum(f.key, v)"
                  />
                  <UInput
                    v-else
                    class="st__input m-mono"
                    :type="f.type === 'secret' && !shown[f.key] ? 'password' : 'text'"
                    :model-value="strText(f.key)"
                    :placeholder="draft[f.key] === undefined ? t('gameServers.settings.unset') : undefined"
                    autocomplete="off"
                    @update:model-value="(v) => (draft[f.key] = String(v))"
                  >
                    <template v-if="f.type === 'secret'" #trailing>
                      <button type="button" class="st__eye" :aria-label="shown[f.key] ? t('gameServers.settings.secretHide') : t('gameServers.settings.secretShow')" @click="shown[f.key] = !shown[f.key]">
                        <UIcon :name="shown[f.key] ? 'i-lucide-eye-off' : 'i-lucide-eye'" />
                      </button>
                    </template>
                  </UInput>
                  <AppIconButton
                    v-if="f.present"
                    icon="HOutline:TrashIcon"
                    :label="t('gameServers.settings.remove')"
                    :box="36"
                    @click="remove(f.key)"
                  />
                </template>
                <span v-if="isChanged(f.key)" class="st__chg">{{ t('gameServers.settings.changed') }}</span>
                <small v-if="hint(f)" class="st__hint">{{ hint(f) }}</small>
              </div>
            </div>
          </section>

          <section class="st__card">
            <h2>{{ t('gameServers.settings.addTitle') }}</h2>
            <div class="st__add">
              <UInput v-model="addPath" class="st__input m-mono" :placeholder="t('gameServers.settings.addPathPh')" :aria-label="t('gameServers.settings.addPath')" />
              <UInput v-model="addValue" class="st__input m-mono" :placeholder="t('gameServers.settings.addValue')" :aria-label="t('gameServers.settings.addValue')" />
              <UButton color="neutral" variant="outline" icon="i-lucide-plus" :disabled="!addPath.trim()" @click="addField">{{ t('gameServers.settings.add') }}</UButton>
            </div>
            <small class="st__hint">{{ t('gameServers.settings.addHint') }}</small>
          </section>
        </template>

        <section v-else class="st__card">
          <textarea v-model="raw" class="st__raw m-mono" spellcheck="false" :aria-label="settings.configFile" />
          <p v-if="rawError" class="st__error">{{ t('gameServers.settings.rawInvalid', { msg: rawError }) }}</p>
        </section>

        <div v-if="dirtyCount" class="st__bar">
          <span class="st__bar-text">
            {{ t('gameServers.settings.changes', { n: dirtyCount }) }}<template v-if="settings.running"> · {{ t('gameServers.settings.needRestart') }}</template>
          </span>
          <UButton size="sm" color="neutral" variant="ghost" class="st__bar-ghost" @click="revert">{{ t('gameServers.settings.revert') }}</UButton>
          <UButton size="sm" color="neutral" variant="solid" :loading="saving === 'save'" :disabled="!!saving" @click="save(false)">
            {{ t('gameServers.settings.saveOnly') }}
          </UButton>
          <UButton v-if="settings.running" size="sm" color="primary" :loading="saving === 'restart'" :disabled="!!saving" @click="save(true)">
            {{ t('gameServers.settings.saveRestart') }}
          </UButton>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGameServerSettings, saveGameServerSettings } from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import type { GameAdapter, GameServerInfo, GameServerSettings, GameSettingsField } from '@/types/v1/game_server'
import { fmtTime } from '../../shared'

const props = defineProps<{ server: GameServerInfo; adapter?: GameAdapter | null }>()
const emit = defineEmits<{ changed: [] }>()
const { t, te } = useI18n()
const fb = useFeedback()
const menuStore = useMenuStore()

type Json = string | number | boolean | null | Json[] | { [k: string]: Json }
type Val = Json | undefined
/** 一项配置（叶子）：type 优先用适配器的提示，没有提示时按值推断 */
interface Field {
  key: string
  type: string
  hint?: GameSettingsField
  /** 配置文件里有这一项（可以删除） */
  present: boolean
  /** 列表的元素是数字 */
  numericList?: boolean
}
interface Section {
  key: string
  label: string
  fields: Field[]
}

const settings = ref<GameServerSettings | null>(null)
const loadError = ref('')
const mode = ref<'form' | 'raw'>('form')
const original = ref<Record<string, Val>>({})
const draft = reactive<Record<string, Val>>({})
const jsonText = reactive<Record<string, string>>({})
const originalJson = ref<Record<string, string>>({})
const removed = reactive(new Set<string>())
const added = ref<string[]>([])
const shown = reactive<Record<string, boolean>>({})
const raw = ref('')
const saving = ref<'' | 'save' | 'restart'>('')
const addPath = ref('')
const addValue = ref('')

const isPlainObject = (v: unknown): v is Record<string, Json> => !!v && typeof v === 'object' && !Array.isArray(v)

// 配置的叶子（非对象的值；空对象也算一项）
const leaves = (obj: Record<string, Json>, prefix = '', out: Record<string, Json> = {}) => {
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k
    if (isPlainObject(v) && Object.keys(v).length) leaves(v, key, out)
    else out[key] = v
  }
  return out
}

const hints = computed(() => {
  const m = new Map<string, { field: GameSettingsField; section: string }>()
  for (const sec of settings.value?.sections || []) for (const f of sec.fields) m.set(f.key, { field: f, section: sec.key })
  return m
})

const inferType = (v: Val): { type: string; numericList?: boolean } => {
  if (typeof v === 'boolean') return { type: 'bool' }
  if (typeof v === 'number') return { type: 'number' }
  if (typeof v === 'string') return { type: 'string' }
  if (Array.isArray(v) && v.every((x) => typeof x === 'string' || typeof x === 'number')) {
    return { type: 'list', numericList: v.length > 0 && v.every((x) => typeof x === 'number') }
  }
  if (v === undefined) return { type: 'string' }
  return { type: 'json' }
}

const fieldOf = (key: string, present: boolean): Field => {
  const h = hints.value.get(key)?.field
  if (h) return { key, type: h.type, hint: h, present }
  return { key, present, ...inferType(present ? original.value[key] ?? draft[key] : draft[key]) }
}

// 分组：适配器提示里的字段归到提示的分组；其余对象里的字段按顶层键分组，顶层的零散字段归「其他」
const sections = computed<Section[]>(() => {
  const keys = new Set<string>([...Object.keys(original.value), ...hints.value.keys(), ...added.value])
  const groups = new Map<string, Field[]>()
  const order: string[] = (settings.value?.sections || []).map((s) => s.key)
  for (const key of keys) {
    const present = key in original.value || added.value.includes(key)
    const group = hints.value.get(key)?.section || (key.includes('.') ? key.slice(0, key.indexOf('.')) : 'other')
    if (!groups.has(group)) {
      groups.set(group, [])
      if (!order.includes(group)) order.push(group)
    }
    groups.get(group)!.push(fieldOf(key, present))
  }
  const rest = order.filter((g) => g !== 'other')
  if (groups.has('other')) rest.push('other')
  return rest
    .filter((g) => groups.get(g)?.length)
    .map((g) => ({ key: g, label: sectionLabel(g), fields: groups.get(g)! }))
})
const fields = computed(() => sections.value.flatMap((s) => s.fields))

const sectionLabel = (key: string) => {
  const k = `gameServers.settings.sections.${key}`
  return te(k) ? t(k) : fieldLabel(key)
}

const load = async () => {
  try {
    const { data } = await getGameServerSettings(props.server.brief?.id || '')
    settings.value = data
    const vals = isPlainObject(data.values) ? leaves(data.values as Record<string, Json>) : {}
    original.value = vals
    for (const k of Object.keys(draft)) delete draft[k]
    for (const k of Object.keys(jsonText)) delete jsonText[k]
    Object.assign(draft, JSON.parse(JSON.stringify(vals)))
    const texts: Record<string, string> = {}
    for (const [k, v] of Object.entries(vals)) if (inferType(v).type === 'json' && !hints.value.has(k)) texts[k] = JSON.stringify(v, null, 2)
    originalJson.value = texts
    Object.assign(jsonText, texts)
    removed.clear()
    added.value = []
    raw.value = data.raw
    loadError.value = ''
  } catch (e) {
    loadError.value = getRequestErrorMessage(e, t('gameServers.common.failed'))
  }
}

const same = (a: Val, b: Val) => JSON.stringify(a ?? null) === JSON.stringify(b ?? null)
const isChanged = (key: string) => {
  if (removed.has(key) || added.value.includes(key)) return true
  if (key in jsonText) return jsonText[key] !== originalJson.value[key]
  return !same(draft[key], original.value[key])
}
const changedKeys = computed(() => fields.value.filter((f) => isChanged(f.key)).map((f) => f.key))
const changedIn = (sec: Section) => sec.fields.filter((f) => isChanged(f.key)).length
const rawError = computed(() => {
  if (mode.value !== 'raw') return ''
  try {
    const v = JSON.parse(raw.value)
    return isPlainObject(v) ? '' : 'object expected'
  } catch (e) {
    return (e as Error).message
  }
})
const dirtyCount = computed(() => (mode.value === 'raw' ? (raw.value !== settings.value?.raw ? 1 : 0) : changedKeys.value.length))

const strText = (key: string) => (draft[key] == null ? '' : String(draft[key]))
const numText = (key: string) => (draft[key] == null || draft[key] === '' ? '' : String(draft[key]))
const listText = (key: string) => (Array.isArray(draft[key]) ? (draft[key] as Json[]).join('\n') : '')
const setList = (f: Field, v: string) => {
  const items = v.split('\n').map((s) => s.trim()).filter(Boolean)
  draft[f.key] = f.numericList && items.every((x) => Number.isFinite(Number(x))) ? items.map(Number) : items
}
const setNum = (key: string, v: string | number) => {
  const n = Number(v)
  draft[key] = v === '' || !Number.isFinite(n) ? '' : n
}

const remove = (key: string) => {
  if (added.value.includes(key)) {
    added.value = added.value.filter((k) => k !== key)
    delete draft[key]
    delete jsonText[key]
    return
  }
  removed.add(key)
}

// 新增一项：值按 JSON 解析（数字、true / false、[…]、{…}、"…"），解析不了就当文本
const addField = () => {
  const key = addPath.value.trim()
  if (!/^[^.\s]+(\.[^.\s]+)*$/.test(key) || fields.value.some((f) => f.key === key && f.present)) {
    fb.warning(t('gameServers.settings.pathInvalid'))
    return
  }
  // 不能和已有的对象或值冲突（例如 log 是对象时不能新增 log 这一项）
  if (fields.value.some((f) => f.key.startsWith(`${key}.`) || key.startsWith(`${f.key}.`))) {
    fb.warning(t('gameServers.settings.pathInvalid'))
    return
  }
  let v: Json = addValue.value
  try {
    v = JSON.parse(addValue.value) as Json
  } catch {
    // 当文本
  }
  draft[key] = v
  if (inferType(v).type === 'json') jsonText[key] = JSON.stringify(v, null, 2)
  added.value = [...added.value, key]
  addPath.value = ''
  addValue.value = ''
}

const fieldLabel = (key: string) => {
  const k = `gameServers.settings.fields.${key}`
  return te(k) && typeof t(k) === 'string' && t(k) !== k ? t(k) : key
}
const hint = (f: Field) => {
  if (!f.present && !f.hint) return ''
  const parts: string[] = []
  const k = `gameServers.settings.hints.${f.key}`
  if (te(k)) parts.push(t(k))
  const h = f.hint
  if (h && h.type === 'int' && h.hasMin && h.hasMax) parts.push(t('gameServers.settings.range', { min: h.min, max: h.max }))
  else if (h && h.type === 'int' && h.hasMin) parts.push(t('gameServers.settings.min', { n: h.min }))
  if (!f.present) parts.push(t('gameServers.settings.unset'))
  return parts.join(' · ')
}

const setMode = (m: 'form' | 'raw') => {
  if (m === mode.value) return
  if (dirtyCount.value) revert()
  mode.value = m
}
const revert = () => {
  for (const k of Object.keys(draft)) delete draft[k]
  Object.assign(draft, JSON.parse(JSON.stringify(original.value)))
  for (const k of Object.keys(jsonText)) delete jsonText[k]
  Object.assign(jsonText, originalJson.value)
  removed.clear()
  added.value = []
  raw.value = settings.value?.raw || ''
}
const scrollTo = (key: string) => document.getElementById(`gss-${key}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })

const save = async (restart: boolean) => {
  const s = settings.value
  if (!s) return
  if (mode.value === 'raw' && rawError.value) return
  const values: Record<string, unknown> = {}
  if (mode.value === 'form') {
    for (const key of changedKeys.value) {
      if (removed.has(key)) {
        values[key] = null
        continue
      }
      const f = fields.value.find((x) => x.key === key)
      if (key in jsonText) {
        try {
          values[key] = JSON.parse(jsonText[key] || 'null')
        } catch {
          fb.warning(t('gameServers.settings.jsonInvalid', { key: fieldLabel(key) }))
          return
        }
        continue
      }
      let v = draft[key]
      if ((f?.type === 'int' || f?.type === 'number') && v === '') {
        fb.warning(`${fieldLabel(key)}: ${t('gameServers.settings.empty')}`)
        return
      }
      if (v === undefined) v = f?.type === 'list' ? [] : ''
      values[key] = v
    }
  }
  saving.value = restart ? 'restart' : 'save'
  try {
    const { data } = await saveGameServerSettings({
      id: props.server.brief?.id || '',
      revision: s.revision,
      ...(mode.value === 'raw' ? { raw: raw.value } : { values }),
      restart,
    })
    if (!data.changedKeys.length) fb.success(t('gameServers.settings.noChanges'))
    else if (data.restarted) fb.success(t('gameServers.settings.savedRestart'))
    else if (data.pendingRestart) fb.success(t('gameServers.settings.savedPending'))
    else fb.success(t('gameServers.settings.saved', { n: data.changedKeys.length }))
    await load()
    emit('changed')
  } catch (e) {
    const msg = getRequestErrorMessage(e, t('gameServers.common.failed'))
    fb.error(msg)
    // 版本冲突：重新读取
    if ((e as { response?: { status?: number } })?.response?.status === 409) {
      fb.warning(t('gameServers.settings.conflict'))
      await load()
    }
  } finally {
    saving.value = ''
  }
}

// 标签页在 KeepAlive 里，首次挂载也会触发 onActivated：只在这里加载，避免挂载时请求两次
onActivated(() => {
  if (!dirtyCount.value) load()
})
</script>

<style scoped lang="scss">
.st {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
.st__nav {
  flex: 0 0 200px;
  position: sticky;
  top: 16px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px;
  border-radius: var(--app-radius-lg);
  border: var(--cmp-card-border-width, 1px) solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.st__nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 12px;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-regular);
  font-size: 0.875rem;
  font-weight: 600;
  text-decoration: none;
}
.st__nav-item span:first-child {
  flex: 1;
}
.st__nav-item:hover {
  background: var(--el-fill-color-light);
}
.st__main {
  flex: 1 1 560px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.st__banner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: var(--app-radius);
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
  font-size: 0.8125rem;
  font-weight: 600;
}
.st__banner-text {
  flex: 1 1 280px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.st__banner-text small {
  font-weight: 500;
  opacity: 0.8;
}
.st__seg {
  display: inline-flex;
  flex-wrap: wrap;
  padding: 3px;
  gap: 2px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.st__seg button {
  min-height: 34px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.st__seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.st__card {
  padding: 6px 22px 10px;
  border-radius: var(--app-radius-lg);
  border: var(--cmp-card-border-width, 1px) solid var(--el-border-color-light);
  background: var(--el-bg-color);
  scroll-margin-top: 16px;
}
.st__card h2 {
  margin: 14px 0 6px;
  font-size: 1.0625rem;
  font-weight: 800;
}
.st__row {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 18px;
  align-items: center;
  padding: 14px 0;
  border-top: 1px solid var(--el-border-color-light);
}
.st__label {
  display: flex;
  flex-direction: column;
  gap: 3px;
  font-size: 0.875rem;
  font-weight: 700;
  min-width: 0;
}
.st__label small {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.st__ctl {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 10px;
  min-width: 0;
}
.st__input {
  flex: 1 1 260px;
  min-width: 0;
}
.st__input--num {
  flex: 0 1 180px;
}
.st__eye {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.st__chg {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: 6px;
  background: var(--m-aSoft);
  color: var(--m-ink);
  font-size: 0.6875rem;
  font-weight: 700;
}
.st__muted {
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.st__row.is-removed .st__label {
  text-decoration: line-through;
  opacity: 0.6;
}
.st__add {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  padding: 10px 0 6px;
}
.st__hint {
  flex-basis: 100%;
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.st__raw {
  width: 100%;
  min-height: 520px;
  margin: 14px 0;
  padding: 12px 14px;
  border-radius: var(--app-radius-sm);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
  line-height: 1.6;
  resize: vertical;
  tab-size: 2;
}
.st__raw:focus {
  outline: none;
  border-color: var(--m-p);
}
.st__error {
  margin: 0 0 12px;
  font-size: 0.8125rem;
  color: var(--m-bad);
}
.st__bar {
  position: sticky;
  bottom: 20px;
  z-index: 2;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 10px 10px 10px 20px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: var(--app-shadow-brand);
}
.st__bar-text {
  flex: 1;
  font-size: 0.875rem;
  font-weight: 700;
}
.st__bar .st__bar-ghost {
  color: var(--m-bg);
}
@media (width <= 768px) {
  .st__row {
    grid-template-columns: 1fr;
    gap: 8px;
  }
  .st__card {
    padding: 4px 16px 8px;
  }
  .st__bar {
    border-radius: var(--app-radius-lg);
    bottom: 12px;
  }
  .st__bar :deep(button) {
    min-height: 44px;
  }
}
</style>
