<!-- 系统实时流水日志 -->
<template>
  <BaseCard :title="t('dashboard.monitor.realtimeLogs')" title-icon="HOutline:DocumentTextIcon">
    <div
      class="bg-(--el-bg-color-page) p-3 rounded-2xl border border-(--el-border-color-extra-light)"
    >
      <el-scrollbar :height="280">
        <div
          v-for="(log, index) in logs"
          :key="index"
          class="flex items-center gap-3 text-sm text-(--el-text-color-secondary) hover:bg-(--el-bg-color-overlay) p-1.5 rounded-lg cursor-pointer"
        >
          <span class="shrink-0">{{ log.time }}</span>
          <span
            class="text-xs text-white font-medium px-2 py-1 rounded-md uppercase"
            :class="tags(log.level)"
            >{{ log.level }}</span
          >
          <span class="text-(--el-text-color-regular)">{{ t(log.contentKey) }}</span>
        </div>
      </el-scrollbar>
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useDashboardMonitorStore } from '@/stores/dashboard/monitor'
import { useI18n } from 'vue-i18n'

const dashboardMonitorStore = useDashboardMonitorStore()
const { logs } = storeToRefs(dashboardMonitorStore)
const { appendLog } = dashboardMonitorStore
const { t } = useI18n()

// 定时器
let timer: ReturnType<typeof setInterval> | null = null

const tags = (level: 'INFO' | 'WARN' | 'ERROR') => {
  switch (level) {
    case 'INFO':
      return 'bg-(--el-color-success)'
    case 'WARN':
      return 'bg-(--el-color-warning)'
    case 'ERROR':
      return 'bg-(--el-color-danger)'
    default:
      return 'bg-(--el-color-primary)'
  }
}

onMounted(() => {
  timer = setInterval(() => {
    appendLog()
  }, 2000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped lang="scss"></style>
