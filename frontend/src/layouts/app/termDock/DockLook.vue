<!-- 悬浮终端的外观设置（标题栏齿轮弹出）：背景不透明度、透明时模糊背后的页面、字号。
     改了马上生效，记在本机；字号只管悬浮终端，不影响其他终端页。 -->
<template>
  <div class="tdl">
    <div class="tdl__row">
      <span class="tdl__k">{{ t('termDock.opacity') }}</span>
      <b class="tdl__v ag-num">{{ opacityPct }}%</b>
    </div>
    <input
      class="tdl__range"
      type="range"
      :min="Math.round(LOOK_OPACITY_MIN * 100)"
      max="100"
      step="5"
      :value="opacityPct"
      :aria-label="t('termDock.opacity')"
      @input="setLook({ opacity: Number(($event.target as HTMLInputElement).value) / 100 })"
    />

    <label class="tdl__row tdl__check" :class="{ 'is-off': dock.look.opacity >= 1 }" :title="t('termDock.blurHint')">
      <input class="ag-cb" type="checkbox" :checked="dock.look.blur" @change="setLook({ blur: ($event.target as HTMLInputElement).checked })" />
      <span class="tdl__k">{{ t('termDock.blur') }}</span>
    </label>

    <div class="tdl__row">
      <span class="tdl__k">{{ t('termDock.fontSize') }}</span>
      <span class="tdl__step">
        <button
          type="button"
          class="ag-ib ag-ib--sm"
          :disabled="dock.look.fontSize <= FONT_SIZE_MIN"
          :aria-label="t('termDock.smaller')"
          :title="t('termDock.smaller')"
          @click="setLook({ fontSize: dock.look.fontSize - 1 })"
        >
          <UIcon name="i-lucide-minus" />
        </button>
        <b class="tdl__v ag-num">{{ dock.look.fontSize }}</b>
        <button
          type="button"
          class="ag-ib ag-ib--sm"
          :disabled="dock.look.fontSize >= FONT_SIZE_MAX"
          :aria-label="t('termDock.bigger')"
          :title="t('termDock.bigger')"
          @click="setLook({ fontSize: dock.look.fontSize + 1 })"
        >
          <UIcon name="i-lucide-plus" />
        </button>
      </span>
    </div>

    <div class="ag-mi-sep" />
    <button type="button" class="ag-mi" @click="resetLook">
      <UIcon name="i-lucide-rotate-ccw" />
      <span class="ag-ellipsis">{{ t('termDock.resetLook') }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { FONT_SIZE_MAX, FONT_SIZE_MIN } from '@/components/terminal/useTerminalX'
import { LOOK_OPACITY_MIN, dock, resetLook, setLook } from './state'

defineOptions({ name: 'DockLook' })

const { t } = useI18n()

const opacityPct = computed(() => Math.round(dock.look.opacity * 100))
</script>

<style scoped lang="scss">
.tdl {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 4px 2px 0;
}
.tdl__row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 30px;
  padding: 0 8px;
}
.tdl__k {
  flex: 1;
  min-width: 0;
  font-size: 12.5px;
  color: var(--m-ink);
}
.tdl__v {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--m-ink);
}
.tdl__range {
  width: calc(100% - 16px);
  margin: 0 8px 4px;
  accent-color: var(--m-p);
  cursor: pointer;
}
.tdl__check {
  cursor: pointer;
}
// 不透明时模糊没有效果，看着淡一点
.tdl__check.is-off .tdl__k {
  color: var(--m-ink2);
}
.tdl__step {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}
.tdl__step .tdl__v {
  min-width: 22px;
  text-align: center;
}
</style>
