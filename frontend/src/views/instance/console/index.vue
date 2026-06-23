<template>
  <TerminalEmulator
    mode="instance-console"
    :terminal-name="terminalInfo?.name || t('instance.currentInstance')"
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
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'InstanceConsoleView' })

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

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

const instanceName = computed(() => terminalInfo.value?.name || t('instance.currentInstance'))

const onInput = (text: string) => {
  commandValue.value = text
  executeCommand()
}

const openFileManager = () => {
  const instanceId = getRouteInstanceId()
  if (!instanceId) {
    ElMessage.warning(t('instance.missingInstanceIdFile'))
    return
  }
  router.push({
    path: `/instance/files/${instanceId}`,
    query: {
      tabTitle: t('instance.instanceFileTab', { name: terminalInfo.value?.name || t('instance.instanceEntity') }),
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
    onConfirm: async () => { await togglePower() },
  })
}

const handleRestart = () => {
  Dialog.confirm({
    title: t('instance.confirmRestartInstanceTitle'),
    content: t('instance.confirmRestartInstanceContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmRestart'),
    onConfirm: async () => { await restartTerminal() },
  })
}

const handleForceStop = () => {
  Dialog.confirm({
    title: t('instance.confirmForceStopInstanceTitle'),
    content: t('instance.confirmForceStopInstanceContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmForceStop'),
    onConfirm: async () => { await forceStopConsole() },
  })
}

const handleForceRestart = () => {
  Dialog.confirm({
    title: t('instance.confirmForceRestartInstanceTitle'),
    content: t('instance.confirmForceRestartInstanceContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmForceRestart'),
    onConfirm: async () => { await forceRestartTerminal() },
  })
}

const handleFeature = (key: string) => {
  if (key === 'file-manager') { openFileManager(); return }
  if (key === 'instance-setting') { void openInstanceEditor(); return }
  ElMessage.info(t('instance.featureNotImplemented'))
}

onBeforeUnmount(() => closeInstanceEditor())
</script>
