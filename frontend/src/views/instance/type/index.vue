<!-- 应用类型：页头（共 N 种 · 被 N 个应用使用 + 应用列表 / 新建类型）+ 来源 chips（带数量）+ 搜索
     + 信息卡网格（贴纸图标 / 说明 / 应用数 / 默认启动命令 / 停止方式 / 快捷命令 / 在用的应用 / 行内删除确认 / 小图标操作）
     + 末尾虚线「新建类型」卡（可从模板创建）+ 手机悬浮「新建类型」。内置类型（isSystem）锁定：不可编辑/删除。
     删除仍在用的类型时，这些应用变为「未分类」。「用这个类型新建应用」直接在本页打开应用编辑器。 -->
<template>
  <div class="itype-page">
    <RouterLink v-if="menuStore.isMobile && canSeeApps" to="/instance/list" class="itype-back">
      <component :is="menuStore.iconComponents['HOutline:ChevronLeftIcon']" />
      {{ t('instance.ui.appList') }}
    </RouterLink>

    <!-- 手机端已有上方「应用列表」返回链接，不再叠一个页头返回按钮 -->
    <PageHeader :title="t('instance.ui.appTypes')" :back="canSeeApps ? false : 'auto'">
      <template #description>
        {{ t('instance.ui.typesDesc', { types: instanceTypeList.length, apps: totalApps })
        }}<span class="itype-hide-sm">{{ t('instance.ui.typesDescTail') }}</span>
      </template>
      <template #actions>
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-lucide-rotate-cw"
          class="itype-ico-btn"
          :aria-label="t('common.refresh')"
          :title="t('common.refresh')"
          :loading="loading"
          @click="getInstanceTypeList"
        />
        <template v-if="!menuStore.isMobile">
          <UButton v-if="canSeeApps" color="neutral" variant="outline" icon="i-lucide-package" to="/instance/list">
            {{ t('instance.ui.appList') }}
          </UButton>
          <UButton color="primary" icon="i-lucide-plus" @click="openEditor()">
            {{ t('instance.ui.newType') }}
          </UButton>
        </template>
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
        <input v-model="keyword" type="search" :placeholder="t('instance.ui.searchType')" :aria-label="t('instance.ui.searchType')" />
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
            <UIcon :name="iconOf(row)" class="itype-card__av-ico" />
          </span>
          <span class="itype-card__title">
            <span class="itype-card__name-row">
              <span class="itype-card__name" :title="row.name">{{ row.name }}</span>
              <span class="itype-tag" :class="row.isSystem ? 'is-builtin' : 'is-custom'">
                {{ row.isSystem ? t('instance.builtIn') : t('instance.custom') }}
              </span>
              <StatusPill v-if="!row.isEnable" variant="neutral" :label="t('instance.ui.chipDisabled')" />
            </span>
            <span v-if="!menuStore.isMobile" class="itype-card__desc">{{ row.description || fallbackDesc(row) }}</span>
          </span>
          <span class="itype-card__count">
            <span class="itype-card__count-n m-num">{{ row.appCount || 0 }}</span>
            <span class="itype-card__count-u">{{ t('instance.ui.appsUnit') }}</span>
          </span>
        </header>

        <p v-if="menuStore.isMobile" class="itype-card__desc is-block">{{ row.description || fallbackDesc(row) }}</p>

        <dl class="itype-card__info">
          <div class="is-wide">
            <dt>{{ t('instance.ui.defaultStartCmd') }}</dt>
            <dd class="itype-cmd">
              <code v-if="row.startCommand" class="m-mono">{{ row.startCommand }}</code>
              <span v-else class="itype-dim itype-cmd__empty">{{ t('instance.ui.notPreset') }}</span>
              <AppIconButton
                v-if="row.startCommand && !menuStore.isMobile"
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
            <dt>{{ t('instanceConsole.quickTitle') }}</dt>
            <dd class="m-num">
              {{ row.quickCommands?.length ? t('instance.ui.quickCount', { n: row.quickCommands.length }) : t('instance.ui.none') }}
            </dd>
          </div>
          <div v-if="menuStore.isMobile" class="is-wide">
            <dt>{{ t('instance.ui.updatedLabel') }}</dt>
            <dd class="m-num" :title="updatedTitle(row)">{{ updatedTime(row) || '—' }}</dd>
          </div>
          <div v-if="menuStore.isMobile" class="is-wide">
            <dt>{{ t('instance.ui.usedBy') }}</dt>
            <dd class="itype-names">{{ appNames(row) || t('instance.ui.noAppsUsing') }}</dd>
          </div>
        </dl>

        <!-- PC：在用的应用 chips（可点进详情） -->
        <div v-if="!menuStore.isMobile" class="itype-card__apps">
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
              <span class="itype-app__dot" :class="dotClass(app.status)" />
            </RouterLink>
            <button v-if="(row.appCount || 0) > row.apps.length" type="button" class="itype-app is-more" @click="viewApps(row)">
              +{{ (row.appCount || 0) - row.apps.length }}
            </button>
          </div>
          <span v-else class="itype-dim itype-card__apps-empty">{{ t('instance.ui.noAppsUsing') }}</span>
        </div>

        <!-- 行内删除确认：在用的应用会变为未分类 -->
        <div v-if="confirmId === row.id" class="itype-confirm" role="alert">
          <span class="itype-confirm__text">
            {{
              row.appCount
                ? t('instance.ui.typeInUseDelete', { count: row.appCount, name: row.name })
                : t('instance.ui.confirmDeleteType', { name: row.name })
            }}
          </span>
          <span class="itype-confirm__acts">
            <UButton color="neutral" variant="outline" class="itype-confirm__btn" @click="confirmId = ''">
              {{ t('common.cancel') }}
            </UButton>
            <UButton color="error" class="itype-confirm__btn" :loading="deleting" @click="doDelete(row)">
              {{ t('instance.ui.confirmDelete') }}
            </UButton>
          </span>
        </div>

        <footer class="itype-card__foot">
          <span v-if="!menuStore.isMobile" class="itype-card__upd m-num" :title="updatedTitle(row)">{{ updatedText(row) }}</span>
          <span v-else class="itype-stack">
            <span
              v-for="app in (row.apps || []).slice(0, 4)"
              :key="app.id"
              class="itype-stack__l m-sticker"
              :class="`m-sticker--${toneOf(app.id)}`"
              :title="app.name"
            >
              {{ initialOf(app.name) }}
            </span>
          </span>
          <AppIconButton
            icon="HOutline:PlusIcon"
            :label="t('instance.ui.newAppWithType')"
            :disabled="!row.isEnable"
            @click="newAppWithType(row)"
          />
          <AppIconButton
            v-if="menuStore.isMobile"
            icon="HOutline:ClipboardDocumentIcon"
            :label="t('instance.ui.copyStartCmd', { name: row.name })"
            :disabled="!row.startCommand"
            @click="copy(row.startCommand)"
          />
          <AppIconButton
            v-else
            icon="HOutline:DocumentDuplicateIcon"
            :label="t('instance.ui.duplicateType')"
            @click="editorRef?.showDialog(undefined, row)"
          />
          <AppIconButton v-if="!row.isSystem" icon="HOutline:PencilSquareIcon" :label="t('common.edit')" @click="openEditor(row)" />
          <AppIconButton
            v-if="!row.isSystem"
            icon="HOutline:TrashIcon"
            class="itype-del"
            :label="t('instance.ui.deleteTypeOf', { name: row.name })"
            @click="confirmId = confirmId === row.id ? '' : row.id"
          />
          <ActionMenu :items="rowActions(row)" :label="t('instance.moreActions')" @select="(key) => onRowAction(key, row)" />
        </footer>
      </article>

      <!-- 新建类型卡 -->
      <button type="button" class="itype-add" @click="openEditor()">
        <span class="itype-add__ico m-sticker m-sticker--solid-p">
          <UIcon name="i-lucide-plus" />
        </span>
        <span class="itype-add__t">{{ t('instance.ui.newType') }}</span>
        <span class="itype-add__d">{{ t('instance.ui.newTypeFromTpl') }}<br />Go · .NET · Deno · Bun</span>
      </button>
    </div>

    <!-- 手机：悬浮新建 -->
    <div v-if="menuStore.isMobile" class="itype-fab">
      <UButton color="primary" size="xl" icon="i-lucide-plus" class="itype-fab__btn" @click="openEditor()">
        {{ t('instance.ui.newType') }}
      </UButton>
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
      allow-run
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
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import type { InstanceTypeInfo } from '@/types/v1/instance'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import InstanceTypeCreate from '@/views/instance/type/create.vue'
import InstanceEditor from '@/views/instance/list/instanceEditor.vue'
import { initialOf, toneOf } from '@/views/instance/shared'
import { typeIconOf } from '@/views/instance/type/templates'

defineOptions({ name: 'TypeView' })

const menuStore = useMenuStore()
const router = useRouter()
const instanceTypeStore = useInstanceTypeStore()
const instanceListStore = useInstanceListStore()
const accessible = useAccessibleRoutes()
const { t } = useI18n()

const canSeeApps = computed(() => accessible.value.has('/instance/list'))
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
    if (
      kw &&
      !`${row.name} ${row.id} ${row.description || ''} ${row.startCommand || ''} ${row.stopCommand || ''}`
        .toLowerCase()
        .includes(kw)
    ) {
      return false
    }
    return true
  })
})

// —— 卡片外观：按名称/命令猜图标，贴纸色按序号轮换 ——
const TONES = ['p', 'ok', 'a2', 'a'] as const
const ROTS = ['-5deg', '4deg', '-3deg', '3deg']
const iconOf = (row: InstanceTypeInfo) => typeIconOf(`${row.name} ${row.startCommand || ''}`)
const dotClass = (status: InstanceStatus) => ({
  'is-run': status === InstanceStatus.INSTANCE_STATUS_RUNNING,
  'is-mnt': status === InstanceStatus.INSTANCE_STATUS_MAINTENANCE,
})

const fallbackDesc = (row: InstanceTypeInfo) => (row.isSystem ? t('instance.ui.builtInLocked') : t('instance.ui.noTypeDesc'))
const appNames = (row: InstanceTypeInfo) => {
  const names = (row.apps || []).map((a) => a.name)
  const more = Number(row.appCount || 0) - names.length
  return more > 0 ? `${names.join('、')} +${more}` : names.join('、')
}

// 卡片空间有限：今年内省略年份，完整时间放在 title 里
const updatedTime = (row: InstanceTypeInfo) => {
  if (row.isSystem) return t('instance.ui.builtInUpdated')
  const d = row.updateTime ? dayjs(row.updateTime as unknown as string) : null
  if (!d || !d.isValid()) return ''
  return d.format(d.isSame(dayjs(), 'year') ? 'MM-DD HH:mm' : 'YYYY-MM-DD')
}
const updatedText = (row: InstanceTypeInfo) => {
  const time = updatedTime(row)
  return row.isSystem || !time ? time : t('instance.ui.updatedAt', { time })
}
const updatedTitle = (row: InstanceTypeInfo) => {
  const d = row.updateTime ? dayjs(row.updateTime as unknown as string) : null
  return d && d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : undefined
}

// —— 操作 ——
const rowActions = (row: InstanceTypeInfo): ActionMenuItem[] => [
  { key: 'duplicate', label: t('instance.ui.duplicateType'), icon: 'HOutline:DocumentDuplicateIcon', hidden: !menuStore.isMobile },
  { key: 'copyCmd', label: t('instance.ui.copyStartCmdShort'), icon: 'HOutline:CommandLineIcon', hidden: menuStore.isMobile || !row.startCommand },
  { key: 'copyId', label: t('instance.ui.copyId'), icon: 'HOutline:ClipboardDocumentIcon' },
  { key: 'viewApps', label: t('instance.ui.viewApps'), icon: 'HOutline:Squares2X2Icon', hidden: !row.appCount },
]

const onRowAction = (key: string, row: InstanceTypeInfo) => {
  if (key === 'copyId') copy(row.id)
  else if (key === 'copyCmd') copy(row.startCommand)
  else if (key === 'duplicate') editorRef.value?.showDialog(undefined, row)
  else if (key === 'viewApps') viewApps(row)
}

const openEditor = (row?: InstanceTypeInfo) => editorRef.value?.showDialog(row)

const copy = async (text: string) => {
  if (!text) return
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
    const n = await deleteTypeById(row.id)
    confirmId.value = ''
    feedback.success(n ? t('instance.ui.typeDeletedUncategorized', { n }) : t('common.deleteSuccess'))
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

const handleEditorSubmit = async (form: InstanceEditorFormValue, options?: { run: boolean }) => {
  try {
    instanceEditorForm.value = { ...form }
    const saved = await instanceListStore.submitInstanceEditor()
    feedback.success(t('instance.createSuccess'))
    void getInstanceTypeList()
    if (options?.run && saved?.id) {
      if (await instanceListStore.changeInstanceStatus(saved.id, InstanceStatus.INSTANCE_STATUS_RUNNING)) {
        feedback.success(t('instance.instanceStarted'))
      }
    }
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
.itype-page .itype-ico-btn {
  width: 40px;
  padding-inline: 0;
  justify-content: center;
}
@media (width <= 768px) {
  .itype-page .itype-ico-btn {
    width: 44px;
  }
}
.itype-back {
  align-self: flex-start;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 44px;
  margin: -8px 0 -10px -6px;
  padding: 0 6px;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  text-decoration: none;
}
.itype-back :deep(svg) {
  width: 18px;
  height: 18px;
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
.itype-card__desc.is-block {
  margin: 0;
  padding-right: 6px;
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
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
.itype-names {
  font-family: var(--app-font-mono);
  font-size: 0.75rem !important;
  word-break: break-all;
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
  background: var(--el-border-color);
}
.itype-app__dot.is-run {
  background: var(--m-ok);
}
.itype-app__dot.is-mnt {
  background: var(--m-warn);
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
.itype-confirm__btn {
  min-height: 36px;
  border-radius: 999px;
  padding: 0 14px;
  justify-content: center;
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
.itype-stack {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  padding-left: 6px;
  overflow: hidden;
}
.itype-stack__l {
  width: 26px;
  height: 26px;
  margin-left: -6px;
  border-radius: 50%;
  border: 2px solid var(--el-bg-color);
  font-size: 0.6875rem;
  flex: none;
}
.itype-stack__l:first-child {
  margin-left: 0;
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
.itype-card__foot :deep(.itype-del) {
  color: var(--m-bad);
}
.itype-card__foot :deep(.itype-del:hover) {
  background: var(--m-badSoft);
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
  max-width: 260px;
  font-size: 0.8125rem;
  line-height: 1.5;
  text-align: center;
}

/* 手机悬浮新建 */
.itype-fab {
  position: sticky;
  bottom: 0;
  display: flex;
  justify-content: flex-end;
  pointer-events: none;
  z-index: 5;
}
.itype-fab__btn {
  pointer-events: auto;
  min-height: 56px;
  padding: 0 22px;
  border-radius: var(--app-radius-pill);
  font-weight: 700;
  box-shadow: var(--app-shadow-brand);
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
  .itype-page {
    padding-bottom: 72px;
  }
  .itype-tools__spacer {
    display: none;
  }
  .itype-search {
    order: -1;
    flex: 1 1 100%;
    min-width: 0;
    min-height: 44px;
  }
  .itype-chips {
    width: 100%;
  }
  .itype-chip {
    min-height: 44px;
  }
  .itype-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: 12px;
  }
  .itype-card {
    gap: 12px;
    padding: 16px 10px 6px 16px;
  }
  .itype-card__head {
    align-items: center;
    gap: 12px;
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
  .itype-card__name-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
  .itype-card__count-n {
    font-size: 1.5rem;
  }
  .itype-add {
    min-height: 160px;
  }
  .itype-hide-sm {
    display: none;
  }
  .itype-confirm__acts {
    flex-wrap: nowrap;
  }
  .itype-confirm__btn {
    flex: 1;
    min-height: 44px;
  }
  .itype-card__foot :deep(.app-icon-btn),
  .itype-card__foot :deep(.action-menu__trigger) {
    width: 44px !important;
    height: 44px !important;
  }
}
</style>
