<template>
  <!-- 各渠道销售表现实时榜单 -->
  <BaseCard :title="t('dashboard.analysis.channelSalesTitle')">
    <el-scrollbar :height="260">
      <el-table :data="channelSales" style="width: 100%">
        <el-table-column :label="t('dashboard.analysis.channelName')" width="250">
          <template #default="{ row }">
            <div class="profile-cell">
              <div class="project-icon" :style="{ backgroundColor: row.color }">
                {{ channelName(row.nameKey).charAt(0) }}
              </div>
              <div class="name-role">
                <div class="name">{{ channelName(row.nameKey) }}</div>
                <div class="role">{{ t('dashboard.analysis.owner') }}: {{ row.owner }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="revenue" :label="t('dashboard.analysis.salesAmount')" min-width="150">
          <template #default="{ row }">
            <div class="font-bold">￥{{ row.revenue }}</div>
          </template>
        </el-table-column>
        <el-table-column :label="t('dashboard.analysis.achievementRate')" min-width="180">
          <template #default="{ row }">
            <el-progress
              :percentage="row.achievement"
              :color="row.achievement > 90 ? '#10b981' : '#5bbff9'"
            />
          </template>
        </el-table-column>
        <el-table-column prop="statusKey" :label="t('dashboard.analysis.status')" min-width="100">
          <template #default="{ row }">
            <BaseTag :type="row.statusType" effect="dark" round :text="t(row.statusKey)" />
          </template>
        </el-table-column>
      </el-table>
    </el-scrollbar>
  </BaseCard>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useDashboardAnalysisStore } from '@/stores/dashboard/analysis'
import { useI18n } from 'vue-i18n'

const dashboardAnalysisStore = useDashboardAnalysisStore()
const { channelSales } = storeToRefs(dashboardAnalysisStore)
const { t } = useI18n()
const channelName = (key: string) => t(key)
</script>

<style scoped lang="scss">
.profile-cell {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  .project-icon {
    width: 2.25rem;
    height: 2.25rem;
    border-radius: 0.5rem;
    color: #fff;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
  }
  .name {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
  .role {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}
</style>
