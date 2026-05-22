<script setup lang="ts">
defineOptions({ name: 'FileMediaDialogContent' })

export type FileMediaKind = 'image' | 'audio' | 'video'

const props = defineProps<{
  kind: FileMediaKind
  objectUrl: string
}>()

const isImage = computed(() => props.kind === 'image')
const isAudio = computed(() => props.kind === 'audio')
</script>

<template>
  <section class="file-media-dialog-content" :class="[`is-${props.kind}`]">
    <el-image v-if="isImage" class="file-media-image" :src="props.objectUrl" fit="contain">
      <template #error>
        <div class="file-media-empty">图片加载失败</div>
      </template>
    </el-image>

    <audio
      v-else-if="isAudio"
      class="file-media-player file-media-audio"
      :src="props.objectUrl"
      controls
      preload="metadata"
    />

    <video
      v-else
      class="file-media-player file-media-video"
      :src="props.objectUrl"
      controls
      playsinline
      preload="metadata"
    />
  </section>
</template>

<style scoped lang="scss">
.file-media-dialog-content {
  display: flex;
  min-height: min(78vh, 48rem);
  align-items: center;
  justify-content: center;
  background: transparent;
}

.file-media-image {
  width: 100%;
  min-height: min(78vh, 48rem);
  overflow: hidden;
  background: transparent;
}

.file-media-image :deep(.el-image__inner) {
  width: 100%;
  max-height: min(78vh, 48rem);
  object-fit: contain;
}

.file-media-empty {
  display: flex;
  width: 100%;
  min-height: min(78vh, 48rem);
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  font-size: 0.9rem;
}

.file-media-player {
  display: block;
  width: 100%;
}

.file-media-video {
  max-height: min(78vh, 48rem);
  background: #000;
}

.file-media-audio {
  width: min(100%, 42rem);
}

:global(.file-media-dialog.el-dialog .el-dialog__body) {
  padding: 0 !important;
}
</style>
