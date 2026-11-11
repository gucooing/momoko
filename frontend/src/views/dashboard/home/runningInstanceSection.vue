<!-- 运行中（DashboardPC / Mobile）：贴纸头像 + 名称 / 类型 · 已运行时长，行尾「控制台」与「停止」小图标按钮。 -->
<template>
  <section class="ri" aria-labelledby="ri-h">
    <div class="ri__head">
      <h2 id="ri-h" class="ri__title">{{ t('dashboard.home.running') }}</h2>
      <StatusPill
        v-if="runningInstanceTotal"
        variant="success"
        :label="t('dashboard.home.countUnit', { n: runningInstanceTotal })"
      />
      <span class="ri__spacer" />
      <RouterLink to="/instance/list" class="ri__all">
        {{ compact ? t('dashboard.home.allShort') : t('dashboard.home.allApps') }} →
      </RouterLink>
    </div>

    <div v-if="runningInstancesLoading && !runningInstances.length" class="ri__list">
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
      <div v-for="(instance, i) in shown" :key="instance.id" class="ri__row">
        <span
          class="ri__av m-sticker"
          :class="`m-sticker--${TONES[i % TONES.length]}`"
          aria-hidden="true"
        >
          {{ initialOf(instance.name) }}
        </span>
        <span class="ri__text">
          <span class="ri__name">{{ instance.name }}</span>
          <span class="ri__sub">{{ subOf(instance) }}</span>
        </span>
        <AppIconButton
          icon="HOutline:CommandLineIcon"
          :label="t('dashboard.home.openConsoleOf', { name: instance.name })"
          :box="compact ? 44 : 36"
          :size="compact ? 20 : 16"
          @click="openConsole(instance)"
        />
        <AdaptiveConfirm
          :title="t('dashboard.home.stopConfirm', { name: instance.name })"
          :disabled="stopping.has(instance.id)"
          @confirm="stop(instance)"
        >
          <template #reference>
            <AppIconButton
              class="ri__stop"
              :icon="stopping.has(instance.id) ? 'HOutline:ArrowPathIcon' : 'HOutline:StopIcon'"
              :label="t('dashboard.home.stopOf', { name: instance.name })"
              :box="compact ? 44 : 36"
              :size="compact ? 20 : 16"
              :class="{ 'is-busy': stopping.has(instance.id) }"
            />
          </template>
        </AdaptiveConfirm>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { stopInstanceRequest } from '@/api/instance'
import { useDashboardHomeStore } from '@/stores/dashboard/home'
import type { InstanceInfo } from '@/types/v1/instance'
import { getRequestErrorMessage } from '@/utils/request'
import { useFeedback } from '@/utils/feedback'
import { useI18n } from 'vue-i18n'

const TONES = ['p', 'a2', 'a', 'ok']

const router = useRouter()
const menuStore = useMenuStore()
const dashboardHomeStore = useDashboardHomeStore()
const { t } = useI18n()
const fb = useFeedback()
const { runningInstancesLoading, runningInstances, runningInstanceTotal } = storeToRefs(dashboardHomeStore)
const { getRunningInstances } = dashboardHomeStore

const compact = computed(() => menuStore.isMobile)
const shown = computed(() => runningInstances.value.slice(0, compact.value ? 3 : 5))
const stopping = ref<Set<string>>(new Set())

const initialOf = (name?: string) => (name || '?').trim().charAt(0).toUpperCase()

/** 已运行时长：x 天 / x 小时 / x 分钟 */
const uptimeOf = (instance: InstanceInfo) => {
  const raw = instance.startTime as unknown as string | Date | undefined
  if (!raw) return ''
  const ms = Date.now() - new Date(raw).getTime()
  if (!Number.isFinite(ms) || ms < 0) return ''
  const min = Math.floor(ms / 60000)
  const days = Math.floor(min / 1440)
  if (days > 0) return t('dashboard.home.upDays', { n: days })
  const hours = Math.floor(min / 60)
  if (hours > 0) return t('dashboard.home.upHours', { n: hours })
  return t('dashboard.home.upMinutes', { n: Math.max(1, min) })
}

const subOf = (instance: InstanceInfo) => {
  const up = uptimeOf(instance)
  const type = instance.type || '—'
  if (!up) return type
  return `${type} · ${compact.value ? up : t('dashboard.home.upFor', { time: up })}`
}

const openConsole = (instance: InstanceInfo) => {
  router.push({ path: `/instance/app/${instance.id}`, query: { tabTitle: instance.name } })
}

const stop = async (instance: InstanceInfo) => {
  const next = new Set(stopping.value)
  next.add(instance.id)
  stopping.value = next
  try {
    await stopInstanceRequest({ id: instance.id, force: false })
    fb.success(t('dashboard.home.stopped', { name: instance.name }))
    await getRunningInstances()
  } catch (error) {
    fb.error(getRequestErrorMessage(error, t('dashboard.home.stopFailed')))
  } finally {
    const done = new Set(stopping.value)
    done.delete(instance.id)
    stopping.value = done
  }
}

onMounted(() => {
  void getRunningInstances()
})
</script>

<style scoped lang="scss">
.ri {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  padding: calc(var(--cmp-card-unit) * 22);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.ri__head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}
.ri__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.ri__spacer {
  flex: 1;
}
.ri__all {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--m-pInk);
  text-decoration: none;
  white-space: nowrap;
}
.ri__all:hover {
  color: var(--el-text-color-primary);
}
.ri__list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ri__row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 10px;
  margin: 0 -10px;
  border-radius: 16px;
  transition: background 0.15s;
}
.ri__row:hover {
  background: var(--el-fill-color-light);
}
.ri__av {
  width: 40px;
  height: 40px;
  border-radius: 13px;
  font-size: 1.0625rem;
}
.ri__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.ri__name {
  font-size: 0.875rem;
  font-weight: 700;
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
.ri__stop:hover {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.ri__stop.is-busy :deep(svg) {
  animation: ri-spin 0.9s linear infinite;
}
@keyframes ri-spin {
  to {
    transform: rotate(360deg);
  }
}
.ri__skeleton {
  height: 56px;
  border-radius: 16px;
  background: var(--el-fill-color-light);
  animation: ri-pulse 1.4s ease-in-out infinite;
}
@keyframes ri-pulse {
  50% {
    opacity: 0.55;
  }
}
@media (width < 640px) {
  .ri {
    padding: 16px;
    gap: 4px;
  }
  .ri__title {
    font-size: 1rem;
  }
  .ri__row {
    padding: 6px 0;
    margin: 0;
  }
  .ri__row:hover {
    background: transparent;
  }
}
</style>
