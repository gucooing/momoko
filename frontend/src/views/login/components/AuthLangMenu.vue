<!-- 公开页语言切换：描边胶囊（地球图标 + 当前语言名）；compact 时只显示图标（手机品牌区用，浅色描边适配主色底）。
     语言列表与切换逻辑复用 locales 的 languageOptions / setAppLocale。 -->
<template>
  <AppDropdown align="end" :width="200">
    <template #trigger="{ open }">
      <button
        type="button"
        class="alm"
        :class="{ 'alm--compact': compact, 'alm--brand': onBrand }"
        :aria-label="t('language.tooltip')"
        :aria-expanded="open"
        aria-haspopup="menu"
      >
        <component :is="menuStore.iconComponents['HOutline:GlobeAltIcon']" class="alm__ico" aria-hidden="true" />
        <span v-if="!compact" class="alm__text">{{ currentLabel }}</span>
      </button>
    </template>
    <template #default="{ close }">
      <div class="alm-menu" role="menu">
        <button
          v-for="item in languageOptions"
          :key="item.code"
          type="button"
          role="menuitemradio"
          class="alm-row"
          :class="{ 'is-on': locale === item.code }"
          :aria-checked="locale === item.code"
          @click="pick(item.code, close)"
        >
          <span class="alm-row__code">{{ item.shortLabel }}</span>
          <span class="alm-row__label">{{ t(item.labelKey) }}</span>
          <component
            :is="menuStore.iconComponents['HOutline:CheckIcon']"
            v-if="locale === item.code"
            class="alm-row__check"
            aria-hidden="true"
          />
        </button>
      </div>
    </template>
  </AppDropdown>
</template>

<script setup lang="ts">
import { languageOptions, setAppLocale } from '@/locales'
import type { AppLocale } from '@/locales/types'
import { useI18n } from 'vue-i18n'

defineProps<{
  /** 只显示图标 */
  compact?: boolean
  /** 放在主色品牌区上 */
  onBrand?: boolean
}>()

const { locale, t } = useI18n()
const menuStore = useMenuStore()

const currentLabel = computed(() => {
  const item = languageOptions.find((o) => o.code === locale.value)
  return item ? t(item.labelKey) : String(locale.value)
})

const pick = (code: AppLocale, close: () => void) => {
  setAppLocale(code)
  close()
}
</script>

<style scoped lang="scss">
.alm {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--m-line);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition:
    border-color 0.15s,
    color 0.15s,
    background-color 0.15s;
}
.alm:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.alm__ico {
  width: 16px;
  height: 16px;
  flex: none;
}
.alm--compact {
  width: 44px;
  height: 44px;
  padding: 0;
  border-radius: 14px;
}
.alm--compact .alm__ico {
  width: 20px;
  height: 20px;
}
/* 品牌区上：实心底 + 主色图标（同品牌区贴纸配色），压在黄色圆上也看得清 */
.alm--brand {
  border-color: transparent;
  background: var(--m-pOn);
  color: var(--m-p);
}
.alm--brand:hover {
  border-color: transparent;
  background: color-mix(in srgb, var(--m-pOn) 86%, var(--m-p));
  color: var(--m-p);
}

.alm-menu {
  padding: 6px;
}
.alm-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 40px;
  padding: 0 10px;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-regular);
  font: inherit;
  cursor: pointer;
}
.alm-row:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.alm-row.is-on {
  color: var(--m-pInk);
  font-weight: 700;
}
.alm-row__code {
  min-width: 28px;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.02em;
}
.alm-row__label {
  flex: 1;
  text-align: left;
  font-size: 0.875rem;
}
.alm-row__check {
  width: 16px;
  height: 16px;
  flex: none;
}
</style>
