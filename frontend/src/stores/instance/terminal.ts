import { defineStore } from 'pinia'
import {
  getTerminalInfoRequest,
  restartTerminalRequest,
  startTerminalRequest,
  stopTerminalRequest,
} from '@/api/instance'
import { createConsoleSession } from '@/stores/instance/consoleSession'
import type { ConsoleFeatureItem } from '@/stores/instance/types'
import { useTabsStore } from '@/stores/tabs'
import { InstanceStatus } from '@/types/v1/instance'

const DEFAULT_OUTPUT_LINES = ['[system] 正在获取终端信息...']

const DEFAULT_FEATURE_ITEMS: ConsoleFeatureItem[] = [
  {
    key: 'terminal-setting',
    title: '终端设置',
    description: '管理终端连接参数',
    icon: 'HOutline:AdjustmentsHorizontalIcon',
  },
]

export const useInstanceTerminalStore = defineStore('instance-terminal', () => {
  const tabsStore = useTabsStore()
  const activeRouteKey = ref('')
  const initialized = ref(false)

  const session = createConsoleSession({
    defaultOutputLines: DEFAULT_OUTPUT_LINES,
    entityLabel: '终端',
    loadTargetLabel: '终端',
    outputLabel: '终端',
    featureItems: DEFAULT_FEATURE_ITEMS,
    getInfo: async () => {
      const { data } = await getTerminalInfoRequest({})
      return data.info
    },
  })

  watch(
    () => tabsStore.tabs.map((tab) => tab.fullPath),
    (fullPaths) => {
      if (!activeRouteKey.value) return

      if (fullPaths.includes(activeRouteKey.value)) {
        return
      }

      initialized.value = false
      activeRouteKey.value = ''
      session.dispose()
    },
    { immediate: true },
  )

  const initialize = async (routeKey: string) => {
    activeRouteKey.value = routeKey.trim()

    if (initialized.value) {
      return
    }

    await session.initializeSession()
    initialized.value = true
  }

  const startTerminal = async () => {
    if (
      session.terminalInfo.value?.status === InstanceStatus.INSTANCE_STATUS_RUNNING ||
      session.pendingAction.value
    ) {
      return
    }

    await session.runAction('start', () => startTerminalRequest({}))
  }

  const stopTerminal = async () => {
    if (
      session.terminalInfo.value?.status === InstanceStatus.INSTANCE_STATUS_STOPPED ||
      session.terminalInfo.value?.status === InstanceStatus.INSTANCE_STATUS_UNSPECIFIED ||
      session.pendingAction.value
    ) {
      return
    }

    await session.runAction('stop', () => stopTerminalRequest({}))
  }

  const togglePower = async () => {
    if (session.terminalInfo.value?.status === InstanceStatus.INSTANCE_STATUS_RUNNING) {
      await stopTerminal()
      return
    }

    await startTerminal()
  }

  const restartTerminal = async () => {
    if (session.pendingAction.value) return

    await session.runAction('restart', () => restartTerminalRequest({}))
  }

  return {
    terminalInfo: session.terminalInfo,
    socketStatus: session.socketStatus,
    commandValue: session.commandValue,
    outputLines: session.outputLines,
    featureItems: session.featureItems,
    canSendCommand: session.canSendCommand,
    isBusy: session.isBusy,
    sendPlaceholder: session.sendPlaceholder,
    initialize,
    togglePower,
    restartTerminal,
    clearScreen: session.clearScreen,
    executeCommand: session.executeCommand,
    selectPrevCommand: session.selectPrevCommand,
    selectNextCommand: session.selectNextCommand,
  }
})
