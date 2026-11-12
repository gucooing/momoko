<!-- 更新：服务端程序（当前构建的版本从可执行文件的构建信息读取；上传新构建或指定路径 → 后台任务：停止、备份、替换、
     启动、健康检查，失败自动回滚）与游戏资源（资源服务器数据目录的清单、发布、下载进度、/health；校验与更新任务）。 -->
<template>
  <div class="up">
    <div class="up__cards">
      <section class="up__card">
        <div class="up__card-head">
          <span class="up__ico m-sticker m-sticker--p"><UIcon name="i-lucide-cpu" /></span>
          <span class="up__card-title">
            <b>{{ t('gameServers.update.serverTitle') }}</b>
            <small class="m-mono">{{ build?.current?.path.split(/[\\/]/).pop() }}</small>
          </span>
          <StatusPill
            v-if="build?.current"
            :variant="build.current.valid ? 'success' : 'error'"
            :label="build.current.valid ? t('gameServers.update.latest') : t('gameServers.update.notFound')"
          />
        </div>
        <dl v-if="build?.current" class="up__dl">
          <div>
            <dt>{{ t('gameServers.update.version') }}</dt>
            <dd class="m-num">{{ build.current.version || t('gameServers.update.versionNone') }}</dd>
          </div>
          <div>
            <dt>{{ t('gameServers.update.running') }}</dt>
            <dd class="m-num">{{ build.runningVersion || '—' }}</dd>
          </div>
          <div>
            <dt>{{ t('gameServers.update.migration') }}</dt>
            <dd class="m-num">{{ build.migration || '—' }}</dd>
          </div>
          <div>
            <dt>{{ t('gameServers.update.lastUpdate') }}</dt>
            <dd class="m-num">{{ fmtTime(build.lastUpdateTime) }}</dd>
          </div>
          <div>
            <dt>{{ t('gameServers.update.size') }}</dt>
            <dd class="m-num">{{ fmtBytes(build.current.size) }}</dd>
          </div>
          <div>
            <dt>{{ t('gameServers.update.modified') }}</dt>
            <dd class="m-num">{{ fmtTime(build.current.modifiedAt, true) }}</dd>
          </div>
          <div>
            <dt>{{ t('gameServers.update.goVersion') }}</dt>
            <dd class="m-num">{{ build.current.goVersion || '—' }}</dd>
          </div>
          <div>
            <dt>SHA-256</dt>
            <dd class="m-mono" :title="build.current.sha256">{{ build.current.sha256 ? `${build.current.sha256.slice(0, 12)}…` : '—' }}</dd>
          </div>
        </dl>
        <p class="up__hint">{{ t('gameServers.update.flow') }}</p>

        <div class="up__row">
          <input ref="fileRef" type="file" class="up__file" @change="onPick" />
          <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-upload" :loading="uploader.uploading.value" :disabled="busy" @click="fileRef?.click()">
            {{ t('gameServers.update.upload') }}
          </UButton>
          <span v-if="uploader.uploading.value" class="up__muted m-num">{{ t('gameServers.update.uploading', { p: Math.round(uploader.overallProgress.value) }) }}</span>
        </div>
        <div class="up__row">
          <UInput v-model="pathInput" class="up__grow m-mono" size="sm" :placeholder="t('gameServers.update.pathPh')" @keyup.enter="checkPath" />
          <UButton color="neutral" variant="soft" size="sm" :loading="checking" :disabled="!pathInput.trim()" @click="checkPath">{{ t('gameServers.update.check') }}</UButton>
        </div>
        <div v-if="candidates.length" class="up__builds">
          <span class="up__label">{{ t('gameServers.update.uploads') }}</span>
          <div v-for="b in candidates" :key="b.path" class="up__build">
            <span class="up__build-text">
              <b class="m-mono">{{ b.path.split('/').pop() }}</b>
              <small>
                <template v-if="b.valid">{{ b.version || t('gameServers.update.versionNone') }} · {{ fmtBytes(b.size) }} · {{ fmtTime(b.modifiedAt) }}</template>
                <template v-else>{{ t(`gameServers.update.buildInvalid.${b.error}`, b.error) }}</template>
              </small>
            </span>
            <UButton v-if="b.valid" size="sm" color="primary" variant="soft" :disabled="busy" @click="askUpdate(b)">{{ t('gameServers.update.useThis') }}</UButton>
          </div>
        </div>
        <TaskPanel v-if="build?.task" :server-id="id" :task="build.task" @finished="onFinished" />
      </section>

      <section class="up__card">
        <div class="up__card-head">
          <span class="up__ico m-sticker m-sticker--a2"><UIcon name="i-lucide-cloud" /></span>
          <span class="up__card-title">
            <b>{{ t('gameServers.update.resourceTitle') }}</b>
            <small>{{ t('gameServers.update.resourceSub') }}</small>
          </span>
          <StatusPill v-if="res && health" :variant="health.ready ? 'success' : 'warning'" :label="health.ready ? t('gameServers.update.healthReady') : t('gameServers.update.healthPartial')" />
        </div>
        <p v-if="res && !res.hosted" class="up__hint">{{ t('gameServers.update.resourceNone') }}</p>
        <dl v-if="res" class="up__dl">
          <div class="is-wide">
            <dt>{{ t('gameServers.update.resourceRoot') }}</dt>
            <dd class="m-mono" :title="res.root">{{ res.root || '—' }}</dd>
          </div>
          <div>
            <dt>{{ t('gameServers.update.current') }}</dt>
            <dd class="m-num">{{ res.version ? `Octo ${res.version} · r${res.revision}` : '—' }}</dd>
          </div>
          <div>
            <dt>{{ t('gameServers.update.published') }}</dt>
            <dd class="m-num" :class="{ 'is-warn': !res.published }">
              {{ res.published ? `Octo ${res.publishedVersion} · r${res.publishedRevision}` : t('gameServers.update.notPublished') }}
            </dd>
          </div>
          <div>
            <dt>{{ t('gameServers.update.objects') }}</dt>
            <dd class="m-num">{{ res.objects ? `${fmtNum(res.objects)} · ${fmtBytes(res.bytes)}` : '—' }}</dd>
          </div>
          <div>
            <dt>{{ t('gameServers.update.masters') }}</dt>
            <dd class="m-num">{{ res.masters ? fmtNum(res.masters) : '—' }}</dd>
          </div>
          <div v-if="loaded">
            <dt>{{ t('gameServers.update.loaded') }}</dt>
            <dd class="m-num" :title="loaded.master">{{ t('gameServers.update.loadedValue', { octo: loaded.octo ?? '—', master: shortHash(loaded.master) }) }}</dd>
          </div>
          <div v-if="health">
            <dt>{{ t('gameServers.update.health') }}</dt>
            <dd class="m-num">{{ t('gameServers.update.healthValue', { available: fmtNum(health.available), missing: fmtNum(health.missing) }) }}</dd>
          </div>
          <div v-else-if="res.hosted && res.healthError">
            <dt>{{ t('gameServers.update.health') }}</dt>
            <dd class="is-bad">{{ t('gameServers.update.healthDown') }}</dd>
          </div>
        </dl>
        <p v-if="res?.error" class="up__error">{{ res.error }}</p>
        <div v-if="progress" class="up__progress">
          <div class="up__progress-line">
            <span>{{ t('gameServers.update.progress') }}</span>
            <span class="m-num">{{ t('gameServers.update.progressValue', progress) }}</span>
          </div>
          <div class="up__bar"><span :style="{ width: progress.pct }" /></div>
        </div>
        <div class="up__row">
          <template v-if="res?.executable">
            <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-shield-check" :loading="running === 'verify'" :disabled="resBusy" @click="runResource('verify')">
              {{ t('gameServers.update.verify') }}
            </UButton>
            <UButton color="primary" variant="soft" size="sm" icon="i-lucide-refresh-cw" :disabled="resBusy" @click="updateOpen = true">
              {{ t('gameServers.update.update') }}
            </UButton>
          </template>
          <span v-else-if="res" class="up__muted">{{ t('gameServers.update.noExecutable') }}</span>
        </div>
        <TaskPanel v-if="res?.task" :server-id="id" :task="res.task" @finished="loadResource" />
      </section>
    </div>

    <HistoryList ref="historyRef" :server-id="id" :title="t('gameServers.update.historyTitle')" :kinds="['server', 'resource', 'config', 'key', 'mode', 'process']" />

    <FormDialog v-model="confirmOpen" :title="t('gameServers.update.confirmTitle')" :width="520" :loading="starting" :confirm-text="t('gameServers.update.start')" @confirm="startUpdate">
      <div class="up__dialog">
        <p>{{ t('gameServers.update.confirmDesc', { from: build?.current?.version || t('gameServers.update.versionNone'), to: target?.version || t('gameServers.update.versionNone') }) }}</p>
        <label class="up__field">
          <span>{{ t('gameServers.update.healthTimeout') }}</span>
          <UInput v-model.number="healthTimeout" type="number" :min="10" :max="600" />
        </label>
      </div>
    </FormDialog>
    <FormDialog v-model="updateOpen" :title="t('gameServers.update.updateConfirm')" :width="560" :loading="running === 'update'" :confirm-text="t('gameServers.update.updateConfirm')" @confirm="runResource('update')">
      <div class="up__dialog">
        <p>{{ t('gameServers.update.updateDesc') }}</p>
        <label class="up__field">
          <span>{{ t('gameServers.update.masterMetadata') }}</span>
          <UInput v-model="masterMetadata" class="m-mono" :placeholder="t('gameServers.update.masterMetadataPh')" />
        </label>
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  checkGameServerBuild,
  getGameServerBuild,
  getGameServerResource,
  getGameServerTask,
  runGameServerResourceTask,
  updateGameServerBuild,
} from '@/api/gameServer'
import { useFileUpload } from '@/components/file/useFileUpload'
import { getRequestErrorMessage } from '@/utils/request'
import type { GameAdapter, GameServerBuild, GameServerBuildInfo, GameServerInfo, GameServerResource, GameServerTask } from '@/types/v1/game_server'
import { createGameServerFileClient } from '../../fileClient'
import { fmtBytes, fmtNum, fmtTime } from '../../shared'
import HistoryList from './HistoryList.vue'
import TaskPanel from './TaskPanel.vue'

const props = defineProps<{ server: GameServerInfo; adapter?: GameAdapter | null }>()
const emit = defineEmits<{ changed: [] }>()
const { t } = useI18n()
const fb = useFeedback()

const id = computed(() => props.server.brief?.id || '')
const build = ref<GameServerBuild | null>(null)
const res = ref<GameServerResource | null>(null)
const pathInput = ref('')
const checked = ref<GameServerBuildInfo | null>(null)
const checking = ref(false)
const confirmOpen = ref(false)
const updateOpen = ref(false)
const target = ref<GameServerBuildInfo | null>(null)
const healthTimeout = ref(60)
const starting = ref(false)
const running = ref<'' | 'verify' | 'update'>('')
const masterMetadata = ref('')
const historyRef = ref<InstanceType<typeof HistoryList>>()
const fileRef = ref<HTMLInputElement>()

const active = (task?: GameServerTask) => task?.status === 'pending' || task?.status === 'running'
const busy = computed(() => (props.server.runningTasks || []).some((x) => !x.kind.startsWith('resource')) || active(build.value?.task))
const resBusy = computed(() => (props.server.runningTasks || []).some((x) => x.kind.startsWith('resource')) || active(res.value?.task))

const candidates = computed(() => {
  const list = [...(build.value?.uploads || [])]
  if (checked.value && !list.some((b) => b.path === checked.value?.path)) list.unshift(checked.value)
  return list
})

type HealthDoc = { status?: string; mirror_complete?: boolean; available?: number; missing?: number }
const health = computed(() => {
  const h = res.value?.health as HealthDoc | undefined
  if (!h || typeof h !== 'object') return null
  return { ready: !!h.mirror_complete, available: h.available ?? 0, missing: h.missing ?? 0 }
})
type LoadedDoc = { app?: string; master?: string; octo?: number }
const loaded = computed(() => {
  const v = res.value?.serverResource as LoadedDoc | undefined
  return v && typeof v === 'object' ? v : null
})
type ProgressDoc = { total?: number; completed?: number; bytes?: number; totalBytes?: number; complete?: boolean }
const progress = computed(() => {
  const p = res.value?.progress as ProgressDoc | undefined
  if (!p || typeof p !== 'object' || !p.total || p.complete) return null
  const pct = Math.min(100, ((p.completed || 0) / p.total) * 100)
  return {
    done: fmtNum(p.completed || 0),
    total: fmtNum(p.total),
    bytes: fmtBytes(p.bytes || 0),
    totalBytes: fmtBytes(p.totalBytes || 0),
    pct: `${pct.toFixed(1)}%`,
  }
})
const shortHash = (v?: string) => (v && v.length > 12 ? `${v.slice(0, 10)}…` : v || '—')

const loadBuild = async () => {
  try {
    build.value = (await getGameServerBuild(id.value)).data
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  }
}
const loadResource = async () => {
  try {
    res.value = (await getGameServerResource(id.value)).data
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  }
}

// 上传新构建：分片上传到服务端目录的 .momoko/uploads
const client = createGameServerFileClient(id.value)
const uploader = useFileUpload(
  () => client,
  () => build.value?.uploadDir || '.momoko/uploads',
  {
    onUploaded: (item) => {
      fb.success(t('gameServers.update.uploaded', { name: item.name }))
      loadBuild()
    },
  },
)
const onPick = (e: Event) => {
  const input = e.target as HTMLInputElement
  if (!input.files?.length) return
  uploader.addFiles(input.files)
  input.value = ''
  uploader.startAll()
}

const checkPath = async () => {
  const p = pathInput.value.trim()
  if (!p) return
  checking.value = true
  try {
    checked.value = (await checkGameServerBuild(id.value, p)).data
    if (!checked.value.valid) fb.warning(t(`gameServers.update.buildInvalid.${checked.value.error}`, checked.value.error))
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    checking.value = false
  }
}

const askUpdate = (b: GameServerBuildInfo) => {
  target.value = b
  healthTimeout.value = 60
  confirmOpen.value = true
}
const startUpdate = async () => {
  if (!target.value) return
  starting.value = true
  try {
    const { data } = await updateGameServerBuild(id.value, target.value.path, Number(healthTimeout.value) || 60)
    fb.success(t('gameServers.update.started'))
    confirmOpen.value = false
    const task = (await getGameServerTask(id.value, data.taskId)).data
    if (build.value) build.value = { ...build.value, task }
    emit('changed')
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    starting.value = false
  }
}
const onFinished = () => {
  checked.value = null
  loadBuild()
  historyRef.value?.reload()
  emit('changed')
}

const runResource = async (action: 'verify' | 'update') => {
  running.value = action
  try {
    const { data } = await runGameServerResourceTask(id.value, action, action === 'update' ? masterMetadata.value.trim() : '')
    fb.success(t(action === 'verify' ? 'gameServers.update.verifyStarted' : 'gameServers.update.updateStarted'))
    updateOpen.value = false
    const task = (await getGameServerTask(id.value, data.taskId)).data
    if (res.value) res.value = { ...res.value, task }
    emit('changed')
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    running.value = ''
  }
}

onMounted(() => {
  loadBuild()
  loadResource()
})
onActivated(() => {
  loadBuild()
  loadResource()
})
</script>

<style scoped lang="scss">
.up {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.up__cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(340px, 1fr));
  gap: 16px;
  align-items: start;
}
.up__card {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 20px 22px;
  border-radius: var(--app-radius-lg);
  border: var(--cmp-card-border-width, 1px) solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.up__card-head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.up__ico {
  flex: none;
  width: 44px;
  height: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  font-size: 1.25rem;
}
.up__card-title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.up__card-title b {
  font-size: 1.0625rem;
}
.up__card-title small,
.up__hint,
.up__muted {
  font-size: 0.75rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.up__hint {
  margin: 0;
}
.up__dl {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 14px;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.up__dl div {
  min-width: 0;
}
.up__dl .is-wide {
  grid-column: 1 / -1;
}
.up__dl dt {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.up__dl dd {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.up__dl dd.is-warn {
  color: var(--m-warn);
}
.up__dl dd.is-bad {
  color: var(--m-bad);
}
.up__row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.up__grow {
  flex: 1 1 220px;
  min-width: 0;
}
.up__file {
  display: none;
}
.up__label {
  font-size: 0.8125rem;
  font-weight: 700;
}
.up__builds {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.up__build {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 48px;
  padding: 6px 12px;
  border-radius: var(--app-radius-sm);
  border: 1px solid var(--el-border-color-light);
}
.up__build-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  font-size: 0.8125rem;
}
.up__build-text b {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.up__build-text small {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.up__error {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--m-bad);
  overflow-wrap: anywhere;
}
.up__progress {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.up__progress-line {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  font-size: 0.8125rem;
  font-weight: 700;
}
.up__bar {
  height: 6px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}
.up__bar span {
  display: block;
  height: 100%;
  background: var(--m-gInk, var(--m-p));
}
.up__dialog {
  display: flex;
  flex-direction: column;
  gap: 14px;
  font-size: 0.875rem;
  line-height: 1.6;
}
.up__dialog p {
  margin: 0;
}
.up__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.up__field > span {
  font-size: 0.8125rem;
  font-weight: 700;
}
@media (width <= 768px) {
  .up__cards {
    grid-template-columns: 1fr;
  }
  .up__card {
    padding: 16px;
  }
}
</style>
