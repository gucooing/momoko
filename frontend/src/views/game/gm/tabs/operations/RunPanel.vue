<!-- 操作与日志 · 执行操作：按步骤来。
     ① 选择操作：按模块（操作的 group）分组铺开，紧凑的小按钮，带风险色点；
     ② 选择玩家：玩家列表（按状态筛选、按账号名 / User ID / 昵称 / 通行证搜索、页码分页，勾选玩家，翻页后勾选仍保留；
        ?pid=a,b 预先勾选这些玩家，从玩家列表带过来时 history.state.players 带着昵称与头像）；
     ③ 填写参数并执行（GmOperationPanel；发礼物用 GmGiftSend，收件人就是勾选的玩家），对勾选的全部玩家执行。
     当前这一步展开，做完的步骤收成一行摘要（点「修改」回到那一步），后面的步骤先灰着。
     援助设置在支援卡、回忆的详情里执行，存档备份与回档在玩家详情页执行，这里不列出。 -->
<template>
  <div class="gro">
    <!-- ① 选择操作 -->
    <section class="gro__step" :class="{ 'is-active': step === 1, 'is-done': step !== 1 && operation }">
      <div class="gro__step-head">
        <span class="gro__no m-num" aria-hidden="true">
          <UIcon v-if="step !== 1 && operation" name="i-lucide-check" />
          <template v-else>1</template>
        </span>
        <h3 class="gro__h">{{ t('gmOps.run.stepOperation') }}</h3>
        <template v-if="step !== 1 && operation">
          <span class="gro__sum">{{ operation.title }}</span>
          <GmRiskPill :risk="operation.risk" />
          <button type="button" class="gro__link" @click="step = 1">{{ t('gmOps.run.change') }}</button>
        </template>
      </div>
      <div v-if="step === 1" class="gro__groups">
        <div v-for="g in groups" :key="g.name" class="gro__group">
          <span class="gro__group-name">{{ g.name }}</span>
          <div class="gro__ops">
            <button
              v-for="op in g.ops"
              :key="op.id"
              type="button"
              class="gro__op"
              :class="{ 'is-on': opId === op.id }"
              :aria-pressed="opId === op.id"
              :title="t('gmOps.run.risk', { risk: t(`gm.risk.${op.risk}`) })"
              @click="chooseOperation(op.id)"
            >
              <span class="gro__dot" :class="`is-${op.risk}`" aria-hidden="true" />
              {{ op.title }}
            </button>
          </div>
        </div>
        <p class="gro__dim gro__legend">
          <span v-for="r in RISKS" :key="r" class="gro__legend-item"><span class="gro__dot" :class="`is-${r}`" aria-hidden="true" />{{ t(`gm.risk.${r}`) }}</span>
        </p>
      </div>
    </section>

    <!-- ② 选择玩家 -->
    <section class="gro__step" :class="{ 'is-active': step === 2, 'is-done': step === 3, 'is-todo': step < 2 }">
      <div class="gro__step-head">
        <span class="gro__no m-num" aria-hidden="true">
          <UIcon v-if="step === 3" name="i-lucide-check" />
          <template v-else>2</template>
        </span>
        <h3 class="gro__h">{{ t('gmOps.run.stepPlayers') }}</h3>
        <template v-if="step === 2">
          <span class="gro__dim m-num">{{ t('gmOps.run.selected', { n: selected.length }) }}</span>
          <button v-if="selected.length" type="button" class="gro__link" @click="selected = []">{{ t('gmOps.run.clear') }}</button>
        </template>
        <template v-else-if="step === 3">
          <span class="gro__sum">{{ selectedSummary }}</span>
          <button type="button" class="gro__link" @click="step = 2">{{ t('gmOps.run.change') }}</button>
        </template>
      </div>
      <template v-if="step === 2">
        <ListToolbar v-model:status="filter" v-model:keyword="keyword" :chips="chips" :placeholder="t('gmOps.run.search')" @search="reload" />

        <div v-if="listError" class="gro__error">{{ listError }}</div>
        <div class="gro__table" role="table">
          <div class="gro__row gro__row--head" role="row">
            <label class="gro__check" :aria-label="t('gmOps.run.selectPage')">
              <input type="checkbox" :checked="pageAllSelected" :indeterminate="pageSomeSelected" :disabled="!players.length" @change="togglePage" />
            </label>
            <span>{{ t('gmData.players.colPlayer') }}</span>
            <span class="gro__num">{{ t('gmData.players.colLevel') }}</span>
            <span>{{ t('gmData.players.colStatus') }}</span>
          </div>
          <div v-if="loading && !players.length" class="gro__dim gro__pad">{{ t('gmOps.common.loading') }}</div>
          <div v-else-if="!players.length" class="gro__dim gro__pad">{{ t('gmOps.run.noPlayer') }}</div>
          <label v-for="p in players" :key="p.pid" class="gro__row" :class="{ 'is-on': isSelected(p.pid) }" role="row">
            <span class="gro__check">
              <input type="checkbox" :checked="isSelected(p.pid)" @change="toggle(p)" />
            </span>
            <span class="gro__player">
              <PlayerAvatar :idol-card-id="p.avatarIdolCardId" :name="p.name" :size="32" :online="p.online" />
              <span class="gro__player-text">
                <span class="gro__strong">{{ p.name || t('gm.player.noName') }}</span>
                <span class="gro__dim m-num">{{ p.displayId }} · {{ p.account }}</span>
              </span>
            </span>
            <span class="gro__num m-num">{{ p.level }}</span>
            <span class="gro__dim">{{ p.online ? t('gmData.players.filter.online') : p.bannedUntil ? t('gmData.players.filter.banned') : '—' }}</span>
          </label>
        </div>
        <div class="gro__foot">
          <Pagination v-if="total" v-model:page="page" v-model:page-size="pageSize" :total="total" @change="load" />
          <span class="gro__spacer" />
          <UButton color="primary" trailing-icon="i-lucide-arrow-right" :disabled="!selected.length" @click="step = 3">
            {{ t('gmOps.run.next') }}
          </UButton>
        </div>
      </template>
    </section>

    <!-- ③ 填写参数并执行 -->
    <section class="gro__step" :class="{ 'is-active': step === 3, 'is-todo': step < 3 }">
      <div class="gro__step-head">
        <span class="gro__no m-num" aria-hidden="true">3</span>
        <h3 class="gro__h">{{ t('gmOps.run.stepParams') }}</h3>
      </div>
      <template v-if="step === 3 && operation">
        <GmGiftSend v-if="operation.id === 'gift.send'" :players="targets" fixed-targets :reset-key="resetKey" @done="afterRun" />
        <GmOperationPanel v-else :operation="operation" :targets="targets" :reset-key="resetKey" @done="afterRun" />
      </template>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { getPlayer, listPlayers } from '@/api/gakumas'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import type { OperationDef } from '@/games/mgp'
import type { Player, PlayerCounts } from '@/games/gakumas/types'
import { getRequestErrorMessage } from '@/utils/request'
import { useGmContext } from '../../context'
import GmGiftSend from '../../components/GmGiftSend.vue'
import GmOperationPanel from '../../components/GmOperationPanel.vue'
import GmRiskPill from '../../components/GmRiskPill.vue'
import { DETAIL_OPERATIONS, PLAYER_PAGE_OPERATIONS } from '../../operations'
import PlayerAvatar from '../../player/shared/PlayerAvatar.vue'

type Filter = 'all' | 'online' | 'activeToday' | 'banned'
type Step = 1 | 2 | 3

/** 勾选的玩家（翻页后仍保留）。 */
interface PickedPlayer {
  pid: string
  name: string
  avatarIdolCardId: string
}

const RISKS = ['low', 'medium', 'high', 'danger'] as const

const { t } = useI18n()
const route = useRoute()
const ctx = useGmContext()

const step = ref<Step>(1)

// ---------- ① 操作 ----------
const HIDDEN = [...DETAIL_OPERATIONS, ...PLAYER_PAGE_OPERATIONS]
const operations = computed(() => ctx.operations.filter((op) => op.target !== 'server' && !HIDDEN.includes(op.id)))
/** 按模块分组，模块与组内顺序沿用操作定义里的顺序。 */
const groups = computed(() => {
  const out: { name: string; ops: OperationDef[] }[] = []
  for (const op of operations.value) {
    const name = op.group || t('gmOps.run.otherGroup')
    const g = out.find((x) => x.name === name)
    if (g) g.ops.push(op)
    else out.push({ name, ops: [op] })
  }
  return out
})
const opId = ref<string | undefined>(undefined)
const operation = computed(() => operations.value.find((op) => op.id === opId.value))
// 选好操作后去选玩家；从参数这一步回来换操作时，玩家已经确认过就直接回到参数
const playersConfirmed = ref(false)
const chooseOperation = (id: string) => {
  opId.value = id
  step.value = playersConfirmed.value && selected.value.length ? 3 : 2
}
watch(step, (s) => {
  if (s === 3) playersConfirmed.value = true
  if (s === 2) playersConfirmed.value = false
})

// ---------- ② 玩家列表 ----------
const filter = ref<Filter>('all')
const keyword = ref('')
const players = ref<Player[]>([])
const summary = ref<PlayerCounts>()
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const listError = ref('')
let seq = 0

const load = async () => {
  const id = ++seq
  loading.value = true
  listError.value = ''
  try {
    const data = await listPlayers({
      serverId: ctx.serverId.value,
      q: keyword.value.trim(),
      filter: filter.value,
      cursor: page.value > 1 ? String((page.value - 1) * pageSize.value) : '',
      limit: pageSize.value,
    })
    if (id !== seq) return
    players.value = data.items
    total.value = data.total
    summary.value = data.summary
  } catch (e) {
    if (id === seq) listError.value = getRequestErrorMessage(e)
  } finally {
    if (id === seq) loading.value = false
  }
}
const reload = () => {
  page.value = 1
  return load()
}
watch(filter, () => void reload())
void load()

const chips = computed<ToolbarChip<Filter>[]>(() => {
  const s = summary.value
  return (['all', 'online', 'activeToday', 'banned'] as Filter[]).map((f) => ({
    label: t(`gmData.players.filter.${f}`),
    value: f,
    count: s ? (s[f] ?? null) : null,
  }))
})

// ---------- 勾选 ----------
const selected = ref<PickedPlayer[]>([])
const isSelected = (pid: string) => selected.value.some((p) => p.pid === pid)
const pick = (p: Pick<Player, 'pid' | 'name' | 'avatarIdolCardId'>): PickedPlayer => ({ pid: p.pid, name: p.name, avatarIdolCardId: p.avatarIdolCardId })
const toggle = (p: Player) => {
  selected.value = isSelected(p.pid) ? selected.value.filter((x) => x.pid !== p.pid) : [...selected.value, pick(p)]
}
const pageAllSelected = computed(() => players.value.length > 0 && players.value.every((p) => isSelected(p.pid)))
const pageSomeSelected = computed(() => !pageAllSelected.value && players.value.some((p) => isSelected(p.pid)))
const togglePage = () => {
  if (pageAllSelected.value) {
    const onPage = new Set(players.value.map((p) => p.pid))
    selected.value = selected.value.filter((x) => !onPage.has(x.pid))
  } else {
    selected.value = [...selected.value, ...players.value.filter((p) => !isSelected(p.pid)).map(pick)]
  }
}
/** 收起后的摘要：前三个昵称 + 共几名。 */
const selectedSummary = computed(() => {
  const names = selected.value.slice(0, 3).map((p) => p.name || t('gm.player.noName'))
  return selected.value.length > 3 ? t('gmOps.run.selectedMore', { names: names.join('、'), n: selected.value.length }) : names.join('、')
})

// 从玩家页、玩家列表进来时带着 ?pid=a,b：预先勾选这些玩家（history.state.players 里已有的不再查询）
const queryPids = () =>
  ([] as unknown[])
    .concat(route.query.pid ?? [])
    .flatMap((v) => String(v ?? '').split(','))
    .map((v) => v.trim())
    .filter(Boolean)
watch(
  () => queryPids().join(','),
  async () => {
    const pids = queryPids().filter((pid) => !isSelected(pid))
    if (!pids.length) return
    const carried = new Map(((history.state?.players as PickedPlayer[] | undefined) || []).map((p) => [p.pid, p]))
    const found = await Promise.all(
      pids.map(async (pid) => {
        const known = carried.get(pid)
        if (known) return pick(known)
        try {
          const { player } = await getPlayer(ctx.serverId.value, pid)
          return player ? pick(player) : undefined
        } catch (e) {
          listError.value = getRequestErrorMessage(e)
          return undefined
        }
      }),
    )
    selected.value = [...selected.value, ...found.filter((p): p is PickedPlayer => !!p && !isSelected(p.pid))]
  },
  { immediate: true },
)
// 玩家全部取消勾选时不能停在参数这一步
watch(
  () => selected.value.length,
  (n) => {
    if (!n && step.value === 3) step.value = 2
  },
)

// ---------- ③ 参数与执行 ----------
const targets = computed(() => selected.value.map((p) => ({ pid: p.pid, name: p.name, avatarIdolCardId: p.avatarIdolCardId })))

// 换操作或换玩家时清空表单与结果
const resetKey = ref(0)
watch([opId, () => selected.value.map((p) => p.pid).join(',')], () => resetKey.value++)

// 执行后刷新玩家列表（状态、等级可能变了）
const afterRun = () => void load()
</script>

<style scoped>
.gro {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}
.gro__step {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 14px 18px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.gro__step.is-active {
  padding-block: 16px 18px;
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.gro__step.is-todo {
  opacity: 0.55;
}
.gro__step-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 10px;
  min-width: 0;
}
.gro__no {
  flex: none;
  width: 24px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 800;
}
.gro__step.is-active .gro__no {
  background: var(--m-ink);
  color: var(--m-bg);
}
.gro__step.is-done .gro__no {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.gro__h {
  margin: 0;
  font-size: 15px;
  font-weight: 800;
}
.gro__sum {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 700;
}
.gro__dim {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gro__pad {
  padding: 14px 4px;
}
.gro__strong {
  font-weight: 700;
}
.gro__error {
  font-size: 13px;
  color: var(--m-bad);
}
.gro__link {
  margin-left: auto;
  border: 0;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}

/* ① 操作：每个模块一行，左边模块名，右边紧凑的操作按钮 */
.gro__groups {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.gro__group {
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr);
  gap: 10px;
  align-items: baseline;
}
.gro__group-name {
  font-size: 12px;
  font-weight: 800;
  color: var(--el-text-color-secondary);
}
.gro__ops {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.gro__op {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 0 12px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}
.gro__op:hover {
  border-color: var(--m-p);
}
.gro__op.is-on {
  border-color: var(--m-ink);
  background: var(--m-ink);
  color: var(--m-bg);
}
.gro__dot {
  flex: none;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--el-border-color);
}
.gro__dot.is-medium {
  background: var(--m-p);
}
.gro__dot.is-high {
  background: var(--m-warn);
}
.gro__dot.is-danger {
  background: var(--m-bad);
}
.gro__legend {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin: 2px 0 0 66px;
}
.gro__legend-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

/* ② 玩家 */
.gro__table {
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--el-border-color-light);
}
.gro__row {
  display: grid;
  grid-template-columns: 32px minmax(0, 1fr) 60px 90px;
  gap: 10px;
  align-items: center;
  min-height: 52px;
  padding: 4px 6px;
  border-bottom: 1px solid var(--el-border-color-light);
  cursor: pointer;
}
.gro__row--head {
  min-height: 36px;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  cursor: default;
}
.gro__row.is-on {
  background: var(--m-pSoft);
}
.gro__check {
  display: flex;
  align-items: center;
  justify-content: center;
}
.gro__check input {
  width: 16px;
  height: 16px;
  accent-color: var(--m-p);
  cursor: pointer;
}
.gro__player {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.gro__player-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.gro__player-text > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gro__num {
  text-align: right;
}
.gro__foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.gro__spacer {
  flex: 1;
}

@media (width < 768px) {
  .gro__step {
    padding: 12px 14px;
  }
  .gro__group {
    grid-template-columns: minmax(0, 1fr);
    gap: 6px;
  }
  .gro__legend {
    margin-left: 0;
  }
  .gro__op {
    min-height: 36px;
  }
}
</style>
