<!-- 授权的应用：用「用户通行证登录」登录过的游戏与应用；撤销后下次需要重新确认。 -->
<template>
  <section class="pp-card pp-section">
    <h1 class="pp-section__title">{{ t('account.apps.title') }}</h1>
    <p class="pp-section__desc">{{ t('account.apps.desc') }}</p>
    <p v-if="error" class="pp-error">{{ error }}</p>
    <PEmpty v-else-if="loaded && !items.length" compact :icon="KeyIcon" :title="t('account.apps.empty')" :description="t('account.apps.emptyDesc')" />
    <div v-for="a in items" :key="a.id" class="pp-frow pp-frow--icon">
      <GameMark v-if="a.server" :game="a.server.game" :name="a.server.gameName || a.name" :size="44" />
      <span v-else class="ico pp-sticker">{{ a.name.slice(0, 1).toUpperCase() }}</span>
      <span class="body">
        <span class="name">{{ a.name }}</span>
        <span class="pp-hint">{{ t('account.apps.reads', { scopes: scopes(a.scopes) }) }} · {{ t('account.apps.lastUsed', { at: fmtTime(a.lastUsedAt) }) }}</span>
      </span>
      <div class="pp-frow__actions">
        <PButton size="sm" :loading="busyId === a.id" @click="revoke(a.id)">{{ t('account.apps.revoke') }}</PButton>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onActivated, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { KeyIcon } from '@heroicons/vue/24/outline'
import GameMark from '@passport/components/GameMark.vue'
import PButton from '@passport/components/PButton.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import * as api from '@passport/api/passport'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { errorText, fmtTime } from '@passport/utils/format'
import type { PassportApp } from '@/types/v1/passport'

const { t, te } = useI18n()
const session = useSessionStore()
const ui = useUiStore()

const items = ref<PassportApp[]>([])
const loaded = ref(false)
const error = ref('')
const busyId = ref('')

const scopes = (list: string[]) =>
  (list || [])
    .filter((s) => s !== 'openid')
    .map((s) => (te(`account.apps.scopes.${s}`) ? t(`account.apps.scopes.${s}`) : s))
    .join(t('home.games.sep')) || t('account.apps.scopes.openid')

const load = async () => {
  try {
    items.value = (await api.listApps()).items || []
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

const revoke = async (id: string) => {
  busyId.value = id
  try {
    await api.revokeApp(id)
    items.value = items.value.filter((a) => a.id !== id)
    ui.toast(t('account.apps.revoked'), 'ok')
    void session.loadMe().catch(() => undefined)
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    busyId.value = ''
  }
}
</script>

<style scoped>
.ico {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  background: var(--m-a2Soft);
  color: var(--m-ink);
  font-size: 18px;
}
.body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.name {
  font-size: 14px;
  font-weight: 800;
}
</style>
