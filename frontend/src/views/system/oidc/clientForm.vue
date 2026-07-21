<!-- OIDC 客户端 创建/编辑弹窗（P1）：FormDialog + 令牌字段 + 内联校验。
     保存后：新建会返回完整 Client Secret → 通过 reveal 事件交父级展示；同时 refresh 列表。 -->
<template>
  <FormDialog
    v-model="open"
    :title="form.id ? t('system.oidc.editTitle') : t('system.oidc.createTitle')"
    :width="560"
    :loading="saving"
    @confirm="confirm"
    @close="close"
  >
    <div class="oidc-form">
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('system.oidc.providerName') }}</label>
        <input
          v-model="form.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          :placeholder="t('system.oidc.providerNamePlaceholder')"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
      </div>

      <div class="app-field">
        <label class="app-label app-label--required">{{ t('system.oidc.redirectUris') }}</label>
        <textarea
          v-model="form.redirectUrisText"
          class="app-textarea"
          :class="{ 'is-error': errors.redirectUris }"
          rows="4"
          :placeholder="t('system.oidc.redirectUrisPlaceholder')"
        />
        <span v-if="errors.redirectUris" class="app-field__error">{{ errors.redirectUris }}</span>
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('system.oidc.scopes') }}</label>
        <input v-model="form.scopesText" class="app-input" placeholder="openid email profile" />
        <span class="oidc-form__hint">{{ t('system.oidc.scopesHint') }}</span>
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('system.common.status') }}</label>
        <div class="oidc-form__switch">
          <AppSwitch v-model="form.active" />
          <span>{{ form.active ? t('system.common.enabled') : t('system.common.inactive') }}</span>
        </div>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { createOIDCClient, updateOIDCClient } from '@/api/oidc'
import type { OIDCClientInfo } from '@/types/v1/oidc'

defineOptions({ name: 'OIDCClientForm' })

const emit = defineEmits<{ refresh: []; reveal: [client: OIDCClientInfo] }>()
const { t } = useI18n()

const open = ref(false)
const saving = ref(false)
const errors = ref<Record<string, string>>({})

const emptyForm = () => ({
  id: '',
  name: 'OIDC',
  redirectUrisText: '',
  scopesText: 'openid email profile',
  active: true,
})
const form = ref(emptyForm())

const splitLines = (value: string) =>
  value
    .split(/[\n,\s]+/)
    .map((item) => item.trim())
    .filter(Boolean)

const close = () => {
  open.value = false
  saving.value = false
  errors.value = {}
  form.value = emptyForm()
}

const validate = (): boolean => {
  const e: Record<string, string> = {}
  if (!form.value.name.trim()) e.name = t('system.oidc.providerNameRequired')
  if (!splitLines(form.value.redirectUrisText).length) e.redirectUris = t('system.oidc.redirectUrisRequired')
  errors.value = e
  return Object.keys(e).length === 0
}

const confirm = async () => {
  if (!validate()) return
  saving.value = true
  try {
    const payload = {
      name: form.value.name.trim(),
      redirectUris: splitLines(form.value.redirectUrisText),
      scopes: splitLines(form.value.scopesText),
      active: form.value.active,
    }
    const { data } = form.value.id
      ? await updateOIDCClient({ id: form.value.id, ...payload })
      : await createOIDCClient(payload)
    feedback.success(form.value.id ? t('system.oidc.editSuccess') : t('system.oidc.createSuccess'))
    emit('refresh')
    // 新建/刷新会带完整 secret（不含掩码 *），交父级弹配置展示
    if (data?.client?.clientSecret && !data.client.clientSecret.includes('*')) {
      emit('reveal', data.client)
    }
    close()
  } finally {
    saving.value = false
  }
}

const showDialog = (row?: OIDCClientInfo) => {
  errors.value = {}
  form.value = {
    id: row?.id || '',
    name: row?.name || 'OIDC',
    redirectUrisText: row?.redirectUris.join('\n') || '',
    scopesText: row?.scopes.join(' ') || 'openid email profile',
    active: row?.active ?? true,
  }
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.oidc-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.oidc-form__hint {
  font-size: 0.72rem;
  color: var(--el-text-color-placeholder);
}
.oidc-form__switch {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
}
</style>
