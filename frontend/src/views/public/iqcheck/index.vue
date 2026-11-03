<!-- 智商检测公示页（公开，无需登录）：路由 /public/iqcheck/:slug。
     主色页头（站点名称/检测间隔/公开路径）+ 统计卡 + 模型筛选 chips + 时间线结果卡（沙箱渲染预览）；
     宽屏右侧粘性原文面板，窄屏点开弹窗详情。?r=<结果 id> 打开分享的单条结果。 -->
<template>
  <div class="iqp">
    <header class="iqp-hero">
      <span class="iqp-hero__blob" aria-hidden="true" />
      <span class="iqp-hero__ring" aria-hidden="true" />
      <div class="iqp-hero__inner">
        <div class="iqp-hero__bar">
          <img :src="APP_CONFIG.logoSrc" alt="" class="iqp-hero__logo" />
          <span class="iqp-hero__name">{{ APP_CONFIG.name }}</span>
          <span class="iqp-hero__chip">{{ t('publicPage.iq.publicChip') }}</span>
          <span class="iqp-hero__spacer" />
          <button
            type="button"
            class="iqp-hero__mode"
            :aria-label="t('login.toggleTheme')"
            @click="toggleTheme"
          >
            <component :is="menuStore.iconComponents[isDark ? 'HOutline:SunIcon' : 'HOutline:MoonIcon']" />
          </button>
          <a class="iqp-hero__login" :href="loginHref">{{ t('iqcheck.public.login') }}</a>
        </div>
        <div class="iqp-hero__main">
          <div class="iqp-hero__copy">
            <h1>{{ site ? t('publicPage.iq.title', { name: site.name }) : t('publicPage.iq.titleFallback') }}</h1>
            <p v-if="site">
              {{ t('iqcheck.public.interval', { minutes: site.intervalMinutes }) }}<template v-if="!site.enabled">
                · {{ t('publicPage.iq.paused') }}</template
              >
              · {{ t('publicPage.iq.heroDesc') }}
            </p>
            <div v-if="site" class="iqp-hero__meta">
              <span class="iqp-hero__pill">{{ protocolLabel }}</span>
              <span class="m-mono iqp-hero__endpoint">{{ site.endpoint }}</span>
            </div>
          </div>
          <span class="iqp-hero__path m-mono">
            <component :is="menuStore.iconComponents['HOutline:GlobeAltIcon']" />
            /public/iqcheck/{{ slug }}
          </span>
        </div>
      </div>
    </header>

    <main class="iqp-main">
      <div v-if="loading" class="iqp-state">{{ t('iqcheck.public.loading') }}</div>
      <EmptyState
        v-else-if="notFound || !site"
        icon="HOutline:LinkSlashIcon"
        :title="t('iqcheck.public.notFound')"
      />

      <template v-else>
        <section class="iqp-stats">
          <div v-for="s in statCards" :key="s.key" class="iqp-stat">
            <span class="m-sticker iqp-stat__ico" :class="`m-sticker--${s.tone}`" :style="{ rotate: s.rot }">
              <component :is="menuStore.iconComponents[s.icon]" />
            </span>
            <span class="iqp-stat__text">
              <span class="iqp-stat__label">{{ s.label }}</span>
              <span class="iqp-stat__value">
                <b class="m-num">{{ s.value }}</b><small>{{ s.unit }}</small>
              </span>
            </span>
          </div>
        </section>

        <div class="iqp-toolbar">
          <span class="iqp-toolbar__title">{{ t('publicPage.iq.records') }}</span>
          <div class="iqp-chips" role="tablist" :aria-label="t('iqcheck.public.model')">
            <button
              v-for="f in modelFilters"
              :key="f.value"
              type="button"
              role="tab"
              class="iqp-chip"
              :class="{ 'is-on': modelFilter === f.value }"
              :aria-selected="modelFilter === f.value"
              @click="pickModel(f.value)"
            >
              <span :class="{ 'm-mono': f.value }">{{ f.label }}</span>
              <span class="iqp-chip__n m-num">{{ f.count }}</span>
            </button>
          </div>
          <span class="iqp-toolbar__spacer" />
          <span class="iqp-toolbar__hint">
            {{ t('publicPage.iq.newestFirst') }} ·
            {{
              site.lastCheckAt
                ? t('iqcheck.public.lastCheck', { time: fmtTime(site.lastCheckAt) })
                : t('iqcheck.public.never')
            }}
          </span>
        </div>

        <div v-if="sharedOnly" class="iqp-shared">
          <component :is="menuStore.iconComponents['HOutline:LinkIcon']" />
          <span>{{ t('publicPage.iq.sharedOnly') }}</span>
          <button type="button" class="iqp-shared__btn" @click="showAll">{{ t('publicPage.iq.showAll') }}</button>
        </div>

        <div class="iqp-body">
          <div class="iqp-list-wrap">
            <div v-if="resultsLoading && !results.length" class="iqp-state">{{ t('iqcheck.public.loading') }}</div>
            <EmptyState
              v-else-if="!results.length"
              icon="HOutline:BeakerIcon"
              :title="t('iqcheck.public.empty')"
              :description="t('publicPage.iq.emptyDesc')"
            />
            <ol v-else class="iqp-list" :class="{ 'is-loading': resultsLoading }">
              <li v-for="r in results" :key="r.id" class="iqp-item">
                <span class="iqp-item__rail" aria-hidden="true">
                  <span class="iqp-item__dot" :class="isSuccess(r) ? 'is-ok' : 'is-bad'" />
                  <span class="iqp-item__line" />
                </span>
                <article class="iqp-card" :class="{ 'is-sel': current?.id === r.id && panelVisible }">
                  <div class="iqp-card__head">
                    <span class="iqp-card__model m-mono">{{ r.model }}</span>
                    <span class="iqp-eff" :class="`is-${effortTone(r.effort)}`">
                      {{ t('publicPage.iq.effortTag', { effort: r.effort || t('iqcheck.public.noEffort') }) }}
                    </span>
                    <span v-if="r.promptTitle" class="iqp-tag">{{ r.promptTitle }}</span>
                    <span class="iqp-card__spacer" />
                    <span class="iqp-card__meta m-num">
                      <component :is="menuStore.iconComponents['HOutline:ClockIcon']" />
                      {{ seconds(r.durationMs) }}
                    </span>
                    <span class="iqp-card__meta m-num">{{ fmtTime(r.requestAt) }}</span>
                    <StatusPill
                      :variant="isSuccess(r) ? 'success' : 'error'"
                      :label="isSuccess(r) ? t('iqcheck.public.success') : t('iqcheck.public.failed')"
                    />
                  </div>

                  <div class="iqp-sbx">
                    <div class="iqp-sbx__head">
                      <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />
                      {{ t('publicPage.iq.sandbox') }}
                    </div>
                    <button
                      v-if="isSuccess(r)"
                      type="button"
                      class="iqp-sbx__frame"
                      :aria-label="t('iqcheck.public.viewDetail')"
                      @click="openDetail(r)"
                    >
                      <ReplyFrame :text="r.responseText" :dark="isDark" preview :title="r.model" />
                    </button>
                    <div v-else class="iqp-sbx__err">
                      <component :is="menuStore.iconComponents['HOutline:ExclamationTriangleIcon']" />
                      <span>
                        <b>{{ t('publicPage.iq.failedTitle') }}</b>
                        <span>{{ r.error || '—' }}</span>
                      </span>
                    </div>
                  </div>

                  <div class="iqp-card__foot">
                    <span class="iqp-card__usage m-num">{{ usageText(r) }}</span>
                    <span class="iqp-card__spacer" />
                    <AppIconButton
                      icon="HOutline:LinkIcon"
                      :box="40"
                      :size="18"
                      :label="t('publicPage.iq.copyLink')"
                      @click="copyLink(r)"
                    />
                    <UButton
                      color="neutral"
                      variant="soft"
                      class="iqp-btn"
                      icon="i-lucide-file-text"
                      :aria-pressed="current?.id === r.id && panelVisible"
                      @click="openDetail(r)"
                    >
                      {{ t('publicPage.iq.viewRaw') }}
                    </UButton>
                  </div>
                </article>
              </li>
            </ol>

            <div v-if="pagination.total > pagination.pageSize && !sharedOnly" class="iqp-pagination">
              <Pagination
                v-model:page="pagination.page"
                v-model:page-size="pagination.pageSize"
                :total="pagination.total"
                :page-sizes="[12, 24, 48]"
                @change="loadResults"
              />
            </div>
          </div>

          <!-- 宽屏：右侧原文面板 -->
          <aside v-if="panelVisible && current" class="iqp-panel" :aria-label="t('iqcheck.public.detailTitle')">
            <div class="iqp-panel__head">
              <span class="iqp-panel__title">
                <span class="iqp-panel__time">{{ t('publicPage.iq.rawAt', { time: fmtTime(current.requestAt) }) }}</span>
                <span class="iqp-panel__model">
                  <b class="m-mono">{{ current.model }}</b>
                  <span class="iqp-eff" :class="`is-${effortTone(current.effort)}`">
                    {{ t('publicPage.iq.effortTag', { effort: current.effort || t('iqcheck.public.noEffort') }) }}
                  </span>
                </span>
              </span>
              <AppIconButton
                icon="HOutline:ArrowsPointingOutIcon"
                :label="t('publicPage.iq.expand')"
                @click="detailOpen = true"
              />
              <AppIconButton icon="HOutline:XMarkIcon" :label="t('publicPage.iq.closePanel')" @click="closePanel" />
            </div>
            <dl class="iqp-panel__dl">
              <div>
                <dt>{{ t('iqcheck.public.duration') }}</dt>
                <dd class="m-num">{{ seconds(current.durationMs) }}</dd>
              </div>
              <div>
                <dt>{{ t('iqcheck.public.status') }}</dt>
                <dd :class="isSuccess(current) ? 'is-ok' : 'is-bad'">
                  {{ isSuccess(current) ? t('iqcheck.public.success') : t('iqcheck.public.failed') }}
                </dd>
              </div>
              <div>
                <dt>{{ t('publicPage.iq.usage') }}</dt>
                <dd class="m-num">{{ tokensShort(current) }}</dd>
              </div>
            </dl>
            <section class="iqp-panel__sec">
              <div class="iqp-panel__sec-head">
                <h2>{{ t('iqcheck.public.prompt') }}</h2>
                <AppIconButton
                  icon="HOutline:ClipboardDocumentIcon"
                  :box="36"
                  :size="17"
                  :label="t('publicPage.iq.copyPrompt')"
                  @click="copy(current.promptContent)"
                />
              </div>
              <div v-if="current.promptTitle" class="iqp-panel__ptitle">{{ current.promptTitle }}</div>
              <pre class="iqp-raw">{{ current.promptContent }}</pre>
            </section>
            <section class="iqp-panel__sec">
              <div class="iqp-panel__sec-head">
                <h2>{{ isSuccess(current) ? t('iqcheck.public.raw') : t('iqcheck.public.error') }}</h2>
                <AppIconButton
                  icon="HOutline:ClipboardDocumentIcon"
                  :box="36"
                  :size="17"
                  :label="t('publicPage.iq.copyRaw')"
                  @click="copy(isSuccess(current) ? current.responseText : current.error)"
                />
              </div>
              <pre class="iqp-raw iqp-raw--tall" :class="{ 'is-bad': !isSuccess(current) }">{{
                isSuccess(current) ? current.responseText : current.error
              }}</pre>
            </section>
          </aside>
        </div>
      </template>
    </main>

    <footer class="iqp-foot">{{ t('publicPage.iq.footer', { name: APP_CONFIG.name }) }}</footer>

    <ResultDetail v-model="detailOpen" :result="current" :dark="isDark" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useMediaQuery } from '@vueuse/core'
import dayjs from 'dayjs'
import { useThemeStore } from '@/stores/theme'
import { APP_CONFIG } from '@/config/app.config'
import { getPublicIQCheckSite, listPublicIQCheckResults } from '@/api/iqcheck'
import { copyTextToClipboard } from '@/utils/file'
import { useFeedback } from '@/utils/feedback'
import {
  IQCheckProtocol,
  IQCheckResultStatus,
  type IQCheckResultInfo,
  type PublicIQCheckSiteInfo,
} from '@/types/v1/iqcheck'
import ReplyFrame from '@/views/public/iqcheck/ReplyFrame.vue'
import ResultDetail from '@/views/public/iqcheck/ResultDetail.vue'

defineOptions({ name: 'PublicIQCheckView' })

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const themeStore = useThemeStore()
const menuStore = useMenuStore()
const fb = useFeedback()
const isDark = computed(() => themeStore.isDarkTheme)
const slug = String(route.params.slug || '')
const loginHref = `${import.meta.env.BASE_URL.replace(/\/$/, '')}/login`
const wide = useMediaQuery('(min-width: 1180px)')

const loading = ref(true)
const notFound = ref(false)
const site = ref<PublicIQCheckSiteInfo | null>(null)

const resultsLoading = ref(false)
const results = ref<IQCheckResultInfo[]>([])
const modelFilter = ref('')
const pagination = ref({ page: 1, pageSize: 12, total: 0 })
const sharedId = ref(typeof route.query.r === 'string' ? route.query.r : '')
const sharedOnly = computed(() => !!sharedId.value)

const detailOpen = ref(false)
const panelOpen = ref(true)
const current = ref<IQCheckResultInfo | null>(null)
const panelVisible = computed(() => wide.value && panelOpen.value && !!current.value)

const toggleTheme = () => themeStore.toggleThemeMode(isDark.value ? 'light' : 'dark')

const protocolLabels: Record<string, string> = {
  [IQCheckProtocol.IQ_CHECK_PROTOCOL_CHAT]: 'chat',
  [IQCheckProtocol.IQ_CHECK_PROTOCOL_RESPONSES]: 'responses',
  [IQCheckProtocol.IQ_CHECK_PROTOCOL_MESSAGES]: 'messages',
}
const protocolLabel = computed(() => (site.value ? protocolLabels[site.value.protocol] || '—' : '—'))

const isSuccess = (r: IQCheckResultInfo) => r.status !== IQCheckResultStatus.IQ_CHECK_RESULT_STATUS_FAILED
const fmtTime = (v: unknown) => (v ? dayjs(v as string | Date).format('YYYY-MM-DD HH:mm') : '—')
const seconds = (ms: number | string) => t('iqcheck.public.seconds', { seconds: (Number(ms) / 1000).toFixed(1) })
const tokensShort = (r: IQCheckResultInfo) => {
  const total = (Number(r.inputTokens) || 0) + (Number(r.outputTokens) || 0)
  return total > 0 ? t('publicPage.iq.tokens', { n: total }) : '—'
}
const usageText = (r: IQCheckResultInfo) => {
  if (!(Number(r.inputTokens) > 0 || Number(r.outputTokens) > 0)) return t('publicPage.iq.noUsage')
  return t('publicPage.iq.usageDetail', { input: r.inputTokens, output: r.outputTokens })
}
const effortTone = (effort: string) => {
  if (['high', 'xhigh', 'max'].includes(effort)) return 'p'
  if (effort === 'medium') return 'a2'
  if (['low', 'minimal', 'none'].includes(effort)) return 'a'
  return 'n'
}

const pct = (a: number, b: number) => (b > 0 ? ((a / b) * 100).toFixed(b >= 100 ? 1 : 0) : '—')
const statCards = computed(() => {
  const s = site.value?.stats
  const total = Number(s?.total) || 0
  const recent = Number(s?.last7dTotal) || 0
  return [
    {
      key: 'rate',
      label: t('publicPage.iq.statRate'),
      value: pct(Number(s?.success) || 0, total),
      unit: total ? '%' : '',
      icon: 'HOutline:TrophyIcon',
      tone: 'p',
      rot: '-4deg',
    },
    {
      key: 'recent',
      label: t('publicPage.iq.statRecent'),
      value: pct(Number(s?.last7dSuccess) || 0, recent),
      unit: recent ? '%' : '',
      icon: 'HOutline:ChartBarIcon',
      tone: 'a2',
      rot: '3deg',
    },
    {
      key: 'total',
      label: t('publicPage.iq.statTotal'),
      value: String(total),
      unit: t('publicPage.iq.unitRecords'),
      icon: 'HOutline:ListBulletIcon',
      tone: 'a',
      rot: '-2deg',
    },
    {
      key: 'avg',
      label: t('publicPage.iq.statAvg'),
      value: Number(s?.avgDurationMs) ? (Number(s?.avgDurationMs) / 1000).toFixed(1) : '—',
      unit: Number(s?.avgDurationMs) ? t('publicPage.iq.unitSeconds') : '',
      icon: 'HOutline:ClockIcon',
      tone: 'ok',
      rot: '4deg',
    },
  ]
})

const modelFilters = computed(() => {
  const counts = new Map<string, number>()
  for (const m of site.value?.stats?.models || []) counts.set(m.model, Number(m.count) || 0)
  for (const m of site.value?.models || []) if (!counts.has(m.model)) counts.set(m.model, 0)
  return [
    { value: '', label: t('iqcheck.public.allModels'), count: Number(site.value?.stats?.total) || 0 },
    ...[...counts.entries()].map(([model, count]) => ({ value: model, label: model, count })),
  ]
})

const loadSite = async () => {
  loading.value = true
  try {
    const { data } = await getPublicIQCheckSite(slug)
    site.value = data?.site || null
    if (!site.value) notFound.value = true
    else document.title = `${site.value.name} · ${APP_CONFIG.name}`
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

const loadResults = async () => {
  resultsLoading.value = true
  try {
    const { data } = await listPublicIQCheckResults({
      slug,
      page: sharedOnly.value ? 1 : pagination.value.page,
      pageSize: sharedOnly.value ? 1 : pagination.value.pageSize,
      model: modelFilter.value || undefined,
      resultId: sharedId.value || undefined,
    })
    results.value = data?.results || []
    pagination.value.total = Number(data?.total || 0)
    // 宽屏默认展开第一条（或分享的那条）
    if (!current.value || !results.value.some((r) => r.id === current.value?.id)) {
      current.value = results.value[0] || null
    }
    if (sharedOnly.value && current.value && !wide.value) detailOpen.value = true
  } catch {
    results.value = []
  } finally {
    resultsLoading.value = false
  }
}

const reload = () => {
  pagination.value.page = 1
  loadResults()
}

const pickModel = (model: string) => {
  if (modelFilter.value === model && !sharedOnly.value) return
  modelFilter.value = model
  if (sharedOnly.value) showAll()
  else reload()
}

const showAll = () => {
  sharedId.value = ''
  router.replace({ query: { ...route.query, r: undefined } })
  reload()
}

const openDetail = (r: IQCheckResultInfo) => {
  current.value = r
  if (wide.value) panelOpen.value = true
  else detailOpen.value = true
}

const closePanel = () => {
  panelOpen.value = false
}

const copy = async (text: string) => {
  try {
    await copyTextToClipboard(text || '')
    fb.success(t('file.share.copied'))
  } catch {
    fb.error(t('file.share.copyFailed'))
  }
}

const copyLink = (r: IQCheckResultInfo) => {
  const url = `${window.location.origin}${window.location.pathname}?r=${encodeURIComponent(r.id)}`
  copy(url)
}

onMounted(async () => {
  await loadSite()
  if (site.value) await loadResults()
})
</script>

<style scoped lang="scss">
.iqp {
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
  color: var(--el-text-color-primary);
}

/* ---- hero ---- */
.iqp-hero {
  position: relative;
  overflow: hidden;
  padding: 22px 48px 30px;
  background: var(--m-p);
  color: var(--m-pOn);
}
.iqp-hero__blob {
  position: absolute;
  right: -60px;
  top: -110px;
  width: 300px;
  height: 300px;
  border-radius: 50%;
  background: var(--m-a);
}
.iqp-hero__ring {
  position: absolute;
  right: 290px;
  bottom: -46px;
  width: 104px;
  height: 104px;
  border-radius: 50%;
  border: 16px solid var(--m-a2);
}
.iqp-hero__inner {
  position: relative;
  max-width: 1240px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.iqp-hero__bar {
  display: flex;
  align-items: center;
  gap: 12px;
}
.iqp-hero__logo {
  width: 40px;
  height: 40px;
  border-radius: 13px;
  object-fit: cover;
  transform: rotate(-6deg);
  box-shadow: 0 0 0 3px var(--m-pOn);
}
.iqp-hero__name {
  font-family: var(--app-font-display);
  font-size: 1.375rem;
  font-weight: 800;
}
.iqp-hero__chip {
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 10px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--m-pOn) 18%, transparent);
  font-size: 0.75rem;
  font-weight: 700;
  white-space: nowrap;
}
.iqp-hero__spacer {
  flex: 1;
}
.iqp-hero__mode {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 14px;
  background: var(--m-pOn);
  color: var(--m-p);
  cursor: pointer;
}
.iqp-hero__mode svg {
  width: 20px;
  height: 20px;
}
.iqp-hero__login {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  padding: 0 18px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  font-size: 0.875rem;
  font-weight: 700;
  text-decoration: none;
}
.iqp-hero__login:hover {
  color: var(--m-bg);
  filter: brightness(1.1);
}
.iqp-hero__main {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 20px;
}
.iqp-hero__copy {
  flex: 1 1 520px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.iqp-hero__copy h1 {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 2.5rem;
  line-height: 1.1;
  font-weight: 800;
  letter-spacing: -0.01em;
  overflow-wrap: anywhere;
}
.iqp-hero__copy p {
  margin: 0;
  font-size: 0.9375rem;
  line-height: 1.6;
}
.iqp-hero__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.iqp-hero__pill {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  background: var(--m-a);
  color: var(--m-aOn);
  font-size: 0.75rem;
  font-weight: 800;
}
.iqp-hero__endpoint {
  min-width: 0;
  font-size: 0.8125rem;
  opacity: 0.92;
  overflow-wrap: anywhere;
}
.iqp-hero__path {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  max-width: 100%;
  height: 36px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--m-pOn);
  color: var(--m-p);
  font-size: 0.8125rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iqp-hero__path svg {
  width: 16px;
  height: 16px;
  flex: none;
}

/* ---- main ---- */
.iqp-main {
  flex: 1;
  width: 100%;
  max-width: 1336px;
  margin: 0 auto;
  padding: 28px 48px 40px;
  display: flex;
  flex-direction: column;
  gap: 22px;
}
.iqp-state {
  padding: 48px 16px;
  text-align: center;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.iqp-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}
.iqp-stat {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.iqp-stat__ico {
  width: 44px;
  height: 44px;
  border-radius: 14px;
}
.iqp-stat__ico svg {
  width: 22px;
  height: 22px;
}
.iqp-stat__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.iqp-stat__label {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.iqp-stat__value {
  display: flex;
  align-items: baseline;
  gap: 4px;
}
.iqp-stat__value b {
  font-size: 1.625rem;
  font-weight: 800;
  line-height: 1;
}
.iqp-stat__value small {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}

.iqp-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.iqp-toolbar__title {
  margin-right: 4px;
  font-size: 0.875rem;
  font-weight: 800;
}
.iqp-chips {
  display: flex;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.iqp-chips::-webkit-scrollbar {
  display: none;
}
.iqp-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: none;
  min-height: 40px;
  padding: 0 14px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  white-space: nowrap;
}
.iqp-chip:hover {
  border-color: var(--el-text-color-secondary);
}
.iqp-chip.is-on {
  border-color: var(--m-ink);
  background: var(--m-ink);
  color: var(--m-bg);
}
.iqp-chip__n {
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
}
.iqp-chip.is-on .iqp-chip__n {
  background: color-mix(in srgb, var(--m-bg) 20%, transparent);
  color: var(--m-bg);
}
.iqp-toolbar__spacer {
  flex: 1;
}
.iqp-toolbar__hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.iqp-shared {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 10px 10px 16px;
  border-radius: var(--app-radius);
  background: var(--m-a2Soft);
  font-size: 0.875rem;
  font-weight: 600;
}
.iqp-shared svg {
  width: 18px;
  height: 18px;
  flex: none;
}
.iqp-shared span {
  flex: 1;
}
.iqp-shared__btn {
  min-height: 40px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--m-pInk);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

.iqp-body {
  display: flex;
  align-items: flex-start;
  gap: 24px;
}
.iqp-list-wrap {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.iqp-list {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
  transition: opacity 0.15s;
}
.iqp-list.is-loading {
  opacity: 0.6;
}
.iqp-item {
  display: flex;
  gap: 16px;
}
.iqp-item__rail {
  flex: none;
  width: 22px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.iqp-item__dot {
  width: 16px;
  height: 16px;
  margin-top: 22px;
  border-radius: 50%;
  border: 4px solid var(--el-bg-color-page);
}
.iqp-item__dot.is-ok {
  background: var(--m-ok);
  box-shadow: 0 0 0 2px var(--m-ok);
}
.iqp-item__dot.is-bad {
  background: var(--m-bad);
  box-shadow: 0 0 0 2px var(--m-bad);
}
.iqp-item__line {
  flex: 1;
  width: 2px;
  margin-top: 4px;
  background: var(--el-border-color-light);
}
.iqp-item:last-child .iqp-item__line {
  background: linear-gradient(var(--el-border-color-light), transparent);
}
.iqp-card {
  flex: 1;
  min-width: 0;
  margin-bottom: 16px;
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  transition:
    border-color 0.15s,
    box-shadow 0.15s;
}
.iqp-card.is-sel {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.iqp-card__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.iqp-card__model {
  font-size: 1rem;
  font-weight: 800;
  overflow-wrap: anywhere;
}
.iqp-card__spacer {
  flex: 1;
}
.iqp-card__meta {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.iqp-card__meta svg {
  width: 14px;
  height: 14px;
}
.iqp-eff,
.iqp-tag {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 700;
  white-space: nowrap;
}
.iqp-eff.is-p {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.iqp-eff.is-a2 {
  background: var(--m-a2Soft);
  color: var(--el-text-color-primary);
}
.iqp-eff.is-a {
  background: var(--m-aSoft);
  color: var(--m-aOn);
}
html.dark .iqp-eff.is-a {
  color: var(--m-a);
}
.iqp-eff.is-n,
.iqp-tag {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.iqp-tag {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.iqp-sbx {
  overflow: hidden;
  border-radius: var(--app-radius);
  border: 1.5px dashed var(--el-border-color-light);
}
.iqp-sbx__head {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: var(--el-fill-color-light);
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.iqp-sbx__head svg {
  width: 13px;
  height: 13px;
}
.iqp-sbx__frame {
  display: block;
  width: 100%;
  height: 240px;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: zoom-in;
}
.iqp-sbx__err {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 22px 20px;
  color: var(--m-bad);
}
.iqp-sbx__err > svg {
  width: 22px;
  height: 22px;
  flex: none;
}
.iqp-sbx__err > span {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.iqp-sbx__err b {
  font-size: 0.9375rem;
  font-weight: 800;
}
.iqp-sbx__err span span {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 4;
  line-clamp: 4;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.iqp-card__foot {
  display: flex;
  align-items: center;
  gap: 8px;
}
.iqp-card__usage {
  min-width: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.iqp-btn {
  min-height: 40px;
  border-radius: 999px !important;
  font-weight: 700 !important;
}
.iqp-pagination {
  display: flex;
  justify-content: center;
}

/* 原文面板 */
.iqp-panel {
  flex: 0 0 440px;
  position: sticky;
  top: 24px;
  max-height: calc(100vh - 48px);
  overflow-y: auto;
  scrollbar-width: none;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.iqp-panel::-webkit-scrollbar {
  display: none;
}
.iqp-panel__head {
  display: flex;
  align-items: center;
  gap: 4px;
}
.iqp-panel__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.iqp-panel__time {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.iqp-panel__model {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.iqp-panel__model b {
  font-size: 1.125rem;
  font-weight: 800;
  overflow-wrap: anywhere;
}
.iqp-panel__dl {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px 12px;
  margin: 0;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.iqp-panel__dl dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.iqp-panel__dl dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
}
.iqp-panel__dl dd.is-ok {
  color: var(--m-ok);
}
.iqp-panel__dl dd.is-bad {
  color: var(--m-bad);
}
.iqp-panel__sec {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.iqp-panel__sec-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.iqp-panel__sec-head h2 {
  flex: 1;
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 800;
}
.iqp-panel__ptitle {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.iqp-raw {
  margin: 0;
  max-height: 220px;
  overflow: auto;
  scrollbar-width: none;
  padding: 12px 14px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
  font-family: var(--app-font-mono);
  font-size: 0.75rem;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}
.iqp-raw::-webkit-scrollbar {
  display: none;
}
.iqp-raw--tall {
  max-height: 420px;
}
.iqp-raw.is-bad {
  color: var(--m-bad);
}
.iqp-foot {
  padding: 18px 48px 28px;
  text-align: center;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

@media (width < 1180px) {
  .iqp-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

/* ---- 手机 ---- */
@media (width < 768px) {
  .iqp-hero {
    padding: 16px 16px 24px;
  }
  .iqp-hero__blob {
    right: -70px;
    top: -90px;
    width: 200px;
    height: 200px;
  }
  .iqp-hero__ring {
    right: 24px;
    bottom: -30px;
    width: 70px;
    height: 70px;
    border-width: 12px;
  }
  .iqp-hero__inner {
    gap: 18px;
  }
  .iqp-hero__bar {
    gap: 10px;
  }
  .iqp-hero__chip {
    display: none;
  }
  .iqp-hero__logo {
    width: 36px;
    height: 36px;
    border-radius: 12px;
  }
  .iqp-hero__name {
    font-size: 1.2rem;
  }
  .iqp-hero__mode {
    width: 40px;
    height: 40px;
  }
  .iqp-hero__login {
    min-height: 40px;
    padding: 0 14px;
    font-size: 0.8125rem;
  }
  .iqp-hero__copy {
    flex-basis: 100%;
  }
  .iqp-hero__copy h1 {
    font-size: 1.75rem;
    line-height: 1.15;
  }
  .iqp-hero__copy p {
    font-size: 0.875rem;
  }
  .iqp-hero__path {
    font-size: 0.75rem;
    height: 32px;
  }
  .iqp-main {
    padding: 16px 16px 28px;
    gap: 16px;
  }
  .iqp-stats {
    gap: 10px;
  }
  .iqp-stat {
    gap: 10px;
    padding: 12px;
  }
  .iqp-stat__ico {
    width: 36px;
    height: 36px;
    border-radius: 12px;
    svg {
      width: 18px;
      height: 18px;
    }
  }
  .iqp-stat__value b {
    font-size: 1.375rem;
  }
  .iqp-toolbar {
    gap: 8px;
  }
  .iqp-toolbar__title {
    display: none;
  }
  .iqp-chips {
    flex: 1 1 100%;
    margin: 0 -16px;
    padding: 0 16px;
    max-width: none;
  }
  .iqp-toolbar__spacer {
    display: none;
  }
  .iqp-toolbar__hint {
    font-size: 0.75rem;
  }
  .iqp-item {
    gap: 0;
  }
  .iqp-item__rail {
    display: none;
  }
  .iqp-card {
    margin-bottom: 12px;
    padding: 14px;
    gap: 12px;
  }
  .iqp-card__head .iqp-card__spacer {
    display: none;
  }
  .iqp-card__model {
    flex: 1 1 100%;
    font-size: 0.9375rem;
  }
  .iqp-sbx__frame {
    height: 200px;
  }
  .iqp-foot {
    padding: 8px 16px 24px;
  }
}
</style>
