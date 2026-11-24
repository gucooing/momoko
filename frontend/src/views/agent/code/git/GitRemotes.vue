<!-- 远程仓库：列出名字和地址，改地址、改名、删除，添加一个新的。 -->
<template>
  <AgDialog :model-value="modelValue" :title="t('agent.git.remote.title')" :width="520" close-on-overlay @update:model-value="emit('update:modelValue', $event)">
    <div class="agg-remotes">
      <div v-for="r in remotes" :key="r.name" class="agg-remote">
        <div class="agg-remote__head">
          <UIcon name="i-lucide-cloud" />
          <span class="agg-remote__n ag-mono">{{ r.name }}</span>
          <span class="agc-flex" />
          <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.git.remote.editUrl')" :title="t('agent.git.remote.editUrl')" @click="editUrl(r)">
            <UIcon name="i-lucide-link" />
          </button>
          <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.git.remote.rename')" :title="t('agent.git.remote.rename')" @click="rename(r)">
            <UIcon name="i-lucide-pencil" />
          </button>
          <button type="button" class="ag-ib ag-ib--sm ag-ib--bad" :aria-label="t('agent.git.remote.remove')" :title="t('agent.git.remote.remove')" @click="remove(r)">
            <UIcon name="i-lucide-trash-2" />
          </button>
        </div>
        <div class="ag-mono ag-muted agg-remote__url">{{ r.fetchUrl }}</div>
        <div v-if="r.pushUrl && r.pushUrl !== r.fetchUrl" class="ag-mono ag-muted agg-remote__url">{{ t('agent.git.remote.pushTo', { url: r.pushUrl }) }}</div>
      </div>
      <div v-if="loaded && !remotes.length" class="agc-right__note">{{ t('agent.git.remote.none') }}</div>

      <form class="agg-remote__add" @submit.prevent="add">
        <div class="ag-fl">{{ t('agent.git.remote.add') }}</div>
        <div class="agg-remote__row">
          <span class="ag-fld agg-remote__name">
            <input v-model="form.name" class="ag-mono" type="text" spellcheck="false" :placeholder="t('agent.git.remote.name')" :aria-label="t('agent.git.remote.name')" />
          </span>
          <span class="ag-fld agc-flex">
            <input v-model="form.url" class="ag-mono" type="text" spellcheck="false" :placeholder="t('agent.git.remote.url')" :aria-label="t('agent.git.remote.url')" />
          </span>
          <button type="submit" class="ag-btn ag-btn--p" :disabled="!form.name.trim() || !form.url.trim() || !!git.busy.value">
            <UIcon name="i-lucide-plus" />{{ t('agent.git.remote.add') }}
          </button>
        </div>
      </form>
    </div>
  </AgDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { listAgentGitRefs, updateAgentGitRemote } from '@/api/agent'
import type { AgentGitRemote } from '@/types/v1/agent'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import { useGit } from './useGit'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const { t } = useI18n()
const git = useGit()

const remotes = ref<AgentGitRemote[]>([])
const loaded = ref(false)
const form = reactive({ name: '', url: '' })

const load = async () => {
  const r = git.repo.value
  if (!r) return
  try {
    const { data } = await listAgentGitRefs(git.sessionId(), r.repo)
    remotes.value = data.remotes ?? []
  } catch {
    remotes.value = []
  } finally {
    loaded.value = true
  }
}
watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    loaded.value = false
    form.name = remotes.value.length ? '' : 'origin'
    form.url = ''
    void load()
  },
  { immediate: true },
)
watch(() => git.rev.value, () => props.modelValue && void load())

const repoName = () => git.repo.value?.repo ?? ''
const op = (data: Parameters<typeof updateAgentGitRemote>[1], ok: string) =>
  git.run(`remote:${data.action}`, () => updateAgentGitRemote(git.sessionId(), data), ok)

const add = async () => {
  const name = form.name.trim()
  const url = form.url.trim()
  if (!name || !url) return
  const done = await op({ repo: repoName(), action: 'add', name, url }, t('agent.git.remote.added', { name }))
  if (done) {
    form.name = ''
    form.url = ''
  }
}
const editUrl = (r: AgentGitRemote) => {
  git.prompt({
    title: `${t('agent.git.remote.editUrl')} · ${r.name}`,
    fields: [{ key: 'url', label: t('agent.git.remote.url'), value: r.fetchUrl, mono: true, required: true }],
    run: (v) => op({ repo: repoName(), action: 'set-url', name: r.name, url: v.url }, t('agent.git.remote.updated', { name: r.name })),
  })
}
const rename = (r: AgentGitRemote) => {
  git.prompt({
    title: `${t('agent.git.remote.rename')} · ${r.name}`,
    fields: [{ key: 'name', label: t('agent.git.remote.name'), value: r.name, mono: true, required: true }],
    run: (v) => op({ repo: repoName(), action: 'rename', name: r.name, newName: v.name }, t('agent.git.remote.updated', { name: v.name })),
  })
}
const remove = (r: AgentGitRemote) => {
  git.ask({
    title: t('agent.git.remote.removeTitle', { name: r.name }),
    message: t('agent.git.remote.removeMsg'),
    confirmText: t('agent.git.remote.remove'),
    danger: true,
    run: () => op({ repo: repoName(), action: 'remove', name: r.name }, t('agent.git.remote.removed', { name: r.name })),
  })
}
</script>
