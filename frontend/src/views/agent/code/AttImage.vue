<!-- 图片附件的缩略图，点开用媒体查看层看大图。src 直接给（输入框里还没发的），
     或者给会话和 file（消息里发过的），这时向后端取回来；取过的记在内存里，翻回来不再取。 -->
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
    <img v-if="url" :src="url" :alt="name" draggable="false" />
    <UIcon v-else :name="failed ? 'i-lucide-image-off' : 'i-lucide-image'" />
  </button>
  <FileMediaDialog v-if="open && url" :model-value="true" :name="name" kind="image" :url="url" @update:model-value="open = false" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import FileMediaDialog from '@/components/file/FileMediaDialog.vue'

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
    loadUpload(sessionId, file).then(
      (u) => {
        if (props.sessionId === sessionId && props.file === file) loaded.value = u
      },
      () => {
        failed.value = true
      },
    )
  },
  { immediate: true },
)
</script>

<script lang="ts">
import { getAgentSessionUpload } from '@/api/agent'

// 取回来的图片按「会话 + file」记着，最多记这么多张（最久没用的先丢）。
const CACHE_MAX = 60
const cache = new Map<string, Promise<string>>()

const loadUpload = (sessionId: string, file: string) => {
  const key = `${sessionId}\n${file}`
  const hit = cache.get(key)
  if (hit) {
    cache.delete(key)
    cache.set(key, hit)
    return hit
  }
  const p = getAgentSessionUpload(sessionId, file).then(({ data }) => `data:${data.mime};base64,${data.data}`)
  p.catch(() => cache.delete(key))
  cache.set(key, p)
  if (cache.size > CACHE_MAX) cache.delete(cache.keys().next().value as string)
  return p
}
</script>
