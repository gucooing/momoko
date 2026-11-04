<!-- SSH 主机：标题（主机数 · 分组数 · 已分享）+ 批量测试 / 添加主机；批量测试结果卡（每台一格状态条、首个失败原因、平均延迟、只看失败）；
     状态 chips（带数量）+ 搜索 + 分组筛选；信息卡片（贴纸头像、地址、延迟 pill、认证/分组/分享标签、上次测试/上次连接信息格、
     打开终端 + 测试/分享/编辑/⋯）；勾选后底部深色批量条（测试 / 删除）；分页。
     创建编辑走 create.vue（ref showDialog(row?) + @refresh），分享走 FormDialog + UserPicker。 -->
<template>
  <div class="ssh-page">
    <PageHeader :title="t('ssh.list.title')">
      <template #description>
        {{ t('ssh.list.summary', { total: stats.total, groups: stats.tags.length }) }}
        <template v-if="stats.shared">
          · <span class="ssh-page__em">{{ t('ssh.list.summaryShared', { count: stats.shared }) }}</span>
        </template>
      </template>
      <template #actions>
        <UButton
          v-if="!menuStore.isMobile"
          color="neutral"
          variant="outline"
          icon="i-lucide-activity"
          :loading="batchTesting"
          :disabled="!stats.total"
          @click="runBatchTest()"
        >
          {{ t('ssh.list.batchTest') }}
        </UButton>
        <AppIconButton
          v-else
          icon="HOutline:SignalIcon"
          :label="t('ssh.list.batchTest')"
          :box="44"
          :class="{ 'is-spinning': batchTesting }"
          @click="runBatchTest()"
        />
        <UButton v-if="!menuStore.isMobile" color="primary" icon="i-lucide-plus" @click="openCreateDialog()">
          {{ t('ssh.list.add') }}
        </UButton>
      </template>
    </PageHeader>

    <!-- 批量测试结果 -->
    <section v-if="stats.lastTestTime" class="ssh-batch" :aria-label="t('ssh.list.batchResult')">
      <span class="ssh-batch__av m-sticker m-sticker--ok">
        <UIcon name="i-lucide-activity" />
      </span>
      <div class="ssh-batch__main">
        <span class="ssh-batch__when">
          {{ batchTesting ? t('ssh.list.batchRunning') : t('ssh.list.batchWhen', { time: relTime(stats.lastTestTime) }) }}
        </span>
        <span class="ssh-batch__nums m-num">
          {{ t('ssh.list.hostsCount', { count: testedCount }) }} ·
          <span class="is-ok">{{ t('ssh.list.okCount', { count: stats.ok }) }}</span> ·
          <span class="is-bad">{{ t('ssh.list.failCount', { count: stats.failed }) }}</span>
        </span>
      </div>
      <div class="ssh-batch__bars" aria-hidden="true">
        <span
          v-for="m in stats.marks"
          :key="m.id"
          class="ssh-batch__bar"
          :class="m.tested ? (m.ok ? 'is-ok' : 'is-bad') : 'is-none'"
          :title="m.name"
        />
      </div>
      <button v-if="firstFailure" type="button" class="ssh-batch__fail" @click="setFilter(SSHHostFilter.SSH_HOST_FILTER_FAILED)">
        <UIcon name="i-lucide-triangle-alert" />
        <span class="ssh-batch__fail-text">{{ firstFailure.name }} · {{ firstFailure.message || t('ssh.list.failed') }}</span>
        <span v-if="menuStore.isMobile" class="ssh-batch__fail-cta">{{ t('ssh.list.onlyFailed') }}</span>
      </button>
      <span v-if="stats.avgLatencyMs" class="ssh-batch__lat m-num">
        {{ t('ssh.list.avgLatency') }} <b>{{ stats.avgLatencyMs }} ms</b>
      </span>
      <UButton
        v-if="stats.failed && !menuStore.isMobile"
        color="neutral"
        variant="soft"
        size="sm"
        @click="setFilter(SSHHostFilter.SSH_HOST_FILTER_FAILED)"
      >
        {{ t('ssh.list.onlyFailed') }}
      </UButton>
    </section>

    <ListToolbar
      v-model:status="filter"
      v-model:keyword="keyword"
      :chips="chips"
      :chips-label="t('ssh.list.filterLabel')"
      :placeholder="t('ssh.list.searchPlaceholder')"
      @search="reload"
    >
      <div v-if="stats.tags.length" class="ssh-page__group">
        <AppSelect v-model="tagFilter" :options="tagOptions" @update:model-value="reload" />
      </div>
    </ListToolbar>

    <div v-if="loading && !list.length" class="ssh-grid">
      <div v-for="i in 4" :key="i" class="ssh-skeleton" />
    </div>
    <EmptyState
      v-else-if="!list.length"
      icon="HOutline:CommandLineIcon"
      :title="stats.total ? t('ssh.list.noMatch') : t('ssh.common.noData')"
      :description="stats.total ? t('ssh.list.noMatchDesc') : t('ssh.common.emptyDesc')"
    >
      <template v-if="!stats.total" #action>
        <UButton color="primary" variant="soft" icon="i-lucide-plus" @click="openCreateDialog()">
          {{ t('ssh.list.add') }}
        </UButton>
      </template>
    </EmptyState>
    <div v-else class="ssh-grid" :class="{ 'is-loading': loading }">
      <article
        v-for="row in list"
        :key="row.id"
        class="ssh-card"
        :class="{ 'is-bad': isFailed(row), 'is-selected': selectedIdSet.has(row.id) }"
      >
        <header class="ssh-card__head">
          <span class="ssh-card__av m-sticker" :class="`m-sticker--${toneOf(row.id)}`" :style="{ transform: `rotate(${rotOf(row.id)})` }">
            {{ initialOf(row.name) }}
          </span>
          <span class="ssh-card__title">
            <span class="ssh-card__name" :title="row.name">{{ row.name }}</span>
            <span class="ssh-card__addr m-mono" :title="addrOf(row)">{{ addrOf(row) }}</span>
          </span>
          <UIcon v-if="testingIds.has(row.id)" name="i-lucide-loader-circle" class="ssh-card__spin" />
          <StatusPill v-else :variant="pillOf(row).variant" :label="pillOf(row).label" />
          <label class="ssh-card__hit">
            <input
              type="checkbox"
              class="ssh-card__check"
              :checked="selectedIdSet.has(row.id)"
              :aria-label="t('ssh.list.select', { name: row.name })"
              @change="toggleSelect(row.id)"
            />
          </label>
        </header>

        <div v-if="!menuStore.isMobile" class="ssh-card__tags">
          <span class="ssh-tag">
            <UIcon :name="isKey(row) ? 'i-lucide-key-round' : 'i-lucide-lock'" />{{ authText(row) }}
          </span>
          <span v-for="tag in tagsOf(row)" :key="tag" class="ssh-tag">#{{ tag }}</span>
          <span v-if="shareTag(row)" class="ssh-tag is-share"><UIcon name="i-lucide-share-2" />{{ shareTag(row) }}</span>
        </div>

        <dl class="ssh-card__info">
          <div v-if="menuStore.isMobile">
            <dt>{{ t('ssh.common.authType') }}</dt>
            <dd class="ssh-card__auth">
              <UIcon :name="isKey(row) ? 'i-lucide-key-round' : 'i-lucide-lock'" />{{ authText(row) }}
            </dd>
          </div>
          <div :class="{ 'is-wide': menuStore.isMobile }" :style="menuStore.isMobile ? { order: 3 } : undefined">
            <dt>
              {{ t('ssh.list.lastTest') }}
              <template v-if="menuStore.isMobile && row.lastTestTime"> · {{ relTime(row.lastTestTime) }}</template>
            </dt>
            <dd class="m-num" :class="{ 'is-bad': isFailed(row) }" :title="testText(row)">{{ testText(row) }}</dd>
          </div>
          <div>
            <dt>{{ t('ssh.list.lastConnect') }}</dt>
            <dd class="m-num">{{ row.lastConnectTime ? relTime(row.lastConnectTime) : t('ssh.list.neverConnected') }}</dd>
          </div>
          <div v-if="row.remark && !menuStore.isMobile" class="is-wide">
            <dt>{{ t('ssh.common.remark') }}</dt>
            <dd :title="row.remark">{{ row.remark }}</dd>
          </div>
        </dl>

        <footer class="ssh-card__foot">
          <template v-if="!menuStore.isMobile">
            <UButton class="ssh-card__open" color="primary" size="md" icon="i-lucide-terminal" @click="connectSsh(row)">
              {{ t('ssh.list.openTerminal') }}
            </UButton>
          </template>
          <template v-else>
            <div class="ssh-card__mtags">
              <span v-for="tag in tagsOf(row).slice(0, 2)" :key="tag" class="ssh-tag">#{{ tag }}</span>
              <span v-if="shareTag(row)" class="ssh-tag is-share">{{ isOwner(row) ? t('ssh.list.sharedShort') : t('ssh.list.sharedToMe') }}</span>
            </div>
            <AppIconButton
              icon="HOutline:CommandLineIcon"
              class="is-primary"
              :label="t('ssh.list.openTerminalOf', { name: row.name })"
              :box="44"
              @click="connectSsh(row)"
            />
          </template>
          <AppIconButton
            icon="HOutline:SignalIcon"
            :label="t('ssh.list.testOf', { name: row.name })"
            :box="menuStore.isMobile ? 44 : 36"
            :size="18"
            :disabled="testingIds.has(row.id)"
            @click="testOne(row)"
          />
          <AppIconButton
            v-if="isOwner(row) && !nodeStore.isRemote"
            icon="HOutline:ShareIcon"
            :label="t('ssh.list.shareOf', { name: row.name })"
            :box="menuStore.isMobile ? 44 : 36"
            :size="18"
            @click="openShareDialog(row)"
          />
          <AppIconButton
            v-if="isOwner(row)"
            icon="HOutline:PencilSquareIcon"
            :label="t('ssh.list.editOf', { name: row.name })"
            :box="menuStore.isMobile ? 44 : 36"
            :size="18"
            @click="createRef?.showDialog(row)"
          />
          <ActionMenu :items="rowActionsFor(row)" :label="t('ssh.list.moreOf', { name: row.name })" @select="(key) => onRowAction(key, row)" />
        </footer>
      </article>
    </div>

    <Pagination
      v-if="pagination.total > pagination.pageSize"
      v-model:page="pagination.page"
      v-model:page-size="pagination.pageSize"
      :total="pagination.total"
      @change="getList"
    />

    <!-- 手机：悬浮添加按钮 -->
    <div v-if="menuStore.isMobile && !selectedIds.length" class="ssh-fab">
      <UButton color="primary" size="xl" icon="i-lucide-plus" class="ssh-fab__btn" @click="openCreateDialog()">
        {{ t('ssh.list.add') }}
      </UButton>
    </div>

    <!-- 底部悬浮批量条 -->
    <Transition name="ssh-bulk">
      <div v-if="selectedIds.length" class="ssh-bulk" role="toolbar" :aria-label="t('ssh.common.selectedCount', { count: selectedIds.length })">
        <span class="ssh-bulk__count">{{ t('ssh.common.selectedCount', { count: selectedIds.length }) }}</span>
        <button type="button" class="ssh-bulk__btn" @click="toggleSelectAll">
          {{ isAllSelected ? t('ssh.common.clearSelection') : t('ssh.common.selectAll') }}
        </button>
        <span class="ssh-bulk__spacer" />
        <button type="button" class="ssh-bulk__btn" :disabled="batchTesting" @click="runBatchTest(selectedIds)">
          <UIcon name="i-lucide-activity" />{{ t('ssh.list.testSelected') }}
        </button>
        <button type="button" class="ssh-bulk__btn is-danger" :disabled="!selectedOwnedIds.length" @click="batchDelete">
          <UIcon name="i-lucide-trash-2" />{{ t('ssh.common.batchDelete') }}
        </button>
        <button type="button" class="ssh-bulk__x" :aria-label="t('ssh.common.clearSelection')" @click="selectedIds = []">
          <UIcon name="i-lucide-x" />
        </button>
      </div>
    </Transition>

    <SshConnectionCreate ref="createRef" @refresh="refresh" />

    <!-- 分享连接 -->
    <FormDialog
      v-model="shareDialogOpen"
      :title="t('ssh.common.shareConnection')"
      :width="520"
      :loading="shareLoading"
      @confirm="confirmShare"
    >
      <div class="app-field">
        <label class="app-label">{{ t('ssh.common.sharedUsers') }}</label>
        <UserPicker v-model="shareUserIds" :placeholder="t('ssh.common.shareUsersPlaceholder')" />
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import SshConnectionCreate from '@/views/openssh/management/create.vue'
import { getSshHosts, deleteSshHost, shareSshHost, testSshHosts } from '@/api/openssh'
import { SSHAuthType, SSHHostFilter, type SSHHostInfo, type SSHHostStats } from '@/types/v1/openssh'
import { Dialog } from '@/utils/dialog'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'

defineOptions({ name: 'SshManagementView' })

const router = useRouter()
const menuStore = useMenuStore()
const nodeStore = useNodeStore()
const { t } = useI18n()
const createRef = useTemplateRef<InstanceType<typeof SshConnectionCreate> | null>('createRef')

const OWNER_ROLE = 'SSH_HOST_ACCESS_ROLE_OWNER'

const loading = ref(false)
const list = ref<SSHHostInfo[]>([])
const selectedIds = ref<string[]>([])
const selectedIdSet = computed(() => new Set(selectedIds.value))

const filter = ref<SSHHostFilter>(SSHHostFilter.SSH_HOST_FILTER_ALL)
const keyword = ref('')
const tagFilter = ref('')
const pagination = ref({ page: 1, pageSize: 24, total: 0 })

const emptyStats = (): SSHHostStats => ({
  total: 0,
  ok: 0,
  failed: 0,
  untested: 0,
  shared: 0,
  sharedWithMe: 0,
  tags: [],
  lastTestTime: undefined,
  avgLatencyMs: 0,
  marks: [],
})
const stats = ref<SSHHostStats>(emptyStats())

const chips = computed<ToolbarChip<SSHHostFilter>[]>(() => {
  const s = stats.value
  const out: ToolbarChip<SSHHostFilter>[] = [
    { value: SSHHostFilter.SSH_HOST_FILTER_ALL, label: t('ssh.list.chipAll'), count: Number(s.total) },
    { value: SSHHostFilter.SSH_HOST_FILTER_OK, label: t('ssh.list.chipOk'), count: Number(s.ok) },
    { value: SSHHostFilter.SSH_HOST_FILTER_FAILED, label: t('ssh.list.chipFailed'), count: Number(s.failed) },
  ]
  if (Number(s.untested)) out.push({ value: SSHHostFilter.SSH_HOST_FILTER_UNTESTED, label: t('ssh.list.chipUntested'), count: Number(s.untested) })
  out.push({ value: SSHHostFilter.SSH_HOST_FILTER_SHARED, label: t('ssh.list.chipShared'), count: Number(s.shared) })
  if (Number(s.sharedWithMe))
    out.push({ value: SSHHostFilter.SSH_HOST_FILTER_SHARED_WITH_ME, label: t('ssh.list.chipSharedWithMe'), count: Number(s.sharedWithMe) })
  return out
})

const tagOptions = computed(() => [
  { label: t('ssh.list.groupAll'), value: '' },
  ...stats.value.tags.map((tag) => ({ label: t('ssh.list.groupOf', { name: tag }), value: tag })),
])

const testedCount = computed(() => Number(stats.value.ok) + Number(stats.value.failed))
const firstFailure = computed(() => stats.value.marks.find((m) => m.tested && !m.ok))

// —— 卡片展示 ——
const TONES = ['p', 'a2', 'a', 'ok']
const ROTS = ['-4deg', '3deg', '-2deg', '4deg']
const hashOf = (id: string) => [...id].reduce((n, c) => (n * 31 + c.charCodeAt(0)) >>> 0, 7)
const toneOf = (id: string) => TONES[hashOf(id) % TONES.length]
const rotOf = (id: string) => ROTS[hashOf(id) % ROTS.length]
const initialOf = (name: string) => (name || '?').trim().charAt(0).toUpperCase()

const isOwner = (row: SSHHostInfo) => row.accessRole === OWNER_ROLE
const isKey = (row: SSHHostInfo) => row.authType === SSHAuthType.SSH_AUTH_TYPE_KEY
const authText = (row: SSHHostInfo) => (isKey(row) ? t('ssh.common.key') : t('ssh.common.passwordAuth'))
const addrOf = (row: SSHHostInfo) => `${row.username}@${row.host}:${row.port}`
const tagsOf = (row: SSHHostInfo) =>
  (row.tags || '')
    .split(/[,，;；\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
    .slice(0, 3)
const isFailed = (row: SSHHostInfo) => !!row.lastTestTime && !row.lastTestOk
const shareTag = (row: SSHHostInfo) => {
  if (!isOwner(row)) return t('ssh.list.sharedToMe')
  const n = row.sharedUsers?.length || 0
  return n ? t('ssh.list.sharedTo', { count: n }) : ''
}

const pillOf = (row: SSHHostInfo): { variant: 'success' | 'warning' | 'error' | 'neutral'; label: string } => {
  if (!row.lastTestTime) return { variant: 'neutral', label: t('ssh.list.untested') }
  if (!row.lastTestOk) return { variant: 'error', label: t('ssh.list.failed') }
  const ms = Number(row.lastTestLatencyMs)
  return { variant: ms > 1500 ? 'warning' : 'success', label: `${ms} ms` }
}
const testText = (row: SSHHostInfo) => {
  if (!row.lastTestTime) return t('ssh.list.neverTested')
  if (!row.lastTestOk) return row.lastTestMessage || t('ssh.list.failed')
  return t('ssh.list.testOk', { ms: row.lastTestLatencyMs })
}

const relTime = (value: unknown): string => {
  if (!value) return '—'
  const d = dayjs(value as string | Date)
  if (!d.isValid()) return '—'
  const diff = dayjs().diff(d, 'minute')
  if (diff < 1) return t('ssh.list.justNow')
  if (diff < 60) return t('ssh.list.minutesAgo', { n: diff })
  if (d.isSame(dayjs(), 'day')) return t('ssh.list.todayAt', { time: d.format('HH:mm') })
  if (d.isSame(dayjs().subtract(1, 'day'), 'day')) return t('ssh.list.yesterdayAt', { time: d.format('HH:mm') })
  if (dayjs().diff(d, 'day') < 7) return t('ssh.list.daysAgo', { n: dayjs().diff(d, 'day') })
  return d.isSame(dayjs(), 'year') ? d.format('MM-DD') : d.format('YYYY-MM-DD')
}

const rowActionsFor = (row: SSHHostInfo): ActionMenuItem[] => {
  const owner = isOwner(row)
  return [
    { key: 'copy', label: t('ssh.list.copyAddr'), icon: 'i-lucide-copy' },
    { key: 'connect', label: t('ssh.list.openTerminal'), icon: 'i-lucide-terminal', hidden: !menuStore.isMobile },
    // 分享对象是本机用户，远程节点上下文不提供分享
    { key: 'share', label: t('ssh.common.share'), icon: 'HOutline:ShareIcon', hidden: !owner || nodeStore.isRemote || !menuStore.isMobile },
    { key: 'delete', label: t('ssh.common.delete'), icon: 'HOutline:TrashIcon', danger: true, hidden: !owner },
  ]
}

const onRowAction = (key: string, row: SSHHostInfo) => {
  if (key === 'edit') createRef.value?.showDialog(row)
  else if (key === 'share') openShareDialog(row)
  else if (key === 'delete') confirmDelete(row)
  else if (key === 'connect') connectSsh(row)
  else if (key === 'copy') copyAddr(row)
}

const copyAddr = async (row: SSHHostInfo) => {
  try {
    await navigator.clipboard.writeText(`ssh ${row.username}@${row.host}${row.port === 22 ? '' : ` -p ${row.port}`}`)
    feedback.success(t('ssh.list.copied'))
  } catch {
    feedback.warning(t('terminal.clipboardDenied'))
  }
}

// —— 数据 ——
const getList = async () => {
  loading.value = true
  try {
    const { data } = await getSshHosts({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: keyword.value.trim() || undefined,
      filter: filter.value,
      tag: tagFilter.value || undefined,
    })
    list.value = data?.infos || []
    pagination.value.total = Number(data?.total) || 0
    stats.value = { ...emptyStats(), ...(data?.stats || {}) }
    // 翻页/筛选后仍保留选择，但剔除已不存在的主机
    const ids = new Set(stats.value.marks.map((m) => m.id))
    selectedIds.value = selectedIds.value.filter((id) => ids.has(id))
  } finally {
    loading.value = false
  }
}

const reload = () => {
  pagination.value.page = 1
  getList()
}

const setFilter = (value: SSHHostFilter) => {
  filter.value = value
  reload()
}

const isAllSelected = computed(() => list.value.length > 0 && list.value.every((r) => selectedIdSet.value.has(r.id)))
const toggleSelect = (id: string) => {
  selectedIds.value = selectedIdSet.value.has(id)
    ? selectedIds.value.filter((x) => x !== id)
    : [...selectedIds.value, id]
}
const toggleSelectAll = () => {
  const pageIds = list.value.map((r) => r.id)
  selectedIds.value = isAllSelected.value
    ? selectedIds.value.filter((id) => !pageIds.includes(id))
    : [...new Set([...selectedIds.value, ...pageIds])]
}
// 只有自己创建的主机可删除
const selectedOwnedIds = computed(() => {
  const owned = new Set(list.value.filter(isOwner).map((r) => r.id))
  return selectedIds.value.filter((id) => owned.has(id))
})

const openCreateDialog = () => createRef.value?.showDialog()

const connectSsh = (row: SSHHostInfo) => {
  router.push({ path: '/openssh/terminal', query: { id: row.id } })
}

// —— 测试 ——
const batchTesting = ref(false)
const testingIds = ref(new Set<string>())

const runBatchTest = async (ids: string[] = []) => {
  if (batchTesting.value) return
  batchTesting.value = true
  const pending = ids.length ? ids : list.value.map((r) => r.id)
  testingIds.value = new Set(pending)
  try {
    const { data } = await testSshHosts({ ids })
    const results = data?.results || []
    const ok = results.filter((r) => r.ok).length
    const failed = results.length - ok
    if (failed) feedback.warning(t('ssh.list.batchDoneWithFail', { ok, failed }))
    else feedback.success(t('ssh.list.batchDone', { ok }))
    await getList()
  } finally {
    batchTesting.value = false
    testingIds.value = new Set()
  }
}

const testOne = async (row: SSHHostInfo) => {
  testingIds.value = new Set([...testingIds.value, row.id])
  try {
    const { data } = await testSshHosts({ ids: [row.id] })
    const r = data?.results?.[0]
    if (r?.ok) feedback.success(t('ssh.common.connectionSuccess', { message: row.name }))
    else feedback.warning(t('ssh.common.connectionFailedWithMessage', { message: r?.message || t('ssh.common.offline') }))
    await getList()
  } finally {
    const next = new Set(testingIds.value)
    next.delete(row.id)
    testingIds.value = next
  }
}

// —— 删除 ——
const confirmDelete = (row: SSHHostInfo) => {
  Dialog.confirm({
    title: t('ssh.common.confirmDeleteTitle'),
    content: t('ssh.common.confirmDeleteContent'),
    confirmText: t('ssh.common.delete'),
    cancelText: t('ssh.common.cancel'),
    onConfirm: async () => {
      await deleteSshHost({ id: row.id })
      feedback.success(t('ssh.common.deleteSuccess'))
      selectedIds.value = selectedIds.value.filter((d) => d !== row.id)
      getList()
    },
  })
}

const batchDelete = () => {
  const ids = selectedOwnedIds.value
  if (!ids.length) return
  Dialog.confirm({
    title: t('ssh.common.confirmDeleteTitle'),
    content: t('ssh.common.confirmDeleteSelected'),
    confirmText: t('ssh.common.batchDelete'),
    cancelText: t('ssh.common.cancel'),
    onConfirm: async () => {
      await Promise.all(ids.map((id) => deleteSshHost({ id })))
      feedback.success(t('ssh.common.batchDeleteSuccess'))
      selectedIds.value = []
      getList()
    },
  })
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') pagination.value.page = 1
  getList()
}

// —— 分享 ——
const shareDialogOpen = ref(false)
const shareLoading = ref(false)
const shareHostId = ref('')
const shareUserIds = ref<string[]>([])

const openShareDialog = (row: SSHHostInfo) => {
  shareHostId.value = row.id
  shareUserIds.value = row.sharedUsers?.map((user) => user.userId) || []
  shareDialogOpen.value = true
}

const confirmShare = async () => {
  shareLoading.value = true
  try {
    await shareSshHost({ id: shareHostId.value, userIds: shareUserIds.value })
    feedback.success(shareUserIds.value.length ? t('ssh.common.shareSuccess') : t('ssh.common.shareAllCancelled'))
    shareDialogOpen.value = false
    getList()
  } finally {
    shareLoading.value = false
  }
}

onMounted(() => {
  getList()
})
</script>

<style scoped lang="scss">
.ssh-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding-bottom: 72px;
}
.ssh-page__em {
  color: var(--m-pInk);
  font-weight: 700;
}
.ssh-page__group {
  width: 180px;
}
/* 与搜索胶囊同高 */
.ssh-page__group :deep(.app-sel__trigger) {
  height: 40px;
  border-radius: 999px;
  padding-inline: 14px 10px;
}
.is-spinning :deep(svg) {
  animation: ssh-spin 1s linear infinite;
}

/* 批量测试结果 */
.ssh-batch {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px 18px;
  padding: 16px 18px 16px 20px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
}
.ssh-batch__av {
  width: 48px;
  height: 48px;
  border-radius: 16px;
  transform: rotate(-4deg);
}
.ssh-batch__av :deep(.iconify) {
  width: 22px;
  height: 22px;
}
.ssh-batch__main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 200px;
}
.ssh-batch__when {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.ssh-batch__nums {
  font-family: var(--app-font-display);
  font-size: 1.5rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}
.ssh-batch__nums .is-ok {
  color: var(--m-ok);
}
.ssh-batch__nums .is-bad {
  color: var(--m-bad);
}
.ssh-batch__bars {
  display: flex;
  gap: 5px;
  max-width: 100%;
  flex-wrap: wrap;
}
.ssh-batch__bar {
  width: 12px;
  height: 34px;
  border-radius: 6px;
  background: var(--el-fill-color-light);
}
.ssh-batch__bar.is-ok {
  background: var(--m-ok);
}
.ssh-batch__bar.is-bad {
  background: var(--m-bad);
}
.ssh-batch__fail {
  flex: 1 1 260px;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 0 14px;
  border: 0;
  border-radius: 14px;
  background: var(--m-badSoft);
  color: var(--m-bad);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.ssh-batch__fail :deep(.iconify) {
  width: 16px;
  height: 16px;
  flex: none;
}
.ssh-batch__fail-text {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ssh-batch__fail-cta {
  font-weight: 800;
  white-space: nowrap;
}
.ssh-batch__lat {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.ssh-batch__lat b {
  color: var(--el-text-color-primary);
}

/* 卡片 */
.ssh-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
  transition: opacity 0.15s;
}
.ssh-grid.is-loading {
  opacity: 0.6;
}
.ssh-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 18px 18px 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.ssh-card.is-bad {
  border-color: color-mix(in srgb, var(--m-bad) 45%, var(--m-line));
}
.ssh-card.is-selected {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.ssh-card__head {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.ssh-card__av {
  width: 46px;
  height: 46px;
  border-radius: 15px;
  font-size: 1.2rem;
  flex: none;
}
.ssh-card__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ssh-card__name {
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ssh-card__addr {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ssh-card__spin {
  width: 18px;
  height: 18px;
  color: var(--m-pInk);
  animation: ssh-spin 1s linear infinite;
}
/* 勾选框 20px，外面一圈透明点击区（PC 40 / 手机 44），负边距不占布局 */
.ssh-card__hit {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 40px;
  height: 40px;
  margin: -10px;
  cursor: pointer;
}
.ssh-card__check {
  width: 20px;
  height: 20px;
  margin: 0;
  flex: none;
  accent-color: var(--m-p);
  cursor: pointer;
}
@media (width < 768px) {
  .ssh-card__hit {
    width: 44px;
    height: 44px;
    margin: -12px;
  }
}
.ssh-card__tags,
.ssh-card__mtags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.ssh-card__mtags {
  flex: 1;
  min-width: 0;
  overflow: hidden;
}
.ssh-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;
}
.ssh-tag :deep(.iconify) {
  width: 13px;
  height: 13px;
}
.ssh-tag.is-share {
  background: var(--m-a2Soft);
  color: var(--m-ink);
}
.ssh-card__info {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 12px;
  margin: 0;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
}
.ssh-card__info > div {
  min-width: 0;
}
.ssh-card__info .is-wide {
  grid-column: 1 / -1;
}
.ssh-card__info dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.ssh-card__info dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ssh-card__info dd.is-bad {
  color: var(--m-bad);
}
.ssh-card__auth {
  display: flex;
  align-items: center;
  gap: 5px;
}
.ssh-card__auth :deep(.iconify) {
  width: 14px;
  height: 14px;
  color: var(--el-text-color-secondary);
}
.ssh-card__foot {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: auto;
}
.ssh-card__open {
  flex: 1;
  justify-content: center;
  min-height: 40px;
  border-radius: var(--app-radius-pill);
}
.ssh-card__foot :deep(.is-primary) {
  color: var(--m-pInk);
}

.ssh-skeleton {
  height: 260px;
  border-radius: var(--app-radius-xl);
  background: var(--el-fill-color-light);
  animation: ssh-pulse 1.4s ease-in-out infinite;
}

/* 悬浮按钮 / 批量条 */
.ssh-fab {
  position: sticky;
  bottom: 0;
  display: flex;
  justify-content: flex-end;
  pointer-events: none;
  z-index: 5;
}
.ssh-fab__btn {
  pointer-events: auto;
  min-height: 56px;
  padding: 0 22px;
  border-radius: var(--app-radius-pill);
  box-shadow: var(--app-shadow-brand);
}
.ssh-bulk {
  position: fixed;
  left: 50%;
  bottom: 24px;
  z-index: 40;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 6px;
  width: min(720px, calc(100vw - 32px));
  min-height: 60px;
  padding: 8px 8px 8px 20px;
  border-radius: var(--app-radius-pill);
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: 0 18px 40px -16px rgba(0, 0, 0, 0.45);
}
.ssh-bulk__count {
  font-weight: 800;
  white-space: nowrap;
}
.ssh-bulk__spacer {
  flex: 1;
}
.ssh-bulk__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 44px;
  padding: 0 14px;
  border: 0;
  border-radius: var(--app-radius-pill);
  background: transparent;
  color: inherit;
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.ssh-bulk__btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--m-bg) 14%, transparent);
}
.ssh-bulk__btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.ssh-bulk__btn.is-danger {
  color: color-mix(in srgb, var(--m-bad) 70%, var(--m-bg));
}
.ssh-bulk__btn :deep(.iconify),
.ssh-bulk__x :deep(.iconify) {
  width: 16px;
  height: 16px;
}
.ssh-bulk__x {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 50%;
  background: color-mix(in srgb, var(--m-bg) 14%, transparent);
  color: inherit;
  cursor: pointer;
}
.ssh-bulk-enter-active,
.ssh-bulk-leave-active {
  transition: opacity 0.18s, transform 0.18s;
}
.ssh-bulk-enter-from,
.ssh-bulk-leave-to {
  opacity: 0;
  transform: translate(-50%, 12px);
}

@keyframes ssh-spin {
  to {
    transform: rotate(360deg);
  }
}
@keyframes ssh-pulse {
  50% {
    opacity: 0.55;
  }
}

@media (width <= 768px) {
  .ssh-page {
    gap: 14px;
    padding-bottom: 8px;
  }
  .ssh-page__group {
    width: 100%;
  }
  .ssh-page__group :deep(.app-sel__trigger) {
    height: 44px;
  }
  .ssh-batch {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
    padding: 14px;
  }
  .ssh-batch > .ssh-batch__av {
    display: none;
  }
  .ssh-batch__bars {
    flex-wrap: nowrap;
    gap: 4px;
  }
  .ssh-batch__bar {
    flex: 1;
    width: auto;
    height: 8px;
    border-radius: 4px;
  }
  .ssh-batch__fail {
    flex: none;
    /* 按钮在纵向 flex 里不会自动收窄，长错误信息会把它撑出卡片 */
    width: 100%;
  }
  .ssh-batch__lat {
    display: none;
  }
  .ssh-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: 12px;
  }
  .ssh-card {
    padding: 16px 12px 6px 16px;
  }
  .ssh-card__foot {
    gap: 0;
    padding-top: 6px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
  .ssh-card__foot :deep(.action-menu__trigger) {
    width: 44px;
    height: 44px;
    border-radius: 14px;
  }
  .ssh-bulk {
    bottom: calc(80px + env(safe-area-inset-bottom));
    padding-left: 16px;
  }
  .ssh-bulk__count {
    font-size: 0.875rem;
  }
}
</style>
