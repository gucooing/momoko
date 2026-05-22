import dayjs from 'dayjs'
import { defineStore } from 'pinia'
import { deleteLoginDeviceRequest, devicesRequest } from '@/api/login'
import defaultSystemAvatar from '@/assets/images/defaultSystemAvatar.svg'
import type {
  LoginDeviceRow,
  LoginLogItem,
  ProfileCurrentTab,
  ProfileTabsMenuItem,
  UserProfileFormValue,
  UserMessageItem,
} from '@/stores/user/types'
import type { UserInfo } from '@/types/v1/user'

const COLUMN_WIDTH_MIN = 150
const COLUMN_WIDTH_MAX = 420
const fixedColumns = new Set(['device', 'ip', 'loginTime', 'updateTime', 'sessionId', 'deviceId'])

const createMenuTabs = (): ProfileTabsMenuItem[] => [
  { key: 'personalInfo', label: '我的资料', icon: 'HOutline:UserIcon' },
  { key: 'permissions', label: '我的权限', icon: 'HOutline:ShieldCheckIcon' },
  { key: 'messages', label: '我的消息', icon: 'HOutline:BellAlertIcon' },
  { key: 'logs', label: '登录日志', icon: 'HOutline:ClockIcon' },
  { key: 'devices', label: '登录设备', icon: 'HOutline:DevicePhoneMobileIcon' },
]

const createDefaultMessages = (): UserMessageItem[] => [
  {
    id: '1',
    title: '系统维护通知',
    content: '系统将于今晚 22:00-24:00 进行维护升级，期间可能无法访问，请提前做好准备。',
    type: 'system',
    read: false,
    time: '2026-01-22 08:30:00',
    avatar: defaultSystemAvatar,
  },
  {
    id: '2',
    title: '运维同事',
    content: '今天的任务清单已经更新，记得先确认实例巡检结果。',
    type: 'user',
    read: false,
    time: '2026-01-22 08:45:00',
    avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=Felix',
  },
  {
    id: '3',
    title: '新功能上线',
    content: '个人中心功能已上线，您可以管理个人信息和查看消息通知。',
    type: 'system',
    read: false,
    time: '2026-01-21 17:20:00',
    avatar: defaultSystemAvatar,
  },
  {
    id: '4',
    title: 'Alice L.',
    content: '你的排行榜进度更新了，你现在是第 2 名，继续保持！',
    type: 'user',
    read: true,
    time: '2026-01-21 16:10:00',
    avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=AliceL',
  },
  {
    id: '5',
    title: '安全提醒',
    content: '请定期修改密码，并启用双重验证，保护账户安全。',
    type: 'system',
    read: true,
    time: '2026-01-21 09:30:00',
    avatar: defaultSystemAvatar,
  },
]

const createDefaultLoginLogs = (): LoginLogItem[] => [
  {
    device: 'Windows 11 · Chrome',
    browser: 'Chrome 134.0.6998',
    ip: '203.0.113.24',
    location: '中国香港',
    time: '2026-03-21 09:12:33',
    status: 'success',
  },
  {
    device: 'macOS 15 · Safari',
    browser: 'Safari 18.3',
    ip: '198.51.100.13',
    location: '新加坡',
    time: '2026-03-20 22:45:17',
    status: 'success',
  },
  {
    device: 'Windows 10 · Edge',
    browser: 'Edge 134.0.3124',
    ip: '192.0.2.77',
    location: '日本东京',
    time: '2026-03-20 18:26:54',
    status: 'danger',
  },
]

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
  const address = ref({
    country: '',
    region: '',
    city: '',
  })
  const addressLoaded = ref(false)

  const currentTab = ref<ProfileCurrentTab>('personalInfo')
  const menuTabs = ref<ProfileTabsMenuItem[]>(createMenuTabs())

  const userMessages = ref<UserMessageItem[]>(createDefaultMessages())
  const messageDraft = ref('')
  const activeMessageTab = ref<'all' | 'unread'>('all')

  const unreadCount = computed(() => userMessages.value.filter((msg) => !msg.read).length)

  const messageTabs = computed(() => [
    { key: 'all', label: '全部消息', badge: 0 },
    { key: 'unread', label: '未读消息', badge: unreadCount.value },
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
  const loginLogs = ref<LoginLogItem[]>(createDefaultLoginLogs())

  const currentDeviceLabel = computed(() => {
    if (!currentDeviceId.value) return '-'
    return loginDevices.value.find((item) => item.deviceId === currentDeviceId.value)?.device || '-'
  })

  const extraLabelMap: Record<string, string> = {
    browser: '浏览器',
    os: '系统',
    platform: '平台',
    location: '位置',
    userAgent: 'User Agent',
    status: '状态',
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
    return extraLabelMap[key] || key
  }

  const deviceColumnWidth = computed(() => {
    return calcColumnWidth(
      loginDevices.value.map((item) => item.device),
      '登录设备',
      220,
    )
  })

  const ipColumnWidth = computed(() => {
    return calcColumnWidth(
      loginDevices.value.map((item) => item.ip),
      'IP 地址',
      150,
      260,
    )
  })

  const sessionIdColumnWidth = computed(() => {
    return calcColumnWidth(
      loginDevices.value.map((item) => item.sessionId),
      '会话 ID',
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

  const ensureAddress = async () => {
    if (addressLoaded.value) return

    try {
      const response = await fetch('https://ipapi.co/json/')
      const data = await response.json()
      address.value = {
        country: data.country_name || '',
        region: data.region || '',
        city: data.city || '',
      }
    } catch {
      address.value = {
        country: '',
        region: '',
        city: '',
      }
    } finally {
      addressLoaded.value = true
    }
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
      title: senderName || '未知用户',
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

  const isDeletingLoginDevice = (deviceId: string) => deletingDeviceIds.value.includes(deviceId)

  const deleteLoginDevice = async (deviceId: string) => {
    if (!deviceId || isDeletingLoginDevice(deviceId)) return false

    deletingDeviceIds.value.push(deviceId)

    try {
      await deleteLoginDeviceRequest({ id: deviceId })
      loginDevices.value = loginDevices.value.filter((item) => item.deviceId !== deviceId)
      return true
    } finally {
      deletingDeviceIds.value = deletingDeviceIds.value.filter((id) => id !== deviceId)
    }
  }

  const clearProfileState = () => {
    address.value = {
      country: '',
      region: '',
      city: '',
    }
    addressLoaded.value = false
    currentTab.value = 'personalInfo'
    activeMessageTab.value = 'all'
    messageDraft.value = ''
    userMessages.value = createDefaultMessages()
    profileForm.value = createEmptyProfileForm()
    loading.value = false
    loginDevices.value = []
    currentDeviceId.value = ''
    deletingDeviceIds.value = []
    loginLogs.value = createDefaultLoginLogs()
  }

  return {
    address,
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
    currentDeviceLabel,
    extraColumns,
    deviceColumnWidth,
    ipColumnWidth,
    sessionIdColumnWidth,
    ensureAddress,
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
