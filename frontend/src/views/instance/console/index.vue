<!-- 应用详情：返回链接 + 头部卡（贴纸 / 名称 / 状态 / 类型 · 运行时长或停止时间 · PID · 备注 + 启停 / 重启 / ⋯）
     + 资源占用（CPU / 内存 / 进程 / 目录占用，GetInstanceStats 轮询）+ 维护提示 + 下划线标签（控制台 / 文件 / 配置）
     + 控制台（TerminalConsole 真 PTY，常驻挂载）/ 文件（FileBrowser，首次打开时挂载）/ 配置（运行配置 + 应用信息 + 最近事件）。
     手机：紧凑头部 + 同样三个标签，⋯ 打开底部操作面板。一个页面一个地址：/instance/app/:id?tab=console|files|config，切换标签不换页。 -->
<template>
  <div class="app-detail" :class="{ 'is-mobile': isMobile }">
    <!-- ========== 头部 ========== -->
    <RouterLink v-if="!isMobile" to="/instance/list" class="app-detail__back">
      <component :is="menuStore.iconComponents['HOutline:ChevronLeftIcon']" />
      {{ t('instance.ui.backToList') }}
    </RouterLink>

    <section class="app-head" :class="{ 'is-compact': isMobile }">
      <div class="app-head__row">
        <RouterLink v-if="isMobile" to="/instance/list" class="app-head__back" :aria-label="t('instance.ui.backToList')">
          <component :is="menuStore.iconComponents['HOutline:ChevronLeftIcon']" />
        </RouterLink>
        <span class="app-head__av m-sticker" :class="`m-sticker--${tone}`">{{ initial }}</span>

        <div class="app-head__main">
          <div class="app-head__title-row">
            <h1 class="app-head__name" :title="displayName">{{ displayName }}</h1>
            <StatusPill v-if="!isMobile && statusPill" :variant="statusPill.variant" :label="statusPill.label" />
          </div>
          <div class="app-head__meta">
            <StatusPill v-if="isMobile && statusPill" :variant="statusPill.variant" :label="statusPill.label" />
            <template v-if="!isMobile && info">
              <RouterLink v-if="info.type" to="/instance/type" class="app-head__type">
                <component :is="menuStore.iconComponents['HOutline:CubeIcon']" />{{ info.type }}
              </RouterLink>
              <span v-else class="app-head__meta-item">
                <component :is="menuStore.iconComponents['HOutline:CubeIcon']" />{{ t('instance.ui.uncategorized') }}
              </span>
            </template>
            <span class="app-head__meta-item">
              <component :is="menuStore.iconComponents['HOutline:ClockIcon']" v-if="!isMobile" />{{ uptimeText }}
            </span>
            <span v-if="!isMobile && pid" class="app-head__meta-item m-num">PID {{ pid }}</span>
            <span v-if="!isMobile && info?.remark" class="app-head__meta-item app-head__remark" :title="info.remark">
              {{ info.remark }}
            </span>
          </div>
        </div>

        <!-- 操作：PC 文字按钮 + ⋯ 下拉；手机小图标 + ⋯ 底部面板 -->
        <div class="app-head__acts">
          <template v-if="!isMobile">
            <UButton
              :color="isRunning ? 'error' : 'primary'"
              :variant="isRunning ? 'soft' : 'solid'"
              :icon="isRunning ? 'i-lucide-square' : 'i-lucide-play'"
              class="app-head__btn"
              :loading="isBusy"
              :disabled="!info"
              @click="handleTogglePower"
            >
              {{ isRunning ? t('instance.stop') : t('instance.start') }}
            </UButton>
            <UButton
              color="neutral"
              variant="outline"
              icon="i-lucide-rotate-cw"
              class="app-head__btn"
              :disabled="isBusy || isMaintenance || !info"
              @click="handleRestart"
            >
              {{ t('instance.restart') }}
            </UButton>
            <AppDropdown align="end" :width="248">
              <template #trigger>
                <button type="button" class="app-head__more" :aria-label="t('instance.moreActions')" :title="t('instance.moreActions')">
                  <UIcon name="i-lucide-ellipsis" />
                </button>
              </template>
              <template #default="{ close }">
                <div class="app-menu" role="menu">
                  <template v-for="item in menuItems" :key="item.key">
                    <div v-if="item.divider" class="app-menu__sep" />
                    <button
                      v-else
                      type="button"
                      role="menuitem"
                      class="app-menu__item"
                      :class="{ 'is-danger': item.danger }"
                      :disabled="item.disabled"
                      @click="close(); onMore(item.key)"
                    >
                      <UIcon :name="item.icon || 'i-lucide-circle'" class="app-menu__ico" />
                      <span class="app-menu__text">
                        <span>{{ item.label }}</span>
                        <span v-if="item.description" class="app-menu__desc">{{ item.description }}</span>
                      </span>
                    </button>
                  </template>
                </div>
              </template>
            </AppDropdown>
          </template>
          <template v-else>
            <AppIconButton
              :icon="isRunning ? 'HOutline:StopIcon' : 'HOutline:PlayIcon'"
              :label="isRunning ? t('instance.stop') : t('instance.start')"
              :box="44"
              :class="isRunning ? 'is-stop' : 'is-start'"
              :disabled="isBusy || !info"
              @click="handleTogglePower"
            />
            <AppIconButton
              icon="HOutline:ArrowPathIcon"
              :label="t('instance.restart')"
              :box="44"
              :disabled="isBusy || isMaintenance || !info"
              @click="handleRestart"
            />
            <AppIconButton icon="HOutline:EllipsisHorizontalIcon" :label="t('instance.moreActions')" :box="44" @click="sheetOpen = true" />
          </template>
        </div>
      </div>

      <!-- 资源占用 -->
      <div v-if="!isMobile" class="app-stats">
        <div v-for="s in statCards" :key="s.key" class="app-stat">
          <div class="app-stat__top">
            <span class="app-stat__t">{{ s.label }}</span>
            <span class="app-stat__v m-num">{{ s.value }}</span>
            <span class="app-stat__sub m-num">{{ s.sub }}</span>
          </div>
          <div class="app-stat__bar"><span :style="{ width: `${s.pct}%`, background: s.color }" /></div>
        </div>
      </div>
    </section>

    <!-- 维护中提示 -->
    <div v-if="isMaintenance" class="app-maint" role="status">
      <UIcon name="i-lucide-wrench" class="app-maint__ico" />
      <span class="app-maint__text">{{ maintenanceText }}</span>
      <UButton color="warning" variant="soft" size="sm" class="app-maint__btn" @click="handleEndMaintenance">
        {{ t('instance.ui.endMaintenance') }}
      </UButton>
    </div>

    <!-- ========== 标签 ========== -->
    <nav class="app-tabs" role="tablist" :aria-label="t('instance.ui.detailTabs')">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        type="button"
        role="tab"
        class="app-tab"
        :class="{ 'is-on': activeTab === tab.key }"
        :aria-selected="activeTab === tab.key"
        @click="onTab(tab.key)"
      >
        <component :is="menuStore.iconComponents[tab.icon]" v-if="!isMobile" class="app-tab__ico" />
        {{ tab.label }}
      </button>
    </nav>

    <!-- ========== 主体 ========== -->
    <div class="app-body" :class="{ 'is-fill': activeTab !== 'config' }">
      <div class="app-main" :class="{ 'is-fill': activeTab !== 'config' }">
        <!-- 控制台始终挂载，切换标签只隐藏，避免重连与丢失回放 -->
        <div v-show="activeTab === 'console'" class="app-term">
          <TerminalConsole
            ref="term"
            variant="console"
            :title="t('instance.console')"
            :subtitle="consoleInfo"
            :status="socketStatus"
            :status-label="statusLabel"
            :actions="terminalActions"
            :busy="isBusy"
            :keys-open="keysOpen"
            @data="sendRaw"
            @resize="({ cols, rows }) => sendResize(cols, rows)"
            @action="onAction"
          >
            <template #below>
              <!-- 快捷命令：来自应用类型的设置；没有设置时显示最近发过的命令 -->
              <div v-if="quickCommands.length" class="app-quick">
                <span v-if="!isMobile" class="app-quick__label">
                  {{ typeQuickCommands.length ? t('instanceConsole.quickTitle') : t('instanceConsole.recentTitle') }}
                </span>
                <button
                  v-for="cmd in quickCommands"
                  :key="cmd"
                  type="button"
                  class="app-quick__chip m-mono"
                  :disabled="socketStatus !== 'connected'"
                  :title="cmd"
                  @click="runCommand(cmd)"
                >
                  {{ cmd }}
                </button>
              </div>
              <!-- 命令输入：回车发送一行命令，↑ / ↓ 翻历史；终端本身仍可直接键入 -->
              <form class="app-cmd" @submit.prevent="submitCommand">
                <label class="app-cmd__field">
                  <span class="app-cmd__prompt m-mono" aria-hidden="true">&gt;</span>
                  <input
                    v-model="command"
                    class="m-mono"
                    type="text"
                    autocomplete="off"
                    spellcheck="false"
                    enterkeyhint="send"
                    :placeholder="isMobile ? t('instanceConsole.cmdPlaceholderShort') : t('instanceConsole.cmdPlaceholder')"
                    :aria-label="t('instanceConsole.cmdLabel')"
                    :disabled="socketStatus !== 'connected'"
                    @keydown.up.prevent="historyStep(-1)"
                    @keydown.down.prevent="historyStep(1)"
                  />
                </label>
                <button
                  v-if="isMobile"
                  type="button"
                  class="app-cmd__kb"
                  :class="{ 'is-on': keysOpen }"
                  :aria-label="keysOpen ? t('instance.ui.keysHide') : t('instance.ui.keysShow')"
                  :aria-pressed="keysOpen"
                  @click="toggleKeys"
                >
                  <UIcon name="i-lucide-keyboard" />
                </button>
                <UButton
                  type="submit"
                  color="primary"
                  icon="i-lucide-send-horizontal"
                  class="app-cmd__send"
                  :class="{ 'is-icon': isMobile }"
                  :aria-label="t('instanceConsole.cmdSend')"
                  :disabled="socketStatus !== 'connected' || !command.trim()"
                >
                  <template v-if="!isMobile">{{ t('instanceConsole.cmdSend') }}</template>
                </UButton>
              </form>
            </template>
          </TerminalConsole>
        </div>

        <!-- 文件：首次切到「文件」时挂载，之后只隐藏，保留当前目录、选择与编辑状态 -->
        <div v-if="filesMounted && routeInstanceId" v-show="activeTab === 'files'" class="app-files-wrap">
          <div class="app-files-bar">
            <component :is="menuStore.iconComponents['HOutline:FolderIcon']" class="app-files-bar__ico" />
            <span class="app-files-bar__path m-mono" :title="info?.absolutePath">{{ info?.absolutePath || info?.instancePath || '—' }}</span>
            <UButton
              v-if="canOpenFileManager && info?.absolutePath"
              color="neutral"
              variant="soft"
              size="sm"
              icon="i-lucide-external-link"
              class="app-files-bar__open"
              :to="pageTabTo('file', 'index', { workdir: info.absolutePath })"
            >
              {{ t('instance.ui.openInFileManager') }}
            </UButton>
          </div>
          <div class="app-files">
            <FileBrowser :key="routeInstanceId" :scope="{ kind: 'instance', id: routeInstanceId }" />
          </div>
        </div>

        <!-- 配置 -->
        <template v-if="activeTab === 'config'">
          <!-- 手机：资源占用放在配置页顶部 -->
          <div v-if="isMobile" class="app-stats is-mobile">
            <div v-for="s in statCards" :key="s.key" class="app-stat">
              <div class="app-stat__top">
                <span class="app-stat__t">{{ s.label }}</span>
                <span class="app-stat__sub m-num">{{ s.sub }}</span>
              </div>
              <span class="app-stat__v m-num">{{ s.value }}</span>
              <div class="app-stat__bar"><span :style="{ width: `${s.pct}%`, background: s.color }" /></div>
            </div>
          </div>

          <section v-if="!isMobile" class="app-card">
            <div class="app-card__head">
              <h2 class="app-card__title">{{ t('instance.ui.runConfig') }}</h2>
              <UButton color="primary" size="sm" icon="i-lucide-pencil" @click="openInstanceEditor">
                {{ t('instance.ui.editConfig') }}
              </UButton>
            </div>
            <dl class="app-kvrows">
              <div v-for="row in configRows" :key="row.key" :ref="row.key === firstEnvKey ? setEnvAnchor : undefined" class="app-kvrow">
                <dt :title="row.label">
                  <UIcon v-if="row.secret" name="i-lucide-lock" class="app-kvrow__lock" />{{ row.label }}
                </dt>
                <dd :class="{ 'm-mono': row.mono, 'is-dim': row.secret }">{{ row.value }}</dd>
              </div>
            </dl>
            <p class="app-note">{{ t('instance.ui.restartHint') }}</p>
          </section>
        </template>
      </div>

      <!-- 配置标签：PC 右侧应用信息 + 最近事件；手机为单列信息卡 -->
      <aside v-if="activeTab === 'config'" class="app-side">
        <section class="app-card">
          <h2 class="app-card__title">{{ t('instance.ui.appInfo') }}</h2>
          <dl class="app-kv">
            <div v-if="isMobile" class="app-kv__item">
              <dt>{{ t('instance.ui.appType') }}</dt>
              <dd>{{ info?.type || t('instance.ui.uncategorized') }}</dd>
            </div>
            <div class="app-kv__item">
              <dt>{{ t('instance.ui.workdir') }}</dt>
              <dd class="app-kv__line">
                <span class="m-mono app-kv__ellipsis" :title="info?.absolutePath">{{ info?.instancePath || '—' }}</span>
                <AppIconButton
                  v-if="info?.instancePath"
                  icon="HOutline:ClipboardDocumentIcon"
                  :size="16"
                  :box="isMobile ? 44 : 36"
                  :label="t('instance.ui.copyPath')"
                  @click="copy(info.absolutePath || info.instancePath)"
                />
              </dd>
            </div>
            <div class="app-kv__item">
              <dt>{{ t('instance.startCommand') }}</dt>
              <dd class="m-mono app-kv__code">{{ info?.startCommand || '—' }}</dd>
            </div>
            <div class="app-kv__item">
              <dt>{{ t('instance.stopCommand') }}</dt>
              <dd class="app-kv__line is-wrap">
                <span v-if="info?.stopCommand" class="m-mono app-kv__kbd">{{ info.stopCommand }}</span>
                <span class="app-kv__hint">
                  {{
                    info?.stopCommand
                      ? t('instance.ui.stopCmdSummary', { n: stopTimeout })
                      : t('instance.ui.stopNoCmdSummary', { n: stopTimeout })
                  }}
                </span>
              </dd>
            </div>
            <div class="app-kv__item">
              <dt>{{ t('instance.environmentVariables') }}</dt>
              <dd class="app-kv__line">
                <span class="m-num app-kv__count">{{ t('instance.ui.envCount', { count: envKeys.length }) }}</span>
                <span v-if="envKeys.length" class="m-mono app-kv__ellipsis app-kv__hint">{{ envKeys.join(' · ') }}</span>
                <span class="app-kv__spacer" />
                <UButton v-if="envKeys.length && !isMobile" color="neutral" variant="soft" size="xs" @click="scrollToEnv">
                  {{ t('instance.ui.view') }}
                </UButton>
              </dd>
            </div>
            <div class="app-kv__item">
              <dt>{{ t('instance.tags') }}</dt>
              <dd class="app-kv__tags">
                <span v-for="tag in tags" :key="tag" class="app-tag">#{{ tag }}</span>
                <span v-if="!tags.length" class="app-kv__hint">{{ t('instance.ui.none') }}</span>
              </dd>
            </div>
            <div class="app-kv__item is-switch">
              <span class="app-kv__switch-text">
                <dt>{{ t('instance.ui.bootStart') }}</dt>
                <dd>{{ t('instance.ui.autoStartDesc') }}</dd>
              </span>
              <AppSwitch
                :model-value="!!info?.autoStart"
                :disabled="!info || savingFlag"
                @update:model-value="(v) => toggleFlag('autoStart', v)"
              />
            </div>
            <div class="app-kv__item is-switch">
              <span class="app-kv__switch-text">
                <dt>{{ t('instance.ui.autoRestart') }}</dt>
                <dd>{{ t('instance.ui.autoRestartDesc') }}</dd>
              </span>
              <AppSwitch
                :model-value="!!info?.autoRestart"
                :disabled="!info || savingFlag"
                @update:model-value="(v) => toggleFlag('autoRestart', v)"
              />
            </div>
            <div class="app-kv__item">
              <dt>{{ t('instance.createTime') }}</dt>
              <dd class="m-num">
                {{ fmtTime(info?.createTime) }}<template v-if="info?.userName"> · {{ info.userName }}</template>
              </dd>
            </div>
            <div v-if="info?.updateTime" class="app-kv__item">
              <dt>{{ t('instance.ui.updatedAtLabel') }}</dt>
              <dd class="m-num">{{ fmtTime(info.updateTime) }}</dd>
            </div>
          </dl>
          <UButton v-if="isMobile" color="primary" icon="i-lucide-pencil" class="app-side__btn" @click="openInstanceEditor">
            {{ t('instance.ui.editConfig') }}
          </UButton>
        </section>

        <section class="app-card">
          <div class="app-card__head">
            <h2 class="app-card__title">{{ t('instance.ui.recentEvents') }}</h2>
            <AppIconButton
              icon="HOutline:ArrowPathIcon"
              :size="16"
              :box="isMobile ? 44 : 36"
              :label="t('common.refresh')"
              @click="loadEvents"
            />
          </div>
          <div v-if="eventsLoading && !events.length" class="app-events__empty">{{ t('common.loading') }}…</div>
          <div v-else-if="!events.length" class="app-events__empty">{{ t('instance.ui.noEvents') }}</div>
          <ol v-else class="app-events">
            <li v-for="(ev, idx) in eventRows" :key="idx" class="app-event">
              <span class="app-event__dot" :style="{ background: ev.color }" />
              <span class="app-event__text">
                <span class="app-event__t">{{ ev.text }}</span>
                <span class="app-event__at m-num">{{ ev.at }}</span>
              </span>
            </li>
          </ol>
          <RouterLink v-if="canSeeLogs" class="app-events__all" :to="{ path: '/system/logs', query: { tab: 'operation' } }">
            {{ t('instance.ui.allRecords') }}
            <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" />
          </RouterLink>
        </section>
      </aside>
    </div>

    <InstanceEditor
      :visible="instanceEditorVisible"
      :mode="instanceEditorMode"
      :loading="instanceEditorLoading"
      :submitting="instanceEditorSubmitting"
      :form="instanceEditorForm"
      :type-options="typeOptions"
      allow-run
      :running="instanceEditorMode === 'edit' && isRunning"
      @close="handleEditorClose"
      @submit="handleEditorSubmit"
    />

    <AppActionSheet v-model="sheetOpen" :title="displayName" :quick="sheetQuick" :items="sheetItems" @select="onMore" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import type { ComponentPublicInstance } from 'vue'
import dayjs from 'dayjs'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { pageTabTo } from '@/config/pageGroups'
import { useI18n } from 'vue-i18n'
import { useInstanceConsoleStore } from '@/stores/instance/console'
import { toEnvRows, useInstanceListStore } from '@/stores/instance/list'
import { useUserStore } from '@/stores/user'
import { InstanceStatus, type InstanceEditorFormValue } from '@/stores/instance/types'
import { getInstanceEventsRequest, getInstanceStatsRequest, updateInstanceRequest } from '@/api/instance'
import FileBrowser from '@/components/file/FileBrowser.vue'
import {
  InstanceEventKind,
  type GetInstanceStatsResponse,
  type InstanceEvent,
} from '@/types/v1/instance'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import { copyTextToClipboard, formatFileSize } from '@/utils/file'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import type { TerminalAction } from '@/components/terminal/TerminalShell.vue'
import TerminalConsole from '@/components/terminal/TerminalConsole.vue'
import InstanceEditor from '@/views/instance/list/instanceEditor.vue'
import AppActionSheet, { type AppSheetItem } from '@/views/instance/console/AppActionSheet.vue'
import { fmtDuration, initialOf, toneOf } from '@/views/instance/shared'

defineOptions({ name: 'InstanceConsoleView' })

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const menuStore = useMenuStore()
const userStore = useUserStore()
const accessible = useAccessibleRoutes()

const instanceConsoleStore = useInstanceConsoleStore()
const instanceListStore = useInstanceListStore()

const { terminalInfo, socketStatus, isBusy } = storeToRefs(instanceConsoleStore)

const {
  instanceEditorVisible,
  instanceEditorMode,
  instanceEditorLoading,
  instanceEditorSubmitting,
  instanceEditorForm,
  typeOptions,
} = storeToRefs(instanceListStore)

const {
  initialize,
  refreshCurrentInfo,
  togglePower,
  restartTerminal,
  forceStopConsole,
  forceRestartTerminal,
  clearScreen,
  reconnect,
  sendRaw,
  sendResize,
  subscribeOutput,
  getOutputBuffer,
} = instanceConsoleStore

const { getInstanceTypeList, openEditEditor, openCreateEditor, closeInstanceEditor, submitInstanceEditor, deleteInstances } =
  instanceListStore

const term = ref<InstanceType<typeof TerminalConsole>>()
const isMobile = computed(() => menuStore.isMobile)
const canOpenFileManager = computed(() => accessible.value.has('/file/index'))
const canSeeLogs = computed(() => accessible.value.has('/system/logs'))

const info = computed(() => terminalInfo.value)
const isRunning = computed(() => info.value?.status === InstanceStatus.INSTANCE_STATUS_RUNNING)
const isMaintenance = computed(() => !!info.value?.maintenance)
const displayName = computed(() => info.value?.name || t('instance.currentInstance'))
const instanceName = displayName

// —— 贴纸（与应用卡片同一套按 id 取色）——
const tone = computed(() => toneOf(getRouteInstanceId()))
const initial = computed(() => initialOf(info.value?.name || ''))

const statusPill = computed(() => {
  switch (info.value?.status) {
    case InstanceStatus.INSTANCE_STATUS_RUNNING:
      return { label: t('instance.running'), variant: 'success' as const }
    case InstanceStatus.INSTANCE_STATUS_STOPPED:
      return { label: t('instance.stopped'), variant: 'neutral' as const }
    case InstanceStatus.INSTANCE_STATUS_MAINTENANCE:
      return { label: t('instance.maintenance'), variant: 'warning' as const }
    default:
      return undefined
  }
})

// —— 运行时长 / 停止时间（30s 刷新一次）——
const now = ref(Date.now())
let clockTimer: ReturnType<typeof setInterval> | undefined
const fmtTime = (value: unknown) => {
  if (!value) return '—'
  const d = dayjs(value as string | Date)
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm') : '—'
}
// 今天的只显示时间，其他显示月日
const fmtWhen = (value: unknown) => {
  const d = dayjs(value as string | Date)
  if (!d.isValid()) return '—'
  return d.isSame(dayjs(), 'day') ? t('instance.ui.todayAt', { time: d.format('HH:mm') }) : d.format('MM-DD HH:mm')
}
const uptimeText = computed(() => {
  const i = info.value
  if (!i) return '—'
  if (isRunning.value) {
    const start = i.startTime ? dayjs(i.startTime as unknown as string) : null
    if (!start || !start.isValid()) return t('instance.running')
    const dur = fmtDuration(t, now.value - start.valueOf())
    return isMobile.value ? dur : t('instance.ui.uptime', { time: dur })
  }
  if (isMaintenance.value && i.maintenanceTime) {
    return `${t('instance.ui.upMaintenance')} ${fmtWhen(i.maintenanceTime)}`
  }
  if (i.stopTime) return `${t('instance.ui.upStoppedAt')} ${fmtWhen(i.stopTime)}`
  return t('instance.ui.neverRun')
})
const maintenanceText = computed(() =>
  info.value?.maintenanceTime
    ? t('instance.ui.maintenanceBanner', { time: fmtTime(info.value.maintenanceTime) })
    : t('instance.ui.maintenanceBannerNoTime'),
)

// —— 资源占用（后端 GetInstanceStats，前台轮询）——
const stats = ref<GetInstanceStatsResponse>()
const pid = computed(() => Number(stats.value?.pid || info.value?.pid || 0))
let statsTimer: ReturnType<typeof setTimeout> | undefined
let statsActive = false

const loadStats = async () => {
  const id = getRouteInstanceId()
  if (!id) return
  try {
    const { data } = await getInstanceStatsRequest({ id })
    stats.value = data
  } catch {
    // 资源占用是辅助信息：失败后停止轮询，避免反复报错
    stopStats()
  }
}
const scheduleStats = () => {
  if (statsTimer) clearTimeout(statsTimer)
  if (!statsActive) return
  statsTimer = setTimeout(async () => {
    if (typeof document === 'undefined' || document.visibilityState === 'visible') await loadStats()
    scheduleStats()
  }, isRunning.value ? 4000 : 20000)
}
const startStats = () => {
  statsActive = true
  void loadStats().then(scheduleStats)
}
const stopStats = () => {
  statsActive = false
  if (statsTimer) clearTimeout(statsTimer)
}

const pctOf = (v: number, total: number) => (total > 0 ? Math.min(100, Math.max(0, (v / total) * 100)) : 0)
const statCards = computed(() => {
  const s = stats.value
  const running = !!s?.running
  const cores = Number(s?.cpuCores || 0)
  const cpu = Number(s?.cpuPercent || 0)
  const mem = Number(s?.memoryBytes || 0)
  const memTotal = Number(s?.memoryTotal || 0)
  const disk = Number(s?.diskUsageBytes || 0)
  const diskTotal = Number(s?.diskTotal || 0)
  const diskFree = Number(s?.diskFree || 0)
  const procs = Number(s?.processCount || 0)
  return [
    {
      key: 'cpu',
      label: 'CPU',
      value: running ? `${cpu.toFixed(cpu >= 100 ? 0 : 1)}%` : '—',
      sub: cores ? t('instance.ui.cores', { n: cores }) : '',
      pct: running ? pctOf(cpu, cores > 0 ? cores * 100 : 100) : 0,
      color: 'var(--m-p)',
    },
    {
      key: 'mem',
      label: t('instance.ui.memory'),
      value: running ? formatFileSize(mem) : '—',
      sub: memTotal ? `/ ${formatFileSize(memTotal)}` : '',
      pct: running ? pctOf(mem, memTotal) : 0,
      color: 'var(--m-a2)',
    },
    {
      key: 'proc',
      label: t('instance.ui.processes'),
      value: running ? String(procs) : '0',
      sub: pid.value ? `PID ${pid.value}` : t('instance.ui.notRunning'),
      pct: running ? Math.min(100, procs * 10) : 0,
      color: 'var(--m-ok)',
    },
    {
      key: 'disk',
      label: t('instance.ui.diskUsage'),
      value: s?.diskScanTime ? formatFileSize(disk) : '…',
      sub: diskTotal ? t('instance.ui.diskFree', { free: formatFileSize(diskFree) }) : '',
      pct: pctOf(disk, diskTotal),
      color: 'var(--m-a)',
    },
  ]
})

// —— 信息 ——
const tags = computed(() =>
  (info.value?.tags || '')
    .split(/[,，]/u)
    .map((s) => s.trim().replace(/^#/u, ''))
    .filter(Boolean),
)
const envKeys = computed(() => (info.value?.env || []).map((e) => e.key).filter(Boolean))
const stopTimeout = computed(() => Number(info.value?.stopTimeout || 0) || 3)
const SECRET_DOTS = '••••••••'
const configRows = computed(() => {
  const i = info.value
  const rows: { key: string; label: string; value: string; mono?: boolean; secret?: boolean }[] = [
    { key: 'type', label: t('instance.ui.appType'), value: i?.type || t('instance.ui.uncategorized') },
    { key: 'dir', label: t('instance.ui.workdir'), value: i?.absolutePath || i?.instancePath || '—', mono: true },
    { key: 'start', label: t('instance.startCommand'), value: i?.startCommand || '—', mono: true },
    { key: 'stop', label: t('instance.stopCommand'), value: i?.stopCommand || '—', mono: true },
    { key: 'timeout', label: t('instance.ui.stopTimeout'), value: t('instance.ui.secondsValue', { n: stopTimeout.value }) },
    { key: 'autoStart', label: t('instance.ui.bootStart'), value: i?.autoStart ? t('common.yes') : t('common.no') },
    { key: 'autoRestart', label: t('instance.ui.autoRestart'), value: i?.autoRestart ? t('common.yes') : t('common.no') },
  ]
  ;(i?.env || []).forEach((env, idx) => {
    rows.push({
      key: `env-${idx}`,
      label: env.key,
      value: env.isSecret ? SECRET_DOTS : env.value || '""',
      mono: true,
      secret: env.isSecret,
    })
  })
  return rows
})
// 「环境变量 · 查看」：滚动到运行配置里的第一个变量
const firstEnvKey = 'env-0'
let envAnchor: HTMLElement | null = null
const setEnvAnchor = (el: Element | ComponentPublicInstance | null) => {
  envAnchor = el instanceof HTMLElement ? el : null
}
const scrollToEnv = () => envAnchor?.scrollIntoView({ behavior: 'smooth', block: 'center' })

// —— 最近事件 ——
const events = ref<InstanceEvent[]>([])
const eventsLoading = ref(false)
const loadEvents = async () => {
  const id = getRouteInstanceId()
  if (!id) return
  eventsLoading.value = true
  try {
    const { data } = await getInstanceEventsRequest({ id, limit: isMobile.value ? 5 : 6 })
    events.value = data?.events || []
  } catch {
    events.value = []
  } finally {
    eventsLoading.value = false
  }
}
const EVENT_LABEL: Partial<Record<InstanceEventKind, string>> = {
  [InstanceEventKind.INSTANCE_EVENT_KIND_CREATE]: 'evCreate',
  [InstanceEventKind.INSTANCE_EVENT_KIND_START]: 'evStart',
  [InstanceEventKind.INSTANCE_EVENT_KIND_STOP]: 'evStop',
  [InstanceEventKind.INSTANCE_EVENT_KIND_FORCE_STOP]: 'evForceStop',
  [InstanceEventKind.INSTANCE_EVENT_KIND_RESTART]: 'evRestart',
  [InstanceEventKind.INSTANCE_EVENT_KIND_FORCE_RESTART]: 'evForceRestart',
  [InstanceEventKind.INSTANCE_EVENT_KIND_UPDATE]: 'evUpdate',
  [InstanceEventKind.INSTANCE_EVENT_KIND_LOG_CLEAR]: 'evLogClear',
  [InstanceEventKind.INSTANCE_EVENT_KIND_FILE]: 'evFile',
  [InstanceEventKind.INSTANCE_EVENT_KIND_MAINTENANCE_ON]: 'evMaintenanceOn',
  [InstanceEventKind.INSTANCE_EVENT_KIND_MAINTENANCE_OFF]: 'evMaintenanceOff',
}
const eventColor = (ev: InstanceEvent) => {
  if (!ev.success && ev.kind !== InstanceEventKind.INSTANCE_EVENT_KIND_EXIT) return 'var(--m-bad)'
  switch (ev.kind) {
    case InstanceEventKind.INSTANCE_EVENT_KIND_START:
    case InstanceEventKind.INSTANCE_EVENT_KIND_RESTART:
      return 'var(--m-ok)'
    case InstanceEventKind.INSTANCE_EVENT_KIND_FORCE_STOP:
    case InstanceEventKind.INSTANCE_EVENT_KIND_FORCE_RESTART:
      return 'var(--m-bad)'
    case InstanceEventKind.INSTANCE_EVENT_KIND_EXIT:
      return ev.exitCode === 0 ? 'var(--el-text-color-secondary)' : 'var(--m-bad)'
    case InstanceEventKind.INSTANCE_EVENT_KIND_AUTO_RESTART:
    case InstanceEventKind.INSTANCE_EVENT_KIND_MAINTENANCE_ON:
    case InstanceEventKind.INSTANCE_EVENT_KIND_MAINTENANCE_OFF:
      return 'var(--m-warn)'
    case InstanceEventKind.INSTANCE_EVENT_KIND_CREATE:
      return 'var(--m-p)'
    case InstanceEventKind.INSTANCE_EVENT_KIND_UPDATE:
    case InstanceEventKind.INSTANCE_EVENT_KIND_FILE:
    case InstanceEventKind.INSTANCE_EVENT_KIND_LOG_CLEAR:
      return 'var(--m-a2)'
    default:
      return 'var(--el-text-color-secondary)'
  }
}
const eventText = (ev: InstanceEvent) => {
  let text: string
  if (ev.kind === InstanceEventKind.INSTANCE_EVENT_KIND_EXIT) {
    text = t('instance.ui.evExit', { code: ev.exitCode })
  } else if (ev.kind === InstanceEventKind.INSTANCE_EVENT_KIND_AUTO_RESTART) {
    text = t('instance.ui.evAutoRestart', { code: ev.exitCode })
  } else {
    const action = t(`instance.ui.${EVENT_LABEL[ev.kind] || 'evUpdate'}`)
    text = ev.userName ? t('instance.ui.evBy', { user: ev.userName, action }) : action
  }
  if (!ev.success && ev.kind !== InstanceEventKind.INSTANCE_EVENT_KIND_EXIT) {
    text += ` · ${ev.error ? t('instance.ui.evFailedWith', { error: ev.error }) : t('instance.ui.evFailed')}`
  } else if (ev.error) {
    text += ` · ${ev.error}`
  }
  return text
}
const eventRows = computed(() =>
  events.value.map((ev) => ({ text: eventText(ev), at: fmtWhen(ev.time), color: eventColor(ev) })),
)

// —— 标签 ——
const ROUTE_NAME = 'InstanceAppView'
type TabKey = 'console' | 'files' | 'config'
const TAB_KEYS: TabKey[] = ['console', 'files', 'config']
const activeTab = ref<TabKey>('console')
const filesMounted = ref(false)
const tabs = computed<{ key: TabKey; label: string; icon: string }[]>(() => [
  { key: 'console', label: t('instance.console'), icon: 'HOutline:CommandLineIcon' },
  { key: 'files', label: t('instance.ui.tabFiles'), icon: 'HOutline:FolderIcon' },
  { key: 'config', label: t('instance.config'), icon: 'HOutline:AdjustmentsHorizontalIcon' },
])
// 标签跟随地址 ?tab=（只在地址属于本页时；页面被缓存、地址已是别的页面时保持不动）
watch(
  () => [route.name, route.query.tab] as const,
  ([name, tab]) => {
    if (name !== ROUTE_NAME) return
    activeTab.value = TAB_KEYS.includes(tab as TabKey) ? (tab as TabKey) : 'console'
  },
  { immediate: true },
)
watch(
  activeTab,
  (key) => {
    if (key === 'files') filesMounted.value = true
    if (key === 'console') nextTick(() => term.value?.focus())
    if (key === 'config') void loadEvents()
  },
  { immediate: true },
)
const onTab = (key: TabKey) => {
  if (key === activeTab.value) return
  void router.replace({ query: { ...route.query, tab: key === 'console' ? undefined : key } })
}

// —— 命令输入条（回车发送 + 历史）——
const command = ref('')
const commandHistory = ref<string[]>([])
let historyIndex = -1 // -1 表示正在编辑新命令
const HISTORY_MAX = 50
const runCommand = (line: string) => {
  if (!line.trim() || socketStatus.value !== 'connected') return
  sendRaw(`${line}\r`)
  if (commandHistory.value[commandHistory.value.length - 1] !== line) {
    commandHistory.value = [...commandHistory.value.filter((c) => c !== line), line].slice(-HISTORY_MAX)
  }
  historyIndex = -1
  if (autoScroll.value) term.value?.scrollToBottom()
}
const submitCommand = () => {
  const line = command.value
  if (!line.trim() || socketStatus.value !== 'connected') return
  runCommand(line)
  command.value = ''
}
const historyStep = (dir: -1 | 1) => {
  const list = commandHistory.value
  if (!list.length) return
  if (historyIndex === -1) {
    if (dir === 1) return
    historyIndex = list.length - 1
  } else {
    historyIndex += dir
    if (historyIndex >= list.length) {
      historyIndex = -1
      command.value = ''
      return
    }
    historyIndex = Math.max(0, historyIndex)
  }
  command.value = list[historyIndex] || ''
}

// —— 连接状态文案 ——
const statusLabel = computed(() => {
  switch (socketStatus.value) {
    case 'connected':
      return isMobile.value ? t('instance.connected') : t('instance.ui.connectedWs')
    case 'connecting':
      return t('instance.connecting')
    case 'error':
      return t('instance.connectFailed')
    default:
      return t('instance.disconnected')
  }
})

// 控制台顶栏按钮（与原型一致）：清屏 / 重新连接 / 自动滚动
const autoScroll = ref(true)
const terminalActions = computed<TerminalAction[]>(() => [
  { key: 'clear', icon: 'HOutline:BackspaceIcon', label: t('instance.clearScreen') },
  { key: 'reconnect', icon: 'HOutline:ArrowPathIcon', label: t('instanceConsole.reconnect') },
  { key: 'autoScroll', icon: 'HOutline:ChevronDoubleDownIcon', label: t('instanceConsole.autoScroll'), toggle: autoScroll.value },
])
// 顶栏信息：已缓冲的输出行数（输出到达后节流统计）
const bufferedLines = ref(0)
let lineTimer: ReturnType<typeof setTimeout> | undefined
const countLines = () => {
  const raw = getOutputBuffer()
  let n = 0
  for (let i = 0; i < raw.length; i++) if (raw.charCodeAt(i) === 10) n++
  bufferedLines.value = n
}
const scheduleCountLines = () => {
  if (lineTimer) return
  lineTimer = setTimeout(() => {
    lineTimer = undefined
    countLines()
  }, 500)
}
const consoleInfo = computed(() => t('instance.ui.bufferedLines', { n: bufferedLines.value.toLocaleString() }))
// 快捷命令：应用类型里设置的；没有时用最近发过的命令（最多 6 条，新的在前）
const typeQuickCommands = computed(() => info.value?.quickCommands || [])
const quickCommands = computed(() =>
  typeQuickCommands.value.length ? typeQuickCommands.value : [...commandHistory.value].reverse().slice(0, 6),
)
// 手机：输入行的键盘按钮展开常用按键条并弹出系统键盘
const keysOpen = ref(false)
const toggleKeys = () => {
  keysOpen.value = !keysOpen.value
  if (keysOpen.value) nextTick(() => term.value?.focus())
  else term.value?.blur()
}

const nodeStore = useNodeStore()
const canDelete = computed(() => {
  const owner = info.value?.userId?.trim() || ''
  const me = nodeStore.isRemote ? nodeStore.remoteUserId : userStore.userInfo?.userId?.trim() || ''
  return !!owner && owner === me
})

// —— 更多操作：PC 下拉（带说明）/ 手机底部面板 ——
interface MenuEntry extends Partial<AppSheetItem> {
  key: string
  divider?: boolean
}
const maintenanceEntry = computed<AppSheetItem>(() =>
  isMaintenance.value
    ? { key: 'maintenanceOff', label: t('instance.ui.endMaintenance'), icon: 'i-lucide-badge-check' }
    : {
        key: 'maintenanceOn',
        label: t('instance.ui.enterMaintenance'),
        icon: 'i-lucide-wrench',
        description: t('instance.ui.maintenanceDesc'),
      },
)
const forceStopEntry = computed<AppSheetItem>(() => ({
  key: 'forceStop',
  label: t('instance.forceStop'),
  icon: 'i-lucide-zap',
  description: t('instance.ui.forceStopDesc'),
  danger: true,
  disabled: !isRunning.value,
}))
const menuItems = computed<MenuEntry[]>(() => [
  forceStopEntry.value,
  { key: 'sep-1', divider: true },
  { key: 'edit', label: t('instance.ui.editApp'), icon: 'i-lucide-pencil' },
  { key: 'duplicate', label: t('instance.ui.duplicateApp'), icon: 'i-lucide-copy' },
  { key: 'download', label: t('instance.ui.downloadLog'), icon: 'i-lucide-file-down' },
  {
    key: 'forceRestart',
    label: t('instance.forceRestart'),
    icon: 'i-lucide-rotate-ccw',
    description: t('instance.ui.forceRestartDesc'),
    disabled: isMaintenance.value,
  },
  maintenanceEntry.value,
  ...(canDelete.value
    ? [
        { key: 'sep-2', divider: true },
        { key: 'delete', label: t('instance.ui.deleteApp'), icon: 'i-lucide-trash-2', danger: true },
      ]
    : []),
])
const sheetOpen = ref(false)
const sheetQuick = computed<AppSheetItem[]>(() => [
  { key: 'restart', label: t('instance.restart'), icon: 'i-lucide-rotate-cw', disabled: isMaintenance.value || isBusy.value },
  { key: 'edit', label: t('instance.ui.editShort'), icon: 'i-lucide-pencil' },
  { key: 'download', label: t('instance.ui.downloadLog'), icon: 'i-lucide-file-down' },
])
const sheetItems = computed<AppSheetItem[]>(() => [
  { key: 'duplicate', label: t('instance.ui.duplicateApp'), icon: 'i-lucide-copy' },
  maintenanceEntry.value,
  {
    key: 'forceRestart',
    label: t('instance.forceRestart'),
    icon: 'i-lucide-rotate-ccw',
    disabled: isMaintenance.value,
  },
  forceStopEntry.value,
  ...(canDelete.value ? [{ key: 'delete', label: t('instance.ui.deleteApp'), icon: 'i-lucide-trash-2', danger: true }] : []),
])

const onMore = (key: string) => {
  switch (key) {
    case 'forceStop':
      handleForceStop()
      break
    case 'forceRestart':
      handleForceRestart()
      break
    case 'restart':
      handleRestart()
      break
    case 'edit':
      void openInstanceEditor()
      break
    case 'duplicate':
      void duplicateApp()
      break
    case 'download':
      downloadLog()
      break
    case 'maintenanceOn':
      handleEnterMaintenance()
      break
    case 'maintenanceOff':
      void handleEndMaintenance()
      break
    case 'delete':
      handleDelete()
      break
  }
}

const onAction = (key: string) => {
  switch (key) {
    case 'clear':
      void handleClear()
      break
    case 'reconnect':
      void reconnect()
      break
    case 'autoScroll':
      autoScroll.value = !autoScroll.value
      if (autoScroll.value) term.value?.scrollToBottom()
      break
    default:
      feedback.info(t('instance.featureNotImplemented'))
  }
}

// —— 输出流订阅 ——
let unsubscribe: (() => void) | null = null

onMounted(() => {
  const buffered = getOutputBuffer()
  if (buffered) term.value?.write(buffered)
  countLines()
  unsubscribe = subscribeOutput((chunk) => {
    term.value?.write(chunk)
    if (autoScroll.value) term.value?.scrollToBottom()
    scheduleCountLines()
  })
  term.value?.focus()
  clockTimer = setInterval(() => (now.value = Date.now()), 30000)
  startStats()
})
onActivated(() => {
  if (!statsActive) startStats()
})
onDeactivated(() => stopStats())

// 连接成功：后端会重放最近日志，先 reset 画面避免重复；同步一次当前窗口尺寸并聚焦
watch(socketStatus, (status, prev) => {
  if (status === prev) return
  if (status === 'connected') {
    const terminal = term.value
    if (!terminal) return
    terminal.reset()
    if (terminal.cols > 0 && terminal.rows > 0) sendResize(terminal.cols, terminal.rows)
    terminal.focus()
    scheduleCountLines()
  }
})

// 启停 / 维护状态变化后立即刷新资源占用与事件
watch(
  () => [info.value?.status, info.value?.maintenance, info.value?.updateTime] as const,
  (next, prev) => {
    if (!prev || next.join() === prev.join()) return
    if (statsActive) void loadStats().then(scheduleStats)
    if (activeTab.value === 'config') void loadEvents()
  },
)

// —— 路由驱动会话初始化 ——
// 本页实例 id：只取属于本页路由时的参数；页面被缓存后地址变成别的页面时保持上一次的值，
// 避免用空 id 去初始化会话 / 拉资源占用（后端会报「您没有该实例权限」）
const routeInstanceId = ref('')
function getRouteInstanceId() {
  return routeInstanceId.value
}

watch(
  () => [route.name, route.path, route.params.instanceId] as const,
  ([name, path, rawInstanceId]) => {
    if (name !== ROUTE_NAME || typeof rawInstanceId !== 'string' || !rawInstanceId) return
    const changed = rawInstanceId !== routeInstanceId.value
    routeInstanceId.value = rawInstanceId
    // 会话按页面路径区分（不含 ?tab=），切换标签不会重建会话
    void initialize(path, rawInstanceId)
    if (changed) {
      stats.value = undefined
      events.value = []
      if (statsActive) void loadStats()
      if (activeTab.value === 'config') void loadEvents()
    }
  },
  { immediate: true },
)

// —— 清屏：清后端日志 + 会话缓冲 + xterm 画面 ——
const handleClear = async () => {
  await clearScreen()
  term.value?.reset()
  bufferedLines.value = 0
}

const copy = async (text: string) => {
  try {
    await copyTextToClipboard(text)
    feedback.success(t('instance.ui.copied'))
  } catch {
    feedback.error(t('instance.ui.copyFailed'))
  }
}

// 下载日志：本页收到的控制台输出（去掉 ANSI 控制序列；密钥变量已由后端打码）
const ANSI_RE = /\u001B(?:\[[0-?]*[ -/]*[@-~]|\][^\u0007\u001B]*(?:\u0007|\u001B\\)|[@-Z\\-_])/gu
const downloadLog = () => {
  const raw = getOutputBuffer()
  if (!raw) {
    feedback.info(t('instance.ui.logEmpty'))
    return
  }
  const text = raw.replace(ANSI_RE, '').replace(/\r\n?/gu, '\n')
  const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${info.value?.name || 'app'}-${dayjs().format('YYYYMMDD-HHmmss')}.log`
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

// —— 编辑 / 复制 ——
const openInstanceEditor = async () => {
  const instanceId = getRouteInstanceId()
  if (!instanceId) {
    feedback.warning(t('instance.missingInstanceIdSetting'))
    return
  }
  try {
    await getInstanceTypeList()
    await openEditEditor(instanceId)
  } catch (error) {
    showRequestError(error, t('instance.loadConfigFailed'))
  }
}

// 复制为新应用：以当前配置打开新建抽屉（密钥变量的值不会下发，需要重新填写）
const duplicateApp = async () => {
  const i = info.value
  if (!i) return
  try {
    await getInstanceTypeList()
    openCreateEditor()
    const typeId = i.typeId || typeOptions.value.find((o) => o.label === i.type)?.value || instanceEditorForm.value.type
    instanceEditorForm.value = {
      ...instanceEditorForm.value,
      name: t('instance.ui.copyName', { name: i.name }),
      remark: i.remark || '',
      tags: i.tags || '',
      type: typeId,
      startCommand: i.startCommand || '',
      stopCommand: i.stopCommand || '',
      instancePath: i.instancePath || '',
      autoStart: false,
      autoRestart: !!i.autoRestart,
      stopTimeout: Number(i.stopTimeout || 0),
      env: toEnvRows(i.env).map((row) => ({ ...row, saved: false })),
    }
  } catch (error) {
    showRequestError(error, t('instance.loadConfigFailed'))
  }
}

const handleEditorClose = () => closeInstanceEditor()

const handleEditorSubmit = async (form: InstanceEditorFormValue, options?: { run: boolean }) => {
  const mode = instanceEditorMode.value
  try {
    instanceEditorForm.value = { ...form }
    const saved = await submitInstanceEditor()
    feedback.success(mode === 'create' ? t('instance.createSuccess') : t('instance.configSaveSuccess'))
    if (mode === 'edit') {
      await refreshCurrentInfo()
      if (options?.run) {
        if (isRunning.value) await restartTerminal()
        else await startWithMaintenanceCheck()
      }
      return
    }
    // 复制出来的新应用：保存后跳到它的详情
    if (saved?.id) {
      if (options?.run) await instanceListStore.changeInstanceStatus(saved.id, InstanceStatus.INSTANCE_STATUS_RUNNING)
      await router.push({ path: `/instance/app/${saved.id}`, query: { tabTitle: saved.name } })
    }
  } catch (error) {
    showRequestError(error, mode === 'create' ? t('instance.createFailed') : t('instance.configSaveFailed'))
  }
}

// 应用信息里的开关：开机自启 / 崩溃后自动重启，直接保存
const savingFlag = ref(false)
const toggleFlag = async (field: 'autoStart' | 'autoRestart', value: boolean) => {
  const i = info.value
  if (!i) return
  savingFlag.value = true
  try {
    const patch = field === 'autoStart' ? { autoStart: value } : { autoRestart: value }
    await updateInstanceRequest({ id: i.id, env: undefined, ...patch })
    await refreshCurrentInfo()
    feedback.success(t('instance.configSaveSuccess'))
  } catch (error) {
    showRequestError(error, t('instance.configSaveFailed'))
  } finally {
    savingFlag.value = false
  }
}

// —— 维护 ——
const setMaintenance = async (on: boolean) => {
  const i = info.value
  if (!i) return false
  try {
    await updateInstanceRequest({ id: i.id, env: undefined, maintenance: on })
    await refreshCurrentInfo()
    return true
  } catch (error) {
    showRequestError(error, t('instance.configSaveFailed'))
    return false
  }
}
const handleEnterMaintenance = () => {
  Dialog.confirm({
    title: t('instance.ui.enterMaintenance'),
    content: t('instance.ui.maintenanceOnContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.ui.enterMaintenance'),
    onConfirm: async () => {
      if (await setMaintenance(true)) feedback.success(t('instance.ui.maintenanceOnDone'))
    },
  })
}
const handleEndMaintenance = async () => {
  if (await setMaintenance(false)) feedback.success(t('instance.ui.maintenanceOffDone'))
}
// 启动维护中的应用：确认后先结束维护
const startWithMaintenanceCheck = async () => {
  if (!isMaintenance.value) {
    await togglePower()
    return
  }
  Dialog.confirm({
    title: t('instance.ui.endMaintenanceStartTitle'),
    content: t('instance.ui.endMaintenanceStartContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.ui.endAndStart'),
    onConfirm: async () => {
      if (await setMaintenance(false)) await togglePower()
    },
  })
}

// —— 启停 / 重启 / 强制（保留二次确认）——
const handleTogglePower = () => {
  if (!isRunning.value) {
    void startWithMaintenanceCheck()
    return
  }
  Dialog.confirm({
    title: t('instance.confirmStopInstanceTitle'),
    content: t('instance.confirmStopInstanceContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmStop'),
    onConfirm: async () => {
      await togglePower()
    },
  })
}

const handleRestart = () => {
  Dialog.confirm({
    title: t('instance.confirmRestartInstanceTitle'),
    content: t('instance.confirmRestartInstanceContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmRestart'),
    onConfirm: async () => {
      await restartTerminal()
    },
  })
}

const handleForceStop = () => {
  Dialog.confirm({
    title: t('instance.confirmForceStopInstanceTitle'),
    content: `${t('instance.confirmForceStopInstanceContent', { name: instanceName.value })} ${t('instance.ui.forceStopDesc')}`,
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmForceStop'),
    onConfirm: async () => {
      await forceStopConsole()
    },
  })
}

const handleForceRestart = () => {
  Dialog.confirm({
    title: t('instance.confirmForceRestartInstanceTitle'),
    content: t('instance.confirmForceRestartInstanceContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmForceRestart'),
    onConfirm: async () => {
      await forceRestartTerminal()
    },
  })
}

const handleDelete = () => {
  const id = getRouteInstanceId()
  if (!id) return
  Dialog.confirm({
    title: t('instance.confirmDeleteInstanceTitle'),
    content: t('instance.confirmDeleteInstanceContent', { name: instanceName.value }),
    cancelText: t('common.cancel'),
    confirmText: t('common.delete'),
    onConfirm: async () => {
      const { successCount } = await deleteInstances([id])
      if (successCount > 0) {
        feedback.success(t('instance.instanceDeleted', { name: instanceName.value, failed: '' }))
        await router.push('/instance/list')
      } else {
        feedback.error(t('instance.ui.deleteAppFailed'))
      }
    },
  })
}

onBeforeUnmount(() => {
  unsubscribe?.()
  closeInstanceEditor()
  stopStats()
  if (clockTimer) clearInterval(clockTimer)
  if (lineTimer) clearTimeout(lineTimer)
})
</script>

<style scoped lang="scss">
/* 路由为全出血（内容区无内边距、不滚动），本页自带内边距与滚动 */
.app-detail {
  height: 100%;
  min-height: 0;
  overflow-y: auto;
  padding: 20px 32px 32px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 1600px;
  width: 100%;
  margin: 0 auto;
  scrollbar-width: thin;
}
.app-detail__back {
  align-self: flex-start;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  text-decoration: none;
}
.app-detail__back:hover {
  color: var(--m-pInk);
}
.app-detail__back :deep(svg) {
  width: 16px;
  height: 16px;
}

/* 头部卡 */
.app-head {
  display: flex;
  flex-direction: column;
  gap: 20px;
  margin-top: -6px;
  padding: 22px 24px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.app-head__row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 18px;
}
.app-head__av {
  width: 72px;
  height: 72px;
  border-radius: 22px;
  font-size: 2rem;
  transform: rotate(-5deg);
  box-shadow: 4px 4px 0 var(--m-a);
  flex: none;
}
.app-head__main {
  flex: 1 1 320px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.app-head__title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  min-width: 0;
}
.app-head__name {
  margin: 0;
  min-width: 0;
  font-family: var(--app-font-display);
  font-size: 2rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  line-height: 1.15;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.app-head__meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 16px;
  min-width: 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.app-head__type,
.app-head__meta-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.app-head__type {
  font-weight: 700;
  color: var(--m-pInk);
  text-decoration: none;
}
.app-head__type:hover {
  color: var(--el-text-color-primary);
}
.app-head__type :deep(svg),
.app-head__meta-item :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
}
.app-head__remark {
  max-width: 420px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
}
.app-head__acts {
  display: flex;
  align-items: center;
  gap: 8px;
}
.app-head__btn {
  min-height: 44px;
  padding: 0 18px;
  border-radius: 999px;
  font-weight: 700;
}
.app-head__more {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.app-head__more:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.app-head__more :deep(.iconify) {
  width: 20px;
  height: 20px;
}
.app-head__acts :deep(.is-stop) {
  color: var(--m-bad);
}
.app-head__acts :deep(.is-start) {
  color: var(--m-ok);
}

/* PC ⋯ 下拉 */
.app-menu {
  display: flex;
  flex-direction: column;
  padding: 6px;
}
.app-menu__sep {
  height: 1px;
  margin: 4px 8px;
  background: var(--el-border-color-lighter);
}
.app-menu__item {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 44px;
  padding: 6px 12px;
  border: none;
  border-radius: 12px;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.app-menu__item:hover:not(:disabled) {
  background: var(--el-fill-color-light);
}
.app-menu__item:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.app-menu__ico {
  width: 18px;
  height: 18px;
  flex: none;
  color: var(--el-text-color-secondary);
}
.app-menu__item.is-danger,
.app-menu__item.is-danger .app-menu__ico {
  color: var(--m-bad);
}
.app-menu__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.app-menu__desc {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}

/* 资源占用 */
.app-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.app-stat {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  padding: 14px 16px;
  border-radius: 18px;
  background: var(--el-fill-color-light);
}
.app-stat__top {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}
.app-stat__t {
  flex: 1;
  min-width: 0;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.app-stat__v {
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}
.app-stat__sub {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.app-stat__bar {
  height: 6px;
  border-radius: 999px;
  background: var(--el-border-color-lighter);
  overflow: hidden;
}
.app-stat__bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  transition: width 0.4s ease;
}

/* 维护中提示 */
.app-maint {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 10px 10px 16px;
  border-radius: 16px;
  background: var(--m-warnSoft);
  color: var(--m-warn);
  font-size: 0.875rem;
  font-weight: 600;
}
.app-maint__ico {
  width: 18px;
  height: 18px;
  flex: none;
}
.app-maint__text {
  flex: 1;
  min-width: 0;
}
.app-maint__btn {
  flex: none;
  min-height: 36px;
  border-radius: 999px;
}

/* 标签 */
.app-tabs {
  display: flex;
  gap: 28px;
  margin-top: -4px;
  border-bottom: 1px solid var(--el-border-color-light);
  overflow-x: auto;
  scrollbar-width: none;
  flex: none;
}
.app-tabs::-webkit-scrollbar {
  display: none;
}
.app-tab {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 2px;
  border: none;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.9375rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.app-tab:hover {
  color: var(--el-text-color-primary);
}
.app-tab.is-on {
  color: var(--el-text-color-primary);
}
.app-tab.is-on::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -1px;
  height: 3px;
  border-radius: 3px 3px 0 0;
  background: var(--m-p);
}
.app-tab__ico {
  width: 16px;
  height: 16px;
}

/* 主体两栏 */
.app-body {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  align-items: flex-start;
}
.app-main {
  flex: 3 1 640px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
/* 控制台 / 文件填满页头下方剩余高度（命令输入始终在首屏内），矮屏时保底高度后整页滚动 */
.app-body.is-fill {
  flex: 1 1 0;
  align-items: stretch;
}
.app-term {
  flex: 1 1 0;
  min-height: 420px;
  display: flex;
  flex-direction: column;
  border-radius: var(--app-radius-lg);
  overflow: hidden;
}
.app-term > :deep(*) {
  flex: 1;
  min-height: 0;
}
/* 控制台卡片底部：快捷命令 + 命令输入（深色卡片内，终端固定配色） */
.app-quick {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px 12px;
  overflow-x: auto;
  scrollbar-width: none;
  flex-shrink: 0;
}
.app-quick::-webkit-scrollbar {
  display: none;
}
.app-quick__label {
  flex: none;
  margin-right: 4px;
  font-size: 0.75rem;
  font-weight: 700;
  color: #9c93a8;
}
.app-quick__chip {
  flex: none;
  display: inline-flex;
  align-items: center;
  min-height: 32px;
  max-width: 260px;
  padding: 0 12px;
  border: 1px solid rgba(255, 255, 255, 0.16);
  border-radius: 999px;
  background: transparent;
  color: #d6cfe0;
  font-size: 0.75rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s;
}
.app-quick__chip:hover:not(:disabled) {
  border-color: var(--m-p);
  color: #ffffff;
}
.app-quick__chip:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.app-cmd {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  flex-shrink: 0;
}
.app-cmd__field {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 46px;
  padding: 0 16px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  border: 1.5px solid rgba(255, 255, 255, 0.14);
  transition: border-color 0.15s;
}
.app-cmd__field:focus-within {
  border-color: var(--m-p);
}
.app-cmd__prompt {
  color: var(--m-a);
  font-weight: 800;
}
.app-cmd__field input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  color: #e8e3ee;
  font-size: 0.875rem;
}
.app-cmd__field input::placeholder {
  color: #9c93a8;
}
.app-cmd__field input:disabled {
  cursor: not-allowed;
}
.app-cmd__send {
  min-height: 46px;
  padding-inline: 18px;
  border-radius: 999px;
}
.app-cmd__send.is-icon {
  width: 44px;
  min-height: 44px;
  padding: 0;
  justify-content: center;
  flex: none;
}
.app-cmd__kb {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: #d6cfe0;
  cursor: pointer;
}
.app-cmd__kb.is-on {
  background: rgba(255, 255, 255, 0.1);
  color: #ffffff;
}
.app-cmd__kb :deep(.iconify) {
  width: 20px;
  height: 20px;
}

/* 文件 */
.app-files-wrap {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
}
.app-files-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 6px 6px 6px 14px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.app-files-bar__ico {
  width: 16px;
  height: 16px;
  flex: none;
}
.app-files-bar__path {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.app-files-bar__open {
  flex: none;
  min-height: 36px;
  border-radius: 999px;
}
.app-main.is-fill > .app-files-wrap {
  flex: 1 1 0;
}
.app-files {
  flex: 1 1 0;
  min-height: 380px;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  overflow: hidden;
}
.app-files > :deep(*) {
  flex: 1;
  min-height: 0;
}
.app-side {
  flex: 1 1 300px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 卡片 */
.app-card {
  display: flex;
  flex-direction: column;
  padding: 8px 20px 14px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.app-card__head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 0 6px;
}
.app-card__head .app-card__title {
  flex: 1;
  margin: 0;
}
.app-card__title {
  margin: 14px 0 6px;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}

/* 信息 kv */
.app-kv {
  margin: 0;
}
.app-kv__item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 0;
  border-top: 1px solid var(--el-border-color-lighter);
  min-width: 0;
}
.app-kv__item:first-child {
  border-top: none;
}
.app-kv__item dt {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.app-kv__item dd {
  margin: 0;
  min-width: 0;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.app-kv__line {
  display: flex;
  align-items: center;
  gap: 8px;
}
.app-kv__line.is-wrap {
  flex-wrap: wrap;
}
.app-kv__ellipsis {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.8125rem;
}
.app-kv__count {
  flex: none;
}
.app-kv__spacer {
  flex: 1;
}
.app-kv__code {
  padding: 9px 12px;
  border-radius: 12px;
  background: var(--el-fill-color-light);
  font-size: 0.8125rem !important;
  font-weight: 500 !important;
  word-break: break-all;
}
.app-kv__kbd {
  flex: none;
  padding: 4px 10px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  font-size: 0.8125rem;
}
.app-kv__hint {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.app-kv__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.app-tag {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 600;
}
.app-kv__item.is-switch {
  flex-direction: row;
  align-items: center;
  gap: 12px;
}
.app-kv__switch-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.app-kv__switch-text dd {
  font-weight: 500;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
}

/* 运行配置行 */
.app-kvrows {
  margin: 0;
}
.app-kvrow {
  display: flex;
  align-items: center;
  gap: 16px;
  min-height: 48px;
  padding: 10px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.app-kvrow dt {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  width: 160px;
  flex: none;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
}
.app-kvrow__lock {
  width: 14px;
  height: 14px;
  flex: none;
  color: var(--m-warn);
}
.app-kvrow dd {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
  word-break: break-all;
}
.app-kvrow dd.m-mono {
  font-size: 0.8125rem;
}
.app-kvrow dd.is-dim {
  color: var(--el-text-color-secondary);
  letter-spacing: 0.1em;
}
.app-note {
  margin: 8px 0 4px;
  padding: 12px 14px;
  border-radius: 14px;
  background: var(--m-warnSoft);
  color: var(--m-warn);
  font-size: 0.8125rem;
  font-weight: 600;
}
.app-side__btn {
  min-height: 48px;
  margin-top: 8px;
  justify-content: center;
  border-radius: 999px;
}

/* 最近事件 */
.app-events {
  margin: 0;
  padding: 0;
  list-style: none;
}
.app-event {
  display: flex;
  gap: 12px;
  padding: 10px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.app-event__dot {
  width: 10px;
  height: 10px;
  margin-top: 5px;
  border-radius: 50%;
  flex: none;
}
.app-event__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.app-event__t {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  word-break: break-word;
}
.app-event__at {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.app-events__empty {
  padding: 12px 0 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.app-events__all {
  align-self: flex-start;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 40px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--m-pInk);
  text-decoration: none;
}
.app-events__all :deep(svg) {
  width: 16px;
  height: 16px;
}

@media (width < 1200px) {
  .app-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

/* ========== 手机 ========== */
.app-detail.is-mobile {
  padding: 0;
  gap: 0;
  overflow: hidden;
}
.app-detail.is-mobile .app-head {
  margin: 0;
  padding: 10px 6px 6px 4px;
  border: none;
  border-radius: 0;
  gap: 0;
}
.app-detail.is-mobile .app-head__row {
  flex-wrap: nowrap;
  gap: 4px;
}
.app-head__back {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  flex: none;
  border-radius: 14px;
  color: var(--el-text-color-secondary);
}
.app-head__back :deep(svg) {
  width: 20px;
  height: 20px;
}
.app-detail.is-mobile .app-head__av {
  width: 40px;
  height: 40px;
  border-radius: 13px;
  font-size: 1.125rem;
  box-shadow: 3px 3px 0 var(--m-a);
}
.app-detail.is-mobile .app-head__main {
  flex: 1 1 0;
  gap: 3px;
  padding-left: 8px;
}
.app-detail.is-mobile .app-head__name {
  font-size: 1.0625rem;
}
.app-detail.is-mobile .app-head__meta {
  flex-wrap: nowrap;
  gap: 6px;
  font-size: 0.75rem;
  white-space: nowrap;
  overflow: hidden;
}
.app-detail.is-mobile .app-head__acts {
  gap: 0;
}
.app-detail.is-mobile .app-maint {
  margin: 0 12px 8px;
  padding: 8px 8px 8px 12px;
  font-size: 0.8125rem;
}
.app-detail.is-mobile .app-maint__btn {
  min-height: 40px;
}
.app-detail.is-mobile .app-tabs {
  margin: 0;
  padding: 0 20px;
  gap: 24px;
  background: var(--el-bg-color);
}
.app-detail.is-mobile .app-body {
  flex: 1;
  min-height: 0;
  flex-direction: column;
  flex-wrap: nowrap;
  gap: 12px;
  overflow-y: auto;
  scrollbar-width: none;
}
.app-detail.is-mobile .app-body::-webkit-scrollbar {
  display: none;
}
.app-detail.is-mobile .app-body:not(.is-fill) {
  padding: 12px 16px 24px;
}
.app-detail.is-mobile .app-main {
  flex: none;
  width: 100%;
}
.app-detail.is-mobile .app-main.is-fill {
  flex: 1;
  min-height: 0;
}
.app-detail.is-mobile .app-term {
  height: 100%;
  min-height: 320px;
  border-radius: 0;
}
.app-detail.is-mobile .app-files-wrap {
  height: 100%;
  gap: 0;
}
.app-detail.is-mobile .app-files-bar {
  margin: 10px 12px;
  padding-left: 12px;
}
.app-detail.is-mobile .app-files-bar__open {
  min-height: 40px;
}
.app-detail.is-mobile .app-files {
  flex: 1;
  height: auto;
  min-height: 320px;
  border-radius: 0;
  border: none;
  border-top: 1px solid var(--el-border-color-lighter);
}
.app-detail.is-mobile .app-side {
  flex: none;
  width: 100%;
  gap: 12px;
}
.app-detail.is-mobile .app-card {
  padding: 4px 16px 12px;
}
.app-detail.is-mobile .app-quick {
  gap: 6px;
  padding: 6px 12px;
}
.app-detail.is-mobile .app-cmd {
  gap: 6px;
  padding: 6px 8px calc(8px + env(safe-area-inset-bottom)) 12px;
}
.app-detail.is-mobile .app-cmd__field {
  min-height: 44px;
  padding: 0 14px;
}
.app-detail.is-mobile .app-cmd__field input {
  font-size: 1rem;
}
.app-stats.is-mobile {
  gap: 10px;
}
.app-stats.is-mobile .app-stat {
  gap: 4px;
  padding: 12px 14px;
}
.app-stats.is-mobile .app-stat__v {
  font-size: 1.125rem;
}
</style>
