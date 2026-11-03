<!-- 公开分享页（无需登录）：主色 hero（分享者 + 名称 + 文件数/有效期/剩余次数贴纸）+ 上浮卡片。
     需要提取码时卡片为输入门；验证后为文件列表（PC 表格可多选打包下载，手机卡片行 + 底部打包下载）。 -->
<template>
  <div class="sp">
    <section class="sp-hero">
      <span class="sp-hero__blob" aria-hidden="true" />
      <span class="sp-hero__ring" aria-hidden="true" />
      <svg class="sp-hero__star" viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 1.5l2.6 7.9 7.9 2.6-7.9 2.6L12 22.5l-2.6-7.9L1.5 12l7.9-2.6z" />
      </svg>

      <div class="sp-hero__bar">
        <img :src="APP_CONFIG.logoSrc" alt="" class="sp-hero__logo" />
        <span class="sp-hero__name">{{ APP_CONFIG.name }}</span>
        <span class="sp-hero__chip">{{ t('file.share.pageTitle') }}</span>
        <span class="sp-hero__spacer" />
        <button
          type="button"
          class="sp-hero__mode"
          :aria-label="t('login.toggleTheme')"
          @click="toggleTheme"
        >
          <component :is="menuStore.iconComponents[themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon']" />
        </button>
        <a class="sp-hero__login" :href="loginHref">{{ t('publicPage.share.loginConsole') }}</a>
      </div>

      <div class="sp-hero__body">
        <template v-if="meta && !notFound">
          <div class="sp-owner">
            <img :src="ownerAvatarUrl" alt="" class="sp-owner__avatar" />
            <i18n-t scope="global" keypath="publicPage.share.sharedBy" tag="span">
              <template #name>
                <b>{{ meta.ownerName || t('publicPage.share.someone') }}</b>
              </template>
              <template #what>{{ sharedWhat }}</template>
            </i18n-t>
          </div>
          <h1 class="sp-title">
            <span class="sp-title__ico" aria-hidden="true">
              <component :is="menuStore.iconComponents[heroIcon]" />
            </span>
            <span class="sp-title__text">{{ meta.name }}</span>
          </h1>
          <div class="sp-tags">
            <span class="sp-tag sp-tag--a">
              <component :is="menuStore.iconComponents['HOutline:DocumentDuplicateIcon']" />
              <span class="m-num">{{ countText }}</span>
            </span>
            <span class="sp-tag sp-tag--b">
              <component :is="menuStore.iconComponents['HOutline:ClockIcon']" />
              <span class="m-num">{{ expiryText }}</span>
            </span>
            <span class="sp-tag sp-tag--c">
              <component :is="menuStore.iconComponents['HOutline:ArrowDownTrayIcon']" />
              <span class="m-num">{{ downloadsText }}</span>
            </span>
          </div>
        </template>
        <h1 v-else class="sp-title">
          <span class="sp-title__text">{{ loading ? t('file.share.loading') : t('file.share.pageTitle') }}</span>
        </h1>
      </div>
    </section>

    <main class="sp-main">
      <!-- 加载 -->
      <section v-if="loading" class="sp-card sp-card--narrow sp-state">
        <span class="sp-skel" />
        <span class="sp-skel sp-skel--sm" />
      </section>

      <!-- 不存在 / 不可用 -->
      <section v-else-if="notFound || (meta && !meta.available)" class="sp-card sp-card--narrow sp-state">
        <span class="m-sticker m-sticker--bad sp-state__ico">
          <component :is="menuStore.iconComponents['HOutline:LinkSlashIcon']" />
        </span>
        <h2>{{ notFound ? t('file.share.notFound') : t('publicPage.share.unavailableTitle') }}</h2>
        <p>{{ notFound ? t('publicPage.share.notFoundDesc') : t('file.share.unavailable') }}</p>
      </section>

      <!-- 提取码门 -->
      <section
        v-else-if="meta && meta.needCode && !codeVerified"
        class="sp-card sp-card--narrow sp-code"
        :aria-label="t('publicPage.share.codeTitle')"
      >
        <div class="sp-code__head">
          <span class="m-sticker m-sticker--a sp-code__ico">
            <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />
          </span>
          <div>
            <h2>{{ t('publicPage.share.codeTitle') }}</h2>
            <p>
              {{
                meta.codeLength
                  ? t('publicPage.share.codeDescLen', { n: meta.codeLength })
                  : t('publicPage.share.codeDesc')
              }}
            </p>
          </div>
        </div>
        <form class="sp-code__form" @submit.prevent="verifyCode">
          <label class="sp-field">
            <span class="sp-field__label">{{ t('file.share.code') }}</span>
            <span class="sp-fld" :class="{ 'is-error': codeError }">
              <component :is="menuStore.iconComponents['HOutline:KeyIcon']" class="sp-fld__ico" />
              <input
                v-model="code"
                class="sp-fld__input m-mono"
                :maxlength="meta.codeLength || undefined"
                autocomplete="off"
                spellcheck="false"
                :placeholder="t('file.share.codePrompt')"
                :disabled="locked"
                @input="codeError = ''"
              />
            </span>
          </label>
          <p v-if="codeError" class="sp-code__err" role="alert">{{ codeError }}</p>
          <p class="sp-code__tip">
            <component :is="menuStore.iconComponents['HOutline:ShieldCheckIcon']" />
            {{ t('publicPage.share.codeLimitTip') }}
          </p>
          <UButton
            type="submit"
            color="primary"
            class="sp-btn sp-btn--lg"
            trailing-icon="i-lucide-arrow-right"
            :loading="verifying"
            :disabled="!code.trim() || locked"
          >
            {{ t('publicPage.share.enter') }}
          </UButton>
        </form>
        <div class="sp-code__link m-mono">{{ shareLinkText }}</div>
      </section>

      <!-- 文件列表 -->
      <template v-else-if="meta">
        <section class="sp-card sp-files" :aria-label="t('publicPage.share.filesLabel')">
          <div class="sp-files__bar">
            <nav class="sp-crumbs" :aria-label="t('publicPage.share.location')">
              <button type="button" class="sp-crumb" :class="{ 'is-on': !subPath }" @click="goTo('')">
                <component :is="menuStore.iconComponents['HOutline:FolderIcon']" />
                <span>{{ meta.name }}</span>
              </button>
              <template v-for="seg in subSegments" :key="seg.path">
                <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" class="sp-crumbs__sep" />
                <button
                  type="button"
                  class="sp-crumb"
                  :class="{ 'is-on': seg.path === subSegments[subSegments.length - 1]?.path }"
                  @click="goTo(seg.path)"
                >
                  <span>{{ seg.name }}</span>
                </button>
              </template>
              <span class="sp-crumbs__count m-num">{{ dirCountText }}</span>
            </nav>
            <a
              v-if="selectedPaths.length"
              class="sp-btn-link sp-btn-link--outline"
              :href="downloadSelectedUrl"
              target="_blank"
              rel="noopener"
            >
              <component :is="menuStore.iconComponents['HOutline:ArrowDownTrayIcon']" />
              {{ t('publicPage.share.downloadSelected', { n: selectedPaths.length }) }}
            </a>
            <a
              class="sp-btn-link sp-btn-link--primary sp-files__all"
              :href="downloadUrl(subPath)"
              target="_blank"
              rel="noopener"
            >
              <component :is="menuStore.iconComponents['HOutline:ArchiveBoxArrowDownIcon']" />
              {{ subPath ? t('publicPage.share.downloadFolder') : t('publicPage.share.downloadAll') }}
              <span v-if="!subPath && Number(meta.totalSize) > 0 && !meta.dirCount" class="m-num">
                · {{ formatFileSize(meta.totalSize) }}
              </span>
            </a>
          </div>

          <div v-if="dirLoading && !entries.length" class="sp-empty">{{ t('file.share.loading') }}</div>
          <EmptyState
            v-else-if="!entries.length"
            icon="HOutline:FolderOpenIcon"
            :title="t('file.share.empty')"
          />

          <template v-else>
            <!-- PC 表格 -->
            <div class="sp-table-wrap" :class="{ 'is-loading': dirLoading }">
              <table class="sp-table">
                <thead>
                  <tr>
                    <th class="sp-table__cb">
                      <input
                        type="checkbox"
                        class="sp-cb"
                        :checked="allSelected"
                        :indeterminate.prop="someSelected"
                        :aria-label="t('publicPage.share.selectAll')"
                        @change="toggleAll"
                      />
                    </th>
                    <th>{{ t('publicPage.share.colName') }}</th>
                    <th class="sp-table__size">{{ t('publicPage.share.colSize') }}</th>
                    <th class="sp-table__time">{{ t('publicPage.share.colTime') }}</th>
                    <th class="sp-table__ops">{{ t('file.share.download') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="entry in entries" :key="entry.relPath" :class="{ 'is-sel': selected[entry.relPath] }">
                    <td class="sp-table__cb">
                      <input
                        v-model="selected[entry.relPath]"
                        type="checkbox"
                        class="sp-cb"
                        :aria-label="t('publicPage.share.selectItem', { name: entry.name })"
                      />
                    </td>
                    <td>
                      <button type="button" class="sp-fname" @click="onEntry(entry)">
                        <span
                          class="m-sticker sp-fname__ico"
                          :class="`m-sticker--${kindOf(entry).tone}`"
                          :style="{ rotate: rotOf(entry.name) }"
                        >
                          <component :is="menuStore.iconComponents[kindOf(entry).icon]" />
                        </span>
                        <span class="sp-fname__text">{{ entry.name }}</span>
                        <component
                          :is="menuStore.iconComponents['HOutline:ChevronRightIcon']"
                          v-if="entry.isDir"
                          class="sp-fname__chev"
                        />
                      </button>
                    </td>
                    <td class="sp-table__size m-num" :class="{ 'is-dim': entry.isDir }">
                      {{ entry.isDir ? t('publicPage.share.folder') : formatFileSize(entry.size) }}
                    </td>
                    <td class="sp-table__time m-num">{{ shortTime(entry.updateTime) }}</td>
                    <td class="sp-table__ops">
                      <div class="sp-ops">
                        <AppIconButton
                          v-if="!entry.isDir"
                          icon="HOutline:EyeIcon"
                          :box="36"
                          :size="18"
                          :label="`${t('file.share.preview')} ${entry.name}`"
                          @click="previewFile(entry.relPath, entry.name, Number(entry.size) || 0)"
                        />
                        <a
                          class="sp-dl"
                          :href="downloadUrl(entry.relPath)"
                          target="_blank"
                          rel="noopener"
                          :aria-label="`${t('file.share.download')} ${entry.name}`"
                          :title="t('file.share.download')"
                        >
                          <component :is="menuStore.iconComponents['HOutline:ArrowDownTrayIcon']" />
                        </a>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <!-- 手机行 -->
            <ul class="sp-rows" :class="{ 'is-loading': dirLoading }">
              <li v-for="entry in entries" :key="entry.relPath" class="sp-row">
                <button type="button" class="sp-row__main" @click="onEntry(entry)">
                  <span
                    class="m-sticker sp-row__ico"
                    :class="`m-sticker--${kindOf(entry).tone}`"
                    :style="{ rotate: rotOf(entry.name) }"
                  >
                    <component :is="menuStore.iconComponents[kindOf(entry).icon]" />
                  </span>
                  <span class="sp-row__text">
                    <span class="sp-row__name">{{ entry.name }}</span>
                    <span class="sp-row__meta m-num">
                      {{ entry.isDir ? t('publicPage.share.folder') : formatFileSize(entry.size) }} ·
                      {{ shortTime(entry.updateTime) }}
                    </span>
                  </span>
                </button>
                <component
                  :is="menuStore.iconComponents['HOutline:ChevronRightIcon']"
                  v-if="entry.isDir"
                  class="sp-row__chev"
                />
                <a
                  v-else
                  class="sp-dl sp-dl--lg"
                  :href="downloadUrl(entry.relPath)"
                  target="_blank"
                  rel="noopener"
                  :aria-label="`${t('file.share.download')} ${entry.name}`"
                >
                  <component :is="menuStore.iconComponents['HOutline:ArrowDownTrayIcon']" />
                </a>
              </li>
            </ul>
          </template>
        </section>

        <div class="sp-notes">
          <span class="sp-note">
            <component :is="menuStore.iconComponents['HOutline:BoltIcon']" class="is-ok" />
            {{ t('publicPage.share.noteDirect') }}
          </span>
          <span class="sp-note">
            <component :is="menuStore.iconComponents['HOutline:InformationCircleIcon']" />
            {{ t('publicPage.share.noteCount') }}
          </span>
          <span class="sp-notes__spacer" />
          <button v-if="meta.needCode" type="button" class="sp-relock" @click="relock">
            <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />
            {{ t('publicPage.share.relock') }}
          </button>
        </div>

        <!-- 手机底部打包下载 -->
        <div class="sp-mbar">
          <a class="sp-btn-link sp-btn-link--primary sp-mbar__btn" :href="downloadUrl(subPath)" target="_blank" rel="noopener">
            <component :is="menuStore.iconComponents['HOutline:ArchiveBoxArrowDownIcon']" />
            {{ subPath ? t('publicPage.share.downloadFolder') : t('publicPage.share.downloadAll') }}
          </a>
        </div>
      </template>

      <footer class="sp-foot">{{ t('file.share.disclaimer') }}</footer>
    </main>

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
import dayjs from 'dayjs'
import { showRequestError } from '@/utils/request'
import { useThemeStore } from '@/stores/theme'
import { APP_CONFIG } from '@/config/app.config'
import { formatFileSize, splitPathSegments } from '@/utils/file'
import { resolveAvatarUrl } from '@/utils/assets'
import defaultAvatarSvg from '@/assets/defaultAvatar.svg'
import {
  getShareMetaRequest,
  createShareSessionRequest,
  listShareDirRequest,
  buildShareDownloadUrl,
} from '@/api/share'
import FileViewerDialog from '@/components/file/FileViewerDialog.vue'
import type { GetShareMetaResponse, ShareEntry } from '@/types/v1/file'

const { t } = useI18n()
const route = useRoute()
const themeStore = useThemeStore()
const menuStore = useMenuStore()
const token = String(route.params.token || '')
const loginHref = `${import.meta.env.BASE_URL.replace(/\/$/, '')}/login`

const loading = ref(true)
const notFound = ref(false)
const meta = ref<GetShareMetaResponse | null>(null)
const ownerAvatarUrl = computed(() => resolveAvatarUrl(meta.value?.ownerAvatar) || defaultAvatarSvg)

const code = ref('')
const codeError = ref('')
const verifying = ref(false)
const lockedUntil = ref<number>(0)
const now = ref(Date.now())
const locked = computed(() => lockedUntil.value > now.value)
// 会话签名：换取后所有请求只带签名（不再逐次携带提取码）；有签名即视为已通过验证。
const sign = ref('')
const codeVerified = computed(() => !!sign.value)

const subPath = ref('')
const entries = ref<ShareEntry[]>([])
const dirLoading = ref(false)
const selected = reactive<Record<string, boolean>>({})

const toggleTheme = () => themeStore.toggleThemeMode(themeStore.isDarkTheme ? 'light' : 'dark')

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

const downloadUrl = (path = '', inline = false) => buildShareDownloadUrl(token, sign.value, { path, inline })

const selectedPaths = computed(() => entries.value.filter((e) => selected[e.relPath]).map((e) => e.relPath))
const downloadSelectedUrl = computed(() =>
  selectedPaths.value.length === 1
    ? downloadUrl(selectedPaths.value[0])
    : buildShareDownloadUrl(token, sign.value, { paths: selectedPaths.value }),
)
const allSelected = computed(() => entries.value.length > 0 && selectedPaths.value.length === entries.value.length)
const someSelected = computed(() => selectedPaths.value.length > 0 && !allSelected.value)
const clearSelection = () => {
  for (const k of Object.keys(selected)) delete selected[k]
}
const toggleAll = () => {
  const next = !allSelected.value
  clearSelection()
  if (next) for (const e of entries.value) selected[e.relPath] = true
}

// ---- hero 文案 ----
const sharedWhat = computed(() => {
  const m = meta.value
  if (!m) return ''
  const files = Number(m.fileCount) || 0
  const dirs = Number(m.dirCount) || 0
  if (dirs === 1 && files === 0) return t('publicPage.share.whatFolder')
  if (files === 1 && dirs === 0) return t('publicPage.share.whatFile')
  return t('publicPage.share.whatItems', { n: files + dirs })
})
const heroIcon = computed(() => {
  const m = meta.value
  if (m && Number(m.fileCount) === 1 && Number(m.dirCount) === 0) return 'HOutline:DocumentIcon'
  return 'HOutline:FolderIcon'
})
const countText = computed(() => {
  const m = meta.value
  if (!m) return ''
  const files = Number(m.fileCount) || 0
  const dirs = Number(m.dirCount) || 0
  const parts: string[] = []
  if (dirs) parts.push(t('publicPage.share.dirs', { n: dirs }))
  if (files) parts.push(t('publicPage.share.files', { n: files }))
  if (files && Number(m.totalSize) > 0) parts.push(formatFileSize(m.totalSize))
  return parts.join(' · ') || t('file.share.empty')
})
const expiryText = computed(() => {
  const exp = meta.value?.expiresAt
  if (!exp) return t('publicPage.share.neverExpire')
  const d = dayjs(exp as unknown as string)
  const diff = d.diff(dayjs(), 'day', true)
  const when = d.format('MM-DD HH:mm')
  if (diff <= 0) return t('publicPage.share.expiredAt', { time: when })
  if (diff < 1) return t('publicPage.share.expireHours', { time: when, n: Math.max(1, Math.ceil(diff * 24)) })
  return t('publicPage.share.expireDays', { time: when, n: Math.floor(diff) })
})
const downloadsText = computed(() => {
  const m = meta.value
  if (!m) return ''
  const max = Number(m.maxDownloads) || 0
  if (max <= 0) return t('publicPage.share.unlimited', { n: Number(m.downloadCount) || 0 })
  return t('publicPage.share.remaining', { left: Math.max(0, max - (Number(m.downloadCount) || 0)), max })
})
const shareLinkText = computed(() =>
  typeof window !== 'undefined' ? `${window.location.host}${window.location.pathname}` : '',
)
const dirCountText = computed(() => {
  const dirs = entries.value.filter((e) => e.isDir).length
  const files = entries.value.length - dirs
  const parts: string[] = []
  if (dirs) parts.push(t('publicPage.share.dirs', { n: dirs }))
  if (files) parts.push(t('publicPage.share.files', { n: files }))
  return parts.join(' · ')
})

// ---- 文件类型 → 贴纸 ----
const EXT_KIND: Record<string, { icon: string; tone: string }> = {}
const register = (exts: string, icon: string, tone: string) => {
  for (const e of exts.split(' ')) EXT_KIND[e] = { icon, tone }
}
register('png jpg jpeg gif webp svg bmp ico avif heic', 'HOutline:PhotoIcon', 'ok')
register('mp4 mov mkv webm avi flv m4v', 'HOutline:FilmIcon', 'p')
register('mp3 wav flac aac ogg m4a', 'HOutline:MusicalNoteIcon', 'p')
register('zip rar 7z tar gz tgz bz2 xz', 'HOutline:ArchiveBoxIcon', 'warn')
register('js ts go py java rs c cpp h json yaml yml toml sh vue html css xml sql', 'HOutline:CodeBracketIcon', 'a2')
const kindOf = (entry: ShareEntry) => {
  if (entry.isDir) return { icon: 'HOutline:FolderIcon', tone: 'a' }
  const ext = entry.name.includes('.') ? entry.name.split('.').pop()!.toLowerCase() : ''
  return EXT_KIND[ext] || { icon: 'HOutline:DocumentTextIcon', tone: 'a2' }
}
const ROTS = ['-4deg', '3deg', '-2deg', '4deg']
const rotOf = (name: string) => ROTS[[...name].reduce((s, c) => s + c.charCodeAt(0), 0) % ROTS.length]
const shortTime = (v: unknown) => {
  if (!v) return '—'
  const d = dayjs(v as string)
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm') : '—'
}

// 用 token(+提取码) 换取会话签名；失败时按 need_code/message 提示。返回是否成功。
const acquireSession = async (codeVal: string): Promise<boolean> => {
  try {
    const { data } = await createShareSessionRequest(token, codeVal)
    if (data.sign) {
      sign.value = data.sign
      return true
    }
    if (data.lockedUntil) {
      lockedUntil.value = dayjs(data.lockedUntil as unknown as string).valueOf()
      codeError.value = t('publicPage.share.lockedUntil', {
        time: dayjs(data.lockedUntil as unknown as string).format('HH:mm'),
      })
    } else if (codeVal && data.needCode) {
      codeError.value =
        Number(data.attemptsLeft) > 0
          ? t('publicPage.share.wrongCode', { n: data.attemptsLeft })
          : data.message || t('file.share.codePrompt')
    } else if (data.message) {
      showRequestError(new Error(data.message), data.message)
    }
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
    if (data?.name) document.title = `${data.name} · ${APP_CONFIG.name}`
    if (data?.lockedUntil) {
      lockedUntil.value = dayjs(data.lockedUntil as unknown as string).valueOf()
      codeError.value = t('publicPage.share.lockedUntil', {
        time: dayjs(data.lockedUntil as unknown as string).format('HH:mm'),
      })
    }
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
    clearSelection()
  } catch (error) {
    showRequestError(error, t('file.share.unavailable'))
  } finally {
    dirLoading.value = false
  }
}

const verifyCode = async () => {
  if (!code.value.trim() || locked.value) return
  verifying.value = true
  try {
    if (!(await acquireSession(code.value.trim()))) return
    codeError.value = ''
    await loadDir('')
  } finally {
    verifying.value = false
  }
}

// 退出分享：丢弃会话签名，回到提取码输入
const relock = () => {
  sign.value = ''
  code.value = ''
  entries.value = []
  subPath.value = ''
  clearSelection()
}

const goTo = (path: string) => loadDir(path)

const onEntry = (entry: ShareEntry) => {
  if (entry.isDir) {
    loadDir(entry.relPath)
    return
  }
  previewFile(entry.relPath, entry.name, Number(entry.size) || 0)
}

let clock: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  fetchMeta()
  clock = setInterval(() => {
    now.value = Date.now()
    if (lockedUntil.value && lockedUntil.value <= now.value && codeError.value) {
      lockedUntil.value = 0
      codeError.value = ''
    }
  }, 5000)
})
onBeforeUnmount(() => {
  if (clock) clearInterval(clock)
})
</script>

<style scoped lang="scss">
.sp {
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
  color: var(--el-text-color-primary);
}

/* ---- hero ---- */
.sp-hero {
  position: relative;
  flex: none;
  overflow: hidden;
  padding: 28px 56px 128px;
  background: var(--m-p);
  color: var(--m-pOn);
}
.sp-hero__blob {
  position: absolute;
  right: -90px;
  top: -140px;
  width: 420px;
  height: 420px;
  border-radius: 50%;
  background: var(--m-a);
}
.sp-hero__ring {
  position: absolute;
  right: 360px;
  bottom: -40px;
  width: 130px;
  height: 130px;
  border-radius: 50%;
  border: 20px solid var(--m-a2);
}
.sp-hero__star {
  position: absolute;
  left: 52%;
  top: 64px;
  width: 52px;
  height: 52px;
  fill: var(--m-pOn);
}
.sp-hero__bar {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
}
.sp-hero__logo {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  object-fit: cover;
  transform: rotate(-6deg);
  box-shadow: 0 0 0 3px var(--m-pOn);
}
.sp-hero__name {
  font-family: var(--app-font-display);
  font-size: 1.5rem;
  font-weight: 800;
}
.sp-hero__chip {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 12px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--m-pOn) 18%, transparent);
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
}
.sp-hero__spacer {
  flex: 1;
}
.sp-hero__mode {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 14px;
  background: var(--m-aSoft);
  color: var(--m-aOn);
  cursor: pointer;
}
.sp-hero__mode svg {
  width: 20px;
  height: 20px;
}
html.dark .sp-hero__mode {
  color: var(--m-a);
}
.sp-hero__login {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  padding: 0 18px;
  border-radius: 999px;
  background: var(--m-pOn);
  color: var(--m-p);
  font-size: 0.875rem;
  font-weight: 700;
  text-decoration: none;
  white-space: nowrap;
}
.sp-hero__login:hover {
  color: var(--m-p);
  filter: brightness(1.04);
}
.sp-hero__body {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 18px;
  max-width: 760px;
  margin-top: 48px;
}
.sp-owner {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 0.9375rem;
  font-weight: 600;
}
.sp-owner__avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--m-a);
  box-shadow: 0 0 0 2px var(--m-pOn);
}
.sp-title {
  display: flex;
  align-items: center;
  gap: 16px;
  margin: 0;
  font-family: var(--app-font-display);
  font-size: clamp(2.25rem, 4vw, 3.25rem);
  line-height: 1.1;
  font-weight: 800;
  letter-spacing: -0.02em;
  min-width: 0;
}
.sp-title__ico {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 64px;
  height: 64px;
  border-radius: 20px;
  background: var(--m-pOn);
  color: var(--m-p);
  transform: rotate(-6deg);
}
.sp-title__ico svg {
  width: 32px;
  height: 32px;
}
.sp-title__text {
  min-width: 0;
  overflow-wrap: anywhere;
}
.sp-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 4px;
}
.sp-tag {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 18px;
  border-radius: 999px;
  font-size: 1rem;
  font-weight: 800;
  white-space: nowrap;
  box-shadow: 0 10px 24px -12px rgba(0, 0, 0, 0.35);
}
.sp-tag svg {
  width: 16px;
  height: 16px;
  flex: none;
}
.sp-tag--a {
  background: var(--m-pOn);
  color: var(--m-p);
  transform: rotate(-2deg);
}
.sp-tag--b {
  background: var(--m-ink);
  color: var(--m-bg);
  transform: rotate(2deg);
}
.sp-tag--c {
  background: var(--m-a2);
  color: #fff;
  transform: rotate(-1deg);
}

/* ---- main ---- */
.sp-main {
  position: relative;
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18px;
  margin-top: -84px;
  padding: 0 56px;
}
.sp-card {
  width: 100%;
  max-width: 1080px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-xl);
  box-shadow: 0 30px 60px -36px color-mix(in srgb, var(--m-ink) 45%, transparent);
}
.sp-card--narrow {
  max-width: 520px;
}
.sp-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 40px 32px;
  text-align: center;
}
.sp-state h2 {
  margin: 6px 0 0;
  font-family: var(--app-font-display);
  font-size: 1.5rem;
  font-weight: 800;
}
.sp-state p {
  margin: 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.sp-state__ico {
  width: 64px;
  height: 64px;
  border-radius: 20px;
  transform: rotate(-6deg);
}
.sp-state__ico svg {
  width: 30px;
  height: 30px;
}
.sp-skel {
  width: 70%;
  height: 18px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  animation: sp-pulse 1.4s ease-in-out infinite;
}
.sp-skel--sm {
  width: 45%;
}
@keyframes sp-pulse {
  50% {
    opacity: 0.5;
  }
}

/* 提取码 */
.sp-code {
  display: flex;
  flex-direction: column;
  gap: 22px;
  padding: 36px;
}
.sp-code__head {
  display: flex;
  align-items: center;
  gap: 14px;
}
.sp-code__head h2 {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.625rem;
  font-weight: 800;
}
.sp-code__head p {
  margin: 4px 0 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.sp-code__ico {
  width: 56px;
  height: 56px;
  border-radius: 18px;
  transform: rotate(-6deg);
}
.sp-code__ico svg {
  width: 26px;
  height: 26px;
}
.sp-code__form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.sp-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.sp-field__label {
  font-size: 0.8125rem;
  font-weight: 700;
}
.sp-fld {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 0 16px;
  border-radius: var(--app-radius);
  border: 1.5px solid transparent;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  transition:
    border-color 0.15s,
    background-color 0.15s;
}
.sp-fld:focus-within {
  border-color: var(--m-p);
  background: var(--el-bg-color);
}
.sp-fld.is-error {
  border-color: var(--m-bad);
}
.sp-fld__ico {
  flex: none;
  width: 20px;
  height: 20px;
}
.sp-fld__input {
  flex: 1;
  min-width: 0;
  height: 48px;
  border: 0;
  background: transparent;
  font-size: 1rem;
  font-weight: 600;
  letter-spacing: 0.12em;
  color: var(--el-text-color-primary);
  outline: none;
}
.sp-fld__input::placeholder {
  letter-spacing: 0;
  font-family: var(--app-font-sans);
  font-weight: 500;
  color: var(--el-text-color-placeholder);
}
.sp-code__err {
  margin: -6px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--m-bad);
}
.sp-code__tip {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.sp-code__tip svg {
  width: 16px;
  height: 16px;
  flex: none;
}
.sp-code__link {
  padding: 12px 14px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  text-align: center;
  overflow-wrap: anywhere;
}
.sp-btn {
  justify-content: center;
  border-radius: 999px !important;
  font-weight: 800 !important;
}
.sp-btn--lg {
  min-height: 56px;
  font-size: 1rem !important;
  box-shadow: var(--app-shadow-brand);
}

/* 文件列表 */
.sp-files {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px 10px;
}
.sp-files__bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 4px 4px 8px;
}
.sp-crumbs {
  flex: 1 1 320px;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 2px;
  overflow-x: auto;
  scrollbar-width: none;
}
.sp-crumbs::-webkit-scrollbar {
  display: none;
}
.sp-crumbs__sep {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.sp-crumbs__count {
  margin-left: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.sp-crumb {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: none;
  max-width: 260px;
  min-height: 40px;
  padding: 0 12px;
  border: 0;
  border-radius: 12px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
}
.sp-crumb span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sp-crumb svg {
  width: 16px;
  height: 16px;
  flex: none;
}
.sp-crumb:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.sp-crumb.is-on {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.sp-btn-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 18px;
  border-radius: 999px;
  font-size: 0.875rem;
  font-weight: 700;
  text-decoration: none;
  white-space: nowrap;
  transition:
    filter 0.15s,
    border-color 0.15s;
}
.sp-btn-link svg {
  width: 18px;
  height: 18px;
  flex: none;
}
.sp-btn-link--primary {
  background: var(--m-p);
  color: var(--m-pOn);
}
.sp-btn-link--primary:hover {
  color: var(--m-pOn);
  filter: brightness(1.08);
}
.sp-btn-link--outline {
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
}
.sp-btn-link--outline:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.sp-empty {
  padding: 40px 0;
  text-align: center;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.sp-table-wrap {
  overflow-x: auto;
  scrollbar-width: none;
  transition: opacity 0.15s;
}
.sp-table-wrap::-webkit-scrollbar {
  display: none;
}
.is-loading {
  opacity: 0.6;
}
.sp-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}
.sp-table th {
  height: 40px;
  padding: 0 12px;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.sp-table td {
  height: 60px;
  padding: 0 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.sp-table tbody tr:last-child td {
  border-bottom: 0;
}
.sp-table tbody tr {
  transition: background-color 0.12s;
}
.sp-table tbody tr:hover {
  background: var(--el-fill-color-lighter);
}
.sp-table tbody tr.is-sel {
  background: var(--m-pSoft);
}
.sp-table__cb {
  width: 44px;
}
.sp-table__size {
  width: 130px;
  font-weight: 700;
  font-size: 0.8125rem;
}
.sp-table__size.is-dim {
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.sp-table__time {
  width: 170px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.sp-table__ops {
  width: 120px;
  text-align: right !important;
}
.sp-cb {
  width: 18px;
  height: 18px;
  accent-color: var(--m-p);
  cursor: pointer;
}
.sp-fname {
  display: flex;
  align-items: center;
  gap: 12px;
  max-width: 100%;
  min-height: 44px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.sp-fname:hover .sp-fname__text {
  color: var(--m-pInk);
}
.sp-fname__ico,
.sp-row__ico {
  width: 38px;
  height: 38px;
  border-radius: 13px;
}
.sp-fname__ico svg,
.sp-row__ico svg {
  width: 18px;
  height: 18px;
}
.sp-fname__text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 700;
}
.sp-fname__chev {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.sp-ops {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 4px;
}
.sp-dl {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 36px;
  height: 36px;
  border-radius: 12px;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  transition: filter 0.15s;
}
.sp-dl:hover {
  color: var(--m-pInk);
  filter: brightness(0.96);
}
.sp-dl svg {
  width: 18px;
  height: 18px;
}
.sp-dl--lg {
  width: 44px;
  height: 44px;
  border-radius: 14px;
}
.sp-rows,
.sp-mbar {
  display: none;
}

.sp-notes {
  width: 100%;
  max-width: 1080px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 20px;
  padding: 0 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.sp-note {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.sp-note svg {
  width: 16px;
  height: 16px;
  flex: none;
}
.sp-note svg.is-ok {
  color: var(--m-ok);
}
.sp-notes__spacer {
  flex: 1;
}
.sp-relock {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.sp-relock:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.sp-relock svg {
  width: 16px;
  height: 16px;
}
.sp-foot {
  margin-top: auto;
  padding: 24px 0 28px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  text-align: center;
}

/* ---- 手机 ---- */
@media (width < 768px) {
  .sp-hero {
    padding: 18px 20px 58px;
  }
  .sp-hero__blob {
    right: -70px;
    top: -80px;
    width: 220px;
    height: 220px;
  }
  .sp-hero__ring {
    right: 36px;
    bottom: 6px;
    width: 64px;
    height: 64px;
    border-width: 11px;
  }
  .sp-hero__star,
  .sp-hero__chip,
  .sp-title__ico {
    display: none;
  }
  .sp-hero__bar {
    gap: 10px;
  }
  .sp-hero__logo {
    width: 38px;
    height: 38px;
    border-radius: 12px;
  }
  .sp-hero__name {
    font-size: 1.25rem;
  }
  .sp-hero__login {
    min-height: 40px;
    padding: 0 14px;
    font-size: 0.8125rem;
  }
  .sp-hero__mode {
    width: 40px;
    height: 40px;
  }
  .sp-hero__body {
    margin-top: 22px;
    gap: 10px;
  }
  .sp-owner {
    gap: 8px;
    font-size: 0.875rem;
  }
  .sp-owner__avatar {
    width: 30px;
    height: 30px;
  }
  .sp-title {
    font-size: 2rem;
    line-height: 1.15;
  }
  .sp-tags {
    gap: 8px;
    margin-top: 4px;
    max-width: 320px;
  }
  .sp-tag {
    height: 34px;
    padding: 0 12px;
    font-size: 0.8125rem;
  }
  .sp-tag svg {
    width: 14px;
    height: 14px;
  }

  .sp-main {
    margin-top: -32px;
    padding: 0;
    gap: 12px;
  }
  .sp-card,
  .sp-card--narrow {
    max-width: none;
    border: 0;
    border-radius: 30px 30px 0 0;
    box-shadow: none;
  }
  .sp-code {
    padding: 26px 20px 20px;
    gap: 16px;
  }
  .sp-code__head h2 {
    font-size: 1.5rem;
  }
  .sp-code__ico {
    width: 48px;
    height: 48px;
    border-radius: 16px;
  }
  .sp-files {
    padding: 14px 12px 4px 16px;
  }
  .sp-files__bar {
    padding: 4px 0;
  }
  .sp-crumbs {
    flex: 1 1 100%;
  }
  .sp-files__all,
  .sp-files__bar .sp-btn-link--outline,
  .sp-table-wrap {
    display: none;
  }
  .sp-rows {
    display: flex;
    flex-direction: column;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .sp-row {
    display: flex;
    align-items: center;
    gap: 8px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  .sp-row:last-child {
    border-bottom: 0;
  }
  .sp-row__main {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 12px;
    min-height: 60px;
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--el-text-color-primary);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  .sp-row__text {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .sp-row__name {
    font-size: 0.9375rem;
    font-weight: 700;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sp-row__meta {
    font-size: 0.75rem;
    color: var(--el-text-color-secondary);
  }
  .sp-row__chev {
    flex: none;
    width: 44px;
    height: 20px;
    color: var(--el-text-color-secondary);
  }
  .sp-notes {
    padding: 0 16px;
    gap: 6px 14px;
    font-size: 0.75rem;
  }
  .sp-mbar {
    position: sticky;
    bottom: 0;
    z-index: 3;
    display: flex;
    width: 100%;
    padding: 12px 16px calc(16px + env(safe-area-inset-bottom));
    border-top: 1px solid var(--el-border-color-lighter);
    background: var(--el-bg-color);
  }
  .sp-mbar__btn {
    flex: 1;
    min-height: 52px;
    font-size: 0.9375rem;
  }
  .sp-foot {
    padding: 8px 16px 20px;
  }
}
</style>
