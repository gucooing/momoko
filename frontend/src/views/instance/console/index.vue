<template>
  <TerminalEmulator
    mode="instance-console"
    :terminal-name="terminalInfo?.name || '当前实例'"
    :terminal-type="terminalInfo?.type || ''"
    :terminal-id="terminalInfo?.id || ''"
    :instance-status="terminalInfo?.status || ''"
    :output-lines="outputLines"
    :socket-status="socketStatus"
    :is-busy="isBusy"
    :can-send-command="canSendCommand"
    :send-placeholder="sendPlaceholder"
    :feature-items="featureItems"
    @input="onInput"
    @toggle-power="handleTogglePower"
    @restart="handleRestart"
    @force-stop="handleForceStop"
    @force-restart="handleForceRestart"
    @clear="clearScreen"
    @feature="handleFeature"
    @reconnect="handleRestart"
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
import { storeToRefs } from 'pinia'
import TerminalEmulator from '@/components/terminal/TerminalEmulator.vue'
import { useInstanceConsoleStore } from '@/stores/instance/console'
import { useInstanceListStore } from '@/stores/instance/list'
import { InstanceStatus, type InstanceEditorFormValue } from '@/stores/instance/types'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import InstanceEditor from '@/views/instance/list/instanceEditor.vue'

defineOptions({ name: 'InstanceConsoleView' })

const route = useRoute()
const router = useRouter()

const instanceConsoleStore = useInstanceConsoleStore()
const instanceListStore = useInstanceListStore()

const {
  commandValue,
  terminalInfo,
  socketStatus,
  outputLines,
  featureItems,
  canSendCommand,
  isBusy,
  sendPlaceholder,
} = storeToRefs(instanceConsoleStore)

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
  executeCommand,
} = instanceConsoleStore

const {
  getInstanceTypeList,
  openEditEditor,
  closeInstanceEditor,
  submitInstanceEditor,
} = instanceListStore

const getRouteInstanceId = () => {
  return typeof route.params.instanceId === 'string' ? route.params.instanceId : ''
}

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

const instanceName = computed(() => terminalInfo.value?.name || '当前实例')

const onInput = (text: string) => {
  commandValue.value = text
  executeCommand()
}

const openFileManager = () => {
  const instanceId = getRouteInstanceId()
  if (!instanceId) {
    ElMessage.warning('缺少实例ID，无法打开文件管理')
    return
  }
  router.push({
    path: `/instance/files/${instanceId}`,
    query: {
      tabTitle: `${terminalInfo.value?.name || '实例'} 文件`,
      from: 'instance',
      status: terminalInfo.value?.status,
      workdir: terminalInfo.value?.instancePath,
    },
  })
}

const openInstanceEditor = async () => {
  const instanceId = getRouteInstanceId()
  if (!instanceId) {
    ElMessage.warning('缺少实例ID，无法打开实例设置')
    return
  }
  try {
    await getInstanceTypeList()
    await openEditEditor(instanceId)
  } catch (error) {
    showRequestError(error, '加载实例配置失败')
  }
}

const handleEditorClose = () => closeInstanceEditor()

const handleEditorSubmit = async (form: InstanceEditorFormValue) => {
  const instanceId = getRouteInstanceId()
  try {
    instanceEditorForm.value = { ...form }
    await submitInstanceEditor()
    ElMessage.success('实例配置保存成功')
    if (instanceId) await refreshCurrentInfo()
  } catch (error) {
    showRequestError(error, '实例配置保存失败')
  }
}

const handleTogglePower = () => {
  if (terminalInfo.value?.status !== InstanceStatus.INSTANCE_STATUS_RUNNING) {
    void togglePower()
    return
  }
  Dialog.confirm({
    title: '确认停止实例',
    content: `确定要停止实例"${instanceName.value}"吗？`,
    cancelText: '取消',
    confirmText: '确认停止',
    onConfirm: async () => { await togglePower() },
  })
}

const handleRestart = () => {
  Dialog.confirm({
    title: '确认重启实例',
    content: `确定要重启实例"${instanceName.value}"吗？`,
    cancelText: '取消',
    confirmText: '确认重启',
    onConfirm: async () => { await restartTerminal() },
  })
}

const handleForceStop = () => {
  Dialog.confirm({
    title: '确认强制停止实例',
    content: `确定要强制停止实例"${instanceName.value}"吗？`,
    cancelText: '取消',
    confirmText: '确认强制停止',
    onConfirm: async () => { await forceStopConsole() },
  })
}

const handleForceRestart = () => {
  Dialog.confirm({
    title: '确认强制重启实例',
    content: `确定要强制重启实例"${instanceName.value}"吗？`,
    cancelText: '取消',
    confirmText: '确认强制重启',
    onConfirm: async () => { await forceRestartTerminal() },
  })
}

const handleFeature = (key: string) => {
  if (key === 'file-manager') { openFileManager(); return }
  if (key === 'instance-setting') { void openInstanceEditor(); return }
  ElMessage.info('功能暂未实现')
}

onBeforeUnmount(() => closeInstanceEditor())
</script>
