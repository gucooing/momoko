<!-- 资源限制输入：数字框 + 单位后缀 + 预设 chips（含「未限制」=0）。创建页与编辑弹窗共用。v-model = 数值。 -->
<template>
  <div class="lim">
    <label class="lim__field">
      <component :is="menuStore.iconComponents[icon]" v-if="icon" class="lim__icon" />
      <input
        v-model.number="model"
        type="number"
        min="0"
        :step="step"
        class="lim__input m-num"
        :placeholder="t('docker.common.unlimited')"
        :aria-label="label"
        inputmode="decimal"
      />
      <span class="lim__unit">{{ unit }}</span>
    </label>
    <div class="lim__chips">
      <button
        v-for="p in presets"
        :key="p"
        type="button"
        class="lim__chip m-num"
        :class="{ 'is-on': model === p }"
        @click="model = p"
      >{{ p }}</button>
      <button type="button" class="lim__chip" :class="{ 'is-on': !model }" @click="model = 0">
        {{ t('docker.common.unlimited') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

withDefaults(
  defineProps<{
    unit: string
    presets: number[]
    step?: number
    icon?: string
    label?: string
  }>(),
  { step: 1, icon: '', label: '' },
)
const model = defineModel<number>({ default: 0 })
const { t } = useI18n()
const menuStore = useMenuStore()
</script>

<style scoped lang="scss">
.lim {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.lim__field {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 12px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-lighter);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.lim__field:focus-within {
  border-color: var(--m-p);
  background: var(--el-bg-color);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--m-p) 16%, transparent);
}
.lim__icon {
  width: 16px;
  height: 16px;
  flex: none;
  color: var(--el-text-color-secondary);
}
.lim__input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 0.9375rem;
  font-weight: 700;
}
.lim__input::placeholder {
  color: var(--el-text-color-placeholder);
  font-weight: 500;
}
.lim__unit {
  flex: none;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.lim__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.lim__chip {
  min-height: 30px;
  padding: 0 11px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}
.lim__chip:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.lim__chip.is-on {
  border-color: var(--m-ink);
  background: var(--m-ink);
  color: var(--m-bg);
}
</style>
