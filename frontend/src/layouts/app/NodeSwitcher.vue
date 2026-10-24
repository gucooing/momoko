<!-- 节点切换器：显示当前操作的节点（本机 / 远程）与在线状态，下拉切换。切换会整页重载（见 stores/node）。
     仅在有 node:view 权限且已接入节点（或当前处于远程节点）时显示。 -->
<template>
  <AppDropdown v-if="visible" align="end" :width="320">
    <template #trigger="{ open }">
      <button
        type="button"
        class="node-switch"
        :class="{ 'is-remote': nodeStore.isRemote }"
        :title="t('node.switcher.tooltip')"
        @click="!open && onOpen()"
      >
        <UIcon :name="nodeStore.isRemote ? 'i-lucide-server' : 'i-lucide-monitor'" class="node-switch__icon" />
        <span class="node-switch__label">{{ currentLabel }}</span>
        <span class="node-switch__dot" :style="{ background: currentTone }" />
        <UIcon name="i-lucide-chevrons-up-down" class="node-switch__caret" />
      </button>
    </template>
    <template #default="{ close }">
      <div class="node-menu">
        <div class="node-menu__head">
          <span>{{ t('node.switcher.title') }}</span>
          <button
            type="button"
            class="node-menu__refresh"
            :title="t('node.switcher.refresh')"
            @click="nodeStore.loadNodes()"
          >
            <UIcon name="i-lucide-refresh-cw" :class="{ 'is-spinning': nodeStore.nodesLoading }" />
          </button>
        </div>

        <button
          type="button"
          class="node-row"
          :class="{ 'is-active': !nodeStore.isRemote }"
          @click="pick('', close)"
        >
          <UIcon name="i-lucide-monitor" class="node-row__icon" />
          <span class="node-row__body">
            <span class="node-row__name">{{ t('node.switcher.local') }}</span>
            <span class="node-row__meta">{{ t('node.switcher.localDesc') }}</span>
          </span>
          <UIcon v-if="!nodeStore.isRemote" name="i-lucide-check" class="node-row__check" />
        </button>

        <div class="node-menu__list">
          <button
            v-for="node in nodeStore.nodes"
            :key="node.id"
            type="button"
            class="node-row"
            :class="{ 'is-active': node.id === nodeStore.nodeId }"
            @click="pick(node.id, close)"
          >
            <UIcon name="i-lucide-server" class="node-row__icon" />
            <span class="node-row__body">
              <span class="node-row__name">{{ node.name }}</span>
              <span class="node-row__meta">{{ node.address }}</span>
            </span>
            <StatusPill
              :variant="stateVariant(node.status?.state)"
              :label="stateLabel(node.status?.state)"
            />
            <UIcon v-if="node.id === nodeStore.nodeId" name="i-lucide-check" class="node-row__check" />
          </button>
          <p v-if="!nodeStore.nodes.length" class="node-menu__empty">{{ t('node.switcher.empty') }}</p>
        </div>

        <button v-if="canViewNodes" type="button" class="node-menu__foot" @click="goManage(close)">
          <UIcon name="i-lucide-settings-2" />
          <span>{{ t('node.switcher.manage') }}</span>
        </button>
      </div>
    </template>
  </AppDropdown>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NodeState } from '@/types/v1/node'

const nodeStore = useNodeStore()
const menuStore = useMenuStore()
const router = useRouter()
const { t } = useI18n()

const canViewNodes = computed(() => menuStore.hasButtonPermission('node:view'))
const visible = computed(
  () => nodeStore.isRemote || (canViewNodes.value && nodeStore.nodes.length > 0),
)

const STATE_TONE: Record<string, string> = {
  [NodeState.NodeState_Online]: 'var(--el-color-success, #16a34a)',
  [NodeState.NodeState_Offline]: 'var(--el-color-danger, #ef4444)',
  [NodeState.NodeState_Unauthorized]: 'var(--el-color-warning, #f59e0b)',
}

const currentLabel = computed(() =>
  nodeStore.isRemote
    ? nodeStore.contextInfo?.name || t('node.switcher.remote')
    : t('node.switcher.local'),
)
const currentTone = computed(() => {
  if (!nodeStore.isRemote) return 'var(--el-color-success, #16a34a)'
  return STATE_TONE[nodeStore.contextInfo?.status?.state || ''] || 'var(--el-text-color-placeholder)'
})

const stateVariant = (state?: NodeState) => {
  if (state === NodeState.NodeState_Online) return 'success'
  if (state === NodeState.NodeState_Offline) return 'error'
  if (state === NodeState.NodeState_Unauthorized) return 'warning'
  return 'neutral'
}
const stateLabel = (state?: NodeState) => {
  if (state === NodeState.NodeState_Online) return t('node.state.online')
  if (state === NodeState.NodeState_Offline) return t('node.state.offline')
  if (state === NodeState.NodeState_Unauthorized) return t('node.state.unauthorized')
  return t('node.state.unknown')
}

const onOpen = () => {
  if (canViewNodes.value) void nodeStore.loadNodes()
}

const pick = (id: string, close: () => void) => {
  close()
  nodeStore.switchNode(id)
}

const goManage = (close: () => void) => {
  close()
  void router.push('/node/list')
}

onMounted(() => {
  if (canViewNodes.value) void nodeStore.loadNodes()
})
</script>

<style scoped lang="scss">
.node-switch {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  max-width: 220px;
  padding: 0 10px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-regular);
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}
.node-switch:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.node-switch.is-remote {
  border-color: color-mix(in srgb, var(--el-color-primary) 45%, transparent);
  background: color-mix(in srgb, var(--el-color-primary) 8%, transparent);
  color: var(--el-color-primary);
}
.node-switch__icon,
.node-switch__caret {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
.node-switch__caret {
  opacity: 0.6;
}
.node-switch__label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.8125rem;
  font-weight: 600;
}
.node-switch__dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  flex-shrink: 0;
}

.node-menu {
  display: flex;
  flex-direction: column;
  padding: 6px;
}
.node-menu__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 8px 6px;
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: var(--el-text-color-secondary);
}
.node-menu__refresh {
  display: inline-flex;
  padding: 4px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: inherit;
  cursor: pointer;
}
.node-menu__refresh:hover {
  background: var(--el-fill-color-light);
}
.is-spinning {
  animation: node-spin 0.9s linear infinite;
}
@keyframes node-spin {
  to {
    transform: rotate(360deg);
  }
}
.node-menu__list {
  max-height: 320px;
  overflow-y: auto;
}
.node-menu__empty {
  padding: 10px 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-placeholder);
}
.node-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-regular);
  text-align: left;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.node-row:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.node-row.is-active {
  color: var(--el-color-primary);
}
.node-row__icon,
.node-row__check {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
.node-row__body {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}
.node-row__name {
  font-size: 0.875rem;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-row__meta {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-menu__foot {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  padding: 8px;
  border: none;
  border-top: 1px solid var(--el-border-color-lighter);
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  cursor: pointer;
}
.node-menu__foot:hover {
  color: var(--el-color-primary);
}
@media (width <= 768px) {
  .node-switch__label,
  .node-switch__caret {
    display: none;
  }
}
</style>
