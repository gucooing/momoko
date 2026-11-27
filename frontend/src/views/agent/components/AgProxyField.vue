<!-- 代理（只有 Linux 能配，别的系统整块不显示）：一个节点，留空不走代理；填分享链接或 sing-box 的出站 JSON。
     填了节点后：沙箱里的时区和语言，「测试代理」经它连到要用的那一家。 -->
<template>
  <div class="ag-px">
    <label class="ag-field">
      <span class="ag-fl">{{ t('agent.proxy.node') }}</span>
      <textarea
        :value="modelValue.node"
        class="ag-ta ag-mono ag-px-node"
        :class="{ 'ag-px-bad': !nodeValid }"
        rows="4"
        :placeholder="t('agent.proxy.placeholder')"
        autocomplete="off"
        spellcheck="false"
        @input="patch({ node: ($event.target as HTMLTextAreaElement).value })"
      />
      <span class="ag-fh" :class="{ 'ag-px-err': !nodeValid }">{{ nodeValid ? t('agent.proxy.format') : t('agent.proxy.invalid') }}</span>
    </label>

    <template v-if="active">
      <div class="ag-px-grid">
        <label class="ag-field">
          <span class="ag-fl">{{ t('agent.proxy.timezone') }}</span>
          <span class="ag-fld" :class="{ 'ag-px-bad': !tzOk }">
            <UIcon name="i-lucide-clock" />
            <input
              :value="modelValue.timezone"
              class="ag-mono"
              type="text"
              list="ag-px-timezones"
              :placeholder="t('agent.proxy.sameAsServer')"
              autocomplete="off"
              spellcheck="false"
              @input="patch({ timezone: ($event.target as HTMLInputElement).value })"
            />
          </span>
        </label>
        <label class="ag-field">
          <span class="ag-fl">{{ t('agent.proxy.locale') }}</span>
          <span class="ag-fld" :class="{ 'ag-px-bad': !localeValid }">
            <UIcon name="i-lucide-languages" />
            <input
              :value="modelValue.locale"
              class="ag-mono"
              type="text"
              list="ag-px-locales"
              :placeholder="t('agent.proxy.sameAsServer')"
              autocomplete="off"
              spellcheck="false"
              @input="patch({ locale: ($event.target as HTMLInputElement).value })"
            />
          </span>
        </label>
      </div>
      <span class="ag-fh ag-px-envhint" :class="{ 'ag-px-err': !tzOk || !localeValid }">
        {{ !tzOk ? t('agent.proxy.timezoneInvalid') : !localeValid ? t('agent.proxy.localeInvalid') : t('agent.proxy.envHint') }}
      </span>
      <datalist id="ag-px-timezones">
        <option v-for="z in COMMON_TIMEZONES" :key="z" :value="z" />
      </datalist>
      <datalist id="ag-px-locales">
        <option v-for="l in COMMON_LOCALES" :key="l" :value="l" />
      </datalist>

      <div class="ag-px-test">
        <button type="button" class="ag-btn ag-btn--sm" :disabled="testing || !nodeValid" @click="runTest">
          <UIcon :name="testing ? 'i-lucide-loader-circle' : 'i-lucide-radar'" :class="{ 'ag-spin': testing }" />{{
            t('agent.proxy.test')
          }}
        </button>
        <span v-if="result?.ok" class="ag-pill ag-pill--ok">
          {{ t('agent.proxy.testOk', { target: result.target, ms: Number(result.latencyMs) || 0 }) }}
        </span>
        <span v-else-if="result" class="ag-pill ag-pill--bad" :title="result.error">{{ t('agent.proxy.testFail') }}</span>
      </div>
      <span v-if="result && !result.ok && result.error" class="ag-px-err">{{ result.error }}</span>
    </template>

    <p class="ag-px-note">
      <UIcon :name="active ? 'i-lucide-shield-check' : 'i-lucide-globe'" />
      <span>{{ active ? note : t('agent.proxy.noteOff') }}</span>
    </p>
    <p v-if="unavailable" class="ag-px-warn">
      <UIcon name="i-lucide-triangle-alert" />
      <span>{{ t('agent.proxy.unavailable', { reason: unavailable }) }}</span>
    </p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { testAgentProxy } from '@/api/agent'
import type { TestAgentProxyResponse } from '@/types/v1/agent'
import {
  COMMON_LOCALES,
  COMMON_TIMEZONES,
  localeOk,
  nodeOk,
  proxyPayload,
  type ProxyForm,
  timezoneOk,
} from '@/views/agent/utils/proxy'

const props = defineProps<{
  modelValue: ProxyForm
  /** Linux 上现在建不了沙箱的原因（比如 Momoko 没有以 root 运行） */
  unavailable?: string
  /** 走代理时底下那一行说明 */
  note: string
  /** 测试时连去哪里：官方账户给 runtimeId；第三方接口给 providerId，或者还没保存时给类型和接口地址 */
  runtimeId?: string
  providerId?: string
  vendor?: string
  baseUrl?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: ProxyForm] }>()
const { t } = useI18n()

const testing = ref(false)
const result = ref<TestAgentProxyResponse | null>(null)
const active = computed(() => !!props.modelValue.node.trim())
const nodeValid = computed(() => nodeOk(props.modelValue))
const tzOk = computed(() => timezoneOk(props.modelValue))
const localeValid = computed(() => localeOk(props.modelValue))

const patch = (p: Partial<ProxyForm>) => {
  result.value = null
  emit('update:modelValue', { ...props.modelValue, ...p })
}

const runTest = async () => {
  if (testing.value || !nodeValid.value) return
  testing.value = true
  result.value = null
  try {
    const { data } = await testAgentProxy({
      proxy: proxyPayload(props.modelValue),
      runtimeId: props.runtimeId || '',
      providerId: props.providerId || '',
      vendor: props.vendor || '',
      baseUrl: props.baseUrl || '',
    })
    result.value = data
  } catch {
    /* shown */
  } finally {
    testing.value = false
  }
}
</script>

<style lang="scss">
.ag-px {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.ag-px-bad {
  border-color: var(--m-bad);
}
.ag-px-node {
  min-height: 84px;
  font-size: 12px;
  word-break: break-all;
  resize: vertical;
}
.ag-px-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 10px;
}
.ag-px-envhint {
  margin-top: -6px;
}
.ag-px-test {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.ag-px-err {
  font-size: 12px;
  color: var(--m-bad);
  overflow-wrap: anywhere;
}
.ag-px-note,
.ag-px-warn {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: 0;
  padding: 8px 10px;
  border-radius: 10px;
  font-size: 12px;
  line-height: 1.55;
}
.ag-px-note {
  background: var(--m-s2);
  color: var(--m-ink2);
}
.ag-px-warn {
  background: var(--m-warnSoft);
  color: var(--m-warn);
}
.ag-px-note .iconify,
.ag-px-warn .iconify {
  flex: none;
  width: 14px;
  height: 14px;
  margin-top: 2px;
}
@media (width < 560px) {
  .ag-px-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
