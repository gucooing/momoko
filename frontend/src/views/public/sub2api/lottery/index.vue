<!-- Sub2API 用户端每日抽奖（P7 独立页，iframe 嵌入）。
     主内容：奖池 / 倒计时 / 报名 / 历史。
     活动说明：首次进入自动弹窗；顶栏左侧按钮可再次打开。 -->
<template>
  <main class="s2a-lot" :class="{ 'is-dark': isDark }">
    <header class="topbar">
      <div class="topbar-inner">
        <div class="topbar-left">
          <button
            class="icon-btn"
            type="button"
            :title="t('sub2api.activity.rulesTitle')"
            :aria-label="t('sub2api.activity.rulesTitle')"
            @click="rulesOpen = true"
          >
            <UIcon name="i-lucide-circle-help" />
          </button>
          <div class="brand">
            <span class="brand-dot" />
            {{ t('sub2api.activity.brand') }}
            <span v-if="store.srcHost" class="brand-host" :title="store.srcHost">{{ store.srcHost }}</span>
          </div>
        </div>
        <button class="icon-btn" type="button" @click="toggleTheme">
          <UIcon :name="isDark ? 'i-lucide-sun' : 'i-lucide-moon'" />
        </button>
      </div>
    </header>

    <section class="body">
      <div v-if="store.loading && !store.status" class="card">
        <EmptyState :title="t('sub2api.activity.loading')" />
      </div>
      <div v-else-if="loadError" class="card">
        <EmptyState :title="t('sub2api.activity.loadFailed')" :description="loadError" />
      </div>
      <div v-else-if="store.status && !store.status.enabled" class="card">
        <EmptyState :title="t('sub2api.activity.disabled')" :description="t('sub2api.activity.needLogin')" />
      </div>

      <template v-else-if="store.status">
        <div class="card hero">
          <div class="hero__label">{{ t('sub2api.activity.accumPool') }}</div>
          <div class="hero__value">{{ fmtMoney(store.status?.accumPool) }}</div>
          <div class="hero__count">
            <div>
              <div class="hero__sub">{{ t('sub2api.activity.countdownSettle') }}</div>
              <div class="hero__cd">{{ settleCd }}</div>
            </div>
            <div>
              <div class="hero__sub">{{ t('sub2api.activity.countdownDraw') }}</div>
              <div class="hero__cd">{{ drawCd }}</div>
            </div>
          </div>
        </div>

        <div v-if="store.status?.current" class="card">
          <div class="card__title">{{ t('sub2api.activity.tabRegistering') }}</div>
          <div class="kv">
            <div class="kv__row"><span>{{ t('sub2api.activity.poolAmount') }}</span><b>{{ fmtMoney(store.status.current.poolAmount) }}</b></div>
            <div class="kv__row"><span>{{ t('sub2api.activity.registered') }}</span><b>{{ store.status.current.registeredCount }}</b></div>
            <div class="kv__row"><span>{{ t('sub2api.activity.eligible') }}</span><b>{{ store.status.current.eligibleCount }}</b></div>
            <div class="kv__row"><span>{{ t('sub2api.activity.mySpend') }}</span><b>{{ fmtMoney(store.status.userSpend) }}</b></div>
          </div>
          <div class="actions">
            <UButton
              v-if="store.status.registered"
              color="neutral"
              variant="soft"
              disabled
            >
              {{ t('sub2api.activity.registeredBtn') }}
            </UButton>
            <UButton
              v-else-if="store.status.eligible"
              color="primary"
              :loading="store.registering"
              @click="onRegister"
            >
              {{ t('sub2api.activity.registerBtn') }}
            </UButton>
            <div v-else class="warn">{{ t('sub2api.activity.notEligible') }}</div>
          </div>
        </div>

        <div class="card">
          <div class="card__title">{{ t('sub2api.activity.myHistory') }}</div>
          <EmptyState
            v-if="!store.history.length"
            :title="t('sub2api.activity.emptyHistory')"
          />
          <div v-else class="hist">
            <div v-for="(item, idx) in store.history" :key="item.round?.id || idx" class="hist__block">
              <div class="hist__row">
                <div class="hist__main">
                  <span class="hist__date">{{ item.round?.id }}</span>
                  <span class="hist__pool">{{ fmtMoney(item.round?.poolAmount) }}</span>
                </div>
                <StatusPill :variant="myVariant(item.myStatus)" :label="myLabel(item)" />
              </div>
              <!-- 中奖名单：userName 后端已对邮箱打码，用户名原样 -->
              <div v-if="item.winners?.length" class="hist__winners">
                <div class="hist__winners-title">{{ t('sub2api.activity.winners') }}</div>
                <div
                  v-for="w in item.winners"
                  :key="w.id || `${w.sub2apiUserId}-${w.prizeAmount}`"
                  class="hist__winner"
                >
                  <span class="hist__winner-name">{{ w.userName || `#${w.sub2apiUserId}` }}</span>
                  <span class="hist__winner-prize">{{ fmtMoney(w.prizeAmount) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </section>

    <!-- 活动说明：首次自动弹出；之后点左上角 ? 再开 -->
    <FormDialog
      v-model="rulesOpen"
      :title="t('sub2api.activity.rulesTitle')"
      :confirm-text="t('sub2api.activity.rulesGotIt')"
      :show-footer="true"
      :close-on-overlay="true"
      :width="440"
      @confirm="closeRules"
      @close="markRulesSeen"
    >
      <template #footer="{ confirm }">
        <UButton color="primary" block @click="confirm">
          {{ t('sub2api.activity.rulesGotIt') }}
        </UButton>
      </template>
      <div class="rules-body">
        <p class="rules__intro">{{ t('sub2api.activity.rulesIntro') }}</p>
        <ol class="rules__list">
          <li>
            <span class="rules__label">{{ t('sub2api.activity.rulesWhatLabel') }}</span>
            <span class="rules__text">{{ t('sub2api.activity.rulesWhat') }}</span>
          </li>
          <li>
            <span class="rules__label">{{ t('sub2api.activity.rulesHowLabel') }}</span>
            <span class="rules__text">{{ t('sub2api.activity.rulesHow') }}</span>
          </li>
          <li>
            <span class="rules__label">{{ t('sub2api.activity.rulesSettleLabel') }}</span>
            <span class="rules__text">{{ t('sub2api.activity.rulesSettle') }}</span>
          </li>
          <li>
            <span class="rules__label">{{ t('sub2api.activity.rulesDrawLabel') }}</span>
            <span class="rules__text">{{ t('sub2api.activity.rulesDraw') }}</span>
          </li>
        </ol>
        <p class="rules__note">{{ t('sub2api.activity.rulesNote') }}</p>
      </div>
    </FormDialog>
  </main>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useLotteryStore } from '@/stores/sub2api/lottery'
import { useThemeStore } from '@/stores/theme'
import { useFeedback } from '@/utils/feedback'
import { LotteryMyStatus, type LotteryHistoryItem } from '@/types/v1/sub2api'

defineOptions({ name: 'Sub2APILottery' })

const RULES_SEEN_KEY = 'sub2api.lottery.rulesSeen'

const route = useRoute()
const store = useLotteryStore()
const themeStore = useThemeStore()
const { t } = useI18n()
const fb = useFeedback()

const isDark = computed(() => themeStore.isDarkTheme)
const toggleTheme = () => themeStore.toggleThemeMode(isDark.value ? 'light' : 'dark')

const rulesOpen = ref(false)
const loadError = ref('')
const now = ref(Date.now())
let timer: number | undefined

const fmtMoney = (n?: number | null) => `$${(Number(n) || 0).toFixed(4)}`

const fmtCd = (target?: Date | string | null) => {
  if (!target) return '—'
  const ts = target instanceof Date ? target.getTime() : new Date(target).getTime()
  if (Number.isNaN(ts)) return '—'
  let diff = Math.max(0, Math.floor((ts - now.value) / 1000))
  const h = Math.floor(diff / 3600)
  diff %= 3600
  const m = Math.floor(diff / 60)
  const s = diff % 60
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

const settleCd = computed(() => fmtCd(store.status?.nextSettleTime))
const drawCd = computed(() => fmtCd(store.status?.nextDrawTime))

const myLabel = (item: LotteryHistoryItem) => {
  switch (item.myStatus) {
    case LotteryMyStatus.LOTTERY_MY_STATUS_WON:
      return t('sub2api.activity.myWon', { amount: fmtMoney(item.myPrize) })
    case LotteryMyStatus.LOTTERY_MY_STATUS_REGISTERED:
      return t('sub2api.activity.myRegistered')
    default:
      return t('sub2api.activity.myNone')
  }
}

const myVariant = (s: LotteryMyStatus): 'success' | 'primary' | 'neutral' => {
  if (s === LotteryMyStatus.LOTTERY_MY_STATUS_WON) return 'success'
  if (s === LotteryMyStatus.LOTTERY_MY_STATUS_REGISTERED) return 'primary'
  return 'neutral'
}

const markRulesSeen = () => {
  try {
    localStorage.setItem(RULES_SEEN_KEY, '1')
  } catch {
    /* ignore quota / private mode */
  }
}

const closeRules = () => {
  markRulesSeen()
  rulesOpen.value = false
}

const maybeOpenRulesFirstVisit = () => {
  try {
    if (localStorage.getItem(RULES_SEEN_KEY) === '1') return
  } catch {
    /* first open anyway */
  }
  rulesOpen.value = true
}

const onRegister = async () => {
  try {
    await store.register()
    fb.success(t('sub2api.activity.registeredBtn'))
  } catch (e: unknown) {
    fb.error(e instanceof Error ? e.message : t('sub2api.activity.loadFailed'))
  }
}

onMounted(async () => {
  if (!store.bootstrap(route.query as Record<string, unknown>)) return
  if (store.theme === 'dark' && !isDark.value) toggleTheme()
  // 先弹规则，再拉数据，避免首屏被状态区抢注意力
  maybeOpenRulesFirstVisit()
  loadError.value = ''
  try {
    await store.loadStatus()
    await store.loadHistory(1)
  } catch (e: unknown) {
    loadError.value = e instanceof Error ? e.message : t('sub2api.activity.loadFailed')
    fb.error(loadError.value)
  }
  timer = window.setInterval(() => {
    now.value = Date.now()
  }, 1000)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<style scoped lang="scss">
.s2a-lot {
  min-height: 100vh;
  background: var(--el-bg-color-page);
  color: var(--el-text-color-primary);
  display: flex;
  flex-direction: column;
}
.topbar {
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: color-mix(in srgb, var(--el-bg-color-page) 80%, transparent);
  backdrop-filter: blur(12px);
  position: sticky;
  top: 0;
  z-index: 10;
}
.topbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  gap: 12px;
}
.topbar-left {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1;
}
.brand {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.9375rem;
  font-weight: 600;
  min-width: 0;
}
.brand-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--el-color-primary);
  flex: none;
}
.brand-host {
  font-size: 0.75rem;
  font-weight: 400;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.icon-btn {
  width: 34px;
  height: 34px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-sm);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  flex: none;
}
.icon-btn:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.body {
  flex: 1;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 720px;
  width: 100%;
  margin: 0 auto;
  box-sizing: border-box;
}
.card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  padding: 16px 18px;
}
.card__title {
  font-size: 0.875rem;
  font-weight: 600;
  margin-bottom: 12px;
}
.rules-body {
  display: flex;
  flex-direction: column;
  gap: 0;
}
.rules__intro {
  margin: 0 0 12px;
  font-size: 0.8125rem;
  line-height: 1.55;
  color: var(--el-text-color-secondary);
}
.rules__list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.rules__list li {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-lighter);
}
.rules__label {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--el-color-primary);
  letter-spacing: 0.04em;
}
.rules__text {
  font-size: 0.8125rem;
  line-height: 1.55;
  color: var(--el-text-color-regular);
}
.rules__note {
  margin: 12px 0 0;
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--el-text-color-placeholder);
}
.hero__label {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
.hero__value {
  margin-top: 6px;
  font-size: 1.75rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
}
.hero__count {
  margin-top: 14px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.hero__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.hero__cd {
  margin-top: 4px;
  font-size: 1rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.kv {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.kv__row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  b {
    color: var(--el-text-color-primary);
    font-variant-numeric: tabular-nums;
    font-weight: 600;
  }
}
.actions {
  margin-top: 14px;
}
.warn {
  font-size: 0.8125rem;
  color: var(--el-color-warning);
}
.hist {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.hist__block {
  padding: 10px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.hist__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.hist__main {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.hist__date {
  font-size: 0.8125rem;
  font-weight: 500;
}
.hist__pool {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.hist__winners {
  margin-top: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
}
.hist__winners-title {
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
}
.hist__winner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 3px 0;
  font-size: 0.8125rem;
}
.hist__winner-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.hist__winner-prize {
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  flex-shrink: 0;
  color: var(--el-color-primary);
}
</style>
