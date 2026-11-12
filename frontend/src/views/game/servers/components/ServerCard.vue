<!-- 游戏服务器卡片：封面（游戏素材）+ 状态、游戏徽章与名称、接入方式 / 位置 / 标签、信息格（在线、版本、资源、MGP）、
     底部能力徽章 + 小图标操作（GM 运营 / 控制台 / 设置 / 启停 / ⋯）。 -->
<template>
  <article class="gs-card" :data-game="server.brief?.game || undefined">
    <RouterLink :to="detailTo('overview')" class="gs-card__cover" :aria-label="name">
      <GameImage v-if="coverUrl" :src="coverUrl" :alt="name" class="gs-card__img" />
      <span v-else class="gs-card__band" aria-hidden="true">{{ gameName }}</span>
      <StatusPill class="gs-card__pill" :variant="stateVariant(state)" :label="t(`gameServers.state.${state}`)" />
    </RouterLink>
    <div class="gs-card__body">
      <div class="gs-card__head">
        <span class="gs-card__logo" aria-hidden="true">{{ initial }}</span>
        <span class="gs-card__title">
          <RouterLink :to="detailTo('overview')" class="gs-card__name" :title="name">{{ name }}</RouterLink>
          <span class="gs-card__game">{{ gameName }}</span>
        </span>
      </div>
      <div class="gs-card__tags">
        <span class="gs-card__tag" :class="hosted ? 'is-hosted' : 'is-remote'">{{ t(`gameServers.mode.${mode}`) }}</span>
        <span class="gs-card__tag m-mono" :title="where">{{ where }}</span>
        <span v-for="tag in server.brief?.tags || []" :key="tag" class="gs-card__tag">{{ tag }}</span>
      </div>
      <dl class="gs-card__kv">
        <div>
          <dt>{{ t('gameServers.list.kvOnline') }}</dt>
          <dd class="m-num">{{ onlineText }}</dd>
        </div>
        <div>
          <dt>{{ t('gameServers.list.kvVersion') }}</dt>
          <dd class="m-num" :title="versionText">{{ versionText }}</dd>
        </div>
        <div>
          <dt>{{ t('gameServers.list.kvResource') }}</dt>
          <dd class="m-num" :title="resource">{{ resource }}</dd>
        </div>
        <div>
          <dt>{{ t('gameServers.list.kvMgp') }}</dt>
          <dd :class="mgpClass">{{ mgpText }}</dd>
        </div>
      </dl>
      <footer class="gs-card__foot">
        <span class="gs-card__caps">
          <span v-for="c in caps" :key="c" class="gs-card__cap">{{ t(`gameServers.caps.${c}`) }}</span>
        </span>
        <RouterLink
          v-if="canGm"
          :to="`/game/gm/${encodeURIComponent(server.brief?.id || '')}`"
          class="gs-card__link"
          :aria-label="`${name} ${t('gameServers.actions.gm')}`"
          :title="t('gameServers.actions.gm')"
        >
          <AppIconButton icon="HOutline:SparklesIcon" tabindex="-1" :label="t('gameServers.actions.gm')" />
        </RouterLink>
        <RouterLink
          v-if="hosted"
          :to="detailTo('console')"
          class="gs-card__link"
          :aria-label="`${name} ${t('gameServers.actions.console')}`"
          :title="t('gameServers.actions.console')"
        >
          <AppIconButton icon="HOutline:CommandLineIcon" tabindex="-1" :label="t('gameServers.actions.console')" />
        </RouterLink>
        <RouterLink
          :to="detailTo(hosted ? 'settings' : 'connection')"
          class="gs-card__link"
          :aria-label="`${name} ${t('gameServers.actions.settings')}`"
          :title="t('gameServers.actions.settings')"
        >
          <AppIconButton icon="HOutline:AdjustmentsHorizontalIcon" tabindex="-1" :label="t('gameServers.actions.settings')" />
        </RouterLink>
        <AppIconButton
          v-if="hosted"
          :icon="running ? 'HOutline:StopIcon' : 'HOutline:PlayIcon'"
          :label="`${running ? t('gameServers.actions.stop') : t('gameServers.actions.start')} ${name}`"
          :class="running ? 'is-stop' : 'is-start'"
          :disabled="busy"
          @click="emit('control', running ? 'stop' : 'start')"
        />
        <ActionMenu :items="menu" :label="`${name} ${t('gameServers.actions.more')}`" @select="(k) => emit('menu', k)" />
      </footer>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { gameAsset } from '@/games'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { GameServerInfo } from '@/types/v1/game_server'
import { resourceText, stateVariant, statusOf } from '../shared'

const props = defineProps<{ server: GameServerInfo; canGm: boolean; busy?: boolean }>()
const emit = defineEmits<{ control: [action: 'start' | 'stop']; menu: [key: string] }>()
const { t, te } = useI18n()

const brief = computed(() => props.server.brief)
const name = computed(() => brief.value?.name || '')
const mode = computed(() => brief.value?.mode || 'remote')
const hosted = computed(() => mode.value === 'hosted')
const state = computed(() => brief.value?.state || 'unknown')
const running = computed(() => ['running', 'maintenance', 'starting', 'restarting'].includes(state.value))
const gameName = computed(() => {
  const key = `game.adapters.${brief.value?.game}`
  return brief.value?.gameName || (te(key) ? t(key) : t('game.unnamed'))
})
const initial = computed(() => (gameName.value || name.value || '?').trim().charAt(0).toUpperCase())
const coverUrl = computed(() => (brief.value?.cover ? gameAsset(brief.value.game, brief.value.cover) : ''))
const st = computed(() => statusOf(props.server))

const where = computed(() => {
  if (hosted.value) return props.server.hostName ? t('gameServers.common.localHost', { host: props.server.hostName }) : t('gameServers.common.local')
  try {
    return props.server.mgpEndpoint ? new URL(props.server.mgpEndpoint).host : t('gameServers.list.mgpNone')
  } catch {
    return props.server.mgpEndpoint
  }
})

const onlineText = computed(() => {
  const b = brief.value
  if (!b || b.online < 0) return '—'
  return `${b.online} / ${b.registered >= 0 ? b.registered : '—'}`
})
const versionText = computed(() => props.server.serverVersion || st.value?.version || '—')
const resource = computed(() => resourceText(st.value))
const mgpText = computed(() => {
  const b = brief.value
  if (!b?.mgpConfigured) return t('gameServers.list.mgpNone')
  if (!props.server.probeAt) return t('gameServers.list.mgpUnknown')
  return b.mgpOk ? t('gameServers.list.mgpOk', { ms: b.mgpLatencyMs }) : t('gameServers.list.mgpDown')
})
const mgpClass = computed(() => {
  const b = brief.value
  if (!b?.mgpConfigured || !props.server.probeAt) return 'is-muted'
  return b.mgpOk ? 'is-ok' : 'is-bad'
})

const SHOWN_CAPS = ['gm', 'passport', 'settings', 'update', 'backup']
const caps = computed(() => (props.server.capabilities || []).filter((c) => SHOWN_CAPS.includes(c)).slice(0, 4))

const detailTo = (tab: string) => ({ path: `/game/servers/${encodeURIComponent(brief.value?.id || '')}`, query: { tab } })

const menu = computed<ActionMenuItem[]>(() => [
  { key: 'edit', label: t('gameServers.actions.edit'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'restart', label: t('gameServers.actions.restart'), icon: 'HOutline:ArrowPathIcon', hidden: !hosted.value || !running.value },
  { key: 'rotate', label: t('gameServers.actions.rotateKey'), icon: 'HOutline:KeyIcon', hidden: !hosted.value },
  { key: 'refresh', label: t('gameServers.actions.refresh'), icon: 'HOutline:ArrowPathRoundedSquareIcon', hidden: !brief.value?.mgpConfigured },
  { key: 'switch', label: t('gameServers.actions.switchMode'), icon: 'HOutline:ArrowsPointingInIcon' },
  { key: 'delete', label: t('gameServers.actions.delete'), icon: 'HOutline:TrashIcon', danger: true },
])
</script>

<style scoped lang="scss">
.gs-card {
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
  border-radius: var(--app-radius-lg);
  border: var(--cmp-card-border-width, 1px) solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.gs-card__cover {
  position: relative;
  display: block;
  height: 120px;
  background: var(--m-gSoft, var(--m-s2));
  overflow: hidden;
}
.gs-card__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: 50% 30%;
}
.gs-card__band {
  position: absolute;
  inset: auto 0 18px 20px;
  font-family: var(--app-font-display);
  font-size: 1.375rem;
  font-weight: 800;
  color: var(--m-gInk, var(--m-ink2));
  opacity: 0.35;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.gs-card__pill {
  position: absolute;
  top: 12px;
  right: 12px;
  background: var(--el-bg-color);
}
.gs-card__body {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 0 18px 16px;
}
.gs-card__head {
  display: flex;
  align-items: flex-end;
  gap: 14px;
  margin-top: -26px;
  position: relative;
}
.gs-card__logo {
  flex: none;
  width: 56px;
  height: 56px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 18px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--m-gInk, var(--m-pInk));
  font-family: var(--app-font-display);
  font-size: 1.375rem;
  font-weight: 800;
  transform: rotate(-4deg);
  box-shadow: 3px 3px 0 var(--m-g, var(--m-p));
}
.gs-card__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-bottom: 2px;
}
.gs-card__name {
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gs-card__name:hover {
  color: var(--m-pInk);
}
.gs-card__game {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gs-card__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.gs-card__tag {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gs-card__tag.is-hosted {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.gs-card__tag.is-remote {
  background: var(--m-a2Soft);
  color: var(--m-ink);
}
.gs-card__kv {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 14px;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.gs-card__kv div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gs-card__kv dt {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.gs-card__kv dd {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gs-card__kv dd.is-ok {
  color: var(--m-ok);
}
.gs-card__kv dd.is-bad {
  color: var(--m-bad);
}
.gs-card__kv dd.is-muted {
  color: var(--el-text-color-secondary);
}
.gs-card__foot {
  display: flex;
  align-items: center;
  gap: 2px;
  padding-top: 10px;
  border-top: 1px solid var(--el-border-color-light);
}
.gs-card__caps {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.gs-card__cap {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
  font-size: 0.6875rem;
  font-weight: 700;
}
.gs-card__link {
  display: inline-flex;
  border-radius: var(--app-radius-sm);
}
.gs-card__link :deep(.app-icon-btn) {
  pointer-events: none;
}
.gs-card__foot :deep(.app-icon-btn) {
  width: 40px !important;
  height: 40px !important;
}
.gs-card__foot .is-stop {
  color: var(--m-bad);
}
.gs-card__foot .is-start {
  color: var(--m-ok);
}
@media (width <= 768px) {
  .gs-card__foot :deep(.app-icon-btn) {
    width: 44px !important;
    height: 44px !important;
  }
}
</style>
