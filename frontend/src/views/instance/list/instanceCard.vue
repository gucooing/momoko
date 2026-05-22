<template>
  <BaseCard :class="['instance-card', { 'is-selected': selected }]">
    <div class="instance-card-shell">
      <el-checkbox
        class="instance-select"
        :model-value="selected"
        @change="emit('toggleSelect', Boolean($event))"
      />

      <div class="instance-header">
        <div class="instance-main">
            <div class="instance-heading">
              <div class="instance-name">{{ item.name }}</div>
              <div class="instance-tag-row">
                <span class="type-pill">{{ typeLabel }}</span>
              </div>
            </div>
        </div>

        <div class="instance-head-meta">
          <span class="provider-pill" :class="statusClassMap[item.status]">
            {{ statusMeta[item.status].label }}
          </span>
        </div>
      </div>

      <div class="instance-content">
        <p class="instance-description">{{ item.remark || '暂无备注' }}</p>
        <p class="instance-detail">创建: {{ formatDateTime(item.createTime) }}</p>
        <p class="instance-detail">路径: {{ item.instancePath || '-' }}</p>
      </div>

      <div class="instance-footer">
        <button type="button" class="footer-action" @click="emit('console')">控制台</button>
        <button type="button" class="footer-action" @click="emit('config')">配置</button>
        <div class="footer-action-wrap">
          <el-popover
            placement="bottom-end"
            trigger="click"
            :width="176"
            popper-class="instance-more-actions-popper"
          >
            <template #reference>
              <button type="button" class="footer-action">更多</button>
            </template>
            <div class="instance-more-actions">
              <button
                type="button"
                class="popover-text-action"
                @click="emit('moreAction', 'forceRestart')"
              >
                强制重启
              </button>
              <button
                type="button"
                class="popover-text-action"
                @click="emit('moreAction', 'fileManager')"
              >
                文件管理
              </button>
              <button
                v-if="canDelete"
                type="button"
                class="popover-text-action"
                @click="emit('moreAction', 'delete')"
              >
                删除
              </button>
            </div>
          </el-popover>
        </div>

        <button
          type="button"
          class="footer-action footer-action--strong"
          @click="
            emit(
              'changeStatus',
              isRunningLike
                ? InstanceStatus.INSTANCE_STATUS_STOPPED
                : InstanceStatus.INSTANCE_STATUS_RUNNING,
            )
          "
        >
          {{ isRunningLike ? '停止' : '启动' }}
        </button>
      </div>
    </div>
  </BaseCard>
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

const isRunningLike = computed(
  () => props.item.status === InstanceStatus.INSTANCE_STATUS_RUNNING,
)

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
  --el-card-bg-color: color-mix(
    in srgb,
    var(--el-fill-color-light) 42%,
    var(--el-bg-color-overlay)
  );
  --instance-card-padding: 1rem;
  --instance-action-color: #111111;
  --instance-action-hover-bg: rgb(15 23 42 / 5%);
  border: 1px solid color-mix(in srgb, var(--el-border-color-extra-light) 86%, #d7dce5);
  background: var(--el-card-bg-color);
  box-shadow: 0 1px 2px rgb(15 23 42 / 4%);
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease,
    transform 0.2s ease;
}

.instance-card:hover {
  border-color: var(--el-border-color);
  box-shadow: 0 8px 20px rgb(15 23 42 / 6%);
}

.instance-card.is-selected {
  border-color: color-mix(in srgb, var(--el-color-primary) 18%, var(--el-border-color));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 10%, transparent);
}

.instance-card :deep(.el-card__body) {
  padding: var(--instance-card-padding) var(--instance-card-padding) 0;
}

.instance-card-shell {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 0.8rem;
}

.instance-select {
  position: absolute;
  top: 0;
  right: 0;
  z-index: 1;
}

.instance-select :deep(.el-checkbox__inner) {
  border-radius: 4px;
}

.instance-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  min-width: 0;
  padding-right: 1.5rem;
}

.instance-main {
  display: flex;
  align-items: flex-start;
  min-width: 0;
  gap: 0;
}

.instance-heading {
  min-width: 0;
}

.instance-name {
  max-width: 100%;
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.instance-tag-row {
  margin-top: 0.25rem;
}

.type-pill {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: 0.2rem 0.55rem;
  border: 1px solid transparent;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 0.71rem;
  font-weight: 700;
  line-height: 1;
}

.instance-head-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.28rem;
  flex-shrink: 0;
}

.provider-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.32rem;
  font-size: 0.72rem;
  line-height: 1;
}

.provider-pill::before {
  content: '';
  width: 0.38rem;
  height: 0.38rem;
  border-radius: 999px;
  background: currentColor;
}

.provider-pill {
  font-weight: 700;
  color: var(--el-text-color-regular);
}

.provider-pill.is-running {
  color: #059669;
}

.provider-pill.is-maintenance {
  color: #d97706;
}

.provider-pill.is-unrecognized {
  color: #dc2626;
}

.provider-pill.is-stopped {
  color: #64748b;
}

.provider-pill.is-unspecified {
  color: #64748b;
}

.instance-content {
  display: flex;
  flex-direction: column;
  gap: 0.36rem;
}

.instance-description {
  margin: 0;
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 0.82rem;
  line-height: 1.45;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.instance-detail {
  margin: 0;
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 0.8rem;
  line-height: 1.35;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.instance-footer {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  align-items: stretch;
  border-top: 1px solid var(--el-border-color-extra-light);
  margin: 0 calc(var(--instance-card-padding) * -1);
  margin-top: 0.1rem;
}

.footer-action-wrap {
  min-width: 0;
}

.footer-action,
.footer-action-wrap :deep(.el-popover__reference) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  width: 100%;
  border: none;
  padding: 0.9rem 0.45rem;
  background: transparent;
  color: var(--instance-action-color);
  font-size: 0.82rem;
  line-height: 1.2;
  font-weight: 500;
  cursor: pointer;
  transition:
    opacity 0.2s ease,
    background-color 0.2s ease;
}

.footer-action-wrap :deep(.el-popover__reference) {
  display: flex;
}

.footer-action:hover,
.popover-text-action:hover {
  background: var(--instance-action-hover-bg);
}

.footer-action:focus-visible,
.popover-text-action:focus-visible {
  outline: none;
  background: var(--instance-action-hover-bg);
}

.footer-action--strong {
  font-weight: 700;
}

.instance-more-actions {
  display: grid;
  gap: 0.1rem;
}

.popover-text-action {
  width: 100%;
  border: none;
  border-radius: 0.62rem;
  padding: 0.5rem 0.65rem;
  background: transparent;
  color: var(--instance-action-color);
  font-size: 0.82rem;
  line-height: 1.2;
  text-align: left;
  cursor: pointer;
  transition:
    background-color 0.2s ease,
    opacity 0.2s ease;
}

.popover-text-action:hover {
  opacity: 1;
}

:global(html.dark .instance-card) {
  --el-card-bg-color: color-mix(in srgb, #ffffff 4%, var(--el-bg-color));
  --instance-action-color: #ffffff;
  --instance-action-hover-bg: rgb(255 255 255 / 8%);
  border-color: color-mix(in srgb, #ffffff 10%, var(--el-border-color));
}

:global(html.dark .instance-card:hover) {
  border-color: color-mix(in srgb, #ffffff 12%, var(--el-border-color));
  box-shadow: 0 10px 24px rgb(0 0 0 / 14%);
}

:global(html.dark .instance-card.is-selected) {
  border-color: color-mix(in srgb, var(--el-color-primary) 28%, var(--el-border-color));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 22%, transparent);
}

:global(html.dark .instance-card .type-pill) {
  background: color-mix(in srgb, #ffffff 8%, #1f2937);
  color: rgb(255 255 255 / 88%);
}

:global(.instance-more-actions-popper.el-popper) {
  border: 1px solid var(--el-border-color-extra-light);
  background: #ffffff;
  border-radius: 0.85rem;
  padding: 0.4rem;
  box-shadow: 0 12px 28px rgb(15 23 42 / 14%);
}

:global(html.dark .instance-more-actions-popper.el-popper) {
  border: 1px solid color-mix(in srgb, var(--el-border-color) 78%, transparent);
  background: color-mix(in srgb, var(--el-bg-color) 92%, #0b1220);
  box-shadow: 0 16px 30px rgb(0 0 0 / 28%);
}

@media (width <= 768px) {
  .instance-card {
    --instance-card-padding: 0.85rem;
  }

  .instance-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.52rem;
    padding-right: 1.4rem;
  }

  .instance-head-meta {
    flex-direction: row;
    align-items: center;
    gap: 0.7rem;
  }

  .instance-footer {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 480px) {
  .instance-footer {
    grid-template-columns: 1fr;
  }
}
</style>
