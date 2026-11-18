<!-- 运营管理里一项配置的编辑窗口（卡池、活动、竞赛赛季、登录奖励、公告共用这一个外壳）：
     顶部状态、标签与右上角菜单（复制 / 删除，删除先确认），中间是这一项的表单；草稿里删掉的显示「撤销删除」，找不到时显示提示。
     改动直接进草稿，底部「完成」关掉窗口，在页面底部的发布条发布。 -->
<template>
  <FormDialog :model-value="modelValue" :title="title" :width="width" @update:model-value="emit('update:modelValue', $event)">
    <div v-if="state === 'ok'" class="gid game-skin" :data-game="ctx.game.value || undefined">
      <div class="gid__head">
        <StatusPill v-if="status" :variant="periodVariant(status)" :label="statusLabel || t(`gmConfig.ops.status.${status}`)" />
        <span v-for="tag in tags" :key="tag" class="gmc-tag gmc-tag--g">{{ tag }}</span>
        <span v-if="draft" class="gmc-tag gmc-tag--warn">{{ t(`gmConfig.ops.draft.${draft}`) }}</span>
        <span class="gid__spacer" />
        <ActionMenu v-if="menu.length" :items="menu" @select="onMenu" />
      </div>
      <slot />
    </div>
    <div v-else-if="state === 'removed'" class="gid__removed">
      <p class="gmc-note"><UIcon name="i-lucide-info" class="gmc-note__icon" />{{ t('gmConfig.ops.removedHint') }}</p>
      <UButton class="gmc-btn" icon="i-lucide-undo-2" @click="emit('restore')">{{ t('gmConfig.ops.restore') }}</UButton>
    </div>
    <EmptyState v-else icon="HOutline:QuestionMarkCircleIcon" :title="t('gmConfig.ops.notFound')" />

    <template #footer="{ close }">
      <slot name="actions" />
      <span class="gid__hint">{{ t('gmConfig.ops.autosaveHint') }}</span>
      <UButton class="gmc-btn" color="primary" @click="close">{{ t('gmConfig.ops.done') }}</UButton>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Dialog } from '@/utils/dialog'
import { useGmContext } from '../../../context'
import { periodVariant, type DraftState, type PeriodStatus } from './entry'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title: string
    width?: number
    /** ok 正常编辑 / removed 草稿里删掉了 / missing 找不到 */
    state: 'ok' | 'removed' | 'missing'
    status?: PeriodStatus
    /** 状态的文字（不填时按 status） */
    statusLabel?: string
    tags?: string[]
    draft?: DraftState
    /** 菜单里「复制」的文字；不填时没有复制 */
    copyLabel?: string
    /** 菜单里有「删除」（确认后发出 delete 并关掉窗口） */
    deletable?: boolean
  }>(),
  { width: 760, status: undefined, statusLabel: '', tags: () => [], draft: undefined, copyLabel: '', deletable: true },
)
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; copy: []; delete: []; restore: [] }>()

const { t } = useI18n()
const ctx = useGmContext()

const menu = computed(() => [
  ...(props.copyLabel ? [{ key: 'copy', label: props.copyLabel, icon: 'i-lucide-copy' }] : []),
  ...(props.deletable ? [{ key: 'delete', label: t('gmConfig.ops.delete'), icon: 'i-lucide-trash-2', danger: true }] : []),
])
const onMenu = (key: string) => {
  if (key === 'copy') emit('copy')
  if (key === 'delete') {
    Dialog.confirm({
      title: t('gmConfig.ops.deleteTitle', { name: props.title }),
      content: t('gmConfig.ops.deleteContent'),
      cancelText: t('gmConfig.common.cancel'),
      confirmText: t('gmConfig.ops.delete'),
      onConfirm: () => {
        emit('delete')
        emit('update:modelValue', false)
      },
    })
  }
}
</script>

<style lang="scss" src="./form.scss"></style>
<style scoped lang="scss">
.gid {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gid__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.gid__spacer {
  flex: 1;
}
.gid__removed {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
}
.gid__hint {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
