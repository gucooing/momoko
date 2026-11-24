<!-- Git 面板的「历史」：提交列表（搜说明、当前分支 / 所有分支、只看某个文件）。点一个提交在中间打开它（说明、改了哪些文件、
     每个文件的 diff）。行尾 ⋯ / 右键：复制、切换过去、新建分支、打标签、拣选、还原、重置。 -->
<template>
  <div class="agg-hist">
    <div class="agg-hist__bar">
      <label class="ag-inp agg-hist__search">
        <UIcon name="i-lucide-search" />
        <input v-model="keyword" type="search" :placeholder="t('agent.git.history.search')" :aria-label="t('agent.git.history.search')" />
      </label>
      <div class="ag-seg agg-hist__scope" role="tablist">
        <button type="button" role="tab" :class="{ 'is-on': !all }" :aria-selected="!all" @click="all = false">{{ t('agent.git.history.current') }}</button>
        <button type="button" role="tab" :class="{ 'is-on': all }" :aria-selected="all" @click="all = true">{{ t('agent.git.history.all') }}</button>
      </div>
    </div>
    <div v-if="git.historyPath.value" class="agg-hist__file">
      <UIcon name="i-lucide-file" />
      <span class="ag-mono ag-ellipsis agc-flex">{{ t('agent.git.history.file', { path: git.historyPath.value }) }}</span>
      <button type="button" class="ag-btn ag-btn--sm" @click="git.historyPath.value = ''">{{ t('agent.git.history.clearFile') }}</button>
    </div>

    <div class="agg-hist__list ag-noscroll">
      <div v-for="c in commits" :key="c.hash" class="agg-cm" :class="{ 'is-open': git.opened.value === c.hash }">
        <div
          class="agg-cm__row"
          role="button"
          tabindex="0"
          :title="t('agent.git.history.open')"
          @click="open(c)"
          @keydown.enter="open(c)"
          @contextmenu.prevent="commitMenu($event, c)"
        >
          <UIcon :name="c.parents.length > 1 ? 'i-lucide-git-merge' : 'i-lucide-git-commit-horizontal'" class="agg-cm__icon" />
          <div class="agg-cm__main">
            <div class="agg-cm__subj">
              <span class="ag-ellipsis">{{ c.subject }}</span>
            </div>
            <div class="agg-cm__meta">
              <span v-for="r in refsOf(c)" :key="r.name" class="ag-tag agg-ref" :class="`agg-ref--${r.kind}`" :title="r.name">
                <UIcon :name="REF_ICON[r.kind]" />{{ r.label }}
              </span>
              <span class="ag-ellipsis">{{ c.author }}</span>
              <span>·</span>
              <span class="agg-nowrap" :title="fullTime(c.date)">{{ relTime(c.date) }}</span>
              <span>·</span>
              <span class="ag-mono">{{ c.shortHash }}</span>
            </div>
          </div>
          <button
            type="button"
            class="ag-ib ag-ib--sm agg-cm__more"
            :aria-label="t('agent.git.history.actions', { hash: c.shortHash })"
            @click.stop="commitMenu($event, c)"
          >
            <UIcon name="i-lucide-ellipsis" />
          </button>
        </div>
      </div>
      <div v-if="loaded && !commits.length" class="agg-empty">
        <UIcon name="i-lucide-history" />{{ keyword.trim() || git.historyPath.value ? t('agent.git.history.noMatch') : t('agent.git.history.none') }}
      </div>
      <button v-if="more" type="button" class="ag-btn ag-btn--sm agg-hist__more" :disabled="loading" @click="load(true)">
        <UIcon v-if="loading" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.git.history.more') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { listAgentGitLog, resetAgentGit, runAgentGitOperation, updateAgentGitRef } from '@/api/agent'
import type { AgentGitCommit } from '@/types/v1/agent'
import { relTime } from '@/views/agent/utils/format'
import { copyText, useGit, type GitMenuItem } from './useGit'

const { t } = useI18n()
const git = useGit()

const PAGE = 60
const REF_ICON: Record<string, string> = { head: 'i-lucide-circle-dot', branch: 'i-lucide-git-branch', remote: 'i-lucide-cloud', tag: 'i-lucide-tag' }

const keyword = ref('')
const all = ref(false)
const commits = ref<AgentGitCommit[]>([])
const more = ref(false)
const loading = ref(false)
const loaded = ref(false)

let seq = 0
const load = async (append = false) => {
  const r = git.repo.value
  if (!r) return
  const id = ++seq
  loading.value = true
  try {
    const { data } = await listAgentGitLog(git.sessionId(), {
      repo: r.repo,
      skip: append ? commits.value.length : 0,
      limit: PAGE,
      all: all.value,
      path: git.historyPath.value || undefined,
      keyword: keyword.value.trim() || undefined,
    })
    if (id !== seq) return
    commits.value = append ? [...commits.value, ...(data.commits ?? [])] : (data.commits ?? [])
    more.value = data.more
    loaded.value = true
  } catch {
    if (id === seq) loaded.value = true
  } finally {
    if (id === seq) loading.value = false
  }
}
const reload = () => void load()
// 换了范围、仓库，做完会改东西的操作，或者 HEAD 变了（Agent 自己提交了）都重读
watch([all, () => git.historyPath.value, () => git.repo.value?.repo, () => git.rev.value, () => git.repo.value?.head], reload, { immediate: true })
watch(keyword, useDebounceFn(reload, 300))

/** 在中间打开这个提交 */
const open = (c: AgentGitCommit) => {
  const r = git.repo.value
  if (!r) return
  git.openCommit({ repo: r.repo, hash: c.hash, short: c.shortHash, subject: c.subject, stash: false })
}

const fullTime = (d: Date | string | undefined) => (d ? new Date(d).toLocaleString() : '')

/** 「HEAD -> main, origin/main, tag: v1」拆成一个个标记 */
const refsOf = (c: AgentGitCommit) => {
  const remotes = new Set(git.repo.value?.remotes ?? [])
  return c.refs.map((raw) => {
    if (raw.startsWith('HEAD -> ')) return { name: raw, label: raw.slice(8), kind: 'head' }
    if (raw === 'HEAD') return { name: raw, label: 'HEAD', kind: 'head' }
    if (raw.startsWith('tag: ')) return { name: raw, label: raw.slice(5), kind: 'tag' }
    const remote = raw.split('/')[0] ?? ''
    if (raw.includes('/') && remotes.has(remote)) return { name: raw, label: raw, kind: 'remote' }
    return { name: raw, label: raw, kind: 'branch' }
  })
}

// —— 对一个提交做的事 ——
const commitMenu = (e: MouseEvent, c: AgentGitCommit) => {
  const r = git.repo.value
  if (!r) return
  const id = git.sessionId()
  const branch = r.branch || 'HEAD'
  const items: GitMenuItem[] = [
    { label: t('agent.git.history.copyHash'), icon: 'i-lucide-copy', run: () => copyText(c.hash) },
    { label: t('agent.git.history.copySubject'), icon: 'i-lucide-clipboard-copy', run: () => copyText(c.subject) },
    { sep: true },
    {
      label: t('agent.git.history.checkout'),
      icon: 'i-lucide-git-commit-horizontal',
      run: () =>
        git.ask({
          title: t('agent.git.history.checkoutTitle', { hash: c.shortHash }),
          message: t('agent.git.history.checkoutMsg'),
          confirmText: t('agent.git.history.checkout'),
          run: () =>
            git.run('ref:checkout', () => updateAgentGitRef(id, { repo: r.repo, action: 'checkout', name: c.hash }), t('agent.git.branch.switched', { name: c.shortHash })),
        }),
    },
    {
      label: t('agent.git.history.newBranch'),
      icon: 'i-lucide-git-branch-plus',
      run: () =>
        git.prompt({
          title: t('agent.git.branch.newTitle', { ref: c.shortHash }),
          fields: [{ key: 'name', label: t('agent.git.branch.name'), mono: true, required: true }],
          checks: [{ key: 'checkout', label: t('agent.git.branch.checkoutAfter'), value: true }],
          run: (v, ck) =>
            git.run(
              'ref:create-branch',
              () => updateAgentGitRef(id, { repo: r.repo, action: 'create-branch', name: v.name!, from: c.hash, checkout: ck.checkout }),
              t('agent.git.branch.created', { name: v.name }),
            ),
        }),
    },
    {
      label: t('agent.git.tag.create'),
      icon: 'i-lucide-tag',
      run: () =>
        git.prompt({
          title: t('agent.git.tag.title', { ref: c.shortHash }),
          fields: [
            { key: 'name', label: t('agent.git.tag.name'), mono: true, required: true },
            { key: 'message', label: t('agent.git.tag.message'), multiline: true },
          ],
          run: (v) =>
            git.run(
              'ref:create-tag',
              () => updateAgentGitRef(id, { repo: r.repo, action: 'create-tag', name: v.name!, from: c.hash, message: v.message }),
              t('agent.git.tag.created', { name: v.name }),
            ),
        }),
    },
    { sep: true },
    {
      label: t('agent.git.history.cherryPick'),
      icon: 'i-lucide-cherry',
      disabled: !r.head,
      run: () => git.run('op:cherry-pick', () => runAgentGitOperation(id, { repo: r.repo, action: 'cherry-pick', commit: c.hash }), t('agent.git.history.picked', { hash: c.shortHash })),
    },
    {
      label: t('agent.git.history.revert'),
      icon: 'i-lucide-rotate-ccw',
      run: () => git.run('op:revert', () => runAgentGitOperation(id, { repo: r.repo, action: 'revert', commit: c.hash }), t('agent.git.history.reverted', { hash: c.shortHash })),
    },
    {
      label: t('agent.git.history.reset'),
      icon: 'i-lucide-history',
      danger: true,
      run: () =>
        git.prompt({
          title: t('agent.git.history.resetTitle', { branch, hash: c.shortHash }),
          fields: [],
          choice: {
            key: 'mode',
            value: 'mixed',
            options: [
              { value: 'soft', label: t('agent.git.history.soft'), hint: t('agent.git.history.softHint') },
              { value: 'mixed', label: t('agent.git.history.mixed'), hint: t('agent.git.history.mixedHint') },
              { value: 'hard', label: t('agent.git.history.hard'), hint: t('agent.git.history.hardHint'), danger: true },
            ],
          },
          confirmText: t('agent.git.history.reset2'),
          danger: true,
          run: (_v, _c, mode) =>
            git.run(
              'reset',
              () => resetAgentGit(id, { repo: r.repo, target: c.hash, mode: mode as 'soft' | 'mixed' | 'hard' }),
              t('agent.git.history.resetDone', { hash: c.shortHash }),
            ),
        }),
    },
  ]
  git.openMenu(e, items)
}
</script>
