<!-- Sub2API 公开绘图（P7 去 EP）：
     顶栏 AppSelect(Key/Model) + AppIconButton(主题)；任务网格；底栏 composer（令牌 seg + textarea）；
     参数 / 任务详情 / 删除确认 / 图片预览 全部 FormDialog；禁 el-* / BaseDialog / v-loading。 -->
<template>
  <main class="s2a-imagine" :class="{ 'is-dark': isDark }">
    <div class="bg-accent" aria-hidden />

    <header class="topbar">
      <div class="topbar-inner">
        <div class="brand">
          <span class="brand-dot" />
          <span class="brand-title">{{ title }}</span>
          <span v-if="store.srcHost" class="brand-host" :title="store.srcHost">{{ store.srcHost }}</span>
          <span class="status-chip" :class="`tone-${status.tone}`">
            <span class="dot" />{{ status.text }}
          </span>
        </div>
        <div class="topbar-actions">
          <AppSelect
            v-model="apiKeyIdModel"
            class="sel-key"
            :options="apiKeyOptions"
            :placeholder="t('sub2api.imagine.selectApiKey')"
          />
          <AppSelect
            v-model="store.modelId"
            class="sel-model"
            :options="modelOptions"
            :placeholder="t('sub2api.imagine.selectModel')"
            :disabled="!store.models.length"
          />
          <AppIconButton
            :icon="isDark ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
            :label="isDark ? t('sub2api.imagine.light') : t('sub2api.imagine.dark')"
            :box="34"
            @click="toggleTheme"
          />
        </div>
      </div>
    </header>

    <section class="tasks">
      <div v-if="loading && !store.generations.length" class="loading-block" role="status">
        <span class="spin" aria-hidden="true" />
        <p>{{ t('sub2api.imagine.generating') }}</p>
      </div>
      <div v-else-if="store.generations.length" class="task-grid">
        <article
          v-for="g in store.generations"
          :key="g.id"
          class="task-card"
          :class="{ active: g.id === store.currentId }"
          @click="store.selectGeneration(g.id)"
        >
          <div class="task-thumb">
            <img
              v-if="g.images?.length"
              :src="imageUrl(g.images[0]!.id)"
              :alt="g.prompt"
              loading="lazy"
            />
            <div v-else-if="g.status === 'pending'" class="task-skeleton" />
            <div v-else class="task-thumb-empty">
              <component :is="menuStore.iconComponents['HOutline:PhotoIcon']" />
            </div>
            <span class="task-mode" :class="g.mode">{{ modeLabel(g.mode) }}</span>
          </div>
          <div class="task-body">
            <p class="task-prompt">{{ g.prompt }}</p>
            <div class="task-meta">
              <span>{{ g.size }}</span>
              <span>·</span>
              <span>{{ t('sub2api.imagine.imageCount', { count: g.n }) }}</span>
              <StatusPill
                v-if="g.status === 'pending'"
                class="task-status"
                variant="primary"
                :label="t('sub2api.imagine.generating')"
              />
              <StatusPill
                v-else-if="g.status === 'failed'"
                class="task-status"
                variant="error"
                :label="t('sub2api.imagine.failed')"
              />
              <StatusPill
                v-else
                class="task-status"
                variant="success"
                :label="t('sub2api.imagine.imageCount', { count: g.resultCount })"
              />
            </div>
          </div>
        </article>
      </div>
      <div v-else class="empty-hero">
        <EmptyState
          :title="t('sub2api.imagine.firstWorkTitle')"
          :description="t('sub2api.imagine.firstWorkDesc')"
        />
      </div>
    </section>

    <!-- 紧凑输入栏 -->
    <div class="bar-wrap">
      <div
        class="composer-bar"
        :class="{ 'is-dragging': dragging }"
        @dragover.prevent="onDragOver"
        @dragleave="onDragLeave"
        @drop.prevent="onDrop"
      >
        <div class="mode-seg" role="tablist">
          <button
            type="button"
            role="tab"
            class="mode-seg__btn"
            :class="{ 'is-active': store.mode === 'text2image' }"
            :aria-selected="store.mode === 'text2image'"
            @click="store.mode = 'text2image'"
          >
            {{ t('sub2api.imagine.text2image') }}
          </button>
          <button
            type="button"
            role="tab"
            class="mode-seg__btn"
            :class="{ 'is-active': store.mode === 'image2image' }"
            :aria-selected="store.mode === 'image2image'"
            @click="store.mode = 'image2image'"
          >
            {{ t('sub2api.imagine.image2image') }}
          </button>
        </div>

        <template v-if="store.mode === 'image2image'">
          <div
            v-if="store.sourceImage"
            class="src-thumb"
            :title="t('sub2api.imagine.changeSource')"
            @click="pickSource"
          >
            <img :src="store.sourceImage" :alt="t('sub2api.imagine.sourceImage')" />
            <button
              class="src-clear"
              type="button"
              :title="t('sub2api.imagine.removeSource')"
              @click.stop="store.clearSource"
            >
              <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
            </button>
          </div>
          <button
            v-else
            class="src-add"
            type="button"
            :title="t('sub2api.imagine.uploadSource')"
            @click="pickSource"
          >
            <component :is="menuStore.iconComponents['HOutline:ArrowUpTrayIcon']" />
          </button>
        </template>

        <textarea
          v-model="store.prompt"
          class="prompt-input"
          rows="1"
          :placeholder="
            store.mode === 'image2image'
              ? t('sub2api.imagine.promptImage2Image')
              : t('sub2api.imagine.promptText2Image')
          "
          @input="autoResize"
          @keydown.ctrl.enter.prevent="store.submitGeneration"
        />

        <div class="bar-actions">
          <button
            class="set-btn"
            type="button"
            :title="t('sub2api.imagine.settings')"
            @click="settingsOpen = true"
          >
            <component :is="menuStore.iconComponents['HOutline:Cog6ToothIcon']" />
            <span class="size-badge">{{ store.resolveSize() }}</span>
          </button>
          <button
            class="send-btn"
            type="button"
            :disabled="!canSubmit || store.busy"
            :title="t('sub2api.imagine.generate')"
            @click="store.submitGeneration"
          >
            <component :is="menuStore.iconComponents['HOutline:PaperAirplaneIcon']" />
          </button>
        </div>

        <div v-if="dragging" class="drop-hint">{{ t('sub2api.imagine.dropHint') }}</div>
        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          class="src-file"
          @change="onFilePicked"
        />
      </div>
    </div>

    <!-- 参数设置 -->
    <FormDialog
      v-model="settingsOpen"
      :title="t('sub2api.imagine.paramsTitle')"
      :width="480"
      :confirm-text="t('sub2api.imagine.apply')"
      :cancel-text="t('sub2api.common.cancel')"
      @confirm="onSettingsConfirm"
    >
      <div class="settings">
        <div class="set-row">
          <label class="set-label">{{ t('sub2api.imagine.shape') }}</label>
          <div class="chip-group">
            <button
              v-for="s in SHAPE_VALUES"
              :key="s"
              type="button"
              class="chip"
              :class="{ 'is-active': draftParams.shape === s }"
              @click="draftParams.shape = s"
            >
              {{ shapeLabel(s) }}
            </button>
          </div>
        </div>
        <div v-if="draftParams.shape !== 'auto' && draftParams.shape !== 'custom'" class="set-row">
          <label class="set-label">{{ t('sub2api.imagine.scale') }}</label>
          <div class="chip-group">
            <button
              v-for="sc in SCALES"
              :key="sc"
              type="button"
              class="chip"
              :class="{ 'is-active': draftParams.scale === sc }"
              @click="draftParams.scale = sc"
            >
              {{ sc }}
            </button>
          </div>
        </div>
        <div v-if="draftParams.shape === 'custom'" class="set-row">
          <label class="set-label">{{ t('sub2api.imagine.size') }}</label>
          <div class="custom-size">
            <input
              v-model.number="draftParams.customW"
              class="app-input size-num"
              type="number"
              min="256"
              max="4096"
              step="64"
            />
            <span class="x">×</span>
            <input
              v-model.number="draftParams.customH"
              class="app-input size-num"
              type="number"
              min="256"
              max="4096"
              step="64"
            />
          </div>
        </div>
        <div class="set-row set-row--inline">
          <label class="set-label">{{ t('sub2api.imagine.count') }}</label>
          <input
            v-model.number="draftParams.n"
            class="app-input size-num"
            type="number"
            min="1"
            max="4"
            step="1"
          />
        </div>
        <div class="set-row set-row--inline">
          <label class="set-label">{{ t('sub2api.imagine.quality') }}</label>
          <AppSelect v-model="draftParams.quality" :options="qualityOptions" fit />
        </div>
        <div class="set-row set-row--inline">
          <label class="set-label">{{ t('sub2api.imagine.format') }}</label>
          <AppSelect v-model="draftParams.outputFormat" :options="formatOptions" fit />
        </div>
        <div class="set-summary">
          {{ t('sub2api.imagine.resolvedSize') }}<b>{{ draftResolvedSize }}</b>
        </div>
      </div>
    </FormDialog>

    <!-- 任务详情 -->
    <FormDialog
      v-model="detailOpen"
      :title="detailTitle"
      :width="780"
      :show-footer="false"
    >
      <div v-if="store.current" class="detail">
        <div class="detail-head">
          <span class="gen-mode" :class="store.current.mode">{{ modeLabel(store.current.mode) }}</span>
          <span class="detail-meta">{{ store.current.model }}</span>
          <span class="detail-meta"
            >{{ store.current.size }} ·
            {{ t('sub2api.imagine.imageCount', { count: store.current.n }) }}</span
          >
          <StatusPill
            v-if="store.current.status === 'pending'"
            variant="primary"
            :label="t('sub2api.imagine.generating')"
          />
          <StatusPill
            v-else-if="store.current.status === 'failed'"
            variant="error"
            :label="t('sub2api.imagine.failed')"
          />
          <StatusPill v-else variant="success" :label="t('sub2api.imagine.completed')" />
          <UButton
            class="detail-del"
            color="error"
            variant="ghost"
            size="sm"
            @click="deleteConfirmOpen = true"
          >
            {{ t('sub2api.imagine.deleteTask') }}
          </UButton>
        </div>
        <p class="detail-prompt">{{ store.current.prompt }}</p>
        <p v-if="store.current.errorMessage" class="gen-error">{{ store.current.errorMessage }}</p>

        <div v-if="store.current.images?.length" class="img-grid">
          <div v-for="(im, idx) in store.current.images" :key="im.id" class="img-card">
            <button type="button" class="img-open" @click="openPreview(idx)">
              <img :src="imageUrl(im.id)" :alt="im.filename || store.current.prompt" loading="lazy" />
            </button>
            <div class="img-overlay">
              <span class="img-tip">{{ t('sub2api.imagine.zoomImage') }}</span>
              <div class="img-acts">
                <button
                  type="button"
                  class="img-act"
                  :title="t('sub2api.imagine.modifyImage')"
                  @click="onModify(im.id)"
                >
                  <component :is="menuStore.iconComponents['HOutline:PencilSquareIcon']" />
                </button>
                <a
                  class="img-act"
                  :href="imageUrl(im.id)"
                  :download="im.filename"
                  :title="t('sub2api.imagine.download')"
                >
                  <component :is="menuStore.iconComponents['HOutline:ArrowDownTrayIcon']" />
                </a>
              </div>
            </div>
          </div>
        </div>
        <div v-else-if="store.current.status === 'pending'" class="img-grid">
          <div v-for="i in store.current.n" :key="i" class="img-skeleton" />
        </div>
        <EmptyState v-else :title="t('sub2api.imagine.noImage')" />
      </div>
      <div v-else class="loading-block is-inline" role="status">
        <span class="spin" aria-hidden="true" />
      </div>
    </FormDialog>

    <!-- 删除确认 -->
    <FormDialog
      v-model="deleteConfirmOpen"
      :title="t('sub2api.imagine.confirmDeleteTitle')"
      :width="420"
      :confirm-text="t('sub2api.imagine.confirmDeleteText')"
      :cancel-text="t('sub2api.common.cancel')"
      :loading="deleting"
      @confirm="confirmDelete"
    >
      <p class="confirm-text">{{ t('sub2api.imagine.confirmDeleteContent') }}</p>
    </FormDialog>

    <!-- 图片预览 lightbox -->
    <Teleport to="body">
      <Transition name="lightbox">
        <div
          v-if="previewOpen"
          class="lightbox"
          role="dialog"
          aria-modal="true"
          @click.self="previewOpen = false"
        >
          <button type="button" class="lightbox__close" :aria-label="t('system.common.close')" @click="previewOpen = false">
            <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
          </button>
          <button
            v-if="previewList.length > 1"
            type="button"
            class="lightbox__nav lightbox__nav--prev"
            @click="previewIndex = (previewIndex - 1 + previewList.length) % previewList.length"
          >
            <component :is="menuStore.iconComponents['HOutline:ChevronLeftIcon']" />
          </button>
          <img class="lightbox__img" :src="previewList[previewIndex]" alt="" />
          <button
            v-if="previewList.length > 1"
            type="button"
            class="lightbox__nav lightbox__nav--next"
            @click="previewIndex = (previewIndex + 1) % previewList.length"
          >
            <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" />
          </button>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { computeSize, useImagineStore, type Params } from '@/stores/sub2api/imagine'
import { useThemeStore } from '@/stores/theme'
import { imageGenImageUrl } from '@/api/sub2api-imagine'

defineOptions({ name: 'Sub2APIImageGen' })

const route = useRoute()
const store = useImagineStore()
const themeStore = useThemeStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const isDark = computed(() => themeStore.isDarkTheme)
const toggleTheme = () => themeStore.toggleThemeMode(isDark.value ? 'light' : 'dark')

const loading = ref(false)
const settingsOpen = ref(false)
const deleteConfirmOpen = ref(false)
const deleting = ref(false)
const title = 'Imagine'
const fileInput = ref<HTMLInputElement | null>(null)
const dragging = ref(false)

const SHAPE_VALUES = ['1:1', '4:3', '3:4', '16:9', '9:16', '3:2', '2:3', 'auto', 'custom'] as const
const SCALES = ['1K', '2K', '4K'] as const
const shapeLabel = (value: string) => {
  if (value === 'auto') return t('sub2api.imagine.auto')
  if (value === 'custom') return t('sub2api.imagine.custom')
  return value
}

const qualityOptions = [
  { label: 'auto', value: 'auto' },
  { label: 'low', value: 'low' },
  { label: 'medium', value: 'medium' },
  { label: 'high', value: 'high' },
]
const formatOptions = [
  { label: 'PNG', value: 'png' },
  { label: 'JPEG', value: 'jpeg' },
  { label: 'WebP', value: 'webp' },
]

const draftParams = ref<Params>({ ...store.params })

watch(settingsOpen, (open) => {
  if (open) draftParams.value = { ...store.params }
})

const imageUrl = (id: string) => imageGenImageUrl(id)

const apiKeyOptions = computed(() =>
  store.apiKeys.map((k) => ({ label: k.name || k.id, value: k.id })),
)
const modelOptions = computed(() =>
  store.models.map((m) => ({ label: m.displayName || m.id, value: m.id })),
)

// AppSelect 无 @change：用 v-model 代理触发 selectApiKey
const apiKeyIdModel = computed({
  get: () => store.apiKeyId,
  set: (id: string) => {
    if (id !== store.apiKeyId) void store.selectApiKey(id)
  },
})

const pickSource = () => fileInput.value?.click()
const onFilePicked = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) await store.setCustomSource(file)
  input.value = ''
}

const onDragOver = (e: DragEvent) => {
  if (e.dataTransfer?.types.includes('Files')) dragging.value = true
}
const onDragLeave = (e: DragEvent) => {
  if (!(e.currentTarget as HTMLElement).contains(e.relatedTarget as Node)) {
    dragging.value = false
  }
}
const onDrop = async (e: DragEvent) => {
  dragging.value = false
  const file = Array.from(e.dataTransfer?.files ?? []).find((f) => f.type.startsWith('image/'))
  if (file) await store.setCustomSource(file)
}

const detailOpen = computed({
  get: () => !!store.currentId,
  set: (v: boolean) => {
    if (!v) store.selectGeneration('')
  },
})

const canSubmit = computed(
  () =>
    !!store.prompt.trim() &&
    !!store.apiKeyId &&
    !!store.modelId &&
    (store.mode !== 'image2image' || !!store.sourceImage),
)

const status = computed(() => {
  if (!store.apiKeyId) return { tone: 'amber', text: t('sub2api.imagine.noApiKey') }
  if (!store.modelId) return { tone: 'amber', text: t('sub2api.imagine.noModel') }
  if (store.hasPending) return { tone: 'blue', text: t('sub2api.imagine.generating') }
  return { tone: 'green', text: t('sub2api.imagine.ready') }
})

const modeLabel = (m: string) =>
  m === 'image2image' ? t('sub2api.imagine.image2image') : t('sub2api.imagine.text2image')

const draftResolvedSize = computed(() => computeSize(draftParams.value))

const onSettingsConfirm = () => {
  store.params = { ...draftParams.value }
  settingsOpen.value = false
}

const detailTitle = computed(() => {
  const g = store.current
  if (!g) return t('sub2api.imagine.taskDetail')
  return `${modeLabel(g.mode)} · ${dayjs(g.createdAt).format('MM-DD HH:mm')}`
})

const onModify = async (imageId: string) => {
  if (await store.modifyImage(imageId)) detailOpen.value = false
}

const confirmDelete = async () => {
  if (!store.currentId) return
  deleting.value = true
  try {
    await store.removeGeneration(store.currentId)
    deleteConfirmOpen.value = false
    detailOpen.value = false
  } finally {
    deleting.value = false
  }
}

// lightbox
const previewOpen = ref(false)
const previewIndex = ref(0)
const previewList = computed(() =>
  (store.current?.images || []).map((im) => imageUrl(im.id)),
)
const openPreview = (idx: number) => {
  previewIndex.value = idx
  previewOpen.value = true
}

const autoResize = (e: Event) => {
  const el = e.target as HTMLTextAreaElement
  el.style.height = 'auto'
  el.style.height = `${Math.min(el.scrollHeight, 120)}px`
}

onMounted(async () => {
  if (!store.bootstrap(route.query)) return
  if (store.theme === 'dark' && !isDark.value) toggleTheme()
  loading.value = true
  try {
    await store.loadApiKeys()
    await store.loadModels()
    await store.loadGenerations()
    if (store.hasPending) store.startPolling()
  } finally {
    loading.value = false
  }
})

onUnmounted(() => store.dispose())
</script>

<style scoped lang="scss">
.s2a-imagine {
  position: relative;
  height: 100vh;
  max-width: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-page);
}

.bg-accent {
  position: absolute;
  inset: 0 0 auto;
  height: 320px;
  z-index: 0;
  pointer-events: none;
  opacity: 0.4;
  background:
    radial-gradient(
      ellipse 48% 40% at 20% 0%,
      color-mix(in srgb, var(--el-color-primary) 30%, transparent),
      transparent 62%
    ),
    radial-gradient(
      ellipse 44% 36% at 82% 6%,
      color-mix(in srgb, #22d3ee 26%, transparent),
      transparent 64%
    );
}
.is-dark .bg-accent {
  opacity: 0.18;
}

.topbar {
  flex: none;
  z-index: 20;
  backdrop-filter: blur(14px);
  background: color-mix(in srgb, var(--el-bg-color-page) 72%, transparent);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.topbar-inner {
  width: 100%;
  box-sizing: border-box;
  padding: 0 16px;
  min-height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: nowrap;
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  font-size: 15px;
  font-weight: 700;
  min-width: 0;
  flex: 0 1 auto;
}
.brand-title {
  white-space: nowrap;
}
.brand-host {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 160px;
}
.brand-dot {
  flex: none;
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--el-color-primary), #22d3ee);
}

.topbar-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 0 1 auto;
}
.sel-key {
  width: 160px;
  min-width: 0;
}
.sel-model {
  width: 180px;
  min-width: 0;
}

.status-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--chip);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  --chip: var(--el-color-primary);
  &.tone-green {
    --chip: #10b981;
  }
  &.tone-amber {
    --chip: #f59e0b;
  }
  &.tone-red {
    --chip: #ef4444;
  }
  &.tone-blue {
    --chip: #3b82f6;
  }
  .dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--chip);
    flex: none;
  }
}

.loading-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 64px 20px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  &.is-inline {
    padding: 40px;
  }
  p {
    margin: 0;
  }
}
.spin {
  width: 22px;
  height: 22px;
  border: 2px solid var(--el-border-color-light);
  border-top-color: var(--el-color-primary);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.tasks {
  position: relative;
  z-index: 1;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  width: 100%;
  box-sizing: border-box;
  padding: 16px;
}

.task-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 14px;
}

.task-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg, 16px);
  background: var(--el-bg-color-overlay);
  overflow: hidden;
  cursor: pointer;
  transition:
    border-color 0.15s,
    transform 0.1s;
  &:hover {
    border-color: var(--el-color-primary);
  }
  &:active {
    transform: scale(0.99);
  }
  &.active {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--el-color-primary) 22%, transparent);
  }
}

.task-thumb {
  position: relative;
  aspect-ratio: 1 / 1;
  background: var(--el-fill-color-lighter);
  display: flex;
  align-items: center;
  justify-content: center;
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  .task-skeleton {
    width: 60%;
    height: 60%;
    border-radius: 12px;
    background: linear-gradient(
      90deg,
      var(--el-fill-color-light) 25%,
      var(--el-fill-color) 37%,
      var(--el-fill-color-light) 63%
    );
    background-size: 400% 100%;
    animation: skeleton 1.4s ease infinite;
  }
  .task-thumb-empty {
    color: var(--el-text-color-placeholder);
    width: 28px;
    height: 28px;
    :deep(svg) {
      width: 100%;
      height: 100%;
    }
  }
  .task-mode {
    position: absolute;
    top: 8px;
    left: 8px;
    font-size: 10.5px;
    font-weight: 700;
    padding: 2px 8px;
    border-radius: 999px;
    background: color-mix(in srgb, var(--el-color-primary) 80%, transparent);
    color: #fff;
    &.image2image {
      background: color-mix(in srgb, #8b5cf6 80%, transparent);
    }
  }
}

.task-body {
  padding: 10px 12px;
  .task-prompt {
    margin: 0 0 8px;
    font-size: 12.5px;
    line-height: 1.5;
    color: var(--el-text-color-regular);
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    min-height: 2.5em;
  }
  .task-meta {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: var(--el-text-color-secondary);
    flex-wrap: wrap;
    .task-status {
      margin-left: auto;
    }
  }
}

.empty-hero {
  padding: 48px 20px;
}

.bar-wrap {
  position: relative;
  z-index: 15;
  flex: none;
  padding: 12px 16px;
  background: linear-gradient(transparent, var(--el-bg-color-page) 30%);
  pointer-events: none;
  > * {
    pointer-events: auto;
  }
}

.composer-bar {
  position: relative;
  max-width: 900px;
  margin: 0 auto;
  display: flex;
  align-items: flex-end;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg, 16px);
  background: var(--el-bg-color-overlay);
  box-shadow: var(--app-shadow-md, 0 4px 8px -4px rgba(16, 24, 40, 0.08));
  transition:
    border-color 0.15s,
    box-shadow 0.15s;
  &.is-dragging {
    border-color: var(--el-color-primary);
    border-style: dashed;
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--el-color-primary) 22%, transparent);
  }
}

.mode-seg {
  flex: none;
  display: inline-flex;
  padding: 2px;
  border-radius: 9px;
  background: var(--el-fill-color-light);
  gap: 2px;
}
.mode-seg__btn {
  height: 28px;
  padding: 0 10px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition:
    background 0.15s,
    color 0.15s,
    box-shadow 0.15s;
  &.is-active {
    background: var(--el-bg-color-overlay);
    color: var(--el-color-primary);
    box-shadow: 0 1px 2px rgba(16, 24, 40, 0.06);
  }
  &:focus-visible {
    outline: none;
    box-shadow: var(--app-focus-ring, 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 25%, transparent));
  }
}

/* 输入外壳 focus：textarea 本身 outline:none，环画在 composer-bar 上 */
.composer-bar:focus-within:not(.is-dragging) {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 18%, transparent);
}

.prompt-input {
  flex: 1;
  min-width: 0;
  max-height: 120px;
  resize: none;
  border: 0;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 14px;
  line-height: 1.45;
  padding: 6px 4px;
  &::placeholder {
    color: var(--el-text-color-placeholder);
  }
}

.bar-actions {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 2px;
}

.set-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 11px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-sm, 8px);
  background: var(--el-bg-color-overlay);
  color: var(--el-text-color-primary);
  cursor: pointer;
  transition:
    border-color 0.2s,
    color 0.2s;
  :deep(svg) {
    width: 16px;
    height: 16px;
  }
  &:hover {
    border-color: var(--el-color-primary);
    color: var(--el-color-primary);
  }
  .size-badge {
    font-size: 11px;
    font-weight: 700;
    color: var(--el-color-primary);
    font-variant-numeric: tabular-nums;
    line-height: 1;
  }
}

.send-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 0;
  border-radius: var(--app-radius-sm, 8px);
  background: var(--el-color-primary);
  color: #fff;
  cursor: pointer;
  transition:
    opacity 0.15s,
    transform 0.1s;
  :deep(svg) {
    width: 16px;
    height: 16px;
  }
  &:hover:not(:disabled) {
    opacity: 0.9;
  }
  &:active:not(:disabled) {
    transform: scale(0.95);
  }
  &:disabled {
    background: var(--el-fill-color);
    color: var(--el-text-color-placeholder);
    cursor: not-allowed;
  }
}

.src-file {
  display: none;
}
.src-add {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 1px dashed var(--el-border-color);
  border-radius: var(--app-radius-sm, 8px);
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition:
    border-color 0.15s,
    color 0.15s;
  :deep(svg) {
    width: 16px;
    height: 16px;
  }
  &:hover {
    border-color: var(--el-color-primary);
    color: var(--el-color-primary);
  }
}
.src-thumb {
  position: relative;
  flex: none;
  width: 34px;
  height: 34px;
  cursor: pointer;
  img {
    display: block;
    width: 34px;
    height: 34px;
    object-fit: cover;
    border-radius: var(--app-radius-sm, 8px);
    border: 1px solid var(--el-border-color-light);
  }
  .src-clear {
    position: absolute;
    top: -6px;
    right: -6px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    border: 0;
    border-radius: 50%;
    background: var(--el-color-danger);
    color: #fff;
    cursor: pointer;
    padding: 0;
    :deep(svg) {
      width: 12px;
      height: 12px;
    }
  }
}

.drop-hint {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--app-radius-lg, 16px);
  background: color-mix(in srgb, var(--el-color-primary) 12%, var(--el-bg-color-overlay));
  color: var(--el-color-primary);
  font-size: 13px;
  font-weight: 700;
  pointer-events: none;
}

/* settings */
.settings {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 4px 0 8px;
}
.set-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
  &.set-row--inline {
    flex-direction: row;
    align-items: center;
    flex-wrap: wrap;
    .set-label {
      width: 48px;
      flex: none;
    }
  }
}
.set-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.chip-group {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.chip {
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition:
    border-color 0.15s,
    color 0.15s,
    background 0.15s;
  &.is-active {
    border-color: var(--el-color-primary);
    color: var(--el-color-primary);
    background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
  }
}
.custom-size {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  .x {
    color: var(--el-text-color-placeholder);
  }
}
.size-num {
  width: 100px;
}
.set-summary {
  margin-top: 4px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 12.5px;
  color: var(--el-text-color-secondary);
  b {
    color: var(--el-color-primary);
    font-variant-numeric: tabular-nums;
  }
}

/* detail */
.detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.detail-head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  .detail-meta {
    font-size: 12.5px;
    color: var(--el-text-color-secondary);
  }
  .detail-del {
    margin-left: auto;
  }
}
.detail-prompt {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  color: var(--el-text-color-regular);
  padding: 10px 12px;
  background: var(--el-fill-color-lighter);
  border-radius: var(--app-radius-sm, 8px);
}
.gen-error {
  margin: 0;
  font-size: 12.5px;
  color: var(--el-color-danger);
}
.gen-mode {
  font-size: 11px;
  font-weight: 700;
  padding: 2px 9px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--el-color-primary) 14%, transparent);
  color: var(--el-color-primary);
  &.image2image {
    background: color-mix(in srgb, #8b5cf6 14%, transparent);
    color: #8b5cf6;
  }
}

.img-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 14px;
}
.img-card {
  position: relative;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  overflow: hidden;
  background: var(--el-fill-color-lighter);
  .img-open {
    display: block;
    width: 100%;
    padding: 0;
    border: 0;
    background: transparent;
    cursor: zoom-in;
    img {
      display: block;
      width: 100%;
      height: auto;
      max-height: 70vh;
      object-fit: contain;
    }
  }
  .img-overlay {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 7px 9px;
    background: linear-gradient(transparent, rgba(0, 0, 0, 0.42));
    color: #fff;
    pointer-events: none;
    .img-tip {
      font-size: 11px;
      opacity: 0.8;
    }
    .img-acts {
      display: inline-flex;
      gap: 4px;
      pointer-events: auto;
    }
    .img-act {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 26px;
      height: 26px;
      border: 0;
      border-radius: 7px;
      background: rgba(255, 255, 255, 0.16);
      color: #fff;
      cursor: pointer;
      text-decoration: none;
      transition: background 0.15s;
      :deep(svg) {
        width: 14px;
        height: 14px;
      }
      &:hover {
        background: rgba(255, 255, 255, 0.3);
      }
    }
  }
}

.img-skeleton {
  aspect-ratio: 1 / 1;
  border-radius: 12px;
  background: linear-gradient(
    90deg,
    var(--el-fill-color-light) 25%,
    var(--el-fill-color) 37%,
    var(--el-fill-color-light) 63%
  );
  background-size: 400% 100%;
  animation: skeleton 1.4s ease infinite;
}
@keyframes skeleton {
  0% {
    background-position: 100% 50%;
  }
  100% {
    background-position: 0 50%;
  }
}

.confirm-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.55;
  color: var(--el-text-color-regular);
}

/* lightbox */
.lightbox {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.78);
  padding: 24px;
}
.lightbox__img {
  max-width: min(96vw, 1200px);
  max-height: 90vh;
  object-fit: contain;
  border-radius: 8px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4);
}
.lightbox__close {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 40px;
  height: 40px;
  border: 0;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  :deep(svg) {
    width: 20px;
    height: 20px;
  }
  &:hover {
    background: rgba(255, 255, 255, 0.22);
  }
}
.lightbox__nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 40px;
  height: 40px;
  border: 0;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  :deep(svg) {
    width: 20px;
    height: 20px;
  }
  &:hover {
    background: rgba(255, 255, 255, 0.22);
  }
  &--prev {
    left: 16px;
  }
  &--next {
    right: 16px;
  }
}
.lightbox-enter-active,
.lightbox-leave-active {
  transition: opacity 0.18s ease;
}
.lightbox-enter-from,
.lightbox-leave-to {
  opacity: 0;
}

@media (max-width: 720px) {
  .topbar-inner {
    padding: 8px 12px;
    min-height: 52px;
    flex-wrap: wrap;
  }
  .topbar-actions {
    flex: 1 1 100%;
    flex-wrap: nowrap;
    overflow: hidden;
  }
  .brand {
    font-size: 14px;
    flex: 1 1 auto;
  }
  .brand-host,
  .status-chip {
    display: none;
  }
  .sel-key,
  .sel-model {
    flex: 1 1 0;
    width: auto;
  }
  .tasks {
    padding: 12px;
  }
  .task-grid {
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 10px;
  }
  .composer-bar {
    flex-wrap: wrap;
    .mode-seg {
      order: -1;
      width: 100%;
      .mode-seg__btn {
        flex: 1;
        min-height: 36px;
        height: 36px;
      }
    }
    .prompt-input {
      order: 0;
      width: 100%;
      flex: 1 1 100%;
    }
    .src-add,
    .src-thumb {
      order: 1;
    }
    .bar-actions {
      order: 2;
      margin-left: auto;
    }
  }
}
</style>
