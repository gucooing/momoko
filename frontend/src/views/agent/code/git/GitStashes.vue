<!-- Git 面板的「储藏」：储藏现在的改动；列出储藏，点一个在中间打开（里面的文件和改动），应用、应用并删除、删除。 -->
<template>
  <div class="agg-stashes">
    <div class="agg-hist__bar">
      <button type="button" class="ag-btn ag-btn--sm ag-btn--p" :disabled="!!git.busy.value || !canPush" @click="push">
        <UIcon name="i-lucide-archive" />{{ t('agent.git.stash.push') }}
      </button>
    </div>
    <div class="agg-hist__list ag-noscroll">
      <div v-for="s in stashes" :key="s.hash" class="agg-cm" :class="{ 'is-open': git.opened.value === s.hash }">
        <div
          class="agg-cm__row"
          role="button"
          tabindex="0"
          :title="t('agent.git.stash.open')"
          @click="open(s)"
          @keydown.enter="open(s)"
          @contextmenu.prevent="stashMenu($event, s)"
        >
          <UIcon name="i-lucide-archive" class="agg-cm__icon" />
          <div class="agg-cm__main">
            <div class="agg-cm__subj"><span class="ag-ellipsis">{{ s.message || stashRef(s) }}</span></div>
            <div class="agg-cm__meta">
              <span class="ag-mono">{{ stashRef(s) }}</span>
              <template v-if="s.branch">
                <span>·</span>
                <span class="ag-mono ag-ellipsis">{{ s.branch }}</span>
              </template>
              <span>·</span>
              <span class="agg-nowrap">{{ relTime(s.date) }}</span>
            </div>
          </div>
          <span class="agg-cm__acts" @click.stop>
            <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.git.stash.pop')" :title="t('agent.git.stash.pop')" :disabled="!!git.busy.value" @click="apply(s, true)">
              <UIcon name="i-lucide-archive-restore" />
            </button>
            <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.git.stash.actions')" @click="stashMenu($event, s)">
              <UIcon name="i-lucide-ellipsis" />
            </button>
          </span>
        </div>
      </div>
      <div v-if="loaded && !stashes.length" class="agg-empty"><UIcon name="i-lucide-archive" />{{ t('agent.git.stash.none') }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { listAgentGitStashes, updateAgentGitStash } from '@/api/agent'
import type { AgentGitStash } from '@/types/v1/agent'
import { relTime } from '@/views/agent/utils/format'
import { useGit, type GitMenuItem } from './useGit'

const { t } = useI18n()
const git = useGit()

/** stash@{n}（直接写在模板里，花括号会被当成插值） */
const stashRef = (s: AgentGitStash) => `stash@{${s.index}}`
const stashes = ref<AgentGitStash[]>([])
const loaded = ref(false)
const canPush = computed(() => {
  const r = git.repo.value
  return !!r?.head && (r.staged.length > 0 || r.unstaged.length > 0)
})

let seq = 0
const load = async () => {
  const r = git.repo.value
  if (!r) return
  const id = ++seq
  try {
    const { data } = await listAgentGitStashes(git.sessionId(), r.repo)
    if (id === seq) stashes.value = data.stashes ?? []
  } catch {
    if (id === seq) stashes.value = []
  } finally {
    if (id === seq) loaded.value = true
  }
}
watch(
  [() => git.repo.value?.repo, () => git.rev.value, () => git.repo.value?.stashes],
  () => void load(),
  { immediate: true },
)

/** 在中间打开这个储藏 */
const open = (s: AgentGitStash) => {
  const r = git.repo.value
  if (!r) return
  git.openCommit({ repo: r.repo, hash: s.hash, short: stashRef(s), subject: s.message || stashRef(s), stash: true })
}

const push = () => {
  const r = git.repo.value
  if (!r) return
  git.prompt({
    title: t('agent.git.stash.pushTitle'),
    fields: [{ key: 'message', label: t('agent.git.stash.message') }],
    checks: [
      { key: 'untracked', label: t('agent.git.stash.untracked'), value: r.unstaged.some((f) => f.kind === 'U') },
      { key: 'staged', label: t('agent.git.stash.staged'), value: false },
    ],
    confirmText: t('agent.git.stash.push'),
    run: (v, ck) =>
      git.run(
        'stash:push',
        () => updateAgentGitStash(git.sessionId(), { repo: r.repo, action: 'push', message: v.message, untracked: ck.untracked, staged: ck.staged }),
        t('agent.git.stash.pushed'),
      ),
  })
}
const apply = (s: AgentGitStash, pop: boolean, index = false) => {
  const r = git.repo.value
  if (!r) return
  void git.run(
    `stash:${pop ? 'pop' : 'apply'}`,
    () => updateAgentGitStash(git.sessionId(), { repo: r.repo, action: pop ? 'pop' : 'apply', hash: s.hash, index }),
    t(pop ? 'agent.git.stash.popped' : 'agent.git.stash.applied'),
  )
}
const stashMenu = (e: MouseEvent, s: AgentGitStash) => {
  const r = git.repo.value
  if (!r) return
  const items: GitMenuItem[] = [
    { label: t('agent.git.stash.apply'), icon: 'i-lucide-archive-restore', run: () => apply(s, false) },
    { label: t('agent.git.stash.pop'), icon: 'i-lucide-package-open', run: () => apply(s, true) },
    { label: t('agent.git.stash.index'), icon: 'i-lucide-layers', run: () => apply(s, false, true) },
    { sep: true },
    {
      label: t('agent.git.stash.drop'),
      icon: 'i-lucide-trash-2',
      danger: true,
      run: () =>
        git.ask({
          title: t('agent.git.stash.dropTitle', { name: s.message || stashRef(s) }),
          message: t('agent.git.stash.dropMsg'),
          confirmText: t('agent.git.stash.drop'),
          danger: true,
          run: () => git.run('stash:drop', () => updateAgentGitStash(git.sessionId(), { repo: r.repo, action: 'drop', hash: s.hash }), t('agent.git.stash.dropped')),
        }),
    },
  ]
  git.openMenu(e, items)
}
</script>
