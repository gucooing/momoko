<!-- 指定玩家：已选玩家的胶囊 + 搜索（昵称、User ID、账号或通行证用户名，MGP 搜索每次最多 8 条）。
     一次粘贴多个（逗号、空格或换行分隔）时逐个查找，只收下能唯一对上的玩家。 -->
<template>
  <div class="ppw">
    <div class="pp" :class="{ 'is-disabled': disabled }">
      <span v-for="p in modelValue" :key="p.pid" class="pp__chip">
        <span class="pp__avatar" aria-hidden="true">{{ initial(p.name || t('gm.player.noName')) }}</span>
        <span class="pp__name">{{ p.name || t('gm.player.noName') }}</span>
        <span v-if="p.displayId" class="pp__id m-num">{{ p.displayId }}</span>
        <button v-if="!disabled" type="button" class="pp__remove" :aria-label="t('gmOps.mail.removePlayer', { name: p.name || t('gm.player.noName') })" @click="remove(p.pid)">
          <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
        </button>
      </span>
      <div v-if="!disabled" class="pp__field">
        <input
          v-model="query"
          type="text"
          class="pp__input"
          :placeholder="t('gmOps.mail.playerSearch')"
          :aria-label="t('gmOps.mail.playerSearch')"
          @paste="onPaste"
          @keydown.enter.prevent="pickFirst"
          @focus="open = true"
          @blur="closeSoon"
        />
        <div v-if="open && (results.length || searching || (query.trim() && !searching))" class="pp__panel" role="listbox">
          <div v-if="searching" class="pp__hint">{{ t('gmOps.common.loading') }}</div>
          <template v-else>
            <button
              v-for="r in shownResults"
              :key="r.pid"
              type="button"
              role="option"
              class="pp__option"
              :aria-selected="false"
              @mousedown.prevent="add(r)"
            >
              <span class="pp__avatar" aria-hidden="true">{{ initial(r.name || t('gm.player.noName')) }}</span>
              <span class="pp__option-text">
                <span class="pp__option-name">{{ r.name || t('gm.player.noName') }}</span>
                <span class="pp__option-sub m-num">{{ [r.displayId, r.account].filter(Boolean).join(' · ') }}</span>
              </span>
              <StatusPill v-if="isBanned(r)" variant="error" :label="t('gmOps.mail.banned')" />
            </button>
            <div v-if="!shownResults.length" class="pp__hint">{{ t('gmOps.mail.noPlayer') }}</div>
          </template>
        </div>
      </div>
    </div>
    <div v-if="unresolved.length" class="pp__unresolved">{{ t('gmOps.mail.unresolved', { list: unresolved.join('、') }) }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import { listPlayers } from '@/api/gakumas'
import type { Player } from '@/games/gakumas/types'
import type { GameTarget } from '@/types/v1/game_common'
import { useGmContext } from '../../context'

type PlayerHit = Pick<Player, 'pid' | 'displayId' | 'account' | 'name' | 'bannedUntil'>

const props = withDefaults(defineProps<{ modelValue: GameTarget[]; disabled?: boolean }>(), { disabled: false })
const emit = defineEmits<{ 'update:modelValue': [value: GameTarget[]] }>()

const { t } = useI18n()
const ctx = useGmContext()
const menuStore = useMenuStore()

const query = ref('')
const results = ref<PlayerHit[]>([])
// 已经加上的玩家不再出现在搜索结果里
const shownResults = computed(() => results.value.filter((r) => !chosen.value.has(r.pid)))
const searching = ref(false)
const open = ref(false)
const unresolved = ref<string[]>([])
const chosen = computed(() => new Set(props.modelValue.map((p) => p.pid)))

const initial = (s: string) => s.slice(0, 1).toUpperCase()
const isBanned = (p: PlayerHit) => p.bannedUntil > Date.now()

const search = async (q: string, limit = 8): Promise<PlayerHit[]> => {
  return (await listPlayers({ serverId: ctx.serverId.value, q, filter: '', cursor: '', limit })).items
}

let seq = 0
const runSearch = useDebounceFn(async () => {
  const q = query.value.trim()
  const mine = ++seq
  if (!q) {
    results.value = []
    searching.value = false
    return
  }
  try {
    const list = await search(q)
    if (mine === seq) results.value = list
  } catch {
    if (mine === seq) results.value = []
  } finally {
    if (mine === seq) searching.value = false
  }
}, 300)

watch(query, (q) => {
  searching.value = !!q.trim()
  void runSearch()
})

const toTarget = (p: PlayerHit): GameTarget => ({ pid: p.pid, name: p.name || '', displayId: p.displayId || '' })

const add = (p: PlayerHit) => {
  if (chosen.value.has(p.pid)) return
  emit('update:modelValue', [...props.modelValue, toTarget(p)])
  query.value = ''
}

const pickFirst = () => {
  const first = results.value.find((r) => !chosen.value.has(r.pid))
  if (first) add(first)
}

const remove = (pid: string) => emit('update:modelValue', props.modelValue.filter((p) => p.pid !== pid))

let closeTimer: ReturnType<typeof setTimeout> | undefined
const closeSoon = () => {
  clearTimeout(closeTimer)
  closeTimer = setTimeout(() => (open.value = false), 150)
}

// 粘贴多个：逐个查找，昵称、User ID、账号或 pid 完全一致且只有一个时才收下。
const onPaste = async (e: ClipboardEvent) => {
  const text = e.clipboardData?.getData('text') || ''
  const tokens = [...new Set(text.split(/[\s,，、;；]+/).map((s) => s.trim()).filter(Boolean))]
  if (tokens.length < 2) return
  e.preventDefault()
  unresolved.value = []
  const next = [...props.modelValue]
  const have = new Set(next.map((p) => p.pid))
  const missed: string[] = []
  for (const token of tokens.slice(0, 200)) {
    try {
      const hits = (await search(token, 10)).filter((p) =>
        [p.pid, p.displayId, p.account, p.name].some((v) => (v || '').toLowerCase() === token.toLowerCase()),
      )
      if (hits.length === 1 && hits[0]) {
        if (!have.has(hits[0].pid)) {
          have.add(hits[0].pid)
          next.push(toTarget(hits[0]))
        }
      } else {
        missed.push(token)
      }
    } catch {
      missed.push(token)
    }
  }
  unresolved.value = missed
  emit('update:modelValue', next)
}
</script>

<style scoped>
.ppw {
  min-width: 0;
}
.pp {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  padding: 8px;
  border-radius: var(--app-radius);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  min-width: 0;
}
.pp:focus-within {
  border-color: var(--m-p);
}
.pp__chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 34px;
  padding: 0 4px 0 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  font-size: 13px;
  font-weight: 700;
  max-width: 100%;
}
.pp__avatar {
  flex: none;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
  font-size: 12px;
  font-weight: 800;
}
.pp__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.pp__id {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.pp__remove {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  border: 0;
  background: transparent;
  color: var(--el-text-color-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}
.pp__remove:hover {
  background: var(--m-s2);
  color: var(--el-text-color-primary);
}
.pp__remove svg {
  width: 14px;
  height: 14px;
}
.pp__field {
  position: relative;
  flex: 1 1 220px;
  min-width: 0;
}
.pp__input {
  width: 100%;
  min-height: 34px;
  border: 0;
  background: transparent;
  font: inherit;
  font-size: 14px;
  color: var(--el-text-color-primary);
  outline: none;
}
.pp__panel {
  position: absolute;
  left: 0;
  right: 0;
  top: calc(100% + 6px);
  z-index: 30;
  display: flex;
  flex-direction: column;
  padding: 6px;
  border-radius: var(--app-radius);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-brand);
  max-height: 320px;
  overflow-y: auto;
}
.pp__option {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 4px 8px;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.pp__option:hover:not(:disabled) {
  background: var(--el-fill-color-light);
}
.pp__option:disabled {
  cursor: default;
  opacity: 0.6;
}
.pp__option-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.pp__option-name {
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pp__option-sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.pp__hint {
  padding: 10px 8px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.pp__unresolved {
  margin-top: 6px;
  font-size: 12px;
  color: var(--m-warn);
}
</style>
