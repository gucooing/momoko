<!-- 一条游戏账号绑定：服务器、主账号标记、昵称 · 游戏内 User ID · 绑定方式、最近使用；右侧解绑按钮。 -->
<template>
  <div class="plink">
    <GameBadge :game="link.server?.game || ''" :name="link.server?.gameName" :size="38" />
    <span class="plink__main">
      <span class="plink__title">
        <span class="plink__server">{{ link.server?.name || t('game.unnamed') }}</span>
        <span v-if="link.isPrimary" class="plink__tag">{{ t('passportAdmin.detail.primary') }}</span>
      </span>
      <span class="plink__sub">{{ who }}</span>
      <span class="plink__sub">{{ when }}</span>
    </span>
    <AppIconButton
      v-if="unbindable"
      icon="HOutline:LinkSlashIcon"
      :label="t('passportAdmin.detail.unbind', { server: link.server?.name || '' })"
      :box="40"
      :size="18"
      @click="emit('unbind', link)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PassportAdminLink } from '@/types/v1/passport_admin'
import GameBadge from './GameBadge.vue'
import { usePassportTime } from '../shared'

const props = withDefaults(defineProps<{ link: PassportAdminLink; unbindable?: boolean }>(), { unbindable: true })
const emit = defineEmits<{ unbind: [link: PassportAdminLink] }>()

const { t, te } = useI18n()
const { ago, date } = usePassportTime()

const methodLabel = (m: string) => (te(`passportAdmin.method.${m}`) ? t(`passportAdmin.method.${m}`) : m)

const who = computed(() => {
  const parts: string[] = []
  if (props.link.nickname) parts.push(props.link.nickname)
  if (props.link.displayId) parts.push(t('passportAdmin.detail.displayId', { id: props.link.displayId }))
  if (props.link.level > 0) parts.push(t('passportAdmin.detail.level', { n: props.link.level }))
  parts.push(methodLabel(props.link.method))
  return parts.join(' · ')
})

const when = computed(() => {
  const seen = props.link.lastSeenAt
    ? t('passportAdmin.detail.lastSeen', { time: ago(props.link.lastSeenAt) })
    : t('passportAdmin.detail.neverSeen')
  return `${seen} · ${t('passportAdmin.detail.linkedAt', { date: date(props.link.linkedAt) })}`
})
</script>

<style scoped>
.plink {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 8px 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.plink__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.plink__title {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.plink__server {
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.plink__tag {
  flex: none;
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-size: 11px;
  font-weight: 800;
}
.plink__sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
