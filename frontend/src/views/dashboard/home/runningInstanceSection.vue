<!-- 运行中实例：贴纸头像 + 名称/类型/启动时间的紧凑行，点击进入控制台。 -->
<template>
  <AppPanel :title="t('dashboard.home.runningInstances')" title-icon="HOutline:ServerStackIcon">
    <template #actions>
      <StatusPill v-if="runningInstanceTotal" variant="success" :label="String(runningInstanceTotal)" />
      <button type="button" class="ri__all" @click="router.push('/instance/list')">
        {{ t('dashboard.home.viewAll') }}
        <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" />
      </button>
    </template>

    <div v-if="runningInstancesLoading" class="ri__list">
      <div v-for="i in 3" :key="i" class="ri__skeleton" />
    </div>

    <EmptyState
      v-else-if="!runningInstances.length"
      icon="HOutline:ServerStackIcon"
      :title="t('dashboard.home.noRunningInstances')"
      :description="t('dashboard.home.noRunningInstancesDesc')"
    >
      <template #action>
        <UButton color="primary" @click="router.push('/instance/list')">
          {{ t('dashboard.home.shortcuts.apps') }}
        </UButton>
      </template>
    </EmptyState>

    <div v-else class="ri__list">
      <button
        v-for="(instance, i) in runningInstances"
        :key="instance.id"
        type="button"
        class="ri__row"
        @click="openConsole(instance)"
      >
        <span
          class="ri__av m-sticker"
          :class="`m-sticker--${TONES[i % TONES.length]}`"
          :style="{ transform: `rotate(${ROTS[i % ROTS.length]})` }"
        >
          {{ initialOf(instance.name) }}
        </span>
        <span class="ri__text">
          <span class="ri__name">{{ instance.name }}</span>
          <span class="ri__sub">
            {{ instance.type || '—' }} · {{ startedText(instance) }}
            <template v-for="tag in tagsOf(instance)" :key="tag"> · #{{ tag }}</template>
          </span>
        </span>
        <span class="ri__go" aria-hidden="true">
          <component :is="menuStore.iconComponents['HOutline:CommandLineIcon']" />
        </span>
      </button>
    </div>
  </AppPanel>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import dayjs from 'dayjs'
import { useDashboardHomeStore } from '@/stores/dashboard/home'
import type { InstanceInfo } from '@/types/v1/instance'
import { useI18n } from 'vue-i18n'

const TONES = ['p', 'a2', 'a', 'ok']
const ROTS = ['-4deg', '3deg', '-2deg', '4deg']

const router = useRouter()
const menuStore = useMenuStore()
const dashboardHomeStore = useDashboardHomeStore()
const { t } = useI18n()
const { runningInstancesLoading, runningInstances, runningInstanceTotal } = storeToRefs(dashboardHomeStore)
const { getRunningInstances } = dashboardHomeStore

const initialOf = (name?: string) => (name || '?').trim().charAt(0).toUpperCase()

const tagsOf = (instance: InstanceInfo) =>
  (instance.tags || '')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
    .slice(0, 2)

const startedText = (instance: InstanceInfo) => {
  const d = instance.startTime || instance.createTime
  return d ? dayjs(d).format('MM-DD HH:mm') : '—'
}

const openConsole = (instance: InstanceInfo) => {
  router.push({ path: `/instance/app/${instance.id}`, query: { tabTitle: instance.name } })
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
  color: var(--m-pInk);
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
}
.ri__all :deep(svg) {
  width: 15px;
  height: 15px;
}
.ri__list {
  display: grid;
  grid-template-columns: 1fr;
  gap: 4px;
  margin: 0 -10px;
}
@media (width >= 1100px) {
  .ri__list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
.ri__row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 60px;
  padding: 8px 10px;
  border: none;
  border-radius: 16px;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s;
}
.ri__row:hover {
  background: var(--el-fill-color-light);
}
.ri__av {
  width: 42px;
  height: 42px;
  border-radius: 14px;
  font-size: 1.05rem;
}
.ri__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ri__name {
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ri__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ri__go {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  flex: none;
  border-radius: 14px;
  color: var(--el-text-color-secondary);
}
.ri__row:hover .ri__go {
  color: var(--m-pInk);
  background: var(--m-pSoft);
}
.ri__go :deep(svg) {
  width: 20px;
  height: 20px;
}
.ri__skeleton {
  height: 60px;
  border-radius: 16px;
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
</style>
