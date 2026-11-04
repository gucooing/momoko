<!-- 公开页表单字段：标签 + 前置图标输入框（浅底圆角，聚焦描主色边）+ 可选「显示密码」与尾部插槽（如获取验证码）。
     登录 / 注册 / 找回密码共用。 -->
<template>
  <div class="af" :class="{ 'is-error': !!error, 'is-disabled': disabled }">
    <label class="af__label" :for="id">{{ label }}</label>
    <span class="af__box">
      <component :is="menuStore.iconComponents[icon]" v-if="icon" class="af__ico" aria-hidden="true" />
      <input
        :id="id"
        v-model="model"
        class="af__input"
        :class="{ 'm-mono': mono }"
        :type="inputType"
        :placeholder="placeholder"
        :autocomplete="autocomplete"
        :inputmode="inputmode"
        :maxlength="maxlength"
        :disabled="disabled"
        :aria-invalid="!!error || undefined"
        :aria-describedby="error ? `${id}-err` : undefined"
        spellcheck="false"
        @input="emit('input', $event)"
      />
      <button
        v-if="revealable"
        type="button"
        class="af__eye"
        :aria-label="shown ? t('login.hidePassword') : t('login.showPassword')"
        :aria-pressed="shown"
        @click="shown = !shown"
      >
        <component
          :is="menuStore.iconComponents[shown ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon']"
          aria-hidden="true"
        />
      </button>
      <slot name="trail" />
    </span>
    <span v-if="error" :id="`${id}-err`" class="af__error" role="alert">{{ error }}</span>
    <slot name="below" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    id: string
    label: string
    icon?: string
    type?: string
    placeholder?: string
    autocomplete?: string
    inputmode?: 'text' | 'email' | 'numeric' | 'tel' | 'url' | 'search' | 'none' | 'decimal'
    maxlength?: number
    mono?: boolean
    /** 密码框：显示「显示 / 隐藏密码」按钮 */
    revealable?: boolean
    /** 受控的显示状态（多个密码框联动时用）；不传则组件自管 */
    reveal?: boolean
    error?: string
    disabled?: boolean
  }>(),
  { type: 'text', reveal: undefined },
)

const emit = defineEmits<{
  (e: 'input', ev: Event): void
  (e: 'update:reveal', v: boolean): void
}>()

const model = defineModel<string>({ default: '' })
const { t } = useI18n()
const menuStore = useMenuStore()

const innerShown = ref(false)
const shown = computed({
  get: () => (props.reveal === undefined ? innerShown.value : props.reveal),
  set: (v: boolean) => {
    innerShown.value = v
    emit('update:reveal', v)
  },
})
const inputType = computed(() => (props.revealable || props.reveal !== undefined) && shown.value ? 'text' : props.type)
</script>

<style scoped lang="scss">
.af {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.af__label {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.af__box {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 0 6px 0 16px;
  border-radius: 16px;
  border: 1.5px solid transparent;
  background: var(--m-s2);
  color: var(--el-text-color-secondary);
  transition:
    border-color 0.15s,
    background-color 0.15s;
}
.af__box:focus-within {
  border-color: var(--m-p);
  background: var(--el-bg-color);
}
.af__box:focus-within .af__ico {
  color: var(--m-pInk);
}
.af.is-error .af__box {
  border-color: var(--m-bad);
}
.af.is-disabled .af__box {
  opacity: 0.6;
}
.af__ico {
  flex: none;
  width: 20px;
  height: 20px;
}
.af__input {
  flex: 1;
  min-width: 0;
  height: 48px;
  padding: 0;
  border: 0;
  background: transparent;
  font: inherit;
  font-size: 0.9375rem;
  font-weight: 500;
  color: var(--el-text-color-primary);
  outline: none;
}
.af__input.m-mono {
  font-size: 0.875rem;
}
.af__input::placeholder {
  color: var(--el-text-color-placeholder);
}
.af__input:disabled {
  cursor: not-allowed;
}
.af__eye {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.af__eye:hover {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
}
.af__eye svg {
  width: 20px;
  height: 20px;
}
.af__error {
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.4;
  color: var(--m-bad);
}

/* 手机：16px 字号避免 iOS 聚焦缩放 */
@media (width < 900px) {
  .af__input {
    font-size: 1rem;
  }
}
</style>
