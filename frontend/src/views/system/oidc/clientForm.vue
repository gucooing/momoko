<!-- OIDC 客户端 创建/编辑弹窗（重写 · P1）：FormDialog + 令牌字段 + 内联校验。
     保存后：新建/刷新会返回完整 Client Secret → 通过 reveal 事件交父级弹「客户端配置」展示；同时 refresh 列表。
     文案沿用原页面硬编码中文（i18n 全量核对属 Phase 4）。 -->
<template>
  <FormDialog
    v-model="open"
    :title="form.id ? '编辑 OIDC 客户端' : '生成 OIDC 客户端配置'"
    :width="560"
    :loading="saving"
    @confirm="confirm"
    @close="close"
  >
    <div class="oidc-form">
      <div class="app-field">
        <label class="app-label app-label--required">Provider 名称</label>
        <input
          v-model="form.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          placeholder="OIDC"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
      </div>

      <div class="app-field">
        <label class="app-label app-label--required">回调地址</label>
        <textarea
          v-model="form.redirectUrisText"
          class="app-textarea"
          :class="{ 'is-error': errors.redirectUris }"
          rows="4"
          placeholder="每行一个 Redirect URI"
        />
        <span v-if="errors.redirectUris" class="app-field__error">{{ errors.redirectUris }}</span>
      </div>

      <div class="app-field">
        <label class="app-label">Scopes</label>
        <input v-model="form.scopesText" class="app-input" placeholder="openid email profile" />
        <span class="oidc-form__hint">留空默认 openid email profile；必须包含 openid</span>
      </div>

      <div class="app-field">
        <label class="app-label">状态</label>
        <div class="oidc-form__switch">
          <AppSwitch v-model="form.active" />
          <span>{{ form.active ? '启用' : '停用' }}</span>
        </div>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { createOIDCClient, updateOIDCClient } from '@/api/oidc'
import type { OIDCClientInfo } from '@/types/v1/oidc'

defineOptions({ name: 'OIDCClientForm' })

const emit = defineEmits<{ refresh: []; reveal: [client: OIDCClientInfo] }>()

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
  if (!form.value.name.trim()) e.name = '请输入 Provider 名称'
  if (!splitLines(form.value.redirectUrisText).length) e.redirectUris = '请至少填写一个回调地址'
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
    feedback.success(form.value.id ? '编辑成功' : '生成成功')
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
