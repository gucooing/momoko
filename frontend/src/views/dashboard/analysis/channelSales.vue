<template>
  <!-- 各渠道销售表现实时榜单 -->
  <BaseCard title="各渠道销售表现实时榜单">
    <el-scrollbar :height="260">
      <el-table :data="channelSales" style="width: 100%">
        <el-table-column label="渠道名称" width="250">
          <template #default="{ row }">
            <div class="profile-cell">
              <div class="project-icon" :style="{ backgroundColor: row.color }">
                {{ row.name.charAt(0) }}
              </div>
              <div class="name-role">
                <div class="name">{{ row.name }}</div>
                <div class="role">负责人: {{ row.owner }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="revenue" label="销售额" min-width="150">
          <template #default="{ row }">
            <div class="font-bold">￥{{ row.revenue }}</div>
          </template>
        </el-table-column>
        <el-table-column label="达成率" min-width="180">
          <template #default="{ row }">
            <el-progress
              :percentage="row.achievement"
              :color="row.achievement > 90 ? '#10b981' : '#5bbff9'"
            />
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" min-width="100">
          <template #default="{ row }">
            <BaseTag :type="row.statusType" effect="dark" round :text="row.status" />
          </template>
        </el-table-column>
      </el-table>
    </el-scrollbar>
  </BaseCard>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useDashboardAnalysisStore } from '@/stores/dashboard/analysis'

const dashboardAnalysisStore = useDashboardAnalysisStore()
const { channelSales } = storeToRefs(dashboardAnalysisStore)
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
