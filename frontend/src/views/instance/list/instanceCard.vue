<template>
  <div :class="['instance-card', { 'is-selected': selected }]">
    <div class="card-check">
      <el-checkbox :model-value="selected" @change="emit('toggleSelect', Boolean($event))" />
    </div>

    <div class="card-header">
      <div class="card-icon" :style="{ backgroundColor: `${accentColor}18`, color: accentColor }">
        <el-icon size="18">
          <component :is="menuStore.iconComponents['HOutline:ServerStackIcon']" />
        </el-icon>
      </div>
      <div class="card-title">
        <div class="card-name">{{ item.name }}</div>
        <div class="card-type">{{ typeLabel }}</div>
      </div>
      <span class="card-status" :class="statusClassMap[item.status]">
        {{ statusMeta[item.status].label }}
      </span>
    </div>

    <div class="card-body">
      <p class="card-remark">{{ item.remark || '暂无备注' }}</p>
      <div class="card-metas">
        <div class="card-meta">
          <el-icon size="12"><component :is="menuStore.iconComponents['HOutline:CalendarDaysIcon']" /></el-icon>
          <span>{{ formatDateTime(item.createTime) }}</span>
        </div>
        <div class="card-meta">
          <el-icon size="12"><component :is="menuStore.iconComponents['HOutline:FolderIcon']" /></el-icon>
          <span class="truncate">{{ item.instancePath || '-' }}</span>
        </div>
      </div>
    </div>

    <div class="card-footer">
      <el-button size="small" text @click="emit('console')">
        <el-icon size="14"><component :is="menuStore.iconComponents['HOutline:CommandLineIcon']" /></el-icon>
        控制台
      </el-button>
      <el-button size="small" text @click="emit('config')">
        <el-icon size="14"><component :is="menuStore.iconComponents['HOutline:Cog6ToothIcon']" /></el-icon>
        配置
      </el-button>
      <el-dropdown trigger="click" @command="(action: string) => emit('moreAction', action as any)">
        <el-button size="small" text>
          <el-icon size="14"><component :is="menuStore.iconComponents['HOutline:EllipsisHorizontalIcon']" /></el-icon>
          更多
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="forceRestart">强制重启</el-dropdown-item>
            <el-dropdown-item command="fileManager">文件管理</el-dropdown-item>
            <el-dropdown-item v-if="canDelete" command="delete" divided>删除</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-button
        size="small"
        :type="isRunningLike ? 'danger' : 'primary'"
        plain
        @click="emit('changeStatus', isRunningLike ? InstanceStatus.INSTANCE_STATUS_STOPPED : InstanceStatus.INSTANCE_STATUS_RUNNING)"
      >
        <el-icon size="14">
          <component :is="menuStore.iconComponents[isRunningLike ? 'HOutline:StopIcon' : 'HOutline:PlayIcon']" />
        </el-icon>
        {{ isRunningLike ? '停止' : '启动' }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { InstanceStatus, statusMeta, type InstanceRecord } from '@/stores/instance/types'

const props = defineProps<{
  item: InstanceRecord
  typeLabel: string
  canDelete: boolean
  selected: boolean
}>()

const emit = defineEmits<{
  toggleSelect: [value: boolean]
  console: []
  config: []
  changeStatus: [
    value:
      | typeof InstanceStatus.INSTANCE_STATUS_RUNNING
      | typeof InstanceStatus.INSTANCE_STATUS_STOPPED,
  ]
  moreAction: [value: 'forceRestart' | 'delete' | 'fileManager']
}>()

const menuStore = useMenuStore()

const isRunningLike = computed(
  () => props.item.status === InstanceStatus.INSTANCE_STATUS_RUNNING,
)

const statusAccentMap: Record<InstanceStatus, string> = {
  [InstanceStatus.INSTANCE_STATUS_RUNNING]: '#10b981',
  [InstanceStatus.INSTANCE_STATUS_STOPPED]: '#94a3b8',
  [InstanceStatus.INSTANCE_STATUS_MAINTENANCE]: '#f59e0b',
  [InstanceStatus.INSTANCE_STATUS_UNSPECIFIED]: '#94a3b8',
  [InstanceStatus.UNRECOGNIZED]: '#ef4444',
}

const accentColor = computed(() => statusAccentMap[props.item.status] || '#94a3b8')

const statusClassMap: Record<InstanceStatus, string> = {
  [InstanceStatus.INSTANCE_STATUS_RUNNING]: 'is-running',
  [InstanceStatus.INSTANCE_STATUS_STOPPED]: 'is-stopped',
  [InstanceStatus.INSTANCE_STATUS_MAINTENANCE]: 'is-maintenance',
  [InstanceStatus.INSTANCE_STATUS_UNSPECIFIED]: 'is-unspecified',
  [InstanceStatus.UNRECOGNIZED]: 'is-unrecognized',
}

const formatDateTime = (value: unknown) => {
  if (!value) return '-'
  const date = dayjs(value as string | Date)
  return date.isValid() ? date.format('YYYY-MM-DD HH:mm:ss') : '-'
}
</script>

<style scoped lang="scss">
.instance-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 0.7rem;
  padding: 0.9rem 1rem;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.75rem;
  background: var(--el-bg-color);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.instance-card:hover {
  border-color: var(--el-border-color);
  box-shadow: 0 2px 8px rgb(15 23 42 / 5%);
}

.instance-card.is-selected {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 20%, transparent);
}

.card-check {
  position: absolute;
  top: 0.65rem;
  right: 0.75rem;
  z-index: 1;
}

.card-header {
  display: flex;
  align-items: flex-start;
  gap: 0.65rem;
  padding-right: 1.8rem;
}

.card-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2.2rem;
  height: 2.2rem;
  border-radius: 0.55rem;
  flex-shrink: 0;
}

.card-title {
  min-width: 0;
  flex: 1;
}

.card-name {
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-type {
  font-size: 0.72rem;
  color: var(--el-text-color-secondary);
  margin-top: 0.12rem;
}

.card-status {
  flex-shrink: 0;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
  line-height: 1;
}

.card-status.is-running {
  color: #059669;
  background: color-mix(in srgb, #10b981 10%, transparent);
}

.card-status.is-maintenance {
  color: #b45309;
  background: color-mix(in srgb, #f59e0b 10%, transparent);
}

.card-status.is-stopped {
  color: #475569;
  background: color-mix(in srgb, #94a3b8 10%, transparent);
}

.card-status.is-unspecified {
  color: #475569;
  background: color-mix(in srgb, #94a3b8 10%, transparent);
}

.card-status.is-unrecognized {
  color: #dc2626;
  background: color-mix(in srgb, #ef4444 10%, transparent);
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.card-remark {
  margin: 0;
  font-size: 0.8rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-metas {
  display: flex;
  flex-direction: column;
  gap: 0.22rem;
}

.card-meta {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  overflow: hidden;
}

.card-meta span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-footer {
  display: flex;
  gap: 0.3rem;
  padding-top: 0.55rem;
  border-top: 1px solid var(--el-border-color-extra-light);
  flex-wrap: wrap;
}

.card-footer :deep(.el-button) {
  font-size: 0.78rem;
}

:global(html.dark .instance-card) {
  background: color-mix(in srgb, #ffffff 3%, var(--el-bg-color));
  border-color: color-mix(in srgb, var(--el-border-color) 55%, transparent);
}

:global(html.dark .instance-card:hover) {
  border-color: color-mix(in srgb, var(--el-border-color) 80%, transparent);
}

:global(html.dark .card-status.is-running) {
  color: #34d399;
  background: color-mix(in srgb, #10b981 18%, transparent);
}

:global(html.dark .card-status.is-maintenance) {
  color: #fbbf24;
  background: color-mix(in srgb, #f59e0b 16%, transparent);
}

:global(html.dark .card-status.is-stopped) {
  color: #94a3b8;
  background: color-mix(in srgb, #94a3b8 12%, transparent);
}

:global(html.dark .card-status.is-unspecified) {
  color: #94a3b8;
  background: color-mix(in srgb, #94a3b8 12%, transparent);
}

:global(html.dark .card-status.is-unrecognized) {
  color: #f87171;
  background: color-mix(in srgb, #ef4444 18%, transparent);
}
</style>
