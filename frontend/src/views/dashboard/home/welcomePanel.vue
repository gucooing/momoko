<template>
  <BaseCard>
    <!-- 欢迎面板 -->
    <el-scrollbar :max-height="620">
      <div class="flex flex-col xl:flex-row justify-between p-6 lg:p-8">
        <div class="flex-1">
          <div class="flex flex-col lg:flex-row items-center lg:items-start xl:items-center gap-6">
            <div class="relative shrink-0">
              <el-avatar :size="110" :src="userStore.resolvedUserAvatar" />
              <div
                class="absolute h-5 w-5 bottom-2 right-2 rounded-full border-3 border-(--el-bg-color) bg-(--el-color-success)"
              ></div>
            </div>

            <div class="flex flex-col gap-4 items-center lg:items-start text-center lg:text-left">
              <h2
                class="flex text-2xl md:text-3xl font-black text-(--el-text-color-primary) cursor-pointer"
              >
                <TextEllipsis
                  :text="`${userStore.userInfo?.name! || userStore.userInfo?.username!}，欢迎回来！`"
                  :clickable="false"
                  class="text-2xl md:text-3xl font-black text-(--el-text-color-primary)"
                />
                <div>👋</div>
              </h2>
              <TextEllipsis
                :text="`“ ${userStore.userInfo?.bio} ”`"
                class="text-(--el-text-color-regular) italic text-sm md:base cursor-pointer"
              />
              <div class="flex flex-wrap justify-center lg:justify-start items-center gap-3">
                <div
                  class="flex items-center gap-2 text-xs font-semibold px-3 py-2 text-(--el-text-color-primary) bg-(--el-bg-color-page) rounded-lg"
                >
                  <el-icon>
                    <component
                      :is="menuStore.iconComponents['Element:Orange']"
                      class="text-orange-500"
                    />
                  </el-icon>
                  <span>{{ weatherText }}</span>
                </div>
                <div
                  class="flex items-center gap-2 text-xs font-semibold px-3 py-2 text-(--el-text-color-primary) bg-(--el-bg-color-page) rounded-lg"
                >
                  <el-icon>
                    <component
                      :is="menuStore.iconComponents['Element:Monitor']"
                      class="text-indigo-500"
                    />
                  </el-icon>
                  <span
                    >{{ userProfileStore.address.country }} · {{ userProfileStore.address.region }} ·
                    {{ userProfileStore.address.city }}</span
                  >
                </div>
                <div
                  class="flex items-center gap-2 text-xs font-semibold px-3 py-2 text-(--el-text-color-primary) bg-(--el-bg-color-page) rounded-lg"
                >
                  <el-icon>
                    <component
                      :is="menuStore.iconComponents['Element:Calendar']"
                      class="text-emerald-500"
                    />
                  </el-icon>
                  <span>{{ currentDate }}</span>
                </div>
              </div>
            </div>
            <div class="flex-1 flex justify-center lg:justify-end xl:justify-center">
              <LottieAnimation
                :path="workTimeLottieUrl"
                :width="180"
                :height="140"
                class="hidden lg:block"
              />
            </div>
          </div>
          <div
            class="flex flex-wrap px-2 md:px-6 py-6 md:py-10 items-center gap-3"
          >
            <div class="sys-info-chip">
              <el-icon size="13">
                <component :is="menuStore.iconComponents['HOutline:ServerStackIcon']" />
              </el-icon>
              <span>{{ overview?.version?.hostname || '--' }}</span>
            </div>
            <div class="sys-info-chip">
              <el-icon size="13">
                <component :is="menuStore.iconComponents['HOutline:Cog6ToothIcon']" />
              </el-icon>
              <span>{{ overview?.version?.os || '--' }} {{ overview?.version?.platformVersion || '' }}</span>
            </div>
            <div class="sys-info-chip">
              <el-icon size="13">
                <component :is="menuStore.iconComponents['HOutline:BoltIcon']" />
              </el-icon>
              <span>{{ overview?.version?.kernelArch || '--' }} · {{ overview?.version?.kernelVersion || '--' }}</span>
            </div>
            <div class="sys-info-chip">
              <el-icon size="13">
                <component :is="menuStore.iconComponents['HOutline:ClockIcon']" />
              </el-icon>
              <span>开机 {{ formatUptime(overview?.uptimeSeconds) }}</span>
            </div>
            <div class="sys-info-chip">
              <el-icon size="13">
                <component :is="menuStore.iconComponents['Element:Calendar']" />
              </el-icon>
              <span>{{ formatBootTime(overview?.bootTime) }}</span>
            </div>
          </div>
        </div>

        <div class="hidden xl:block mx-7 my-6">
          <div class="w-px h-full border-(--el-border-color) border-l"></div>
        </div>

        <div class="flex-1 xl:flex-[0.8] grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div class="sys-stat-item">
            <div class="flex items-center justify-center w-9 h-9 rounded-[10px] p-2 mb-3" style="color: #6366f1; background-color: #6366f110">
              <el-icon size="18">
                <component :is="menuStore.iconComponents['HOutline:BoltIcon']" />
              </el-icon>
            </div>
            <div>
              <div class="text-[13px] font-semibold text-(--el-text-color-secondary) mb-1">CPU 使用率</div>
              <div class="flex items-baseline gap-2">
                <span class="text-[20px] font-extrabold text-(--el-text-color-primary)">
                  {{ (status?.cpu?.totalPercent ?? 0).toFixed(1) }}%
                </span>
              </div>
              <div class="text-[11px] text-(--el-text-color-placeholder) mt-1 truncate">
                {{ overview?.cpu?.modelName || '--' }}
              </div>
            </div>
          </div>
          <div class="sys-stat-item">
            <div class="flex items-center justify-center w-9 h-9 rounded-[10px] p-2 mb-3" style="color: #10b981; background-color: #10b98110">
              <el-icon size="18">
                <component :is="menuStore.iconComponents['Element:Monitor']" />
              </el-icon>
            </div>
            <div>
              <div class="text-[13px] font-semibold text-(--el-text-color-secondary) mb-1">物理内存</div>
              <div class="flex items-baseline gap-2">
                <span class="text-[20px] font-extrabold text-(--el-text-color-primary)">
                  {{ (status?.memory?.physicalMemory?.usedPercent ?? 0).toFixed(1) }}%
                </span>
              </div>
              <div class="text-[11px] text-(--el-text-color-placeholder) mt-1">
                {{ formatBytes(status?.memory?.physicalMemory?.usedBytes) }} / {{ formatBytes(status?.memory?.physicalMemory?.totalBytes) }}
              </div>
              <div class="text-[11px] text-(--el-text-color-placeholder) mt-0.5">
                Swap {{ (status?.memory?.virtualMemory?.usedPercent ?? 0).toFixed(1) }}% · {{ formatBytes(status?.memory?.virtualMemory?.usedBytes) }} / {{ formatBytes(status?.memory?.virtualMemory?.totalBytes) }}
              </div>
            </div>
          </div>
          <div class="sys-stat-item">
            <div class="flex items-center justify-center w-9 h-9 rounded-[10px] p-2 mb-3" style="color: #f59e0b; background-color: #f59e0b10">
              <el-icon size="18">
                <component :is="menuStore.iconComponents['HOutline:GlobeAltIcon']" />
              </el-icon>
            </div>
            <div>
              <div class="text-[13px] font-semibold text-(--el-text-color-secondary) mb-1">网络信息</div>
              <div class="flex flex-col gap-0.5">
                <span class="text-[13px] font-extrabold text-(--el-color-success)">
                  ↓ {{ formatBytes(status?.network?.total?.downloadRateBytesPerSecond) }}/s
                </span>
                <span class="text-[11px] text-(--el-text-color-placeholder)">
                  总接收 {{ formatBytes(status?.network?.total?.bytesRecv) }}
                </span>
                <span class="text-[13px] font-extrabold text-(--el-color-primary)">
                  ↑ {{ formatBytes(status?.network?.total?.uploadRateBytesPerSecond) }}/s
                </span>
                <span class="text-[11px] text-(--el-text-color-placeholder)">
                  总发送 {{ formatBytes(status?.network?.total?.bytesSent) }}
                </span>
              </div>
            </div>
          </div>
          <div class="sys-stat-item">
            <div class="flex items-center justify-center w-9 h-9 rounded-[10px] p-2 mb-3" style="color: #ef4444; background-color: #ef444410">
              <el-icon size="18">
                <component :is="menuStore.iconComponents['HOutline:FolderIcon']" />
              </el-icon>
            </div>
            <div>
              <div class="text-[13px] font-semibold text-(--el-text-color-secondary) mb-1">磁盘 使用率</div>
              <div class="flex items-baseline gap-2">
                <span class="text-[20px] font-extrabold text-(--el-text-color-primary)">
                  {{ (status?.disk?.total?.usedPercent ?? 0).toFixed(1) }}%
                </span>
              </div>
              <div class="text-[11px] text-(--el-text-color-placeholder) mt-1">
                {{ formatBytes(status?.disk?.total?.usedBytes) }} / {{ formatBytes(status?.disk?.total?.totalBytes) }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-scrollbar>
  </BaseCard>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import LottieAnimation from '@/components/animation/LottieAnimation.vue'
import workTimeLottieUrl from '@/assets/lotties/welcome.json?url'
import dayjs from 'dayjs'
import { useDashboardHomeStore } from '@/stores/dashboard/home'
import { useUserProfileStore } from '@/stores/user/profile'

const userStore = useUserStore()
const userProfileStore = useUserProfileStore()
const menuStore = useMenuStore()
const dashboardHomeStore = useDashboardHomeStore()

const { currentDate, weatherText, overview, status } =
  storeToRefs(dashboardHomeStore)
const { startCurrentDateTicker, stopCurrentDateTicker } = dashboardHomeStore

const formatBytes = (bytes?: number | string) => {
  const num = Number(bytes)
  if (!num || num <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const base = 1024
  const i = Math.min(Math.floor(Math.log(num) / Math.log(base)), units.length - 1)
  return (num / base ** i).toFixed(i === 0 ? 0 : 1) + ' ' + units[i]
}

const formatUptime = (seconds?: number) => {
  if (seconds == null) return '--'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}天 ${h}时 ${m}分`
  if (h > 0) return `${h}时 ${m}分`
  return `${m}分`
}

const formatBootTime = (t?: Date) => {
  if (!t) return '--'
  return dayjs(t).format('YYYY-MM-DD HH:mm')
}

onMounted(() => {
  startCurrentDateTicker()
  void userProfileStore.ensureAddress()
})

onBeforeUnmount(() => {
  stopCurrentDateTicker()
})
</script>

<style scoped lang="scss">
.el-divider--vertical {
  height: 2.5rem;
}

.sys-stat-item {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 1rem;
  border-radius: 1rem;
  position: relative;
  transition: all 0.3s;
  cursor: default;

  &:hover {
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.08);
    transform: translateY(-2px);
  }
}

.sys-info-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  padding: 6px 12px;
  background: var(--el-bg-color-page);
  border-radius: 10px;
}
</style>
