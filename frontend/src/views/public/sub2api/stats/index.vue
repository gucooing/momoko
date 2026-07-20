<!-- Sub2API 公开统计（P7 去 EP）：
     - 顶栏：LanguageMenu + AppIconButton(主题) + 返回首页（禁止 EP 图标；移动端也保留）
     - MetricStrip 单色 + Heroicons 指标图标（store.card.icon）
     - 区块 stagger 入场（fade+translateY，prefers-reduced-motion 关闭）
     - 详情页：首屏/切换区间用明显 loading 区（非门户首页，不做「—」渐进占位） -->
<template>
  <main class="s2a-stats" :class="{ 'is-dark': isDark }">
    <header class="topbar">
      <div class="topbar-inner">
        <button class="brand" type="button" @click="goHome">
          <span class="brand-dot" />{{ title }}
        </button>
        <div class="topbar-actions">
          <LanguageMenu />
          <AppIconButton
            :icon="isDark ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
            :label="t('login.toggleTheme')"
            :box="36"
            @click="toggleTheme"
          />
          <button class="btn btn--ghost" type="button" @click="goHome">{{ t('sub2api.stats.backHome') }}</button>
        </div>
      </div>
    </header>

    <section class="wrap">
      <div class="head reveal" style="--d: 0ms">
        <div>
          <h1>{{ t('sub2api.stats.title') }}</h1>
          <p>{{ t('sub2api.stats.desc') }}</p>
        </div>
        <div class="head-tools">
          <div class="seg" role="tablist">
            <button
              v-for="opt in rangeOptions"
              :key="opt.value"
              type="button"
              role="tab"
              class="seg__btn"
              :class="{ 'is-active': range === opt.value }"
              :aria-selected="range === opt.value"
              :disabled="store.statsLoading"
              @click="onRange(opt.value)"
            >
              {{ opt.label }}
            </button>
          </div>
        </div>
      </div>

      <!-- 首屏/切换：明确加载态（详情页不要求渐进显示） -->
      <div v-if="store.statsLoading && !stats" class="loading-block reveal" style="--d: 40ms" role="status">
        <span class="spin spin--lg" aria-hidden="true" />
        <p>{{ t('sub2api.stats.loading') }}</p>
      </div>

      <template v-else>
        <div class="body" :class="{ 'is-reloading': store.statsLoading }">
          <div v-if="store.statsLoading" class="reload-bar" role="status">
            <span class="spin" aria-hidden="true" />
            <span>{{ t('sub2api.stats.loading') }}</span>
          </div>

          <MetricStrip :columns="5" class="metrics reveal" style="--d: 60ms">
            <MetricItem
              v-for="card in store.statsMetricCards"
              :key="card.label"
              :icon="card.icon"
              :label="card.label"
              :value="card.value"
              :caption="card.detail"
            />
          </MetricStrip>

          <article class="panel reveal" style="--d: 120ms">
            <div class="panel-head">
              <h3>{{ t('sub2api.common.usageTrend') }}</h3>
              <span>{{ stats?.rangeLabel || '' }}</span>
            </div>
            <VChart class="chart" :option="store.statsTrendOption" :update-options="chartUpdate" autoresize />
          </article>

          <article class="panel reveal" style="--d: 180ms">
            <div class="panel-head">
              <h3>{{ t('sub2api.stats.modelDetails') }}</h3>
              <span>{{ t('sub2api.common.modelCount', { count: models.length }) }}</span>
            </div>
            <div v-if="models.length" class="desk-table-wrap">
              <table class="desk-table">
                <thead>
                  <tr>
                    <th>{{ t('sub2api.common.model') }}</th>
                    <th class="num">{{ t('sub2api.common.requestCount') }}</th>
                    <th class="num">Token</th>
                    <th class="num">{{ t('sub2api.common.generationSpeed') }}</th>
                    <th class="num">{{ t('sub2api.common.successRate') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in models" :key="row.name">
                    <td class="name">{{ row.name || t('sub2api.common.unknownModel') }}</td>
                    <td class="num">{{ store.formatNumber(row.requestCount) }}</td>
                    <td class="num">{{ store.formatToken(row.tokenCount) }}</td>
                    <td class="num">{{ store.formatThroughput(row.avgTps) }} token/s</td>
                    <td class="num rate">{{ store.formatPercent(row.successRate) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-if="models.length" class="mobile-list">
              <article v-for="(row, i) in models" :key="row.name || i" class="rank-row">
                <span class="rank-no" :class="{ top: i < 3 }">{{ i + 1 }}</span>
                <div class="rank-body">
                  <div class="rank-line">
                    <strong>{{ row.name || t('sub2api.common.unknownModel') }}</strong>
                    <span
                      >{{ store.formatThroughput(row.avgTps) }} t/s · {{ store.formatToken(row.tokenCount) }}</span
                    >
                  </div>
                  <div class="rank-bar"><i :style="{ width: modelBarWidth(row.tokenCount) }" /></div>
                </div>
                <span class="rank-rate"
                  ><em>{{ t('sub2api.common.successRate') }}</em>{{ store.formatPercent(row.successRate) }}</span
                >
              </article>
            </div>
            <EmptyState v-if="!models.length" :title="t('sub2api.common.noData')" />
          </article>

      <article v-if="groups.length" class="panel reveal" style="--d: 240ms">
            <div class="panel-head">
              <h3>{{ t('sub2api.stats.groupDetails') }}</h3>
            </div>
            <div class="desk-table-wrap">
              <table class="desk-table">
                <thead>
                  <tr>
                    <th>{{ t('sub2api.common.group') }}</th>
                    <th class="num">{{ t('sub2api.common.requestCount') }}</th>
                    <th class="num">Token</th>
                    <th class="num">{{ t('sub2api.common.successRate') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in groups" :key="row.name">
                    <td class="name">{{ row.name || t('sub2api.common.unknownGroup') }}</td>
                    <td class="num">{{ store.formatNumber(row.requestCount) }}</td>
                    <td class="num">{{ store.formatToken(row.tokenCount) }}</td>
                    <td class="num rate">{{ store.formatPercent(row.successRate) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="mobile-list">
              <article v-for="(row, i) in groups" :key="row.name || i" class="rank-row">
                <span class="rank-no" :class="{ top: i < 3 }">{{ i + 1 }}</span>
                <div class="rank-body">
                  <div class="rank-line">
                    <strong>{{ row.name || t('sub2api.common.unknownGroup') }}</strong>
                    <span
                      >{{ t('sub2api.common.countTimes', { count: store.formatNumber(row.requestCount) }) }} ·
                      {{ store.formatToken(row.tokenCount) }}</span
                    >
                  </div>
                  <div class="rank-bar"><i :style="{ width: groupBarWidth(row.tokenCount) }" /></div>
                </div>
                <span class="rank-rate"
                  ><em>{{ t('sub2api.common.successRate') }}</em>{{ store.formatPercent(row.successRate) }}</span
                >
              </article>
            </div>
          </article>

          <article v-if="userAgents.length" class="panel reveal" style="--d: 300ms">
            <div class="panel-head">
              <h3>{{ t('sub2api.stats.uaDetails') }}</h3>
              <span>{{ t('sub2api.common.countTimes', { count: store.formatNumber(totalUaRequests) }) }}</span>
            </div>
            <div class="desk-table-wrap">
              <table class="desk-table">
                <thead>
                  <tr>
                    <th>{{ t('sub2api.common.ua') }}</th>
                    <th class="num">{{ t('sub2api.common.requestCount') }}</th>
                    <th class="num">Token</th>
                    <th class="num">{{ t('sub2api.common.successRate') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in userAgents" :key="row.name">
                    <td class="name">{{ row.name || t('sub2api.common.unknownUa') }}</td>
                    <td class="num">{{ store.formatNumber(row.requestCount) }}</td>
                    <td class="num">{{ store.formatToken(row.tokenCount) }}</td>
                    <td class="num rate">{{ store.formatPercent(row.successRate) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="mobile-list">
              <article v-for="(row, i) in userAgents" :key="row.name || i" class="rank-row">
                <span class="rank-no" :class="{ top: i < 3 }">{{ i + 1 }}</span>
                <div class="rank-body">
                  <div class="rank-line">
                    <strong>{{ row.name || t('sub2api.common.unknownUa') }}</strong>
                    <span
                      >{{ t('sub2api.common.countTimes', { count: store.formatNumber(row.requestCount) }) }} ·
                      {{ store.formatToken(row.tokenCount) }}</span
                    >
                  </div>
                  <div class="rank-bar"><i :style="{ width: uaBarWidth(row.requestCount) }" /></div>
                </div>
                <span class="rank-rate"
                  ><em>{{ t('sub2api.common.successRate') }}</em>{{ store.formatPercent(row.successRate) }}</span
                >
              </article>
            </div>
          </article>
        </div>
      </template>
    </section>
  </main>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import VChart from '@/components/chart/VChart.vue'
import LanguageMenu from '@/layouts/app/LanguageMenu.vue'
import { useSub2APIStore } from '@/stores/sub2api'
import { useThemeStore } from '@/stores/theme'

defineOptions({ name: 'Sub2APIPublicStats' })

const store = useSub2APIStore()
const themeStore = useThemeStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const isDark = computed(() => themeStore.isDarkTheme)
const toggleTheme = () => themeStore.toggleThemeMode(isDark.value ? 'light' : 'dark')

const chartUpdate = { notMerge: true }
const rangeOptions = computed(() => [
  { label: t('sub2api.common.today'), value: 1 },
  { label: t('sub2api.common.last7Days'), value: 7 },
  { label: t('sub2api.common.last14Days'), value: 14 },
  { label: t('sub2api.common.last30Days'), value: 30 },
])

const parseRange = (value: unknown) => {
  const n = Number(value)
  return [1, 7, 14, 30].includes(n) ? n : 7
}
const range = ref(parseRange(route.query.range))

const stats = computed(() => store.stats)
const models = computed(() =>
  [...(stats.value?.models || [])].sort(
    (a, b) => (Number(b.tokenCount) || 0) - (Number(a.tokenCount) || 0),
  ),
)
const groups = computed(() =>
  [...(stats.value?.groups || [])].sort(
    (a, b) => (Number(b.tokenCount) || 0) - (Number(a.tokenCount) || 0),
  ),
)
const userAgents = computed(() =>
  [...(stats.value?.userAgents || [])].sort(
    (a, b) => (Number(b.requestCount) || 0) - (Number(a.requestCount) || 0),
  ),
)
const totalUaRequests = computed(() =>
  userAgents.value.reduce((sum, item) => sum + (Number(item.requestCount) || 0), 0),
)
const title = computed(() => store.home?.title || 'Sub2API')

const maxModelToken = computed(() =>
  Math.max(...models.value.map((item) => Number(item.tokenCount) || 0), 1),
)
const maxGroupToken = computed(() =>
  Math.max(...groups.value.map((item) => Number(item.tokenCount) || 0), 1),
)
const maxUaCount = computed(() =>
  Math.max(...userAgents.value.map((item) => Number(item.requestCount) || 0), 1),
)
const modelBarWidth = (token: unknown) =>
  `${Math.max(((Number(token) || 0) / maxModelToken.value) * 100, 4)}%`
const groupBarWidth = (token: unknown) =>
  `${Math.max(((Number(token) || 0) / maxGroupToken.value) * 100, 4)}%`
const uaBarWidth = (count: unknown) =>
  `${Math.max(((Number(count) || 0) / maxUaCount.value) * 100, 4)}%`

const onRange = (days: number) => {
  if (range.value === days) return
  range.value = days
  router.replace({ query: { ...route.query, range: String(days) } })
  store.loadStats(days)
}

const goHome = () => router.push('/public/sub2api/home')

onMounted(() => {
  if (!store.home) store.loadPublicHome()
  store.loadStats(range.value)
})
</script>

<style scoped lang="scss">
.s2a-stats {
  min-height: 100vh;
  max-width: 100%;
  overflow-x: hidden;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-page);
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 20;
  backdrop-filter: blur(14px);
  background: color-mix(in srgb, var(--el-bg-color-page) 72%, transparent);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.topbar-inner,
.wrap {
  width: min(1180px, calc(100% - 40px));
  margin: 0 auto;
}

.topbar-inner {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
  border: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 16px;
  font-weight: 700;
  cursor: pointer;
}

.brand-dot {
  flex: none;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--el-color-primary), #22d3ee);
}

.topbar-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 36px;
  padding: 0 18px;
  border: 1px solid var(--el-border-color);
  border-radius: 999px;
  background: var(--el-bg-color-overlay);
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition:
    border-color 0.18s cubic-bezier(0.4, 0, 0.2, 1),
    background 0.18s cubic-bezier(0.4, 0, 0.2, 1),
    color 0.18s cubic-bezier(0.4, 0, 0.2, 1);
}
.btn:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.btn--ghost {
  background: var(--el-bg-color-overlay);
  margin-left: 4px;
}

.wrap {
  padding: 32px 0 64px;
}

/* 区块入场：快、克制、无弹跳（01 §6）；--d 做 stagger */
.reveal {
  opacity: 0;
  transform: translateY(10px);
  animation: s2a-reveal 0.22s cubic-bezier(0.4, 0, 0.2, 1) forwards;
  animation-delay: var(--d, 0ms);
}
@keyframes s2a-reveal {
  to {
    opacity: 1;
    transform: none;
  }
}
@media (prefers-reduced-motion: reduce) {
  .reveal {
    opacity: 1;
    transform: none;
    animation: none;
  }
}

.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 24px;

  h1 {
    margin: 0;
    font-size: clamp(1.5rem, 3vw, 2rem);
    font-weight: 700;
    letter-spacing: -0.01em;
  }
  p {
    margin: 8px 0 0;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 1.55;
    max-width: 36rem;
  }
}

.head-tools {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.seg {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 2px;
  padding: 3px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
}
.seg__btn {
  height: 30px;
  padding: 0 12px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition:
    background 0.15s cubic-bezier(0.4, 0, 0.2, 1),
    color 0.15s cubic-bezier(0.4, 0, 0.2, 1);
}
.seg__btn:hover {
  color: var(--el-text-color-primary);
}
.seg__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-color-primary);
  box-shadow: 0 0 0 1px var(--el-border-color-lighter);
}

.spin {
  width: 14px;
  height: 14px;
  border: 2px solid var(--el-border-color);
  border-top-color: var(--el-color-primary);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
.spin--lg {
  width: 28px;
  height: 28px;
  border-width: 2.5px;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* 详情页首屏：明确 loading 区（非门户渐进占位） */
.loading-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  min-height: 280px;
  margin-top: 8px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 16px;
  background: var(--el-bg-color-overlay);
  color: var(--el-text-color-secondary);
  font-size: 14px;
  p {
    margin: 0;
  }
}

.reload-bar {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px 12px;
  border-radius: 10px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-secondary);
  font-size: 12.5px;
  font-weight: 600;
}

.body.is-reloading {
  opacity: 0.72;
  transition: opacity 0.18s cubic-bezier(0.4, 0, 0.2, 1);
  pointer-events: none;
}

.metrics {
  margin-bottom: 16px;
}

.panel {
  margin-top: 16px;
  min-width: 0;
  border: 1px solid var(--el-border-color-light);
  border-radius: 16px;
  background: var(--el-bg-color-overlay);
  padding: 18px 20px;
}

.panel-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 700;
  }
  span {
    color: var(--el-text-color-placeholder);
    font-size: 12.5px;
  }
}

.chart {
  width: 100%;
  height: 320px;
}

.desk-table-wrap {
  overflow-x: auto;
  max-height: 320px;
  overflow-y: auto;
}

.desk-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;

  th,
  td {
    padding: 9px 10px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    text-align: left;
    vertical-align: middle;
  }
  th {
    position: sticky;
    top: 0;
    z-index: 1;
    background: var(--el-bg-color-overlay);
    color: var(--el-text-color-secondary);
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.02em;
  }
  tbody tr {
    transition: background 0.15s cubic-bezier(0.4, 0, 0.2, 1);
  }
  tbody tr:hover td {
    background: color-mix(in srgb, var(--el-color-primary) 4%, transparent);
  }
  .name {
    min-width: 140px;
    max-width: 320px;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  .rate {
    color: #16a34a;
    font-weight: 600;
  }
}

.mobile-list {
  display: none;
}

.rank-row {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr) 52px;
  align-items: center;
  gap: 10px;
  padding: 10px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
  &:last-child {
    border-bottom: 0;
  }
}

.rank-no {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 7px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 700;
  &.top {
    color: #fff;
    background: var(--el-color-primary);
  }
}

.rank-body {
  min-width: 0;
}

.rank-line {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;

  strong {
    min-width: 0;
    font-size: 13.5px;
    font-weight: 700;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  span {
    flex: none;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}

.rank-bar {
  height: 5px;
  border-radius: 999px;
  background: var(--el-fill-color);
  overflow: hidden;
  i {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: linear-gradient(90deg, var(--el-color-primary), #22d3ee);
    transition: width 0.5s cubic-bezier(0.4, 0, 0.2, 1);
  }
}

.rank-rate {
  display: inline-flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 1px;
  color: #16a34a;
  font-size: 12px;
  font-weight: 700;
  text-align: right;
  em {
    color: var(--el-text-color-placeholder);
    font-size: 10px;
    font-style: normal;
    font-weight: 600;
  }
}

@media (max-width: 640px) {
  .topbar-inner,
  .wrap {
    width: min(1180px, calc(100% - 24px));
  }

  /* 单行顶栏：不 wrap，避免动作区位移 */
  .topbar-inner {
    height: 56px;
    flex-wrap: nowrap;
    gap: 8px;
    overflow: hidden;
  }

  .brand {
    flex: 1 1 auto;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 15px;
  }

  .topbar-actions {
    flex: 0 0 auto;
    flex-wrap: nowrap;
    gap: 2px;
  }

  .btn--ghost {
    height: 32px;
    padding: 0 10px;
    font-size: 12.5px;
  }

  .wrap {
    padding: 22px 0 40px;
  }

  .head {
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 14px;

    h1 {
      font-size: 22px;
    }
    p {
      margin-top: 4px;
      font-size: 12.5px;
    }
  }

  .head-tools {
    width: 100%;
  }

  .seg {
    max-width: 100%;
    overflow-x: auto;
    flex-wrap: nowrap;
    scrollbar-width: none;
    &::-webkit-scrollbar {
      display: none;
    }
  }

  .panel {
    margin-top: 10px;
    padding: 12px;
    border-radius: 12px;
  }

  .panel-head h3 {
    font-size: 15px;
  }

  .chart {
    height: 250px;
  }

  .desk-table-wrap {
    display: none;
  }

  .mobile-list {
    display: flex;
    flex-direction: column;
    max-height: 360px;
    overflow-y: auto;
  }
}

@media (max-width: 360px) {
  /* 极窄：返回首页略缩，避免顶栏挤出 */
  .btn--ghost {
    padding: 0 8px;
    font-size: 12px;
  }

  .rank-line {
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;

    strong {
      overflow: visible;
      white-space: normal;
      line-height: 1.35;
    }
    span {
      flex: auto;
    }
  }
}
</style>
