<template>
  <div class="file-module share-public" :class="{ 'is-dark': isDark }">
    <header class="sp-site-header">
      <div class="sp-site-brand">
        <img :src="APP_CONFIG.logoSrc" :alt="APP_CONFIG.name" class="sp-site-logo" />
        <span class="sp-site-name">{{ APP_CONFIG.name }}</span>
      </div>
      <a class="sp-login-button" :href="loginHref">登录</a>
    </header>

    <div class="sp-card">
      <!-- 加载中 -->
      <div v-if="loading" class="sp-state">{{ t('file.share.loading') }}</div>

      <!-- 不存在 / 失效 -->
      <div v-else-if="notFound" class="sp-state">
        <el-icon class="sp-state-icon"><IconWarning /></el-icon>
        <p>{{ t('file.share.notFound') }}</p>
      </div>

      <!-- 已关闭/过期/超次数 -->
      <div v-else-if="meta && !meta.available" class="sp-state">
        <el-icon class="sp-state-icon"><IconWarning /></el-icon>
        <p>{{ t('file.share.unavailable') }}</p>
      </div>

      <template v-else-if="meta">
        <!-- 头部 -->
        <header class="sp-header">
          <div class="sp-header-info">
            <h1 class="sp-name">{{ meta.name }}</h1>
            <div class="sp-meta">
              <span v-if="meta.expiresAt">{{
                t('file.share.expiresAt', { time: formatDateTime(meta.expiresAt) })
              }}</span>
              <span v-if="Number(meta.maxDownloads) > 0">
                {{ t('file.share.downloads') }}: {{ meta.downloadCount }} / {{ meta.maxDownloads }}
              </span>
            </div>
          </div>
          <div v-if="meta.ownerName || meta.ownerAvatar" class="sp-owner">
            <img :src="ownerAvatarUrl" :alt="meta.ownerName" class="sp-owner-avatar" />
            <span class="sp-owner-name">{{ meta.ownerName }}</span>
          </div>
        </header>

        <!-- 提取码门 -->
        <div v-if="meta.needCode && !codeVerified" class="sp-code">
          <input
            v-model="code"
            class="fm-input sp-code-input"
            :placeholder="t('file.share.codePrompt')"
            @keyup.enter="verifyCode"
          />
          <button type="button" class="fm-btn fm-btn--primary" @click="verifyCode">
            {{ t('file.share.access') }}
          </button>
        </div>

        <template v-else>
          <div class="sp-dir">
            <div class="sp-dir-bar">
              <div class="sp-breadcrumb">
                <button type="button" class="sp-crumb" @click="goTo('')">
                  {{ t('file.share.root') }}
                </button>
                <template v-for="(seg, index) in subSegments" :key="seg.path">
                  <el-icon class="sp-crumb-sep"><IconChevronRight /></el-icon>
                  <button type="button" class="sp-crumb" @click="goTo(seg.path)">
                    {{ seg.name }}
                  </button>
                  <span v-if="index === subSegments.length - 1"></span>
                </template>
              </div>
              <a class="fm-btn" :href="downloadUrl(subPath)" target="_blank" rel="noopener">
                <el-icon><IconDownload /></el-icon>{{ t('file.share.downloadZip') }}
              </a>
            </div>

            <div v-if="dirLoading" class="sp-state">{{ t('file.share.loading') }}</div>
            <div v-else-if="!entries.length" class="sp-state">{{ t('file.share.empty') }}</div>
            <ul v-else class="sp-list">
              <li v-for="entry in entries" :key="entry.relPath" class="sp-entry">
                <button type="button" class="sp-entry-main" @click="onEntry(entry)">
                  <el-icon class="sp-entry-icon" :class="entry.isDir ? 'is-folder' : 'is-file'">
                    <component :is="entry.isDir ? IconFolder : IconFile" />
                  </el-icon>
                  <span class="sp-entry-name">{{ entry.name }}</span>
                </button>
                <div class="sp-entry-meta-row">
                  <span class="sp-entry-size">{{
                    entry.isDir ? '' : formatFileSize(entry.size)
                  }}</span>
                  <span class="sp-entry-time">{{ formatDateTime(entry.updateTime) }}</span>
                </div>
                <span class="sp-entry-ops">
                  <button
                    v-if="!entry.isDir"
                    type="button"
                    class="sp-entry-link"
                    @click.stop="previewFile(entry.relPath, entry.name, Number(entry.size) || 0)"
                  >
                    {{ t('file.share.preview') }}
                  </button>
                  <a
                    v-if="!entry.isDir"
                    class="sp-entry-link"
                    :href="downloadUrl(entry.relPath)"
                    target="_blank"
                    rel="noopener"
                    @click.stop
                  >
                    {{ t('file.share.download') }}
                  </a>
                </span>
              </li>
            </ul>
          </div>
        </template>
      </template>

      <footer class="sp-disclaimer">{{ t('file.share.disclaimer') }}</footer>
    </div>

    <FileViewerDialog
      v-model="viewer.open"
      :name="viewer.name"
      :preview-url="viewer.previewUrl"
      :download-url="viewer.downloadUrl"
      :size="viewer.size"
    />
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'PublicShareView' })
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { showRequestError } from '@/utils/request'
import { useThemeStore } from '@/stores/theme'
import { APP_CONFIG } from '@/config/app.config'
import { formatDateTime, formatFileSize, splitPathSegments } from '@/utils/file'
import { resolveAvatarUrl } from '@/utils/assets'
import defaultAvatarSvg from '@/assets/defaultAvatar.svg'
import {
  getShareMetaRequest,
  createShareSessionRequest,
  listShareDirRequest,
  buildShareDownloadUrl,
} from '@/api/share'
import FileViewerDialog from '@/components/file/FileViewerDialog.vue'
import {
  IconFolder,
  IconFile,
  IconDownload,
  IconChevronRight,
  IconWarning,
} from '@/components/file/icons'
import type { GetShareMetaResponse, ShareEntry } from '@/types/v1/file'

const { t } = useI18n()
const route = useRoute()
const themeStore = useThemeStore()
const isDark = computed(() => themeStore.isDarkTheme)
const token = String(route.params.token || '')
const loginHref = `${import.meta.env.BASE_URL.replace(/\/$/, '')}/login`

const loading = ref(true)
const notFound = ref(false)
const meta = ref<GetShareMetaResponse | null>(null)
const ownerAvatarUrl = computed(() => resolveAvatarUrl(meta.value?.ownerAvatar) || defaultAvatarSvg)

const code = ref('')
// 会话签名：换取后所有请求只带签名（不再逐次携带提取码）；有签名即视为已通过验证。
const sign = ref('')
const codeVerified = computed(() => !!sign.value)

const subPath = ref('')
const entries = ref<ShareEntry[]>([])
const dirLoading = ref(false)

// 分享页内只读预览（与文件管理“打开文件”一致：文本高亮 / 媒体内联），仅查看 + 下载，不可编辑。
const viewer = reactive({ open: false, name: '', previewUrl: '', downloadUrl: '', size: 0 })
const previewFile = (relPath: string, name: string, size = 0) => {
  viewer.name = name
  viewer.previewUrl = buildShareDownloadUrl(token, sign.value, { path: relPath, inline: true })
  viewer.downloadUrl = buildShareDownloadUrl(token, sign.value, { path: relPath })
  viewer.size = size
  viewer.open = true
}

const subSegments = computed(() => {
  if (!subPath.value) return []
  return splitPathSegments(subPath.value.replace(/\\/g, '/'))
})

const downloadUrl = (path = '', inline = false) =>
  buildShareDownloadUrl(token, sign.value, { path, inline })

// 用 token(+提取码) 换取会话签名；失败时按 need_code/message 提示。返回是否成功。
const acquireSession = async (codeVal: string): Promise<boolean> => {
  try {
    const { data } = await createShareSessionRequest(token, codeVal)
    if (data.sign) {
      sign.value = data.sign
      return true
    }
    if (data.message) showRequestError(new Error(data.message), data.message)
    return false
  } catch (error) {
    showRequestError(error, t('file.share.unavailable'))
    return false
  }
}

const fetchMeta = async () => {
  loading.value = true
  try {
    const { data } = await getShareMetaRequest(token)
    meta.value = data
    // 无需提取码的分享：直接换取会话签名并加载根目录。
    if (data?.available && !data.needCode) {
      if (await acquireSession('')) {
        await loadDir('')
      }
    }
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

const loadDir = async (path: string) => {
  dirLoading.value = true
  try {
    const { data } = await listShareDirRequest({ token, sign: sign.value, subPath: path })
    entries.value = data?.items || []
    subPath.value = data?.subPath ?? path
  } catch (error) {
    showRequestError(error, t('file.share.unavailable'))
  } finally {
    dirLoading.value = false
  }
}

const verifyCode = async () => {
  if (!code.value.trim()) return
  if (!(await acquireSession(code.value))) return
  await loadDir('')
}

const goTo = (path: string) => loadDir(path)

const onEntry = (entry: ShareEntry) => {
  if (entry.isDir) {
    loadDir(entry.relPath)
    return
  }
  previewFile(entry.relPath, entry.name, Number(entry.size) || 0)
}

onMounted(fetchMeta)
</script>

<style scoped>
.share-public {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  padding: 0 1rem 3rem;
  background: var(--fm-subtle);
}
.sp-site-header {
  align-self: stretch;
  width: auto;
  height: 56px;
  margin: 0 -1rem 2rem;
  padding: 0 1.5rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  background: var(--fm-surface);
  border-bottom: 1px solid var(--fm-border);
}
.sp-site-brand {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.sp-site-logo {
  width: 32px;
  height: 32px;
  border-radius: 0.5rem;
  object-fit: cover;
  flex-shrink: 0;
}
.sp-site-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--fm-text);
  font-size: 1.05rem;
  font-weight: 700;
}
.sp-login-button {
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  padding: 0 0.875rem;
  border-radius: 0.375rem;
  background: var(--fm-accent);
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  line-height: 1;
  text-decoration: none;
}
.sp-login-button:hover {
  color: #fff;
  filter: brightness(0.96);
}
.sp-card {
  width: 100%;
  max-width: 760px;
  display: flex;
  flex-direction: column;
  background: var(--fm-surface);
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius);
  box-shadow: var(--fm-shadow-sm);
  overflow: hidden;
}
.sp-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.625rem;
  padding: 3rem 1rem;
  color: var(--fm-text-3);
  font-size: 14px;
}
.sp-state-icon {
  font-size: 34px;
  color: var(--fm-folder);
}
.sp-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.5rem;
  border-bottom: 1px solid var(--fm-border);
}
.sp-header-info {
  min-width: 0;
  flex: 1;
}
.sp-name {
  margin: 0 0 0.25rem;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--fm-text);
  word-break: break-all;
}
.sp-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  font-size: 12.5px;
  color: var(--fm-text-3);
}
.sp-owner {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}
.sp-owner-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
  border: 1px solid var(--fm-border);
}
.sp-owner-name {
  font-size: 13px;
  color: var(--fm-text-2);
}
.sp-code {
  display: flex;
  gap: 0.625rem;
  padding: 1.25rem 1.5rem;
}
.sp-code-input {
  max-width: 240px;
}
.sp-dir {
  display: flex;
  flex-direction: column;
}
.sp-dir-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.875rem 1.5rem;
  border-bottom: 1px solid var(--fm-border);
}
.sp-breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  flex-wrap: wrap;
}
.sp-crumb {
  border: none;
  background: transparent;
  color: var(--fm-text-2);
  font-size: 13px;
  cursor: pointer;
}
.sp-crumb:hover {
  color: var(--fm-accent);
}
.sp-crumb-sep {
  font-size: 13px;
  color: var(--fm-text-3);
}
.sp-list {
  margin: 0;
  padding: 0;
  list-style: none;
}
.sp-entry {
  display: grid;
  grid-template-columns: 1fr 290px 90px;
  align-items: center;
  gap: 0.5rem;
  padding: 0 1.5rem;
  height: 46px;
  border-bottom: 1px solid var(--fm-border);
  font-size: 13px;
}
.sp-entry:hover {
  background: var(--fm-hover);
}
.sp-entry-main {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border: none;
  background: transparent;
  color: var(--fm-text);
  cursor: pointer;
  min-width: 0;
  text-align: left;
}
.sp-entry-icon {
  font-size: 18px;
  flex-shrink: 0;
}
.sp-entry-icon.is-folder {
  color: var(--fm-folder);
}
.sp-entry-icon.is-file {
  color: var(--fm-text-3);
}
.sp-entry-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sp-entry-meta-row {
  display: grid;
  grid-template-columns: 110px 170px;
  gap: 0.5rem;
}
.sp-entry-size,
.sp-entry-time {
  color: var(--fm-text-3);
  font-size: 12px;
}
.sp-entry-ops {
  text-align: right;
}
.sp-entry-link {
  color: var(--fm-accent);
  font-size: 12.5px;
  border: none;
  background: transparent;
  cursor: pointer;
  padding: 0 0 0 0.625rem;
}
.sp-disclaimer {
  padding: 1rem 1.5rem;
  font-size: 12px;
  color: var(--fm-text-3);
  border-top: 1px solid var(--fm-border);
  background: var(--fm-subtle);
}

@media (max-width: 767px) {
  .share-public {
    min-height: 100dvh;
    padding: 0 0.75rem 1rem;
    align-items: center;
    justify-content: flex-start;
  }
  .sp-site-header {
    height: 52px;
    margin: 0 -0.75rem 0.75rem;
    padding: 0 1rem;
  }
  .sp-site-logo {
    width: 28px;
    height: 28px;
  }
  .sp-site-name {
    font-size: 1rem;
  }
  .sp-login-button {
    height: 30px;
    padding: 0 0.75rem;
  }
  .sp-card {
    max-width: none;
    border-radius: 0.5rem;
  }
  .sp-state {
    padding: 2rem 1rem;
  }
  .sp-header {
    align-items: center;
    gap: 0.75rem;
    padding: 1rem;
  }
  .sp-header-info {
    min-width: 0;
    flex: 1;
  }
  .sp-name {
    font-size: 1rem;
  }
  .sp-meta {
    gap: 0.35rem;
    flex-direction: column;
  }
  .sp-owner {
    width: auto;
    margin-left: auto;
    max-width: 132px;
  }
  .sp-owner-name {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sp-code {
    padding: 1rem;
    flex-direction: column;
  }
  .sp-code-input {
    max-width: none;
  }
  .sp-code .fm-btn {
    width: 100%;
  }
  .sp-dir-bar {
    align-items: center;
    flex-direction: row;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
  }
  .sp-breadcrumb {
    flex: 1;
    min-width: 0;
    max-width: none;
    overflow: hidden;
  }
  .sp-crumb {
    min-width: 0;
    max-width: 9rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sp-dir-bar .fm-btn {
    width: auto;
    flex-shrink: 0;
    padding: 0 0.65rem;
  }
  .sp-entry {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    height: 44px;
    padding: 0 1rem;
  }
  .sp-entry-main {
    flex: 1;
    min-width: 0;
  }
  .sp-entry-meta-row {
    display: flex;
    flex-shrink: 0;
    padding-left: 0;
    white-space: nowrap;
  }
  .sp-entry-size {
    display: none;
  }
  .sp-entry-ops {
    display: flex;
    flex-shrink: 0;
    gap: 0.45rem;
    padding-left: 0;
  }
  .sp-entry-link {
    padding: 0;
    white-space: nowrap;
  }
  .sp-disclaimer {
    padding: 0.8rem 1rem;
  }
}
</style>
