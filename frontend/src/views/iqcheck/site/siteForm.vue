<!-- 智商检测 · 站点 创建/编辑弹窗：名称/站点/API Key/User-Agent/协议 + 拉取模型列表勾选（每个模型单独设推理强度）
     + 定时检测开关/间隔/次数。编辑时 API Key 留空表示保留原值，可用已保存的 Key 拉取模型列表。 -->
<template>
  <FormDialog
    v-model="open"
    :title="form.id ? t('iqcheck.site.editTitle') : t('iqcheck.site.createTitle')"
    :width="760"
    :loading="saving"
    @confirm="confirm"
    @close="close"
  >
    <div class="iqf">
      <!-- 站点 -->
      <section class="iqf-sec">
        <header class="iqf-sec__head">
          <span class="m-sticker m-sticker--p iqf-sec__ico" style="rotate: -4deg">
            <component :is="menuStore.iconComponents['HOutline:GlobeAltIcon']" />
          </span>
          <span class="iqf-sec__text">
            <h4>{{ t('publicPage.siteForm.secSite') }}</h4>
            <p>{{ t('publicPage.siteForm.secSiteDesc') }}</p>
          </span>
        </header>
        <div class="iqf-grid">
          <div class="app-field">
            <label class="app-label app-label--required">{{ t('iqcheck.site.form.name') }}</label>
            <input
              v-model="form.name"
              class="app-input iqf-input"
              :class="{ 'is-error': errors.name }"
              :placeholder="t('iqcheck.site.form.namePlaceholder')"
            />
            <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('iqcheck.site.form.protocol') }}</label>
            <AppSelect v-model="form.protocol" :options="protocolOptions" />
          </div>
        </div>
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('iqcheck.site.form.endpoint') }}</label>
          <input
            v-model="form.endpoint"
            class="app-input iqf-input m-mono"
            :class="{ 'is-error': errors.endpoint }"
            :placeholder="t('iqcheck.site.form.endpointPlaceholder')"
            spellcheck="false"
          />
          <span v-if="errors.endpoint" class="app-field__error">{{ errors.endpoint }}</span>
          <span v-else class="iqf-hint">{{ t('iqcheck.site.form.endpointHint') }}</span>
        </div>
        <div class="iqf-grid">
          <div class="app-field">
            <label class="app-label" :class="{ 'app-label--required': !form.id }">
              {{ t('iqcheck.site.form.apiKey') }}
            </label>
            <input
              v-model="form.apiKey"
              type="password"
              autocomplete="off"
              class="app-input iqf-input m-mono"
              :class="{ 'is-error': errors.apiKey }"
              :placeholder="form.id ? maskedApiKey : t('iqcheck.site.form.apiKeyPlaceholder')"
            />
            <span v-if="errors.apiKey" class="app-field__error">{{ errors.apiKey }}</span>
            <span v-else-if="form.id" class="iqf-hint">{{ t('iqcheck.site.form.apiKeyKeepHint') }}</span>
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('iqcheck.site.form.userAgent') }}</label>
            <input v-model="form.userAgent" class="app-input iqf-input m-mono" placeholder="pi" spellcheck="false" />
            <span class="iqf-hint">{{ t('iqcheck.site.form.userAgentHint') }}</span>
          </div>
        </div>
      </section>

      <!-- 模型 -->
      <section class="iqf-sec">
        <header class="iqf-sec__head">
          <span class="m-sticker m-sticker--a2 iqf-sec__ico" style="rotate: 3deg">
            <component :is="menuStore.iconComponents['HOutline:CpuChipIcon']" />
          </span>
          <span class="iqf-sec__text">
            <h4>
              {{ t('iqcheck.site.form.models') }}
              <span class="iqf-count m-num">{{ form.models.length }}</span>
            </h4>
            <p>{{ t('iqcheck.site.form.fetchHint') }}</p>
          </span>
          <UButton
            color="neutral"
            variant="soft"
            class="iqf-btn"
            icon="i-lucide-download"
            :loading="fetching"
            :disabled="!canFetch"
            @click="fetchModels"
          >
            {{ fetching ? t('iqcheck.site.form.fetching') : t('iqcheck.site.form.fetchModels') }}
          </UButton>
        </header>

        <div v-if="form.models.length" class="iqf-picked" :aria-label="t('iqcheck.site.form.selectedModels', { count: form.models.length })">
          <span v-for="m in form.models" :key="m.model" class="iqf-picked__chip">
            <span class="m-mono">{{ m.model }}</span>
            <span v-if="m.effort" class="iqf-picked__eff">{{ m.effort }}</span>
            <button
              type="button"
              class="iqf-picked__x"
              :aria-label="`${t('publicPage.siteForm.remove')} ${m.model}`"
              @click="toggle(m.model)"
            >
              <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
            </button>
          </span>
        </div>

        <div class="iqf-models">
          <div class="iqf-models__tools">
            <span class="iqf-search">
              <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" />
              <input v-model="search" class="iqf-search__input" :placeholder="t('iqcheck.site.form.searchModels')" />
            </span>
            <div class="iqf-manual">
              <input
                v-model="manualModel"
                class="app-input iqf-input m-mono"
                :placeholder="t('iqcheck.site.form.manualModel')"
                @keyup.enter="addManual"
              />
              <UButton color="neutral" variant="soft" class="iqf-btn" icon="i-lucide-plus" :disabled="!manualModel.trim()" @click="addManual">
                {{ t('iqcheck.site.form.add') }}
              </UButton>
            </div>
          </div>

          <div v-if="!rows.length" class="iqf-models__empty">
            <component :is="menuStore.iconComponents['HOutline:CpuChipIcon']" />
            {{ t('iqcheck.site.form.noModels') }}
          </div>
          <ul v-else class="iqf-list">
            <li v-for="model in rows" :key="model" class="iqf-item" :class="{ 'is-selected': isSelected(model) }">
              <label class="iqf-item__check">
                <input type="checkbox" :checked="isSelected(model)" @change="toggle(model)" />
                <span class="iqf-item__model m-mono" :title="model">{{ model }}</span>
              </label>
              <select
                class="app-select iqf-item__effort"
                :disabled="!isSelected(model)"
                :value="effortOf(model)"
                :title="t('iqcheck.site.form.effort')"
                :aria-label="`${t('iqcheck.site.form.effort')} ${model}`"
                @change="setEffort(model, ($event.target as HTMLSelectElement).value)"
              >
                <option v-for="opt in effortOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
              </select>
            </li>
          </ul>
        </div>
      </section>

      <!-- 定时检测 -->
      <section class="iqf-sec">
        <div class="iqf-switch">
          <span class="m-sticker m-sticker--ok iqf-sec__ico" style="rotate: -2deg">
            <component :is="menuStore.iconComponents['HOutline:ClockIcon']" />
          </span>
          <span class="iqf-sec__text">
            <h4>{{ t('iqcheck.site.form.enabled') }}</h4>
            <p>{{ form.enabled ? t('publicPage.siteForm.scheduleOn', { minutes: form.intervalMinutes || 60 }) : t('publicPage.siteForm.scheduleOff') }}</p>
          </span>
          <AppSwitch v-model="form.enabled" :aria-label="t('iqcheck.site.form.enabled')" />
        </div>
        <div class="iqf-grid" :class="{ 'is-dim': !form.enabled }">
          <div class="app-field">
            <label class="app-label">{{ t('iqcheck.site.form.interval') }}</label>
            <input v-model.number="form.intervalMinutes" type="number" min="1" class="app-input iqf-input m-num" />
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('iqcheck.site.form.maxRuns') }}</label>
            <input v-model.number="form.maxRuns" type="number" min="0" class="app-input iqf-input m-num" />
          </div>
        </div>
        <label v-if="form.id" class="iqf-reset">
          <input v-model="form.resetRunCount" type="checkbox" />
          <span>
            {{ t('iqcheck.site.form.resetRunCount') }}
            <em class="m-num">{{ t('iqcheck.site.form.runCount', { count: form.runCount }) }}</em>
          </span>
        </label>
      </section>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { createIQCheckSite, fetchIQCheckModels, updateIQCheckSite } from '@/api/iqcheck'
import { IQCheckProtocol, type IQCheckModelConfig, type IQCheckSiteInfo } from '@/types/v1/iqcheck'

defineOptions({ name: 'IQCheckSiteForm' })

const emit = defineEmits<{ refresh: [] }>()
const { t } = useI18n()
const menuStore = useMenuStore()

const open = ref(false)
const saving = ref(false)
const fetching = ref(false)
const errors = ref<Record<string, string>>({})
const available = ref<string[]>([])
const search = ref('')
const manualModel = ref('')
const maskedApiKey = ref('')

const emptyForm = () => ({
  id: '',
  name: '',
  endpoint: '',
  apiKey: '',
  userAgent: '',
  protocol: IQCheckProtocol.IQ_CHECK_PROTOCOL_CHAT as IQCheckProtocol,
  models: [] as IQCheckModelConfig[],
  enabled: true,
  intervalMinutes: 60,
  maxRuns: 0,
  resetRunCount: false,
  runCount: 0,
})
const form = ref(emptyForm())

const protocolOptions = computed(() => [
  { label: t('iqcheck.site.form.protocolChat'), value: IQCheckProtocol.IQ_CHECK_PROTOCOL_CHAT },
  { label: t('iqcheck.site.form.protocolResponses'), value: IQCheckProtocol.IQ_CHECK_PROTOCOL_RESPONSES },
  { label: t('iqcheck.site.form.protocolMessages'), value: IQCheckProtocol.IQ_CHECK_PROTOCOL_MESSAGES },
])

// 推理强度：空=不设置；其余值原样透传给上游（chat: reasoning_effort / responses: reasoning.effort / messages: output_config.effort）。
const effortOptions = computed(() => [
  { label: t('iqcheck.site.form.effortNone'), value: '' },
  ...['none', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max'].map((v) => ({ label: v, value: v })),
])

// 列表 = 拉取到的模型 ∪ 已勾选但不在列表里的模型（手动添加/历史配置，放最前）。
const rows = computed(() => {
  const known = new Set(available.value)
  const extra = form.value.models.map((m) => m.model).filter((m) => !known.has(m))
  const list = [...extra, ...available.value]
  const q = search.value.trim().toLowerCase()
  return q ? list.filter((m) => m.toLowerCase().includes(q)) : list
})

const selectedIndex = (model: string) => form.value.models.findIndex((m) => m.model === model)
const isSelected = (model: string) => selectedIndex(model) >= 0
const toggle = (model: string) => {
  const index = selectedIndex(model)
  if (index >= 0) form.value.models.splice(index, 1)
  else form.value.models.push({ model, effort: '' })
}
const effortOf = (model: string) => form.value.models[selectedIndex(model)]?.effort ?? ''
const setEffort = (model: string, effort: string) => {
  const entry = form.value.models[selectedIndex(model)]
  if (entry) entry.effort = effort
}

const addManual = () => {
  const model = manualModel.value.trim()
  if (!model) return
  if (!isSelected(model)) form.value.models.push({ model, effort: '' })
  manualModel.value = ''
}

// 新建时需填 API Key；编辑时可复用已保存的 Key。
const canFetch = computed(() => !!form.value.endpoint.trim() && (!!form.value.apiKey.trim() || !!form.value.id))

const fetchModels = async () => {
  fetching.value = true
  try {
    const { data } = await fetchIQCheckModels({
      endpoint: form.value.endpoint.trim(),
      apiKey: form.value.apiKey.trim(),
      protocol: form.value.protocol,
      siteId: form.value.id,
      userAgent: form.value.userAgent.trim(),
    })
    available.value = data?.models || []
    feedback.success(t('iqcheck.site.form.fetched', { count: available.value.length }))
  } finally {
    fetching.value = false
  }
}

const close = () => {
  open.value = false
  saving.value = false
  fetching.value = false
  errors.value = {}
  available.value = []
  search.value = ''
  manualModel.value = ''
  form.value = emptyForm()
}

const validate = (): boolean => {
  const e: Record<string, string> = {}
  if (!form.value.name.trim()) e.name = t('iqcheck.site.form.nameRequired')
  if (!form.value.endpoint.trim()) e.endpoint = t('iqcheck.site.form.endpointRequired')
  if (!form.value.id && !form.value.apiKey.trim()) e.apiKey = t('iqcheck.site.form.apiKeyRequired')
  errors.value = e
  return Object.keys(e).length === 0
}

const confirm = async () => {
  if (!validate()) return
  saving.value = true
  try {
    const payload = {
      name: form.value.name.trim(),
      endpoint: form.value.endpoint.trim(),
      apiKey: form.value.apiKey.trim(),
      userAgent: form.value.userAgent.trim(),
      protocol: form.value.protocol,
      models: form.value.models.map((m) => ({ model: m.model, effort: m.effort })),
      enabled: form.value.enabled,
      intervalMinutes: Math.max(1, Number(form.value.intervalMinutes) || 60),
      maxRuns: Math.max(0, Number(form.value.maxRuns) || 0),
    }
    if (form.value.id) {
      await updateIQCheckSite({ id: form.value.id, resetRunCount: form.value.resetRunCount, ...payload })
    } else {
      await createIQCheckSite(payload)
    }
    feedback.success(form.value.id ? t('iqcheck.site.editSuccess') : t('iqcheck.site.createSuccess'))
    emit('refresh')
    close()
  } finally {
    saving.value = false
  }
}

const showDialog = (row?: IQCheckSiteInfo) => {
  errors.value = {}
  available.value = []
  search.value = ''
  manualModel.value = ''
  maskedApiKey.value = row?.apiKey || ''
  form.value = {
    id: row?.id || '',
    name: row?.name || '',
    endpoint: row?.endpoint || '',
    apiKey: '',
    userAgent: row?.userAgent || '',
    protocol: row?.protocol || IQCheckProtocol.IQ_CHECK_PROTOCOL_CHAT,
    models: (row?.models || []).map((m) => ({ model: m.model, effort: m.effort })),
    enabled: row?.enabled ?? true,
    intervalMinutes: row?.intervalMinutes || 60,
    maxRuns: row?.maxRuns ?? 0,
    resetRunCount: false,
    runCount: row?.runCount ?? 0,
  }
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.iqf {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.iqf-sec {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.iqf-sec__head,
.iqf-switch {
  display: flex;
  align-items: center;
  gap: 12px;
}
.iqf-sec__ico {
  width: 40px;
  height: 40px;
  border-radius: 13px;
  svg {
    width: 20px;
    height: 20px;
  }
}
.iqf-sec__text {
  flex: 1;
  min-width: 0;
  h4 {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0;
    font-size: 0.9375rem;
    font-weight: 800;
    color: var(--el-text-color-primary);
  }
  p {
    margin: 2px 0 0;
    font-size: 0.75rem;
    line-height: 1.45;
    color: var(--el-text-color-secondary);
  }
}
.iqf-count {
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-size: 0.75rem;
  font-weight: 700;
}
.iqf-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  transition: opacity 0.15s;
  &.is-dim {
    opacity: 0.6;
  }
}
.iqf-input {
  height: 42px;
}
.iqf-hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.iqf-btn {
  flex: none;
  min-height: 40px;
  border-radius: 999px !important;
  font-weight: 700 !important;
}
.iqf-picked {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.iqf-picked__chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  min-height: 32px;
  padding: 0 4px 0 12px;
  border-radius: 999px;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-size: 0.75rem;
  font-weight: 700;
  .m-mono {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
.iqf-picked__eff {
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--el-bg-color);
  font-size: 0.6875rem;
}
.iqf-picked__x {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 26px;
  height: 26px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: inherit;
  cursor: pointer;
  svg {
    width: 14px;
    height: 14px;
  }
  &:hover {
    background: var(--el-bg-color);
  }
}
.iqf-models {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.iqf-models__tools {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}
.iqf-search {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 42px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  &:focus-within {
    border-color: var(--m-p);
  }
  svg {
    width: 18px;
    height: 18px;
    flex: none;
  }
}
.iqf-search__input {
  flex: 1;
  min-width: 0;
  height: 100%;
  border: 0;
  background: transparent;
  font: inherit;
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
  outline: none;
}
.iqf-manual {
  display: flex;
  gap: 8px;
  .app-input {
    flex: 1;
    min-width: 0;
    background: var(--el-bg-color);
  }
}
.iqf-models__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 22px 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  svg {
    width: 24px;
    height: 24px;
    opacity: 0.7;
  }
}
.iqf-list {
  max-height: 280px;
  margin: 0;
  padding: 0;
  overflow: auto;
  list-style: none;
  scrollbar-width: thin;
}
.iqf-item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 2px 4px 2px 10px;
  border-radius: 12px;
  &:hover {
    background: var(--el-bg-color);
  }
  &.is-selected {
    background: var(--el-bg-color);
    box-shadow: inset 3px 0 0 var(--m-p);
  }
}
.iqf-item__check {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 10px;
  min-width: 0;
  min-height: 40px;
  cursor: pointer;
  input {
    flex-shrink: 0;
    width: 17px;
    height: 17px;
    accent-color: var(--m-p);
  }
}
.iqf-item__model {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.8rem;
}
.iqf-item__effort {
  width: 130px;
  flex-shrink: 0;
  height: 36px;
}
.iqf-reset {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
  input {
    width: 17px;
    height: 17px;
    accent-color: var(--m-p);
  }
  em {
    margin-left: 4px;
    font-style: normal;
    color: var(--el-text-color-secondary);
  }
}
@media (max-width: 640px) {
  .iqf-sec {
    padding: 14px;
  }
  .iqf-grid,
  .iqf-models__tools {
    grid-template-columns: 1fr;
  }
  .iqf-sec__head {
    flex-wrap: wrap;
  }
  .iqf-sec__head .iqf-btn {
    width: 100%;
    justify-content: center;
  }
  .iqf-item__effort {
    width: 108px;
  }
}
</style>
