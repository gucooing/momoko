<!-- 智商检测 · 提示词：标题 + 新建提示词；子导航（站点 / 提示词）；说明 + 搜索；
     提示词卡片：贴纸图标 + 标题 + 使用情况（已用于 N 个站点 · 共 N 次 · 排序）+ 启用开关（有编辑权限可直接切换）；
     内容预览块；底部标签（更新时间 / 字数 / 最近使用）+ 编辑 / 复制 / ⋯（删除）。子弹窗：promptForm。 -->
<template>
  <div class="iq-page">
    <PageHeader :title="t('iqcheck.nav.title')">
      <template #description>
        {{ t('iqcheck.promptList.summary', { total: allCount, enabled: enabledCount }) }}
      </template>
      <template #actions>
        <UButton v-permission="[PERM.IQCHECK_EDIT]" color="primary" icon="i-lucide-plus" @click="openForm()">
          {{ t('iqcheck.prompt.create') }}
        </UButton>
      </template>
    </PageHeader>

    <IqSubNav :prompt-count="allCount" />

    <ListToolbar
      v-model:keyword="keyword"
      :placeholder="t('iqcheck.prompt.keywordsPlaceholder')"
      @search="reload"
    >
      <template #default>
        <p class="iq-prompt__hint">{{ t('iqcheck.promptList.hint') }}</p>
      </template>
    </ListToolbar>

    <div v-if="loading && !prompts.length" class="iq-prompts">
      <div v-for="i in 3" :key="i" class="iq-skeleton" />
    </div>
    <EmptyState
      v-else-if="!prompts.length"
      icon="HOutline:ChatBubbleLeftRightIcon"
      :title="t('iqcheck.prompt.emptyTitle')"
      :description="t('iqcheck.prompt.emptyDesc')"
    >
      <template v-if="canEdit && !keyword" #action>
        <UButton color="primary" variant="soft" icon="i-lucide-plus" @click="openForm()">{{ t('iqcheck.prompt.create') }}</UButton>
      </template>
    </EmptyState>
    <div v-else class="iq-prompts" :class="{ 'is-loading': loading }">
      <article v-for="p in prompts" :key="p.id" class="iq-prompt" :class="{ 'is-off': !p.enabled }">
        <header class="iq-prompt__head">
          <span class="iq-prompt__av m-sticker" :class="`m-sticker--${toneOf(p.id)}`" :style="{ transform: `rotate(${rotOf(p.id)})` }">
            <UIcon :name="iconOf(p)" />
          </span>
          <span class="iq-prompt__title">
            <span class="iq-prompt__name" :title="p.title">{{ p.title }}</span>
            <span class="iq-prompt__meta">{{ usageText(p) }}</span>
          </span>
          <label class="iq-prompt__switch">
            <span>{{ p.enabled ? t('system.common.enabled') : t('system.common.inactive') }}</span>
            <AppSwitch
              :model-value="p.enabled"
              :disabled="!canEdit || togglingId === p.id"
              @update:model-value="(v) => toggleEnabled(p, v)"
            />
          </label>
        </header>

        <div class="iq-prompt__content m-mono">{{ p.content }}</div>

        <footer class="iq-prompt__foot">
          <div class="iq-prompt__tags">
            <span class="iq-tag">{{ t('iqcheck.promptList.updatedAt', { time: fmtDate(p.updateTime) }) }}</span>
            <span class="iq-tag">{{ t('iqcheck.promptList.chars', { n: p.content.length }) }}</span>
            <span v-if="p.lastUsedAt" class="iq-tag">{{ t('iqcheck.promptList.lastUsed', { time: fmtDate(p.lastUsedAt) }) }}</span>
          </div>
          <AppIconButton
            v-if="canEdit"
            icon="HOutline:PencilSquareIcon"
            :label="t('iqcheck.promptList.editOf', { name: p.title })"
            :box="menuStore.isMobile ? 44 : 36"
            :size="18"
            @click="openForm(p)"
          />
          <AppIconButton
            icon="HOutline:DocumentDuplicateIcon"
            :label="t('iqcheck.promptList.copyOf', { name: p.title })"
            :box="menuStore.isMobile ? 44 : 36"
            :size="18"
            @click="copyContent(p)"
          />
          <ActionMenu v-if="canEdit" :items="rowActions" :label="t('iqcheck.siteList.moreOf', { name: p.title })" @select="(key) => onRowAction(key, p)" />
        </footer>
      </article>
    </div>

    <Pagination
      v-if="pagination.total > pagination.pageSize"
      v-model:page="pagination.page"
      v-model:page-size="pagination.pageSize"
      :total="pagination.total"
      @change="loadPrompts"
    />

    <PromptForm ref="formRef" @refresh="loadPrompts" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { listIQCheckPrompts, deleteIQCheckPrompt, updateIQCheckPrompt } from '@/api/iqcheck'
import { PERM } from '@/config/permission'
import { Dialog } from '@/utils/dialog'
import type { IQCheckPromptInfo } from '@/types/v1/iqcheck'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import PromptForm from '@/views/iqcheck/prompt/promptForm.vue'
import IqSubNav from '@/views/iqcheck/components/IqSubNav.vue'

defineOptions({ name: 'IQCheckPromptView' })

const menuStore = useMenuStore()
const { t } = useI18n()

const formRef = useTemplateRef<InstanceType<typeof PromptForm> | null>('formRef')
const canEdit = computed(() => menuStore.hasButtonPermission(PERM.IQCHECK_EDIT))

const loading = ref(false)
const prompts = ref<IQCheckPromptInfo[]>([])
const keyword = ref('')
const pagination = ref({ page: 1, pageSize: 10, total: 0 })
const allCount = ref<number | null>(null)
const enabledCount = ref(0)

const rowActions = computed<ActionMenuItem[]>(() => [
  { key: 'edit', label: t('system.common.edit'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'delete', label: t('system.common.delete'), icon: 'HOutline:TrashIcon', danger: true },
])

const TONES = ['p', 'a2', 'a', 'ok']
const ROTS = ['-4deg', '3deg', '-2deg', '4deg']
const ICONS = ['i-lucide-brain', 'i-lucide-pen-tool', 'i-lucide-code-xml', 'i-lucide-message-square-text']
const hashOf = (id: string) => [...id].reduce((n, c) => (n * 31 + c.charCodeAt(0)) >>> 0, 7)
const toneOf = (id: string) => TONES[hashOf(id) % TONES.length]
const rotOf = (id: string) => ROTS[hashOf(id) % ROTS.length]
// 按内容猜个图标：画图（svg/画）/ 编程（代码/函数）/ 其余按 id 稳定取
const iconOf = (p: IQCheckPromptInfo) => {
  const text = `${p.title}\n${p.content}`.toLowerCase()
  if (/svg|html|画|draw|canvas/.test(text)) return 'i-lucide-pen-tool'
  if (/代码|函数|编程|function|code|typescript|python|sql/.test(text)) return 'i-lucide-code-xml'
  return ICONS[hashOf(p.id) % ICONS.length]!
}

const usageText = (p: IQCheckPromptInfo) => {
  const parts: string[] = []
  if (Number(p.useCount)) parts.push(t('iqcheck.promptList.usedBy', { sites: p.siteCount, count: p.useCount }))
  else parts.push(t('iqcheck.promptList.unused'))
  parts.push(t('iqcheck.promptList.order', { n: p.order }))
  return parts.join(' · ')
}

const fmtDate = (v: unknown) => {
  if (!v) return '—'
  const d = dayjs(v as string | Date)
  if (!d.isValid()) return '—'
  return d.isSame(dayjs(), 'year') ? d.format('MM-DD HH:mm') : d.format('YYYY-MM-DD')
}

const loadPrompts = async () => {
  loading.value = true
  try {
    const { data } = await listIQCheckPrompts({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: keyword.value.trim() || undefined,
    })
    prompts.value = data?.prompts || []
    pagination.value.total = Number(data?.total || 0)
    allCount.value = Number(data?.allCount || 0)
    enabledCount.value = Number(data?.enabledCount || 0)
  } finally {
    loading.value = false
  }
}

const reload = () => {
  pagination.value.page = 1
  loadPrompts()
}

const openForm = (row?: IQCheckPromptInfo) => formRef.value?.showDialog(row)

const onRowAction = (key: string, prompt: IQCheckPromptInfo) => {
  if (key === 'edit') openForm(prompt)
  else if (key === 'delete') confirmDelete(prompt)
}

const togglingId = ref('')
const toggleEnabled = async (p: IQCheckPromptInfo, enabled: boolean) => {
  togglingId.value = p.id
  try {
    await updateIQCheckPrompt({ id: p.id, title: p.title, content: p.content, enabled, order: p.order })
    p.enabled = enabled
    enabledCount.value += enabled ? 1 : -1
    feedback.success(t('iqcheck.prompt.editSuccess'))
  } finally {
    togglingId.value = ''
  }
}

const copyContent = async (p: IQCheckPromptInfo) => {
  try {
    await navigator.clipboard.writeText(p.content)
    feedback.success(t('iqcheck.promptList.copied'))
  } catch {
    feedback.warning(t('terminal.clipboardDenied'))
  }
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
.iq-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.iq-prompt__hint {
  order: -1;
  flex: 1 1 320px;
  margin: 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.iq-prompts {
  display: flex;
  flex-direction: column;
  gap: 14px;
  transition: opacity 0.15s;
}
.iq-prompts.is-loading {
  opacity: 0.6;
}
.iq-prompt {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 20px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
}
.iq-prompt.is-off .iq-prompt__content {
  opacity: 0.6;
}
.iq-prompt__head {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.iq-prompt__av {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  flex: none;
}
.iq-prompt__av :deep(.iconify) {
  width: 20px;
  height: 20px;
}
.iq-prompt__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.iq-prompt__name {
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-prompt__meta {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.iq-prompt__switch {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  flex: none;
  min-height: 44px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.iq-prompt__content {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 6;
  line-clamp: 6;
  padding: 12px 14px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}
.iq-prompt__foot {
  display: flex;
  align-items: center;
  gap: 4px;
}
.iq-prompt__tags {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.iq-tag {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;
}
.iq-skeleton {
  height: 220px;
  border-radius: var(--app-radius-xl);
  background: var(--el-fill-color-light);
}

@media (width <= 768px) {
  .iq-page {
    gap: 14px;
  }
  .iq-prompt {
    padding: 16px 12px 8px 16px;
  }
  .iq-prompt__switch > span {
    display: none;
  }
  .iq-prompt__foot {
    gap: 0;
    padding-top: 6px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
  .iq-prompt__foot :deep(.action-menu__trigger) {
    width: 44px;
    height: 44px;
    border-radius: 14px;
  }
}
</style>
