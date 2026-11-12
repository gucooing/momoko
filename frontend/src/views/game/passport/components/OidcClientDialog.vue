<!-- 新建 / 编辑「用户通行证登录」的 OIDC 应用。 -->
<template>
  <FormDialog
    :model-value="modelValue"
    :title="client ? t('passportAdmin.oidc.editTitle') : t('passportAdmin.oidc.createTitle')"
    :width="560"
    :loading="saving"
    :confirm-text="t('passportAdmin.common.save')"
    @update:model-value="emit('update:modelValue', $event)"
    @confirm="submit"
  >
    <div class="poc">
      <label class="poc__field">
        <span class="poc__label">{{ t('passportAdmin.oidc.name') }}</span>
        <UInput v-model="form.name" :placeholder="t('passportAdmin.oidc.namePlaceholder')" maxlength="60" />
      </label>
      <label class="poc__field">
        <span class="poc__label">{{ t('passportAdmin.oidc.description') }}</span>
        <UInput v-model="form.description" :placeholder="t('passportAdmin.oidc.descriptionPlaceholder')" maxlength="200" />
      </label>
      <label class="poc__field">
        <span class="poc__label">{{ t('passportAdmin.oidc.redirectUris') }}</span>
        <UTextarea v-model="form.redirectUris" :rows="3" autoresize class="m-mono" />
        <span class="poc__hint">{{ t('passportAdmin.oidc.redirectHint') }}</span>
      </label>
      <div class="poc__field">
        <span class="poc__label">{{ t('passportAdmin.oidc.scopes') }}</span>
        <div class="poc__scopes">
          <label v-for="s in SCOPES" :key="s" class="poc__scope">
            <input v-model="form.scopes" type="checkbox" :value="s" :disabled="s === 'openid'" />
            <span class="m-mono">{{ s }}</span>
          </label>
        </div>
      </div>
      <div class="poc__field">
        <span class="poc__label">{{ t('passportAdmin.oidc.server') }}</span>
        <div class="poc__server">
          <GameServerPicker v-model="form.serverId" :selected="server" :search="searchServers" allow-all :placeholder="t('passportAdmin.oidc.serverNone')" @select="server = $event" />
        </div>
        <span class="poc__hint">{{ t('passportAdmin.oidc.serverHint') }}</span>
      </div>
      <div class="poc__row">
        <span class="poc__label">{{ t('passportAdmin.oidc.active') }}</span>
        <AppSwitch v-model="form.active" />
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import GameServerPicker from '@/views/game/components/GameServerPicker.vue'
import { createPassportOidcClient, searchPassportServers, updatePassportOidcClient } from '@/api/passportAdmin'
import type { GameServerBrief } from '@/types/v1/game_common'
import type { PassportOidcClient } from '@/types/v1/passport_admin'
import { feedback } from '@/utils/feedback'
import { notifyError } from '../shared'

const props = defineProps<{ modelValue: boolean; client?: PassportOidcClient }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [client: PassportOidcClient, created: boolean] }>()

/** 通行证 OIDC Provider 支持的授权范围（openid 必选）。 */
const SCOPES = ['openid', 'profile', 'email']

const { t } = useI18n()
const form = reactive({
  name: '',
  description: '',
  redirectUris: '',
  scopes: ['openid', 'profile'] as string[],
  serverId: undefined as string | undefined,
  active: true,
})
const server = ref<GameServerBrief>()
const saving = ref(false)

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    const c = props.client
    form.name = c?.name || ''
    form.description = c?.description || ''
    form.redirectUris = (c?.redirectUris || []).join('\n')
    form.scopes = c ? Array.from(new Set(['openid', ...c.scopes])) : ['openid', 'profile']
    form.serverId = c?.server?.id || undefined
    form.active = c ? c.active : true
    server.value = c?.server
  },
  { immediate: true },
)

const searchServers = async (keyword: string) => (await searchPassportServers({ q: keyword, limit: 20 })).data

const submit = async () => {
  const name = form.name.trim()
  const uris = form.redirectUris
    .split(/[\s,]+/)
    .map((u) => u.trim())
    .filter(Boolean)
  if (!name) {
    feedback.warning(t('passportAdmin.oidc.nameRequired'))
    return
  }
  if (!uris.length) {
    feedback.warning(t('passportAdmin.oidc.redirectRequired'))
    return
  }
  saving.value = true
  try {
    const body = {
      id: props.client?.id || '',
      name,
      description: form.description.trim(),
      redirectUris: uris,
      scopes: form.scopes,
      active: form.active,
      serverId: form.serverId || '',
    }
    const res = props.client ? await updatePassportOidcClient(body) : await createPassportOidcClient(body)
    feedback.success(t('passportAdmin.oidc.saved'))
    emit('saved', res.data, !props.client)
    emit('update:modelValue', false)
  } catch (e) {
    notifyError(e)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.poc {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.poc__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.poc__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 44px;
}
.poc__label {
  font-size: 13px;
  font-weight: 700;
}
.poc__hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.poc__scopes {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.poc__scope {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  cursor: pointer;
}
.poc__scope input {
  accent-color: var(--m-p);
}
.poc__server {
  display: flex;
}
</style>
