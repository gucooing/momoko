// 界面状态：深浅色、手机布局、轻提示。
import { defineStore } from 'pinia'
import { onScopeDispose, ref } from 'vue'

export type ToastTone = 'ok' | 'bad' | 'info'

export interface Toast {
  id: number
  text: string
  tone: ToastTone
}

const SCHEME_KEY = 'passport.scheme'

export const useUiStore = defineStore('passport-ui', () => {
  const dark = ref(false)
  const isMobile = ref(false)
  const toasts = ref<Toast[]>([])
  let seq = 0

  function applyScheme(value: boolean) {
    dark.value = value
    document.documentElement.classList.toggle('dark', value)
    document.documentElement.style.colorScheme = value ? 'dark' : 'light'
  }

  function initScheme() {
    const saved = localStorage.getItem(SCHEME_KEY)
    applyScheme(saved ? saved === 'dark' : window.matchMedia('(prefers-color-scheme: dark)').matches)
  }

  function toggleDark() {
    applyScheme(!dark.value)
    localStorage.setItem(SCHEME_KEY, dark.value ? 'dark' : 'light')
  }

  const mq = window.matchMedia('(max-width: 768px)')
  const onMq = () => {
    isMobile.value = mq.matches
  }
  onMq()
  mq.addEventListener('change', onMq)
  onScopeDispose(() => mq.removeEventListener('change', onMq))

  function toast(text: string, tone: ToastTone = 'info') {
    const id = ++seq
    toasts.value.push({ id, text, tone })
    setTimeout(() => {
      toasts.value = toasts.value.filter((x) => x.id !== id)
    }, 3600)
  }

  return { dark, isMobile, toasts, initScheme, toggleDark, toast }
})
