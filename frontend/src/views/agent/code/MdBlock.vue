<!-- 一段 Agent 的回答（Markdown）。单独成组件：只有文字变了的这一段重新渲染，正在写的回答每来一段字，
     前面已经写完的回答不会跟着重新解析。图片显示成缩略图，点开用媒体查看层看大图。
     引用本机文件的图片和链接（file:// 地址或绝对路径），按会话从接口取回来显示；会话能访问的目录以外的取不到。 -->
<template>
  <div ref="root" class="agc-md" @click="onClick" v-html="html" />
  <FileMediaDialog v-if="view" :model-value="true" :name="view.name" kind="image" :url="view.url" @update:model-value="view = null" />
</template>

<script setup lang="ts">
import MarkdownIt from 'markdown-it'
import FileMediaDialog from '@/components/file/FileMediaDialog.vue'
import { localImage } from './images'

const props = defineProps<{ text: string; sessionId?: string }>()

const html = computed(() => md.render(props.text))
const root = ref<HTMLElement | null>(null)

// 引用本机文件的图片（渲染时把路径记在 data-local 上）取回来再显示
const loadLocal = () => {
  const sessionId = props.sessionId
  if (!sessionId || !root.value) return
  for (const img of root.value.querySelectorAll<HTMLImageElement>('img[data-local]')) {
    const path = img.dataset.local || ''
    localImage(sessionId, path).then(
      (url) => {
        if (url && img.dataset.local === path) img.src = url
      },
      () => {},
    )
  }
}
watch(html, () => nextTick(loadLocal))
onMounted(loadLocal)

const view = shallowRef<{ url: string; name: string } | null>(null)

/** 图片没写说明时用地址里的文件名。 */
const nameOf = (src: string) => {
  try {
    return decodeURIComponent(new URL(src, location.href).pathname.split('/').pop() || '') || 'image'
  } catch {
    return 'image'
  }
}

// 点到图片（包括包在链接里的）就打开大图，不跟着链接跳走；点到指向本机图片的链接也打开大图。
const onClick = (e: MouseEvent) => {
  const target = e.target
  if (target instanceof HTMLImageElement) {
    e.preventDefault()
    const url = target.currentSrc || target.src
    if (url) view.value = { url, name: target.alt || nameOf(url) }
    return
  }
  const link = target instanceof Element ? target.closest<HTMLElement>('a[data-local]') : null
  const path = link?.dataset.local || ''
  if (!path || !props.sessionId) return
  e.preventDefault()
  localImage(props.sessionId, path).then(
    (url) => {
      if (url.startsWith('data:image/')) view.value = { url, name: path.split(/[\\/]/).pop() || 'image' }
    },
    () => {},
  )
}
</script>

<script lang="ts">
import { localPathOf } from './images'

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })
// file:// 默认不让过：放它过去，渲染时换成从接口取（见下面的 link_open、image）
const validateLink = md.validateLink.bind(md)
md.validateLink = (url) => /^file:/i.test(url) || validateLink(url)
const defaultLink = md.renderer.rules.link_open ?? ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))
md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  const path = localPathOf(token?.attrGet('href') || '')
  if (token && path) {
    // 本机文件在浏览器里打不开：不跳转，点了按会话取回来（是图片就看大图）
    token.attrs = (token.attrs ?? []).filter(([k]) => k !== 'href')
    token.attrSet('data-local', path)
    token.attrSet('title', path)
    return defaultLink(tokens, idx, options, env, self)
  }
  token?.attrSet('target', '_blank')
  token?.attrSet('rel', 'noopener noreferrer')
  return defaultLink(tokens, idx, options, env, self)
}
const defaultImage = md.renderer.rules.image ?? ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))
md.renderer.rules.image = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  token?.attrSet('loading', 'lazy')
  token?.attrSet('decoding', 'async')
  const path = localPathOf(token?.attrGet('src') || '')
  if (token && path) {
    token.attrs = (token.attrs ?? []).filter(([k]) => k !== 'src')
    token.attrSet('data-local', path)
  }
  return defaultImage(tokens, idx, options, env, self)
}
</script>
