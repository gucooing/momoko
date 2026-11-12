<!-- 发送到礼物箱（操作 gift.send）：发给谁（最多 200 人；可搜索添加，或由外部固定）、礼物内容（GmGiftContent：奖励与文案，
     与兑换码批次共用）、领取期限；「预览」是试运行（礼物箱前后件数、已持有的卡领取时转成什么），「发送」真实执行并写进 GM 记录。
     参数上限（奖励项数、文案长度、期限范围）取自操作的参数 Schema。发送礼物对话框与「GM 操作」页的执行面板共用。 -->
<template>
  <div v-if="!operation" class="ggs">
    <EmptyState icon="HOutline:GiftIcon" :title="t('gmData.gift.unavailable')" />
  </div>
  <div v-else class="ggs">
    <div class="ggs__field">
      <span class="ggs__label">{{ t('gmData.gift.to') }} <span class="ggs__hint m-num">{{ targets.length }} / {{ MAX_TARGETS }}</span></span>
      <div v-if="targets.length" class="ggs__chips">
        <span v-for="tg in targets" :key="tg.pid" class="ggs__chip">
          <PlayerAvatar :idol-card-id="tg.avatarIdolCardId" :name="tg.name || t('gm.player.noName')" :size="26" />
          <span class="ggs__chip-name">{{ tg.name || t('gm.player.noName') }}</span>
          <button
            v-if="!fixedTargets"
            type="button"
            class="ggs__chip-x"
            :aria-label="t('gmData.gift.removeTarget', { name: tg.name || t('gm.player.noName') })"
            :disabled="running"
            @click="removeTarget(tg.pid)"
          >
            ×
          </button>
        </span>
      </div>
      <div v-if="!fixedTargets && targets.length < MAX_TARGETS" class="ggs__add">
        <label class="ggs__search">
          <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" class="ggs__search-icon" />
          <input v-model="playerQ" type="search" :placeholder="t('gmData.gift.addPlayer')" :aria-label="t('gmData.gift.addPlayer')" :disabled="running" />
        </label>
        <div v-if="playerQ.trim()" class="ggs__found" role="listbox">
          <button v-for="p in shownFound" :key="p.pid" type="button" role="option" class="ggs__found-row" :aria-selected="false" @click="addTarget(p)">
            <PlayerAvatar :idol-card-id="p.avatarIdolCardId" :name="p.name" :size="32" :online="p.online" />
            <span class="ggs__found-text">
              <span class="ggs__found-name">{{ p.name || t('gm.player.noName') }}</span>
              <span class="ggs__hint m-num">ID {{ p.displayId }} · {{ p.account }}</span>
            </span>
          </button>
          <div v-if="searching" class="ggs__hint ggs__found-note">{{ t('gmData.common.loading') }}</div>
          <div v-else-if="searchError" class="ggs__hint ggs__found-note">{{ searchError }}</div>
          <div v-else-if="!shownFound.length" class="ggs__hint ggs__found-note">{{ t('gmData.gift.noPlayer') }}</div>
        </div>
      </div>
    </div>

    <GmGiftContent v-model:rewards="rewards" v-model:message="message" :max-rewards="maxRewards" :max-message="maxMessage" :disabled="running" />

    <div class="ggs__field">
      <span class="ggs__label">{{ t('gmData.gift.expire') }}</span>
      <div class="ggs__seg" role="radiogroup" :aria-label="t('gmData.gift.expire')">
        <button
          v-for="d in expireOptions"
          :key="d"
          type="button"
          role="radio"
          :aria-checked="expireDays === d"
          :class="{ 'is-on': expireDays === d }"
          :disabled="running"
          @click="expireDays = d"
        >
          {{ d ? t('gmData.gift.days', { n: d }) : t('gmData.gift.noExpire') }}
        </button>
      </div>
    </div>

    <div v-if="preview" class="ggs__preview">
      <span class="ggs__preview-title">{{ t('gmData.gift.previewTitle') }}</span>
      <div v-for="r in previewRows" :key="r.pid" class="ggs__preview-row">
        <span class="ggs__preview-who">{{ r.name || t('gm.player.noName') }}</span>
        <template v-if="r.status === 'success' && r.detail">
          <span class="m-num">{{ t('gmData.gift.boxChange', { before: r.detail.giftBoxBefore, after: r.detail.giftBoxAfter }) }}</span>
          <span v-for="(c, i) in convertsOf(r.detail)" :key="i" class="ggs__preview-note">
            {{ t('gmData.gift.converts', { name: c.name, to: c.to, n: c.n }) }}
          </span>
        </template>
        <span v-else class="ggs__error">{{ r.message || r.code }}</span>
      </div>
      <span v-if="previewMore" class="ggs__hint">{{ t('gmData.gift.previewMore', { n: previewMore }) }}</span>
    </div>

    <GmOperationResult v-if="result" :outcome="result" />
    <div v-if="error" class="ggs__error">{{ error }}</div>

    <div class="ggs__actions">
      <span class="ggs__hint ggs__actions-hint">{{ t('gmData.gift.footHint') }}</span>
      <UButton v-if="closable" color="neutral" variant="ghost" :disabled="running" @click="emit('close')">{{ t('gm.command.close') }}</UButton>
      <UButton color="neutral" variant="outline" :loading="running && dryRunning" :disabled="!canRun" @click="run(true)">
        {{ t('gmData.gift.preview') }}
      </UButton>
      <UButton color="primary" icon="i-lucide-send" :loading="running && !dryRunning" :disabled="!canRun" @click="run(false)">
        {{ t('gmData.gift.sendTo', { n: targets.length }) }}
      </UButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import { listPlayers, sendGift } from '@/api/gakumas'
import type { OperationOutcome } from '@/games/mgp'
import type { GiftSendResult, Player } from '@/games/gakumas/types'
import { getRequestErrorMessage } from '@/utils/request'
import { useGmContext } from '../context'
import { outcome } from '../operations'
import PlayerAvatar from '../player/shared/PlayerAvatar.vue'
import { useCatalogLookup } from '../player/shared/lookup'
import GmGiftContent from './GmGiftContent.vue'
import GmOperationResult from './GmOperationResult.vue'
import type { GmReward } from './types'

export interface GiftTarget {
  pid: string
  name?: string
  avatarIdolCardId?: string
}

/** 预览里一名玩家：发送后礼物箱的变化与重复转换，失败时的原因。 */
interface PreviewRow {
  pid: string
  name: string
  status: string
  code: string
  message: string
  detail?: GiftSendResult
}

// 一次 1–200 名玩家
const MAX_TARGETS = 200
const OPERATION = 'gift.send'
const PREVIEW_ROWS = 5

const props = withDefaults(
  defineProps<{
    /** 收件人：可添加时为初始收件人，fixedTargets 时就是收件人 */
    players?: GiftTarget[]
    /** 收件人由外部决定（「GM 操作」页左侧勾选的玩家），这里不能增删 */
    fixedTargets?: boolean
    /** 变化时清空内容与结果（对话框每次打开加一） */
    resetKey?: number
    /** 显示「关闭」按钮（对话框里） */
    closable?: boolean
  }>(),
  { players: () => [], fixedTargets: false, resetKey: 0, closable: false },
)
const emit = defineEmits<{ done: []; busy: [running: boolean]; close: [] }>()

const { t } = useI18n()
const ctx = useGmContext()
const menuStore = useMenuStore()
const lookup = useCatalogLookup()

const operation = computed(() => ctx.operation(OPERATION))
const schema = computed(() => operation.value?.params?.properties || {})
const maxRewards = computed(() => schema.value.rewards?.maxItems || 20)
const maxMessage = computed(() => schema.value.message?.maxLength || 64)
const expireOptions = computed(() => {
  const min = schema.value.expireDays?.minimum ?? 0
  const max = schema.value.expireDays?.maximum ?? 365
  return [7, 30, 90, 0].filter((d) => d >= min && d <= max)
})

const ownTargets = ref<GiftTarget[]>([])
const targets = computed(() => (props.fixedTargets ? props.players.slice(0, MAX_TARGETS) : ownTargets.value))
const rewards = ref<GmReward[]>([])
const message = ref('')
const expireDays = ref(30)
const preview = ref<PreviewRow[]>()
const result = ref<OperationOutcome>()
const error = ref('')
const running = ref(false)
const dryRunning = ref(false)
const playerQ = ref('')
const found = ref<Player[]>([])
const searching = ref(false)
const searchError = ref('')

watch(
  () => props.resetKey,
  () => {
    ownTargets.value = props.players.slice(0, MAX_TARGETS).map((p) => ({ ...p }))
    rewards.value = []
    message.value = ''
    expireDays.value = expireOptions.value.includes(30) ? 30 : (expireOptions.value[0] ?? 0)
    preview.value = undefined
    result.value = undefined
    error.value = ''
    playerQ.value = ''
  },
  { immediate: true },
)
watch(running, (v) => emit('busy', v))

// 改了内容，之前的预览就不再对应
watch(
  [targets, rewards, message, expireDays],
  () => {
    preview.value = undefined
  },
  { deep: true },
)

const hasTarget = (pid: string) => targets.value.some((x) => x.pid === pid)
const shownFound = computed(() => found.value.filter((p) => !hasTarget(p.pid)))
const addTarget = (p: Player) => {
  if (hasTarget(p.pid) || ownTargets.value.length >= MAX_TARGETS) return
  ownTargets.value = [...ownTargets.value, { pid: p.pid, name: p.name, avatarIdolCardId: p.avatarIdolCardId }]
}
const removeTarget = (pid: string) => {
  ownTargets.value = ownTargets.value.filter((x) => x.pid !== pid)
}

let searchSeq = 0
const search = async () => {
  const q = playerQ.value.trim()
  if (!q) {
    found.value = []
    return
  }
  const id = ++searchSeq
  searching.value = true
  searchError.value = ''
  try {
    const data = await listPlayers({ serverId: ctx.serverId.value, q, filter: 'all', cursor: '', limit: 8 })
    if (id !== searchSeq) return
    found.value = data.items
  } catch (e) {
    if (id === searchSeq) searchError.value = getRequestErrorMessage(e)
  } finally {
    if (id === searchSeq) searching.value = false
  }
}
const debouncedSearch = useDebounceFn(search, 300)
watch(playerQ, () => void debouncedSearch())

const canRun = computed(() => !!operation.value && !running.value && targets.value.length > 0 && rewards.value.length > 0)

const run = async (dryRun: boolean) => {
  if (!canRun.value) return
  running.value = true
  dryRunning.value = dryRun
  error.value = ''
  if (!dryRun) result.value = undefined
  try {
    const r = await sendGift({
      serverId: ctx.serverId.value,
      pids: targets.value.map((x) => x.pid),
      rewards: rewards.value,
      message: message.value.trim(),
      expireDays: expireDays.value,
      dryRun,
      summary: {
        message: message.value.trim(),
        rewards: String(rewards.value.length),
        expireDays: String(expireDays.value),
      },
    })
    const execution = r.execution
    const status = execution?.status || ''
    if (status === 'failed' && execution?.errorCode) error.value = `${execution.errorCode} ${execution.errorMessage}`
    if (dryRun) {
      preview.value = r.results.map((x) => ({
        pid: x.target?.pid || '',
        name: x.target?.name || '',
        status: x.target?.status || '',
        code: x.target?.code || '',
        message: x.target?.message || '',
        detail: x.data,
      }))
    } else {
      result.value = outcome(execution, r.results, false, () => undefined)
      if (status === 'success' || status === 'partial') {
        feedback.success(t('gmData.gift.sent', { n: targets.value.length }))
        emit('done')
      }
    }
  } catch (e) {
    error.value = getRequestErrorMessage(e)
  } finally {
    running.value = false
    dryRunning.value = false
  }
}

const previewRows = computed(() => (preview.value || []).slice(0, PREVIEW_ROWS))
const previewMore = computed(() => Math.max(0, (preview.value?.length || 0) - PREVIEW_ROWS))

const rewardName = (resourceType: number, resourceId: string) => {
  const rr = lookup.rewardRef(resourceType, resourceId)
  return rr ? lookup.name(rr.kind, rr.id) : t('game.unnamed')
}
const convertsOf = (detail: GiftSendResult) =>
  detail.preview
    .filter((r) => r.owned && r.convertsTo)
    .map((r) => ({
      name: rewardName(r.resourceType, r.resourceId),
      to: lookup.name('item', r.convertsTo),
      n: r.convertedQuantity,
    }))
</script>

<style scoped>
.ggs {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.ggs__field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.ggs__label {
  font-size: 13px;
  font-weight: 700;
}
.ggs__hint {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.ggs__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  max-height: 140px;
  overflow-y: auto;
}
.ggs__chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 34px;
  padding: 0 4px 0 4px;
  border-radius: 999px;
  background: var(--m-s2);
  font-size: 13px;
  font-weight: 700;
  max-width: 100%;
}
.ggs__chip-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ggs__chip-x {
  flex: none;
  width: 26px;
  height: 26px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 16px;
  cursor: pointer;
}
.ggs__add {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ggs__search {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.ggs__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
}
.ggs__search-icon {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.ggs__found {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 240px;
  overflow-y: auto;
  padding: 4px;
  border-radius: var(--app-radius-sm);
  border: 1px solid var(--el-border-color-light);
}
.ggs__found-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 48px;
  padding: 4px 8px;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.ggs__found-row:hover {
  background: var(--m-pSoft);
}
.ggs__found-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.ggs__found-name {
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ggs__found-note {
  padding: 8px;
}
.ggs__seg {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  align-self: flex-start;
}
.ggs__seg button {
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
.ggs__seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: 0 1px 4px color-mix(in srgb, var(--m-ink) 14%, transparent);
}
.ggs__preview {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px;
  border-radius: var(--app-radius-lg);
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
  font-size: 13px;
}
.ggs__preview-title {
  font-size: 14px;
  font-weight: 800;
}
.ggs__preview-row {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  line-height: 1.6;
}
.ggs__preview-who {
  font-weight: 800;
}
.ggs__preview-note {
  flex-basis: 100%;
}
.ggs__error {
  font-size: 13px;
  color: var(--m-bad);
}
.ggs__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}
.ggs__actions-hint {
  flex: 1 1 200px;
}
</style>
