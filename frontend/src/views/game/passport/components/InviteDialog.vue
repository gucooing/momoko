<!-- 邀请玩家：生成邀请码（可指定邀请码、限定邮箱、附带一台「凭邀请码」的服务器、次数、有效天数、备注）+ 已生成的邀请码。 -->
<template>
  <FormDialog :model-value="modelValue" :title="t('passportAdmin.invite.title')" :width="720" :show-footer="false" @update:model-value="emit('update:modelValue', $event)">
    <div class="pinv">
      <p v-if="registerMode" class="pinv__mode">{{ t('passportAdmin.invite.modeHint', { mode: t(`passportAdmin.settings.modes.${registerMode}`) }) }}</p>

      <form class="pinv__form" @submit.prevent="create">
        <label class="pinv__field">
          <span class="pinv__label">{{ t('passportAdmin.invite.code') }}</span>
          <UInput v-model="form.code" class="m-mono" :placeholder="t('passportAdmin.invite.codePlaceholder')" maxlength="32" />
        </label>
        <label class="pinv__field">
          <span class="pinv__label">{{ t('passportAdmin.invite.email') }}</span>
          <UInput v-model="form.email" type="email" :placeholder="t('passportAdmin.invite.emailPlaceholder')" />
        </label>
        <label class="pinv__field">
          <span class="pinv__label">{{ t('passportAdmin.invite.maxUses') }}</span>
          <UInput v-model.number="form.maxUses" type="number" min="1" max="100000" />
        </label>
        <label class="pinv__field">
          <span class="pinv__label">{{ t('passportAdmin.invite.expiresDays') }}</span>
          <UInput v-model.number="form.expiresDays" type="number" min="0" max="3650" />
          <span class="pinv__hint">{{ t('passportAdmin.invite.expiresHint') }}</span>
        </label>
        <div class="pinv__field pinv__field--wide">
          <span class="pinv__label">{{ t('passportAdmin.invite.server') }}</span>
          <div class="pinv__server">
            <GameServerPicker v-model="form.serverId" :selected="server" :search="searchServers" allow-all :placeholder="t('passportAdmin.invite.serverNone')" @select="server = $event" />
          </div>
          <span class="pinv__hint">{{ t('passportAdmin.invite.serverHint') }}</span>
        </div>
        <label class="pinv__field pinv__field--wide">
          <span class="pinv__label">{{ t('passportAdmin.invite.note') }}</span>
          <UInput v-model="form.note" :placeholder="t('passportAdmin.invite.notePlaceholder')" maxlength="200" />
        </label>
        <div class="pinv__actions">
          <UButton type="submit" icon="i-lucide-plus" :loading="creating">{{ t('passportAdmin.invite.create') }}</UButton>
        </div>
      </form>

      <div class="pinv__created" v-if="justCreated">
        <span class="m-mono pinv__code">{{ justCreated }}</span>
        <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('passportAdmin.oidc.copy')" :box="40" :size="18" @click="copy(justCreated)" />
      </div>

      <div class="pinv__list-head">
        <span class="pinv__list-title">{{ t('passportAdmin.invite.list') }}</span>
        <span class="pinv__hint m-num">{{ total }}</span>
      </div>
      <p v-if="loading && !items.length" class="pinv__hint">{{ t('passportAdmin.common.loading') }}</p>
      <EmptyState v-else-if="!items.length" icon="HOutline:TicketIcon" :title="t('passportAdmin.invite.empty')" />
      <div v-else class="pinv__list">
        <div v-for="inv in items" :key="inv.id" class="pinv__item">
          <span class="pinv__item-main">
            <span class="pinv__item-title">
              <span class="m-mono pinv__code">{{ inv.code }}</span>
              <StatusPill :variant="stateVariant(inv.state)" :label="t(`passportAdmin.invite.states.${inv.state}`)" />
            </span>
            <span class="pinv__item-sub">{{ inviteLine(inv) }}</span>
            <span v-if="inv.note" class="pinv__item-sub">{{ inv.note }}</span>
          </span>
          <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('passportAdmin.oidc.copy')" :box="40" :size="18" @click="copy(inv.code)" />
          <AppIconButton
            v-if="inv.state === 'active'"
            icon="HOutline:NoSymbolIcon"
            :label="t('passportAdmin.invite.revoke')"
            :box="40"
            :size="18"
            @click="revoke(inv)"
          />
        </div>
        <Pagination v-if="total > pageSize" v-model:page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[10, 20, 50]" @change="load" />
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import GameServerPicker from '@/views/game/components/GameServerPicker.vue'
import { createPassportInvite, listPassportInvites, revokePassportInvite, searchPassportServers } from '@/api/passportAdmin'
import type { GameServerBrief } from '@/types/v1/game_common'
import type { PassportInvite } from '@/types/v1/passport_admin'
import { feedback } from '@/utils/feedback'
import { Dialog } from '@/utils/dialog'
import { copyText, notifyError, usePassportTime } from '../shared'

const props = defineProps<{ modelValue: boolean; registerMode?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()
const { date } = usePassportTime()

const form = reactive({ code: '', email: '', serverId: undefined as string | undefined, maxUses: 1, expiresDays: 7, note: '' })
const server = ref<GameServerBrief>()
const creating = ref(false)
const justCreated = ref('')

const items = ref<PassportInvite[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const loading = ref(false)

const searchServers = async (keyword: string) => (await searchPassportServers({ q: keyword, limit: 20 })).data

const load = async () => {
  loading.value = true
  try {
    const res = await listPassportInvites(page.value, pageSize.value)
    items.value = res.data.items || []
    total.value = res.data.total || 0
  } catch (e) {
    notifyError(e)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    justCreated.value = ''
    page.value = 1
    void load()
  },
  { immediate: true },
)

const create = async () => {
  creating.value = true
  try {
    const res = await createPassportInvite({
      code: form.code.trim(),
      email: form.email.trim(),
      serverId: form.serverId || '',
      maxUses: Number(form.maxUses) || 1,
      expiresDays: Number(form.expiresDays) || 0,
      note: form.note.trim(),
    })
    justCreated.value = res.data.code
    feedback.success(t('passportAdmin.invite.created', { code: res.data.code }))
    form.code = ''
    form.email = ''
    form.note = ''
    page.value = 1
    await load()
  } catch (e) {
    notifyError(e)
  } finally {
    creating.value = false
  }
}

const revoke = (inv: PassportInvite) => {
  Dialog.confirm({
    title: t('passportAdmin.invite.revokeTitle'),
    content: t('passportAdmin.invite.revokeConfirm', { code: inv.code }),
    confirmText: t('passportAdmin.invite.revoke'),
    cancelText: t('passportAdmin.common.cancel'),
    onConfirm: async () => {
      try {
        await revokePassportInvite(inv.id)
        feedback.success(t('passportAdmin.invite.revoked'))
        await load()
      } catch (e) {
        notifyError(e)
      }
    },
  })
}

const copy = async (text: string) => {
  if (await copyText(text)) feedback.success(t('passportAdmin.oidc.copied'))
  else feedback.error(t('passportAdmin.oidc.copyFailed'))
}

const stateVariant = (state: string) => {
  switch (state) {
    case 'active':
      return 'success'
    case 'revoked':
      return 'error'
    default:
      return 'neutral'
  }
}

const inviteLine = (inv: PassportInvite) =>
  [
    t('passportAdmin.invite.uses', { used: inv.used, max: inv.maxUses }),
    inv.expiresAt ? t('passportAdmin.invite.expires', { date: date(inv.expiresAt) }) : t('passportAdmin.invite.noExpire'),
    inv.email ? t('passportAdmin.invite.forEmail', { email: inv.email }) : '',
    inv.server?.name || '',
    inv.createdByName ? t('passportAdmin.invite.by', { name: inv.createdByName }) : '',
  ]
    .filter(Boolean)
    .join(' · ')
</script>

<style scoped>
.pinv {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.pinv__mode {
  margin: 0;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.pinv__form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 14px;
}
.pinv__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.pinv__field--wide {
  grid-column: 1 / -1;
}
.pinv__label {
  font-size: 13px;
  font-weight: 700;
}
.pinv__hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.pinv__server {
  display: flex;
}
.pinv__actions {
  grid-column: 1 / -1;
  display: flex;
  justify-content: flex-end;
}
.pinv__created {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 8px 10px 14px;
  border-radius: var(--app-radius-sm);
  background: var(--m-okSoft);
}
.pinv__created .pinv__code {
  flex: 1;
}
.pinv__code {
  font-weight: 800;
  letter-spacing: 0.04em;
  word-break: break-all;
}
.pinv__list-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding-top: 10px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.pinv__list-title {
  font-weight: 800;
}
.pinv__list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.pinv__item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 6px 8px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
}
.pinv__item-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.pinv__item-title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.pinv__item-sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
@media (width < 640px) {
  .pinv__form {
    grid-template-columns: 1fr;
  }
}
</style>
