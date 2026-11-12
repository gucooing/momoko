<!-- 变更记录：服务端更新、资源、配置、备份、密钥、接入方式、进程（按类型筛选，分页加载）。 -->
<template>
  <section class="hl">
    <div class="hl__head">
      <h2>{{ title }}</h2>
      <div class="hl__chips">
        <button
          v-for="k in kindsShown"
          :key="k"
          type="button"
          class="hl__chip"
          :class="{ 'is-on': kind === k }"
          :aria-pressed="kind === k"
          @click="setKind(k)"
        >
          {{ k ? t(`gameServers.history.kind.${k}`) : t('gameServers.list.all') }}
        </button>
      </div>
    </div>
    <p v-if="loaded && !items.length" class="hl__empty">{{ t('gameServers.update.historyEmpty') }}</p>
    <ul v-else class="hl__list">
      <li v-for="h in items" :key="h.id" class="hl__row">
        <span class="hl__at m-num">{{ fmtTime(h.createTime) }}</span>
        <span class="hl__kind">{{ t(`gameServers.history.kind.${h.kind}`, h.kind) }}</span>
        <span class="hl__text">
          {{ historyText(t, te, h) }}
          <small v-if="historyMessage(t, h)">{{ historyMessage(t, h) }}</small>
        </span>
        <span class="hl__by">{{ h.operatorName }}</span>
        <StatusPill :variant="historyVariant(h.result)" :label="t(`gameServers.history.result.${h.result}`, h.result)" />
      </li>
    </ul>
    <UButton v-if="items.length < total" size="sm" color="neutral" variant="soft" class="hl__more" :loading="loading" @click="load(false)">
      {{ t('gameServers.logs.more') }}
    </UButton>
  </section>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { listGameServerHistory } from '@/api/gameServer'
import type { GameServerHistoryEntry } from '@/types/v1/game_server'
import { fmtTime, historyMessage, historyText, historyVariant } from '../../shared'

const props = defineProps<{ serverId: string; title: string; kinds: string[] }>()
const { t, te } = useI18n()

const kind = ref('')
const items = ref<GameServerHistoryEntry[]>([])
const total = ref(0)
const page = ref(0)
const loading = ref(false)
const loaded = ref(false)
const kindsShown = computed(() => ['', ...props.kinds])

const load = async (reset: boolean) => {
  loading.value = true
  try {
    const next = reset ? 1 : page.value + 1
    const { data } = await listGameServerHistory(props.serverId, { kind: kind.value, page: next, pageSize: 20 })
    items.value = reset ? data.items ?? [] : [...items.value, ...(data.items ?? [])]
    total.value = data.total
    page.value = next
  } finally {
    loading.value = false
    loaded.value = true
  }
}
const setKind = (k: string) => {
  kind.value = k
  load(true)
}
defineExpose({ reload: () => load(true) })
onMounted(() => load(true))
</script>

<style scoped lang="scss">
.hl {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 18px 20px;
  border-radius: var(--app-radius-lg);
  border: var(--cmp-card-border-width, 1px) solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.hl__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.hl__head h2 {
  flex: 1 1 auto;
  margin: 0;
  font-size: 1.0625rem;
  font-weight: 800;
}
.hl__chips {
  display: flex;
  gap: 6px;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.hl__chips::-webkit-scrollbar {
  display: none;
}
.hl__chip {
  flex: none;
  min-height: 34px;
  padding: 0 12px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.hl__chip.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.hl__empty {
  margin: 0;
  padding: 12px 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.hl__list {
  margin: 0;
  padding: 0;
  list-style: none;
}
.hl__row {
  display: grid;
  grid-template-columns: 110px 84px minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 12px;
  min-height: 52px;
  padding: 6px 0;
  border-top: 1px solid var(--el-border-color-light);
  font-size: 0.875rem;
}
.hl__at,
.hl__by {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.hl__kind {
  justify-self: start;
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  font-size: 0.75rem;
  font-weight: 700;
}
.hl__text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-wrap: anywhere;
}
.hl__text small {
  font-size: 0.75rem;
  color: var(--m-bad);
}
.hl__more {
  align-self: center;
}
@media (width <= 768px) {
  .hl__row {
    grid-template-columns: auto minmax(0, 1fr) auto;
  }
  .hl__at {
    grid-column: 1 / -1;
  }
  .hl__by {
    display: none;
  }
}
</style>
