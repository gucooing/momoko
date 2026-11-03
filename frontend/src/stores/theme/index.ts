import { defineStore } from 'pinia'
import { useDark, usePreferredDark, useToggle } from '@vueuse/core'

export type ThemeMode = 'light' | 'dark' | 'auto'
export type PaletteKey = 'sakura' | 'mint' | 'volt' | 'grape' | 'sunset' | 'classic'

export interface PaletteOption {
  key: PaletteKey
  /** i18n 名称键 */
  labelKey: string
  /** 英文短名（展示用，不翻译） */
  en: string
  /** 预览色：主色 / 强调色 / 第二强调色（浅色版），用于选择器色块 */
  swatches: [string, string, string]
}

/** 六套配色，色值本身定义在 styles/design-tokens.css（data-palette） */
export const PALETTE_OPTIONS: PaletteOption[] = [
  { key: 'sakura', labelKey: 'theme.palettes.sakura', en: 'SAKURA SODA', swatches: ['#D6245F', '#FFC93C', '#7B5CFF'] },
  { key: 'mint', labelKey: 'theme.palettes.mint', en: 'MINT LEMON', swatches: ['#00805F', '#FFE14D', '#2E9BFF'] },
  { key: 'volt', labelKey: 'theme.palettes.volt', en: 'VOLT', swatches: ['#3450FF', '#FF7A1A', '#00B8A9'] },
  { key: 'grape', labelKey: 'theme.palettes.grape', en: 'GRAPE FIZZ', swatches: ['#7C3AED', '#FFD23F', '#E8457A'] },
  { key: 'sunset', labelKey: 'theme.palettes.sunset', en: 'SUNSET', swatches: ['#C2410C', '#FFC93C', '#2F6FED'] },
  { key: 'classic', labelKey: 'theme.palettes.classic', en: 'CLASSIC', swatches: ['#0F7F72', '#FFB020', '#5B6CFF'] },
]

const DEFAULT_PALETTE: PaletteKey = 'sakura'
const isPaletteKey = (v: unknown): v is PaletteKey => PALETTE_OPTIONS.some((p) => p.key === v)
const readBool = (key: string, fallback: boolean) => {
  try {
    const raw = localStorage.getItem(key)
    return raw == null ? fallback : JSON.parse(raw) === true
  } catch {
    return fallback
  }
}

export const useThemeStore = defineStore('theme', () => {
  const colorSchemeIsDark = useDark()
  const toggleDark = useToggle(colorSchemeIsDark)
  const preferredDark = usePreferredDark()

  // ---- 深浅色 ----
  const themeModeStorage = localStorage.getItem('themeMode')
  const themeMode = ref<ThemeMode>(
    themeModeStorage === 'light' || themeModeStorage === 'dark' || themeModeStorage === 'auto'
      ? themeModeStorage
      : 'auto',
  )
  const isDarkTheme = computed(() =>
    themeMode.value === 'auto' ? preferredDark.value : themeMode.value === 'dark',
  )

  // ---- 配色 ----
  const storedPalette = localStorage.getItem('theme-palette')
  const palette = ref<PaletteKey>(isPaletteKey(storedPalette) ? storedPalette : DEFAULT_PALETTE)
  const currentPalette = computed(
    () => PALETTE_OPTIONS.find((p) => p.key === palette.value) ?? PALETTE_OPTIONS[0]!,
  )

  // ---- 细节 ----
  const compact = ref(readBool('theme-compact', false))
  const reduceMotion = ref(readBool('theme-reduce-motion', false))
  const showTabs = ref(readBool('showTabs', false))

  const applyAttributes = () => {
    const root = document.documentElement
    root.dataset.palette = palette.value
    root.classList.toggle('is-compact', compact.value)
    root.classList.toggle('is-reduce-motion', reduceMotion.value)
  }

  const applyThemeMode = () => {
    toggleDark(isDarkTheme.value)
    applyAttributes()
  }

  const toggleThemeMode = (newVal: ThemeMode) => {
    themeMode.value = newVal
    localStorage.setItem('themeMode', newVal)
    applyThemeMode()
  }

  /** 顶栏按钮：在浅色/深色之间快速切换 */
  const toggleDarkQuick = () => toggleThemeMode(isDarkTheme.value ? 'light' : 'dark')

  const setPalette = (key: PaletteKey) => {
    palette.value = key
    localStorage.setItem('theme-palette', key)
    applyAttributes()
  }

  const setCompact = (value: boolean) => {
    compact.value = value
    localStorage.setItem('theme-compact', JSON.stringify(value))
    applyAttributes()
  }

  const setReduceMotion = (value: boolean) => {
    reduceMotion.value = value
    localStorage.setItem('theme-reduce-motion', JSON.stringify(value))
    applyAttributes()
  }

  const toggleShowTabs = (value: boolean) => {
    showTabs.value = value
    localStorage.setItem('showTabs', JSON.stringify(value))
  }

  const resetAppearance = () => {
    setPalette(DEFAULT_PALETTE)
    setCompact(false)
    setReduceMotion(false)
    toggleThemeMode('auto')
  }

  const themeConfigDrawerOpen = ref(false)

  watch(preferredDark, () => {
    if (themeMode.value === 'auto') applyThemeMode()
  })

  applyThemeMode()

  return {
    themeMode,
    isDarkTheme,
    palette,
    currentPalette,
    paletteOptions: PALETTE_OPTIONS,
    compact,
    reduceMotion,
    showTabs,
    themeConfigDrawerOpen,
    toggleThemeMode,
    toggleDarkQuick,
    setPalette,
    setCompact,
    setReduceMotion,
    toggleShowTabs,
    resetAppearance,
  }
})
