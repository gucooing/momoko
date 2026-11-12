<!-- 游戏空间 · 自助服务：管理员开放的指令（规则与使用情况）、兑换码、我的申请、操作记录。 -->
<template>
  <div class="gsv">
    <div class="gsv__banner">
      <ShieldCheckIcon aria-hidden="true" />
      <span>{{ t('game.self.banner', { server: ctx.data.value?.server?.name || '', nickname: nickname }) }}</span>
    </div>

    <div class="gsv__main">
      <section class="gsv__fns">
        <div v-for="(f, i) in items" :key="f.command" class="pp-card gsv__fn">
          <div class="gsv__fn-head">
            <span class="gsv__fn-icon pp-sticker" :style="{ transform: `rotate(${tilt(i)}deg)` }">
              <SparklesIcon aria-hidden="true" />
            </span>
            <span class="gsv__fn-text">
              <span class="gsv__fn-title">{{ f.title }}</span>
              <span v-if="f.playerNote" class="pp-hint">{{ f.playerNote }}</span>
            </span>
          </div>
          <div class="gsv__rules">
            <span v-for="r in rules(f)" :key="r" class="pp-tag">{{ r }}</span>
          </div>
          <div class="gsv__fn-foot">
            <span class="gsv__state" :class="`is-${stateOf(f).tone}`">{{ stateOf(f).text }}</span>
            <PButton size="sm" :variant="usable(f) ? 'primary' : 'outline'" :disabled="!usable(f)" @click="active = f">
              {{ f.needApproval ? t('game.self.apply') : t('game.self.run') }}
            </PButton>
          </div>
        </div>
        <PEmpty
          v-if="loaded && !items.length"
          class="pp-card gsv__empty"
          :icon="SparklesIcon"
          :title="error || t('game.self.empty')"
          :description="error ? '' : t('game.self.emptyDesc')"
        />
      </section>

      <aside class="gsv__aside">
        <div class="pp-card gsv__box">
          <span class="gsv__box-head">
            <TicketIcon aria-hidden="true" />
            {{ t('game.self.redeem') }}
          </span>
          <form class="gsv__redeem" @submit.prevent="redeem">
            <input
              v-model="code"
              class="pp-mono"
              type="text"
              :placeholder="t('game.self.redeemPlaceholder')"
              :aria-label="t('game.self.redeem')"
              autocomplete="off"
              maxlength="64"
            />
            <PButton type="submit" variant="ink" size="sm" :loading="redeeming" :disabled="!code.trim()">{{ t('game.self.redeemBtn') }}</PButton>
          </form>
          <div v-if="redeemResult" class="gsv__rr" :class="redeemResult.result === 'success' ? 'is-ok' : 'is-bad'">
            <span>{{ redeemText(redeemResult.result) }}</span>
            <span v-if="redeemResult.rewards.length" class="gsv__rewards">
              <span v-for="(r, i) in redeemResult.rewards" :key="i" class="gsv__reward">
                <span class="gsv__reward-img"><GameImage :src="ctx.resourceImage(r.resourceType, r.resourceId)" /></span>
                {{ ctx.resourceName(r.resourceType, r.resourceId) || t('game.unnamed') }} ×{{ fmtNum(r.quantity) }}
              </span>
            </span>
          </div>
          <span v-if="redeemError" class="gsv__rr is-bad">{{ redeemError }}</span>
          <span class="pp-hint">{{ t('game.self.redeemHint') }}</span>
        </div>

        <div class="pp-card gsv__box">
          <span class="gsv__box-head">{{ t('game.self.requests') }}</span>
          <div v-if="requests.items.length" class="pp-rows">
            <div v-for="r in requests.items" :key="r.id" class="gsv__row">
              <span class="gsv__row-text">
                <span class="gsv__row-t">{{ r.commandTitle || r.command }}</span>
                <span class="pp-hint">{{ requestLine(r) }}</span>
                <span v-if="r.reviewNote" class="pp-hint">{{ t('game.self.reviewNote', { note: r.reviewNote }) }}</span>
              </span>
              <PPill :tone="statusTone(r.status)">{{ requestStatus(r) }}</PPill>
            </div>
          </div>
          <span v-else-if="requests.loaded" class="pp-hint">{{ requests.error || t('game.self.requestsEmpty') }}</span>
          <PButton v-if="requests.items.length < requests.total" size="sm" variant="ghost" :loading="requests.loading" @click="loadRequests(true)">
            {{ t('game.loadMore') }}
          </PButton>
        </div>

        <div class="pp-card gsv__box">
          <span class="gsv__box-head">{{ t('game.self.records') }}</span>
          <div v-if="records.items.length" class="pp-rows">
            <div v-for="r in records.items" :key="`${r.kind}-${r.id}`" class="gsv__log">
              <span class="pp-hint pp-num gsv__log-at">{{ fmtTime(r.createTime) }}</span>
              <span class="gsv__log-t">{{ recordText(r) }}</span>
            </div>
          </div>
          <span v-else-if="records.loaded" class="pp-hint">{{ records.error || t('game.self.recordsEmpty') }}</span>
          <PButton v-if="records.items.length < records.total" size="sm" variant="ghost" :loading="records.loading" @click="loadRecords(true)">
            {{ t('game.loadMore') }}
          </PButton>
        </div>
      </aside>
    </div>

    <SelfServiceDialog :item="active" @close="active = null" @done="onDone" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ShieldCheckIcon, SparklesIcon, TicketIcon } from '@heroicons/vue/24/outline'
import GameImage from '@/components/game/GameImage.vue'
import PButton from '@passport/components/PButton.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import PPill from '@passport/components/PPill.vue'
import { listGameRecords, listGameRequests, listSelfService, redeemGiftCode } from '@passport/api/passport'
import { useUiStore } from '@passport/stores/ui'
import { errorText, fmtDuration, fmtNum, fmtTime, toDate } from '@passport/utils/format'
import type { PassportGameRecord, PassportSelfService, RedeemPassportGiftCodeResponse } from '@/types/v1/passport'
import SelfServiceDialog from './SelfServiceDialog.vue'
import { useGameContext, valueText } from './context'

const PAGE = 10

const { t, te } = useI18n()
const ctx = useGameContext()
const ui = useUiStore()

const nickname = computed(() => ctx.data.value?.link?.nickname || ctx.player.value?.name || '')

// ---------- 自助指令 ----------
const items = ref<PassportSelfService[]>([])
const loaded = ref(false)
const error = ref('')
const active = ref<PassportSelfService | null>(null)

async function loadItems() {
  try {
    items.value = (await listSelfService(ctx.serverId.value)).items ?? []
    error.value = ''
  } catch (e) {
    error.value = errorText(e)
  } finally {
    loaded.value = true
  }
}

const tilt = (i: number) => [-3, -2, 4, -4, 2, 3][i % 6]

const cooling = (f: PassportSelfService) => {
  const next = toDate(f.nextAvailable)
  return next && next.getTime() > Date.now() ? next : null
}

const usable = (f: PassportSelfService) => f.remaining !== 0 && !f.hasPending && !cooling(f)

const rules = (f: PassportSelfService) => {
  const out: string[] = []
  if (f.dailyLimit > 0) out.push(t('game.self.perDay', { n: f.dailyLimit }))
  if (f.cooldownSec > 0) out.push(t('game.self.cooldown', { d: fmtDuration(f.cooldownSec) }))
  if (f.needApproval) out.push(t('game.self.needApproval'))
  if (f.needTotp) out.push(t('game.self.needTotp'))
  if (!out.length) out.push(t('game.self.unlimited'))
  return out
}

const statusText = (status: string) => (te(`game.self.status.${status}`) ? t(`game.self.status.${status}`) : status)

const stateOf = (f: PassportSelfService): { text: string; tone: 'ok' | 'warn' | 'off' } => {
  if (f.hasPending) return { text: t('game.self.pending'), tone: 'warn' }
  const next = cooling(f)
  if (next) return { text: t('game.self.coolingUntil', { time: fmtTime(next) }), tone: 'off' }
  if (f.remaining === 0) return { text: t('game.self.usedUp'), tone: 'off' }
  if (f.remaining > 0) return { text: t('game.self.leftToday', { n: f.remaining }), tone: 'ok' }
  if (f.lastUsedAt) return { text: t('game.self.last', { time: fmtTime(f.lastUsedAt), status: statusText(f.lastStatus) }), tone: 'off' }
  return { text: t('game.self.never'), tone: 'ok' }
}

const onDone = (status: string) => {
  ui.toast(status === 'pending' ? t('game.self.resultPending') : statusText(status), status === 'failed' ? 'bad' : 'ok')
  void loadItems()
  void loadRequests()
  void loadRecords()
  void ctx.reload()
}

// ---------- 兑换码 ----------
const code = ref('')
const redeeming = ref(false)
const redeemResult = ref<RedeemPassportGiftCodeResponse | null>(null)
const redeemError = ref('')

const redeemText = (result: string) =>
  te(`game.self.redeemResult.${result}`) ? t(`game.self.redeemResult.${result}`) : result

async function redeem() {
  const c = code.value.trim()
  if (!c) return
  redeeming.value = true
  redeemError.value = ''
  redeemResult.value = null
  try {
    redeemResult.value = await redeemGiftCode(ctx.serverId.value, c)
    if (redeemResult.value.result === 'success') {
      code.value = ''
      void loadRecords()
    }
  } catch (e) {
    redeemError.value = errorText(e)
  } finally {
    redeeming.value = false
  }
}

// ---------- 我的申请、操作记录 ----------
interface RecordList {
  items: PassportGameRecord[]
  total: number
  page: number
  loading: boolean
  loaded: boolean
  error: string
}

const requests = reactive<RecordList>({ items: [], total: 0, page: 0, loading: false, loaded: false, error: '' })
const records = reactive<RecordList>({ items: [], total: 0, page: 0, loading: false, loaded: false, error: '' })

async function loadList(list: RecordList, fetch: (page: number) => Promise<{ items: PassportGameRecord[]; total: number }>, more: boolean) {
  if (list.loading) return
  list.loading = true
  const page = more ? list.page + 1 : 1
  try {
    const res = await fetch(page)
    list.items = more ? [...list.items, ...(res.items ?? [])] : (res.items ?? [])
    list.total = res.total ?? 0
    list.page = page
    list.error = ''
  } catch (e) {
    list.error = errorText(e)
  } finally {
    list.loading = false
    list.loaded = true
  }
}

const loadRequests = (more = false) => loadList(requests, (p) => listGameRequests(ctx.serverId.value, p, PAGE), more)
const loadRecords = (more = false) => loadList(records, (p) => listGameRecords(ctx.serverId.value, p, PAGE), more)

const statusTone = (status: string): 'ok' | 'warn' | 'bad' | 'off' => {
  switch (status) {
    case 'pending':
    case 'running':
      return 'warn'
    case 'success':
    case 'partial':
      return 'ok'
    case 'rejected':
    case 'failed':
      return 'bad'
    default:
      return 'off'
  }
}

const requestStatus = (r: PassportGameRecord) =>
  (r.status === 'success' || r.status === 'partial') && r.reviewedAt ? t('game.self.status.approved') : statusText(r.status)

const requestLine = (r: PassportGameRecord) => {
  const parts = [t('game.self.submittedAt', { time: fmtTime(r.createTime) })]
  if (r.reviewedAt) parts.push(t('game.self.reviewedAt', { time: fmtTime(r.reviewedAt) }))
  return parts.join(' · ')
}

const recordText = (r: PassportGameRecord) => {
  if (r.kind === 'redeem') return `${t('game.self.redeemRecord', { code: r.code })} · ${redeemText(r.status)}`
  const changes = (r.changes ?? []).map((c) => `${valueText(c.before)} → ${valueText(c.after)}`).join(' · ')
  const title = r.commandTitle || r.command
  return changes ? `${title} · ${changes}` : `${title} · ${statusText(r.status)}`
}

onMounted(() => {
  void loadItems()
  void loadRequests()
  void loadRecords()
})
</script>

<style scoped>
.gsv {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.gsv__banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px;
  border-radius: 20px;
  background: var(--m-gSoft);
  font-size: 14px;
  line-height: 1.6;
}
.gsv__banner svg {
  flex: none;
  width: 24px;
  height: 24px;
  color: var(--m-gInk);
}
.gsv__main {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  align-items: flex-start;
}
.gsv__fns {
  flex: 3 1 600px;
  min-width: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}
.gsv__empty {
  grid-column: 1 / -1;
}
.gsv__fn {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 18px;
  min-width: 0;
}
.gsv__fn-head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.gsv__fn-icon {
  width: 46px;
  height: 46px;
  border-radius: 15px;
  background: var(--m-gSoft);
  color: var(--m-gInk);
}
.gsv__fn-icon svg {
  width: 24px;
  height: 24px;
}
.gsv__fn-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gsv__fn-title {
  font-size: 16px;
  font-weight: 800;
}
.gsv__rules {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.gsv__fn-foot {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--m-line);
  margin-top: auto;
}
.gsv__state {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 700;
  color: var(--m-ink2);
}
.gsv__state.is-ok {
  color: var(--m-gInk);
}
.gsv__state.is-warn {
  color: var(--m-warn);
}
.gsv__aside {
  flex: 2 1 340px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gsv__box {
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.gsv__box-head {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 800;
}
.gsv__box-head svg {
  width: 24px;
  height: 24px;
  color: var(--m-gInk);
}
.gsv__redeem {
  display: flex;
  gap: 8px;
}
.gsv__redeem input {
  flex: 1;
  min-width: 0;
  min-height: 44px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink);
  font-size: 14px;
  outline: none;
}
.gsv__redeem input:focus {
  border-color: var(--m-g);
}
.gsv__redeem :deep(.pb) {
  min-height: 44px;
}
.gsv__rr {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  font-weight: 700;
}
.gsv__rr.is-ok {
  color: var(--m-ok);
}
.gsv__rr.is-bad {
  color: var(--m-bad);
}
.gsv__rewards {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  color: var(--m-ink);
}
.gsv__reward {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 700;
}
.gsv__reward-img,
.gsv__reward-img :deep(img) {
  width: 24px;
  height: 24px;
  object-fit: contain;
  background: transparent;
}
.gsv__row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 6px 0;
}
.gsv__row-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.gsv__row-t {
  font-size: 14px;
  font-weight: 700;
}
.gsv__log {
  display: flex;
  gap: 10px;
  padding: 8px 0;
  font-size: 13px;
}
.gsv__log-at {
  flex: none;
  width: 92px;
}
.gsv__log-t {
  flex: 1;
  min-width: 0;
  word-break: break-word;
}
@media (max-width: 768px) {
  .gsv__fns {
    grid-template-columns: minmax(0, 1fr);
  }
  .gsv__fn-foot :deep(.pb) {
    min-height: 44px;
  }
}
</style>
