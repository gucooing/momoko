<!-- 实例卡片（重写 · P1 卡片流）：EntityCard 外壳 + 令牌控件（替代 el-button/el-checkbox/el-dropdown）。
     保留 props(item/typeLabel/canDelete/selected) 与 emits(toggleSelect/console/config/changeStatus/moreAction) 契约。 -->
<template>
  <EntityCard class="inst-card" :class="{ 'is-selected': selected }">
    <template #title>
      <div class="inst-card__head">
        <input
          type="checkbox"
          class="inst-card__check"
          :checked="selected"
          @change="emit('toggleSelect', ($event.target as HTMLInputElement).checked)"
        />
        <span class="inst-card__icon">
          <component :is="menuStore.iconComponents['HOutline:ServerStackIcon']" />
        </span>
        <span class="inst-card__titletext">
          <span class="inst-card__name">{{ item.name }}</span>
          <span class="inst-card__type">{{ typeLabel }}</span>
        </span>
      </div>
    </template>

    <template #status>
      <StatusPill :variant="statusVariant" :label="t(statusMeta[item.status].labelKey)" />
    </template>

    <template #meta>
      <span class="inst-card__remark">{{ item.remark || t('instance.noRemark') }}</span>
      <span class="inst-card__metaitem">
        <component :is="menuStore.iconComponents['HOutline:FolderIcon']" />
        <span class="truncate">{{ item.instancePath || '—' }}</span>
      </span>
      <span class="inst-card__metaitem">
        <component :is="menuStore.iconComponents['HOutline:CalendarDaysIcon']" />
        {{ fmtTime(item.createTime) }}
      </span>
    </template>

    <template #footer>
      <div class="inst-card__acts">
        <UButton
          color="neutral"
          variant="ghost"
          size="xs"
          icon="i-lucide-square-terminal"
          @click="emit('console')"
        >
          {{ t('instance.console') }}
        </UButton>
        <UButton
          color="neutral"
          variant="ghost"
          size="xs"
          icon="i-lucide-settings"
          @click="emit('config')"
        >
          {{ t('instance.config') }}
        </UButton>
      </div>
      <div class="inst-card__acts">
        <ActionMenu :items="moreItems" @select="(key) => emit('moreAction', key as MoreAction)" />
        <UButton
          :color="isRunningLike ? 'error' : 'primary'"
          variant="soft"
          size="xs"
          :icon="isRunningLike ? 'i-lucide-square' : 'i-lucide-play'"
          @click="emit('changeStatus', isRunningLike ? InstanceStatus.INSTANCE_STATUS_STOPPED : InstanceStatus.INSTANCE_STATUS_RUNNING)"
        >
          {{ isRunningLike ? t('instance.stop') : t('instance.start') }}
        </UButton>
      </div>
    </template>
  </EntityCard>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { useI18n } from 'vue-i18n'
import { InstanceStatus, statusMeta, type InstanceRecord } from '@/stores/instance/types'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'

type MoreAction = 'forceRestart' | 'delete' | 'fileManager'

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
  moreAction: [value: MoreAction]
}>()

const menuStore = useMenuStore()
const { t } = useI18n()

const isRunningLike = computed(
  () => props.item.status === InstanceStatus.INSTANCE_STATUS_RUNNING,
)

const statusVariantMap: Record<InstanceStatus, 'success' | 'neutral' | 'warning' | 'error'> = {
  [InstanceStatus.INSTANCE_STATUS_RUNNING]: 'success',
  [InstanceStatus.INSTANCE_STATUS_STOPPED]: 'neutral',
  [InstanceStatus.INSTANCE_STATUS_MAINTENANCE]: 'warning',
  [InstanceStatus.INSTANCE_STATUS_UNSPECIFIED]: 'neutral',
  [InstanceStatus.UNRECOGNIZED]: 'error',
}
const statusVariant = computed(() => statusVariantMap[props.item.status] || 'neutral')

const moreItems = computed<ActionMenuItem[]>(() => [
  { key: 'forceRestart', label: t('instance.forceRestart'), icon: 'HOutline:ArrowPathIcon' },
  { key: 'fileManager', label: t('instance.fileManagerTitle'), icon: 'HOutline:FolderIcon' },
  {
    key: 'delete',
    label: t('common.delete'),
    icon: 'HOutline:TrashIcon',
    danger: true,
    hidden: !props.canDelete,
  },
])

const fmtTime = (value: unknown) => {
  if (!value) return '—'
  const date = dayjs(value as string | Date)
  return date.isValid() ? date.format('YYYY-MM-DD HH:mm') : '—'
}
</script>

<style scoped lang="scss">
.inst-card.is-selected {
  border-color: color-mix(in srgb, var(--el-color-primary) 55%, var(--el-border-color));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 30%, transparent);
}

/* 头部：勾选 + 单色图标 + 名称/类型 */
.inst-card__head {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.inst-card__check {
  width: 15px;
  height: 15px;
  accent-color: var(--el-color-primary);
  cursor: pointer;
  flex-shrink: 0;
}
.inst-card__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: var(--app-radius-sm);
  flex-shrink: 0;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.inst-card__icon :deep(svg) {
  width: 17px;
  height: 17px;
}
.inst-card__titletext {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.inst-card__name {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.inst-card__type {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* meta：备注独占一行，路径/时间成行 */
.inst-card__remark {
  flex-basis: 100%;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.inst-card__metaitem {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  color: var(--el-text-color-placeholder);
  font-size: 0.75rem;
}
.inst-card__metaitem :deep(svg) {
  width: 13px;
  height: 13px;
  flex-shrink: 0;
}
.inst-card__metaitem .truncate {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 底部操作组 */
.inst-card__acts {
  display: flex;
  align-items: center;
  gap: 4px;
}
</style>
