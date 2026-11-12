<!-- 发送到礼物箱（指令 gift.send）抽屉：发给谁（可搜索添加，最多 200 人）、奖励（来自 MGP 目录）、
     礼物箱文案、领取期限、操作原因；「预览」是试运行（礼物箱前后件数、已持有的卡领取时转成什么），
     「发送」真实执行并写进 GM 记录。参数上限（奖励项数、文案长度、期限范围）取自 manifest 的参数 Schema。 -->
<template>
  <Teleport to="body">
    <Transition name="gdr">
      <div v-if="modelValue" class="gdr" :data-game="ctx.game.value || undefined">
        <div class="gdr__scrim" @click="close" />
        <section class="gdr__panel game-skin" role="dialog" aria-modal="true" :aria-label="command?.title || t('gmData.gift.title')">
          <header class="gdr__head">
            <span class="gdr__sticker" aria-hidden="true">
              <component :is="menuStore.iconComponents['HOutline:GiftIcon']" />
            </span>
            <span class="gdr__heading">
              <span class="gdr__title">{{ command?.title || t('gmData.gift.title') }}</span>
              <span class="gdr__meta">
                <GmRiskPill v-if="command" :risk="command.risk" />
              </span>
            </span>
            <AppIconButton icon="HOutline:XMarkIcon" :label="t('gmData.common.close')" :box="40" @click="close" />
          </header>

          <div v-if="!command" class="gdr__body">
            <EmptyState icon="HOutline:GiftIcon" :title="t('gmData.gift.unavailable')" />
          </div>

          <div v-else class="gdr__body">
            <div class="gdr__field">
              <span class="gdr__label">{{ t('gmData.gift.to') }} <span class="gdr__hint m-num">{{ targets.length }} / {{ MAX_TARGETS }}</span></span>
              <div v-if="targets.length" class="gdr__chips">
                <span v-for="tg in targets" :key="tg.pid" class="gdr__chip">
                  <PlayerAvatar :idol-card-id="tg.avatarIdolCardId" :name="tg.name || t('gm.player.noName')" :size="30" />
                  <span class="gdr__chip-name">{{ tg.name || t('gm.player.noName') }}</span>
                  <button
                    type="button"
                    class="gdr__chip-x"
                    :aria-label="t('gmData.gift.removeTarget', { name: tg.name || t('gm.player.noName') })"
                    :disabled="running"
                    @click="removeTarget(tg.pid)"
                  >
                    <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
                  </button>
                </span>
              </div>
              <div v-if="targets.length < MAX_TARGETS" class="gdr__add">
                <label class="gdr__search">
                  <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" class="gdr__search-icon" />
                  <input
                    v-model="playerQ"
                    type="search"
                    :placeholder="t('gmData.gift.addPlayer')"
                    :aria-label="t('gmData.gift.addPlayer')"
                    :disabled="running"
                  />
                </label>
                <div v-if="playerQ.trim()" class="gdr__found" role="listbox">
                  <button
                    v-for="p in found"
                    :key="p.pid"
                    type="button"
                    role="option"
                    class="gdr__found-row"
                    :aria-selected="hasTarget(p.pid)"
                    :disabled="hasTarget(p.pid)"
                    @click="addTarget(p)"
                  >
                    <PlayerAvatar :idol-card-id="p.avatarIdolCardId" :name="p.name" :size="32" :online="p.online" />
                    <span class="gdr__found-text">
                      <span class="gdr__found-name">{{ p.name || t('gm.player.noName') }}</span>
                      <span class="gdr__hint m-num">ID {{ p.displayId }} · {{ p.account }}</span>
                    </span>
                    <component v-if="hasTarget(p.pid)" :is="menuStore.iconComponents['HOutline:CheckIcon']" class="gdr__found-check" />
                  </button>
                  <div v-if="searching" class="gdr__hint gdr__found-note">{{ t('gmData.common.loading') }}</div>
                  <div v-else-if="searchError" class="gdr__hint gdr__found-note">{{ searchError }}</div>
                  <div v-else-if="!found.length" class="gdr__hint gdr__found-note">{{ t('gmData.gift.noPlayer') }}</div>
                </div>
              </div>
            </div>

            <div class="gdr__field">
              <span class="gdr__label">{{ t('gmData.gift.rewards') }}</span>
              <GmRewardPicker v-model="rewards" :max="maxRewards" :disabled="running" />
            </div>

            <label class="gdr__field">
              <span class="gdr__label">
                {{ t('gmData.gift.message') }}
                <span class="gdr__hint">{{ t('gmData.gift.messageNote', { max: maxMessage }) }}</span>
              </span>
              <input
                v-model="message"
                class="gdr__input"
                :maxlength="maxMessage"
                :placeholder="t('gmData.gift.messageHint')"
                :disabled="running"
              />
            </label>

            <div class="gdr__field">
              <span class="gdr__label">{{ t('gmData.gift.expire') }}</span>
              <div class="gdr__seg" role="radiogroup" :aria-label="t('gmData.gift.expire')">
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

            <div v-if="preview" class="gdr__preview">
              <span class="gdr__preview-title">
                <component :is="menuStore.iconComponents['HOutline:EyeIcon']" />
                {{ t('gmData.gift.previewTitle') }}
              </span>
              <div v-for="r in previewRows" :key="r.pid" class="gdr__preview-row">
                <span class="gdr__preview-who">{{ r.name || t('gm.player.noName') }}</span>
                <template v-if="r.status === 'success' && r.detail">
                  <span class="m-num">{{ t('gmData.gift.boxChange', { before: r.detail.giftBoxBefore, after: r.detail.giftBoxAfter }) }}</span>
                  <span v-for="(c, i) in convertsOf(r.detail)" :key="i" class="gdr__preview-note">
                    {{ t('gmData.gift.converts', { name: c.name, to: c.to, n: c.n }) }}
                  </span>
                </template>
                <span v-else class="gdr__error">{{ r.message || r.code }}</span>
              </div>
              <span v-if="previewMore" class="gdr__hint">{{ t('gmData.gift.previewMore', { n: previewMore }) }}</span>
            </div>

            <GmCommandResult v-if="result" :result="result" />
            <div v-if="error" class="gdr__error">{{ error }}</div>
          </div>

          <footer class="gdr__foot">
            <span class="gdr__hint gdr__foot-hint">{{ t('gmData.gift.footHint') }}</span>
            <UButton color="neutral" variant="outline" :loading="running && dryRunning" :disabled="!canRun" @click="run(true)">
              {{ t('gmData.gift.preview') }}
            </UButton>
            <UButton v-if="sent" color="primary" @click="close">{{ t('gmData.gift.done') }}</UButton>
            <UButton v-else color="primary" icon="i-lucide-send" :loading="running && !dryRunning" :disabled="!canRun" @click="run(false)">
              {{ t('gmData.gift.sendTo', { n: targets.length }) }}
            </UButton>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import { executeGmCommand, listGmPlayers } from '@/api/gameGm'
import { getRequestErrorMessage } from '@/utils/request'
import type { CommandResult, TargetResult } from '@/games/mgp'
import type { GiftSendPreviewDetail, PlayerListItem } from '@/games/gakumas/types'
import { useGmContext } from '../context'
import GmCommandResult from '../components/GmCommandResult.vue'
import GmRewardPicker from '../components/GmRewardPicker.vue'
import GmRiskPill from '../components/GmRiskPill.vue'
import type { GmReward } from '../components/types'
import PlayerAvatar from './shared/PlayerAvatar.vue'
import { useCatalogLookup } from './shared/lookup'

export interface GiftTarget {
  pid: string
  name?: string
  avatarIdolCardId?: string
}

// 协议约定：players 指令一次 1–200 个目标
const MAX_TARGETS = 200
const COMMAND = 'gift.send'

const props = withDefaults(defineProps<{ modelValue: boolean; players?: GiftTarget[] }>(), { players: () => [] })
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; done: [] }>()

const { t } = useI18n()
const ctx = useGmContext()
const menuStore = useMenuStore()
const lookup = useCatalogLookup()

const command = computed(() => ctx.command(COMMAND))
const schema = computed(() => command.value?.params?.properties || {})
const maxRewards = computed(() => schema.value.rewards?.maxItems || 20)
const maxMessage = computed(() => schema.value.message?.maxLength || 64)
const expireOptions = computed(() => {
  const min = schema.value.expireDays?.minimum ?? 0
  const max = schema.value.expireDays?.maximum ?? 365
  return [7, 30, 90, 0].filter((d) => d >= min && d <= max)
})

const targets = ref<GiftTarget[]>([])
const rewards = ref<GmReward[]>([])
const message = ref('')
const expireDays = ref(30)
const preview = ref<CommandResult<GiftSendPreviewDetail>>()
const result = ref<CommandResult>()
const error = ref('')
const running = ref(false)
const dryRunning = ref(false)
// 发送成功后不能原样再发一次（改了内容才能再发）
const sent = ref(false)
// 添加玩家：按账号、User ID、昵称搜索（每次最多 8 条）
const playerQ = ref('')
const found = ref<PlayerListItem[]>([])
const searching = ref(false)
const searchError = ref('')

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    targets.value = props.players.slice(0, MAX_TARGETS).map((p) => ({ ...p }))
    rewards.value = []
    message.value = ''
    expireDays.value = expireOptions.value.includes(30) ? 30 : (expireOptions.value[0] ?? 0)
    preview.value = undefined
    result.value = undefined
    error.value = ''
    sent.value = false
    playerQ.value = ''
  },
  { immediate: true },
)

// 改了内容，之前的预览与结果就不再对应
watch(
  [targets, rewards, message, expireDays],
  () => {
    preview.value = undefined
    sent.value = false
  },
  { deep: true },
)

const hasTarget = (pid: string) => targets.value.some((x) => x.pid === pid)
const addTarget = (p: PlayerListItem) => {
  if (hasTarget(p.pid) || targets.value.length >= MAX_TARGETS) return
  targets.value = [...targets.value, { pid: p.pid, name: p.name, avatarIdolCardId: p.avatarIdolCardId }]
}
const removeTarget = (pid: string) => {
  targets.value = targets.value.filter((x) => x.pid !== pid)
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
    const { data } = await listGmPlayers({ serverId: ctx.serverId.value, q, filter: 'all', cursor: '', limit: 8 })
    if (id !== searchSeq) return
    found.value = (data.items as PlayerListItem[] | undefined) || []
  } catch (e) {
    if (id === searchSeq) searchError.value = getRequestErrorMessage(e)
  } finally {
    if (id === searchSeq) searching.value = false
  }
}
const debouncedSearch = useDebounceFn(search, 300)
watch(playerQ, () => void debouncedSearch())

const canRun = computed(
  () =>
    !!command.value &&
    !running.value &&
    !sent.value &&
    targets.value.length > 0 &&
    rewards.value.length > 0,
)

const params = () => {
  const out: Record<string, unknown> = { rewards: rewards.value, expireDays: expireDays.value }
  if (message.value.trim()) out.message = message.value.trim()
  return out
}

const run = async (dryRun: boolean) => {
  if (!canRun.value) return
  running.value = true
  dryRunning.value = dryRun
  error.value = ''
  if (!dryRun) result.value = undefined
  try {
    const { data } = await executeGmCommand({
      serverId: ctx.serverId.value,
      command: COMMAND,
      targets: targets.value.map((x) => x.pid),
      params: params(),
      dryRun,
      confirmName: '',
      summary: {
        message: message.value.trim(),
        rewards: String(rewards.value.length),
        expireDays: String(expireDays.value),
      },
    })
    const res = (data.result as CommandResult | undefined) || undefined
    if (data.status === 'failed' && data.errorCode) error.value = `${data.errorCode} ${data.errorMessage}`
    if (dryRun) {
      preview.value = res as CommandResult<GiftSendPreviewDetail> | undefined
    } else {
      result.value = res
      if (data.status === 'success' || data.status === 'partial') {
        sent.value = true
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

const PREVIEW_ROWS = 5
const previewRows = computed<TargetResult<GiftSendPreviewDetail>[]>(() => (preview.value?.results || []).slice(0, PREVIEW_ROWS))
const previewMore = computed(() => Math.max(0, (preview.value?.results?.length || 0) - PREVIEW_ROWS))

const rewardName = (resourceType: number, resourceId: string) => {
  const rr = lookup.rewardRef(resourceType, resourceId)
  return rr ? lookup.name(rr.kind, rr.id) : t('game.unnamed')
}
const convertsOf = (detail: GiftSendPreviewDetail) =>
  (detail.rewards || [])
    .filter((r) => r.owned && r.convertsTo)
    .map((r) => ({
      name: rewardName(r.resourceType, r.resourceId),
      to: lookup.name('item', r.convertsTo || ''),
      n: r.convertedQuantity || 0,
    }))

const close = () => {
  if (running.value) return
  emit('update:modelValue', false)
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.modelValue) close()
}
watch(
  () => props.modelValue,
  (open) => {
    if (open) window.addEventListener('keydown', onKeydown)
    else window.removeEventListener('keydown', onKeydown)
  },
  { immediate: true },
)
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped>
.gdr {
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: flex;
  justify-content: flex-end;
}
.gdr__scrim {
  position: absolute;
  inset: 0;
  background: color-mix(in srgb, var(--m-ink) 40%, transparent);
}
.gdr__panel {
  position: relative;
  display: flex;
  flex-direction: column;
  width: min(560px, 100%);
  height: 100%;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: -24px 0 48px -24px color-mix(in srgb, var(--m-ink) 45%, transparent);
}
.gdr__head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 18px 20px;
  border-bottom: 1px solid var(--el-border-color-light);
}
.gdr__sticker {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 14px;
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
  transform: rotate(-4deg);
}
.gdr__sticker svg {
  width: 22px;
  height: 22px;
}
.gdr__heading {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.gdr__title {
  font-size: 19px;
  font-weight: 800;
}
.gdr__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gdr__body {
  flex: 1;
  overflow-y: auto;
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 18px;
  scrollbar-width: thin;
}
.gdr__field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.gdr__label {
  font-size: 13px;
  font-weight: 700;
}
.gdr__hint {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.gdr__req {
  margin-left: 2px;
  color: var(--m-bad);
}
.gdr__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  max-height: 160px;
  overflow-y: auto;
}
.gdr__chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 4px 0 5px;
  border-radius: 999px;
  background: var(--m-s2);
  font-size: 13px;
  font-weight: 700;
  max-width: 100%;
}
.gdr__chip-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gdr__chip-x {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.gdr__chip-x:hover {
  background: var(--el-fill-color);
  color: var(--el-text-color-primary);
}
.gdr__chip-x svg {
  width: 16px;
  height: 16px;
}
.gdr__add {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gdr__search {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.gdr__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
}
.gdr__search-icon {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.gdr__found {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 260px;
  overflow-y: auto;
  padding: 4px;
  border-radius: var(--app-radius-sm);
  border: 1px solid var(--el-border-color-light);
}
.gdr__found-row {
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
.gdr__found-row:hover:not(:disabled) {
  background: var(--m-pSoft);
}
.gdr__found-row:disabled {
  cursor: default;
  opacity: 0.7;
}
.gdr__found-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.gdr__found-name {
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gdr__found-check {
  width: 18px;
  height: 18px;
  color: var(--m-ok);
}
.gdr__found-note {
  padding: 8px;
}
.gdr__input {
  min-height: 40px;
  padding: 0 12px;
  border-radius: var(--app-radius-sm);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
}
.gdr__textarea {
  padding: 8px 12px;
  resize: vertical;
}
.gdr__seg {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  align-self: flex-start;
}
.gdr__seg button {
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
.gdr__seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: 0 1px 4px color-mix(in srgb, var(--m-ink) 14%, transparent);
}
.gdr__preview {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px;
  border-radius: var(--app-radius-lg);
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
  font-size: 13px;
}
.gdr__preview-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 800;
}
.gdr__preview-title svg {
  width: 16px;
  height: 16px;
}
.gdr__preview-row {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  line-height: 1.6;
}
.gdr__preview-who {
  font-weight: 800;
}
.gdr__preview-note {
  flex-basis: 100%;
}
.gdr__error {
  font-size: 13px;
  color: var(--m-bad);
}
.gdr__foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  padding: 14px 20px;
  border-top: 1px solid var(--el-border-color-light);
}
.gdr__foot-hint {
  flex: 1 1 200px;
}
.gdr-enter-active,
.gdr-leave-active {
  transition: opacity 0.18s ease;
}
.gdr-enter-active .gdr__panel,
.gdr-leave-active .gdr__panel {
  transition: transform 0.22s ease;
}
.gdr-enter-from,
.gdr-leave-to {
  opacity: 0;
}
.gdr-enter-from .gdr__panel,
.gdr-leave-to .gdr__panel {
  transform: translateX(40px);
}
@media (max-width: 640px) {
  .gdr {
    align-items: flex-end;
  }
  .gdr__panel {
    width: 100%;
    height: 92vh;
    border-radius: 24px 24px 0 0;
  }
  .gdr-enter-from .gdr__panel,
  .gdr-leave-to .gdr__panel {
    transform: translateY(40px);
  }
  .gdr__foot :deep(button) {
    min-height: 44px;
  }
}
</style>
