<!-- 运行中实例：AppPanel + EntityCard 卡片流；空态 EmptyState 引导（06a）。 -->
<template>
  <AppPanel :title="t('dashboard.home.runningInstances')" title-icon="HOutline:ServerStackIcon">
    <template #actions>
      <button type="button" class="ri__all" @click="router.push('/instance/list')">
        {{ t('dashboard.home.viewAll') }}
        <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" />
      </button>
    </template>

    <div v-if="runningInstancesLoading" class="ri__grid">
      <div v-for="i in 3" :key="i" class="ri__skeleton" />
    </div>

    <EmptyState
      v-else-if="!runningInstances.length"
      icon="HOutline:ServerStackIcon"
      :title="t('dashboard.home.noRunningInstances')"
      :description="t('dashboard.home.noRunningInstancesDesc')"
    >
      <template #action>
        <UButton color="primary" variant="soft" size="sm" @click="router.push('/instance/list')">
          {{ t('dashboard.home.shortcuts.apps') }}
        </UButton>
      </template>
    </EmptyState>

    <div v-else class="ri__grid">
      <EntityCard
        v-for="instance in runningInstances"
        :key="instance.id"
        clickable
        @click="openConsole(instance)"
      >
        <template #title>{{ instance.name }}</template>
        <template #status>
          <StatusPill variant="success" :label="t('dashboard.home.running')" />
        </template>
        <template #meta>
          <span class="ri__type">{{ instance.type || '—' }}</span>
          <span v-for="tag in tagsOf(instance)" :key="tag" class="ri__tag">{{ tag }}</span>
        </template>
        <template #footer>
          <span>{{ startedText(instance) }}</span>
          <component :is="menuStore.iconComponents['HOutline:ArrowRightIcon']" class="ri__arrow" />
        </template>
      </EntityCard>
    </div>
  </AppPanel>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import dayjs from 'dayjs'
import { useDashboardHomeStore } from '@/stores/dashboard/home'
import type { InstanceInfo } from '@/types/v1/instance'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const menuStore = useMenuStore()
const dashboardHomeStore = useDashboardHomeStore()
const { t } = useI18n()
const { runningInstancesLoading, runningInstances } = storeToRefs(dashboardHomeStore)
const { getRunningInstances } = dashboardHomeStore

const tagsOf = (instance: InstanceInfo) =>
  (instance.tags || '')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
    .slice(0, 3)

const startedText = (instance: InstanceInfo) => {
  const d = instance.startTime || instance.createTime
  return d ? dayjs(d).format('MM-DD HH:mm') : '—'
}

const openConsole = (instance: InstanceInfo) => {
  router.push({ path: `/instance/console/${instance.id}`, query: { tabTitle: instance.name } })
}

onMounted(() => {
  void getRunningInstances()
})
</script>

<style scoped lang="scss">
.ri__all {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  border: none;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.15s;
}
.ri__all:hover {
  color: var(--el-color-primary);
}
.ri__all :deep(svg) {
  width: 15px;
  height: 15px;
}
.ri__grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
}
@media (width >= 560px) {
  .ri__grid {
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  }
}
.ri__skeleton {
  height: 118px;
  border-radius: var(--app-radius-lg);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: ri-shimmer 1.4s ease-in-out infinite;
}
@keyframes ri-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
.ri__type {
  color: var(--el-text-color-regular);
  font-weight: 500;
}
.ri__tag {
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
}
.ri__arrow {
  width: 15px;
  height: 15px;
  color: var(--el-text-color-placeholder);
}
</style>
