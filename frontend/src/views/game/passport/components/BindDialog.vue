<!-- 替玩家绑定：选服务器（搜索 + 最近查看）→ 在游戏服务器上搜玩家（MGP /players，最多 20 条）→ 绑定。
     已经绑定到别的通行证的玩家不能选。 -->
<template>
  <FormDialog
    :model-value="modelValue"
    :title="t('passportAdmin.bind.title', { name })"
    :width="560"
    :loading="saving"
    :confirm-text="t('passportAdmin.bind.confirm')"
    @update:model-value="emit('update:modelValue', $event)"
    @confirm="submit"
  >
    <div class="pbind">
      <div class="pbind__field">
        <span class="pbind__label">{{ t('passportAdmin.bind.server') }}</span>
        <GameServerPicker v-model="serverId" :selected="server" :search="searchServers" @select="onServer" />
      </div>

      <div class="pbind__field">
        <span class="pbind__label">{{ t('passportAdmin.bind.player') }}</span>
        <p v-if="!serverId" class="pbind__hint">{{ t('passportAdmin.bind.pickServer') }}</p>
        <template v-else>
          <label class="pbind__search">
            <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" class="pbind__icon" />
            <input v-model="q" type="search" :placeholder="t('passportAdmin.bind.search')" :aria-label="t('passportAdmin.bind.search')" />
          </label>
          <p v-if="loading && !players.length" class="pbind__hint">{{ t('passportAdmin.bind.searching') }}</p>
          <p v-else-if="error" class="pbind__error">{{ error }}</p>
          <p v-else-if="!players.length" class="pbind__hint">{{ t('passportAdmin.bind.noPlayers') }}</p>
          <div v-else class="pbind__list" role="listbox" :aria-label="t('passportAdmin.bind.player')">
            <button
              v-for="p in players"
              :key="p.pid"
              type="button"
              role="option"
              class="pbind__row"
              :class="{ 'is-on': p.pid === pid }"
              :aria-selected="p.pid === pid"
              :disabled="!!p.linkedUserId"
              @click="pid = p.pid"
            >
              <span class="pbind__name">{{ p.name || p.account || t('gm.player.noName') }}</span>
              <span class="pbind__sub">{{ playerLine(p) }}</span>
              <span v-if="p.linkedUserId" class="pbind__taken">{{ linkedLabel(p) }}</span>
            </button>
          </div>
          <p class="pbind__hint">{{ t('passportAdmin.bind.playersHint') }}</p>
        </template>
      </div>

      <p class="pbind__note">{{ t('passportAdmin.bind.hint') }}</p>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import GameServerPicker from '@/views/game/components/GameServerPicker.vue'
import { adminBindPassportLink, searchPassportGamePlayers, searchPassportServers } from '@/api/passportAdmin'
import type { GameServerBrief } from '@/types/v1/game_common'
import type { GetPassportUserResponse, PassportGamePlayer } from '@/types/v1/passport_admin'
import { feedback } from '@/utils/feedback'
import { getRequestErrorMessage } from '@/utils/request'
import { notifyError } from '../shared'

const props = defineProps<{ modelValue: boolean; userId: string; name: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; bound: [detail: GetPassportUserResponse] }>()

const { t } = useI18n()
const menuStore = useMenuStore()

const serverId = ref<string>()
const server = ref<GameServerBrief>()
const q = ref('')
const players = ref<PassportGamePlayer[]>([])
const pid = ref('')
const loading = ref(false)
const error = ref('')
const saving = ref(false)
let seq = 0

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    serverId.value = undefined
    server.value = undefined
    q.value = ''
    players.value = []
    pid.value = ''
    error.value = ''
  },
)

const searchServers = async (keyword: string) => (await searchPassportServers({ q: keyword, limit: 20 })).data

const loadPlayers = async () => {
  if (!serverId.value) return
  const id = ++seq
  loading.value = true
  error.value = ''
  try {
    const res = await searchPassportGamePlayers(serverId.value, q.value.trim())
    if (id !== seq) return
    players.value = res.data.items || []
  } catch (e) {
    if (id !== seq) return
    players.value = []
    error.value = getRequestErrorMessage(e)
  } finally {
    if (id === seq) loading.value = false
  }
}
const debouncedLoad = useDebounceFn(loadPlayers, 300)
watch(q, () => void debouncedLoad())

const onServer = (s: GameServerBrief | undefined) => {
  server.value = s
  pid.value = ''
  players.value = []
  void loadPlayers()
}

const playerLine = (p: PassportGamePlayer) =>
  [p.account, p.displayId ? `ID ${p.displayId}` : '', p.level > 0 ? t('passportAdmin.detail.level', { n: p.level }) : '']
    .filter(Boolean)
    .join(' · ')

const linkedLabel = (p: PassportGamePlayer) =>
  p.linkedUserId === props.userId
    ? t('passportAdmin.bind.linkedSelf')
    : t('passportAdmin.bind.linkedTo', { name: p.linkedUsername || t('game.unnamed') })

const submit = async () => {
  if (!serverId.value || !pid.value) {
    feedback.warning(t('passportAdmin.bind.pickPlayer'))
    return
  }
  saving.value = true
  try {
    const res = await adminBindPassportLink({ userId: props.userId, serverId: serverId.value, pid: pid.value })
    feedback.success(t('passportAdmin.bind.success'))
    emit('bound', res.data)
    emit('update:modelValue', false)
  } catch (e) {
    notifyError(e)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.pbind {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.pbind__field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.pbind__label {
  font-size: 13px;
  font-weight: 700;
}
.pbind__hint,
.pbind__note {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.pbind__note {
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  line-height: 1.6;
}
.pbind__error {
  margin: 0;
  font-size: 13px;
  color: var(--m-bad);
}
.pbind__search {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 42px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.pbind__search:focus-within {
  border-color: var(--m-p);
}
.pbind__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
}
.pbind__icon {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.pbind__list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 320px;
  overflow-y: auto;
}
.pbind__row {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 1px;
  width: 100%;
  min-height: 44px;
  padding: 8px 12px;
  border: 1.5px solid transparent;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.pbind__row:hover:not(:disabled) {
  background: var(--el-fill-color-light);
}
.pbind__row.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
}
.pbind__row:disabled {
  cursor: default;
  opacity: 0.6;
}
.pbind__name {
  font-weight: 800;
}
.pbind__sub,
.pbind__taken {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.pbind__taken {
  color: var(--m-warn);
  font-weight: 700;
}
</style>
