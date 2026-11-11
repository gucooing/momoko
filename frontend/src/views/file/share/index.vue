<!-- 文件分享：页头(总数/有效数[/本月下载] + 去文件管理/新建) + 统计卡(有效链接 · 本月下载 · 3 天内到期 · 302 直链占比) +
     状态胶囊(带数量)/搜索 + 表格(内容贴纸 · 公开链接复制 · 提取码 · 过期剩余 · 下载进度 · 状态 · 二维码/编辑/删除/⋯) /
     手机卡片(信息格 + 类型大小标签 + 小图标操作) + 「显示 N 个 · 过期 30 天后自动清理」+ 分页 + 悬浮批量条；手机右下悬浮「新建分享」。
     状态统计、本月下载与 302 次数由后端 ListShares 返回（stats）；过期满 30 天的分享由后台任务自动清理。 -->
<template>
  <div class="shr">
    <PageHeader :title="t('file.share.pageTitle')">
      <template #description>
        <span class="m-num">{{ isMobile ? t('file.share.summaryTotalShort', { total: stats.total }) : t('file.share.summaryTotal', { total: stats.total }) }}</span>
        ·
        <span class="shr-ok m-num">{{ t('file.share.summaryValid', { valid: stats.active }) }}</span>
        <template v-if="isMobile">
          · <span class="m-num">{{ t('file.share.summaryMonth', { n: stats.monthDownloads }) }}</span>
        </template>
      </template>
      <template v-if="!isMobile" #actions>
        <UButton v-if="canGoFiles" color="neutral" variant="outline" icon="i-lucide-folder" :to="pageTabTo('file', 'index')">
          {{ t('file.share.goFiles') }}
        </UButton>
        <UButton color="primary" icon="i-lucide-plus" @click="openCreate">
          {{ t('file.share.create') }}
        </UButton>
      </template>
    </PageHeader>
    <FileSubNav />

    <!-- 统计卡（手机端并入页头一句话） -->
    <div v-if="!isMobile" class="shr-stats">
      <div v-for="(card, i) in statCards" :key="card.key" class="shr-stat">
        <span class="shr-stat__sticker m-sticker" :class="`m-sticker--${card.tone}`" :style="{ transform: `rotate(${ROTS[i]})` }">
          <UIcon :name="card.icon" />
        </span>
        <div class="shr-stat__text">
          <div class="shr-stat__label">{{ card.label }}</div>
          <div class="shr-stat__value m-num">
            {{ card.value }}<span v-if="card.unit" class="shr-stat__unit">{{ card.unit }}</span>
          </div>
          <div v-if="card.hint" class="shr-stat__hint m-num">{{ card.hint }}</div>
        </div>
      </div>
    </div>

    <ListToolbar
      v-model:status="status"
      v-model:keyword="keywords"
      :chips="statusChips"
      :chips-label="t('system.common.status')"
      :placeholder="t('file.share.searchPlaceholder')"
      @search="search"
    />

    <!-- 桌面：表格 -->
    <div v-if="!isMobile" class="shr-card shr-tablewrap">
      <table class="shr-table">
        <thead>
          <tr>
            <th class="shr-th--check">
              <input
                type="checkbox"
                class="shr-cb"
                :checked="allSelected"
                :indeterminate.prop="someSelected"
                :aria-label="t('fileManager.selectAll')"
                @change="toggleAll"
              />
            </th>
            <th>{{ t('file.share.content') }}</th>
            <th>{{ t('file.share.link') }}</th>
            <th>{{ t('file.share.code') }}</th>
            <th>{{ t('file.share.expires') }}</th>
            <th>{{ t('file.share.downloads') }}</th>
            <th>{{ t('system.common.status') }}</th>
            <th class="shr-th--ops">{{ t('system.common.operation') }}</th>
          </tr>
        </thead>
        <tbody v-if="loading && !items.length">
          <tr v-for="i in 4" :key="i">
            <td colspan="8"><div class="shr-skeleton" /></td>
          </tr>
        </tbody>
        <tbody v-else-if="items.length">
          <tr v-for="(row, i) in items" :key="row.id" :class="{ 'is-selected': selected.has(row.id) }">
            <td class="shr-td--check">
              <input
                type="checkbox"
                class="shr-cb"
                :checked="selected.has(row.id)"
                :aria-label="t('fileManager.selectItem', { name: row.name })"
                @change="toggleOne(row.id)"
              />
            </td>
            <td class="shr-td--content">
              <div class="shr-content">
                <span class="shr-sticker m-sticker" :class="`m-sticker--${visual(row).tone}`" :style="{ transform: `rotate(${ROTS[i % 4]})` }">
                  <UIcon :name="visual(row).icon" />
                </span>
                <span class="shr-content__text">
                  <span class="shr-content__name" :title="fullPaths(row)">{{ row.name || firstName(row) }}</span>
                  <span class="shr-content__meta m-num">{{ contentMeta(row) }}</span>
                </span>
              </div>
            </td>
            <td>
              <div class="shr-link">
                <span class="shr-link__text m-mono" :class="{ 'is-dim': statusOf(row) !== 'active' }">{{ shortLink(row) }}</span>
                <AppIconButton
                  :icon="copiedId === row.id ? 'HOutline:CheckIcon' : 'HOutline:DocumentDuplicateIcon'"
                  :class="{ 'is-copied': copiedId === row.id }"
                  :label="t('file.share.copyLink')"
                  :box="36"
                  :size="16"
                  @click="copyLink(row)"
                />
              </div>
            </td>
            <td>
              <span v-if="row.code" class="shr-code m-mono">{{ row.code }}</span>
              <span v-else class="shr-dim">{{ t('file.share.noCode') }}</span>
            </td>
            <td>
              <div class="shr-exp m-num" :title="row.expiresAt ? formatDateTime(row.expiresAt) : undefined">
                {{ row.expiresAt ? formatDateTime(row.expiresAt).slice(0, 16) : t('file.share.never') }}
              </div>
              <div class="shr-left" :class="`is-${leftTone(row)}`">{{ leftText(row) }}</div>
            </td>
            <td class="shr-td--dl">
              <div class="shr-dl m-num">
                {{ Number(row.downloadCount || 0) }}
                <span class="shr-dl__limit">/ {{ Number(row.maxDownloads) > 0 ? row.maxDownloads : t('file.share.unlimited') }}</span>
              </div>
              <div class="shr-bar"><span :style="barStyle(row)" /></div>
            </td>
            <td><StatusPill :variant="statusVariant(row)" :label="statusLabel(row)" /></td>
            <td>
              <div class="shr-ops">
                <AppIconButton icon="HOutline:QrCodeIcon" :label="t('file.share.qrOf', { name: row.name })" :box="36" :size="18" @click="openQr(row)" />
                <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('file.share.editOf', { name: row.name })" :box="36" :size="18" @click="openEdit(row)" />
                <AppIconButton class="tone-bad" icon="HOutline:TrashIcon" :label="t('file.share.deleteOf', { name: row.name })" :box="36" :size="18" @click="remove(row)" />
                <ActionMenu :items="rowMenu(row)" :label="t('fileManager.moreFor', { name: row.name })" @select="(key) => onRowAction(key, row)" />
              </div>
            </td>
          </tr>
        </tbody>
      </table>
      <EmptyState
        v-if="!loading && !items.length"
        icon="HOutline:ShareIcon"
        :title="keywords || status ? t('system.common.noData') : t('file.share.emptyDesc')"
        :description="keywords || status ? '' : t('file.share.emptyHint')"
      >
        <template v-if="!keywords && !status" #action>
          <UButton color="primary" icon="i-lucide-plus" @click="openCreate">{{ t('file.share.create') }}</UButton>
        </template>
      </EmptyState>
    </div>

    <!-- 手机：卡片 -->
    <template v-else>
      <div v-if="loading && !items.length" class="shr-cards">
        <div v-for="i in 3" :key="i" class="shr-skeleton shr-skeleton--card" />
      </div>
      <div v-else-if="!items.length" class="shr-card">
        <EmptyState
          icon="HOutline:ShareIcon"
          :title="keywords || status ? t('system.common.noData') : t('file.share.emptyDesc')"
          :description="keywords || status ? '' : t('file.share.emptyHint')"
        />
      </div>
      <div v-else class="shr-cards">
        <article
          v-for="(row, i) in items"
          :key="row.id"
          class="shr-card shr-mcard"
          :class="{ 'is-selected': selected.has(row.id) }"
        >
          <header class="shr-mcard__head">
            <span class="shr-sticker shr-sticker--lg m-sticker" :class="`m-sticker--${visual(row).tone}`" :style="{ transform: `rotate(${ROTS[i % 4]})` }">
              <UIcon :name="visual(row).icon" />
            </span>
            <span class="shr-content__text">
              <span class="shr-content__name">{{ row.name || firstName(row) }}</span>
              <span class="shr-content__meta m-num">{{ contentMeta(row) }}</span>
            </span>
            <StatusPill :variant="statusVariant(row)" :label="statusLabel(row)" />
          </header>
          <div class="shr-mcard__link">
            <UIcon name="i-lucide-link" class="shr-mcard__link-ico" />
            <span class="shr-link__text m-mono" :class="{ 'is-dim': statusOf(row) !== 'active' }">{{ shortLink(row) }}</span>
            <AppIconButton
              :icon="copiedId === row.id ? 'HOutline:CheckIcon' : 'HOutline:DocumentDuplicateIcon'"
              :class="{ 'is-copied': copiedId === row.id }"
              :label="t('file.share.copyLink')"
              :box="44"
              :size="18"
              @click="copyLink(row)"
            />
          </div>
          <dl class="shr-mcard__info">
            <div>
              <dt>{{ t('file.share.code') }}</dt>
              <dd>
                <span v-if="row.code" class="shr-code m-mono">{{ row.code }}</span>
                <span v-else class="shr-dim shr-dim--b">{{ t('file.share.noCodeShort') }}</span>
              </dd>
            </div>
            <div>
              <dt>{{ t('file.share.expires') }}</dt>
              <dd class="m-num">{{ row.expiresAt ? shortDate(row.expiresAt) : t('file.share.never') }}</dd>
              <dd class="shr-left" :class="`is-${leftTone(row)}`">{{ leftText(row) }}</dd>
            </div>
            <div>
              <dt>{{ t('file.share.downloads') }}</dt>
              <dd class="m-num">
                {{ Number(row.downloadCount || 0) }}
                <span class="shr-dl__limit">/ {{ Number(row.maxDownloads) > 0 ? row.maxDownloads : t('file.share.unlimited') }}</span>
              </dd>
              <dd><div class="shr-bar"><span :style="barStyle(row)" /></div></dd>
            </div>
          </dl>
          <footer class="shr-mcard__foot">
            <span class="shr-mcard__tags">
              <span class="shr-mcard__tag m-num">{{ kindTag(row) }}</span>
            </span>
            <AppIconButton icon="HOutline:QrCodeIcon" :label="t('file.share.qrOf', { name: row.name })" :box="44" @click="openQr(row)" />
            <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('file.share.editOf', { name: row.name })" :box="44" @click="openEdit(row)" />
            <AppIconButton class="tone-bad" icon="HOutline:TrashIcon" :label="t('file.share.deleteOf', { name: row.name })" :box="44" @click="remove(row)" />
            <ActionMenu :items="rowMenu(row, true)" :label="t('fileManager.moreFor', { name: row.name })" @select="(key) => onRowAction(key, row)" />
          </footer>
        </article>
      </div>
    </template>

    <div v-if="items.length || pagination.total" class="shr-foot">
      <span class="shr-foot__hint m-num">{{ t('file.share.footHint', { n: items.length }) }}</span>
      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="loadList"
      />
    </div>

    <!-- 批量条 -->
    <div v-if="selected.size" class="shr-bulk" role="toolbar" :aria-label="t('file.share.selectedShares', { count: selected.size })">
      <span class="shr-bulk__count m-num">{{ t('file.share.selectedShares', { count: selected.size }) }}</span>
      <button v-for="act in bulkActions" :key="act.key" type="button" class="shr-bulk__btn" :class="{ 'is-danger': act.danger }" :aria-label="act.label" :title="act.label" :disabled="bulkBusy" @click="onBulk(act.key)">
        <UIcon :name="act.icon" />
        <span v-if="!isMobile">{{ act.label }}</span>
      </button>
      <button type="button" class="shr-bulk__btn shr-bulk__btn--x" :aria-label="t('fileManager.cancelSelect')" :title="t('fileManager.cancelSelect')" @click="selected = new Set()">
        <UIcon name="i-lucide-x" />
      </button>
    </div>

    <!-- 手机：悬浮新建 -->
    <button v-if="isMobile && !selected.size" type="button" class="shr-fab" @click="openCreate">
      <UIcon name="i-lucide-plus" />
      {{ t('file.share.create') }}
    </button>

    <ShareFormDialog v-model="formOpen" :share="editing" @saved="loadList" />

    <!-- 二维码 -->
    <FormDialog v-model="qr.open" :title="t('file.share.qrTitle')" :width="380">
      <div class="shr-qr">
        <div class="shr-qr__img">
          <img v-if="qr.src" :src="qr.src" :alt="t('file.share.qrOf', { name: qr.name })" />
          <span v-else class="shr-qr__loading" />
        </div>
        <div class="shr-qr__name">{{ qr.name }}</div>
        <div class="shr-qr__link m-mono">{{ qr.link }}</div>
        <div v-if="qr.code" class="shr-qr__code">
          {{ t('file.share.code') }} <span class="shr-code m-mono">{{ qr.code }}</span>
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" icon="i-lucide-download" :disabled="!qr.src" @click="downloadQr">
          {{ t('file.share.saveQr') }}
        </UButton>
        <UButton color="primary" icon="i-lucide-copy" @click="qr.row && copyLink(qr.row)">{{ t('file.share.copyLink') }}</UButton>
        <UButton color="neutral" variant="ghost" @click="close">{{ t('system.common.close') }}</UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { formatDateTime, copyTextToClipboard, getBaseName, formatFileSize, resolveFilePreviewKind } from '@/utils/file'
import { Dialog } from '@/utils/dialog'
import {
  listSharesRequest,
  deleteShareRequest,
  updateShareRequest,
  buildShareLink,
  getShareQRCodeRequest,
} from '@/api/share'
import { listFileSourcesRequest } from '@/api/fileSource'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import { pageTabTo } from '@/config/pageGroups'
import FileSubNav from '@/views/file/components/FileSubNav.vue'
import ShareFormDialog from '@/components/share/ShareFormDialog.vue'
import type { ShareInfo, ShareItem, ShareStats } from '@/types/v1/file'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'

defineOptions({ name: 'FileShareView' })

const { t } = useI18n()
const menuStore = useMenuStore()
const isMobile = computed(() => menuStore.isMobile)
const accessible = useAccessibleRoutes()
const canGoFiles = computed(() => accessible.value.has('/file/index'))
const ROTS = ['-4deg', '3deg', '-2deg', '4deg']

type ShareStatus = '' | 'active' | 'expired' | 'exhausted' | 'disabled'

const items = ref<ShareInfo[]>([])
const loading = ref(false)
const keywords = ref('')
const status = ref<ShareStatus>('')
const pagination = ref({ page: 1, pageSize: 20, total: 0 })
const emptyStats = (): ShareStats => ({
  total: 0,
  active: 0,
  expired: 0,
  exhausted: 0,
  disabled: 0,
  expiringSoon: 0,
  monthDownloads: 0,
  monthRedirectDownloads: 0,
})
const stats = ref<ShareStats>(emptyStats())

const formOpen = ref(false)
const editing = ref<ShareInfo | null>(null)

// 来源名（分享条目可跨来源）：本地磁盘 + 已配置来源。读取失败时只显示本地/类型。
const sourceNames = ref<Record<string, string>>({})
const loadSources = async () => {
  try {
    const { data } = await listFileSourcesRequest()
    sourceNames.value = Object.fromEntries((data.items ?? []).map((s) => [s.id, s.name]))
  } catch {
    sourceNames.value = {}
  }
}

// ---- 展示 ----
const itemsOf = (row: ShareInfo) => (row.items as ShareItem[] | undefined) || []
const itemCount = (row: ShareInfo) => itemsOf(row).length
const fullPaths = (row: ShareInfo) => itemsOf(row).map((i) => i.path).join('\n')
const firstName = (row: ShareInfo) => {
  const first = itemsOf(row)[0]
  if (!first) return '—'
  return first.name || getBaseName(first.path) || first.path
}
const sourceName = (row: ShareInfo) => {
  const ids = Array.from(new Set(itemsOf(row).map((i) => i.sourceId || '')))
  if (!ids.length) return t('file.share.localSource')
  const names = ids.map((id) => (id ? sourceNames.value[id] || t('fileSource.title') : t('file.share.localSource')))
  return names.join(' + ')
}
// 总大小：目录条目为其下文件的递归总大小（创建时统计并缓存）
const totalSize = (row: ShareInfo) => itemsOf(row).reduce((sum, i) => sum + (Number(i.size) || 0), 0)
const pad2 = (n: number) => String(n).padStart(2, '0')
const shortDate = (value: unknown) => {
  const d = new Date(value as string)
  if (Number.isNaN(d.getTime())) return '—'
  const md = `${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`
  return d.getFullYear() === new Date().getFullYear() ? md : `${d.getFullYear()}-${md}`
}
const contentMeta = (row: ShareInfo) => {
  const parts = [sourceName(row)]
  const n = itemCount(row)
  if (n > 1) parts.push(t('file.share.itemsCount', { count: n }))
  const size = totalSize(row)
  if (size) parts.push(formatFileSize(size))
  parts.push(t('file.share.createdAt', { time: shortDate(row.createTime) }))
  return parts.join(' · ')
}
const visual = (row: ShareInfo): { icon: string; tone: string } => {
  const list = itemsOf(row)
  if (list.length > 1) return { icon: 'i-lucide-layers', tone: 'p' }
  const first = list[0]
  if (!first || first.isDir) return { icon: 'i-lucide-folder', tone: 'a' }
  const kind = resolveFilePreviewKind(first.name || first.path)
  if (kind === 'image') return { icon: 'i-lucide-image', tone: 'ok' }
  if (kind === 'video') return { icon: 'i-lucide-film', tone: 'p' }
  if (kind === 'audio') return { icon: 'i-lucide-music', tone: 'a2' }
  if (/\.(zip|tar|gz|tgz|rar|7z)$/i.test(first.path)) return { icon: 'i-lucide-file-archive', tone: 'warn' }
  return { icon: 'i-lucide-file-text', tone: 'a2' }
}
const shortLink = (row: ShareInfo) => buildShareLink(row.token).replace(/^https?:\/\//, '')
// 手机卡片底部类型标签：「文件夹 · 14 个文件 · 1.36 GB」/「压缩包 · 1.21 GB」/「3 项 · …」
const KIND_KEY: Record<string, string> = {
  'i-lucide-folder': 'file.share.kindFolder',
  'i-lucide-layers': 'file.share.kindMulti',
  'i-lucide-image': 'file.share.kindImage',
  'i-lucide-film': 'file.share.kindVideo',
  'i-lucide-music': 'file.share.kindAudio',
  'i-lucide-file-archive': 'file.share.kindArchive',
  'i-lucide-file-text': 'file.share.kindFile',
}
const kindTag = (row: ShareInfo) => {
  const list = itemsOf(row)
  const parts = [t(KIND_KEY[visual(row).icon] ?? 'file.share.kindFile', { count: list.length })]
  const files = list.reduce((sum, i) => (sum >= 0 && Number(i.fileCount) >= 0 ? sum + Number(i.fileCount) : -1), 0)
  const hasDir = list.some((i) => i.isDir)
  if (hasDir && files > 0) parts.push(t('file.share.filesCount', { n: files }))
  const size = list.reduce((sum, i) => sum + (Number(i.size) || 0), 0)
  if (size) parts.push(formatFileSize(size))
  return parts.join(' · ')
}

// ---- 状态（与后端统计口径一致：停用 > 过期 > 次数用完 > 有效）----
const isExpired = (row: ShareInfo) => !!row.expiresAt && new Date(row.expiresAt as unknown as string).getTime() <= Date.now()
const isExhausted = (row: ShareInfo) =>
  Number(row.maxDownloads) > 0 && Number(row.downloadCount) >= Number(row.maxDownloads)
const statusOf = (row: ShareInfo): Exclude<ShareStatus, ''> => {
  if (!row.enabled) return 'disabled'
  if (isExpired(row)) return 'expired'
  if (isExhausted(row)) return 'exhausted'
  return 'active'
}
type PillVariant = 'success' | 'warning' | 'neutral' | 'error'
const statusVariant = (row: ShareInfo): PillVariant =>
  ({ active: 'success', expired: 'neutral', exhausted: 'warning', disabled: 'neutral' } as const)[statusOf(row)]
const statusLabel = (row: ShareInfo) =>
  ({
    active: t('file.share.statusValid'),
    expired: t('file.share.statusExpired'),
    exhausted: t('file.share.statusExhausted'),
    disabled: t('file.share.statusDisabled'),
  })[statusOf(row)]

const leftMs = (row: ShareInfo) =>
  row.expiresAt ? new Date(row.expiresAt as unknown as string).getTime() - Date.now() : Infinity
const leftText = (row: ShareInfo) => {
  const ms = leftMs(row)
  if (ms === Infinity) return t('file.share.forever')
  if (ms <= 0) return t('file.share.expiredLabel')
  const hours = Math.floor(ms / 3600000)
  if (hours < 1) return t('file.share.leftSoon')
  if (hours < 48) return t('file.share.leftHours', { n: hours })
  return t('file.share.leftDays', { n: Math.floor(hours / 24) })
}
const leftTone = (row: ShareInfo) => {
  const ms = leftMs(row)
  if (ms <= 0) return 'bad'
  if (ms < 3 * 86400000) return 'warn'
  return 'dim'
}
const barStyle = (row: ShareInfo) => {
  const max = Number(row.maxDownloads) || 0
  if (!max) return { width: '100%', background: 'var(--m-line)' }
  const pct = Math.min(100, Math.round((Number(row.downloadCount || 0) / max) * 100))
  const bg = pct >= 100 ? 'var(--m-bad)' : pct >= 80 ? 'var(--m-warn)' : 'var(--m-p)'
  return { width: `${pct}%`, background: bg }
}

// ---- 统计卡 / 筛选胶囊 ----
// 302 直链占比：本月下载中由存储端直接下发的比例（本月无下载为 —）
const redirectRatio = computed(() => {
  const month = Number(stats.value.monthDownloads) || 0
  return month ? `${Math.round(((Number(stats.value.monthRedirectDownloads) || 0) / month) * 100)}%` : '—'
})
const statCards = computed(() => [
  { key: 'active', label: t('file.share.statValid'), value: stats.value.active, icon: 'i-lucide-link-2', tone: 'p', unit: '', hint: '' },
  { key: 'month', label: t('file.share.statMonth'), value: stats.value.monthDownloads, icon: 'i-lucide-download', tone: 'a2', unit: t('file.share.times'), hint: '' },
  { key: 'soon', label: t('file.share.statSoon'), value: stats.value.expiringSoon, icon: 'i-lucide-clock', tone: 'a', unit: t('file.share.unitCount'), hint: '' },
  { key: 'redirect', label: t('file.share.statRedirect'), value: redirectRatio.value, icon: 'i-lucide-zap', tone: 'ok', unit: '', hint: '' },
])
const statusChips = computed<ToolbarChip<ShareStatus>[]>(() => [
  { label: t('file.share.filterAll'), value: '', count: Number(stats.value.total) },
  { label: t('file.share.statusValid'), value: 'active', count: Number(stats.value.active) },
  { label: t('file.share.statusExpired'), value: 'expired', count: Number(stats.value.expired) },
  { label: t('file.share.statusExhausted'), value: 'exhausted', count: Number(stats.value.exhausted) },
  { label: t('file.share.statusDisabled'), value: 'disabled', count: Number(stats.value.disabled) },
])

// ---- 加载 ----
const loadList = async () => {
  loading.value = true
  try {
    const { data } = await listSharesRequest({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: keywords.value.trim() || undefined,
      status: status.value || undefined,
    })
    items.value = data?.items || []
    // int64 经 protobuf-JSON 可能为字符串，统一转 Number。
    pagination.value.total = Number(data?.total || 0)
    const s = data?.stats
    stats.value = s
      ? {
          total: Number(s.total || 0),
          active: Number(s.active || 0),
          expired: Number(s.expired || 0),
          exhausted: Number(s.exhausted || 0),
          disabled: Number(s.disabled || 0),
          expiringSoon: Number(s.expiringSoon || 0),
          monthDownloads: Number(s.monthDownloads || 0),
          monthRedirectDownloads: Number(s.monthRedirectDownloads || 0),
        }
      : emptyStats()
    const ids = new Set(items.value.map((i) => i.id))
    selected.value = new Set([...selected.value].filter((id) => ids.has(id)))
  } catch {
    items.value = []
    pagination.value.total = 0
  } finally {
    loading.value = false
  }
}
const search = () => {
  pagination.value.page = 1
  loadList()
}

const openCreate = () => {
  editing.value = null
  formOpen.value = true
}
const openEdit = (row: ShareInfo) => {
  editing.value = row
  formOpen.value = true
}

// 全量覆盖可编辑字段的更新（items 为空=不改内容）。
const patchShare = (row: ShareInfo, patch: Partial<Pick<ShareInfo, 'enabled' | 'expiresAt'>>) =>
  updateShareRequest({
    id: row.id,
    name: row.name,
    code: row.code,
    expiresAt: 'expiresAt' in patch ? patch.expiresAt : row.expiresAt,
    maxDownloads: row.maxDownloads,
    enabled: 'enabled' in patch ? !!patch.enabled : row.enabled,
    items: [],
  })

const toggleEnabled = async (row: ShareInfo) => {
  try {
    await patchShare(row, { enabled: !row.enabled })
    loadList()
  } catch {
    /* interceptor */
  }
}

const copiedId = ref('')
let copiedTimer: ReturnType<typeof setTimeout> | undefined
const copyLink = async (row: ShareInfo) => {
  try {
    await copyTextToClipboard(buildShareLink(row.token))
    feedback.success(t('file.share.copied'))
    copiedId.value = row.id
    clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => (copiedId.value = ''), 1600)
  } catch {
    feedback.error(t('file.share.copyFailed'))
  }
}
const openLink = (row: ShareInfo) => window.open(buildShareLink(row.token), '_blank', 'noopener')

const remove = (row: ShareInfo) => {
  Dialog.confirm({
    content: t('file.share.confirmDelete'),
    onConfirm: async () => {
      await deleteShareRequest(row.id)
      feedback.success(t('system.common.deleteSuccess'))
      loadList()
    },
  })
}

// 「⋯」：启停（停 红 / 启 绿 区分）、复制/打开链接、延长、选择
const rowMenu = (row: ShareInfo, mobile = false): ActionMenuItem[] => [
  {
    key: 'toggle',
    label: row.enabled ? t('file.share.disable') : t('file.share.enable'),
    icon: row.enabled ? 'i-lucide-circle-pause' : 'i-lucide-circle-play',
    danger: row.enabled,
  },
  ...(mobile ? [] : [{ key: 'copy', label: t('file.share.copyLink'), icon: 'i-lucide-copy' }]),
  { key: 'open', label: t('file.share.openLink'), icon: 'i-lucide-external-link' },
  { key: 'extend', label: t('file.share.extend7'), icon: 'i-lucide-calendar-plus', hidden: !row.expiresAt },
  { key: 'select', label: selected.value.has(row.id) ? t('fileManager.cancelSelect') : t('fileManager.select'), icon: 'i-lucide-square-check' },
]
const onRowAction = (key: string, row: ShareInfo) => {
  if (key === 'toggle') toggleEnabled(row)
  else if (key === 'copy') copyLink(row)
  else if (key === 'open') openLink(row)
  else if (key === 'extend') extendShares([row])
  else if (key === 'select') toggleOne(row.id)
}

// ---- 二维码 ----
const qr = reactive({ open: false, src: '', link: '', name: '', code: '', row: null as ShareInfo | null })
const openQr = async (row: ShareInfo) => {
  Object.assign(qr, { open: true, src: '', link: shortLink(row), name: row.name || firstName(row), code: row.code, row })
  try {
    const res = await getShareQRCodeRequest(row.id)
    if (qr.row === row) {
      qr.src = res.src
      qr.link = res.link.replace(/^https?:\/\//, '')
    }
  } catch {
    /* interceptor */
  }
}
const downloadQr = () => {
  if (!qr.src) return
  const a = document.createElement('a')
  a.href = qr.src
  a.download = `${qr.name || 'share'}-qrcode.png`
  a.click()
}

// ---- 选择 / 批量 ----
const selected = ref<Set<string>>(new Set())
const allSelected = computed(() => items.value.length > 0 && items.value.every((r) => selected.value.has(r.id)))
const someSelected = computed(() => selected.value.size > 0 && !allSelected.value)
const toggleOne = (id: string) => {
  const next = new Set(selected.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selected.value = next
}
const toggleAll = () => {
  selected.value = allSelected.value ? new Set() : new Set(items.value.map((r) => r.id))
}
const selectedRows = computed(() => items.value.filter((r) => selected.value.has(r.id)))

const bulkActions = computed(() => [
  { key: 'copy', label: t('file.share.copyLink'), icon: 'i-lucide-copy', danger: false },
  { key: 'extend', label: t('file.share.extend7'), icon: 'i-lucide-calendar-plus', danger: false },
  { key: 'disable', label: t('file.share.disable'), icon: 'i-lucide-circle-pause', danger: false },
  { key: 'delete', label: t('system.common.delete'), icon: 'i-lucide-trash-2', danger: true },
])
const bulkBusy = ref(false)
const runBulk = async (rows: ShareInfo[], fn: (row: ShareInfo) => Promise<unknown>) => {
  bulkBusy.value = true
  let done = 0
  try {
    for (const row of rows) {
      try {
        await fn(row)
        done += 1
      } catch {
        /* 单条失败由拦截器提示，继续其余 */
      }
    }
    feedback.success(t('file.share.batchDone', { count: done }))
  } finally {
    bulkBusy.value = false
    loadList()
  }
}
// 延长 7 天：在「当前过期时间与现在」较晚者基础上 +7 天；永久分享不变。
const extendShares = (rows: ShareInfo[]) =>
  runBulk(
    rows.filter((r) => !!r.expiresAt),
    (row) => {
      const base = Math.max(Date.now(), new Date(row.expiresAt as unknown as string).getTime())
      return patchShare(row, { expiresAt: new Date(base + 7 * 86400000) })
    },
  )
const onBulk = async (key: string) => {
  const rows = selectedRows.value
  if (!rows.length) return
  if (key === 'copy') {
    try {
      await copyTextToClipboard(rows.map((r) => buildShareLink(r.token)).join('\n'))
      feedback.success(t('file.share.linksCopied', { count: rows.length }))
    } catch {
      feedback.error(t('file.share.copyFailed'))
    }
  } else if (key === 'extend') {
    extendShares(rows)
  } else if (key === 'disable') {
    runBulk(rows.filter((r) => r.enabled), (row) => patchShare(row, { enabled: false }))
  } else if (key === 'delete') {
    Dialog.confirm({
      content: t('file.share.batchDeleteConfirm', { count: rows.length }),
      onConfirm: async () => {
        await runBulk(rows, (row) => deleteShareRequest(row.id))
        selected.value = new Set()
      },
    })
  }
}

onMounted(() => {
  loadSources()
  loadList()
})
</script>

<style scoped lang="scss">
.shr {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.shr-ok {
  color: var(--m-ok);
  font-weight: 700;
}

/* 统计卡 */
.shr-stats {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 14px;
}
.shr-stat {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: calc(var(--cmp-card-unit) * 16) calc(var(--cmp-card-unit) * 18);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
}
.shr-stat__sticker {
  width: 46px;
  height: 46px;
  border-radius: 15px;
}
.shr-stat__sticker :deep(svg),
.shr-stat__sticker :deep(.iconify) {
  width: 22px;
  height: 22px;
}
.shr-stat__label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.shr-stat__value {
  font-size: 1.625rem;
  font-weight: 800;
  line-height: 1.2;
}
.shr-stat__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.shr-stat__unit {
  margin-left: 4px;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}

/* 卡片外壳 */
.shr-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
}

/* 表格 */
.shr-tablewrap {
  padding: 4px 8px;
  overflow-x: auto;
  scrollbar-width: none;
}
.shr-tablewrap::-webkit-scrollbar {
  display: none;
}
.shr-table {
  width: 100%;
  min-width: 960px;
  border-collapse: separate;
  border-spacing: 0;
}
.shr-table th {
  padding: 12px 10px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--el-text-color-secondary);
  text-align: left;
  white-space: nowrap;
}
.shr-table td {
  padding: 10px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  font-size: 0.875rem;
  vertical-align: middle;
}
.shr-table tbody tr:last-child td {
  border-bottom: none;
}
.shr-table tbody tr:hover td {
  background: var(--m-s2);
}
.shr-table tbody tr.is-selected td {
  background: color-mix(in srgb, var(--m-pSoft) 70%, var(--el-bg-color));
}
.shr-th--check,
.shr-td--check {
  width: 40px;
  text-align: center;
}
.shr-th--ops {
  text-align: right !important;
}
.shr-cb {
  width: 18px;
  height: 18px;
  margin: 0;
  accent-color: var(--m-p);
  cursor: pointer;
}
/* 1440 宽下整表要放得下（操作列不能被挤到横向滚动里）：内容列与原型一致 240px */
.shr-td--content {
  max-width: 240px;
}
.shr-content {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.shr-sticker {
  width: 38px;
  height: 38px;
  border-radius: 13px;
}
.shr-sticker :deep(svg),
.shr-sticker :deep(.iconify) {
  width: 18px;
  height: 18px;
}
.shr-sticker--lg {
  width: 44px;
  height: 44px;
  border-radius: 15px;
}
.shr-content__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.shr-content__name {
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.shr-content__meta {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.shr-link {
  display: flex;
  align-items: center;
  gap: 4px;
}
.shr-link__text {
  font-size: 0.75rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 180px;
}
.shr-link__text.is-dim {
  color: var(--el-text-color-secondary);
}
.is-copied {
  color: var(--m-ok) !important;
}
.shr-code {
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 10px;
  border-radius: 8px;
  background: var(--m-aSoft);
  color: var(--m-aOn);
  font-size: 0.8125rem;
  font-weight: 700;
  letter-spacing: 0.06em;
}
:global(html.dark) .shr-code {
  color: var(--m-a);
}
.shr-dim {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.shr-exp {
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
}
.shr-left {
  margin-top: 2px;
  font-size: 0.75rem;
  white-space: nowrap;
  color: var(--el-text-color-secondary);
}
.shr-left.is-warn {
  color: var(--m-warn);
  font-weight: 700;
}
.shr-left.is-bad {
  color: var(--m-bad);
  font-weight: 700;
}
.shr-td--dl {
  min-width: 110px;
}
.shr-dl {
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
}
.shr-dl__limit {
  color: var(--el-text-color-secondary);
  font-weight: 600;
}
.shr-bar {
  height: 6px;
  margin-top: 6px;
  border-radius: 999px;
  background: var(--m-s2);
  overflow: hidden;
}
.shr-bar span {
  display: block;
  height: 100%;
  border-radius: 999px;
}
.shr-ops {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 2px;
}
.tone-bad,
.shr-ops .tone-bad,
.shr-mcard__foot .tone-bad {
  color: var(--m-bad);
}
.tone-ok {
  color: var(--m-ok);
}
.shr-skeleton {
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(100deg, var(--m-s2) 30%, var(--el-fill-color-light) 50%, var(--m-s2) 70%);
  background-size: 200% 100%;
  animation: shr-shimmer 1.4s ease-in-out infinite;
}
.shr-skeleton--card {
  height: 220px;
  border-radius: var(--app-radius-lg);
}
@keyframes shr-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}

/* 手机卡片 */
.shr-cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.shr-mcard {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 12px 6px 16px;
}
.shr-mcard.is-selected {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.shr-mcard__head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-right: 4px;
}
.shr-mcard__head .shr-content__name {
  font-size: 1rem;
}
.shr-mcard__link {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-right: 4px;
  padding-left: 12px;
  border-radius: 14px;
  background: var(--m-s2);
}
.shr-mcard__link .shr-link__text {
  flex: 1;
  min-width: 0;
  max-width: none;
}
.shr-mcard__link-ico {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.shr-mcard__info {
  display: grid;
  grid-template-columns: 0.8fr 1.1fr 1.1fr;
  gap: 10px;
  margin: 0 4px 0 0;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
}
.shr-mcard__info > div {
  min-width: 0;
}
.shr-mcard__info dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.shr-mcard__info dd {
  margin: 3px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
}
.shr-mcard__info .shr-code {
  height: 24px;
  padding: 0 8px;
}
.shr-mcard__foot {
  display: flex;
  align-items: center;
  gap: 2px;
  padding-top: 6px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.shr-mcard__tag {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--m-s2);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 600;
}
.shr-mcard__tags {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.shr-mcard__tag {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.shr-dim--b {
  font-weight: 700;
}

/* 底部：显示条数 + 分页 */
.shr-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.shr-foot__hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.shr-foot :deep(.pagination) {
  flex: 1;
}

/* 二维码 */
.shr-qr {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  text-align: center;
}
.shr-qr__img {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 240px;
  height: 240px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 18px;
  background: var(--m-s2);
}
.shr-qr__img img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.shr-qr__loading {
  width: 32px;
  height: 32px;
  border: 3px solid var(--m-line);
  border-top-color: var(--m-p);
  border-radius: 50%;
  animation: shr-spin 0.8s linear infinite;
}
@keyframes shr-spin {
  to {
    transform: rotate(360deg);
  }
}
.shr-qr__name {
  font-size: 1rem;
  font-weight: 800;
  word-break: break-all;
}
.shr-qr__link {
  max-width: 100%;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.shr-qr__code {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}

/* 手机悬浮新建 */
.shr-fab {
  position: fixed;
  right: 16px;
  bottom: calc(82px + env(safe-area-inset-bottom));
  z-index: 6;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 56px;
  padding: 0 22px;
  border: none;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-size: 0.9375rem;
  font-weight: 700;
  box-shadow: var(--app-shadow-brand);
  cursor: pointer;
}
.shr-fab :deep(svg),
.shr-fab :deep(.iconify) {
  width: 20px;
  height: 20px;
}
.shr-mcard__foot :deep(.action-menu__trigger) {
  width: 44px;
  height: 44px;
}

/* 批量条 */
.shr-bulk {
  position: sticky;
  bottom: 16px;
  z-index: 5;
  align-self: center;
  display: flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  padding: 6px 6px 6px 20px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: var(--app-shadow-lg);
}
.shr-bulk__count {
  margin-right: 10px;
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
}
.shr-bulk__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 40px;
  min-height: 40px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--m-bg);
  font-size: 0.875rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
}
.shr-bulk__btn:hover {
  background: color-mix(in srgb, var(--m-bg) 14%, transparent);
}
.shr-bulk__btn:disabled {
  opacity: 0.5;
}
.shr-bulk__btn.is-danger {
  color: var(--m-a);
}
.shr-bulk__btn--x {
  padding: 0;
}
.shr-bulk__btn :deep(svg),
.shr-bulk__btn :deep(.iconify) {
  width: 16px;
  height: 16px;
}

@media (width <= 768px) {
  .shr {
    gap: 12px;
    padding-bottom: 72px;
  }
  .shr-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }
  .shr-stat {
    gap: 10px;
    padding: 12px;
  }
  .shr-stat__sticker {
    width: 38px;
    height: 38px;
    border-radius: 12px;
  }
  .shr-stat__value {
    font-size: 1.375rem;
  }
  .shr-bulk {
    position: fixed;
    left: 12px;
    right: 12px;
    bottom: calc(78px + env(safe-area-inset-bottom));
    justify-content: space-between;
    padding-left: 16px;
  }
  .shr-bulk__count {
    margin-right: auto;
  }
  .shr-bulk__btn {
    padding: 0;
  }
}
</style>
