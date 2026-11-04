<!-- 节点切换器：显示当前操作的节点（本机 / 远程）、在线状态与主机名，下拉切换。切换会整页重载（见 stores/node）。
     variant='sidebar'：侧栏卡片（虚线框 + 贴纸）；'pill'：手机工作台的虚线胶囊；'hero'：主色卡片上的半透明胶囊。
     没有可切换的节点时只展示当前机器，不弹出菜单。 -->
<template>
  <AppDropdown v-if="canSwitch" align="start" :width="320" :block="variant === 'sidebar'">
    <template #trigger="{ open }">
      <button
        type="button"
        :class="triggerClass"
        :title="t('node.switcher.tooltip')"
        :aria-label="`${t('node.switcher.tooltip')}：${stateLine} · ${nameLine}`"
        :aria-expanded="open"
        @click="!open && onOpen()"
      >
        <template v-if="variant === 'sidebar'">
          <span class="ns-card__sk m-sticker m-sticker--a2" aria-hidden="true">
            <component :is="ic('HOutline:ServerIcon')" />
          </span>
          <span v-if="!collapsed" class="ns-card__text">
            <span class="ns-card__state" :style="{ color: tone }">● {{ stateLine }}</span>
            <span class="ns-card__name">{{ nameLine }}</span>
          </span>
          <component :is="ic('HOutline:ChevronUpDownIcon')" v-if="!collapsed" class="ns-card__caret" />
        </template>
        <template v-else>
          <span class="ns-pill__dot" :style="{ background: variant === 'hero' ? 'var(--m-a)' : tone }" />
          <span class="ns-pill__label">{{ pillLabel }}</span>
          <component :is="ic('HOutline:ChevronUpDownIcon')" class="ns-pill__caret" />
        </template>
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
            :aria-label="t('node.switcher.refresh')"
            @click="nodeStore.loadNodes()"
          >
            <UIcon name="i-lucide-refresh-cw" :class="{ 'is-spinning': nodeStore.nodesLoading }" />
          </button>
        </div>

        <button
          type="button"
          class="node-row"
          :class="{ 'is-active': !nodeStore.isRemote }"
          :aria-pressed="!nodeStore.isRemote"
          @click="pick('', close)"
        >
          <span class="node-row__sk m-sticker m-sticker--a2" aria-hidden="true">
            <component :is="ic('HOutline:ComputerDesktopIcon')" />
          </span>
          <span class="node-row__body">
            <span class="node-row__name">{{ localHostname || t('node.switcher.local') }}</span>
            <span class="node-row__meta" style="color: var(--m-ok)">
              ● {{ t('node.switcher.local') }} · {{ t('node.state.online') }}
            </span>
          </span>
          <component :is="ic('HOutline:CheckIcon')" v-if="!nodeStore.isRemote" class="node-row__check" />
        </button>

        <div class="node-menu__list">
          <button
            v-for="node in nodeStore.nodes"
            :key="node.id"
            type="button"
            class="node-row"
            :class="{ 'is-active': node.id === nodeStore.nodeId }"
            :aria-pressed="node.id === nodeStore.nodeId"
            @click="pick(node.id, close)"
          >
            <span class="node-row__sk m-sticker m-sticker--a2" aria-hidden="true">
              <component :is="ic('HOutline:ServerIcon')" />
            </span>
            <span class="node-row__body">
              <span class="node-row__name">{{ node.name }}</span>
              <span class="node-row__meta" :style="{ color: stateTone(node.status?.state) }">
                ● {{ t('node.switcher.remote') }} · {{ nodeStateText(node) }}
              </span>
            </span>
            <component :is="ic('HOutline:CheckIcon')" v-if="node.id === nodeStore.nodeId" class="node-row__check" />
          </button>
          <p v-if="!nodeStore.nodes.length" class="node-menu__empty">{{ t('node.switcher.empty') }}</p>
        </div>

        <button v-if="canViewNodes" type="button" class="node-menu__foot" @click="goManage(close)">
          {{ t('shell.node.manage') }}
        </button>
      </div>
    </template>
  </AppDropdown>

  <!-- 不可切换：只展示当前机器 -->
  <div v-else :class="triggerClass" class="is-static">
    <template v-if="variant === 'sidebar'">
      <span class="ns-card__sk m-sticker m-sticker--a2" aria-hidden="true">
        <component :is="ic('HOutline:ServerIcon')" />
      </span>
      <span v-if="!collapsed" class="ns-card__text">
        <span class="ns-card__state" :style="{ color: tone }">● {{ stateLine }}</span>
        <span class="ns-card__name">{{ nameLine }}</span>
      </span>
    </template>
    <template v-else>
      <span class="ns-pill__dot" :style="{ background: variant === 'hero' ? 'var(--m-a)' : tone }" />
      <span class="ns-pill__label">{{ pillLabel }}</span>
    </template>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NodeState, type NodeInfo } from '@/types/v1/node'
import { useCurrentHostname } from './useLocalHost'

const props = withDefaults(
  defineProps<{ variant?: 'sidebar' | 'pill' | 'hero'; collapsed?: boolean }>(),
  { variant: 'sidebar', collapsed: false },
)

const nodeStore = useNodeStore()
const menuStore = useMenuStore()
const router = useRouter()
const { t } = useI18n()
const ic = (name: string) => menuStore.iconComponents[name]
const hostname = useCurrentHostname()
const localHostname = computed(() => (nodeStore.isRemote ? '' : hostname.value))

const canViewNodes = computed(() => menuStore.hasButtonPermission('node:view'))
const canSwitch = computed(
  () => nodeStore.isRemote || (canViewNodes.value && nodeStore.nodes.length > 0),
)

const stateLabel = (state?: NodeState) => {
  if (state === NodeState.NodeState_Online) return t('node.state.online')
  if (state === NodeState.NodeState_Offline) return t('node.state.offline')
  if (state === NodeState.NodeState_Unauthorized) return t('node.state.unauthorized')
  return t('node.state.unknown')
}
const stateTone = (state?: NodeState) => {
  if (state === NodeState.NodeState_Online) return 'var(--m-ok)'
  if (state === NodeState.NodeState_Offline) return 'var(--m-bad)'
  if (state === NodeState.NodeState_Unauthorized) return 'var(--m-warn)'
  return 'var(--el-text-color-secondary)'
}
const nodeStateText = (node: NodeInfo) => {
  const st = node.status
  const label = stateLabel(st?.state)
  if (st?.state === NodeState.NodeState_Online && st.latencyMs) return `${label} · ${st.latencyMs} ms`
  return label
}

const tone = computed(() =>
  nodeStore.isRemote ? stateTone(nodeStore.contextInfo?.status?.state) : 'var(--m-ok)',
)
const stateLine = computed(() =>
  nodeStore.isRemote
    ? `${t('node.switcher.remote')} · ${stateLabel(nodeStore.contextInfo?.status?.state)}`
    : `${t('node.switcher.local')} · ${t('node.state.online')}`,
)
const nameLine = computed(() =>
  nodeStore.isRemote
    ? nodeStore.contextInfo?.name || hostname.value || t('node.switcher.remote')
    : hostname.value || t('node.switcher.localDesc'),
)
const pillLabel = computed(() =>
  nodeStore.isRemote
    ? `${t('node.switcher.remote')} · ${nameLine.value}`
    : hostname.value
      ? `${t('node.switcher.local')} · ${hostname.value}`
      : t('node.switcher.local'),
)

const triggerClass = computed(() =>
  props.variant === 'sidebar'
    ? ['ns-card', { 'is-collapsed': props.collapsed, 'is-remote': nodeStore.isRemote }]
    : ['ns-pill', `ns-pill--${props.variant}`],
)

const onOpen = () => {
  if (canViewNodes.value) void nodeStore.loadNodes()
}

const pick = (id: string, close: () => void) => {
  close()
  nodeStore.switchNode(id)
}

const goManage = (close: () => void) => {
  close()
  if (menuStore.isMobileMenuOpen) menuStore.isMobileMenuOpen = false
  void router.push('/node')
}

onMounted(() => {
  if (canViewNodes.value && !nodeStore.nodes.length) void nodeStore.loadNodes()
})
</script>

<style scoped lang="scss">
/* ---- 侧栏卡片 ---- */
.ns-card {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 56px;
  padding: 8px 12px;
  border: 1.5px dashed var(--el-border-color-light);
  border-radius: 16px;
  background: var(--el-bg-color-page);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
button.ns-card:hover {
  border-color: var(--m-p);
}
.ns-card.is-static {
  cursor: default;
}
.ns-card.is-remote {
  border-style: solid;
  border-color: color-mix(in srgb, var(--m-p) 45%, transparent);
  background: var(--m-pSoft);
}
.ns-card.is-collapsed {
  justify-content: center;
  width: 52px;
  min-height: 52px;
  padding: 0;
  margin: 0 auto;
}
.ns-card__sk {
  width: 36px;
  height: 36px;
  border-radius: 11px;
}
.ns-card__sk :deep(svg) {
  width: 17px;
  height: 17px;
}
.ns-card__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.ns-card__state {
  font-size: 0.6875rem;
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ns-card__name {
  font-size: 0.8125rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ns-card__caret {
  width: 16px;
  height: 16px;
  flex: none;
  color: var(--el-text-color-secondary);
}

/* ---- 胶囊 ---- */
.ns-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  max-width: 100%;
  padding: 0 14px;
  border-radius: 999px;
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.ns-pill.is-static {
  cursor: default;
}
.ns-pill--pill {
  border: 1.5px dashed var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
}
.ns-pill--hero {
  border: 0;
  background: color-mix(in srgb, var(--m-pOn) 20%, transparent);
  color: var(--m-pOn);
}
.ns-pill__dot {
  width: 8px;
  height: 8px;
  flex: none;
  border-radius: 50%;
}
.ns-pill__label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ns-pill__caret {
  width: 16px;
  height: 16px;
  flex: none;
  opacity: 0.75;
}

/* ---- 下拉 ---- */
.node-menu {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px;
}
.node-menu__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 6px 6px 8px;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--el-text-color-secondary);
}
.node-menu__refresh {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 10px;
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
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 320px;
  overflow-y: auto;
  scrollbar-width: none;
}
.node-menu__list::-webkit-scrollbar {
  display: none;
}
.node-menu__empty {
  padding: 10px 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.node-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 52px;
  padding: 8px 10px;
  border: none;
  border-radius: 14px;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s;
}
.node-row:hover {
  background: var(--el-fill-color-light);
}
.node-row.is-active {
  background: var(--m-pSoft);
}
.node-row__sk {
  width: 34px;
  height: 34px;
  border-radius: 11px;
}
.node-row__sk :deep(svg) {
  width: 16px;
  height: 16px;
}
.node-row__check {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  color: var(--m-pInk);
  stroke-width: 2.6;
}
.node-row__body {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}
.node-row__name {
  font-size: 0.875rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-row__meta {
  font-size: 0.75rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-menu__foot {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 44px;
  margin-top: 2px;
  border: none;
  border-radius: 12px;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.node-menu__foot:hover {
  background: var(--el-fill-color-light);
}
</style>
