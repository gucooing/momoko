<!-- 分支按钮点开的浮层：上面一个输入框（筛选，或者直接输入新分支名回车建一个），下面是本地分支、远程分支、标签。
     点一行切换过去；右键（或行尾 ⋯）有合并、变基、新建、改名、删除、推送标签之类。上下键挪、回车切换，Esc 关。 -->
<template>
  <Teleport to="body">
    <div
      ref="popEl"
      class="ag ag-menu agg-bm"
      role="dialog"
      :aria-label="t('agent.git.branch.switch')"
      :style="{ left: `${pos.x}px`, top: `${pos.y}px`, width: `${pos.w}px` }"
      @keydown.esc.stop="emit('close')"
    >
      <label class="ag-inp agg-bm__search">
        <UIcon name="i-lucide-git-branch" />
        <input
          ref="inputEl"
          v-model="query"
          type="text"
          spellcheck="false"
          :placeholder="t('agent.git.branch.search')"
          :aria-label="t('agent.git.branch.search')"
          @keydown="onKey"
        />
      </label>
      <div ref="listEl" class="agg-bm__list ag-noscroll">
        <button v-if="createName" type="button" class="ag-mi" :class="{ 'is-hl': hl === 0 }" :data-i="0" @click="create" @pointermove="hl = 0">
          <UIcon name="i-lucide-git-branch-plus" />
          <span class="ag-ellipsis agc-flex">{{ t('agent.git.branch.create', { name: createName }) }}</span>
        </button>
        <template v-for="g in groups" :key="g.key">
          <div v-if="g.items.length" class="ag-mi-label">{{ g.title }}</div>
          <div
            v-for="it in g.items"
            :key="`${g.key}:${it.name}`"
            class="ag-mi agg-bm__row"
            :class="{ 'is-hl': hl === it.i, 'is-on': it.current }"
            role="button"
            tabindex="-1"
            :data-i="it.i"
            :title="it.subject"
            @click="checkout(it)"
            @pointermove="hl = it.i"
            @contextmenu.prevent="itemMenu($event, g.key, it)"
          >
            <UIcon :name="it.current ? 'i-lucide-check' : g.icon" />
            <span class="agg-bm__name ag-mono ag-ellipsis">{{ it.name }}</span>
            <span v-if="it.ahead" class="ag-num agg-bm__ab" :title="t('agent.git.branch.ahead', { n: it.ahead })">↑{{ it.ahead }}</span>
            <span v-if="it.behind" class="ag-num agg-bm__ab" :title="t('agent.git.branch.behind', { n: it.behind })">↓{{ it.behind }}</span>
            <UIcon v-if="it.gone" name="i-lucide-cloud-off" class="ag-muted" :title="t('agent.git.branch.gone')" />
            <span class="agc-flex" />
            <span class="ag-muted agc-small agg-nowrap">{{ relTime(it.date) }}</span>
            <button type="button" class="ag-ib ag-ib--sm agg-bm__more" :aria-label="t('agent.git.branch.actions', { name: it.name })" @click.stop="itemMenu($event, g.key, it)">
              <UIcon name="i-lucide-ellipsis" />
            </button>
          </div>
        </template>
        <div v-if="refs && !createName && !flat.length" class="agc-pop__note">{{ t('agent.git.branch.none') }}</div>
        <div v-if="!refs" class="agc-pop__note">{{ t('agent.git.diff.loading') }}</div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { onClickOutside, useEventListener } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { listAgentGitRefs, updateAgentGitRef } from '@/api/agent'
import type { ListAgentGitRefsResponse } from '@/types/v1/agent'
import { relTime } from '@/views/agent/utils/format'
import { useGit, type GitMenuItem } from './useGit'

const props = defineProps<{ anchor: HTMLElement | null }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const git = useGit()

type Kind = 'local' | 'remote' | 'tag'
interface Item {
  i: number
  name: string
  subject: string
  date: Date | string | undefined
  current?: boolean
  ahead?: number
  behind?: number
  gone?: boolean
}

const refs = ref<ListAgentGitRefsResponse | null>(null)
const query = ref('')
const popEl = ref<HTMLElement | null>(null)
const inputEl = ref<HTMLInputElement | null>(null)
const listEl = ref<HTMLElement | null>(null)
const pos = ref({ x: 0, y: 0, w: 320 })

const place = () => {
  const r = props.anchor?.getBoundingClientRect()
  if (!r) return
  const w = Math.min(340, window.innerWidth - 16)
  pos.value = { x: Math.max(8, Math.min(r.left, window.innerWidth - w - 8)), y: r.bottom + 4, w }
}
onMounted(async () => {
  place()
  await nextTick()
  inputEl.value?.focus()
  const r = git.repo.value
  if (!r) return
  try {
    const { data } = await listAgentGitRefs(git.sessionId(), r.repo)
    refs.value = data
  } catch {
    emit('close')
  }
})
// 点分支按钮本身由按钮自己开关（不然这里先关、按钮再打开，就关不上了）
onClickOutside(popEl, () => emit('close'), { ignore: [toRef(props, 'anchor'), '.agg-menu', '.ag-dlg'] })
useEventListener(window, 'resize', place)

const match = (name: string) => !query.value.trim() || name.toLowerCase().includes(query.value.trim().toLowerCase())
/** 输入的名字还没有这个分支：第一项是「新建」 */
const createName = computed(() => {
  const q = query.value.trim()
  if (!q || /\s/.test(q) || q.startsWith('-') || !refs.value) return ''
  return refs.value.branches.some((b) => b.name === q) ? '' : q
})
const groups = computed(() => {
  const data = refs.value
  if (!data) return []
  let i = createName.value ? 1 : 0
  const pick = <T extends { name: string }>(list: T[], fn: (x: T) => Omit<Item, 'i'>) =>
    list.filter((x) => match(x.name)).map((x) => ({ ...fn(x), i: i++ }))
  return [
    {
      key: 'local' as Kind,
      title: t('agent.git.branch.local'),
      icon: 'i-lucide-git-branch',
      items: pick(data.branches, (b) => ({ name: b.name, subject: b.subject, date: b.date, current: b.current, ahead: b.ahead, behind: b.behind, gone: b.upstreamGone })),
    },
    {
      key: 'remote' as Kind,
      title: t('agent.git.branch.remote'),
      icon: 'i-lucide-cloud',
      items: pick(data.remoteBranches, (b) => ({ name: b.name, subject: b.subject, date: b.date })),
    },
    {
      key: 'tag' as Kind,
      title: t('agent.git.branch.tags'),
      icon: 'i-lucide-tag',
      items: pick(data.tags, (tg) => ({ name: tg.name, subject: tg.subject, date: tg.date })),
    },
  ]
})
const flat = computed(() => groups.value.flatMap((g) => g.items.map((it) => ({ kind: g.key, it }))))
const hl = ref(0)
watch(query, () => {
  hl.value = 0
})

const reveal = () =>
  void nextTick(() => listEl.value?.querySelector(`[data-i="${hl.value}"]`)?.scrollIntoView({ block: 'nearest' }))
const onKey = (e: KeyboardEvent) => {
  if (e.isComposing) return
  const n = flat.value.length + (createName.value ? 1 : 0)
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    if (!n) return
    hl.value = (hl.value + (e.key === 'ArrowDown' ? 1 : -1) + n) % n
    reveal()
  } else if (e.key === 'Enter') {
    e.preventDefault()
    if (createName.value && hl.value === 0) return create()
    const hit = flat.value.find((x) => x.it.i === hl.value)
    if (hit) checkout(hit.it)
  }
}

// —— 操作 ——
const repoName = () => git.repo.value?.repo ?? ''
const refOp = (data: Parameters<typeof updateAgentGitRef>[1], ok?: string) =>
  git.run(`ref:${data.action}`, () => updateAgentGitRef(git.sessionId(), data), ok)

const create = () => {
  const name = createName.value
  if (!name) return
  emit('close')
  void refOp({ repo: repoName(), action: 'create-branch', name, checkout: true }, t('agent.git.branch.created', { name }))
}
const checkout = (it: Item) => {
  emit('close')
  if (it.current) return
  void refOp({ repo: repoName(), action: 'checkout', name: it.name }, t('agent.git.branch.switched', { name: it.name }))
}

const itemMenu = (e: MouseEvent, kind: Kind, it: Item) => {
  const repo = repoName()
  const current = git.repo.value?.branch || 'HEAD'
  const close = (fn: () => unknown) => () => {
    emit('close')
    return fn()
  }
  const newFrom = close(() =>
    git.prompt({
      title: t('agent.git.branch.newTitle', { ref: it.name }),
      fields: [{ key: 'name', label: t('agent.git.branch.name'), mono: true, required: true, value: kind === 'remote' ? it.name.split('/').slice(1).join('/') : '' }],
      checks: [{ key: 'checkout', label: t('agent.git.branch.checkoutAfter'), value: true }],
      run: (v, ck) => refOp({ repo, action: 'create-branch', name: v.name!, from: it.name, checkout: ck.checkout }, t('agent.git.branch.created', { name: v.name })),
    }),
  )
  const merge = close(() =>
    git.ask({
      title: t('agent.git.branch.mergeTitle', { name: it.name, current }),
      message: t('agent.git.branch.mergeMsg'),
      confirmText: t('agent.git.branch.merge'),
      run: () => refOp({ repo, action: 'merge', name: it.name }, t('agent.git.branch.merged', { name: it.name })),
    }),
  )
  const rebase = close(() =>
    git.ask({
      title: t('agent.git.branch.rebaseTitle', { name: it.name, current }),
      message: t('agent.git.branch.rebaseMsg', { current }),
      confirmText: t('agent.git.branch.rebase'),
      run: () => refOp({ repo, action: 'rebase', name: it.name }, t('agent.git.branch.rebased', { name: it.name })),
    }),
  )
  const items: GitMenuItem[] = []
  if (!it.current) items.push({ label: t('agent.git.branch.checkout'), icon: 'i-lucide-arrow-right-left', run: () => checkout(it) })
  if (kind !== 'tag' && !it.current && git.repo.value?.head) {
    items.push({ label: t('agent.git.branch.merge'), icon: 'i-lucide-git-merge', run: merge })
    items.push({ label: t('agent.git.branch.rebase'), icon: 'i-lucide-git-pull-request-arrow', run: rebase })
  }
  items.push({ label: t('agent.git.branch.newFrom'), icon: 'i-lucide-git-branch-plus', run: newFrom })
  if (kind === 'local') {
    items.push({ sep: true })
    items.push({
      label: t('agent.git.branch.rename'),
      icon: 'i-lucide-pencil',
      run: close(() =>
        git.prompt({
          title: t('agent.git.branch.renameTitle', { name: it.name }),
          fields: [{ key: 'name', label: t('agent.git.branch.newName'), mono: true, required: true, value: it.name }],
          run: (v) => refOp({ repo, action: 'rename-branch', name: it.name, newName: v.name }, t('agent.git.branch.renamed', { name: v.name })),
        }),
      ),
    })
    if (!it.current) {
      items.push({
        label: t('agent.git.branch.delete'),
        icon: 'i-lucide-trash-2',
        danger: true,
        run: close(() =>
          git.ask({
            title: t('agent.git.branch.deleteTitle', { name: it.name }),
            message: t('agent.git.branch.deleteMsg'),
            check: t('agent.git.branch.force'),
            confirmText: t('agent.git.branch.delete'),
            danger: true,
            run: (force) => refOp({ repo, action: 'delete-branch', name: it.name, force }, t('agent.git.branch.deleted', { name: it.name })),
          }),
        ),
      })
    }
  } else if (kind === 'remote') {
    items.push({ sep: true })
    items.push({
      label: t('agent.git.branch.deleteRemote'),
      icon: 'i-lucide-trash-2',
      danger: true,
      run: close(() =>
        git.ask({
          title: t('agent.git.branch.deleteRemoteTitle', { name: it.name }),
          message: t('agent.git.branch.deleteRemoteMsg'),
          confirmText: t('agent.git.branch.delete'),
          danger: true,
          run: () => refOp({ repo, action: 'delete-remote-branch', name: it.name }, t('agent.git.branch.deleted', { name: it.name })),
        }),
      ),
    })
  } else {
    items.push({ sep: true })
    items.push({
      label: t('agent.git.tag.push'),
      icon: 'i-lucide-upload',
      disabled: !git.repo.value?.remotes.length,
      run: close(() => refOp({ repo, action: 'push-tag', name: it.name }, t('agent.git.tag.pushed', { name: it.name }))),
    })
    items.push({
      label: t('agent.git.tag.delete'),
      icon: 'i-lucide-trash-2',
      danger: true,
      run: close(() =>
        git.ask({
          title: t('agent.git.tag.deleteTitle', { name: it.name }),
          message: t('agent.git.tag.deleteMsg'),
          confirmText: t('agent.git.tag.delete'),
          danger: true,
          run: () => refOp({ repo, action: 'delete-tag', name: it.name }, t('agent.git.tag.deleted', { name: it.name })),
        }),
      ),
    })
    items.push({
      label: t('agent.git.tag.deleteRemote'),
      icon: 'i-lucide-cloud-off',
      danger: true,
      disabled: !git.repo.value?.remotes.length,
      run: close(() =>
        git.ask({
          title: t('agent.git.tag.deleteRemoteTitle', { name: it.name }),
          message: t('agent.git.tag.deleteRemoteMsg'),
          confirmText: t('agent.git.tag.deleteRemote'),
          danger: true,
          run: () => refOp({ repo, action: 'delete-remote-tag', name: it.name }, t('agent.git.tag.remoteDeleted', { name: it.name })),
        }),
      ),
    })
  }
  git.openMenu(e, items)
}
</script>
