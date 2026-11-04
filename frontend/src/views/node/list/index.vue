<!-- 节点列表：主控经 gRPC 接入的远程节点 + 主控本机卡片。API Key 在目标节点上创建（只作签名种子，不在网络上传输），
     其授权范围决定主控在该节点上可见的页面与操作。
     布局：标题 + 子导航 → 主色概况 hero（在线数 / 异常节点 / 平均延迟 / 正在操作）→ 状态 chips + 搜索 → 节点卡片。
     卡片操作：切换 / 测试 / 编辑 / 更多（删除）；顶部「全部测试」逐个探测远程节点。 -->
<template>
  <div class="nodes-page">
    <PageHeader :title="t('node.page.title')">
      <template #description>
        <span class="tnum">{{ t('accessV2.node.total', { total: totalCount }) }}</span>
        <span class="nodes-dot">·</span>
        <span class="nodes-online tnum">{{ t('accessV2.node.online', { online: onlineCount }) }}</span>
        <template v-if="offlineCount && !menuStore.isMobile">
          <span class="nodes-dot">·</span>
          <span class="nodes-offline tnum">{{ t('accessV2.node.offline', { count: offlineCount }) }}</span>
        </template>
      </template>
      <template #actions>
        <template v-if="!menuStore.isMobile">
          <UButton
            v-if="canManage"
            color="neutral"
            variant="outline"
            icon="i-lucide-activity"
            :loading="testingAll"
            @click="testAll"
          >
            {{ t('access.node.testAll') }}
          </UButton>
          <UButton color="neutral" variant="soft" icon="i-lucide-rotate-cw" @click="getList">
            {{ t('node.list.refresh') }}
          </UButton>
          <UButton v-if="canManage" color="primary" icon="i-lucide-plus" @click="nodeCreateRef?.showDialog()">
            {{ t('node.list.add') }}
          </UButton>
        </template>
        <template v-else>
          <UButton
            v-if="canManage"
            color="neutral"
            variant="ghost"
            icon="i-lucide-activity"
            :loading="testingAll"
            :aria-label="t('access.node.testAll')"
            @click="testAll"
          />
          <UButton
            color="neutral"
            variant="ghost"
            icon="i-lucide-rotate-cw"
            :aria-label="t('node.list.refresh')"
            @click="getList"
          />
        </template>
      </template>
    </PageHeader>
    <NodeSubNav :node-count="list.length + 1" />

    <!-- 概况 hero -->
    <section class="nodes-hero" :aria-label="t('access.node.heroTitle', { total: totalCount, online: onlineCount })">
      <span class="nodes-hero__blob" aria-hidden="true" />
      <span class="nodes-hero__ring" aria-hidden="true" />
      <div class="nodes-hero__body">
        <span class="nodes-hero__title m-num">
          {{ t('access.node.heroTitle', { total: totalCount, online: onlineCount }) }}
        </span>
        <span class="nodes-hero__sub" :title="heroSub">{{ heroSub }}</span>
      </div>
      <div class="nodes-hero__stats">
        <div class="nodes-hero__stat">
          <span class="nodes-hero__stat-label">{{ t('access.node.avgLatency') }}</span>
          <span class="nodes-hero__stat-value m-num">{{ avgLatency }}</span>
        </div>
        <div class="nodes-hero__stat is-name">
          <span class="nodes-hero__stat-label">{{ t('access.node.operating') }}</span>
          <span class="nodes-hero__stat-value is-text" :title="operatingName">{{ operatingName }}</span>
        </div>
      </div>
    </section>

    <!-- 筛选 -->
    <div class="nodes-tools">
      <div class="nodes-chips" role="tablist" :aria-label="t('node.list.status')">
        <button
          v-for="chip in chips"
          :key="chip.value"
          type="button"
          role="tab"
          class="nodes-chip"
          :class="{ 'is-on': filter === chip.value }"
          :aria-selected="filter === chip.value"
          @click="filter = chip.value"
        >
          {{ chip.label }}
          <span class="nodes-chip__n m-num">{{ chip.count }}</span>
        </button>
      </div>
      <span class="nodes-tools__spacer" />
      <label class="nodes-search">
        <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" />
        <input
          v-model="keywords"
          type="search"
          :placeholder="t('access.node.searchPlaceholder')"
          :aria-label="t('access.node.searchPlaceholder')"
          @keyup.enter="getList"
        />
      </label>
    </div>

    <!-- 卡片 -->
    <div v-if="loading && !list.length && !localInfo" class="nodes-grid">
      <div v-for="i in 2" :key="i" class="nodes-skeleton" />
    </div>
    <template v-else>
      <div v-if="visibleCards.length" class="nodes-grid">
        <NodeCard
          v-for="(card, idx) in visibleCards"
          :key="card.key"
          :index="idx"
          :local="card.local"
          :node="card.node"
          :local-info="localInfo"
          :local-permissions="localPermissions"
          :is-current="card.local ? !nodeStore.isRemote : card.node?.id === nodeStore.nodeId"
          :can-manage="canManage"
          :testing="!!card.node && testingIds.has(card.node.id)"
          :master-version="menuStore.currentVersion"
          @switch="nodeStore.switchNode(card.node?.id || '')"
          @test="card.node && runTest(card.node)"
          @edit="card.node && nodeCreateRef?.showDialog(card.node)"
          @delete="card.node && confirmDelete(card.node)"
        />
      </div>
      <EmptyState
        v-else
        icon="HOutline:ServerStackIcon"
        :title="t('node.page.noMatch')"
        :description="t('node.list.emptyDesc')"
      />
    </template>

    <!-- 还没有远程节点：接入步骤 -->
    <AppPanel v-if="!loading && !list.length && !keywords.trim()" class="nodes-guide">
      <div class="nodes-guide__head">
        <span class="nodes-guide__icon m-sticker m-sticker--a2">
          <component :is="menuStore.iconComponents['HOutline:LightBulbIcon']" />
        </span>
        <div class="nodes-guide__text">
          <h2 class="nodes-guide__title">{{ t('access.node.guideTitle') }}</h2>
          <p class="nodes-guide__desc">{{ t('access.node.heroEmpty') }}</p>
        </div>
      </div>
      <ol class="nodes-guide__steps">
        <li v-for="(step, i) in guideSteps" :key="i">
          <span class="nodes-guide__n m-num">{{ i + 1 }}</span>
          <span>{{ step }}</span>
        </li>
      </ol>
      <div v-if="canManage" class="nodes-guide__actions">
        <UButton color="primary" icon="i-lucide-plus" @click="nodeCreateRef?.showDialog()">
          {{ t('node.list.add') }}
        </UButton>
        <UButton color="neutral" variant="soft" icon="i-lucide-key-round" :to="pageTabTo('node', 'key')">
          {{ t('node.key.title') }}
        </UButton>
      </div>
    </AppPanel>

    <!-- 手机：悬浮添加 -->
    <div v-if="menuStore.isMobile && canManage" class="nodes-fab">
      <UButton color="primary" size="xl" icon="i-lucide-plus" class="nodes-fab__btn" @click="nodeCreateRef?.showDialog()">
        {{ t('node.list.add') }}
      </UButton>
    </div>

    <NodeCreate ref="nodeCreateRef" @refresh="onFormRefresh" />

    <FormDialog v-model="testDialog.open" :title="t('node.list.testTitle', { name: testDialog.name })" :width="520" :show-footer="false">
      <div v-if="testDialog.loading" class="nodes-test-loading">
        <span class="nodes-spinner" />
        <span>{{ t('node.list.testing') }}</span>
      </div>
      <NodeTestResult v-else-if="testDialog.status" :status="testDialog.status" />
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { pageTabTo } from '@/config/pageGroups'
import { useDebounceFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { deleteNode, listNodes, testNode } from '@/api/node'
import { NodeState, type NodeInfo, type NodeRemoteInfo, type NodeStatus } from '@/types/v1/node'
import { NODE_PERMISSION_ITEMS } from '@/config/nodePermission'
import { Dialog } from '@/utils/dialog'
import NodeCreate from '@/views/node/list/create.vue'
import NodeTestResult from '@/views/node/list/testResult.vue'
import NodeCard from '@/views/node/list/nodeCard.vue'
import NodeSubNav from '@/views/node/components/NodeSubNav.vue'
import { useNodeTime } from '@/views/node/components/useNodeTime'

defineOptions({ name: 'NodeListView' })

type Filter = 'all' | 'online' | 'offline'

const menuStore = useMenuStore()
const nodeStore = useNodeStore()
const { t } = useI18n()
const { ago, shortTime } = useNodeTime()
const nodeCreateRef = useTemplateRef<InstanceType<typeof NodeCreate> | null>('nodeCreateRef')

const loading = ref(false)
const list = ref<NodeInfo[]>([])
const localInfo = ref<NodeRemoteInfo | undefined>()
const keywords = ref('')
const filter = ref<Filter>('all')
const testDialog = ref({ open: false, loading: false, name: '', status: null as NodeStatus | null })
const testingIds = ref<Set<string>>(new Set())
const testingAll = ref(false)

const canManage = computed(() => menuStore.hasButtonPermission('node:manage'))

const isOnline = (node: NodeInfo) => node.status?.state === NodeState.NodeState_Online

// 本机卡片：当前用户在主控上拥有的节点类功能（仅需登录的项恒为允许）
const localPermissions = computed(() =>
  NODE_PERMISSION_ITEMS.filter((i) => i.loginOnly || menuStore.buttonPermissions.includes(i.code)).map((i) => i.code),
)

const localMatches = computed(() => {
  const kw = keywords.value.trim().toLowerCase()
  if (!kw) return true
  return [localInfo.value?.hostname, t('node.page.localName'), window.location.host]
    .filter(Boolean)
    .some((s) => String(s).toLowerCase().includes(kw))
})

// 计数含本机（本机恒在线）
const remoteOnline = computed(() => list.value.filter(isOnline).length)
const totalCount = computed(() => list.value.length + 1)
const onlineCount = computed(() => remoteOnline.value + 1)
const offlineCount = computed(() => list.value.length - remoteOnline.value)

const chips = computed(() => [
  { value: 'all' as Filter, label: t('node.page.filterAll'), count: list.value.length + (localMatches.value ? 1 : 0) },
  { value: 'online' as Filter, label: t('node.state.online'), count: remoteOnline.value + (localMatches.value ? 1 : 0) },
  { value: 'offline' as Filter, label: t('node.state.offline'), count: list.value.length - remoteOnline.value },
])

interface Card {
  key: string
  local: boolean
  node?: NodeInfo
}
const visibleCards = computed<Card[]>(() => {
  const cards: Card[] = []
  if (localMatches.value && filter.value !== 'offline') cards.push({ key: 'local', local: true })
  for (const node of list.value) {
    if (filter.value === 'online' && !isOnline(node)) continue
    if (filter.value === 'offline' && isOnline(node)) continue
    cards.push({ key: node.id, local: false, node })
  }
  return cards
})

// —— hero 文案 ——
const problemNodes = computed(() => list.value.filter((n) => !isOnline(n) && n.status?.state !== NodeState.NodeState_Unknown))
const heroSub = computed(() => {
  if (!list.value.length) return t('access.node.heroEmpty')
  const first = problemNodes.value[0]
  if (!first) return t('access.node.heroAllOnline')
  const s = first.status
  let text: string
  if (s?.lastOnlineTime) {
    text = t('node.page.heroOffline', { name: first.name, ago: ago(s.lastOnlineTime), time: shortTime(s.lastOnlineTime) })
  } else {
    text = t('node.page.heroNeverOnline', { name: first.name, reason: s?.message || t('node.state.offline') })
  }
  const more = problemNodes.value.length - 1
  return more > 0 ? text + t('access.node.heroMoreIssues', { count: more }) : text
})
const avgLatency = computed(() => {
  const online = list.value.filter(isOnline)
  if (!online.length) return '—'
  const sum = online.reduce((acc, n) => acc + Number(n.status?.latencyMs || 0), 0)
  return `${Math.round(sum / online.length)} ms`
})
const operatingName = computed(() => {
  if (nodeStore.isRemote) {
    return list.value.find((n) => n.id === nodeStore.nodeId)?.name || nodeStore.contextInfo?.name || '—'
  }
  return localInfo.value?.hostname || t('node.page.localName')
})

const guideSteps = computed(() => [t('node.list.guideStep1'), t('node.list.guideStep2'), t('node.list.guideStep3')])

// —— 数据 ——
const getList = async () => {
  loading.value = true
  try {
    const { data } = await listNodes({ keywords: keywords.value.trim() || undefined })
    list.value = data?.infos || []
    localInfo.value = data?.local
    if (!keywords.value.trim()) nodeStore.nodes = list.value
  } finally {
    loading.value = false
  }
}
const debouncedList = useDebounceFn(() => void getList(), 350)
watch(keywords, () => debouncedList())

const markTesting = (id: string, on: boolean) => {
  const next = new Set(testingIds.value)
  if (on) next.add(id)
  else next.delete(id)
  testingIds.value = next
}

const runTest = async (node: NodeInfo) => {
  testDialog.value = { open: true, loading: true, name: node.name, status: null }
  markTesting(node.id, true)
  try {
    const { data } = await testNode({ id: node.id })
    testDialog.value.status = data?.status || null
    void getList()
  } catch {
    testDialog.value.open = false
  } finally {
    testDialog.value.loading = false
    markTesting(node.id, false)
  }
}

const testAll = async () => {
  if (!list.value.length) {
    feedback.info(t('node.page.testAllEmpty'))
    return
  }
  testingAll.value = true
  const targets = [...list.value]
  targets.forEach((n) => markTesting(n.id, true))
  try {
    await Promise.allSettled(
      targets.map((n) => testNode({ id: n.id }).finally(() => markTesting(n.id, false))),
    )
    await getList()
    feedback.success(t('access.node.testAllDone', { count: targets.length }))
  } finally {
    testingAll.value = false
  }
}

const confirmDelete = (node: NodeInfo) => {
  Dialog.confirm({
    title: t('node.list.deleteTitle'),
    content: t('node.list.deleteContent', { name: node.name }),
    confirmText: t('node.list.confirm'),
    cancelText: t('node.list.cancel'),
    onConfirm: async () => {
      await deleteNode({ id: node.id })
      feedback.success(t('node.list.deleteSuccess'))
      // 删除的是当前节点：回到本机
      if (node.id === nodeStore.nodeId) {
        nodeStore.switchNode('')
        return
      }
      getList()
    },
  })
}

const onFormRefresh = () => {
  getList()
  // 新增 / 编辑后后台会异步探测一次，稍后刷新状态
  setTimeout(() => void getList(), 2500)
}

onMounted(() => {
  getList()
})
</script>

<style scoped lang="scss">
.nodes-page {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.nodes-dot {
  margin: 0 6px;
  opacity: 0.6;
}
.nodes-online {
  color: var(--m-ok);
  font-weight: 700;
}
.nodes-offline {
  color: var(--m-bad);
  font-weight: 700;
}

/* hero */
.nodes-hero {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 18px 22px;
  padding: 22px 26px;
  border-radius: var(--app-radius-lg);
  background: var(--m-p);
  color: var(--m-pOn);
}
.nodes-hero__blob {
  position: absolute;
  right: -50px;
  bottom: -90px;
  width: 220px;
  height: 220px;
  border-radius: 50%;
  background: var(--m-a);
}
.nodes-hero__ring {
  position: absolute;
  right: 380px;
  top: -34px;
  width: 78px;
  height: 78px;
  border-radius: 50%;
  border: 12px solid var(--m-a2);
}
.nodes-hero__body {
  position: relative;
  flex: 1 1 360px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.nodes-hero__title {
  font-family: var(--app-font-display);
  font-size: 2rem;
  line-height: 1.15;
  font-weight: 800;
  letter-spacing: -0.01em;
}
.nodes-hero__sub {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  font-size: 0.875rem;
  line-height: 1.5;
  word-break: break-word;
}
.nodes-hero__stats {
  position: relative;
  display: flex;
  gap: 10px;
  margin-right: 150px;
  min-width: 0;
}
.nodes-hero__stat {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 10px 16px;
  border-radius: 18px;
  background: var(--m-pOn);
  color: var(--m-ink);
}
.nodes-hero__stat.is-name {
  max-width: 220px;
}
.nodes-hero__stat-label {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--m-ink2);
}
.nodes-hero__stat-value {
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
  white-space: nowrap;
}
.nodes-hero__stat-value.is-text {
  padding-top: 3px;
  font-size: 0.9375rem;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 工具条 */
.nodes-tools {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.nodes-tools__spacer {
  flex: 1;
}
.nodes-chips {
  display: flex;
  gap: 8px;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.nodes-chips::-webkit-scrollbar {
  display: none;
}
.nodes-chip {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}
.nodes-chip:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.nodes-chip.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.nodes-chip__n {
  font-size: 0.75rem;
  opacity: 0.7;
}
.nodes-search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 300px;
  min-width: 180px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  transition: border-color 0.15s;
}
.nodes-search:focus-within {
  border-color: var(--m-p);
}
.nodes-search :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
}
.nodes-search input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
}

/* 卡片 */
.nodes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 460px), 1fr));
  gap: 16px;
}
.nodes-skeleton {
  height: 300px;
  border-radius: var(--app-radius-lg);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: nodes-shimmer 1.4s ease-in-out infinite;
}
@keyframes nodes-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}

/* 接入步骤 */
.nodes-guide {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.nodes-guide__head {
  display: flex;
  align-items: center;
  gap: 14px;
}
.nodes-guide__icon {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  flex: none;
  transform: rotate(-4deg);
}
.nodes-guide__icon :deep(svg) {
  width: 22px;
  height: 22px;
}
.nodes-guide__title {
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.nodes-guide__desc {
  margin-top: 2px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.nodes-guide__steps {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 240px), 1fr));
  gap: 10px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.nodes-guide__steps li {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  font-size: 0.8125rem;
  line-height: 1.55;
  color: var(--el-text-color-regular);
}
.nodes-guide__n {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  flex: none;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-size: 0.75rem;
  font-weight: 800;
}
.nodes-guide__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

/* 手机 */
.nodes-fab {
  position: sticky;
  bottom: 0;
  display: flex;
  justify-content: flex-end;
  pointer-events: none;
  z-index: 5;
}
.nodes-fab__btn {
  pointer-events: auto;
  min-height: 56px;
  padding: 0 22px;
  border-radius: var(--app-radius-pill);
  box-shadow: var(--app-shadow-brand);
}
@media (width < 768px) {
  .nodes-page {
    gap: 14px;
  }
  .nodes-chip,
  .nodes-search {
    min-height: 44px;
  }
  .nodes-hero {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
    padding: 16px 18px;
  }
  .nodes-hero__blob {
    right: -40px;
    bottom: -70px;
    width: 150px;
    height: 150px;
  }
  .nodes-hero__ring {
    right: -18px;
    top: -22px;
    width: 56px;
    height: 56px;
    border-width: 9px;
  }
  .nodes-hero__body {
    flex: none;
  }
  .nodes-hero__title {
    font-size: 1.625rem;
  }
  .nodes-hero__sub {
    font-size: 0.8125rem;
    padding-right: 40px;
  }
  .nodes-hero__stats {
    margin-right: 0;
    gap: 8px;
  }
  .nodes-hero__stat {
    padding: 8px 12px;
    border-radius: 14px;
  }
  .nodes-hero__stat.is-name {
    flex: 0 1 auto;
    max-width: 60%;
  }
  .nodes-hero__stat-value {
    font-size: 1rem;
  }
  .nodes-hero__stat-value.is-text {
    font-size: 0.875rem;
  }
  .nodes-tools__spacer {
    display: none;
  }
  .nodes-search {
    order: -1;
    flex: 1 1 100%;
    min-height: 44px;
  }
  .nodes-chips {
    width: 100%;
  }
  .nodes-grid {
    gap: 12px;
  }
}

.nodes-test-loading {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 4px;
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
}
.nodes-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--el-border-color);
  border-top-color: var(--m-p);
  border-radius: 999px;
  animation: nodes-spin 0.8s linear infinite;
}
@keyframes nodes-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
