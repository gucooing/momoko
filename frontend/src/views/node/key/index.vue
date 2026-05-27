<template>
  <div>
    <el-card shadow="never" class="card-clear-mb">
      <el-form :model="queryForm" label-width="auto" ref="queryFormRef" @keyup.enter="getList">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item label="名称" prop="keywords">
              <el-input v-model="queryForm.keywords" placeholder="请输入 API Key 名称" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item>
              <el-button type="primary" :icon="menuStore.iconComponents.Search" @click="getList">
                搜索
              </el-button>
              <el-button :icon="menuStore.iconComponents.Refresh" @click="reset">重置</el-button>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <el-card shadow="never" class="card-mt-16">
      <div class="operation-container">
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" @click="apiKeyCreateRef?.showDialog()">
          新增 API Key
        </el-button>
      </div>

      <!-- desktop: table -->
      <VxeGrid
        v-if="!menuStore.isMobile"
        v-bind="gridConfig"
      >
        <template #column-expiresAt="{ row }: { row: APIKeyInfo }">
          <span v-if="row.expiresAt">{{ formatTime(row.expiresAt) }}</span>
          <BaseTag v-else text="永久有效" type="success" />
        </template>
        <template #column-createTime="{ row }: { row: APIKeyInfo }">
          {{ formatTime(row.createTime) }}
        </template>
        <template #column-updateTime="{ row }: { row: APIKeyInfo }">
          {{ formatTime(row.updateTime) }}
        </template>
        <template #column-operation="{ row }: { row: APIKeyInfo }">
          <el-button type="primary" :icon="menuStore.iconComponents.Edit" link @click="openEdit(row)">编辑</el-button>
          <el-button type="primary" :icon="menuStore.iconComponents['HOutline:ClipboardDocumentIcon']" link @click="openCopy(row)">
            复制
          </el-button>
          <el-button type="primary" :icon="menuStore.iconComponents.Refresh" link @click="openRefresh(row)">刷新</el-button>
        </template>
      </VxeGrid>

      <!-- mobile: cards -->
      <div v-else class="mobile-card-list">
        <div v-if="!list.length" class="mobile-empty"><el-empty description="暂无数据" /></div>
        <div v-for="row in list" :key="row.id" class="mobile-card">
          <div class="mobile-card-body">
            <div class="mobile-card-header">
              <span class="mobile-card-title">{{ row.name }}</span>
              <BaseTag v-if="!row.expiresAt" text="永久有效" type="success" />
            </div>
            <div class="mobile-card-meta">
              <span class="text-muted">{{ row.apiKey }}</span>
            </div>
            <div class="mobile-card-meta" v-if="row.expiresAt">
              <span>过期时间：{{ formatTime(row.expiresAt) }}</span>
            </div>
            <div class="mobile-card-meta">
              <span>创建时间：{{ formatTime(row.createTime) }}</span>
            </div>
          </div>
          <div class="mobile-card-actions">
            <el-button size="small" plain type="primary" @click.stop="openEdit(row)">编辑</el-button>
            <el-button size="small" plain type="primary" @click.stop="openCopy(row)">复制</el-button>
            <el-button size="small" plain type="primary" @click.stop="openRefresh(row)">刷新</el-button>
          </div>
        </div>
      </div>

      <TablePagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :is-mobile="menuStore.isMobile"
        @change="getList"
      />
    </el-card>

    <ApiKeyCreate ref="apiKeyCreateRef" @refresh="refresh" />

    <!-- Copy key dialog -->
    <BaseDialog
      v-model="copyDialogOpen"
      title="复制 API Key"
      width="550"
      show-footer
      :show-confirm-button="false"
      cancel-text="关闭"
      @close="copyDialogOpen = false"
    >
      <div class="copy-dialog-content">
        <el-alert
          title="请立即复制并妥善保存此 API Key，关闭后无法再次查看完整值。"
          type="warning"
          show-icon
          :closable="false"
          style="margin-bottom: 16px"
        />
        <el-input
          v-model="copyKeyValue"
          readonly
          style="width: 100%"
        >
          <template #append>
            <el-button :icon="menuStore.iconComponents['HOutline:ClipboardDocumentIcon']" @click="doCopy">
              复制
            </el-button>
          </template>
        </el-input>
      </div>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { copyAPIKey, listAPIKeys, refreshAPIKey } from '@/api/node'
import { VxeGrid } from '@/plugins/vxeGrid'
import type { APIKeyInfo } from '@/types/v1/node'
import { Dialog } from '@/utils/dialog'
import ApiKeyCreate from '@/views/node/key/create.vue'
import type { FormInstance } from 'element-plus'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'ApiKeyView' })

const menuStore = useMenuStore()
const queryFormRef = useTemplateRef<FormInstance>('queryFormRef')
const apiKeyCreateRef = useTemplateRef<InstanceType<typeof ApiKeyCreate> | null>('apiKeyCreateRef')

const list = ref<APIKeyInfo[]>([])
const copyDialogOpen = ref(false)
const copyKeyValue = ref('')

const queryForm = ref({
  keywords: '',
})

const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0,
})

const formatTime = (t: Date | string | undefined): string => {
  if (!t) return '-'
  const d = new Date(t)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const gridConfig = computed<VxeGridProps>(() => ({
  border: true,
  showOverflow: true,
  rowConfig: { isHover: true },
  data: list.value,
  columns: [
    { type: 'seq', title: '序号', width: 55, fixed: 'left' },
    { field: 'name', title: '名称', minWidth: 160, fixed: 'left' },
    { field: 'apiKey', title: 'API Key', minWidth: 260 },
    { field: 'expiresAt', title: '过期时间', minWidth: 180, slots: { default: 'column-expiresAt' } },
    { field: 'createTime', title: '创建时间', minWidth: 180, slots: { default: 'column-createTime' } },
    { field: 'updateTime', title: '更新时间', minWidth: 180, slots: { default: 'column-updateTime' } },
    { title: '操作', width: 220, fixed: 'right', slots: { default: 'column-operation' } },
  ],
}))

const reset = () => {
  queryFormRef.value?.resetFields()
  pagination.value.page = 1
  getList()
}

const getList = async () => {
  const { data: res } = await listAPIKeys({
    page: pagination.value.page,
    pageSize: pagination.value.pageSize,
    keywords: queryForm.value.keywords || undefined,
  })

  list.value = res?.infos || []
  pagination.value.total = Number(res?.total || 0)
  pagination.value.page = Number(res?.page || pagination.value.page)
  pagination.value.pageSize = Number(res?.pageSize || pagination.value.pageSize)
}

const openEdit = (row: APIKeyInfo) => {
  apiKeyCreateRef.value?.showDialog(row)
}

const openCopy = async (row: APIKeyInfo) => {
  const { data: res } = await copyAPIKey({ id: row.id })
  copyKeyValue.value = res?.info?.apiKey || ''
  copyDialogOpen.value = true
}

const openRefresh = async (row: APIKeyInfo) => {
  try {
    await Dialog.confirm({
      title: '刷新 API Key',
      content: '刷新后将生成新的 API Key，原 Key 将失效，确定刷新？',
      confirmText: '确定',
      cancelText: '取消',
    })
  } catch {
    return
  }
  await refreshAPIKey({ id: row.id })
  ElMessage.success('刷新成功，API Key 已更新')
  getList()
}

const doCopy = async () => {
  await navigator.clipboard.writeText(copyKeyValue.value)
  ElMessage.success('已复制到剪贴板')
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') {
    pagination.value.page = 1
  }
  getList()
}

onMounted(() => {
  getList()
})
</script>

<style lang="scss" scoped>
.copy-dialog-content {
  padding: 4px 0;
}

.mobile-card-list { display: flex; flex-direction: column; gap: 0.55rem; }
.mobile-empty { padding: 1.5rem 0; }
.mobile-card {
  position: relative; display: flex; align-items: flex-start; gap: 0.6rem;
  padding: 0.7rem 0.8rem; border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.6rem; background: var(--el-bg-color);
}
.mobile-card-body { flex: 1; min-width: 0; }
.mobile-card-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.mobile-card-title { font-size: 0.88rem; font-weight: 700; color: var(--el-text-color-primary); }
.mobile-card-meta { display: flex; align-items: center; gap: 0.3rem; margin-top: 0.25rem; font-size: 0.74rem; color: var(--el-text-color-secondary); }
.text-muted { color: var(--el-text-color-placeholder); }
.mobile-card-actions { display: flex; flex-direction: column; gap: 0.3rem; flex-shrink: 0; }
.mobile-card-actions .el-button + .el-button { margin-left: 0; }
</style>
