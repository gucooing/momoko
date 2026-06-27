<template>
  <FileDialog v-model="visible" :title="t('fileManager.uploadTitle')" :width="600">
    <div class="fu">
      <div class="fu-target">
        <span class="fu-target-label">{{ t('fileManager.uploadTargetDirectory') }}</span>
        <span class="fu-target-path" :title="targetPath">{{ targetPath || '—' }}</span>
      </div>

      <div
        class="fu-dropzone"
        :class="{ 'is-over': dragOver }"
        @click="pick"
        @dragover.prevent="dragOver = true"
        @dragleave.prevent="dragOver = false"
        @drop.prevent="onDrop"
      >
        <el-icon class="fu-dropzone-icon"><IconUpload /></el-icon>
        <p class="fu-dropzone-title">{{ t('fileManager.uploadDropzoneTitle') }}</p>
        <p class="fu-dropzone-desc">
          {{ t('fileManager.uploadDropzoneDescription', { count: upload.threads.value }) }}
        </p>
        <input ref="inputRef" type="file" multiple class="fu-input" @change="onPick" />
      </div>

      <div class="fu-controls">
        <label class="fu-threads">
          {{ t('fileManager.threads') }}
          <select v-model.number="upload.threads.value" class="fu-threads-select">
            <option v-for="n in [1, 2, 3, 4, 6]" :key="n" :value="n">{{ n }}</option>
          </select>
        </label>
      </div>

      <div v-if="upload.items.value.length" class="fu-list">
        <div v-for="item in upload.items.value" :key="item.id" class="fu-item">
          <div class="fu-item-main">
            <span class="fu-item-name" :title="item.name">{{ item.name }}</span>
            <span class="fu-item-size">{{ formatFileSize(item.size) }}</span>
          </div>
          <div class="fu-item-bar">
            <div
              class="fu-item-bar-fill"
              :class="`is-${item.status}`"
              :style="{ width: `${item.progress}%` }"
            ></div>
          </div>
          <div class="fu-item-meta">
            <span class="fu-item-status" :class="`is-${item.status}`">
              {{ item.error || item.statusText }}
            </span>
            <button
              v-if="item.status === 'uploading' || item.status === 'hashing'"
              type="button"
              class="fu-item-action"
              @click="upload.cancel(item)"
            >
              {{ t('system.common.cancel') }}
            </button>
            <button
              v-else-if="item.status !== 'finishing' && item.status !== 'verifying'"
              type="button"
              class="fu-item-action"
              @click="upload.removeItem(item.id)"
            >
              {{ t('fileManager.remove') }}
            </button>
          </div>
        </div>
      </div>
      <p v-else class="fu-empty">{{ t('fileManager.uploadEmpty') }}</p>
    </div>

    <template #footer>
      <span class="fu-summary">
        {{ t('fileManager.uploadSuccessCount', { success: upload.successCount.value, total: upload.items.value.length }) }}
        <template v-if="upload.failedCount.value">
          {{ t('fileManager.uploadFailedCount', { failed: upload.failedCount.value }) }}
        </template>
      </span>
      <div class="fu-footer-actions">
        <button type="button" class="fm-btn" @click="visible = false">
          {{ t('system.common.cancel') }}
        </button>
        <button
          v-if="upload.hasFailed.value"
          type="button"
          class="fm-btn"
          :disabled="upload.uploading.value"
          @click="onRetry"
        >
          {{ t('fileManager.retryFailedItems') }}
        </button>
        <button
          type="button"
          class="fm-btn fm-btn--primary"
          :disabled="!upload.hasPending.value || upload.uploading.value"
          @click="onStart"
        >
          {{ upload.uploading.value ? t('fileManager.uploading') : t('fileManager.startUpload') }}
        </button>
      </div>
    </template>
  </FileDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { formatFileSize } from '@/utils/file'
import FileDialog from './FileDialog.vue'
import { IconUpload } from './icons'
import { useFileUpload } from './useFileUpload'
import type { FileClient } from './types'

const props = defineProps<{
  modelValue: boolean
  client: FileClient
  targetPath: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  uploaded: []
}>()

const { t } = useI18n()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const upload = useFileUpload(
  () => props.client,
  () => props.targetPath,
)

const inputRef = ref<HTMLInputElement | null>(null)
const dragOver = ref(false)

const pick = () => inputRef.value?.click()
const onPick = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files?.length) upload.addFiles(input.files)
  input.value = ''
}
const onDrop = (event: DragEvent) => {
  dragOver.value = false
  if (event.dataTransfer?.files?.length) upload.addFiles(event.dataTransfer.files)
}

const onStart = async () => {
  const ok = await upload.startAll()
  if (ok) emit('uploaded')
}
const onRetry = async () => {
  const ok = await upload.retryFailed()
  if (ok) emit('uploaded')
}
</script>

<style scoped>
.fu {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
}
.fu-target {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 13px;
}
.fu-target-label {
  color: var(--fm-text-3);
}
.fu-target-path {
  color: var(--fm-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fu-dropzone {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
  padding: 1.75rem 1rem;
  border: 1.5px dashed var(--fm-border-strong);
  border-radius: var(--fm-radius);
  background: var(--fm-subtle);
  cursor: pointer;
  transition:
    border-color 0.15s,
    background 0.15s;
}
.fu-dropzone:hover,
.fu-dropzone.is-over {
  border-color: var(--fm-accent);
  background: var(--fm-accent-soft);
}
.fu-dropzone-icon {
  font-size: 30px;
  color: var(--fm-accent);
}
.fu-dropzone-title {
  margin: 0;
  font-size: 13.5px;
  color: var(--fm-text);
}
.fu-dropzone-desc {
  margin: 0;
  font-size: 12px;
  color: var(--fm-text-3);
}
.fu-input {
  display: none;
}

.fu-controls {
  display: flex;
  align-items: center;
}
.fu-threads {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 13px;
  color: var(--fm-text-2);
}
.fu-threads-select {
  height: 30px;
  padding: 0 0.5rem;
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius-sm);
  background: var(--fm-surface);
  color: var(--fm-text);
}

.fu-list {
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
  max-height: 280px;
  overflow: auto;
}
.fu-item {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  padding: 0.625rem 0.75rem;
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius-sm);
  background: var(--fm-surface);
}
.fu-item-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  font-size: 13px;
}
.fu-item-name {
  color: var(--fm-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fu-item-size {
  flex-shrink: 0;
  color: var(--fm-text-3);
  font-size: 12px;
}
.fu-item-bar {
  height: 6px;
  border-radius: 999px;
  background: var(--fm-subtle);
  overflow: hidden;
}
.fu-item-bar-fill {
  height: 100%;
  background: var(--fm-accent);
  transition: width 0.2s;
}
.fu-item-bar-fill.is-done {
  background: #18a058;
}
.fu-item-bar-fill.is-failed {
  background: var(--fm-danger);
}
.fu-item-bar-fill.is-canceled {
  background: var(--fm-text-3);
}
.fu-item-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  font-size: 12px;
}
.fu-item-status {
  color: var(--fm-text-3);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fu-item-status.is-done {
  color: #18a058;
}
.fu-item-status.is-failed {
  color: var(--fm-danger);
}
.fu-item-action {
  flex-shrink: 0;
  border: none;
  background: transparent;
  color: var(--fm-accent);
  font-size: 12px;
  cursor: pointer;
}
.fu-empty {
  padding: 1rem;
  text-align: center;
  font-size: 13px;
  color: var(--fm-text-3);
}

.fu-summary {
  flex: 1;
  font-size: 12px;
  color: var(--fm-text-3);
}
.fu-footer-actions {
  display: flex;
  gap: 0.5rem;
}
</style>
