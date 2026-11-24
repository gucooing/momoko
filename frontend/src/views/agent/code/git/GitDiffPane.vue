<!-- Git 面板下半部分：选中的那个文件的 diff。工作区的可以按段 / 按行暂存、撤销，暂存区的可以按段 / 按行取消暂存，
     冲突的显示文件本身（标出两边），提交和储藏里的只读。文件改了（状态重读）就跟着重读。 -->
<template>
  <section v-if="target" class="agg-diff" :aria-label="target.path">
    <header class="agg-diff__head">
      <span class="ag-tag agg-diff__src">{{ sourceLabel }}</span>
      <span class="ag-mono ag-ellipsis agc-flex" :title="target.path">{{ target.path }}</span>
      <button
        v-if="canOpen"
        type="button"
        class="ag-ib ag-ib--sm"
        :aria-label="t('agent.git.files.open')"
        :title="t('agent.git.files.open')"
        @click="emit('openFile', target.path)"
      >
        <UIcon name="i-lucide-file-symlink" />
      </button>
      <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.git.diff.close')" :title="t('agent.git.diff.close')" @click="git.diff.value = null">
        <UIcon name="i-lucide-x" />
      </button>
    </header>
    <div v-if="target.origPath && target.origPath !== target.path" class="agg-diff__note ag-mono ag-ellipsis">
      {{ t('agent.git.files.renamedFrom', { path: target.origPath }) }}
    </div>
    <div class="agg-diff__body ag-noscroll">
      <div v-if="loading && !data" class="agc-patch__empty">{{ t('agent.git.diff.loading') }}</div>
      <GitPatch
        v-else
        :patch="data?.patch || ''"
        :actions="actions"
        :lines="lineWise"
        :conflict="target.source === 'conflict'"
        :disabled="!!git.busy.value"
        :empty="emptyText"
        @apply="apply"
      />
    </div>
    <div v-if="actions.length && !lineWise && data?.patch" class="agg-patch__tip">{{ t('agent.git.diff.wholeOnly') }}</div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { applyAgentGitLines, getAgentGitDiff, type AgentGitLineAction } from '@/api/agent'
import type { GetAgentGitDiffResponse } from '@/types/v1/agent'
import GitPatch from './GitPatch.vue'
import { useGit } from './useGit'

const emit = defineEmits<{ openFile: [path: string] }>()
const { t } = useI18n()
const git = useGit()

const target = computed(() => git.diffShown.value)
const data = ref<GetAgentGitDiffResponse | null>(null)
const loading = ref(false)

const sourceLabel = computed(() => {
  const d = target.value
  if (!d) return ''
  if (d.source === 'commit') return t('agent.git.diff.commit', { hash: (d.ref || '').slice(0, 7) })
  return t(`agent.git.diff.${d.source}`)
})
/** 工作区里还在的文件才能打开（删掉的、提交里的不行） */
const canOpen = computed(() => {
  const d = target.value
  return !!d && (d.source === 'unstaged' || d.source === 'staged' || d.source === 'conflict') && d.kind !== 'D'
})
const actions = computed<AgentGitLineAction[]>(() => {
  const s = target.value?.source
  if (s === 'unstaged') return ['stage', 'discard']
  if (s === 'staged') return ['unstage']
  return []
})
/** 只有修改过的文件能按行操作：新文件、删掉的、改名的、没跟踪的只能整个来 */
const lineWise = computed(() => {
  const k = target.value?.kind
  return k === 'M' || k === 'T'
})
const emptyText = computed(() => {
  if (data.value?.binary) return t('agent.git.diff.binary')
  if (data.value?.tooLarge) return t('agent.git.diff.tooLarge')
  return t('agent.git.diff.none')
})

let seq = 0
const load = async () => {
  const d = target.value
  const repo = git.repo.value
  if (!d || !repo) {
    data.value = null
    return
  }
  const id = ++seq
  loading.value = true
  try {
    const { data: res } = await getAgentGitDiff(git.sessionId(), {
      repo: repo.repo,
      source: d.source,
      path: d.path,
      origPath: d.origPath || undefined,
      ref: d.ref || undefined,
    })
    if (id === seq) data.value = res
  } catch {
    if (id === seq) data.value = null
  } finally {
    if (id === seq) loading.value = false
  }
}
// 换了文件重读；同一个文件在状态重读后也重读（暂存了几行、Agent 又改了它）
watch(
  () => [target.value?.source, target.value?.path, target.value?.ref, git.repo.value?.repo],
  () => {
    data.value = null
    void load()
  },
  { immediate: true },
)
watch(
  () => git.repos.value,
  () => {
    const d = target.value
    if (!d || d.source === 'commit' || d.source === 'stash') return
    // 文件已经不在这一组里了（整个暂存、撤销掉了）：换到它现在在的那一组，哪都不在了就关掉
    const r = git.repo.value
    const inList = (list: { path: string }[] | undefined) => !!list?.some((f) => f.path === d.path)
    const lists = { conflict: r?.conflicted, unstaged: r?.unstaged, staged: r?.staged }
    if (!inList(lists[d.source as keyof typeof lists])) {
      const next = (['conflict', 'unstaged', 'staged'] as const).find((s) => inList(lists[s]))
      if (!next) {
        git.diff.value = null
        return
      }
      const f = lists[next]!.find((x) => x.path === d.path)!
      git.diff.value = { ...d, source: next, kind: f.kind, origPath: f.origPath, binary: f.binary }
      return
    }
    void load()
  },
)

const apply = (action: AgentGitLineAction, lines: number[]) => {
  const d = target.value
  const repo = git.repo.value
  const patch = data.value?.patch
  if (!d || !repo || !patch || !lines.length) return
  const go = () =>
    git.run(`lines:${action}`, () => applyAgentGitLines(git.sessionId(), { repo: repo.repo, path: d.path, action, patch, lines }))
  if (action === 'discard') {
    git.ask({ title: t('agent.git.diff.discardTitle'), message: t('agent.git.diff.discardMsg'), confirmText: t('agent.git.diff.linesDiscard'), danger: true, run: go })
    return
  }
  void go()
}
</script>
