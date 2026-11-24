<!-- Git 面板的「更改」：进行中的合并 / 变基的横条，提交框，然后是冲突的、已暂存的、没暂存的三组文件。
     点文件在下面看它的 diff，双击打开文件，右键（或行尾的 ⋯）有更多操作。 -->
<template>
  <div v-if="repo" class="agg-changes">
    <!-- 进行中的合并、变基、拣选、还原 -->
    <div v-if="repo.operation" class="agg-op" :class="{ 'is-ready': !repo.conflicted.length }">
      <div class="agg-op__t">
        <UIcon :name="repo.operation === 'rebase' ? 'i-lucide-git-pull-request-arrow' : 'i-lucide-git-merge'" />
        <span class="agc-flex">
          {{ t(`agent.git.op.${repo.operation}`, { ref: repo.operationRef || 'HEAD', onto: repo.operationOnto || '' }) }}
          <span v-if="repo.operationTotal" class="ag-muted ag-num">· {{ t('agent.git.op.step', { step: repo.operationStep, total: repo.operationTotal }) }}</span>
        </span>
      </div>
      <div class="agg-op__d">
        {{ repo.conflicted.length ? t('agent.git.op.conflicts', { n: repo.conflicted.length }) : t('agent.git.op.ready') }}
      </div>
      <div class="agg-op__acts">
        <button
          v-if="repo.operation !== 'bisect'"
          type="button"
          class="ag-btn ag-btn--sm ag-btn--p"
          :disabled="!!git.busy.value || repo.conflicted.length > 0"
          @click="sequence('continue')"
        >
          <UIcon :name="git.busy.value === 'op:continue' ? 'i-lucide-loader-circle' : 'i-lucide-play'" :class="{ 'ag-spin': git.busy.value === 'op:continue' }" />
          {{ t('agent.git.op.continue') }}
        </button>
        <button v-if="canSkip" type="button" class="ag-btn ag-btn--sm" :disabled="!!git.busy.value" @click="sequence('skip')">
          <UIcon name="i-lucide-skip-forward" />{{ t('agent.git.op.skip') }}
        </button>
        <button type="button" class="ag-btn ag-btn--sm ag-btn--stop" :disabled="!!git.busy.value" @click="askAbort">
          <UIcon name="i-lucide-circle-x" />{{ t('agent.git.op.abort') }}
        </button>
      </div>
    </div>

    <!-- 提交 -->
    <div class="agg-commit">
      <div class="agg-commit__box">
        <textarea
          ref="msgEl"
          v-model="git.message.value"
          class="agg-commit__ta"
          :rows="msgRows"
          :placeholder="repo.branch ? t('agent.git.commit.placeholderTo', { branch: repo.branch }) : t('agent.git.commit.placeholder')"
          :aria-label="t('agent.git.commit.placeholder')"
          @keydown.enter.ctrl.prevent="commit()"
          @keydown.enter.meta.prevent="commit()"
        />
        <button
          type="button"
          class="ag-ib ag-ib--sm agg-commit__ai"
          :aria-label="generating ? t('agent.git.commit.generating') : t('agent.git.commit.generate')"
          :title="generating ? t('agent.git.commit.generating') : t('agent.git.commit.generate')"
          :disabled="generating || (!hasChanges && !amend)"
          @click="generate"
        >
          <UIcon :name="generating ? 'i-lucide-loader-circle' : 'i-lucide-sparkles'" :class="{ 'ag-spin': generating }" />
        </button>
      </div>
      <div class="agg-commit__row">
        <label v-if="repo.head" class="agg-check" :title="t('agent.git.commit.amendHint')">
          <input v-model="amend" type="checkbox" class="ag-cb" />
          <span>{{ t('agent.git.commit.amend') }}</span>
        </label>
        <span class="agc-flex" />
        <div class="agg-split">
          <button type="button" class="ag-btn ag-btn--sm ag-btn--p agg-split__main" :disabled="!canCommit" :title="commitLabel" @click="commit()">
            <UIcon :name="git.busy.value === 'commit' ? 'i-lucide-loader-circle' : 'i-lucide-check'" :class="{ 'ag-spin': git.busy.value === 'commit' }" />
            <span class="ag-ellipsis">{{ commitLabel }}</span>
          </button>
          <button
            type="button"
            class="ag-btn ag-btn--sm ag-btn--p agg-split__more"
            :aria-label="t('agent.git.commit.options')"
            :title="t('agent.git.commit.options')"
            aria-haspopup="menu"
            @click="commitMenu"
          >
            <UIcon name="i-lucide-chevron-down" />
          </button>
        </div>
      </div>
    </div>

    <!-- 文件 -->
    <div class="agg-files ag-noscroll" :class="{ 'is-full': !git.diffShown.value }">
      <template v-for="s in sections" :key="s.key">
        <div v-if="s.files.length" class="agg-sec">
          <div class="agg-sec__head">
            <button type="button" class="agg-sec__toggle" :aria-expanded="!closed.has(s.key)" @click="toggle(s.key)">
              <UIcon :name="closed.has(s.key) ? 'i-lucide-chevron-right' : 'i-lucide-chevron-down'" class="agg-sec__chev" />
              <span class="agg-sec__t">{{ s.title }}</span>
              <span class="ag-tag ag-num">{{ s.files.length }}</span>
            </button>
            <span class="agg-sec__acts">
              <button
                v-for="a in s.headActions"
                :key="a.icon"
                type="button"
                class="ag-ib ag-ib--sm"
                :class="{ 'ag-ib--bad': a.danger }"
                :aria-label="a.label"
                :title="a.label"
                :disabled="!!git.busy.value"
                @click="a.run"
              >
                <UIcon :name="a.icon" />
              </button>
            </span>
          </div>
          <template v-if="!closed.has(s.key)">
            <div
              v-for="f in s.files"
              :key="f.path"
              class="agg-fr"
              :class="{ 'is-on': isOn(s.source, f) }"
              role="button"
              tabindex="0"
              :title="fileTitle(s.source, f)"
              @click="select(s.source, f)"
              @keydown.enter="select(s.source, f)"
              @dblclick="open(f)"
              @contextmenu.prevent="rowMenu($event, s.source, f)"
            >
              <span class="ag-mono agg-fr__k" :class="kindCls(s.source, f)">{{ kindLetter(s.source, f) }}</span>
              <span class="agg-fr__n ag-ellipsis">{{ baseName(f.path) }}</span>
              <span class="agg-fr__d ag-ellipsis ag-mono">{{ dirName(f.path) }}</span>
              <span class="agc-flex" />
              <span v-if="f.binary" class="ag-muted agc-small">bin</span>
              <template v-else-if="s.source !== 'conflict'">
                <span v-if="f.added" class="agc-plus ag-num">+{{ f.added }}</span>
                <span v-if="f.deleted" class="agc-minus ag-num">−{{ f.deleted }}</span>
              </template>
              <span class="agg-fr__acts" @click.stop @dblclick.stop>
                <button
                  v-for="a in rowActions(s.source, f)"
                  :key="a.icon"
                  type="button"
                  class="ag-ib ag-ib--sm"
                  :aria-label="a.label"
                  :title="a.label"
                  :disabled="!!git.busy.value"
                  @click="a.run"
                >
                  <UIcon :name="a.icon" />
                </button>
                <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.git.files.actions', { name: baseName(f.path) })" @click="rowMenu($event, s.source, f)">
                  <UIcon name="i-lucide-ellipsis" />
                </button>
              </span>
            </div>
          </template>
        </div>
      </template>
      <div v-if="!hasChanges && !repo.conflicted.length" class="agg-empty">
        <UIcon name="i-lucide-circle-check" />{{ t('agent.git.files.none') }}
      </div>
      <div v-if="repo.truncated" class="agc-right__note">{{ t('agent.git.files.truncated') }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  commitAgentGit,
  generateAgentGitMessage,
  resetAgentGit,
  runAgentGitOperation,
  syncAgentGit,
  updateAgentGitFiles,
  updateAgentGitStash,
  type AgentGitFileAction,
} from '@/api/agent'
import type { AgentGitFile } from '@/types/v1/agent'
import { baseName, copyText, dirName, repoPath, useGit, type GitMenuItem } from './useGit'

type Source = 'conflict' | 'staged' | 'unstaged'

const { t } = useI18n()
const git = useGit()
const repo = computed(() => git.repo.value)

const hasStaged = computed(() => (repo.value?.staged.length ?? 0) > 0)
const hasUnstaged = computed(() => (repo.value?.unstaged.length ?? 0) > 0)
const hasChanges = computed(() => hasStaged.value || hasUnstaged.value)
const canSkip = computed(() => ['rebase', 'am', 'cherry-pick', 'revert'].includes(repo.value?.operation || ''))

// —— 文件 ——
const closed = reactive(new Set<string>())
const toggle = (key: string) => (closed.has(key) ? closed.delete(key) : closed.add(key))

const files = (action: AgentGitFileAction, paths: string[], all = false, ok?: string) => {
  const r = repo.value
  if (!r) return
  void git.run(`files:${action}`, () => updateAgentGitFiles(git.sessionId(), { repo: r.repo, action, paths, all }), ok)
}
/** 暂存全部：有冲突时只暂存列出来的改动，冲突的文件要一个个标记已解决（别让 git add -A 把冲突标记一起当成已解决） */
const stageAll = () => {
  const r = repo.value
  if (!r) return
  if (r.conflicted.length) files('stage', pathsOf(r.unstaged))
  else files('stage', [], true)
}
const pathsOf = (list: AgentGitFile[]) => list.map((f) => f.path)
/** 取消暂存时改名的要连旧路径一起给 */
const unstagePaths = (list: AgentGitFile[]) => list.flatMap((f) => (f.origPath && f.origPath !== f.path ? [f.origPath, f.path] : [f.path]))
const askDiscard = (f?: AgentGitFile) => {
  git.ask({
    title: f ? t('agent.git.files.discardTitle', { name: baseName(f.path) }) : t('agent.git.files.discardAllTitle'),
    message: f ? t('agent.git.files.discardMsg') : t('agent.git.files.discardAllMsg'),
    confirmText: f ? t('agent.git.files.discard') : t('agent.git.files.discardAll'),
    danger: true,
    run: () => files('discard', f ? [f.path] : [], !f, t('agent.git.files.discarded')),
  })
}

const sections = computed(() => {
  const r = repo.value
  if (!r) return []
  return [
    {
      key: 'conflict',
      source: 'conflict' as const,
      title: t('agent.git.files.conflicts'),
      files: r.conflicted,
      headActions: [],
    },
    {
      key: 'staged',
      source: 'staged' as const,
      title: t('agent.git.files.staged'),
      files: r.staged,
      headActions: [{ icon: 'i-lucide-minus', label: t('agent.git.files.unstageAll'), danger: false, run: () => files('unstage', [], true) }],
    },
    {
      key: 'unstaged',
      source: 'unstaged' as const,
      title: t('agent.git.files.unstaged'),
      files: r.unstaged,
      headActions: [
        { icon: 'i-lucide-undo-2', label: t('agent.git.files.discardAll'), danger: true, run: () => askDiscard() },
        { icon: 'i-lucide-plus', label: t('agent.git.files.stageAll'), danger: false, run: stageAll },
      ],
    },
  ]
})

const KIND_CLS: Record<string, string> = { A: 'agc-ok', U: 'agc-ok', D: 'agc-bad', R: 'agg-k--r', C: 'agg-k--r' }
const kindLetter = (s: Source, f: AgentGitFile) => (s === 'conflict' ? '!' : f.kind)
const kindCls = (s: Source, f: AgentGitFile) => (s === 'conflict' ? 'agc-bad' : (KIND_CLS[f.kind] ?? 'agc-warn'))
const fileTitle = (s: Source, f: AgentGitFile) => {
  const kind = s === 'conflict' ? t(`agent.git.files.conflictKinds.${f.kind}`) : t(`agent.git.files.kinds.${f.kind}`)
  const from = f.origPath && f.origPath !== f.path ? ` · ${t('agent.git.files.renamedFrom', { path: f.origPath })}` : ''
  return `${f.path} · ${kind}${from}`
}
const isOn = (s: Source, f: AgentGitFile) => git.diffShown.value?.source === s && git.diffShown.value.path === f.path
const select = (s: Source, f: AgentGitFile) => {
  git.diff.value = { repo: repo.value?.repo ?? '', source: s, path: f.path, origPath: f.origPath || undefined, kind: f.kind, binary: f.binary }
}
const sessionPath = (f: AgentGitFile) => repoPath(repo.value?.repo ?? '', f.path)
const open = (f: AgentGitFile) => {
  if (f.kind !== 'D') git.openFile(sessionPath(f))
}

const rowActions = (s: Source, f: AgentGitFile) => {
  const list: { icon: string; label: string; run: () => void }[] = []
  if (f.kind !== 'D' && !(s === 'conflict' && f.kind.includes('D'))) list.push({ icon: 'i-lucide-file-symlink', label: t('agent.git.files.open'), run: () => open(f) })
  if (s === 'conflict') list.push({ icon: 'i-lucide-check', label: t('agent.git.files.resolve'), run: () => files('resolve', [f.path], false, t('agent.git.files.resolved')) })
  if (s === 'staged') list.push({ icon: 'i-lucide-minus', label: t('agent.git.files.unstage'), run: () => files('unstage', unstagePaths([f])) })
  if (s === 'unstaged') {
    list.push({ icon: 'i-lucide-undo-2', label: t('agent.git.files.discard'), run: () => askDiscard(f) })
    list.push({ icon: 'i-lucide-plus', label: t('agent.git.files.stage'), run: () => files('stage', [f.path]) })
  }
  return list
}
const rowMenu = (e: MouseEvent, s: Source, f: AgentGitFile) => {
  const r = repo.value
  if (!r) return
  const items: GitMenuItem[] = []
  if (f.kind !== 'D') items.push({ label: t('agent.git.files.open'), icon: 'i-lucide-file-symlink', run: () => open(f) })
  items.push({ sep: true })
  if (s === 'conflict') {
    items.push(
      { label: t('agent.git.files.ours'), icon: 'i-lucide-arrow-left-to-line', run: () => files('ours', [f.path], false, t('agent.git.files.resolved')) },
      { label: t('agent.git.files.theirs'), icon: 'i-lucide-arrow-right-to-line', run: () => files('theirs', [f.path], false, t('agent.git.files.resolved')) },
      { label: t('agent.git.files.resolve'), icon: 'i-lucide-check', run: () => files('resolve', [f.path], false, t('agent.git.files.resolved')) },
    )
  } else if (s === 'staged') {
    items.push({ label: t('agent.git.files.unstage'), icon: 'i-lucide-minus', run: () => files('unstage', unstagePaths([f])) })
  } else {
    items.push(
      { label: t('agent.git.files.stage'), icon: 'i-lucide-plus', run: () => files('stage', [f.path]) },
      { label: t('agent.git.files.discard'), icon: 'i-lucide-undo-2', danger: true, run: () => askDiscard(f) },
    )
  }
  if (s !== 'conflict' && f.kind !== 'U') {
    items.push({
      label: t('agent.git.files.stash'),
      icon: 'i-lucide-archive',
      run: () =>
        git.run('stash:push', () => updateAgentGitStash(git.sessionId(), { repo: r.repo, action: 'push', paths: [f.path] }), t('agent.git.stash.pushed')),
    })
  }
  if (f.kind === 'U') items.push({ label: t('agent.git.files.ignore'), icon: 'i-lucide-eye-off', run: () => files('ignore', [f.path], false, t('agent.git.files.ignored')) })
  items.push({ sep: true })
  if (f.kind !== 'U' && f.kind !== 'A') {
    items.push({
      label: t('agent.git.files.history'),
      icon: 'i-lucide-history',
      run: () => {
        git.historyPath.value = f.path
        git.view.value = 'history'
      },
    })
  }
  if (f.kind !== 'D') items.push({ label: t('agent.git.files.reference'), icon: 'i-lucide-at-sign', run: () => git.reference(sessionPath(f)) })
  items.push({ label: t('agent.git.files.copyPath'), icon: 'i-lucide-copy', run: () => copyText(sessionPath(f)) })
  git.openMenu(e, items)
}

// —— 进行中的操作 ——
const sequence = (action: 'continue' | 'skip' | 'abort') => {
  const r = repo.value
  if (!r) return
  const ok = { continue: t('agent.git.op.continued'), skip: t('agent.git.op.skipped'), abort: t('agent.git.op.aborted') }[action]
  return git.run(`op:${action}`, () => runAgentGitOperation(git.sessionId(), { repo: r.repo, action }), ok)
}
const askAbort = () => {
  git.ask({ title: t('agent.git.op.abortTitle'), message: t('agent.git.op.abortMsg'), confirmText: t('agent.git.op.abort'), danger: true, run: () => sequence('abort') })
}

// —— 提交 ——
const msgEl = ref<HTMLTextAreaElement | null>(null)
const msgRows = computed(() => Math.min(8, Math.max(2, git.message.value.split('\n').length)))
const amend = ref(false)
// 勾上「修改上一次提交」时输入框是空的：放上上一次的说明；取消时说明没动过就清掉
watch(amend, (on) => {
  const last = repo.value?.lastMessage || ''
  if (on && !git.message.value.trim()) git.message.value = last
  else if (!on && git.message.value === last) git.message.value = ''
})
watch(
  () => [git.sessionId(), repo.value?.repo],
  () => {
    amend.value = false
  },
)
/** 没暂存任何东西但有改动时：提交就是全部暂存再提交 */
const commitAll = computed(() => !hasStaged.value && hasUnstaged.value && !amend.value)
const commitLabel = computed(() => (amend.value ? t('agent.git.commit.amend') : commitAll.value ? t('agent.git.commit.commitAll') : t('agent.git.commit.commit')))
const canCommit = computed(() => {
  if (git.busy.value || !repo.value || repo.value.conflicted.length) return false
  if (amend.value) return true
  return hasChanges.value && !!git.message.value.trim()
})

const commit = async (extra: { noVerify?: boolean; signoff?: boolean; then?: 'push' | 'sync' } = {}) => {
  const r = repo.value
  if (!r) return
  const message = git.message.value.trim()
  if (!message && !amend.value) {
    feedback.warning(hasChanges.value ? t('agent.git.commit.needMessage') : t('agent.git.commit.nothing'))
    msgEl.value?.focus()
    return
  }
  if (!canCommit.value) {
    if (!hasChanges.value && !amend.value) feedback.warning(t('agent.git.commit.nothing'))
    return
  }
  const res = await git.run(
    'commit',
    () =>
      commitAgentGit(git.sessionId(), {
        repo: r.repo,
        message,
        amend: amend.value,
        all: commitAll.value,
        noVerify: extra.noVerify,
        signoff: extra.signoff,
      }),
    (res) => t('agent.git.commit.done', { head: res.data.head }),
  )
  if (!res) return
  git.message.value = ''
  amend.value = false
  if (extra.then === 'sync' && (repo.value?.behind ?? 0) > 0) {
    const pulled = await git.run('sync:pull', () => syncAgentGit(git.sessionId(), { repo: r.repo, action: 'pull' }))
    if (!pulled) return
  }
  if (extra.then) {
    await git.run('sync:push', () => syncAgentGit(git.sessionId(), { repo: r.repo, action: 'push' }), t(extra.then === 'sync' ? 'agent.git.sync.synced' : 'agent.git.sync.pushed'))
  }
}

const commitMenu = (e: MouseEvent) => {
  const r = repo.value
  if (!r) return
  const off = !canCommit.value
  const remote = r.remotes.length > 0
  const items: GitMenuItem[] = [
    { label: t('agent.git.commit.commitPush'), icon: 'i-lucide-upload', disabled: off || !remote, run: () => commit({ then: 'push' }) },
    { label: t('agent.git.commit.commitSync'), icon: 'i-lucide-refresh-cw', disabled: off || !remote || !r.upstream, run: () => commit({ then: 'sync' }) },
    { sep: true },
    { label: t('agent.git.commit.noVerify'), icon: 'i-lucide-shield-off', disabled: off, run: () => commit({ noVerify: true }) },
    { label: t('agent.git.commit.signoff'), icon: 'i-lucide-signature', disabled: off, run: () => commit({ signoff: true }) },
    { sep: true },
    {
      label: t('agent.git.commit.undo'),
      icon: 'i-lucide-undo',
      disabled: !r.head || !!git.busy.value,
      run: () =>
        git.ask({
          title: t('agent.git.commit.undoTitle'),
          message: t('agent.git.commit.undoMsg'),
          confirmText: t('agent.git.commit.undo'),
          run: async () => {
            const done = await git.run(
              'reset',
              () => resetAgentGit(git.sessionId(), { repo: r.repo, target: 'HEAD~1', mode: 'soft' }),
              t('agent.git.commit.undone'),
            )
            // 撤销掉的那次说明放回输入框，改改再提交
            if (done && !git.message.value.trim()) git.message.value = r.lastMessage
          },
        }),
    },
  ]
  git.openMenu(e, items)
}

// —— 让 Agent 写提交说明 ——
const generating = ref(false)
const generate = async () => {
  const r = repo.value
  if (!r || generating.value) return
  generating.value = true
  try {
    const { data } = await generateAgentGitMessage(git.sessionId(), r.repo)
    if (data.message) {
      git.message.value = data.message
      feedback.success(t('agent.git.commit.generated'))
      void nextTick(() => msgEl.value?.focus())
    }
  } catch {
    /* shown */
  } finally {
    generating.value = false
  }
}
</script>
