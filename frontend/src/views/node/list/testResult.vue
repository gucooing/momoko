<!-- 节点检测结果：状态 + 延迟；在线时展示节点自报信息与 API Key 生效权限，失败时展示原因。 -->
<template>
  <div class="node-test" :class="`is-${tone}`">
    <div class="node-test__head">
      <StatusPill :variant="tone" :label="stateLabel" />
      <span v-if="status.latencyMs" class="node-test__latency">{{ status.latencyMs }} ms</span>
    </div>
    <p v-if="!online && status.message" class="node-test__message">{{ status.message }}</p>
    <template v-if="online && remote">
      <dl class="node-test__grid">
        <dt>{{ t('node.list.hostname') }}</dt>
        <dd>{{ remote.hostname || '—' }}</dd>
        <dt>{{ t('node.list.version') }}</dt>
        <dd>{{ remote.version || '—' }}</dd>
        <dt>{{ t('node.list.platform') }}</dt>
        <dd>{{ [remote.platform, `${remote.os}/${remote.arch}`].filter(Boolean).join(' · ') }}</dd>
        <dt>{{ t('node.list.apiKeyOwner') }}</dt>
        <dd>{{ remote.apiKeyName }} · {{ remote.username }}</dd>
        <template v-if="remote.expiresAt">
          <dt>{{ t('node.key.expiresAt') }}</dt>
          <dd>{{ formatDateTime(remote.expiresAt) }}</dd>
        </template>
      </dl>
      <div class="node-test__perms">
        <span class="node-test__perms-title">{{ t('node.list.grantedPermissions') }}</span>
        <NodePermissionChips :permissions="remote.permissions" />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NodeState, type NodeStatus } from '@/types/v1/node'
import { formatDateTime } from '@/utils/file'
import NodePermissionChips from '@/views/node/components/permissionChips.vue'

const props = defineProps<{ status: NodeStatus }>()
const { t } = useI18n()

const online = computed(() => props.status.state === NodeState.NodeState_Online)
const remote = computed(() => props.status.remote)
const tone = computed(() => {
  if (props.status.state === NodeState.NodeState_Online) return 'success'
  if (props.status.state === NodeState.NodeState_Unauthorized) return 'warning'
  if (props.status.state === NodeState.NodeState_Offline) return 'error'
  return 'neutral'
})
const stateLabel = computed(() => {
  if (props.status.state === NodeState.NodeState_Online) return t('node.state.online')
  if (props.status.state === NodeState.NodeState_Unauthorized) return t('node.state.unauthorized')
  if (props.status.state === NodeState.NodeState_Offline) return t('node.state.offline')
  return t('node.state.unknown')
})
</script>

<style scoped lang="scss">
.node-test {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-fill-color-lighter, var(--el-fill-color-light));
}
.node-test__head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.node-test__latency {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.node-test__message {
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--el-text-color-regular);
  word-break: break-word;
}
.node-test__grid {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 4px 14px;
  font-size: 0.8125rem;
}
.node-test__grid dt {
  color: var(--el-text-color-secondary);
}
.node-test__grid dd {
  margin: 0;
  color: var(--el-text-color-primary);
  word-break: break-all;
}
.node-test__perms {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.node-test__perms-title {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
</style>
