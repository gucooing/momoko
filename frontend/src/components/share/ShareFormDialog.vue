<!-- 分享 新建/编辑弹窗（重写 · P1）：FormDialog 外壳 + 令牌字段 + 原生 datetime-local + AppSwitch。
     创建成功切换为详情视图（链接/提取码复制）。保留 FilePicker(跨来源选文件) 与全部 create/update 逻辑。 -->
<template>
  <FormDialog
    v-model="visible"
    :title="dialogTitle"
    :width="560"
    :loading="saving"
  >
    <!-- 创建成功：展示分享详情，便于复制/转发 -->
    <div v-if="created" class="share-result">
      <div class="share-result__hero">
        <component :is="menuStore.iconComponents['HSolid:CheckCircleIcon']" class="share-result__ico" />
        <div>
          <div class="share-result__title">{{ t('file.share.createdTitle') }}</div>
          <div class="share-result__sub">{{ created.name }}</div>
        </div>
      </div>
      <div class="share-result__list">
        <div class="share-kv">
          <span class="share-kv__label">{{ t('file.share.link') }}</span>
          <div class="share-kv__copy">
            <input class="app-input" :value="link" readonly />
            <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-copy" @click="copy(link)" />
          </div>
        </div>
        <div v-if="created.code" class="share-kv">
          <span class="share-kv__label">{{ t('file.share.code') }}</span>
          <div class="share-kv__copy">
            <input class="app-input" :value="created.code" readonly />
            <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-copy" @click="copy(created.code)" />
          </div>
        </div>
        <div class="share-kv">
          <span class="share-kv__label">{{ t('file.share.expires') }}</span>
          <span class="share-kv__val">{{ created.expiresAt ? formatTime(created.expiresAt) : t('file.share.never') }}</span>
        </div>
        <div class="share-kv">
          <span class="share-kv__label">{{ t('file.share.maxDownloads') }}</span>
          <span class="share-kv__val">{{ Number(created.maxDownloads) > 0 ? created.maxDownloads : t('file.share.unlimited') }}</span>
        </div>
      </div>
    </div>

    <!-- 表单：新建 / 编辑 -->
    <div v-else class="share-form">
      <div class="app-field">
        <div class="share-picked__head">
          <label class="app-label app-label--required">{{ t('file.share.path') }}</label>
          <button type="button" class="share-picked__pick" @click="pickerOpen = true">
            <component :is="menuStore.iconComponents['HOutline:FolderPlusIcon']" />
            {{ form.items.length ? t('file.share.reselect') : t('fileManager.selectFile') }}
          </button>
        </div>

        <!-- 已选条目：定高滚动列表（多文件不撑破弹窗），逐项可移除 -->
        <div v-if="form.items.length" class="share-picked" :class="{ 'is-error': pathError }">
          <div class="share-picked__bar">
            <span>{{ t('file.share.selectedCount', { count: form.items.length }) }}</span>
            <button type="button" class="share-picked__clear" @click="clearItems">{{ t('file.share.clearAll') }}</button>
          </div>
          <ul class="share-picked__list">
            <li v-for="item in form.items" :key="itemKey(item)" class="share-picked__item">
              <component :is="menuStore.iconComponents['HOutline:DocumentIcon']" class="share-picked__ico" />
              <span class="share-picked__name">{{ baseName(item.path) }}</span>
              <span class="share-picked__path" :title="item.path">{{ item.path }}</span>
              <button type="button" class="share-picked__rm" :aria-label="t('system.common.delete')" @click="removeItem(item)">
                <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
              </button>
            </li>
          </ul>
        </div>
        <button v-else type="button" class="share-picked__empty" @click="pickerOpen = true">
          {{ t('file.share.noItems') }}
        </button>

        <span v-if="pathError" class="app-field__error">{{ t('file.share.pathRequired') }}</span>
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('file.share.name') }}</label>
        <input v-model="form.name" class="app-input" :placeholder="t('file.share.namePlaceholder')" />
      </div>

      <div class="share-form__grid">
        <div class="app-field">
          <label class="app-label">{{ t('file.share.code') }}</label>
          <input v-model="form.code" class="app-input" :placeholder="t('file.share.codePlaceholder')" maxlength="16" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('file.share.maxDownloads') }}</label>
          <input v-model.number="form.maxDownloads" class="app-input" type="number" min="0" placeholder="0" />
        </div>
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('file.share.expires') }}</label>
        <input v-model="form.expiresAt" class="app-input" type="datetime-local" />
        <span class="share-form__hint">{{ t('file.share.neverPlaceholder') }}</span>
      </div>

      <div class="app-field share-form__switch">
        <label class="app-label">{{ t('file.share.enabled') }}</label>
        <AppSwitch v-model="form.enabled" />
      </div>
    </div>

    <FilePicker v-model="pickerOpen" multiple :initial-items="form.items" @confirm="onPickItems" />

    <template #footer>
      <template v-if="created">
        <UButton color="primary" @click="visible = false">{{ t('file.share.done') }}</UButton>
      </template>
      <template v-else>
        <UButton color="neutral" variant="soft" @click="visible = false">{{ t('system.common.cancel') }}</UButton>
        <UButton color="primary" :loading="saving" @click="save">{{ t('system.common.confirm') }}</UButton>
      </template>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import FilePicker from '@/components/file/FilePicker.vue'
import { getBaseName } from '@/utils/file'
import type { PickedFile } from '@/components/file/types'
import { buildShareLink, createShareRequest, updateShareRequest } from '@/api/share'
import type { ShareInfo } from '@/types/v1/file'

const props = defineProps<{
  modelValue: boolean
  // 编辑已有分享（优先于 items）
  share?: ShareInfo | null
  // 新建时预置的条目（如从文件管理器选中文件/文件夹，自带各自来源）
  items?: PickedFile[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
}>()

const { t } = useI18n()
const menuStore = useMenuStore()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})

const isEdit = computed(() => !!props.share)
const dialogTitle = computed(() => {
  if (created.value) return t('file.share.createdTitle')
  return isEdit.value ? t('file.share.editTitle') : t('file.share.createTitle')
})

const saving = ref(false)
const pathError = ref(false)
const created = ref<ShareInfo | null>(null)
const link = computed(() => (created.value ? buildShareLink(created.value.token) : ''))

// Date → datetime-local 字符串（YYYY-MM-DDTHH:mm，本地时区）
const toLocalInput = (value: Date | string): string => {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const defaultForm = () => ({
  items: [] as PickedFile[],
  name: '',
  code: '',
  expiresAt: '' as string,
  maxDownloads: 0,
  enabled: true,
})
const form = reactive(defaultForm())
const itemKey = (item: PickedFile) => `${item.sourceId}\n${item.path}`
const baseName = (path: string) => getBaseName(path) || path
const removeItem = (item: PickedFile) => {
  form.items = form.items.filter((i) => itemKey(i) !== itemKey(item))
}
const clearItems = () => {
  form.items = []
}

// 通过文件树选择分享条目（可跨来源），取代手动输入绝对路径。
const pickerOpen = ref(false)
const onPickItems = (items: PickedFile[]) => {
  form.items = items
  if (items.length) pathError.value = false
}

const formatTime = (v: unknown) => (v ? new Date(v as string).toLocaleString() : '')

const copy = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('file.share.copied'))
  } catch {
    ElMessage.error(t('file.share.copyFailed'))
  }
}

// 弹窗打开时重置/回填（modelValue false→true）
watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    created.value = null
    pathError.value = false
    Object.assign(form, defaultForm())
    if (props.share) {
      Object.assign(form, {
        items: props.share.items.map((i) => ({ sourceId: i.sourceId, path: i.path })),
        name: props.share.name,
        code: props.share.code,
        expiresAt: props.share.expiresAt ? toLocalInput(props.share.expiresAt as unknown as string) : '',
        maxDownloads: Number(props.share.maxDownloads) || 0,
        enabled: props.share.enabled,
      })
    } else if (props.items?.length) {
      form.items = props.items.map((i) => ({ sourceId: i.sourceId, path: i.path }))
    }
  },
)

// 请求条目：仅 sourceId + path 有意义，名称/类型/大小由服务端探测缓存（此处占位以满足类型）。
const toRequestItems = () =>
  form.items.map((i) => ({ sourceId: i.sourceId, path: i.path, name: '', isDir: false, size: 0 }))

const save = async () => {
  if (!form.items.length) {
    pathError.value = true
    ElMessage.warning(t('file.share.pathRequired'))
    return
  }
  saving.value = true
  try {
    const expiresAt = form.expiresAt ? new Date(form.expiresAt) : undefined
    if (props.share) {
      await updateShareRequest({
        id: props.share.id,
        name: form.name,
        code: form.code,
        expiresAt,
        maxDownloads: form.maxDownloads,
        enabled: form.enabled,
        items: toRequestItems(),
      })
      ElMessage.success(t('system.common.editSuccess'))
      visible.value = false
      emit('saved')
    } else {
      const { data } = await createShareRequest({
        items: toRequestItems(),
        name: form.name,
        code: form.code,
        expiresAt,
        maxDownloads: form.maxDownloads,
        enabled: form.enabled,
      })
      // 创建成功后切换到详情视图（展示链接/提取码），不直接关闭
      created.value = data?.info ?? null
      emit('saved')
    }
  } finally {
    saving.value = false
  }
}
</script>

<style scoped lang="scss">
.share-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.share-form__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}
.share-form__switch {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.share-form__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.share-picked__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 6px;
}
.share-picked__pick {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 9px;
  border: 1px solid var(--el-border-color);
  border-radius: var(--app-radius-sm);
  background: var(--el-bg-color);
  color: var(--el-color-primary);
  font-size: 0.78rem;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.share-picked__pick:hover {
  border-color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 6%, transparent);
}
.share-picked__pick :deep(svg) {
  width: 14px;
  height: 14px;
}
/* 定高滚动容器：多文件不撑破弹窗 */
.share-picked {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-lighter);
  overflow: hidden;
}
.share-picked.is-error {
  border-color: var(--el-color-danger, #ef4444);
}
.share-picked__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.share-picked__clear {
  border: none;
  background: transparent;
  color: var(--el-color-primary);
  font-size: 0.75rem;
  cursor: pointer;
}
.share-picked__list {
  max-height: 148px;
  overflow-y: auto;
  margin: 0;
  padding: 4px;
  list-style: none;
}
.share-picked__item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 6px;
  border-radius: var(--app-radius-xs);
  min-width: 0;
}
.share-picked__item:hover {
  background: var(--el-fill-color-light);
}
.share-picked__ico {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
  color: var(--el-text-color-placeholder);
}
.share-picked__name {
  flex-shrink: 0;
  max-width: 180px;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.share-picked__path {
  flex: 1;
  min-width: 0;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', Consolas, monospace;
  font-size: 0.7rem;
  color: var(--el-text-color-placeholder);
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  direction: rtl;
}
.share-picked__rm {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-xs);
  color: var(--el-text-color-placeholder);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.share-picked__rm:hover {
  background: color-mix(in srgb, var(--el-color-danger, #ef4444) 12%, transparent);
  color: var(--el-color-danger, #ef4444);
}
.share-picked__rm :deep(svg) {
  width: 13px;
  height: 13px;
}
.share-picked__empty {
  width: 100%;
  padding: 14px;
  border: 1px dashed var(--el-border-color);
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-placeholder);
  font-size: 0.8125rem;
  cursor: pointer;
  transition: border-color 0.15s;
}
.share-picked__empty:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
/* 创建成功详情 */
.share-result {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.share-result__hero {
  display: flex;
  align-items: center;
  gap: 12px;
}
.share-result__ico {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  color: var(--el-color-success, #16a34a);
}
.share-result__title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.share-result__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.share-result__list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.share-kv {
  display: flex;
  align-items: center;
  gap: 12px;
}
.share-kv__label {
  width: 72px;
  flex-shrink: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.share-kv__val {
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
  font-variant-numeric: tabular-nums;
}
.share-kv__copy {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}
@media (width <= 768px) {
  .share-form__grid {
    grid-template-columns: 1fr;
  }
}
</style>
