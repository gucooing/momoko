<!-- 应用卡片：贴纸头像 + 名称 / 类型 / 开机自启 + 状态 + 勾选，备注，信息格（已运行 · 停止于 · 维护开始 / 创建于 / 启动命令 / 工作目录），
     底部 #标签 + 小图标操作（控制台 / 文件 / 启停 / ⋯）。
     props(item/typeLabel/canDelete/selected/now) + emits(toggleSelect/console/config/changeStatus/moreAction)。 -->
<template>
  <article class="inst-card" :class="{ 'is-selected': selected }">
    <header class="inst-card__head">
      <span class="inst-card__av m-sticker" :class="`m-sticker--${tone}`" :style="{ transform: `rotate(${rot})` }">
        {{ initial }}
      </span>
      <span class="inst-card__title">
        <RouterLink class="inst-card__name" :to="detailRoute" :title="item.name">{{ item.name }}</RouterLink>
        <span class="inst-card__type">
          <span class="inst-card__type-t">{{ item.type ? typeLabel : t('instance.ui.uncategorized') }}</span>
          <span v-if="item.autoStart" class="inst-card__badge">{{ t('instance.ui.bootStart') }}</span>
        </span>
      </span>
      <StatusPill :variant="statusVariant" :label="t(statusMeta[item.status]?.labelKey || 'instance.unknownStatus')" />
      <label class="inst-card__hit">
        <input
          type="checkbox"
          class="inst-card__check"
          :checked="selected"
          :aria-label="t('instance.ui.selectApp', { name: item.name })"
          @change="emit('toggleSelect', ($event.target as HTMLInputElement).checked)"
        />
      </label>
    </header>

    <p class="inst-card__remark" :class="{ 'is-empty': !item.remark }" :title="item.remark">
      {{ item.remark || t('instance.noRemark') }}
    </p>

    <dl class="inst-card__info">
      <div>
        <dt>{{ upLabel }}</dt>
        <dd class="m-num">{{ upValue }}</dd>
      </div>
      <div>
        <dt>{{ t('instance.ui.createdAt') }}</dt>
        <dd class="m-num">{{ fmtDate(item.createTime) }}</dd>
      </div>
      <div class="is-wide">
        <dt>{{ t('instance.startCommand') }}</dt>
        <dd class="m-mono" :title="item.startCommand">{{ item.startCommand || '—' }}</dd>
      </div>
      <div class="is-wide">
        <dt>{{ t('instance.ui.workdir') }}</dt>
        <dd class="m-mono" :title="item.absolutePath || item.instancePath">{{ item.instancePath || '—' }}</dd>
      </div>
    </dl>

    <footer class="inst-card__foot">
      <div class="inst-card__tags">
        <span v-for="tag in tags" :key="tag" class="inst-card__tag">#{{ tag }}</span>
      </div>
      <AppIconButton icon="HOutline:CommandLineIcon" :label="t('instance.ui.consoleOf', { name: item.name })" @click="emit('console')" />
      <AppIconButton
        icon="HOutline:FolderIcon"
        :label="t('instance.ui.filesOf', { name: item.name })"
        @click="emit('moreAction', 'fileManager')"
      />
      <AppIconButton
        :icon="isRunning ? 'HOutline:StopIcon' : 'HOutline:PlayIcon'"
        :label="isRunning ? t('instance.stop') : t('instance.start')"
        :class="isRunning ? 'is-stop' : 'is-start'"
        @click="
          emit(
            'changeStatus',
            isRunning ? InstanceStatus.INSTANCE_STATUS_STOPPED : InstanceStatus.INSTANCE_STATUS_RUNNING,
          )
        "
      />
      <ActionMenu :items="moreItems" :label="t('instance.ui.moreOf', { name: item.name })" @select="onMore" />
    </footer>
  </article>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { useI18n } from 'vue-i18n'
import { InstanceStatus, statusMeta, type InstanceRecord } from '@/stores/instance/types'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import { fmtDuration, initialOf, rotOf, toneOf } from '@/views/instance/shared'

type MoreAction = 'forceRestart' | 'restart' | 'delete' | 'fileManager' | 'maintenanceOn' | 'maintenanceOff'

const props = defineProps<{
  item: InstanceRecord
  typeLabel: string
  canDelete: boolean
  selected: boolean
  /** 当前时间（父级定时刷新，用于运行时长） */
  now: number
}>()

const emit = defineEmits<{
  toggleSelect: [value: boolean]
  console: []
  config: []
  changeStatus: [value: typeof InstanceStatus.INSTANCE_STATUS_RUNNING | typeof InstanceStatus.INSTANCE_STATUS_STOPPED]
  moreAction: [value: MoreAction]
}>()

const { t } = useI18n()

const tone = computed(() => toneOf(props.item.id))
const rot = computed(() => rotOf(props.item.id))
const initial = computed(() => initialOf(props.item.name))
const detailRoute = computed(() => ({ path: `/instance/app/${props.item.id}`, query: { tabTitle: props.item.name } }))

const tags = computed(() =>
  (props.item.tags || '')
    .split(/[,，]/u)
    .map((s) => s.trim().replace(/^#/u, ''))
    .filter(Boolean)
    .slice(0, 3),
)

const isRunning = computed(() => props.item.status === InstanceStatus.INSTANCE_STATUS_RUNNING)
const isMaintenance = computed(() => props.item.status === InstanceStatus.INSTANCE_STATUS_MAINTENANCE)

const statusVariantMap: Record<InstanceStatus, 'success' | 'neutral' | 'warning' | 'error'> = {
  [InstanceStatus.INSTANCE_STATUS_RUNNING]: 'success',
  [InstanceStatus.INSTANCE_STATUS_STOPPED]: 'neutral',
  [InstanceStatus.INSTANCE_STATUS_MAINTENANCE]: 'warning',
  [InstanceStatus.INSTANCE_STATUS_UNSPECIFIED]: 'neutral',
  [InstanceStatus.UNRECOGNIZED]: 'error',
}
const statusVariant = computed(() => statusVariantMap[props.item.status] || 'neutral')

const fmtTime = (value: unknown) => {
  if (!value) return '—'
  const date = dayjs(value as string | Date)
  return date.isValid() ? date.format('MM-DD HH:mm') : '—'
}
const fmtDate = (value: unknown) => {
  if (!value) return '—'
  const date = dayjs(value as string | Date)
  return date.isValid() ? date.format('YYYY-MM-DD') : '—'
}

// 信息格第一格：运行中显示已运行时长；维护中显示维护开始时间；否则显示最近一次停止时间
const upLabel = computed(() => {
  if (isRunning.value) return t('instance.ui.upRunning')
  if (isMaintenance.value) return t('instance.ui.upMaintenance')
  return t('instance.ui.upStoppedAt')
})
const upValue = computed(() => {
  const i = props.item
  if (isRunning.value) {
    return i.startTime ? fmtDuration(t, props.now - dayjs(i.startTime as unknown as string).valueOf()) : '—'
  }
  if (isMaintenance.value) return fmtTime(i.maintenanceTime)
  return i.stopTime ? fmtTime(i.stopTime) : t('instance.ui.neverRun')
})

const moreItems = computed<ActionMenuItem[]>(() => [
  { key: 'config', label: t('instance.ui.editApp'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'restart', label: t('instance.restart'), icon: 'HOutline:ArrowPathIcon', disabled: isMaintenance.value },
  { key: 'forceRestart', label: t('instance.forceRestart'), icon: 'HOutline:BoltIcon', disabled: isMaintenance.value },
  props.item.maintenance
    ? { key: 'maintenanceOff', label: t('instance.ui.endMaintenance'), icon: 'HOutline:CheckBadgeIcon' }
    : { key: 'maintenanceOn', label: t('instance.ui.enterMaintenance'), icon: 'HOutline:WrenchScrewdriverIcon' },
  { key: 'delete', label: t('common.delete'), icon: 'HOutline:TrashIcon', danger: true, hidden: !props.canDelete },
])

const onMore = (key: string) => {
  if (key === 'config') emit('config')
  else emit('moreAction', key as MoreAction)
}
</script>

<style scoped lang="scss">
.inst-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: calc(var(--cmp-card-unit) * 18) calc(var(--cmp-card-unit) * 14) calc(var(--cmp-card-unit) * 6) calc(var(--cmp-card-unit) * 18);
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.inst-card.is-selected {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.inst-card__head {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding-right: 4px;
}
.inst-card__av {
  width: 46px;
  height: 46px;
  border-radius: 15px;
  font-size: 1.2rem;
  flex: none;
}
.inst-card__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.inst-card__name {
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.inst-card__name:hover {
  color: var(--m-pInk);
}
.inst-card__type {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.inst-card__type-t {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}
.inst-card__badge {
  flex: none;
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: 6px;
  background: var(--m-a2Soft);
  color: var(--m-ink);
  font-size: 0.6875rem;
  font-weight: 700;
}
/* 勾选框本身 20px，外面一圈透明点击区（PC 40 / 手机 44），负边距不占布局 */
.inst-card__hit {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 40px;
  height: 40px;
  margin: -10px;
  cursor: pointer;
}
.inst-card__check {
  width: 20px;
  height: 20px;
  margin: 0;
  flex: none;
  accent-color: var(--m-p);
  cursor: pointer;
}
@media (width < 768px) {
  .inst-card__hit {
    width: 44px;
    height: 44px;
    margin: -12px;
  }
}
.inst-card__remark {
  padding-right: 4px;
  font-size: 0.875rem;
  line-height: 1.45;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.inst-card__remark.is-empty {
  color: var(--el-text-color-secondary);
}
.inst-card__info {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 12px;
  margin: 0 4px 0 0;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
}
.inst-card__info > div {
  min-width: 0;
}
.inst-card__info .is-wide {
  grid-column: 1 / -1;
}
.inst-card__info dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.inst-card__info dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.inst-card__info dd.m-mono {
  font-size: 0.75rem;
  font-weight: 500;
}
:global(html.is-compact) .inst-card__info {
  display: none;
}
.inst-card__foot {
  display: flex;
  align-items: center;
  gap: 2px;
  padding-top: 6px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.inst-card__tags {
  flex: 1;
  min-width: 0;
  display: flex;
  gap: 6px;
  overflow: hidden;
}
.inst-card__tag {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;
}
.inst-card__foot :deep(.is-stop) {
  color: var(--m-bad);
}
.inst-card__foot :deep(.is-start) {
  color: var(--m-ok);
}
.inst-card__foot :deep(.is-stop:hover) {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.inst-card__foot :deep(.is-start:hover) {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.inst-card__foot :deep(.action-menu__trigger) {
  width: 40px;
  height: 40px;
  border-radius: 14px;
}
@media (width < 768px) {
  .inst-card__foot :deep(.app-icon-btn),
  .inst-card__foot :deep(.action-menu__trigger) {
    width: 44px !important;
    height: 44px !important;
  }
}
</style>
