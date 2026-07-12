// 终端主题（手动切换 · 不受全局明暗影响 · 默认黑色）。
// 两处终端（实例控制台 / SSH）共享同一份设置，持久化到 localStorage。
import { ref } from 'vue'

export type TerminalThemeMode = 'dark' | 'light'

const STORAGE_KEY = 'momoko.terminalTheme'

const readInitial = (): TerminalThemeMode => {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'light' ? 'light' : 'dark'
  } catch {
    return 'dark'
  }
}

// 模块级共享 ref：两个终端页共用一致主题
const theme = ref<TerminalThemeMode>(readInitial())

export function useTerminalTheme() {
  const setTheme = (mode: TerminalThemeMode) => {
    theme.value = mode
    try {
      localStorage.setItem(STORAGE_KEY, mode)
    } catch {
      /* ignore quota/private-mode errors */
    }
  }

  const toggleTheme = () => setTheme(theme.value === 'dark' ? 'light' : 'dark')

  return { theme, setTheme, toggleTheme }
}
