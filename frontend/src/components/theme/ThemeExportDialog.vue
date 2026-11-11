<!-- 导出主题包（ThemeExportPC / ThemeMobile 底部面板）：配色必含；背景图片、页面单独设置、细节偏好可选；
     文件名、预计大小；「复制主题代码」复制浅色 / 深色两套 CSS 变量。 -->
<template>
  <ThemeModal
    :model-value="modelValue"
    :title="t('theme.export.title')"
    :subtitle="isMobile ? '' : name"
    :width="520"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="tex">
      <label v-for="o in opts" :key="o.key" class="tex-row" :class="{ 'is-locked': o.lock }">
        <span class="tex-row__text">
          <b>{{ o.title }}</b>
          <small>{{ o.sub }}</small>
        </span>
        <AppSwitch :model-value="o.on" :disabled="o.lock" :aria-label="o.title" @update:model-value="setOpt(o.key, $event)" />
      </label>
      <template v-if="!isMobile">
        <label class="tex-field">
          <span class="tex-lbl">{{ t('theme.export.fileName') }}</span>
          <span class="tex-inp"><input v-model="fileName" class="m-mono" type="text" :aria-label="t('theme.export.fileName')" /></span>
        </label>
        <p class="tex-hint">{{ t('theme.export.hint', { size: sizeText }) }}</p>
      </template>
      <p v-else class="tex-hint">{{ t('theme.export.hintMobile', { file: fileName, size: sizeText }) }}</p>
    </div>
    <template #footer>
      <button v-if="!isMobile" type="button" class="tex-btn" @click="copyCode">
        <DocumentDuplicateIcon aria-hidden="true" />{{ t('theme.export.copyCode') }}
      </button>
      <span v-if="!isMobile" class="tex-spacer" />
      <button type="button" class="tex-btn" :class="{ 'is-grow': isMobile }" @click="emit('update:modelValue', false)">{{ t('common.cancel') }}</button>
      <button type="button" class="tex-btn is-primary" :class="{ 'is-grow2': isMobile }" :disabled="busy" @click="doExport">
        <ArrowDownTrayIcon aria-hidden="true" />{{ t('theme.export.export') }}
      </button>
    </template>
  </ThemeModal>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowDownTrayIcon, DocumentDuplicateIcon } from '@heroicons/vue/24/outline'
import type { ThemeContent } from '@/types/v1/theme'
import { downloadBase64, exportThemePackage } from '@/api/theme'
import { normalizeContent, pagesWithOverrides, themeCssCode } from '@/theme/content'
import { useThemeAssets } from '@/theme/useThemeAssets'
import { useThemePages } from '@/theme/pages'
import { copyTextToClipboard, formatFileSize } from '@/utils/file'
import ThemeModal from './ThemeModal.vue'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    name: string
    content: ThemeContent
    /** 已保存的主题 id（内置 / 未保存的外观为空） */
    themeId?: string
    /** 只导出这个页面的单独设置 */
    onlyPage?: string
  }>(),
  { themeId: '', onlyPage: '' },
)
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()
const menuStore = useMenuStore()
const isMobile = computed(() => menuStore.isMobile)
const { labelOf } = useThemePages()
const { findAsset } = useThemeAssets()

const c = computed(() => normalizeContent(props.content))
const pages = computed(() => pagesWithOverrides(c.value))
const imageRefs = computed(() => {
  const refs = new Set<string>()
  const bgs = [c.value.background, ...Object.values(c.value.pages).map((p) => p.background)]
  for (const b of bgs) if (b?.type === 'image' && b.image) refs.add(b.image)
  const banners = [c.value.banner, ...Object.values(c.value.pages).map((p) => p.banner)]
  for (const b of banners) if (b?.type === 'image' && b.image) refs.add(b.image)
  return [...refs]
})
const imageBytes = computed(() => imageRefs.value.reduce((s, id) => s + Number(findAsset(id)?.size || 0), 0))

const includeImages = ref(true)
const includePages = ref(true)
const includeShape = ref(false)
const fileName = ref('')
const busy = ref(false)

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    includeImages.value = imageRefs.value.length > 0
    includePages.value = pages.value.length > 0
    includeShape.value = false
    const base = (props.name || 'theme').split(/\s+/)[0] || 'theme'
    fileName.value = `${base}${props.onlyPage ? `-${labelOf(props.onlyPage)}` : ''}.momoko-theme`
  },
  { immediate: true },
)

const firstImage = computed(() => (imageRefs.value[0] ? findAsset(imageRefs.value[0]) : null))
const opts = computed(() => [
  { key: 'tokens', title: t('theme.export.optTokens'), sub: t('theme.export.required'), on: true, lock: true },
  {
    key: 'images',
    title: t('theme.export.optImages'),
    sub: imageRefs.value.length
      ? [firstImage.value?.name, imageBytes.value ? formatFileSize(imageBytes.value) : ''].filter(Boolean).join(' · ') ||
        t('theme.import.tagImages', { n: imageRefs.value.length })
      : t('theme.export.notImage'),
    on: includeImages.value && imageRefs.value.length > 0,
    lock: imageRefs.value.length === 0,
  },
  {
    key: 'pages',
    title: t('theme.export.optPages'),
    sub: pages.value.length ? t('theme.export.pagesN', { n: pages.value.length }) : t('theme.export.noPages'),
    on: includePages.value && pages.value.length > 0,
    lock: pages.value.length === 0 || !!props.onlyPage,
  },
  { key: 'shape', title: t('theme.export.optShape'), sub: t('theme.export.optShapeSub'), on: includeShape.value, lock: false },
])
const setOpt = (key: string, v: boolean) => {
  if (key === 'images') includeImages.value = v
  if (key === 'pages') includePages.value = v
  if (key === 'shape') includeShape.value = v
}

const sizeText = computed(() =>
  t('theme.export.about', { size: formatFileSize((includeImages.value ? imageBytes.value : 0) + 3 * 1024) }),
)

const copyCode = async () => {
  await copyTextToClipboard(themeCssCode(props.name, c.value))
  feedback.success(t('theme.export.copied'))
}

const doExport = async () => {
  busy.value = true
  try {
    const { data } = await exportThemePackage({
      themeId: props.themeId,
      content: c.value,
      name: props.name,
      description: '',
      includeImages: includeImages.value && imageRefs.value.length > 0,
      includePages: includePages.value && pages.value.length > 0,
      includeShape: includeShape.value,
      onlyPage: props.onlyPage,
      fileName: fileName.value.trim(),
    })
    if (!data?.data) return
    downloadBase64(data.data, data.fileName || fileName.value)
    feedback.success(t('theme.export.done', { file: data.fileName || fileName.value }))
    emit('update:modelValue', false)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped lang="scss">
.tex {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.tex-row {
  display: flex;
  align-items: center;
  gap: 16px;
  min-height: 56px;
  padding: 8px 0;
  border-top: 1px solid var(--m-line);
  cursor: pointer;
  &:first-child {
    border-top: 0;
  }
  &.is-locked {
    cursor: default;
  }
}
.tex-row__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  b {
    font-size: 0.875rem;
  }
  small {
    font-size: 0.75rem;
    color: var(--m-ink2);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
.tex-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 8px;
}
.tex-lbl {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--m-ink2);
}
.tex-inp {
  display: flex;
  align-items: center;
  min-height: 44px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--m-s);
  border: 1.5px solid var(--m-line);
  input {
    flex: 1;
    min-width: 0;
    border: 0;
    background: transparent;
    font-size: 0.875rem;
    color: var(--m-ink);
    outline: none;
  }
  &:focus-within {
    border-color: var(--m-p);
  }
}
.tex-hint {
  margin: 8px 0 0;
  font-size: 0.75rem;
  line-height: 1.6;
  color: var(--m-ink2);
}
.tex-spacer {
  flex: 1;
}
.tex-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 18px;
  border: 0;
  border-radius: var(--app-radius-btn);
  background: var(--m-s2);
  color: var(--m-ink);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  svg {
    width: 16px;
    height: 16px;
  }
  &:hover:not(:disabled) {
    background: var(--m-pSoft);
    color: var(--m-pInk);
  }
  &.is-primary {
    background: var(--m-p);
    color: var(--m-pOn);
  }
  &.is-grow {
    flex: 1;
  }
  &.is-grow2 {
    flex: 2;
  }
  &:disabled {
    opacity: 0.55;
    cursor: progress;
  }
}
</style>
