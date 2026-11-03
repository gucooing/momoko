<!-- 节点检测结果：状态 + 延迟；在线时展示节点自报信息与 API Key 生效权限，失败时展示原因。 -->
<template>
  <div class="node-test" :class="`is-${tone}`">
    <div class="node-test__head">
      <span class="node-test__av m-sticker" :class="`m-sticker--${sticker}`">
        <component :is="menuStore.iconComponents[online ? 'HOutline:CheckIcon' : 'HOutline:ExclamationTriangleIcon']" />
      </span>
      <StatusPill :variant="tone" :label="stateLabel" />
      <span v-if="status.latencyMs" class="node-test__latency m-num">{{ status.latencyMs }} ms</span>
    </div>
    <p v-if="!online && status.message" class="node-test__message">{{ status.message }}</p>
    <template v-if="online && remote">
      <dl class="node-test__grid">
        <div>
          <dt>{{ t('node.list.hostname') }}</dt>
          <dd>{{ remote.hostname || '—' }}</dd>
        </div>
        <div>
          <dt>{{ t('node.list.version') }}</dt>
          <dd class="m-num">{{ remote.version || '—' }}</dd>
        </div>
        <div class="is-wide">
          <dt>{{ t('node.list.platform') }}</dt>
          <dd>{{ [remote.platform, `${remote.os}/${remote.arch}`].filter(Boolean).join(' · ') }}</dd>
        </div>
        <div>
          <dt>{{ t('node.list.apiKeyOwner') }}</dt>
          <dd>{{ remote.apiKeyName }} · {{ remote.username }}</dd>
        </div>
        <div v-if="remote.expiresAt">
          <dt>{{ t('node.key.expiresAt') }}</dt>
          <dd class="m-num">{{ formatDateTime(remote.expiresAt) }}</dd>
        </div>
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
const menuStore = useMenuStore()

const online = computed(() => props.status.state === NodeState.NodeState_Online)
const remote = computed(() => props.status.remote)
const tone = computed(() => {
  if (props.status.state === NodeState.NodeState_Online) return 'success'
  if (props.status.state === NodeState.NodeState_Unauthorized) return 'warning'
  if (props.status.state === NodeState.NodeState_Offline) return 'error'
  return 'neutral'
})
const sticker = computed(() => (tone.value === 'success' ? 'ok' : tone.value === 'warning' ? 'warn' : tone.value === 'error' ? 'bad' : 'a2'))
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
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
}
.node-test__head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.node-test__av {
  width: 36px;
  height: 36px;
  border-radius: 12px;
  transform: rotate(-4deg);
}
.node-test__av :deep(svg) {
  width: 18px;
  height: 18px;
}
.node-test__latency {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.node-test__message {
  padding: 9px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-badSoft);
  color: var(--m-bad);
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.5;
  word-break: break-word;
}
.node-test.is-warning .node-test__message {
  background: var(--m-warnSoft);
  color: var(--m-warn);
}
.node-test__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 12px;
  margin: 0;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
}
.node-test__grid > div {
  min-width: 0;
}
.node-test__grid .is-wide {
  grid-column: 1 / -1;
}
.node-test__grid dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.node-test__grid dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  word-break: break-all;
}
.node-test__perms {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.node-test__perms-title {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
</style>
