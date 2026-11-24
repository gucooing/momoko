<!-- 中间的「提交」标签：在右侧 Git 的历史（或储藏）里点了一个提交，这里显示它的完整说明、作者、指向它的分支标签，
     以及每个改动文件的 diff（文件多时默认收着，点开才读）。只读；文件还在的话可以打开它现在的版本。 -->
<template>
  <div class="agg-cv agc-vbar">
    <header class="agg-cv__head">
      <div class="agg-cv__kind ag-muted">
        <UIcon :name="target.stash ? 'i-lucide-archive' : 'i-lucide-git-commit-horizontal'" />
        <span>{{ target.stash ? t('agent.git.view.stash') : t('agent.git.view.commit') }}</span>
        <span v-if="target.repo" class="ag-mono">· {{ target.repo }}</span>
      </div>
      <h2 class="agg-cv__subj">{{ detail?.commit?.subject || target.subject }}</h2>
      <pre v-if="detail?.body" class="agg-cv__body">{{ detail.body }}</pre>
      <div v-if="detail?.commit" class="agg-cv__meta">
        <span :title="detail.commit.email">{{ detail.commit.author }}</span>
        <span>·</span>
        <span :title="fullTime(detail.commit.date)">{{ fullTime(detail.commit.date) }}</span>
        <template v-if="detail.committer && detail.committer !== detail.commit.author">
          <span>·</span>
          <span>{{ t('agent.git.history.committedBy', { name: detail.committer, time: relTime(detail.commitDate) }) }}</span>
        </template>
      </div>
      <div v-if="detail?.commit" class="agg-cv__meta">
        <button type="button" class="ag-tag ag-mono agg-cv__hash" :title="t('agent.git.history.copyHash')" @click="copyText(detail.commit.hash)">
          <UIcon name="i-lucide-copy" />{{ detail.commit.shortHash }}
        </button>
        <span v-if="detail.commit.parents.length" class="ag-mono">
          {{ t('agent.git.view.parents') }} {{ detail.commit.parents.map((p) => p.slice(0, 7)).join(' ') }}
        </span>
        <span v-for="r in detail.commit.refs" :key="r" class="ag-tag agg-ref" :class="r.startsWith('tag: ') ? 'agg-ref--tag' : 'agg-ref--branch'">{{ r }}</span>
      </div>
    </header>

    <div v-if="loading" class="agc-right__note">{{ t('agent.git.diff.loading') }}</div>
    <template v-else-if="detail">
      <div class="agg-cv__bar">
        <span class="agc-flex">
          {{ files.length ? t('agent.git.history.files', { n: files.length }) : t('agent.git.history.noFiles') }}
          <span class="agc-plus ag-num">+{{ totalAdd }}</span>
          <span class="agc-minus ag-num">−{{ totalDel }}</span>
        </span>
        <button v-if="files.length" type="button" class="ag-btn ag-btn--sm" @click="setAll(!allOpen)">
          <UIcon :name="allOpen ? 'i-lucide-chevrons-down-up' : 'i-lucide-chevrons-up-down'" />
          {{ allOpen ? t('agent.git.view.collapseAll') : t('agent.git.view.expandAll') }}
        </button>
      </div>
      <section v-for="f in files" :key="f.path" class="agg-cv__file">
        <div class="agg-cv__fh">
          <button type="button" class="agg-cv__ft" :aria-expanded="opened.has(f.path)" @click="toggle(f)">
            <UIcon :name="opened.has(f.path) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="ag-muted" />
            <span class="ag-mono agg-fr__k" :class="KIND_CLS[f.kind] ?? 'agc-warn'" :title="t(`agent.git.files.kinds.${f.kind}`)">{{ f.kind }}</span>
            <span class="ag-mono ag-ellipsis">
              <template v-if="f.origPath && f.origPath !== f.path">{{ f.origPath }} → </template>{{ f.path }}
            </span>
            <span class="agc-flex" />
            <span v-if="f.binary" class="ag-muted agc-small">bin</span>
            <template v-else>
              <span class="agc-plus ag-num">+{{ f.added }}</span>
              <span class="agc-minus ag-num">−{{ f.deleted }}</span>
            </template>
          </button>
          <button
            v-if="f.kind !== 'D'"
            type="button"
            class="ag-ib ag-ib--sm"
            :aria-label="t('agent.git.files.open')"
            :title="t('agent.git.files.open')"
            @click="emit('openFile', repoPath(target.repo, f.path))"
          >
            <UIcon name="i-lucide-file-symlink" />
          </button>
        </div>
        <div v-if="opened.has(f.path)" class="agg-cv__fb">
          <div v-if="!diffs[f.path]" class="agc-patch__empty">{{ t('agent.git.diff.loading') }}</div>
          <GitPatch v-else :patch="diffs[f.path]!.patch" :empty="emptyOf(diffs[f.path]!)" />
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getAgentGitCommit, getAgentGitDiff } from '@/api/agent'
import type { AgentGitFile, GetAgentGitCommitResponse, GetAgentGitDiffResponse } from '@/types/v1/agent'
import { relTime } from '@/views/agent/utils/format'
import GitPatch from './GitPatch.vue'
import { copyText, repoPath, type GitCommitTarget } from './useGit'

const props = defineProps<{ sessionId: string; target: GitCommitTarget }>()
const emit = defineEmits<{ openFile: [path: string] }>()
const { t } = useI18n()

const KIND_CLS: Record<string, string> = { A: 'agc-ok', D: 'agc-bad', R: 'agg-k--r', C: 'agg-k--r' }
/** 文件不多时全部展开并读出来；多了先收着，点开哪个读哪个 */
const AUTO_OPEN = 25

const detail = ref<GetAgentGitCommitResponse | null>(null)
const loading = ref(true)
const files = computed<AgentGitFile[]>(() => detail.value?.files ?? [])
const totalAdd = computed(() => files.value.reduce((n, f) => n + Number(f.added || 0), 0))
const totalDel = computed(() => files.value.reduce((n, f) => n + Number(f.deleted || 0), 0))
const opened = reactive(new Set<string>())
const diffs = reactive<Record<string, GetAgentGitDiffResponse>>({})
const allOpen = computed(() => files.value.length > 0 && files.value.every((f) => opened.has(f.path)))

const fullTime = (d: Date | string | undefined) => (d ? new Date(d).toLocaleString() : '')
const emptyOf = (d: GetAgentGitDiffResponse) =>
  d.binary ? t('agent.git.diff.binary') : d.tooLarge ? t('agent.git.diff.tooLarge') : t('agent.git.diff.none')

const loadDiff = async (f: AgentGitFile) => {
  if (diffs[f.path]) return
  try {
    const { data } = await getAgentGitDiff(props.sessionId, {
      repo: props.target.repo,
      source: props.target.stash ? 'stash' : 'commit',
      path: f.path,
      origPath: f.origPath || undefined,
      ref: props.target.hash,
    })
    diffs[f.path] = data
  } catch {
    diffs[f.path] = { patch: '', binary: false, tooLarge: false }
  }
}
/** 同时最多读几个文件的 diff */
const loadMany = async (list: AgentGitFile[]) => {
  const queue = [...list]
  const worker = async () => {
    for (let f = queue.shift(); f; f = queue.shift()) await loadDiff(f)
  }
  await Promise.all(Array.from({ length: Math.min(4, queue.length) }, worker))
}
const toggle = (f: AgentGitFile) => {
  if (opened.has(f.path)) {
    opened.delete(f.path)
    return
  }
  opened.add(f.path)
  void loadDiff(f)
}
const setAll = (open: boolean) => {
  opened.clear()
  if (!open) return
  for (const f of files.value) opened.add(f.path)
  void loadMany(files.value)
}

onMounted(async () => {
  try {
    const { data } = await getAgentGitCommit(props.sessionId, props.target.repo, props.target.hash, props.target.stash)
    detail.value = data
    if (data.files.length <= AUTO_OPEN) setAll(true)
  } catch {
    /* shown */
  } finally {
    loading.value = false
  }
})
</script>
