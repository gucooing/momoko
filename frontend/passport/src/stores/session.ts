// 登录状态、当前账号与公开配置。
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@passport/api/passport'
import { tokenStore } from '@passport/api/request'
import { setLocale } from '@passport/locales'
import type { GetPassportConfigResponse, PassportLoginResponse, PassportMe } from '@/types/v1/passport'

export const useSessionStore = defineStore('passport-session', () => {
  const me = ref<PassportMe | null>(null)
  const config = ref<GetPassportConfigResponse | null>(null)
  const loggedIn = ref(!!tokenStore.access())

  const unread = computed(() => me.value?.unreadMessages ?? 0)

  async function loadConfig(force = false) {
    if (config.value && !force) return config.value
    config.value = await api.getConfig()
    if (config.value.siteName) document.title = config.value.siteName
    return config.value
  }

  function applyMe(next: PassportMe) {
    const langChanged = next.language !== me.value?.language
    me.value = next
    if (langChanged && next.language) setLocale(next.language)
  }

  async function loadMe() {
    applyMe(await api.getMe())
    return me.value
  }

  function acceptTokens(res: PassportLoginResponse) {
    tokenStore.set(res.accessToken, res.refreshToken)
    loggedIn.value = true
  }

  async function logout() {
    try {
      if (tokenStore.access()) await api.logout()
    } catch {
      // 会话可能已经失效，忽略
    }
    clear()
  }

  function clear() {
    tokenStore.clear()
    loggedIn.value = false
    me.value = null
  }

  function setUnread(n: number) {
    if (me.value) me.value = { ...me.value, unreadMessages: n }
  }

  return { me, config, loggedIn, unread, loadConfig, loadMe, applyMe, acceptTokens, logout, clear, setUnread }
})
