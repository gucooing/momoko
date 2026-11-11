import dayjs from 'dayjs'
import { defineStore } from 'pinia'
import {
  deleteLoginDeviceRequest,
  deleteMessagesRequest,
  devicesRequest,
  listMessagesRequest,
  logoutOtherDevicesRequest,
  myLoginLogs,
  publishMessageRequest,
  readMessagesRequest,
} from '@/api/login'
import defaultSystemAvatar from '@/assets/images/defaultSystemAvatar.svg'
import type {
  LoginDeviceRow,
  LoginLogItem,
  MessageType,
  ProfileCurrentTab,
  ProfileTabsMenuItem,
  UserProfileFormValue,
  UserMessageItem,
} from '@/stores/user/types'
import type { UserMessage } from '@/types/v1/auth'
import type { LoginLogInfo } from '@/types/v1/user'
import type { UserInfo } from '@/types/v1/user'
import { resolveAvatarUrl } from '@/utils/assets'
import { translate } from '@/locales'

const fixedColumns = new Set([
  'device',
  'ip',
  'loginTime',
  'updateTime',
  'sessionId',
  'deviceId',
  'userAgent',
  'deviceType',
  'ipScope',
  'lastActiveTime',
  'lastIp',
  'loginMethod',
  'expireTime',
])

/** 个人中心分区（顶栏「查看全部消息」等入口会把它设为 messages） */
const createMenuTabs = (): ProfileTabsMenuItem[] => [
  { key: 'messages', labelKey: 'user.tabs.messages', icon: 'HOutline:BellAlertIcon' },
  { key: 'permissions', labelKey: 'user.tabs.permissions', icon: 'HOutline:ShieldCheckIcon' },
  { key: 'devices', labelKey: 'user.tabs.devices', icon: 'HOutline:DevicePhoneMobileIcon' },
  { key: 'logs', labelKey: 'user.tabs.logs', icon: 'HOutline:ClockIcon' },
]

const MESSAGE_KINDS = ['login_new_device', 'login_failed', 'password_changed', 'password_reset']
const MESSAGE_POLL_MS = 60_000
const MESSAGE_PAGE_SIZE = 50

const toIso = (value: unknown) => (value ? String(value instanceof Date ? value.toISOString() : value) : '')

/** IP 网络范围 → 文案 */
export const ipScopeLabel = (scope?: string) => {
  if (scope === 'loopback' || scope === 'lan' || scope === 'public') {
    return translate(`profile.ipScope.${scope}`)
  }
  return ''
}

/** 后端消息 → 当前语言的标题 / 内容 */
const resolveMessage = (msg: UserMessage): UserMessageItem => {
  const params = msg.params || {}
  const tplParams: Record<string, string> = {
    ...params,
    device:
      params.device && params.device !== 'unknown' ? params.device : translate('profile.device.unknown'),
    ip: params.ip || '—',
    scope: ipScopeLabel(params.ip_scope),
    method: params.method === 'email_code' ? translate('profile.method.emailCode') : translate('profile.method.password'),
  }
  const known = MESSAGE_KINDS.includes(msg.kind)
  const isNotice = msg.kind === 'notice'
  const type: MessageType =
    msg.category === 'security' || msg.category === 'notice' ? msg.category : 'system'
  const title = isNotice
    ? msg.title || translate('profile.msg.noticeFrom', { name: msg.senderName || '—' })
    : msg.title || (known ? translate(`profile.msg.kinds.${msg.kind}.title`, tplParams) : msg.kind)
  const content = isNotice
    ? msg.content
    : msg.content || (known ? translate(`profile.msg.kinds.${msg.kind}.content`, tplParams) : '')
  const createTime = toIso(msg.createTime)
  return {
    id: msg.id,
    title,
    content,
    type,
    kind: msg.kind,
    params,
    read: msg.read,
    createTime,
    time: createTime ? dayjs(createTime).format('YYYY-MM-DD HH:mm:ss') : '',
    avatar: isNotice ? resolveAvatarUrl(msg.senderAvatar) || defaultSystemAvatar : defaultSystemAvatar,
    senderName: msg.senderName,
  }
}

const toLoginLogItem = (log: LoginLogInfo): LoginLogItem => ({
  time: toIso(log.time),
  ip: log.ip,
  ipScope: log.ipScope,
  userAgent: log.userAgent,
  device: log.device,
  deviceType: log.deviceType,
  method: log.method,
  success: log.success,
  error: log.error,
  detail: log.detail,
})

const createEmptyProfileForm = (): UserProfileFormValue => ({
  username: '',
  name: '',
  email: '',
  bio: '',
  tags: '',
  phone: '',
})

const hasToken = () => {
  try {
    return Boolean(localStorage.getItem('accessToken'))
  } catch {
    return false
  }
}

export const useUserProfileStore = defineStore('user-profile', () => {
  const currentTab = ref<ProfileCurrentTab>('messages')
  const menuTabs = computed<ProfileTabsMenuItem[]>(createMenuTabs)

  // ---- 站内消息（后端） ----
  const rawMessages = ref<UserMessage[]>([])
  const messagesTotal = ref(0)
  const serverUnread = ref(0)
  const messagesLoading = ref(false)
  const messagesLoaded = ref(false)
  const messageDraft = ref('')
  const messageDraftTitle = ref('')
  const publishing = ref(false)
  const activeMessageTab = ref<'all' | 'unread'>('all')

  /** 已按当前语言解析（computed，切换语言自动更新） */
  const userMessages = computed<UserMessageItem[]>(() => rawMessages.value.map(resolveMessage))
  const unreadCount = computed(() => serverUnread.value)

  const messageTabs = computed(() => [
    { key: 'all', labelKey: 'user.messages.all', badge: messagesTotal.value },
    { key: 'unread', labelKey: 'user.messages.unread', badge: unreadCount.value },
  ])

  const messageList = computed(() => {
    if (activeMessageTab.value === 'unread') {
      return userMessages.value.filter((item) => !item.read)
    }
    return userMessages.value
  })

  const refreshMessages = async () => {
    if (!hasToken()) return
    messagesLoading.value = true
    try {
      const { data } = await listMessagesRequest({ page: 1, pageSize: MESSAGE_PAGE_SIZE })
      rawMessages.value = data?.messages || []
      messagesTotal.value = Number(data?.allCount || 0)
      serverUnread.value = Number(data?.unreadCount || 0)
      messagesLoaded.value = true
    } catch {
      // 静默：轮询失败不打扰
    } finally {
      messagesLoading.value = false
    }
  }

  const markAsRead = async (id: string) => {
    const msg = rawMessages.value.find((m) => m.id === id)
    if (!msg || msg.read) return
    msg.read = true
    serverUnread.value = Math.max(0, serverUnread.value - 1)
    try {
      const { data } = await readMessagesRequest({ ids: [id] })
      serverUnread.value = Number(data?.unreadCount || 0)
    } catch {
      void refreshMessages()
    }
  }

  const markAllAsRead = async () => {
    rawMessages.value.forEach((m) => (m.read = true))
    serverUnread.value = 0
    try {
      const { data } = await readMessagesRequest({ all: true })
      serverUnread.value = Number(data?.unreadCount || 0)
    } catch {
      void refreshMessages()
    }
  }

  const deleteMessage = async (id: string) => {
    const msg = rawMessages.value.find((m) => m.id === id)
    rawMessages.value = rawMessages.value.filter((m) => m.id !== id)
    if (msg) messagesTotal.value = Math.max(0, messagesTotal.value - 1)
    if (msg && !msg.read) serverUnread.value = Math.max(0, serverUnread.value - 1)
    try {
      const { data } = await deleteMessagesRequest({ ids: [id] })
      serverUnread.value = Number(data?.unreadCount || 0)
    } catch {
      void refreshMessages()
    }
  }

  const deleteAllMessages = async () => {
    rawMessages.value = []
    messagesTotal.value = 0
    serverUnread.value = 0
    try {
      await deleteMessagesRequest({ all: true })
    } catch {
      void refreshMessages()
    }
  }

  /** 发布通知给全部启用的用户；返回送达人数，失败抛错 */
  const publishMessage = async () => {
    const content = messageDraft.value.trim()
    if (!content) return 0
    publishing.value = true
    try {
      const { data } = await publishMessageRequest({
        title: messageDraftTitle.value.trim(),
        content,
      })
      messageDraft.value = ''
      messageDraftTitle.value = ''
      activeMessageTab.value = 'all'
      await refreshMessages()
      return Number(data?.recipients || 0)
    } finally {
      publishing.value = false
    }
  }

  let pollTimer: ReturnType<typeof setInterval> | null = null
  const startPolling = () => {
    if (pollTimer || typeof window === 'undefined') return
    pollTimer = setInterval(() => {
      if (document.visibilityState === 'visible') void refreshMessages()
    }, MESSAGE_POLL_MS)
  }

  // ---- 个人资料表单 ----
  const profileForm = ref<UserProfileFormValue>(createEmptyProfileForm())

  // ---- 登录设备 ----
  const loading = ref(false)
  const loginDevices = ref<LoginDeviceRow[]>([])
  const currentSessionId = ref('')
  const deletingDeviceIds = ref<string[]>([])
  const loggingOutOthers = ref(false)

  const currentDeviceLabel = computed(() => {
    if (!currentSessionId.value) return '-'
    return loginDevices.value.find((item) => item.sessionId === currentSessionId.value)?.device || '-'
  })

  const extraLabelMap: Record<string, string> = {
    browser: 'user.extra.browser',
    os: 'user.extra.os',
    platform: 'user.extra.platform',
    location: 'user.extra.location',
    status: 'user.extra.status',
  }

  /** 后端将来新增、页面未专门展示的字段：详情里兜底列出 */
  const extraColumns = computed(() => {
    const keys = new Set<string>()
    loginDevices.value.forEach((item) => {
      Object.keys(item).forEach((key) => {
        if (!fixedColumns.has(key)) keys.add(key)
      })
    })
    return Array.from(keys)
  })

  const formatDateTime = (value: unknown) => {
    if (!value) return '-'
    const nextValue = value instanceof Date ? value : String(value)
    const date = dayjs(nextValue)
    if (date.isValid()) return date.format('YYYY-MM-DD HH:mm:ss')
    return String(value)
  }

  const formatCellValue = (value: unknown) => {
    if (value === null || value === undefined || value === '') return '-'
    if (Array.isArray(value)) return value.join(' / ')
    if (value instanceof Date) return formatDateTime(value)
    if (typeof value === 'object') return JSON.stringify(value)
    return String(value)
  }

  const getExtraColumnLabel = (key: string) => {
    const labelKey = extraLabelMap[key]
    return labelKey ? translate(labelKey) : key
  }

  const syncProfileForm = (userInfo?: Partial<UserInfo> | null) => {
    profileForm.value.username = userInfo?.username || ''
    profileForm.value.name = userInfo?.name || ''
    profileForm.value.email = userInfo?.email || ''
    profileForm.value.bio = userInfo?.bio || ''
    profileForm.value.tags = userInfo?.tags || ''
    profileForm.value.phone = userInfo?.phone || ''
  }

  const isCurrentDevice = (device: LoginDeviceRow) => {
    return Boolean(currentSessionId.value) && device.sessionId === currentSessionId.value
  }

  const sortDevicesByCurrent = (devices: LoginDeviceRow[], sessionId: string) => {
    if (!sessionId) return devices
    const currentDevices: LoginDeviceRow[] = []
    const otherDevices: LoginDeviceRow[] = []
    devices.forEach((item) => {
      if (item.sessionId === sessionId) currentDevices.push(item)
      else otherDevices.push(item)
    })
    return [...currentDevices, ...otherDevices]
  }

  const getLoginDevices = async () => {
    loading.value = true
    try {
      const { data: devicesRes } = await devicesRequest({})
      if (!devicesRes) {
        loginDevices.value = []
        currentSessionId.value = ''
        return
      }
      currentSessionId.value = devicesRes.sessionId || ''
      const deviceList = (devicesRes.devices || []) as LoginDeviceRow[]
      loginDevices.value = sortDevicesByCurrent(deviceList, currentSessionId.value)
    } finally {
      loading.value = false
    }
  }

  const isDeletingLoginDevice = (sessionId: string) => deletingDeviceIds.value.includes(sessionId)

  const deleteLoginDevice = async (sessionId: string) => {
    if (!sessionId || isDeletingLoginDevice(sessionId)) return false
    deletingDeviceIds.value.push(sessionId)
    try {
      await deleteLoginDeviceRequest({ sessionId })
      loginDevices.value = loginDevices.value.filter((item) => item.sessionId !== sessionId)
      return true
    } finally {
      deletingDeviceIds.value = deletingDeviceIds.value.filter((id) => id !== sessionId)
    }
  }

  /** 下线除当前会话外的全部设备 */
  const logoutOtherDevices = async () => {
    if (loggingOutOthers.value) return false
    loggingOutOthers.value = true
    try {
      await logoutOtherDevicesRequest()
      loginDevices.value = loginDevices.value.filter((item) => isCurrentDevice(item))
      return true
    } finally {
      loggingOutOthers.value = false
    }
  }

  // ---- 登录日志 ----
  const loginLogs = ref<LoginLogItem[]>([])
  const loginLogsLoading = ref(false)
  const loginLogsPagination = ref({ page: 1, pageSize: 10, total: 0 })
  /** 结果筛选：all / success / failed */
  const loginLogsFilter = ref<'all' | 'success' | 'failed'>('all')
  const loginLogsStats = ref({ success: 0, failed: 0 })

  const getMyLoginLogs = async () => {
    loginLogsLoading.value = true
    try {
      const { data } = await myLoginLogs({
        page: loginLogsPagination.value.page,
        pageSize: loginLogsPagination.value.pageSize,
        success: loginLogsFilter.value === 'all' ? undefined : loginLogsFilter.value === 'success',
      })
      loginLogs.value = (data?.logs || []).map(toLoginLogItem)
      loginLogsPagination.value = {
        page: Number(data?.page || 1),
        pageSize: Number(data?.pageSize || 10),
        total: Number(data?.total || 0),
      }
      loginLogsStats.value = {
        success: Number(data?.successCount || 0),
        failed: Number(data?.failedCount || 0),
      }
    } catch {
      loginLogs.value = []
    } finally {
      loginLogsLoading.value = false
    }
  }

  const clearProfileState = () => {
    currentTab.value = 'messages'
    activeMessageTab.value = 'all'
    messageDraft.value = ''
    messageDraftTitle.value = ''
    rawMessages.value = []
    messagesTotal.value = 0
    serverUnread.value = 0
    messagesLoaded.value = false
    profileForm.value = createEmptyProfileForm()
    loading.value = false
    loginDevices.value = []
    currentSessionId.value = ''
    deletingDeviceIds.value = []
    loginLogs.value = []
    loginLogsFilter.value = 'all'
    loginLogsPagination.value = { page: 1, pageSize: 10, total: 0 }
    loginLogsStats.value = { success: 0, failed: 0 }
  }

  // 顶栏通知会在登录后创建本 store：立即拉一次并开始轮询。
  void refreshMessages()
  startPolling()

  return {
    currentTab,
    menuTabs,
    userMessages,
    unreadCount,
    messagesTotal,
    messagesLoading,
    messagesLoaded,
    messageDraft,
    messageDraftTitle,
    publishing,
    activeMessageTab,
    messageTabs,
    messageList,
    refreshMessages,
    markAsRead,
    deleteMessage,
    markAllAsRead,
    deleteAllMessages,
    publishMessage,
    profileForm,
    loading,
    loginDevices,
    currentSessionId,
    loggingOutOthers,
    loginLogs,
    loginLogsLoading,
    loginLogsPagination,
    loginLogsFilter,
    loginLogsStats,
    getMyLoginLogs,
    currentDeviceLabel,
    extraColumns,
    syncProfileForm,
    formatDateTime,
    formatCellValue,
    getExtraColumnLabel,
    isCurrentDevice,
    isDeletingLoginDevice,
    getLoginDevices,
    deleteLoginDevice,
    logoutOtherDevices,
    clearProfileState,
  }
})
