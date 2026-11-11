<!-- 编辑器右栏：选中项（取色器 / 欢迎横幅 / 组件尺寸）、当前页的单独设置（可逐条改回跟随全局）、对比度检查（低于 4.5:1 提示，可一键修正）。 -->
<template>
  <div class="ein">
    <BannerEditor v-if="ed.sel.value === 'hero'" />
    <SizeInspector v-else-if="ed.isSizeSel.value" />
    <div v-else class="ein-sec">
      <div class="ein-head">
        <span class="ein-lbl">{{ ed.selName.value }}</span>
        <span class="ein-tag" :class="{ 'is-page': ed.scope.value === 'page' && !isTermKey }">{{ ed.scopeTag.value }}</span>
      </div>
      <ColorPicker :model-value="ed.selHex.value" :suggestions="ed.suggestions.value" @update:model-value="ed.setToken(ed.sel.value, $event)" />
      <button v-if="ed.selEdited.value" type="button" class="ein-btn" @click="ed.revertToken(ed.sel.value)">
        <ArrowUturnLeftIcon aria-hidden="true" />{{ t('theme.editor.revert', { hex: ed.selBase.value }) }}
      </button>
    </div>

    <div class="ein-sec ein-sec--line">
      <div class="ein-head">
        <span class="ein-lbl">{{ t('theme.editor.pageOverrides', { page: ed.pageName.value }) }}</span>
        <b class="m-num ein-count">{{ ed.pageOvCount.value }}</b>
      </div>
      <template v-if="ed.pageOvItems.value.length">
        <div v-for="it in ed.pageOvItems.value" :key="it.id" class="ein-ov">
          <span class="ein-ov__sw" :style="{ background: it.swatch }" />
          <span class="ein-ov__text"><b>{{ it.title }}</b><small>{{ it.value }}</small></span>
          <AppIconButton icon="HOutline:ArrowUturnLeftIcon" :label="t('theme.editor.followGlobal')" :size="16" :box="32" @click="it.drop()" />
        </div>
      </template>
      <p v-else class="ein-empty">{{ t('theme.editor.noPageOv') }}</p>
    </div>

    <div class="ein-sec ein-sec--line">
      <span class="ein-lbl">{{ t('theme.editor.checks', { page: ed.pageName.value }) }}</span>
      <div v-for="c in ed.checks.value" :key="c.key" class="ein-check">
        <span class="ein-check__ic" :class="c.ok ? 'is-ok' : 'is-warn'">
          <component :is="c.ok ? CheckIcon : ExclamationTriangleIcon" aria-hidden="true" />
        </span>
        <span class="ein-check__t">{{ t(`theme.checks.${c.key}`) }}</span>
        <b class="m-num">{{ formatRatio(c.ratio) }}</b>
      </div>
      <button v-if="ed.hasWarn.value" type="button" class="ein-btn is-warn" @click="ed.fixAll()">
        <SparklesIcon aria-hidden="true" />{{ t('theme.editor.fixContrast') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowUturnLeftIcon, CheckIcon, ExclamationTriangleIcon, SparklesIcon } from '@heroicons/vue/24/outline'
import ColorPicker from '@/components/theme/ColorPicker.vue'
import { formatRatio } from '@/theme/color'
import BannerEditor from './BannerEditor.vue'
import SizeInspector from './SizeInspector.vue'
import { useThemeEditor } from './useThemeEditor'

const ed = useThemeEditor()
const { t } = ed
const isTermKey = computed(() => ed.sel.value.startsWith('term') || ed.sel.value.startsWith('ansi'))
</script>

<style scoped lang="scss">
.ein {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.ein-sec {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.ein-sec--line {
  gap: 8px;
  padding-top: 14px;
  border-top: 1px solid var(--m-line);
}
.ein-head {
  display: flex;
  align-items: center;
  gap: 8px;
  .ein-lbl {
    flex: 1;
    min-width: 0;
  }
}
.ein-lbl {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--m-ink2);
}
.ein-count {
  font-size: 0.75rem;
  color: var(--m-ink2);
}
.ein-tag {
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
.ein-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 14px;
  border: 0;
  border-radius: var(--app-radius-btn);
  background: var(--m-s2);
  color: var(--m-ink);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
  svg {
    width: 16px;
    height: 16px;
  }
  &.is-warn {
    background: var(--m-warnSoft);
    color: var(--m-warn);
  }
}
.ein-ov {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 4px 4px 4px 10px;
  border-radius: 14px;
  background: var(--m-s2);
}
.ein-ov__sw {
  flex: none;
  width: 20px;
  height: 20px;
  border-radius: 6px;
  border: 1px solid var(--m-line);
}
.ein-ov__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  b {
    font-size: 0.8125rem;
  }
  small {
    font-size: 0.6875rem;
    color: var(--m-ink2);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}
.ein-empty {
  margin: 0;
  padding: 12px;
  border-radius: 14px;
  border: 1.5px dashed var(--m-line);
  font-size: 0.75rem;
  line-height: 1.6;
  color: var(--m-ink2);
}
.ein-check {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 30px;
  font-size: 0.75rem;
  b {
    font-weight: 800;
  }
}
.ein-check__t {
  flex: 1;
}
.ein-check__ic {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  svg {
    width: 12px;
    height: 12px;
    stroke-width: 3;
  }
  &.is-ok {
    background: var(--m-okSoft);
    color: var(--m-ok);
  }
  &.is-warn {
    background: var(--m-warnSoft);
    color: var(--m-warn);
  }
}
</style>
