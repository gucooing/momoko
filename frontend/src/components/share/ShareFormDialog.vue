<template>
  <BaseDialog v-model="visible" :title="dialogTitle" width="560" @opened="onOpened">
    <!-- 创建成功：展示分享详情，便于复制/转发 -->
    <div v-if="created" class="share-result">
      <el-result icon="success" :title="t('file.share.createdTitle')" :sub-title="created.name" />
      <el-form label-width="112px" label-position="right">
        <el-form-item :label="t('file.share.link')">
          <el-input :model-value="link" readonly>
            <template #append>
              <el-button @click="copy(link)">{{ t('file.share.copy') }}</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item v-if="created.code" :label="t('file.share.code')">
          <el-input :model-value="created.code" readonly>
            <template #append>
              <el-button @click="copy(created.code)">{{ t('file.share.copy') }}</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item :label="t('file.share.expires')">
          <span>{{
            created.expiresAt ? formatTime(created.expiresAt) : t('file.share.never')
          }}</span>
        </el-form-item>
        <el-form-item :label="t('file.share.maxDownloads')">
          <span>{{
            Number(created.maxDownloads) > 0 ? created.maxDownloads : t('file.share.unlimited')
          }}</span>
        </el-form-item>
      </el-form>
    </div>

    <!-- 表单：新建 / 编辑 -->
    <el-form v-else :model="form" label-width="110px" label-position="right">
      <el-form-item :label="t('file.share.path')" required>
        <div class="share-paths">
          <el-tag v-for="item in form.items" :key="itemKey(item)" class="share-path-tag">{{
            item.path
          }}</el-tag>
          <el-button @click="pickerOpen = true">{{ t('fileManager.selectFile') }}</el-button>
        </div>
      </el-form-item>
      <el-form-item :label="t('file.share.name')">
        <el-input v-model="form.name" :placeholder="t('file.share.namePlaceholder')" />
      </el-form-item>
      <el-form-item :label="t('file.share.code')">
        <el-input
          v-model="form.code"
          :placeholder="t('file.share.codePlaceholder')"
          maxlength="16"
        />
      </el-form-item>
      <el-form-item :label="t('file.share.expires')">
        <el-date-picker
          v-model="form.expiresAt"
          type="datetime"
          :placeholder="t('file.share.neverPlaceholder')"
          class="full-input"
        />
      </el-form-item>
      <el-form-item :label="t('file.share.maxDownloads')">
        <el-input-number
          v-model="form.maxDownloads"
          :min="0"
          controls-position="right"
          class="full-input"
        />
      </el-form-item>
      <el-form-item :label="t('file.share.enabled')">
        <el-switch v-model="form.enabled" />
      </el-form-item>
    </el-form>

    <FilePicker v-model="pickerOpen" multiple :initial-items="form.items" @confirm="onPickItems" />

    <template #footer>
      <template v-if="created">
        <el-button type="primary" @click="visible = false">{{ t('file.share.done') }}</el-button>
      </template>
      <template v-else>
        <el-button @click="visible = false">{{ t('system.common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="save">{{
          t('system.common.confirm')
        }}</el-button>
      </template>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import FilePicker from '@/components/file/FilePicker.vue'
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
const created = ref<ShareInfo | null>(null)
const link = computed(() => (created.value ? buildShareLink(created.value.token) : ''))

const defaultForm = () => ({
  items: [] as PickedFile[],
  name: '',
  code: '',
  expiresAt: null as Date | null,
  maxDownloads: 0,
  enabled: true,
})
const form = reactive(defaultForm())
const itemKey = (item: PickedFile) => `${item.sourceId}\n${item.path}`

// 通过文件树选择分享条目（可跨来源），取代手动输入绝对路径。
const pickerOpen = ref(false)
const onPickItems = (items: PickedFile[]) => {
  form.items = items
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

const onOpened = () => {
  created.value = null
  Object.assign(form, defaultForm())
  if (props.share) {
    Object.assign(form, {
      items: props.share.items.map((i) => ({ sourceId: i.sourceId, path: i.path })),
      name: props.share.name,
      code: props.share.code,
      expiresAt: props.share.expiresAt
        ? new Date(props.share.expiresAt as unknown as string)
        : null,
      maxDownloads: Number(props.share.maxDownloads) || 0,
      enabled: props.share.enabled,
    })
  } else if (props.items?.length) {
    form.items = props.items.map((i) => ({ sourceId: i.sourceId, path: i.path }))
  }
}

// 请求条目：仅 sourceId + path 有意义，名称/类型/大小由服务端探测缓存（此处占位以满足类型）。
const toRequestItems = () =>
  form.items.map((i) => ({ sourceId: i.sourceId, path: i.path, name: '', isDir: false, size: 0 }))

const save = async () => {
  if (!form.items.length) {
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
.full-input {
  width: 100%;
}
.share-result :deep(.el-result) {
  padding: 0.4rem 0 0.8rem;
}
.share-result :deep(.el-form-item__label) {
  white-space: nowrap;
}
.share-paths {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  width: 100%;
}
.share-path-tag {
  max-width: 100%;
}
</style>
