<!-- 实例控制台（重写 · 真 PTY 终端）：调用公共组件 TerminalConsole，与 SSH 终端同一组件、同一协议。
     本页只负责实例传输层（consoleSession 的 ws + 回放缓冲）与实例动作（启停/重启/强制/清屏/功能项
     + InstanceEditor 受控弹窗契约与二次确认）。键盘/鼠标输入 @data 原样发往后端 PTY。 -->
<template>
  <TerminalConsole
    ref="term"
    :title="terminalInfo?.name || t('instance.currentInstance')"
    :subtitle="terminalInfo?.type || ''"
    :tag="statusTag"
    :status="socketStatus"
    :status-label="statusLabel"
    :actions="actions"
    :busy="isBusy"
    @data="sendRaw"
    @resize="({ cols, rows }) => sendResize(cols, rows)"
    @action="onAction"
  />

  <InstanceEditor
    :visible="instanceEditorVisible"
    :mode="instanceEditorMode"
    :loading="instanceEditorLoading"
    :submitting="instanceEditorSubmitting"
    :form="instanceEditorForm"
    :type-options="typeOptions"
    @close="handleEditorClose"
    @submit="handleEditorSubmit"
  />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useInstanceConsoleStore } from '@/stores/instance/console'
import { useInstanceListStore } from '@/stores/instance/list'
import { InstanceStatus, type InstanceEditorFormValue } from '@/stores/instance/types'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import type { TerminalAction } from '@/components/terminal/TerminalShell.vue'
import TerminalConsole from '@/components/terminal/TerminalConsole.vue'
import InstanceEditor from '@/views/instance/list/instanceEditor.vue'

defineOptions({ name: 'InstanceConsoleView' })

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const instanceConsoleStore = useInstanceConsoleStore()
const instanceListStore = useInstanceListStore()

const { terminalInfo, socketStatus, isBusy, featureItems } = storeToRefs(instanceConsoleStore)

const {
  instanceEditorVisible,
  instanceEditorMode,
  instanceEditorLoading,
  instanceEditorSubmitting,
  instanceEditorForm,
  typeOptions,
} = storeToRefs(instanceListStore)

const {
  initialize,
  refreshCurrentInfo,
  togglePower,
  restartTerminal,
  forceStopConsole,
  forceRestartTerminal,
  clearScreen,
  sendRaw,
  sendResize,
  subscribeOutput,
  getOutputBuffer,
} = instanceConsoleStore

const { getInstanceTypeList, openEditEditor, closeInstanceEditor, submitInstanceEditor } =
  instanceListStore

const term = ref<InstanceType<typeof TerminalConsole>>()

// —— 头部：连接状态标签 + 实例运行态 chip ——
const statusLabel = computed(() => {
  switch (socketStatus.value) {
    case 'connected':
      return t('instance.connected')
    case 'connecting':
      return t('instance.connecting')
    case 'error':
      return t('instance.connectFailed')
    default:
      return t('instance.disconnected')
  }
})

const statusTag = computed(() => {
  switch (terminalInfo.value?.status) {
    case InstanceStatus.INSTANCE_STATUS_RUNNING:
      return { label: t('instance.running'), tone: 'success' as const }
    case InstanceStatus.INSTANCE_STATUS_STOPPED:
      return { label: t('instance.stopped'), tone: 'neutral' as const }
    case InstanceStatus.INSTANCE_STATUS_MAINTENANCE:
      return { label: t('instance.maintenance'), tone: 'warning' as const }
    default:
      return undefined
  }
})

// —— 工具条动作：启停/重启/强制/清屏 + store 功能项 ——
const actions = computed<TerminalAction[]>(() => {
  const running = terminalInfo.value?.status === InstanceStatus.INSTANCE_STATUS_RUNNING
  return [
    {
      key: running ? 'stop' : 'start',
      icon: running ? 'HOutline:StopIcon' : 'HOutline:PlayIcon',
      label: running ? t('instance.stop') : t('instance.start'),
      tone: running ? 'danger' : 'primary',
    },
    { key: 'restart', icon: 'HOutline:ArrowPathIcon', label: t('instance.restart') },
    { key: 'forceStop', icon: 'HOutline:NoSymbolIcon', label: t('instance.forceStop'), tone: 'danger' },
    { key: 'forceRestart', icon: 'HOutline:BoltIcon', label: t('instance.forceRestart') },
    { key: 'clear', icon: 'HOutline:SparklesIcon', label: t('instance.clearScreen') },
    ...featureItems.value.map((f) => ({
      key: f.key,
      icon: f.icon,
      label: f.titleKey ? t(f.titleKey) : f.title || f.key,
    })),
  ]
})

const onAction = (key: string) => {
  switch (key) {
    case 'start':
    case 'stop':
      handleTogglePower()
      break
    case 'restart':
      handleRestart()
      break
    case 'forceStop':
      handleForceStop()
      break
    case 'forceRestart':
      handleForceRestart()
      break
    case 'clear':
      handleClear()
      break
    default:
      handleFeature(key)
  }
}

// —— 输出流订阅 ——
let unsubscribe: (() => void) | null = null

onMounted(() => {
  const buffered = getOutputBuffer()
  if (buffered) term.value?.write(buffered)
  unsubscribe = subscribeOutput((chunk) => term.value?.write(chunk))
  term.value?.focus()
})

// 连接成功：后端会重放最近日志，先 reset 画面避免重复；
// 同步一次当前窗口尺寸并聚焦，直接进入可键入状态
watch(socketStatus, (status, prev) => {
  if (status === prev) return
  if (status === 'connected') {
    const terminal = term.value
    if (!terminal) return
    terminal.reset()
    if (terminal.cols > 0 && terminal.rows > 0) sendResize(terminal.cols, terminal.rows)
    terminal.focus()
  }
})

// —— 路由驱动会话初始化（保留原语义）——
const getRouteInstanceId = () =>
  typeof route.params.instanceId === 'string' ? route.params.instanceId : ''

watch(
  () => [route.fullPath, route.params.instanceId],
  ([fullPath, rawInstanceId]) => {
    void initialize(
      typeof fullPath === 'string' ? fullPath : '',
      typeof rawInstanceId === 'string' ? rawInstanceId : '',
    )
  },
  { immediate: true },
)

const instanceName = computed(() => terminalInfo.value?.name || t('instance.currentInstance'))

// —— 清屏：清后端日志 + 会话缓冲 + xterm 画面 ——
const handleClear = async () => {
  await clearScreen()
  term.value?.reset()
}

// —— 导航 / 功能项 ——
const openFileManager = () => {
  const instanceId = getRouteInstanceId()
  if (!instanceId) {
    ElMessage.warning(t('instance.missingInstanceIdFile'))
    return
  }
  router.push({
    path: `/instance/files/${instanceId}`,
    query: {
      tabTitle: t('instance.instanceFileTab', {
        name: terminalInfo.value?.name || t('instance.instanceEntity'),
      }),
      from: 'instance',
      status: terminalInfo.value?.status,
      workdir: terminalInfo.value?.instancePath,
    },
  })
}

const openInstanceEditor = async () => {
  const instanceId = getRouteInstanceId()
  if (!instanceId) {
    ElMessage.warning(t('instance.missingInstanceIdSetting'))
    return
  }
  try {
    await getInstanceTypeList()
    await openEditEditor(instanceId)
  } catch (error) {
    showRequestError(error, t('instance.loadConfigFailed'))
  }
}

const handleFeature = (key: string) => {
  if (key === 'file-manager') {
    openFileManager()
    return
  }
  if (key === 'instance-setting') {
    void openInstanceEditor()
    return
  }
  ElMessage.info(t('instance.featureNotImplemented'))
}

const handleEditorClose = () => closeInstanceEditor()

const handleEditorSubmit = async (form: InstanceEditorFormValue) => {
  const instanceId = getRouteInstanceId()
  try {
    instanceEditorForm.value = { ...form }
    await submitInstanceEditor()
    ElMessage.success(t('instance.configSaveSuccess'))
    if (instanceId) await refreshCurrentInfo()
  } catch (error) {
    showRequestError(error, t('instance.configSaveFailed'))
  }
}

// —— 启停/重启/强制（保留二次确认）——
const handleTogglePower = () => {
  if (terminalInfo.value?.status !== InstanceStatus.INSTANCE_STATUS_RUNNING) {
    void togglePower()
    return
  }
  Dialog.confirm({
    title: t('instance.confirmStopInstanceTitle'),
    content: t('instance.confirmStopInstanceContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmStop'),
    onConfirm: async () => {
      await togglePower()
    },
  })
}

const handleRestart = () => {
  Dialog.confirm({
    title: t('instance.confirmRestartInstanceTitle'),
    content: t('instance.confirmRestartInstanceContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmRestart'),
    onConfirm: async () => {
      await restartTerminal()
    },
  })
}

const handleForceStop = () => {
  Dialog.confirm({
    title: t('instance.confirmForceStopInstanceTitle'),
    content: t('instance.confirmForceStopInstanceContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmForceStop'),
    onConfirm: async () => {
      await forceStopConsole()
    },
  })
}

const handleForceRestart = () => {
  Dialog.confirm({
    title: t('instance.confirmForceRestartInstanceTitle'),
    content: t('instance.confirmForceRestartInstanceContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmForceRestart'),
    onConfirm: async () => {
      await forceRestartTerminal()
    },
  })
}

onBeforeUnmount(() => {
  unsubscribe?.()
  closeInstanceEditor()
})
</script>
