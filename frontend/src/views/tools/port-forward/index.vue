<template>
  <div>
    <el-card shadow="never" class="card-clear-mb">
      <el-form :model="queryForm" label-width="auto" ref="queryFormRef" @keyup.enter="getList">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item label="关键词" prop="keywords">
              <el-input v-model="queryForm.keywords" placeholder="搜索名称/地址/标签" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item label="转发类型" prop="type">
              <el-select v-model="queryForm.type" placeholder="全部" clearable style="width: 100%">
                <el-option label="TCP" value="PORT_FORWARD_TYPE_TCP" />
                <el-option label="UDP" value="PORT_FORWARD_TYPE_UDP" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item label="启用状态" prop="isEnable">
              <el-select v-model="queryForm.isEnable" placeholder="全部" clearable style="width: 100%">
                <el-option label="已启用" :value="true" />
                <el-option label="已禁用" :value="false" />
              </el-select>
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
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" @click="openCreateDialog()">
          新增规则
        </el-button>
        <AdaptiveConfirm
          title="确定要删除选中的规则吗？"
          :placement="POPCONFIRM_CONFIG.placement"
          :width="POPCONFIRM_CONFIG.width"
          @confirm="batchDelete"
        >
          <template #reference>
            <el-button
              type="danger"
              :icon="menuStore.iconComponents.Delete"
              :disabled="!deleteIds.length"
            >
              批量删除
            </el-button>
          </template>
        </AdaptiveConfirm>
      </div>

      <!-- desktop: table -->
      <VxeGrid
        v-if="!menuStore.isMobile"
        v-bind="gridConfig"
        @checkbox-change="selectionChange"
        @checkbox-all="selectionChange"
      >
        <template #column-type="{ row }">
          <el-tag :type="row.type === 'PORT_FORWARD_TYPE_TCP' ? 'primary' : 'warning'" size="small">
            {{ row.type === 'PORT_FORWARD_TYPE_TCP' ? 'TCP' : 'UDP' }}
          </el-tag>
        </template>
        <template #column-listen="{ row }">
          <span class="mono-text">{{ row.listenAddress }}:{{ row.listenPort }}</span>
        </template>
        <template #column-target="{ row }">
          <span class="mono-text">{{ row.targetAddress }}:{{ row.targetPort }}</span>
        </template>
        <template #column-isEnable="{ row }">
          <el-switch
            :model-value="row.isEnable"
            size="small"
            @change="(val: string | number | boolean) => toggleEnable(row, !!val)"
          />
        </template>
        <template #column-tags="{ row }">
          <el-tag
            v-for="(tag, idx) in splitTags(row.tags)"
            :key="idx"
            size="small"
            type="info"
            class="tag-item"
          >
            {{ tag }}
          </el-tag>
        </template>
        <template #column-name="{ row }">
          <span>{{ row.name }}</span>
          <el-tooltip v-if="row.error" :content="row.error" placement="top" :show-after="300">
            <span class="error-hint">?</span>
          </el-tooltip>
        </template>
        <template #column-operation="{ row }">
          <div class="pf-actions">
            <el-button
              type="primary"
              :icon="menuStore.iconComponents.Edit"
              link
              @click="openEditDialog(row)"
            >
              编辑
            </el-button>
            <el-popconfirm
              title="确定要删除该规则吗？"
              :width="POPCONFIRM_CONFIG.width"
              @confirm="confirmDelete(row)"
            >
              <template #reference>
                <el-button
                  type="danger"
                  :icon="menuStore.iconComponents.Delete"
                  link
                >
                  删除
                </el-button>
              </template>
            </el-popconfirm>
          </div>
        </template>
      </VxeGrid>

      <!-- mobile: cards -->
      <div v-else v-loading="loading" class="mobile-card-list">
        <div v-if="!list.length" class="mobile-empty">
          <el-empty description="暂无数据" />
        </div>
        <div
          v-for="row in list"
          :key="row.id"
          class="pf-card"
          :class="{ 'is-selected': deleteIds.includes(row.id) }"
        >
          <div class="pf-card-check" @click.stop="toggleMobileSelect(row.id)">
            <el-checkbox :model-value="deleteIds.includes(row.id)" size="small" />
          </div>

          <div class="pf-card-header">
            <span class="pf-card-name">{{ row.name }}</span>
            <el-tooltip v-if="row.error" :content="row.error" placement="top" :show-after="300">
              <span class="error-hint">?</span>
            </el-tooltip>
            <el-switch
              :model-value="row.isEnable"
              size="small"
              @change="(val: string | number | boolean) => toggleEnable(row, !!val)"
            />
          </div>

          <div class="pf-card-body">
            <div class="pf-card-meta">
              <el-tag :type="row.type === 'PORT_FORWARD_TYPE_TCP' ? 'primary' : 'warning'" size="small">
                {{ row.type === 'PORT_FORWARD_TYPE_TCP' ? 'TCP' : 'UDP' }}
              </el-tag>
            </div>
            <div class="pf-card-meta">
              <el-icon size="12"><component :is="menuStore.iconComponents['HOutline:ArrowDownTrayIcon']" /></el-icon>
              <span class="mono-text">{{ row.listenAddress }}:{{ row.listenPort }}</span>
            </div>
            <div class="pf-card-meta">
              <el-icon size="12"><component :is="menuStore.iconComponents['HOutline:ArrowUpTrayIcon']" /></el-icon>
              <span class="mono-text">{{ row.targetAddress }}:{{ row.targetPort }}</span>
            </div>
            <div v-if="row.tags" class="pf-card-tags">
              <el-tag
                v-for="(tag, idx) in splitTags(row.tags)"
                :key="idx"
                size="small"
                type="info"
              >
                {{ tag }}
              </el-tag>
            </div>
            <div v-if="row.remark" class="pf-card-remark">{{ row.remark }}</div>
            <div class="pf-card-time">{{ row.createTime }}</div>
          </div>

          <div class="pf-card-footer">
            <el-button size="small" plain type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-popconfirm
              title="确定要删除该规则吗？"
              :width="POPCONFIRM_CONFIG.width"
              @confirm="confirmDelete(row)"
            >
              <template #reference>
                <el-button size="small" plain type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </div>
        </div>
      </div>

      <!-- mobile: batch bar -->
      <div v-if="menuStore.isMobile && deleteIds.length" class="mobile-batch-bar">
        <span>已选 {{ deleteIds.length }} 项</span>
        <el-button size="small" type="danger" @click="batchDelete">批量删除</el-button>
      </div>

      <TablePagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :is-mobile="menuStore.isMobile"
        @change="getList"
      />
    </el-card>

    <PortForwardCreate ref="createRef" @refresh="refresh" />
  </div>
</template>

<script setup lang="ts">
import TablePagination from '@/components/pagination/TablePagination.vue'
import { POPCONFIRM_CONFIG } from '@/config/elementConfig'
import { VxeGrid } from '@/plugins/vxeGrid'
import PortForwardCreate from '@/views/tools/port-forward/create.vue'
import { listPortForwards, deletePortForward, updatePortForward } from '@/api/network'
import { PortForwardType, type PortForwardInfo } from '@/types/v1/network'
import type { FormInstance } from 'element-plus'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'PortForwardView' })

const menuStore = useMenuStore()
const queryFormRef = useTemplateRef<FormInstance>('queryFormRef')
const createRef = useTemplateRef<InstanceType<typeof PortForwardCreate> | null>('createRef')

const deleteIds = ref<string[]>([])
const list = ref<PortForwardInfo[]>([])
const loading = ref(false)

const queryForm = ref({
  keywords: '',
  type: undefined as PortForwardType | undefined,
  isEnable: undefined as boolean | undefined,
})

const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0,
})

const splitTags = (tags: string): string[] => {
  if (!tags) return []
  return tags.split(',').map((t) => t.trim()).filter(Boolean)
}

const gridConfig = computed<VxeGridProps>(() => ({
  border: true,
  showOverflow: true,
  rowConfig: { isHover: true },
  checkboxConfig: { highlight: true },
  loading: loading.value,
  data: list.value,
  columns: [
    { type: 'checkbox', width: 55, fixed: 'left' },
    { type: 'seq', title: '序号', width: 55, fixed: 'left' },
    { field: 'name', title: '名称', minWidth: 80, fixed: 'left', slots: { default: 'column-name' } },
    { field: 'type', title: '类型', width: 80, slots: { default: 'column-type' } },
    { title: '监听地址', minWidth: 180, slots: { default: 'column-listen' } },
    { title: '目标地址', minWidth: 180, slots: { default: 'column-target' } },
    { field: 'isEnable', title: '启用', width: 80, slots: { default: 'column-isEnable' } },
    { field: 'tags', title: '标签', minWidth: 80, slots: { default: 'column-tags' } },
    { field: 'remark', title: '备注', minWidth: 120 },
    { field: 'createTime', title: '创建时间', minWidth: 180 },
    { title: '操作', width: 160, fixed: 'right', showOverflow: false, slots: { default: 'column-operation' } },
  ],
}))

const reset = () => {
  queryFormRef.value?.resetFields()
  pagination.value.page = 1
  getList()
}

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listPortForwards({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.value.keywords || undefined,
      type: queryForm.value.type || undefined,
      isEnable: queryForm.value.isEnable,
    })
    list.value = data?.infos || []
    pagination.value.total = Number(data?.total) || 0
  } finally {
    loading.value = false
  }
}

const selectionChange = ({ records }: { records: PortForwardInfo[] }) => {
  deleteIds.value = records.map((item) => item.id)
}

const toggleMobileSelect = (id: string) => {
  const idx = deleteIds.value.indexOf(id)
  if (idx === -1) {
    deleteIds.value = [...deleteIds.value, id]
  } else {
    deleteIds.value = deleteIds.value.filter((d) => d !== id)
  }
}

const openCreateDialog = () => {
  createRef.value?.showDialog()
}

const openEditDialog = (row: PortForwardInfo) => {
  createRef.value?.showDialog(row)
}

const toggleEnable = async (row: PortForwardInfo, val: boolean) => {
  try {
    const { data } = await updatePortForward({ id: row.id, isEnable: val })
    if (data?.info) {
      row.isEnable = data.info.isEnable
      row.error = data.info.error
      if (data.info.error) {
        ElMessage.error(data.info.error)
      } else {
        ElMessage.success(data.info.isEnable ? '已启用' : '已禁用')
      }
    }
  } catch {
    // error handled by interceptor
  }
}

const confirmDelete = (row: PortForwardInfo) => {
  ElMessageBox.confirm('确定要删除该规则吗？', '确认删除', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消',
  })
    .then(async () => {
      await deletePortForward({ id: row.id })
      ElMessage.success('删除成功')
      deleteIds.value = deleteIds.value.filter((d) => d !== row.id)
      getList()
    })
    .catch(() => { /* cancelled */ })
}

const batchDelete = async () => {
  if (!deleteIds.value.length) return
  try {
    await Promise.all(deleteIds.value.map((id) => deletePortForward({ id })))
    ElMessage.success('批量删除成功')
    deleteIds.value = []
    getList()
  } catch {
    // error handled by interceptor
  }
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

<style scoped lang="scss">
.pf-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.pf-actions :deep(.el-button) {
  margin-left: 0;
}

.mono-text {
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', monospace;
  font-size: 0.88rem;
}

.tag-item {
  margin-right: 4px;
  margin-bottom: 2px;
}

.error-hint {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  font-size: 10px;
  font-weight: 700;
  line-height: 1;
  color: var(--el-color-danger);
  border: 1px solid var(--el-color-danger);
  border-radius: 50%;
  cursor: help;
  vertical-align: middle;
  margin-left: 4px;
  flex-shrink: 0;
}

.text-placeholder {
  color: var(--el-text-color-placeholder);
}

/* ===== mobile ===== */
.mobile-card-list {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.mobile-empty {
  padding: 1.5rem 0;
}

.pf-card {
  position: relative;
  padding: 0.75rem 0.85rem;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.65rem;
  background: var(--el-bg-color);
  transition: border-color 0.15s;
}

.pf-card.is-selected {
  border-color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 4%, var(--el-bg-color));
}

.pf-card-check {
  position: absolute;
  top: 0.6rem;
  right: 0.7rem;
  z-index: 1;
}

.pf-card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding-right: 2rem;
}

.pf-card-name {
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.pf-card-body {
  margin-top: 0.55rem;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.pf-card-meta {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.76rem;
  color: var(--el-text-color-secondary);
}

.pf-card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-top: 0.15rem;
}

.pf-card-remark {
  font-size: 0.74rem;
  color: var(--el-text-color-placeholder);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pf-card-time {
  font-size: 0.7rem;
  color: var(--el-text-color-placeholder);
}

.pf-card-footer {
  display: flex;
  gap: 0.4rem;
  margin-top: 0.6rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--el-border-color-extra-light);
}

/* ===== mobile batch bar ===== */
.mobile-batch-bar {
  position: sticky;
  bottom: 0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0.65rem -0.5rem -0.5rem;
  padding: 0.6rem 0.75rem;
  background: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color);
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
</style>
