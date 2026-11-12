<!-- 一台游戏服务器的单行摘要：名称、游戏、在线人数与 MGP 状态（选择器与列表共用）。 -->
<template>
  <span class="gsl">
    <span class="gsl__dot" :class="`is-${server.state}`" aria-hidden="true" />
    <span class="gsl__main">
      <span class="gsl__name">{{ server.name }}</span>
      <span class="gsl__sub">
        {{ server.gameName || (te(`game.adapters.${server.game}`) ? t(`game.adapters.${server.game}`) : t('game.unnamed')) }}
        <template v-if="server.online >= 0"> · {{ t('game.server.online', { n: server.online }) }}</template>
        <template v-else-if="!server.mgpConfigured"> · {{ t('game.server.noMgp') }}</template>
        <template v-else-if="!server.mgpOk"> · {{ t('game.server.unreachable') }}</template>
      </span>
    </span>
  </span>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { GameServerBrief } from '@/types/v1/game_common'

defineProps<{ server: GameServerBrief }>()
const { t, te } = useI18n()
</script>

<style scoped>
.gsl {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.gsl__dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--el-text-color-secondary);
}
.gsl__dot.is-running {
  background: var(--m-ok);
}
.gsl__dot.is-maintenance {
  background: var(--m-warn);
}
.gsl__dot.is-unreachable,
.gsl__dot.is-stopped {
  background: var(--m-bad);
}
.gsl__main {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.gsl__name {
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gsl__sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
