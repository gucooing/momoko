<!-- 添加 / 编辑第三方接口：名称、接口协议、接口地址、API Key、模型目录（每个模型的推理等级表）、
     主模型 / 快速模型（或默认模型）、额外环境变量、测试连接。 -->
<template>
  <AgDialog
    :model-value="modelValue"
    :title="isEdit ? t('agent.providers.form.editTitle') : t('agent.providers.form.addTitle')"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <label class="ag-field">
      <span class="ag-fl">{{ t('agent.common.name') }}</span>
      <span class="ag-fld"><input v-model="form.name" type="text" :aria-label="t('agent.common.name')" /></span>
    </label>

    <div class="ag-field">
      <span class="ag-fl">{{ t('agent.providers.form.protocol') }}</span>
      <div class="ag-pv-grid ag-grid-2" role="radiogroup" :aria-label="t('agent.providers.form.protocol')">
        <button
          v-for="p in PROTOCOLS"
          :key="p.key"
          type="button"
          class="ag-opt ag-pv-opt"
          role="radio"
          :class="{ 'is-on': form.vendor === p.key }"
          :aria-checked="form.vendor === p.key"
          :disabled="isEdit && form.vendor !== p.key"
          @click="form.vendor = p.key"
        >
          <span class="ag-rd" />
          <span class="ag-pv-opt__main">
            <span class="ag-pv-opt__t">{{ t(`agent.protocol.${p.key}`) }}</span>
            <i18n-t :keypath="`agent.providers.form.protocolDesc.${p.key}`" tag="span" scope="global" class="ag-pv-opt__d">
              <template #agent><AgAgentTag :kind="p.agentKind" :name="t(`agent.family.${p.agentKind}`)" /></template>
            </i18n-t>
          </span>
        </button>
      </div>
    </div>

    <div class="ag-field">
      <span class="ag-fl">{{ t('agent.providers.form.baseUrl') }}</span>
      <span class="ag-fld">
        <input v-model="form.baseUrl" class="ag-mono" type="url" placeholder="https://" :aria-label="t('agent.providers.form.baseUrl')" />
      </span>
      <span class="ag-fh">{{ t(`agent.providers.form.urlHint.${form.vendor}`) }}</span>
    </div>

    <div class="ag-field">
      <span class="ag-fl">{{ t('agent.providers.form.apiKey') }}</span>
      <span class="ag-fld ag-pv-key">
        <UIcon name="i-lucide-key-round" />
        <input
          v-model="form.apiKey"
          class="ag-mono"
          :type="showKey ? 'text' : 'password'"
          :placeholder="keyPlaceholder"
          autocomplete="new-password"
          :aria-label="t('agent.providers.form.apiKey')"
        />
        <button
          type="button"
          class="ag-ib ag-ib--sm"
          :aria-label="t('agent.providers.form.toggleKey')"
          :title="t('agent.providers.form.toggleKey')"
          @click="showKey = !showKey"
        >
          <UIcon :name="showKey ? 'i-lucide-eye-off' : 'i-lucide-eye'" />
        </button>
      </span>
      <span class="ag-fh">{{ t('agent.providers.form.keyHint') }}</span>
    </div>

    <div class="ag-field">
      <div class="ag-pv-row">
        <span class="ag-fl">{{ t('agent.providers.form.catalog') }}</span>
        <button type="button" class="ag-btn ag-btn--sm" :disabled="fetching" @click="fetchModels">
          <UIcon name="i-lucide-refresh-cw" :class="{ 'ag-spin': fetching }" />{{ t('agent.providers.form.fetchModels') }}
        </button>
      </div>
      <div class="ag-pv-models">
        <div v-for="m in form.models" :key="m.id" class="ag-pv-model">
          <div class="ag-pv-model__head">
            <!-- 点这一行展开或收起它的配置 -->
            <button
              type="button"
              class="ag-pv-model__toggle"
              :aria-expanded="openModel === m.id"
              @click="openModel = openModel === m.id ? '' : m.id"
            >
              <UIcon :name="openModel === m.id ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="ag-pv-model__chev" />
              <span class="ag-pv-model__n ag-mono" :title="m.id">{{ m.id }}</span>
              <span class="ag-pv-model__sum">{{ levelSummary(m) }}</span>
            </button>
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="t('agent.providers.form.removeModel', { name: m.id })"
              @click="removeModel(m.id)"
            >
              <UIcon name="i-lucide-x" />
            </button>
          </div>
          <div v-if="openModel === m.id" class="ag-pv-model__body">
            <span class="ag-pv-model__lt">{{ t('agent.providers.form.limits') }}</span>
            <div class="ag-pv-levels ag-pv-levels--head" aria-hidden="true">
              <span>{{ t('agent.providers.form.contextWindow') }}</span>
              <span>{{ form.vendor === 'anthropic' ? t('agent.providers.form.maxOutput') : '' }}</span>
              <span />
            </div>
            <div class="ag-pv-levels" role="group" :aria-label="t('agent.providers.form.limitsOf', { name: m.id })">
              <span class="ag-fld ag-fld--plain ag-pv-lv">
                <input
                  v-model.number="m.contextWindow"
                  class="ag-mono"
                  type="number"
                  min="1"
                  step="1000"
                  :aria-label="t('agent.providers.form.contextWindow')"
                />
                <span class="ag-pv-unit ag-num">{{ fmtLimit(m.contextWindow) }}</span>
              </span>
              <span v-if="form.vendor === 'anthropic'" class="ag-fld ag-fld--plain ag-pv-lv">
                <input
                  v-model.number="m.maxOutput"
                  class="ag-mono"
                  type="number"
                  min="1"
                  step="1000"
                  :aria-label="t('agent.providers.form.maxOutput')"
                />
                <span class="ag-pv-unit ag-num">{{ fmtLimit(m.maxOutput) }}</span>
              </span>
            </div>
            <span class="ag-fh">{{ t(form.vendor === 'anthropic' ? 'agent.providers.form.limitsHint' : 'agent.providers.form.limitsHintCodex') }}</span>
            <div class="ag-pv-row">
              <span class="ag-pv-model__lt">{{ t('agent.providers.form.levels') }}</span>
              <button type="button" class="ag-btn ag-btn--sm ag-btn--plain" @click="addLevel(m)">
                <UIcon name="i-lucide-plus" />{{ t('agent.providers.form.addLevel') }}
              </button>
            </div>
            <div class="ag-pv-levels ag-pv-levels--head" aria-hidden="true">
              <span>{{ t('agent.providers.form.level') }}</span>
              <span>{{ t('agent.providers.form.levelValue') }}</span>
              <span />
            </div>
            <div v-for="(r, j) in m.levels" :key="r.uid" class="ag-pv-levels">
              <span class="ag-fld ag-fld--plain ag-pv-lv">
                <input
                  v-model="r.name"
                  class="ag-mono"
                  type="text"
                  :placeholder="t('agent.providers.form.levelName')"
                  :aria-label="t('agent.providers.form.level')"
                />
              </span>
              <span class="ag-fld ag-fld--plain ag-pv-lv">
                <input
                  v-model="r.value"
                  class="ag-mono"
                  type="text"
                  :placeholder="valuePlaceholder(r)"
                  :aria-label="t('agent.providers.form.levelValueOf', { name: r.name })"
                />
              </span>
              <button
                type="button"
                class="ag-ib ag-ib--sm"
                :aria-label="t('agent.providers.form.removeLevel', { name: r.name })"
                @click="m.levels.splice(j, 1)"
              >
                <UIcon name="i-lucide-x" />
              </button>
            </div>
            <span class="ag-fh">{{ t('agent.providers.form.levelsHint') }}</span>
          </div>
        </div>
        <span class="ag-fld ag-pv-add">
          <input
            v-model="modelDraft"
            class="ag-mono"
            type="text"
            :placeholder="t('agent.providers.form.addModelPlaceholder')"
            :aria-label="t('agent.providers.form.addModel')"
            @keydown.enter.prevent="addModel"
          />
        </span>
      </div>
    </div>

    <div class="ag-pv-grid ag-grid-2">
      <div v-for="k in picks" :key="k.key" class="ag-field">
        <span class="ag-fl">{{ k.title }}</span>
        <AgSelect
          :model-value="k.value"
          :options="modelOptions"
          :placeholder="t('agent.providers.noModels')"
          :empty-text="t('agent.providers.noModels')"
          :label="k.title"
          mono
          @update:model-value="k.set"
        />
        <span class="ag-fh">{{ k.hint }}</span>
      </div>
    </div>

    <div class="ag-field">
      <div class="ag-pv-row">
        <span class="ag-fl">{{ t('agent.providers.form.env') }}</span>
        <button type="button" class="ag-btn ag-btn--sm" @click="addEnv">
          <UIcon name="i-lucide-plus" />{{ t('agent.common.addRow') }}
        </button>
      </div>
      <div v-for="(r, j) in form.env" :key="r.uid" class="ag-kv ag-pv-env">
        <span class="ag-fld">
          <input
            v-model="r.key"
            class="ag-mono"
            type="text"
            :placeholder="t('agent.providers.form.envKey')"
            :aria-label="t('agent.providers.form.envKey')"
          />
        </span>
        <span class="ag-fld">
          <input
            v-model="r.value"
            class="ag-mono"
            type="text"
            :placeholder="t('agent.providers.form.envValue')"
            :aria-label="t('agent.providers.form.envValue')"
          />
        </span>
        <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.common.deleteRow')" @click="form.env.splice(j, 1)">
          <UIcon name="i-lucide-trash-2" />
        </button>
      </div>
    </div>

    <div class="ag-pv-test">
      <button type="button" class="ag-btn ag-btn--sm ag-btn--plain" :disabled="test.state === 'busy'" @click="runTest">
        <UIcon
          :name="test.state === 'busy' ? 'i-lucide-loader-circle' : 'i-lucide-zap'"
          :class="{ 'ag-spin': test.state === 'busy' }"
        />{{ t('agent.common.testConnection') }}
      </button>
      <span class="ag-pill" :class="testPill.cls">{{ testPill.text }}</span>
      <span class="ag-pv-test__note">{{ testNote }}</span>
    </div>

    <template #footer>
      <button type="button" class="ag-btn" @click="emit('update:modelValue', false)">{{ t('agent.common.cancel') }}</button>
      <button type="button" class="ag-btn ag-btn--p" :disabled="saving" @click="save">
        <UIcon v-if="saving" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.common.save') }}
      </button>
    </template>
  </AgDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { createAgentProvider, fetchAgentProviderModels, testAgentProvider, updateAgentProvider } from '@/api/agent'
import type { AgentProvider } from '@/types/v1/agent'
import { getRequestErrorMessage } from '@/utils/request'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import AgSelect from '@/views/agent/components/AgSelect.vue'

type Vendor = 'anthropic' | 'openai'
interface LevelRow {
  uid: number
  name: string
  value: string
}
interface ModelRow {
  id: string
  levels: LevelRow[]
  /** 最长上下文和最长输出（token），新模型默认 1M / 128K */
  contextWindow: number
  maxOutput: number
}
interface EnvRow {
  uid: number
  key: string
  value: string
}

const props = defineProps<{ modelValue: boolean; provider?: AgentProvider | null }>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
}>()
const { t } = useI18n()

const PROTOCOLS: { key: Vendor; agentKind: string }[] = [
  { key: 'anthropic', agentKind: 'claude' },
  { key: 'openai', agentKind: 'codex' },
]
// 推理等级从低到高；「添加一级」按这个顺序带出下一档
const LADDER = ['off', 'low', 'medium', 'high', 'xhigh', 'max']
const DEFAULT_LEVELS = ['off', 'low', 'medium', 'high']
const DEFAULT_CONTEXT = 1_000_000
const DEFAULT_OUTPUT = 128_000
// 新加进目录的模型：默认推理等级和默认的最长上下文、最长输出
const newModel = (id: string): ModelRow => ({ id, levels: defaultLevels(), contextWindow: DEFAULT_CONTEXT, maxOutput: DEFAULT_OUTPUT })
// 上下文、输出的读数：1000000 → 1M，128000 → 128K（多余的 0 不显示）
const fmtLimit = (value: number) => {
  const n = Number(value) || 0
  if (n >= 1_000_000) return `${+(n / 1_000_000).toFixed(2)}M`
  if (n >= 1000) return `${+(n / 1000).toFixed(1)}K`
  return String(n)
}

let uidSeed = 0
const levelRow = (name: string, value: string): LevelRow => ({ uid: ++uidSeed, name, value })
// off 的参数值留空 = 不带这个参数，其它默认和等级名相同
const presetLevel = (name: string) => levelRow(name, name === 'off' ? '' : name)
const defaultLevels = () => DEFAULT_LEVELS.map(presetLevel)

const form = reactive({
  name: '',
  vendor: 'anthropic' as Vendor,
  baseUrl: '',
  apiKey: '',
  models: [] as ModelRow[],
  mainModel: '',
  fastModel: '',
  env: [] as EnvRow[],
})
const showKey = ref(false)
const modelDraft = ref('')
const openModel = ref('')
const fetching = ref(false)
const saving = ref(false)
const test = reactive({ state: 'idle' as 'idle' | 'busy' | 'ok' | 'fail', note: '' })
// 每次重新测试、改了连接信息或重开弹窗都加一，过期的测试结果不再写回
let testSeq = 0

const isEdit = computed(() => !!props.provider)
const keyPlaceholder = computed(() => (props.provider?.apiKeyTail ? `sk-••••${props.provider.apiKeyTail}` : ''))

const init = () => {
  const p = props.provider
  form.name = p?.name || ''
  form.vendor = p?.vendor === 'openai' ? 'openai' : 'anthropic'
  form.baseUrl = p?.baseUrl || ''
  form.apiKey = ''
  form.models = (p?.models || []).map((m) => ({
    id: m.id,
    levels: (m.levels || []).map((l) => levelRow(l.name, l.value)),
    contextWindow: Number(m.contextWindow) || DEFAULT_CONTEXT,
    maxOutput: Number(m.maxOutput) || DEFAULT_OUTPUT,
  }))
  form.mainModel = p?.mainModel || ''
  form.fastModel = p?.fastModel || ''
  form.env = (p?.env || []).map((e) => ({ uid: ++uidSeed, key: e.key, value: e.value }))
  showKey.value = false
  modelDraft.value = ''
  openModel.value = form.models[0]?.id || ''
  fetching.value = false
  saving.value = false
  testSeq += 1
  test.state = 'idle'
  test.note = ''
}
watch(
  () => props.modelValue,
  (open) => {
    if (open) init()
  },
  { immediate: true },
)

// —— 模型目录 ——
// 收起时一行看完：推理等级，加上最长上下文（和 Anthropic 兼容接口的最长输出）
const levelSummary = (m: ModelRow) => {
  const names = m.levels.map((r) => r.name.trim()).filter(Boolean)
  const levels = names.length ? t('agent.providers.form.levelsSummary', { list: names.join(' · ') }) : t('agent.providers.form.noLevels')
  const limits =
    form.vendor === 'anthropic'
      ? t('agent.providers.form.limitsSummary', { ctx: fmtLimit(m.contextWindow), out: fmtLimit(m.maxOutput) })
      : t('agent.providers.form.limitsSummaryCtx', { ctx: fmtLimit(m.contextWindow) })
  return `${levels} · ${limits}`
}
const valuePlaceholder = (r: LevelRow) =>
  r.name === 'off' ? t('agent.providers.form.offPlaceholder') : r.name || t('agent.providers.form.valuePlaceholder')

// 带出比当前最高一档更高的那一档；中间有缺的按顺序补进对应位置；六档全有之后才是空白行
const addLevel = (m: ModelRow) => {
  const rank = (r: LevelRow) => LADDER.indexOf(r.name.trim().toLowerCase())
  const have = m.levels.map((r) => r.name.trim().toLowerCase())
  const top = Math.max(-1, ...m.levels.map(rank))
  const next = LADDER.find((x, q) => q > top && !have.includes(x)) || LADDER.find((x) => !have.includes(x))
  if (!next) {
    m.levels.push(levelRow('', ''))
    return
  }
  const at = m.levels.findIndex((r) => rank(r) > LADDER.indexOf(next))
  if (at < 0) m.levels.push(presetLevel(next))
  else m.levels.splice(at, 0, presetLevel(next))
}

const addModel = () => {
  const id = modelDraft.value.trim()
  if (!id) return
  if (!form.models.some((m) => m.id === id)) form.models.push(newModel(id))
  modelDraft.value = ''
  openModel.value = id
}
const removeModel = (id: string) => {
  form.models = form.models.filter((m) => m.id !== id)
}

const fetchModels = async () => {
  if (fetching.value) return
  if (!form.baseUrl.trim()) {
    feedback.warning(t('agent.providers.form.needUrl'))
    return
  }
  fetching.value = true
  try {
    const { data } = await fetchAgentProviderModels({
      ...(props.provider ? { id: props.provider.id } : {}),
      vendor: form.vendor,
      baseUrl: form.baseUrl.trim(),
      apiKey: form.apiKey.trim(),
    })
    const got = data.models || []
    const add = [...new Set(got)].filter((id) => id && !form.models.some((m) => m.id === id))
    add.forEach((id) => form.models.push(newModel(id)))
    feedback.success(
      add.length
        ? t('agent.providers.form.fetched', { n: got.length, m: add.length })
        : t('agent.providers.form.fetchedNone', { n: got.length }),
    )
  } catch {
    /* shown */
  } finally {
    fetching.value = false
  }
}

// —— 主模型 / 快速模型（OpenAI 兼容只有一个默认模型）——
const modelIds = computed(() => form.models.map((m) => m.id))
const modelOptions = computed(() => modelIds.value.map((id) => ({ value: id, label: id })))
// 选过的模型被移出目录后，退回目录里的第一个
const inCatalog = (id: string) => (modelIds.value.includes(id) ? id : (modelIds.value[0] ?? ''))
const mainModel = computed(() => inCatalog(form.mainModel))
const fastModel = computed(() => inCatalog(form.fastModel))
const setMain = (id: string) => {
  form.mainModel = id
}
const setFast = (id: string) => {
  form.fastModel = id
}
const picks = computed(() =>
  form.vendor === 'anthropic'
    ? [
        {
          key: 'main',
          title: t('agent.providers.form.mainModel'),
          hint: t('agent.providers.form.mainModelHint'),
          value: mainModel.value,
          set: setMain,
        },
        {
          key: 'fast',
          title: t('agent.providers.form.fastModel'),
          hint: t('agent.providers.form.fastModelHint'),
          value: fastModel.value,
          set: setFast,
        },
      ]
    : [
        {
          key: 'default',
          title: t('agent.providers.form.defaultModel'),
          hint: t('agent.providers.form.defaultModelHint'),
          value: mainModel.value,
          set: setMain,
        },
      ],
)

const addEnv = () => {
  form.env.push({ uid: ++uidSeed, key: '', value: '' })
}

// —— 提交的内容 ——
const draft = (): Partial<AgentProvider> => ({
  vendor: form.vendor,
  name: form.name.trim(),
  baseUrl: form.baseUrl.trim(),
  apiKey: form.apiKey.trim(),
  models: form.models.map((m) => ({
    id: m.id,
    levels: m.levels.filter((r) => r.name.trim()).map((r) => ({ name: r.name.trim(), value: r.value.trim() })),
    contextWindow: Number(m.contextWindow) || DEFAULT_CONTEXT,
    maxOutput: Number(m.maxOutput) || DEFAULT_OUTPUT,
  })),
  mainModel: mainModel.value,
  fastModel: form.vendor === 'anthropic' ? fastModel.value : '',
  env: form.env.filter((r) => r.key.trim()).map((r) => ({ key: r.key.trim(), value: r.value })),
})
// 接口地址必填；新建时还要有 API Key（编辑时留空表示沿用已保存的）
const connectionReady = () => {
  if (!form.baseUrl.trim()) {
    feedback.warning(t('agent.providers.form.needUrl'))
    return false
  }
  if (!isEdit.value && !form.apiKey.trim()) {
    feedback.warning(t('agent.providers.form.needKey'))
    return false
  }
  return true
}

// —— 测试连接 ——
const testPill = computed(() => {
  if (test.state === 'ok') return { cls: 'ag-pill--ok', text: t('agent.providers.form.testOk') }
  if (test.state === 'fail') return { cls: 'ag-pill--bad', text: t('agent.providers.form.testFail') }
  return { cls: '', text: t('agent.common.notTested') }
})
const testNote = computed(() => (test.state === 'ok' || test.state === 'fail' ? test.note : t('agent.providers.form.testIdle')))
const runTest = async () => {
  if (test.state === 'busy' || !connectionReady()) return
  addModel()
  const my = ++testSeq
  test.state = 'busy'
  try {
    const { data } = await testAgentProvider({ id: props.provider?.id, draft: draft() })
    if (my !== testSeq) return
    test.state = data.ok ? 'ok' : 'fail'
    test.note = data.ok
      ? t('agent.providers.form.testNote', { model: data.model, ms: Number(data.latencyMs) || 0 })
      : data.message || t('agent.providers.status.failed')
  } catch (error) {
    if (my !== testSeq) return
    test.state = 'fail'
    test.note = getRequestErrorMessage(error)
  }
}
// 协议、地址或密钥改了，之前的测试结果就不算数了
watch(
  () => [form.vendor, form.baseUrl, form.apiKey],
  () => {
    testSeq += 1
    test.state = 'idle'
    test.note = ''
  },
)

// —— 保存 ——
const save = async () => {
  if (saving.value) return
  if (!form.name.trim()) {
    feedback.warning(t('agent.providers.form.needName'))
    return
  }
  if (!connectionReady()) return
  addModel()
  const body = draft()
  const name = body.name || ''
  saving.value = true
  try {
    await (props.provider ? updateAgentProvider(props.provider.id, body) : createAgentProvider(body))
    feedback.success(t(isEdit.value ? 'agent.providers.form.saved' : 'agent.providers.form.added', { name }))
    emit('saved')
    emit('update:modelValue', false)
  } catch {
    /* shown */
  } finally {
    saving.value = false
  }
}
</script>

<style lang="scss">
// 标题在左、小按钮在右的一行
.ag-pv-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.ag-pv-row > :first-child {
  flex: 1;
  min-width: 0;
}
// 编辑时接口协议不能改
.ag-opt.ag-pv-opt:disabled {
  opacity: 0.55;
  cursor: not-allowed;
  border-color: var(--m-line);
}
.ag-fld.ag-pv-key {
  padding-right: 4px;
}
// 模型目录
.ag-pv-models {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ag-pv-model {
  border-radius: 10px;
  background: var(--m-s2);
  overflow: hidden;
}
.ag-pv-model__head {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 2px 2px 2px 6px;
}
.ag-pv-model__toggle {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 0 4px;
  border: 0;
  border-radius: 8px;
  background: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.ag-pv-model__toggle:hover {
  background: var(--m-s);
}
.ag-pv-model__chev {
  flex: none;
  width: 14px;
  height: 14px;
  color: var(--m-ink2);
}
.ag-pv-model__n {
  flex: 0 0 170px;
  min-width: 0;
  font-size: 13px;
  color: var(--m-ink);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ag-pv-model__sum {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--m-ink2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ag-pv-model__body {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 2px 10px 10px;
}
.ag-pv-model__lt {
  font-size: 13px;
  font-weight: 600;
}
// 推理等级表：等级 + API 参数值 + 删除
.ag-pv-levels {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr) 30px;
  gap: 6px;
  align-items: center;
}
.ag-pv-levels--head {
  padding: 0 2px;
  font-size: 12px;
  font-weight: 500;
  color: var(--m-ink2);
}
.ag-fld.ag-pv-lv {
  min-height: 30px;
  border-radius: 8px;
}
.ag-fld.ag-pv-lv input {
  height: 26px;
}
// 上下文、输出框里跟着显示换算后的读数（1M、128K），填错位数一眼看得出来
.ag-pv-unit {
  flex: none;
  font-size: 11.5px;
  color: var(--m-ink2);
}
.ag-fld.ag-pv-add {
  border-radius: 8px;
}
// 测试连接
.ag-pv-test {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 10px;
  background: var(--m-s2);
}
.ag-pv-test__note {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  color: var(--m-ink2);
  overflow-wrap: anywhere;
}
@media (width < 992px) {
  .ag-pv-model__toggle {
    flex-wrap: wrap;
    padding: 4px;
  }
  .ag-pv-model__n {
    flex: 1 1 0;
  }
  .ag-pv-model__sum {
    flex: 1 1 100%;
    padding-left: 20px;
  }
  .ag-pv-levels {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr) 40px;
  }
  .ag-kv.ag-pv-env {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1.6fr) 40px;
  }
  .ag-fld.ag-pv-lv {
    min-height: 40px;
  }
  .ag-fld.ag-pv-lv input {
    height: 36px;
  }
  .ag-pv-test {
    flex-wrap: wrap;
  }
  .ag-pv-test__note {
    flex: 1 1 100%;
  }
}
</style>
