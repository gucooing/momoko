<!-- 添加 / 编辑供应商，分三步，每步一页，可以回上一步：
     ① 类型：通用的兼容接口，或对某一家的专门适配（类型由后端列出；编辑时类型不能改，没有这一步）；
     ② 配置：名称、接口地址（地址固定的类型没有）、API Key、模型目录、主模型 / 快速模型、额外环境变量；
     （Linux 上）代理：用这个接口的会话、测试、拉模型、查余额都只经它联网；
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
        <AgModelCatalog
          ref="catalogRef"
          v-model="form.models"
          addable
          levels
          context
          :output="forClaude"
          :limits-hint="t(forClaude ? 'agent.providers.form.limitsHint' : 'agent.providers.form.limitsHintCodex')"
          :value-label="t('agent.providers.form.levelValue')"
          :levels-hint="t('agent.providers.form.levelsHint')"
          :create="newModel"
        />
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

    <!-- 代理（只有 Linux） -->
    <template v-else-if="step === 'proxy'">
      <AgProxyField
        v-model="proxy"
        :unavailable="proxyUnavailable"
        :note="t('agent.proxy.noteProvider')"
        :provider-id="provider?.id"
        :vendor="form.vendor"
        :base-url="fixedUrl ? '' : form.baseUrl.trim()"
      />
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
import AgModelCatalog from '@/views/agent/components/AgModelCatalog.vue'
import AgProxyField from '@/views/agent/components/AgProxyField.vue'
import AgSelect from '@/views/agent/components/AgSelect.vue'
import AgShareField from '@/views/agent/components/AgShareField.vue'
import { fmtTokens, protocolAgent } from '@/views/agent/utils/format'
import { type CatalogRow, defaultLevels, levelRow } from '@/views/agent/utils/catalog'
import { vendorDesc, vendorName } from '@/views/agent/utils/labels'
import { proxyForm, proxyPayload, proxyReady, type ProxyForm } from '@/views/agent/utils/proxy'

type Step = 'vendor' | 'config' | 'proxy' | 'test'
/** 模型目录的一行：最长上下文和最长输出（token），新模型默认 1M / 128K */
type ModelRow = CatalogRow
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
  /** 这台服务器能配代理（只有 Linux）；proxyUnavailable 是现在用不了的原因 */
  proxySupported?: boolean
  proxyUnavailable?: string
}>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
}>()
const { t } = useI18n()

const DEFAULT_CONTEXT = 1_000_000
const DEFAULT_OUTPUT = 128_000
const DEFAULT_MESSAGE = 'hello'
let uidSeed = 0
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
const proxy = ref<ProxyForm>(proxyForm())
const showKey = ref(false)
const catalogRef = ref<InstanceType<typeof AgModelCatalog> | null>(null)
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
// 能配代理（Linux）时配置之后多一步「代理」
const steps = computed<Step[]>(() => {
  const list: Step[] = isEdit.value ? ['config', 'test'] : ['vendor', 'config', 'test']
  if (props.proxySupported) list.splice(list.indexOf('test'), 0, 'proxy')
  return list
})
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
  proxy.value = proxyForm(p?.proxy)
  showKey.value = false
  catalogRef.value?.reset()
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
  catalogRef.value?.reset()
  if (!form.name.trim() || form.name === before) form.name = fixedUrl.value ? vendorName(id) : ''
}

// —— 模型目录（列表本身在 AgModelCatalog 里）——
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
      ...(props.proxySupported ? { proxy: proxyPayload(proxy.value) } : {}),
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
  // 不能配代理的系统不带这一项（后端当作不改）
  ...(props.proxySupported ? { proxy: proxyPayload(proxy.value) } : {}),
})
// 配置这一步填完了才能往下走：名称、地址（地址固定的不用）、新建时的 API Key、至少一个模型（测试要用）
const configReady = () => {
  // 手动添加框里还没回车的也算进目录
  catalogRef.value?.flush()
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
// 代理这一步：走代理就要填地址和端口
const proxyStepReady = () => {
  if (!props.proxySupported || proxyReady(proxy.value)) return true
  feedback.warning(t('agent.proxy.notReady'))
  return false
}

const next = () => {
  if (step.value === 'vendor') {
    if (vendor.value) step.value = 'config'
    return
  }
  if (step.value === 'config' && configReady()) step.value = props.proxySupported ? 'proxy' : 'test'
  else if (step.value === 'proxy' && proxyStepReady()) step.value = 'test'
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
watch(() => [form.vendor, form.baseUrl, form.apiKey, mainModel.value, JSON.stringify(proxy.value)], resetTest)

// —— 保存 ——
const save = async () => {
  if (saving.value || !configReady() || !proxyStepReady()) return
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
.ag-fld.ag-pv-key {
  padding-right: 4px;
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
