<!-- 一段 Agent 的回答（Markdown）。单独成组件：只有文字变了的这一段重新渲染，正在写的回答每来一段字，
     前面已经写完的回答不会跟着重新解析。图片显示成缩略图，点开用媒体查看层看大图。
     引用本机文件的图片（file:// 地址或绝对路径，包括指向本机图片文件的链接）换成签名下载链接直接显示；
     会话能访问的目录以外的取不到。 -->
<template>
  <div ref="root" class="agc-md" @click="onClick" v-html="html" />
  <FileMediaDialog v-if="view" :model-value="true" :name="view.name" kind="image" :url="view.url" @update:model-value="view = null" />
</template>

<script setup lang="ts">
import MarkdownIt from 'markdown-it'
import FileMediaDialog from '@/components/file/FileMediaDialog.vue'
import { fileLink } from './fileLinks'

const props = defineProps<{ text: string; sessionId?: string }>()

const html = computed(() => md.render(props.text))
const root = ref<HTMLElement | null>(null)

// 引用本机文件的图片（渲染时把路径记在 data-local 上）换成签名链接
const loadLocal = () => {
  const sessionId = props.sessionId
  if (!sessionId || !root.value) return
  for (const img of root.value.querySelectorAll<HTMLImageElement>('img[data-local]')) {
    const path = img.dataset.local || ''
    fileLink({ sessionId }, path).then(
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

// 点到图片（包括包在链接里的）就打开大图，不跟着链接跳走。
const onClick = (e: MouseEvent) => {
  const target = e.target
  if (!(target instanceof HTMLImageElement)) return
  e.preventDefault()
  const url = target.currentSrc || target.src
  if (url) view.value = { url, name: target.alt || nameOf(url) }
}
</script>

<script lang="ts">
import { isImagePath, localPathOf } from './fileLinks'

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })
// file:// 默认不让过：放它过去，渲染时换成签名链接（见下面的 link_open、image）
const validateLink = md.validateLink.bind(md)
md.validateLink = (url) => /^file:/i.test(url) || validateLink(url)
// 指向本机图片文件的链接（[说明](/路径/图.png)）就是这张图：换成图片，链接的文字作说明，
// 再由下面的 image 换成签名链接显示出来。链接里本来就包着图片的不动
md.core.ruler.push('local_image_links', (state) => {
  for (const block of state.tokens) {
    const list = block.type === 'inline' ? block.children : null
    if (!list) continue
    for (let i = 0; i < list.length; i++) {
      const open = list[i]
      if (open?.type !== 'link_open') continue
      const href = open.attrGet('href') || ''
      if (!isImagePath(localPathOf(href))) continue
      let end = i + 1
      while (end < list.length && list[end]?.type !== 'link_close') end++
      const inner = list.slice(i + 1, end)
      if (end >= list.length || inner.some((t) => t.type === 'image')) continue
      const img = new state.Token('image', 'img', 0)
      img.attrs = [
        ['src', href],
        ['alt', ''],
      ]
      img.children = inner
      img.content = inner.map((t) => t.content).join('')
      list.splice(i, end - i + 1, img)
    }
  }
})
const defaultLink = md.renderer.rules.link_open ?? ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))
md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  const path = localPathOf(token?.attrGet('href') || '')
  if (token && path) {
    // 本机的其他文件在浏览器里打不开：不跳转，悬停显示路径
    token.attrs = (token.attrs ?? []).filter(([k]) => k !== 'href')
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
