<!-- OIDC 服务端配置弹窗（重写 · P1）：FormDialog + 令牌字段（AppSwitch/输入/数字）+ 只读端点复制区。
     打开时拉取配置回填；保存后 emit saved(config) 供父级更新缓存（客户端配置展示需要端点）。文案硬编码中文（Phase 4 统一 i18n）。 -->
<template>
  <FormDialog
    v-model="open"
    title="OIDC 配置"
    :width="720"
    :loading="saving"
    confirm-text="保存配置"
    @confirm="save"
    @close="open = false"
  >
    <div class="oidc-config">
      <!-- 启用开关 -->
      <div class="oidc-config__row">
        <div class="oidc-config__info">
          <span class="oidc-config__label">启用 OIDC 服务端</span>
          <span class="oidc-config__desc">
            开启后可通过 Discovery、Authorize、Token、UserInfo、JWKS 端点接入第三方系统
          </span>
        </div>
        <AppSwitch v-model="form.enabled" :disabled="!canEdit" />
      </div>

      <!-- Issuer URL -->
      <div class="app-field">
        <label class="app-label">Issuer URL</label>
        <div class="oidc-config__inline">
          <input
            v-model="form.issuerUrl"
            class="app-input"
            :disabled="!canEdit"
            placeholder="https://example.com"
          />
          <UButton color="neutral" variant="soft" :disabled="!canEdit" @click="useCurrentOrigin">
            当前域名
          </UButton>
        </div>
      </div>

      <!-- TTL -->
      <div class="oidc-config__ttl">
        <div class="app-field">
          <label class="app-label">Access Token 有效期（秒）</label>
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
          <label class="app-label">ID Token 有效期（秒）</label>
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
          <label class="app-label">授权码有效期（秒）</label>
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
              label="复制"
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
import { getOIDCConfig, updateOIDCConfig } from '@/api/oidc'
import type { OIDCConfig } from '@/types/v1/oidc'

defineOptions({ name: 'OIDCConfigDialog' })

defineProps<{ canEdit: boolean }>()
const emit = defineEmits<{ saved: [config: OIDCConfig] }>()

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
    { label: 'Issuer URL', value: e.issuerUrl },
    { label: 'Discovery URL', value: e.discoveryUrl },
    { label: 'Authorize URL', value: e.authorizeUrl },
    { label: 'Token URL', value: e.tokenUrl },
    { label: 'UserInfo URL', value: e.userinfoUrl },
    { label: 'JWKS URL', value: e.jwksUrl },
    { label: 'Scopes', value: e.scopes },
    { label: 'Token 鉴权方式', value: e.tokenEndpointAuthMethod },
    { label: '允许的签名算法', value: e.allowedIdTokenSigningAlgs },
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
    feedback.success('保存成功')
    open.value = false
  } finally {
    saving.value = false
  }
}

const copyText = async (value: string) => {
  await navigator.clipboard.writeText(value)
  feedback.success('已复制')
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
