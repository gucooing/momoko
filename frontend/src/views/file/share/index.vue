<template>
  <el-card class="share-page" shadow="never">
    <div class="share-toolbar">
      <h2 class="share-title">{{ t('file.share.pageTitle') }}</h2>
      <div class="share-toolbar-right">
        <el-input
          v-model="keywords"
          :placeholder="t('file.share.keywordPlaceholder')"
          clearable
          class="share-search"
          @input="onSearch"
        />
        <el-button type="primary" @click="openCreate">{{ t('file.share.create') }}</el-button>
      </div>
    </div>

    <el-table v-if="!menuStore.isMobile" v-loading="loading" :data="items" class="share-table">
      <el-table-column :label="t('file.share.name')" min-width="160">
        <template #default="{ row }">
          <div class="share-name">
            <el-icon><IconFolder /></el-icon>
            <span>{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="t('file.share.path')" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">
          {{ row.paths.join(', ') }}
        </template>
      </el-table-column>
      <el-table-column :label="t('file.share.code')" width="100">
        <template #default="{ row }">
          {{ row.code || t('file.share.noCode') }}
        </template>
      </el-table-column>
      <el-table-column :label="t('file.share.expires')" width="170">
        <template #default="{ row }">
          {{ row.expiresAt ? formatDateTime(row.expiresAt) : t('file.share.never') }}
        </template>
      </el-table-column>
      <el-table-column :label="t('file.share.downloads')" width="110">
        <template #default="{ row }">
          {{ row.downloadCount
          }}<span v-if="Number(row.maxDownloads) > 0"> / {{ row.maxDownloads }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('file.share.enabled')" width="90">
        <template #default="{ row }">
          <el-switch :model-value="row.enabled" @change="toggleEnabled(row)" />
        </template>
      </el-table-column>
      <el-table-column :label="t('fileManager.operation')" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="copyLink(row)">{{
            t('file.share.copyLink')
          }}</el-button>
          <el-button link type="primary" @click="openEdit(row)">{{
            t('system.common.edit')
          }}</el-button>
          <el-button link type="danger" @click="remove(row)">{{
            t('fileManager.delete')
          }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div v-else v-loading="loading" class="share-mobile-list">
      <el-empty v-if="!items.length" :description="t('system.common.noData')" />
      <div v-for="row in items" v-else :key="row.id" class="share-mobile-card">
        <div class="share-mobile-body">
          <div class="share-mobile-header">
            <div class="share-mobile-name">
              <el-icon><IconFolder /></el-icon>
              <span>{{ row.name }}</span>
            </div>
            <el-switch :model-value="row.enabled" @change="toggleEnabled(row)" />
          </div>
          <div class="share-mobile-paths">
            <el-tag v-for="path in row.paths" :key="path" class="share-mobile-path" size="small">
              {{ path }}
            </el-tag>
          </div>
          <div class="share-mobile-meta">
            <span>{{ row.code || t('file.share.noCode') }}</span>
            <span>{{ row.expiresAt ? formatDateTime(row.expiresAt) : t('file.share.never') }}</span>
            <span>
              {{ row.downloadCount
              }}<template v-if="Number(row.maxDownloads) > 0"> / {{ row.maxDownloads }}</template>
            </span>
          </div>
        </div>
        <div class="share-mobile-actions">
          <el-button link type="primary" @click="copyLink(row)">{{
            t('file.share.copyLink')
          }}</el-button>
          <el-button link type="primary" @click="openEdit(row)">{{
            t('system.common.edit')
          }}</el-button>
          <el-button link type="danger" @click="remove(row)">{{
            t('fileManager.delete')
          }}</el-button>
        </div>
      </div>
    </div>

    <div class="share-footer">
      <TablePagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :is-mobile="menuStore.isMobile"
        @change="loadList"
      />
    </div>

    <ShareFormDialog v-model="formOpen" :share="editing" @saved="loadList" />
  </el-card>
</template>

<script setup lang="ts">
defineOptions({ name: 'FileShareView' })
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import { showRequestError } from '@/utils/request'
import { useMenuStore } from '@/stores/menu'
import { formatDateTime, copyTextToClipboard } from '@/utils/file'
import {
  listSharesRequest,
  deleteShareRequest,
  updateShareRequest,
  buildShareLink,
} from '@/api/share'
import TablePagination from '@/components/pagination/TablePagination.vue'
import ShareFormDialog from '@/components/share/ShareFormDialog.vue'
import { IconFolder } from '@/components/file/icons'
import type { ShareInfo } from '@/types/v1/file'

const { t } = useI18n()
const menuStore = useMenuStore()

const items = ref<ShareInfo[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keywords = ref('')

const formOpen = ref(false)
const editing = ref<ShareInfo | null>(null)

const loadList = async () => {
  loading.value = true
  try {
    const { data } = await listSharesRequest({
      page: page.value,
      pageSize: pageSize.value,
      keywords: keywords.value.trim() || undefined,
    })
    items.value = data?.items || []
    total.value = data?.total || 0
  } catch (error) {
    showRequestError(error)
  } finally {
    loading.value = false
  }
}

const onSearch = useDebounceFn(() => {
  page.value = 1
  loadList()
}, 350)

const openCreate = () => {
  editing.value = null
  formOpen.value = true
}
const openEdit = (row: ShareInfo) => {
  editing.value = row
  formOpen.value = true
}

const toggleEnabled = async (row: ShareInfo) => {
  try {
    await updateShareRequest({
      id: row.id,
      name: row.name,
      code: row.code,
      expiresAt: row.expiresAt,
      maxDownloads: row.maxDownloads,
      enabled: !row.enabled,
      paths: [],
      sourceId: row.sourceId,
    })
    loadList()
  } catch (error) {
    showRequestError(error)
  }
}

const copyLink = async (row: ShareInfo) => {
  try {
    await copyTextToClipboard(buildShareLink(row.token))
    ElMessage.success(t('file.share.copied'))
  } catch {
    ElMessage.error(t('file.share.copyFailed'))
  }
}

const remove = async (row: ShareInfo) => {
  try {
    await ElMessageBox.confirm(t('file.share.confirmDelete'), t('fileManager.confirmDelete'), {
      type: 'warning',
      confirmButtonText: t('system.common.confirm'),
      cancelButtonText: t('system.common.cancel'),
    })
  } catch {
    return
  }
  try {
    await deleteShareRequest(row.id)
    ElMessage.success(t('system.common.deleteSuccess'))
    loadList()
  } catch (error) {
    showRequestError(error)
  }
}

onMounted(loadList)
</script>

<style scoped>
.share-page {
  display: flex;
  flex-direction: column;
}
.share-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
  margin-bottom: 1rem;
}
.share-title {
  margin: 0;
  font-size: 1.05rem;
  font-weight: 600;
}
.share-toolbar-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.share-search {
  width: 240px;
}
.share-name {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.share-mobile-list {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}
.share-mobile-card {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.75rem 0.8rem;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.6rem;
  background: var(--el-bg-color);
}
.share-mobile-body {
  flex: 1;
  min-width: 0;
}
.share-mobile-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}
.share-mobile-name {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  min-width: 0;
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.share-mobile-name span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.share-mobile-paths {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-top: 0.45rem;
}
.share-mobile-path {
  max-width: 100%;
}
.share-mobile-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem 0.75rem;
  margin-top: 0.45rem;
  font-size: 0.74rem;
  color: var(--el-text-color-secondary);
}
.share-mobile-actions {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  flex-shrink: 0;
}
.share-mobile-actions .el-button + .el-button {
  margin-left: 0;
}
.share-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}

@media (max-width: 767px) {
  .share-page {
    border: none;
  }
  .share-page :deep(.el-card__body) {
    padding: 0.85rem;
  }
  .share-toolbar {
    align-items: stretch;
    gap: 0.75rem;
  }
  .share-title {
    font-size: 1rem;
  }
  .share-toolbar-right {
    width: 100%;
    gap: 0.55rem;
  }
  .share-search {
    flex: 1;
    min-width: 0;
    width: auto;
  }
  .share-mobile-card {
    gap: 0.55rem;
  }
  .share-footer {
    justify-content: center;
  }
}
</style>
