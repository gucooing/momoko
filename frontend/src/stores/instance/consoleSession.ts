import { ElMessage } from 'element-plus'
import type { ConsoleFeatureItem, ConsoleSocketStatus } from '@/stores/instance/types'
import { InstanceStatus, type InstanceInfo } from '@/types/v1/instance'
import { normalizeAuthToken, toBearerAuthHeader } from '@/utils/request'

const MAX_OUTPUT_LINES = 600
const SOCKET_RETRY_DELAY = 5000
const SOCKET_AUTH_QUERY_KEYS = ['accessToken', 'token', 'authorization', 'auth']

type SessionAction = 'load' | 'start' | 'stop' | 'restart'

interface ConsoleSessionOptions {
  defaultOutputLines: string[]
  entityLabel: string
  loadTargetLabel: string
  outputLabel: string
  featureItems: ConsoleFeatureItem[]
  getInfo: () => Promise<InstanceInfo | undefined>
  getContextId?: () => string
  contextIdLabel?: string
  extendSocketUrl?: (url: URL) => void
}

interface LoadInfoOptions {
  silent?: boolean
  refreshOutput?: boolean
}

interface RunActionOptions {
  refreshOutputOnSuccess?: boolean
}

const normalizeLines = (value: string) =>
  value
    .split(/\r?\n/u)
    .map((line) => line.replace(/\r/g, ''))

const buildConsoleLines = (
  info: InstanceInfo | undefined,
  socketState: ConsoleSocketStatus,
  options: Pick<ConsoleSessionOptions, 'defaultOutputLines' | 'entityLabel' | 'outputLabel'>,
) => {
  if (!info) {
    return [...options.defaultOutputLines]
  }

  const lines = [
    `[system] ${options.entityLabel}名称：${info.name || '未提供'}`,
    `[system] ${options.entityLabel}状态：${info.status || '未提供'}`,
  ]

  if (info.status !== InstanceStatus.INSTANCE_STATUS_RUNNING) {
    lines.push(`[system] 当前${options.entityLabel}未运行，命令输入已禁用`)
  }

  if (!info.wsPath) {
    lines.push(`[system] 当前${options.entityLabel}未提供 WS 地址`)
    return lines
  }

  if (socketState === 'connecting') {
    lines.push('[system] WS 连接中...')
  } else if (socketState === 'connected') {
    lines.push(`[system] WS 已连接，等待${options.outputLabel}输出...`)
  } else if (socketState === 'error') {
    lines.push('[system] WS 连接异常，请检查网络或服务状态')
  } else {
    lines.push('[system] WS 未连接')
  }

  return lines
}

export const createConsoleSession = (options: ConsoleSessionOptions) => {
  const terminalInfo = ref<InstanceInfo>()
  const socketStatus = ref<ConsoleSocketStatus>('disconnected')
  const commandValue = ref('')
  const featureItems = ref<ConsoleFeatureItem[]>([...options.featureItems])
  const commandHistory = ref<string[]>([])
  const historyCursor = ref(0)
  const pendingAction = ref<SessionAction | null>(null)
  const outputLines = ref<string[]>([...options.defaultOutputLines])
  const requestToken = ref(0)

  let terminalSocket: WebSocket | null = null
  let activeSocketUrl = ''
  let socketRetryTimer: ReturnType<typeof setTimeout> | null = null
  let manualCloseSocket: WebSocket | null = null

  const canSendCommand = computed(
    () =>
      terminalInfo.value?.status === InstanceStatus.INSTANCE_STATUS_RUNNING &&
      socketStatus.value === 'connected',
  )

  const isBusy = computed(() => pendingAction.value !== null)

  const sendPlaceholder = computed(() => {
    if (terminalInfo.value?.status !== InstanceStatus.INSTANCE_STATUS_RUNNING) {
      return `${options.entityLabel}未运行，当前仅可查看日志`
    }

    if (socketStatus.value !== 'connected') {
      return 'WS 未连接，暂时无法发送命令'
    }

    return '输入命令后回车发送，可用上下键选择历史命令'
  })

  const nextRequestToken = () => {
    requestToken.value += 1
    return requestToken.value
  }

  const isActiveRequest = (token: number) => token === requestToken.value

  const hasRequiredContext = () => {
    if (!options.getContextId) return true
    return !!options.getContextId().trim()
  }

  const appendOutput = (...lines: string[]) => {
    outputLines.value.push(...lines.flatMap((line) => normalizeLines(line)))

    if (outputLines.value.length > MAX_OUTPUT_LINES) {
      outputLines.value = outputLines.value.slice(-MAX_OUTPUT_LINES)
    }
  }

  const replaceOutput = (...lines: string[]) => {
    outputLines.value = []
    appendOutput(...lines)
  }

  const hasDefaultOutput = () =>
    outputLines.value.length === options.defaultOutputLines.length &&
    outputLines.value.every((line, index) => line === options.defaultOutputLines[index])

  const replaceConsoleOutput = (
    info?: InstanceInfo,
    socketState: ConsoleSocketStatus = socketStatus.value,
  ) => replaceOutput(...buildConsoleLines(info, socketState, options))

  const refreshConsoleOnConnected = (info?: InstanceInfo) => {
    if (socketStatus.value !== 'connected') return

    if (hasDefaultOutput()) {
      replaceConsoleOutput(info, 'connected')
      return
    }

    appendOutput('[system] WS 已连接')
  }

  const syncConsolePlaceholder = (
    info?: InstanceInfo,
    socketState: ConsoleSocketStatus = socketStatus.value,
  ) => {
    if (!hasDefaultOutput()) return

    replaceConsoleOutput(info, socketState)
  }

  const clearSocketRetryTimer = () => {
    if (!socketRetryTimer) return

    clearTimeout(socketRetryTimer)
    socketRetryTimer = null
  }

  const closeTerminalSocket = () => {
    clearSocketRetryTimer()
    const socket = terminalSocket
    terminalSocket = null
    activeSocketUrl = ''
    socketStatus.value = 'disconnected'

    if (!socket) {
      return
    }

    manualCloseSocket = socket

    if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
      socket.close()
      return
    }

    if (manualCloseSocket === socket) {
      manualCloseSocket = null
    }
  }

  const getAccessToken = () => normalizeAuthToken(localStorage.getItem('accessToken') || '')

  const buildTerminalSocketUrl = (wsPath: string) => {
    if (!wsPath.trim()) return ''

    let wsUrl: URL

    if (/^wss?:\/\//i.test(wsPath)) {
      wsUrl = new URL(wsPath)
    } else if (/^https?:\/\//i.test(wsPath)) {
      wsUrl = new URL(wsPath)
      wsUrl.protocol = wsUrl.protocol === 'https:' ? 'wss:' : 'ws:'
    } else {
      const apiBaseUrl = new URL(
        import.meta.env.VITE_API_BASE_URL || window.location.origin,
        window.location.origin,
      )
      const wsOrigin = `${apiBaseUrl.protocol === 'https:' ? 'wss:' : 'ws:'}//${apiBaseUrl.host}`
      wsUrl = new URL(wsPath.startsWith('/') ? wsPath : `/${wsPath}`, wsOrigin)
    }

    const accessToken = getAccessToken()
    const hasAuthQuery = SOCKET_AUTH_QUERY_KEYS.some((key) => wsUrl.searchParams.has(key))

    if (accessToken && !hasAuthQuery) {
      wsUrl.searchParams.set('accessToken', toBearerAuthHeader(accessToken))
    }

    options.extendSocketUrl?.(wsUrl)

    return wsUrl.toString()
  }

  const decodeSocketPayload = async (payload: Blob | ArrayBuffer | string) => {
    if (typeof payload === 'string') return payload
    if (payload instanceof Blob) return payload.text()
    return new TextDecoder().decode(payload)
  }

  const handleSocketMessage = async (payload: Blob | ArrayBuffer | string) => {
    const decodedPayload = await decodeSocketPayload(payload)
    if (!decodedPayload) return
    appendOutput(decodedPayload)
  }

  const canRetrySocketConnection = (info?: InstanceInfo) => {
    return !!info?.wsPath
  }

  const scheduleSocketRetry = (token: number) => {
    if (!isActiveRequest(token) || socketRetryTimer || !canRetrySocketConnection(terminalInfo.value)) {
      return
    }

    socketRetryTimer = setTimeout(() => {
      socketRetryTimer = null
      void retryTerminalSocketConnection(token)
    }, SOCKET_RETRY_DELAY)
  }

  const connectTerminalSocket = async (wsPath: string, token = requestToken.value) => {
    if (!isActiveRequest(token)) return

    if (!wsPath) {
      closeTerminalSocket()
      syncConsolePlaceholder(terminalInfo.value)
      return
    }

    const socketUrl = buildTerminalSocketUrl(wsPath)
    if (!socketUrl) {
      socketStatus.value = 'error'
      if (hasDefaultOutput()) {
        replaceOutput('[system] WS 地址无效，无法建立连接')
      } else {
        appendOutput('[system] WS 地址无效，无法建立连接')
      }
      return
    }

    if (
      terminalSocket &&
      activeSocketUrl === socketUrl &&
      (terminalSocket.readyState === WebSocket.OPEN ||
        terminalSocket.readyState === WebSocket.CONNECTING)
    ) {
      return
    }

    closeTerminalSocket()
    activeSocketUrl = socketUrl
    socketStatus.value = 'connecting'
    manualCloseSocket = null

    const socket = new WebSocket(socketUrl)
    terminalSocket = socket

    socket.onopen = () => {
      if (terminalSocket !== socket || !isActiveRequest(token)) return

      socketStatus.value = 'connected'
      refreshConsoleOnConnected(terminalInfo.value)
    }

    socket.onmessage = (event) => {
      if (terminalSocket !== socket || !isActiveRequest(token)) return
      void handleSocketMessage(event.data as Blob | ArrayBuffer | string)
    }

    socket.onerror = () => {
      if (terminalSocket !== socket || !isActiveRequest(token)) return

      socketStatus.value = 'error'
      if (hasDefaultOutput()) {
        syncConsolePlaceholder(terminalInfo.value, 'error')
      } else {
        appendOutput('[system] WS 连接异常')
      }
    }

    socket.onclose = () => {
      const isManualClose = manualCloseSocket === socket
      if (isManualClose) {
        manualCloseSocket = null
      }

      if (terminalSocket !== socket || !isActiveRequest(token)) return

      terminalSocket = null
      activeSocketUrl = ''
      socketStatus.value = 'disconnected'

      if (hasDefaultOutput()) {
        syncConsolePlaceholder(terminalInfo.value, 'disconnected')
      }

      if (isManualClose) {
        if (!hasDefaultOutput()) {
          appendOutput('[system] WS 已断开')
        }
        return
      }

      if (canRetrySocketConnection(terminalInfo.value)) {
        appendOutput(`[system] WS 已断开，${SOCKET_RETRY_DELAY / 1000} 秒后重连`)
        scheduleSocketRetry(token)
        return
      }

      if (!hasDefaultOutput()) {
        appendOutput('[system] WS 已断开')
      }
    }
  }

  const syncTerminalSocket = async (info?: InstanceInfo, token = requestToken.value) => {
    if (info?.wsPath) {
      await connectTerminalSocket(info.wsPath, token)
      return
    }

    closeTerminalSocket()
    syncConsolePlaceholder(info)
  }

  const applyTerminalInfo = async (
    info: InstanceInfo | undefined,
    token: number,
    refreshOutput = true,
  ) => {
    terminalInfo.value = info

    if (refreshOutput) {
      await syncTerminalSocket(info, token)
    }
  }

  const loadTerminalInfo = async (
    loadOptions: LoadInfoOptions = {},
    token = requestToken.value,
  ) => {
    const { silent = false, refreshOutput = true } = loadOptions

    if (!isActiveRequest(token)) return

    if (!silent) {
      pendingAction.value = 'load'
    }

    if (!hasRequiredContext()) {
      closeTerminalSocket()
      terminalInfo.value = undefined

      if (options.contextIdLabel) {
        outputLines.value = [
          `[system] 缺少${options.contextIdLabel}，无法加载${options.loadTargetLabel}`,
        ]
      } else {
        outputLines.value = [...options.defaultOutputLines]
      }

      if (!silent && isActiveRequest(token)) {
        pendingAction.value = null
      }
      return
    }

    try {
      const info = await options.getInfo()
      if (!isActiveRequest(token)) return

      await applyTerminalInfo(info, token, refreshOutput)
    } catch {
      if (!isActiveRequest(token)) return

      if (terminalInfo.value) {
        if (refreshOutput) {
          syncConsolePlaceholder(terminalInfo.value, socketStatus.value)
        }
      } else {
        outputLines.value = [
          `[system] ${options.loadTargetLabel}信息获取失败`,
          '[system] 请检查后端状态后重试',
        ]
      }
    } finally {
      if (!silent && isActiveRequest(token)) {
        pendingAction.value = null
      }
    }
  }

  const retryTerminalSocketConnection = async (token: number) => {
    if (!isActiveRequest(token) || !canRetrySocketConnection(terminalInfo.value)) return
    await loadTerminalInfo({ silent: true }, token)
  }

  const runAction = async (
    action: Exclude<SessionAction, 'load'>,
    requestAction: () => Promise<unknown>,
    actionOptions: RunActionOptions = {},
  ) => {
    const { refreshOutputOnSuccess = false } = actionOptions

    if (!hasRequiredContext()) {
      if (options.contextIdLabel) {
        ElMessage.warning(`缺少${options.contextIdLabel}，无法${action === 'start' ? '启动' : action === 'stop' ? '停止' : '重启'}${options.entityLabel}`)
      }
      return
    }

    const token = requestToken.value
    let actionSucceeded = false
    pendingAction.value = action

    try {
      try {
        await requestAction()
        actionSucceeded = true
      } catch {
        actionSucceeded = false
      }

      await loadTerminalInfo(
        { silent: true, refreshOutput: actionSucceeded ? refreshOutputOnSuccess : true },
        token,
      )

      if (actionSucceeded && isActiveRequest(token)) {
        const actionLabelMap: Record<Exclude<SessionAction, 'load'>, string> = {
          start: '启动',
          stop: '停止',
          restart: '重启',
        }
        ElMessage.success(`${actionLabelMap[action]}成功`)
      }
    } finally {
      if (isActiveRequest(token)) {
        pendingAction.value = null
      }
    }
  }

  const resetSession = () => {
    closeTerminalSocket()
    terminalInfo.value = undefined
    socketStatus.value = 'disconnected'
    commandValue.value = ''
    commandHistory.value = []
    historyCursor.value = 0
    pendingAction.value = null
    outputLines.value = [...options.defaultOutputLines]
  }

  const initializeSession = async () => {
    const token = nextRequestToken()
    resetSession()
    await loadTerminalInfo({}, token)
  }

  const refreshInfo = async (loadOptions: LoadInfoOptions = {}) => {
    await loadTerminalInfo(loadOptions)
  }

  const clearScreen = () => {
    outputLines.value = []
  }

  const executeCommand = () => {
    const command = commandValue.value.trim()
    if (!command) return

    if (!terminalSocket || terminalSocket.readyState !== WebSocket.OPEN) {
      ElMessage.warning('WS 未连接，无法发送命令')
      return
    }

    const lastCommand = commandHistory.value[commandHistory.value.length - 1]
    if (!commandHistory.value.length || lastCommand !== command) {
      commandHistory.value.push(command)
    }

    historyCursor.value = commandHistory.value.length
    commandValue.value = ''
    terminalSocket.send(`${command}\n`)
  }

  const selectPrevCommand = () => {
    if (!commandHistory.value.length) return

    if (historyCursor.value <= 0) {
      historyCursor.value = 0
    } else {
      historyCursor.value -= 1
    }

    commandValue.value = commandHistory.value[historyCursor.value] || ''
  }

  const selectNextCommand = () => {
    if (!commandHistory.value.length) return

    if (historyCursor.value >= commandHistory.value.length - 1) {
      historyCursor.value = commandHistory.value.length
      commandValue.value = ''
      return
    }

    historyCursor.value += 1
    commandValue.value = commandHistory.value[historyCursor.value] || ''
  }

  const dispose = () => {
    nextRequestToken()
    resetSession()
  }

  return {
    pendingAction,
    terminalInfo,
    socketStatus,
    commandValue,
    featureItems,
    outputLines,
    canSendCommand,
    isBusy,
    sendPlaceholder,
    runAction,
    initializeSession,
    refreshInfo,
    clearScreen,
    executeCommand,
    selectPrevCommand,
    selectNextCommand,
    dispose,
  }
}
