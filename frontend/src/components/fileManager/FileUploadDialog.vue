<script setup lang="ts">
import { Close, DocumentAdd, FolderOpened, UploadFilled } from '@element-plus/icons-vue'
import type {
  FileManagerUploadPreSignPayload,
  FileManagerUploadSession,
} from '@/components/fileManager/index.vue'
import {
  type UploadTask,
  isUploadCanceledError,
  useFileUpload,
} from '@/components/fileManager/useFileUpload'
import { getRequestErrorMessage, showRequestError } from '@/utils/request'

defineOptions({ name: 'FileUploadDialog' })

interface UploadQueueItem {
  id: string
  file: File
  progress: number
  status:
    | 'pending'
    | 'hashing'
    | 'uploading'
    | 'verifying'
    | 'finishing'
    | 'canceling'
    | 'success'
    | 'error'
  message: string
  uploadId: string
  canceling: boolean
  task: UploadTask | null
}

const props = withDefaults(
  defineProps<{
    path: string
    onRefresh?: (path: string) => Promise<void> | void
    onGetUploadPreSign?: (
      payload: FileManagerUploadPreSignPayload,
    ) => Promise<FileManagerUploadSession | void> | void
  }>(),
  {
    onRefresh: undefined,
    onGetUploadPreSign: undefined,
  },
)

const visible = defineModel<boolean>({ default: false })

const DEFAULT_UPLOAD_CONCURRENCY = 8

const fileInputRef = useTemplateRef<HTMLInputElement>('fileInput')
const isDragOver = ref(false)
const submitting = ref(false)
const queueItems = ref<UploadQueueItem[]>([])
const uploadConcurrency = ref(DEFAULT_UPLOAD_CONCURRENCY)

const { createUploadTask, cancelUploadSession } = useFileUpload()

const normalizeUploadConcurrency = (value: unknown) => {
  const nextValue = Math.trunc(Number(value))
  return nextValue >= 1 ? nextValue : 1
}

const selectableCount = computed(
  () =>
    queueItems.value.filter((item) => item.status === 'pending' || item.status === 'error').length,
)
const successCount = computed(
  () => queueItems.value.filter((item) => item.status === 'success').length,
)
const errorCount = computed(() => queueItems.value.filter((item) => item.status === 'error').length)
const hasItems = computed(() => queueItems.value.length > 0)
const canSubmit = computed(() => !submitting.value && selectableCount.value > 0)
const effectiveUploadConcurrency = computed(() =>
  normalizeUploadConcurrency(uploadConcurrency.value),
)
const submitText = computed(() => {
  if (submitting.value) return '上传中'
  if (errorCount.value > 0 && selectableCount.value === errorCount.value) return '重试失败项'
  return '开始上传'
})

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 B'

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = Number(bytes) || 0
  let unitIndex = 0

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }

  return `${value >= 100 ? value.toFixed(0) : value.toFixed(2).replace(/\.00$/, '')} ${units[unitIndex]}`
}

const buildQueueItemId = (file: File) => `${file.name}-${file.size}-${file.lastModified}`

const hasRemoteUploadSession = (item: UploadQueueItem) => Boolean(item.uploadId)

const updateQueueItem = (id: string, patch: Partial<UploadQueueItem>) => {
  const targetItem = queueItems.value.find((item) => item.id === id)
  if (!targetItem) return
  Object.assign(targetItem, patch)
}

const removeQueueItemById = (id: string) => {
  queueItems.value = queueItems.value.filter((item) => item.id !== id)
}

const resetQueue = () => {
  queueItems.value = []
  submitting.value = false
  isDragOver.value = false
  if (fileInputRef.value) {
    fileInputRef.value.value = ''
  }
}

const closeDialog = () => {
  if (submitting.value) return
  visible.value = false
}

const appendFiles = (fileList: File[]) => {
  if (!fileList.length) return

  const existingIds = new Set(queueItems.value.map((item) => item.id))
  const nextItems = fileList
    .filter((file) => file.size >= 0)
    .map((file) => ({
      id: buildQueueItemId(file),
      file,
      progress: 0,
      status: 'pending' as const,
      message: '等待上传',
      uploadId: '',
      canceling: false,
      task: null,
    }))
    .filter((item) => !existingIds.has(item.id))

  if (!nextItems.length) {
    ElMessage.info('所选文件已在上传队列中')
    return
  }

  queueItems.value = [...queueItems.value, ...nextItems]
}

const handleFileInputChange = (event: Event) => {
  const target = event.target as HTMLInputElement | null
  appendFiles(Array.from(target?.files || []))

  if (target) {
    target.value = ''
  }
}

const triggerFileSelect = () => {
  if (submitting.value) return
  fileInputRef.value?.click()
}

const handleDrop = (event: DragEvent) => {
  event.preventDefault()
  if (submitting.value) return

  isDragOver.value = false
  appendFiles(Array.from(event.dataTransfer?.files || []))
}

const handleDragOver = (event: DragEvent) => {
  event.preventDefault()
  if (submitting.value) return
  isDragOver.value = true
}

const handleDragLeave = () => {
  isDragOver.value = false
}

const resolveStatusText = (item: UploadQueueItem) => {
  if (item.status === 'hashing') return '计算快速哈希中'
  if (item.status === 'uploading') return item.message || '分片并发上传中'
  if (item.status === 'verifying') return '校验分片状态中'
  if (item.status === 'finishing') return '正在合并分片'
  if (item.status === 'canceling') return '正在取消上传'
  if (item.status === 'success') return item.message || '上传完成'
  if (item.status === 'error') return item.message || '上传失败'
  return item.message || '等待上传'
}

const canShowHeaderAction = (item: UploadQueueItem) => {
  if (item.canceling) return false
  if (!submitting.value) return true
  return hasRemoteUploadSession(item) && item.status !== 'success'
}

const canShowInlineCancel = (item: UploadQueueItem) =>
  hasRemoteUploadSession(item) && item.status !== 'success' && !item.canceling

const cancelQueueItem = async (item: UploadQueueItem) => {
  if (item.canceling) {
    return false
  }

  updateQueueItem(item.id, {
    canceling: true,
    status: 'canceling',
    message: '正在取消上传',
  })

  try {
    if (item.task) {
      await item.task.cancel()
    } else if (item.uploadId) {
      await cancelUploadSession(item.uploadId)
    }

    return true
  } catch (error) {
    if (item.task) {
      return false
    }

    updateQueueItem(item.id, {
      canceling: false,
      task: null,
      status: 'error',
      message: getRequestErrorMessage(error, '取消上传失败'),
    })
    showRequestError(error, `文件 ${item.file.name} 取消上传失败`)
    return false
  }
}

const handleRemoveOrCancel = async (id: string) => {
  const item = queueItems.value.find((queueItem) => queueItem.id === id)
  if (!item) {
    return
  }

  if (hasRemoteUploadSession(item) && item.status !== 'success') {
    const canceled = await cancelQueueItem(item)
    if (!canceled) {
      return
    }
  } else if (submitting.value) {
    return
  }

  removeQueueItemById(id)
}

const refreshDirectory = async () => {
  if (!props.onRefresh) return
  await props.onRefresh(props.path)
}

const submitUpload = async () => {
  if (submitting.value) return

  if (!props.onGetUploadPreSign) {
    ElMessage.info('当前文件视图暂未实现上传')
    return
  }

  const targetItems = queueItems.value.filter(
    (item) => item.status === 'pending' || item.status === 'error',
  )

  if (!targetItems.length) {
    ElMessage.warning('请先选择要上传的文件')
    return
  }

  submitting.value = true
  const concurrency = effectiveUploadConcurrency.value

  let uploadedCount = 0
  let failedCount = 0

  for (const item of targetItems) {
    updateQueueItem(item.id, {
      progress: 0,
      status: 'hashing',
      message: '计算快速哈希中',
      uploadId: '',
      canceling: false,
      task: null,
    })

    const uploadTask = createUploadTask({
      path: props.path,
      file: item.file,
      getUploadPreSign: props.onGetUploadPreSign,
      concurrency,
      onPhaseChange: (phase) => {
        if (phase === 'hashing') {
          updateQueueItem(item.id, {
            status: 'hashing',
            message: '计算快速哈希中',
          })
          return
        }

        if (phase === 'uploading') {
          updateQueueItem(item.id, {
            status: 'uploading',
            message: `分片并发上传中（${concurrency} 线程）`,
          })
          return
        }

        if (phase === 'verifying') {
          updateQueueItem(item.id, {
            status: 'verifying',
            message: '校验分片状态中',
          })
          return
        }

        updateQueueItem(item.id, {
          status: 'finishing',
          message: '正在合并分片',
        })
      },
      onSessionCreated: (session) => {
        updateQueueItem(item.id, {
          uploadId: session.uploadId,
        })
      },
      onProgress: (progress) => {
        updateQueueItem(item.id, {
          progress,
        })
      },
    })

    updateQueueItem(item.id, {
      task: uploadTask,
    })

    try {
      const result = await uploadTask.promise
      uploadedCount += 1
      updateQueueItem(item.id, {
        progress: 100,
        status: 'success',
        message: result.completedBeforeUpload ? '文件已存在，已直接完成' : '上传完成',
        canceling: false,
        task: null,
      })
    } catch (error) {
      if (isUploadCanceledError(error)) {
        updateQueueItem(item.id, {
          canceling: false,
          task: null,
        })
        continue
      }

      failedCount += 1
      updateQueueItem(item.id, {
        progress: 0,
        canceling: false,
        task: null,
        status: 'error',
        message: getRequestErrorMessage(error, '上传失败'),
      })
      showRequestError(error, `文件 ${item.file.name} 上传失败`)
    }
  }

  try {
    if (uploadedCount > 0) {
      await refreshDirectory()
    }
  } catch (error) {
    showRequestError(error, '目录刷新失败')
  } finally {
    submitting.value = false
  }

  if (uploadedCount > 0 && failedCount === 0) {
    ElMessage.success(uploadedCount === 1 ? '文件上传成功' : `已上传 ${uploadedCount} 个文件`)
    visible.value = false
    return
  }

  if (uploadedCount > 0 && failedCount > 0) {
    ElMessage.warning(`已上传 ${uploadedCount} 个文件，另有 ${failedCount} 个失败`)
  }
}

watch(visible, (nextVisible) => {
  if (!nextVisible) {
    resetQueue()
    uploadConcurrency.value = DEFAULT_UPLOAD_CONCURRENCY
  }
})

watch(uploadConcurrency, (value) => {
  const nextValue = normalizeUploadConcurrency(value)

  if (value !== nextValue) {
    uploadConcurrency.value = nextValue
  }
})
</script>

<template>
  <el-dialog
    v-model="visible"
    title="上传文件"
    width="min(720px, calc(100vw - 24px))"
    destroy-on-close
    :close-on-click-modal="!submitting"
    :close-on-press-escape="!submitting"
    :show-close="!submitting"
  >
    <div class="upload-dialog">
      <div class="upload-summary">
        <div class="upload-summary__item">
          <span class="upload-summary__label">目标目录</span>
          <span class="upload-summary__value">{{ props.path }}</span>
        </div>
        <div class="upload-summary__item">
          <span class="upload-summary__label">队列状态</span>
          <span class="upload-summary__value">
            {{ successCount }}/{{ queueItems.length }} 成功
            <template v-if="errorCount > 0">，{{ errorCount }} 失败</template>
          </span>
        </div>
      </div>

      <input
        ref="fileInput"
        type="file"
        multiple
        class="upload-input"
        @change="handleFileInputChange"
      />

      <button
        type="button"
        :class="['upload-dropzone', { 'is-dragover': isDragOver, 'is-disabled': submitting }]"
        @click="triggerFileSelect"
        @dragover="handleDragOver"
        @dragleave="handleDragLeave"
        @drop="handleDrop"
      >
        <span class="upload-dropzone__icon">
          <el-icon size="22"><component :is="UploadFilled" /></el-icon>
        </span>
        <span class="upload-dropzone__title">拖拽文件到此处，或点击选择文件</span>
        <span class="upload-dropzone__description">
          公共上传流程已接入分片并发上传，当前 {{ effectiveUploadConcurrency }} 线程
        </span>
      </button>

      <div v-if="hasItems" class="upload-list">
        <div v-for="item in queueItems" :key="item.id" class="upload-item">
          <div class="upload-item__header">
            <div class="upload-item__meta">
              <span class="upload-item__name">{{ item.file.name }}</span>
              <span class="upload-item__size">{{ formatBytes(item.file.size) }}</span>
            </div>
            <button
              v-if="canShowHeaderAction(item)"
              type="button"
              class="upload-item__remove"
              @click="handleRemoveOrCancel(item.id)"
            >
              移除
            </button>
          </div>

          <div class="upload-item__status">
            <span class="upload-item__status-text">{{ resolveStatusText(item) }}</span>
            <span class="upload-item__progress">{{ item.progress }}%</span>
          </div>

          <div class="upload-item__progress-row">
            <el-progress
              class="upload-item__progress-bar"
              :percentage="item.progress"
              :stroke-width="7"
              :show-text="false"
              :status="
                item.status === 'error'
                  ? 'exception'
                  : item.status === 'success'
                    ? 'success'
                    : undefined
              "
            />
            <button
              v-if="canShowInlineCancel(item)"
              type="button"
              class="upload-item__inline-cancel"
              @click="handleRemoveOrCancel(item.id)"
            >
              <el-icon size="12"><component :is="Close" /></el-icon>
            </button>
          </div>
        </div>
      </div>

      <el-empty v-else class="upload-empty" :image-size="84" description="当前还没有待上传的文件">
        <template #image>
          <el-icon size="56" class="upload-empty__icon">
            <component :is="FolderOpened" />
          </el-icon>
        </template>
      </el-empty>
    </div>

    <template #footer>
      <div class="upload-footer">
        <div class="upload-footer__start">
          <el-button :icon="DocumentAdd" :disabled="submitting" @click="triggerFileSelect">
            选择文件
          </el-button>
          <div class="upload-footer__concurrency">
            <span class="upload-footer__concurrency-label">线程数</span>
            <el-input-number
              v-model="uploadConcurrency"
              :min="1"
              :step="1"
              :disabled="submitting"
              :controls="true"
              controls-position="right"
            />
          </div>
        </div>
        <div class="upload-footer__actions">
          <el-button :disabled="submitting" @click="closeDialog">
            {{ submitting ? '上传中' : '关闭' }}
          </el-button>
          <el-button
            type="primary"
            :loading="submitting"
            :disabled="!canSubmit"
            @click="submitUpload"
          >
            {{ submitText }}
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.upload-dialog {
  display: grid;
  gap: 0.9rem;
}

.upload-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

.upload-summary__item {
  display: grid;
  gap: 0.2rem;
  border-radius: 0.9rem;
  background: color-mix(in srgb, var(--el-color-primary) 5%, var(--el-bg-color-page));
  padding: 0.8rem 0.9rem;
}

.upload-summary__label {
  color: var(--el-text-color-secondary);
  font-size: 0.76rem;
}

.upload-summary__value {
  color: var(--el-text-color-primary);
  font-size: 0.84rem;
  font-weight: 600;
  line-height: 1.5;
  word-break: break-all;
}

.upload-input {
  display: none;
}

.upload-dropzone {
  display: grid;
  justify-items: center;
  gap: 0.45rem;
  border: 1px dashed color-mix(in srgb, var(--el-color-primary) 24%, var(--el-border-color));
  border-radius: 1rem;
  background: color-mix(in srgb, var(--el-color-primary) 4%, var(--el-bg-color));
  padding: 1.35rem 1rem;
  color: inherit;
  cursor: pointer;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease,
    transform 0.2s ease;
}

.upload-dropzone:hover:not(.is-disabled),
.upload-dropzone.is-dragover {
  border-color: color-mix(in srgb, var(--el-color-primary) 60%, var(--el-border-color));
  background: color-mix(in srgb, var(--el-color-primary) 8%, var(--el-bg-color));
  transform: translateY(-1px);
}

.upload-dropzone.is-disabled {
  cursor: not-allowed;
  opacity: 0.72;
}

.upload-dropzone__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 3rem;
  height: 3rem;
  border-radius: 999px;
  background: color-mix(in srgb, var(--el-color-primary) 14%, var(--el-bg-color));
  color: color-mix(in srgb, var(--el-color-primary) 82%, var(--el-text-color-primary));
}

.upload-dropzone__title {
  color: var(--el-text-color-primary);
  font-size: 0.92rem;
  font-weight: 600;
}

.upload-dropzone__description {
  color: var(--el-text-color-secondary);
  font-size: 0.78rem;
}

.upload-list {
  display: grid;
  gap: 0.75rem;
  max-height: 22rem;
  overflow: auto;
  padding-right: 0.1rem;
}

.upload-item {
  display: grid;
  gap: 0.55rem;
  border-radius: 0.95rem;
  background: color-mix(in srgb, var(--el-bg-color-page) 88%, var(--el-bg-color));
  padding: 0.85rem 0.9rem;
}

.upload-item__header,
.upload-item__status,
.upload-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.upload-item__meta {
  display: grid;
  min-width: 0;
  gap: 0.18rem;
}

.upload-item__name {
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 0.84rem;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.upload-item__size,
.upload-item__status-text,
.upload-item__progress {
  color: var(--el-text-color-secondary);
  font-size: 0.76rem;
}

.upload-item__status-text {
  line-height: 1.5;
}

.upload-item__progress-row {
  display: flex;
  align-items: center;
  gap: 0.65rem;
}

.upload-item__progress-bar {
  flex: 1;
}

.upload-item__remove {
  border: none;
  background: transparent;
  padding: 0;
  color: var(--el-color-danger);
  font-size: 0.76rem;
  cursor: pointer;
}

.upload-item__inline-cancel {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.4rem;
  height: 1.4rem;
  border: 1px solid color-mix(in srgb, var(--el-color-danger) 55%, transparent);
  border-radius: 999px;
  background: transparent;
  padding: 0;
  color: var(--el-color-danger);
  cursor: pointer;
}

.upload-empty {
  border-radius: 1rem;
  background: color-mix(in srgb, var(--el-bg-color-page) 90%, var(--el-bg-color));
  padding-block: 1rem;
}

.upload-empty__icon {
  color: color-mix(in srgb, var(--el-color-primary) 70%, var(--el-text-color-secondary));
}

.upload-footer__start,
.upload-footer__actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.upload-footer__start {
  min-width: 0;
  flex-wrap: wrap;
}

.upload-footer__concurrency {
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  color: var(--el-text-color-secondary);
  font-size: 0.84rem;
}

.upload-footer__concurrency-label {
  white-space: nowrap;
}

.upload-footer__concurrency :deep(.el-input-number) {
  width: 104px;
}

@media (width <= 768px) {
  .upload-summary {
    grid-template-columns: 1fr;
  }

  .upload-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .upload-footer__start,
  .upload-footer__actions {
    width: 100%;
  }

  .upload-footer__start {
    justify-content: space-between;
  }

  .upload-footer__actions {
    justify-content: flex-end;
  }

  .upload-footer__concurrency {
    justify-content: space-between;
    width: 100%;
  }

  .upload-footer__concurrency :deep(.el-input-number) {
    width: 116px;
  }
}
</style>
