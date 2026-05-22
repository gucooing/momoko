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
            class="flex flex-col md:flex-row px-2 md:px-6 py-6 md:py-10 items-center gap-6 md:gap-0"
          >
            <div class="flex w-full md:flex-1 flex-col gap-2">
              <div class="text-xs font-semibold text-(--el-text-color-secondary)">今日任务进度</div>
              <div class="flex items-center gap-2">
                <div class="flex items-center gap-2 shrink-0">
                  <span class="text-xl font-extrabold text-(--el-color-primary)">{{
                    todayTask.done
                  }}</span>
                  <span class="text-sm font-semibold text-(--el-text-color-secondary)"
                    >/ {{ todayTask.total }}</span
                  >
                </div>
                <el-progress :percentage="todayTask.percentage" :stroke-width="8" class="flex-1" />
              </div>
            </div>
            <div class="hidden md:block mx-7">
              <el-divider direction="vertical" />
            </div>
            <div class="flex w-full md:flex-1 flex-col gap-2">
              <div class="text-xs font-semibold text-(--el-text-color-secondary)">待处理审批</div>
              <div class="flex items-center gap-2">
                <div class="flex items-center gap-2 shrink-0">
                  <span class="text-xl font-extrabold text-(--el-color-primary)">{{
                    pendingApproval.count
                  }}</span>
                  <span class="text-sm font-semibold text-(--el-text-color-secondary)">个任务</span>
                </div>
                <div class="flex items-center">
                  <el-avatar
                    v-for="(avatar, index) in pendingApproval.avatars"
                    :key="avatar"
                    :size="20"
                    :src="avatar"
                    :class="['border-2 rounded-full shadow-xl', { '-ml-2': index > 0 }]"
                  />
                  <span class="text-xs ml-1 font-semibold text-(--el-text-color-secondary)"
                    >+{{ pendingApproval.extraCount }}</span
                  >
                </div>
              </div>
            </div>
            <div class="hidden md:block mx-7">
              <el-divider direction="vertical" />
            </div>
          </div>
        </div>

        <div class="hidden xl:block mx-7 my-6">
          <div class="w-px h-full border-(--el-border-color) border-l"></div>
        </div>

        <div class="flex-1 xl:flex-[0.8] grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div
            v-for="item in welcomeCardsWithOption"
            :key="item.label"
            class="flex flex-col justify-between p-4 rounded-2xl relative transition-all duration-300 cursor-pointer hover:shadow-xl hover:-translate-y-1"
          >
            <div
              class="flex items-center justify-center w-9 h-9 rounded-[10px] p-2 mb-3"
              :style="{ color: item.color, backgroundColor: item.color + '10' }"
            >
              <el-icon size="18">
                <component :is="menuStore.iconComponents[item.icon]" />
              </el-icon>
            </div>
            <div>
              <div class="text-[13px] font-semibold text-(--el-text-color-secondary) mb-1">
                {{ item.label }}
              </div>
              <div class="flex items-baseline gap-2">
                <span class="text-[20px] font-extrabold text-(--el-text-color-primary)">{{
                  item.value
                }}</span>
                <BaseTag
                  :text="item.trend"
                  :type="item.trendType === 'up' ? 'success' : 'danger'"
                />
              </div>
            </div>
            <div class="w-20 h-10 opacity-60 absolute -bottom-1 -right-1">
              <VChart class="w-full h-full" :option="item.chartOption" autoresize />
            </div>
          </div>
        </div>
      </div>
    </el-scrollbar>
  </BaseCard>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import VChart from '@/components/chart/VChart.vue'
import LottieAnimation from '@/components/animation/LottieAnimation.vue'
import workTimeLottieUrl from '@/assets/lotties/welcome.json?url'
import { useDashboardHomeStore } from '@/stores/dashboard/home'
import { useUserProfileStore } from '@/stores/user/profile'

const userStore = useUserStore()
const userProfileStore = useUserProfileStore()
const menuStore = useMenuStore()
const dashboardHomeStore = useDashboardHomeStore()

const { currentDate, weatherText, todayTask, pendingApproval, welcomeCards } =
  storeToRefs(dashboardHomeStore)
const { startCurrentDateTicker, stopCurrentDateTicker } = dashboardHomeStore

const createMiniLineChart = (data: number[], color: string) => {
  return {
    grid: { left: 0, right: 0, top: 10, bottom: 0 },
    xAxis: { type: 'category', show: false },
    yAxis: { type: 'value', show: false },
    series: [
      {
        data,
        type: 'line',
        smooth: true,
        symbol: 'none',
        lineStyle: { color, width: 2 },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: color + '30' },
              { offset: 1, color: 'transparent' },
            ],
          },
        },
      },
    ],
  }
}

const welcomeCardsWithOption = computed(() =>
  welcomeCards.value.map((item) => ({
    ...item,
    chartOption: createMiniLineChart(item.chartData, item.color),
  })),
)

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
</style>
