<!-- API Key 节点权限选择：按功能分组的卡片（组标题可整组勾选），组内每项是可点的勾选行（v-model 为权限标识数组）。
     远程主控只能使用这里勾选、且 API Key 所属用户本身拥有的功能。 -->
<template>
  <div class="perm-picker">
    <div class="perm-picker__bar">
      <span class="perm-picker__count m-num">{{ t('node.permission.selected', { count: modelValue.length }) }}</span>
      <button type="button" class="perm-picker__link" @click="selectAll">{{ t('node.permission.selectAll') }}</button>
      <button type="button" class="perm-picker__link" @click="clear">{{ t('node.permission.clear') }}</button>
    </div>
    <div class="perm-picker__groups">
      <fieldset
        v-for="group in groups"
        :key="group.key"
        class="perm-group"
        :class="`is-${group.state}`"
      >
        <legend class="perm-group__legend">
          <button type="button" class="perm-group__toggle" :aria-pressed="group.state === 'on'" @click="toggleGroup(group.key)">
            <span class="perm-group__box">
              <UIcon v-if="group.state !== 'off'" :name="group.state === 'on' ? 'i-lucide-check' : 'i-lucide-minus'" />
            </span>
            {{ t(group.labelKey) }}
            <span class="perm-group__n m-num">{{ group.granted }}/{{ group.total }}</span>
          </button>
        </legend>
        <label
          v-for="item in group.items"
          :key="item.code"
          class="perm-item"
          :class="{ 'is-checked': selected.has(item.code) }"
        >
          <input type="checkbox" :checked="selected.has(item.code)" @change="toggle(item.code)" />
          <span class="perm-item__label">{{ t(item.labelKey) }}</span>
          <code class="perm-item__code">{{ item.code }}</code>
        </label>
      </fieldset>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NODE_PERMISSION_GROUPS, NODE_PERMISSION_ITEMS } from '@/config/nodePermission'

const props = defineProps<{ modelValue: string[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()
const { t } = useI18n()

const selected = computed(() => new Set(props.modelValue))

const groups = computed(() =>
  NODE_PERMISSION_GROUPS.map((g) => {
    const granted = g.items.filter((i) => selected.value.has(i.code)).length
    const state = granted === 0 ? 'off' : granted === g.items.length ? 'on' : 'part'
    return { ...g, granted, total: g.items.length, state }
  }),
)

// 按目录顺序输出，便于与后端规范化结果一致
const emitOrdered = (set: Set<string>) => {
  emit(
    'update:modelValue',
    NODE_PERMISSION_ITEMS.filter((item) => set.has(item.code)).map((item) => item.code),
  )
}

const toggle = (code: string) => {
  const next = new Set(selected.value)
  if (next.has(code)) next.delete(code)
  else next.add(code)
  emitOrdered(next)
}
const toggleGroup = (key: string) => {
  const group = NODE_PERMISSION_GROUPS.find((g) => g.key === key)
  if (!group) return
  const next = new Set(selected.value)
  const allOn = group.items.every((i) => next.has(i.code))
  for (const item of group.items) {
    if (allOn) next.delete(item.code)
    else next.add(item.code)
  }
  emitOrdered(next)
}
const selectAll = () => emitOrdered(new Set(NODE_PERMISSION_ITEMS.map((item) => item.code)))
const clear = () => emitOrdered(new Set())
</script>

<style scoped lang="scss">
.perm-picker {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.perm-picker__bar {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.8125rem;
}
.perm-picker__count {
  flex: 1;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.perm-picker__link {
  min-height: 36px;
  padding: 0 10px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--m-pInk);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.perm-picker__link:hover {
  background: var(--m-pSoft);
}
.perm-picker__groups {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}
@media (width <= 640px) {
  .perm-picker__groups {
    grid-template-columns: 1fr;
  }
}
.perm-group {
  min-width: 0;
  margin: 0;
  padding: 4px 12px 10px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
  transition: border-color 0.15s, background 0.15s;
}
.perm-group.is-on {
  border-color: color-mix(in srgb, var(--m-p) 45%, transparent);
  background: color-mix(in srgb, var(--m-pSoft) 45%, var(--el-bg-color));
}
.perm-group__legend {
  padding: 0 4px;
  margin-left: -4px;
}
.perm-group__toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding: 0 6px;
  border: none;
  border-radius: 10px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font-size: 0.875rem;
  font-weight: 800;
  cursor: pointer;
}
.perm-group__box {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 6px;
  border: 1.5px solid var(--el-border-color);
  color: var(--m-pOn);
}
.perm-group.is-on .perm-group__box,
.perm-group.is-part .perm-group__box {
  background: var(--m-p);
  border-color: var(--m-p);
}
.perm-group__box :deep(svg),
.perm-group__box :deep(span) {
  width: 13px;
  height: 13px;
}
.perm-group__n {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.perm-item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  padding: 0 4px;
  border-radius: 10px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.perm-item:hover {
  background: var(--el-fill-color-light);
}
.perm-item input {
  width: 18px;
  height: 18px;
  flex: none;
  accent-color: var(--m-p);
}
.perm-item.is-checked {
  color: var(--el-text-color-primary);
  font-weight: 600;
}
.perm-item__label {
  flex: 1;
  min-width: 0;
}
.perm-item__code {
  font-family: var(--app-font-mono);
  font-size: 0.6875rem;
  color: var(--el-text-color-placeholder);
  white-space: nowrap;
}
</style>
