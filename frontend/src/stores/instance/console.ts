import type { Ref } from 'vue'
import { defineStore } from 'pinia'
import {
  deleteInstanceLogRequest,
  getInstanceInfoRequest,
  restartInstanceRequest,
  startInstanceRequest,
  stopInstanceRequest,
} from '@/api/instance'
import { createConsoleSession } from '@/stores/instance/consoleSession'
import type {
  ConsoleFeatureItem,
  ConsoleSocketStatus,
} from '@/stores/instance/types'
import { InstanceStatus } from '@/types/v1/instance'
import { useTabsStore } from '@/stores/tabs'
import { translate } from '@/locales'

const DEFAULT_FEATURE_ITEMS: ConsoleFeatureItem[] = [
  {
    key: 'file-manager',
    titleKey: 'instance.fileManagerTitle',
    descriptionKey: 'instance.fileManagerDescription',
    icon: 'HOutline:FolderIcon',
  },
  {
    key: 'event-task',
    titleKey: 'instance.eventTaskTitle',
    descriptionKey: 'instance.eventTaskDescription',
    icon: 'HOutline:ClockIcon',
  },
  {
    key: 'terminal-setting',
    titleKey: 'instance.consoleSettingTitle',
    descriptionKey: 'instance.consoleSettingDescription',
    icon: 'HOutline:AdjustmentsHorizontalIcon',
  },
  {
    key: 'instance-setting',
    titleKey: 'instance.instanceSettingTitle',
    descriptionKey: 'instance.instanceSettingDescription',
    icon: 'HOutline:SquaresPlusIcon',
  },
]

type ConsoleSessionHandle = ReturnType<typeof createConsoleSession>

interface ConsoleSessionEntry {
  instanceId: Ref<string>
  initialized: boolean
  session: ConsoleSessionHandle
}

export const useInstanceConsoleStore = defineStore('instance-console', () => {
  const tabsStore = useTabsStore()
  const activeSessionKey = ref('')
  const sessionEntries = shallowRef<Record<string, ConsoleSessionEntry>>({})

  const createSessionEntry = (): ConsoleSessionEntry => {
    const instanceId = ref('')
    const getCurrentInstanceId = () => instanceId.value.trim()
    const session = createConsoleSession({
      entityLabel: translate('instance.instanceEntity'),
      loadTargetLabel: translate('instance.instanceConsole'),
      featureItems: DEFAULT_FEATURE_ITEMS,
      contextIdLabel: translate('instance.contextInstanceId'),
      getContextId: getCurrentInstanceId,
      extendSocketUrl: (url) => {
        const currentInstanceId = getCurrentInstanceId()
        if (currentInstanceId) {
          url.searchParams.set('instanceID', currentInstanceId)
        }
      },
      getInfo: async () => {
        const { data } = await getInstanceInfoRequest({ id: getCurrentInstanceId() })
        return data.info
      },
    })

    return {
      instanceId,
      initialized: false,
      session,
    }
  }

  const closeSession = (routeKey: string) => {
    const normalizedRouteKey = routeKey.trim()
    const entry = normalizedRouteKey ? sessionEntries.value[normalizedRouteKey] : undefined
    if (!entry) return

    entry.session.dispose()
    const nextEntries = { ...sessionEntries.value }
    delete nextEntries[normalizedRouteKey]
    sessionEntries.value = nextEntries

    if (activeSessionKey.value === normalizedRouteKey) {
      activeSessionKey.value = ''
    }
  }

  const activeEntry = computed(() =>
    activeSessionKey.value ? sessionEntries.value[activeSessionKey.value] : undefined,
  )
  const activeSession = computed(() => activeEntry.value?.session)

  watch(
    () => tabsStore.tabs.map((tab) => tab.fullPath),
    (fullPaths) => {
      const openedTabSet = new Set(fullPaths)

      Object.keys(sessionEntries.value).forEach((routeKey) => {
        if (!openedTabSet.has(routeKey)) {
          closeSession(routeKey)
        }
      })
    },
    { immediate: true },
  )

  const initialize = async (routeKey: string, nextInstanceId: string) => {
    const normalizedRouteKey = routeKey.trim()
    const normalizedInstanceId = nextInstanceId.trim()
    activeSessionKey.value = normalizedRouteKey

    if (!normalizedRouteKey) return

    let entry = sessionEntries.value[normalizedRouteKey]
    if (!entry) {
      entry = createSessionEntry()
      sessionEntries.value = {
        ...sessionEntries.value,
        [normalizedRouteKey]: entry,
      }
    }

    entry.instanceId.value = normalizedInstanceId

    if (entry.initialized) {
      return
    }

    await entry.session.initializeSession()
    entry.initialized = !!normalizedInstanceId
  }

  const refreshCurrentInfo = async () => {
    const session = activeSession.value
    if (!session) return

    await session.refreshInfo({ silent: true, refreshOutput: false })
  }

  const togglePower = async () => {
    const session = activeSession.value
    const currentInstanceId = activeEntry.value?.instanceId.value.trim() || ''
    if (
      !session ||
      session.terminalInfo.value?.status === InstanceStatus.INSTANCE_STATUS_RUNNING ||
      session.pendingAction.value
    ) {
      return
    }

    await session.runAction('start', () => startInstanceRequest({ id: currentInstanceId }))
  }

  const stopConsole = async (force = false) => {
    const session = activeSession.value
    const currentInstanceId = activeEntry.value?.instanceId.value.trim() || ''
    if (
      !session ||
      session.terminalInfo.value?.status === InstanceStatus.INSTANCE_STATUS_STOPPED ||
      session.terminalInfo.value?.status === InstanceStatus.INSTANCE_STATUS_UNSPECIFIED ||
      session.pendingAction.value
    ) {
      return
    }

    await session.runAction('stop', () => stopInstanceRequest({ id: currentInstanceId, force }))
  }

  const togglePowerState = async () => {
    const session = activeSession.value
    if (!session) return

    if (session.terminalInfo.value?.status === InstanceStatus.INSTANCE_STATUS_RUNNING) {
      await stopConsole()
      return
    }

    await togglePower()
  }

  const restartTerminal = async (force = false) => {
    const session = activeSession.value
    const currentInstanceId = activeEntry.value?.instanceId.value.trim() || ''
    if (!session || session.pendingAction.value) return

    await session.runAction('restart', () =>
      restartInstanceRequest({ id: currentInstanceId, force }),
    )
  }

  const forceStopConsole = async () => {
    await stopConsole(true)
  }

  const forceRestartTerminal = async () => {
    await restartTerminal(true)
  }

  const instanceId = computed(() => activeEntry.value?.instanceId.value || '')
  const terminalInfo = computed(() => activeSession.value?.terminalInfo.value)
  const socketStatus = computed<ConsoleSocketStatus>(
    () => activeSession.value?.socketStatus.value || 'disconnected',
  )
  const featureItems = computed<ConsoleFeatureItem[]>(
    () => activeSession.value?.featureItems.value || DEFAULT_FEATURE_ITEMS,
  )
  const isBusy = computed(() => activeSession.value?.isBusy.value ?? false)

  const clearScreen = () => {
    const session = activeSession.value
    const currentInstanceId = activeEntry.value?.instanceId.value.trim() || ''
    if (!session || !currentInstanceId) return

    return deleteInstanceLogRequest({ id: currentInstanceId }).then(() => {
      session.clearScreen()
      feedback.success(translate('instance.consoleCleared'))
    })
  }

  // —— PTY 终端直连（与 SSH 终端同一协议）——
  const sendRaw = (data: string | Uint8Array) => {
    activeSession.value?.sendRaw(data)
  }
  const sendResize = (cols: number, rows: number) => {
    activeSession.value?.sendResize(cols, rows)
  }
  const subscribeOutput = (cb: (chunk: string) => void) =>
    activeSession.value?.onOutput(cb) ?? (() => {})
  const getOutputBuffer = () => activeSession.value?.getBuffer() ?? ''

  return {
    activeSessionKey,
    instanceId,
    terminalInfo,
    socketStatus,
    featureItems,
    isBusy,
    initialize,
    refreshCurrentInfo,
    togglePower: togglePowerState,
    restartTerminal,
    forceStopConsole,
    forceRestartTerminal,
    clearScreen,
    sendRaw,
    sendResize,
    subscribeOutput,
    getOutputBuffer,
  }
})
