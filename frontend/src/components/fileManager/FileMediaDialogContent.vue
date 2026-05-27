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
  <section class="file-media-content" :class="[`is-${props.kind}`]">
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
.file-media-content {
  display: flex;
  align-items: center;
  justify-content: center;
}

.file-media-content.is-audio {
  width: 100%;
}

.file-media-image {
  max-width: 92vw;
  max-height: 92vh;
}

.file-media-image :deep(.el-image__inner) {
  max-width: 92vw;
  max-height: 92vh;
  object-fit: contain;
}

.file-media-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20rem;
  height: 16rem;
  color: rgb(255 255 255 / 55%);
  font-size: 0.9rem;
}

.file-media-player {
  display: block;
}

.file-media-video {
  max-width: 92vw;
  max-height: 92vh;
  background: #000;
}

.file-media-audio {
  width: min(100%, 42rem);
}
</style>
