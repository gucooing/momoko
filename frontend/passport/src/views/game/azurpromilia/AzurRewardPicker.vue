<!-- 奖励列表：选种类 → 按名称或 ID 搜索条目 → 填数量；只显示名称。 -->
<template>
  <div class="azrp">
    <ul v-if="modelValue.length" class="azrp__list">
      <li v-for="(r, i) in modelValue" :key="`${r.type}:${r.id}:${i}`" class="azrp__item">
        <span class="pp-hint">{{ rewardTypeName(r.type) }}</span>
        <span class="azrp__name">{{ itemName(r.type, r.id) }}</span>
        <input class="azrp__num pp-num" type="number" min="1" :value="r.num" :aria-label="t('game.azur.form.num')" @input="setNum(i, ($event.target as HTMLInputElement).value)" />
        <PButton size="sm" variant="ghost" @click="remove(i)">{{ t('game.azur.form.remove') }}</PButton>
      </li>
    </ul>
    <p v-else class="pp-hint">{{ t('game.azur.form.noRewards') }}</p>
    <div class="azrp__add">
      <select v-model.number="type" :aria-label="t('game.azur.form.kind')">
        <option v-for="k in kinds" :key="k.type" :value="k.type">{{ rewardTypeName(k.type) }}</option>
      </select>
      <div class="azrp__search">
        <input v-model="q" type="search" :placeholder="t('game.azur.form.searchItem')" @input="search" @focus="search" @blur="open = false" />
        <ul v-if="open && hits.length" class="azrp__hits">
          <li v-for="h in hits" :key="h.id">
            <button type="button" @mousedown.prevent="add(h.id)">
              <span>{{ h.name }}</span>
              <span class="pp-hint pp-num">{{ h.id }}</span>
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
import PButton from '@passport/components/PButton.vue'
import { REWARD_KINDS, typeOfKind, type AzurReward } from '@/games/azurpromilia/names'
import { itemName, rewardTypeName, searchItems } from './names'

const props = defineProps<{ modelValue: AzurReward[]; types?: number[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: AzurReward[]] }>()
const { t } = useI18n()

const kinds = computed(() => (props.types ? REWARD_KINDS.filter((k) => props.types!.includes(k.type)) : REWARD_KINDS))
const type = ref(kinds.value.some((k) => k.kind === 'common_item') ? typeOfKind('common_item') : (kinds.value[0]?.type ?? 0))
const q = ref('')
const hits = ref<{ id: number; name: string }[]>([])
const open = ref(false)

let seq = 0
const search = async () => {
  const my = ++seq
  open.value = true
  const res = await searchItems(type.value, q.value)
  if (my === seq) hits.value = res
}
watch(type, () => {
  hits.value = []
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
  const cur = props.modelValue[i]
  if (!cur) return
  const list = [...props.modelValue]
  list[i] = { type: cur.type, id: cur.id, num: Math.max(1, Math.floor(Number(v) || 1)) }
  emit('update:modelValue', list)
}

const remove = (i: number) => {
  const list = [...props.modelValue]
  list.splice(i, 1)
  emit('update:modelValue', list)
}
</script>

<style scoped>
.azrp {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.azrp__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.azrp__item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 10px;
  background: var(--m-s2);
}
.azrp__name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.azrp__num {
  width: 88px;
  height: 30px;
  padding: 0 8px;
  border: 1px solid var(--m-line);
  border-radius: 8px;
  background: var(--m-s);
  color: var(--m-ink);
}
.azrp__add {
  display: flex;
  gap: 8px;
}
.azrp__add select,
.azrp__search input {
  height: 34px;
  padding: 0 10px;
  border: 1px solid var(--m-line);
  border-radius: 10px;
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
}
.azrp__search {
  position: relative;
  flex: 1;
}
.azrp__search input {
  width: 100%;
}
.azrp__hits {
  position: absolute;
  z-index: 20;
  left: 0;
  right: 0;
  top: 38px;
  max-height: 240px;
  overflow: auto;
  margin: 0;
  padding: 4px;
  list-style: none;
  border: 1px solid var(--m-line);
  border-radius: 10px;
  background: var(--m-s);
}
.azrp__hits button {
  width: 100%;
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 8px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--m-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.azrp__hits button:hover {
  background: var(--m-s2);
}
</style>
