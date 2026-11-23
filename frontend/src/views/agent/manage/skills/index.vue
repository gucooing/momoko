<!-- Agent 设置 · 技能：一行一个技能（名称、来源和文件数、说明），行尾是启用开关、编辑、删除；
     右上可以新建，或者从压缩包、Git 地址、服务器上已有的目录导入。
     自己的技能，后面是管理员共享给自己的（只能看，不能改、不能停用）。
     在 Agent 管理页里（useAgentAll）分两组：共享的（管理员在这里新建、导入、改、删，选共享给所有人还是指定的人），
     和各人自己的（行上带主人和启用状态，只能删除）。 -->
<template>
  <div class="ag ag-page">
    <AgManageHeader>
      <template #default>
        <template v-if="loaded">
          {{ t('agent.skills.count', { n: skills.length }) }} ·
          <span class="ag-sk-onn">{{ t('agent.skills.enabledCount', { n: enabledCount }) }}</span>
        </template>
        <template v-else>{{ t('agent.common.loading') }}</template>
      </template>
      <template #actions>
        <AppDropdown panel-class="ag ag-menu">
          <template #trigger="{ open }">
            <button type="button" class="ag-btn" aria-haspopup="menu" :aria-expanded="open" :disabled="importing">
              <UIcon v-if="importing" name="i-lucide-loader-circle" class="ag-spin" />
              <UIcon v-else name="i-lucide-download" />
              {{ t('agent.skills.import') }}
              <UIcon name="i-lucide-chevron-down" />
            </button>
          </template>
          <template #default="{ close }">
            <button v-for="k in importKinds" :key="k" type="button" class="ag-mi" role="menuitem" @click="pickImport(k, close)">
              {{ t(`agent.skills.importMenu.${k}`) }}
            </button>
          </template>
        </AppDropdown>
        <button type="button" class="ag-btn ag-btn--p" @click="openCreate">
          <UIcon name="i-lucide-plus" />{{ all ? t('agent.share.addShared') : t('agent.skills.create') }}
        </button>
      </template>
    </AgManageHeader>

    <div class="ag-body">
      <template v-for="g in groups" :key="g.key">
        <h2 v-if="all" class="ag-sh" :class="{ 'ag-gap': g.key === 'personal' }">{{ g.title }}</h2>
        <div class="ag-grp">
        <div v-for="s in g.skills" :key="s.id" class="ag-rw">
          <div class="ag-rm">
            <div class="ag-rt">
              <span class="ag-mono">{{ s.name }}</span>
              <span class="ag-sk-meta">{{ t('agent.skills.meta', { source: sourceText(s.source), n: s.fileCount }) }}</span>
              <span v-if="isShared(s)" class="ag-tag ag-tag--p" :title="scopeOf(s)">{{ shareTag(s) }}</span>
              <span v-if="all || isShared(s)" class="ag-pill" :class="{ 'ag-pill--ok': s.enabled }">
                {{ t(s.enabled ? 'agent.admin.enabled' : 'agent.admin.disabled') }}
              </span>
            </div>
            <div class="ag-rs">
              <template v-if="all && !isShared(s)">{{ ownerOf(s) }} · </template>
              <template v-if="all && isShared(s) && scopeOf(s)">{{ scopeOf(s) }} · </template>
              {{ s.description || t('agent.skills.noDesc') }}
            </div>
          </div>
          <div v-if="!all && isShared(s)" class="ag-rc">
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="t('agent.skills.viewOf', { name: s.name })"
              :title="t('agent.share.sharedReadonly')"
              :disabled="opening === s.id"
              @click="openEdit(s)"
            >
              <UIcon v-if="opening === s.id" name="i-lucide-loader-circle" class="ag-spin" />
              <UIcon v-else name="i-lucide-eye" />
            </button>
          </div>
          <div v-else-if="all && !isShared(s)" class="ag-rc">
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="t('agent.skills.deleteOf', { name: s.name })"
              :title="t('agent.common.delete')"
              @click="askDelete(s)"
            >
              <UIcon name="i-lucide-trash-2" />
            </button>
          </div>
          <div v-else class="ag-rc">
            <AgSwitch
              class="ag-sk-tg"
              :model-value="s.enabled"
              :label="t('agent.skills.enableOf', { name: s.name })"
              @update:model-value="setEnabled(s, $event)"
            />
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="t('agent.skills.editOf', { name: s.name })"
              :title="t('agent.common.edit')"
              :disabled="opening === s.id"
              @click="openEdit(s)"
            >
              <UIcon v-if="opening === s.id" name="i-lucide-loader-circle" class="ag-spin" />
              <UIcon v-else name="i-lucide-pencil" />
            </button>
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="t('agent.skills.deleteOf', { name: s.name })"
              :title="t('agent.common.delete')"
              @click="askDelete(s)"
            >
              <UIcon name="i-lucide-trash-2" />
            </button>
          </div>
        </div>
        <div v-if="!loaded" class="ag-empty">{{ t('agent.common.loading') }}</div>
        <div v-else-if="!g.skills.length" class="ag-empty">{{ g.empty }}</div>
        </div>
      </template>
    </div>

    <input ref="zipInput" type="file" accept=".zip" hidden @change="onZipPicked" />

    <SkillEditorDialog
      v-model="editorOpen"
      :detail="editorDetail"
      :shared="editorShared"
      :readonly="editorShared && !all"
      @saved="reloadAll"
    />

    <!-- Agent 管理页里导入压缩包：先选共享给谁，再选文件 -->
    <AgDialog v-model="zipOpen" :title="t('agent.skills.importMenu.zip')" :width="420">
      <AgShareField v-model="importShare" />
      <div class="ag-sk-acts">
        <button type="button" class="ag-btn" @click="zipOpen = false">{{ t('agent.common.cancel') }}</button>
        <button type="button" class="ag-btn ag-btn--p" :disabled="importing" @click="chooseZip">
          <UIcon v-if="importing" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.skills.chooseZip') }}
        </button>
      </div>
    </AgDialog>

    <AgDialog v-model="gitOpen" :title="t('agent.skills.importGit.title')" :width="420">
      <span class="ag-fld">
        <input
          v-model="gitUrl"
          class="ag-mono"
          type="text"
          autocomplete="off"
          spellcheck="false"
          :placeholder="t('agent.skills.importGit.placeholder')"
          :aria-label="t('agent.skills.importGit.label')"
          @keydown.enter="onEnter($event, importGit)"
        />
      </span>
      <AgShareField v-if="all" v-model="importShare" />
      <div class="ag-sk-acts">
        <button type="button" class="ag-btn" @click="gitOpen = false">{{ t('agent.common.cancel') }}</button>
        <button type="button" class="ag-btn ag-btn--p" :disabled="importing" @click="importGit">
          <UIcon v-if="importing" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.skills.import') }}
        </button>
      </div>
    </AgDialog>

    <AgDialog v-model="dirOpen" :title="t('agent.skills.importDir.title')" :width="480">
      <span class="ag-fld">
        <input
          v-model="dirPath"
          class="ag-mono"
          type="text"
          autocomplete="off"
          spellcheck="false"
          :placeholder="t('agent.skills.importDir.placeholder')"
          :aria-label="t('agent.skills.importDir.label')"
          @keydown.enter="onEnter($event, importDir)"
        />
      </span>
      <AgDirList v-model="dirPath" />
      <AgShareField v-if="all" v-model="importShare" />
      <div class="ag-sk-acts">
        <button type="button" class="ag-btn" @click="dirOpen = false">{{ t('agent.common.cancel') }}</button>
        <button type="button" class="ag-btn ag-btn--p" :disabled="importing" @click="importDir">
          <UIcon v-if="importing" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.skills.import') }}
        </button>
      </div>
    </AgDialog>

    <AgConfirm
      v-model="deleteOpen"
      danger
      :title="t('agent.skills.confirmDelete.title')"
      :message="t('agent.skills.confirmDelete.message', { name: deleteTarget?.name ?? '' })"
      :confirm-text="t('agent.common.delete')"
      :on-confirm="doDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  AGENT_SHARED_OWNER,
  type AgentShare,
  deleteAgentSkill,
  getAgentSkill,
  importAgentSkill,
  listAgentSkills,
  setAgentSkillEnabled,
} from '@/api/agent'
import type { AgentSkill, AgentSkillFile, AgentUser } from '@/types/v1/agent'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import AgDirList from '@/views/agent/components/AgDirList.vue'
import AgManageHeader from '@/views/agent/components/AgManageHeader.vue'
import AgShareField from '@/views/agent/components/AgShareField.vue'
import AgSwitch from '@/views/agent/components/AgSwitch.vue'
import { useAgentAll } from '@/views/agent/composables/useAgentAll'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'
import { fileToBase64 } from '@/views/agent/utils/format'
import SkillEditorDialog from './SkillEditorDialog.vue'

defineOptions({ name: 'AgentManageSkillsView' })

const { t, te } = useI18n()
/** 在 Agent 管理页里：所有人的技能 */
const all = useAgentAll()
const { overview, refresh } = useAgentOverview()

// —— 列表 ——
const skills = ref<AgentSkill[]>([])
const users = ref<AgentUser[]>([])
const loaded = ref(false)
const load = async () => {
  try {
    const { data } = await listAgentSkills({ all })
    skills.value = data.skills || []
    users.value = data.users || []
  } catch {
    // 请求层已经提示过
  } finally {
    loaded.value = true
  }
}
/** 数量变了：列表和标签上的数字一起刷新 */
const reloadAll = () => Promise.all([load(), refresh()])
onMounted(load)
// 标签页是 keep-alive 的：再切回来时悄悄刷新一次（会话里的 /install-skill 也会往这里加技能）
onActivated(() => {
  if (loaded.value) void load()
})

const enabledCount = computed(() => Number(overview.value?.skillsEnabled ?? 0))
const userName = (id: string) => {
  const u = users.value.find((x) => x.id === id)
  return u ? u.name || u.username : t('agent.admin.goneUser')
}
const ownerOf = (s: AgentSkill) => userName(s.userId)

// —— 共享的 ——
const isShared = (s: AgentSkill) => s.userId === AGENT_SHARED_OWNER
/** 共享的技能上的标记：Agent 管理页里写共享给谁，自己的列表里写「管理员共享」 */
const shareTag = (s: AgentSkill) => {
  if (!all) return t('agent.share.tagSharedToMe')
  return s.share === 'users' ? t('agent.share.tagUsers', { n: (s.shareUserIds || []).length }) : t('agent.share.tagAll')
}
const scopeOf = (s: AgentSkill) => {
  if (!all) return t('agent.share.sharedReadonly')
  return s.share === 'users' ? (s.shareUserIds || []).map(userName).join(t('agent.share.sep')) : ''
}
// Agent 管理页里分两组：共享的在前；自己的列表不分组
const groups = computed(() => {
  if (!all) return [{ key: 'mine', title: '', skills: skills.value, empty: t('agent.skills.empty') }]
  return [
    { key: 'shared', title: t('agent.share.sharedTitle'), skills: skills.value.filter(isShared), empty: t('agent.share.noShared') },
    {
      key: 'personal',
      title: t('agent.share.personalTitle'),
      skills: skills.value.filter((s) => !isShared(s)),
      empty: t('agent.skills.empty'),
    },
  ]
})
const sourceText = (source: string) => {
  const key = `agent.skills.source.${source}`
  return te(key) ? t(key) : source
}

// —— 启用开关 ——
const setEnabled = async (s: AgentSkill, on: boolean) => {
  s.enabled = on
  try {
    await setAgentSkillEnabled(s.id, on)
    void refresh()
  } catch {
    s.enabled = !on
  }
}

// —— 新建 / 编辑：Agent 管理页里新建的是共享的；自己的列表里共享的只能看 ——
const editorOpen = ref(false)
const editorDetail = ref<{ skill: AgentSkill; files: AgentSkillFile[] } | null>(null)
const editorShared = ref(false)
const opening = ref('')
const openCreate = () => {
  editorDetail.value = null
  editorShared.value = all
  editorOpen.value = true
}
const openEdit = async (s: AgentSkill) => {
  if (opening.value) return
  opening.value = s.id
  try {
    const { data } = await getAgentSkill(s.id)
    editorDetail.value = { skill: data.skill ?? s, files: data.files || [] }
    editorShared.value = isShared(s)
    editorOpen.value = true
  } catch {
    // 请求层已经提示过
  } finally {
    opening.value = ''
  }
}

// —— 导入 ——
type ImportKind = 'zip' | 'git' | 'dir'
const importKinds: ImportKind[] = ['zip', 'git', 'dir']
const ZIP_MAX_BYTES = 20 * 1024 * 1024

const importing = ref(false)
const zipInput = useTemplateRef<HTMLInputElement>('zipInput')
const zipOpen = ref(false)
const gitOpen = ref(false)
const gitUrl = ref('')
const dirOpen = ref(false)
const dirPath = ref('')
// Agent 管理页里导入成共享的：共享给谁
const importShare = ref<AgentShare>({ share: 'all', shareUserIds: [] })
const shareReady = () => {
  if (!all || importShare.value.share === 'all' || importShare.value.shareUserIds.length) return true
  feedback.warning(t('agent.share.needUsers'))
  return false
}
const chooseZip = () => {
  if (shareReady()) zipInput.value?.click()
}

const pickImport = (kind: ImportKind, close: () => void) => {
  close()
  importShare.value = { share: 'all', shareUserIds: [] }
  if (kind === 'zip') {
    if (all) zipOpen.value = true
    else zipInput.value?.click()
  } else if (kind === 'git') {
    gitUrl.value = ''
    gitOpen.value = true
  } else {
    dirPath.value = ''
    dirOpen.value = true
  }
}

/** 一个仓库或压缩包里可能有好几个技能；成功返回 true */
const runImport = async (req: Parameters<typeof importAgentSkill>[0]) => {
  if (importing.value || !shareReady()) return false
  importing.value = true
  try {
    const { data } = await importAgentSkill({ ...req, ...(all ? importShare.value : {}) })
    const names = (data.skills || []).map((s) => s.name)
    if (names.length) feedback.success(t('agent.skills.toast.imported', { names: names.join(t('agent.skills.sep')) }))
    else feedback.warning(t('agent.skills.toast.importedNone'))
    await reloadAll()
    return true
  } catch {
    // 请求层已经提示过
    return false
  } finally {
    importing.value = false
  }
}

const onZipPicked = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  // 清空，下次选同一个文件也会触发
  input.value = ''
  if (!file) return
  if (file.size > ZIP_MAX_BYTES) {
    feedback.warning(t('agent.skills.toast.zipTooLarge'))
    return
  }
  let data = ''
  try {
    data = await fileToBase64(file)
  } catch {
    feedback.error(t('agent.skills.toast.zipUnreadable'))
    return
  }
  if (await runImport({ source: 'zip', data })) zipOpen.value = false
}

const importGit = async () => {
  const location = gitUrl.value.trim()
  if (!location) {
    feedback.warning(t('agent.skills.toast.needGit'))
    return
  }
  if (await runImport({ source: 'git', location })) gitOpen.value = false
}

const importDir = async () => {
  const location = dirPath.value.trim()
  if (!location) {
    feedback.warning(t('agent.skills.toast.needDir'))
    return
  }
  if (await runImport({ source: 'dir', location })) dirOpen.value = false
}

// 输入法选字时的回车不算提交
const onEnter = (e: KeyboardEvent, submit: () => void) => {
  if (!e.isComposing) submit()
}

// —— 删除 ——
const deleteOpen = ref(false)
const deleteTarget = ref<AgentSkill | null>(null)
const askDelete = (s: AgentSkill) => {
  deleteTarget.value = s
  deleteOpen.value = true
}
const doDelete = async () => {
  const s = deleteTarget.value
  if (!s) return
  await deleteAgentSkill(s.id)
  feedback.success(t('agent.skills.toast.deleted', { name: s.name }))
  await reloadAll()
}
</script>

<style lang="scss">
.ag-sk-onn {
  color: var(--m-ok);
  font-weight: 600;
}
.ag-sk-meta {
  min-width: 0;
  font-size: 12px;
  font-weight: 400;
  color: var(--m-ink2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ag-sk-tg {
  margin-right: 6px;
}
.ag-sk-acts {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
}
@media (width < 992px) {
  .ag-sk-meta {
    white-space: normal;
  }
  // 卡片里：开关靠左，两个图标按钮靠右
  .ag-sk-tg {
    margin-right: auto;
  }
}
</style>
