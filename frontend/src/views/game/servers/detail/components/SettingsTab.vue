<!-- 专用设置：适配器按服务端配置结构给出的表单（分组、类型、选项、取值范围、是否需要重启），或直接编辑 JSON。
     保存带上读取时的版本（内容哈希），被别处改过就拒绝；未识别的字段原样保留；可「仅保存」或「保存并重启」。 -->
<template>
  <div class="st">
    <EmptyState v-if="loadError && !settings" icon="HOutline:ExclamationTriangleIcon" :title="loadError" />
    <template v-else-if="settings">
      <nav v-if="mode === 'form' && !menuStore.isMobile" class="st__nav" :aria-label="t('gameServers.detail.tabs.settings')">
        <a v-for="sec in settings.sections" :key="sec.key" :href="`#gss-${sec.key}`" class="st__nav-item" @click.prevent="scrollTo(sec.key)">
          <span>{{ t(`gameServers.settings.sections.${sec.key}`, sec.key) }}</span>
          <span v-if="changedIn(sec.key)" class="st__chg m-num">{{ changedIn(sec.key) }}</span>
        </a>
      </nav>
      <div class="st__main">
        <div class="st__banner">
          <UIcon name="i-lucide-info" />
          <span class="st__banner-text">
            {{
              settings.unknownKeys.length
                ? t('gameServers.settings.banner', { file: settings.configFile, n: settings.unknownKeys.length })
                : t('gameServers.settings.bannerNoUnknown', { file: settings.configFile })
            }}
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
          <section v-for="sec in settings.sections" :id="`gss-${sec.key}`" :key="sec.key" class="st__card">
            <h2>{{ t(`gameServers.settings.sections.${sec.key}`, sec.key) }}</h2>
            <div v-for="f in sec.fields" :key="f.key" class="st__row">
              <span class="st__label">
                {{ fieldLabel(f.key) }}
                <small class="m-mono">{{ f.key }}</small>
              </span>
              <div class="st__ctl">
                <AppSwitch v-if="f.type === 'bool'" :model-value="draft[f.key] === true" :aria-label="fieldLabel(f.key)" @update:model-value="(v: boolean) => (draft[f.key] = v)" />
                <div v-else-if="f.type === 'enum'" class="st__seg" role="radiogroup" :aria-label="fieldLabel(f.key)">
                  <button
                    v-for="o in f.options"
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
                  @update:model-value="(v: string) => setList(f.key, v)"
                />
                <UInput
                  v-else-if="f.type === 'int'"
                  class="st__input st__input--num m-num"
                  type="number"
                  :model-value="numText(f.key)"
                  :min="f.hasMin ? f.min : undefined"
                  :max="f.hasMax ? f.max : undefined"
                  @update:model-value="(v) => setNum(f.key, v)"
                />
                <span v-else-if="f.managed" class="st__managed m-mono">{{ f.type === 'secret' ? '••••••••' : draft[f.key] || t('gameServers.settings.empty') }}</span>
                <UInput
                  v-else
                  class="st__input m-mono"
                  :type="f.type === 'secret' && !shown[f.key] ? 'password' : 'text'"
                  :model-value="strText(f.key)"
                  autocomplete="off"
                  @update:model-value="(v) => (draft[f.key] = String(v))"
                >
                  <template v-if="f.type === 'secret'" #trailing>
                    <button type="button" class="st__eye" :aria-label="shown[f.key] ? t('gameServers.settings.secretHide') : t('gameServers.settings.secretShow')" @click="shown[f.key] = !shown[f.key]">
                      <UIcon :name="shown[f.key] ? 'i-lucide-eye-off' : 'i-lucide-eye'" />
                    </button>
                  </template>
                </UInput>
                <span v-if="isChanged(f.key)" class="st__chg">{{ t('gameServers.settings.changed') }}</span>
                <small v-if="hint(f)" class="st__hint">{{ hint(f) }}</small>
              </div>
            </div>
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
import { computed, onActivated, onMounted, reactive, ref } from 'vue'
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

type Val = string | number | boolean | string[] | null | undefined
const settings = ref<GameServerSettings | null>(null)
const loadError = ref('')
const mode = ref<'form' | 'raw'>('form')
const original = ref<Record<string, Val>>({})
const draft = reactive<Record<string, Val>>({})
const shown = reactive<Record<string, boolean>>({})
const raw = ref('')
const saving = ref<'' | 'save' | 'restart'>('')

const lookup = (obj: unknown, path: string): Val => {
  let cur: unknown = obj
  for (const part of path.split('.')) {
    if (!cur || typeof cur !== 'object') return undefined
    cur = (cur as Record<string, unknown>)[part]
  }
  return cur as Val
}

const fields = computed(() => (settings.value?.sections || []).flatMap((s) => s.fields))

const load = async () => {
  try {
    const { data } = await getGameServerSettings(props.server.brief?.id || '')
    settings.value = data
    const vals: Record<string, Val> = {}
    for (const f of data.sections.flatMap((s) => s.fields)) {
      const v = lookup(data.values, f.key)
      vals[f.key] = Array.isArray(v) ? [...v] : v
    }
    original.value = vals
    for (const k of Object.keys(draft)) delete draft[k]
    Object.assign(draft, JSON.parse(JSON.stringify(vals)))
    raw.value = data.raw
    loadError.value = ''
  } catch (e) {
    loadError.value = getRequestErrorMessage(e, t('gameServers.common.failed'))
  }
}

const same = (a: Val, b: Val) => JSON.stringify(a ?? null) === JSON.stringify(b ?? null)
const isChanged = (key: string) => !same(draft[key], original.value[key])
const changedKeys = computed(() => fields.value.filter((f) => !f.managed && isChanged(f.key)).map((f) => f.key))
const changedIn = (section: string) => {
  const sec = settings.value?.sections.find((s) => s.key === section)
  return sec ? sec.fields.filter((f) => !f.managed && isChanged(f.key)).length : 0
}
const rawError = computed(() => {
  if (mode.value !== 'raw') return ''
  try {
    const v = JSON.parse(raw.value)
    return v && typeof v === 'object' && !Array.isArray(v) ? '' : 'object expected'
  } catch (e) {
    return (e as Error).message
  }
})
const dirtyCount = computed(() => (mode.value === 'raw' ? (raw.value !== settings.value?.raw ? 1 : 0) : changedKeys.value.length))

const strText = (key: string) => (draft[key] == null ? '' : String(draft[key]))
const numText = (key: string) => (draft[key] == null || draft[key] === '' ? '' : String(draft[key]))
const listText = (key: string) => (Array.isArray(draft[key]) ? (draft[key] as string[]).join('\n') : '')
const setList = (key: string, v: string) => {
  draft[key] = v.split('\n').map((s) => s.trim()).filter(Boolean)
}
const setNum = (key: string, v: string | number) => {
  const n = Number(v)
  draft[key] = v === '' || !Number.isFinite(n) ? '' : n
}

const fieldLabel = (key: string) => {
  const k = `gameServers.settings.fields.${key}`
  return te(k) ? t(k) : key
}
const hint = (f: GameSettingsField) => {
  if (f.managed) return t('gameServers.settings.managed')
  const k = `gameServers.settings.hints.${f.key}`
  const parts: string[] = []
  if (te(k)) parts.push(t(k))
  if (f.type === 'int' && f.hasMin && f.hasMax) parts.push(t('gameServers.settings.range', { min: f.min, max: f.max }))
  else if (f.type === 'int' && f.hasMin) parts.push(t('gameServers.settings.min', { n: f.min }))
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
      const f = fields.value.find((x) => x.key === key)
      let v = draft[key]
      if (f?.type === 'int' && v === '') {
        fb.warning(`${fieldLabel(key)}: ${t('gameServers.settings.empty')}`)
        return
      }
      if (v == null) v = f?.type === 'list' ? [] : ''
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

onMounted(load)
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
.st__managed {
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
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
