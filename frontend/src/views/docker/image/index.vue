<!-- Docker 镜像（第二期）：标题(引擎状态) + 统计说明 + 清理悬空/拉取 → 子导航 → 进行中的拉取卡 →
     ListToolbar(使用情况 chips 带数量 + 搜索 + 排序) → 桌面表格（可多选批量删除）/ 手机卡片 → 分页。
     行内常用：打标签 / 构建历史 / 删除，其余（详情、编辑标签、用它建容器、复制 ID）收进「⋯」。 -->
<template>
  <div class="dk-page">
    <DockerHeader @tasks="openTasks">
      <!-- 引擎不可用时没有统计，不再显示一句功能介绍，与容器 / 连接配置标签一致 -->
      <template v-if="stats" #description>
        {{ t('docker.image.statsCount', { n: stats.total }) }} · {{ t('docker.image.statsSize') }}
        <strong class="dk-em m-num">{{ formatBytes(stats.totalSize) }}</strong>
        <template v-if="Number(stats.reclaimableSize)">
          · {{ t('docker.image.statsReclaimable') }}
          <strong class="dk-em dk-em--warn m-num">{{ formatBytes(stats.reclaimableSize) }}</strong>
          </template>
      </template>
      <template #actions>
        <UButton
          v-if="canManage && ready"
          color="neutral"
          variant="outline"
          icon="i-lucide-brush-cleaning"
          :disabled="!Number(stats?.dangling)"
          @click="handlePrune"
        >
          {{ t('docker.image.pruneDangling') }}
        </UButton>
        <UButton v-if="canManage && ready" color="neutral" variant="outline" icon="i-lucide-upload" @click="openImport">
          {{ t('docker.imgx.import') }}
        </UButton>
        <UButton v-if="canManage" color="primary" icon="i-lucide-download" :disabled="!ready" @click="openPull">
          {{ t('docker.image.pullImage') }}
        </UButton>
      </template>
    </DockerHeader>

    <DockerSubNav />

    <DockerUnavailable v-if="engineLoaded && !ready" @retry="init(true)" />

    <template v-else>
      <DockerPullTasks ref="pullTasksRef" :can-cancel="canManage" @open="openTask" @finished="onTasksFinished" />

      <ListToolbar
        v-model:status="queryForm.usage"
        v-model:keyword="queryForm.keyword"
        :chips="usageChips"
        :chips-label="t('docker.image.usage')"
        :placeholder="t('docker.image.searchPlaceholder')"
        @search="search"
      >
        <div class="dk-sort">
          <AppSelect v-model="queryForm.sort" :options="sortOptions" fit @update:model-value="search" />
        </div>
      </ListToolbar>

      <div v-if="menuStore.isMobile && stats" class="dk-mstats">
        <span>
          {{ t('docker.image.statsSize') }} <strong class="dk-em m-num">{{ formatBytes(stats.totalSize) }}</strong>
          · {{ t('docker.image.statsReclaimable') }} <strong class="dk-em dk-em--warn m-num">{{ formatBytes(stats.reclaimableSize) }}</strong>
        </span>
        <button v-if="canManage" type="button" class="dk-linkbtn" @click="openImport">
          <UIcon name="i-lucide-upload" />{{ t('docker.imgx.import') }}
        </button>
        <button v-if="canManage" type="button" class="dk-linkbtn" :disabled="!Number(stats.dangling)" @click="handlePrune">
          <UIcon name="i-lucide-brush-cleaning" />{{ t('docker.image.pruneShort') }}
        </button>
      </div>

      <div class="dk-page__body">
        <!-- 桌面：表格 -->
        <DataTable
          v-if="!menuStore.isMobile"
          v-model="selected"
          :columns="columns"
          :rows="list"
          row-key="id"
          :selectable="canManage"
          :loading="loading"
          :empty-text="t('docker.image.noImages')"
        >
          <template #cell-repo="{ row }">
            <div class="dk-repo">
              <span class="dk-repo__name m-mono" :class="{ 'is-none': row.dangling }">{{ repoOf(row) }}</span>
              <div class="dk-tags">
                <span v-for="tag in tagNamesOf(row).slice(0, 3)" :key="tag" class="dk-tag" :class="{ 'dk-tag--none': row.dangling }">{{ tag }}</span>
                <span v-if="tagNamesOf(row).length > 3" class="dk-dim">+{{ tagNamesOf(row).length - 3 }}</span>
              </div>
            </div>
          </template>
          <template #cell-id="{ row }"><span class="m-mono dk-sub">{{ shortId(row.id) }}</span></template>
          <template #cell-size="{ row }"><strong class="m-num">{{ formatBytes(row.size) }}</strong></template>
          <template #cell-created="{ row }">
            <div class="dk-when">
              <span class="m-num">{{ formatAgo(row.created, t) }}</span>
              <span class="m-num dk-sub">{{ formatDate(row.created) }}</span>
            </div>
          </template>
          <template #cell-usage="{ row }">
            <RouterLink v-if="usedByOf(row).length" class="dk-usage" :to="pageTabTo('docker', 'container')">
              <span class="dk-usage__sticker m-sticker m-sticker--ok">
                <component :is="menuStore.iconComponents['HOutline:CubeIcon']" />
              </span>
              <span class="dk-usage__text">
                <strong class="m-num">{{ t('docker.image.usedCount', { n: usedByOf(row).length }) }}</strong>
                <span class="m-mono dk-sub">{{ usedByOf(row).join('、') }}</span>
              </span>
            </RouterLink>
            <StatusPill v-else-if="row.dangling" variant="warning" :label="t('docker.image.usageDangling')" />
            <StatusPill v-else variant="neutral" :label="t('docker.image.usageUnused')" />
          </template>
          <template #cell-operation="{ row }">
            <div class="dk-acts">
              <template v-if="canManage">
                <AppIconButton icon="HOutline:TagIcon" :label="t('docker.image.tagImage')" :box="36" :size="18" @click="onRowAction('tag', row)" />
                <AppIconButton icon="HOutline:ClockIcon" :label="t('docker.image.history')" :box="36" :size="18" @click="onRowAction('history', row)" />
                <AppIconButton
                  class="is-bad"
                  icon="HOutline:TrashIcon"
                  :label="Number(row.containers) ? t('docker.image.inUseNoDelete') : t('docker.common.delete')"
                  :box="36"
                  :size="18"
                  :disabled="!!Number(row.containers)"
                  @click="onRowAction('delete', row)"
                />
              </template>
              <ActionMenu :items="menuActionsFor(row)" @select="(key) => onRowAction(key, row)" />
            </div>
          </template>
        </DataTable>

        <!-- 手机：卡片 -->
        <template v-else>
          <div v-if="loading && !list.length" class="dk-cards">
            <div v-for="i in 3" :key="i" class="dk-skeleton" />
          </div>
          <EmptyState
            v-else-if="!list.length"
            icon="HOutline:PhotoIcon"
            :title="t('docker.image.noImages')"
            :description="t('docker.image.emptyDesc')"
          />
          <div v-else class="dk-cards">
            <article v-for="row in list" :key="row.id" class="dk-card">
              <header class="dk-card__head">
                <span class="dk-card__name m-mono" :class="{ 'is-none': row.dangling }">{{ repoOf(row) }}</span>
                <StatusPill :variant="usageVariant(row)" :label="usageLabel(row)" />
              </header>
              <dl class="dk-card__info">
                <div><dt>{{ t('docker.image.imageId') }}</dt><dd class="m-mono">{{ shortId(row.id) }}</dd></div>
                <div><dt>{{ t('docker.common.size') }}</dt><dd class="m-num">{{ formatBytes(row.size) }}</dd></div>
                <div><dt>{{ t('docker.image.createdOn') }}</dt><dd class="m-num">{{ formatAgo(row.created, t) }}</dd></div>
                <div><dt>{{ t('docker.image.usedBy') }}</dt><dd class="m-mono">{{ usedByOf(row).join('、') || '—' }}</dd></div>
              </dl>
              <footer class="dk-card__foot">
                <div class="dk-card__lead">
                  <span v-for="tag in tagNamesOf(row)" :key="tag" class="dk-tag" :class="{ 'dk-tag--none': row.dangling }">{{ tag }}</span>
                </div>
                <template v-if="canManage">
                  <AppIconButton icon="HOutline:TagIcon" :label="`${t('docker.image.tagImage')} ${labelOf(row)}`" :box="44" @click="onRowAction('tag', row)" />
                  <AppIconButton icon="HOutline:ClockIcon" :label="`${t('docker.image.history')} ${labelOf(row)}`" :box="44" @click="onRowAction('history', row)" />
                  <AppIconButton
                    class="is-bad"
                    icon="HOutline:TrashIcon"
                    :label="Number(row.containers) ? t('docker.image.inUseNoDelete') : `${t('docker.common.delete')} ${labelOf(row)}`"
                    :box="44"
                    :disabled="!!Number(row.containers)"
                    @click="onRowAction('delete', row)"
                  />
                </template>
                <AppIconButton icon="HOutline:EllipsisHorizontalIcon" :label="t('docker.imgx.moreOf', { name: labelOf(row) })" :box="44" @click="openSheet(row)" />
              </footer>
            </article>
          </div>
        </template>

        <Pagination
          v-model:page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          @change="getList"
        />

        <Transition name="dk-pop">
          <div v-if="selected.length" class="dk-batch" role="toolbar" :aria-label="t('docker.image.batchLabel')">
            <span class="dk-batch__count">{{ t('docker.image.selectedCount', { n: selected.length }) }}</span>
            <button type="button" class="dk-batch__btn" :disabled="batchDeleting" @click="openBatchTag">
              <component :is="menuStore.iconComponents['HOutline:TagIcon']" />{{ t('docker.image.tagImage') }}
            </button>
            <button type="button" class="dk-batch__btn" :disabled="batchDeleting || exporting" @click="exportImages(selected)">
              <component :is="menuStore.iconComponents['HOutline:ArrowDownTrayIcon']" />{{ t('docker.imgx.exportTar') }}
            </button>
            <button type="button" class="dk-batch__btn is-danger" :disabled="batchDeleting || !deletableSelected.length" @click="handleBatchDelete">
              <component :is="menuStore.iconComponents['HOutline:TrashIcon']" />{{ t('docker.common.delete') }}
            </button>
            <button type="button" class="dk-batch__btn is-icon" :aria-label="t('docker.image.clearSelection')" @click="selected = []">
              <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
            </button>
          </div>
        </Transition>
      </div>
    </template>

    <!-- 手机：悬浮拉取 -->
    <button v-if="menuStore.isMobile && canManage && ready" type="button" class="dk-fab" :aria-label="t('docker.image.pullImage')" @click="openPull">
      <UIcon name="i-lucide-download" />
    </button>

    <!-- 手机：行操作面板 -->
    <DockerActionSheet
      v-model="sheet.open"
      :title="sheet.row ? labelOf(sheet.row) : ''"
      :subtitle="sheet.row ? `${shortId(sheet.row.id)} · ${formatBytes(sheet.row.size)}` : ''"
      icon="i-lucide-layers"
      tone="a2"
      :quick="sheetQuick"
      :items="sheetItems"
      @select="onSheetSelect"
    />

    <!-- 导入镜像 -->
    <FormDialog v-model="importVisible" :title="t('docker.imgx.importTitle')" :width="520" :loading="importing">
      <div class="dk-form">
        <p class="dk-hint">{{ t('docker.imgx.importDesc') }}</p>
        <button type="button" class="im-drop" :class="{ 'has-file': !!importFile }" :disabled="importing" @click="importInput?.click()">
          <span class="im-drop__sticker m-sticker" :class="importFile ? 'm-sticker--ok' : 'm-sticker--a'">
            <UIcon :name="importFile ? 'i-lucide-file-archive' : 'i-lucide-upload'" />
          </span>
          <span class="im-drop__text">
            <strong class="m-mono">{{ importFile ? importFile.name : t('docker.imgx.pickTar') }}</strong>
            <small>{{ importFile ? formatBytes(importFile.size) : t('docker.imgx.tarHint') }}</small>
          </span>
        </button>
        <input ref="importInput" type="file" class="im-hidden" accept=".tar,.tar.gz,.tgz,.tar.xz,.tar.bz2,application/x-tar,application/gzip" @change="onImportPicked" />
        <div v-if="importing" class="im-progress" role="progressbar" :aria-valuenow="importPct" aria-valuemin="0" aria-valuemax="100">
          <span :style="{ width: `${importPct}%` }" />
        </div>
        <span v-if="importing" class="dk-hint m-num">{{ importPct < 100 ? t('docker.imgx.uploading', { pct: importPct }) : t('docker.imgx.loading') }}</span>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" :disabled="importing" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" icon="i-lucide-upload" :loading="importing" :disabled="!importFile" @click="submitImport">{{ t('docker.imgx.import') }}</UButton>
      </template>
    </FormDialog>

    <!-- 批量打标签：给选中镜像加仓库前缀（推到私有仓库前常用） -->
    <FormDialog v-model="batchTagVisible" :title="t('docker.imgx.batchTagTitle', { n: batchTagRows.length })" :width="560" :loading="batchTagging" @confirm="submitBatchTag">
      <div class="dk-form">
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('docker.imgx.prefix') }}</label>
          <input v-model.trim="batchPrefix" class="app-input m-mono" placeholder="registry.example.com/team" @keyup.enter="submitBatchTag" />
          <span class="dk-hint">{{ t('docker.imgx.prefixHint') }}</span>
        </div>
        <div v-if="batchTagPreview.length" class="im-preview">
          <div v-for="p in batchTagPreview" :key="p.from" class="im-preview__row">
            <span class="m-mono">{{ p.from }}</span>
            <UIcon name="i-lucide-arrow-right" />
            <strong class="m-mono">{{ p.to }}</strong>
          </div>
        </div>
        <span v-if="batchTagSkipped" class="dk-hint">{{ t('docker.imgx.danglingSkipped', { n: batchTagSkipped }) }}</span>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="batchTagging" :disabled="!batchTagPreview.length" @click="submitBatchTag">{{ t('docker.image.tagImage') }}</UButton>
      </template>
    </FormDialog>

    <!-- 拉取镜像 -->
    <FormDialog v-model="pullVisible" :title="t('docker.image.pullImage')" :width="480" :loading="pullSubmitting" @confirm="submitPull">
      <div class="dk-form">
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('docker.image.imageReference') }}</label>
          <input v-model="pullForm.reference" class="app-input m-mono" :placeholder="t('docker.image.referencePlaceholder')" @keyup.enter="submitPull" />
          <span class="dk-hint">{{ t('docker.image.referenceHint') }}</span>
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.common.platform') }}</label>
          <input v-model="pullForm.platform" class="app-input m-mono" placeholder="linux/amd64" />
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="pullSubmitting" @click="submitPull">{{ t('docker.image.pull') }}</UButton>
      </template>
    </FormDialog>

    <!-- 打标签 -->
    <FormDialog v-model="tagVisible" :title="t('docker.image.tagImage')" :width="440" :loading="tagSubmitting" @confirm="submitTag">
      <div class="dk-form">
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('docker.image.newTag') }}</label>
          <input v-model="tagTarget" class="app-input m-mono" :placeholder="t('docker.image.newTagPlaceholder')" @keyup.enter="submitTag" />
        </div>
      </div>
    </FormDialog>

    <!-- 编辑标签 -->
    <FormDialog v-model="editVisible" :title="t('docker.image.editImageTags')" :width="520" :loading="editSubmitting" @confirm="submitEdit">
      <div class="dk-form">
        <div v-if="editCurrentTags.length" class="app-field">
          <label class="app-label">{{ t('docker.image.currentTags') }}</label>
          <div class="dk-tags">
            <button
              v-for="tag in editCurrentTags"
              :key="tag"
              type="button"
              class="dk-tagbtn m-mono"
              :class="{ 'is-del': editDelTags.includes(tag) }"
              :aria-pressed="editDelTags.includes(tag)"
              @click="toggleDelTag(tag)"
            >
              {{ tag }}
              <component :is="menuStore.iconComponents[editDelTags.includes(tag) ? 'HOutline:ArrowUturnLeftIcon' : 'HOutline:XMarkIcon']" />
            </button>
          </div>
          <span class="dk-hint">{{ t('docker.image.tagToggleHint') }}</span>
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.image.addTags') }}</label>
          <textarea v-model="editAddTagsText" class="app-textarea m-mono" rows="3" :placeholder="t('docker.image.oneTagPerLine')" />
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="editSubmitting" @click="submitEdit">{{ t('docker.common.save') }}</UButton>
      </template>
    </FormDialog>

    <!-- 镜像详情 -->
    <FormDialog v-model="detailVisible" :title="t('docker.image.imageDetail')" :width="720" :show-footer="false">
      <div v-if="detail" class="dk-detail">
        <div class="dk-detail__hero">
          <span class="dk-detail__sticker m-sticker m-sticker--a2"><component :is="menuStore.iconComponents['HOutline:PhotoIcon']" /></span>
          <div class="dk-detail__head">
            <div class="dk-detail__title">{{ (detail.repoTags || [])[0] || '<none>' }}</div>
            <div class="dk-detail__sub">{{ shortId(detail.id) }} · {{ detail.architecture || '-' }} / {{ detail.os || '-' }} · {{ formatBytes(detail.size) }}</div>
          </div>
        </div>
        <dl class="dk-kv">
          <div><dt>ID</dt><dd class="m-mono">{{ shortId(detail.id) }}</dd></div>
          <div><dt>{{ t('docker.common.size') }}</dt><dd class="m-num">{{ formatBytes(detail.size) }}</dd></div>
          <div><dt>{{ t('docker.common.architecture') }}</dt><dd>{{ detail.architecture || '-' }}</dd></div>
          <div><dt>{{ t('docker.common.os') }}</dt><dd>{{ detail.os || '-' }}</dd></div>
          <div><dt>{{ t('docker.common.author') }}</dt><dd>{{ detail.author || '-' }}</dd></div>
          <div><dt>{{ t('docker.common.createdAt') }}</dt><dd class="m-num">{{ formatDateTime(detail.created) }}</dd></div>
        </dl>
        <div v-if="detail.exposedPorts?.length || detail.volumes?.length" class="dk-grid">
          <div v-if="detail.exposedPorts?.length">
            <div class="dk-detail__label">{{ t('docker.image.exposedPorts') }}</div>
            <div class="dk-tags"><span v-for="p in detail.exposedPorts" :key="p" class="dk-tag dk-tag--plain">{{ p }}</span></div>
          </div>
          <div v-if="detail.volumes?.length">
            <div class="dk-detail__label">{{ t('docker.image.volumes') }}</div>
            <div class="dk-tags"><span v-for="v in detail.volumes" :key="v" class="dk-tag dk-tag--plain">{{ v }}</span></div>
          </div>
        </div>
        <div v-if="detail.cmd?.length">
          <div class="dk-detail__label">{{ t('docker.image.defaultCmd') }}</div>
          <div class="dk-code">{{ detail.cmd.join(' ') }}</div>
        </div>
        <div v-if="detail.repoTags?.length">
          <div class="dk-detail__label">{{ t('docker.image.repoTags') }}</div>
          <div class="dk-tags"><span v-for="tag in detail.repoTags" :key="tag" class="dk-tag">{{ tag }}</span></div>
        </div>
        <div v-if="detail.repoDigests?.length">
          <div class="dk-detail__label">{{ t('docker.common.summary') }}</div>
          <div class="dk-code"><div v-for="digest in detail.repoDigests" :key="digest">{{ digest }}</div></div>
        </div>
      </div>
      <div v-else class="dk-detail__loading">{{ t('docker.common.noData') }}</div>
    </FormDialog>

    <!-- 镜像构建历史 -->
    <FormDialog v-model="historyVisible" :title="t('docker.image.imageHistory')" :width="860" :show-footer="false">
      <div v-if="historyLoading" class="dk-detail__loading">{{ t('system.common.loading') }}</div>
      <div v-else-if="!historyItems.length" class="dk-detail__loading">{{ t('docker.common.noData') }}</div>
      <ol v-else class="dk-hist">
        <li v-for="(item, idx) in historyItems" :key="idx" class="dk-hist__row">
          <span class="dk-hist__dot" :class="{ 'is-empty': !Number(item.size) }" />
          <div class="dk-hist__main">
            <code class="dk-hist__cmd m-mono">{{ cleanCreatedBy(item.createdBy) }}</code>
            <span class="dk-hist__meta m-num">
              {{ shortId(item.id) === '<missing>' ? '' : shortId(item.id) }} {{ formatDateTime(item.created) }}
            </span>
          </div>
          <strong class="dk-hist__size m-num">{{ Number(item.size) ? formatBytes(item.size) : '0 B' }}</strong>
        </li>
      </ol>
    </FormDialog>

    <DockerTaskDialogs v-model="taskDialogsVisible" :active-task="activeTask" @finished="handleTaskFinished" />
  </div>
</template>

<script setup lang="ts">
import { pageTabTo } from '@/config/pageGroups'
import { useI18n } from 'vue-i18n'
import DockerSubNav from '@/views/docker/components/DockerSubNav.vue'
import DockerHeader from '@/views/docker/components/DockerHeader.vue'
import DockerActionSheet from '@/views/docker/components/DockerActionSheet.vue'
import DockerUnavailable from '@/views/docker/components/DockerUnavailable.vue'
import DockerPullTasks from '@/views/docker/components/DockerPullTasks.vue'
import DockerTaskDialogs from '@/views/docker/components/DockerTaskDialogs.vue'
import { useDockerStatus } from '@/views/docker/components/useDockerStatus'
import { formatAgo, formatBytes, formatDate, formatDateTime, shortId } from '@/views/docker/components/format'
import {
  deleteDockerImage, exportDockerImages, getDockerImage, imageHistory, importDockerImages, listDockerImages,
  pruneDockerImages, pullDockerImage, tagDockerImage, updateDockerImageTags,
} from '@/api/docker'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import { copyTextToClipboard, downloadFileFromUrl } from '@/utils/file'
import type {
  DockerImageHistoryItem, DockerImageInfo, DockerImageStats, DockerImageSummary, DockerTaskInfo,
} from '@/types/v1/docker'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'

defineOptions({ name: 'DockerImageView' })

const menuStore = useMenuStore()
const router = useRouter()
const { t } = useI18n()
const canManage = useButtonPermission([PERM.DOCKER_IMAGE_MANAGE], [])
const { ready, loaded: engineLoaded, refresh: refreshEngine } = useDockerStatus()

type Usage = '' | 'used' | 'unused' | 'dangling'
const list = ref<DockerImageSummary[]>([])
const stats = ref<DockerImageStats | null>(null)
const loading = ref(false)
const selected = ref<string[]>([])
const queryForm = reactive({ keyword: '', usage: '' as Usage, sort: 'created' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })
const pullTasksRef = ref<InstanceType<typeof DockerPullTasks> | null>(null)

const usageChips = computed(() => [
  { label: t('docker.common.all'), value: '' as Usage, count: stats.value?.total ?? null },
  { label: t('docker.image.usageUsed'), value: 'used' as Usage, count: stats.value?.used ?? null },
  { label: t('docker.image.usageUnused'), value: 'unused' as Usage, count: stats.value?.unused ?? null },
  { label: t('docker.image.usageDangling'), value: 'dangling' as Usage, count: stats.value?.dangling ?? null },
])
const sortOptions = computed(() => [
  { label: t('docker.image.sortCreated'), value: 'created' },
  { label: t('docker.image.sortSize'), value: 'size' },
  { label: t('docker.image.sortName'), value: 'name' },
])

// 仓库名 + 标签拆开显示：nginx + [1.27-alpine, stable-alpine]
const tagsOf = (row: Record<string, unknown>) =>
  ((row.repoTags as string[] | undefined) || []).filter((tag) => tag && tag !== '<none>:<none>')
const splitRef = (ref: string) => {
  const i = ref.lastIndexOf(':')
  return i > ref.lastIndexOf('/') && i > 0 ? [ref.slice(0, i), ref.slice(i + 1)] : [ref, 'latest']
}
const repoOf = (row: Record<string, unknown>) => {
  const first = tagsOf(row)[0]
  return first ? splitRef(first)[0] : '<none>'
}
const tagNamesOf = (row: Record<string, unknown>) => {
  const tags = tagsOf(row)
  if (!tags.length) return ['<none>']
  const repo = repoOf(row)
  return tags.map((tag) => (splitRef(tag)[0] === repo ? splitRef(tag)[1] : tag)) as string[]
}
const usedByOf = (row: Record<string, unknown>) => (row.usedBy as string[] | undefined) || []
const usageVariant = (row: Record<string, unknown>) =>
  usedByOf(row).length ? 'success' : row.dangling ? 'warning' : 'neutral'
const usageLabel = (row: Record<string, unknown>) =>
  usedByOf(row).length ? t('docker.image.usageUsed') : row.dangling ? t('docker.image.usageDangling') : t('docker.image.usageUnused')
const labelOf = (row: DockerImageSummary) =>
  row.dangling ? `${t('docker.image.usageDangling')} ${shortId(row.id)}` : tagsOf(row as unknown as Record<string, unknown>)[0] || shortId(row.id)

const columns = computed<DataTableColumn[]>(() => [
  { key: 'repo', title: t('docker.image.repoTagCol'), minWidth: 240, ellipsis: false },
  { key: 'id', title: t('docker.image.imageId'), width: 140 },
  { key: 'size', title: t('docker.common.size'), width: 100 },
  { key: 'created', title: t('docker.common.createdAt'), width: 130 },
  { key: 'usage', title: t('docker.image.usage'), minWidth: 190 },
  { key: 'operation', title: t('docker.common.operation'), width: 180, align: 'right' },
])

const menuActionsFor = (row: Record<string, unknown>, mobile = false): ActionMenuItem[] => {
  const manage = canManage.value
  return [
    { key: 'create', label: t('docker.image.createFrom'), icon: 'HOutline:PlusCircleIcon', hidden: !!row.dangling },
    { key: 'detail', label: t('docker.common.detail'), icon: 'HOutline:InformationCircleIcon' },
    { key: 'history', label: t('docker.image.history'), icon: 'HOutline:ClockIcon', hidden: !manage || !mobile },
    { key: 'editTags', label: t('docker.image.editTags'), icon: 'HOutline:PencilSquareIcon', hidden: !manage },
    { key: 'repull', label: t('docker.image.repull'), icon: 'HOutline:ArrowPathIcon', hidden: !manage || !!row.dangling },
    { key: 'copyId', label: t('docker.image.copyId'), icon: 'HOutline:ClipboardDocumentIcon' },
    { key: 'export', label: t('docker.imgx.exportTar'), icon: 'HOutline:ArrowDownTrayIcon', hidden: !manage },
    { key: 'forceDelete', label: t('docker.image.forceDelete'), icon: 'HOutline:TrashIcon', danger: true, hidden: !manage },
  ]
}

const findRow = (id: string) => list.value.find((x) => x.id === id)
const onRowAction = (key: string, row: Record<string, unknown>) => {
  const record = findRow(String(row.id))
  if (!record) return
  if (key === 'detail') openDetail(record)
  else if (key === 'editTags') openEdit(record)
  else if (key === 'tag') openTag(record)
  else if (key === 'history') openHistory(record)
  else if (key === 'delete') handleDelete(record, false)
  else if (key === 'forceDelete') handleDelete(record, true)
  else if (key === 'create') router.push(pageTabTo('docker', 'container', { create: '1', image: tagsOf(row)[0] || '' }))
  else if (key === 'repull') { pullForm.reference = tagsOf(row)[0] || ''; pullForm.platform = ''; void submitPull() }
  else if (key === 'export') void exportImages([record.id])
  else if (key === 'copyId') {
    void copyTextToClipboard(String(record.id).replace(/^sha256:/, '')).then(() => feedback.success(t('docker.image.copied')))
  }
}

const getList = async () => {
  if (!ready.value) return
  loading.value = true
  try {
    const { data } = await listDockerImages({
      all: false, page: pagination.value.page, pageSize: pagination.value.pageSize,
      keyword: queryForm.keyword.trim(), usage: queryForm.usage, sort: queryForm.sort,
      dangling: undefined, labels: {},
    })
    list.value = data?.items || []
    stats.value = data?.stats || null
    pagination.value.total = Number(data?.total || 0)
    const ids = new Set(list.value.map((i) => i.id))
    selected.value = selected.value.filter((id) => ids.has(id))
  } catch {
    list.value = []
    pagination.value.total = 0
  } finally {
    loading.value = false
  }
}
const search = () => { pagination.value.page = 1; void getList() }

const init = async (force = false) => {
  await refreshEngine(force)
  if (ready.value) await getList()
}

// —— 任务 ——
const taskDialogsVisible = ref(false)
const activeTask = ref<DockerTaskInfo | null>(null)
const openTasks = () => { activeTask.value = null; taskDialogsVisible.value = true }
const openTask = (task: DockerTaskInfo | undefined) => {
  if (!task?.id) return
  activeTask.value = task
  taskDialogsVisible.value = true
}
const handleTaskFinished = async () => { await getList(); void pullTasksRef.value?.refresh() }
const onTasksFinished = () => { void getList(); void refreshEngine(true) }

// —— 手机操作面板 ——
const sheet = reactive({ open: false, row: null as DockerImageSummary | null })
const openSheet = (row: DockerImageSummary) => {
  sheet.row = row
  sheet.open = true
}
const onSheetSelect = (key: string) => {
  if (sheet.row) onRowAction(key, sheet.row as unknown as Record<string, unknown>)
}
const sheetQuick = computed<ActionMenuItem[]>(() => {
  const row = sheet.row
  if (!row) return []
  return [
    { key: 'create', label: t('docker.image.createFrom'), icon: 'i-lucide-box', hidden: row.dangling },
    { key: 'repull', label: t('docker.image.repull'), icon: 'i-lucide-download', hidden: !canManage.value || row.dangling },
    { key: 'copyId', label: t('docker.image.copyId'), icon: 'i-lucide-copy' },
  ]
})
const sheetItems = computed<ActionMenuItem[]>(() => {
  const row = sheet.row
  if (!row) return []
  const manage = canManage.value
  return [
    { key: 'export', label: t('docker.imgx.exportTarLong'), icon: 'i-lucide-upload', hidden: !manage },
    { key: 'editTags', label: t('docker.image.editTags'), icon: 'i-lucide-tags', hidden: !manage },
    { key: 'detail', label: t('docker.imgx.inspect'), icon: 'i-lucide-file-text' },
    { key: 'forceDelete', label: t('docker.image.forceDelete'), icon: 'i-lucide-trash-2', danger: true, hidden: !manage },
  ]
})

// —— 导出 .tar ——
const exporting = ref(false)
const exportImages = async (ids: string[]) => {
  const rows = ids.map(findRow).filter((r): r is DockerImageSummary => !!r)
  if (!rows.length || exporting.value) return
  // 有标签的按标签导出（导入时能带回标签），悬空镜像按 ID
  const refs = rows.map((r) => (r.dangling ? r.id : tagsOf(r as unknown as Record<string, unknown>)[0] || r.id))
  exporting.value = true
  feedback.info(t('docker.imgx.exporting'))
  try {
    const { data } = await exportDockerImages(refs)
    const first = String(refs[0] || 'image').replace(/^sha256:/, '').replace(/[^a-zA-Z0-9._-]+/g, '_')
    const name = refs.length === 1 ? `${first}.tar` : `images-${refs.length}.tar`
    const url = URL.createObjectURL(data)
    downloadFileFromUrl(url, name)
    setTimeout(() => URL.revokeObjectURL(url), 60_000)
    feedback.success(t('docker.imgx.exported', { n: refs.length, size: formatBytes(data.size) }))
  } catch (e) { showRequestError(e, t('docker.imgx.exportFailed')) }
  finally { exporting.value = false }
}

// —— 导入 .tar ——
const importVisible = ref(false)
const importing = ref(false)
const importPct = ref(0)
const importFile = ref<File | null>(null)
const importInput = ref<HTMLInputElement | null>(null)
const openImport = () => {
  importFile.value = null
  importPct.value = 0
  importVisible.value = true
}
const onImportPicked = (e: Event) => {
  importFile.value = (e.target as HTMLInputElement).files?.[0] || null
}
const submitImport = async () => {
  if (!importFile.value || importing.value) return
  importing.value = true
  importPct.value = 0
  try {
    const { data } = await importDockerImages(importFile.value, (pct) => { importPct.value = pct })
    const loaded = data?.loaded || []
    feedback.success(loaded.length ? t('docker.imgx.imported', { names: loaded.join('、') }) : t('docker.imgx.importedNone'))
    importVisible.value = false
    await getList()
    void refreshEngine(true)
  } catch (e) { showRequestError(e, t('docker.imgx.importFailed')) }
  finally { importing.value = false }
}

// —— 批量打标签：加仓库前缀 ——
const batchTagVisible = ref(false)
const batchTagging = ref(false)
const batchPrefix = ref('')
const batchTagRows = computed(() => selected.value.map(findRow).filter((r): r is DockerImageSummary => !!r))
const batchTagSkipped = computed(() => batchTagRows.value.filter((r) => r.dangling).length)
const batchTagPreview = computed(() => {
  const prefix = batchPrefix.value.replace(/\/+$/, '')
  if (!prefix) return []
  return batchTagRows.value
    .filter((r) => !r.dangling)
    .map((r) => {
      const from = tagsOf(r as unknown as Record<string, unknown>)[0] || ''
      const [repo = '', tag = 'latest'] = splitRef(from)
      const parts = repo.split('/')
      const first = parts[0] || ''
      // 原引用带仓库主机时去掉主机部分，换成新前缀
      const base = parts.length > 1 && (first.includes('.') || first.includes(':') || first === 'localhost') ? parts.slice(1).join('/') : repo
      return { id: r.id, from, to: `${prefix}/${base}:${tag}` }
    })
    .filter((p) => p.from)
})
const openBatchTag = () => {
  batchPrefix.value = ''
  batchTagVisible.value = true
}
const submitBatchTag = async () => {
  if (!batchTagPreview.value.length || batchTagging.value) return
  batchTagging.value = true
  let ok = 0
  for (const p of batchTagPreview.value) {
    try {
      await tagDockerImage({ id: p.id, target: p.to })
      ok++
    } catch (e) { showRequestError(e, `${p.from}：${t('docker.image.tagFailed')}`) }
  }
  batchTagging.value = false
  if (ok) feedback.success(t('docker.imgx.batchTagged', { n: ok }))
  batchTagVisible.value = false
  selected.value = []
  await getList()
}

// —— 清理悬空 ——
const handlePrune = () => {
  if (!stats.value?.dangling) return
  Dialog.confirm({
    title: t('docker.image.pruneTitle'),
    content: t('docker.image.pruneContent', { n: stats.value.dangling, size: formatBytes(stats.value.reclaimableSize) }),
    confirmText: t('docker.image.pruneConfirm'),
    cancelText: t('docker.common.cancel'),
    onConfirm: async () => {
      try {
        const { data } = await pruneDockerImages({ all: false })
        feedback.success(t('docker.image.pruneSuccess', { n: data?.deleted || 0, size: formatBytes(data?.spaceReclaimed || 0) }))
        await getList()
        void refreshEngine(true)
      } catch (e) { showRequestError(e, t('docker.image.pruneFailed')) }
    },
  })
}

// —— 拉取 ——
const pullVisible = ref(false)
const pullSubmitting = ref(false)
const pullForm = reactive({ reference: '', platform: '' })
const openPull = () => { pullForm.reference = ''; pullForm.platform = ''; pullVisible.value = true }
const submitPull = async () => {
  const reference = pullForm.reference.trim()
  if (!reference) { feedback.error(t('docker.image.enterReference')); return }
  pullSubmitting.value = true
  try {
    await pullDockerImage({ reference, platform: pullForm.platform.trim(), registryAuth: undefined })
    feedback.success(t('docker.image.pullTaskCreated'))
    pullVisible.value = false
    // 拉取进度显示在页面顶部的卡片里，日志可从卡片打开
    void pullTasksRef.value?.refresh()
  } catch (e) { showRequestError(e, t('docker.image.pullFailed')) }
  finally { pullSubmitting.value = false }
}

// —— 打标签 ——
const tagVisible = ref(false)
const tagSubmitting = ref(false)
const tagTarget = ref('')
const tagId = ref('')
const openTag = (row: DockerImageSummary) => {
  tagId.value = row.id
  tagTarget.value = row.dangling ? '' : `${repoOf(row as unknown as Record<string, unknown>)}:`
  tagVisible.value = true
}
const submitTag = async () => {
  const target = tagTarget.value.trim()
  if (!target || target.endsWith(':')) { feedback.error(t('docker.image.enterNewTag')); return }
  tagSubmitting.value = true
  try {
    await tagDockerImage({ id: tagId.value, target })
    feedback.success(t('docker.image.tagSuccess'))
    tagVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, t('docker.image.tagFailed')) }
  finally { tagSubmitting.value = false }
}

// —— 删除 ——
const handleDelete = (row: DockerImageSummary, force: boolean) => {
  const name = labelOf(row)
  Dialog.confirm({
    title: force ? t('docker.image.forceDelete') : t('docker.image.confirmDeleteTitle'),
    content: force ? t('docker.image.forceDeleteContent', { name }) : t('docker.image.confirmDeleteContent', { name }),
    confirmText: t('docker.image.confirmDeleteText'),
    cancelText: t('docker.common.cancel'),
    onConfirm: async () => {
      try {
        // 多标签镜像按 ID 删除需要强制；普通删除传第一个标签，只删这一个引用
        const id = !force && tagsOf(row as unknown as Record<string, unknown>).length === 1 ? tagsOf(row as unknown as Record<string, unknown>)[0] as string : row.id
        await deleteDockerImage({ id, force, pruneChildren: false })
        feedback.success(t('docker.common.deletedName', { name }))
        await getList()
        void refreshEngine(true)
      } catch (e) { showRequestError(e, t('docker.image.deleteFailed')) }
    },
  })
}
const batchDeleting = ref(false)
const deletableSelected = computed(() => selected.value.map(findRow).filter((r): r is DockerImageSummary => !!r && !Number(r.containers)))
const handleBatchDelete = () => {
  const rows = deletableSelected.value
  if (!rows.length) return
  Dialog.confirm({
    title: t('docker.image.batchDeleteTitle'),
    content: t('docker.image.batchDeleteContent', { n: rows.length }),
    confirmText: t('docker.image.confirmDeleteText'),
    cancelText: t('docker.common.cancel'),
    onConfirm: async () => {
      batchDeleting.value = true
      let ok = 0
      for (const row of rows) {
        try {
          await deleteDockerImage({ id: row.id, force: tagsOf(row as unknown as Record<string, unknown>).length > 1, pruneChildren: false })
          ok++
        } catch (e) { showRequestError(e, t('docker.image.deleteFailed')) }
      }
      batchDeleting.value = false
      if (ok) feedback.success(t('docker.image.batchDeleteSuccess', { n: ok }))
      selected.value = []
      await getList()
      void refreshEngine(true)
    },
  })
}

// —— 详情 ——
const detailVisible = ref(false)
const detail = ref<DockerImageInfo | null>(null)
const openDetail = async (row: DockerImageSummary) => {
  detailVisible.value = true
  detail.value = null
  try { const { data } = await getDockerImage({ id: row.id }); detail.value = data?.info || null }
  catch (e) { showRequestError(e, t('docker.image.getDetailFailed')) }
}

// —— 构建历史 ——
const historyVisible = ref(false)
const historyLoading = ref(false)
const historyItems = ref<DockerImageHistoryItem[]>([])
const cleanCreatedBy = (text: string) =>
  (text || '').replace(/^\/bin\/sh -c #\(nop\)\s*/, '').replace(/^\/bin\/sh -c /, 'RUN ').trim() || '—'
const openHistory = async (row: DockerImageSummary) => {
  historyVisible.value = true
  historyLoading.value = true
  historyItems.value = []
  try { const { data } = await imageHistory({ id: row.id }); historyItems.value = data?.items || [] }
  catch (e) { showRequestError(e, t('docker.image.getHistoryFailed')) }
  finally { historyLoading.value = false }
}

// —— 编辑标签：点现有标签标记删除，文本框新增 ——
const editVisible = ref(false)
const editSubmitting = ref(false)
const editImageId = ref('')
const editCurrentTags = ref<string[]>([])
const editDelTags = ref<string[]>([])
const editAddTagsText = ref('')
const openEdit = (row: DockerImageSummary) => {
  editImageId.value = row.id
  editCurrentTags.value = tagsOf(row as unknown as Record<string, unknown>)
  editDelTags.value = []
  editAddTagsText.value = ''
  editVisible.value = true
}
const toggleDelTag = (tag: string) => {
  editDelTags.value = editDelTags.value.includes(tag) ? editDelTags.value.filter((x) => x !== tag) : [...editDelTags.value, tag]
}
const submitEdit = async () => {
  const addTags = editAddTagsText.value.split('\n').map((s) => s.trim()).filter(Boolean)
  const deleteTags = [...editDelTags.value]
  if (!addTags.length && !deleteTags.length) { feedback.error(t('docker.image.enterTagsToEdit')); return }
  editSubmitting.value = true
  try {
    await updateDockerImageTags({ imageId: editImageId.value, addTags, deleteTags, forceDelete: false })
    feedback.success(t('docker.image.tagUpdateSuccess'))
    editVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, t('docker.image.tagUpdateFailed')) }
  finally { editSubmitting.value = false }
}

onMounted(() => { void init() })
</script>

<style scoped lang="scss">
@use '../components/dk-page.scss';

.dk-em {
  color: var(--el-text-color-primary);
  font-weight: 800;
}
.dk-em--warn {
  color: var(--m-warn);
}
.dk-sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.dk-sort {
  flex: none;
  min-width: 150px;
}
.dk-repo {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.dk-repo__name {
  font-size: 0.875rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dk-repo__name.is-none,
.dk-card__name.is-none {
  color: var(--el-text-color-secondary);
}
.dk-when {
  display: flex;
  flex-direction: column;
  gap: 1px;
  font-size: 0.8125rem;
  font-weight: 600;
}
.dk-usage {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
  color: var(--el-text-color-primary);
  text-decoration: none;
}
.dk-usage__sticker {
  width: 28px;
  height: 28px;
  border-radius: 9px;
}
.dk-usage__sticker :deep(svg) {
  width: 15px;
  height: 15px;
}
.dk-usage__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  font-size: 0.8125rem;
}
.dk-usage__text > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 手机：统计行 */
.dk-mstats {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: -4px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.dk-mstats > span {
  flex: 1;
  min-width: 0;
}
.dk-linkbtn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 44px;
  padding: 0 10px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--m-pInk);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.dk-linkbtn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.dk-linkbtn :deep(svg),
.dk-linkbtn :deep(.iconify) {
  width: 16px;
  height: 16px;
}

/* 编辑标签：可点的标签 */
.dk-tagbtn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 0 8px 0 12px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
}
.dk-tagbtn :deep(svg) {
  width: 14px;
  height: 14px;
  color: var(--el-text-color-secondary);
}
.dk-tagbtn.is-del {
  border-color: var(--m-bad);
  background: var(--m-badSoft);
  color: var(--m-bad);
  text-decoration: line-through;
}

/* 构建历史 */
.dk-hist {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}
.dk-hist__row {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 10px 4px 10px 0;
}
.dk-hist__row + .dk-hist__row {
  border-top: 1px solid var(--el-border-color-lighter);
}
.dk-hist__dot {
  flex: none;
  width: 10px;
  height: 10px;
  margin-top: 5px;
  border-radius: 50%;
  background: var(--m-a2);
}
.dk-hist__dot.is-empty {
  background: var(--el-border-color-light);
}
.dk-hist__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.dk-hist__cmd {
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--el-text-color-primary);
  word-break: break-all;
  white-space: pre-wrap;
}
.dk-hist__meta {
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
}
.dk-hist__size {
  flex: none;
  font-size: 0.8125rem;
}

/* 导入 */
.im-drop {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  min-height: 72px;
  padding: calc(var(--cmp-card-unit) * 12) calc(var(--cmp-card-unit) * 16);
  border-radius: var(--app-radius-lg);
  border: 1.5px dashed var(--el-border-color-light);
  background: var(--el-bg-color);
  text-align: left;
  cursor: pointer;
}
.im-drop:hover:not(:disabled),
.im-drop.has-file {
  border-color: var(--m-p);
}
.im-drop__sticker {
  flex: none;
  width: 44px;
  height: 44px;
  border-radius: 14px;
}
.im-drop__sticker :deep(.iconify),
.im-drop__sticker :deep(svg) {
  width: 22px;
  height: 22px;
}
.im-drop__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.im-drop__text strong {
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.im-drop__text small {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.im-hidden {
  display: none;
}
.im-progress {
  height: 8px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}
.im-progress span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--m-p);
  transition: width 0.3s;
}
.im-preview {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 240px;
  overflow-y: auto;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  font-size: 0.75rem;
}
.im-preview__row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.im-preview__row > span,
.im-preview__row > strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.im-preview__row :deep(.iconify) {
  flex: none;
  width: 14px;
  height: 14px;
  color: var(--el-text-color-secondary);
}

.dk-pop-enter-active,
.dk-pop-leave-active {
  transition: opacity 0.18s, transform 0.18s;
}
.dk-pop-enter-from,
.dk-pop-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
@media (width < 768px) {
  .dk-sort {
    flex: 1 1 100%;
  }
}
</style>
