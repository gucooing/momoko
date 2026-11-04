<!-- 输入框尾部的「获取验证码」小胶囊：可发送时主色实底，倒计时中为描边灰字；手机上用短文案。 -->
<template>
  <button
    type="button"
    class="scb"
    :class="{ 'is-wait': countdown > 0 }"
    :disabled="disabled || sending || countdown > 0"
    :aria-disabled="disabled || sending || countdown > 0"
    @click="emit('send')"
  >
    <template v-if="countdown > 0">
      <span class="scb__long">{{ t('authPage.code.resendIn', { n: countdown }) }}</span>
      <span class="scb__short">{{ t('authPage.code.resendShort', { n: countdown }) }}</span>
    </template>
    <template v-else>
      <span class="scb__long">{{ t('authPage.code.get') }}</span>
      <span class="scb__short">{{ t('authPage.code.getShort') }}</span>
    </template>
  </button>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{ countdown: number; sending?: boolean; disabled?: boolean }>()
const emit = defineEmits<{ (e: 'send'): void }>()
const { t } = useI18n()
</script>

<style scoped lang="scss">
.scb {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 40px;
  padding: 0 14px;
  border: 1.5px solid transparent;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  font-variant-numeric: tabular-nums;
  transition:
    filter 0.15s,
    transform 0.15s;
}
.scb:hover:not(:disabled) {
  filter: brightness(1.08);
}
.scb:active:not(:disabled) {
  transform: scale(0.98);
}
.scb:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.scb.is-wait,
.scb.is-wait:disabled {
  opacity: 1;
  border-color: var(--m-line);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  cursor: default;
}
.scb__short {
  display: none;
}
@media (width < 900px) {
  .scb {
    padding: 0 12px;
  }
  .scb__long {
    display: none;
  }
  .scb__short {
    display: inline;
  }
}
</style>
