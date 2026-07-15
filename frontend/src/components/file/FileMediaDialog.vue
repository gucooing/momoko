<!-- 媒体预览弹窗（图片/视频/音频）：令牌驱动 FormDialog；页脚提供下载与关闭。 -->
<template>
  <FormDialog v-model="visible" :title="name" :width="720">
    <div class="fmd">
      <img v-if="kind === 'image'" :src="url" :alt="name" class="fmd__image" />
      <video v-else-if="kind === 'video'" :src="url" class="fmd__video" controls autoplay />
      <audio v-else-if="kind === 'audio'" :src="url" class="fmd__audio" controls autoplay />
      <p v-else class="fmd__empty">{{ t('fileManager.cannotEditBinary') }}</p>
    </div>

    <template #footer="{ close }">
      <UButton color="neutral" variant="soft" icon="i-lucide-download" @click="download">
        {{ t('fileManager.download') }}
      </UButton>
      <UButton color="primary" @click="close">
        {{ t('system.common.confirm') }}
      </UButton>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { downloadFileFromUrl, type FilePreviewKind } from '@/utils/file'

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

const download = () => downloadFileFromUrl(props.url, props.name)
</script>

<style scoped>
.fmd {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 240px;
  max-height: 64vh;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius-sm);
  overflow: auto;
}
.fmd__image {
  max-width: 100%;
  max-height: 64vh;
  object-fit: contain;
}
.fmd__video {
  max-width: 100%;
  max-height: 64vh;
}
.fmd__audio {
  width: 80%;
}
.fmd__empty {
  color: var(--el-text-color-placeholder);
  font-size: 0.8125rem;
}
</style>
