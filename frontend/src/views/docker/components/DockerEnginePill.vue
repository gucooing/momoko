<!-- 引擎状态 pill（放在 Docker 各页标题旁）：已连接 / 未连接 / 未启用。 -->
<template>
  <StatusPill v-if="state !== 'unknown'" class="dk-engine-pill" :variant="variant" :label="label" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useDockerStatus } from './useDockerStatus'

const { t } = useI18n()
const { state } = useDockerStatus()
const variant = computed(() =>
  state.value === 'connected' ? 'success' : state.value === 'disconnected' ? 'error' : 'neutral',
)
const label = computed(() =>
  state.value === 'connected'
    ? t('docker.engine.connected')
    : state.value === 'disconnected'
      ? t('docker.common.notConnected')
      : t('docker.common.notEnabled'),
)
</script>

<style scoped lang="scss">
.dk-engine-pill {
  vertical-align: middle;
  margin-left: 12px;
  font-family: inherit;
  letter-spacing: 0;
}
</style>
