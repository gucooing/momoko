<template>
  <div class="sub2api-admin" v-loading="store.configLoading">
    <header class="page-head">
      <div>
        <h1>Sub2API 管理</h1>
        <p>配置统一网关连接、查看用量看板，并维护公开首页的公告与时间线。</p>
      </div>
      <div class="head-actions">
        <el-tag :type="store.statusType(snapshot?.status)" effect="light" round>
          {{ store.statusText(snapshot?.status) }}
        </el-tag>
        <el-button :loading="store.syncing" @click="onSync(false)">增量同步</el-button>
        <el-button type="primary" :loading="store.syncing" @click="onSync(true)">全量同步</el-button>
      </div>
    </header>

    <el-tabs v-model="activeTab" class="admin-tabs">
      <!-- 概览 -->
      <el-tab-pane label="用量概览" name="overview">
        <div class="metric-row">
          <div v-for="card in store.adminMetricCards" :key="card.label" class="metric-card" :class="`tone-${card.tone}`">
            <span class="metric-label">{{ card.label }}</span>
            <strong class="metric-value">{{ card.value }}</strong>
            <small class="metric-detail">{{ card.detail }}</small>
          </div>
          <div class="metric-card tone-blue">
            <span class="metric-label">累计 Token</span>
            <strong class="metric-value">{{ store.formatToken(snapshot?.tokenCount) }}</strong>
            <small class="metric-detail">累计请求 {{ store.formatNumber(snapshot?.requestCount) }}</small>
          </div>
        </div>

        <div class="sync-meta">
          <span>最近同步：{{ store.formatDateTime(snapshot?.lastSyncTime) }}</span>
          <span>下次同步：{{ store.formatDateTime(snapshot?.nextSyncTime) }}</span>
          <span>最新记录：{{ store.formatDateTime(snapshot?.latestRecordTime) }}</span>
          <span>数据范围：{{ snapshot?.dataRange || '-' }}</span>
        </div>

        <el-card shadow="never" class="chart-card">
          <template #header><span class="card-title">用量趋势</span></template>
          <VChart class="chart" :option="store.adminTrendOption" :update-options="chartUpdate" autoresize />
        </el-card>

        <div class="chart-grid">
          <el-card shadow="never" class="chart-card">
            <template #header><span class="card-title">模型请求量 Top</span></template>
            <VChart class="chart sm" :option="store.adminModelOption" :update-options="chartUpdate" autoresize />
          </el-card>
          <el-card shadow="never" class="chart-card">
            <template #header><span class="card-title">接口请求量 Top</span></template>
            <VChart class="chart sm" :option="store.adminEndpointOption" :update-options="chartUpdate" autoresize />
          </el-card>
        </div>

        <el-card shadow="never" class="chart-card">
          <template #header><span class="card-title">最近请求</span></template>
          <el-table :data="snapshot?.recentRequests || []" size="small" stripe>
            <el-table-column prop="model" label="模型" min-width="140" show-overflow-tooltip />
            <el-table-column prop="endpoint" label="接口" min-width="140" show-overflow-tooltip />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.success ? 'success' : 'danger'" effect="light">
                  {{ row.success ? '成功' : '失败' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="延迟" width="100">
              <template #default="{ row }">{{ store.formatLatency(row.latencyMs) }}</template>
            </el-table-column>
            <el-table-column label="Token" width="100">
              <template #default="{ row }">{{ store.formatToken(row.tokenCount) }}</template>
            </el-table-column>
            <el-table-column label="时间" min-width="170">
              <template #default="{ row }">{{ store.formatDateTime(row.requestTime) }}</template>
            </el-table-column>
            <template #empty>暂无请求记录</template>
          </el-table>
        </el-card>
      </el-tab-pane>

      <!-- 配置 -->
      <el-tab-pane label="连接配置" name="config">
        <el-card shadow="never" class="form-card">
          <el-form :model="form" label-width="130px" label-position="right">
            <el-form-item label="开启公开首页">
              <el-switch v-model="form.homeEnabled" />
              <span class="form-hint">开启后访客可访问 /sub2api 公共首页</span>
            </el-form-item>
            <el-form-item label="自动同步">
              <el-switch v-model="form.syncEnabled" />
            </el-form-item>
            <el-form-item label="Sub2API 地址">
              <el-input v-model="form.baseUrl" placeholder="https://your-sub2api.example.com" />
            </el-form-item>
            <el-form-item label="管理员 API Key">
              <el-input v-model="form.adminApiKey" type="password" show-password placeholder="sk-..." />
            </el-form-item>
            <el-form-item label="控制台地址">
              <el-input v-model="form.consoleUrl" placeholder="https://your-sub2api.example.com" />
              <span class="form-hint">首页“前往控制台”将跳转到该地址下的 /dashboard</span>
            </el-form-item>
            <el-form-item label="站点标题">
              <el-input v-model="form.title" />
            </el-form-item>
            <el-form-item label="副标题">
              <el-input v-model="form.subtitle" />
            </el-form-item>
            <el-form-item label="介绍">
              <el-input v-model="form.introduction" type="textarea" :rows="3" />
            </el-form-item>
            <el-form-item label="同步间隔(分钟)">
              <el-input-number v-model="form.syncIntervalMinutes" :min="1" :max="1440" />
            </el-form-item>
            <el-form-item label="历史天数">
              <el-input-number v-model="form.historyDays" :min="1" :max="365" />
            </el-form-item>
            <el-form-item label="分页大小">
              <el-input-number v-model="form.pageSize" :min="50" :max="1000" :step="50" />
            </el-form-item>
            <el-form-item>
              <el-button :loading="store.testing" @click="onTest">测试连接</el-button>
              <el-button type="primary" :loading="store.saving" @click="onSave">保存配置</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <!-- 公告 -->
      <el-tab-pane label="公告" name="announcements">
        <div class="list-toolbar">
          <span>共 {{ store.announcements.length }} 条</span>
          <el-button type="primary" @click="openAnnouncement()">新增公告</el-button>
        </div>
        <el-table :data="store.announcements" v-loading="store.listLoading" stripe>
          <el-table-column prop="title" label="标题" min-width="160" show-overflow-tooltip />
          <el-table-column prop="content" label="内容" min-width="220" show-overflow-tooltip />
          <el-table-column label="级别" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="levelTagType(row.level)" effect="light">{{ levelText(row.level) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="置顶" width="80">
            <template #default="{ row }"><el-tag v-if="row.pinned" size="small" type="warning">置顶</el-tag></template>
          </el-table-column>
          <el-table-column label="发布时间" min-width="170">
            <template #default="{ row }">{{ store.formatDateTime(row.publishedAt) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openAnnouncement(row)">编辑</el-button>
              <el-button link type="danger" @click="onDeleteAnnouncement(row)">删除</el-button>
            </template>
          </el-table-column>
          <template #empty>暂无公告</template>
        </el-table>
      </el-tab-pane>

      <!-- 时间线 -->
      <el-tab-pane label="时间线" name="timeline">
        <div class="list-toolbar">
          <span>共 {{ store.timeline.length }} 条</span>
          <el-button type="primary" @click="openTimeline()">新增时间线</el-button>
        </div>
        <el-table :data="store.timeline" v-loading="store.listLoading" stripe>
          <el-table-column prop="title" label="标题" min-width="160" show-overflow-tooltip />
          <el-table-column prop="content" label="内容" min-width="220" show-overflow-tooltip />
          <el-table-column prop="category" label="分类" width="120" />
          <el-table-column label="发布时间" min-width="170">
            <template #default="{ row }">{{ store.formatDateTime(row.publishedAt) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openTimeline(row)">编辑</el-button>
              <el-button link type="danger" @click="onDeleteTimeline(row)">删除</el-button>
            </template>
          </el-table-column>
          <template #empty>暂无时间线</template>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 公告弹窗 -->
    <el-dialog v-model="announcementDialog" :title="annForm.id ? '编辑公告' : '新增公告'" width="520px">
      <el-form :model="annForm" label-width="90px">
        <el-form-item label="标题"><el-input v-model="annForm.title" /></el-form-item>
        <el-form-item label="内容"><el-input v-model="annForm.content" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="级别">
          <el-select v-model="annForm.level">
            <el-option label="信息" value="info" />
            <el-option label="成功" value="success" />
            <el-option label="警告" value="warning" />
            <el-option label="危险" value="danger" />
          </el-select>
        </el-form-item>
        <el-form-item label="置顶"><el-switch v-model="annForm.pinned" /></el-form-item>
        <el-form-item label="发布时间">
          <el-date-picker v-model="annForm.publishedAt" type="datetime" placeholder="默认当前时间" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="announcementDialog = false">取消</el-button>
        <el-button type="primary" :loading="store.listLoading" @click="submitAnnouncement">保存</el-button>
      </template>
    </el-dialog>

    <!-- 时间线弹窗 -->
    <el-dialog v-model="timelineDialog" :title="tlForm.id ? '编辑时间线' : '新增时间线'" width="520px">
      <el-form :model="tlForm" label-width="90px">
        <el-form-item label="标题"><el-input v-model="tlForm.title" /></el-form-item>
        <el-form-item label="内容"><el-input v-model="tlForm.content" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="分类"><el-input v-model="tlForm.category" placeholder="更新" /></el-form-item>
        <el-form-item label="发布时间">
          <el-date-picker v-model="tlForm.publishedAt" type="datetime" placeholder="默认当前时间" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="timelineDialog = false">取消</el-button>
        <el-button type="primary" :loading="store.listLoading" @click="submitTimeline">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import VChart from '@/components/chart/VChart.vue'
import { useSub2APIStore } from '@/stores/sub2api'
import type { Sub2APIAnnouncement, Sub2APITimelineItem } from '@/types/v1/sub2api'

defineOptions({ name: 'Sub2APIAdmin' })

const store = useSub2APIStore()
const { snapshot, configForm: form } = storeToRefs(store)

const chartUpdate = { notMerge: true }

const activeTab = ref('overview')

const onSync = async (full: boolean) => {
  const ok = await store.syncUsage(full)
  if (ok) ElMessage.success('同步完成')
}

const onTest = async () => {
  const result = await store.testConfig()
  if (!result) return
  if (result.connected) ElMessage.success(result.message || '连接成功')
  else ElMessage.error(result.message || '连接失败')
}

const onSave = async () => {
  const ok = await store.saveConfig()
  if (ok) ElMessage.success('已保存')
}

// 公告
const announcementDialog = ref(false)
const annForm = reactive<{ id?: string; title: string; content: string; level: string; pinned: boolean; publishedAt: Date | undefined }>({
  title: '',
  content: '',
  level: 'info',
  pinned: false,
  publishedAt: undefined,
})
const openAnnouncement = (row?: Sub2APIAnnouncement) => {
  announcementDialog.value = true
  Object.assign(annForm, {
    id: row?.id,
    title: row?.title || '',
    content: row?.content || '',
    level: row?.level || 'info',
    pinned: row?.pinned || false,
    publishedAt: row?.publishedAt ? new Date(row.publishedAt) : undefined,
  })
}
const submitAnnouncement = async () => {
  const ok = await store.saveAnnouncement({ ...annForm })
  if (ok) {
    announcementDialog.value = false
    ElMessage.success('已保存')
  }
}
const onDeleteAnnouncement = async (row: Sub2APIAnnouncement) => {
  await ElMessageBox.confirm('确定删除该公告？', '提示', { type: 'warning' })
  if (await store.removeAnnouncement(row.id)) ElMessage.success('已删除')
}

// 时间线
const timelineDialog = ref(false)
const tlForm = reactive<{ id?: string; title: string; content: string; category: string; publishedAt: Date | undefined }>({
  title: '',
  content: '',
  category: '更新',
  publishedAt: undefined,
})
const openTimeline = (row?: Sub2APITimelineItem) => {
  timelineDialog.value = true
  Object.assign(tlForm, {
    id: row?.id,
    title: row?.title || '',
    content: row?.content || '',
    category: row?.category || '更新',
    publishedAt: row?.publishedAt ? new Date(row.publishedAt) : undefined,
  })
}
const submitTimeline = async () => {
  const ok = await store.saveTimelineItem({ ...tlForm })
  if (ok) {
    timelineDialog.value = false
    ElMessage.success('已保存')
  }
}
const onDeleteTimeline = async (row: Sub2APITimelineItem) => {
  await ElMessageBox.confirm('确定删除该时间线？', '提示', { type: 'warning' })
  if (await store.removeTimelineItem(row.id)) ElMessage.success('已删除')
}

const levelText = (level: string) =>
  ({ info: '信息', success: '成功', warning: '警告', danger: '危险' })[level] || '信息'
const levelTagType = (level: string): 'info' | 'success' | 'warning' | 'danger' =>
  (({ info: 'info', success: 'success', warning: 'warning', danger: 'danger' }) as const)[
    level as 'info' | 'success' | 'warning' | 'danger'
  ] || 'info'

onMounted(() => {
  store.loadAdmin()
})
</script>

<style scoped lang="scss">
.sub2api-admin {
  padding: 16px;
}

.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 12px;

  h1 {
    margin: 0;
    font-size: 20px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  p {
    margin: 6px 0 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
}

.head-actions {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

.metric-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}

.metric-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 16px 18px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  background: var(--el-bg-color-overlay);
  border-left: 3px solid var(--el-color-primary);

  &.tone-green { border-left-color: var(--el-color-success); }
  &.tone-amber { border-left-color: var(--el-color-warning); }
  &.tone-red { border-left-color: var(--el-color-danger); }
}

.metric-label {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.metric-value {
  color: var(--el-text-color-primary);
  font-size: 22px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.metric-detail {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.sync-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 22px;
  margin-bottom: 14px;
  color: var(--el-text-color-secondary);
  font-size: 12.5px;
}

.chart-card {
  margin-bottom: 14px;
  border-radius: 12px;
}

.card-title {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.chart {
  width: 100%;
  height: 320px;
}

.chart.sm {
  height: 280px;
}

.chart-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.form-card {
  max-width: 720px;
  border-radius: 12px;
}

.form-hint {
  margin-left: 10px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.list-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

@media (max-width: 900px) {
  .metric-row,
  .chart-grid {
    grid-template-columns: 1fr;
  }
}
</style>
