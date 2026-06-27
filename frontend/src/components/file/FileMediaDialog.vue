<template>
  <FileDialog v-model="visible" :title="name" :width="720">
    <div class="fmd-stage">
      <img v-if="kind === 'image'" :src="url" :alt="name" class="fmd-image" />
      <video v-else-if="kind === 'video'" :src="url" class="fmd-video" controls autoplay />
      <audio v-else-if="kind === 'audio'" :src="url" class="fmd-audio" controls autoplay />
      <p v-else class="fmd-empty">{{ t('fileManager.cannotEditBinary') }}</p>
    </div>

    <template #footer>
      <a class="fm-btn" :href="url" :download="name" target="_blank" rel="noopener">
        <el-icon><IconDownload /></el-icon>{{ t('fileManager.download') }}
      </a>
      <button type="button" class="fm-btn fm-btn--primary" @click="visible = false">
        {{ t('system.common.confirm') }}
      </button>
    </template>
  </FileDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import FileDialog from './FileDialog.vue'
import { IconDownload } from './icons'
import type { FilePreviewKind } from '@/utils/file'

const props = defineProps<{
  modelValue: boolean
  name: string
  kind: FilePreviewKind
  url: string
}>()

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
</script>

<style scoped>
.fmd-stage {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 240px;
  max-height: 64vh;
  background: var(--fm-subtle);
  border-radius: var(--fm-radius-sm);
  overflow: auto;
}
.fmd-image {
  max-width: 100%;
  max-height: 64vh;
  object-fit: contain;
}
.fmd-video {
  max-width: 100%;
  max-height: 64vh;
}
.fmd-audio {
  width: 80%;
}
.fmd-empty {
  color: var(--fm-text-3);
  font-size: 13px;
}
</style>
