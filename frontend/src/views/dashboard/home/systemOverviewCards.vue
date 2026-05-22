<template>
  <el-row :gutter="20">
    <el-col :xs="24" :sm="12" :lg="6" class="mb-5 lg:mb-0">
      <el-card class="stat-card" shadow="never">
        <div class="flex items-center gap-3 mb-3">
          <div class="stat-icon" style="background-color: #6366f110">
            <el-icon size="20" color="#6366f1">
              <component :is="menuStore.iconComponents['HOutline:BoltIcon']" />
            </el-icon>
          </div>
          <span class="text-sm font-semibold text-(--el-text-color-secondary)">CPU</span>
        </div>
        <div class="flex items-baseline gap-2 mb-1">
          <span class="text-3xl font-extrabold text-(--el-text-color-primary)">
            {{ (status?.cpu?.totalPercent ?? 0).toFixed(1) }}%
          </span>
        </div>
        <div class="text-xs text-(--el-text-color-secondary) truncate">
          {{ overview?.cpu?.modelName || '--' }}
        </div>
        <div class="text-xs text-(--el-text-color-placeholder) mt-1">
          {{ overview?.cpu?.logicalCount ?? '--' }} 逻辑核心
        </div>
      </el-card>
    </el-col>

    <el-col :xs="24" :sm="12" :lg="6" class="mb-5 lg:mb-0">
      <el-card class="stat-card" shadow="never">
        <div class="flex items-center gap-3 mb-3">
          <div class="stat-icon" style="background-color: #10b98110">
            <el-icon size="20" color="#10b981">
              <component :is="menuStore.iconComponents['Element:Monitor']" />
            </el-icon>
          </div>
          <span class="text-sm font-semibold text-(--el-text-color-secondary)">内存</span>
        </div>
        <div class="flex items-baseline gap-2 mb-1">
          <span class="text-3xl font-extrabold text-(--el-text-color-primary)">
            {{ (status?.memory?.physicalMemory?.usedPercent ?? 0).toFixed(1) }}%
          </span>
        </div>
        <div class="text-xs text-(--el-text-color-secondary)">
          {{ formatBytes(status?.memory?.physicalMemory?.usedBytes) }} / {{ formatBytes(status?.memory?.physicalMemory?.totalBytes) }}
        </div>
        <div class="text-xs text-(--el-text-color-placeholder) mt-1">
          可用 {{ formatBytes(status?.memory?.physicalMemory?.availableBytes) }}
        </div>
      </el-card>
    </el-col>

    <el-col :xs="24" :sm="12" :lg="6" class="mb-5 lg:mb-0">
      <el-card class="stat-card" shadow="never">
        <div class="flex items-center gap-3 mb-3">
          <div class="stat-icon" style="background-color: #f59e0b10">
            <el-icon size="20" color="#f59e0b">
              <component :is="menuStore.iconComponents['HOutline:GlobeAltIcon']" />
            </el-icon>
          </div>
          <span class="text-sm font-semibold text-(--el-text-color-secondary)">网络</span>
        </div>
        <div class="flex flex-col gap-1 mb-1">
          <div class="flex items-center gap-2">
            <span class="text-xs text-(--el-color-success)">↓</span>
            <span class="text-lg font-extrabold text-(--el-text-color-primary)">
              {{ formatBytes(status?.network?.total?.downloadRateBytesPerSecond) }}/s
            </span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-xs text-(--el-color-primary)">↑</span>
            <span class="text-lg font-extrabold text-(--el-text-color-primary)">
              {{ formatBytes(status?.network?.total?.uploadRateBytesPerSecond) }}/s
            </span>
          </div>
        </div>
        <div class="text-xs text-(--el-text-color-placeholder) mt-1">
          {{ overview?.networkInterfaces?.length ?? 0 }} 个网卡
        </div>
      </el-card>
    </el-col>

    <el-col :xs="24" :sm="12" :lg="6">
      <el-card class="stat-card" shadow="never">
        <div class="flex items-center gap-3 mb-3">
          <div class="stat-icon" style="background-color: #ef444410">
            <el-icon size="20" color="#ef4444">
              <component :is="menuStore.iconComponents['HOutline:FolderIcon']" />
            </el-icon>
          </div>
          <span class="text-sm font-semibold text-(--el-text-color-secondary)">磁盘</span>
        </div>
        <div class="flex items-baseline gap-2 mb-1">
          <span class="text-3xl font-extrabold text-(--el-text-color-primary)">
            {{ (status?.disk?.total?.usedPercent ?? 0).toFixed(1) }}%
          </span>
        </div>
        <div class="text-xs text-(--el-text-color-secondary)">
          {{ formatBytes(status?.disk?.total?.usedBytes) }} / {{ formatBytes(status?.disk?.total?.totalBytes) }}
        </div>
        <div class="text-xs text-(--el-text-color-placeholder) mt-1">
          {{ overview?.diskPartitions?.length ?? 0 }} 个分区
        </div>
      </el-card>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import type { SystemOverviewResponse, SystemStatusResponse } from '@/types/v1/system'

defineProps<{
  overview: SystemOverviewResponse | null
  status: SystemStatusResponse | null
}>()

const menuStore = useMenuStore()

const formatBytes = (bytes?: number | string) => {
  const num = Number(bytes)
  if (!num || num <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const base = 1024
  const i = Math.min(Math.floor(Math.log(num) / Math.log(base)), units.length - 1)
  return (num / base ** i).toFixed(i === 0 ? 0 : 1) + ' ' + units[i]
}
</script>

<style scoped lang="scss">
.stat-card {
  border-radius: 1rem;
  border: none;

  .stat-icon {
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 12px;
  }
}
</style>
