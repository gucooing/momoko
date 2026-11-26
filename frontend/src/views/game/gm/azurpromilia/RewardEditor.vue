<!-- 奖励列表编辑：选种类 → 按名称或 ID 搜索条目 → 填数量；列表里只显示名称，不让人写 ID 或 JSON。 -->
<template>
  <div class="azr">
    <ul v-if="modelValue.length" class="azr__list">
      <li v-for="(r, i) in modelValue" :key="`${r.type}:${r.id}:${i}`" class="azr__item">
        <span class="azr__kind">{{ rewardTypeName(r.type) }}</span>
        <span class="azr__name">{{ itemName(r.type, r.id) }}</span>
        <input
          class="azr__num m-num"
          type="number"
          min="1"
          :max="max"
          :value="r.num"
          :aria-label="t('azurGm.form.num')"
          @input="setNum(i, ($event.target as HTMLInputElement).value)"
        />
        <AppIconButton icon="HOutline:XMarkIcon" :label="t('azurGm.form.remove')" :box="28" @click="remove(i)" />
      </li>
    </ul>
    <p v-else class="azr__none">{{ t('azurGm.form.noRewards') }}</p>
    <div class="azr__add">
      <AppSelect v-model="type" :options="kindOptions" searchable class="azr__type" />
      <div class="azr__search">
        <input v-model="q" type="search" :placeholder="t('azurGm.form.searchItem')" @input="search" @focus="search" />
        <ul v-if="hits.length && open" class="azr__hits">
          <li v-for="h in hits" :key="h.id">
            <button type="button" @mousedown.prevent="add(h.id)">
              <span>{{ h.name }}</span>
              <span class="azr__hid m-num">{{ h.id }}</span>
            </button>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import type { Reward } from '@/types/v1/azurpromilia/types'
import { REWARD_KINDS } from '@/games/azurpromilia/names'
import { itemName, rewardTypeName, searchItems } from './shared'

const props = withDefaults(defineProps<{ modelValue: Reward[]; types?: number[]; max?: number }>(), { types: undefined, max: 2147483647 })
const emit = defineEmits<{ 'update:modelValue': [value: Reward[]] }>()
const { t } = useI18n()

const kinds = computed(() => (props.types ? REWARD_KINDS.filter((k) => props.types!.includes(k.type)) : REWARD_KINDS))
const kindOptions = computed(() => kinds.value.map((k) => ({ label: rewardTypeName(k.type), value: k.type })))
const type = ref(kinds.value.find((k) => k.kind === 'common_item')?.type ?? kinds.value[0]?.type ?? 0)
const q = ref('')
const hits = ref<{ id: number; name: string }[]>([])
const open = ref(false)

const search = useDebounceFn(async () => {
  open.value = true
  hits.value = await searchItems(type.value, q.value)
}, 200)

watch(type, () => {
  hits.value = []
  if (open.value) void search()
})

const add = (id: number) => {
  const list = [...props.modelValue]
  const at = list.findIndex((r) => r.type === type.value && r.id === id)
  const cur = list[at]
  if (cur) list[at] = { type: cur.type, id: cur.id, num: cur.num + 1 }
  else list.push({ type: type.value, id, num: 1 })
  emit('update:modelValue', list)
  open.value = false
  q.value = ''
}

const setNum = (i: number, v: string) => {
  const n = Math.max(1, Math.min(props.max, Math.floor(Number(v) || 1)))
  const list = [...props.modelValue]
  const cur = list[i]
  if (!cur) return
  list[i] = { type: cur.type, id: cur.id, num: n }
  emit('update:modelValue', list)
}

const remove = (i: number) => {
  const list = [...props.modelValue]
  list.splice(i, 1)
  emit('update:modelValue', list)
}
</script>

<style scoped lang="scss">
.azr {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.azr__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.azr__item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
}
.azr__kind {
  flex: none;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.azr__name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.azr__num {
  width: 96px;
  height: 28px;
  padding: 0 8px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-xs);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
}
.azr__none {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.azr__add {
  display: flex;
  gap: 8px;
  align-items: flex-start;
}
.azr__type {
  width: 160px;
  flex: none;
}
.azr__search {
  position: relative;
  flex: 1;
  input {
    width: 100%;
    height: 32px;
    padding: 0 10px;
    border: 1px solid var(--el-border-color-light);
    border-radius: var(--app-radius-sm);
    background: var(--el-bg-color);
    color: var(--el-text-color-primary);
  }
}
.azr__hits {
  position: absolute;
  z-index: 20;
  left: 0;
  right: 0;
  top: 36px;
  max-height: 260px;
  overflow: auto;
  margin: 0;
  padding: 4px;
  list-style: none;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-sm);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-brand);
  button {
    width: 100%;
    display: flex;
    justify-content: space-between;
    gap: 8px;
    padding: 6px 8px;
    border: 0;
    border-radius: var(--app-radius-xs);
    background: transparent;
    color: var(--el-text-color-primary);
    font: inherit;
    text-align: left;
    cursor: pointer;
    &:hover {
      background: var(--el-fill-color-light);
    }
  }
}
.azr__hid {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
