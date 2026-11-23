<!-- Agent 设置 · 工作区：一行一个，行内「新会话」「查看文件」（本页弹窗只读浏览目录）「编辑」「⋯」。
     新建 / 编辑用 AgWorkspaceDialog；移除只移除记录，不删目录里的文件。有工作区还在克隆时每 3 秒刷新一次。
     只有自己的工作区；在 Agent 管理页里（useAgentAll）是所有人的，行上带主人，只能移除。 -->
<template>
  <div class="ag ag-page">
    <AgManageHeader>
      <template #default>
        <template v-if="count != null">{{ t('agent.workspaces.total', { n: count }) }}</template>
      </template>
      <template v-if="!all" #actions>
        <button type="button" class="ag-btn ag-btn--p" @click="openCreate">
          <UIcon name="i-lucide-plus" />{{ t('agent.workspaces.create') }}
        </button>
      </template>
    </AgManageHeader>

    <div class="ag-body">
      <div class="ag-grp">
        <div v-for="w in workspaces" :key="w.id" class="ag-rw ag-ws-row">
          <UIcon name="i-lucide-folder" class="ag-ws-row__ico" />
          <div class="ag-rm">
            <div class="ag-rt">
              <span>{{ w.name }}</span>
              <span
                v-for="tag in tagsOf(w)"
                :key="tag.text"
                class="ag-tag"
                :class="{ 'ag-tag--warn': tag.warn }"
                :title="tag.title || undefined"
              >
                <UIcon v-if="tag.spin" name="i-lucide-loader-circle" class="ag-spin" />{{ tag.text }}
              </span>
            </div>
            <div class="ag-rs">
              <span class="ag-mono">{{ w.path }}</span> · {{ metaOf(w) }}
            </div>
          </div>
          <div v-if="all" class="ag-rc">
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="t('agent.workspaces.removeOf', { name: w.name })"
              :title="t('agent.workspaces.remove')"
              @click="askRemove(w)"
            >
              <UIcon name="i-lucide-trash-2" />
            </button>
          </div>
          <div v-else class="ag-rc">
            <RouterLink class="ag-btn ag-btn--sm ag-ws-row__new" :to="{ path: '/agent/code', query: { workspace: w.id } }">
              <UIcon name="i-lucide-plus" />{{ t('agent.common.newSession') }}
            </RouterLink>
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              aria-haspopup="dialog"
              :aria-label="t('agent.workspaces.viewFilesOf', { name: w.name })"
              :title="t('agent.workspaces.viewFiles')"
              @click="openFiles(w)"
            >
              <UIcon name="i-lucide-folder" />
            </button>
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="t('agent.workspaces.editOf', { name: w.name })"
              :title="t('agent.common.edit')"
              @click="openEdit(w)"
            >
              <UIcon name="i-lucide-pencil" />
            </button>
            <AppDropdown panel-class="ag ag-menu">
              <template #trigger="{ open }">
                <button
                  type="button"
                  class="ag-ib ag-ib--sm"
                  aria-haspopup="menu"
                  :class="{ 'is-on': open }"
                  :aria-expanded="open"
                  :aria-label="t('agent.workspaces.moreOf', { name: w.name })"
                  :title="t('agent.common.more')"
                >
                  <UIcon name="i-lucide-ellipsis" />
                </button>
              </template>
              <template #default="{ close }">
                <button type="button" class="ag-mi" role="menuitem" @click="close(); openEdit(w)">
                  {{ t('agent.common.edit') }}
                </button>
                <button type="button" class="ag-mi is-bad" role="menuitem" @click="close(); askRemove(w)">
                  {{ t('agent.workspaces.remove') }}
                </button>
              </template>
            </AppDropdown>
          </div>
        </div>
        <div v-if="!loaded" class="ag-empty">{{ t('agent.common.loading') }}</div>
        <div v-else-if="!workspaces.length" class="ag-empty">{{ t('agent.workspaces.empty') }}</div>
      </div>
    </div>

    <AgWorkspaceDialog v-model="dialogOpen" :workspace="dialogTarget" @saved="onSaved" />

    <!-- 查看文件：只读地浏览工作区目录 -->
    <AgDialog v-model="files.open" :title="files.name" :width="520" flush close-on-overlay>
      <template #icon><UIcon name="i-lucide-folder" class="ag-ws-files__ico" /></template>
      <div class="ag-mono ag-noscroll ag-ws-crumbs" :aria-label="t('agent.workspaces.files.location')">
        <template v-for="(c, i) in crumbs" :key="c.dir">
          <button type="button" :class="{ 'is-cur': i === crumbs.length - 1 }" @click="browse(c.dir)">
            {{ c.text }}
          </button>
          <span aria-hidden="true">/</span>
        </template>
      </div>
      <div ref="fileBox" class="ag-noscroll ag-ws-files" :aria-busy="files.loading">
        <button
          v-if="files.dir"
          type="button"
          class="ag-mi"
          :aria-label="t('agent.workspaces.files.up')"
          @click="browse(parentDir)"
        >
          <UIcon name="i-lucide-folder" />
          <span class="ag-mono ag-ws-files__name">..</span>
        </button>
        <button
          v-for="d in fileDirs"
          :key="d.path"
          type="button"
          class="ag-mi ag-ws-files__dir"
          @click="browse(d.path)"
        >
          <UIcon name="i-lucide-folder" />
          <span class="ag-mono ag-ellipsis ag-ws-files__name">{{ d.name }}</span>
        </button>
        <div v-for="f in fileFiles" :key="f.path" class="ag-ws-files__file">
          <UIcon name="i-lucide-file" />
          <span class="ag-mono ag-ellipsis ag-ws-files__name">{{ f.name }}</span>
          <span class="ag-num ag-ws-files__size">{{ fmtBytes(f.size) }}</span>
          <span class="ag-ws-files__time">{{ relTime(f.modTime) }}</span>
        </div>
        <div v-if="!files.loaded" class="ag-ws-files__note">{{ t('agent.common.loading') }}</div>
        <div v-else-if="!files.entries.length" class="ag-ws-files__note">
          {{ t('agent.workspaces.files.empty') }}
        </div>
      </div>
    </AgDialog>

    <AgConfirm
      v-model="removeOpen"
      danger
      :title="t('agent.workspaces.removeTitle', { name: removeTarget?.name ?? '' })"
      :message="t('agent.workspaces.removeMessage')"
      :confirm-text="t('agent.common.remove')"
      :on-confirm="doRemove"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { browseAgentWorkspace, deleteAgentWorkspace, listAgentWorkspaces } from '@/api/agent'
import type { AgentFileEntry, AgentUser, AgentWorkspace } from '@/types/v1/agent'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import AgManageHeader from '@/views/agent/components/AgManageHeader.vue'
import AgWorkspaceDialog from '@/views/agent/components/AgWorkspaceDialog.vue'
import { useAgentAll } from '@/views/agent/composables/useAgentAll'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'
import { fmtBytes, relTime } from '@/views/agent/utils/format'

defineOptions({ name: 'AgentManageWorkspacesView' })

/** 有工作区还在克隆时，隔这么久刷新一次 */
const POLL_MS = 3000

const { t } = useI18n()
/** 在 Agent 管理页里：所有人的工作区 */
const all = useAgentAll()
const { overview, refresh: refreshOverview } = useAgentOverview()

// —— 列表 ——
const workspaces = ref<AgentWorkspace[]>([])
const users = ref<AgentUser[]>([])
const loaded = ref(false)
let seq = 0
let active = false
let timer: ReturnType<typeof setTimeout> | undefined

const stopPoll = () => {
  if (timer) clearTimeout(timer)
  timer = undefined
}

const load = async () => {
  const id = ++seq
  stopPoll()
  try {
    const { data } = await listAgentWorkspaces(all)
    if (id !== seq) return
    workspaces.value = data.workspaces ?? []
    users.value = data.users ?? []
    // 请求出错时不再排下一次，免得每隔几秒弹一次错误
    if (active && workspaces.value.some((w) => w.cloneStatus === 'cloning')) {
      timer = setTimeout(() => void load(), POLL_MS)
    }
  } catch {
    // 请求层已经提示过
  } finally {
    if (id === seq) loaded.value = true
  }
}
/** 数量变了：列表和标签上的数字一起刷新 */
const reloadAll = () => Promise.all([load(), refreshOverview()])

const count = computed(() => {
  if (loaded.value) return workspaces.value.length
  return overview.value ? Number(overview.value.workspaces || 0) : null
})

onMounted(() => {
  active = true
  void load()
})
// 标签页是 keep-alive 的：切走就停掉定时刷新，切回来悄悄刷新一次（Code 页里也能添加工作区）
onActivated(() => {
  active = true
  if (loaded.value) void reloadAll()
})
onDeactivated(() => {
  active = false
  stopPoll()
})
onBeforeUnmount(() => {
  active = false
  stopPoll()
})

const tagsOf = (w: AgentWorkspace) => {
  const tags: { text: string; warn?: boolean; spin?: boolean; title?: string }[] = []
  if (w.isGit && w.branch) tags.push({ text: t('agent.workspaces.branch', { name: w.branch }) })
  if (w.cloneStatus === 'cloning') {
    tags.push({ text: t('agent.workspaces.cloning'), warn: true, spin: true })
  } else if (w.cloneStatus === 'failed') {
    tags.push({ text: t('agent.workspaces.cloneFailed'), warn: true, title: w.cloneError })
  } else if (w.missing) {
    tags.push({ text: t('agent.workspaces.missing'), warn: true })
  } else if (!w.isGit) {
    tags.push({ text: t('agent.workspaces.notGit'), warn: true })
  }
  return tags
}
const metaOf = (w: AgentWorkspace) => {
  const sessions = sessionsOf(w)
  if (!all) return sessions
  const u = users.value.find((x) => x.id === w.userId)
  return `${sessions} · ${u ? u.name || u.username : t('agent.admin.goneUser')}`
}
const sessionsOf = (w: AgentWorkspace) => {
  const n = Number(w.sessionCount || 0)
  if (!n) return t('agent.workspaces.noSessions')
  const time = relTime(w.activeAt)
  return time ? t('agent.workspaces.meta', { n, time }) : t('agent.workspaces.sessionCount', { n })
}

// —— 新建 / 编辑 ——
const dialogOpen = ref(false)
const dialogTarget = ref<AgentWorkspace | null>(null)
const openCreate = () => {
  dialogTarget.value = null
  dialogOpen.value = true
}
const openEdit = (w: AgentWorkspace) => {
  dialogTarget.value = w
  dialogOpen.value = true
}
const onSaved = (w: AgentWorkspace) => {
  const i = workspaces.value.findIndex((x) => x.id === w.id)
  if (i >= 0) workspaces.value.splice(i, 1, w)
  else workspaces.value.push(w)
  void reloadAll()
}

// —— 移除 ——
const removeOpen = ref(false)
const removeTarget = ref<AgentWorkspace | null>(null)
const askRemove = (w: AgentWorkspace) => {
  removeTarget.value = w
  removeOpen.value = true
}
const doRemove = async () => {
  const w = removeTarget.value
  if (!w) return
  await deleteAgentWorkspace(w.id)
  feedback.success(t('agent.workspaces.removed', { name: w.name }))
  await reloadAll()
}

// —— 查看文件 ——
const fileBox = useTemplateRef<HTMLElement>('fileBox')
const files = reactive({
  open: false,
  id: '',
  name: '',
  root: '',
  /** 相对工作区根目录，正斜杠；空 = 根 */
  dir: '',
  entries: [] as AgentFileEntry[],
  loaded: false,
  loading: false,
})
let fileSeq = 0

const browse = async (dir: string) => {
  const id = ++fileSeq
  files.loading = true
  try {
    const { data } = await browseAgentWorkspace(files.id, dir)
    if (id !== fileSeq) return
    files.dir = dir
    files.entries = data.entries ?? []
    await nextTick()
    if (fileBox.value) fileBox.value.scrollTop = 0
  } catch {
    // 请求层已经提示过，留在原来的目录
  } finally {
    if (id === fileSeq) {
      files.loaded = true
      files.loading = false
    }
  }
}
const openFiles = (w: AgentWorkspace) => {
  files.id = w.id
  files.name = w.name
  files.root = w.path
  files.dir = ''
  files.entries = []
  files.loaded = false
  files.open = true
  void browse('')
}

const dirParts = computed(() => files.dir.split('/').filter(Boolean))
const parentDir = computed(() => dirParts.value.slice(0, -1).join('/'))
const crumbs = computed(() => [
  { text: files.root, dir: '' },
  ...dirParts.value.map((text, i) => ({ text, dir: dirParts.value.slice(0, i + 1).join('/') })),
])
const fileDirs = computed(() => files.entries.filter((e) => e.isDir))
const fileFiles = computed(() => files.entries.filter((e) => !e.isDir))
</script>

<style lang="scss">
.ag-ws-row__ico {
  flex: none;
  width: 20px;
  height: 20px;
  color: var(--m-ink2);
}
.ag-ws-row__new {
  margin-right: 4px;
}
.ag-ws-files__ico {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--m-ink2);
}
.ag-ws-crumbs {
  display: flex;
  align-items: center;
  gap: 2px;
  margin: 4px 14px 0;
  padding: 0 4px;
  min-height: 30px;
  border-radius: 8px;
  background: var(--m-s2);
  font-size: 12px;
  color: var(--m-ink2);
  overflow-x: auto;
  white-space: nowrap;
}
.ag-ws-crumbs button {
  flex: none;
  min-height: 24px;
  padding: 0 6px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  font: inherit;
  color: var(--m-ink2);
  cursor: pointer;
}
.ag-ws-crumbs button:hover,
.ag-ws-crumbs button.is-cur {
  color: var(--m-ink);
}
.ag-ws-files {
  display: flex;
  flex-direction: column;
  gap: 1px;
  height: 320px;
  overflow-y: auto;
  padding: 6px 10px 12px;
}
.ag-ws-files .ag-mi {
  flex: none;
}
.ag-ws-files .ag-mi > .iconify {
  color: var(--m-ink2);
}
.ag-ws-files .ag-ws-files__dir > .iconify {
  color: var(--m-pInk);
}
.ag-ws-files__file,
.ag-ws-files__note {
  flex: none;
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 30px;
  padding: 0 9px;
  border-radius: 8px;
  font-size: 13px;
}
.ag-ws-files__file > .iconify {
  flex: none;
  width: 15px;
  height: 15px;
  color: var(--m-ink2);
}
.ag-ws-files__note {
  color: var(--m-ink2);
}
.ag-ws-files__name {
  flex: 1;
  min-width: 0;
  font-size: 12.5px;
}
.ag-ws-files__size,
.ag-ws-files__time {
  flex: none;
  text-align: right;
  font-size: 12px;
  color: var(--m-ink2);
  white-space: nowrap;
}
.ag-ws-files__size {
  width: 64px;
}
.ag-ws-files__time {
  width: 84px;
}
@media (width < 992px) {
  // 文件夹图标和名称留在同一行，操作换到下一行
  .ag-ws-row > .ag-rm {
    flex: 1 1 0;
  }
  .ag-ws-row__new {
    margin-right: auto;
  }
  .ag-ws-crumbs {
    min-height: 40px;
  }
  .ag-ws-crumbs button {
    min-height: 36px;
  }
  .ag-ws-files {
    height: min(320px, 50dvh);
  }
}
</style>
