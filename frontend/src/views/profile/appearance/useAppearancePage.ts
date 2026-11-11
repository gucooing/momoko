// 外观页（ThemePC / ThemeMobile）的共用逻辑：主题列表、卡片操作、当前主题信息与对比度检查、弹窗开关。
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import type { Theme, ThemeBackground, ThemeContent } from '@/types/v1/theme'
import { PALETTE_OPTIONS, isPaletteKey, type PaletteKey } from '@/theme/palettes'
import {
  GLASS_TYPES,
  assetUrl,
  backdropBrightColor,
  brightestStop,
  contrastChecks,
  dimForContrast,
  effectiveDim,
  emptyContent,
  normalizeContent,
  pagesWithOverrides,
  type BgType,
} from '@/theme/content'
import { imageStats } from '@/theme/imageStats'
import { formatRatio } from '@/theme/color'
import { useThemeAssets } from '@/theme/useThemeAssets'
import { Dialog } from '@/utils/dialog'
import { feedback } from '@/utils/feedback'
import { useThemeStore } from '@/stores/theme'
import { useMenuStore } from '@/stores/menu'

export interface ExportTarget {
  name: string
  content: ThemeContent
  themeId: string
}

/** 外观页的弹窗状态（手机页头的「导入」按钮在 profile/index.vue 里，也要能打开） */
export const appearanceUi = reactive({
  importOpen: false,
  importFile: null as File | null,
  exportOpen: false,
  exportTarget: null as ExportTarget | null,
  saveAsOpen: false,
})

export const openThemeImport = (file?: File | null) => {
  appearanceUi.importFile = file ?? null
  appearanceUi.importOpen = true
}

export const SITE_THEME_PERMISSION = 'system_config:edit'

export const useAppearancePage = () => {
  const { t } = useI18n()
  const router = useRouter()
  const themeStore = useThemeStore()
  const menuStore = useMenuStore()
  const { findAsset } = useThemeAssets()

  const canSetSite = computed(() => menuStore.hasButtonPermission(SITE_THEME_PERMISSION))
  const scheme = computed(() => themeStore.scheme)

  const loadThemes = async () => {
    try {
      await themeStore.loadThemes()
    } catch {
      /* 列表拉不到时只显示内置主题 */
    }
  }

  const paletteName = (key: string) => t(`theme.palettes.${isPaletteKey(key) ? key : 'sakura'}`)

  const builtins = computed(() =>
    PALETTE_OPTIONS.map((p) => ({
      key: p.key,
      name: t(p.labelKey),
      en: p.en,
      content: emptyContent(p.key),
      on: !themeStore.usingSite && themeStore.currentTheme === p.key,
    })),
  )

  const srcText = (th: Theme) =>
    th.source === 'import' ? t('theme.srcImport', { version: th.version || '1.0' }) : t('theme.srcCustom')
  const byText = (th: Theme) => (th.source === 'import' ? th.author || t('theme.unknownAuthor') : t('theme.byMe'))

  const mine = computed(() =>
    themeStore.myThemes.map((th) => {
      const content = normalizeContent(th.content)
      const pages = pagesWithOverrides(content).length
      return {
        theme: th,
        content,
        pages,
        src: srcText(th),
        by: byText(th),
        site: th.siteDefault || (!!themeStore.siteTheme && themeStore.siteTheme.themeId === th.id),
        on: !themeStore.usingSite && themeStore.currentTheme === th.id,
      }
    }),
  )

  // ---- 操作 ----
  const selectBuiltin = (key: PaletteKey) => themeStore.setTheme(key)
  const selectMine = (th: Theme) => themeStore.setTheme(th.id, th)

  const editTheme = (id?: string) => {
    void router.push(`/profile/theme/editor/${id || 'new'}`)
  }
  /** 「在线编辑」当前主题：内置主题从当前外观新建 */
  const editCurrent = () => editTheme(themeStore.isBuiltin || themeStore.usingSite ? 'new' : themeStore.currentTheme)

  const previewMine = (th: Theme) => {
    themeStore.startPreview({
      name: th.name,
      content: normalizeContent(th.content),
      scheme: scheme.value,
      from: 'mine',
      themeId: th.id,
      accept: () => selectMine(th),
    })
  }

  const openExport = (target?: ExportTarget) => {
    appearanceUi.exportTarget = target ?? {
      name: currentInfo.value.name,
      content: themeStore.usingSite ? normalizeContent(themeStore.siteTheme?.content) : themeStore.currentEffectiveContent,
      themeId: themeStore.isBuiltin || themeStore.usingSite ? '' : themeStore.currentTheme,
    }
    appearanceUi.exportOpen = true
  }
  const exportMine = (th: Theme) => openExport({ name: th.name, content: normalizeContent(th.content), themeId: th.id })

  const duplicate = async (th: Theme) => {
    const copy = await themeStore.duplicateThemeById(th.id)
    if (copy) feedback.success(t('theme.duplicated', { name: copy.name }))
  }

  const remove = (th: Theme) => {
    const base = normalizeContent(th.content).base
    Dialog.confirm({
      title: t('theme.deleteTitle'),
      content: t('theme.deleteConfirm', { name: th.name, base: paletteName(base) }),
      confirmText: t('theme.actions.delete'),
      cancelText: t('common.cancel'),
      onConfirm: async () => {
        await themeStore.removeTheme(th.id)
        feedback.success(t('theme.deleted'))
      },
    })
  }

  const toggleSiteFor = async (th: Theme | null, enabled: boolean) => {
    await themeStore.setSiteDefault(enabled, th ? { id: th.id, name: th.name, content: normalizeContent(th.content) } : undefined)
    feedback.success(enabled ? t('theme.current.siteSet') : t('theme.current.siteUnset'))
  }

  const mineMenu = (row: (typeof mine.value)[number]) => [
    { key: 'duplicate', label: t('theme.actions.duplicate'), icon: 'i-lucide-copy' },
    {
      key: 'site',
      label: row.site ? t('theme.actions.unsetSite') : t('theme.actions.setSite'),
      icon: 'i-lucide-globe',
      hidden: !canSetSite.value,
    },
    { key: 'delete', label: t('theme.actions.delete'), icon: 'i-lucide-trash-2', danger: true },
  ]
  const onMineMenu = (row: (typeof mine.value)[number], key: string) => {
    if (key === 'duplicate') void duplicate(row.theme)
    else if (key === 'site') void toggleSiteFor(row.theme, !row.site)
    else if (key === 'delete') remove(row.theme)
  }

  // ---- 背景（外观页上的修改：存为当前主题之上的覆盖，自动保存到账号）----
  const bgModel = computed<ThemeBackground>({
    get: () =>
      themeStore.usingSite
        ? normalizeContent(themeStore.siteTheme?.content).background!
        : (themeStore.bgOverride ?? themeStore.currentContent.background!),
    set: (bg) => themeStore.setBackground(bg),
  })

  // ---- 当前主题 ----
  const bgImageName = (bg: ThemeBackground) => {
    const a = bg.image ? findAsset(bg.image) : null
    return (a?.name || '').replace(/\.\w+$/, '')
  }
  const currentInfo = computed(() => {
    const content = themeStore.usingSite ? normalizeContent(themeStore.siteTheme?.content) : themeStore.currentContent
    const base = paletteName(content.base)
    const bg = themeStore.background.bg
    let name = base
    let src = t('theme.current.builtin')
    if (themeStore.usingSite) {
      name = themeStore.siteTheme?.name || base
      src = t('theme.current.followSite')
    } else if (!themeStore.isBuiltin && themeStore.currentCustom) {
      name = themeStore.currentCustom.name
      src = `${srcText(themeStore.currentCustom)} · ${byText(themeStore.currentCustom)}`
    }
    const pages = pagesWithOverrides(content).length
    const imgName = bg.type === 'image' ? bgImageName(bg) : ''
    return {
      name,
      src,
      baseName: base,
      bgName: t(`theme.bgTypes.${bg.type}`) + (imgName ? ` · ${imgName}` : ''),
      pagesLong: pages ? t('theme.pagesLong', { n: pages }) : t('theme.pagesNone'),
      modeName: t(`theme.${themeStore.themeMode}`),
    }
  })

  const swatches = computed(() => {
    const tk = themeStore.tokens
    return (['p', 'a', 'a2', 's', 'ink', 'ink2'] as const).map((k) => ({ key: k, label: t(`theme.swatches.${k}`), color: tk[k] }))
  })

  /** 背景最亮处（叠上遮罩后）的颜色；非图片 / 渐变背景为 null */
  const backdropBright = computed(() => {
    const bg = themeStore.background.bg
    if (!GLASS_TYPES.includes(bg.type as BgType)) return null
    let bright: string | null = null
    if (bg.type === 'gradient') bright = brightestStop(bg.gradientStops)
    else bright = imageStats(assetUrl(bg.image, themeStore.activeResolver))?.bright ?? null
    if (!bright) return null
    return { bright, mixed: backdropBrightColor(bright, themeStore.tokens.bg, effectiveDim(bg, themeStore.scheme)) }
  })

  const contrast = computed(() =>
    contrastChecks(themeStore.tokens, { backdropBright: backdropBright.value?.mixed ?? null }).map((c) => ({
      ...c,
      label: t(`theme.checks.${c.key}`),
      value: formatRatio(c.ratio),
    })),
  )
  const lowContrastDim = computed(() => {
    const b = backdropBright.value
    const low = contrast.value.find((c) => c.key === 'backdrop' && !c.ok)
    if (!b || !low) return null
    return dimForContrast(themeStore.tokens.ink, b.bright, themeStore.tokens.bg)
  })

  return {
    t,
    themeStore,
    canSetSite,
    scheme,
    loadThemes,
    builtins,
    mine,
    selectBuiltin,
    selectMine,
    editTheme,
    editCurrent,
    previewMine,
    openExport,
    exportMine,
    mineMenu,
    onMineMenu,
    toggleSiteFor,
    bgModel,
    currentInfo,
    swatches,
    contrast,
    lowContrastDim,
  }
}
