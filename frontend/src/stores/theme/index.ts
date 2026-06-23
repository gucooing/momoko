import { defineStore } from 'pinia'
import { useDark, usePreferredDark, useToggle } from '@vueuse/core'

type ThemeMode = 'light' | 'dark' | 'auto'
type LayoutMode = 'leftMode' | 'topMode'

export const useThemeStore = defineStore('theme', () => {
  const colorSchemeIsDark = useDark()
  const toggleDark = useToggle(colorSchemeIsDark)
  const preferredDark = usePreferredDark()

  // 作为 iframe 被 Sub2API 控制台嵌入时，父级通过 ?theme=light|dark 下发主题，
  // 并在切换主题时携带新值重载 iframe；这里读取后作为初始主题随之切换。
  // 仅在 ui_mode=embedded 时生效，且不写入 localStorage，避免覆盖独立访问时用户自己的选择。
  const readEmbeddedTheme = (): 'light' | 'dark' | null => {
    if (typeof window === 'undefined') return null
    const params = new URLSearchParams(window.location.search)
    if (params.get('ui_mode') !== 'embedded') return null
    const theme = params.get('theme')
    return theme === 'light' || theme === 'dark' ? theme : null
  }

  const themeModeStorage = localStorage.getItem('themeMode')
  const themeMode = ref<ThemeMode>(
    readEmbeddedTheme() ??
      (themeModeStorage === 'light' || themeModeStorage === 'dark' || themeModeStorage === 'auto'
        ? themeModeStorage
        : 'auto'),
  )

  const isDarkTheme = computed(() => {
    if (themeMode.value === 'auto') return preferredDark.value
    return themeMode.value === 'dark'
  })

  const setPrimaryColor = (color: string) => {
    const root = document.documentElement
    const isDark = isDarkTheme.value
    const mixLightTarget = isDark ? '#141414' : '#ffffff'
    const mixDarkTarget = isDark ? '#ffffff' : '#000000'

    root.style.setProperty('--el-color-primary', color)
    root.style.setProperty(
      '--el-color-primary-light-3',
      `color-mix(in srgb, ${color} 70%, ${mixLightTarget})`,
    )
    root.style.setProperty(
      '--el-color-primary-light-5',
      `color-mix(in srgb, ${color} 50%, ${mixLightTarget})`,
    )
    root.style.setProperty(
      '--el-color-primary-light-7',
      `color-mix(in srgb, ${color} 30%, ${mixLightTarget})`,
    )
    root.style.setProperty(
      '--el-color-primary-light-8',
      `color-mix(in srgb, ${color} 20%, ${mixLightTarget})`,
    )
    root.style.setProperty(
      '--el-color-primary-light-9',
      `color-mix(in srgb, ${color} 10%, ${mixLightTarget})`,
    )
    root.style.setProperty(
      '--el-color-primary-dark-2',
      `color-mix(in srgb, ${color} 80%, ${mixDarkTarget})`,
    )
  }

  const primaryColorOptions = [
    { value: '#8B5CF6', labelKey: 'theme.colors.purple' },
    { value: '#F7CEE0', labelKey: 'theme.colors.sakura' },
    { value: '#10B981', labelKey: 'theme.colors.green' },
    { value: '#F59E0B', labelKey: 'theme.colors.orange' },
    { value: '#EF4444', labelKey: 'theme.colors.red' },
    { value: '#6366F1', labelKey: 'theme.colors.indigo' },
    { value: '#1677FF', labelKey: 'theme.colors.blue' },
    { value: '#0EA5E9', labelKey: 'theme.colors.skyBlue' },
    { value: '#00BCD4', labelKey: 'theme.colors.cyan' },
    { value: '#909399', labelKey: 'theme.colors.gray' },
  ]

  const primaryColor = ref(localStorage.getItem('theme-color-primary') || '#8B5CF6')

  const applyThemeMode = () => {
    toggleDark(isDarkTheme.value)
    setPrimaryColor(primaryColor.value)
  }

  const toggleThemeMode = (newVal: ThemeMode) => {
    themeMode.value = newVal
    localStorage.setItem('themeMode', newVal)
    applyThemeMode()
  }

  const togglePrimaryColor = (colorValue: string) => {
    primaryColor.value = colorValue
    localStorage.setItem('theme-color-primary', colorValue)
    setPrimaryColor(colorValue)
  }

  const layout = ref<LayoutMode>((localStorage.getItem('layout') as LayoutMode) || 'leftMode')
  const toggleLayout = (newVal: LayoutMode) => {
    layout.value = newVal
    localStorage.setItem('layout', newVal)
  }

  const showLogo = ref(JSON.parse(localStorage.getItem('showLogo') || 'true'))
  const toggleShowLogo = (value: boolean) => {
    showLogo.value = value
    localStorage.setItem('showLogo', JSON.stringify(showLogo.value))
  }

  const showTabs = ref(JSON.parse(localStorage.getItem('showTabs') || 'true'))
  const toggleShowTabs = (value: boolean) => {
    showTabs.value = value
    localStorage.setItem('showTabs', JSON.stringify(showTabs.value))
  }

  const themeConfigDrawerOpen = ref(false)

  watch(preferredDark, () => {
    if (themeMode.value === 'auto') {
      applyThemeMode()
    }
  })

  // 作为 iframe 被 Sub2API 控制台嵌入时，父级在切换主题时通过 postMessage 下发最新主题，
  // 这里实时应用（不写入 localStorage，避免覆盖独立访问时用户自己的选择）。
  if (typeof window !== 'undefined') {
    window.addEventListener('message', (event: MessageEvent) => {
      const data = event.data
      if (!data || data.source !== 'sub2api' || data.type !== 'theme') return
      if (data.theme === 'light' || data.theme === 'dark') {
        themeMode.value = data.theme
        applyThemeMode()
      }
    })
  }

  applyThemeMode()

  return {
    layout,
    themeMode,
    isDarkTheme,
    primaryColor,
    showLogo,
    showTabs,
    themeConfigDrawerOpen,
    primaryColorOptions,
    toggleThemeMode,
    toggleLayout,
    togglePrimaryColor,
    toggleShowLogo,
    toggleShowTabs,
  }
})
