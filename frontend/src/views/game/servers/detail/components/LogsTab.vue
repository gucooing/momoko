<!-- 日志：服务端按模块写的 JSONL 日志（适配器给出模块）；从文件末尾向前读，按级别与关键字筛选，可继续加载更早的记录。 -->
<template>
  <div class="lg">
    <p v-if="modules?.fileDisabled" class="lg__notice">{{ t('gameServers.logs.fileDisabled') }}</p>
    <div class="lg__tools">
      <div class="lg__chips" role="tablist" :aria-label="t('gameServers.logs.modules')">
        <button
          v-for="m in modules?.modules || []"
          :key="m.key"
          type="button"
          role="tab"
          class="lg__chip"
          :class="{ 'is-on': module === m.key }"
          :aria-selected="module === m.key"
          @click="pick(m.key)"
        >
          {{ m.key }}
          <span class="lg__chip-n m-num">{{ m.exists ? fmtBytes(m.size) : '—' }}</span>
        </button>
      </div>
      <span class="lg__spacer" />
      <div class="lg__level">
        <AppSelect v-model="level" :options="levelOptions" @update:model-value="reload" />
      </div>
      <label class="lg__search">
        <UIcon name="i-lucide-search" />
        <input v-model="keyword" type="search" :placeholder="t('gameServers.logs.search')" :aria-label="t('gameServers.logs.search')" @keyup.enter="reload" @search="reload" />
      </label>
      <label class="lg__follow">
        <AppSwitch v-model="follow" :aria-label="t('gameServers.logs.follow')" />
        <span>{{ t('gameServers.logs.follow') }}</span>
      </label>
      <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('gameServers.common.refresh')" :box="40" @click="reload" />
    </div>

    <section class="lg__card">
      <p v-if="current && !current.exists" class="lg__empty">{{ t('gameServers.logs.missing', { file: `${modules?.dir}/${current.file}` }) }}</p>
      <p v-else-if="loaded && !entries.length" class="lg__empty">{{ t('gameServers.logs.empty') }}</p>
      <ul v-else class="lg__list m-mono">
        <li v-for="(e, i) in entries" :key="`${i}-${e.time}`" class="lg__row" :class="`is-${e.level}`">
          <span class="lg__time">{{ e.time ? fmtLogTime(e.time) : '' }}</span>
          <span class="lg__lvl">{{ e.level || '·' }}</span>
          <span class="lg__msg">
            {{ e.message || e.raw }}
            <span v-if="attrText(e)" class="lg__attrs">{{ attrText(e) }}</span>
          </span>
        </li>
      </ul>
      <div class="lg__foot">
        <UButton v-if="nextBefore > 0" size="sm" color="neutral" variant="soft" :loading="loading" @click="more">{{ t('gameServers.logs.more') }}</UButton>
        <span v-else-if="entries.length" class="lg__end">{{ t('gameServers.logs.end') }}</span>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { listGameServerLogs, tailGameServerLog } from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import type { GameAdapter, GameServerInfo, GameServerLogEntry, ListGameServerLogsResponse } from '@/types/v1/game_server'
import { fmtBytes } from '../../shared'

const props = defineProps<{ server: GameServerInfo; adapter?: GameAdapter | null }>()
const { t } = useI18n()
const fb = useFeedback()

const id = computed(() => props.server.brief?.id || '')
const modules = ref<ListGameServerLogsResponse | null>(null)
const module = ref('')
const level = ref('')
const keyword = ref('')
const follow = ref(false)
const entries = ref<GameServerLogEntry[]>([])
const nextBefore = ref(0)
const loading = ref(false)
const loaded = ref(false)
const current = computed(() => modules.value?.modules.find((m) => m.key === module.value))

const levelOptions = computed(() =>
  ['', 'debug', 'info', 'warn', 'error'].map((v) => ({ value: v, label: t(`gameServers.logs.levels.${v || 'all'}`) })),
)

const fmtLogTime = (v: unknown) => dayjs(v as string).format('MM-DD HH:mm:ss.SSS')
const attrText = (e: GameServerLogEntry) => {
  const a = e.attrs as Record<string, unknown> | undefined
  if (!a) return ''
  return Object.entries(a)
    .map(([k, v]) => `${k}=${typeof v === 'string' ? v : JSON.stringify(v)}`)
    .join(' ')
}

const loadModules = async () => {
  try {
    modules.value = (await listGameServerLogs(id.value)).data
    if (!module.value) module.value = modules.value.modules.find((m) => m.exists)?.key || modules.value.modules[0]?.key || ''
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  }
}

const fetchPage = async (before: number) => {
  if (!module.value) return
  loading.value = true
  try {
    const { data } = await tailGameServerLog(id.value, module.value, { limit: 200, level: level.value, q: keyword.value.trim(), before })
    entries.value = before ? [...entries.value, ...(data.items ?? [])] : data.items ?? []
    nextBefore.value = data.nextBefore
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    loading.value = false
    loaded.value = true
  }
}
const reload = () => fetchPage(0)
const more = () => fetchPage(nextBefore.value)
const pick = (key: string) => {
  module.value = key
  reload()
}

let timer: ReturnType<typeof setInterval> | undefined
const startFollow = () => {
  clearInterval(timer)
  if (follow.value) timer = setInterval(() => document.visibilityState === 'visible' && reload(), 5000)
}
watch(follow, startFollow)

onMounted(async () => {
  await loadModules()
  reload()
})
onActivated(() => {
  loadModules()
  startFollow()
})
onDeactivated(() => clearInterval(timer))
onBeforeUnmount(() => clearInterval(timer))
</script>

<style scoped lang="scss">
.lg {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.lg__notice {
  margin: 0;
  padding: 10px 14px;
  border-radius: var(--app-radius);
  background: var(--m-warnSoft);
  color: var(--m-warn);
  font-size: 0.8125rem;
  font-weight: 700;
}
.lg__tools {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.lg__spacer {
  flex: 1;
}
.lg__chips {
  display: flex;
  gap: 8px;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.lg__chips::-webkit-scrollbar {
  display: none;
}
.lg__chip {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
}
.lg__chip.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.lg__chip-n {
  font-size: 0.75rem;
  opacity: 0.7;
}
.lg__level {
  width: 130px;
}
.lg__level :deep(.app-sel__trigger) {
  height: 40px;
  border-radius: 999px;
}
.lg__search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 240px;
  min-width: 160px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
}
.lg__search input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
}
.lg__follow {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.lg__card {
  min-width: 0;
  padding: 8px 16px;
  border-radius: var(--app-radius-lg);
  border: var(--cmp-card-border-width, 1px) solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.lg__empty {
  margin: 0;
  padding: 24px 0;
  text-align: center;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.lg__list {
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: 0.75rem;
  line-height: 1.6;
}
.lg__row {
  display: grid;
  grid-template-columns: 150px 52px minmax(0, 1fr);
  gap: 10px;
  padding: 5px 0;
  border-top: 1px solid var(--el-border-color-lighter, var(--el-border-color-light));
}
.lg__row:first-child {
  border-top: none;
}
.lg__time {
  color: var(--el-text-color-secondary);
}
.lg__lvl {
  font-weight: 700;
  text-transform: uppercase;
  color: var(--el-text-color-secondary);
}
.lg__row.is-warn .lg__lvl {
  color: var(--m-warn);
}
.lg__row.is-error .lg__lvl {
  color: var(--m-bad);
}
.lg__msg {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--el-text-color-primary);
}
.lg__attrs {
  margin-left: 8px;
  color: var(--el-text-color-secondary);
}
.lg__foot {
  display: flex;
  justify-content: center;
  padding: 10px 0 6px;
}
.lg__end {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
@media (width <= 768px) {
  .lg__row {
    grid-template-columns: 1fr auto;
  }
  .lg__msg {
    grid-column: 1 / -1;
  }
  .lg__search {
    flex: 1 1 100%;
  }
}
</style>
