<!-- 近期运营大事件 -->
<template>
  <BaseCard :title="t('dashboard.analysis.operationalEventsTitle')">
    <el-scrollbar :height="260">
      <div class="flex items-center justify-between gap-4">
        <el-button
          :type="currentEventTab === 'toBeOpened' ? 'primary' : ''"
          round
          class="flex-1 btn"
          @click="setCurrentEventTab('toBeOpened')"
          >{{ t('dashboard.analysis.toBeOpened') }}</el-button
        >
        <el-button
          :type="currentEventTab === 'inProgress' ? 'primary' : ''"
          round
          class="flex-1 btn"
          @click="setCurrentEventTab('inProgress')"
          >{{ t('dashboard.analysis.inProgress') }}</el-button
        >
        <el-button
          :type="currentEventTab === 'review' ? 'primary' : ''"
          round
          class="flex-1 btn"
          @click="setCurrentEventTab('review')"
          >{{ t('dashboard.analysis.review') }}</el-button
        >
      </div>

      <Transition name="zoom-in-top" mode="out-in">
        <div :key="currentEventTab">
          <div v-for="item in events" :key="item.id" class="flex gap-4 mt-4">
            <div class="w-15 text-sm text-(--el-text-color-regular)">{{ t(item.dateKey) }}</div>
            <div
              class="flex-1 flex flex-col gap-3 border-l-4 px-4 py-2 bg-(--el-bg-color-page) rounded-2xl"
              :style="{ borderLeftColor: item.color }"
            >
              <div class="text-sm font-semibold">{{ t(item.titleKey) }}</div>
              <div class="text-xs text-(--el-text-color-secondary) flex items-center gap-2">
                <el-icon size="14">
                  <component :is="menuStore.iconComponents['HOutline:CalendarIcon']" />
                </el-icon>
                {{ t('dashboard.analysis.eventTime', { range: item.range }) }}
              </div>
              <div class="flex items-center gap-4">
                <el-avatar
                  :size="24"
                  :src="`https://api.dicebear.com/7.x/avataaars/svg?seed=${item.id}`"
                />
                <span class="text-xs text-(--el-text-color-secondary)">{{
                  currentEventTab === 'toBeOpened'
                    ? t('dashboard.analysis.planning')
                    : currentEventTab === 'inProgress'
                      ? t('dashboard.analysis.running')
                      : t('dashboard.analysis.ended')
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
import { useI18n } from 'vue-i18n'

const menuStore = useMenuStore()
const dashboardAnalysisStore = useDashboardAnalysisStore()
const { currentEventTab, events } = storeToRefs(dashboardAnalysisStore)
const { setCurrentEventTab } = dashboardAnalysisStore
const { t } = useI18n()
</script>

<style scoped lang="scss"></style>
