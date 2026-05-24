<template>
  <TerminalEmulator
    mode="system-terminal"
    :terminal-name="terminalInfo?.name || '系统终端'"
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

defineOptions({ name: 'SystemTerminalView' })

const route = useRoute()
const terminalStore = useInstanceTerminalStore()

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

const terminalName = computed(() => terminalInfo.value?.name || '系统终端')

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
    title: '确认停止终端',
    content: `确定要停止"${terminalName.value}"吗？`,
    cancelText: '取消',
    confirmText: '确认停止',
    onConfirm: async () => {
      await togglePower()
    },
  })
}

const handleRestart = () => {
  Dialog.confirm({
    title: '确认重启终端',
    content: `确定要重启"${terminalName.value}"吗？`,
    cancelText: '取消',
    confirmText: '确认重启',
    onConfirm: async () => {
      await restartTerminal()
    },
  })
}

const handleFeature = () => {
  ElMessage.info('功能暂未实现')
}
</script>
