<template>
  <div class="update-release-content">
    <p class="version-summary">
      {{
        t('layout.updateAvailableContent', {
          current: update.currentVersion,
          latest: update.latestVersion,
        })
      }}
    </p>

    <section class="release-section">
      <div class="release-heading">{{ t('layout.updateReleaseContent') }}</div>
      <div v-if="hasReleaseBody" class="release-markdown" v-html="renderedReleaseBody"></div>
      <EmptyState
        v-else
        icon="HOutline:DocumentTextIcon"
        :title="t('layout.updateReleaseEmpty')"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import MarkdownIt from 'markdown-it'
import type { CheckUpdateResponse } from '@/types/v1/system'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  update: CheckUpdateResponse
}>()

const { t } = useI18n()

// GitHub Release Body 使用 Markdown；这里关闭 HTML，避免远程内容直接注入页面。
const markdown = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
})

// 发布说明中的链接在新窗口打开，避免离开当前管理页面。
markdown.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  if (!token) return ''
  token.attrSet('target', '_blank')
  token.attrSet('rel', 'noopener noreferrer')
  return self.renderToken(tokens, idx, options)
}

const releaseBody = computed(() => props.update.releaseBody?.trim() || '')
const hasReleaseBody = computed(() => releaseBody.value.length > 0)
const renderedReleaseBody = computed(() => markdown.render(releaseBody.value))
</script>

<style scoped lang="scss">
.update-release-content {
  width: 100%;
  min-width: 0;
}

.version-summary {
  margin: 0;
  color: var(--el-text-color-regular);
}

.release-section {
  margin-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
  padding-top: 14px;
}

.release-heading {
  margin-bottom: 10px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.release-markdown {
  max-height: min(52vh, 520px);
  overflow: auto;
  padding-right: 4px;
  color: var(--el-text-color-primary);
  line-height: 1.7;
  word-break: break-word;
}

.release-markdown :deep(h1),
.release-markdown :deep(h2),
.release-markdown :deep(h3),
.release-markdown :deep(h4) {
  margin: 14px 0 8px;
  font-weight: 600;
  line-height: 1.35;
}

.release-markdown :deep(h1) {
  font-size: 1.25rem;
}

.release-markdown :deep(h2) {
  font-size: 1.125rem;
}

.release-markdown :deep(h3),
.release-markdown :deep(h4) {
  font-size: 1rem;
}

.release-markdown :deep(p),
.release-markdown :deep(ul),
.release-markdown :deep(ol),
.release-markdown :deep(pre),
.release-markdown :deep(blockquote) {
  margin: 0 0 10px;
}

.release-markdown :deep(ul),
.release-markdown :deep(ol) {
  padding-left: 20px;
}

.release-markdown :deep(a) {
  color: var(--el-color-primary);
  text-decoration: none;
}

.release-markdown :deep(a:hover) {
  text-decoration: underline;
}

.release-markdown :deep(code) {
  border-radius: 4px;
  background: var(--el-fill-color-light);
  padding: 2px 5px;
  font-size: 0.92em;
}

.release-markdown :deep(pre) {
  overflow: auto;
  border-radius: 6px;
  background: var(--el-fill-color-light);
  padding: 10px 12px;
}

.release-markdown :deep(pre code) {
  background: transparent;
  padding: 0;
}

.release-markdown :deep(blockquote) {
  border-left: 3px solid var(--el-border-color);
  padding-left: 10px;
  color: var(--el-text-color-regular);
}
</style>
