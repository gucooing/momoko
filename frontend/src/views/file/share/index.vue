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

    <el-table v-loading="loading" :data="items" class="share-table">
      <el-table-column :label="t('file.share.name')" min-width="160">
        <template #default="{ row }">
          <div class="share-name">
            <el-icon><component :is="row.isDir ? IconFolder : IconFile" /></el-icon>
            <span>{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="t('file.share.path')" prop="targetPath" min-width="200" show-overflow-tooltip />
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
          {{ row.downloadCount }}<span v-if="Number(row.maxDownloads) > 0"> / {{ row.maxDownloads }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('file.share.enabled')" width="90">
        <template #default="{ row }">
          <el-switch :model-value="row.enabled" @change="toggleEnabled(row)" />
        </template>
      </el-table-column>
      <el-table-column :label="t('fileManager.operation')" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="copyLink(row)">{{ t('file.share.copyLink') }}</el-button>
          <el-button link type="primary" @click="openEdit(row)">{{ t('system.common.edit') }}</el-button>
          <el-button link type="danger" @click="remove(row)">{{ t('fileManager.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="share-footer">
      <TablePagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :is-mobile="false"
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
import { formatDateTime, copyTextToClipboard } from '@/utils/file'
import { listSharesRequest, deleteShareRequest, updateShareRequest, buildShareLink } from '@/api/share'
import TablePagination from '@/components/pagination/TablePagination.vue'
import ShareFormDialog from '@/components/share/ShareFormDialog.vue'
import { IconFolder, IconFile } from '@/components/file/icons'
import type { ShareInfo } from '@/types/v1/file'

const { t } = useI18n()

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
      path: '',
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
.share-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}
</style>
