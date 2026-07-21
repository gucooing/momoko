import type { ConsoleFeatureItem, ConsoleSocketStatus } from '@/stores/instance/types'
import type { InstanceInfo } from '@/types/v1/instance'
import { normalizeAuthToken, toBearerAuthHeader } from '@/utils/request'
import { translate } from '@/locales'

const SOCKET_RETRY_DELAY = 5000
const SOCKET_AUTH_QUERY_KEYS = ['accessToken', 'token', 'authorization', 'auth']
// 回放缓冲上限：xterm 挂载晚于 ws 连接时，用它补齐已收到的输出
const RAW_BUFFER_LIMIT = 200_000

type SessionAction = 'load' | 'start' | 'stop' | 'restart'

const getActionLabel = (action: Exclude<SessionAction, 'load'>) => {
  if (action === 'start') return translate('instance.actionStart')
  if (action === 'stop') return translate('instance.actionStop')
  return translate('instance.actionRestart')
}

interface ConsoleSessionOptions {
  entityLabel: string
  loadTargetLabel: string
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

// createConsoleSession 管理一条实例终端会话：
// 输出是后端 PTY 的原始字节流（二进制 ws 帧），原样推给 xterm 渲染；
// 输入是原始键盘流(sendRaw)与 resize 控制帧(sendResize)，与 SSH 终端协议同构。
export const createConsoleSession = (options: ConsoleSessionOptions) => {
  const terminalInfo = ref<InstanceInfo>()
  const socketStatus = ref<ConsoleSocketStatus>('disconnected')
  const featureItems = ref<ConsoleFeatureItem[]>([...options.featureItems])
  const pendingAction = ref<SessionAction | null>(null)
  const requestToken = ref(0)

  // —— 原始输出流：订阅者(xterm)实时接收，rawBuffer 供挂载时回放 ——
  const outputListeners = new Set<(chunk: string) => void>()
  let rawBuffer = ''
  // 流式 UTF-8 解码：多字节字符可能跨帧边界，stream:true 缓存半个字符避免乱码
  let streamDecoder = new TextDecoder('utf-8')

  const emitOutput = (chunk: string) => {
    if (!chunk) return
    rawBuffer += chunk
    if (rawBuffer.length > RAW_BUFFER_LIMIT) {
      rawBuffer = rawBuffer.slice(-RAW_BUFFER_LIMIT)
    }
    outputListeners.forEach((cb) => cb(chunk))
  }

  const onOutput = (cb: (chunk: string) => void) => {
    outputListeners.add(cb)
    return () => outputListeners.delete(cb)
  }

  const getBuffer = () => rawBuffer

  // system 以暗淡样式向终端写入一条本地状态消息（连接失败/重试等），不经过后端
  const system = (text: string) => {
    emitOutput(`\x1b[2m[system] ${text}\x1b[0m\r\n`)
  }

  // 每次连接成功后后端会回放最近日志，清空本地缓冲避免重复；
  // 视图侧监听 socketStatus 变为 connected 时同步 reset xterm。
  const resetOutput = () => {
    rawBuffer = ''
    streamDecoder = new TextDecoder('utf-8')
  }

  let terminalSocket: WebSocket | null = null
  let activeSocketUrl = ''
  let socketRetryTimer: ReturnType<typeof setTimeout> | null = null
  let manualCloseSocket: WebSocket | null = null

  const isBusy = computed(() => pendingAction.value !== null)

  const nextRequestToken = () => {
    requestToken.value += 1
    return requestToken.value
  }

  const isActiveRequest = (token: number) => token === requestToken.value

  const hasRequiredContext = () => {
    if (!options.getContextId) return true
    return !!options.getContextId().trim()
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

  const decodeSocketPayload = (payload: ArrayBuffer | string): string => {
    if (typeof payload === 'string') return payload
    // 流式解码：跨帧缓存不完整的多字节序列
    return streamDecoder.decode(payload, { stream: true })
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
      system(translate('instance.noWsAddress', { entity: options.entityLabel }))
      return
    }

    const socketUrl = buildTerminalSocketUrl(wsPath)
    if (!socketUrl) {
      socketStatus.value = 'error'
      system(translate('instance.invalidWsAddress'))
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
    socket.binaryType = 'arraybuffer' // 二进制帧取 ArrayBuffer，便于同步、按序流式解码
    terminalSocket = socket

    socket.onopen = () => {
      if (terminalSocket !== socket || !isActiveRequest(token)) return

      resetOutput()
      socketStatus.value = 'connected'
    }

    socket.onmessage = (event) => {
      if (terminalSocket !== socket || !isActiveRequest(token)) return
      emitOutput(decodeSocketPayload(event.data as ArrayBuffer | string))
    }

    socket.onerror = () => {
      if (terminalSocket !== socket || !isActiveRequest(token)) return

      socketStatus.value = 'error'
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

      if (isManualClose) {
        return
      }

      if (canRetrySocketConnection(terminalInfo.value)) {
        system(translate('instance.wsRetryAfter', { seconds: SOCKET_RETRY_DELAY / 1000 }))
        scheduleSocketRetry(token)
        return
      }

      system(translate('instance.wsDisconnectedLine'))
    }
  }

  const syncTerminalSocket = async (info?: InstanceInfo, token = requestToken.value) => {
    if (info?.wsPath) {
      await connectTerminalSocket(info.wsPath, token)
      return
    }

    closeTerminalSocket()
    if (info) {
      system(translate('instance.noWsAddress', { entity: options.entityLabel }))
    }
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
        system(translate('instance.missingContextLoad', {
          context: options.contextIdLabel,
          target: options.loadTargetLabel,
        }))
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

      if (!terminalInfo.value) {
        system(translate('instance.loadInfoFailed', { target: options.loadTargetLabel }))
        system(translate('instance.checkBackendRetry'))
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
        feedback.warning(translate('instance.missingContextAction', {
          context: options.contextIdLabel,
          action: getActionLabel(action),
          entity: options.entityLabel,
        }))
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
        feedback.success(translate('instance.actionSuccess', { action: getActionLabel(action) }))
      }
    } finally {
      if (isActiveRequest(token)) {
        pendingAction.value = null
      }
    }
  }

  // —— PTY 输入：与 SSH 终端同一协议 ——
  // sendRaw 把 xterm onData/onBinary 的原始输入原样发给后端；未连接时静默丢弃。
  // Uint8Array 走二进制帧（legacy 鼠标编码等高位字节按 UTF-8 文本发送会被损坏）。
  const sendRaw = (data: string | Uint8Array) => {
    if (!data.length || !terminalSocket || terminalSocket.readyState !== WebSocket.OPEN) return
    terminalSocket.send(data)
  }

  // sendResize 通知后端调整 PTY 窗口尺寸
  const sendResize = (cols: number, rows: number) => {
    if (cols <= 0 || rows <= 0) return
    if (!terminalSocket || terminalSocket.readyState !== WebSocket.OPEN) return
    terminalSocket.send(JSON.stringify({ type: 'resize', cols, rows }))
  }

  const clearScreen = () => {
    rawBuffer = ''
  }

  const resetSession = () => {
    closeTerminalSocket()
    terminalInfo.value = undefined
    socketStatus.value = 'disconnected'
    pendingAction.value = null
    resetOutput()
  }

  const initializeSession = async () => {
    const token = nextRequestToken()
    resetSession()
    await loadTerminalInfo({}, token)
  }

  const refreshInfo = async (loadOptions: LoadInfoOptions = {}) => {
    await loadTerminalInfo(loadOptions)
  }

  const dispose = () => {
    nextRequestToken()
    resetSession()
    outputListeners.clear()
  }

  return {
    pendingAction,
    terminalInfo,
    socketStatus,
    featureItems,
    isBusy,
    runAction,
    initializeSession,
    refreshInfo,
    clearScreen,
    sendRaw,
    sendResize,
    onOutput,
    getBuffer,
    dispose,
  }
}
