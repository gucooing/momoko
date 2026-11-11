<!-- 节点卡片：贴纸图标 + 名称 / 地址 / TLS + 状态 pill；信息格（延迟 / 版本 / 上次探测 / 系统 / 主机名 / Key）；
     异常原因条；授权范围（按功能分组）；底部 切换 / 测试 + 编辑 / 更多。
     local=true 时是主控本机卡片：信息来自 ListNodesResponse.local，只能「切换到本机」。 -->
<template>
  <article class="node-card" :class="{ 'is-current': isCurrent }">
    <header class="node-card__head">
      <span class="node-card__av m-sticker" :class="`m-sticker--${tone}`" :style="{ transform: `rotate(${rot})` }">
        <component :is="menuStore.iconComponents[local ? 'HOutline:ComputerDesktopIcon' : 'HOutline:ServerIcon']" />
      </span>
      <span class="node-card__title">
        <span class="node-card__name-row">
          <span class="node-card__name" :title="name">{{ name }}</span>
          <span v-if="local && !menuStore.isMobile" class="node-tag is-main">{{ t('node.page.localTag') }}</span>
        </span>
        <span class="node-card__addr-row">
          <span class="node-card__addr m-mono" :title="address">{{ address }}</span>
          <span v-if="!menuStore.isMobile" class="node-tag" :class="{ 'is-ok': tlsOn }">
            <component :is="menuStore.iconComponents[tlsOn ? 'HOutline:ShieldCheckIcon' : 'HOutline:LinkIcon']" />
            {{ tlsLabel }}
          </span>
        </span>
      </span>
      <StatusPill :variant="pill.variant" :label="pill.label" />
    </header>

    <div v-if="menuStore.isMobile" class="node-card__tags">
      <span v-if="local" class="node-tag is-main">{{ t('node.page.localTag') }}</span>
      <span class="node-tag" :class="{ 'is-ok': tlsOn }">
        <component :is="menuStore.iconComponents[tlsOn ? 'HOutline:ShieldCheckIcon' : 'HOutline:LinkIcon']" />
        {{ tlsLabel }}
      </span>
      <span v-if="upgradable" class="node-tag is-warn" :title="upgradeTip">{{ t('node.page.upgradable') }}</span>
    </div>

    <p v-if="node?.remark" class="node-card__remark" :title="node.remark">{{ node.remark }}</p>

    <dl class="node-card__info">
      <div v-for="cell in cells" :key="cell.key" :class="{ 'is-wide': cell.wide }">
        <dt>{{ cell.label }}</dt>
        <dd :class="[cell.cls, { 'm-num': cell.num }]" :title="cell.title || cell.value">
          {{ cell.value }}
          <span
            v-if="cell.key === 'version' && upgradable && !menuStore.isMobile"
            class="node-tag is-warn is-xs"
            :title="upgradeTip"
          >
            {{ t('node.page.upgradable') }}
          </span>
        </dd>
      </div>
    </dl>

    <div v-if="errorText" class="node-card__error">
      <component :is="menuStore.iconComponents['HOutline:ExclamationTriangleIcon']" />
      <span>{{ errorText }}</span>
    </div>

    <div class="node-card__scope">
      <span class="node-card__scope-title">{{ scopeTitle }}</span>
      <NodePermissionChips v-if="permissions" variant="groups" :permissions="permissions" />
      <span v-else class="node-card__scope-empty">{{ t('access.node.scopeUnknown') }}</span>
    </div>

    <footer class="node-card__foot">
      <template v-if="!menuStore.isMobile">
        <span v-if="isCurrent" class="node-card__current">
          <component :is="menuStore.iconComponents['HOutline:CheckIcon']" />
          {{ t('access.node.currentNode') }}
        </span>
        <button v-else-if="canSwitch" type="button" class="node-pill is-primary" @click="emit('switch')">
          <component :is="menuStore.iconComponents['HOutline:ArrowsRightLeftIcon']" />
          {{ t('node.list.switchTo') }}
        </button>
        <button v-if="!local && canManage" type="button" class="node-pill" :disabled="testing" @click="emit('test')">
          <component :is="menuStore.iconComponents['HOutline:SignalIcon']" />
          {{ testing ? t('node.list.testing') : t('node.list.test') }}
        </button>
        <span class="node-card__spacer" />
      </template>
      <template v-else>
        <div class="node-card__foot-tags">
          <span v-if="isCurrent" class="node-card__current">{{ t('access.node.currentNode') }}</span>
          <span v-if="testing" class="node-tag">{{ t('node.list.testing') }}</span>
        </div>
        <AppIconButton
          v-if="canSwitch && !isCurrent"
          class="is-switch"
          icon="HOutline:ArrowsRightLeftIcon"
          :box="44"
          :label="t('node.list.switchTo')"
          @click="emit('switch')"
        />
        <AppIconButton
          v-if="!local && canManage"
          icon="HOutline:SignalIcon"
          :box="44"
          :label="t('node.list.test')"
          @click="emit('test')"
        />
      </template>
      <AppIconButton
        v-if="!local && canManage"
        icon="HOutline:PencilSquareIcon"
        :box="menuStore.isMobile ? 44 : 40"
        :size="18"
        :label="t('node.list.edit')"
        @click="emit('edit')"
      />
      <ActionMenu v-if="!local && canManage" :items="moreItems" @select="onMore" />
    </footer>
  </article>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NodeState, type NodeInfo, type NodeRemoteInfo } from '@/types/v1/node'
import { NODE_PERMISSION_GROUPS } from '@/config/nodePermission'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import NodePermissionChips from '@/views/node/components/permissionChips.vue'
import { useNodeTime } from '@/views/node/components/useNodeTime'

const props = defineProps<{
  /** 远程节点；本机卡片不传 */
  node?: NodeInfo
  /** 本机信息（local 卡片） */
  localInfo?: NodeRemoteInfo
  local?: boolean
  /** 本机卡片的授权（当前用户在主控上的节点类权限） */
  localPermissions?: string[]
  isCurrent: boolean
  canManage: boolean
  testing?: boolean
  /** 主控版本，节点版本低于它时提示可升级 */
  masterVersion?: string
  index: number
}>()

const emit = defineEmits<{ switch: []; test: []; edit: []; delete: [] }>()

const { t } = useI18n()
const menuStore = useMenuStore()
const { ago, dateTime, since } = useNodeTime()

const TONES = ['p', 'a2', 'a', 'ok']
const ROTS = ['-4deg', '3deg', '-2deg', '4deg']
const tone = computed(() => TONES[props.index % TONES.length])
const rot = computed(() => ROTS[props.index % ROTS.length])

const status = computed(() => props.node?.status)
const state = computed(() => (props.local ? NodeState.NodeState_Online : status.value?.state ?? NodeState.NodeState_Unknown))
const online = computed(() => state.value === NodeState.NodeState_Online)
/** 节点自报信息：在线时为最新，离线时用最近一次在线的记录 */
const remote = computed<NodeRemoteInfo | undefined>(() =>
  props.local ? props.localInfo : status.value?.remote || status.value?.lastRemote || undefined,
)

const name = computed(() =>
  props.local ? remote.value?.hostname || t('node.page.localName') : props.node?.name || '—',
)
const address = computed(() => (props.local ? window.location.host : props.node?.address || '—'))
const tlsOn = computed(() => !props.local && !!props.node?.tls)
const tlsLabel = computed(() => {
  if (props.local) return t('node.page.localDirect')
  if (!props.node?.tls) return t('node.page.plaintext')
  return props.node.tlsSkipVerify ? t('node.page.tlsSelfSigned') : t('node.page.tls')
})

const canSwitch = computed(() => !props.isCurrent && (props.local || online.value))

const pill = computed<{ variant: 'success' | 'error' | 'warning' | 'neutral'; label: string }>(() => {
  if (props.local) return { variant: 'success', label: t('accessV2.node.onlineLocal') }
  switch (state.value) {
    case NodeState.NodeState_Online:
      return { variant: 'success', label: t('node.page.onlineLatency', { ms: status.value?.latencyMs ?? 0 }) }
    case NodeState.NodeState_Offline: {
      // 「离线 · 3 天」：按最近一次在线时间算离线时长
      const last = status.value?.lastOnlineTime
      return {
        variant: 'error',
        label: last ? t('accessV2.node.offlineFor', { duration: since(last) }) : t('node.state.offline'),
      }
    }
    case NodeState.NodeState_Unauthorized:
      return { variant: 'warning', label: t('node.state.unauthorized') }
    default:
      return { variant: 'neutral', label: t('node.state.unknown') }
  }
})

// —— 版本比较：x.y.z（忽略前缀 v 与预发布后缀）；无法解析时不提示 ——
const parseVer = (v?: string) => {
  const m = /^v?(\d+)\.(\d+)\.(\d+)/.exec((v || '').trim())
  return m ? [Number(m[1]), Number(m[2]), Number(m[3])] : null
}
const upgradable = computed(() => {
  if (props.local) return false
  const a = parseVer(remote.value?.version)
  const b = parseVer(props.masterVersion)
  if (!a || !b) return false
  for (let i = 0; i < 3; i++) {
    if (a[i] !== b[i]) return a[i]! < b[i]!
  }
  return false
})
const upgradeTip = computed(() => t('node.page.upgradableTip', { version: props.masterVersion || '' }))

// 系统：发行版（去掉内核）· 架构
const systemText = computed(() => {
  const r = remote.value
  if (!r) return '—'
  const distro = (r.platform || '').split(' · ')[0] || r.os
  return [distro, r.arch].filter(Boolean).join(' · ')
})
const systemTitle = computed(() => {
  const r = remote.value
  return r ? [r.platform, `${r.os}/${r.arch}`].filter(Boolean).join(' · ') : ''
})

interface Cell {
  key: string
  label: string
  value: string
  title?: string
  num?: boolean
  wide?: boolean
  cls?: string
}
const cells = computed<Cell[]>(() => {
  const r = remote.value
  if (props.local) {
    // 本机：直连无延迟，信息随列表实时返回
    return [
      { key: 'latency', label: t('access.node.latency'), value: t('accessV2.node.localLatency'), num: true, cls: 'is-ok' },
      { key: 'version', label: t('node.list.version'), value: r?.version || '—', num: true },
      { key: 'check', label: t('access.node.lastCheck'), value: t('accessV2.node.live') },
      { key: 'system', label: t('node.list.platform'), value: systemText.value, title: systemTitle.value },
    ]
  }
  const s = status.value
  const latency = online.value ? `${s?.latencyMs ?? 0} ms` : '—'
  const list: Cell[] = [
    { key: 'latency', label: t('access.node.latency'), value: latency, num: true, cls: online.value ? 'is-ok' : '' },
    { key: 'version', label: t('node.list.version'), value: r?.version || '—', num: true },
    {
      key: 'check',
      label: t('access.node.lastCheck'),
      value: s?.checkTime ? ago(s.checkTime) : t('node.page.neverChecked'),
      title: s?.checkTime ? dateTime(s.checkTime) : '',
      num: true,
    },
    { key: 'system', label: t('node.list.platform'), value: systemText.value, title: systemTitle.value },
  ]
  if (!online.value) {
    list.push({
      key: 'lastOnline',
      label: t('node.page.lastOnline'),
      value: s?.lastOnlineTime ? ago(s.lastOnlineTime) : '—',
      title: s?.lastOnlineTime ? dateTime(s.lastOnlineTime) : '',
      num: true,
    })
  } else {
    list.push({ key: 'hostname', label: t('node.list.hostname'), value: r?.hostname || '—' })
  }
  list.push({
    key: 'key',
    label: t('access.node.keyOwner'),
    value: r ? `${r.apiKeyName || '—'} · ${r.username || '—'}` : props.node?.keyId || '—',
    title: props.node?.keyId,
    wide: true,
  })
  return list
})

const errorText = computed(() => {
  if (props.local || online.value) return ''
  return status.value?.message || ''
})

const permissions = computed<string[] | null>(() => {
  if (props.local) return props.localPermissions || []
  return remote.value ? remote.value.permissions || [] : null
})
const grantedGroups = computed(() => {
  const set = new Set(permissions.value || [])
  return NODE_PERMISSION_GROUPS.filter((g) => g.items.some((i) => set.has(i.code))).length
})
const scopeTitle = computed(() =>
  props.local
    ? t('node.page.localScope', { granted: grantedGroups.value, total: NODE_PERMISSION_GROUPS.length })
    : t('access.node.scope', { granted: grantedGroups.value, total: NODE_PERMISSION_GROUPS.length }),
)

const moreItems = computed<ActionMenuItem[]>(() => [
  { key: 'switch', label: t('node.list.switchTo'), icon: 'HOutline:ArrowsRightLeftIcon', hidden: !canSwitch.value },
  { key: 'test', label: t('node.list.test'), icon: 'HOutline:SignalIcon' },
  { key: 'delete', label: t('node.list.delete'), icon: 'HOutline:TrashIcon', danger: true },
])
const onMore = (key: string) => {
  if (key === 'switch') emit('switch')
  else if (key === 'test') emit('test')
  else if (key === 'delete') emit('delete')
}
</script>

<style scoped lang="scss">
.node-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: calc(var(--cmp-card-unit) * 20) calc(var(--cmp-card-unit) * 16) calc(var(--cmp-card-unit) * 8) calc(var(--cmp-card-unit) * 20);
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.node-card.is-current {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.node-card__head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
  padding-right: 4px;
}
.node-card__av {
  width: 48px;
  height: 48px;
  border-radius: 16px;
  flex: none;
}
.node-card__av :deep(svg) {
  width: 22px;
  height: 22px;
}
.node-card__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.node-card__name-row,
.node-card__addr-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.node-card__name {
  min-width: 0;
  max-width: 100%;
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-card__addr {
  min-width: 0;
  max-width: 100%;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 700;
  white-space: nowrap;
}
.node-tag :deep(svg) {
  width: 12px;
  height: 12px;
}
.node-tag.is-ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.node-tag.is-main {
  background: var(--m-a);
  color: var(--m-aOn);
}
.node-tag.is-warn {
  background: var(--m-warnSoft);
  color: var(--m-warn);
}
.node-tag.is-xs {
  height: 20px;
  padding: 0 6px;
  font-size: 0.6875rem;
  vertical-align: 1px;
}
.node-card__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: -4px;
}
.node-card__remark {
  margin: -4px 4px 0 0;
  font-size: 0.875rem;
  line-height: 1.45;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-card__info {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px 12px;
  margin: 0 4px 0 0;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.node-card__info > div {
  min-width: 0;
}
.node-card__info .is-wide {
  grid-column: span 2;
}
.node-card__info dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.node-card__info dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-card__info dd.is-ok {
  color: var(--m-ok);
}
.node-card__error {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-right: 4px;
  padding: 9px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-badSoft);
  color: var(--m-bad);
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.45;
  word-break: break-word;
}
.node-card__error :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
  margin-top: 1px;
}
.node-card__scope {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 4px;
}
.node-card__scope-title {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.node-card__scope-empty {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.node-card__foot {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: auto;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.node-card__spacer {
  flex: 1;
}
.node-card__foot-tags {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.node-card__current {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--m-okSoft);
  color: var(--m-ok);
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
}
.node-card__current :deep(svg) {
  width: 16px;
  height: 16px;
}
.node-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, filter 0.15s;
}
.node-pill:hover:not(:disabled) {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.node-pill.is-primary {
  background: var(--m-p);
  color: var(--m-pOn);
}
.node-pill.is-primary:hover {
  filter: brightness(1.08);
  background: var(--m-p);
  color: var(--m-pOn);
}
.node-pill:disabled {
  opacity: 0.6;
  cursor: progress;
}
.node-pill :deep(svg) {
  width: 16px;
  height: 16px;
}
.node-card__foot :deep(.is-switch) {
  color: var(--m-pInk);
}
.node-card__foot :deep(.action-menu__trigger) {
  width: 40px;
  height: 40px;
  border-radius: 14px;
}

@media (width < 768px) {
  .node-card {
    gap: 12px;
    padding: 16px 12px 6px 16px;
  }
  .node-card__av {
    width: 46px;
    height: 46px;
    border-radius: 15px;
  }
  .node-card__info {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    padding: 10px 12px;
  }
  .node-card__foot {
    gap: 0;
    padding-top: 6px;
  }
  .node-card__foot :deep(.action-menu__trigger) {
    width: 44px;
    height: 44px;
  }
}
</style>
