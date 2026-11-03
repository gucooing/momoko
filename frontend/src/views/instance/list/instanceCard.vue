<!-- 实例卡片：贴纸头像 + 名称/类型/自启动 + 状态，备注，信息格（运行时长/创建时间/启动命令/目录），
     底部标签 + 小图标操作（控制台 / 文件 / 启停 / 更多）。
     保留 props(item/typeLabel/canDelete/selected) 与 emits(toggleSelect/console/config/changeStatus/moreAction) 契约。 -->
<template>
  <article class="inst-card" :class="{ 'is-selected': selected }">
    <header class="inst-card__head">
      <span class="inst-card__av m-sticker" :class="`m-sticker--${tone}`" :style="{ transform: `rotate(${rot})` }">
        {{ initial }}
      </span>
      <span class="inst-card__title">
        <span class="inst-card__name" :title="item.name">{{ item.name }}</span>
        <span class="inst-card__type">
          {{ typeLabel || '—' }}
          <span v-if="item.autoStart" class="inst-card__badge">{{ t('instance.autoStart') }}</span>
        </span>
      </span>
      <StatusPill :variant="statusVariant" :label="t(statusMeta[item.status]?.labelKey || 'instance.unknownStatus')" />
      <input
        type="checkbox"
        class="inst-card__check"
        :checked="selected"
        :aria-label="item.name"
        @change="emit('toggleSelect', ($event.target as HTMLInputElement).checked)"
      />
    </header>

    <p class="inst-card__remark">{{ item.remark || t('instance.noRemark') }}</p>

    <dl class="inst-card__info">
      <div>
        <dt>{{ isRunningLike ? t('instance.startedAt') : t('instance.createTime') }}</dt>
        <dd class="m-num">{{ isRunningLike ? fmtTime(item.startTime) : fmtTime(item.createTime) }}</dd>
      </div>
      <div>
        <dt>{{ t('instance.autoStart') }}</dt>
        <dd>{{ item.autoStart ? t('common.yes') : t('common.no') }}</dd>
      </div>
      <div class="is-wide">
        <dt>{{ t('instance.startCommand') }}</dt>
        <dd class="m-mono" :title="item.startCommand">{{ item.startCommand || '—' }}</dd>
      </div>
      <div class="is-wide">
        <dt>{{ t('instance.instancePath') }}</dt>
        <dd class="m-mono" :title="item.instancePath">{{ item.instancePath || '—' }}</dd>
      </div>
    </dl>

    <footer class="inst-card__foot">
      <div class="inst-card__tags">
        <span v-for="tag in tags" :key="tag" class="inst-card__tag">#{{ tag }}</span>
      </div>
      <AppIconButton icon="HOutline:CommandLineIcon" :label="t('instance.console')" @click="emit('console')" />
      <AppIconButton
        icon="HOutline:FolderIcon"
        :label="t('instance.fileManagerTitle')"
        @click="emit('moreAction', 'fileManager')"
      />
      <AppIconButton
        :icon="isRunningLike ? 'HOutline:StopIcon' : 'HOutline:PlayIcon'"
        :label="isRunningLike ? t('instance.stop') : t('instance.start')"
        :class="isRunningLike ? 'is-stop' : 'is-start'"
        @click="
          emit(
            'changeStatus',
            isRunningLike ? InstanceStatus.INSTANCE_STATUS_STOPPED : InstanceStatus.INSTANCE_STATUS_RUNNING,
          )
        "
      />
      <ActionMenu :items="moreItems" @select="onMore" />
    </footer>
  </article>
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

const { t } = useI18n()

const TONES = ['p', 'a2', 'a', 'ok']
const ROTS = ['-4deg', '3deg', '-2deg', '4deg']
// 按 id 稳定取色，同一个应用每次都是同一种贴纸色
const hash = computed(() => [...(props.item.id || '')].reduce((n, c) => (n * 31 + c.charCodeAt(0)) >>> 0, 7))
const tone = computed(() => TONES[hash.value % TONES.length])
const rot = computed(() => ROTS[hash.value % ROTS.length])
const initial = computed(() => (props.item.name || '?').trim().charAt(0).toUpperCase())

const tags = computed(() =>
  (props.item.tags || '')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
    .slice(0, 3),
)

const isRunningLike = computed(() => props.item.status === InstanceStatus.INSTANCE_STATUS_RUNNING)

const statusVariantMap: Record<InstanceStatus, 'success' | 'neutral' | 'warning' | 'error'> = {
  [InstanceStatus.INSTANCE_STATUS_RUNNING]: 'success',
  [InstanceStatus.INSTANCE_STATUS_STOPPED]: 'neutral',
  [InstanceStatus.INSTANCE_STATUS_MAINTENANCE]: 'warning',
  [InstanceStatus.INSTANCE_STATUS_UNSPECIFIED]: 'neutral',
  [InstanceStatus.UNRECOGNIZED]: 'error',
}
const statusVariant = computed(() => statusVariantMap[props.item.status] || 'neutral')

const moreItems = computed<ActionMenuItem[]>(() => [
  { key: 'config', label: t('instance.config'), icon: 'HOutline:Cog6ToothIcon' },
  { key: 'forceRestart', label: t('instance.forceRestart'), icon: 'HOutline:ArrowPathIcon' },
  {
    key: 'delete',
    label: t('common.delete'),
    icon: 'HOutline:TrashIcon',
    danger: true,
    hidden: !props.canDelete,
  },
])

const onMore = (key: string) => {
  if (key === 'config') emit('config')
  else emit('moreAction', key as MoreAction)
}

const fmtTime = (value: unknown) => {
  if (!value) return '—'
  const date = dayjs(value as string | Date)
  return date.isValid() ? date.format('MM-DD HH:mm') : '—'
}
</script>

<style scoped lang="scss">
.inst-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 18px 14px 6px 18px;
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
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
.inst-card__badge {
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
.inst-card__check {
  width: 20px;
  height: 20px;
  flex: none;
  accent-color: var(--m-p);
  cursor: pointer;
}
.inst-card__remark {
  padding-right: 4px;
  font-size: 0.875rem;
  line-height: 1.45;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  font-weight: 600;
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
</style>
