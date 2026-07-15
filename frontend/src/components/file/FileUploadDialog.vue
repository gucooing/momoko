<!-- 文件上传弹窗（分片并发/哈希/重试）：令牌驱动 FormDialog + AppSelect 线程数 + 令牌进度条。
     上传逻辑（useFileUpload：addFiles/startAll/retryFailed/cancel/removeItem/threads）全部保留，只重画观感。 -->
<template>
  <FormDialog v-model="visible" :title="t('fileManager.uploadTitle')" :width="600">
    <div class="fu">
      <div class="fu__target">
        <span class="fu__target-label">{{ t('fileManager.uploadTargetDirectory') }}</span>
        <span class="fu__target-path" :title="targetPath">{{ targetPath || '—' }}</span>
      </div>

      <div
        class="fu__drop"
        :class="{ 'is-over': dragOver }"
        @click="pick"
        @dragover.prevent="dragOver = true"
        @dragleave.prevent="dragOver = false"
        @drop.prevent="onDrop"
      >
        <component :is="menuStore.iconComponents['HOutline:ArrowUpTrayIcon']" class="fu__drop-ico" />
        <p class="fu__drop-title">{{ t('fileManager.uploadDropzoneTitle') }}</p>
        <p class="fu__drop-desc">
          {{ t('fileManager.uploadDropzoneDescription', { count: upload.threads.value }) }}
        </p>
        <input ref="inputRef" type="file" multiple class="fu__input" @change="onPick" />
      </div>

      <div class="fu__controls">
        <span class="fu__threads-label">{{ t('fileManager.threads') }}</span>
        <AppSelect
          :model-value="upload.threads.value"
          :options="threadOptions"
          fit
          @update:model-value="(v) => (upload.threads.value = v)"
        />
      </div>

      <div v-if="upload.items.value.length" class="fu__list">
        <div v-for="item in upload.items.value" :key="item.id" class="fu__item">
          <div class="fu__item-main">
            <span class="fu__item-name" :title="item.name">{{ item.name }}</span>
            <span class="fu__item-size">{{ formatFileSize(item.size) }}</span>
          </div>
          <div class="fu__bar">
            <div class="fu__bar-fill" :class="`is-${item.status}`" :style="{ width: `${item.progress}%` }"></div>
          </div>
          <div class="fu__item-meta">
            <span class="fu__item-status" :class="`is-${item.status}`">
              {{ item.error || item.statusText }}
            </span>
            <button
              v-if="item.status === 'uploading' || item.status === 'hashing'"
              type="button"
              class="fu__item-action"
              @click="upload.cancel(item)"
            >
              {{ t('system.common.cancel') }}
            </button>
            <button
              v-else-if="item.status !== 'finishing' && item.status !== 'verifying'"
              type="button"
              class="fu__item-action"
              @click="upload.removeItem(item.id)"
            >
              {{ t('fileManager.remove') }}
            </button>
          </div>
        </div>
      </div>
      <p v-else class="fu__empty">{{ t('fileManager.uploadEmpty') }}</p>
    </div>

    <template #footer="{ close }">
      <span class="fu__summary">
        {{ t('fileManager.uploadSuccessCount', { success: upload.successCount.value, total: upload.items.value.length }) }}
        <template v-if="upload.failedCount.value">
          {{ t('fileManager.uploadFailedCount', { failed: upload.failedCount.value }) }}
        </template>
      </span>
      <UButton color="neutral" variant="soft" @click="close">
        {{ t('system.common.cancel') }}
      </UButton>
      <UButton
        v-if="upload.hasFailed.value"
        color="neutral"
        variant="soft"
        :disabled="upload.uploading.value"
        @click="onRetry"
      >
        {{ t('fileManager.retryFailedItems') }}
      </UButton>
      <UButton
        color="primary"
        :loading="upload.uploading.value"
        :disabled="!upload.hasPending.value || upload.uploading.value"
        @click="onStart"
      >
        {{ upload.uploading.value ? t('fileManager.uploading') : t('fileManager.startUpload') }}
      </UButton>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { formatFileSize } from '@/utils/file'
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
const menuStore = useMenuStore()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const threadOptions = [1, 2, 3, 4, 6].map((n) => ({ label: String(n), value: n }))

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
  gap: 14px;
}
.fu__target {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
}
.fu__target-label {
  color: var(--el-text-color-secondary);
}
.fu__target-path {
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fu__drop {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 28px 16px;
  border: 1.5px dashed var(--el-border-color);
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.fu__drop:hover,
.fu__drop.is-over {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.fu__drop-ico {
  width: 30px;
  height: 30px;
  color: var(--el-color-primary);
}
.fu__drop-title {
  margin: 0;
  font-size: 0.85rem;
  color: var(--el-text-color-primary);
}
.fu__drop-desc {
  margin: 0;
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.fu__input {
  display: none;
}

.fu__controls {
  display: flex;
  align-items: center;
  gap: 8px;
}
.fu__threads-label {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}

.fu__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 280px;
  overflow: auto;
}
.fu__item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-sm);
  background: var(--el-bg-color);
}
.fu__item-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 0.8125rem;
}
.fu__item-name {
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fu__item-size {
  flex-shrink: 0;
  color: var(--el-text-color-placeholder);
  font-size: 0.75rem;
  font-variant-numeric: tabular-nums;
}
.fu__bar {
  height: 6px;
  border-radius: 999px;
  background: var(--el-fill-color);
  overflow: hidden;
}
.fu__bar-fill {
  height: 100%;
  background: var(--el-color-primary);
  transition: width 0.2s;
}
.fu__bar-fill.is-done {
  background: var(--el-color-success);
}
.fu__bar-fill.is-failed {
  background: var(--el-color-danger);
}
.fu__bar-fill.is-canceled {
  background: var(--el-text-color-placeholder);
}
.fu__item-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 0.75rem;
}
.fu__item-status {
  color: var(--el-text-color-placeholder);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fu__item-status.is-done {
  color: var(--el-color-success);
}
.fu__item-status.is-failed {
  color: var(--el-color-danger);
}
.fu__item-action {
  flex-shrink: 0;
  border: none;
  background: transparent;
  color: var(--el-color-primary);
  font-size: 0.75rem;
  cursor: pointer;
}
.fu__empty {
  padding: 16px;
  text-align: center;
  font-size: 0.8125rem;
  color: var(--el-text-color-placeholder);
}

.fu__summary {
  flex: 1;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
</style>
