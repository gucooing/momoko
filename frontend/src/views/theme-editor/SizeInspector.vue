<!-- 编辑器右栏 · 组件尺寸：选中组件（左栏「组件尺寸」或在「组件」标签下点预览里的元素）后列出它登记的所有尺寸，
     每项一个滑块 + 数值框，改过的可以恢复；作用范围跟随「改动作用于 全局 / 仅本页」。尺寸清单来自 theme/sizeTokens.ts。 -->
<template>
  <div v-if="group" class="szi">
    <div class="szi-head">
      <span class="szi-lbl">{{ ed.selName.value }}</span>
      <span class="szi-tag" :class="{ 'is-page': ed.scope.value === 'page' }">{{ ed.scopeTag.value }}</span>
    </div>
    <div v-for="tk in group.tokens" :key="tk.key" class="szi-row">
      <div class="szi-row__head">
        <span class="szi-row__name">
          {{ ed.sizeLabel(tk.key) }}
          <span v-if="ed.isPageSize(tk.key)" class="szi-tagx">{{ t('theme.editor.pageTag') }}</span>
          <span v-if="ed.isSizeEdited(tk.key)" class="szi-dot" :title="t('theme.editor.edited')" />
        </span>
        <label class="szi-num">
          <input
            type="number"
            :min="tk.min"
            :max="tk.max"
            :step="tk.step || 1"
            :value="ed.sizeValue(tk.key)"
            :aria-label="ed.sizeLabel(tk.key)"
            @change="onNumber(tk.key, $event)"
          />
          <small>{{ tk.unit === 'scale' ? '%' : 'px' }}</small>
        </label>
        <button
          v-if="ed.isSizeEdited(tk.key)"
          type="button"
          class="szi-reset"
          :title="t('theme.editor.sizeReset', { v: display(tk.key, tk.default) })"
          :aria-label="t('theme.editor.sizeReset', { v: display(tk.key, tk.default) })"
          @click="ed.revertSize(tk.key)"
        >
          <ArrowUturnLeftIcon aria-hidden="true" />
        </button>
      </div>
      <input
        class="szi-rng"
        type="range"
        :min="tk.min"
        :max="tk.max"
        :step="tk.step || 1"
        :value="ed.sizeValue(tk.key)"
        :aria-label="ed.sizeLabel(tk.key)"
        @input="ed.setSize(tk.key, Number(($event.target as HTMLInputElement).value))"
      />
    </div>
    <p class="szi-hint">{{ t('theme.editor.sizeHint') }}</p>
  </div>
</template>

<script setup lang="ts">
import { ArrowUturnLeftIcon } from '@heroicons/vue/24/outline'
import { sizeDisplay, sizeGroup, sizeToken } from '@/theme/sizeTokens'
import { useThemeEditor } from './useThemeEditor'

const ed = useThemeEditor()
const { t } = ed
const group = computed(() => (ed.isSizeSel.value ? sizeGroup(ed.sel.value.slice(5)) : undefined))
const display = (key: string, v: number) => {
  const tk = sizeToken(key)
  return tk ? sizeDisplay(tk, v) : String(v)
}
const onNumber = (key: string, e: Event) => {
  const input = e.target as HTMLInputElement
  const v = Number(input.value)
  if (Number.isFinite(v)) ed.setSize(key, v)
  else input.value = String(ed.sizeValue(key))
}
</script>

<style scoped lang="scss">
.szi {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.szi-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.szi-lbl {
  flex: 1;
  min-width: 0;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--m-ink2);
}
.szi-tag {
  display: inline-flex;
  align-items: center;
  height: 18px;
  padding: 0 7px;
  border-radius: 999px;
  background: var(--m-s2);
  color: var(--m-ink2);
  font-size: 0.625rem;
  font-weight: 800;
  white-space: nowrap;
  &.is-page {
    background: var(--m-aSoft);
    color: var(--m-aOn);
  }
}
.szi-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.szi-row__head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.szi-row__name {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8125rem;
  font-weight: 700;
}
.szi-tagx {
  display: inline-flex;
  align-items: center;
  height: 18px;
  padding: 0 7px;
  border-radius: 999px;
  background: var(--m-aSoft);
  color: var(--m-aOn);
  font-size: 0.625rem;
  font-weight: 800;
}
.szi-dot {
  flex: none;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--m-p);
}
.szi-num {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 32px;
  padding: 0 8px;
  border-radius: 10px;
  background: var(--m-s2);
  input {
    width: 48px;
    border: 0;
    background: transparent;
    font: inherit;
    font-size: 0.8125rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    text-align: right;
    color: var(--m-ink);
    outline: none;
  }
  small {
    font-size: 0.6875rem;
    color: var(--m-ink2);
  }
  &:focus-within {
    box-shadow: var(--app-focus-ring);
  }
}
.szi-reset {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: var(--m-ink2);
  cursor: pointer;
  svg {
    width: 16px;
    height: 16px;
  }
  &:hover {
    background: var(--m-s2);
    color: var(--m-ink);
  }
}
.szi-rng {
  width: 100%;
  height: 24px;
  margin: 0;
  accent-color: var(--m-p);
  cursor: pointer;
}
.szi-hint {
  margin: 0;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--m-s2);
  font-size: 0.75rem;
  line-height: 1.6;
  color: var(--m-ink2);
}
</style>
