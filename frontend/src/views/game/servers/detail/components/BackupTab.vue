<!-- 备份与回档：立即备份（备注）、定时备份与保留份数、备份范围（适配器给出）、备份列表（下载 / 回档 / 删除）。 -->
<template>
  <div class="bk">
    <div class="bk__top">
      <section class="bk__card bk__create">
        <h2>{{ t('gameServers.backup.title') }}</h2>
        <div class="bk__row">
          <UInput v-model="note" class="bk__grow" :placeholder="t('gameServers.backup.notePh')" :maxlength="200" />
          <UButton color="primary" icon="i-lucide-archive" :loading="creating" :disabled="busy" @click="create">{{ t('gameServers.backup.create') }}</UButton>
        </div>
        <div v-if="data" class="bk__scope">
          <span class="bk__label">{{ t('gameServers.backup.scope') }}</span>
          <span class="bk__chips">
            <span v-for="s in data.scope" :key="s" class="bk__chip m-mono">{{ s.startsWith('external:') ? s.slice(9) : s }}</span>
          </span>
          <small v-if="hasDb">{{ t('gameServers.backup.scopeDb') }}</small>
          <small v-for="n in data.notes" :key="n" class="is-warn">{{ n === 'database_external' ? t('gameServers.backup.noteExternal') : n }}</small>
          <small>{{ t('gameServers.backup.dir', { dir: data.dir, size: fmtBytes(data.totalSize) }) }}</small>
        </div>
        <TaskPanel v-if="task" :server-id="id" :task="task" @finished="onFinished" />
      </section>
      <section class="bk__card bk__policy">
        <h2>{{ t('gameServers.backup.policy') }}</h2>
        <label class="bk__toggle">
          <span>{{ t('gameServers.backup.policyEnabled') }}</span>
          <AppSwitch v-model="policy.enabled" :aria-label="t('gameServers.backup.policyEnabled')" />
        </label>
        <div class="bk__grid">
          <label class="bk__field">
            <span>{{ t('gameServers.backup.interval') }}</span>
            <UInput v-model.number="policy.intervalHours" type="number" :min="1" :max="720" />
          </label>
          <label class="bk__field">
            <span>{{ t('gameServers.backup.keep') }}</span>
            <UInput v-model.number="policy.keep" type="number" :min="1" :max="200" />
          </label>
        </div>
        <small class="bk__hint">{{ t('gameServers.backup.keepHint') }}</small>
        <div class="bk__row bk__row--end">
          <span class="bk__hint">
            <template v-if="data?.policy?.lastRunTime">{{ t('gameServers.backup.lastRun', { t: fmtTime(data.policy.lastRunTime) }) }}</template>
            <template v-if="data?.policy?.nextRunTime"> · {{ t('gameServers.backup.nextRun', { t: fmtTime(data.policy.nextRunTime) }) }}</template>
          </span>
          <UButton size="sm" color="neutral" variant="soft" :loading="savingPolicy" @click="savePolicy">{{ t('gameServers.common.save') }}</UButton>
        </div>
      </section>
    </div>

    <section class="bk__card">
      <EmptyState v-if="loaded && !items.length" icon="HOutline:ArchiveBoxIcon" :title="t('gameServers.backup.empty')" />
      <ul v-else class="bk__list">
        <li v-for="b in items" :key="b.id" class="bk__item">
          <span class="bk__item-ico m-sticker" :class="b.trigger === 'manual' ? 'm-sticker--p' : 'm-sticker--a2'"><UIcon name="i-lucide-archive" /></span>
          <span class="bk__item-text">
            <b class="m-mono">{{ b.fileName || t('game.unnamed') }}</b>
            <small>
              {{ t(`gameServers.backup.trigger.${b.trigger}`) }} · {{ fmtTime(b.createTime, true) }} · {{ fmtBytes(b.size) }}
              <template v-if="b.operatorName"> · {{ b.operatorName }}</template>
              <template v-if="b.note"> · {{ b.note }}</template>
            </small>
            <small v-if="b.error" class="is-bad">{{ b.error }}</small>
            <small v-else-if="b.state === 'ready' && !b.exists" class="is-bad">{{ t('gameServers.backup.missing') }}</small>
          </span>
          <StatusPill :variant="b.state === 'ready' ? 'success' : b.state === 'failed' ? 'error' : 'info'" :label="t(`gameServers.backup.state.${b.state}`)" />
          <span class="bk__item-acts">
            <AppIconButton icon="HOutline:ArrowDownTrayIcon" :label="t('gameServers.backup.download')" :disabled="b.state !== 'ready' || !b.exists" @click="download(b)" />
            <AppIconButton icon="HOutline:ArrowUturnLeftIcon" :label="t('gameServers.backup.restore')" :disabled="b.state !== 'ready' || !b.exists || busy" @click="askRestore(b)" />
            <AppIconButton icon="HOutline:TrashIcon" :label="t('gameServers.backup.delete')" class="is-danger" :disabled="b.state === 'running'" @click="askDelete(b)" />
          </span>
        </li>
      </ul>
      <Pagination v-if="total > pageSize" v-model:page="page" v-model:page-size="pageSize" :total="total" @change="load" />
    </section>

    <FormDialog v-model="restoreOpen" :title="t('gameServers.backup.restoreTitle')" :width="520" :loading="acting" :confirm-text="t('gameServers.backup.restoreConfirm')" @confirm="restore">
      <p class="bk__dialog">{{ t('gameServers.backup.restoreDesc', { file: active?.fileName || '' }) }}</p>
    </FormDialog>
    <FormDialog v-model="deleteOpen" :title="t('gameServers.backup.deleteTitle')" :width="460" :loading="acting" :confirm-text="t('gameServers.backup.delete')" @confirm="remove">
      <p class="bk__dialog">{{ t('gameServers.backup.deleteDesc', { file: active?.fileName || '' }) }}</p>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  createGameServerBackup,
  deleteGameServerBackup,
  downloadGameServerBackup,
  getGameServerTask,
  listGameServerBackups,
  restoreGameServerBackup,
  updateGameServerBackupPolicy,
} from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import { downloadFileFromUrl, resolvePreSignedFileUrl } from '@/utils/file'
import type { GameAdapter, GameServerBackupInfo, GameServerInfo, GameServerTask, ListGameServerBackupsResponse } from '@/types/v1/game_server'
import { fmtBytes, fmtTime } from '../../shared'
import TaskPanel from './TaskPanel.vue'

const props = defineProps<{ server: GameServerInfo; adapter?: GameAdapter | null }>()
const emit = defineEmits<{ changed: [] }>()
const { t } = useI18n()
const fb = useFeedback()

const id = computed(() => props.server.brief?.id || '')
const data = ref<ListGameServerBackupsResponse | null>(null)
const items = computed(() => data.value?.items || [])
const total = computed(() => data.value?.total || 0)
const page = ref(1)
const pageSize = ref(20)
const loaded = ref(false)
const note = ref('')
const creating = ref(false)
const task = ref<GameServerTask | null>(null)
const policy = reactive({ enabled: false, intervalHours: 24, keep: 7 })
const savingPolicy = ref(false)
const active = ref<GameServerBackupInfo | null>(null)
const restoreOpen = ref(false)
const deleteOpen = ref(false)
const acting = ref(false)

const busy = computed(() => (props.server.runningTasks || []).some((x) => !x.kind.startsWith('resource')))
const hasDb = computed(() => (data.value?.scope || []).some((s) => s.endsWith('.sqlite') || s.startsWith('external:') || s.includes('.db')))

const load = async () => {
  try {
    const res = (await listGameServerBackups(id.value, page.value, pageSize.value)).data
    data.value = res
    if (res.policy) Object.assign(policy, { enabled: res.policy.enabled, intervalHours: res.policy.intervalHours, keep: res.policy.keep })
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    loaded.value = true
  }
}

const trackTask = async (taskId: string) => {
  try {
    task.value = (await getGameServerTask(id.value, taskId)).data
  } catch {
    task.value = null
  }
}

const create = async () => {
  creating.value = true
  try {
    const { data: res } = await createGameServerBackup(id.value, note.value.trim())
    fb.success(t('gameServers.backup.started'))
    note.value = ''
    await trackTask(res.taskId)
    emit('changed')
    load()
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    creating.value = false
  }
}
const onFinished = () => {
  load()
  emit('changed')
}

const savePolicy = async () => {
  savingPolicy.value = true
  try {
    await updateGameServerBackupPolicy(id.value, {
      enabled: policy.enabled,
      intervalHours: Number(policy.intervalHours) || 24,
      keep: Number(policy.keep) || 7,
    })
    fb.success(t('gameServers.backup.policySaved'))
    load()
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    savingPolicy.value = false
  }
}

const download = async (b: GameServerBackupInfo) => {
  try {
    const { data: res } = await downloadGameServerBackup(id.value, b.id)
    downloadFileFromUrl(resolvePreSignedFileUrl(res.downloadUrlPath), b.fileName)
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  }
}
const askRestore = (b: GameServerBackupInfo) => {
  active.value = b
  restoreOpen.value = true
}
const askDelete = (b: GameServerBackupInfo) => {
  active.value = b
  deleteOpen.value = true
}
const restore = async () => {
  if (!active.value) return
  acting.value = true
  try {
    const { data: res } = await restoreGameServerBackup(id.value, active.value.id)
    fb.success(t('gameServers.backup.restoreStarted'))
    restoreOpen.value = false
    await trackTask(res.taskId)
    emit('changed')
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    acting.value = false
  }
}
const remove = async () => {
  if (!active.value) return
  acting.value = true
  try {
    await deleteGameServerBackup(id.value, active.value.id)
    fb.success(t('gameServers.backup.deleted'))
    deleteOpen.value = false
    load()
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    acting.value = false
  }
}

onMounted(load)
onActivated(load)
</script>

<style scoped lang="scss">
.bk {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.bk__top {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 16px;
  align-items: start;
}
.bk__card {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 18px 20px;
  border-radius: var(--app-radius-lg);
  border: var(--cmp-card-border-width, 1px) solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.bk__card h2 {
  margin: 0;
  font-size: 1.0625rem;
  font-weight: 800;
}
.bk__row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.bk__row--end {
  justify-content: space-between;
}
.bk__grow {
  flex: 1 1 200px;
  min-width: 0;
}
.bk__scope {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.bk__scope .is-warn {
  color: var(--m-warn);
  font-weight: 700;
}
.bk__label {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.bk__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.bk__chip {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.75rem;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
}
.bk__toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 44px;
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
}
.bk__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.bk__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.bk__field > span {
  font-size: 0.8125rem;
  font-weight: 700;
}
.bk__hint {
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.bk__list {
  margin: 0;
  padding: 0;
  list-style: none;
}
.bk__item {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 60px;
  padding: 8px 0;
  border-top: 1px solid var(--el-border-color-light);
}
.bk__item:first-child {
  border-top: none;
}
.bk__item-ico {
  flex: none;
  width: 38px;
  height: 38px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
}
.bk__item-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 0.875rem;
}
.bk__item-text b {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bk__item-text small {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.bk__item-text small.is-bad {
  color: var(--m-bad);
}
.bk__item-acts {
  display: flex;
  gap: 2px;
}
.bk__item-acts :deep(.app-icon-btn) {
  width: 40px !important;
  height: 40px !important;
}
.bk__item-acts .is-danger {
  color: var(--m-bad);
}
.bk__dialog {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.6;
}
@media (width <= 768px) {
  .bk__item {
    flex-wrap: wrap;
  }
  .bk__item-acts {
    width: 100%;
    justify-content: flex-end;
  }
  .bk__item-acts :deep(.app-icon-btn) {
    width: 44px !important;
    height: 44px !important;
  }
  .bk__grid {
    grid-template-columns: 1fr;
  }
}
</style>
