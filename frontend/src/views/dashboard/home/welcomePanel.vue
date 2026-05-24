<template>
  <BaseCard>
    <el-scrollbar :max-height="620" class="welcome-scrollbar">
      <div class="welcome-body">
        <div class="welcome-main">
          <div class="welcome-hero">
            <div class="relative shrink-0">
              <el-avatar class="welcome-avatar" :src="userStore.resolvedUserAvatar" />
              <div class="avatar-dot"></div>
            </div>

            <div class="welcome-greeting">
              <h2 class="welcome-title-row">
                <TextEllipsis
                  :text="`${userStore.userInfo?.name! || userStore.userInfo?.username!}，欢迎回来！`"
                  :clickable="false"
                  class="welcome-title"
                />
                <span>👋</span>
              </h2>
              <TextEllipsis
                :text="`“ ${userStore.userInfo?.bio} ”`"
                class="welcome-bio"
              />
              <div class="welcome-tags">
                <div class="welcome-tag">
                  <el-icon><component :is="menuStore.iconComponents['Element:Orange']" class="text-orange-500" /></el-icon>
                  <span>{{ weatherText }}</span>
                </div>
                <div class="welcome-tag">
                  <el-icon><component :is="menuStore.iconComponents['Element:Monitor']" class="text-indigo-500" /></el-icon>
                  <span>{{ userProfileStore.address.country }} · {{ userProfileStore.address.region }} · {{ userProfileStore.address.city }}</span>
                </div>
                <div class="welcome-tag">
                  <el-icon><component :is="menuStore.iconComponents['Element:Calendar']" class="text-emerald-500" /></el-icon>
                  <span>{{ currentDate }}</span>
                </div>
              </div>
            </div>

          </div>

          <div class="welcome-info-section">
            <div class="sys-info-grid">
              <div class="sys-info-line">
                <span class="sys-info-label">设备名称</span>
                <span>{{ overview?.version?.hostname || '--' }}</span>
              </div>
              <div class="sys-info-line">
                <span class="sys-info-label">CPU</span>
                <span>{{ overview?.cpu?.modelName || '--' }}</span>
              </div>
              <div class="sys-info-line">
                <span class="sys-info-label">OS</span>
                <span>{{ overview?.version?.os || '--' }}</span>
              </div>
              <div class="sys-info-line">
                <span class="sys-info-label">系统架构</span>
                <span>{{ overview?.version?.kernelArch || '--' }}</span>
              </div>
              <div class="sys-info-line">
                <span class="sys-info-label">操作系统</span>
                <span>{{ overview?.version?.os || '--' }} {{ overview?.version?.platformVersion || '' }}</span>
              </div>
              <div class="sys-info-line">
                <span class="sys-info-label">内核版本</span>
                <span>{{ overview?.version?.kernelVersion || '--' }}</span>
              </div>
              <div class="sys-info-line">
                <span class="sys-info-label">运行时间</span>
                <span>{{ formatUptime(overview?.uptimeSeconds) }}</span>
              </div>
              <div class="sys-info-line">
                <span class="sys-info-label">启动时间</span>
                <span>{{ formatBootTime(overview?.bootTime) }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="welcome-divider"></div>

        <div class="welcome-stats">
          <div class="sys-stat-item">
            <div class="sys-stat-icon" style="color: #6366f1; background-color: #6366f110">
              <el-icon size="18"><component :is="menuStore.iconComponents['HOutline:BoltIcon']" /></el-icon>
            </div>
            <div>
              <div class="sys-stat-label">CPU 使用率</div>
              <div class="sys-stat-value">{{ (status?.cpu?.totalPercent ?? 0).toFixed(1) }}%</div>
              <div class="sys-stat-desc truncate">{{ overview?.cpu?.modelName || '--' }}</div>
            </div>
          </div>
          <div class="sys-stat-item">
            <div class="sys-stat-icon" style="color: #10b981; background-color: #10b98110">
              <el-icon size="18"><component :is="menuStore.iconComponents['Element:Monitor']" /></el-icon>
            </div>
            <div>
              <div class="sys-stat-label">内存信息</div>
              <div class="sys-stat-value">{{ (status?.memory?.physicalMemory?.usedPercent ?? 0).toFixed(1) }}%</div>
              <div class="sys-stat-desc">物理内存 {{ formatBytes(status?.memory?.physicalMemory?.usedBytes) }} / {{ formatBytes(status?.memory?.physicalMemory?.totalBytes) }}</div>
              <div class="sys-stat-desc">Swap {{ (status?.memory?.virtualMemory?.usedPercent ?? 0).toFixed(1) }}% · {{ formatBytes(status?.memory?.virtualMemory?.usedBytes) }} / {{ formatBytes(status?.memory?.virtualMemory?.totalBytes) }}</div>
            </div>
          </div>
          <div class="sys-stat-item">
            <div class="sys-stat-icon" style="color: #f59e0b; background-color: #f59e0b10">
              <el-icon size="18"><component :is="menuStore.iconComponents['HOutline:GlobeAltIcon']" /></el-icon>
            </div>
            <div>
              <div class="sys-stat-label">网络信息</div>
              <div class="net-stat">
                <span class="net-stat-down">↓ {{ formatBytes(status?.network?.total?.downloadRateBytesPerSecond) }}/s</span>
                <span class="net-stat-detail">总接收 {{ formatBytes(status?.network?.total?.bytesRecv) }}</span>
                <span class="net-stat-up">↑ {{ formatBytes(status?.network?.total?.uploadRateBytesPerSecond) }}/s</span>
                <span class="net-stat-detail">总发送 {{ formatBytes(status?.network?.total?.bytesSent) }}</span>
              </div>
            </div>
          </div>
          <div class="sys-stat-item">
            <div class="sys-stat-icon" style="color: #ef4444; background-color: #ef444410">
              <el-icon size="18"><component :is="menuStore.iconComponents['HOutline:FolderIcon']" /></el-icon>
            </div>
            <div>
              <div class="sys-stat-label">磁盘 使用率</div>
              <div class="sys-stat-value">{{ (status?.disk?.total?.usedPercent ?? 0).toFixed(1) }}%</div>
              <div class="sys-stat-desc">{{ formatBytes(status?.disk?.total?.usedBytes) }} / {{ formatBytes(status?.disk?.total?.totalBytes) }}</div>
            </div>
          </div>
        </div>
      </div>
    </el-scrollbar>
  </BaseCard>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
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
/* ===== layout ===== */
.welcome-body {
  display: flex;
  flex-direction: column;
  padding: 1.5rem 1.25rem;

  @media (width >= 640px) {
    padding: 1.5rem;
  }

  @media (width >= 1024px) {
    padding: 2rem;
  }

  @media (width >= 1280px) {
    flex-direction: row;
    padding: 2rem 2.5rem;
  }
}

.welcome-main {
  flex: 1;
  min-width: 0;
}

/* ===== hero ===== */
.welcome-hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 1rem;

  @media (width >= 640px) {
    flex-direction: row;
    align-items: flex-start;
    text-align: left;
  }
}

.welcome-avatar {
  width: 80px;
  height: 80px;

  @media (width >= 640px) {
    width: 110px;
    height: 110px;
  }
}

.avatar-dot {
  position: absolute;
  height: 12px;
  width: 12px;
  bottom: 4px;
  right: 4px;
  border-radius: 999px;
  border: 3px solid var(--el-bg-color);
  background: var(--el-color-success);

  @media (width >= 640px) {
    height: 20px;
    width: 20px;
    bottom: 8px;
    right: 8px;
  }
}

.welcome-greeting {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  align-items: center;
  flex: 1;
  min-width: 0;

  @media (width >= 640px) {
    align-items: flex-start;
  }
}

.welcome-title-row {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 1.25rem;
  font-weight: 900;
  color: var(--el-text-color-primary);
  cursor: pointer;

  @media (width >= 640px) {
    font-size: 1.5rem;
  }

  @media (width >= 768px) {
    font-size: 1.875rem;
  }
}

.welcome-title {
  font-size: inherit;
  font-weight: inherit;
  color: inherit;
}

.welcome-bio {
  color: var(--el-text-color-regular);
  font-style: italic;
  font-size: 0.8rem;
  cursor: pointer;

  @media (width >= 768px) {
    font-size: 0.875rem;
  }
}

.welcome-tags {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  align-items: center;
  gap: 0.5rem;

  @media (width >= 640px) {
    justify-content: flex-start;
  }
}

.welcome-tag {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.7rem;
  font-weight: 600;
  padding: 0.4rem 0.65rem;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-page);
  border-radius: 0.5rem;
}

/* ===== system info grid ===== */
.welcome-info-section {
  padding: 1rem 0.25rem 0;

  @media (width >= 640px) {
    padding: 1.5rem 0.5rem 0;
  }

  @media (width >= 768px) {
    padding: 2rem 1.5rem 0;
  }
}

.sys-info-grid {
  display: grid;
  grid-template-columns: 1fr;
  column-gap: 1rem;
  row-gap: 0.5rem;

  @media (width >= 640px) {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    column-gap: 2rem;
  }
}

.sys-info-line {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  font-size: 0.78rem;
  color: var(--el-text-color-primary);

  @media (width >= 640px) {
    flex-direction: row;
    gap: 0.75rem;
  }
}

.sys-info-label {
  font-weight: 600;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;

  @media (width >= 640px) {
    min-width: 4.5rem;
  }
}

/* ===== divider ===== */
.welcome-divider {
  display: none;

  @media (width >= 1280px) {
    display: block;
    margin: 0 1.5rem;
    width: 1px;
    align-self: stretch;
    background: var(--el-border-color);
  }
}

/* ===== stat cards ===== */
.welcome-stats {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.75rem;

  @media (width >= 640px) {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  @media (width >= 1280px) {
    flex: 0.8;
  }
}

.sys-stat-item {
  display: flex;
  flex-direction: column;
  padding: 0.85rem;
  border-radius: 0.75rem;
  background: var(--el-bg-color-page);
  transition: box-shadow 0.3s, transform 0.3s;

  &:hover {
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.08);
    transform: translateY(-2px);
  }
}

.sys-stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 0.6rem;
  margin-bottom: 0.65rem;
}

.sys-stat-label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 0.2rem;
}

.sys-stat-value {
  font-size: 1.2rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  line-height: 1.2;
}

.sys-stat-desc {
  font-size: 0.68rem;
  color: var(--el-text-color-placeholder);
  margin-top: 0.2rem;
}

.net-stat {
  display: flex;
  flex-direction: column;
  gap: 0.12rem;
}

.net-stat-down {
  font-size: 0.8rem;
  font-weight: 800;
  color: var(--el-color-success);
}

.net-stat-up {
  font-size: 0.8rem;
  font-weight: 800;
  color: var(--el-color-primary);
}

.net-stat-detail {
  font-size: 0.68rem;
  color: var(--el-text-color-placeholder);
}
</style>
