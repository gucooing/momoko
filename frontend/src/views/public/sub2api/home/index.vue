<template>
  <main class="s2a-home" :class="{ 'is-dark': isDark }">
    <!-- 乐观渲染：元信息到达前先出骨架外壳，各区块数据到达后各自填充；仅明确「已禁用」才显示禁用态 -->
    <template v-if="!home || home.enabled">
      <div class="bg-accent" aria-hidden />

      <!-- 顶栏 -->
      <header class="topbar">
        <div class="topbar-inner">
          <button class="brand" type="button" @click="scrollTo('top')">
            <span class="brand-dot" />
            {{ home?.title || 'Sub2API' }}
          </button>
          <div class="topbar-actions">
            <LanguageMenu />
            <AppIconButton
              :icon="isDark ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
              :label="t('login.toggleTheme')"
              :box="36"
              @click="toggleTheme"
            />
            <span class="bell-wrap">
              <AppIconButton
                icon="HOutline:BellIcon"
                :label="t('sub2api.common.announcements')"
                :box="36"
                @click="openAnnouncements"
              />
              <span v-if="announcements.length" class="badge-dot" aria-hidden="true" />
            </span>
          </div>
        </div>
      </header>

      <!-- Hero -->
      <section id="top" class="hero">
        <span class="status-chip" :class="`tone-${status.tone}`">
          <span class="ping"><i /><em /></span>{{ status.text }}
        </span>
        <h1 class="hero-title">
          {{ heroLead }}<br />
          <span class="grad">{{ heroHighlight }}</span>
        </h1>
        <p class="hero-sub">{{ heroSubtitle }}</p>
        <div class="hero-cta">
          <button v-if="dashboardUrl" class="btn btn--primary btn--lg" type="button" @click="openConsole">
            {{ t('sub2api.home.goConsole') }}
            <component :is="menuStore.iconComponents['HOutline:ArrowRightIcon']" class="btn-ico" />
          </button>
          <button class="btn btn--ghost btn--lg" type="button" @click="goStats()">{{ t('sub2api.home.usageDetails') }}</button>
        </div>

        <div class="stat-strip">
          <div v-for="stat in heroStats" :key="stat.label" class="stat">
            <strong>{{ stat.value }}</strong>
            <span>{{ stat.label }}</span>
          </div>
        </div>
      </section>

      <!-- 用量：首页只看今日，切换区间进入详细统计页 -->
      <section id="usage" class="usage">
        <div class="section-head">
          <div>
            <h2>{{ t('sub2api.home.todayOverview') }}</h2>
            <span>{{ t('sub2api.home.todayOverviewDesc') }}</span>
          </div>
        </div>
        <div class="usage-grid">
          <article class="panel chart-panel">
            <div class="panel-head">
              <h3>{{ t('sub2api.home.successAndSpeed') }}</h3>
              <span>{{ t('sub2api.home.todayByTime') }}</span>
            </div>
            <VChart
              class="chart"
              :option="store.todaySeriesOption"
              :update-options="chartUpdate"
              autoresize
            />
          </article>
          <article class="panel rank-panel">
            <div class="panel-head">
              <h3>{{ t('sub2api.home.todayHotModels') }}</h3>
              <button class="btn btn--text" type="button" @click="goStats(7)">{{ t('sub2api.home.detailArrow') }}</button>
            </div>
            <div v-if="models.length" class="rank-list">
              <div v-for="(item, i) in models" :key="item.name || i" class="rank-row">
                <span class="rank-no" :class="{ top: i < 3 }">{{ i + 1 }}</span>
                <div class="rank-body">
                  <div class="rank-line">
                    <b>{{ item.name || t('sub2api.common.unknownModel') }}</b>
                    <span
                      >{{ store.formatThroughput(item.avgTps) }} t/s ·
                      {{ store.formatToken(item.tokenCount) }}</span
                    >
                  </div>
                  <div class="rank-bar"><i :style="{ width: barWidth(item.tokenCount) }" /></div>
                </div>
                <span class="rank-rate"
                  ><em>{{ t('sub2api.common.successRate') }}</em>{{ store.formatPercent(item.successRate) }}</span
                >
              </div>
            </div>
            <div v-else class="empty">{{ t('sub2api.home.noModelsToday') }}</div>
          </article>
        </div>
      </section>

      <!-- 公告 / 时间线 -->
      <section v-if="announcements.length || timeline.length" id="updates" class="updates">
        <div class="updates-grid">
          <article v-if="announcements.length" class="panel">
            <div class="panel-head"><h3>{{ t('sub2api.home.platformAnnouncements') }}</h3></div>
            <div class="notice-list">
              <div
                v-for="item in announcements"
                :key="item.id"
                class="notice"
                :class="`lv-${item.level || 'info'}`"
              >
                <div class="notice-head">
                  <b>{{ item.title || t('sub2api.common.announcements') }}</b>
                  <StatusPill v-if="item.pinned" variant="warning" :dot="false" :label="t('sub2api.common.pinned')" />
                </div>
                <p>{{ item.content }}</p>
                <time v-if="item.publishedAt">{{ store.formatDateTime(item.publishedAt) }}</time>
              </div>
            </div>
          </article>

          <article v-if="timeline.length" class="panel">
            <div class="panel-head"><h3>{{ t('sub2api.home.updateTimeline') }}</h3></div>
            <div class="timeline">
              <div v-for="item in timeline" :key="item.id" class="tl-item">
                <span class="tl-dot" />
                <div class="tl-body">
                  <div class="tl-head">
                    <b>{{ item.title || t('sub2api.common.update') }}</b>
                    <em>{{ item.category || t('sub2api.common.update') }}</em>
                  </div>
                  <p>{{ item.content }}</p>
                  <time v-if="item.publishedAt">{{ store.formatDateTime(item.publishedAt) }}</time>
                </div>
              </div>
            </div>
          </article>
        </div>
      </section>

      <footer class="footer">
        <span>© {{ year }} {{ home?.title || 'Sub2API' }}</span>
        <a v-if="dashboardUrl" :href="dashboardUrl" target="_top">{{ t('sub2api.home.goConsole') }}</a>
      </footer>

      <!-- 公告入口弹窗 -->
      <FormDialog v-model="announcementVisible" :title="t('sub2api.home.platformAnnouncements')" :width="520" :show-footer="false">
        <div v-if="announcements.length" class="notice-list">
          <div
            v-for="item in announcements"
            :key="item.id"
            class="notice"
            :class="`lv-${item.level || 'info'}`"
          >
            <div class="notice-head">
              <b>{{ item.title || t('sub2api.common.announcements') }}</b>
              <StatusPill v-if="item.pinned" variant="warning" :dot="false" :label="t('sub2api.common.pinned')" />
            </div>
            <p>{{ item.content }}</p>
            <time v-if="item.publishedAt">{{ store.formatDateTime(item.publishedAt) }}</time>
          </div>
        </div>
        <EmptyState v-else :title="t('sub2api.home.noAnnouncements')" />
      </FormDialog>
    </template>

    <section v-else class="disabled">
      <h1>{{ t('sub2api.home.disabledTitle') }}</h1>
      <p>{{ t('sub2api.home.disabledDesc') }}</p>
    </section>
  </main>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import VChart from '@/components/chart/VChart.vue'
import LanguageMenu from '@/layouts/app/LanguageMenu.vue'
import { useSub2APIStore } from '@/stores/sub2api'
import { useThemeStore } from '@/stores/theme'

defineOptions({ name: 'Sub2APIPublicHome' })

const store = useSub2APIStore()
const themeStore = useThemeStore()
const menuStore = useMenuStore()
const router = useRouter()
const { home } = storeToRefs(store)
const { t } = useI18n()

const isDark = computed(() => themeStore.isDarkTheme)
const toggleTheme = () => themeStore.toggleThemeMode(isDark.value ? 'light' : 'dark')

const chartUpdate = { notMerge: true }
const announcementVisible = ref(false)

const goStats = (range = 7) =>
  router.push({ path: '/public/sub2api/stats', query: { range: String(range) } })

const SEEN_ANNOUNCEMENTS_KEY = 'sub2api:seen-announcements'

const readSeenAnnouncements = (): string[] => {
  try {
    const raw = localStorage.getItem(SEEN_ANNOUNCEMENTS_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    return Array.isArray(parsed) ? parsed.map(String) : []
  } catch {
    return []
  }
}

const markAnnouncementsSeen = () => {
  const ids = announcements.value.map((a) => a.id)
  localStorage.setItem(SEEN_ANNOUNCEMENTS_KEY, JSON.stringify(ids))
}

const openAnnouncements = () => {
  announcementVisible.value = true
  markAnnouncementsSeen()
}

// 有未读公告时自动弹出（利用 localStorage 判断是否为新公告）
const maybePopAnnouncements = () => {
  const ids = announcements.value.map((a) => a.id)
  if (!ids.length) return
  const seen = readSeenAnnouncements()
  if (ids.some((id) => !seen.includes(id))) {
    openAnnouncements()
  }
}

const year = dayjs().year()
// 今日概览来自独立的公开概览接口（状态 + 今日标量 + 今日曲线）
const snapshot = computed(() => store.publicOverview)
const announcements = computed(() => home.value?.announcements || [])
const timeline = computed(() => home.value?.timeline || [])
// 首页只看今日：模型排行取今日区间
const models = computed(() =>
  [...(store.stats?.models || [])]
    .sort((a, b) => (Number(b.tokenCount) || 0) - (Number(a.tokenCount) || 0))
    .slice(0, 8),
)

const dashboardUrl = computed(() => {
  const base = home.value?.consoleUrl?.trim()
  if (!base) return ''
  return `${base.replace(/\/+$/, '')}/dashboard`
})
const openConsole = () => {
  // 首页可能以 iframe 形式嵌入运营站点，直接改 location 只会导航 iframe 自身，
  // 会被父级 frame-src CSP 拦截；跳转顶层窗口以离开 iframe。
  if (dashboardUrl.value) (window.top ?? window).location.href = dashboardUrl.value
}

const heroLead = computed(() => t('sub2api.home.heroLead'))
const heroHighlight = computed(() => {
  const title = home.value?.title?.trim()
  return title && title !== 'Sub2API' ? title : t('sub2api.home.heroHighlight')
})
const heroSubtitle = computed(
  () =>
    home.value?.introduction?.trim() ||
    home.value?.subtitle?.trim() ||
    t('sub2api.home.heroSubtitle'),
)

const status = computed(() => {
  switch (snapshot.value?.status) {
    case 'syncing':
      return { tone: 'amber', text: t('sub2api.home.statusSyncing') }
    case 'error':
      return { tone: 'red', text: t('sub2api.home.statusError') }
    case 'success':
    case 'idle':
      return { tone: 'green', text: t('sub2api.home.statusRunning') }
    default:
      return { tone: 'blue', text: t('sub2api.home.statusDashboard') }
  }
})

const heroStats = computed(() => {
  const s = snapshot.value
  // 概览未到达前显示占位「—」，到达后填充真实数据（渐进渲染，不显示整页 loading）
  if (!s) {
    return [
      { label: t('sub2api.home.todayRequests'), value: '—' },
      { label: t('sub2api.home.todayToken'), value: '—' },
      { label: t('sub2api.home.todaySuccessRate'), value: '—' },
      { label: t('sub2api.home.tokenSpeed'), value: '—' },
    ]
  }
  return [
    { label: t('sub2api.home.todayRequests'), value: store.formatNumber(s.todayRequestCount) },
    { label: t('sub2api.home.todayToken'), value: store.formatToken(s.todayTokenCount) },
    { label: t('sub2api.home.todaySuccessRate'), value: store.formatPercent(s.todaySuccessRate) },
    { label: t('sub2api.home.tokenSpeed'), value: `${store.formatThroughput(s.recentTps)} t/s` },
  ]
})

const maxToken = computed(() => Math.max(...models.value.map((m) => Number(m.tokenCount) || 0), 1))
const barWidth = (token: unknown) =>
  `${Math.max(((Number(token) || 0) / maxToken.value) * 100, 4)}%`

const scrollTo = (id: string) => document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })

onMounted(() => {
  // 三模块并行拉取、各自渲染，避免单个大请求阻塞整页；公告在元信息到达后再判断是否弹出
  store.loadPublicHome().then(() => maybePopAnnouncements())
  store.loadPublicOverview()
  store.loadStats(1)
})
</script>

<style scoped lang="scss">
.s2a-home {
  position: relative;
  min-height: 100vh;
  max-width: 100%;
  overflow-x: hidden;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-page);
}

.bg-accent {
  position: absolute;
  inset: 0 0 auto;
  height: 560px;
  z-index: 0;
  pointer-events: none;
  opacity: 0.5;
  background:
    radial-gradient(
      ellipse 48% 40% at 20% 0%,
      color-mix(in srgb, var(--el-color-primary) 30%, transparent),
      transparent 62%
    ),
    radial-gradient(
      ellipse 44% 36% at 82% 6%,
      color-mix(in srgb, #22d3ee 26%, transparent),
      transparent 64%
    );
}

.is-dark .bg-accent {
  opacity: 0.2;
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
.hero,
.usage,
.updates,
.footer {
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
  flex: 0 1 auto;
  min-width: 0;
  gap: 9px;
  border: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 16px;
  font-weight: 800;
  line-height: 1.25;
  text-align: left;
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
  justify-content: flex-end;
  min-width: 0;
  gap: 4px;

  .btn {
    flex: none;
    margin-left: 4px;
  }
}

.bell-wrap {
  position: relative;
  display: inline-flex;
}
.badge-dot {
  position: absolute;
  top: 7px;
  right: 8px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--el-color-danger, #ef4444);
  box-shadow: 0 0 0 2px var(--el-bg-color-page);
  pointer-events: none;
}
.btn-ico {
  width: 16px;
  height: 16px;
}

/* 令牌按钮（替代 el-button：圆角胶囊、默认/主色/大号/文本四态） */
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
  transition: border-color 0.2s, background 0.2s, color 0.2s, transform 0.08s;
}
.btn:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.btn:active {
  transform: translateY(1px);
}
.btn :deep(svg) {
  width: 16px;
  height: 16px;
}
.btn--primary {
  border-color: transparent;
  background: var(--el-color-primary);
  color: #fff;
}
.btn--primary:hover {
  background: color-mix(in srgb, var(--el-color-primary) 88%, #000);
  color: #fff;
}
.btn--lg {
  height: 44px;
  padding: 0 26px;
  font-size: 15px;
}
.btn--text {
  height: auto;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--el-color-primary);
  font-size: 13px;
  font-weight: 500;
}
.btn--text:hover {
  background: transparent;
  color: color-mix(in srgb, var(--el-color-primary) 78%, #000);
}

/* hero */
.hero {
  position: relative;
  z-index: 1;
  padding: 72px 0 36px;
  text-align: center;
}

.status-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  border-radius: 999px;
  border: 1px solid color-mix(in srgb, var(--chip) 26%, transparent);
  background: color-mix(in srgb, var(--chip) 10%, transparent);
  color: var(--chip);
  font-size: 13px;
  font-weight: 700;

  --chip: var(--el-color-primary);
  &.tone-green {
    --chip: #10b981;
  }
  &.tone-amber {
    --chip: #f59e0b;
  }
  &.tone-red {
    --chip: #ef4444;
  }
  &.tone-blue {
    --chip: #3b82f6;
  }

  i,
  b {
    position: absolute;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--chip);
  }
  i {
    position: static;
  }
  b {
    animation: chip-ping 1.6s cubic-bezier(0, 0, 0.2, 1) infinite;
    margin-left: -7px;
  }
}

@keyframes chip-ping {
  75%,
  100% {
    transform: scale(2.4);
    opacity: 0;
  }
}

.hero-title {
  margin: 22px 0 16px;
  font-size: clamp(2rem, 4.5vw, 3.2rem);
  font-weight: 800;
  line-height: 1.14;
  letter-spacing: -0.01em;

  .grad {
    background: linear-gradient(90deg, var(--el-color-primary), #22d3ee 60%, #8b5cf6);
    background-clip: text;
    -webkit-background-clip: text;
    color: transparent;
  }
}

.hero-sub {
  max-width: 36rem;
  margin: 0 auto;
  color: var(--el-text-color-secondary);
  font-size: 16px;
  line-height: 1.8;
}

.hero-cta {
  display: flex;
  justify-content: center;
  gap: 12px;
  margin-top: 28px;

  .ml {
    margin-left: 4px;
  }
}

.stat-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  max-width: 760px;
  margin: 44px auto 0;
}

.stat {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 18px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 14px;
  background: var(--el-bg-color-overlay);

  strong {
    font-size: 22px;
    font-weight: 800;
    font-variant-numeric: tabular-nums;
    color: var(--el-text-color-primary);
  }
  span {
    font-size: 12.5px;
    color: var(--el-text-color-secondary);
  }
}

/* sections */
.usage,
.updates {
  position: relative;
  z-index: 1;
  padding: 56px 0;
}

.section-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 22px;

  h2 {
    margin: 0;
    font-size: clamp(1.5rem, 3vw, 2rem);
    font-weight: 800;
  }
  span {
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }
}

.panel {
  min-width: 0;
  border: 1px solid var(--el-border-color-light);
  border-radius: 16px;
  background: var(--el-bg-color-overlay);
  padding: 20px 22px;
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
  strong {
    font-variant-numeric: tabular-nums;
    font-size: 18px;
    font-weight: 800;
    span {
      color: var(--el-text-color-placeholder);
      font-size: 11px;
      font-weight: 600;
    }
  }
}

.usage-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr);
  gap: 16px;
}

.chart {
  width: 100%;
  height: 320px;
}

.rank-list {
  display: flex;
  flex-direction: column;
}

.rank-body {
  min-width: 0;
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

.rank-line {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
  b {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13.5px;
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
  }
}

.rank-rate {
  display: inline-flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 1px;
  text-align: right;
  color: #10b981;
  font-size: 12px;
  font-weight: 700;
}

.rank-rate em {
  color: var(--el-text-color-placeholder);
  font-size: 10px;
  font-style: normal;
  font-weight: 600;
}

.empty {
  padding: 40px;
  text-align: center;
  color: var(--el-text-color-placeholder);
}

.updates-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.notice-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.notice {
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-left: 3px solid var(--el-color-primary);
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
  &.lv-success {
    border-left-color: var(--el-color-success);
  }
  &.lv-warning {
    border-left-color: var(--el-color-warning);
  }
  &.lv-danger {
    border-left-color: var(--el-color-danger);
  }

  .notice-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    b {
      font-size: 14px;
    }
  }
  p {
    margin: 8px 0 0;
    color: var(--el-text-color-regular);
    font-size: 13px;
    line-height: 1.7;
    white-space: pre-wrap;
  }
  time {
    display: block;
    margin-top: 8px;
    color: var(--el-text-color-placeholder);
    font-size: 12px;
  }
}

.timeline {
  display: flex;
  flex-direction: column;
}

.tl-item {
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr);
  gap: 12px;
  position: relative;
  padding-bottom: 16px;
  &::before {
    content: '';
    position: absolute;
    left: 5px;
    top: 16px;
    bottom: -2px;
    width: 1px;
    background: var(--el-border-color-light);
  }
  &:last-child {
    padding-bottom: 0;
    &::before {
      display: none;
    }
  }
}

.tl-dot {
  width: 11px;
  height: 11px;
  margin-top: 4px;
  border-radius: 50%;
  background: var(--el-color-primary);
  border: 3px solid color-mix(in srgb, var(--el-color-primary) 26%, transparent);
}

.tl-head {
  display: flex;
  align-items: center;
  gap: 10px;
  b {
    font-size: 14px;
  }
  em {
    font-style: normal;
    font-size: 12px;
    font-weight: 700;
    color: var(--el-color-primary);
    background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
    padding: 2px 8px;
    border-radius: 999px;
  }
}

.tl-body p {
  margin: 7px 0 0;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.7;
  white-space: pre-wrap;
}
.tl-body time {
  display: block;
  margin-top: 7px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.footer {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 28px 0;
  border-top: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-secondary);
  font-size: 13px;
  a {
    color: var(--el-color-primary);
    font-weight: 700;
    text-decoration: none;
  }
}

.disabled {
  position: relative;
  z-index: 1;
  width: min(640px, calc(100% - 32px));
  margin: 16vh auto;
  padding: 36px;
  text-align: center;
  border: 1px solid var(--el-border-color-light);
  border-radius: 16px;
  background: var(--el-bg-color-overlay);
  h1 {
    margin: 0 0 10px;
    font-size: 22px;
  }
  p {
    margin: 0;
    color: var(--el-text-color-secondary);
    line-height: 1.7;
  }
}

@media (max-width: 900px) {
  .usage-grid,
  .updates-grid {
    grid-template-columns: 1fr;
  }
  .stat-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .topbar-inner,
  .hero,
  .usage,
  .updates,
  .footer {
    width: min(1180px, calc(100% - 24px));
  }

  /* 单行顶栏：禁止 wrap 导致按钮「位移」到第二行 */
  .topbar-inner {
    height: 56px;
    flex-wrap: nowrap;
    gap: 8px;
    padding: 0;
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
    max-width: none;
  }

  .usage,
  .updates {
    padding: 34px 0;
  }

  .section-head {
    margin-bottom: 16px;

    h2 {
      font-size: 24px;
    }

    span {
      font-size: 12.5px;
      line-height: 1.5;
    }
  }

  .panel {
    padding: 14px 16px;
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

  .rank-row {
    grid-template-columns: 24px minmax(0, 1fr) auto;
    gap: 8px;
    padding: 9px 0;
  }

  .rank-line {
    align-items: flex-start;
    flex-direction: column;
    gap: 2px;

    b {
      overflow: visible;
      white-space: normal;
      line-height: 1.35;
    }

    span {
      flex: auto;
      line-height: 1.35;
    }
  }

  .rank-rate {
    white-space: nowrap;
  }

  .empty {
    padding: 24px;
  }
}
</style>
