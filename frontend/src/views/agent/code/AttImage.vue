<!-- 图片附件的缩略图，点开用媒体查看层看大图。src 直接给（输入框里还没发的，本地的对象地址），
     或者给会话和 file（消息附件、工具调用产出的图片，在会话的附件目录里），这时拿签名下载链接让浏览器直接加载。 -->
<template>
  <button
    type="button"
    class="agc-thumb"
    :class="{ 'is-empty': !url }"
    :title="name"
    :aria-label="t('agent.code.composer.viewImage', { name })"
    :disabled="!url"
    @click="open = true"
  >
    <img v-if="url && !failed" :src="url" :alt="name" draggable="false" @error="failed = true" />
    <UIcon v-else :name="failed ? 'i-lucide-image-off' : 'i-lucide-image'" />
  </button>
  <FileMediaDialog v-if="open && url" :model-value="true" :name="name" kind="image" :url="url" @update:model-value="open = false" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import FileMediaDialog from '@/components/file/FileMediaDialog.vue'
import { fileLink } from './fileLinks'

const props = defineProps<{
  name: string
  src?: string
  sessionId?: string
  file?: string
}>()
const { t } = useI18n()

const loaded = ref('')
const failed = ref(false)
const open = ref(false)
const url = computed(() => props.src || loaded.value)

watch(
  () => [props.src, props.sessionId, props.file] as const,
  ([src, sessionId, file]) => {
    loaded.value = ''
    failed.value = false
    if (src || !sessionId || !file) return
    fileLink({ sessionId }, file, true).then(
      (u) => {
        if (props.sessionId !== sessionId || props.file !== file) return
        loaded.value = u
        failed.value = !u
      },
      () => {
        failed.value = true
      },
    )
  },
  { immediate: true },
)
</script>

