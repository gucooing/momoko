<!-- 更换头像：只能上传图片（jpg / png / webp / gif，20MB 以内），确认后由调用方直传到对象存储 -->
<template>
  <BaseDialog v-model="open" :title="t('avatar.title')" :width="menuStore.isMobile ? '94%' : '520px'" @close="close">
    <label class="upload-drop" :class="{ 'has-preview': !!previewUrl }" @dragover.prevent @drop.prevent="onDrop">
      <input class="upload-drop__input" type="file" :accept="AVATAR_TYPES.join(',')" @change="onFileInput" />
      <div v-if="!previewUrl" class="upload-content">
        <component :is="ic('HSolid:PhotoIcon')" class="upload-icon" />
        <div class="upload-text">
          <div class="upload-text-main">{{ t('avatar.dragUpload') }}</div>
          <div class="upload-text-tip">{{ t('avatar.uploadTip') }}</div>
        </div>
      </div>
      <div v-else class="upload-preview-content">
        <AppAvatar :src="previewUrl" :size="previewAvatarSize" />
        <div class="preview-hint">{{ t('avatar.reuploadHint') }}</div>
      </div>
    </label>

    <template #footer>
      <UButton color="neutral" variant="soft" @click="close">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" :disabled="!selectedFile" @click="confirm">
        {{ t('common.confirm') }}
      </UButton>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'SelectAvatarDialog' })

const emits = defineEmits<{ (e: 'pick', file: File): void }>()

const AVATAR_TYPES = ['image/jpeg', 'image/png', 'image/webp', 'image/gif']
const AVATAR_MAX_SIZE = 20 * 1024 * 1024

const menuStore = useMenuStore()
const { t } = useI18n()
const ic = (name: string) => menuStore.iconComponents[name]

const open = ref(false)
const previewAvatarSize = computed(() => (menuStore.isMobile ? 132 : 180))
const selectedFile = ref<File | null>(null)
const previewUrl = ref('')

const setPreview = (file: File | null) => {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  selectedFile.value = file
  previewUrl.value = file ? URL.createObjectURL(file) : ''
}
onBeforeUnmount(() => setPreview(null))

const pickFile = (file: File) => {
  if (!AVATAR_TYPES.includes(file.type)) {
    feedback.warning(t('avatar.imageTypeLimit'))
    return
  }
  if (file.size > AVATAR_MAX_SIZE) {
    feedback.warning(t('avatar.imageSizeLimit'))
    return
  }
  setPreview(file)
}

const onFileInput = (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) pickFile(file)
  input.value = ''
}

const onDrop = (event: DragEvent) => {
  const file = event.dataTransfer?.files?.[0]
  if (file) pickFile(file)
}

const confirm = () => {
  if (!selectedFile.value) return
  emits('pick', selectedFile.value)
  close()
}

const close = () => {
  open.value = false
  setPreview(null)
}

const showDialog = () => {
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.upload-drop {
  width: 100%;
  min-height: 280px;
  border: 2px dashed var(--el-border-color);
  border-radius: 12px;
  background: var(--el-fill-color-lighter);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: border-color 0.15s;
  position: relative;
}
.upload-drop:hover {
  border-color: var(--el-color-primary);
}
.upload-drop__input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}
.upload-content,
.upload-preview-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 1.5rem;
  pointer-events: none;
}
.upload-icon {
  width: 48px;
  height: 48px;
  color: var(--el-color-primary);
}
.upload-text {
  text-align: center;
}
.upload-text-main {
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 0.25rem;
}
.upload-text-tip {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.preview-hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
</style>
