<!-- 邮件编辑 / 详情：收件人 → 奖励 → 内容（含游戏内礼物箱预览）→ 发送。
     草稿自动保存；定时邮件可以改内容、取消定时或立即发送；已发送、失败、已撤回的邮件只读，可复制一份再改。
     试运行与发送都经服务端 mail.broadcast 执行并写进 GM 记录。 -->
<template>
  <div class="me">
    <header class="me__head">
      <span class="me__head-text">
        <span class="me__title">{{ headTitle }}</span>
        <span class="me__meta">{{ headMeta }}</span>
      </span>
      <GmRiskPill v-if="giftOperation" :risk="giftOperation.risk" />
      <ActionMenu v-if="mail && menuItems.length" :items="menuItems" @select="onMenu" />
    </header>

    <div v-if="mail && (mail.status === 'sending' || mail.status === 'revoking')" class="me__banner me__banner--info">
      <UIcon name="i-lucide-loader-circle" class="me__spin" />
      {{ mail.status === 'sending' ? t('gmOps.mail.sendingHint') : t('gmOps.mail.revokingHint') }}
    </div>
    <div v-if="mail?.error" class="me__banner me__banner--bad">
      <UIcon name="i-lucide-triangle-alert" />
      <span>{{ errorText(mail.error) }}</span>
    </div>

    <!-- 已发送：领取进度 -->
    <section v-if="mail && isDelivered" class="me__claim">
      <div class="me__claim-row">
        <span class="me__claim-label">{{ t('gmOps.mail.claimed') }}</span>
        <span class="me__claim-num m-num">{{ mail.claimedCount }} / {{ mail.recipientCount }}</span>
        <span class="me__spacer" />
        <span v-if="mail.statsAt" class="me__hint">{{ t('gmOps.mail.statsAt', { time: fmtDateTime(mail.statsAt) }) }}</span>
      </div>
      <div class="me__bar"><span :style="{ width: `${claimPercent}%` }" /></div>
      <div class="me__claim-facts">
        <span>{{ t('gmOps.mail.delivered', { n: mail.recipientCount }) }}</span>
        <span v-if="mail.failedCount" class="me__warn">{{ t('gmOps.mail.deliverFailed', { n: mail.failedCount }) }}</span>
        <span v-if="mail.status === 'revoked'">{{ t('gmOps.mail.revokedFacts', { n: mail.revokedCount, time: fmtDateTime(mail.revokedAt), name: mail.revokedByName || '—' }) }}</span>
      </div>
    </section>

    <!-- 1 收件人 -->
    <section class="me__step">
      <div class="me__step-head"><span class="me__sn">1</span>{{ t('gmOps.mail.recipients') }}</div>
      <div class="me__seg" role="radiogroup" :aria-label="t('gmOps.mail.recipients')">
        <button
          v-for="opt in targetOptions"
          :key="opt.value"
          type="button"
          role="radio"
          class="me__seg-btn"
          :class="{ 'is-on': form.target === opt.value }"
          :aria-checked="form.target === opt.value"
          :disabled="!editable"
          @click="form.target = opt.value"
        >
          {{ opt.label }}
        </button>
      </div>

      <PlayerPicker v-if="form.target === 'players'" v-model="form.players" :disabled="!editable" />

      <div v-if="form.target === 'condition'" class="me__conds">
        <div v-for="(row, i) in form.conds" :key="i" class="me__cond">
          <AppSelect v-model="row.field" :options="fieldOptions" :disabled="!editable" />
          <AppSelect v-model="row.op" :options="opOptions" :disabled="!editable" />
          <input v-model="row.value" type="datetime-local" class="me__input" :aria-label="condLabel(row)" :disabled="!editable" />
          <AppIconButton v-if="editable" icon="HOutline:XMarkIcon" :label="t('gmOps.mail.removeCond')" :box="40" @click="form.conds.splice(i, 1)" />
        </div>
        <div v-if="!form.conds.length" class="me__hint">{{ t('gmOps.mail.noCond') }}</div>
        <div class="me__cond-foot">
          <UButton v-if="editable && form.conds.length < 4" color="neutral" variant="soft" size="sm" icon="i-lucide-plus" @click="addCond">
            {{ t('gmOps.mail.addCond') }}
          </UButton>
          <span class="me__hint">{{ t('gmOps.mail.timezone', { zone: zone }) }}</span>
        </div>
      </div>

      <div v-if="form.target === 'all'" class="me__note me__note--warn">
        <UIcon name="i-lucide-triangle-alert" class="me__note-icon" />
        <span>{{ allHint }}</span>
      </div>

      <div class="me__recip-foot">
        <label class="me__check">
          <input v-model="form.excludeBanned" type="checkbox" :disabled="!editable" />
          {{ t('gmOps.mail.excludeBanned') }}
        </label>
        <span class="me__spacer" />
        <template v-if="matchedKnown">
          <span class="me__hint">{{ t('gmOps.mail.matchedLabel') }}</span>
          <span class="me__matched m-num">{{ mail?.matchedCount }}</span>
          <span v-if="registered >= 0" class="me__hint">{{ t('gmOps.mail.matchedOf', { n: registered }) }}</span>
        </template>
        <span v-else-if="editable" class="me__hint">{{ t('gmOps.mail.matchedUnknown') }}</span>
      </div>
    </section>

    <!-- 2 奖励 -->
    <section class="me__step">
      <div class="me__step-head">
        <span class="me__sn">2</span>{{ t('gmOps.mail.rewards') }}
        <span class="me__spacer" />
        <span class="me__hint">{{ t('gmOps.mail.rewardsHint') }}</span>
      </div>
      <GmRewardPicker v-model="form.rewards" :max="rewardMax" :exclude="[RESOURCE_JEWEL_PAID]" :disabled="!editable" />
    </section>

    <!-- 3 内容 + 预览 -->
    <section class="me__step me__step--split">
      <div class="me__content">
        <div class="me__step-head"><span class="me__sn">3</span>{{ t('gmOps.mail.content') }}</div>
        <label class="me__field">
          <span class="me__label">{{ t('gmOps.mail.titleLabel') }}</span>
          <input v-model="form.title" type="text" maxlength="64" class="me__input" :placeholder="t('gmOps.mail.titlePlaceholder')" :disabled="!editable" />
          <span class="me__hint">{{ t('gmOps.mail.titleHint', { n: form.title.length }) }}</span>
        </label>
        <label class="me__field">
          <span class="me__label">{{ t('gmOps.mail.body') }}</span>
          <textarea v-model="form.body" rows="4" maxlength="2000" class="me__input me__textarea" :placeholder="t('gmOps.mail.bodyPlaceholder')" :disabled="!editable" />
          <span class="me__hint">{{ t('gmOps.mail.bodyHint') }}</span>
        </label>
        <div class="me__field">
          <span class="me__label">{{ t('gmOps.mail.expire') }}</span>
          <div class="me__seg" role="radiogroup" :aria-label="t('gmOps.mail.expire')">
            <button
              v-for="d in expireOptions"
              :key="d"
              type="button"
              role="radio"
              class="me__seg-btn"
              :class="{ 'is-on': form.expireDays === d }"
              :aria-checked="form.expireDays === d"
              :disabled="!editable"
              @click="form.expireDays = d"
            >
              {{ d ? t('gmOps.mail.days', { n: d }) : t('gmOps.mail.noExpire') }}
            </button>
          </div>
        </div>
      </div>
      <div class="me__preview">
        <span class="me__label-sm">{{ t('gmOps.mail.preview') }}</span>
        <div class="me__preview-box">
          <div class="me__gift">
            <GameImage v-if="firstImage" :src="firstImage" class="me__gift-img" width="48" height="48" />
            <span v-else class="me__gift-img me__gift-img--empty"><UIcon name="i-lucide-gift" /></span>
            <span class="me__gift-text">
              <span class="me__gift-title">{{ form.title || t('gmOps.mail.titlePlaceholder') }}</span>
              <span class="me__gift-sum">{{ rewardSummary || t('gmOps.mail.noRewards') }}</span>
              <span class="me__hint">{{ form.expireDays ? t('gmOps.mail.expireLeft', { n: form.expireDays }) : t('gmOps.mail.noExpireHint') }}</span>
            </span>
            <span class="me__gift-btn">{{ t('gmOps.mail.claim') }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- 4 发送 -->
    <section v-if="canSend" class="me__step">
      <div class="me__step-head"><span class="me__sn">4</span>{{ t('gmOps.mail.send') }}</div>
      <div class="me__send">
        <div class="me__send-when">
          <div class="me__seg" role="radiogroup" :aria-label="t('gmOps.mail.when')">
            <button type="button" role="radio" class="me__seg-btn" :class="{ 'is-on': form.when === 'now' }" :aria-checked="form.when === 'now'" @click="form.when = 'now'">
              {{ t('gmOps.mail.sendNow') }}
            </button>
            <button type="button" role="radio" class="me__seg-btn" :class="{ 'is-on': form.when === 'later' }" :aria-checked="form.when === 'later'" @click="form.when = 'later'">
              {{ t('gmOps.mail.sendLater') }}
            </button>
          </div>
          <template v-if="form.when === 'later'">
            <input v-model="form.scheduledAt" type="datetime-local" class="me__input" :min="minSchedule" :aria-label="t('gmOps.mail.scheduleAt')" />
            <span class="me__hint">{{ t('gmOps.mail.timezone', { zone: zone }) }}</span>
          </template>
        </div>
      </div>
    </section>

    <footer class="me__foot">
      <span class="me__hint me__foot-hint">
        {{ t('gmOps.mail.footHint') }}
        <template v-if="savedAt"> · {{ t('gmOps.mail.autoSaved', { time: savedAt }) }}</template>
      </span>
      <template v-if="editable">
        <UButton v-if="!mail || mail.status === 'draft'" color="neutral" variant="outline" :loading="busy === 'save'" :disabled="!!busy" @click="save(true)">
          {{ t('gmOps.mail.saveDraft') }}
        </UButton>
        <UButton v-else color="neutral" variant="outline" :loading="busy === 'save'" :disabled="!!busy" @click="save(true)">
          {{ t('gmOps.mail.saveChanges') }}
        </UButton>
        <UButton color="neutral" variant="soft" icon="i-lucide-eye" :loading="busy === 'dry'" :disabled="!!busy || !ready" @click="dryRun">
          {{ t('gmOps.mail.dryRun') }}
        </UButton>
      </template>
      <UButton v-if="mail?.status === 'scheduled'" color="neutral" variant="outline" icon="i-lucide-calendar-x" :disabled="!!busy" @click="cancelSchedule">
        {{ t('gmOps.mail.cancelSchedule') }}
      </UButton>
      <UButton v-if="canSend" color="primary" icon="i-lucide-send" :loading="busy === 'send'" :disabled="!!busy || !ready || (form.when === 'later' && !scheduleValid)" @click="send">
        {{ cta }}
      </UButton>
      <UButton v-if="mail && isDelivered && mail.status === 'sent'" color="error" variant="soft" icon="i-lucide-undo-2" :disabled="!!busy" @click="revokeOpen = true">
        {{ t('gmOps.mail.revoke') }}
      </UButton>
      <UButton v-if="mail && !editable" color="neutral" variant="outline" icon="i-lucide-copy" :loading="busy === 'dup'" :disabled="!!busy" @click="duplicate">
        {{ t('gmOps.mail.duplicate') }}
      </UButton>
    </footer>

    <FormDialog
      v-model="revokeOpen"
      :title="t('gmOps.mail.revokeTitle')"
      :width="480"
      :loading="busy === 'revoke'"
      :confirm-text="t('gmOps.mail.revoke')"
      @confirm="revoke"
    >
      <div class="me__dialog">
        <p class="me__dialog-text">{{ t('gmOps.mail.revokeDesc', { n: Math.max((mail?.recipientCount || 0) - (mail?.claimedCount || 0), 0) }) }}</p>
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import dayjs from 'dayjs'
import {
  cancelGameMail,
  createGameMail,
  deleteGameMail,
  dryRunGameMail,
  duplicateGameMail,
  revokeGameMail,
  sendGameMail,
  updateGameMail,
} from '@/api/gameOps'
import { listPlayers } from '@/api/gakumas'
import { RESOURCE_JEWEL_PAID } from '@/games/gakumas/catalog'
import type { GameReward, GameTarget } from '@/types/v1/game_common'
import type { GameMail, SaveGameMailRequest } from '@/types/v1/game_ops'
import { Dialog } from '@/utils/dialog'
import { feedback } from '@/utils/feedback'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import { useGmContext } from '../../context'
import GmRewardPicker from '../../components/GmRewardPicker.vue'
import GmRiskPill from '../../components/GmRiskPill.vue'
import PlayerPicker from './PlayerPicker.vue'
import { fmtDateTime, fmtShort, fromLocalInput, localZone, toDate, toLocalInput, useRewardLabels } from './shared'

type Target = 'players' | 'condition' | 'all'
type CondField = 'created' | 'lastLogin'
type CondOp = 'since' | 'before'
interface CondRow {
  field: CondField
  op: CondOp
  value: string
}

const props = defineProps<{ mail?: GameMail }>()
const emit = defineEmits<{
  /** 服务端返回了新的邮件状态（保存、试运行、发送、取消、撤回） */
  changed: [mail: GameMail]
  deleted: [id: string]
  /** 复制出了新草稿 */
  duplicated: [mail: GameMail]
}>()

const { t } = useI18n()
const ctx = useGmContext()
const { label, summary } = useRewardLabels()
const zone = localZone()

// 邮件奖励与礼物的风险、上限相同（领取时都进礼物箱）
const giftOperation = computed(() => ctx.operation('gift.send'))
const rewardMax = computed(() => giftOperation.value?.params?.properties?.rewards?.maxItems || 20)

const blank = () => ({
  title: '',
  body: '',
  rewards: [] as GameReward[],
  target: 'all' as Target,
  players: [] as GameTarget[],
  conds: [] as CondRow[],
  excludeBanned: true,
  expireDays: 14,
  when: 'now' as 'now' | 'later',
  scheduledAt: '',
})
const form = reactive(blank())

const condRowsOf = (m: GameMail): CondRow[] => {
  const c = m.condition
  if (!c) return []
  const rows: CondRow[] = []
  const push = (field: CondField, op: CondOp, ms: number) => {
    if (ms > 0) rows.push({ field, op, value: toLocalInput(new Date(ms)) })
  }
  push('created', 'since', Number(c.createdSince))
  push('created', 'before', Number(c.createdBefore))
  push('lastLogin', 'since', Number(c.lastLoginSince))
  push('lastLogin', 'before', Number(c.lastLoginBefore))
  return rows
}

const fill = (m?: GameMail) => {
  const next = blank()
  if (m) {
    next.title = m.title
    next.body = m.body
    next.rewards = (m.rewards || []).map((r) => ({ resourceType: r.resourceType, resourceId: r.resourceId, quantity: Number(r.quantity) }))
    next.target = (['players', 'condition', 'all'].includes(m.target) ? m.target : 'all') as Target
    next.players = (m.players || []).map((p) => ({ ...p }))
    next.conds = condRowsOf(m)
    next.excludeBanned = m.excludeBanned
    next.expireDays = m.expireDays
    const at = toDate(m.scheduledAt)
    next.when = m.status === 'scheduled' && at ? 'later' : 'now'
    next.scheduledAt = at ? toLocalInput(at) : ''
  }
  Object.assign(form, next)
  lastSaved = JSON.stringify(payloadBody())
}

const editable = computed(() => !props.mail || props.mail.status === 'draft' || props.mail.status === 'scheduled')
const canSend = computed(() => !props.mail || ['draft', 'scheduled', 'failed'].includes(props.mail.status))
const isDelivered = computed(() => ['sent', 'revoking', 'revoked'].includes(props.mail?.status || ''))
const claimPercent = computed(() => {
  const m = props.mail
  if (!m || !m.recipientCount) return 0
  return Math.min(100, Math.round((m.claimedCount / m.recipientCount) * 100))
})

// ---- 收件人 ----
const targetOptions = computed(() => [
  { value: 'players' as Target, label: t('gmOps.mail.target.players') },
  { value: 'condition' as Target, label: t('gmOps.mail.target.condition') },
  { value: 'all' as Target, label: t('gmOps.mail.target.all') },
])
const fieldOptions = computed(() => [
  { value: 'created' as CondField, label: t('gmOps.mail.cond.created') },
  { value: 'lastLogin' as CondField, label: t('gmOps.mail.cond.lastLogin') },
])
const opOptions = computed(() => [
  { value: 'since' as CondOp, label: t('gmOps.mail.cond.since') },
  { value: 'before' as CondOp, label: t('gmOps.mail.cond.before') },
])
const condLabel = (row: CondRow) => `${t(`gmOps.mail.cond.${row.field}`)} ${t(`gmOps.mail.cond.${row.op}`)}`
const addCond = () => {
  const all: Array<[CondField, CondOp]> = [
    ['created', 'before'],
    ['lastLogin', 'since'],
    ['created', 'since'],
    ['lastLogin', 'before'],
  ]
  const free = all.find(([f, o]) => !form.conds.some((c) => c.field === f && c.op === o))
  if (free) form.conds.push({ field: free[0], op: free[1], value: '' })
}

const conditionOf = () => {
  const c = { createdSince: 0, createdBefore: 0, lastLoginSince: 0, lastLoginBefore: 0 }
  if (form.target !== 'condition') return c
  for (const row of form.conds) {
    const ms = fromLocalInput(row.value)?.getTime() || 0
    if (!ms) continue
    if (row.field === 'created') c[row.op === 'since' ? 'createdSince' : 'createdBefore'] = ms
    else c[row.op === 'since' ? 'lastLoginSince' : 'lastLoginBefore'] = ms
  }
  return c
}

const registered = computed(() => (typeof ctx.status.value?.registered === 'number' ? ctx.status.value.registered : -1))
const bannedCount = ref(-1)
const loadBanned = async () => {
  try {
    const data = await listPlayers({ serverId: ctx.serverId.value, q: '', filter: 'banned', cursor: '', limit: 1 })
    bannedCount.value = data.summary ? data.summary.banned : -1
  } catch {
    bannedCount.value = -1
  }
}
watch(
  () => form.target,
  (target) => {
    if (target === 'all' && bannedCount.value < 0) void loadBanned()
  },
  { immediate: true },
)
const allHint = computed(() => {
  const parts: string[] = []
  if (registered.value >= 0) parts.push(t('gmOps.mail.allCount', { n: registered.value }))
  if (bannedCount.value > 0) parts.push(t('gmOps.mail.allBanned', { n: bannedCount.value }))
  parts.push(t('gmOps.mail.allLater'))
  return parts.join(' ')
})

// 收件人与已保存的一致时，邮件上记的试运行人数才有效。
const recipientKey = (target: string, players: GameTarget[], cond: ReturnType<typeof conditionOf>, excludeBanned: boolean) =>
  JSON.stringify([target, target === 'players' ? players.map((p) => p.pid) : [], target === 'condition' ? cond : null, excludeBanned])
const matchedKnown = computed(() => {
  const m = props.mail
  if (!m || !m.matchedAt) return false
  const saved = recipientKey(m.target, m.players || [], {
    createdSince: Number(m.condition?.createdSince || 0),
    createdBefore: Number(m.condition?.createdBefore || 0),
    lastLoginSince: Number(m.condition?.lastLoginSince || 0),
    lastLoginBefore: Number(m.condition?.lastLoginBefore || 0),
  }, m.excludeBanned)
  return saved === recipientKey(form.target, form.players, conditionOf(), form.excludeBanned)
})

// ---- 奖励、内容 ----
const rewardSummary = computed(() => summary(form.rewards))
const firstImage = computed(() => (form.rewards[0] ? label(form.rewards[0]).image : ''))
const expireOptions = computed(() => {
  const list = [7, 14, 30, 0]
  if (!list.includes(form.expireDays)) list.splice(3, 0, form.expireDays)
  return list
})

const ready = computed(
  () => !!form.title.trim() && form.rewards.length > 0 && (form.target !== 'players' || form.players.length > 0),
)

// ---- 发送 ----
const minSchedule = computed(() => toLocalInput(new Date()))
const scheduleValid = computed(() => {
  const d = fromLocalInput(form.scheduledAt)
  return !!d && d.getTime() > Date.now() + 30_000
})
const cta = computed(() => {
  if (form.when === 'later') return t('gmOps.mail.ctaLater')
  if (matchedKnown.value && props.mail) return t('gmOps.mail.ctaN', { n: props.mail.matchedCount })
  if (form.target === 'players' && form.players.length) return t('gmOps.mail.ctaN', { n: form.players.length })
  return t('gmOps.mail.ctaNow')
})

// ---- 页头 ----
const headTitle = computed(() => {
  const m = props.mail
  if (!m) return t('gmOps.mail.newTitle')
  if (m.status === 'draft') return t('gmOps.mail.editDraft', { title: m.title || t('gmOps.mail.untitled') })
  return m.title || t('gmOps.mail.untitled')
})
const today = computed(() => t('gmOps.common.today'))
const headMeta = computed(() => {
  const m = props.mail
  if (!m) return t('gmOps.mail.newMeta')
  const status = t(`gmOps.mail.status.${m.status}`)
  switch (m.status) {
    case 'draft':
      return `${status} · ${t('gmOps.mail.editedBy', { name: m.updatedByName || m.createdByName || '—', time: fmtShort(m.updateTime, today.value) })}`
    case 'scheduled':
      return `${status} · ${t('gmOps.mail.scheduledFor', { time: fmtDateTime(m.scheduledAt) })} · ${m.sentByName || '—'}`
    default:
      return `${status} · ${fmtDateTime(m.sentAt || m.updateTime)} · ${m.sentByName || '—'}${editable.value ? '' : ` · ${t('gmOps.mail.readonly')}`}`
  }
})

const errorText = (code: string) => (code === 'INTERRUPTED' ? t('gmOps.mail.interrupted') : code)

const menuItems = computed<ActionMenuItem[]>(() => {
  const m = props.mail
  if (!m) return []
  return [
    { key: 'duplicate', label: t('gmOps.mail.duplicate'), icon: 'i-lucide-copy' },
    { key: 'delete', label: t('gmOps.mail.deleteDraft'), icon: 'i-lucide-trash-2', danger: true, hidden: m.status !== 'draft' },
  ]
})

// ---- 保存 ----
const busy = ref<'' | 'save' | 'dry' | 'send' | 'dup' | 'revoke'>('')
const savedAt = ref('')
let lastSaved = ''
// 自己新建出来的邮件：父组件换成它时不要重置表单（期间可能还在输入）。
let adoptId = ''

const payloadBody = () => ({
  title: form.title.trim(),
  body: form.body.trim(),
  rewards: form.rewards,
  target: form.target,
  players: form.target === 'players' ? form.players : [],
  condition: conditionOf(),
  excludeBanned: form.excludeBanned,
  expireDays: form.expireDays,
  scheduledAt: form.when === 'later' ? fromLocalInput(form.scheduledAt) : undefined,
})

// 组件卸载后（换了一封邮件）收尾保存时不再通知父组件，免得把当前选中的邮件换回来。
let unmounted = false

const save = async (manual: boolean): Promise<GameMail | undefined> => {
  const body = payloadBody()
  const req: SaveGameMailRequest = { serverId: ctx.serverId.value, id: props.mail?.id || '', ...body }
  // 草稿里定时时间只是计划，过去的时间不发给服务端。
  if (req.scheduledAt && req.scheduledAt.getTime() <= Date.now()) req.scheduledAt = undefined
  if (manual) busy.value = 'save'
  try {
    const { data } = props.mail ? await updateGameMail(req) : await createGameMail(req)
    lastSaved = JSON.stringify(body)
    savedAt.value = dayjs().format('HH:mm')
    if (!props.mail) adoptId = data.id
    if (!unmounted) emit('changed', data)
    if (manual) feedback.success(t('gmOps.mail.saved'))
    return data
  } catch {
    return undefined
  } finally {
    if (manual) busy.value = ''
  }
}

const autosave = useDebounceFn(() => {
  const m = props.mail
  if (!m || m.status !== 'draft' || busy.value) return
  if (JSON.stringify(payloadBody()) === lastSaved) return
  void save(false)
}, 1500)
watch(form, () => void autosave(), { deep: true })

/** 试运行与发送前先把编辑内容存好。 */
const ensureSaved = async (): Promise<GameMail | undefined> => {
  if (!editable.value && props.mail) return props.mail
  if (props.mail && JSON.stringify(payloadBody()) === lastSaved) return props.mail
  return save(false)
}

const dryRun = async () => {
  busy.value = 'dry'
  try {
    const m = await ensureSaved()
    if (!m) return
    const { data } = await dryRunGameMail(ctx.serverId.value, m.id)
    if (data.mail) emit('changed', data.mail)
    feedback.info(
      data.registered
        ? t('gmOps.mail.dryRunResultOf', { n: data.matched, total: data.registered })
        : t('gmOps.mail.dryRunResult', { n: data.matched }),
    )
  } catch {
    // 错误已由请求层提示
  } finally {
    busy.value = ''
  }
}

const send = async () => {
  const later = form.when === 'later'
  const run = async () => {
    busy.value = 'send'
    try {
      const m = props.mail?.status === 'failed' ? props.mail : await ensureSaved()
      if (!m) return
      const { data } = await sendGameMail({
        serverId: ctx.serverId.value,
        id: m.id,
        scheduledAt: later ? fromLocalInput(form.scheduledAt) : undefined,
      })
      emit('changed', data)
      feedback.success(later ? t('gmOps.mail.scheduledOk') : t('gmOps.mail.sendStarted'))
    } catch {
      // 错误已由请求层提示
    } finally {
      busy.value = ''
    }
  }
  if (later) {
    await run()
    return
  }
  Dialog.confirm({
    title: t('gmOps.mail.confirmSendTitle'),
    content: t('gmOps.mail.confirmSendDesc', { title: form.title.trim(), who: recipientsText.value }),
    confirmText: cta.value,
    cancelText: t('gmOps.common.cancel'),
    onConfirm: run,
  })
}

const recipientsText = computed(() => {
  if (matchedKnown.value && props.mail) return t('gmOps.mail.toN', { n: props.mail.matchedCount })
  if (form.target === 'players') return t('gmOps.mail.toN', { n: form.players.length })
  return t(`gmOps.mail.target.${form.target}`)
})

const cancelSchedule = () => {
  const m = props.mail
  if (!m) return
  Dialog.confirm({
    title: t('gmOps.mail.cancelSchedule'),
    content: t('gmOps.mail.cancelScheduleDesc'),
    cancelText: t('gmOps.common.cancel'),
    onConfirm: async () => {
      try {
        const { data } = await cancelGameMail(ctx.serverId.value, m.id)
        emit('changed', data)
      } catch {
        // 错误已由请求层提示
      }
    },
  })
}

const revokeOpen = ref(false)
const revoke = async () => {
  const m = props.mail
  if (!m) return
  busy.value = 'revoke'
  try {
    const { data } = await revokeGameMail({ serverId: ctx.serverId.value, id: m.id })
    revokeOpen.value = false
    emit('changed', data)
    feedback.success(t('gmOps.mail.revokeStarted'))
  } catch {
    // 错误已由请求层提示
  } finally {
    busy.value = ''
  }
}

const duplicate = async () => {
  const m = props.mail
  if (!m) return
  busy.value = 'dup'
  try {
    const { data } = await duplicateGameMail(ctx.serverId.value, m.id)
    emit('duplicated', data)
    feedback.success(t('gmOps.mail.duplicated'))
  } catch {
    // 错误已由请求层提示
  } finally {
    busy.value = ''
  }
}

const remove = () => {
  const m = props.mail
  if (!m) return
  Dialog.confirm({
    title: t('gmOps.mail.deleteDraft'),
    content: t('gmOps.mail.deleteDesc', { title: m.title || t('gmOps.mail.untitled') }),
    cancelText: t('gmOps.common.cancel'),
    onConfirm: async () => {
      try {
        await deleteGameMail(ctx.serverId.value, m.id)
        emit('deleted', m.id)
      } catch {
        // 错误已由请求层提示
      }
    },
  })
}

const onMenu = (key: string) => {
  if (key === 'duplicate') void duplicate()
  else if (key === 'delete') remove()
}

// 换了一封邮件（或状态变了）时重置表单；自己刚新建的那封不重置。
watch(
  () => [props.mail?.id || '', props.mail?.status || ''] as const,
  ([id], old) => {
    const oldId = old?.[0]
    if (id && id === adoptId && oldId === '') {
      adoptId = ''
      return
    }
    savedAt.value = ''
    fill(props.mail)
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  unmounted = true
  // 离开时把未保存的草稿改动存下
  const m = props.mail
  if (m && m.status === 'draft' && JSON.stringify(payloadBody()) !== lastSaved) void save(false)
})
</script>

<style scoped>
.me {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.me__head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-bottom: 14px;
}
.me__head-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.me__title {
  font-family: var(--app-font-display);
  font-size: 20px;
  font-weight: 800;
  overflow-wrap: anywhere;
}
.me__meta,
.me__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.me__banner {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 10px 14px;
  border-radius: var(--app-radius);
  font-size: 13px;
  font-weight: 600;
  overflow-wrap: anywhere;
}
.me__banner--info {
  background: var(--m-a2Soft);
  color: var(--el-text-color-primary);
}
.me__banner--bad {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.me__spin {
  animation: me-spin 1s linear infinite;
}
@keyframes me-spin {
  to {
    transform: rotate(360deg);
  }
}
.me__claim {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  margin-bottom: 4px;
  border-radius: var(--app-radius-lg);
  background: var(--m-gSoft, var(--el-fill-color-light));
}
.me__claim-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.me__claim-label {
  font-weight: 800;
}
.me__claim-num {
  font-family: var(--app-font-display);
  font-size: 20px;
  font-weight: 800;
}
.me__claim-facts {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 14px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.me__warn {
  color: var(--m-warn);
  font-weight: 700;
}
.me__bar {
  height: 8px;
  border-radius: 999px;
  background: var(--el-bg-color);
  overflow: hidden;
}
.me__bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--m-gInk, var(--m-pInk));
}
.me__step {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 18px 0;
  border-top: 1px solid var(--el-border-color-light);
}
.me__step:first-of-type {
  border-top: 0;
  padding-top: 4px;
}
.me__step--split {
  flex-direction: row;
  flex-wrap: wrap;
  gap: 20px;
}
.me__step-head {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 15px;
  font-weight: 800;
  flex-wrap: wrap;
}
.me__sn {
  width: 24px;
  height: 24px;
  border-radius: 8px;
  background: var(--m-ink);
  color: var(--m-bg);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-family: var(--app-font-display);
  font-size: 12px;
  font-weight: 800;
}
.me__spacer {
  flex: 1;
}
.me__seg {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  align-self: flex-start;
  max-width: 100%;
}
.me__seg-btn {
  min-height: 36px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}
.me__seg-btn.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: 0 1px 3px color-mix(in srgb, var(--m-ink) 12%, transparent);
}
.me__seg-btn:disabled {
  cursor: default;
}
.me__seg-btn:disabled:not(.is-on) {
  opacity: 0.55;
}
.me__conds {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.me__cond {
  display: grid;
  grid-template-columns: 140px 110px minmax(0, 1fr) 40px;
  gap: 8px;
  align-items: center;
}
.me__cond-foot,
.me__recip-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.me__matched {
  font-family: var(--app-font-display);
  font-size: 20px;
  font-weight: 800;
}
.me__check {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}
.me__check input {
  width: 18px;
  height: 18px;
  accent-color: var(--m-g, var(--m-p));
}
.me__note {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  font-size: 13px;
  line-height: 1.6;
}
.me__note--warn {
  background: var(--m-warnSoft);
}
.me__note-icon {
  flex: none;
  margin-top: 3px;
  color: var(--m-warn);
}
.me__content {
  flex: 1 1 300px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.me__preview {
  flex: 0 1 300px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.me__label-sm {
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.04em;
  color: var(--el-text-color-secondary);
}
.me__preview-box {
  padding: 14px;
  border-radius: var(--app-radius-xl);
  background: var(--m-gSoft, var(--el-fill-color-light));
}
.me__gift {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 2px solid var(--m-gLine, var(--el-border-color-light));
}
.me__gift-img {
  flex: none;
  width: 48px;
  height: 48px;
  object-fit: contain;
}
.me__gift-img--empty {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 22px;
}
.me__gift-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.me__gift-title {
  font-size: 14px;
  font-weight: 800;
  overflow-wrap: anywhere;
}
.me__gift-sum {
  font-size: 12px;
  font-weight: 700;
  color: var(--m-gInk, var(--m-pInk));
  overflow-wrap: anywhere;
}
.me__gift-btn {
  flex: none;
  padding: 6px 12px;
  border-radius: 999px;
  background: var(--m-g, var(--m-p));
  color: var(--m-gOn, var(--m-pOn));
  font-size: 12px;
  font-weight: 800;
}
.me__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.me__label {
  font-size: 13px;
  font-weight: 700;
}
.me__req {
  color: var(--m-bad);
}
.me__input {
  width: 100%;
  min-height: 40px;
  padding: 0 12px;
  border-radius: var(--app-radius-sm);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 14px;
  outline: none;
  min-width: 0;
}
.me__input:focus {
  border-color: var(--m-p);
}
.me__input:disabled {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.me__textarea {
  padding: 10px 12px;
  resize: vertical;
  line-height: 1.6;
}
.me__send {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-start;
}
.me__send-when {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.me__foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-light);
}
.me__foot-hint {
  flex: 1 1 260px;
}
.me__dialog {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.me__dialog-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
}
@media (width < 768px) {
  .me__cond {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 44px;
  }
  .me__cond .me__input {
    grid-column: 1 / 3;
    grid-row: 2;
  }
  .me__preview {
    flex: 1 1 100%;
  }
  .me__seg-btn {
    min-height: 44px;
  }
  .me__foot :deep(button) {
    min-height: 44px;
  }
}
</style>
