<!-- 个人中心通用行：贴纸图标 + 标题/说明 + 右侧插槽。clickable 时整行可点（≥ 44px）。 -->
<template>
  <component
    :is="clickable ? 'button' : 'div'"
    :type="clickable ? 'button' : undefined"
    class="pf-row"
    :class="{ 'is-clickable': clickable, 'is-compact': compact }"
    @click="clickable && emit('click')"
  >
    <span
      v-if="icon"
      class="m-sticker pf-row__sk"
      :class="`m-sticker--${tone}`"
      :style="tilt ? { transform: `rotate(${tilt}deg)` } : undefined"
      aria-hidden="true"
    >
      <UIcon :name="icon" />
    </span>
    <span class="pf-row__text">
      <span class="pf-row__title">
        <slot name="title">{{ title }}</slot>
      </span>
      <span v-if="sub || $slots.sub" class="pf-row__sub" :class="{ 'm-num': numSub }">
        <slot name="sub">{{ sub }}</slot>
      </span>
    </span>
    <span v-if="$slots.default" class="pf-row__end" @click.stop>
      <slot />
    </span>
    <UIcon v-if="chevron" name="i-lucide-chevron-right" class="pf-row__chev" aria-hidden="true" />
  </component>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    icon?: string
    tone?: 'p' | 'a' | 'a2' | 'ok' | 'bad' | 'warn' | 'solid-p' | 'solid-a' | 'solid-a2' | 'solid-ink'
    tilt?: number
    title?: string
    sub?: string
    numSub?: boolean
    clickable?: boolean
    chevron?: boolean
    compact?: boolean
  }>(),
  { tone: 'p', tilt: 0 },
)
const emit = defineEmits<{ click: [] }>()
</script>

<style scoped lang="scss">
.pf-row {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  min-height: 68px;
  padding: 12px 0;
  border: 0;
  border-top: 1px solid var(--el-border-color-light);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
}
.pf-row:first-child {
  border-top: 0;
}
.pf-row.is-compact {
  min-height: 58px;
  padding: 8px 0;
}
.pf-row.is-clickable {
  cursor: pointer;
  border-radius: 0;
  transition: background 0.15s;
}
.pf-row.is-clickable:hover {
  background: color-mix(in srgb, var(--el-fill-color-light) 60%, transparent);
}
.pf-row.is-clickable:focus-visible {
  outline: 2px solid var(--m-p);
  outline-offset: -2px;
}
.pf-row__sk {
  width: 42px;
  height: 42px;
  border-radius: 14px;
  font-size: 20px;
}
.pf-row__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.pf-row__title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
  font-size: 0.9375rem;
  font-weight: 700;
  overflow-wrap: anywhere;
}
.pf-row__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.pf-row__end {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.pf-row__chev {
  flex: none;
  width: 18px;
  height: 18px;
  color: var(--el-text-color-secondary);
}
</style>
