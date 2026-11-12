<!-- GM 操作 · 玩家权限：玩家在通行证「自助服务」里能执行哪些 GM 操作；只列出允许玩家自助的操作。
     每条只有五项设置：开放给玩家 / 每天次数 / 两次之间至少间隔（0 为不限）/ 需要审批 / 玩家看到的说明；
     改动自动保存、立即生效。玩家的审批申请在「GM 记录」里处理。 -->
<template>
  <div class="gss">
    <div class="gss__head">
      <p class="gss__sub">
        {{ t('gmOps.self.subline') }} ·
        <span class="gss__open">{{ t('gmOps.self.openCount', { n: openCount }) }}</span>
        · {{ t('gmOps.self.instant') }}
      </p>
    </div>

    <section class="gss__card">
      <div v-if="!isMobile && rows.length" class="gss__row gss__row--head">
        <span />
        <span>{{ t('gmOps.self.col.command') }}</span>
        <span>{{ t('gmOps.self.col.open') }}</span>
        <span>{{ t('gmOps.self.col.limits') }}</span>
        <span>{{ t('gmOps.self.col.approval') }}</span>
        <span />
      </div>

      <div v-if="loading && !rows.length" class="gss__loading">{{ t('gmOps.common.loading') }}</div>
      <EmptyState
        v-else-if="!rows.length"
        icon="HOutline:IdentificationIcon"
        :title="loadError ? t('gmOps.self.loadFailed') : t('gmOps.self.empty')"
        :description="loadError || t('gmOps.self.emptyDesc')"
      />

      <div v-for="row in rows" :key="row.command" class="gss__item" :class="{ 'is-open': expanded === row.command }">
        <div class="gss__row">
          <span class="gss__avatar m-sticker" :class="row.enabled ? 'm-sticker--p' : ''" aria-hidden="true">{{ row.title.slice(0, 1) }}</span>
          <span class="gss__cmd">
            <span class="gss__title">{{ row.title }}</span>
            <span v-if="row.group" class="gss__hint">{{ row.group }}</span>
          </span>
          <span class="gss__cell">
            <span v-if="isMobile" class="gss__cell-label">{{ t('gmOps.self.col.open') }}</span>
            <AppSwitch :model-value="row.enabled" :aria-label="t('gmOps.self.openAria', { title: row.title })" @update:model-value="(v) => set(row, { enabled: v })" />
          </span>
          <span class="gss__limits">
            <template v-if="row.enabled">
              <span class="gss__tag">{{ dailyText(row.dailyLimit) }}</span>
              <span class="gss__tag">{{ cooldownTag(row.cooldownSec) }}</span>
            </template>
            <span v-else class="gss__hint">{{ t('gmOps.self.closed') }}</span>
          </span>
          <span class="gss__cell">
            <span v-if="isMobile" class="gss__cell-label">{{ t('gmOps.self.col.approval') }}</span>
            <AppSwitch
              :model-value="row.needApproval"
              :disabled="!row.enabled"
              :aria-label="t('gmOps.self.approvalAria', { title: row.title })"
              @update:model-value="(v) => set(row, { needApproval: v })"
            />
          </span>
          <span class="gss__end">
            <span v-if="state[row.command]" class="gss__state" :class="`is-${state[row.command]}`">{{ t(`gmOps.self.state.${state[row.command]}`) }}</span>
            <AppIconButton
              icon="HOutline:ChevronRightIcon"
              class="gss__chev"
              :class="{ 'is-rot': expanded === row.command }"
              :label="t('gmOps.self.editAria', { title: row.title })"
              :aria-expanded="expanded === row.command"
              @click="expanded = expanded === row.command ? '' : row.command"
            />
          </span>
        </div>

        <div v-if="expanded === row.command" class="gss__ex">
          <div class="gss__field">
            <span class="gss__label">{{ t('gmOps.self.daily') }}</span>
            <span class="gss__stepper-wrap">
              <span class="gss__stepper">
                <button type="button" :aria-label="t('gmOps.self.decrease')" :disabled="row.dailyLimit <= 0" @click="set(row, { dailyLimit: Math.max(0, row.dailyLimit - 1) }, 500)">−</button>
                <span class="m-num">{{ row.dailyLimit || '∞' }}</span>
                <button type="button" :aria-label="t('gmOps.self.increase')" :disabled="row.dailyLimit >= 1000" @click="set(row, { dailyLimit: row.dailyLimit + 1 }, 500)">+</button>
              </span>
              <span class="gss__hint">{{ row.dailyLimit ? t('gmOps.self.timesPerDay') : t('gmOps.self.unlimitedHint') }}</span>
            </span>
          </div>
          <div class="gss__field">
            <span class="gss__label">{{ t('gmOps.self.cooldown') }}</span>
            <div class="gss__seg" role="radiogroup" :aria-label="t('gmOps.self.cooldown')">
              <button
                v-for="sec in cooldownPresets"
                :key="sec"
                type="button"
                role="radio"
                class="gss__seg-btn"
                :class="{ 'is-on': row.cooldownSec === sec && !custom[row.command] }"
                :aria-checked="row.cooldownSec === sec && !custom[row.command]"
                @click="pickCooldown(row, sec)"
              >
                {{ sec ? cooldownText(sec) : t('gmOps.self.noLimit') }}
              </button>
              <button
                type="button"
                role="radio"
                class="gss__seg-btn"
                :class="{ 'is-on': isCustom(row) }"
                :aria-checked="isCustom(row)"
                @click="custom[row.command] = true"
              >
                {{ isCustom(row) && !custom[row.command] ? cooldownText(row.cooldownSec) : t('gmOps.self.custom') }}
              </button>
            </div>
            <label v-if="custom[row.command]" class="gss__inline">
              <input
                type="number"
                min="1"
                :max="30 * 24 * 60"
                class="gss__input gss__input--short m-num"
                :value="Math.round(row.cooldownSec / 60) || ''"
                :aria-label="t('gmOps.self.minutes')"
                @input="onMinutes(row, ($event.target as HTMLInputElement).valueAsNumber)"
              />
              {{ t('gmOps.self.minutes') }}
            </label>
          </div>
          <label class="gss__field gss__field--grow">
            <span class="gss__label">{{ t('gmOps.self.note') }}</span>
            <input
              type="text"
              maxlength="200"
              class="gss__input"
              :value="row.playerNote"
              :placeholder="t('gmOps.self.notePlaceholder')"
              @input="set(row, { playerNote: ($event.target as HTMLInputElement).value }, 800)"
              @blur="flush(row)"
            />
          </label>
        </div>
      </div>
    </section>

    <div class="gss__info">
      <UIcon name="i-lucide-info" class="gss__info-icon" />
      <i18n-t keypath="gmOps.self.info" tag="span" scope="global">
        <template #logs>
          <RouterLink :to="{ path: route.path, query: { tab: 'logs' } }">{{ t('gm.tabs.logs') }}</RouterLink>
        </template>
      </i18n-t>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { listGmSelfService, updateGmSelfService } from '@/api/gameGm'
import type { GmSelfServicePolicy } from '@/types/v1/game_gm'
import { getRequestErrorMessage } from '@/utils/request'
import { useGmContext } from '../../context'

defineOptions({ name: 'GmPermissionsPanel' })

type SaveState = 'saving' | 'saved' | 'failed'
type Patch = Partial<Pick<GmSelfServicePolicy, 'enabled' | 'dailyLimit' | 'cooldownSec' | 'needApproval' | 'playerNote'>>

const { t } = useI18n()
const route = useRoute()
const ctx = useGmContext()
const menuStore = useMenuStore()
const isMobile = computed(() => menuStore.isMobile)

const rows = ref<GmSelfServicePolicy[]>([])
const loading = ref(false)
const loadError = ref('')
const expanded = ref('')
const state = reactive<Record<string, SaveState | undefined>>({})
const custom = reactive<Record<string, boolean>>({})

const cooldownPresets = [0, 3600, 6 * 3600, 24 * 3600]
const openCount = computed(() => rows.value.filter((r) => r.enabled).length)

const load = async () => {
  loading.value = true
  try {
    const { data } = await listGmSelfService(ctx.serverId.value)
    rows.value = data.items || []
    loadError.value = ''
  } catch (e) {
    loadError.value = getRequestErrorMessage(e)
  } finally {
    loading.value = false
  }
}

// ---- 自动保存：每条指令各自防抖；保存期间又有改动时，结束后再存一次 ----
const timers = new Map<string, ReturnType<typeof setTimeout>>()
const inflight = new Set<string>()
const again = new Set<string>()

const save = async (row: GmSelfServicePolicy) => {
  const key = row.command
  if (inflight.has(key)) {
    again.add(key)
    return
  }
  inflight.add(key)
  state[key] = 'saving'
  try {
    const { data } = await updateGmSelfService({
      serverId: ctx.serverId.value,
      command: row.command,
      enabled: row.enabled,
      dailyLimit: row.dailyLimit,
      cooldownSec: row.cooldownSec,
      needApproval: row.needApproval,
      playerNote: row.playerNote,
    })
    if (!again.has(key)) {
      row.updatedByName = data.updatedByName
      row.updateTime = data.updateTime
    }
    state[key] = 'saved'
    ctx.selfServiceEnabled.value = openCount.value
  } catch {
    state[key] = 'failed'
  } finally {
    inflight.delete(key)
    if (again.delete(key)) void save(row)
  }
}

const set = (row: GmSelfServicePolicy, patch: Patch, delay = 0) => {
  Object.assign(row, patch)
  const key = row.command
  clearTimeout(timers.get(key))
  if (!delay) {
    timers.delete(key)
    void save(row)
    return
  }
  state[key] = 'saving'
  timers.set(
    key,
    setTimeout(() => {
      timers.delete(key)
      void save(row)
    }, delay),
  )
}

const flush = (row: GmSelfServicePolicy) => {
  const timer = timers.get(row.command)
  if (!timer) return
  clearTimeout(timer)
  timers.delete(row.command)
  void save(row)
}

const pickCooldown = (row: GmSelfServicePolicy, sec: number) => {
  custom[row.command] = false
  set(row, { cooldownSec: sec })
}
const isCustom = (row: GmSelfServicePolicy) => !!custom[row.command] || !cooldownPresets.includes(row.cooldownSec)
const onMinutes = (row: GmSelfServicePolicy, minutes: number) => {
  if (!Number.isFinite(minutes) || minutes < 1) return
  set(row, { cooldownSec: Math.min(Math.round(minutes), 30 * 24 * 60) * 60 }, 600)
}

// ---- 文本 ----
const cooldownText = (sec: number) => {
  if (sec % 86400 === 0) return t('gmOps.self.days', { n: sec / 86400 })
  if (sec % 3600 === 0) return t('gmOps.self.hours', { n: sec / 3600 })
  return t('gmOps.self.mins', { n: Math.max(1, Math.round(sec / 60)) })
}
const dailyText = (n: number) => (n ? t('gmOps.self.perDay', { n }) : t('gmOps.self.unlimited'))
const cooldownTag = (sec: number) => (sec ? t('gmOps.self.gap', { gap: cooldownText(sec) }) : t('gmOps.self.noGap'))

let activatedOnce = false
onMounted(() => void load())
onActivated(() => {
  if (activatedOnce) void load()
  activatedOnce = true
})
</script>

<style scoped>
.gss {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.gss__head {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.gss__h {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 30px;
  font-weight: 800;
  letter-spacing: -0.01em;
}
.gss__sub {
  margin: 0;
  font-size: 14px;
  color: var(--el-text-color-secondary);
}
.gss__open {
  color: var(--m-ok);
  font-weight: 700;
}
.gss__card {
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  overflow: hidden;
}
.gss__loading {
  padding: 24px;
  text-align: center;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.gss__row {
  display: grid;
  grid-template-columns: 44px minmax(0, 1.7fr) 96px minmax(0, 1.2fr) 110px 120px;
  gap: 14px;
  align-items: center;
  min-height: 72px;
  padding: 0 18px;
}
.gss__row--head {
  min-height: 40px;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gss__item {
  border-top: 1px solid var(--el-border-color-light);
}
.gss__item.is-open > .gss__row,
.gss__item.is-open > .gss__ex {
  background: var(--m-gSoft, var(--m-pSoft));
}
.gss__avatar {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-family: var(--app-font-display);
  font-size: 18px;
  font-weight: 800;
  background: var(--el-fill-color-light);
}
.gss__cmd {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gss__title {
  font-size: 15px;
  font-weight: 800;
  overflow-wrap: anywhere;
}
.gss__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
}
.gss__cell {
  display: flex;
  align-items: center;
  gap: 10px;
}
.gss__cell-label {
  font-size: 13px;
  font-weight: 700;
}
.gss__limits {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}
.gss__tag {
  display: inline-flex;
  align-items: center;
  min-height: 26px;
  padding: 0 10px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  font-size: 12px;
  font-weight: 700;
}
.gss__end {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
}
.gss__chev :deep(svg) {
  transition: transform 0.15s ease;
}
.gss__chev.is-rot :deep(svg) {
  transform: rotate(90deg);
}
.gss__state {
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gss__state.is-saved {
  color: var(--m-ok);
}
.gss__state.is-failed {
  color: var(--m-bad);
}
.gss__ex {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  padding: 4px 18px 20px 76px;
}
.gss__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.gss__field--grow {
  flex: 1 1 260px;
}
.gss__label {
  font-size: 13px;
  font-weight: 700;
}
.gss__stepper-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
}
.gss__stepper {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  overflow: hidden;
}
.gss__stepper button {
  width: 40px;
  height: 40px;
  border: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 18px;
  cursor: pointer;
}
.gss__stepper button:disabled {
  opacity: 0.4;
  cursor: default;
}
.gss__stepper span {
  min-width: 34px;
  text-align: center;
  font-weight: 800;
}
.gss__seg {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.gss__seg-btn {
  min-height: 36px;
  padding: 0 12px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}
.gss__seg-btn.is-on {
  background: var(--m-ink);
  color: var(--m-bg);
}
.gss__inline {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.gss__input {
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
}
.gss__input:focus {
  border-color: var(--m-p);
}
.gss__input--short {
  width: 110px;
}
.gss__info {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 14px 16px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
  font-size: 13px;
  line-height: 1.7;
  color: var(--el-text-color-secondary);
}
.gss__info-icon {
  flex: none;
  margin-top: 4px;
}
.gss__info a {
  color: var(--m-pInk);
  font-weight: 700;
}
@media (width < 768px) {
  .gss__h {
    font-size: 24px;
  }
  .gss__row {
    grid-template-columns: 44px minmax(0, 1fr) auto;
    grid-template-areas:
      'av cmd end'
      'open open open'
      'lim lim lim'
      'appr appr appr';
    gap: 10px 12px;
    padding: 14px 16px;
  }
  .gss__row > .gss__avatar {
    grid-area: av;
  }
  .gss__row > .gss__cmd {
    grid-area: cmd;
  }
  .gss__row > .gss__end {
    grid-area: end;
  }
  .gss__row > .gss__cell:nth-of-type(3) {
    grid-area: open;
  }
  .gss__row > .gss__limits {
    grid-area: lim;
  }
  .gss__row > .gss__cell:nth-of-type(5) {
    grid-area: appr;
  }
  .gss__cell {
    justify-content: space-between;
    min-height: 44px;
  }
  .gss__ex {
    flex-direction: column;
    padding: 4px 16px 16px;
  }
  .gss__seg-btn,
  .gss__stepper button {
    min-height: 44px;
    min-width: 44px;
  }
}
</style>
