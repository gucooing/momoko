<!-- 游戏账号：每台服务器一个；进入游戏空间、设为常用、解绑（解绑后冷却期内不能绑别的账号）。 -->
<template>
  <section class="pp-card pp-section">
    <div class="pp-section__head">
      <h1 class="pp-section__title">{{ t('account.games.title') }}</h1>
      <PButton size="sm" variant="primary" :icon="PlusIcon" to="/bind">{{ t('home.games.bind') }}</PButton>
    </div>
    <p class="pp-section__desc">{{ t('account.games.desc', { days: session.config?.unbindCooldownDays ?? 7 }) }}</p>
    <p v-if="error" class="pp-error">{{ error }}</p>
    <PEmpty v-else-if="loaded && !items.length" compact :icon="LinkIcon" :title="t('home.games.emptyTitle')" />
    <div v-for="l in items" :key="l.id" class="pp-frow pp-frow--icon">
      <GameMark :game="l.game" :name="l.server?.gameName || l.game" :size="44" />
      <span class="body">
        <span class="name">
          {{ serverLabel(l.server) }}
          <span v-if="l.isPrimary" class="pp-tag pp-tag--p">{{ t('home.games.primary') }}</span>
        </span>
        <span class="pp-hint">{{ line(l) }}</span>
      </span>
      <div class="pp-frow__actions">
        <PButton size="sm" variant="soft" :to="`/games/${l.server?.id}`">{{ t('common.enter') }}</PButton>
        <PButton v-if="!l.isPrimary && sameGame(l) > 1" size="sm" variant="ghost" :loading="busyId === `p-${l.id}`" @click="primary(l)">
          {{ t('account.games.setPrimary') }}
        </PButton>
        <PButton size="sm" @click="askUnbind(l)">{{ t('account.games.unbind') }}</PButton>
      </div>
    </div>

    <PDialog v-model="confirmOpen" :title="t('account.games.unbindTitle')">
      <p v-if="target" class="pp-note">
        {{ t('account.games.unbindWarn', { server: serverLabel(target.server), nickname: target.nickname, days: session.config?.unbindCooldownDays ?? 7 }) }}
      </p>
      <template #footer>
        <PButton @click="confirmOpen = false">{{ t('common.cancel') }}</PButton>
        <PButton variant="danger" :loading="busyId === `u-${target?.id}`" @click="unbind">{{ t('account.games.unbind') }}</PButton>
      </template>
    </PDialog>
  </section>
</template>

<script setup lang="ts">
import { onActivated, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { LinkIcon, PlusIcon } from '@heroicons/vue/24/outline'
import GameMark from '@passport/components/GameMark.vue'
import PButton from '@passport/components/PButton.vue'
import PDialog from '@passport/components/PDialog.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import * as api from '@passport/api/passport'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { errorText, fmtDate } from '@passport/utils/format'
import { serverLabel } from '@passport/utils/messages'
import type { PassportLink } from '@/types/v1/passport'

const { t, te } = useI18n()
const session = useSessionStore()
const ui = useUiStore()

const items = ref<PassportLink[]>([])
const loaded = ref(false)
const error = ref('')
const busyId = ref('')
const confirmOpen = ref(false)
const target = ref<PassportLink | null>(null)

const sameGame = (l: PassportLink) => items.value.filter((x) => x.game === l.game).length

const line = (l: PassportLink) => {
  const method = te(`common.bindMethod.${l.method}`) ? t(`common.bindMethod.${l.method}`) : l.method
  const who = [l.nickname || l.account, l.displayId ? `ID ${l.displayId}` : ''].filter(Boolean).join(' · ')
  return `${who} · ${method} · ${fmtDate(l.linkedAt)}`
}

const load = async () => {
  try {
    items.value = (await api.listLinks()).items || []
    error.value = ''
  } catch (e) {
    error.value = errorText(e)
  } finally {
    loaded.value = true
  }
}
onMounted(load)
onActivated(() => {
  if (loaded.value) void load()
})

const primary = async (l: PassportLink) => {
  busyId.value = `p-${l.id}`
  try {
    await api.setPrimaryLink(l.id)
    await load()
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    busyId.value = ''
  }
}

const askUnbind = (l: PassportLink) => {
  target.value = l
  confirmOpen.value = true
}

const unbind = async () => {
  const l = target.value
  if (!l) return
  busyId.value = `u-${l.id}`
  try {
    await api.unbindLink(l.id)
    confirmOpen.value = false
    ui.toast(t('account.games.unbound'), 'ok')
    await load()
    void session.loadMe().catch(() => undefined)
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    busyId.value = ''
  }
}
</script>

<style scoped>
.body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.name {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 800;
}
</style>
