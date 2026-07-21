<!-- OIDC 服务端配置弹窗（P1）：FormDialog + 令牌字段（AppSwitch/输入/数字）+ 只读端点复制区。
     打开时拉取配置回填；保存后 emit saved(config) 供父级更新缓存。 -->
<template>
  <FormDialog
    v-model="open"
    :title="t('system.oidc.configTitle')"
    :width="720"
    :loading="saving"
    :confirm-text="t('system.oidc.saveConfig')"
    @confirm="save"
    @close="open = false"
  >
    <div class="oidc-config">
      <!-- 启用开关 -->
      <div class="oidc-config__row">
        <div class="oidc-config__info">
          <span class="oidc-config__label">{{ t('system.oidc.enableServer') }}</span>
          <span class="oidc-config__desc">{{ t('system.oidc.enableServerDesc') }}</span>
        </div>
        <AppSwitch v-model="form.enabled" :disabled="!canEdit" />
      </div>

      <!-- Issuer URL -->
      <div class="app-field">
        <label class="app-label">{{ t('system.oidc.issuerUrl') }}</label>
        <div class="oidc-config__inline">
          <input
            v-model="form.issuerUrl"
            class="app-input"
            :disabled="!canEdit"
            placeholder="https://example.com"
          />
          <UButton color="neutral" variant="soft" :disabled="!canEdit" @click="useCurrentOrigin">
            {{ t('system.oidc.useCurrentOrigin') }}
          </UButton>
        </div>
      </div>

      <!-- TTL -->
      <div class="oidc-config__ttl">
        <div class="app-field">
          <label class="app-label">{{ t('system.oidc.accessTokenTtl') }}</label>
          <input
            v-model.number="form.accessTokenTtlSeconds"
            type="number"
            min="60"
            max="86400"
            class="app-input"
            :disabled="!canEdit"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('system.oidc.idTokenTtl') }}</label>
          <input
            v-model.number="form.idTokenTtlSeconds"
            type="number"
            min="60"
            max="86400"
            class="app-input"
            :disabled="!canEdit"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('system.oidc.authCodeTtl') }}</label>
          <input
            v-model.number="form.authorizationCodeTtlSeconds"
            type="number"
            min="60"
            max="1800"
            class="app-input"
            :disabled="!canEdit"
          />
        </div>
      </div>

      <!-- 端点（只读 + 复制） -->
      <div v-if="endpointItems.length" class="oidc-config__endpoints">
        <div v-for="item in endpointItems" :key="item.label" class="app-field">
          <label class="app-label">{{ item.label }}</label>
          <div class="oidc-config__inline">
            <input class="app-input" :value="item.value" readonly />
            <AppIconButton
              icon="HOutline:ClipboardDocumentIcon"
              :label="t('common.copy')"
              :box="32"
              @click="copyText(item.value)"
            />
          </div>
        </div>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getOIDCConfig, updateOIDCConfig } from '@/api/oidc'
import type { OIDCConfig } from '@/types/v1/oidc'

defineOptions({ name: 'OIDCConfigDialog' })

defineProps<{ canEdit: boolean }>()
const emit = defineEmits<{ saved: [config: OIDCConfig] }>()
const { t } = useI18n()

const open = ref(false)
const saving = ref(false)
const config = ref<OIDCConfig | null>(null)

const form = ref({
  enabled: false,
  issuerUrl: '',
  accessTokenTtlSeconds: 3600,
  idTokenTtlSeconds: 3600,
  authorizationCodeTtlSeconds: 300,
})

const endpointItems = computed(() => {
  const e = config.value?.endpoints
  if (!e) return []
  return [
    { label: t('system.oidc.issuerUrl'), value: e.issuerUrl },
    { label: t('system.oidc.discoveryUrl'), value: e.discoveryUrl },
    { label: t('system.oidc.authorizeUrl'), value: e.authorizeUrl },
    { label: t('system.oidc.tokenUrl'), value: e.tokenUrl },
    { label: t('system.oidc.userinfoUrl'), value: e.userinfoUrl },
    { label: t('system.oidc.jwksUrl'), value: e.jwksUrl },
    { label: t('system.oidc.scopes'), value: e.scopes },
    { label: t('system.oidc.tokenAuthMethod'), value: e.tokenEndpointAuthMethod },
    { label: t('system.oidc.signingAlgs'), value: e.allowedIdTokenSigningAlgs },
  ].filter((item) => item.value)
})

const setForm = (value: OIDCConfig) => {
  form.value = {
    enabled: value.enabled,
    issuerUrl: value.issuerUrl,
    accessTokenTtlSeconds: value.accessTokenTtlSeconds,
    idTokenTtlSeconds: value.idTokenTtlSeconds,
    authorizationCodeTtlSeconds: value.authorizationCodeTtlSeconds,
  }
}

const loadConfig = async () => {
  const { data } = await getOIDCConfig()
  config.value = data?.config || null
  if (data?.config) setForm(data.config)
}

const useCurrentOrigin = () => {
  form.value.issuerUrl = window.location.origin
}

const save = async () => {
  saving.value = true
  try {
    const { data } = await updateOIDCConfig({ ...form.value })
    config.value = data?.config || null
    if (data?.config) {
      setForm(data.config)
      emit('saved', data.config)
    }
    feedback.success(t('system.oidc.saveSuccess'))
    open.value = false
  } finally {
    saving.value = false
  }
}

const copyText = async (value: string) => {
  await navigator.clipboard.writeText(value)
  feedback.success(t('system.oidc.copied'))
}

const showDialog = async () => {
  open.value = true
  await loadConfig()
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.oidc-config {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.oidc-config__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-lighter);
}
.oidc-config__info {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}
.oidc-config__label {
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
}
.oidc-config__desc {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.oidc-config__inline {
  display: flex;
  align-items: center;
  gap: 8px;
}
.oidc-config__inline .app-input {
  flex: 1;
  min-width: 0;
}
.oidc-config__ttl {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}
.oidc-config__endpoints {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  padding-top: 4px;
  border-top: 1px solid var(--el-border-color-lighter);
}
@media (width <= 768px) {
  .oidc-config__ttl,
  .oidc-config__endpoints {
    grid-template-columns: 1fr;
  }
}
</style>
