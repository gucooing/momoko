<template>
  <div>
    <el-card shadow="never" class="card-clear-mb">
      <el-form :model="queryForm" label-width="auto" @keyup.enter="getList">
        <el-row :gutter="12">
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item :label="t('file.share.keyword')">
              <el-input
                v-model="queryForm.keywords"
                :placeholder="t('file.share.keywordPlaceholder')"
                :prefix-icon="menuStore.iconComponents.Search"
                clearable
              />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item>
              <el-button type="primary" :icon="menuStore.iconComponents.Search" @click="getList">
                {{ t('system.common.search') }}
              </el-button>
              <el-button :icon="menuStore.iconComponents.Refresh" @click="reset">
                {{ t('system.common.reset') }}
              </el-button>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <el-card shadow="never" class="card-mt-16">
      <div class="operation-container">
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" @click="openCreate">
          {{ t('file.share.create') }}
        </el-button>
      </div>

      <el-table v-loading="loading" :data="list" class="table-mt-16" border>
        <el-table-column :label="t('file.share.name')" prop="name" min-width="140" show-overflow-tooltip />
        <el-table-column v-if="!isMobile" :label="t('file.share.type')" width="90">
          <template #default="{ row }">
            <el-tag :type="row.isDir ? 'warning' : 'info'" size="small">
              {{ row.isDir ? t('file.share.folder') : t('file.share.file') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="!isMobile" :label="t('file.share.code')" width="100">
          <template #default="{ row }">
            <span>{{ row.code || t('file.share.noCode') }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="!isMobile" :label="t('file.share.expires')" min-width="160">
          <template #default="{ row }">
            <span>{{ row.expiresAt ? formatTime(row.expiresAt) : t('file.share.never') }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="!isMobile" :label="t('file.share.downloads')" width="110">
          <template #default="{ row }">
            <span>{{ row.downloadCount }} / {{ Number(row.maxDownloads) > 0 ? row.maxDownloads : '∞' }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('file.share.enabled')" width="80">
          <template #default="{ row }">
            <el-switch :model-value="row.enabled" @change="toggleEnabled(row)" />
          </template>
        </el-table-column>
        <el-table-column :label="t('system.common.operation')" :width="isMobile ? 130 : 180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="copyLink(row)">{{ t('file.share.copyLink') }}</el-button>
            <el-button link type="primary" @click="openEdit(row)">{{ t('system.common.edit') }}</el-button>
            <el-popconfirm :title="t('file.share.confirmDelete')" @confirm="remove(row)">
              <template #reference>
                <el-button link type="danger">{{ t('system.common.delete') }}</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        class="pagination-mt-16"
        :current-page="queryForm.page"
        :page-size="queryForm.pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="onPageChange"
      />
    </el-card>

    <ShareFormDialog v-model="dialogVisible" :share="editingShare" @saved="getList" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useWindowSize } from '@vueuse/core'
import ShareFormDialog from '@/components/share/ShareFormDialog.vue'
import { buildShareLink, deleteShareRequest, listSharesRequest, updateShareRequest } from '@/api/share'
import type { ShareInfo } from '@/types/v1/file'

defineOptions({ name: 'FileShareManage' })

const { t } = useI18n()
const menuStore = useMenuStore()

const { width } = useWindowSize()
const isMobile = computed(() => width.value <= 640)

const loading = ref(false)
const list = ref<ShareInfo[]>([])
const total = ref(0)
const queryForm = reactive({ keywords: '', page: 1, pageSize: 10 })

const formatTime = (v: unknown) => (v ? new Date(v as string).toLocaleString() : '')

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listSharesRequest({
      page: queryForm.page,
      pageSize: queryForm.pageSize,
      keywords: queryForm.keywords || undefined,
    })
    list.value = data?.items ?? []
    total.value = Number(data?.total ?? 0)
  } finally {
    loading.value = false
  }
}

const reset = () => {
  queryForm.keywords = ''
  queryForm.page = 1
  getList()
}

const onPageChange = (page: number) => {
  queryForm.page = page
  getList()
}

const copyLink = async (row: ShareInfo) => {
  try {
    await navigator.clipboard.writeText(buildShareLink(row.token))
    ElMessage.success(t('file.share.copied'))
  } catch {
    ElMessage.error(t('file.share.copyFailed'))
  }
}

const dialogVisible = ref(false)
const editingShare = ref<ShareInfo | null>(null)

const openCreate = () => {
  editingShare.value = null
  dialogVisible.value = true
}

const openEdit = (row: ShareInfo) => {
  editingShare.value = row
  dialogVisible.value = true
}

const toggleEnabled = async (row: ShareInfo) => {
  await updateShareRequest({
    id: row.id,
    name: row.name,
    code: row.code,
    expiresAt: row.expiresAt ? new Date(row.expiresAt as unknown as string) : undefined,
    maxDownloads: Number(row.maxDownloads) || 0,
    enabled: !row.enabled,
  })
  getList()
}

const remove = async (row: ShareInfo) => {
  await deleteShareRequest({ id: row.id })
  ElMessage.success(t('system.common.deleteSuccess'))
  getList()
}

onMounted(getList)
</script>

<style scoped lang="scss">
.table-mt-16 {
  margin-top: 16px;
}
.pagination-mt-16 {
  margin-top: 16px;
  justify-content: flex-end;
}
</style>
