<!-- 提问卡片：Agent 向人提的问题（ACP 表单询问，Claude 的 AskUserQuestion），照官方客户端的样子一题一题答：
     上面一排是各题的标签（答过的打勾，点了能回去改），一次只显示一题，选项带编号（数字键也能选），
     最后一项「其他」直接打字。选了不自动往下走，点「下一题」（或回车）才走；几道题都答完先确认一遍再提交，
     只有一题时没有确认页，选好点「提交」。
     答过之后收成一行，显示答了什么。 -->
<template>
  <div
    v-if="!question.outcome"
    ref="card"
    class="agc-ask"
    role="group"
    tabindex="-1"
    :aria-label="t('agent.code.ask.need', { agent: agentName })"
    @keydown="onKey"
  >
    <div class="agc-ask__head">
      <UIcon name="i-lucide-message-circle-question" />
      <span>{{ t('agent.code.ask.need', { agent: agentName }) }}</span>
      <span v-if="steps" class="agc-ask__count ag-num">{{ Math.min(step + 1, mains.length) }} / {{ mains.length }}</span>
    </div>
    <div v-if="intro" class="agc-ask__intro ag-muted">{{ question.message }}</div>

    <div v-if="steps" class="agc-ask__tabs" role="tablist">
      <button
        v-for="(f, i) in mains"
        :key="f.key"
        type="button"
        class="agc-ask__tab"
        :class="{ 'is-on': step === i, 'is-done': answered(f) }"
        role="tab"
        :aria-selected="step === i"
        :disabled="busy"
        @click="go(i)"
      >
        <UIcon :name="answered(f) ? 'i-lucide-check' : 'i-lucide-circle-dashed'" />
        <span class="ag-ellipsis">{{ f.title || t('agent.code.ask.nth', { n: i + 1 }) }}</span>
      </button>
      <button
        type="button"
        class="agc-ask__tab"
        :class="{ 'is-on': reviewing }"
        role="tab"
        :aria-selected="reviewing"
        :disabled="busy || !complete"
        @click="go(mains.length)"
      >
        <UIcon name="i-lucide-send" />
        <span>{{ t('agent.code.ask.confirm') }}</span>
      </button>
    </div>

    <!-- 一道题 -->
    <template v-if="current">
      <div class="agc-ask__qtext">
        {{ textOfQuestion(current) }}
        <span v-if="current.kind === 'multi'" class="ag-muted agc-small">{{ t('agent.code.ask.multi') }}</span>
      </div>
      <div v-if="choice(current)" class="agc-ask__opts" :role="current.kind === 'multi' ? 'group' : 'radiogroup'">
        <button
          v-for="(o, i) in optionsOf(current)"
          :key="o.value"
          type="button"
          class="agc-ask__opt"
          :class="{ 'is-on': picked(current.key, o.value) }"
          :role="current.kind === 'multi' ? 'checkbox' : 'radio'"
          :aria-checked="picked(current.key, o.value)"
          :disabled="busy"
          @click="pick(current, o.value)"
        >
          <span class="agc-ask__no ag-num">{{ i + 1 }}</span>
          <span class="agc-ask__opttext">
            <span class="agc-ask__label">{{ o.label }}</span>
            <span v-if="o.description" class="ag-muted agc-ask__desc">{{ o.description }}</span>
          </span>
          <UIcon v-if="picked(current.key, o.value)" name="i-lucide-check" class="agc-ask__tick" />
        </button>
        <!-- 「其他」：最后一个选项，直接打字；单选时打了字就算选了它 -->
        <label v-if="customOf(current.key)" class="agc-ask__opt agc-ask__opt--other" :class="{ 'is-on': !!textOf(customOf(current.key)!.key) }">
          <span class="agc-ask__no ag-num">{{ optionsOf(current).length + 1 }}</span>
          <input
            ref="otherInput"
            v-model="texts[customOf(current.key)!.key]"
            class="agc-ask__other"
            type="text"
            :placeholder="t('agent.code.ask.otherHint')"
            :aria-label="t('agent.code.ask.other')"
            :disabled="busy"
            @input="onOther(current)"
            @keydown.enter.prevent="next"
          />
        </label>
      </div>
      <input
        v-else
        v-model="texts[current.key]"
        class="agc-ask__input"
        :type="current.kind === 'number' || current.kind === 'integer' ? 'number' : 'text'"
        :step="current.kind === 'integer' ? 1 : undefined"
        :aria-label="textOfQuestion(current)"
        :disabled="busy"
        @keydown.enter.prevent="next"
      />
    </template>

    <!-- 几道题都答完：确认一遍 -->
    <template v-else>
      <div class="agc-ask__qtext">{{ t('agent.code.ask.reviewTitle') }}</div>
      <dl class="agc-ask__review">
        <template v-for="(f, i) in mains" :key="f.key">
          <dt>
            <button type="button" class="agc-ask__back" :disabled="busy" @click="go(i)">{{ textOfQuestion(f) }}</button>
          </dt>
          <dd>{{ answerText(f) || t('agent.code.ask.none') }}</dd>
        </template>
      </dl>
    </template>

    <div class="agc-perm__acts">
      <button v-if="step > 0" type="button" class="ag-btn ag-btn--o ag-btn--sm" :disabled="busy" @click="go(step - 1)">
        {{ t('agent.code.ask.prev') }}
      </button>
      <button type="button" class="ag-btn ag-btn--sm ag-btn--p" :disabled="busy || !canNext" @click="next">
        {{ lastStep ? t('agent.code.ask.submit') : t('agent.code.ask.next') }}
      </button>
      <span class="agc-flex" />
      <button type="button" class="ag-btn ag-btn--o ag-btn--sm" :disabled="busy" @click="emit('answer', 'decline', {})">
        {{ t('agent.code.ask.skip') }}
      </button>
    </div>
  </div>

  <div v-else class="agc-tool agc-tool--flat">
    <UIcon :name="doneIcon" :class="question.outcome === 'accept' ? 'agc-ok' : 'ag-muted'" />
    <span class="agc-tool__verb">{{ doneVerb }}</span>
    <span class="ag-ellipsis agc-tool__arg" :title="doneText">{{ doneText }}</span>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { AgentQuestion, AgentQuestionField, AgentQuestionOption } from './types'

defineOptions({ name: 'AgentQuestionCard' })

const props = withDefaults(defineProps<{ question: AgentQuestion; agentName?: string; busy?: boolean }>(), { agentName: '', busy: false })
const emit = defineEmits<{ answer: [action: 'accept' | 'decline', answers: Record<string, string[]>] }>()
const { t } = useI18n()

/** 每道题一个字段；「其他」那格挂在它的题下面，不单独算一题 */
const mains = computed(() => props.question.fields.filter((f) => !f.for))
const customs = computed(() => new Map(props.question.fields.filter((f) => f.for).map((f) => [f.for!, f])))
const customOf = (key: string) => customs.value.get(key)
/** 不止一题：上面一排标签，最后确认一遍 */
const steps = computed(() => mains.value.length > 1)
/** 总的说明只在它不是废话时显示：一题时它就是题目本身；几题都各有题目时它只是「请回答下面的问题」 */
const intro = computed(() => steps.value && !mains.value.every((f) => f.description))
const textOfQuestion = (f: AgentQuestionField) => f.description || (steps.value ? f.title || '' : props.question.message || f.title || '')

// —— 答案：选中的值（单选、多选、是否）和填的字（填空、「其他」）——
const picks = reactive<Record<string, string[]>>({})
const texts = reactive<Record<string, string | number>>({})
const textOf = (key: string) => String(texts[key] ?? '').trim()

const choice = (f: AgentQuestionField) => f.kind === 'single' || f.kind === 'multi' || f.kind === 'boolean'
const optionsOf = (f: AgentQuestionField): AgentQuestionOption[] =>
  f.kind === 'boolean'
    ? [
        { value: 'true', label: t('agent.code.ask.yes') },
        { value: 'false', label: t('agent.code.ask.no') },
      ]
    : (f.options ?? [])
const picked = (key: string, value: string) => !!picks[key]?.includes(value)
/** 这一题答了没有：选了，或者填了字（包括「其他」）；不是必答的填空题空着也算 */
const answered = (f: AgentQuestionField) => {
  const custom = customOf(f.key)
  const has = (picks[f.key]?.length ?? 0) > 0 || !!textOf(f.key) || (!!custom && !!textOf(custom.key))
  return has || (!choice(f) && !f.required)
}
const complete = computed(() => mains.value.every(answered))

// —— 一题一题来 ——
const step = ref(0)
const reviewing = computed(() => steps.value && step.value >= mains.value.length)
const current = computed(() => (reviewing.value ? null : (mains.value[step.value] ?? null)))
/** 这一步按下去就提交：确认页，或者只有一题 */
const lastStep = computed(() => reviewing.value || !steps.value)
const canNext = computed(() => (reviewing.value ? complete.value : !!current.value && answered(current.value)))
const card = ref<HTMLElement | null>(null)
const otherInput = ref<HTMLInputElement | HTMLInputElement[] | null>(null)
const go = (i: number) => {
  step.value = Math.max(0, Math.min(i, mains.value.length))
  void nextTick(() => card.value?.focus({ preventScroll: true }))
}
const next = () => {
  if (props.busy || !canNext.value) return
  if (lastStep.value) submit()
  else go(step.value + 1)
}

const pick = (f: AgentQuestionField, value: string) => {
  const cur = picks[f.key] ?? []
  if (f.kind === 'multi') {
    picks[f.key] = cur.includes(value) ? cur.filter((v) => v !== value) : [...cur, value]
    return
  }
  // 单选：选了选项就不用「其他」了。选了不自动往下走，点「下一题」（或回车）才走
  picks[f.key] = [value]
  const custom = customOf(f.key)
  if (custom) texts[custom.key] = ''
}
/** 单选时在「其他」里打字：就当选了它，清掉选中的选项 */
const onOther = (f: AgentQuestionField) => {
  if (f.kind !== 'multi' && textOf(customOf(f.key)!.key)) picks[f.key] = []
}

/** 数字键选第几个（最后一个是「其他」），回车下一步；在输入框里打字、焦点在按钮上按回车（按那个按钮）时不管 */
const onKey = (e: KeyboardEvent) => {
  if (e.target instanceof HTMLInputElement || e.altKey || e.ctrlKey || e.metaKey) return
  const f = current.value
  if (e.key === 'Enter' && !(e.target instanceof HTMLButtonElement)) {
    e.preventDefault()
    next()
    return
  }
  if (!f || !choice(f) || !/^[1-9]$/.test(e.key)) return
  const i = Number(e.key) - 1
  const opts = optionsOf(f)
  if (i < opts.length) {
    e.preventDefault()
    pick(f, opts[i]!.value)
  } else if (i === opts.length && customOf(f.key)) {
    e.preventDefault()
    const el = Array.isArray(otherInput.value) ? otherInput.value[0] : otherInput.value
    el?.focus()
  }
}
// 卡片出现时把焦点给它（数字键马上能用）；正在别处打字时不抢
onMounted(() => {
  const active = document.activeElement
  if (!active || active === document.body) card.value?.focus({ preventScroll: true })
})

const submit = () => {
  const answers: Record<string, string[]> = {}
  for (const f of props.question.fields) {
    const vals = picks[f.key]?.length ? picks[f.key]! : textOf(f.key) ? [textOf(f.key)] : []
    if (vals.length) answers[f.key] = vals
  }
  emit('answer', 'accept', answers)
}

/** 一题答了什么：选中的选项名，加上「其他」里写的 */
const answerText = (f: AgentQuestionField, answers?: Record<string, string[]>) => {
  const get = (key: string) => (answers ? (answers[key] ?? []) : picks[key]?.length ? picks[key]! : textOf(key) ? [textOf(key)] : [])
  const labels = get(f.key).map((v) => optionsOf(f).find((o) => o.value === v)?.label ?? v)
  const custom = customOf(f.key)
  if (custom) labels.push(...get(custom.key))
  return labels.join('、')
}

// —— 答过之后 ——
const doneIcon = computed(
  () => ({ accept: 'i-lucide-circle-check', decline: 'i-lucide-circle-slash' })[props.question.outcome || ''] ?? 'i-lucide-circle-minus',
)
const doneVerb = computed(() => {
  if (props.question.outcome === 'accept') return t('agent.code.ask.answered')
  if (props.question.outcome === 'decline') return t('agent.code.ask.skipped')
  return t('agent.code.ask.stopped')
})
const doneText = computed(() => {
  const q = props.question
  if (q.outcome === 'cancelled') return t('agent.code.ask.stoppedHint')
  if (q.outcome !== 'accept') return q.message
  const parts = mains.value.map((f) => answerText(f, q.answers ?? {})).filter(Boolean)
  return parts.join('；') || q.message
})
</script>
