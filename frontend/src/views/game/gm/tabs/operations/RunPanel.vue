<!-- GM 操作 · 执行操作：左边玩家列表（按状态筛选、按账号名 / User ID / 昵称 / 通行证搜索、页码分页，勾选玩家，
     翻页后勾选仍保留；?pid=a,b 预先勾选这些玩家，从玩家列表带过来时 history.state.players 带着昵称与头像），
     右边选择操作并填写参数执行（GmOperationPanel；发礼物用 GmGiftSend，收件人就是勾选的玩家）。每个操作都对勾选的
     全部玩家执行。援助设置在支援卡、回忆的详情里执行，存档备份与回档在玩家详情页执行，这里不列出。 -->
<template>
  <div class="gro">
    <section class="gro__card gro__players">
      <div class="gro__head">
        <h3 class="gro__h">{{ t('gmOps.run.players') }}</h3>
        <span class="gro__dim m-num">{{ t('gmOps.run.selected', { n: selected.length }) }}</span>
        <button v-if="selected.length" type="button" class="gro__link" @click="selected = []">{{ t('gmOps.run.clear') }}</button>
      </div>
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
      <Pagination v-if="total" v-model:page="page" v-model:page-size="pageSize" :total="total" @change="load" />
    </section>

    <section class="gro__card gro__op">
      <h3 class="gro__h">{{ t('gmOps.run.operation') }}</h3>
      <AppSelect v-model="opId" :options="opOptions" :placeholder="t('gmOps.run.pickOperation')" searchable />
      <template v-if="operation">
        <p v-if="!selected.length" class="gro__warn">{{ t('gmOps.run.needAny') }}</p>
        <GmGiftSend v-else-if="operation.id === 'gift.send'" :players="targets" fixed-targets :reset-key="resetKey" @done="afterRun" />
        <GmOperationPanel v-else :operation="operation" :targets="targets" :reset-key="resetKey" @done="afterRun" />
      </template>
      <p v-else class="gro__dim">{{ t('gmOps.run.pickHint') }}</p>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { getPlayer, listPlayers } from '@/api/gakumas'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import type { Player, PlayerCounts } from '@/games/gakumas/types'
import { getRequestErrorMessage } from '@/utils/request'
import { useGmContext } from '../../context'
import GmGiftSend from '../../components/GmGiftSend.vue'
import GmOperationPanel from '../../components/GmOperationPanel.vue'
import { DETAIL_OPERATIONS, PLAYER_PAGE_OPERATIONS } from '../../operations'
import PlayerAvatar from '../../player/shared/PlayerAvatar.vue'

type Filter = 'all' | 'online' | 'activeToday' | 'banned'

/** 勾选的玩家（翻页后仍保留）。 */
interface PickedPlayer {
  pid: string
  name: string
  avatarIdolCardId: string
}

const { t } = useI18n()
const route = useRoute()
const ctx = useGmContext()

// ---------- 玩家列表 ----------
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

// ---------- 操作 ----------
const HIDDEN = [...DETAIL_OPERATIONS, ...PLAYER_PAGE_OPERATIONS]
const operations = computed(() => ctx.operations.filter((op) => op.target !== 'server' && !HIDDEN.includes(op.id)))
const opOptions = computed(() => operations.value.map((op) => ({ label: op.title, value: op.id, description: op.group })))
const opId = ref<string | undefined>(undefined)
const operation = computed(() => operations.value.find((op) => op.id === opId.value))
const targets = computed(() => selected.value.map((p) => ({ pid: p.pid, name: p.name, avatarIdolCardId: p.avatarIdolCardId })))

// 换操作或换玩家时清空表单与结果
const resetKey = ref(0)
watch([opId, () => selected.value.map((p) => p.pid).join(',')], () => resetKey.value++)

// 执行后刷新当前页（状态、等级可能变了）
const afterRun = () => void load()
</script>

<style scoped>
.gro {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(0, 1fr);
  gap: 16px;
  align-items: start;
  min-width: 0;
}
.gro__card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 16px 18px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.gro__op {
  position: sticky;
  top: 12px;
}
.gro__head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.gro__h {
  margin: 0;
  font-size: 15px;
  font-weight: 800;
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
.gro__warn {
  margin: 0;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-warnSoft);
  font-size: 13px;
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
@media (width < 1100px) {
  .gro {
    grid-template-columns: minmax(0, 1fr);
  }
  .gro__op {
    position: static;
  }
}
</style>
