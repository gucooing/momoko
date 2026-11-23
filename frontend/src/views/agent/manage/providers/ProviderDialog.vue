<!-- 添加 / 编辑供应商，分三步，每步一页，可以回上一步：
     ① 类型：通用的兼容接口，或对某一家的专门适配（类型由后端列出；编辑时类型不能改，没有这一步）；
     ② 配置：名称、接口地址（地址固定的类型没有）、API Key、模型目录、主模型 / 快速模型、额外环境变量；
     ③ 测试：输入一条消息（默认 hello）用主模型发出去，类型给的每种协议各发一次，看回复；在这一步保存。 -->
<template>
  <AgDialog
    :model-value="modelValue"
    :title="isEdit ? t('agent.providers.form.editTitle') : t('agent.providers.form.addTitle')"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <ol class="ag-pv-steps" :aria-label="t('agent.providers.form.steps')">
      <li
        v-for="(s, i) in steps"
        :key="s"
        class="ag-pv-step"
        :class="{ 'is-on': s === step, 'is-done': i < stepIndex }"
        :aria-current="s === step ? 'step' : undefined"
      >
        <span class="ag-pv-step__n">
          <UIcon v-if="i < stepIndex" name="i-lucide-check" />
          <template v-else>{{ i + 1 }}</template>
        </span>
        <span>{{ t(`agent.providers.form.step.${s}`) }}</span>
      </li>
    </ol>

    <!-- ① 类型 -->
    <template v-if="step === 'vendor'">
      <div v-for="g in vendorGroups" :key="g.key" class="ag-field">
        <span class="ag-fl">{{ t(`agent.providers.form.group.${g.key}`) }}</span>
        <div class="ag-pv-grid" role="radiogroup" :aria-label="t(`agent.providers.form.group.${g.key}`)">
          <button
            v-for="v in g.items"
            :key="v.id"
            type="button"
            class="ag-opt ag-pv-opt"
            role="radio"
            :class="{ 'is-on': form.vendor === v.id }"
            :aria-checked="form.vendor === v.id"
            @click="pickVendor(v.id)"
          >
            <span class="ag-rd" />
            <span class="ag-pv-opt__main">
              <span class="ag-pv-opt__t">{{ vendorName(v.id) }}</span>
              <span v-if="vendorDesc(v.id)" class="ag-pv-opt__d">{{ vendorDesc(v.id) }}</span>
              <span class="ag-pv-opt__d">
                {{ t('agent.providers.form.forAgents') }}
                <AgAgentTag
                  v-for="p in v.protocols"
                  :key="p"
                  :kind="protocolAgent(p)"
                  :name="t(`agent.family.${protocolAgent(p)}`)"
                />
              </span>
            </span>
          </button>
        </div>
      </div>
      <div v-if="!vendors.length" class="ag-empty">{{ t('agent.common.loading') }}</div>
    </template>

    <!-- ② 配置 -->
    <template v-else-if="step === 'config'">
      <label class="ag-field">
        <span class="ag-fl">{{ t('agent.common.name') }}</span>
        <span class="ag-fld"><input v-model="form.name" type="text" :aria-label="t('agent.common.name')" /></span>
      </label>
      <AgShareField v-if="shared" v-model="share" />

      <div v-if="!fixedUrl" class="ag-field">
        <span class="ag-fl">{{ t('agent.providers.form.baseUrl') }}</span>
        <span class="ag-fld">
          <input v-model="form.baseUrl" class="ag-mono" type="url" placeholder="https://" :aria-label="t('agent.providers.form.baseUrl')" />
        </span>
        <span class="ag-fh">{{ urlHint }}</span>
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
                <span>{{ forClaude ? t('agent.providers.form.maxOutput') : '' }}</span>
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
                <span v-if="forClaude" class="ag-fld ag-fld--plain ag-pv-lv">
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
              <span class="ag-fh">{{ t(forClaude ? 'agent.providers.form.limitsHint' : 'agent.providers.form.limitsHintCodex') }}</span>
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

      <div class="ag-pv-grid">
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
    </template>

    <!-- ③ 测试 -->
    <template v-else>
      <div class="ag-field">
        <div class="ag-pv-row">
          <span class="ag-fl">{{ t('agent.providers.form.testMessage') }}</span>
          <span class="ag-pv-test__model">
            {{ t('agent.providers.form.testWith') }}<span class="ag-mono">{{ mainModel }}</span>
          </span>
        </div>
        <textarea
          v-model="test.message"
          class="ag-ta ag-pv-test__ta"
          rows="3"
          spellcheck="false"
          :aria-label="t('agent.providers.form.testMessage')"
          @keydown.enter.ctrl.prevent="runTest"
          @keydown.enter.meta.prevent="runTest"
        />
        <div class="ag-pv-row">
          <span class="ag-fh">{{ t('agent.providers.form.testHint') }}</span>
          <button type="button" class="ag-btn ag-btn--sm ag-btn--o" :disabled="test.busy || !test.message.trim()" @click="runTest">
            <UIcon
              :name="test.busy ? 'i-lucide-loader-circle' : 'i-lucide-send'"
              :class="{ 'ag-spin': test.busy }"
            />{{ t('agent.providers.form.testSend') }}
          </button>
        </div>
      </div>

      <div v-for="r in testRows" :key="r.protocol" class="ag-pv-res">
        <div class="ag-pv-res__head">
          <AgAgentTag :kind="protocolAgent(r.protocol)" :name="t(`agent.family.${protocolAgent(r.protocol)}`)" />
          <span class="ag-pv-res__p">{{ t(`agent.protocol.${r.protocol}`) }}</span>
          <span class="ag-pill" :class="r.pill.cls">{{ r.pill.text }}</span>
        </div>
        <template v-if="r.result">
          <div v-if="r.result.ok" class="ag-pv-res__reply">{{ r.result.reply || t('agent.providers.form.noReply') }}</div>
          <div v-else class="ag-pv-res__err">{{ r.result.error || t('agent.providers.status.failed') }}</div>
          <span v-if="r.result.ok" class="ag-fh">
            {{ t('agent.providers.form.testUsage', { input: fmtTokens(r.result.inputTokens), output: fmtTokens(r.result.outputTokens) }) }}
          </span>
        </template>
      </div>
      <span v-if="test.error" class="ag-pv-res__err">{{ test.error }}</span>
    </template>

    <template #footer>
      <button v-if="stepIndex > 0" type="button" class="ag-btn" @click="back">
        <UIcon name="i-lucide-chevron-left" />{{ t('agent.providers.form.back') }}
      </button>
      <span class="ag-pv-foot-gap" />
      <button type="button" class="ag-btn" @click="emit('update:modelValue', false)">{{ t('agent.common.cancel') }}</button>
      <button v-if="step !== 'test'" type="button" class="ag-btn ag-btn--p" :disabled="step === 'vendor' && !vendor" @click="next">
        {{ t('agent.providers.form.next') }}<UIcon name="i-lucide-chevron-right" />
      </button>
      <button v-else type="button" class="ag-btn ag-btn--p" :disabled="saving" @click="save">
        <UIcon v-if="saving" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.common.save') }}
      </button>
    </template>
  </AgDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  type AgentShare,
  createAgentProvider,
  fetchAgentProviderModels,
  testAgentProvider,
  updateAgentProvider,
} from '@/api/agent'
import type { AgentModel, AgentProvider, AgentProviderTestResult, AgentVendor } from '@/types/v1/agent'
import { getRequestErrorMessage } from '@/utils/request'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import AgSelect from '@/views/agent/components/AgSelect.vue'
import AgShareField from '@/views/agent/components/AgShareField.vue'
import { fmtTokens, protocolAgent } from '@/views/agent/utils/format'
import { vendorDesc, vendorName } from '@/views/agent/utils/labels'

type Step = 'vendor' | 'config' | 'test'
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

const props = defineProps<{
  modelValue: boolean
  provider?: AgentProvider | null
  vendors: AgentVendor[]
  /** 共享的（Agent 管理页里加、改）：配置这一步多一项「共享给」 */
  shared?: boolean
}>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
}>()
const { t } = useI18n()

// 推理等级从低到高；「添加一级」按这个顺序带出下一档
const LADDER = ['off', 'low', 'medium', 'high', 'xhigh', 'max']
const DEFAULT_LEVELS = ['off', 'low', 'medium', 'high']
const DEFAULT_CONTEXT = 1_000_000
const DEFAULT_OUTPUT = 128_000
const DEFAULT_MESSAGE = 'hello'
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
// 新加进目录的模型：接口给了推理等级、上下文和输出的用接口的，没给的用默认值
const newModel = (id: string, from?: AgentModel): ModelRow => ({
  id,
  levels: from?.levels?.length ? from.levels.map((l) => levelRow(l.name, l.value)) : defaultLevels(),
  contextWindow: Number(from?.contextWindow) || DEFAULT_CONTEXT,
  maxOutput: Number(from?.maxOutput) || DEFAULT_OUTPUT,
})

const form = reactive({
  vendor: '',
  name: '',
  baseUrl: '',
  apiKey: '',
  models: [] as ModelRow[],
  mainModel: '',
  fastModel: '',
  env: [] as EnvRow[],
})
const step = ref<Step>('vendor')
const showKey = ref(false)
const modelDraft = ref('')
const openModel = ref('')
const fetching = ref(false)
const saving = ref(false)
const test = reactive({
  message: DEFAULT_MESSAGE,
  busy: false,
  results: [] as AgentProviderTestResult[],
  error: '',
})
// 每次重新测试、改了连接信息或重开弹窗都加一，过期的测试结果不再写回
let testSeq = 0

const isEdit = computed(() => !!props.provider)
const keyPlaceholder = computed(() => (props.provider?.apiKeyTail ? `sk-••••${props.provider.apiKeyTail}` : ''))

// —— 步骤 ——
const steps = computed<Step[]>(() => (isEdit.value ? ['config', 'test'] : ['vendor', 'config', 'test']))
const stepIndex = computed(() => steps.value.indexOf(step.value))

// —— 供应商类型 ——
const vendor = computed(() => props.vendors.find((v) => v.id === form.vendor))
const protocols = computed(() => vendor.value?.protocols || [])
const fixedUrl = computed(() => vendor.value?.baseUrl || '')
// 给 Claude 系的才有快速模型和最长输出
const forClaude = computed(() => protocols.value.includes('anthropic'))
// 第一步分两组：地址自己填的通用兼容接口，和地址固定的官方接口
const vendorGroups = computed(() =>
  [
    { key: 'compatible', items: props.vendors.filter((v) => !v.baseUrl) },
    { key: 'official', items: props.vendors.filter((v) => !!v.baseUrl) },
  ].filter((g) => g.items.length),
)
const urlHint = computed(() => {
  const key = `agent.providers.form.urlHint.${form.vendor}`
  const text = t(key)
  return text === key ? '' : text
})

// 共享范围，只有 shared 时用
const share = ref<AgentShare>({ share: 'all', shareUserIds: [] })

const init = () => {
  const p = props.provider
  step.value = p ? 'config' : 'vendor'
  share.value = { share: p?.share === 'users' ? 'users' : 'all', shareUserIds: [...(p?.shareUserIds ?? [])] }
  form.vendor = p?.vendor || ''
  form.name = p?.name || ''
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
  test.message = DEFAULT_MESSAGE
  resetTest()
}
watch(
  () => props.modelValue,
  (open) => {
    if (open) init()
  },
  { immediate: true },
)

// 换了类型：地址和模型目录都是上一种的，清掉；地址固定的类型名称默认用它的名字
const pickVendor = (id: string) => {
  if (form.vendor === id) return
  const before = vendorName(form.vendor)
  form.vendor = id
  form.baseUrl = ''
  form.models = []
  form.mainModel = ''
  form.fastModel = ''
  openModel.value = ''
  if (!form.name.trim() || form.name === before) form.name = fixedUrl.value ? vendorName(id) : ''
}

// —— 模型目录 ——
// 收起时一行看完：推理等级，加上最长上下文（和给 Claude 系时的最长输出）
const levelSummary = (m: ModelRow) => {
  const names = m.levels.map((r) => r.name.trim()).filter(Boolean)
  const levels = names.length ? t('agent.providers.form.levelsSummary', { list: names.join(' · ') }) : t('agent.providers.form.noLevels')
  const limits = forClaude.value
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
  if (!fixedUrl.value && !form.baseUrl.trim()) {
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
    const add = got.filter((m, i) => m.id && got.findIndex((x) => x.id === m.id) === i && !form.models.some((x) => x.id === m.id))
    add.forEach((m) => form.models.push(newModel(m.id, m)))
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

// —— 主模型 / 快速模型（只给 Codex 系的只有一个默认模型）——
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
  forClaude.value
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
  baseUrl: fixedUrl.value ? '' : form.baseUrl.trim(),
  apiKey: form.apiKey.trim(),
  models: form.models.map((m) => ({
    id: m.id,
    levels: m.levels.filter((r) => r.name.trim()).map((r) => ({ name: r.name.trim(), value: r.value.trim() })),
    contextWindow: Number(m.contextWindow) || DEFAULT_CONTEXT,
    maxOutput: Number(m.maxOutput) || DEFAULT_OUTPUT,
  })),
  mainModel: mainModel.value,
  fastModel: forClaude.value ? fastModel.value : '',
  env: form.env.filter((r) => r.key.trim()).map((r) => ({ key: r.key.trim(), value: r.value })),
})
// 配置这一步填完了才能往下走：名称、地址（地址固定的不用）、新建时的 API Key、至少一个模型（测试要用）
const configReady = () => {
  addModel()
  if (props.shared && share.value.share === 'users' && !share.value.shareUserIds.length) {
    feedback.warning(t('agent.share.needUsers'))
    return false
  }
  let missing = ''
  if (!form.name.trim()) missing = 'needName'
  else if (!fixedUrl.value && !form.baseUrl.trim()) missing = 'needUrl'
  else if (!isEdit.value && !form.apiKey.trim()) missing = 'needKey'
  else if (!form.models.length) missing = 'needModel'
  if (missing) feedback.warning(t(`agent.providers.form.${missing}`))
  return !missing
}

const next = () => {
  if (step.value === 'vendor') {
    if (vendor.value) step.value = 'config'
    return
  }
  if (step.value === 'config' && configReady()) step.value = 'test'
}
const back = () => {
  const i = stepIndex.value
  if (i > 0) step.value = steps.value[i - 1] as Step
}

// —— 测试 ——
function resetTest() {
  testSeq += 1
  test.busy = false
  test.results = []
  test.error = ''
}
// 每种协议一行：还没测时空着，测的时候是「等待回复」，测完是结果
const testRows = computed(() => {
  if (!test.busy && !test.results.length) return []
  return protocols.value.map((protocol) => {
    const result = test.results.find((r) => r.protocol === protocol)
    let pill = { cls: '', text: t('agent.providers.form.testWaiting') }
    if (result?.ok) pill = { cls: 'ag-pill--ok', text: t('agent.providers.status.okMs', { ms: Number(result.latencyMs) || 0 }) }
    else if (result) pill = { cls: 'ag-pill--bad', text: t('agent.providers.form.testFail') }
    return { protocol, result, pill }
  })
})
const runTest = async () => {
  if (test.busy || !test.message.trim()) return
  const my = ++testSeq
  test.busy = true
  test.results = []
  test.error = ''
  try {
    const { data } = await testAgentProvider({ id: props.provider?.id, draft: draft(), message: test.message.trim() })
    if (my !== testSeq) return
    test.results = data.results || []
  } catch (error) {
    if (my !== testSeq) return
    test.error = getRequestErrorMessage(error)
  } finally {
    if (my === testSeq) test.busy = false
  }
}
// 类型、地址、密钥或主模型改了，之前的测试结果就不算数了
watch(() => [form.vendor, form.baseUrl, form.apiKey, mainModel.value], resetTest)

// —— 保存 ——
const save = async () => {
  if (saving.value || !configReady()) return
  const body = { ...draft(), ...(props.shared ? share.value : {}) }
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
// 步骤条：做过的打勾，当前这一步高亮
.ag-pv-steps {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 2px;
  padding: 0;
  list-style: none;
}
.ag-pv-step {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  color: var(--m-ink2);
}
.ag-pv-step + .ag-pv-step::before {
  content: '';
  width: 18px;
  height: 1px;
  margin-right: 2px;
  background: var(--m-line);
}
.ag-pv-step__n {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--m-s2);
  font-size: 11px;
  font-weight: 600;
}
.ag-pv-step__n .iconify {
  width: 12px;
  height: 12px;
}
.ag-pv-step.is-on {
  color: var(--m-ink);
  font-weight: 600;
}
.ag-pv-step.is-on .ag-pv-step__n {
  background: var(--m-p);
  color: var(--m-s);
}
.ag-pv-step.is-done .ag-pv-step__n {
  background: var(--m-pSoft);
  color: var(--m-p);
}
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
// 测试：消息框、每种协议一块结果
.ag-pv-test__model {
  flex: none;
  font-size: 12px;
  color: var(--m-ink2);
}
.ag-pv-test__model .ag-mono {
  margin-left: 4px;
  color: var(--m-ink);
}
.ag-ta.ag-pv-test__ta {
  font-family: inherit;
  font-size: 13px;
}
.ag-pv-res {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border-radius: 10px;
  background: var(--m-s2);
}
.ag-pv-res__head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.ag-pv-res__p {
  flex: 1;
  min-width: 0;
  font-size: 12.5px;
  color: var(--m-ink2);
}
.ag-pv-res__reply {
  max-height: 220px;
  overflow-y: auto;
  font-size: 13px;
  line-height: 1.6;
  color: var(--m-ink);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.ag-pv-res__err {
  font-size: 13px;
  color: var(--m-bad);
  overflow-wrap: anywhere;
}
// 底栏：「上一步」靠左，其余靠右
.ag-pv-foot-gap {
  flex: 1;
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
}
</style>
