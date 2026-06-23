<template>
  <div class="docker-page">
    <el-card shadow="never">
      <el-form :model="queryForm" label-width="auto" @keyup.enter="search">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item label="名称">
              <el-input v-model="queryForm.name" placeholder="容器名称" clearable />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item label="状态">
              <el-select v-model="queryForm.status" placeholder="全部" clearable style="width: 100%">
                <el-option label="运行中" value="running" />
                <el-option label="已停止" value="exited" />
                <el-option label="暂停" value="paused" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item label="镜像">
              <el-input v-model="queryForm.image" placeholder="镜像名" clearable />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item>
              <el-button type="primary" :icon="menuStore.iconComponents.Search" @click="search">搜索</el-button>
              <el-button :icon="menuStore.iconComponents.Refresh" @click="reset">重置</el-button>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <el-card shadow="never" class="card-mt-16">
      <div class="operation-container">
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" :disabled="!canManage" @click="openCreate">创建容器</el-button>
        <el-button :icon="menuStore.iconComponents['HOutline:ClockIcon']" @click="openTasks">任务</el-button>
      </div>

      <div v-loading="loading">
        <VxeGrid v-if="!menuStore.isMobile" v-bind="gridConfig">
          <template #column-state="{ row }: { row: DockerContainerSummary }">
            <BaseTag :text="stateLabel(row.state)" :type="stateTagType(row.state)" />
          </template>
          <template #column-names="{ row }: { row: DockerContainerSummary }">
            <span>{{ displayNames(row) }}</span>
          </template>
          <template #column-runtime="{ row }: { row: DockerContainerSummary }">
            <span>{{ formatContainerRuntime(row) }}</span>
          </template>
          <template #column-network="{ row }: { row: DockerContainerSummary }">
            <template v-if="containerNetworkItems(row).length">
              <span
                v-for="item in containerNetworkItems(row).slice(0, 2)"
                :key="item"
                class="network-tag"
              >
                {{ item }}
              </span>
              <span v-if="containerNetworkItems(row).length > 2" class="text-muted">
                +{{ containerNetworkItems(row).length - 2 }}
              </span>
            </template>
            <span v-else class="text-muted">-</span>
          </template>
          <template #column-operation="{ row }: { row: DockerContainerSummary }">
            <div
              :ref="el => setActionCellRef(row.id, el)"
              class="container-actions"
              :data-row-id="row.id"
            >
              <el-button
                v-for="item in desktopInlineActions(row)"
                :key="item.key"
                :type="item.danger ? 'danger' : 'primary'"
                link
                size="small"
                :loading="item.loading"
                :disabled="item.disabled"
                @click="handleContainerActionCommand(row, item.key)"
              >
                {{ item.label }}
              </el-button>
              <el-dropdown
                v-if="desktopMoreActions(row).length"
                trigger="click"
                :disabled="isRowActionLoading(row)"
                @command="(command: string) => handleContainerActionCommand(row, command)"
              >
                <el-button
                  class="container-actions__more"
                  type="primary"
                  link
                  size="small"
                  :disabled="isRowActionLoading(row)"
                  title="更多"
                >
                  ...
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item
                      v-for="item in desktopMoreActions(row)"
                      :key="item.key"
                      :command="item.key"
                      :disabled="item.disabled"
                      :divided="item.divided"
                      :class="{ 'container-actions__danger': item.danger }"
                    >
                      {{ item.label }}
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </VxeGrid>

        <div v-else-if="list.length" class="mobile-card-list">
          <div v-for="row in list" :key="row.id" class="mobile-card">
            <div class="mobile-card-body">
              <div class="mobile-card-header">
                <span class="mobile-card-title">{{ displayNames(row) }}</span>
                <BaseTag :text="stateLabel(row.state)" :type="stateTagType(row.state)" />
              </div>
              <div class="mobile-card-meta"><span>镜像：{{ row.image }}</span></div>
              <div class="mobile-card-meta"><span>运行时间：{{ formatContainerRuntime(row) }}</span></div>
              <div class="mobile-card-meta"><span>网络：{{ formatNetworkText(row) }}</span></div>
            </div>
            <div class="mobile-card-actions">
              <el-button
                v-for="item in mobileInlineActions(row)"
                :key="item.key"
                size="small"
                plain
                :type="item.danger ? 'danger' : 'primary'"
                :loading="item.loading"
                :disabled="item.disabled"
                @click="handleContainerActionCommand(row, item.key)"
              >
                {{ item.label }}
              </el-button>
              <el-dropdown v-if="mobileMoreActions(row).length" trigger="click" :disabled="isRowActionLoading(row)" @command="(command: string) => handleContainerActionCommand(row, command)">
                <el-button size="small" plain :loading="isRowActionLoading(row)">更多</el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item
                      v-for="item in mobileMoreActions(row)"
                      :key="item.key"
                      :command="item.key"
                      :disabled="item.disabled"
                      :divided="item.divided"
                      :class="{ 'container-actions__danger': item.danger }"
                    >
                      {{ item.label }}
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </div>
        </div>
        <el-empty v-else-if="!loading" description="暂无容器" />
      </div>

      <TablePagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :is-mobile="menuStore.isMobile"
        @change="getList"
      />
    </el-card>

    <!-- Create Dialog -->
    <BaseDialog v-model="createVisible" title="创建容器" width="700">
      <el-form :model="createForm" label-position="top">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="容器名称">
              <el-input v-model="createForm.name" placeholder="可选" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="镜像" required>
              <el-input v-model="createForm.image" placeholder="选择或输入镜像" class="image-combo-input">
                <template #suffix>
                  <span class="image-combo-actions">
                    <el-icon v-if="createForm.image" class="image-combo-clear" @click.stop="createForm.image = ''">
                      <component :is="menuStore.iconComponents['HSolid:XCircleIcon']" />
                    </el-icon>
                    <el-dropdown trigger="click" max-height="260" @visible-change="visible => visible && loadImageOptions()" @command="value => createForm.image = String(value)">
                      <el-icon><component :is="menuStore.iconComponents['Element:ArrowDown']" /></el-icon>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item v-for="item in imageOptions" :key="item" :command="item">{{ item }}</el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </span>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="启动命令">
          <el-input v-model="createForm.cmdText" placeholder="如 /bin/bash -c 'echo hello'" />
        </el-form-item>
        <el-form-item label="环境变量">
          <el-input v-model="createForm.envText" type="textarea" :rows="2" placeholder="KEY=VALUE" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="端口映射">
              <el-input v-model="createForm.portsText" type="textarea" :rows="2" placeholder="8080:80" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="卷挂载">
              <el-input v-model="createForm.mountsText" type="textarea" :rows="2" placeholder="vol_name:/data" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="网络">
              <el-select v-model="createForm.network" clearable placeholder="默认 bridge" style="width: 100%" @focus="loadNetworkOptions">
                <el-option v-for="item in networkOptions" :key="item" :label="item" :value="item" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="重启策略">
              <el-select v-model="createForm.restartPolicy" style="width: 100%">
                <el-option label="不重启" value="" />
                <el-option label="总是重启" value="always" />
                <el-option label="失败时重启" value="on-failure" />
                <el-option label="除非停止" value="unless-stopped" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="createSubmitting" @click="submitCreate">创建</el-button>
      </template>
    </BaseDialog>

    <!-- Detail Dialog -->
    <BaseDialog v-model="detailVisible" title="容器详情" width="860">
      <div v-if="detail" v-loading="detailLoading" class="container-detail">
        <div class="detail-hero">
          <div class="detail-hero__main">
            <div class="detail-hero__title">
              <span>{{ detail.name || '-' }}</span>
              <BaseTag :text="stateLabel(detail.state?.status || '')" :type="stateTagType(detail.state?.status || '')" />
            </div>
            <div class="detail-hero__sub">
              <span>{{ shortId(detail.id) }}</span>
              <span>{{ detailImageName }}</span>
            </div>
          </div>
          <div class="detail-network-summary">
            <div class="detail-network-summary__mode">{{ detailNetworkMode }}</div>
            <div class="detail-network-summary__ports">
              <template v-if="detailPortTags.length">
                <span v-for="item in detailPortTags" :key="item" class="port-tag">{{ item }}</span>
              </template>
              <span v-else class="text-muted">{{ detailPortText }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section-grid">
          <div class="detail-section">
            <div class="detail-section__title">运行</div>
            <div class="detail-kv">
              <div><span>PID</span><strong>{{ detail.state?.pid || '-' }}</strong></div>
              <div><span>重启</span><strong>{{ detail.restartCount }}</strong></div>
              <div><span>退出码</span><strong>{{ detail.state?.exitCode ?? '-' }}</strong></div>
              <div><span>平台</span><strong>{{ detail.platform || '-' }}</strong></div>
              <div><span>创建</span><strong>{{ formatDateTime(detail.created) }}</strong></div>
              <div><span>启动</span><strong>{{ formatDateTime(detail.state?.startedAt) }}</strong></div>
            </div>
          </div>

          <div class="detail-section">
            <div class="detail-section__title">资源</div>
            <div class="detail-kv">
              <div><span>重启策略</span><strong>{{ detailRestartPolicy }}</strong></div>
              <div><span>网络模式</span><strong>{{ detailNetworkMode }}</strong></div>
              <div><span>内存</span><strong>{{ detailMemoryLimit }}</strong></div>
              <div><span>CPU</span><strong>{{ detailCpuLimit }}</strong></div>
              <div><span>自删</span><strong>{{ detailAutoRemove }}</strong></div>
              <div><span>特权</span><strong>{{ detailPrivileged }}</strong></div>
            </div>
          </div>
        </div>

        <div class="detail-section" v-if="detailNetworks.length">
          <div class="detail-section__title">网络</div>
          <div v-if="detailNetworks.length" class="network-list">
            <div v-for="item in detailNetworks" :key="item.name" class="network-item">
              <span class="network-name">{{ item.name }}</span>
              <span>{{ item.ipAddress || '-' }}</span>
              <span>{{ item.gateway || '-' }}</span>
              <span>{{ item.macAddress || '-' }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <div class="detail-section__title">启动</div>
          <div class="detail-code">{{ detailCommand }}</div>
        </div>

        <div v-if="detail.mounts?.length" class="detail-section">
          <div class="detail-section__title">挂载</div>
          <el-table :data="detail.mounts" size="small" border>
            <el-table-column prop="type" label="类型" width="80" />
            <el-table-column prop="source" label="来源" min-width="220" show-overflow-tooltip />
            <el-table-column prop="destination" label="目标" min-width="160" show-overflow-tooltip />
            <el-table-column prop="mode" label="模式" width="80" />
            <el-table-column label="读写" width="64">
              <template #default="{ row: m }">
                <BaseTag :text="m.rw ? 'RW' : 'RO'" :type="m.rw ? 'success' : 'warning'" />
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <template #footer>
        <el-button :icon="menuStore.iconComponents['HOutline:DocumentTextIcon']" :disabled="!detail" @click="openDetailLogs">
          日志
        </el-button>
        <el-button :icon="menuStore.iconComponents['HOutline:ChartBarIcon']" :disabled="!detail" @click="openDetailStats">
          统计
        </el-button>
        <el-button v-if="canManage" :icon="menuStore.iconComponents['HOutline:CommandLineIcon']" :disabled="!detail" @click="openDetailExec">
          终端
        </el-button>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </BaseDialog>

    <!-- Edit Dialog -->
    <BaseDialog v-model="editVisible" title="编辑容器" width="760">
      <el-form :model="editForm" label-position="top" class="container-edit-form">
        <el-row :gutter="12">
          <el-col :xs="24" :sm="12">
            <el-form-item label="容器名称">
              <el-input v-model="editForm.name" placeholder="容器名称" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="重启策略">
              <el-select v-model="editForm.restartPolicy" style="width: 100%">
                <el-option label="不重启" value="" />
                <el-option label="总是重启" value="always" />
                <el-option label="失败时重启" value="on-failure" />
                <el-option label="除非停止" value="unless-stopped" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :xs="24" :sm="12">
            <el-form-item label="内存限制 (MB)">
              <el-input-number v-model="editForm.memoryMB" :min="0" :step="64" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="CPU 限制 (核)">
              <el-input-number v-model="editForm.nanoCpuCores" :min="0" :step="0.5" :precision="1" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-divider />
        <el-checkbox v-model="editForm.recreate" style="margin-bottom: 8px">重建容器（需要修改镜像/端口/挂载等时勾选）</el-checkbox>
        <template v-if="editForm.recreate">
          <el-row :gutter="12">
            <el-col :xs="24" :sm="16">
              <el-form-item label="镜像" required>
                <el-input v-model="editForm.image" placeholder="选择或输入镜像" class="image-combo-input">
                  <template #suffix>
                    <span class="image-combo-actions">
                      <el-icon v-if="editForm.image" class="image-combo-clear" @click.stop="editForm.image = ''">
                        <component :is="menuStore.iconComponents['HSolid:XCircleIcon']" />
                      </el-icon>
                      <el-dropdown trigger="click" max-height="260" @visible-change="visible => visible && loadImageOptions()" @command="value => editForm.image = String(value)">
                        <el-icon><component :is="menuStore.iconComponents['Element:ArrowDown']" /></el-icon>
                        <template #dropdown>
                          <el-dropdown-menu>
                            <el-dropdown-item v-for="item in imageOptions" :key="item" :command="item">{{ item }}</el-dropdown-item>
                          </el-dropdown-menu>
                        </template>
                      </el-dropdown>
                    </span>
                  </template>
                </el-input>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="8">
              <el-form-item label="网络">
                <el-select v-model="editForm.network" clearable placeholder="默认 bridge" style="width: 100%" @focus="loadNetworkOptions">
                  <el-option v-for="item in networkOptions" :key="item" :label="item" :value="item" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <div class="edit-block">
            <div class="edit-block__head">
              <span>端口映射</span>
              <el-button type="primary" link size="small" :icon="menuStore.iconComponents.Plus" @click="addEditPort()">新增</el-button>
            </div>
            <div v-if="editPorts.length" class="edit-row-labels edit-row-labels--ports">
              <span>主机 IP</span>
              <span>主机端口</span>
              <span></span>
              <span>容器端口</span>
              <span>协议</span>
              <span></span>
            </div>
            <div v-for="(item, index) in editPorts" :key="`port-${index}`" class="edit-row edit-row--ports">
              <el-input v-model="item.hostIp" placeholder="0.0.0.0" />
              <el-input v-model="item.hostPort" placeholder="8080" />
              <span class="edit-row__arrow">-></span>
              <el-input v-model="item.containerPort" placeholder="80" />
              <el-select v-model="item.protocol" placeholder="协议">
                <el-option label="tcp" value="tcp" />
                <el-option label="udp" value="udp" />
              </el-select>
              <el-button type="danger" link :icon="menuStore.iconComponents.Delete" @click="removeEditPort(index)" />
            </div>
            <div v-if="!editPorts.length" class="edit-empty">暂无端口映射</div>
          </div>

          <div class="edit-block">
            <div class="edit-block__head">
              <span>卷挂载</span>
              <el-button type="primary" link size="small" :icon="menuStore.iconComponents.Plus" @click="addEditMount()">新增</el-button>
            </div>
            <div v-if="editMounts.length" class="edit-row-labels edit-row-labels--mounts">
              <span>类型</span>
              <span>来源</span>
              <span>目标</span>
              <span>只读</span>
              <span></span>
            </div>
            <div v-for="(item, index) in editMounts" :key="`mount-${index}`" class="edit-row edit-row--mounts">
              <el-select v-model="item.type" placeholder="类型">
                <el-option label="bind" value="bind" />
                <el-option label="volume" value="volume" />
                <el-option label="tmpfs" value="tmpfs" />
              </el-select>
              <el-input v-model="item.source" placeholder="来源" />
              <el-input v-model="item.target" placeholder="/data" />
              <el-switch v-model="item.readOnly" />
              <el-button type="danger" link :icon="menuStore.iconComponents.Delete" @click="removeEditMount(index)" />
            </div>
            <div v-if="!editMounts.length" class="edit-empty">暂无卷挂载</div>
          </div>

          <div class="edit-block">
            <div class="edit-block__head">
              <span>环境变量</span>
              <el-button type="primary" link size="small" :icon="menuStore.iconComponents.Plus" @click="addEditEnv()">新增</el-button>
            </div>
            <div v-if="editEnv.length" class="edit-row-labels edit-row-labels--env">
              <span>键</span>
              <span>值</span>
              <span></span>
            </div>
            <div v-for="(item, index) in editEnv" :key="`env-${index}`" class="edit-row edit-row--env">
              <el-input v-model="item.key" placeholder="KEY" />
              <el-input v-model="item.value" placeholder="VALUE" />
              <el-button type="danger" link :icon="menuStore.iconComponents.Delete" @click="removeEditEnv(index)" />
            </div>
            <div v-if="!editEnv.length" class="edit-empty">暂无环境变量</div>
          </div>

          <el-form-item label="强制删除旧容器" class="edit-force-item">
            <el-switch v-model="editForm.force" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="editSubmitting" @click="submitEdit">保存</el-button>
      </template>
    </BaseDialog>

    <DockerTaskDialogs v-model="taskDialogsVisible" :active-task="activeTask" @finished="handleTaskFinished" />
    <DockerContainerLogsDialog
      v-model="logsVisible"
      :container-id="logsContainerId"
      :container-name="logsContainerName"
      :ws-path="logsWsPath"
    />
    <DockerContainerStatsDialog
      v-model="statsVisible"
      :container-id="statsContainerId"
      :container-name="statsContainerName"
    />
    <DockerContainerExecDialog
      v-model="execVisible"
      :container-id="execContainerId"
      :container-name="execContainerName"
      :ws-path="execWsPath"
    />
  </div>
</template>

<script setup lang="ts">
import {
  createDockerContainer, deleteDockerContainer,
  getDockerContainer, killDockerContainer, listDockerContainers,
  listDockerImages, listDockerNetworks,
  pauseDockerContainer, restartDockerContainer, startDockerContainer,
  stopDockerContainer, unpauseDockerContainer, updateDockerContainer,
} from '@/api/docker'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import DockerTaskDialogs from '@/views/docker/components/DockerTaskDialogs.vue'
import DockerContainerLogsDialog from '@/views/docker/components/DockerContainerLogsDialog.vue'
import DockerContainerStatsDialog from '@/views/docker/components/DockerContainerStatsDialog.vue'
import DockerContainerExecDialog from '@/views/docker/components/DockerContainerExecDialog.vue'
import BaseTag from '@/components/tag/BaseTag.vue'
import TablePagination from '@/components/pagination/TablePagination.vue'
import { VxeGrid } from '@/plugins/vxeGrid'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import type { DockerContainerInfo, DockerContainerSummary, DockerPortBinding, DockerMount, DockerTaskInfo } from '@/types/v1/docker'
import type { ComponentPublicInstance } from 'vue'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'DockerContainerView' })

const menuStore = useMenuStore()
const canManage = useButtonPermission([PERM.DOCKER_CONTAINER_MANAGE], [])

const list = ref<DockerContainerSummary[]>([])
const loading = ref(false)
const queryForm = reactive({ name: '', status: '', image: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const STATE_MAP: Record<string, string> = { running: '运行中', exited: '已停止', paused: '暂停', restarting: '重启中', created: '已创建', removing: '删除中', dead: '已死亡' }
const STATE_TAG: Record<string, 'success' | 'info' | 'warning' | 'danger'> = { running: 'success', exited: 'info', paused: 'warning', restarting: 'warning', created: 'info', dead: 'danger' }
const OPERATION_COLUMN_WIDTH = 210
const OPERATION_COLUMN_READONLY_WIDTH = 96

const stateLabel = (s: string) => STATE_MAP[s] || s || '-'
const stateTagType = (s: string) => STATE_TAG[s] || 'info'
const displayNames = (row: DockerContainerSummary) => (row.names || []).map(n => n.replace(/^\//, '')).join(', ') || '-'
const DURATION_UNIT_MAP: Record<string, string> = {
  second: '秒',
  seconds: '秒',
  minute: '分钟',
  minutes: '分钟',
  hour: '小时',
  hours: '小时',
  day: '天',
  days: '天',
  week: '周',
  weeks: '周',
  month: '个月',
  months: '个月',
  year: '年',
  years: '年',
}

const formatDockerDuration = (value: string) => {
  const text = value.trim()
  const lower = text.toLowerCase()
  if (lower === 'less than a second') return '<1 秒'
  if (lower === 'about a minute') return '约 1 分钟'
  if (lower === 'about an hour') return '约 1 小时'
  const parts: string[] = []
  for (const match of text.matchAll(/(\d+)\s+([a-z]+)/gi)) {
    const amount = match[1]
    const unit = match[2]
    if (!amount || !unit) continue
    parts.push(`${amount} ${DURATION_UNIT_MAP[unit.toLowerCase()] || unit}`)
  }
  return parts.length ? parts.join(' ') : text
}

const formatContainerRuntime = (row: DockerContainerSummary) => {
  if (row.state !== 'running') return '-'
  const match = (row.status || '').match(/^Up\s+(.+?)(?:\s+\(.+\))?$/i)
  return match?.[1] ? formatDockerDuration(match[1]) : '-'
}

const containerNetworkItems = (row: DockerContainerSummary) => {
  if (row.networkMode === 'host' || row.networks?.includes('host')) return ['host']
  const endpoints = row.networkEndpoints || []
  if (endpoints.length) {
    return endpoints
      .map(endpoint => endpoint.ipAddress ? `${endpoint.ipAddress} · ${endpoint.name}` : endpoint.name)
      .filter(Boolean)
  }
  return (row.networks || []).filter(Boolean)
}

const formatNetworkText = (row: DockerContainerSummary) => containerNetworkItems(row).join('，') || '-'

const gridConfig = computed<VxeGridProps>(() => ({
  border: true, showOverflow: true, rowConfig: { isHover: true },
  data: list.value,
  columns: [
    { type: 'seq', title: '#', width: 50, fixed: 'left' },
    { field: 'names', title: '名称', minWidth: 160, slots: { default: 'column-names' } },
    { field: 'image', title: '镜像', minWidth: 180, showOverflow: true },
    { field: 'state', title: '状态', width: 90, slots: { default: 'column-state' } },
    { field: 'status', title: '运行时间', width: 120, slots: { default: 'column-runtime' } },
    { field: 'networkEndpoints', title: '网络', minWidth: 220, slots: { default: 'column-network' } },
    { title: '操作', width: canManage.value ? OPERATION_COLUMN_WIDTH : OPERATION_COLUMN_READONLY_WIDTH, fixed: 'right', slots: { default: 'column-operation' } },
  ],
}))

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listDockerContainers({
      all: true, page: pagination.value.page, pageSize: pagination.value.pageSize,
      status: queryForm.status || '', name: queryForm.name || '', image: queryForm.image || '',
      network: '', labels: {},
    })
    list.value = data?.items || []
    pagination.value.total = Number(data?.total || 0)
  } catch {
    list.value = []
    pagination.value.total = 0
  } finally { loading.value = false }
}

const search = () => { pagination.value.page = 1; getList() }
const reset = () => { queryForm.name = ''; queryForm.status = ''; queryForm.image = ''; search() }

const taskDialogsVisible = ref(false)
const activeTask = ref<DockerTaskInfo | null>(null)
const openTasks = () => { taskDialogsVisible.value = true }
const openTask = (task: DockerTaskInfo | undefined) => {
  if (!task?.id) return
  activeTask.value = task
  taskDialogsVisible.value = true
}
const handleTaskFinished = async () => {
  await getList()
}

const imageOptions = ref<string[]>([])
const networkOptions = ref<string[]>([])
const imageOptionsLoading = ref(false)
const networkOptionsLoading = ref(false)

const loadImageOptions = async () => {
  if (imageOptionsLoading.value || imageOptions.value.length) return
  imageOptionsLoading.value = true
  try {
    const { data } = await listDockerImages({ all: true, keyword: '', labels: {}, page: 1, pageSize: 200 })
    imageOptions.value = Array.from(new Set((data?.items || [])
      .flatMap(item => item.repoTags || [])
      .filter(item => item && item !== '<none>:<none>')))
  } finally {
    imageOptionsLoading.value = false
  }
}

const loadNetworkOptions = async () => {
  if (networkOptionsLoading.value || networkOptions.value.length) return
  networkOptionsLoading.value = true
  try {
    const { data } = await listDockerNetworks({ name: '', driver: '', scope: '', labels: {}, page: 1, pageSize: 200 })
    networkOptions.value = Array.from(new Set((data?.items || [])
      .map(item => item.name)
      .filter(Boolean)))
  } finally {
    networkOptionsLoading.value = false
  }
}

// -- create --
const createVisible = ref(false)
const createSubmitting = ref(false)
const createForm = reactive({ name: '', image: '', cmdText: '', envText: '', portsText: '', mountsText: '', network: '', restartPolicy: '' })
const openCreate = () => {
  Object.assign(createForm, { name: '', image: '', cmdText: '', envText: '', portsText: '', mountsText: '', network: '', restartPolicy: '' })
  createVisible.value = true
  void loadImageOptions()
  void loadNetworkOptions()
}
const parsePorts = (text: string): DockerPortBinding[] =>
  text.split('\n').filter(Boolean).map(line => {
    const [hostPort, containerPort] = line.trim().split(':')
    return { containerPort: containerPort || hostPort || '', hostPort: containerPort ? hostPort || '' : '', hostIp: '' }
  })
const parseMounts = (text: string): DockerMount[] =>
  text.split('\n').filter(Boolean).map(line => {
    const [source, target] = line.trim().split(':')
    return { type: !!source ? 'bind' : 'volume', source: source || '', target: target || '', readOnly: false }
  })
const submitCreate = async () => {
  const image = createForm.image.trim()
  if (!image) { ElMessage.error('请输入镜像名'); return }
  createSubmitting.value = true
  try {
    await createDockerContainer({
      options: {
        name: (createForm.name.trim() || undefined) as unknown as string,
        image,
        cmd: createForm.cmdText.trim() ? createForm.cmdText.trim().split(/\s+/) : [],
        env: createForm.envText.trim() ? createForm.envText.trim().split('\n').filter(Boolean) : [],
        ports: parsePorts(createForm.portsText),
        mounts: parseMounts(createForm.mountsText),
        network: (createForm.network.trim() || undefined) as unknown as string,
        restartPolicy: (createForm.restartPolicy || undefined) as unknown as string,
        hostname: '', user: '', entrypoint: [], workingDir: '', labels: {},
        tty: false, openStdin: false, autoRemove: false, privileged: false,
        memory: 0, memorySwap: 0, cpuShares: 0, cpuQuota: 0, cpuPeriod: 0, nanoCpus: 0, platform: '',
      },
    })
    ElMessage.success('容器创建成功')
    createVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, '创建容器失败') }
  finally { createSubmitting.value = false }
}

// -- actions --
type ContainerLifecycleAction = 'start' | 'stop' | 'restart' | 'kill' | 'pause' | 'unpause' | 'delete'
type ContainerCommand = ContainerLifecycleAction | 'logs' | 'stats' | 'exec' | 'edit' | 'detail'
type ContainerActionItem = {
  key: ContainerCommand
  label: string
  loading?: boolean
  disabled?: boolean
  danger?: boolean
  divided?: boolean
}

const DESKTOP_INLINE_ACTION_KEYS = new Set<ContainerCommand>(['pause', 'stop', 'restart', 'unpause', 'start'])
const MOBILE_INLINE_ACTION_KEYS = new Set<ContainerCommand>(['stop', 'start', 'detail'])
const ACTION_BUTTON_WIDTH = 42
const ACTION_MORE_WIDTH = 30
const ACTION_MIN_INLINE_WIDTH = 92

const actionMap: Record<ContainerLifecycleAction, { label: string; fn: (id: string) => Promise<unknown>; confirm: boolean }> = {
  start: { label: '启动', fn: (id: string) => startDockerContainer({ id }), confirm: false },
  stop: { label: '停止', fn: (id: string) => stopDockerContainer({ id, timeoutSeconds: 10 }), confirm: true },
  restart: { label: '重启', fn: (id: string) => restartDockerContainer({ id, timeoutSeconds: 10 }), confirm: true },
  kill: { label: '强制停止', fn: (id: string) => killDockerContainer({ id, signal: 'SIGKILL' }), confirm: true },
  pause: { label: '暂停', fn: (id: string) => pauseDockerContainer({ id }), confirm: false },
  unpause: { label: '恢复', fn: (id: string) => unpauseDockerContainer({ id }), confirm: false },
  delete: { label: '删除', fn: (id: string) => deleteDockerContainer({ id, force: false, removeVolumes: false }), confirm: true },
}

const actionLoading = reactive<Record<string, string>>({})
const isRowActionLoading = (row: DockerContainerSummary) => !!actionLoading[row.id]
const isActionLoading = (row: DockerContainerSummary, action: ContainerLifecycleAction) => actionLoading[row.id] === action

const actionCellWidths = reactive<Record<string, number>>({})
const actionCellElements = new Map<string, HTMLElement>()
let actionResizeObserver: ResizeObserver | null = null

const ensureActionResizeObserver = () => {
  if (actionResizeObserver || typeof ResizeObserver === 'undefined') return
  actionResizeObserver = new ResizeObserver((entries) => {
    entries.forEach((entry) => {
      const id = (entry.target as HTMLElement).dataset.rowId
      if (id) actionCellWidths[id] = entry.contentRect.width
    })
  })
}

const setActionCellRef = (id: string, el: Element | ComponentPublicInstance | null) => {
  ensureActionResizeObserver()
  const element = el instanceof HTMLElement ? el : null
  const oldElement = actionCellElements.get(id)
  if (oldElement && oldElement !== element) actionResizeObserver?.unobserve(oldElement)
  if (!element) {
    actionCellElements.delete(id)
    delete actionCellWidths[id]
    return
  }
  actionCellElements.set(id, element)
  actionCellWidths[id] = element.clientWidth
  actionResizeObserver?.observe(element)
}

const containerActions = (row: DockerContainerSummary): ContainerActionItem[] => {
  const disabled = isRowActionLoading(row)
  if (!canManage.value) return [{ key: 'detail', label: '详情', disabled }]

  const actions: ContainerActionItem[] = []
  if (row.state === 'running') {
    actions.push(
      { key: 'pause', label: '暂停', loading: isActionLoading(row, 'pause'), disabled },
      { key: 'stop', label: '停止', loading: isActionLoading(row, 'stop'), disabled },
      { key: 'restart', label: '重启', loading: isActionLoading(row, 'restart'), disabled },
      { key: 'logs', label: '日志', disabled },
      { key: 'stats', label: '统计', disabled },
      { key: 'exec', label: '终端', disabled },
    )
  } else if (row.state === 'paused') {
    actions.push({ key: 'unpause', label: '恢复', loading: isActionLoading(row, 'unpause'), disabled })
  } else if (row.state === 'exited' || row.state === 'created') {
    actions.push({ key: 'start', label: '启动', loading: isActionLoading(row, 'start'), disabled })
  }

  actions.push(
    { key: 'edit', label: '编辑', disabled },
    { key: 'detail', label: '详情', disabled },
    { key: 'delete', label: '删除', loading: isActionLoading(row, 'delete'), disabled, danger: true, divided: true },
  )
  return actions
}

const desktopInlineLimit = (row: DockerContainerSummary, preferredCount: number, hasMore: boolean) => {
  if (!canManage.value) return preferredCount
  const width = actionCellWidths[row.id] || OPERATION_COLUMN_WIDTH
  if (width < ACTION_MIN_INLINE_WIDTH) return 0
  const reservedWidth = hasMore ? ACTION_MORE_WIDTH : 0
  const count = Math.floor((width - reservedWidth) / ACTION_BUTTON_WIDTH)
  return Math.max(1, Math.min(preferredCount, count))
}

const desktopInlineActions = (row: DockerContainerSummary) => {
  const actions = containerActions(row)
  if (!canManage.value) return actions
  const preferred = actions.filter(item => DESKTOP_INLINE_ACTION_KEYS.has(item.key))
  const limit = desktopInlineLimit(row, preferred.length, actions.length > preferred.length)
  return preferred.slice(0, limit)
}

const desktopMoreActions = (row: DockerContainerSummary) => {
  const inlineKeys = new Set(desktopInlineActions(row).map(item => item.key))
  return containerActions(row).filter(item => !inlineKeys.has(item.key))
}

const mobileInlineActions = (row: DockerContainerSummary) => {
  return containerActions(row).filter(item => MOBILE_INLINE_ACTION_KEYS.has(item.key))
}

const mobileMoreActions = (row: DockerContainerSummary) => {
  const inlineKeys = new Set(mobileInlineActions(row).map(item => item.key))
  return containerActions(row).filter(item => !inlineKeys.has(item.key))
}

const handleAction = async (row: DockerContainerSummary, action: ContainerLifecycleAction) => {
  const def = actionMap[action]
  if (isRowActionLoading(row)) return
  const name = displayNames(row)
  if (def.confirm) {
    try { await Dialog.confirm({ title: `确认${def.label}`, content: `确定要${def.label}容器「${name}」吗？`, confirmText: `确认${def.label}`, cancelText: '取消' }) }
    catch { return }
  }
  actionLoading[row.id] = action
  try {
    await def.fn(row.id)
    ElMessage.success(`${def.label}操作成功`)
    await getList()
  } catch (e) { showRequestError(e, `${def.label}操作失败`) }
  finally { delete actionLoading[row.id] }
}

const handleContainerActionCommand = (row: DockerContainerSummary, command: string) => {
  if (command in actionMap) {
    void handleAction(row, command as ContainerLifecycleAction)
    return
  }
  if (command === 'logs') {
    void openLogs(row)
    return
  }
  if (command === 'stats') {
    void openStats(row)
    return
  }
  if (command === 'exec') {
    void openExec(row)
    return
  }
  if (command === 'edit') {
    void openEdit(row)
    return
  }
  if (command === 'detail') {
    void openDetail(row)
  }
}

// -- detail --
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<DockerContainerInfo | null>(null)

const shortId = (id: string) => id ? id.slice(0, 12) : '-'
const formatDateTime = (value?: string) => {
  if (!value || value.startsWith('0001-')) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

const formatBytes = (bytes?: number | string) => {
  const n = Number(bytes)
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0; let v = n
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

const detailCommand = computed(() => {
  const item = detail.value
  if (!item) return '-'
  return [item.path, ...(item.args || [])].filter(Boolean).join(' ') || '-'
})

const detailHostConfig = computed(() => detail.value?.hostConfig)
const detailNetwork = computed(() => detail.value?.network)
const detailConfig = computed(() => detail.value?.config)
const detailImageName = computed(() => {
  const image = detailConfig.value?.image || detail.value?.image || ''
  return image || '-'
})
const detailRestartPolicy = computed(() => detailHostConfig.value?.restartPolicy || '-')
const detailNetworkMode = computed(() => detailHostConfig.value?.networkMode || '-')
const detailAutoRemove = computed(() => detailHostConfig.value?.autoRemove ? '是' : '否')
const detailPrivileged = computed(() => detailHostConfig.value?.privileged ? '是' : '否')
const detailMemoryLimit = computed(() => {
  const memory = detailHostConfig.value?.memory || 0
  return memory ? formatBytes(memory) : '未限制'
})
const detailCpuLimit = computed(() => {
  const nanoCpus = detailHostConfig.value?.nanoCpus || 0
  if (nanoCpus) return `${(nanoCpus / 1e9).toFixed(2)} 核`
  const cpuQuota = detailHostConfig.value?.cpuQuota || 0
  const cpuPeriod = detailHostConfig.value?.cpuPeriod || 0
  if (cpuQuota && cpuPeriod) return `${(cpuQuota / cpuPeriod).toFixed(2)} 核`
  return '未限制'
})

type DetailEndpoint = {
  name: string
  ipAddress: string
  gateway: string
  macAddress: string
}

const detailNetworks = computed<DetailEndpoint[]>(() => {
  const networks = detailNetwork.value?.networks || {}
  return Object.entries(networks).map(([name, item]) => ({
    name,
    ipAddress: item?.ipAddress || '',
    gateway: item?.gateway || '',
    macAddress: item?.macAddress || '',
  }))
})

const detailPortTags = computed(() => {
  const bindings = detailHostConfig.value?.portBindings || []
  return bindings.map((binding) => {
    const containerPort = binding.containerPort || ''
    const hostIp = binding.hostIp || ''
    const hostPort = binding.hostPort || ''
    return hostPort ? `${hostIp ? `${hostIp}:` : ''}${hostPort}->${containerPort}` : containerPort
  }).filter(Boolean)
})
const detailPortText = computed(() => {
  if (detailNetworkMode.value === 'host') return 'host 网络'
  return '无端口映射'
})

const openDetailLogs = () => {
  if (!detail.value) return
  if (!detail.value.logsWsPath) {
    ElMessage.warning('缺少日志 WS 路径')
    return
  }
  logsContainerId.value = detail.value.id
  logsContainerName.value = detail.value.name || detail.value.id
  logsWsPath.value = detail.value.logsWsPath
  logsVisible.value = true
}

const openDetailStats = () => {
  if (!detail.value) return
  statsContainerId.value = detail.value.id
  statsContainerName.value = detail.value.name || detail.value.id
  statsVisible.value = true
}

const openDetailExec = () => {
  if (!detail.value) return
  if (!detail.value.execWsPath) {
    ElMessage.warning('缺少 Exec WS 路径')
    return
  }
  execContainerId.value = detail.value.id
  execContainerName.value = detail.value.name || detail.value.id
  execWsPath.value = detail.value.execWsPath
  execVisible.value = true
}

const openDetail = async (row: DockerContainerSummary) => {
  detailVisible.value = true; detail.value = null; detailLoading.value = true
  try {
    const { data } = await getDockerContainer({ id: row.id })
    detail.value = data?.info || null
  } catch (e) { showRequestError(e, '获取容器详情失败') }
  finally { detailLoading.value = false }
}

// -- logs --
const logsVisible = ref(false)
const logsContainerId = ref('')
const logsContainerName = ref('')
const logsWsPath = ref('')
const openLogs = async (row: DockerContainerSummary) => {
  try {
    const { data } = await getDockerContainer({ id: row.id })
    const info = data?.info
    if (!info?.logsWsPath) {
      ElMessage.warning('缺少日志 WS 路径')
      return
    }
    logsContainerId.value = info.id || row.id
    logsContainerName.value = info.name || displayNames(row)
    logsWsPath.value = info.logsWsPath
    logsVisible.value = true
  } catch (e) {
    showRequestError(e, '获取容器详情失败')
  }
}

// -- stats --
const statsVisible = ref(false)
const statsContainerId = ref('')
const statsContainerName = ref('')
const openStats = (row: DockerContainerSummary) => {
  statsContainerId.value = row.id
  statsContainerName.value = displayNames(row)
  statsVisible.value = true
}

// -- exec --
const execVisible = ref(false)
const execContainerId = ref('')
const execContainerName = ref('')
const execWsPath = ref('')
const openExec = async (row: DockerContainerSummary) => {
  try {
    const { data } = await getDockerContainer({ id: row.id })
    const info = data?.info
    if (!info?.execWsPath) {
      ElMessage.warning('缺少 Exec WS 路径')
      return
    }
    execContainerId.value = info.id || row.id
    execContainerName.value = info.name || displayNames(row)
    execWsPath.value = info.execWsPath
    execVisible.value = true
  } catch (e) {
    showRequestError(e, '获取容器详情失败')
  }
}

// -- edit --
const editVisible = ref(false)
const editSubmitting = ref(false)
const editId = ref('')

type EditablePort = { hostIp: string; hostPort: string; containerPort: string; protocol: 'tcp' | 'udp' }
type EditableMount = { type: string; source: string; target: string; readOnly: boolean }
type EditableEnv = { key: string; value: string }
type LooseRecord = Record<string, unknown>

const editForm = reactive({ name: '', restartPolicy: '', memoryMB: 0, nanoCpuCores: 0, recreate: false, image: '', network: '', force: false })
const editPorts = ref<EditablePort[]>([])
const editMounts = ref<EditableMount[]>([])
const editEnv = ref<EditableEnv[]>([])

const asRecord = (value: unknown): LooseRecord => (value && typeof value === 'object' && !Array.isArray(value) ? value as LooseRecord : {})
const readString = (source: LooseRecord, ...keys: string[]) => {
  for (const key of keys) {
    const value = source[key]
    if (value !== undefined && value !== null && value !== '') return String(value)
  }
  return ''
}
const resetEditForm = () => {
  Object.assign(editForm, { name: '', restartPolicy: '', memoryMB: 0, nanoCpuCores: 0, recreate: false, image: '', network: '', force: false })
  editPorts.value = []
  editMounts.value = []
  editEnv.value = []
}
const splitPortProtocol = (value: string): Pick<EditablePort, 'containerPort' | 'protocol'> => {
  const [port, protocol] = value.trim().split('/')
  return { containerPort: port || '', protocol: protocol?.toLowerCase() === 'udp' ? 'udp' : 'tcp' }
}
const addEditPort = (item?: Partial<EditablePort>) => {
  editPorts.value.push({
    hostIp: item?.hostIp || '',
    hostPort: item?.hostPort || '',
    containerPort: item?.containerPort || '',
    protocol: item?.protocol || 'tcp',
  })
}
const removeEditPort = (index: number) => {
  editPorts.value.splice(index, 1)
}
const addEditMount = (item?: Partial<EditableMount>) => {
  editMounts.value.push({ type: item?.type || 'bind', source: item?.source || '', target: item?.target || '', readOnly: !!item?.readOnly })
}
const removeEditMount = (index: number) => {
  editMounts.value.splice(index, 1)
}
const addEditEnv = (item?: Partial<EditableEnv>) => {
  editEnv.value.push({ key: item?.key || '', value: item?.value || '' })
}
const removeEditEnv = (index: number) => {
  editEnv.value.splice(index, 1)
}
const fillEditPorts = (items: unknown) => {
  if (!Array.isArray(items)) return
  editPorts.value = items.map((raw) => {
    const item = asRecord(raw)
    const port = splitPortProtocol(readString(item, 'ContainerPort', 'containerPort'))
    return {
      hostIp: readString(item, 'HostIP', 'HostIp', 'hostIp', 'hostIP'),
      hostPort: readString(item, 'HostPort', 'hostPort'),
      containerPort: port.containerPort,
      protocol: port.protocol,
    }
  }).filter(item => item.containerPort)
}
const fillEditMounts = (hostMounts: unknown, points: DockerContainerInfo['mounts']) => {
  if (Array.isArray(hostMounts) && hostMounts.length) {
    editMounts.value = hostMounts.map((raw) => {
      const item = asRecord(raw)
      return {
        type: readString(item, 'Type', 'type') || 'bind',
        source: readString(item, 'Source', 'source'),
        target: readString(item, 'Target', 'target'),
        readOnly: !!(item.ReadOnly ?? item.readOnly),
      }
    }).filter(item => item.target)
    return
  }
  editMounts.value = (points || []).map((item) => ({
    type: item.type || 'bind',
    source: item.source || item.name || '',
    target: item.destination || '',
    readOnly: !item.rw,
  })).filter(item => item.target)
}
const fillEditEnv = (items: unknown) => {
  if (!Array.isArray(items)) return
  editEnv.value = items.map((raw) => {
    const text = String(raw)
    const index = text.indexOf('=')
    return index >= 0
      ? { key: text.slice(0, index), value: text.slice(index + 1) }
      : { key: text, value: '' }
  }).filter(item => item.key)
}
const buildEditPorts = (): DockerPortBinding[] =>
  editPorts.value.map(item => ({
    hostIp: item.hostIp.trim(),
    hostPort: item.hostPort.trim(),
    containerPort: `${item.containerPort.trim()}/${item.protocol || 'tcp'}`,
  })).filter(item => item.containerPort)
const buildEditMounts = (): DockerMount[] =>
  editMounts.value.map(item => ({
    type: item.type || 'bind',
    source: item.source.trim(),
    target: item.target.trim(),
    readOnly: item.readOnly,
  })).filter(item => item.target)
const buildEditEnv = () =>
  editEnv.value
    .filter(item => item.key.trim())
    .map(item => `${item.key.trim()}=${item.value}`)

const openEdit = async (row: DockerContainerSummary) => {
  editId.value = row.id
  resetEditForm()
  editForm.name = displayNames(row)
  editForm.image = row.image
  editVisible.value = true
  void loadImageOptions()
  void loadNetworkOptions()
  try {
    const { data } = await getDockerContainer({ id: row.id })
    const info = data?.info
    if (info) {
      editForm.name = info.name || editForm.name
      const config = info.config
      const hostConfig = info.hostConfig
      editForm.image = config?.image || info.image || editForm.image
      editForm.restartPolicy = hostConfig?.restartPolicy || ''
      editForm.network = hostConfig?.networkMode || ''
      const memory = hostConfig?.memory || 0
      const nanoCpus = hostConfig?.nanoCpus || 0
      if (memory) editForm.memoryMB = Math.round(memory / 1024 / 1024)
      if (nanoCpus) editForm.nanoCpuCores = Math.round((nanoCpus / 1e9) * 10) / 10
      fillEditPorts(hostConfig?.portBindings)
      fillEditMounts(hostConfig?.mounts, info.mounts)
      fillEditEnv(config?.env)
    }
  } catch { /* use defaults */ }
}
const submitEdit = async () => {
  editSubmitting.value = true
  try {
    if (editForm.recreate) {
      if (!editForm.image.trim()) { ElMessage.error('请输入镜像名'); editSubmitting.value = false; return }
      const { data } = await updateDockerContainer({
        id: editId.value,
        name: editForm.name.trim(),
        restartPolicy: editForm.restartPolicy,
        memory: editForm.memoryMB * 1024 * 1024,
        memorySwap: 0, cpuShares: 0, cpuQuota: 0, cpuPeriod: 0,
        nanoCpus: Math.round(editForm.nanoCpuCores * 1e9),
        recreate: true,
        force: editForm.force,
        removeVolumes: false,
        options: {
          name: (editForm.name.trim() || undefined) as unknown as string,
          image: editForm.image.trim(),
          cmd: [], env: buildEditEnv(),
          ports: buildEditPorts(),
          mounts: buildEditMounts(),
          network: (editForm.network.trim() || undefined) as unknown as string,
          restartPolicy: (editForm.restartPolicy || undefined) as unknown as string,
          hostname: '', user: '', entrypoint: [], workingDir: '', labels: {},
          tty: false, openStdin: false, autoRemove: false, privileged: false,
          memory: editForm.memoryMB * 1024 * 1024, memorySwap: 0,
          cpuShares: 0, cpuQuota: 0, cpuPeriod: 0,
          nanoCpus: Math.round(editForm.nanoCpuCores * 1e9), platform: '',
        },
      })
      openTask(data?.task)
    } else {
      await updateDockerContainer({
        id: editId.value,
        name: editForm.name.trim(),
        restartPolicy: editForm.restartPolicy,
        memory: editForm.memoryMB * 1024 * 1024,
        memorySwap: 0, cpuShares: 0, cpuQuota: 0, cpuPeriod: 0,
        nanoCpus: Math.round(editForm.nanoCpuCores * 1e9),
        recreate: false, force: false, removeVolumes: false, options: undefined,
      })
    }
    ElMessage.success(editForm.recreate ? '容器重建任务已创建' : '容器更新成功')
    editVisible.value = false
    if (!editForm.recreate) await getList()
  } catch (e) { showRequestError(e, '更新容器失败') }
  finally { editSubmitting.value = false }
}

onMounted(() => { getList() })
onBeforeUnmount(() => {
  actionResizeObserver?.disconnect()
  actionCellElements.clear()
})
</script>

<style scoped lang="scss">
.docker-page { .card-mt-16 { margin-top: 16px; } }
.operation-container { margin-bottom: 12px; }
.container-actions {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
}
.container-actions :deep(.el-button) {
  margin-left: 0;
  padding: 0 0.1rem;
}
.container-actions__more {
  min-width: 1.4rem;
  font-weight: 700;
}
.container-actions__danger {
  color: var(--el-color-danger);
}
.port-tag {
  display: inline-block; font-size: 0.72rem; background: var(--el-fill-color-light);
  padding: 1px 6px; border-radius: 4px; margin-right: 4px; margin-bottom: 2px;
}
.network-tag {
  display: inline-block;
  max-width: 9.5rem;
  padding: 1px 6px;
  margin-right: 4px;
  margin-bottom: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 0.72rem;
  vertical-align: middle;
}
.text-muted { color: var(--el-text-color-placeholder); font-size: 0.82rem; }
.container-detail {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.detail-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
  background: var(--el-fill-color-lighter);
}
.detail-hero__main {
  min-width: 0;
  flex: 1;
}
.detail-hero__title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.detail-hero__title > span:first-child {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.detail-hero__sub {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.35rem;
  font-size: 0.78rem;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.detail-network-summary {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.35rem;
  max-width: 42%;
}
.detail-network-summary__mode {
  color: var(--el-text-color-secondary);
  font-size: 0.78rem;
  font-weight: 700;
}
.detail-network-summary__ports {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 0.25rem;
}
.detail-section-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}
.detail-section {
  min-width: 0;
}
.detail-section__title {
  margin-bottom: 0.4rem;
  font-size: 0.86rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.detail-kv {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
  overflow: hidden;
}
.detail-kv > div {
  min-width: 0;
  padding: 0.45rem 0.55rem;
  border-right: 1px solid var(--el-border-color-extra-light);
  border-bottom: 1px solid var(--el-border-color-extra-light);
}
.detail-kv > div:nth-child(2n) {
  border-right: 0;
}
.detail-kv > div:nth-last-child(-n + 2) {
  border-bottom: 0;
}
.detail-kv span {
  display: block;
  color: var(--el-text-color-secondary);
  font-size: 0.72rem;
}
.detail-kv strong {
  display: block;
  margin-top: 0.12rem;
  color: var(--el-text-color-primary);
  font-size: 0.8rem;
  font-weight: 600;
  word-break: break-word;
}
.network-list {
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
  overflow: hidden;
}
.network-item {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr 1.2fr;
  gap: 0.5rem;
  padding: 0.45rem 0.55rem;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  font-size: 0.78rem;
  color: var(--el-text-color-regular);
}
.network-item:last-child {
  border-bottom: 0;
}
.network-name {
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.detail-code {
  padding: 0.5rem 0.6rem;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-regular);
  font-size: 0.78rem;
  line-height: 1.45;
  word-break: break-all;
}
h4 { margin: 0 0 8px; font-size: 0.9rem; }
.container-edit-form :deep(.el-form-item) {
  margin-bottom: 12px;
}
.container-edit-form :deep(.el-divider--horizontal) {
  margin: 4px 0 12px;
}
.image-combo-input,
.image-combo-input :deep(.el-input__wrapper) {
  width: 100%;
}
.image-combo-actions {
  display: inline-flex;
  align-items: center;
  position: relative;
  width: 14px;
}
.image-combo-input :deep(.el-dropdown) {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  line-height: 1;
}
.image-combo-clear {
  position: absolute;
  right: 18px;
  color: var(--el-text-color-placeholder);
}
.image-combo-input :deep(.el-icon) {
  cursor: pointer;
}
.edit-block {
  margin-bottom: 12px;
  padding: 10px;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
}
.edit-block__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 0.86rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.edit-row-labels,
.edit-row {
  display: grid;
  gap: 8px;
  align-items: center;
}
.edit-row-labels {
  margin-bottom: 4px;
  color: var(--el-text-color-secondary);
  font-size: 0.72rem;
}
.edit-row {
  margin-bottom: 6px;
}
.edit-row:last-child {
  margin-bottom: 0;
}
.edit-row--ports,
.edit-row-labels--ports {
  grid-template-columns: minmax(90px, 1fr) minmax(82px, 1fr) 22px minmax(90px, 1fr) 82px 28px;
}
.edit-row--mounts,
.edit-row-labels--mounts {
  grid-template-columns: 92px minmax(120px, 1fr) minmax(120px, 1fr) 44px 28px;
}
.edit-row--env,
.edit-row-labels--env {
  grid-template-columns: minmax(120px, 1fr) minmax(160px, 1.4fr) 28px;
}
.edit-row__arrow {
  color: var(--el-text-color-secondary);
  text-align: center;
  font-size: 0.78rem;
}
.edit-empty {
  padding: 4px 0;
  color: var(--el-text-color-placeholder);
  font-size: 0.78rem;
}
.edit-force-item {
  margin-bottom: 0;
}

@media (width <= 768px) {
  .detail-hero,
  .detail-section-grid {
    display: block;
  }
  .detail-network-summary {
    align-items: flex-start;
    max-width: none;
    margin-top: 0.5rem;
  }
  .detail-network-summary__ports {
    justify-content: flex-start;
  }
  .detail-section + .detail-section {
    margin-top: 0.75rem;
  }
  .detail-kv,
  .network-item {
    grid-template-columns: 1fr;
  }
  .detail-kv > div {
    border-right: 0;
  }
  .detail-kv > div:nth-last-child(-n + 2) {
    border-bottom: 1px solid var(--el-border-color-extra-light);
  }
  .detail-kv > div:last-child {
    border-bottom: 0;
  }
  .edit-row-labels {
    display: none;
  }
  .edit-row--ports,
  .edit-row--mounts,
  .edit-row--env {
    grid-template-columns: 1fr;
    padding-bottom: 8px;
    border-bottom: 1px solid var(--el-border-color-extra-light);
  }
  .edit-row:last-child {
    padding-bottom: 0;
    border-bottom: 0;
  }
  .edit-row__arrow {
    display: none;
  }
  .edit-row .el-button {
    justify-self: end;
  }
}

.mobile-card-list { display: flex; flex-direction: column; gap: 0.55rem; }
.mobile-card {
  display: flex; align-items: flex-start; gap: 0.6rem;
  padding: 0.7rem 0.8rem; border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.6rem; background: var(--el-bg-color);
  .mobile-card-body { flex: 1; min-width: 0; }
  .mobile-card-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
  .mobile-card-title { font-size: 0.88rem; font-weight: 700; }
  .mobile-card-meta { margin-top: 0.25rem; font-size: 0.74rem; color: var(--el-text-color-secondary); }
  .mobile-card-actions {
    display: flex;
    align-items: flex-start;
    justify-content: flex-end;
    flex-wrap: wrap;
    gap: 0.3rem;
    flex-shrink: 0;
    width: 6.2rem;
  }
  .mobile-card-actions .el-button + .el-button { margin-left: 0; }
  .mobile-card-actions :deep(.el-dropdown) { line-height: 1; }
}
</style>
