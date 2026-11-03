<!-- 应用类型（第二期）：PageHeader(类型数 · 被多少应用使用) + 应用子导航 + 来源 chips(带数量) + 搜索
     + 信息卡网格（贴纸图标 / 说明 / 应用数 / 默认启动命令 / 停止命令 / 在用的应用 / 行内删除确认 / 图标操作）
     + 末尾虚线「新建类型」卡。内置类型（isSystem）锁定：不可编辑/删除。
     「用这个类型新建应用」直接在本页打开应用编辑器（与应用列表共用 useInstanceListStore 受控契约）。 -->
<template>
  <div class="itype-page">
    <PageHeader :title="t('instance.ui.appTypes')">
      <template #description>
        {{ t('instance.ui.typesDesc', { types: instanceTypeList.length, apps: totalApps })
        }}<span class="itype-hide-sm">{{ t('instance.ui.typesDescTail') }}</span>
      </template>
      <template #actions>
        <UButton color="neutral" variant="soft" icon="i-lucide-rotate-cw" :loading="loading" @click="getInstanceTypeList">
          <span class="itype-hide-sm">{{ t('common.refresh') }}</span>
        </UButton>
        <UButton color="primary" icon="i-lucide-plus" @click="openEditor()">
          {{ t('instance.ui.newType') }}
        </UButton>
      </template>
    </PageHeader>

    <!-- 工具条：来源/状态 chips（带数量）+ 搜索 -->
    <div class="itype-tools">
      <div class="itype-chips" role="tablist" :aria-label="t('instance.ui.filterBySource')">
        <button
          v-for="chip in chips"
          :key="chip.value"
          type="button"
          role="tab"
          class="itype-chip"
          :class="{ 'is-on': filter === chip.value }"
          :aria-selected="filter === chip.value"
          @click="filter = chip.value"
        >
          {{ chip.label }} <span class="itype-chip__n m-num">{{ chip.count }}</span>
        </button>
      </div>
      <span class="itype-tools__spacer" />
      <label class="itype-search">
        <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" />
        <input
          v-model="keyword"
          type="search"
          :placeholder="t('instance.ui.searchType')"
          :aria-label="t('instance.ui.searchType')"
        />
      </label>
    </div>

    <!-- 加载骨架 -->
    <div v-if="loading && !instanceTypeList.length" class="itype-grid">
      <div v-for="i in 6" :key="i" class="itype-skeleton" />
    </div>

    <EmptyState
      v-else-if="!instanceTypeList.length"
      icon="HOutline:CubeIcon"
      :title="t('instance.ui.typeEmptyTitle')"
      :description="t('instance.ui.newTypeCardDesc')"
    >
      <template #action>
        <UButton color="primary" icon="i-lucide-plus" @click="openEditor()">{{ t('instance.ui.newType') }}</UButton>
      </template>
    </EmptyState>

    <EmptyState
      v-else-if="!filteredList.length"
      icon="HOutline:MagnifyingGlassIcon"
      :title="t('instance.ui.noMatchedTypes')"
      :description="t('instance.ui.noMatchedTypesDesc')"
    />

    <div v-else class="itype-grid">
      <article
        v-for="(row, idx) in filteredList"
        :key="row.id"
        class="itype-card"
        :class="{ 'is-disabled': !row.isEnable, 'is-confirming': confirmId === row.id }"
      >
        <header class="itype-card__head">
          <span
            class="itype-card__av m-sticker"
            :class="`m-sticker--${TONES[idx % TONES.length]}`"
            :style="{ transform: `rotate(${ROTS[idx % ROTS.length]})` }"
          >
            <UIcon :name="look(row).icon" class="itype-card__av-ico" />
          </span>
          <span class="itype-card__title">
            <span class="itype-card__name-row">
              <span class="itype-card__name" :title="row.name">{{ row.name }}</span>
              <span class="itype-tag" :class="row.isSystem ? 'is-builtin' : 'is-custom'">
                {{ row.isSystem ? t('instance.builtIn') : t('instance.custom') }}
              </span>
              <StatusPill v-if="!row.isEnable" variant="neutral" :label="t('instance.ui.chipDisabled')" />
            </span>
            <span class="itype-card__desc">{{ row.description || fallbackDesc(row) }}</span>
          </span>
          <span class="itype-card__count">
            <span class="itype-card__count-n m-num">{{ row.appCount || 0 }}</span>
            <span class="itype-card__count-u">{{ t('instance.ui.appsUnit') }}</span>
          </span>
        </header>

        <dl class="itype-card__info">
          <div class="is-wide">
            <dt>{{ t('instance.ui.defaultStartCmd') }}</dt>
            <dd class="itype-cmd">
              <code v-if="row.startCommand" class="m-mono">{{ row.startCommand }}</code>
              <span v-else class="itype-dim itype-cmd__empty">{{ t('instance.ui.notPreset') }}</span>
              <AppIconButton
                v-if="row.startCommand"
                icon="HOutline:ClipboardDocumentIcon"
                :size="16"
                :box="36"
                :label="t('instance.ui.copyStartCmd', { name: row.name })"
                @click="copy(row.startCommand)"
              />
            </dd>
          </div>
          <div>
            <dt>{{ t('instance.ui.stopMethod') }}</dt>
            <dd>
              <span v-if="row.stopCommand" class="itype-kbd m-mono">{{ row.stopCommand }}</span>
              <span v-else class="itype-dim">{{ t('instance.ui.stopBySignal') }}</span>
            </dd>
          </div>
          <div>
            <dt>{{ t('instance.typeId') }}</dt>
            <dd class="itype-id">
              <span class="m-mono" :title="row.id">{{ row.id }}</span>
            </dd>
          </div>
        </dl>

        <div class="itype-card__apps">
          <span class="itype-card__apps-label">{{ t('instance.ui.usedBy') }}</span>
          <div v-if="row.apps?.length" class="itype-apps">
            <RouterLink
              v-for="app in row.apps"
              :key="app.id"
              class="itype-app"
              :to="{ path: `/instance/app/${app.id}`, query: { tabTitle: app.name } }"
              :title="app.name"
            >
              <span class="itype-app__l m-sticker" :class="`m-sticker--${toneOf(app.id)}`">{{ initialOf(app.name) }}</span>
              <span class="itype-app__n">{{ app.name }}</span>
              <span class="itype-app__dot" :class="{ 'is-run': app.status === InstanceStatus.INSTANCE_STATUS_RUNNING }" />
            </RouterLink>
            <button
              v-if="(row.appCount || 0) > row.apps.length"
              type="button"
              class="itype-app is-more"
              @click="viewApps(row)"
            >
              +{{ (row.appCount || 0) - row.apps.length }}
            </button>
          </div>
          <span v-else class="itype-dim itype-card__apps-empty">{{ t('instance.ui.noAppsUsing') }}</span>
        </div>

        <!-- 行内删除确认 -->
        <div v-if="confirmId === row.id" class="itype-confirm" role="alert">
          <template v-if="row.appCount">
            <span class="itype-confirm__text">
              {{ t('instance.ui.typeInUse', { count: row.appCount, name: row.name }) }}
              {{ t('instance.ui.typeInUseHint') }}
            </span>
            <span class="itype-confirm__acts">
              <UButton color="neutral" variant="outline" size="sm" @click="confirmId = ''">{{ t('common.cancel') }}</UButton>
              <UButton color="neutral" variant="soft" size="sm" @click="viewApps(row)">{{ t('instance.ui.viewApps') }}</UButton>
            </span>
          </template>
          <template v-else>
            <span class="itype-confirm__text">{{ t('instance.ui.confirmDeleteType', { name: row.name }) }}</span>
            <span class="itype-confirm__acts">
              <UButton color="neutral" variant="outline" size="sm" @click="confirmId = ''">{{ t('common.cancel') }}</UButton>
              <UButton color="error" size="sm" :loading="deleting" @click="doDelete(row)">{{ t('instance.ui.confirmDelete') }}</UButton>
            </span>
          </template>
        </div>

        <footer class="itype-card__foot">
          <span class="itype-card__upd m-num">{{ updatedText(row) }}</span>
          <AppIconButton
            icon="HOutline:PlusIcon"
            :label="t('instance.ui.newAppWithType')"
            :disabled="!row.isEnable"
            @click="newAppWithType(row)"
          />
          <AppIconButton
            v-if="!row.isSystem"
            icon="HOutline:PencilSquareIcon"
            :label="t('common.edit')"
            @click="openEditor(row)"
          />
          <ActionMenu :items="rowActions(row)" @select="(key) => onRowAction(key, row)" />
        </footer>
      </article>

      <!-- 新建类型卡 -->
      <button type="button" class="itype-add" @click="openEditor()">
        <span class="itype-add__ico m-sticker m-sticker--solid-p">
          <UIcon name="i-lucide-plus" />
        </span>
        <span class="itype-add__t">{{ t('instance.ui.newType') }}</span>
        <span class="itype-add__d">{{ t('instance.ui.newTypeCardDesc') }}</span>
      </button>
    </div>

    <InstanceTypeCreate ref="editorRef" @refresh="getInstanceTypeList" />

    <!-- 「用这个类型新建应用」：复用应用列表的受控编辑器 -->
    <InstanceEditor
      :visible="instanceEditorVisible"
      :mode="instanceEditorMode"
      :loading="instanceEditorLoading"
      :submitting="instanceEditorSubmitting"
      :form="instanceEditorForm"
      :type-options="typeOptions"
      @close="closeInstanceEditor"
      @submit="handleEditorSubmit"
    />
  </div>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useInstanceTypeStore } from '@/stores/instance/type'
import { useInstanceListStore } from '@/stores/instance/list'
import { InstanceStatus, type InstanceEditorFormValue } from '@/stores/instance/types'
import { showRequestError } from '@/utils/request'
import { copyTextToClipboard } from '@/utils/file'
import type { InstanceTypeInfo } from '@/types/v1/instance'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import InstanceTypeCreate from '@/views/instance/type/create.vue'
import InstanceEditor from '@/views/instance/list/instanceEditor.vue'

defineOptions({ name: 'TypeView' })

const menuStore = useMenuStore()
const router = useRouter()
const instanceTypeStore = useInstanceTypeStore()
const instanceListStore = useInstanceListStore()
const { t } = useI18n()

const editorRef = useTemplateRef<InstanceType<typeof InstanceTypeCreate> | null>('editorRef')

const { loading, instanceTypeList } = storeToRefs(instanceTypeStore)
const { getInstanceTypeList, deleteTypeById } = instanceTypeStore

const {
  instanceEditorVisible,
  instanceEditorMode,
  instanceEditorLoading,
  instanceEditorSubmitting,
  instanceEditorForm,
  typeOptions,
  queryForm,
} = storeToRefs(instanceListStore)
const { closeInstanceEditor } = instanceListStore

// —— 客户端筛选（store 只加载全量，视图侧过滤）——
type Filter = 'all' | 'builtin' | 'custom' | 'disabled'
const filter = ref<Filter>('all')
const keyword = ref('')

const totalApps = computed(() => instanceTypeList.value.reduce((n, r) => n + Number(r.appCount || 0), 0))

const chips = computed(() => {
  const list = instanceTypeList.value
  const items: { value: Filter; label: string; count: number }[] = [
    { value: 'all', label: t('instance.ui.chipAll'), count: list.length },
    { value: 'builtin', label: t('instance.builtIn'), count: list.filter((r) => r.isSystem).length },
    { value: 'custom', label: t('instance.custom'), count: list.filter((r) => !r.isSystem).length },
  ]
  const disabled = list.filter((r) => !r.isEnable).length
  if (disabled > 0 || filter.value === 'disabled') {
    items.push({ value: 'disabled', label: t('instance.ui.chipDisabled'), count: disabled })
  }
  return items
})

const filteredList = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return instanceTypeList.value.filter((row) => {
    if (filter.value === 'builtin' && !row.isSystem) return false
    if (filter.value === 'custom' && row.isSystem) return false
    if (filter.value === 'disabled' && row.isEnable) return false
    if (kw && !`${row.name} ${row.id} ${row.description || ''} ${row.startCommand || ''}`.toLowerCase().includes(kw)) {
      return false
    }
    return true
  })
})

// —— 卡片外观：按名称猜一个图标，按 id 稳定取贴纸色 ——
const TONES = ['p', 'a2', 'a', 'ok'] as const
const ROTS = ['-5deg', '4deg', '-3deg', '3deg']
const hashOf = (s: string) => [...(s || '')].reduce((n, c) => (n * 31 + c.charCodeAt(0)) >>> 0, 7)
const toneOf = (id: string) => TONES[hashOf(id) % TONES.length]
const initialOf = (name: string) => (name || '?').trim().charAt(0).toUpperCase()

const ICON_RULES: [RegExp, string][] = [
  [/java|jar|minecraft|spring/i, 'i-lucide-coffee'],
  [/node|npm|pnpm|bun|deno|js/i, 'i-lucide-hexagon'],
  [/python|py\b|django|flask/i, 'i-lucide-code-xml'],
  [/go\b|golang|rust|二进制|binary|exe|bin/i, 'i-lucide-cpu'],
  [/game|游戏|palworld|terraria|steam/i, 'i-lucide-gamepad-2'],
  [/php|web|nginx|http/i, 'i-lucide-globe'],
  [/\.net|dotnet|c#/i, 'i-lucide-box'],
  [/shell|bash|脚本|script|cmd/i, 'i-lucide-terminal'],
]
const look = (row: InstanceTypeInfo) => ({
  icon: ICON_RULES.find(([re]) => re.test(`${row.name} ${row.startCommand || ''}`))?.[1] || 'i-lucide-layers',
})

const fallbackDesc = (row: InstanceTypeInfo) =>
  row.isSystem ? t('instance.ui.builtInLocked') : t('instance.ui.noTypeDesc')

const updatedText = (row: InstanceTypeInfo) => {
  if (row.isSystem) return t('instance.ui.builtInUpdated')
  const d = row.updateTime ? dayjs(row.updateTime as unknown as string) : null
  return d && d.isValid() ? t('instance.ui.updatedAt', { time: d.format('YYYY-MM-DD HH:mm') }) : ''
}

// —— 操作 ——
const rowActions = (row: InstanceTypeInfo): ActionMenuItem[] => [
  { key: 'copyId', label: t('instance.ui.copyId'), icon: 'HOutline:ClipboardDocumentIcon' },
  { key: 'duplicate', label: t('instance.ui.duplicateType'), icon: 'HOutline:Square2StackIcon' },
  {
    key: 'viewApps',
    label: t('instance.ui.viewApps'),
    icon: 'HOutline:Squares2X2Icon',
    hidden: !row.appCount,
  },
  { key: 'delete', label: t('common.delete'), icon: 'HOutline:TrashIcon', danger: true, hidden: row.isSystem },
]

const onRowAction = (key: string, row: InstanceTypeInfo) => {
  if (key === 'copyId') copy(row.id)
  else if (key === 'duplicate') editorRef.value?.showDialog(undefined, row)
  else if (key === 'viewApps') viewApps(row)
  else if (key === 'delete') confirmId.value = row.id
}

const openEditor = (row?: InstanceTypeInfo) => editorRef.value?.showDialog(row)

const copy = async (text: string) => {
  try {
    await copyTextToClipboard(text)
    feedback.success(t('instance.ui.copied'))
  } catch {
    feedback.error(t('instance.ui.copyFailed'))
  }
}

const confirmId = ref('')
const deleting = ref(false)
const doDelete = async (row: InstanceTypeInfo) => {
  deleting.value = true
  try {
    await deleteTypeById(row.id)
    confirmId.value = ''
    feedback.success(t('common.deleteSuccess'))
  } catch (error) {
    showRequestError(error, t('instance.ui.deleteTypeFailed'))
  } finally {
    deleting.value = false
  }
}

// 跳到应用列表并按该类型筛选（list store 的筛选是共享状态）
const viewApps = async (row: InstanceTypeInfo) => {
  confirmId.value = ''
  queryForm.value.type = row.id
  // applyFilters 同步写入已生效筛选，列表页挂载后按该类型加载
  void instanceListStore.applyFilters()
  await router.push('/instance/list')
}

const newAppWithType = async (row: InstanceTypeInfo) => {
  if (!row.isEnable) return
  try {
    await instanceListStore.getInstanceTypeList()
    instanceListStore.openCreateEditor()
    const opt = typeOptions.value.find((o) => o.value === row.id)
    instanceEditorForm.value = {
      ...instanceEditorForm.value,
      type: row.id,
      startCommand: opt?.startCommand || instanceEditorForm.value.startCommand,
      stopCommand: opt?.stopCommand || instanceEditorForm.value.stopCommand,
    }
  } catch (error) {
    showRequestError(error, t('instance.loadConfigFailed'))
  }
}

const handleEditorSubmit = async (form: InstanceEditorFormValue) => {
  try {
    instanceEditorForm.value = { ...form }
    await instanceListStore.submitInstanceEditor()
    feedback.success(t('instance.createSuccess'))
    void getInstanceTypeList()
  } catch (error) {
    showRequestError(error, t('instance.createFailed'))
  }
}

onMounted(() => {
  void getInstanceTypeList()
})

onBeforeUnmount(() => {
  if (instanceEditorVisible.value) instanceListStore.closeInstanceEditor()
})
</script>

<style scoped lang="scss">
.itype-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 工具条 */
.itype-tools {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.itype-tools__spacer {
  flex: 1;
}
.itype-chips {
  display: flex;
  gap: 8px;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.itype-chips::-webkit-scrollbar {
  display: none;
}
.itype-chip {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}
.itype-chip:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.itype-chip.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.itype-chip__n {
  font-size: 0.75rem;
  opacity: 0.7;
}
.itype-search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 300px;
  min-width: 180px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  transition: border-color 0.15s;
}
.itype-search:focus-within {
  border-color: var(--m-p);
}
.itype-search :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
}
.itype-search input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
}

/* 网格 */
.itype-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 16px;
  align-items: start;
}

/* 卡片 */
.itype-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: 20px 14px 6px 20px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.itype-card.is-confirming {
  border-color: var(--m-bad);
  box-shadow: 0 0 0 3px var(--m-badSoft);
}
.itype-card.is-disabled .itype-card__av {
  filter: grayscale(0.7);
  opacity: 0.7;
}
.itype-card__head {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding-right: 6px;
  min-width: 0;
}
.itype-card__av {
  width: 56px;
  height: 56px;
  border-radius: 18px;
  flex: none;
}
.itype-card__av-ico {
  width: 28px;
  height: 28px;
}
.itype-card__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.itype-card__name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 8px;
  min-width: 0;
}
.itype-card__name {
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}
.itype-tag {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  font-size: 0.75rem;
  font-weight: 700;
  white-space: nowrap;
}
.itype-tag.is-builtin {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.itype-tag.is-custom {
  background: var(--m-aSoft);
  color: var(--m-ink);
}
.itype-card__desc {
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.itype-card__count {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  flex: none;
}
.itype-card__count-n {
  font-family: var(--app-font-display);
  font-size: 1.625rem;
  font-weight: 800;
  line-height: 1;
  color: var(--el-text-color-primary);
}
.itype-card__count-u {
  margin-top: 2px;
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

/* 信息格 */
.itype-card__info {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 12px;
  margin: 0 6px 0 0;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
}
.itype-card__info > div {
  min-width: 0;
}
.itype-card__info .is-wide {
  grid-column: 1 / -1;
}
.itype-card__info dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.itype-card__info dd {
  margin: 3px 0 0;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  min-width: 0;
}
.itype-cmd {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 1px !important;
}
.itype-cmd code {
  flex: 1;
  min-width: 0;
  padding: 4px 0;
  font-size: 0.78rem;
  font-weight: 500;
  line-height: 1.5;
  word-break: break-all;
}
.itype-cmd__empty {
  padding: 4px 0;
  font-size: 0.8125rem;
}
.itype-kbd {
  display: inline-flex;
  padding: 1px 8px;
  border-radius: 6px;
  background: var(--el-bg-color);
  font-size: 0.75rem;
}
.itype-id span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.72rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.itype-dim {
  color: var(--el-text-color-secondary);
  font-weight: 500;
}

/* 在用的应用 */
.itype-card__apps {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-right: 6px;
}
.itype-card__apps-label {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.itype-card__apps-empty {
  font-size: 0.8125rem;
}
.itype-apps {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.itype-app {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  height: 32px;
  padding: 0 10px 0 4px;
  border: none;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.75rem;
  font-weight: 700;
  text-decoration: none;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.itype-app:hover {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.itype-app.is-more {
  padding: 0 12px;
}
.itype-app__l {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  font-size: 0.6875rem;
  flex: none;
}
.itype-app__n {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.itype-app__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex: none;
  background: var(--el-text-color-secondary);
  opacity: 0.5;
}
.itype-app__dot.is-run {
  background: var(--m-ok);
  opacity: 1;
}

/* 行内删除确认 */
.itype-confirm {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-right: 6px;
  padding: 12px 14px;
  border-radius: 16px;
  background: var(--m-badSoft);
}
.itype-confirm__text {
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.5;
  color: var(--m-bad);
}
.itype-confirm__acts {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
}

/* 底部 */
.itype-card__foot {
  display: flex;
  align-items: center;
  gap: 2px;
  padding-top: 6px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.itype-card__upd {
  flex: 1;
  min-width: 0;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.itype-card__foot :deep(.action-menu__trigger) {
  width: 40px;
  height: 40px;
  border-radius: 14px;
}
.itype-card__foot :deep(.app-icon-btn:disabled) {
  opacity: 0.4;
  cursor: not-allowed;
}

/* 新建类型卡 */
.itype-add {
  min-height: 300px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 24px;
  border-radius: var(--app-radius-lg);
  border: 2px dashed var(--el-border-color-light);
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.itype-add:hover {
  border-color: var(--m-p);
  background: var(--m-pSoft);
}
.itype-add__ico {
  width: 56px;
  height: 56px;
  border-radius: 18px;
  transform: rotate(-6deg);
}
.itype-add__ico :deep(svg),
.itype-add__ico > span {
  width: 26px;
  height: 26px;
}
.itype-add__t {
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.itype-add__d {
  max-width: 240px;
  font-size: 0.8125rem;
  line-height: 1.5;
  text-align: center;
}

.itype-skeleton {
  height: 300px;
  border-radius: var(--app-radius-lg);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-bg-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: itype-shimmer 1.4s ease-in-out infinite;
}
@keyframes itype-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}

@media (width < 768px) {
  .itype-tools__spacer {
    display: none;
  }
  .itype-search {
    order: -1;
    flex: 1 1 100%;
    min-width: 0;
  }
  .itype-chips {
    width: 100%;
  }
  .itype-grid {
    grid-template-columns: minmax(0, 1fr);
  }
  .itype-card {
    padding: 16px 10px 6px 16px;
  }
  .itype-card__av {
    width: 48px;
    height: 48px;
    border-radius: 16px;
  }
  .itype-card__av-ico {
    width: 24px;
    height: 24px;
  }
  .itype-add {
    min-height: 160px;
  }
  .itype-hide-sm {
    display: none;
  }
  .itype-card__foot :deep(.app-icon-btn),
  .itype-card__foot :deep(.action-menu__trigger) {
    width: 44px !important;
    height: 44px !important;
  }
}
</style>
