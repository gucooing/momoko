<!-- 通行证设置：注册与登录（存在 Momoko 系统配置里）、通行证服务的只读信息（配置文件）、「用户通行证登录」的 OIDC 应用。 -->
<template>
  <FormDialog :model-value="modelValue" :title="t('passportAdmin.settings.title')" :width="760" :show-footer="false" @update:model-value="emit('update:modelValue', $event)">
    <div v-if="loading && !info" class="pset__hint">{{ t('passportAdmin.common.loading') }}</div>
    <div v-else class="pset">
      <section class="pset__section">
        <h3 class="pset__title">{{ t('passportAdmin.settings.general') }}</h3>
        <label class="pset__field">
          <span class="pset__label">{{ t('passportAdmin.settings.siteName') }}</span>
          <UInput v-model="form.siteName" :placeholder="t('passportAdmin.settings.siteNamePlaceholder')" maxlength="40" />
        </label>
        <div class="pset__field">
          <span class="pset__label">{{ t('passportAdmin.settings.registerMode') }}</span>
          <div class="pset__seg" role="radiogroup" :aria-label="t('passportAdmin.settings.registerMode')">
            <button
              v-for="m in MODES"
              :key="m"
              type="button"
              role="radio"
              class="pset__seg-btn"
              :class="{ 'is-on': form.registerMode === m }"
              :aria-checked="form.registerMode === m"
              @click="form.registerMode = m"
            >
              {{ t(`passportAdmin.settings.modes.${m}`) }}
            </button>
          </div>
        </div>
        <p v-if="info && !info.emailConfigured" class="pset__warn">{{ t('passportAdmin.settings.emailMissing') }}</p>
        <div class="pset__row">
          <span class="pset__label">{{ t('passportAdmin.settings.registerEmail') }}</span>
          <AppSwitch v-model="form.registerEmailVerification" />
        </div>
        <div class="pset__row">
          <span class="pset__label">{{ t('passportAdmin.settings.emailLogin') }}</span>
          <AppSwitch v-model="form.emailLoginEnabled" />
        </div>
        <label class="pset__field">
          <span class="pset__label">{{ t('passportAdmin.settings.sessionDays') }}</span>
          <UInput v-model.number="form.sessionDays" type="number" min="1" max="365" class="pset__num" />
          <span class="pset__hint">{{ t('passportAdmin.settings.sessionDaysHint') }}</span>
        </label>
        <div class="pset__row">
          <span class="pset__label-col">
            <span class="pset__label">{{ t('passportAdmin.settings.oidcEnabled') }}</span>
            <span class="pset__hint">{{ t('passportAdmin.settings.oidcHint') }}</span>
          </span>
          <AppSwitch v-model="form.oidcEnabled" />
        </div>
        <div class="pset__actions">
          <UButton :loading="saving" @click="save">{{ t('passportAdmin.settings.save') }}</UButton>
        </div>
      </section>

      <section v-if="info" class="pset__section">
        <h3 class="pset__title">{{ t('passportAdmin.settings.server') }}</h3>
        <p v-if="!info.serverEnabled" class="pset__warn">{{ t('passportAdmin.settings.serverOff') }}</p>
        <dl class="pset__dl">
          <div><dt>{{ t('passportAdmin.settings.listen') }}</dt><dd class="m-mono">{{ info.listenAddr }}</dd></div>
          <div><dt>{{ t('passportAdmin.settings.publicUrl') }}</dt><dd class="m-mono">{{ info.publicUrl || t('passportAdmin.settings.publicUrlEmpty') }}</dd></div>
          <div><dt>{{ t('passportAdmin.settings.issuer') }}</dt><dd class="m-mono">{{ info.oidcIssuer || t('passportAdmin.settings.issuerEmpty') }}</dd></div>
          <div><dt>{{ t('passportAdmin.settings.discovery') }}</dt><dd class="m-mono">{{ info.oidcDiscoveryUrl || t('passportAdmin.settings.issuerEmpty') }}</dd></div>
          <div><dt>{{ t('passportAdmin.settings.avatar') }}</dt><dd>{{ info.avatarUploadEnabled ? t('passportAdmin.settings.avatarOn') : t('passportAdmin.settings.avatarOff') }}</dd></div>
        </dl>
        <p class="pset__hint">{{ t('passportAdmin.settings.configHint') }}</p>
      </section>

      <section class="pset__section">
        <div class="pset__head">
          <h3 class="pset__title">{{ t('passportAdmin.oidc.title') }} <span class="pset__hint m-num">{{ t('passportAdmin.oidc.count', { n: clients.length }) }}</span></h3>
          <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-plus" @click="openClient()">{{ t('passportAdmin.oidc.create') }}</UButton>
        </div>

        <div v-if="secret" class="pset__secret">
          <span class="pset__label">{{ t('passportAdmin.oidc.clientSecret') }} · {{ secret.name }}</span>
          <span class="pset__secret-row">
            <code class="m-mono pset__secret-value">{{ secret.value }}</code>
            <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('passportAdmin.oidc.copy')" :box="40" :size="18" @click="copy(secret.value)" />
          </span>
          <span class="pset__hint">{{ t('passportAdmin.oidc.secretOnce') }}</span>
        </div>

        <EmptyState v-if="!clients.length" icon="HOutline:KeyIcon" :title="t('passportAdmin.oidc.empty')" :description="t('passportAdmin.oidc.emptyDesc')" />
        <div v-else class="pset__clients">
          <div v-for="c in clients" :key="c.id" class="pset__client">
            <span class="pset__client-main">
              <span class="pset__client-title">
                <span class="pset__client-name">{{ c.name }}</span>
                <StatusPill v-if="!c.active" variant="neutral" :label="t('passportAdmin.oidc.inactive')" />
              </span>
              <span class="pset__client-sub m-mono">{{ c.clientId }}</span>
              <span class="pset__client-sub">{{ clientLine(c) }}</span>
            </span>
            <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('passportAdmin.oidc.copy')" :box="40" :size="18" @click="copy(c.clientId)" />
            <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('passportAdmin.oidc.edit')" :box="40" :size="18" @click="openClient(c)" />
            <AppIconButton icon="HOutline:KeyIcon" :label="t('passportAdmin.oidc.refreshSecret')" :box="40" :size="18" @click="onClientAction(c, 'secret')" />
            <AppIconButton icon="HOutline:TrashIcon" class="pset__danger" :label="t('passportAdmin.oidc.delete')" :box="40" :size="18" @click="onClientAction(c, 'delete')" />
          </div>
        </div>
      </section>
    </div>

    <OidcClientDialog v-model="clientOpen" :client="editing" @saved="onClientSaved" />
  </FormDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  deletePassportOidcClient,
  getPassportSettings,
  listPassportOidcClients,
  refreshPassportOidcClientSecret,
  updatePassportSettings,
} from '@/api/passportAdmin'
import type { GetPassportSettingsResponse, PassportOidcClient } from '@/types/v1/passport_admin'
import { feedback } from '@/utils/feedback'
import { Dialog } from '@/utils/dialog'
import OidcClientDialog from './OidcClientDialog.vue'
import { copyText, notifyError, usePassportTime } from '../shared'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [info: GetPassportSettingsResponse] }>()

const MODES = ['open', 'invite_only', 'closed']

const { t } = useI18n()
const { ago } = usePassportTime()

const info = ref<GetPassportSettingsResponse>()
const loading = ref(false)
const saving = ref(false)
const form = reactive({
  siteName: '',
  registerMode: 'open',
  registerEmailVerification: false,
  emailLoginEnabled: true,
  sessionDays: 30,
  oidcEnabled: false,
})

const clients = ref<PassportOidcClient[]>([])
const clientOpen = ref(false)
const editing = ref<PassportOidcClient>()
const secret = ref<{ name: string; value: string }>()

const fill = (res: GetPassportSettingsResponse) => {
  info.value = res
  const s = res.settings
  if (!s) return
  form.siteName = s.siteName
  form.registerMode = s.registerMode || 'open'
  form.registerEmailVerification = s.registerEmailVerification
  form.emailLoginEnabled = s.emailLoginEnabled
  form.sessionDays = s.sessionDays || 30
  form.oidcEnabled = s.oidcEnabled
}

const loadClients = async () => {
  try {
    clients.value = (await listPassportOidcClients()).data.items || []
  } catch (e) {
    notifyError(e)
  }
}

const load = async () => {
  loading.value = true
  try {
    fill((await getPassportSettings()).data)
  } catch (e) {
    notifyError(e)
  } finally {
    loading.value = false
  }
  await loadClients()
}

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    secret.value = undefined
    void load()
  },
  { immediate: true },
)

const save = async () => {
  saving.value = true
  try {
    const res = await updatePassportSettings({
      siteName: form.siteName.trim(),
      registerMode: form.registerMode,
      registerEmailVerification: form.registerEmailVerification,
      emailLoginEnabled: form.emailLoginEnabled,
      sessionDays: Number(form.sessionDays) || 30,
      oidcEnabled: form.oidcEnabled,
    })
    fill(res.data)
    emit('saved', res.data)
    feedback.success(t('passportAdmin.settings.saved'))
  } catch (e) {
    notifyError(e)
  } finally {
    saving.value = false
  }
}

const openClient = (c?: PassportOidcClient) => {
  editing.value = c
  clientOpen.value = true
}

const onClientSaved = async (c: PassportOidcClient, created: boolean) => {
  if (created) secret.value = { name: c.name, value: c.clientSecret }
  await loadClients()
}

const onClientAction = (c: PassportOidcClient, key: string) => {
  if (key === 'secret') {
    Dialog.confirm({
      title: t('passportAdmin.oidc.refreshTitle'),
      content: t('passportAdmin.oidc.refreshConfirm'),
      confirmText: t('passportAdmin.oidc.refreshSecret'),
      cancelText: t('passportAdmin.common.cancel'),
      onConfirm: async () => {
        try {
          const res = await refreshPassportOidcClientSecret(c.id)
          secret.value = { name: res.data.name, value: res.data.clientSecret }
          await loadClients()
        } catch (e) {
          notifyError(e)
        }
      },
    })
  } else if (key === 'delete') {
    Dialog.confirm({
      title: t('passportAdmin.oidc.deleteTitle'),
      content: t('passportAdmin.oidc.deleteConfirm', { name: c.name }),
      confirmText: t('passportAdmin.oidc.delete'),
      cancelText: t('passportAdmin.common.cancel'),
      onConfirm: async () => {
        try {
          await deletePassportOidcClient(c.id)
          feedback.success(t('passportAdmin.oidc.deleted'))
          if (secret.value?.name === c.name) secret.value = undefined
          await loadClients()
        } catch (e) {
          notifyError(e)
        }
      },
    })
  }
}

const clientLine = (c: PassportOidcClient) =>
  [
    c.server?.name || '',
    c.scopes.join(' '),
    t('passportAdmin.oidc.grants', { n: c.grantCount }),
    c.lastUsedAt ? t('passportAdmin.oidc.lastUsed', { time: ago(c.lastUsedAt) }) : t('passportAdmin.oidc.neverUsed'),
  ]
    .filter(Boolean)
    .join(' · ')

const copy = async (text: string) => {
  if (await copyText(text)) feedback.success(t('passportAdmin.oidc.copied'))
  else feedback.error(t('passportAdmin.oidc.copyFailed'))
}
</script>

<style scoped>
.pset {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.pset__section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.pset__section + .pset__section {
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.pset__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.pset__title {
  margin: 0;
  font-size: 16px;
  font-weight: 800;
}
.pset__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.pset__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 44px;
}
.pset__label-col {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.pset__label {
  font-size: 13px;
  font-weight: 700;
}
.pset__hint {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.pset__warn {
  margin: 0;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-warnSoft);
  color: var(--m-ink);
  font-size: 13px;
}
.pset__num {
  max-width: 160px;
}
.pset__seg {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.pset__seg-btn {
  min-height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
}
.pset__seg-btn.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.pset__actions {
  display: flex;
  justify-content: flex-end;
}
.pset__dl {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.pset__dl dt {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.pset__dl dd {
  margin: 0;
  font-size: 13px;
  font-weight: 700;
  word-break: break-all;
}
.pset__secret {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--m-okSoft);
}
.pset__secret-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.pset__secret-value {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 700;
  word-break: break-all;
}
.pset__clients {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.pset__client {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 6px 8px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
}
.pset__client-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.pset__client-title {
  display: flex;
  align-items: center;
  gap: 8px;
}
.pset__client-name {
  font-weight: 800;
}
.pset__client-sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pset__danger {
  color: var(--m-bad);
}
.pset__hint:only-child {
  padding: 24px 0;
  text-align: center;
}
@media (width < 640px) {
  .pset__dl {
    grid-template-columns: 1fr;
  }
}
</style>
