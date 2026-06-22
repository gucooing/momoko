<template>
  <main class="s2a-imagine" :class="{ 'is-dark': isDark }" v-loading="loading">
    <div class="bg-accent" aria-hidden />

    <!-- 顶栏 -->
    <header class="topbar">
      <div class="topbar-inner">
        <div class="brand">
          <span class="brand-dot" />
          {{ title }}
          <span v-if="store.srcHost" class="brand-host" :title="store.srcHost">{{ store.srcHost }}</span>
          <span class="status-chip" :class="`tone-${status.tone}`">
            <span class="dot" />{{ status.text }}
          </span>
        </div>
        <div class="topbar-actions">
          <el-select v-model="store.apiKeyId" class="sel-key" placeholder="选择 ApiKey" @change="store.selectApiKey">
            <el-option v-for="k in store.apiKeys" :key="k.id" :value="k.id" :label="k.name || k.id">
              <span class="opt-name">{{ k.name || k.id }}</span>
            </el-option>
          </el-select>
          <el-select
            v-model="store.modelId"
            class="sel-model"
            placeholder="选择模型"
            :disabled="!store.models.length"
          >
            <el-option v-for="m in store.models" :key="m.id" :value="m.id" :label="m.displayName || m.id">
              <span class="opt-name">{{ m.displayName || m.id }}</span>
            </el-option>
          </el-select>
          <button class="icon-btn" type="button" :title="isDark ? '浅色' : '深色'" @click="toggleTheme">
            <el-icon><component :is="isDark ? Sunny : Moon" /></el-icon>
          </button>
        </div>
      </div>
    </header>

    <!-- 任务列表 -->
    <section class="tasks">
      <div v-if="store.generations.length" class="task-grid">
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
              <el-icon><Picture /></el-icon>
            </div>
            <span class="task-mode" :class="g.mode">{{ modeLabel(g.mode) }}</span>
          </div>
          <div class="task-body">
            <p class="task-prompt">{{ g.prompt }}</p>
            <div class="task-meta">
              <span>{{ g.size }}</span>
              <span>·</span>
              <span>{{ g.n }}张</span>
              <span
                v-if="g.status === 'pending'"
                class="status-chip sm tone-blue"
              ><span class="dot" />生成中</span>
              <span v-else-if="g.status === 'failed'" class="status-chip sm tone-red">失败</span>
              <span v-else class="status-chip sm tone-green">{{ g.resultCount }}张</span>
            </div>
          </div>
        </article>
      </div>
      <div v-else class="empty-hero">
        <h3><span class="grad">开始你的第一张作品</span></h3>
        <p>在下方输入提示词，选择形状与清晰度后点“生成”。</p>
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
        <el-radio-group v-model="store.mode" size="small" class="mode-toggle">
          <el-radio-button value="text2image">文生图</el-radio-button>
          <el-radio-button value="image2image">图生图</el-radio-button>
        </el-radio-group>
        <!-- 图生图源图：紧挨模式切换，支持点击上传 / 拖入图片 -->
        <template v-if="store.mode === 'image2image'">
          <div v-if="store.sourceImage" class="src-thumb" title="点击更换源图" @click="pickSource">
            <img :src="store.sourceImage" alt="源图" />
            <button class="src-clear" type="button" title="移除源图" @click.stop="store.clearSource">
              <el-icon><Close /></el-icon>
            </button>
          </div>
          <button v-else class="src-add" type="button" title="上传源图（可拖入）" @click="pickSource">
            <el-icon><Upload /></el-icon>
          </button>
        </template>
        <el-input
          v-model="store.prompt"
          type="textarea"
          :autosize="{ minRows: 1, maxRows: 5 }"
          :placeholder="store.mode === 'image2image' ? '描述如何修改源图，或拖入图片…' : '描述你想生成的图片…'"
          resize="none"
          class="prompt-input"
          @keydown.ctrl.enter="store.submitGeneration"
        />
        <div class="bar-actions">
          <button class="set-btn" type="button" title="参数设置" @click="settingsOpen = true">
            <el-icon><Setting /></el-icon>
            <span class="size-badge">{{ store.resolveSize() }}</span>
          </button>
          <button
            class="send-btn"
            type="button"
            :disabled="!canSubmit"
            title="生成 (Ctrl+Enter)"
            @click="store.submitGeneration"
          >
            <el-icon><Promotion /></el-icon>
          </button>
        </div>
        <div v-if="dragging" class="drop-hint">松开以作为图生图源图</div>
        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          class="src-file"
          @change="onFilePicked"
        />
      </div>
    </div>

    <!-- 参数设置弹窗 -->
    <BaseDialog
      v-model="settingsOpen"
      title="生图参数"
      width="480px"
      confirm-text="应用"
      cancel-text="取消"
      @close="onSettingsClose"
      @confirm="onSettingsConfirm"
    >
      <div class="settings">
        <div class="set-row">
          <label>形状</label>
          <el-radio-group v-model="draftParams.shape" size="small">
            <el-radio-button value="1:1">1:1</el-radio-button>
            <el-radio-button value="4:3">4:3</el-radio-button>
            <el-radio-button value="3:4">3:4</el-radio-button>
            <el-radio-button value="16:9">16:9</el-radio-button>
            <el-radio-button value="9:16">9:16</el-radio-button>
            <el-radio-button value="3:2">3:2</el-radio-button>
            <el-radio-button value="2:3">2:3</el-radio-button>
            <el-radio-button value="auto">自动</el-radio-button>
            <el-radio-button value="custom">自定义</el-radio-button>
          </el-radio-group>
        </div>
        <div v-if="draftParams.shape !== 'auto' && draftParams.shape !== 'custom'" class="set-row">
          <label>清晰度</label>
          <el-radio-group v-model="draftParams.scale" size="small">
            <el-radio-button value="1K">1K</el-radio-button>
            <el-radio-button value="2K">2K</el-radio-button>
            <el-radio-button value="4K">4K</el-radio-button>
          </el-radio-group>
        </div>
        <div v-if="draftParams.shape === 'custom'" class="set-row">
          <label>尺寸</label>
          <div class="custom-size">
            <el-input-number v-model="draftParams.customW" :min="256" :max="4096" :step="64" size="small" controls-position="right" />
            <span class="x">×</span>
            <el-input-number v-model="draftParams.customH" :min="256" :max="4096" :step="64" size="small" controls-position="right" />
          </div>
        </div>
        <div class="set-row">
          <label>张数</label>
          <el-input-number v-model="draftParams.n" :min="1" :max="4" size="small" controls-position="right" />
        </div>
        <div class="set-row">
          <label>质量</label>
          <el-select v-model="draftParams.quality" size="small" class="sel-sm">
            <el-option value="auto" label="auto" />
            <el-option value="low" label="low" />
            <el-option value="medium" label="medium" />
            <el-option value="high" label="high" />
          </el-select>
        </div>
        <div class="set-row">
          <label>格式</label>
          <el-select v-model="draftParams.outputFormat" size="small" class="sel-sm">
            <el-option value="png" label="PNG" />
            <el-option value="jpeg" label="JPEG" />
            <el-option value="webp" label="WebP" />
          </el-select>
        </div>
        <div class="set-summary">
          解析分辨率：<b>{{ draftResolvedSize }}</b>
        </div>
      </div>
    </BaseDialog>

    <!-- 任务详情弹窗 -->
    <BaseDialog
      v-model="detailOpen"
      :title="detailTitle"
      width="780px"
      :show-footer="false"
      fullscreen-button
    >
      <div v-if="store.current" class="detail">
        <div class="detail-head">
          <span class="gen-mode" :class="store.current.mode">{{ modeLabel(store.current.mode) }}</span>
          <span class="detail-meta">{{ store.current.model }}</span>
          <span class="detail-meta">{{ store.current.size }} · {{ store.current.n }}张</span>
          <span v-if="store.current.status === 'pending'" class="status-chip tone-blue"
            ><span class="dot" />生成中</span
          >
          <span v-else-if="store.current.status === 'failed'" class="status-chip tone-red">失败</span>
          <span v-else class="status-chip tone-green">完成</span>
          <el-button
            class="detail-del"
            text
            type="danger"
            size="small"
            @click="deleteCurrent"
          >删除</el-button>
        </div>
        <p class="detail-prompt">{{ store.current.prompt }}</p>
        <p v-if="store.current.errorMessage" class="gen-error">{{ store.current.errorMessage }}</p>

        <div v-if="store.current.images?.length" class="img-grid">
          <div v-for="(im, idx) in store.current.images" :key="im.id" class="img-card">
            <el-image
              :src="imageUrl(im.id)"
              :preview-src-list="store.current.images.map((x) => imageUrl(x.id))"
              :initial-index="idx"
              fit="contain"
              preview-teleported
              hide-on-click-modal
              class="img-el"
            />
            <div class="img-overlay">
              <span class="img-tip">点击放大</span>
              <div class="img-acts">
                <button class="img-act" title="修改（图生图）" @click="onModify(im.id)">
                  <el-icon><EditPen /></el-icon>
                </button>
                <a class="img-act" :href="imageUrl(im.id)" :download="im.filename" title="下载">
                  <el-icon><Download /></el-icon>
                </a>
              </div>
            </div>
          </div>
        </div>
        <div v-else-if="store.current.status === 'pending'" class="img-grid">
          <div v-for="i in store.current.n" :key="i" class="img-skeleton" />
        </div>
        <div v-else class="empty">无图片</div>
      </div>
    </BaseDialog>
  </main>
</template>

<script setup lang="ts">
import { Close, Download, EditPen, Moon, Picture, Promotion, Setting, Sunny, Upload } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { computeSize, useImagineStore } from '@/stores/sub2api/imagine'
import { useThemeStore } from '@/stores/theme'
import { imageGenImageUrl } from '@/api/sub2api-imagine'
import { Dialog } from '@/utils/dialog'

defineOptions({ name: 'Sub2APIImageGen' })

const route = useRoute()
const store = useImagineStore()
const themeStore = useThemeStore()
const isDark = computed(() => themeStore.isDarkTheme)
const toggleTheme = () => themeStore.toggleThemeMode(isDark.value ? 'light' : 'dark')

const loading = ref(false)
const settingsOpen = ref(false)
const title = 'Imagine'
const fileInput = ref<HTMLInputElement | null>(null)
const dragging = ref(false)

// 参数设置草稿：编辑时暂存，点“应用”才写回 store；点“取消”丢弃。
const draftParams = ref({ ...store.params })

// 监听弹窗打开，重置草稿为当前 store 值
watch(settingsOpen, (open) => {
  if (open) draftParams.value = { ...store.params }
})

const imageUrl = (id: string) => imageGenImageUrl(id)

// 选择本地图片作为图生图源图
const pickSource = () => fileInput.value?.click()
const onFilePicked = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) await store.setCustomSource(file)
  input.value = '' // 允许重复选择同一文件
}

// 拖入图片作为源图：dragenter/over 时高亮，drop 时取第一张图片（自动切到图生图）
const onDragOver = (e: DragEvent) => {
  if (e.dataTransfer?.types.includes('Files')) dragging.value = true
}
const onDragLeave = (e: DragEvent) => {
  // 仅当离开 composer-bar 边界时取消高亮，避免子元素间移动误触发
  if (!(e.currentTarget as HTMLElement).contains(e.relatedTarget as Node)) {
    dragging.value = false
  }
}
const onDrop = async (e: DragEvent) => {
  dragging.value = false
  const file = Array.from(e.dataTransfer?.files ?? []).find((f) => f.type.startsWith('image/'))
  if (file) await store.setCustomSource(file)
}

// 详情弹窗开关：跟随 currentId
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
  if (!store.apiKeyId) return { tone: 'amber', text: '未选 ApiKey' }
  if (!store.modelId) return { tone: 'amber', text: '未选模型' }
  if (store.hasPending) return { tone: 'blue', text: '生成中' }
  return { tone: 'green', text: '就绪' }
})

const modeLabel = (m: string) => (m === 'image2image' ? '图生图' : '文生图')

// 草稿解析分辨率（实时预览），与 store 共用 computeSize 实现
const draftResolvedSize = computed(() => computeSize(draftParams.value))

// 应用本次设置
const onSettingsConfirm = async () => {
  store.params = { ...draftParams.value }
  settingsOpen.value = false
}

// 取消：丢弃草稿（关闭即丢弃，无需额外操作）
const onSettingsClose = () => {
  // 草稿会在下次打开时重置
}

const detailTitle = computed(() => {
  const g = store.current
  if (!g) return '任务详情'
  return `${modeLabel(g.mode)} · ${dayjs(g.createdAt).format('MM-DD HH:mm')}`
})

const onModify = async (imageId: string) => {
  if (await store.modifyImage(imageId)) detailOpen.value = false
}

const deleteCurrent = async () => {
  if (!store.currentId) return
  try {
    await Dialog.confirm({
      title: '确认删除',
      content: '删除该任务及其图片？此操作不可撤销。',
      confirmText: '确认删除',
      cancelText: '取消',
    })
  } catch {
    return
  }
  await store.removeGeneration(store.currentId)
  detailOpen.value = false
}

onMounted(async () => {
  if (!store.bootstrap(route.query)) return
  if (store.theme === 'dark' && !isDark.value) toggleTheme()
  loading.value = true
  try {
    await store.loadApiKeys()
    await store.loadModels()
    await store.loadGenerations()
    // 刷新页面后若仍有生成中任务，自动恢复轮询
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

/* topbar：固定不随滚动 */
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
  flex-wrap: wrap;
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  font-size: 15px;
  font-weight: 800;
  min-width: 0;
}

.brand-host {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}

.brand-dot {
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--el-color-primary), #22d3ee);
}

.topbar-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  :deep(.el-input__wrapper) {
    border-radius: 10px;
  }
}

.sel-key {
  width: 160px;
}
.sel-model {
  width: 180px;
}

.opt-name {
  margin-right: 8px;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 9px;
  background: var(--el-bg-color-overlay);
  color: var(--el-text-color-primary);
  font-size: 16px;
  cursor: pointer;
  transition:
    border-color 0.2s,
    color 0.2s;
  &:hover {
    border-color: var(--el-color-primary);
    color: var(--el-color-primary);
  }
}

/* 参数设置按钮：自宽，图标 + 分辨率徽标并排 */
.set-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 11px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 9px;
  background: var(--el-bg-color-overlay);
  color: var(--el-text-color-primary);
  font-size: 16px;
  cursor: pointer;
  transition:
    border-color 0.2s,
    color 0.2s;
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

/* status：纯文字+颜色，无胶囊框、无大圆 */
.status-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--chip);
  font-size: 12px;
  font-weight: 700;
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
  &.sm {
    font-size: 11px;
  }
  .dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--chip);
    flex: none;
  }
}

/* tasks：唯一可滚动区 */
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
  border-radius: 14px;
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
    font-size: 28px;
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
    .status-chip {
      margin-left: auto;
    }
  }
}

.empty-hero {
  text-align: center;
  padding: 60px 20px;
  h3 {
    margin: 0 0 8px;
    font-size: 22px;
    font-weight: 800;
    .grad {
      background: linear-gradient(90deg, var(--el-color-primary), #22d3ee 60%, #8b5cf6);
      background-clip: text;
      -webkit-background-clip: text;
      color: transparent;
    }
  }
  p {
    margin: 0;
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }
}

/* composer bar：固定在底部，不随任务列表滚动 */
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
  border-radius: 14px;
  background: var(--el-bg-color-overlay);
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.08);
  transition: border-color 0.15s, box-shadow 0.15s;
  &.is-dragging {
    border-color: var(--el-color-primary);
    border-style: dashed;
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--el-color-primary) 22%, transparent);
  }
  .mode-toggle {
    flex: none;
  }
  .prompt-input {
    flex: 1;
    min-width: 0;
    :deep(.el-textarea__inner) {
      box-shadow: none;
      background: transparent;
      padding: 6px 4px;
      font-size: 14px;
      border-radius: 10px;
    }
  }
  .bar-actions {
    flex: none;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding-bottom: 2px;
  }
  .send-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 34px;
    height: 34px;
    border: 0;
    border-radius: 9px;
    background: var(--el-color-primary);
    color: #fff;
    font-size: 16px;
    cursor: pointer;
    transition: opacity 0.15s, transform 0.1s;
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
}

.src-file {
  display: none;
}

/* 源图控件：紧挨模式切换，与发送按钮等高 */
.src-add {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 1px dashed var(--el-border-color);
  border-radius: 9px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 16px;
  cursor: pointer;
  transition:
    border-color 0.15s,
    color 0.15s;
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
    border-radius: 9px;
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
    font-size: 11px;
    cursor: pointer;
  }
}

/* 拖拽提示：覆盖在输入栏上 */
.drop-hint {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  background: color-mix(in srgb, var(--el-color-primary) 12%, var(--el-bg-color-overlay));
  color: var(--el-color-primary);
  font-size: 13px;
  font-weight: 700;
  pointer-events: none;
}

/* settings dialog */
.settings {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 4px 0 8px;
  /* 统一 Element 输入控件圆角，避免 4px 硬角与柔和面板不协调 */
  :deep(.el-input__wrapper),
  :deep(.el-select .el-input__wrapper),
  :deep(.el-input-number .el-input__wrapper) {
    border-radius: 10px;
  }
  :deep(.el-input-number) {
    border-radius: 10px;
  }
}
.set-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  label {
    flex: none;
    width: 48px;
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }
  .custom-size {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    .x {
      color: var(--el-text-color-placeholder);
    }
  }
}
.sel-sm {
  width: 120px;
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

/* detail dialog */
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
  border-radius: 10px;
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

/* image grid (detail) */
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
  .img-el {
    display: block;
    width: 100%;
    height: auto;
    max-height: 70vh;
    cursor: zoom-in;
    :deep(img) {
      display: block;
      width: 100%;
      height: auto;
      object-fit: contain;
    }
    :deep(.el-image__wrapper),
    :deep(.el-image__inner) {
      width: 100%;
      height: auto;
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
    .img-tip {
      font-size: 11px;
      opacity: 0.8;
    }
    .img-acts {
      display: inline-flex;
      gap: 4px;
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
      font-size: 14px;
      cursor: pointer;
      transition: background 0.15s;
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

.empty {
  padding: 24px;
  text-align: center;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

@media (max-width: 720px) {
  .topbar-inner {
    padding: 8px 12px;
    height: auto;
    min-height: 52px;
    flex-wrap: wrap;
  }
  .topbar-actions {
    flex: 1 1 100%;
    flex-wrap: wrap;
  }
  .brand {
    font-size: 14px;
    flex: 1 1 auto;
    min-width: 0;
  }
  .sel-key,
  .sel-model {
    flex: 1 1 120px;
    width: auto;
    min-width: 0;
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
    .mode-toggle {
      order: -1;
      width: 100%;
    }
    .prompt-input {
      order: 0;
      width: 100%;
      flex: 1 1 100%;
    }
    /* 源图控件与操作按钮同处末行，避免单独占一行卡在模式切换与输入框之间 */
    .src-add,
    .src-thumb {
      order: 1;
    }
    .bar-actions {
      order: 2;
      margin-left: auto;
    }
  }
  .set-row label {
    width: auto;
  }
}
</style>
