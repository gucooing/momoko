<!-- 右侧「Git」标签：顶上是仓库（会话目录下有几个仓库时能换）、分支、同步按钮和更多操作，下面分「更改 / 历史 / 储藏」三个视图，
     选中的文件的 diff 放在最下面。确认框、输入框、右键菜单整个面板共用一份（在 useGit 里）。 -->
<template>
  <div class="agg">
    <div v-if="!git.loaded.value" class="agc-right__note">{{ t('agent.git.loading') }}</div>
    <div v-else-if="!git.repos.value.length" class="agg-none">
      <UIcon name="i-lucide-git-branch" class="agg-none__icon" />
      <div>{{ t('agent.git.noRepo') }}</div>
      <button type="button" class="ag-btn ag-btn--sm ag-btn--p" :disabled="!!git.busy.value" @click="init">
        <UIcon name="i-lucide-git-branch-plus" />{{ t('agent.git.init') }}
      </button>
    </div>
    <template v-else-if="repo">
      <div class="agg-head">
        <button
          v-if="git.repos.value.length > 1 && !reposOpen"
          type="button"
          class="agg-chip"
          :title="t('agent.git.pickRepo')"
          aria-haspopup="menu"
          @click="repoMenu"
        >
          <UIcon name="i-lucide-folder-git-2" />
          <span class="ag-mono ag-ellipsis">{{ repo.repo || t('agent.git.root') }}</span>
          <UIcon name="i-lucide-chevron-down" class="agg-chip__chev" />
        </button>
        <button
          ref="branchBtn"
          type="button"
          class="agg-chip"
          :class="{ 'is-on': branchOpen }"
          :title="t('agent.git.branch.switch')"
          aria-haspopup="dialog"
          :aria-expanded="branchOpen"
          @click="branchOpen = !branchOpen"
        >
          <UIcon :name="repo.branch ? 'i-lucide-git-branch' : 'i-lucide-git-commit-horizontal'" />
          <span class="ag-mono ag-ellipsis">{{ headLabel }}</span>
          <UIcon name="i-lucide-chevron-down" class="agg-chip__chev" />
        </button>
        <button v-if="sync" type="button" class="agg-chip agg-sync" :title="sync.title" :disabled="!!git.busy.value" @click="doSync">
          <UIcon
            :name="git.busy.value.startsWith('sync:') ? 'i-lucide-loader-circle' : sync.icon"
            :class="{ 'ag-spin': git.busy.value.startsWith('sync:') }"
          />
          <template v-if="sync.kind === 'sync'">
            <span class="ag-num">↓{{ repo.behind }}</span>
            <span class="ag-num">↑{{ repo.ahead }}</span>
          </template>
          <span v-else>{{ sync.label }}</span>
        </button>
        <span class="agc-flex" />
        <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.git.refresh')" :title="t('agent.git.refresh')" @click="git.load(true)">
          <UIcon name="i-lucide-refresh-cw" :class="{ 'ag-spin': git.loading.value }" />
        </button>
        <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.git.more')" :title="t('agent.git.more')" aria-haspopup="menu" @click="moreMenu">
          <UIcon name="i-lucide-ellipsis" />
        </button>
      </div>

      <!-- 会话目录下有几个仓库：列出来，每个看得到分支、领先落后、进行中的操作和改了几个文件，点哪个就操作哪个 -->
      <div v-if="git.repos.value.length > 1" class="agg-repos">
        <button type="button" class="agg-repos__head" :aria-expanded="reposOpen" @click="reposOpen = !reposOpen">
          <UIcon :name="reposOpen ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="ag-muted" />
          <span class="agg-repos__t">{{ t('agent.git.repos') }}</span>
          <span class="ag-tag ag-num">{{ git.repos.value.length }}</span>
          <span v-if="!reposOpen && otherChanged" class="ag-muted agc-small agc-flex">{{ t('agent.git.reposOther', { n: otherChanged }) }}</span>
        </button>
        <div v-if="reposOpen" class="agg-repos__list ag-noscroll" role="listbox" :aria-label="t('agent.git.pickRepo')">
          <button
            v-for="r in git.repos.value"
            :key="r.repo"
            type="button"
            class="agg-repo"
            :class="{ 'is-on': r.repo === repo.repo }"
            role="option"
            :aria-selected="r.repo === repo.repo"
            :title="r.repo || t('agent.git.root')"
            @click="pickRepo(r.repo)"
          >
            <UIcon name="i-lucide-folder-git-2" class="agg-repo__icon" />
            <span class="ag-mono ag-ellipsis agg-repo__n">{{ r.repo || t('agent.git.root') }}</span>
            <span class="ag-mono ag-ellipsis agg-repo__br">{{ r.branch || r.head }}</span>
            <span class="agc-flex" />
            <span v-if="r.operation" class="ag-tag ag-tag--warn">{{ t(`agent.git.repoOp.${r.operation}`) }}</span>
            <span v-if="r.behind" class="ag-num agg-repo__ab">↓{{ r.behind }}</span>
            <span v-if="r.ahead" class="ag-num agg-repo__ab">↑{{ r.ahead }}</span>
            <span v-if="git.changedCount(r)" class="ag-tag ag-num" :title="t('agent.git.repoChanges', { n: git.changedCount(r) })">{{ git.changedCount(r) }}</span>
          </button>
        </div>
      </div>

      <div class="ag-seg agg-views" role="tablist">
        <button
          v-for="v in views"
          :key="v.key"
          type="button"
          role="tab"
          :class="{ 'is-on': git.view.value === v.key }"
          :aria-selected="git.view.value === v.key"
          @click="git.view.value = v.key"
        >
          {{ v.label }}<span v-if="v.n" class="ag-num agg-views__n">{{ v.n }}</span>
        </button>
      </div>

      <div class="agg-body" :class="{ 'has-diff': !!git.diffShown.value }">
        <GitChanges v-if="git.view.value === 'changes'" />
        <GitHistory v-else-if="git.view.value === 'history'" />
        <GitStashes v-else />
        <GitDiffPane @open-file="(p) => git.openFile(repoPath(repo!.repo, p))" />
      </div>
    </template>

    <GitBranchMenu v-if="branchOpen && repo" :anchor="branchBtn" @close="branchOpen = false" />
    <GitMenu />
    <GitPrompt />
    <GitRemotes v-model="remotesOpen" />
    <AgConfirm
      :model-value="!!ask"
      :title="ask?.title || ''"
      :message="ask?.message"
      :confirm-text="ask?.confirmText"
      :danger="ask?.danger"
      :on-confirm="confirmAsk"
      @update:model-value="(v) => !v && (git.askState.value = null)"
    >
      <label v-if="ask?.check" class="agg-check agg-check--dlg">
        <input v-model="askChecked" type="checkbox" class="ag-cb" />
        <span>{{ ask.check }}</span>
      </label>
    </AgConfirm>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getAgentGitDiff, initAgentGitRepo, syncAgentGit } from '@/api/agent'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import GitBranchMenu from './GitBranchMenu.vue'
import GitChanges from './GitChanges.vue'
import GitDiffPane from './GitDiffPane.vue'
import GitHistory from './GitHistory.vue'
import GitMenu from './GitMenu.vue'
import GitPrompt from './GitPrompt.vue'
import GitRemotes from './GitRemotes.vue'
import GitStashes from './GitStashes.vue'
import { repoPath, useGit, type GitMenuItem, type GitView } from './useGit'

const { t } = useI18n()
const git = useGit()
const repo = computed(() => git.repo.value)

const headLabel = computed(() => {
  const r = repo.value
  if (!r) return ''
  if (!r.branch) return t('agent.git.head.detached', { head: r.head || 'HEAD' })
  if (!r.head) return t('agent.git.head.unborn', { branch: r.branch })
  return r.branch
})

const views = computed<{ key: GitView; label: string; n: number }[]>(() => [
  { key: 'changes', label: t('agent.git.views.changes'), n: git.changedCount(repo.value) },
  { key: 'history', label: t('agent.git.views.history'), n: 0 },
  { key: 'stashes', label: t('agent.git.views.stashes'), n: repo.value?.stashes ?? 0 },
])

// —— 仓库（会话目录下有几个仓库时） ——
/** 仓库列表展开着；仓库多了默认收起来，只用顶上的下拉换 */
const reposOpen = ref(true)
watch(
  () => git.repos.value.length,
  (n, old) => {
    if (!old && n) reposOpen.value = n <= 8
  },
  { immediate: true },
)
/** 别的仓库里有改动的文件数（列表收起时提示一下） */
const otherChanged = computed(() => git.repos.value.filter((r) => r.repo !== repo.value?.repo).reduce((n, r) => n + git.changedCount(r), 0))
const pickRepo = (key: string) => {
  if (key === repo.value?.repo) return
  git.repoKey.value = key
  git.historyPath.value = ''
}
const repoMenu = (e: MouseEvent) => {
  git.openMenu(
    e,
    git.repos.value.map((r) => ({
      label: `${r.repo || t('agent.git.root')}${r.branch ? ` · ${r.branch}` : ''}`,
      icon: r.repo === repo.value?.repo ? 'i-lucide-check' : 'i-lucide-folder-git-2',
      hint: git.changedCount(r) ? String(git.changedCount(r)) : '',
      run: () => pickRepo(r.repo),
    })),
  )
}
const init = () => git.run('init', () => initAgentGitRepo(git.sessionId()), t('agent.git.initDone'))

// —— 分支 ——
const branchBtn = ref<HTMLElement | null>(null)
const branchOpen = ref(false)
watch(() => repo.value?.repo, () => (branchOpen.value = false))

// —— 同步 ——
const sync = computed(() => {
  const r = repo.value
  if (!r || !r.remotes.length || !r.branch || !r.head) return null
  if (!r.upstream || r.upstreamGone) return { kind: 'publish', icon: 'i-lucide-cloud-upload', label: t('agent.git.sync.publish'), title: t('agent.git.sync.publishTip') }
  return {
    kind: 'sync',
    icon: 'i-lucide-refresh-cw',
    label: t('agent.git.sync.sync'),
    title: r.ahead || r.behind ? t('agent.git.sync.syncTip', { ahead: r.ahead, behind: r.behind }) : t('agent.git.sync.upToDate'),
  }
})
const syncOp = (action: 'fetch' | 'pull' | 'push', extra: { mode?: 'rebase'; force?: boolean; tags?: boolean } = {}, ok?: string) => {
  const r = repo.value
  if (!r) return Promise.resolve(undefined)
  return git.run(`sync:${action}`, () => syncAgentGit(git.sessionId(), { repo: r.repo, action, ...extra }), ok)
}
const doSync = async () => {
  const r = repo.value
  if (!r || !sync.value) return
  if (sync.value.kind === 'publish') {
    await syncOp('push', {}, t('agent.git.sync.pushed'))
    return
  }
  if (!r.ahead && !r.behind) {
    await syncOp('fetch', {}, t('agent.git.sync.fetched'))
    return
  }
  if (r.behind && !(await syncOp('pull'))) return
  if ((repo.value?.ahead ?? 0) > 0) await syncOp('push', {}, t('agent.git.sync.synced'))
  else feedback.success(t('agent.git.sync.pulled'))
}

// —— 更多 ——
const remotesOpen = ref(false)
const copyPatch = async () => {
  const r = repo.value
  if (!r) return
  try {
    const { data } = await getAgentGitDiff(git.sessionId(), { repo: r.repo, source: 'head' })
    if (data.tooLarge) {
      feedback.warning(t('agent.git.tooLarge'))
      return
    }
    await navigator.clipboard.writeText(data.patch)
    feedback.success(t('agent.git.copied'))
  } catch {
    /* shown */
  }
}
const moreMenu = (e: MouseEvent) => {
  const r = repo.value
  if (!r) return
  const noRemote = !r.remotes.length
  const detached = !r.branch
  const items: GitMenuItem[] = [
    { label: t('agent.git.sync.fetch'), icon: 'i-lucide-cloud-download', disabled: noRemote, run: () => syncOp('fetch', {}, t('agent.git.sync.fetched')) },
    { label: t('agent.git.sync.pull'), icon: 'i-lucide-arrow-down-to-line', disabled: noRemote || detached, run: () => syncOp('pull', {}, t('agent.git.sync.pulled')) },
    {
      label: t('agent.git.sync.pullRebase'),
      icon: 'i-lucide-git-pull-request-arrow',
      disabled: noRemote || detached,
      run: () => syncOp('pull', { mode: 'rebase' }, t('agent.git.sync.pulled')),
    },
    { label: t('agent.git.sync.push'), icon: 'i-lucide-arrow-up-from-line', disabled: noRemote || detached || !r.head, run: () => syncOp('push', {}, t('agent.git.sync.pushed')) },
    {
      label: t('agent.git.sync.pushTags'),
      icon: 'i-lucide-tags',
      disabled: noRemote || detached || !r.head,
      run: () => syncOp('push', { tags: true }, t('agent.git.sync.pushed')),
    },
    {
      label: t('agent.git.sync.forcePush'),
      icon: 'i-lucide-triangle-alert',
      danger: true,
      disabled: noRemote || detached || !r.head,
      run: () =>
        git.ask({
          title: t('agent.git.sync.forcePushTitle'),
          message: t('agent.git.sync.forcePushMsg'),
          confirmText: t('agent.git.sync.forcePush'),
          danger: true,
          run: () => syncOp('push', { force: true }, t('agent.git.sync.pushed')),
        }),
    },
    { sep: true },
    { label: t('agent.git.remote.manage'), icon: 'i-lucide-cloud-cog', run: () => (remotesOpen.value = true) },
    { label: t('agent.git.copyPatch'), icon: 'i-lucide-copy', disabled: !git.changedCount(r), run: copyPatch },
  ]
  git.openMenu(e, items)
}

// —— 确认框 ——
const ask = computed(() => git.askState.value)
const askChecked = ref(false)
watch(ask, (a) => {
  askChecked.value = !!a?.checked
})
const confirmAsk = async () => {
  const a = ask.value
  if (!a) return
  await a.run(askChecked.value)
}
</script>
