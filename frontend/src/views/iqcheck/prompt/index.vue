<!-- 智商检测 · 提示词配置：PageHeader(新增) + FilterBar + DataTable / 移动卡片 + Pagination。
     子弹窗：promptForm(创建编辑)。行内 编辑/删除（Dialog.info 确认）。文案走 iqcheck.prompt.*。 -->
<template>
  <div class="iq-prompt-page">
    <PageHeader :title="t('iqcheck.prompt.title')" :description="t('iqcheck.prompt.pageDesc')">
      <template #actions>
        <UButton v-permission="[PERM.IQCHECK_EDIT]" color="primary" icon="i-lucide-plus" @click="openForm()">
          {{ t('iqcheck.prompt.create') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="reload" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('iqcheck.prompt.colTitle') }}</label>
          <input
            v-model="queryForm.keywords"
            class="app-input"
            :placeholder="t('iqcheck.prompt.keywordsPlaceholder')"
            @keyup.enter="reload"
          />
        </div>
      </template>
    </FilterBar>

    <div class="iq-prompt-page__body">
      <div class="iq-prompt-page__bar">
        <span class="iq-prompt-page__hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
      </div>

      <DataTable v-if="!menuStore.isMobile" :columns="columns" :rows="prompts" row-key="id" :loading="loading">
        <template #cell-content="{ row }">
          <span class="iq-prompt-content" :title="String(row.content)">{{ row.content }}</span>
        </template>
        <template #cell-enabled="{ row }">
          <StatusPill
            :variant="row.enabled ? 'success' : 'neutral'"
            :label="row.enabled ? t('system.common.enabled') : t('system.common.inactive')"
          />
        </template>
        <template #cell-updateTime="{ row }">{{ fmtTime(row.updateTime) }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu v-if="canEdit" :items="rowActions" @select="(key) => onRowAction(key, row)" />
          <span v-else class="text-dim">—</span>
        </template>
      </DataTable>

      <template v-else>
        <div v-if="loading" class="iq-prompt-cards">
          <div v-for="i in 4" :key="i" class="iq-prompt-skeleton" />
        </div>
        <EmptyState
          v-else-if="!prompts.length"
          icon="HOutline:ChatBubbleLeftRightIcon"
          :title="t('iqcheck.prompt.emptyTitle')"
          :description="t('iqcheck.prompt.emptyDesc')"
        />
        <div v-else class="iq-prompt-cards">
          <EntityCard v-for="row in prompts" :key="row.id">
            <template #title>{{ row.title }}</template>
            <template #status>
              <StatusPill
                :variant="row.enabled ? 'success' : 'neutral'"
                :label="row.enabled ? t('system.common.enabled') : t('system.common.inactive')"
              />
            </template>
            <template #meta>
              <span class="iq-prompt-card__content">{{ row.content }}</span>
            </template>
            <template #footer>
              <span>{{ fmtTime(row.updateTime) }}</span>
              <ActionMenu v-if="canEdit" :items="rowActions" @select="(key) => onRowAction(key, row)" />
            </template>
          </EntityCard>
        </div>
      </template>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="loadPrompts"
      />
    </div>

    <PromptForm ref="formRef" @refresh="loadPrompts" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { listIQCheckPrompts, deleteIQCheckPrompt } from '@/api/iqcheck'
import { PERM } from '@/config/permission'
import { Dialog } from '@/utils/dialog'
import type { IQCheckPromptInfo } from '@/types/v1/iqcheck'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import PromptForm from '@/views/iqcheck/prompt/promptForm.vue'

defineOptions({ name: 'IQCheckPromptView' })

const menuStore = useMenuStore()
const { t } = useI18n()

const formRef = useTemplateRef<InstanceType<typeof PromptForm> | null>('formRef')
const canEdit = computed(() => menuStore.hasButtonPermission(PERM.IQCHECK_EDIT))

const loading = ref(false)
const prompts = ref<IQCheckPromptInfo[]>([])
const queryForm = ref({ keywords: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const columns = computed<DataTableColumn[]>(() => [
  { key: 'title', title: t('iqcheck.prompt.colTitle'), minWidth: 160 },
  { key: 'content', title: t('iqcheck.prompt.colContent'), minWidth: 320 },
  { key: 'enabled', title: t('iqcheck.prompt.colEnabled'), width: 90 },
  { key: 'order', title: t('iqcheck.prompt.colOrder'), width: 80, align: 'center' },
  { key: 'updateTime', title: t('iqcheck.prompt.colUpdateTime'), width: 160 },
  { key: 'operation', title: t('system.common.operation'), width: 70, align: 'center' },
])

const rowActions = computed<ActionMenuItem[]>(() => [
  { key: 'edit', label: t('system.common.edit'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'delete', label: t('system.common.delete'), icon: 'HOutline:TrashIcon', danger: true },
])

const fmtTime = (v: unknown) => (v ? dayjs(v as string | Date).format('YYYY-MM-DD HH:mm') : '—')

const loadPrompts = async () => {
  loading.value = true
  try {
    const { data } = await listIQCheckPrompts({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.value.keywords || undefined,
    })
    prompts.value = data?.prompts || []
    pagination.value.total = Number(data?.total || 0)
  } finally {
    loading.value = false
  }
}

const reload = () => {
  pagination.value.page = 1
  loadPrompts()
}
const reset = () => {
  queryForm.value.keywords = ''
  pagination.value.page = 1
  loadPrompts()
}

const openForm = (row?: IQCheckPromptInfo) => formRef.value?.showDialog(row)

const onRowAction = (key: string, row: Record<string, unknown>) => {
  const prompt = prompts.value.find((p) => p.id === String(row.id))
  if (!prompt) return
  if (key === 'edit') openForm(prompt)
  else if (key === 'delete') confirmDelete(prompt)
}

const confirmDelete = (prompt: IQCheckPromptInfo) => {
  Dialog.info({
    showCancelButton: true,
    content: t('iqcheck.prompt.deleteConfirm', { title: prompt.title }),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    onConfirm: async () => {
      await deleteIQCheckPrompt({ id: prompt.id })
      feedback.success(t('common.deleteSuccess'))
      loadPrompts()
    },
  })
}

onMounted(loadPrompts)
</script>

<style scoped lang="scss">
.iq-prompt-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.iq-prompt-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.iq-prompt-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.iq-prompt-page__hint {
  font-size: 0.78rem;
  color: var(--el-text-color-secondary);
}
.iq-prompt-content {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  white-space: pre-wrap;
  word-break: break-word;
}
.iq-prompt-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.iq-prompt-card__content {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--el-text-color-regular);
}
.iq-prompt-skeleton {
  height: 96px;
  border-radius: 12px;
  background: var(--el-fill-color-light);
}
</style>
