<!-- 智商检测公示页（公开，无需登录）：路由 /public/iqcheck/:slug。
     顶部展示站点名称/端点/协议/检测间隔/最近检测；下面按时间倒序铺检测结果卡片（渲染后的回复 + 模型/时间/推理强度），
     点击卡片弹出详情（请求提示词、回复原文、耗时等）。沿用文件模块的浅/暗设计令牌。 -->
<template>
  <div class="file-module iq-public" :class="{ 'is-dark': isDark }">
    <header class="iq-topbar">
      <div class="iq-topbar__brand">
        <img :src="APP_CONFIG.logoSrc" :alt="APP_CONFIG.name" class="iq-topbar__logo" />
        <span class="iq-topbar__name">{{ APP_CONFIG.name }}</span>
      </div>
      <a class="iq-topbar__login" :href="loginHref">{{ t('iqcheck.public.login') }}</a>
    </header>

    <main class="iq-main">
      <div v-if="loading" class="iq-state">{{ t('iqcheck.public.loading') }}</div>
      <div v-else-if="notFound || !site" class="iq-state">{{ t('iqcheck.public.notFound') }}</div>

      <template v-else>
        <section class="iq-hero">
          <div class="iq-hero__main">
            <h1 class="iq-hero__title">{{ site.name }}</h1>
            <div class="iq-hero__endpoint">{{ site.endpoint }}</div>
            <div class="iq-hero__meta">
              <span class="iq-pill">{{ protocolLabel }}</span>
              <span>{{ t('iqcheck.public.interval', { minutes: site.intervalMinutes }) }}</span>
              <span>
                {{
                  site.lastCheckAt
                    ? t('iqcheck.public.lastCheck', { time: fmtTime(site.lastCheckAt) })
                    : t('iqcheck.public.never')
                }}
              </span>
            </div>
          </div>
          <div class="iq-hero__tools">
            <select v-model="modelFilter" class="fm-input iq-hero__select" @change="reload">
              <option value="">{{ t('iqcheck.public.allModels') }}</option>
              <option v-for="m in site.models" :key="m.model" :value="m.model">{{ m.model }}</option>
            </select>
            <span class="iq-hero__total">{{ t('iqcheck.public.total', { total: pagination.total }) }}</span>
          </div>
        </section>

        <div v-if="resultsLoading && !results.length" class="iq-state">{{ t('iqcheck.public.loading') }}</div>
        <div v-else-if="!results.length" class="iq-state">{{ t('iqcheck.public.empty') }}</div>
        <section v-else class="iq-grid" :class="{ 'is-loading': resultsLoading }">
          <article v-for="r in results" :key="r.id" class="iq-card" @click="openDetail(r)">
            <div class="iq-card__preview">
              <ReplyFrame v-if="isSuccess(r)" :text="r.responseText" :dark="isDark" preview :title="r.model" />
              <div v-else class="iq-card__error">
                <span class="iq-card__error-title">{{ t('iqcheck.public.failed') }}</span>
                <span class="iq-card__error-text">{{ r.error }}</span>
              </div>
              <span class="iq-card__badge">{{ t('iqcheck.public.viewDetail') }} ↗</span>
            </div>
            <div class="iq-card__body">
              <div class="iq-card__row iq-card__row--head">
                <span class="iq-card__time">{{ fmtTime(r.requestAt) }}</span>
                <span class="iq-tag" :class="isSuccess(r) ? 'iq-tag--ok' : 'iq-tag--fail'">
                  {{ isSuccess(r) ? t('iqcheck.public.success') : t('iqcheck.public.failed') }}
                </span>
              </div>
              <div class="iq-card__row">
                <span class="iq-card__label">{{ t('iqcheck.public.model') }}</span>
                <span class="iq-card__model">{{ r.model }}</span>
              </div>
              <div class="iq-card__row">
                <span class="iq-card__label">{{ t('iqcheck.public.promptLabel') }}</span>
                <span class="iq-card__value">{{ r.promptTitle || '—' }}</span>
              </div>
              <div class="iq-card__row iq-card__row--dim">
                <span>{{ t('iqcheck.public.effort') }}: {{ r.effort || t('iqcheck.public.noEffort') }}</span>
                <span>·</span>
                <span>{{ t('iqcheck.public.seconds', { seconds: (Number(r.durationMs) / 1000).toFixed(1) }) }}</span>
              </div>
            </div>
          </article>
        </section>

        <div v-if="pagination.total > pagination.pageSize" class="iq-pagination">
          <Pagination
            v-model:page="pagination.page"
            v-model:page-size="pagination.pageSize"
            :total="pagination.total"
            :page-sizes="[12, 24, 48]"
            @change="loadResults"
          />
        </div>
      </template>
    </main>

    <ResultDetail v-model="detailOpen" :result="current" :dark="isDark" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import dayjs from 'dayjs'
import { useThemeStore } from '@/stores/theme'
import { APP_CONFIG } from '@/config/app.config'
import { getPublicIQCheckSite, listPublicIQCheckResults } from '@/api/iqcheck'
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
const themeStore = useThemeStore()
const isDark = computed(() => themeStore.isDarkTheme)
const slug = String(route.params.slug || '')
const loginHref = `${import.meta.env.BASE_URL.replace(/\/$/, '')}/login`

const loading = ref(true)
const notFound = ref(false)
const site = ref<PublicIQCheckSiteInfo | null>(null)

const resultsLoading = ref(false)
const results = ref<IQCheckResultInfo[]>([])
const modelFilter = ref('')
const pagination = ref({ page: 1, pageSize: 12, total: 0 })

const detailOpen = ref(false)
const current = ref<IQCheckResultInfo | null>(null)

const protocolLabels: Record<string, string> = {
  [IQCheckProtocol.IQ_CHECK_PROTOCOL_CHAT]: 'chat',
  [IQCheckProtocol.IQ_CHECK_PROTOCOL_RESPONSES]: 'responses',
  [IQCheckProtocol.IQ_CHECK_PROTOCOL_MESSAGES]: 'messages',
}
const protocolLabel = computed(() => (site.value ? protocolLabels[site.value.protocol] || '—' : '—'))

const isSuccess = (r: IQCheckResultInfo) => r.status !== IQCheckResultStatus.IQ_CHECK_RESULT_STATUS_FAILED
const fmtTime = (v: unknown) => (v ? dayjs(v as string | Date).format('YYYY-MM-DD HH:mm') : '—')

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
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      model: modelFilter.value || undefined,
    })
    results.value = data?.results || []
    pagination.value.total = Number(data?.total || 0)
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

const openDetail = (r: IQCheckResultInfo) => {
  current.value = r
  detailOpen.value = true
}

onMounted(async () => {
  await loadSite()
  if (site.value) await loadResults()
})
</script>

<style scoped>
.iq-public {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--fm-subtle);
}
.iq-topbar {
  height: 56px;
  padding: 0 1.5rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  background: var(--fm-surface);
  border-bottom: 1px solid var(--fm-border);
}
.iq-topbar__brand {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.iq-topbar__logo {
  width: 32px;
  height: 32px;
  border-radius: 0.5rem;
  object-fit: cover;
}
.iq-topbar__name {
  color: var(--fm-text);
  font-size: 1.05rem;
  font-weight: 700;
}
.iq-topbar__login {
  height: 32px;
  display: inline-flex;
  align-items: center;
  padding: 0 0.875rem;
  border-radius: 0.375rem;
  background: var(--fm-accent);
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  text-decoration: none;
}
.iq-topbar__login:hover {
  color: #fff;
  filter: brightness(0.96);
}
.iq-main {
  width: 100%;
  max-width: 1400px;
  margin: 0 auto;
  padding: 1.5rem 1rem 3rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}
.iq-state {
  padding: 3rem 1rem;
  text-align: center;
  color: var(--fm-text-3);
  font-size: 14px;
}
.iq-hero {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  padding: 1.25rem 1.5rem;
  background: var(--fm-surface);
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius);
  box-shadow: var(--fm-shadow-sm);
}
.iq-hero__main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}
.iq-hero__title {
  margin: 0;
  font-size: 1.35rem;
  font-weight: 700;
  color: var(--fm-text);
}
.iq-hero__endpoint {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 13px;
  color: var(--fm-text-2);
  overflow-wrap: anywhere;
}
.iq-hero__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
  font-size: 13px;
  color: var(--fm-text-3);
}
.iq-pill {
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--fm-accent-soft);
  color: var(--fm-accent-active);
  font-size: 12px;
  font-weight: 600;
}
.iq-hero__tools {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.iq-hero__select {
  min-width: 180px;
  height: 32px;
  padding: 0 0.625rem;
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius-sm);
  background: var(--fm-surface);
  color: var(--fm-text);
  font-size: 13px;
}
.iq-hero__total {
  font-size: 13px;
  color: var(--fm-text-3);
  white-space: nowrap;
}
.iq-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1rem;
  transition: opacity 0.15s;
}
.iq-grid.is-loading {
  opacity: 0.6;
}
.iq-card {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--fm-surface);
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius);
  box-shadow: var(--fm-shadow-sm);
  cursor: pointer;
  transition:
    transform 0.15s,
    box-shadow 0.15s,
    border-color 0.15s;
}
.iq-card:hover {
  transform: translateY(-2px);
  border-color: var(--fm-border-strong);
  box-shadow: var(--fm-shadow);
}
.iq-card__preview {
  position: relative;
  height: 240px;
  overflow: hidden;
  background: #fff;
  border-bottom: 1px solid var(--fm-border);
}
.is-dark .iq-card__preview {
  background: #1b1f24;
}
.iq-card__error {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
  overflow: hidden;
  background: var(--fm-danger-soft);
  color: var(--fm-danger);
  font-size: 13px;
}
.iq-card__error-title {
  font-weight: 700;
}
.iq-card__error-text {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 8;
  line-clamp: 8;
  word-break: break-word;
}
.iq-card__badge {
  position: absolute;
  right: 10px;
  bottom: 10px;
  padding: 3px 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid var(--fm-border);
  color: var(--fm-text-2);
  font-size: 12px;
}
.iq-card__body {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  padding: 0.875rem 1rem 1rem;
}
.iq-card__row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  font-size: 13px;
  color: var(--fm-text-2);
}
.iq-card__row--head {
  justify-content: space-between;
}
.iq-card__row--dim {
  color: var(--fm-text-3);
  font-size: 12px;
}
.iq-card__time {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  color: var(--fm-text-3);
}
.iq-card__label {
  flex-shrink: 0;
  color: var(--fm-text-3);
}
.iq-card__label::after {
  content: ':';
}
.iq-card__model {
  font-weight: 600;
  color: var(--fm-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-card__value {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-tag {
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
}
.iq-tag--ok {
  background: var(--fm-accent-soft);
  color: var(--fm-accent-active);
}
.iq-tag--fail {
  background: var(--fm-danger-soft);
  color: var(--fm-danger);
}
.iq-pagination {
  display: flex;
  justify-content: center;
}
@media (max-width: 640px) {
  .iq-topbar {
    padding: 0 1rem;
  }
  .iq-hero {
    padding: 1rem;
  }
  .iq-hero__tools {
    width: 100%;
  }
  .iq-hero__select {
    flex: 1;
    min-width: 0;
  }
}
</style>
