<template>
  <TerminalEmulator
    mode="system-terminal"
    :terminal-name="terminalInfo?.name || t('instance.systemTerminal')"
    :terminal-type="terminalInfo?.type || 'system'"
    :terminal-id="terminalInfo?.id || ''"
    :instance-status="terminalInfo?.status || ''"
    :output-lines="outputLines"
    :socket-status="socketStatus"
    :is-busy="isBusy"
    :can-send-command="canSendCommand"
    :send-placeholder="sendPlaceholder"
    @input="onInput"
    @toggle-power="handleTogglePower"
    @restart="handleRestart"
    @clear="clearScreen"
    @feature="handleFeature"
    @reconnect="handleRestart"
  />
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import TerminalEmulator from '@/components/terminal/TerminalEmulator.vue'
import { useInstanceTerminalStore } from '@/stores/instance/terminal'
import { InstanceStatus } from '@/types/v1/instance'
import { Dialog } from '@/utils/dialog'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'SystemTerminalView' })

const route = useRoute()
const terminalStore = useInstanceTerminalStore()
const { t } = useI18n()

const {
  commandValue,
  terminalInfo,
  socketStatus,
  outputLines,
  canSendCommand,
  isBusy,
  sendPlaceholder,
} = storeToRefs(terminalStore)

const {
  initialize,
  togglePower,
  restartTerminal,
  clearScreen,
  executeCommand,
} = terminalStore

watch(
  () => route.fullPath,
  (fullPath) => {
    void initialize(typeof fullPath === 'string' ? fullPath : '')
  },
  { immediate: true },
)

const terminalName = computed(() => terminalInfo.value?.name || t('instance.systemTerminal'))

const onInput = (text: string) => {
  commandValue.value = text
  executeCommand()
}

const handleTogglePower = () => {
  if (terminalInfo.value?.status !== InstanceStatus.INSTANCE_STATUS_RUNNING) {
    void togglePower()
    return
  }

  Dialog.confirm({
    title: t('instance.confirmStopTerminalTitle'),
    content: t('instance.confirmStopTerminalContent', { name: terminalName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmStop'),
    onConfirm: async () => {
      await togglePower()
    },
  })
}

const handleRestart = () => {
  Dialog.confirm({
    title: t('instance.confirmRestartTerminalTitle'),
    content: t('instance.confirmRestartTerminalContent', { name: terminalName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmRestart'),
    onConfirm: async () => {
      await restartTerminal()
    },
  })
}

const handleFeature = () => {
  ElMessage.info(t('instance.featureNotImplemented'))
}
</script>
