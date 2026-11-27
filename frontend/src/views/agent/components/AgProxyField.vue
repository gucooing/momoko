<!-- 代理（只有 Linux 能配，别的系统整块不显示）：就一个地址，留空不走代理；http:// 或 socks5:// 开头决定类型，
     账号密码写在地址里（已保存的密码显示成 ******）。填了地址才有「测试代理」：经它连到要用的那一家。 -->
<template>
  <div class="ag-px">
    <label class="ag-field">
      <span class="ag-fl">{{ t('agent.proxy.address') }}</span>
      <span class="ag-fld" :class="{ 'ag-px-bad': invalid }">
        <UIcon name="i-lucide-shield" />
        <input
          :value="modelValue"
          class="ag-mono"
          type="text"
          :placeholder="t('agent.proxy.placeholder')"
          autocomplete="off"
          spellcheck="false"
          @input="onInput(($event.target as HTMLInputElement).value)"
        />
      </span>
      <span class="ag-fh" :class="{ 'ag-px-err': invalid }">{{ invalid ? t('agent.proxy.invalid') : t('agent.proxy.format') }}</span>
    </label>

    <div v-if="modelValue.trim()" class="ag-px-test">
      <button type="button" class="ag-btn ag-btn--sm" :disabled="testing || invalid" @click="runTest">
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

    <p class="ag-px-note">
      <UIcon :name="modelValue.trim() ? 'i-lucide-shield-check' : 'i-lucide-globe'" />
      <span>{{ modelValue.trim() ? note : t('agent.proxy.noteOff') }}</span>
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
import { proxyPayload, proxyReady, type ProxyForm } from '@/views/agent/utils/proxy'

const props = defineProps<{
  modelValue: ProxyForm
  /** Linux 上现在用不了的原因（比如 Momoko 没有以 root 运行） */
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
const invalid = computed(() => !proxyReady(props.modelValue))

const onInput = (value: string) => {
  result.value = null
  emit('update:modelValue', value)
}

const runTest = async () => {
  if (testing.value || invalid.value) return
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
</style>
