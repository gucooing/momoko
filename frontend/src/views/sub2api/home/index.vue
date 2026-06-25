<template>
  <div class="sub2api-admin" v-loading="store.configLoading">
    <header class="page-head">
      <div>
        <h1>{{ t('sub2api.admin.title') }}</h1>
        <p>{{ t('sub2api.admin.subtitle') }}</p>
      </div>
      <div class="head-actions">
        <el-tag :type="store.statusType(snapshot?.status)" effect="light" round>
          {{ store.statusText(snapshot?.status) }}
        </el-tag>
        <el-button @click="openPublicHome">
          <el-icon class="btn-icon"><Link /></el-icon>{{ t('sub2api.common.publicHome') }}
        </el-button>
        <el-button v-if="canEdit" :loading="store.syncing" @click="onSync(false)">{{
          t('sub2api.common.incrementalSync')
        }}</el-button>
        <el-button v-if="canEdit" type="primary" :loading="store.syncing" @click="onSync(true)">{{
          t('sub2api.common.fullSync')
        }}</el-button>
      </div>
    </header>

    <el-tabs v-model="activeTab" class="admin-tabs">
      <!-- 概览 -->
      <el-tab-pane :label="t('sub2api.common.usageOverview')" name="overview">
        <div class="range-bar">
          <el-segmented v-model="rangeKey" :options="rangeOptions" @change="onRangeChange" />
          <!-- 自定义：开始/结束各用单个 datetime 选择器，仅弹单月日历，移动端不溢出 -->
          <template v-if="rangeKey === 'custom'">
            <el-date-picker
              v-model="customStart"
              type="datetime"
              format="YYYY-MM-DD HH:mm"
              :placeholder="t('sub2api.common.startTime')"
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
              :placeholder="t('sub2api.common.endTime')"
              size="small"
              :clearable="false"
              class="custom-dt"
              @change="onCustomChange"
            />
          </template>
          <span class="range-current">{{ store.adminStats?.rangeLabel }}</span>
        </div>

        <div class="sync-meta">
          <span>{{
            t('sub2api.common.lastSync', { time: store.formatDateTime(snapshot?.lastSyncTime) })
          }}</span>
          <span>{{
            t('sub2api.common.nextSync', { time: store.formatDateTime(snapshot?.nextSyncTime) })
          }}</span>
          <span>{{
            t('sub2api.common.latestRecord', {
              time: store.formatDateTime(snapshot?.latestRecordTime),
            })
          }}</span>
          <span>{{ t('sub2api.common.dataRange', { range: snapshot?.dataRange || '-' }) }}</span>
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
            <template #header
              ><span class="card-title">{{ t('sub2api.common.usageTrend') }}</span></template
            >
            <VChart
              class="chart"
              :option="store.adminTrendOption"
              :update-options="chartUpdate"
              autoresize
            />
          </el-card>

          <div class="chart-grid">
            <el-card shadow="never" class="chart-card">
              <template #header
                ><span class="card-title">{{ t('sub2api.common.modelRequestTop') }}</span></template
              >
              <VChart
                class="chart sm"
                :option="store.adminModelOption"
                :update-options="chartUpdate"
                autoresize
              />
            </el-card>
            <el-card shadow="never" class="chart-card">
              <template #header
                ><span class="card-title">{{
                  t('sub2api.common.endpointRequestTop')
                }}</span></template
              >
              <VChart
                class="chart sm"
                :option="store.adminEndpointOption"
                :update-options="chartUpdate"
                autoresize
              />
            </el-card>
          </div>

          <el-card shadow="never" class="chart-card" v-loading="store.recentLoading">
            <template #header
              ><span class="card-title">{{ t('sub2api.common.recentRequests') }}</span></template
            >
            <el-table class="desktop-table" :data="store.adminRecent" size="small" stripe>
              <el-table-column
                prop="model"
                :label="t('sub2api.common.model')"
                min-width="140"
                show-overflow-tooltip
              />
              <el-table-column
                prop="endpoint"
                :label="t('sub2api.common.endpoint')"
                min-width="130"
                show-overflow-tooltip
              />
              <el-table-column
                :label="t('sub2api.common.account')"
                min-width="110"
                show-overflow-tooltip
              >
                <template #default="{ row }">{{ row.accountName || '-' }}</template>
              </el-table-column>
              <el-table-column :label="t('sub2api.common.status')" width="100">
                <template #default="{ row }">
                  <el-tag size="small" :type="row.success ? 'success' : 'danger'" effect="light">
                    {{ statusLabel(row) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column :label="t('sub2api.common.latency')" width="92">
                <template #default="{ row }">{{ store.formatLatency(row.latencyMs) }}</template>
              </el-table-column>
              <el-table-column label="Token" width="92">
                <template #default="{ row }">{{ store.formatToken(row.tokenCount) }}</template>
              </el-table-column>
              <el-table-column :label="t('sub2api.common.time')" min-width="160">
                <template #default="{ row }">{{ store.formatDateTime(row.requestTime) }}</template>
              </el-table-column>
              <el-table-column :label="t('sub2api.common.operation')" width="76" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="openDetail(row)">{{
                    t('sub2api.common.detail')
                  }}</el-button>
                </template>
              </el-table-column>
              <template #empty>{{ t('sub2api.common.noRequestRecords') }}</template>
            </el-table>
            <div class="mobile-list request-list">
              <article
                v-for="row in store.adminRecent"
                :key="`${row.requestId}-${row.requestTime}`"
                class="mobile-row request-row"
                @click="openDetail(row)"
              >
                <div class="row-main">
                  <strong class="row-title">{{
                    row.model || t('sub2api.common.unknownModel')
                  }}</strong>
                  <small
                    >{{ row.accountName ? `${row.accountName} · ` : ''
                    }}{{ row.endpoint || '-' }}</small
                  >
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
              <el-empty
                v-if="!store.adminRecent.length"
                :description="t('sub2api.common.noRequestRecords')"
              />
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

      <!-- 首页设置 -->
      <el-tab-pane :label="t('sub2api.common.homeSettings')" name="settings">
        <el-card shadow="never" class="form-card">
          <el-form :model="form" label-width="130px" label-position="right" :disabled="!canEdit">
            <el-form-item :label="t('sub2api.admin.enablePublicHome')">
              <el-switch v-model="form.homeEnabled" />
              <span class="form-hint">{{ t('sub2api.admin.publicHomeHint') }}</span>
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.siteTitle')">
              <el-input v-model="form.title" />
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.subtitleLabel')">
              <el-input v-model="form.subtitle" />
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.introduction')">
              <el-input v-model="form.introduction" type="textarea" :rows="3" />
            </el-form-item>
            <el-divider content-position="left">{{ t('sub2api.admin.imageGen') }}</el-divider>
            <el-form-item :label="t('sub2api.admin.enableImageGen')">
              <el-switch v-model="form.imageEnabled" />
              <span class="form-hint">{{ t('sub2api.admin.imageGenHint') }}</span>
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.hostWhitelist')">
              <el-switch v-model="form.srcHostWhitelistEnabled" />
              <span class="form-hint">{{ t('sub2api.admin.hostWhitelistHint') }}</span>
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.allowedHosts')">
              <div class="host-tags">
                <el-tag
                  v-for="(host, i) in form.allowedSrcHosts"
                  :key="i"
                  :closable="canEdit"
                  @close="canEdit && form.allowedSrcHosts.splice(i, 1)"
                  >{{ host }}</el-tag
                >
                <el-input
                  v-if="hostInputVisible"
                  ref="hostInputRef"
                  v-model="hostInputValue"
                  size="small"
                  class="host-input"
                  @keyup.enter="addHost"
                  @blur="addHost"
                />
                <el-button v-else-if="canEdit" size="small" @click="showHostInput">{{
                  t('sub2api.admin.addHost')
                }}</el-button>
              </div>
            </el-form-item>
            <el-form-item>
              <template v-if="canEdit">
                <el-button type="primary" :loading="store.saving" @click="onSave">{{
                  t('sub2api.common.saveConfig')
                }}</el-button>
              </template>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <!-- 公告 -->
      <el-tab-pane :label="t('sub2api.common.announcements')" name="announcements">
        <div class="list-toolbar">
          <span>{{ t('sub2api.common.totalItems', { count: store.announcements.length }) }}</span>
          <el-button v-if="canEdit" type="primary" @click="openAnnouncement()">{{
            t('sub2api.admin.addAnnouncement')
          }}</el-button>
        </div>
        <el-table
          class="desktop-table"
          :data="store.announcements"
          v-loading="store.listLoading"
          stripe
        >
          <el-table-column
            prop="title"
            :label="t('sub2api.common.title')"
            min-width="160"
            show-overflow-tooltip
          />
          <el-table-column
            prop="content"
            :label="t('sub2api.common.content')"
            min-width="220"
            show-overflow-tooltip
          />
          <el-table-column :label="t('sub2api.common.level')" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="levelTagType(row.level)" effect="light">{{
                levelText(row.level)
              }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('sub2api.common.pinned')" width="80">
            <template #default="{ row }"
              ><el-tag v-if="row.pinned" size="small" type="warning">{{
                t('sub2api.common.pinned')
              }}</el-tag></template
            >
          </el-table-column>
          <el-table-column :label="t('sub2api.common.publishTime')" min-width="170">
            <template #default="{ row }">{{ store.formatDateTime(row.publishedAt) }}</template>
          </el-table-column>
          <el-table-column
            v-if="canEdit"
            :label="t('sub2api.common.operation')"
            width="140"
            fixed="right"
          >
            <template #default="{ row }">
              <el-button link type="primary" @click="openAnnouncement(row)">{{
                t('sub2api.common.edit')
              }}</el-button>
              <el-button link type="danger" @click="onDeleteAnnouncement(row)">{{
                t('sub2api.common.delete')
              }}</el-button>
            </template>
          </el-table-column>
          <template #empty>{{ t('sub2api.home.noAnnouncements') }}</template>
        </el-table>
        <div class="mobile-list edit-list" v-loading="store.listLoading">
          <article v-for="row in store.announcements" :key="row.id" class="edit-card">
            <div class="edit-card-head">
              <strong>{{ row.title || t('sub2api.common.announcements') }}</strong>
              <span>
                <el-tag size="small" :type="levelTagType(row.level)" effect="light">{{
                  levelText(row.level)
                }}</el-tag>
                <el-tag v-if="row.pinned" size="small" type="warning" effect="light">{{
                  t('sub2api.common.pinned')
                }}</el-tag>
              </span>
            </div>
            <p>{{ row.content || '-' }}</p>
            <div class="edit-card-foot">
              <time>{{ store.formatDateTime(row.publishedAt) }}</time>
              <span v-if="canEdit">
                <el-button link type="primary" @click="openAnnouncement(row)">{{
                  t('sub2api.common.edit')
                }}</el-button>
                <el-button link type="danger" @click="onDeleteAnnouncement(row)">{{
                  t('sub2api.common.delete')
                }}</el-button>
              </span>
            </div>
          </article>
          <el-empty
            v-if="!store.announcements.length"
            :description="t('sub2api.home.noAnnouncements')"
          />
        </div>
      </el-tab-pane>

      <!-- 时间线 -->
      <el-tab-pane :label="t('sub2api.common.timeline')" name="timeline">
        <div class="list-toolbar">
          <span>{{ t('sub2api.common.totalItems', { count: store.timeline.length }) }}</span>
          <el-button v-if="canEdit" type="primary" @click="openTimeline()">{{
            t('sub2api.admin.addTimeline')
          }}</el-button>
        </div>
        <el-table class="desktop-table" :data="store.timeline" v-loading="store.listLoading" stripe>
          <el-table-column
            prop="title"
            :label="t('sub2api.common.title')"
            min-width="160"
            show-overflow-tooltip
          />
          <el-table-column
            prop="content"
            :label="t('sub2api.common.content')"
            min-width="220"
            show-overflow-tooltip
          />
          <el-table-column prop="category" :label="t('sub2api.common.category')" width="120" />
          <el-table-column :label="t('sub2api.common.publishTime')" min-width="170">
            <template #default="{ row }">{{ store.formatDateTime(row.publishedAt) }}</template>
          </el-table-column>
          <el-table-column
            v-if="canEdit"
            :label="t('sub2api.common.operation')"
            width="140"
            fixed="right"
          >
            <template #default="{ row }">
              <el-button link type="primary" @click="openTimeline(row)">{{
                t('sub2api.common.edit')
              }}</el-button>
              <el-button link type="danger" @click="onDeleteTimeline(row)">{{
                t('sub2api.common.delete')
              }}</el-button>
            </template>
          </el-table-column>
          <template #empty>{{ t('sub2api.common.noData') }}</template>
        </el-table>
        <div class="mobile-list edit-list" v-loading="store.listLoading">
          <article v-for="row in store.timeline" :key="row.id" class="edit-card">
            <div class="edit-card-head">
              <strong>{{ row.title || t('sub2api.common.update') }}</strong>
              <el-tag size="small" effect="light">{{
                row.category || t('sub2api.common.update')
              }}</el-tag>
            </div>
            <p>{{ row.content || '-' }}</p>
            <div class="edit-card-foot">
              <time>{{ store.formatDateTime(row.publishedAt) }}</time>
              <span v-if="canEdit">
                <el-button link type="primary" @click="openTimeline(row)">{{
                  t('sub2api.common.edit')
                }}</el-button>
                <el-button link type="danger" @click="onDeleteTimeline(row)">{{
                  t('sub2api.common.delete')
                }}</el-button>
              </span>
            </div>
          </article>
          <el-empty v-if="!store.timeline.length" :description="t('sub2api.common.noData')" />
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 公告弹窗 -->
    <BaseDialog
      v-model="announcementDialog"
      :title="annForm.id ? t('sub2api.admin.editAnnouncement') : t('sub2api.admin.addAnnouncement')"
      width="520px"
    >
      <el-form :model="annForm" label-width="90px" :disabled="!canEdit">
        <el-form-item :label="t('sub2api.common.title')"
          ><el-input v-model="annForm.title"
        /></el-form-item>
        <el-form-item :label="t('sub2api.common.content')"
          ><el-input v-model="annForm.content" type="textarea" :rows="4"
        /></el-form-item>
        <el-form-item :label="t('sub2api.common.level')">
          <el-select v-model="annForm.level">
            <el-option :label="t('sub2api.common.info')" value="info" />
            <el-option :label="t('sub2api.common.success')" value="success" />
            <el-option :label="t('sub2api.common.warning')" value="warning" />
            <el-option :label="t('sub2api.common.danger')" value="danger" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('sub2api.common.pinned')"
          ><el-switch v-model="annForm.pinned"
        /></el-form-item>
        <el-form-item :label="t('sub2api.common.publishTime')">
          <el-date-picker
            v-model="annForm.publishedAt"
            type="datetime"
            :placeholder="t('sub2api.common.defaultNow')"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="announcementDialog = false">{{ t('sub2api.common.cancel') }}</el-button>
        <el-button
          v-if="canEdit"
          type="primary"
          :loading="store.listLoading"
          @click="submitAnnouncement"
          >{{ t('sub2api.common.save') }}</el-button
        >
      </template>
    </BaseDialog>

    <!-- 时间线弹窗 -->
    <BaseDialog
      v-model="timelineDialog"
      :title="tlForm.id ? t('sub2api.admin.editTimeline') : t('sub2api.admin.addTimeline')"
      width="520px"
    >
      <el-form :model="tlForm" label-width="90px" :disabled="!canEdit">
        <el-form-item :label="t('sub2api.common.title')"
          ><el-input v-model="tlForm.title"
        /></el-form-item>
        <el-form-item :label="t('sub2api.common.content')"
          ><el-input v-model="tlForm.content" type="textarea" :rows="4"
        /></el-form-item>
        <el-form-item :label="t('sub2api.common.category')"
          ><el-input v-model="tlForm.category" :placeholder="t('sub2api.common.update')"
        /></el-form-item>
        <el-form-item :label="t('sub2api.common.publishTime')">
          <el-date-picker
            v-model="tlForm.publishedAt"
            type="datetime"
            :placeholder="t('sub2api.common.defaultNow')"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="timelineDialog = false">{{ t('sub2api.common.cancel') }}</el-button>
        <el-button
          v-if="canEdit"
          type="primary"
          :loading="store.listLoading"
          @click="submitTimeline"
          >{{ t('sub2api.common.save') }}</el-button
        >
      </template>
    </BaseDialog>

    <!-- 最近请求详情 -->
    <BaseDialog v-model="detailDialog" :title="t('sub2api.common.requestDetail')" width="520px">
      <div v-if="detailRow" class="req-detail">
        <div class="detail-row">
          <span class="k">{{ t('sub2api.common.status') }}</span>
          <span class="v">
            <el-tag size="small" :type="detailRow.success ? 'success' : 'danger'" effect="light">{{
              statusLabel(detailRow)
            }}</el-tag>
          </span>
        </div>
        <div class="detail-row">
          <span class="k">{{ t('sub2api.common.model') }}</span
          ><span class="v">{{ detailRow.model || '-' }}</span>
        </div>
        <div class="detail-row">
          <span class="k">{{ t('sub2api.common.endpoint') }}</span
          ><span class="v">{{ detailRow.endpoint || '-' }}</span>
        </div>
        <div class="detail-row">
          <span class="k">{{ t('sub2api.common.accountName') }}</span
          ><span class="v">{{ detailRow.accountName || '-' }}</span>
        </div>
        <div class="detail-row">
          <span class="k">{{ t('sub2api.common.cost') }}</span
          ><span class="v">{{ store.formatCost(detailRow.cost) }}</span>
        </div>
        <div class="detail-row">
          <span class="k">{{ t('sub2api.common.token') }}</span
          ><span class="v">{{ store.formatToken(detailRow.tokenCount) }}</span>
        </div>
        <div class="detail-row">
          <span class="k">{{ t('sub2api.common.duration') }}</span
          ><span class="v">{{ store.formatLatency(detailRow.latencyMs) }}</span>
        </div>
        <div class="detail-row">
          <span class="k">{{ t('sub2api.common.firstToken') }}</span
          ><span class="v">{{ firstTokenText(detailRow.firstTokenMs) }}</span>
        </div>
        <div class="detail-row">
          <span class="k">{{ t('sub2api.common.reasoningEffort') }}</span
          ><span class="v">{{ detailRow.reasoningEffort || '-' }}</span>
        </div>
        <div class="detail-row">
          <span class="k">{{ t('sub2api.common.time') }}</span
          ><span class="v">{{ store.formatDateTime(detailRow.requestTime) }}</span>
        </div>
        <template v-if="!detailRow.success">
          <div class="detail-row">
            <span class="k">{{ t('sub2api.common.httpCode') }}</span
            ><span class="v">{{ detailRow.httpStatus || '-' }}</span>
          </div>
          <div class="detail-row detail-row--block">
            <span class="k">{{ t('sub2api.common.errorDetail') }}</span>
            <span class="v err">{{ detailRow.errorMessage || '-' }}</span>
          </div>
        </template>
        <div class="detail-row detail-row--block">
          <span class="k">{{ t('sub2api.common.requestId') }}</span
          ><span class="v mono">{{ detailRow.requestId || '-' }}</span>
        </div>
      </div>
      <template #footer>
        <el-button type="primary" @click="detailDialog = false">{{
          t('sub2api.common.close')
        }}</el-button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { ArrowRight, Link } from '@element-plus/icons-vue'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import VChart from '@/components/chart/VChart.vue'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { useSub2APIStore } from '@/stores/sub2api'
import { Dialog } from '@/utils/dialog'
import type {
  Sub2APIAnnouncement,
  Sub2APIRecentRequest,
  Sub2APITimelineItem,
} from '@/types/v1/sub2api'

defineOptions({ name: 'Sub2APIHome' })

const store = useSub2APIStore()
const router = useRouter()
const { snapshot, configForm: form } = storeToRefs(store)
const { t } = useI18n()
const canEdit = useButtonPermission([PERM.SUB2API_EDIT], [])

const chartUpdate = { notMerge: true }

const activeTab = ref('overview')

// 用量概览时间区间：默认近 24h，整页数据随之切换
const rangeOptions = computed(() => [
  { label: t('sub2api.common.today'), value: 'today' },
  { label: t('sub2api.common.last24h'), value: '24h' },
  { label: t('sub2api.common.sevenDays'), value: '7d' },
  { label: t('sub2api.common.thirtyDays'), value: '30d' },
  { label: t('sub2api.common.custom'), value: 'custom' },
])
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
  if (row.success) return t('sub2api.common.success')
  return row.httpStatus
    ? t('sub2api.common.failedWithCode', { code: row.httpStatus })
    : t('sub2api.common.failed')
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
  if (!canEdit.value) return
  const ok = await store.syncUsage(full)
  if (ok) {
    ElMessage.success(t('sub2api.common.synced'))
    reloadStats()
  }
}

const onSave = async () => {
  if (!canEdit.value) return
  const ok = await store.saveConfig()
  if (ok) ElMessage.success(t('sub2api.common.saved'))
}

// 允许站点动态 tag 输入
const hostInputVisible = ref(false)
const hostInputValue = ref('')
const hostInputRef = ref<{ focus: () => void } | null>(null)
const showHostInput = () => {
  if (!canEdit.value) return
  hostInputVisible.value = true
  nextTick(() => hostInputRef.value?.focus())
}
const addHost = () => {
  if (!canEdit.value) return
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
  if (!canEdit.value) return
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
  if (!canEdit.value) return
  const ok = await store.saveAnnouncement({ ...annForm })
  if (ok) {
    announcementDialog.value = false
    ElMessage.success(t('sub2api.common.saved'))
  }
}
const onDeleteAnnouncement = (row: Sub2APIAnnouncement) => {
  if (!canEdit.value) return
  Dialog.confirm({
    title: t('sub2api.common.tip'),
    content: t('sub2api.admin.confirmDeleteAnnouncement'),
    onConfirm: async () => {
      if (await store.removeAnnouncement(row.id)) ElMessage.success(t('sub2api.common.deleted'))
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
  category: t('sub2api.common.update'),
  publishedAt: undefined,
})
const openTimeline = (row?: Sub2APITimelineItem) => {
  if (!canEdit.value) return
  timelineDialog.value = true
  Object.assign(tlForm, {
    id: row?.id,
    title: row?.title || '',
    content: row?.content || '',
    category: row?.category || t('sub2api.common.update'),
    publishedAt: row?.publishedAt ? new Date(row.publishedAt) : undefined,
  })
}
const submitTimeline = async () => {
  if (!canEdit.value) return
  const ok = await store.saveTimelineItem({ ...tlForm })
  if (ok) {
    timelineDialog.value = false
    ElMessage.success(t('sub2api.common.saved'))
  }
}
const onDeleteTimeline = (row: Sub2APITimelineItem) => {
  if (!canEdit.value) return
  Dialog.confirm({
    title: t('sub2api.common.tip'),
    content: t('sub2api.admin.confirmDeleteTimeline'),
    onConfirm: async () => {
      if (await store.removeTimelineItem(row.id)) ElMessage.success(t('sub2api.common.deleted'))
    },
  }).catch(() => undefined)
}

const levelText = (level: string) =>
  ({
    info: t('sub2api.common.info'),
    success: t('sub2api.common.success'),
    warning: t('sub2api.common.warning'),
    danger: t('sub2api.common.danger'),
  })[level] || t('sub2api.common.info')
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
