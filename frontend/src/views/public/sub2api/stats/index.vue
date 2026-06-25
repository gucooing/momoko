<template>
  <main class="s2a-stats" :class="{ 'is-dark': isDark }">
    <header class="topbar">
      <div class="topbar-inner">
        <button class="brand" type="button" @click="goHome">
          <span class="brand-dot" />{{ title }}
        </button>
        <div class="topbar-actions">
          <button class="icon-btn" type="button" @click="toggleTheme">
            <el-icon><component :is="isDark ? Sunny : Moon" /></el-icon>
          </button>
          <el-button text @click="goHome">{{ t('sub2api.stats.backHome') }}</el-button>
        </div>
      </div>
    </header>

    <section class="wrap">
      <div class="head">
        <div>
          <h1>{{ t('sub2api.stats.title') }}</h1>
          <p>{{ t('sub2api.stats.desc') }}</p>
        </div>
        <el-segmented v-model="range" :options="rangeOptions" @change="onRange" />
      </div>

      <div v-loading="store.statsLoading" class="content">
        <div class="metrics">
          <div
            v-for="card in store.statsMetricCards"
            :key="card.label"
            class="metric"
            :class="`tone-${card.tone}`"
          >
            <span class="m-label">{{ card.label }}</span>
            <strong class="m-value">{{ card.value }}</strong>
            <small class="m-detail">{{ card.detail }}</small>
          </div>
        </div>

        <article class="panel">
          <div class="panel-head">
            <h3>{{ t('sub2api.common.usageTrend') }}</h3>
            <span>{{ stats?.rangeLabel }}</span>
          </div>
          <VChart
            class="chart"
            :option="store.statsTrendOption"
            :update-options="chartUpdate"
            autoresize
          />
        </article>

        <article class="panel">
          <div class="panel-head">
            <h3>{{ t('sub2api.stats.modelDetails') }}</h3>
            <span>{{ t('sub2api.common.modelCount', { count: models.length }) }}</span>
          </div>
          <el-table class="desktop-table" :data="models" :max-height="288" stripe>
            <el-table-column type="index" label="#" width="56" />
            <el-table-column prop="name" :label="t('sub2api.common.model')" min-width="180" show-overflow-tooltip />
            <el-table-column
              :label="t('sub2api.common.requestCount')"
              width="120"
              sortable
              :sort-method="(a: any, b: any) => a.requestCount - b.requestCount"
            >
              <template #default="{ row }">{{ store.formatNumber(row.requestCount) }}</template>
            </el-table-column>
            <el-table-column label="Token" width="120">
              <template #default="{ row }">{{ store.formatToken(row.tokenCount) }}</template>
            </el-table-column>
            <el-table-column :label="t('sub2api.common.generationSpeed')" width="130">
              <template #default="{ row }"
                >{{ store.formatThroughput(row.avgTps) }} token/s</template
              >
            </el-table-column>
            <el-table-column :label="t('sub2api.common.successRate')" width="110">
              <template #default="{ row }">{{ store.formatPercent(row.successRate) }}</template>
            </el-table-column>
            <template #empty>{{ t('sub2api.common.noData') }}</template>
          </el-table>
          <div class="mobile-detail-list">
            <article v-for="(row, i) in models" :key="row.name || i" class="detail-rank-row">
              <span class="detail-index">{{ i + 1 }}</span>
              <div class="detail-rank-body">
                <div class="detail-rank-line">
                  <strong>{{ row.name || t('sub2api.common.unknownModel') }}</strong>
                  <span
                    >{{ store.formatThroughput(row.avgTps) }} t/s ·
                    {{ store.formatToken(row.tokenCount) }}</span
                  >
                </div>
                <div class="detail-bar">
                  <i :style="{ width: modelBarWidth(row.tokenCount) }" />
                </div>
              </div>
              <span class="detail-rate"
                ><em>{{ t('sub2api.common.successRate') }}</em>{{ store.formatPercent(row.successRate) }}</span
              >
            </article>
            <el-empty v-if="!models.length" :description="t('sub2api.common.noData')" />
          </div>
        </article>

        <article v-if="endpoints.length" class="panel">
          <div class="panel-head"><h3>{{ t('sub2api.stats.endpointDetails') }}</h3></div>
          <el-table class="desktop-table" :data="endpoints" :max-height="288" stripe>
            <el-table-column type="index" label="#" width="56" />
            <el-table-column prop="name" :label="t('sub2api.common.endpoint')" min-width="220" show-overflow-tooltip />
            <el-table-column :label="t('sub2api.common.requestCount')" width="120">
              <template #default="{ row }">{{ store.formatNumber(row.requestCount) }}</template>
            </el-table-column>
            <el-table-column label="Token" width="120">
              <template #default="{ row }">{{ store.formatToken(row.tokenCount) }}</template>
            </el-table-column>
            <el-table-column :label="t('sub2api.common.successRate')" width="110">
              <template #default="{ row }">{{ store.formatPercent(row.successRate) }}</template>
            </el-table-column>
          </el-table>
          <div class="mobile-detail-list">
            <article v-for="(row, i) in endpoints" :key="row.name || i" class="detail-rank-row">
              <span class="detail-index">{{ i + 1 }}</span>
              <div class="detail-rank-body">
                <div class="detail-rank-line">
                  <strong>{{ row.name || t('sub2api.common.unknownEndpoint') }}</strong>
                  <span
                    >{{ t('sub2api.common.countTimes', { count: store.formatNumber(row.requestCount) }) }} ·
                    {{ store.formatToken(row.tokenCount) }}</span
                  >
                </div>
                <div class="detail-bar">
                  <i :style="{ width: endpointBarWidth(row.tokenCount) }" />
                </div>
              </div>
              <span class="detail-rate"
                ><em>{{ t('sub2api.common.successRate') }}</em>{{ store.formatPercent(row.successRate) }}</span
              >
            </article>
          </div>
        </article>

        <article v-if="userAgents.length" class="panel">
          <div class="panel-head">
            <h3>{{ t('sub2api.stats.uaDetails') }}</h3>
            <span>{{ t('sub2api.common.countTimes', { count: store.formatNumber(totalUaRequests) }) }}</span>
          </div>
          <el-table class="desktop-table" :data="userAgents" :max-height="288" stripe>
            <el-table-column type="index" label="#" width="56" />
            <el-table-column prop="name" :label="t('sub2api.common.ua')" min-width="260" show-overflow-tooltip />
            <el-table-column :label="t('sub2api.common.requestCount')" width="120">
              <template #default="{ row }">{{ store.formatNumber(row.requestCount) }}</template>
            </el-table-column>
            <el-table-column label="Token" width="120">
              <template #default="{ row }">{{ store.formatToken(row.tokenCount) }}</template>
            </el-table-column>
            <el-table-column :label="t('sub2api.common.successRate')" width="110">
              <template #default="{ row }">{{ store.formatPercent(row.successRate) }}</template>
            </el-table-column>
          </el-table>
          <div class="mobile-detail-list">
            <article v-for="(row, i) in userAgents" :key="row.name || i" class="detail-rank-row">
              <span class="detail-index">{{ i + 1 }}</span>
              <div class="detail-rank-body">
                <div class="detail-rank-line">
                  <strong>{{ row.name || t('sub2api.common.unknownUa') }}</strong>
                  <span
                    >{{ t('sub2api.common.countTimes', { count: store.formatNumber(row.requestCount) }) }} ·
                    {{ store.formatToken(row.tokenCount) }}</span
                  >
                </div>
                <div class="detail-bar">
                  <i :style="{ width: uaBarWidth(row.requestCount) }" />
                </div>
              </div>
              <span class="detail-rate"
                ><em>{{ t('sub2api.common.successRate') }}</em>{{ store.formatPercent(row.successRate) }}</span
              >
            </article>
          </div>
        </article>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Moon, Sunny } from '@element-plus/icons-vue'
import VChart from '@/components/chart/VChart.vue'
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
const endpoints = computed(() =>
  [...(stats.value?.endpoints || [])].sort(
    (a, b) => (Number(b.tokenCount) || 0) - (Number(a.tokenCount) || 0),
  ),
)
// UA 明细按使用数量（请求数）降序
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
const maxEndpointToken = computed(() =>
  Math.max(...endpoints.value.map((item) => Number(item.tokenCount) || 0), 1),
)
const maxUaCount = computed(() =>
  Math.max(...userAgents.value.map((item) => Number(item.requestCount) || 0), 1),
)
const modelBarWidth = (token: unknown) =>
  `${Math.max(((Number(token) || 0) / maxModelToken.value) * 100, 4)}%`
const endpointBarWidth = (token: unknown) =>
  `${Math.max(((Number(token) || 0) / maxEndpointToken.value) * 100, 4)}%`
const uaBarWidth = (count: unknown) =>
  `${Math.max(((Number(count) || 0) / maxUaCount.value) * 100, 4)}%`

const onRange = (value: number | string | boolean) => {
  const days = parseRange(value)
  router.replace({ query: { ...route.query, range: String(days) } })
  store.loadStats(days)
}

const goHome = () => router.push('/public/sub2api/home')

onMounted(() => {
  store.loadStats(range.value)
})
</script>

<style scoped lang="scss">
.s2a-stats {
  min-height: 100vh;
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
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  border: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 16px;
  font-weight: 800;
  cursor: pointer;
}

.brand-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--el-color-primary), #22d3ee);
}

.topbar-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 9px;
  background: var(--el-bg-color-overlay);
  color: var(--el-text-color-primary);
  font-size: 17px;
  cursor: pointer;
}

.wrap {
  padding: 32px 0 64px;
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
    font-weight: 800;
  }
  p {
    margin: 8px 0 0;
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }
}

.metrics {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.metric {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 16px 18px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 14px;
  background: var(--el-bg-color-overlay);
  border-top: 3px solid var(--el-color-primary);

  &.tone-green {
    border-top-color: var(--el-color-success);
  }
  &.tone-amber {
    border-top-color: var(--el-color-warning);
  }
  &.tone-red {
    border-top-color: var(--el-color-danger);
  }
}

.m-label {
  color: var(--el-text-color-secondary);
  font-size: 12.5px;
}
.m-value {
  color: var(--el-text-color-primary);
  font-size: 20px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}
.m-detail {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
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

.mobile-detail-list {
  display: none;
}

.detail-rank-row {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr) 52px;
  align-items: center;
  gap: 10px;
  padding: 10px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.detail-rank-row:last-child {
  border-bottom: 0;
}

.detail-index {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 7px;
  background: var(--el-color-primary);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
}

.detail-rank-body {
  min-width: 0;
}

.detail-rank-line {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;

  strong {
    min-width: 0;
    color: var(--el-text-color-primary);
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

.detail-bar {
  height: 5px;
  border-radius: 999px;
  background: var(--el-fill-color);
  overflow: hidden;

  i {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: linear-gradient(90deg, var(--el-color-primary), #22d3ee);
  }
}

.detail-rate {
  display: inline-flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 1px;
  color: #10b981;
  font-size: 12px;
  font-weight: 700;
  text-align: right;
}

.detail-rate em {
  color: var(--el-text-color-placeholder);
  font-size: 10px;
  font-style: normal;
  font-weight: 600;
}

@media (max-width: 900px) {
  .metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .topbar-inner,
  .wrap {
    width: min(1180px, calc(100% - 24px));
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
      line-height: 1.5;
    }
  }

  .head :deep(.el-segmented) {
    max-width: 100%;
    overflow-x: auto;
    scrollbar-width: none;
  }

  .head :deep(.el-segmented::-webkit-scrollbar) {
    display: none;
  }

  .metrics {
    gap: 8px;
    margin-bottom: 10px;
  }

  .metric {
    gap: 4px;
    padding: 10px 12px;
    border-radius: 10px;
  }

  .m-label,
  .m-detail {
    font-size: 11.5px;
    line-height: 1.35;
  }

  .m-value {
    font-size: 17px;
    line-height: 1.25;
    overflow-wrap: anywhere;
  }

  .panel {
    margin-top: 10px;
    padding: 12px;
    border-radius: 12px;
  }

  .panel-head {
    margin-bottom: 10px;

    h3 {
      font-size: 15px;
    }
  }

  .chart {
    height: 250px;
  }

  .desktop-table {
    display: none;
  }

  .mobile-detail-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 360px;
    overflow-y: auto;
  }
}

@media (max-width: 360px) {
  .metrics,
  .detail-rank-line {
    align-items: flex-start;
    grid-template-columns: 1fr;
    flex-direction: column;
    gap: 2px;

    strong {
      overflow: visible;
      white-space: normal;
      line-height: 1.35;
    }

    span {
      flex: auto;
      line-height: 1.35;
    }
  }
}
</style>
