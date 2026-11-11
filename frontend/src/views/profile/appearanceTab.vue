<!-- 个人中心 · 外观（ThemePC / ThemeMobile / ThemeImport* / ThemeExportPC）
     PC：左栏 主题 / 背景 / 深浅色与细节，右栏吸顶「当前主题」；手机：当前主题卡片 → 横向主题 → 背景 → 深浅色 → 细节。
     导入主题包（也可以把 .momoko-theme 直接拖到页面上）、导出主题包、另存为都在这里打开。 -->
<template>
  <div class="appr" @dragover="onDragOver" @drop="onDrop">
    <AppearancePC v-if="!isMobile" />
    <AppearanceMobile v-else />

    <ThemeImportDialog v-model="appearanceUi.importOpen" :initial-file="appearanceUi.importFile" />
    <ThemeExportDialog
      v-if="appearanceUi.exportTarget"
      v-model="appearanceUi.exportOpen"
      :name="appearanceUi.exportTarget.name"
      :content="appearanceUi.exportTarget.content"
      :theme-id="appearanceUi.exportTarget.themeId"
    />
    <ThemeModal v-model="appearanceUi.saveAsOpen" :title="t('theme.saveAsTitle')" :width="440">
      <label class="appr-field">
        <span>{{ t('theme.saveAsName') }}</span>
        <input v-model="saveAsName" class="app-input" type="text" maxlength="40" @keydown.enter="saveAs" />
      </label>
      <template #footer>
        <span class="appr-spacer" />
        <UButton color="neutral" variant="soft" @click="appearanceUi.saveAsOpen = false">{{ t('common.cancel') }}</UButton>
        <UButton color="primary" :loading="saving" :disabled="!saveAsName.trim()" @click="saveAs">{{ t('theme.save') }}</UButton>
      </template>
    </ThemeModal>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ThemeImportDialog from '@/components/theme/ThemeImportDialog.vue'
import ThemeExportDialog from '@/components/theme/ThemeExportDialog.vue'
import ThemeModal from '@/components/theme/ThemeModal.vue'
import AppearancePC from './appearance/AppearancePC.vue'
import AppearanceMobile from './appearance/AppearanceMobile.vue'
import { appearanceUi, openThemeImport } from './appearance/useAppearancePage'

defineOptions({ name: 'ProfileAppearanceTab' })

const { t } = useI18n()
const menuStore = useMenuStore()
const themeStore = useThemeStore()
const isMobile = computed(() => menuStore.isMobile)

onMounted(() => {
  themeStore.loadThemes().catch(() => undefined)
})

// 把 .momoko-theme 直接拖到页面上就打开导入
const isThemeFile = (f?: File | null) => !!f && /\.momoko-theme$/i.test(f.name)
const onDragOver = (e: DragEvent) => {
  if (e.dataTransfer?.types?.includes('Files')) e.preventDefault()
}
const onDrop = (e: DragEvent) => {
  const f = e.dataTransfer?.files?.[0]
  if (!isThemeFile(f)) return
  e.preventDefault()
  openThemeImport(f)
}

// 另存为
const saveAsName = ref('')
const saving = ref(false)
watch(
  () => appearanceUi.saveAsOpen,
  (open) => {
    if (!open) return
    const base = themeStore.activeName || t(themeStore.currentPalette.labelKey)
    saveAsName.value = t('theme.saveAsDefault', { name: base }).slice(0, 40)
  },
)
const saveAs = async () => {
  const name = saveAsName.value.trim()
  if (!name || saving.value) return
  saving.value = true
  try {
    const th = await themeStore.saveCurrentAs(name)
    if (th) feedback.success(t('theme.savedAs', { name: th.name }))
    appearanceUi.saveAsOpen = false
  } finally {
    saving.value = false
  }
}
</script>

<style scoped lang="scss">
.appr {
  min-width: 0;
}
.appr-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  span {
    font-size: 0.8125rem;
    font-weight: 600;
  }
}
.appr-spacer {
  flex: 1;
}
</style>
