<template>
  <div class="share-page">
    <!-- 顶栏 -->
    <header class="share-topbar">
      <div class="topbar-inner">
        <div class="brand">
          <span class="brand-logo">M</span>
          <span class="brand-text">{{ t('file.share.pageTitle') }}</span>
        </div>
      </div>
    </header>

    <main class="share-main">
      <div class="share-card">
        <div v-if="loading" class="share-state">{{ t('file.share.loading') }}</div>

        <div v-else-if="notFound" class="share-state">{{ t('file.share.notFound') }}</div>

        <template v-else>
          <!-- 头部：分享信息 + 操作 -->
          <div class="card-head">
            <div class="head-info">
              <el-icon class="head-icon" :size="40"><component :is="meta.isDir ? Folder : kindIcon(meta.name)" /></el-icon>
              <div class="head-text">
                <div class="head-name" :title="meta.name">{{ meta.name }}</div>
                <div class="head-sub">
                  <span>{{ meta.expiresAt ? t('file.share.expiresAt', { time: formatTime(meta.expiresAt) }) : t('file.share.never') }}</span>
                  <span v-if="!meta.isDir"> · {{ formatSize(meta.size) }}</span>
                </div>
              </div>
            </div>
            <div v-if="meta.available && unlocked" class="head-actions">
              <el-button
                v-if="!meta.isDir && isPreviewable(meta.name)"
                :icon="View"
                @click="previewItem(meta.name, '')"
              >
                {{ t('file.share.preview') }}
              </el-button>
              <el-button type="primary" :icon="Download" @click="download(meta.isDir ? subPath : '')">
                {{ meta.isDir ? t('file.share.downloadZip') : t('file.share.download') }}
              </el-button>
            </div>
          </div>

          <div v-if="!meta.available" class="share-state share-unavailable">
            {{ t('file.share.unavailable') }}
          </div>

          <!-- 提取码门禁 -->
          <div v-else-if="meta.needCode && !unlocked" class="share-gate">
            <el-input
              v-model="code"
              :placeholder="t('file.share.codePrompt')"
              maxlength="16"
              size="large"
              @keyup.enter="unlock"
            />
            <el-button type="primary" size="large" :loading="unlocking" @click="unlock">
              {{ t('file.share.access') }}
            </el-button>
          </div>

          <!-- 文件列表 -->
          <template v-else>
            <div class="list-bar">
              <span class="list-count">{{ t('file.share.itemCount', { count: displayEntries.length }) }}</span>
              <nav v-if="meta.isDir" class="crumbs">
                <a class="crumb" :class="{ active: !crumbs.length }" @click="goTo('')">{{ t('file.share.allFiles') }}</a>
                <template v-for="(seg, i) in crumbs" :key="i">
                  <span class="crumb-sep">›</span>
                  <a class="crumb" :class="{ active: i === crumbs.length - 1 }" @click="goTo(crumbPath(i))">{{ seg }}</a>
                </template>
              </nav>
            </div>

            <div class="list-head">
              <span class="col-name">{{ t('file.share.name') }}</span>
              <span class="col-time">{{ t('file.share.modifiedTime') }}</span>
              <span class="col-size">{{ t('file.share.size') }}</span>
              <span class="col-ops"></span>
            </div>

            <div v-loading="browsing" class="file-list">
              <div
                v-for="entry in displayEntries"
                :key="entry.relPath || entry.name"
                class="file-row"
                @click="onRowClick(entry)"
              >
                <div class="col-name file-main">
                  <el-icon class="file-icon" :class="{ 'is-dir': entry.isDir }" :size="22">
                    <component :is="entry.isDir ? Folder : kindIcon(entry.name)" />
                  </el-icon>
                  <div class="file-name-wrap">
                    <span class="file-name" :title="entry.name">{{ entry.name }}</span>
                    <span class="file-submeta">
                      {{ entry.isDir ? '-' : formatSize(entry.size) }}<template v-if="entry.updateTime"> · {{ formatTime(entry.updateTime) }}</template>
                    </span>
                  </div>
                </div>
                <div class="col-time file-meta">{{ entry.updateTime ? formatTime(entry.updateTime) : '-' }}</div>
                <div class="col-size file-meta">{{ entry.isDir ? '-' : formatSize(entry.size) }}</div>
                <div class="col-ops file-ops">
                  <el-button
                    v-if="!entry.isDir && isPreviewable(entry.name)"
                    link
                    :icon="View"
                    @click.stop="previewItem(entry.name, entry.relPath)"
                  />
                  <el-button link :icon="Download" @click.stop="download(entry.relPath)" />
                </div>
              </div>
              <div v-if="!displayEntries.length && !browsing" class="list-empty">{{ t('file.share.empty') }}</div>
            </div>
          </template>
        </template>
      </div>

      <p class="share-disclaimer">{{ t('file.share.disclaimer') }}</p>
    </main>

    <!-- 预览 -->
    <el-dialog v-model="previewVisible" :title="previewName" width="min(960px, 94vw)" align-center class="preview-dialog" @closed="onPreviewClosed">
      <div class="preview-body">
        <img v-if="previewKind === 'image'" :src="previewUrl" class="preview-image" :alt="previewName" />
        <video v-else-if="previewKind === 'video'" :src="previewUrl" class="preview-video" controls autoplay />
        <audio v-else-if="previewKind === 'audio'" :src="previewUrl" controls autoplay />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { Download, Document, Folder, Headset, Picture, VideoPlay, View } from '@element-plus/icons-vue'
import { buildShareDownloadUrl, getShareMetaRequest, listShareDirRequest } from '@/api/share'
import { resolveFilePreviewKind } from '@/utils/filePreview'
import type { GetShareMetaResponse, ShareEntry } from '@/types/v1/file'

defineOptions({ name: 'PublicShare' })

const { t } = useI18n()
const route = useRoute()
const token = String(route.params.token || '')

const loading = ref(true)
const notFound = ref(false)
const meta = ref<GetShareMetaResponse>({} as GetShareMetaResponse)
const code = ref('')
const unlocked = ref(false)
const unlocking = ref(false)

const subPath = ref('')
const entries = ref<ShareEntry[]>([])
const browsing = ref(false)

const crumbs = computed(() => subPath.value.split('/').filter(Boolean))
const crumbPath = (i: number) => crumbs.value.slice(0, i + 1).join('/')

// 单文件分享时也以「一行」形式展示，统一交互
const displayEntries = computed<ShareEntry[]>(() => {
  if (meta.value.isDir) return entries.value
  if (!meta.value.name) return []
  return [{ name: meta.value.name, isDir: false, size: meta.value.size, relPath: '', updateTime: undefined }]
})

const isPreviewable = (name: string) => resolveFilePreviewKind(name) !== null
const kindIcon = (name: string) => {
  switch (resolveFilePreviewKind(name)) {
    case 'image':
      return Picture
    case 'video':
      return VideoPlay
    case 'audio':
      return Headset
    default:
      return Document
  }
}

const formatTime = (v: unknown) => (v ? new Date(v as string).toLocaleString() : '')
const formatSize = (n: unknown) => {
  let size = Number(n) || 0
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

const loadMeta = async () => {
  loading.value = true
  try {
    const { data } = await getShareMetaRequest(token)
    meta.value = data
    if (data.available && !data.needCode) {
      unlocked.value = true
      if (data.isDir) await loadDir('')
    }
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

const loadDir = async (path: string) => {
  browsing.value = true
  try {
    const { data } = await listShareDirRequest({ token, code: code.value, subPath: path })
    entries.value = data?.items ?? []
    subPath.value = data?.subPath ?? path
  } finally {
    browsing.value = false
  }
}

const unlock = async () => {
  unlocking.value = true
  try {
    if (meta.value.isDir) await loadDir('')
    unlocked.value = true
  } catch {
    // 提取码错误：request 拦截器已提示，保持门禁
  } finally {
    unlocking.value = false
  }
}

const goTo = (path: string) => loadDir(path)

const download = (relPath: string) => {
  window.location.href = buildShareDownloadUrl(token, code.value, relPath)
}

const onRowClick = (entry: ShareEntry) => {
  if (entry.isDir) {
    goTo(entry.relPath)
  } else if (isPreviewable(entry.name)) {
    previewItem(entry.name, entry.relPath)
  } else {
    download(entry.relPath)
  }
}

// 预览
const previewVisible = ref(false)
const previewName = ref('')
const previewUrl = ref('')
const previewKind = ref<ReturnType<typeof resolveFilePreviewKind>>(null)

const previewItem = (name: string, relPath: string) => {
  previewKind.value = resolveFilePreviewKind(name)
  previewName.value = name
  previewUrl.value = buildShareDownloadUrl(token, code.value, relPath, true)
  previewVisible.value = true
}
const onPreviewClosed = () => {
  previewUrl.value = ''
  previewName.value = ''
}

onMounted(loadMeta)
</script>

<style scoped lang="scss">
.share-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--el-fill-color-light);
}

.share-topbar {
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.topbar-inner {
  max-width: 1100px;
  margin: 0 auto;
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 1rem;
}
.brand {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  font-size: 1.05rem;
  font-weight: 600;
}
.brand-logo {
  width: 1.9rem;
  height: 1.9rem;
  border-radius: 0.55rem;
  background: var(--el-color-primary);
  color: #fff;
  display: grid;
  place-items: center;
  font-weight: 700;
}

.share-main {
  flex: 1;
  width: 100%;
  max-width: 1100px;
  margin: 0 auto;
  padding: 1.6rem 1rem 2.4rem;
  box-sizing: border-box;
}

.share-card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
  padding: 1.4rem 1.5rem;
  box-shadow: var(--el-box-shadow-light);
}

.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}
.head-info {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  min-width: 0;
}
.head-icon {
  color: var(--el-color-primary);
  flex-shrink: 0;
}
.head-text {
  min-width: 0;
}
.head-name {
  font-size: 1.05rem;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.head-sub {
  margin-top: 0.2rem;
  font-size: 0.8rem;
  color: var(--el-text-color-secondary);
}
.head-actions {
  display: flex;
  gap: 0.6rem;
  flex-shrink: 0;
}

.share-state {
  padding: 2.5rem 0;
  text-align: center;
  color: var(--el-text-color-secondary);
}
.share-unavailable {
  color: var(--el-color-danger);
}

.share-gate {
  display: flex;
  gap: 0.6rem;
  margin-top: 1.6rem;
  max-width: 420px;
}

.list-bar {
  display: flex;
  align-items: center;
  gap: 0.8rem;
  margin-top: 1.5rem;
  padding-top: 1.1rem;
  border-top: 1px solid var(--el-border-color-lighter);
  flex-wrap: wrap;
}
.list-count {
  font-weight: 600;
}
.crumbs {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.86rem;
  color: var(--el-text-color-secondary);
  flex-wrap: wrap;
}
.crumb {
  cursor: pointer;
  color: var(--el-color-primary);
}
.crumb.active {
  color: var(--el-text-color-regular);
  cursor: default;
}
.crumb-sep {
  color: var(--el-text-color-placeholder);
}

.list-head,
.file-row {
  display: grid;
  grid-template-columns: 1fr 170px 110px 84px;
  align-items: center;
  gap: 0.6rem;
}
.list-head {
  padding: 0.7rem 0.6rem;
  font-size: 0.82rem;
  color: var(--el-text-color-secondary);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.file-row {
  padding: 0.6rem;
  border-radius: 8px;
  cursor: pointer;
  border-bottom: 1px solid var(--el-border-color-extra-light);
}
.file-row:hover {
  background: var(--el-fill-color-light);
}
.file-main {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  min-width: 0;
}
.file-icon {
  color: var(--el-color-primary);
  flex-shrink: 0;
}
.file-icon.is-dir {
  color: var(--el-color-warning);
}
.file-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.file-name-wrap {
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.file-submeta {
  display: none;
  font-size: 0.74rem;
  color: var(--el-text-color-secondary);
}
.file-meta {
  font-size: 0.82rem;
  color: var(--el-text-color-secondary);
}
.col-size {
  text-align: right;
}
.file-ops {
  display: flex;
  justify-content: flex-end;
  gap: 0.3rem;
  opacity: 0;
  transition: opacity 0.15s;
}
.file-row:hover .file-ops {
  opacity: 1;
}
.list-empty {
  padding: 2rem 0;
  text-align: center;
  color: var(--el-text-color-secondary);
}

.share-disclaimer {
  margin: 1.6rem 0 0;
  text-align: center;
  font-size: 0.78rem;
  color: var(--el-text-color-secondary);
}

.preview-body {
  display: flex;
  justify-content: center;
  align-items: center;
}
.preview-image {
  max-width: 100%;
  max-height: 72vh;
  object-fit: contain;
}
.preview-video {
  max-width: 100%;
  max-height: 72vh;
}

/* 移动端适配 */
@media (max-width: 640px) {
  .share-card {
    padding: 1.1rem;
    border-radius: 12px;
  }
  .head-actions {
    width: 100%;
  }
  .head-actions .el-button {
    flex: 1;
  }
  .share-gate {
    max-width: none;
  }
  /* 列表收起为「名称 + 操作」，时间/大小并入名称下方 */
  .list-head .col-time,
  .list-head .col-size,
  .file-row .col-time,
  .file-row .col-size {
    display: none;
  }
  .list-head,
  .file-row {
    grid-template-columns: 1fr auto;
  }
  .file-main {
    flex-direction: row;
    align-items: center;
  }
  .file-submeta {
    display: block;
  }
  .file-ops {
    opacity: 1;
  }
}
</style>
