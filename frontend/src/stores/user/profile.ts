import dayjs from 'dayjs'
import { defineStore } from 'pinia'
import { deleteLoginDeviceRequest, devicesRequest, myLoginLogs } from '@/api/login'
import defaultSystemAvatar from '@/assets/images/defaultSystemAvatar.svg'
import type {
  LoginDeviceRow,
  LoginLogItem,
  ProfileCurrentTab,
  ProfileTabsMenuItem,
  UserProfileFormValue,
  UserMessageItem,
} from '@/stores/user/types'
import type { OperationLogInfo } from '@/types/v1/system'
import type { UserInfo } from '@/types/v1/user'
import { translate } from '@/locales'

const COLUMN_WIDTH_MIN = 150
const COLUMN_WIDTH_MAX = 420
const fixedColumns = new Set(['device', 'ip', 'loginTime', 'updateTime', 'sessionId', 'deviceId'])

/** 右栏 Tabs 顺序对齐 06a：消息 → 权限 → 设备 → 日志（资料在左栏，不进 Tab） */
const createMenuTabs = (): ProfileTabsMenuItem[] => [
  { key: 'messages', labelKey: 'user.tabs.messages', icon: 'HOutline:BellAlertIcon' },
  { key: 'permissions', labelKey: 'user.tabs.permissions', icon: 'HOutline:ShieldCheckIcon' },
  { key: 'devices', labelKey: 'user.tabs.devices', icon: 'HOutline:DevicePhoneMobileIcon' },
  { key: 'logs', labelKey: 'user.tabs.logs', icon: 'HOutline:ClockIcon' },
]

const createDefaultMessages = (): UserMessageItem[] => [
  {
    id: '1',
    titleKey: 'user.messages.maintenanceTitle',
    contentKey: 'user.messages.maintenanceContent',
    type: 'system',
    read: false,
    time: '2026-01-22 08:30:00',
    avatar: defaultSystemAvatar,
  },
  {
    id: '2',
    titleKey: 'user.messages.opsTitle',
    contentKey: 'user.messages.opsContent',
    type: 'user',
    read: false,
    time: '2026-01-22 08:45:00',
    avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=Felix',
  },
  {
    id: '3',
    titleKey: 'user.messages.newFeatureTitle',
    contentKey: 'user.messages.newFeatureContent',
    type: 'system',
    read: false,
    time: '2026-01-21 17:20:00',
    avatar: defaultSystemAvatar,
  },
  {
    id: '4',
    title: 'Alice L.',
    contentKey: 'user.messages.rankingContent',
    type: 'user',
    read: true,
    time: '2026-01-21 16:10:00',
    avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=AliceL',
  },
  {
    id: '5',
    titleKey: 'user.messages.securityTitle',
    contentKey: 'user.messages.securityContent',
    type: 'system',
    read: true,
    time: '2026-01-21 09:30:00',
    avatar: defaultSystemAvatar,
  },
]

const toLoginLogItem = (log: OperationLogInfo): LoginLogItem => ({
  detail: log.detail,
  userAgent: log.userAgent,
  ip: log.ip,
  operationTime: log.operationTime ? dayjs(log.operationTime).format('YYYY-MM-DD HH:mm:ss') : '-',
  success: log.success,
})

const SKILL_TAG_TYPES = ['success', 'info', 'warning', 'danger', 'primary'] as const
type SkillTagType = (typeof SKILL_TAG_TYPES)[number]

const createEmptyProfileForm = (): UserProfileFormValue => ({
  avatar: '',
  username: '',
  name: '',
  email: '',
  bio: '',
  tags: '',
})

export const useUserProfileStore = defineStore('user-profile', () => {
  const currentTab = ref<ProfileCurrentTab>('messages')
  const menuTabs = computed<ProfileTabsMenuItem[]>(createMenuTabs)

  const userMessages = ref<UserMessageItem[]>(createDefaultMessages())
  const messageDraft = ref('')
  const activeMessageTab = ref<'all' | 'unread'>('all')

  const unreadCount = computed(() => userMessages.value.filter((msg) => !msg.read).length)

  const messageTabs = computed(() => [
    { key: 'all', labelKey: 'user.messages.all', badge: 0 },
    { key: 'unread', labelKey: 'user.messages.unread', badge: unreadCount.value },
  ])

  const messageList = computed(() => {
    if (activeMessageTab.value === 'unread') {
      return userMessages.value.filter((item) => !item.read)
    }
    return userMessages.value
  })

  const profileForm = ref<UserProfileFormValue>(createEmptyProfileForm())

  const loading = ref(false)
  const loginDevices = ref<LoginDeviceRow[]>([])
  const currentDeviceId = ref('')
  const deletingDeviceIds = ref<string[]>([])
  const loginLogs = ref<LoginLogItem[]>([])

  const currentDeviceLabel = computed(() => {
    if (!currentDeviceId.value) return '-'
    return loginDevices.value.find((item) => item.deviceId === currentDeviceId.value)?.device || '-'
  })

  const extraLabelMap: Record<string, string> = {
    browser: 'user.extra.browser',
    os: 'user.extra.os',
    platform: 'user.extra.platform',
    location: 'user.extra.location',
    userAgent: 'User Agent',
    status: 'user.extra.status',
  }

  const extraColumns = computed(() => {
    const keys = new Set<string>()

    loginDevices.value.forEach((item) => {
      Object.keys(item).forEach((key) => {
        if (!fixedColumns.has(key)) keys.add(key)
      })
    })

    return Array.from(keys)
  })

  const buildSkillTags = (tagsText?: string | null): Array<{ name: string; type: SkillTagType }> => {
    if (!tagsText?.trim()) return []

    return tagsText
      .split(',')
      .map((tag) => tag.trim())
      .filter(Boolean)
      .map((name, index) => ({
        name,
        type: SKILL_TAG_TYPES[index % SKILL_TAG_TYPES.length] ?? 'primary',
      }))
  }

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

  const estimateTextWidth = (value: unknown) => {
    const text = formatCellValue(value)
    const chars = Array.from(text)
    const wideCharCount = chars.filter((char) => /[^\x00-\xff]/.test(char)).length
    const normalCharCount = chars.length - wideCharCount

    return wideCharCount * 16 + normalCharCount * 8 + 32
  }

  const calcColumnWidth = (
    values: unknown[],
    label: string,
    minWidth = COLUMN_WIDTH_MIN,
    maxWidth = COLUMN_WIDTH_MAX,
  ) => {
    const maxWidthByText = Math.max(
      estimateTextWidth(label),
      ...values.map((item) => estimateTextWidth(item)),
    )
    return Math.min(maxWidth, Math.max(minWidth, maxWidthByText))
  }

  const getExtraColumnLabel = (key: string) => {
    const labelKey = extraLabelMap[key]
    return labelKey ? translate(labelKey) : key
  }

  const deviceColumnWidth = computed(() => {
    return calcColumnWidth(
      loginDevices.value.map((item) => item.device),
      translate('user.extra.loginDevice'),
      220,
    )
  })

  const ipColumnWidth = computed(() => {
    return calcColumnWidth(
      loginDevices.value.map((item) => item.ip),
      translate('user.extra.ipAddress'),
      150,
      260,
    )
  })

  const sessionIdColumnWidth = computed(() => {
    return calcColumnWidth(
      loginDevices.value.map((item) => item.sessionId),
      translate('user.extra.sessionId'),
      220,
    )
  })

  const extraColumnWidthMap = computed<Record<string, number>>(() => {
    const widthMap: Record<string, number> = {}

    extraColumns.value.forEach((key) => {
      const values = loginDevices.value.map((item) => item[key])
      widthMap[key] = calcColumnWidth(values, getExtraColumnLabel(key), 160)
    })

    return widthMap
  })

  const getExtraColumnWidth = (key: string) => {
    return extraColumnWidthMap.value[key] || 160
  }

  const markAsRead = (id: string) => {
    const message = userMessages.value.find((msg) => msg.id === id)
    if (message) message.read = true
  }

  const deleteMessage = (id: string) => {
    const index = userMessages.value.findIndex((msg) => msg.id === id)
    if (index !== -1) userMessages.value.splice(index, 1)
  }

  const markAllAsRead = () => {
    userMessages.value.forEach((msg) => {
      if (!msg.read) msg.read = true
    })
  }

  const deleteAllMessages = () => {
    userMessages.value = []
  }

  const sendMessage = (senderName: string, senderAvatar?: string) => {
    const content = messageDraft.value.trim()
    if (!content) return false

    userMessages.value.unshift({
      id: String(Date.now()),
      title: senderName || translate('user.messages.unknownUser'),
      content,
      type: 'user',
      read: false,
      time: dayjs().format('YYYY-MM-DD HH:mm:ss'),
      avatar: senderAvatar || defaultSystemAvatar,
    })
    messageDraft.value = ''
    activeMessageTab.value = 'all'
    return true
  }

  const syncProfileForm = (userInfo?: Partial<UserInfo> | null) => {
    profileForm.value.avatar = userInfo?.avatar || ''
    profileForm.value.username = userInfo?.username || ''
    profileForm.value.name = userInfo?.name || ''
    profileForm.value.email = userInfo?.email || ''
    profileForm.value.bio = userInfo?.bio || ''
    profileForm.value.tags = userInfo?.tags || ''
  }

  const setProfileAvatar = (avatar: string) => {
    profileForm.value.avatar = avatar
  }

  const isCurrentDevice = (device: LoginDeviceRow) => {
    return Boolean(currentDeviceId.value) && device.deviceId === currentDeviceId.value
  }

  const sortDevicesByCurrent = (devices: LoginDeviceRow[], deviceId: string) => {
    if (!deviceId) return devices

    const currentDevices: LoginDeviceRow[] = []
    const otherDevices: LoginDeviceRow[] = []

    devices.forEach((item) => {
      if (item.deviceId === deviceId) currentDevices.push(item)
      else otherDevices.push(item)
    })

    return [...currentDevices, ...otherDevices]
  }

  const loginLogsPagination = ref({ page: 1, pageSize: 10, total: 0 })

  const getMyLoginLogs = async () => {
    try {
      const { data } = await myLoginLogs({
        page: loginLogsPagination.value.page,
        pageSize: loginLogsPagination.value.pageSize,
      })
      loginLogs.value = (data?.logs || []).map(toLoginLogItem)
      loginLogsPagination.value = {
        page: Number(data?.page || 1),
        pageSize: Number(data?.pageSize || 10),
        total: Number(data?.total || 0),
      }
    } catch {
      loginLogs.value = []
    }
  }

  const getLoginDevices = async () => {
    loading.value = true

    try {
      const { data: devicesRes } = await devicesRequest({})

      if (!devicesRes) {
        loginDevices.value = []
        currentDeviceId.value = ''
        return
      }

      currentDeviceId.value = devicesRes.deviceId || ''
      const deviceList = (devicesRes.devices || []) as LoginDeviceRow[]
      loginDevices.value = sortDevicesByCurrent(deviceList, currentDeviceId.value)
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

  const clearProfileState = () => {
    currentTab.value = 'messages'
    activeMessageTab.value = 'all'
    messageDraft.value = ''
    userMessages.value = createDefaultMessages()
    profileForm.value = createEmptyProfileForm()
    loading.value = false
    loginDevices.value = []
    currentDeviceId.value = ''
    deletingDeviceIds.value = []
    loginLogs.value = []
  }

  return {
    currentTab,
    menuTabs,
    userMessages,
    unreadCount,
    messageDraft,
    activeMessageTab,
    messageTabs,
    messageList,
    profileForm,
    loading,
    loginDevices,
    loginLogs,
    loginLogsPagination,
    getMyLoginLogs,
    currentDeviceLabel,
    extraColumns,
    deviceColumnWidth,
    ipColumnWidth,
    sessionIdColumnWidth,
    buildSkillTags,
    markAsRead,
    deleteMessage,
    markAllAsRead,
    deleteAllMessages,
    sendMessage,
    syncProfileForm,
    setProfileAvatar,
    formatDateTime,
    formatCellValue,
    getExtraColumnLabel,
    getExtraColumnWidth,
    isCurrentDevice,
    isDeletingLoginDevice,
    getLoginDevices,
    deleteLoginDevice,
    clearProfileState,
  }
})
