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
    <div class="iq-site-form">
      <div class="iq-site-form__grid">
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('iqcheck.site.form.name') }}</label>
          <input
            v-model="form.name"
            class="app-input"
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
          class="app-input"
          :class="{ 'is-error': errors.endpoint }"
          :placeholder="t('iqcheck.site.form.endpointPlaceholder')"
        />
        <span v-if="errors.endpoint" class="app-field__error">{{ errors.endpoint }}</span>
        <span v-else class="iq-site-form__hint">{{ t('iqcheck.site.form.endpointHint') }}</span>
      </div>

      <div class="iq-site-form__grid">
        <div class="app-field">
          <label class="app-label" :class="{ 'app-label--required': !form.id }">
            {{ t('iqcheck.site.form.apiKey') }}
          </label>
          <input
            v-model="form.apiKey"
            type="password"
            autocomplete="off"
            class="app-input"
            :class="{ 'is-error': errors.apiKey }"
            :placeholder="form.id ? maskedApiKey : t('iqcheck.site.form.apiKeyPlaceholder')"
          />
          <span v-if="errors.apiKey" class="app-field__error">{{ errors.apiKey }}</span>
          <span v-else-if="form.id" class="iq-site-form__hint">{{ t('iqcheck.site.form.apiKeyKeepHint') }}</span>
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('iqcheck.site.form.userAgent') }}</label>
          <input v-model="form.userAgent" class="app-input" placeholder="pi" />
          <span class="iq-site-form__hint">{{ t('iqcheck.site.form.userAgentHint') }}</span>
        </div>
      </div>

      <!-- 模型选择 -->
      <div class="app-field">
        <div class="iq-site-form__models-head">
          <label class="app-label">{{ t('iqcheck.site.form.models') }}</label>
          <span class="iq-site-form__count">{{ t('iqcheck.site.form.selectedModels', { count: form.models.length }) }}</span>
          <UButton
            color="neutral"
            variant="soft"
            size="sm"
            icon="i-lucide-download"
            :loading="fetching"
            :disabled="!canFetch"
            @click="fetchModels"
          >
            {{ fetching ? t('iqcheck.site.form.fetching') : t('iqcheck.site.form.fetchModels') }}
          </UButton>
        </div>
        <span class="iq-site-form__hint">{{ t('iqcheck.site.form.fetchHint') }}</span>

        <div class="iq-site-form__models">
          <div class="iq-site-form__models-tools">
            <input
              v-model="search"
              class="app-input"
              :placeholder="t('iqcheck.site.form.searchModels')"
            />
            <div class="iq-site-form__manual">
              <input
                v-model="manualModel"
                class="app-input"
                :placeholder="t('iqcheck.site.form.manualModel')"
                @keyup.enter="addManual"
              />
              <UButton color="neutral" variant="soft" size="sm" :disabled="!manualModel.trim()" @click="addManual">
                {{ t('iqcheck.site.form.add') }}
              </UButton>
            </div>
          </div>

          <div v-if="!rows.length" class="iq-site-form__models-empty">{{ t('iqcheck.site.form.noModels') }}</div>
          <ul v-else class="iq-site-form__list">
            <li v-for="model in rows" :key="model" class="iq-site-form__item" :class="{ 'is-selected': isSelected(model) }">
              <label class="iq-site-form__check">
                <input type="checkbox" :checked="isSelected(model)" @change="toggle(model)" />
                <span class="iq-site-form__model" :title="model">{{ model }}</span>
              </label>
              <select
                class="app-input iq-site-form__effort"
                :disabled="!isSelected(model)"
                :value="effortOf(model)"
                :title="t('iqcheck.site.form.effort')"
                @change="setEffort(model, ($event.target as HTMLSelectElement).value)"
              >
                <option v-for="opt in effortOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
              </select>
            </li>
          </ul>
        </div>
      </div>

      <div class="iq-site-form__grid iq-site-form__grid--3">
        <div class="app-field">
          <label class="app-label">{{ t('iqcheck.site.form.interval') }}</label>
          <input v-model.number="form.intervalMinutes" type="number" min="1" class="app-input" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('iqcheck.site.form.maxRuns') }}</label>
          <input v-model.number="form.maxRuns" type="number" min="0" class="app-input" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('iqcheck.site.form.enabled') }}</label>
          <div class="iq-site-form__switch">
            <AppSwitch v-model="form.enabled" />
            <span>{{ form.enabled ? t('system.common.enabled') : t('system.common.inactive') }}</span>
          </div>
        </div>
      </div>

      <label v-if="form.id" class="iq-site-form__reset">
        <input v-model="form.resetRunCount" type="checkbox" />
        <span>{{ t('iqcheck.site.form.resetRunCount') }}（{{ t('iqcheck.site.form.runCount', { count: form.runCount }) }}）</span>
      </label>
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
.iq-site-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.iq-site-form__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  &--3 {
    grid-template-columns: 1fr 1fr 1fr;
  }
  @media (max-width: 640px) {
    grid-template-columns: 1fr;
    &--3 {
      grid-template-columns: 1fr;
    }
  }
}
.iq-site-form__hint {
  font-size: 0.72rem;
  color: var(--el-text-color-placeholder);
}
.iq-site-form__switch {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
}
.iq-site-form__models-head {
  display: flex;
  align-items: center;
  gap: 10px;
  .app-label {
    margin: 0;
  }
}
.iq-site-form__count {
  flex: 1;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.iq-site-form__models {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 6px;
  padding: 10px;
  border: 1px solid var(--el-border-color-lighter, #e5e7eb);
  border-radius: 10px;
}
.iq-site-form__models-tools {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  @media (max-width: 640px) {
    grid-template-columns: 1fr;
  }
}
.iq-site-form__manual {
  display: flex;
  gap: 8px;
  .app-input {
    flex: 1;
  }
}
.iq-site-form__models-empty {
  padding: 18px 0;
  text-align: center;
  font-size: 0.8rem;
  color: var(--el-text-color-placeholder);
}
.iq-site-form__list {
  max-height: 280px;
  margin: 0;
  padding: 0;
  overflow: auto;
  list-style: none;
}
.iq-site-form__item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 4px;
  border-radius: 6px;
  &.is-selected {
    background: var(--el-fill-color-light);
  }
}
.iq-site-form__check {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 8px;
  min-width: 0;
  cursor: pointer;
  input {
    flex-shrink: 0;
  }
}
.iq-site-form__model {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.78rem;
}
.iq-site-form__effort {
  width: 120px;
  flex-shrink: 0;
  height: 30px;
  padding: 0 8px;
}
.iq-site-form__reset {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
</style>
