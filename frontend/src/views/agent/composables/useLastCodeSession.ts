// Code 页上次看的会话，按用户记在本地：从别的页回到 Code、地址里没带会话时回到它；去新会话页时清掉。
const KEY = 'agent.code.session'

const read = (): { user: string; id: string } => {
  try {
    return { user: '', id: '', ...JSON.parse(localStorage.getItem(KEY) || '{}') }
  } catch {
    return { user: '', id: '' }
  }
}
const saved = ref(read())

export const useLastCodeSession = () => {
  const userStore = useUserStore()
  const user = () => String(userStore.userInfo?.userId ?? '')
  const id = computed(() => (saved.value.user === user() ? saved.value.id : ''))
  const remember = (v: string) => {
    if (saved.value.user === user() && saved.value.id === v) return
    saved.value = { user: user(), id: v }
    if (v) localStorage.setItem(KEY, JSON.stringify(saved.value))
    else localStorage.removeItem(KEY)
  }
  return { id, remember }
}
