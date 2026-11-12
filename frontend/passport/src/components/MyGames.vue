<!-- 我的游戏：按游戏分组，每台服务器一行；礼物箱提醒按需（逐个绑定）向游戏服务端取摘要。 -->
<template>
  <div class="mg" :class="{ 'mg--compact': compact }">
    <article v-for="g in groups" :key="g.game" class="mg__card pp-card" :data-game="g.game">
      <div class="mg__band" aria-hidden="true">
        <span class="mg__band-dot" />
        <span class="mg__band-ring" />
      </div>
      <div class="mg__body">
        <div class="mg__head">
          <span class="mg__mark"><GameMark :game="g.game" :name="g.name" :size="compact ? 40 : 46" /></span>
          <span class="mg__title">
            <span class="mg__name">{{ g.name }}</span>
            <span class="pp-hint">{{ t('home.games.accounts', { n: g.links.length }) }}</span>
          </span>
        </div>
        <div class="mg__rows">
          <RouterLink v-for="l in g.links" :key="l.id" :to="`/games/${l.server?.id}`" class="mg__row">
            <LinkAvatar :game="l.game" :asset="l.avatarAsset" :name="l.nickname" :size="40" />
            <span class="mg__who">
              <span class="mg__server">
                {{ l.server?.name }}
                <span v-if="l.isPrimary" class="pp-tag pp-tag--g">{{ t('home.games.primary') }}</span>
              </span>
              <span class="pp-hint mg__line">{{ who(l) }}</span>
            </span>
            <span v-if="badge(l)" class="pp-tag pp-tag--g">{{ badge(l) }}</span>
            <PPill v-else-if="stateTone(l)" :tone="stateTone(l)">{{ t(`home.games.state.${l.server?.state}`) }}</PPill>
            <ChevronRightIcon class="mg__chev" aria-hidden="true" />
          </RouterLink>
        </div>
        <div v-if="!compact && hints(g).length" class="mg__hints">
          <span v-for="h in hints(g)" :key="h" class="pp-tag pp-tag--g">{{ h }}</span>
        </div>
      </div>
    </article>
    <RouterLink v-if="showAdd" to="/bind" class="mg__add">
      <span class="mg__add-ico pp-sticker"><PlusIcon aria-hidden="true" /></span>
      <span class="mg__add-title">{{ t('home.games.bind') }}</span>
      <span v-if="bindable" class="pp-hint mg__add-hint">{{ t('home.games.bindable', { list: bindable }) }}</span>
    </RouterLink>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronRightIcon, PlusIcon } from '@heroicons/vue/24/outline'
import GameMark from '@passport/components/GameMark.vue'
import LinkAvatar from '@passport/components/LinkAvatar.vue'
import PPill from '@passport/components/PPill.vue'
import * as api from '@passport/api/passport'
import * as gk from '@passport/api/gakumas'
import type { PassportLink } from '@/types/v1/passport'

const props = withDefaults(defineProps<{ links: PassportLink[]; compact?: boolean; showAdd?: boolean; bindable?: string }>(), {
  showAdd: true,
  bindable: '',
})

const { t } = useI18n()

interface Brief {
  unreceived: number
  expiring: number
}
const briefs = reactive<Record<string, Brief>>({})

const groups = computed(() => {
  const map = new Map<string, { game: string; name: string; links: PassportLink[] }>()
  for (const l of props.links) {
    const key = l.game || l.server?.game || ''
    let g = map.get(key)
    if (!g) {
      g = { game: key, name: l.server?.gameName || key, links: [] }
      map.set(key, g)
    }
    g.links.push(l)
  }
  return [...map.values()]
})

const spaced = (id: string) => id.replace(/(\d{3})(?=\d)/g, '$1 ')

const who = (l: PassportLink) => {
  const parts = [l.nickname || l.account]
  if (l.level) parts.push(t('home.games.level', { n: l.level }))
  if (l.displayId) parts.push(`ID ${spaced(l.displayId)}`)
  return parts.filter(Boolean).join(' · ')
}

const badge = (l: PassportLink) => {
  const b = briefs[l.id]
  return b && b.unreceived > 0 ? t('home.games.gifts', { n: b.unreceived }) : ''
}

type Tone = 'warn' | 'bad' | 'off'
const stateTone = (l: PassportLink): Tone | undefined => {
  switch (l.server?.state) {
    case 'maintenance':
      return 'warn'
    case 'unreachable':
    case 'stopped':
      return 'bad'
    default:
      return undefined
  }
}

const hints = (g: { links: PassportLink[] }) => {
  const out: string[] = []
  for (const l of g.links) {
    const b = briefs[l.id]
    if (b && b.expiring > 0) out.push(t('home.games.expiring', { server: l.server?.name || '', n: b.expiring }))
  }
  return out
}

// 首页卡片的礼物箱摘要：每个绑定一次请求，最多 8 个，逐个进行。
let loadedFor = ''
watch(
  () => props.links.map((l) => l.id).join(','),
  async (key) => {
    if (!key || key === loadedFor) return
    loadedFor = key
    for (const l of props.links.slice(0, 8)) {
      if (!l.server?.id || briefs[l.id]) continue
      try {
        // 刷新绑定快照；礼物提醒来自游戏模块的玩家概览
        await api.getGameBrief(l.server.id)
        if (l.server.game !== 'gakumas') continue
        const gifts = (await gk.getMyPlayer(l.server.id)).overview?.gifts
        briefs[l.id] = { unreceived: gifts?.unreceived ?? 0, expiring: gifts?.expiring ?? 0 }
      } catch {
        // 服务器暂时连不上：不显示提醒
      }
    }
  },
  { immediate: true },
)
</script>

<style scoped>
.mg {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 18px;
}
.mg__card {
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.mg__band {
  position: relative;
  height: 96px;
  background: var(--m-gSoft, var(--m-pSoft));
  overflow: hidden;
}
.mg__band-dot {
  position: absolute;
  right: 28px;
  top: 18px;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: var(--m-g, var(--m-p));
  opacity: 0.35;
}
.mg__band-ring {
  position: absolute;
  right: 92px;
  top: -20px;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  border: 10px solid var(--m-g2Soft, var(--m-a2Soft));
}
.mg__body {
  padding: 0 18px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.mg__head {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  margin-top: -24px;
  position: relative;
}
.mg__mark {
  display: inline-flex;
  padding: 6px;
  border-radius: 18px;
  background: var(--m-s);
  border: 1px solid var(--m-line);
  transform: rotate(-4deg);
  box-shadow: 3px 3px 0 var(--m-g, var(--m-p));
}
.mg__title {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.mg__name {
  font-size: 17px;
  font-weight: 800;
}
.mg__rows > * + * {
  border-top: 1px solid var(--m-line);
}
.mg__rows {
  border-top: 1px solid var(--m-line);
}
.mg__row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 60px;
  padding: 0 4px;
  color: var(--m-ink);
  text-decoration: none;
}
.mg__row:hover {
  background: var(--m-s2);
}
.mg__who {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.mg__server {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 15px;
  font-weight: 800;
}
.mg__line {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mg__chev {
  width: 18px;
  height: 18px;
  color: var(--m-ink2);
  flex: none;
}
.mg__hints {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.mg__add {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  min-height: 220px;
  padding: 24px;
  border-radius: var(--app-radius-xl, 24px);
  border: 2px dashed var(--m-line);
  color: var(--m-ink);
  text-decoration: none;
  text-align: center;
}
.mg__add:hover {
  border-color: var(--m-p);
}
.mg__add-ico {
  width: 56px;
  height: 56px;
  border-radius: 18px;
  background: var(--m-a2Soft);
  transform: rotate(-4deg);
}
.mg__add-ico svg {
  width: 26px;
  height: 26px;
}
.mg__add-title {
  font-size: 17px;
  font-weight: 800;
}
.mg__add-hint {
  max-width: 260px;
}
.mg--compact {
  grid-template-columns: minmax(0, 1fr);
  gap: 12px;
}
.mg--compact .mg__band {
  height: 72px;
}
.mg--compact .mg__add {
  min-height: 0;
  flex-direction: row;
  justify-content: flex-start;
  padding: 14px 16px;
}
.mg--compact .mg__add-ico {
  width: 44px;
  height: 44px;
  border-radius: 14px;
}
@media (max-width: 768px) {
  .mg {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
