<!-- Docker 四个标签共用的页头：「Docker」+ 引擎状态 pill + 一句统计说明；右侧「任务」（进行中数量角标）+ 本页主操作。
     手机：只保留任务图标（有进行中任务时带圆点），主操作由各页的右下悬浮按钮承担。 -->
<template>
  <PageHeader class="dk-hdr">
    <template #title>{{ t('pageGroup.docker.label') }}<DockerEnginePill /></template>
    <template v-if="$slots.description" #description><slot name="description" /></template>
    <template #actions>
      <template v-if="!menuStore.isMobile">
        <UButton color="neutral" variant="outline" icon="i-lucide-clock" @click="emit('tasks')">
          {{ t('docker.common.tasks') }}
          <span v-if="active.length" class="dk-hdr__n m-num">{{ active.length }}</span>
        </UButton>
        <slot name="actions" />
      </template>
      <template v-else>
        <slot name="mobile-actions" />
        <AppIconButton
          class="dk-hdr__ib"
          icon="HOutline:ClockIcon"
          :label="active.length ? t('docker.page.tasksActive', { n: active.length }) : t('docker.common.tasks')"
          :box="44"
          :size="22"
          @click="emit('tasks')"
        >
          <span v-if="active.length" class="dk-hdr__dot" />
        </AppIconButton>
      </template>
    </template>
  </PageHeader>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import DockerEnginePill from './DockerEnginePill.vue'
import { useDockerTasks } from './useDockerTasks'

const emit = defineEmits<{ tasks: [] }>()
const { t } = useI18n()
const menuStore = useMenuStore()
const { active } = useDockerTasks()
</script>

<style scoped lang="scss">
.dk-hdr__n {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  margin-left: 2px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--m-a);
  color: var(--m-aOn, var(--m-ink));
  font-size: 0.75rem;
  font-weight: 800;
}
.dk-hdr :deep(.page-header__actions .dk-hdr__ib) {
  position: relative;
  width: 44px;
  height: 44px;
  min-height: 44px;
  padding: 0;
  border-radius: 14px;
}
.dk-hdr__dot {
  position: absolute;
  top: 9px;
  right: 9px;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--m-a);
  border: 2px solid var(--el-bg-color);
}
</style>
