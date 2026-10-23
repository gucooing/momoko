<!-- 模型回复渲染：把回复拼成独立 HTML 文档放进 sandbox iframe（禁止脚本），Markdown/HTML/SVG 都能正常显示。
     preview 模式用于卡片缩略：不可交互、隐藏滚动。 -->
<template>
  <iframe
    class="reply-frame"
    :class="{ 'reply-frame--preview': preview }"
    :srcdoc="doc"
    sandbox=""
    referrerpolicy="no-referrer"
    loading="lazy"
    :title="title"
  />
</template>

<script setup lang="ts">
import { buildReplyDocument } from '@/utils/iqcheckRender'

const props = withDefaults(defineProps<{ text: string; dark?: boolean; preview?: boolean; title?: string }>(), {
  dark: false,
  preview: false,
  title: '',
})

const doc = computed(() => buildReplyDocument(props.text, { dark: props.dark, preview: props.preview }))
</script>

<style scoped>
.reply-frame {
  display: block;
  width: 100%;
  height: 100%;
  border: 0;
  background: transparent;
}
.reply-frame--preview {
  pointer-events: none;
}
</style>
