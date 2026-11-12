<!-- 游戏服务器选择：服务器可能有几百个，所以只用「搜索 + 最近查看」的下拉（结果最多 20 条），
     不把全部服务器铺开。搜索函数由使用方传入（GM 运营与用户通行证后台各用自己有权限的接口）。 -->
<template>
  <AppDropdown align="start" :width="360" panel-class="gsp-panel">
    <template #trigger="{ open }">
      <slot name="trigger" :open="open" :current="current">
        <button type="button" class="gsp-trigger" :aria-expanded="open">
          <span class="gsp-trigger__text">{{ current?.name || placeholder || t('game.picker.placeholder') }}</span>
          <component :is="menuStore.iconComponents['HOutline:ChevronUpDownIcon']" class="gsp-trigger__icon" />
        </button>
      </slot>
    </template>
    <template #default="{ close }">
      <div class="gsp">
        <label class="gsp__search">
          <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" class="gsp__search-icon" />
          <input
            v-model="q"
            type="search"
            :placeholder="t('game.picker.search')"
            :aria-label="t('game.picker.search')"
            @keydown.enter.prevent="pickFirst(close)"
          />
        </label>
        <button v-if="allowAll" type="button" class="gsp__row" :class="{ 'is-on': !modelValue }" @click="pick(undefined, close)">
          <span class="gsp__name">{{ t('game.picker.all') }}</span>
        </button>
        <template v-if="!q && recent.length">
          <div class="gsp__label">{{ t('game.picker.recent') }}</div>
          <button
            v-for="s in recent"
            :key="`r-${s.id}`"
            type="button"
            class="gsp__row"
            :class="{ 'is-on': s.id === modelValue }"
            @click="pick(s, close)"
          >
            <GameServerLine :server="s" />
          </button>
        </template>
        <div class="gsp__label">{{ q ? t('game.picker.results') : t('game.picker.servers') }}</div>
        <div v-if="loading && !items.length" class="gsp__hint">{{ t('game.picker.loading') }}</div>
        <div v-else-if="!items.length" class="gsp__hint">{{ t('game.picker.empty') }}</div>
        <button
          v-for="s in items"
          :key="s.id"
          type="button"
          class="gsp__row"
          :class="{ 'is-on': s.id === modelValue }"
          @click="pick(s, close)"
        >
          <GameServerLine :server="s" />
        </button>
        <div class="gsp__foot">{{ t('game.picker.limit', { total }) }}</div>
      </div>
    </template>
  </AppDropdown>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import type { GameServerBrief, SearchGameServersResponse } from '@/types/v1/game_common'
import GameServerLine from './GameServerLine.vue'

const props = defineProps<{
  modelValue?: string
  /** 当前选中服务器的摘要（触发按钮显示用） */
  selected?: GameServerBrief
  /** 搜索函数：q 为空时应同时返回最近查看 */
  search: (q: string) => Promise<SearchGameServersResponse>
  /** 是否有「所有服务器」选项 */
  allowAll?: boolean
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string | undefined]
  select: [server: GameServerBrief | undefined]
}>()

const { t } = useI18n()
const menuStore = useMenuStore()

const q = ref('')
const items = ref<GameServerBrief[]>([])
const recent = ref<GameServerBrief[]>([])
const total = ref(0)
const loading = ref(false)
let seq = 0

const current = computed(() => props.selected)

const load = async () => {
  const id = ++seq
  loading.value = true
  try {
    const res = await props.search(q.value.trim())
    if (id !== seq) return
    items.value = res.items || []
    recent.value = res.recent || []
    total.value = res.total || 0
  } catch {
    if (id === seq) items.value = []
  } finally {
    if (id === seq) loading.value = false
  }
}

const debouncedLoad = useDebounceFn(load, 250)
watch(q, () => void debouncedLoad())
void load()

const pick = (s: GameServerBrief | undefined, close: () => void) => {
  emit('update:modelValue', s?.id)
  emit('select', s)
  close()
}

const pickFirst = (close: () => void) => {
  const first = items.value[0]
  if (first) pick(first, close)
}
</script>

<style scoped lang="scss">
.gsp-trigger {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}
.gsp-trigger__icon {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.gsp {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px;
  max-height: min(70vh, 520px);
  overflow-y: auto;
}
.gsp__search {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 12px;
  margin-bottom: 6px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.gsp__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
}
.gsp__search-icon {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.gsp__label {
  padding: 8px 8px 4px;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gsp__row {
  display: flex;
  align-items: center;
  width: 100%;
  min-height: 44px;
  padding: 6px 8px;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gsp__row:hover,
.gsp__row.is-on {
  background: var(--m-pSoft);
}
.gsp__name {
  font-weight: 700;
}
.gsp__hint,
.gsp__foot {
  padding: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
