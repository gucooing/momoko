<!-- 文件来源（第二期改版）：页头(总数/启用数 + 全部测试/新增) + 文件子导航 + 类型胶囊(带数量)/搜索 +
     来源卡片网格（贴纸 · 名称/类型 · 启用开关 · 地址/根路径信息块 · 能力标签 · 最近测试结果 + 测试/编辑/⋯）。
     本地磁盘为内置默认来源，只读展示。最近一次测试结果由后端持久化（FileSourceInfo.lastTest）。 -->
<template>
  <div class="src">
    <PageHeader :title="t('fileSource.title')">
      <template #description>
        <span class="m-num">{{ t('fileSource.summaryTotal', { total: items.length + 1 }) }}</span>
        ·
        <span class="src-ok m-num">{{ t('fileSource.summaryEnabled', { enabled: enabledCount + 1 }) }}</span>
        <template v-if="!menuStore.isMobile"> · {{ t('fileSource.summaryTail') }}</template>
      </template>
      <template #actions>
        <UButton
          v-if="items.length"
          color="neutral"
          variant="outline"
          icon="i-lucide-plug-zap"
          :loading="testingAll"
          @click="testAll"
        >
          {{ t('fileSource.testAll') }}
        </UButton>
        <UButton color="primary" icon="i-lucide-plus" @click="openCreate">
          {{ menuStore.isMobile ? t('fileManager.newItem') : t('fileSource.add') }}
        </UButton>
      </template>
    </PageHeader>


    <ListToolbar
      v-model:status="typeFilter"
      v-model:keyword="keywords"
      :chips="typeChips"
      :chips-label="t('fileSource.type')"
      :placeholder="t('fileSource.keywordPlaceholder')"
      @search="search"
    />

    <div v-if="loading && !items.length" class="src-grid">
      <div v-for="i in 3" :key="i" class="src-skeleton" />
    </div>

    <div v-else class="src-grid">
      <!-- 内置：本地磁盘 -->
      <article v-if="showLocal" class="src-card">
        <header class="src-card__head">
          <span class="src-card__sticker m-sticker m-sticker--a" style="transform: rotate(-4deg)">
            <UIcon name="i-lucide-hard-drive" />
          </span>
          <span class="src-card__title">
            <span class="src-card__name">{{ t('fileSource.localDisk') }}</span>
            <span class="src-card__sub">{{ t('fileSource.localDesc') }}</span>
          </span>
          <StatusPill variant="primary" :label="t('fileSource.builtin')" :dot="false" />
        </header>
        <div class="src-card__info">
          <div class="src-card__line m-mono">
            <UIcon name="i-lucide-server" />
            <span>{{ t('fileManager.localTag') }}</span>
          </div>
          <div class="src-card__line src-card__line--dim m-mono">
            <UIcon name="i-lucide-folder" />
            <span>/</span>
          </div>
        </div>
        <div class="src-caps">
          <span class="src-cap"><UIcon name="i-lucide-minus" />{{ t('fileSource.no302') }}</span>
          <span v-for="cap in capsOf(localCaps)" :key="cap.key" class="src-cap" :class="{ 'is-on': cap.on }">
            <UIcon :name="cap.on ? cap.icon : 'i-lucide-minus'" />{{ cap.label }}
          </span>
        </div>
        <footer class="src-card__foot">
          <span class="src-res">
            <span class="src-res__icon is-ok"><UIcon name="i-lucide-check" /></span>
            <span class="src-res__text">
              <span class="src-res__title is-ok">{{ t('fileSource.alwaysOn') }}</span>
              <span class="src-res__sub">{{ t('fileSource.localHint') }}</span>
            </span>
          </span>
          <UButton v-if="canBrowse" color="neutral" variant="soft" size="sm" icon="i-lucide-folder-open" to="/file/index">
            {{ t('fileSource.browse') }}
          </UButton>
        </footer>
      </article>

      <article
        v-for="(row, i) in filtered"
        :key="row.id"
        class="src-card"
        :class="{ 'is-off': !row.enabled }"
      >
        <header class="src-card__head">
          <span class="src-card__sticker m-sticker" :class="`m-sticker--${visual(row.type).tone}`" :style="{ transform: `rotate(${ROTS[(i + 1) % ROTS.length]})` }">
            <UIcon :name="visual(row.type).icon" />
          </span>
          <span class="src-card__title">
            <span class="src-card__name">{{ row.name }}</span>
            <span class="src-card__sub">{{ subLine(row) }}</span>
          </span>
          <AppSwitch
            :model-value="!!row.enabled"
            :aria-label="row.name"
            @update:model-value="(v: boolean) => toggleEnabled(row, v)"
          />
        </header>
        <div class="src-card__info">
          <div class="src-card__line m-mono" :title="addressOf(row)">
            <UIcon name="i-lucide-link-2" />
            <span>{{ addressOf(row) || '—' }}</span>
          </div>
          <div class="src-card__line src-card__line--dim m-mono" :title="rootOf(row)">
            <UIcon name="i-lucide-folder" />
            <span>{{ rootOf(row) }}</span>
          </div>
        </div>
        <div class="src-caps">
          <span class="src-cap" :class="{ 'is-302': hasRedirect(row) }">
            <UIcon :name="hasRedirect(row) ? 'i-lucide-zap' : 'i-lucide-minus'" />{{ hasRedirect(row) ? t('fileSource.redirect302') : t('fileSource.no302') }}
          </span>
          <span v-for="cap in capsOf(row.caps)" :key="cap.key" class="src-cap" :class="{ 'is-on': cap.on }">
            <UIcon :name="cap.on ? cap.icon : 'i-lucide-minus'" />{{ cap.label }}
          </span>
        </div>
        <footer class="src-card__foot">
          <span class="src-res">
            <span class="src-res__icon" :class="`is-${result(row).tone}`">
              <UIcon :name="result(row).icon" :class="{ 'is-spin': testingIds.has(row.id) }" />
            </span>
            <span class="src-res__text">
              <span class="src-res__title m-num" :class="`is-${result(row).tone}`">{{ result(row).title }}</span>
              <span class="src-res__sub m-num" :title="result(row).sub">{{ result(row).sub }}</span>
            </span>
          </span>
          <UButton
            color="neutral"
            variant="soft"
            size="sm"
            icon="i-lucide-plug-zap"
            :loading="testingIds.has(row.id)"
            @click="testOne(row)"
          >
            {{ t('fileSource.test') }}
          </UButton>
          <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('system.common.edit')" :box="menuStore.isMobile ? 44 : 36" :size="18" @click="openEdit(row)" />
          <ActionMenu :items="rowMenu" :label="row.name" @select="(key) => onRowAction(key, row)" />
        </footer>
      </article>

      <div v-if="!loading && !filtered.length && (keywords || typeFilter)" class="src-card src-card--empty">
        <EmptyState icon="HOutline:CircleStackIcon" :title="t('system.common.noData')" />
      </div>
      <button v-else-if="!loading && !items.length" type="button" class="src-card src-card--add" @click="openCreate">
        <span class="src-card__sticker m-sticker m-sticker--p"><UIcon name="i-lucide-plus" /></span>
        <span class="src-card__name">{{ t('fileSource.add') }}</span>
        <span class="src-card__sub">{{ t('fileSource.emptyDesc') }}</span>
      </button>
    </div>

    <SourceCreate ref="createRef" @refresh="getList" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import SourceCreate from '@/views/file/source/create.vue'
import { Dialog } from '@/utils/dialog'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import {
  listFileSourcesRequest,
  updateFileSourceRequest,
  deleteFileSourceRequest,
  testFileSourceRequest,
} from '@/api/fileSource'
import type { FileSourceInfo, FileSourceConfig, FileSourceCaps, FileSourceTestResult } from '@/types/v1/file'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'

defineOptions({ name: 'FileSourceView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const router = useRouter()
const accessible = useAccessibleRoutes()
const canBrowse = computed(() => accessible.value.has('/file/index'))
const createRef = useTemplateRef<InstanceType<typeof SourceCreate> | null>('createRef')
const ROTS = ['-4deg', '3deg', '-2deg', '4deg', '-3deg']

const loading = ref(false)
const items = ref<FileSourceInfo[]>([])
const keywords = ref('')
type TypeFilter = '' | 'local' | 'oss' | 'ftp' | 'webdav'
const typeFilter = ref<TypeFilter>('')

const enabledCount = computed(() => items.value.filter((i) => i.enabled).length)
const countOf = (type: string) => items.value.filter((i) => i.type === type).length
const typeChips = computed<ToolbarChip<TypeFilter>[]>(() => [
  { label: t('fileSource.filterAll'), value: '', count: items.value.length + 1 },
  { label: t('fileSource.localDisk'), value: 'local', count: 1 },
  { label: t('fileSource.typeShort.oss'), value: 'oss', count: countOf('oss') },
  { label: t('fileSource.typeShort.ftp'), value: 'ftp', count: countOf('ftp') },
  { label: t('fileSource.typeShort.webdav'), value: 'webdav', count: countOf('webdav') },
])
const showLocal = computed(() => !keywords.value.trim() && (!typeFilter.value || typeFilter.value === 'local'))
const filtered = computed(() =>
  typeFilter.value === 'local' ? [] : items.value.filter((i) => !typeFilter.value || i.type === typeFilter.value),
)

// ---- 展示 ----
const VISUAL: Record<string, { icon: string; tone: string }> = {
  oss: { icon: 'i-lucide-cloud', tone: 'p' },
  ftp: { icon: 'i-lucide-arrow-left-right', tone: 'ok' },
  webdav: { icon: 'i-lucide-globe', tone: 'a2' },
}
const visual = (type: string) => VISUAL[type] ?? { icon: 'i-lucide-database', tone: 'a2' }
const typeName = (type: string) => (VISUAL[type] ? t(`fileSource.typeShort.${type}`) : type)
const subLine = (row: FileSourceInfo) => {
  const c = row.config
  const extra = row.type === 'oss' ? c?.region : c?.username
  return [typeName(row.type), extra].filter(Boolean).join(' · ')
}
const addressOf = (row: FileSourceInfo) => {
  const c = row.config
  if (!c) return ''
  if (row.type === 'oss') {
    if (!c.endpoint) return ''
    return /^https?:\/\//.test(c.endpoint) ? c.endpoint : `${c.useSsl ? 'https' : 'http'}://${c.endpoint}`
  }
  if (row.type === 'ftp') return c.host ? `${c.tls ? 'ftps' : 'ftp'}://${c.host}${c.port ? `:${c.port}` : ''}` : ''
  return c.url || ''
}
const rootOf = (row: FileSourceInfo) => {
  const c = row.config
  if (row.type === 'oss') return `bucket: ${c?.bucket || '—'}${c?.prefix ? ` / ${c.prefix}` : ''}`
  return c?.basePath || '/'
}
const hasRedirect = (row: FileSourceInfo) => !!row.caps?.presign && !!row.redirect302
const localCaps: FileSourceCaps = { presign: false, copy: true, move: true, compress: true, resumableUpload: true }
const capsOf = (caps?: FileSourceCaps) => [
  { key: 'presign', label: t('fileSource.capPresign'), icon: 'i-lucide-key-round', on: !!caps?.presign },
  { key: 'copy', label: t('fileSource.capCopyMove'), icon: 'i-lucide-copy', on: !!caps?.copy && !!caps?.move },
  { key: 'compress', label: t('fileSource.capCompress'), icon: 'i-lucide-archive', on: !!caps?.compress },
  { key: 'resumable', label: t('fileSource.capResumable'), icon: 'i-lucide-puzzle', on: !!caps?.resumableUpload },
]

// 相对时间：刚刚 / n 分钟前 / n 小时前 / 日期
const relTime = (value: unknown) => {
  const d = new Date(value as string)
  const ms = Date.now() - d.getTime()
  if (Number.isNaN(ms)) return ''
  const min = Math.floor(ms / 60000)
  if (min < 1) return t('fileSource.testedJustNow')
  if (min < 60) return t('fileSource.minutesAgo', { n: min })
  const h = Math.floor(min / 60)
  if (h < 24) return t('fileSource.hoursAgo', { n: h })
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
type ResultView = { tone: 'ok' | 'bad' | 'dim'; icon: string; title: string; sub: string }
const result = (row: FileSourceInfo): ResultView => {
  if (testingIds.value.has(row.id)) {
    return { tone: 'dim', icon: 'i-lucide-loader-circle', title: t('fileSource.testing'), sub: addressOf(row) }
  }
  if (!row.enabled) return { tone: 'dim', icon: 'i-lucide-minus', title: t('fileSource.disabled'), sub: t('fileSource.disabledHint') }
  const last = row.lastTest
  if (!last) return { tone: 'dim', icon: 'i-lucide-circle-dashed', title: t('fileSource.notTested'), sub: t('fileSource.notTestedHint') }
  const when = t('fileSource.lastTested', { time: relTime(last.testedAt) })
  if (last.ok) {
    return { tone: 'ok', icon: 'i-lucide-check', title: t('fileSource.latency', { ms: Number(last.latencyMs) || 1 }), sub: when }
  }
  return { tone: 'bad', icon: 'i-lucide-circle-alert', title: t('fileSource.testFailed'), sub: `${last.message || ''} · ${when}` }
}

const emptyConfig = (): FileSourceConfig => ({
  endpoint: '',
  region: '',
  bucket: '',
  accessKey: '',
  secretKey: '',
  prefix: '',
  useSsl: true,
  pathStyle: false,
  host: '',
  port: 21,
  username: '',
  password: '',
  basePath: '',
  tls: false,
  url: '',
})

const rowMenu = computed<ActionMenuItem[]>(() => [
  ...(canBrowse.value ? [{ key: 'browse', label: t('fileSource.browse'), icon: 'i-lucide-folder-open' }] : []),
  { key: 'delete', label: t('system.common.delete'), icon: 'i-lucide-trash-2', danger: true },
])
const onRowAction = (key: string, row: FileSourceInfo) => {
  if (key === 'browse') router.push('/file/index')
  else if (key === 'delete') confirmDelete(row)
}

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listFileSourcesRequest({ keywords: keywords.value.trim() || undefined })
    items.value = data.items ?? []
  } catch {
    items.value = []
  } finally {
    loading.value = false
  }
}
const search = () => getList()

const openCreate = () => createRef.value?.showDialog()
const openEdit = (row: FileSourceInfo) => createRef.value?.showDialog(row)

// ---- 测试：单个 / 全部（结果由后端回写为最近一次测试，这里就地更新卡片）----
const testingIds = ref<Set<string>>(new Set())
const setTesting = (id: string, on: boolean) => {
  const next = new Set(testingIds.value)
  if (on) next.add(id)
  else next.delete(id)
  testingIds.value = next
}
const runTest = async (row: FileSourceInfo) => {
  setTesting(row.id, true)
  try {
    const { data } = await testFileSourceRequest({ id: row.id, type: '', config: undefined })
    const last: FileSourceTestResult = {
      ok: data.ok,
      latencyMs: Number(data.latencyMs) || 0,
      message: data.message,
      testedAt: data.testedAt ?? new Date(),
    }
    row.lastTest = last
    return data.ok
  } catch {
    return false
  } finally {
    setTesting(row.id, false)
  }
}
const testOne = async (row: FileSourceInfo) => {
  const ok = await runTest(row)
  if (ok) feedback.success(t('fileSource.testOk'))
  else feedback.error(row.lastTest?.message || t('fileSource.testFailed'))
}
const testingAll = ref(false)
const testAll = async () => {
  testingAll.value = true
  try {
    const targets = items.value.filter((i) => i.enabled)
    const results = await Promise.all(targets.map((row) => runTest(row)))
    const ok = results.filter(Boolean).length
    feedback.info(t('fileSource.testAllDone', { ok, total: targets.length }))
  } finally {
    testingAll.value = false
  }
}

const toggleEnabled = async (row: FileSourceInfo, enabled: boolean) => {
  try {
    await updateFileSourceRequest({
      id: row.id,
      name: row.name,
      enabled,
      redirect302: row.redirect302,
      config: { ...emptyConfig(), ...(row.config ?? {}), secretKey: '', password: '' },
    })
    row.enabled = enabled
    feedback.success(t('fileSource.updateSuccess'))
  } catch {
    /* interceptor */
  }
}

const confirmDelete = (row: FileSourceInfo) => {
  Dialog.confirm({
    content: t('fileSource.confirmDelete', { name: row.name }),
    onConfirm: async () => {
      await deleteFileSourceRequest(row.id)
      feedback.success(t('fileSource.deleteSuccess'))
      getList()
    },
  })
}

onMounted(getList)
</script>

<style scoped lang="scss">
.src {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.src-ok {
  color: var(--m-ok);
  font-weight: 700;
}
.src-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(330px, 1fr));
  gap: 16px;
}
.src-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: 18px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
}
.src-card.is-off .src-card__name {
  color: var(--el-text-color-secondary);
}
.src-card.is-off .src-card__sticker {
  filter: grayscale(0.7);
  opacity: 0.7;
}
.src-card__head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.src-card__sticker {
  width: 50px;
  height: 50px;
  border-radius: 17px;
}
.src-card__sticker :deep(svg),
.src-card__sticker :deep(.iconify) {
  width: 24px;
  height: 24px;
}
.src-card__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.src-card__name {
  font-size: 1.0625rem;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.src-card__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.src-card__info {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
}
.src-card__line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  font-size: 0.75rem;
  color: var(--el-text-color-primary);
}
.src-card__line span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.src-card__line :deep(svg),
.src-card__line :deep(.iconify) {
  flex: none;
  width: 14px;
  height: 14px;
  color: var(--el-text-color-secondary);
}
.src-card__line--dim {
  color: var(--el-text-color-secondary);
}
.src-caps {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.src-cap {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 26px;
  padding: 0 10px;
  border-radius: 999px;
  background: var(--m-s2);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 600;
  opacity: 0.75;
}
.src-cap :deep(svg),
.src-cap :deep(.iconify) {
  width: 13px;
  height: 13px;
}
.src-cap.is-on {
  background: var(--m-okSoft);
  color: var(--m-ok);
  opacity: 1;
}
.src-cap.is-302 {
  background: var(--m-aSoft);
  color: var(--m-aOn);
  opacity: 1;
}
:global(html.dark) .src-cap.is-302 {
  color: var(--m-a);
}
.src-card__foot {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: auto;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.src-res {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}
.src-res__icon {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 10px;
  background: var(--m-s2);
  color: var(--el-text-color-secondary);
}
.src-res__icon :deep(svg),
.src-res__icon :deep(.iconify) {
  width: 15px;
  height: 15px;
}
.src-res__icon.is-ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.src-res__icon.is-bad {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.is-spin {
  animation: src-spin 0.8s linear infinite;
}
@keyframes src-spin {
  to {
    transform: rotate(360deg);
  }
}
.src-res__text {
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.src-res__title {
  font-size: 0.8125rem;
  font-weight: 800;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.src-res__title.is-ok {
  color: var(--m-ok);
}
.src-res__title.is-bad {
  color: var(--m-bad);
}
.src-res__sub {
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.src-card--add {
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 220px;
  border: 1.5px dashed var(--el-border-color);
  background: transparent;
  text-align: center;
  cursor: pointer;
}
.src-card--add:hover {
  border-color: var(--m-p);
}
.src-card--add .src-card__sub {
  white-space: normal;
}
.src-card--empty {
  grid-column: 1 / -1;
}
.src-skeleton {
  height: 260px;
  border-radius: var(--app-radius-lg);
  background: linear-gradient(100deg, var(--m-s2) 30%, var(--el-fill-color-light) 50%, var(--m-s2) 70%);
  background-size: 200% 100%;
  animation: src-shimmer 1.4s ease-in-out infinite;
}
@keyframes src-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
@media (width <= 768px) {
  .src {
    gap: 12px;
  }
  .src-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: 12px;
  }
  .src-card {
    padding: 16px;
  }
  .src-card__foot :deep(.action-menu__trigger) {
    width: 44px;
    height: 44px;
  }
}
</style>
