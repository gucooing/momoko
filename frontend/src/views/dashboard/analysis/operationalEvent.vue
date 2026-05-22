<!-- 近期运营大事件 -->
<template>
  <BaseCard title="近期运营大事件">
    <el-scrollbar :height="260">
      <div class="flex items-center justify-between gap-4">
        <el-button
          :type="currentEventTab === 'toBeOpened' ? 'primary' : ''"
          round
          class="flex-1 btn"
          @click="setCurrentEventTab('toBeOpened')"
          >待开启</el-button
        >
        <el-button
          :type="currentEventTab === 'inProgress' ? 'primary' : ''"
          round
          class="flex-1 btn"
          @click="setCurrentEventTab('inProgress')"
          >进行中</el-button
        >
        <el-button
          :type="currentEventTab === 'review' ? 'primary' : ''"
          round
          class="flex-1 btn"
          @click="setCurrentEventTab('review')"
          >回顾</el-button
        >
      </div>

      <Transition name="zoom-in-top" mode="out-in">
        <div :key="currentEventTab">
          <div v-for="item in events" :key="item.id" class="flex gap-4 mt-4">
            <div class="w-15 text-sm text-(--el-text-color-regular)">{{ item.date }}</div>
            <div
              class="flex-1 flex flex-col gap-3 border-l-4 px-4 py-2 bg-(--el-bg-color-page) rounded-2xl"
              :style="{ borderLeftColor: item.color }"
            >
              <div class="text-sm font-semibold">{{ item.title }}</div>
              <div class="text-xs text-(--el-text-color-secondary) flex items-center gap-2">
                <el-icon size="14">
                  <component :is="menuStore.iconComponents['HOutline:CalendarIcon']" />
                </el-icon>
                时间: {{ item.range }}
              </div>
              <div class="flex items-center gap-4">
                <el-avatar
                  :size="24"
                  :src="`https://api.dicebear.com/7.x/avataaars/svg?seed=${item.id}`"
                />
                <span class="text-xs text-(--el-text-color-secondary)">{{
                  currentEventTab === 'toBeOpened'
                    ? '策划中...'
                    : currentEventTab === 'inProgress'
                      ? '进行中...'
                      : '已结束'
                }}</span>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </el-scrollbar>
  </BaseCard>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useDashboardAnalysisStore } from '@/stores/dashboard/analysis'

const menuStore = useMenuStore()
const dashboardAnalysisStore = useDashboardAnalysisStore()
const { currentEventTab, events } = storeToRefs(dashboardAnalysisStore)
const { setCurrentEventTab } = dashboardAnalysisStore
</script>

<style scoped lang="scss"></style>
