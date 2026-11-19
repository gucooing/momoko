<!-- 一段 Agent 的回答（Markdown）。单独成组件：只有文字变了的这一段重新渲染，正在写的回答每来一段字，
     前面已经写完的回答不会跟着重新解析。图片显示成缩略图，点开用媒体查看层看大图。 -->
<template>
  <div class="agc-md" @click="onClick" v-html="html" />
  <FileMediaDialog v-if="view" :model-value="true" :name="view.name" kind="image" :url="view.url" @update:model-value="view = null" />
</template>

<script setup lang="ts">
import MarkdownIt from 'markdown-it'
import FileMediaDialog from '@/components/file/FileMediaDialog.vue'

const props = defineProps<{ text: string }>()

const html = computed(() => md.render(props.text))

const view = shallowRef<{ url: string; name: string } | null>(null)

/** 图片没写说明时用地址里的文件名。 */
const nameOf = (src: string) => {
  try {
    return decodeURIComponent(new URL(src, location.href).pathname.split('/').pop() || '') || 'image'
  } catch {
    return 'image'
  }
}

// 点到图片（包括包在链接里的）就打开大图，不跟着链接跳走。
const onClick = (e: MouseEvent) => {
  const target = e.target
  if (!(target instanceof HTMLImageElement)) return
  e.preventDefault()
  const url = target.currentSrc || target.src
  view.value = { url, name: target.alt || nameOf(url) }
}
</script>

<script lang="ts">
const md = new MarkdownIt({ html: false, linkify: true, breaks: true })
const defaultLink = md.renderer.rules.link_open ?? ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))
md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  tokens[idx]?.attrSet('target', '_blank')
  tokens[idx]?.attrSet('rel', 'noopener noreferrer')
  return defaultLink(tokens, idx, options, env, self)
}
const defaultImage = md.renderer.rules.image ?? ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))
md.renderer.rules.image = (tokens, idx, options, env, self) => {
  tokens[idx]?.attrSet('loading', 'lazy')
  tokens[idx]?.attrSet('decoding', 'async')
  return defaultImage(tokens, idx, options, env, self)
}
</script>
