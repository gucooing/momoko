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
        <el-button @click="openPublicHome">
          <el-icon class="btn-icon"><Link /></el-icon>公开首页
        </el-button>
        <el-button :loading="store.syncing" @click="onSync(false)">增量同步</el-button>
        <el-button type="primary" :loading="store.syncing" @click="onSync(true)"
          >全量同步</el-button
        >
      </div>
    </header>

    <el-tabs v-model="activeTab" class="admin-tabs">
      <!-- 概览 -->
      <el-tab-pane label="用量概览" name="overview">
        <div class="range-bar">
          <el-segmented v-model="rangeKey" :options="rangeOptions" @change="onRangeChange" />
          <!-- 自定义：开始/结束各用单个 datetime 选择器，仅弹单月日历，移动端不溢出 -->
          <template v-if="rangeKey === 'custom'">
            <el-date-picker
              v-model="customStart"
              type="datetime"
              format="YYYY-MM-DD HH:mm"
              placeholder="开始时间"
              size="small"
              :clearable="false"
              class="custom-dt"
              @change="onCustomChange"
            />
            <span class="custom-sep">~</span>
            <el-date-picker
              v-model="customEnd"
              type="datetime"
              format="YYYY-MM-DD HH:mm"
              placeholder="结束时间"
              size="small"
              :clearable="false"
              class="custom-dt"
              @change="onCustomChange"
            />
          </template>
          <span class="range-current">{{ store.adminStats?.rangeLabel }}</span>
        </div>

        <div class="sync-meta">
          <span>最近同步：{{ store.formatDateTime(snapshot?.lastSyncTime) }}</span>
          <span>下次同步：{{ store.formatDateTime(snapshot?.nextSyncTime) }}</span>
          <span>最新记录：{{ store.formatDateTime(snapshot?.latestRecordTime) }}</span>
          <span>数据范围：{{ snapshot?.dataRange || '-' }}</span>
        </div>

        <div v-loading="store.adminStatsLoading" class="overview-body">
          <div class="metric-row">
            <div
              v-for="card in store.adminStatsMetricCards"
              :key="card.label"
              class="metric-card"
              :class="`tone-${card.tone}`"
            >
              <span class="metric-label">{{ card.label }}</span>
              <strong class="metric-value">{{ card.value }}</strong>
              <small class="metric-detail">{{ card.detail }}</small>
            </div>
          </div>

          <el-card shadow="never" class="chart-card">
            <template #header><span class="card-title">用量趋势</span></template>
            <VChart
              class="chart"
              :option="store.adminTrendOption"
              :update-options="chartUpdate"
              autoresize
            />
          </el-card>

          <div class="chart-grid">
            <el-card shadow="never" class="chart-card">
              <template #header><span class="card-title">模型请求量 Top</span></template>
              <VChart
                class="chart sm"
                :option="store.adminModelOption"
                :update-options="chartUpdate"
                autoresize
              />
            </el-card>
            <el-card shadow="never" class="chart-card">
              <template #header><span class="card-title">接口请求量 Top</span></template>
              <VChart
                class="chart sm"
                :option="store.adminEndpointOption"
                :update-options="chartUpdate"
                autoresize
              />
            </el-card>
          </div>

          <el-card shadow="never" class="chart-card" v-loading="store.recentLoading">
            <template #header><span class="card-title">最近请求</span></template>
            <el-table class="desktop-table" :data="store.adminRecent" size="small" stripe>
              <el-table-column prop="model" label="模型" min-width="140" show-overflow-tooltip />
              <el-table-column prop="endpoint" label="接口" min-width="130" show-overflow-tooltip />
              <el-table-column label="账号" min-width="110" show-overflow-tooltip>
                <template #default="{ row }">{{ row.accountName || '-' }}</template>
              </el-table-column>
              <el-table-column label="状态" width="100">
                <template #default="{ row }">
                  <el-tag size="small" :type="row.success ? 'success' : 'danger'" effect="light">
                    {{ statusLabel(row) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="延迟" width="92">
                <template #default="{ row }">{{ store.formatLatency(row.latencyMs) }}</template>
              </el-table-column>
              <el-table-column label="Token" width="92">
                <template #default="{ row }">{{ store.formatToken(row.tokenCount) }}</template>
              </el-table-column>
              <el-table-column label="时间" min-width="160">
                <template #default="{ row }">{{ store.formatDateTime(row.requestTime) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="76" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="openDetail(row)">详情</el-button>
                </template>
              </el-table-column>
              <template #empty>暂无请求记录</template>
            </el-table>
            <div class="mobile-list request-list">
              <article
                v-for="row in store.adminRecent"
                :key="`${row.requestId}-${row.requestTime}`"
                class="mobile-row request-row"
                @click="openDetail(row)"
              >
                <div class="row-main">
                  <strong class="row-title">{{ row.model || '未标记模型' }}</strong>
                  <small>{{ row.accountName ? `${row.accountName} · ` : '' }}{{ row.endpoint || '-' }}</small>
                  <small
                    >{{ store.formatLatency(row.latencyMs) }} ·
                    {{ store.formatToken(row.tokenCount) }} ·
                    {{ store.formatDateTime(row.requestTime) }}</small
                  >
                </div>
                <span class="row-status">
                  <el-tag size="small" :type="row.success ? 'success' : 'danger'" effect="light">
                    {{ statusLabel(row) }}
                  </el-tag>
                  <el-icon class="row-chevron"><ArrowRight /></el-icon>
                </span>
              </article>
              <el-empty v-if="!store.adminRecent.length" description="暂无请求记录" />
            </div>
            <div class="recent-pager">
              <el-pagination
                layout="prev, pager, next"
                :total="store.adminRecentTotal"
                :page-size="store.recentPageSize"
                :current-page="store.recentPage"
                :pager-count="5"
                small
                background
                hide-on-single-page
                @current-change="onRecentPageChange"
              />
            </div>
          </el-card>
        </div>
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
              <el-input
                v-model="form.adminApiKey"
                type="password"
                show-password
                placeholder="sk-..."
              />
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
            <el-divider content-position="left">生图</el-divider>
            <el-form-item label="启用生图">
              <el-switch v-model="form.imageEnabled" />
              <span class="form-hint">关闭后，嵌入的生图页将拒绝所有生图请求</span>
            </el-form-item>
            <el-form-item label="站点白名单">
              <el-switch v-model="form.srcHostWhitelistEnabled" />
              <span class="form-hint">开启后，仅允许下方列表中的 sub2api 站点调用生图</span>
            </el-form-item>
            <el-form-item label="允许站点">
              <div class="host-tags">
                <el-tag
                  v-for="(host, i) in form.allowedSrcHosts"
                  :key="i"
                  closable
                  @close="form.allowedSrcHosts.splice(i, 1)"
                >{{ host }}</el-tag>
                <el-input
                  v-if="hostInputVisible"
                  ref="hostInputRef"
                  v-model="hostInputValue"
                  size="small"
                  class="host-input"
                  @keyup.enter="addHost"
                  @blur="addHost"
                />
                <el-button v-else size="small" @click="showHostInput">+ 添加站点</el-button>
              </div>
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
        <el-table
          class="desktop-table"
          :data="store.announcements"
          v-loading="store.listLoading"
          stripe
        >
          <el-table-column prop="title" label="标题" min-width="160" show-overflow-tooltip />
          <el-table-column prop="content" label="内容" min-width="220" show-overflow-tooltip />
          <el-table-column label="级别" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="levelTagType(row.level)" effect="light">{{
                levelText(row.level)
              }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="置顶" width="80">
            <template #default="{ row }"
              ><el-tag v-if="row.pinned" size="small" type="warning">置顶</el-tag></template
            >
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
        <div class="mobile-list edit-list" v-loading="store.listLoading">
          <article v-for="row in store.announcements" :key="row.id" class="edit-card">
            <div class="edit-card-head">
              <strong>{{ row.title || '公告' }}</strong>
              <span>
                <el-tag size="small" :type="levelTagType(row.level)" effect="light">{{
                  levelText(row.level)
                }}</el-tag>
                <el-tag v-if="row.pinned" size="small" type="warning" effect="light">置顶</el-tag>
              </span>
            </div>
            <p>{{ row.content || '-' }}</p>
            <div class="edit-card-foot">
              <time>{{ store.formatDateTime(row.publishedAt) }}</time>
              <span>
                <el-button link type="primary" @click="openAnnouncement(row)">编辑</el-button>
                <el-button link type="danger" @click="onDeleteAnnouncement(row)">删除</el-button>
              </span>
            </div>
          </article>
          <el-empty v-if="!store.announcements.length" description="暂无公告" />
        </div>
      </el-tab-pane>

      <!-- 时间线 -->
      <el-tab-pane label="时间线" name="timeline">
        <div class="list-toolbar">
          <span>共 {{ store.timeline.length }} 条</span>
          <el-button type="primary" @click="openTimeline()">新增时间线</el-button>
        </div>
        <el-table class="desktop-table" :data="store.timeline" v-loading="store.listLoading" stripe>
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
        <div class="mobile-list edit-list" v-loading="store.listLoading">
          <article v-for="row in store.timeline" :key="row.id" class="edit-card">
            <div class="edit-card-head">
              <strong>{{ row.title || '更新' }}</strong>
              <el-tag size="small" effect="light">{{ row.category || '更新' }}</el-tag>
            </div>
            <p>{{ row.content || '-' }}</p>
            <div class="edit-card-foot">
              <time>{{ store.formatDateTime(row.publishedAt) }}</time>
              <span>
                <el-button link type="primary" @click="openTimeline(row)">编辑</el-button>
                <el-button link type="danger" @click="onDeleteTimeline(row)">删除</el-button>
              </span>
            </div>
          </article>
          <el-empty v-if="!store.timeline.length" description="暂无时间线" />
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 公告弹窗 -->
    <BaseDialog
      v-model="announcementDialog"
      :title="annForm.id ? '编辑公告' : '新增公告'"
      width="520px"
    >
      <el-form :model="annForm" label-width="90px">
        <el-form-item label="标题"><el-input v-model="annForm.title" /></el-form-item>
        <el-form-item label="内容"
          ><el-input v-model="annForm.content" type="textarea" :rows="4"
        /></el-form-item>
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
          <el-date-picker
            v-model="annForm.publishedAt"
            type="datetime"
            placeholder="默认当前时间"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="announcementDialog = false">取消</el-button>
        <el-button type="primary" :loading="store.listLoading" @click="submitAnnouncement"
          >保存</el-button
        >
      </template>
    </BaseDialog>

    <!-- 时间线弹窗 -->
    <BaseDialog
      v-model="timelineDialog"
      :title="tlForm.id ? '编辑时间线' : '新增时间线'"
      width="520px"
    >
      <el-form :model="tlForm" label-width="90px">
        <el-form-item label="标题"><el-input v-model="tlForm.title" /></el-form-item>
        <el-form-item label="内容"
          ><el-input v-model="tlForm.content" type="textarea" :rows="4"
        /></el-form-item>
        <el-form-item label="分类"
          ><el-input v-model="tlForm.category" placeholder="更新"
        /></el-form-item>
        <el-form-item label="发布时间">
          <el-date-picker v-model="tlForm.publishedAt" type="datetime" placeholder="默认当前时间" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="timelineDialog = false">取消</el-button>
        <el-button type="primary" :loading="store.listLoading" @click="submitTimeline"
          >保存</el-button
        >
      </template>
    </BaseDialog>

    <!-- 最近请求详情 -->
    <BaseDialog v-model="detailDialog" title="请求详情" width="520px">
      <div v-if="detailRow" class="req-detail">
        <div class="detail-row">
          <span class="k">状态</span>
          <span class="v">
            <el-tag
              size="small"
              :type="detailRow.success ? 'success' : 'danger'"
              effect="light"
              >{{ statusLabel(detailRow) }}</el-tag
            >
          </span>
        </div>
        <div class="detail-row"><span class="k">模型</span><span class="v">{{ detailRow.model || '-' }}</span></div>
        <div class="detail-row"><span class="k">接口</span><span class="v">{{ detailRow.endpoint || '-' }}</span></div>
        <div class="detail-row"><span class="k">账号名称</span><span class="v">{{ detailRow.accountName || '-' }}</span></div>
        <div class="detail-row"><span class="k">费用</span><span class="v">{{ store.formatCost(detailRow.cost) }}</span></div>
        <div class="detail-row"><span class="k">Token</span><span class="v">{{ store.formatToken(detailRow.tokenCount) }}</span></div>
        <div class="detail-row"><span class="k">耗时</span><span class="v">{{ store.formatLatency(detailRow.latencyMs) }}</span></div>
        <div class="detail-row"><span class="k">首 Token</span><span class="v">{{ firstTokenText(detailRow.firstTokenMs) }}</span></div>
        <div class="detail-row"><span class="k">推理强度</span><span class="v">{{ detailRow.reasoningEffort || '-' }}</span></div>
        <div class="detail-row"><span class="k">时间</span><span class="v">{{ store.formatDateTime(detailRow.requestTime) }}</span></div>
        <template v-if="!detailRow.success">
          <div class="detail-row"><span class="k">HTTP 码</span><span class="v">{{ detailRow.httpStatus || '-' }}</span></div>
          <div class="detail-row detail-row--block">
            <span class="k">错误详情</span>
            <span class="v err">{{ detailRow.errorMessage || '-' }}</span>
          </div>
        </template>
        <div class="detail-row detail-row--block">
          <span class="k">请求 ID</span><span class="v mono">{{ detailRow.requestId || '-' }}</span>
        </div>
      </div>
      <template #footer>
        <el-button type="primary" @click="detailDialog = false">关闭</el-button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { ArrowRight, Link } from '@element-plus/icons-vue'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import VChart from '@/components/chart/VChart.vue'
import { useSub2APIStore } from '@/stores/sub2api'
import { Dialog } from '@/utils/dialog'
import type {
  Sub2APIAnnouncement,
  Sub2APIRecentRequest,
  Sub2APITimelineItem,
} from '@/types/v1/sub2api'

defineOptions({ name: 'Sub2APIAdmin' })

const store = useSub2APIStore()
const router = useRouter()
const { snapshot, configForm: form } = storeToRefs(store)

const chartUpdate = { notMerge: true }

const activeTab = ref('overview')

// 用量概览时间区间：默认近 24h，整页数据随之切换
const rangeOptions = [
  { label: '今日', value: 'today' },
  { label: '24h', value: '24h' },
  { label: '7天', value: '7d' },
  { label: '30天', value: '30d' },
  { label: '自定义', value: 'custom' },
]
const rangeKey = ref<'today' | '24h' | '7d' | '30d' | 'custom'>('24h')
// 自定义时间段：开始/结束两个单独的 datetime 选择器（精度到分钟）
const customStart = ref<Date | null>(null)
const customEnd = ref<Date | null>(null)

const HOUR = 3_600_000
const MINUTE = 60_000
const floorMinute = (ms: number) => ms - (ms % MINUTE)
const startOfTodayMs = () => {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  return d.getTime()
}

// 当前选择对应的时间段 [startTime, endTime]（Unix 毫秒，精度到分钟）。
// 用函数而非 computed，避免缓存住 Date.now()，每次刷新都按当前时间重新锚定。
const resolveRange = (): { startTime: number; endTime: number } => {
  const end = floorMinute(Date.now())
  switch (rangeKey.value) {
    case 'today':
      return { startTime: floorMinute(startOfTodayMs()), endTime: end }
    case '24h':
      return { startTime: end - 24 * HOUR, endTime: end }
    case '7d':
      return { startTime: end - 7 * 24 * HOUR, endTime: end }
    case '30d':
      return { startTime: end - 30 * 24 * HOUR, endTime: end }
    case 'custom':
      if (customStart.value && customEnd.value) {
        let s = floorMinute(customStart.value.getTime())
        let e = floorMinute(customEnd.value.getTime())
        if (s > e) [s, e] = [e, s] // 起止反选时自动纠正
        return { startTime: s, endTime: e }
      }
      return { startTime: end - 24 * HOUR, endTime: end }
    default:
      return { startTime: end - 24 * HOUR, endTime: end }
  }
}

// 记录当前生效的时间段，供最近请求翻页时复用同一区间（避免相对区间随墙钟漂移）
const lastRange = ref<{ startTime: number; endTime: number }>({ startTime: 0, endTime: 0 })

const reloadStats = () => {
  const range = resolveRange()
  lastRange.value = range
  store.loadAdminStats(range.startTime, range.endTime)
  store.loadAdminRecent(range.startTime, range.endTime, 1)
}

const onRecentPageChange = (page: number) => {
  store.loadAdminRecent(lastRange.value.startTime, lastRange.value.endTime, page)
}

// 最近请求：状态标签（失败时附带 HTTP 码）与首 token 文案
const statusLabel = (row: Sub2APIRecentRequest) => {
  if (row.success) return '成功'
  return row.httpStatus ? `失败 ${row.httpStatus}` : '失败'
}
const firstTokenText = (ms?: number) => (ms && ms > 0 ? store.formatLatency(ms) : '-')

// 最近请求详情弹窗
const detailDialog = ref(false)
const detailRow = ref<Sub2APIRecentRequest>()
const openDetail = (row: Sub2APIRecentRequest) => {
  detailRow.value = row
  detailDialog.value = true
}

const onRangeChange = () => {
  // 首次切到自定义时给个默认区间（近 24h），避免空选择
  if (rangeKey.value === 'custom' && (!customStart.value || !customEnd.value)) {
    const now = new Date()
    customStart.value = new Date(now.getTime() - 24 * HOUR)
    customEnd.value = now
  }
  reloadStats()
}

const onCustomChange = () => {
  if (customStart.value && customEnd.value) reloadStats()
}

const openPublicHome = () => {
  window.open(router.resolve('/public/sub2api/home').href, '_blank')
}

const onSync = async (full: boolean) => {
  const ok = await store.syncUsage(full)
  if (ok) {
    ElMessage.success('同步完成')
    reloadStats()
  }
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

// 允许站点动态 tag 输入
const hostInputVisible = ref(false)
const hostInputValue = ref('')
const hostInputRef = ref<{ focus: () => void } | null>(null)
const showHostInput = () => {
  hostInputVisible.value = true
  nextTick(() => hostInputRef.value?.focus())
}
const addHost = () => {
  const v = hostInputValue.value.trim().replace(/\/+$/, '')
  if (v && !form.value.allowedSrcHosts.includes(v)) form.value.allowedSrcHosts.push(v)
  hostInputVisible.value = false
  hostInputValue.value = ''
}

// 公告
const announcementDialog = ref(false)
const annForm = reactive<{
  id?: string
  title: string
  content: string
  level: string
  pinned: boolean
  publishedAt: Date | undefined
}>({
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
const onDeleteAnnouncement = (row: Sub2APIAnnouncement) => {
  Dialog.confirm({
    title: '提示',
    content: '确定删除该公告？',
    onConfirm: async () => {
      if (await store.removeAnnouncement(row.id)) ElMessage.success('已删除')
    },
  }).catch(() => undefined)
}

// 时间线
const timelineDialog = ref(false)
const tlForm = reactive<{
  id?: string
  title: string
  content: string
  category: string
  publishedAt: Date | undefined
}>({
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
const onDeleteTimeline = (row: Sub2APITimelineItem) => {
  Dialog.confirm({
    title: '提示',
    content: '确定删除该时间线？',
    onConfirm: async () => {
      if (await store.removeTimelineItem(row.id)) ElMessage.success('已删除')
    },
  }).catch(() => undefined)
}

const levelText = (level: string) =>
  ({ info: '信息', success: '成功', warning: '警告', danger: '危险' })[level] || '信息'
const levelTagType = (level: string): 'info' | 'success' | 'warning' | 'danger' =>
  (({ info: 'info', success: 'success', warning: 'warning', danger: 'danger' }) as const)[
    level as 'info' | 'success' | 'warning' | 'danger'
  ] || 'info'

onMounted(() => {
  store.loadAdmin()
  reloadStats()
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

  :deep(.el-button + .el-button) {
    margin-left: 0;
  }
}

.range-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.custom-dt {
  width: 184px;
}

.custom-sep {
  color: var(--el-text-color-secondary);
}

.range-current {
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.btn-icon {
  margin-right: 4px;
}

.metric-row {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
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

  &.tone-green {
    border-left-color: var(--el-color-success);
  }
  &.tone-amber {
    border-left-color: var(--el-color-warning);
  }
  &.tone-red {
    border-left-color: var(--el-color-danger);
  }
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
  min-width: 0;
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

.mobile-list {
  display: none;
}

.mobile-list-head,
.mobile-row {
  display: grid;
  grid-template-columns: 84px minmax(0, 1fr) 70px;
  align-items: center;
  gap: 8px;
}

.mobile-list-head {
  min-height: 32px;
  padding: 0 0 6px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.mobile-row {
  min-height: 38px;
  padding: 7px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
  font-size: 12.5px;
}

.mobile-row:last-child {
  border-bottom: 0;
}

.row-title,
.row-content {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-title {
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.row-content {
  display: block;
  color: var(--el-text-color-regular);
}

.row-main {
  min-width: 0;
}

.row-main small {
  display: block;
  margin-top: 2px;
  color: var(--el-text-color-secondary);
  font-size: 11.5px;
  line-height: 1.25;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-status,
.row-actions {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.request-list .mobile-row {
  grid-template-columns: minmax(0, 1fr) auto;
}

.request-row {
  cursor: pointer;
}

.row-chevron {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

.recent-pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}

// 请求详情：标签 + 值的两列布局，移动端自动堆叠
.req-detail {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.detail-row {
  display: grid;
  grid-template-columns: 84px minmax(0, 1fr);
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
  font-size: 13px;
}

.detail-row:last-child {
  border-bottom: 0;
}

.detail-row .k {
  color: var(--el-text-color-secondary);
}

.detail-row .v {
  min-width: 0;
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
}

.detail-row--block {
  grid-template-columns: 1fr;
  gap: 4px;
}

.detail-row .v.err {
  color: var(--el-color-danger);
  white-space: pre-wrap;
}

.detail-row .v.mono {
  font-family: var(--el-font-family-mono, monospace);
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.row-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

.edit-list {
  gap: 8px;
}

.edit-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-bg-color-overlay);
  padding: 10px 12px;
}

.edit-card-head,
.edit-card-foot {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.edit-card-head strong {
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 14px;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.edit-card-head > span,
.edit-card-foot > span {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 8px;
}

.edit-card p {
  margin: 7px 0;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.55;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.edit-card time {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.edit-card-foot :deep(.el-button + .el-button) {
  margin-left: 0;
}

@media (max-width: 900px) {
  .metric-row {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .chart-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .sub2api-admin {
    padding: 10px;
  }

  .page-head {
    gap: 10px;
    margin-bottom: 8px;

    h1 {
      font-size: 19px;
    }

    p {
      margin-top: 4px;
      font-size: 12.5px;
      line-height: 1.45;
    }
  }

  .head-actions {
    gap: 8px;
    flex-wrap: wrap;

    :deep(.el-button) {
      padding: 8px 14px;
    }
  }

  .range-bar {
    gap: 8px;
  }

  // 分段控件可横向滚动，避免 5 个选项撑破屏幕
  .range-bar :deep(.el-segmented) {
    max-width: 100%;
    overflow-x: auto;
    scrollbar-width: none;
  }

  .range-bar :deep(.el-segmented::-webkit-scrollbar) {
    display: none;
  }

  // 自定义起止选择器在移动端各占满整行，单月日历弹层自动收进视口
  .range-bar :deep(.el-date-editor) {
    width: 100%;
  }

  // 两个选择器各占一行后，中间的 ~ 分隔符无意义，隐藏
  .custom-sep {
    display: none;
  }

  .range-current {
    width: 100%;
    margin-left: 0;
  }

  .admin-tabs {
    :deep(.el-tabs__header) {
      margin-bottom: 10px;
    }

    :deep(.el-tabs__item) {
      height: 36px;
      padding: 0 14px;
      font-size: 14px;
    }
  }

  .metric-row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
    margin-bottom: 10px;
  }

  .metric-card {
    gap: 4px;
    min-width: 0;
    padding: 10px 12px;
    border-radius: 9px;
  }

  .metric-label {
    font-size: 12px;
  }

  .metric-value {
    font-size: 18px;
    line-height: 1.25;
    overflow-wrap: anywhere;
  }

  .metric-detail {
    font-size: 11.5px;
    line-height: 1.35;
  }

  .sync-meta {
    gap: 4px 12px;
    margin-bottom: 10px;
    font-size: 12px;
    line-height: 1.45;
  }

  .chart-card {
    margin-bottom: 10px;
    border-radius: 10px;
  }

  .chart-card :deep(.el-card__header) {
    padding: 10px 12px;
  }

  .chart-card :deep(.el-card__body) {
    padding: 10px 12px;
  }

  .chart {
    height: 240px;
  }

  .chart.sm {
    height: 210px;
  }

  .chart-grid {
    gap: 10px;
  }

  .form-card {
    max-width: none;
    border-radius: 10px;
  }

  .form-card :deep(.el-card__body) {
    padding: 12px;
  }

  .form-card :deep(.el-form-item) {
    display: block;
    margin-bottom: 13px;
  }

  .form-card :deep(.el-form-item__label) {
    display: block;
    width: auto !important;
    height: auto;
    margin-bottom: 5px;
    padding: 0;
    text-align: left;
    line-height: 1.4;
  }

  .form-card :deep(.el-form-item__content) {
    width: 100%;
    margin-left: 0 !important;
    line-height: 32px;
  }

  .form-card :deep(.el-input),
  .form-card :deep(.el-textarea),
  .form-card :deep(.el-input-number),
  .form-card :deep(.el-date-editor),
  .form-card :deep(.el-select) {
    width: 100%;
  }

  .form-hint {
    display: block;
    width: 100%;
    margin: 4px 0 0;
    line-height: 1.45;
  }

  .list-toolbar {
    margin-bottom: 8px;

    :deep(.el-button) {
      padding: 8px 14px;
    }
  }

  .recent-pager {
    justify-content: center;
  }

  .desktop-table {
    display: none;
  }

  .mobile-list {
    display: flex;
    flex-direction: column;
    gap: 0;
  }

  .edit-list {
    gap: 8px;
  }
}

@media (max-width: 380px) {
  .metric-row {
    grid-template-columns: 1fr;
  }
}

/* 生图配置：允许站点动态 tag */
.form-hint {
  margin-left: 10px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.host-tags {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.host-input {
  width: 220px;
}
</style>

