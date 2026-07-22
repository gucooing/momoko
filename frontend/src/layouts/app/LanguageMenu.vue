<!-- 语言切换（复用 languageOptions / setAppLocale 逻辑，重写视觉）。 -->
<template>
  <AppDropdown align="end" :width="180">
    <template #trigger>
      <AppIconButton icon="HOutline:GlobeAltIcon" :label="t('language.tooltip')" />
    </template>
    <template #default="{ close }">
      <div class="lang-menu">
        <button
          v-for="item in languageOptions"
          :key="item.code"
          type="button"
          class="lang-row"
          :class="{ 'is-active': locale === item.code }"
          @click="onPick(item.code, close)"
        >
          <span class="lang-row__code">{{ item.shortLabel }}</span>
          <span class="lang-row__label">{{ t(item.labelKey) }}</span>
          <component
            :is="menuStore.iconComponents['HOutline:CheckIcon']"
            v-if="locale === item.code"
            class="lang-row__check"
          />
        </button>
      </div>
    </template>
  </AppDropdown>
</template>

<script setup lang="ts">
import { languageOptions, setAppLocale } from '@/locales'
import { useI18n } from 'vue-i18n'

const { locale, t } = useI18n()
const menuStore = useMenuStore()

const onPick = (code: string, close: () => void) => {
  setAppLocale(code as never)
  close()
}
</script>

<style scoped lang="scss">
.lang-menu {
  padding: 6px;
}
.lang-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  padding: 8px 10px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  cursor: pointer;
  color: var(--el-text-color-regular);
  transition: background 0.15s, color 0.15s;
}
.lang-row:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.lang-row.is-active {
  color: var(--el-color-primary);
}
.lang-row__code {
  min-width: 30px;
  font-size: 0.8125rem;
  font-weight: 600;
  letter-spacing: 0.02em;
}
.lang-row__label {
  flex: 1;
  text-align: left;
  font-size: 0.875rem;
}
.lang-row__check {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
</style>
