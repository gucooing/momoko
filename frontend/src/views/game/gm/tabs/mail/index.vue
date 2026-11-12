<!-- GM 运营 · 邮件补偿：左边邮件列表（状态 chips、搜索、领取进度），右边编写 / 详情。
     邮件经服务端 mail.broadcast 投递到礼物箱；投递与撤回在后台执行，期间每 3 秒刷新一次。 -->
<template>
  <div class="gmm game-skin">
    <div class="gmm__head">
      <div class="gmm__head-text">
        <h2 class="gmm__h">{{ t('gmOps.mail.title') }}</h2>
        <p class="gmm__sub">{{ subline }}</p>
      </div>
      <UButton color="primary" icon="i-lucide-pencil" @click="newMail">{{ t('gmOps.mail.write') }}</UButton>
    </div>

    <div class="gmm__body">
      <section class="gmm__list" :aria-label="t('gmOps.mail.listLabel')">
        <div class="gmm__toolbar">
          <ListToolbar
            v-model:status="status"
            v-model:keyword="keyword"
            :chips="chips"
            :chips-label="t('gmOps.mail.statusLabel')"
            :placeholder="t('gmOps.mail.search')"
            @search="reload(1)"
          />
        </div>
        <div v-if="loading && !items.length" class="gmm__loading">{{ t('gmOps.common.loading') }}</div>
        <EmptyState
          v-else-if="!items.length"
          icon="HOutline:EnvelopeIcon"
          :title="keyword || status ? t('gmOps.mail.emptyFiltered') : t('gmOps.mail.empty')"
          :description="keyword || status ? '' : t('gmOps.mail.emptyDesc')"
        />
        <button
          v-for="m in items"
          :key="m.id"
          type="button"
          class="gmm__item"
          :class="{ 'is-on': current?.id === m.id }"
          :aria-pressed="current?.id === m.id"
          @click="select(m)"
        >
          <span class="gmm__icon m-sticker" :class="stickerOf(m)" aria-hidden="true"><UIcon :name="iconOf(m)" /></span>
          <span class="gmm__item-text">
            <span class="gmm__item-top">
              <span class="gmm__item-title">{{ m.title || t('gmOps.mail.untitled') }}</span>
              <StatusPill :variant="pillOf(m.status)" :label="t(`gmOps.mail.status.${m.status}`)" />
            </span>
            <span class="gmm__hint">{{ toLine(m) }} · {{ atLine(m) }}</span>
            <span v-if="m.rewards.length" class="gmm__rewards">
              <template v-for="(r, i) in m.rewards.slice(0, 3)" :key="i">
                <GameImage v-if="label(r).image" :src="label(r).image" class="gmm__reward-img" width="24" height="24" />
              </template>
              <span class="gmm__hint gmm__reward-sum">{{ summary(m.rewards) }}</span>
            </span>
            <span v-if="m.status === 'sent' || m.status === 'revoked' || m.status === 'revoking'" class="gmm__claim">
              <span class="gmm__bar"><span :style="{ width: `${claimPct(m)}%` }" /></span>
              <span class="gmm__hint m-num">{{ t('gmOps.mail.claimedShort', { n: m.claimedCount, total: m.recipientCount }) }}</span>
            </span>
          </span>
        </button>
        <div v-if="total > pageSize" class="gmm__pager">
          <Pagination v-model:page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[20, 50, 100]" @change="reload()" />
        </div>
      </section>

      <section ref="editorRef" class="gmm__editor">
        <MailEditor
          :key="editorKey"
          :mail="current"
          @changed="onChanged"
          @deleted="onDeleted"
          @duplicated="onDuplicated"
        />
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGameMail, listGameMails } from '@/api/gameOps'
import type { GameMail } from '@/types/v1/game_ops'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import { useGmContext } from '../../context'
import MailEditor from './MailEditor.vue'
import { fmtDateTime, fmtShort, useRewardLabels } from './shared'

defineOptions({ name: 'GmMailTab' })

type Pill = 'success' | 'warning' | 'error' | 'info' | 'neutral' | 'primary'

const { t } = useI18n()
const ctx = useGmContext()
const menuStore = useMenuStore()
const { label, summary } = useRewardLabels()

const status = ref('')
const keyword = ref('')
const page = ref(1)
const pageSize = ref(20)
const items = ref<GameMail[]>([])
const total = ref(0)
const counts = ref<Record<string, number>>({})
const sentThisMonth = ref(0)
const loading = ref(false)
const current = ref<GameMail>()
// 新建时用递增的 key 重建编辑器（清空表单）
const editorKey = ref('new-0')
const editorRef = ref<HTMLElement>()

const chips = computed<ToolbarChip<string>[]>(() => [
  { label: t('gmOps.mail.chips.all'), value: '', count: counts.value.all ?? null },
  { label: t('gmOps.mail.chips.sent'), value: 'sent', count: counts.value.sent ?? null },
  { label: t('gmOps.mail.chips.scheduled'), value: 'scheduled', count: counts.value.scheduled ?? null },
  { label: t('gmOps.mail.chips.draft'), value: 'draft', count: counts.value.draft ?? null },
])

const subline = computed(() =>
  [
    t('gmOps.mail.subline'),
    t('gmOps.mail.sentThisMonth', { n: sentThisMonth.value }),
    t('gmOps.mail.scheduledCount', { n: counts.value.scheduled || 0 }),
    t('gmOps.mail.draftCount', { n: counts.value.draft || 0 }),
  ].join(' · '),
)

let seq = 0
const reload = async (toPage?: number) => {
  if (toPage) page.value = toPage
  const mine = ++seq
  loading.value = true
  try {
    const { data } = await listGameMails({
      serverId: ctx.serverId.value,
      status: status.value,
      q: keyword.value.trim(),
      page: page.value,
      pageSize: pageSize.value,
    })
    if (mine !== seq) return
    items.value = data.items || []
    total.value = data.total || 0
    counts.value = data.counts || {}
    sentThisMonth.value = data.sentThisMonth || 0
    // 当前打开的邮件同步成列表里的最新状态
    const cur = current.value
    if (cur) {
      const fresh = items.value.find((m) => m.id === cur.id)
      if (fresh && (fresh.status !== cur.status || fresh.claimedCount !== cur.claimedCount || fresh.statsAt !== cur.statsAt)) current.value = fresh
    }
  } catch {
    // 错误已由请求层提示
  } finally {
    if (mine === seq) loading.value = false
  }
}

const scrollToEditor = async () => {
  if (!menuStore.isMobile) return
  await nextTick()
  editorRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const select = async (m: GameMail) => {
  if (current.value?.id === m.id) {
    void scrollToEditor()
    return
  }
  current.value = m
  editorKey.value = m.id
  void scrollToEditor()
  try {
    const { data } = await getGameMail(ctx.serverId.value, m.id)
    if (current.value?.id === data.id) current.value = data
  } catch {
    // 用列表里的数据
  }
}

let newCount = 0
const newMail = () => {
  current.value = undefined
  editorKey.value = `new-${++newCount}`
  void scrollToEditor()
}

const upsert = (m: GameMail) => {
  const i = items.value.findIndex((x) => x.id === m.id)
  if (i >= 0) items.value.splice(i, 1, m)
}

const onChanged = (m: GameMail) => {
  const isNew = !items.value.some((x) => x.id === m.id)
  current.value = m
  // 新建的邮件：编辑器 key 保持不变（不重建），只是换上服务端的邮件
  upsert(m)
  if (isNew || m.status !== 'draft') void reload()
}

const onDeleted = (id: string) => {
  items.value = items.value.filter((m) => m.id !== id)
  newMail()
  void reload()
}

const onDuplicated = (m: GameMail) => {
  current.value = m
  editorKey.value = m.id
  void reload()
}

// ---- 列表行 ----
const pillOf = (s: string): Pill => {
  switch (s) {
    case 'sent':
      return 'success'
    case 'scheduled':
      return 'info'
    case 'sending':
    case 'revoking':
      return 'primary'
    case 'failed':
      return 'error'
    default:
      return 'neutral'
  }
}
const stickerOf = (m: GameMail) => {
  switch (m.status) {
    case 'sent':
      return 'm-sticker--ok'
    case 'scheduled':
      return 'm-sticker--a2'
    case 'failed':
      return 'm-sticker--bad'
    default:
      return 'm-sticker--p'
  }
}
const iconOf = (m: GameMail) => {
  switch (m.status) {
    case 'draft':
      return 'i-lucide-pencil'
    case 'scheduled':
      return 'i-lucide-clock'
    case 'revoked':
      return 'i-lucide-undo-2'
    case 'failed':
      return 'i-lucide-triangle-alert'
    default:
      return 'i-lucide-mail'
  }
}
const toLine = (m: GameMail) => {
  const target = t(`gmOps.mail.target.${m.target}`)
  if (m.target === 'players') return `${target} · ${t('gmOps.mail.people', { n: m.players?.length || 0 })}`
  if (m.recipientCount && m.status !== 'draft' && m.status !== 'scheduled') return `${target} · ${t('gmOps.mail.people', { n: m.recipientCount })}`
  if (m.matchedAt) return `${target} · ${t('gmOps.mail.people', { n: m.matchedCount })}`
  return target
}
const today = computed(() => t('gmOps.common.today'))
const atLine = (m: GameMail) => {
  switch (m.status) {
    case 'draft':
      return t('gmOps.mail.editedShort', { time: fmtShort(m.updateTime, today.value) })
    case 'scheduled':
      return t('gmOps.mail.scheduledShort', { time: fmtDateTime(m.scheduledAt) })
    case 'revoked':
      return t('gmOps.mail.revokedShort', { time: fmtShort(m.revokedAt, today.value) })
    case 'failed':
      return t('gmOps.mail.failedShort', { time: fmtShort(m.updateTime, today.value) })
    case 'sending':
    case 'revoking':
      return t(`gmOps.mail.status.${m.status}`)
    default:
      return `${fmtShort(m.sentAt, today.value)} · ${m.sentByName || '—'}`
  }
}
const claimPct = (m: GameMail) => (m.recipientCount ? Math.min(100, Math.round((m.claimedCount / m.recipientCount) * 100)) : 0)

// ---- 后台投递 / 撤回中：轮询 ----
let timer: ReturnType<typeof setInterval> | undefined
let active = true
const busy = computed(
  () =>
    items.value.some((m) => m.status === 'sending' || m.status === 'revoking') ||
    current.value?.status === 'sending' ||
    current.value?.status === 'revoking',
)
const stopPoll = () => {
  if (timer) clearInterval(timer)
  timer = undefined
}
const poll = async () => {
  await reload()
  const cur = current.value
  if (cur && (cur.status === 'sending' || cur.status === 'revoking')) {
    try {
      const { data } = await getGameMail(ctx.serverId.value, cur.id)
      if (current.value?.id === data.id) current.value = data
    } catch {
      // 下次再试
    }
  }
}
const syncPoll = () => {
  stopPoll()
  if (active && busy.value) timer = setInterval(() => void poll(), 3000)
}
watch(busy, syncPoll)

let activatedOnce = false
onMounted(() => void reload(1))
onActivated(() => {
  active = true
  if (activatedOnce) void reload()
  activatedOnce = true
  syncPoll()
})
onDeactivated(() => {
  active = false
  stopPoll()
})
onBeforeUnmount(stopPoll)
</script>

<style scoped>
.gmm {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.gmm__head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 12px 16px;
}
.gmm__head-text {
  flex: 1 1 320px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.gmm__h {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 30px;
  font-weight: 800;
  letter-spacing: -0.01em;
}
.gmm__sub {
  margin: 0;
  font-size: 14px;
  color: var(--el-text-color-secondary);
}
.gmm__body {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-start;
}
.gmm__list,
.gmm__editor {
  min-width: 0;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.gmm__list {
  flex: 1 1 360px;
  overflow: hidden;
}
.gmm__editor {
  flex: 2 1 600px;
  padding: 20px 24px;
}
.gmm__toolbar {
  padding: 14px 14px 12px;
}
.gmm__toolbar :deep(.ltb__search) {
  flex: 1 1 100%;
}
.gmm__loading {
  padding: 24px;
  text-align: center;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.gmm__item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  width: 100%;
  padding: 14px;
  border: 0;
  border-top: 1px solid var(--el-border-color-light);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gmm__item:hover {
  background: var(--el-fill-color-light);
}
.gmm__item.is-on {
  background: var(--m-gSoft, var(--m-pSoft));
  box-shadow: inset 3px 0 0 var(--m-g, var(--m-p));
}
.gmm__icon {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 13px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}
.gmm__item-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.gmm__item-top {
  display: flex;
  align-items: center;
  gap: 8px;
}
.gmm__item-title {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gmm__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.gmm__rewards {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}
.gmm__reward-img {
  flex: none;
  width: 24px;
  height: 24px;
  object-fit: contain;
}
.gmm__reward-sum {
  margin-left: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gmm__claim {
  display: flex;
  align-items: center;
  gap: 8px;
}
.gmm__bar {
  flex: 1;
  height: 6px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}
.gmm__bar span {
  display: block;
  height: 100%;
  background: var(--el-text-color-secondary);
}
.gmm__pager {
  padding: 12px 14px;
  border-top: 1px solid var(--el-border-color-light);
}
@media (width < 768px) {
  .gmm__h {
    font-size: 24px;
  }
  .gmm__editor {
    padding: 16px;
  }
  .gmm__item {
    min-height: 44px;
  }
}
</style>
