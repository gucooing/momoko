<!-- SSH 连接 新建/编辑弹窗（重写 · P1）：FormDialog + 令牌字段 + authType seg + 条件密码/密钥（眼睛显隐）
     + UserPicker 分享用户 + 内联校验。保留 patch-diff 更新（仅提交变更字段 + 非空凭据）与 ref 契约 showDialog(row?) + @refresh。 -->
<template>
  <FormDialog
    v-model="open"
    :title="editingId ? t('ssh.common.editConnection') : t('ssh.common.addConnection')"
    :width="680"
    :loading="loading"
    @close="close"
  >
    <div class="ssh-form">
      <div class="app-field ssh-form__full">
        <label class="app-label app-label--required">{{ t('ssh.common.name') }}</label>
        <input
          v-model="form.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          :placeholder="t('ssh.common.namePlaceholder')"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
      </div>

      <div class="app-field ssh-form__host">
        <label class="app-label app-label--required">{{ t('ssh.common.host') }}</label>
        <input
          v-model="form.host"
          class="app-input"
          :class="{ 'is-error': errors.host }"
          :placeholder="t('ssh.common.hostPlaceholder')"
        />
        <span v-if="errors.host" class="app-field__error">{{ errors.host }}</span>
      </div>
      <div class="app-field ssh-form__port">
        <label class="app-label app-label--required">{{ t('ssh.common.port') }}</label>
        <input v-model.number="form.port" type="number" min="1" max="65535" class="app-input" />
      </div>

      <div class="app-field ssh-form__full">
        <label class="app-label app-label--required">{{ t('ssh.common.username') }}</label>
        <input
          v-model="form.username"
          class="app-input"
          :class="{ 'is-error': errors.username }"
          :placeholder="t('ssh.common.usernamePlaceholder')"
        />
        <span v-if="errors.username" class="app-field__error">{{ errors.username }}</span>
      </div>

      <div class="app-field ssh-form__full">
        <label class="app-label">{{ t('ssh.common.authType') }}</label>
        <div class="seg seg--wide">
          <button
            type="button"
            class="seg__btn"
            :class="{ 'is-active': form.authType === 'SSH_AUTH_TYPE_PASSWORD' }"
            @click="form.authType = 'SSH_AUTH_TYPE_PASSWORD'"
          >
            {{ t('ssh.common.passwordLogin') }}
          </button>
          <button
            type="button"
            class="seg__btn"
            :class="{ 'is-active': form.authType === 'SSH_AUTH_TYPE_KEY' }"
            @click="form.authType = 'SSH_AUTH_TYPE_KEY'"
          >
            {{ t('ssh.common.keyLogin') }}
          </button>
        </div>
      </div>

      <div v-if="form.authType === 'SSH_AUTH_TYPE_PASSWORD'" class="app-field ssh-form__full">
        <label class="app-label" :class="{ 'app-label--required': !editingId }">{{ t('ssh.common.password') }}</label>
        <div class="ssh-form__pwd">
          <input
            v-model="form.password"
            :type="showPassword ? 'text' : 'password'"
            class="app-input"
            :class="{ 'is-error': errors.password }"
            :placeholder="t('ssh.common.passwordPlaceholder')"
            autocomplete="new-password"
          />
          <AppIconButton
            :icon="showPassword ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon'"
            :label="t('ssh.common.password')"
            :box="32"
            @click="showPassword = !showPassword"
          />
        </div>
        <span v-if="errors.password" class="app-field__error">{{ errors.password }}</span>
      </div>

      <template v-else>
        <div class="app-field ssh-form__full">
          <label class="app-label" :class="{ 'app-label--required': !editingId }">{{ t('ssh.common.privateKey') }}</label>
          <textarea
            v-model="form.privateKey"
            class="app-textarea"
            :class="{ 'is-error': errors.privateKey }"
            rows="4"
            :placeholder="t('ssh.common.privateKeyPlaceholder')"
          />
          <span v-if="errors.privateKey" class="app-field__error">{{ errors.privateKey }}</span>
        </div>
        <div class="app-field ssh-form__full">
          <label class="app-label">{{ t('ssh.common.passphrase') }}</label>
          <div class="ssh-form__pwd">
            <input
              v-model="form.passphrase"
              :type="showPassphrase ? 'text' : 'password'"
              class="app-input"
              :placeholder="t('ssh.common.passphrasePlaceholder')"
              autocomplete="new-password"
            />
            <AppIconButton
              :icon="showPassphrase ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon'"
              :label="t('ssh.common.passphrase')"
              :box="32"
              @click="showPassphrase = !showPassphrase"
            />
          </div>
        </div>
      </template>

      <div class="app-field ssh-form__full">
        <label class="app-label">{{ t('ssh.common.fingerprint') }}</label>
        <input v-model="form.fingerprint" class="app-input" :placeholder="t('ssh.common.fingerprintPlaceholder')" />
      </div>

      <div class="app-field ssh-form__full">
        <label class="app-label">{{ t('ssh.common.tags') }}</label>
        <input v-model="form.tags" class="app-input" :placeholder="t('ssh.common.tagsPlaceholder')" />
      </div>

      <div class="app-field ssh-form__full">
        <label class="app-label">{{ t('ssh.common.sharedUsers') }}</label>
        <UserPicker v-model="form.sharedUserIds" :placeholder="t('ssh.common.shareUsersPlaceholder')" />
      </div>

      <div class="app-field ssh-form__full">
        <label class="app-label">{{ t('ssh.common.remark') }}</label>
        <textarea v-model="form.remark" class="app-textarea" rows="2" :placeholder="t('ssh.common.remarkPlaceholder')" />
      </div>
    </div>

    <template #footer="{ close: closeDialog }">
      <UButton color="neutral" variant="soft" :disabled="loading || testing" @click="closeDialog()">
        {{ t('ssh.common.cancel') }}
      </UButton>
      <UButton color="neutral" variant="soft" :loading="testing" :disabled="loading" @click="testDraft">
        {{ t('ssh.common.testConnection') }}
      </UButton>
      <UButton color="primary" :loading="loading" :disabled="testing" @click="confirm">
        {{ t('ssh.common.confirm') }}
      </UButton>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { createSshHost, updateSshHost, testSshHost } from '@/api/openssh'
import {
  SSHAuthType,
  type SSHHostInfo,
  type TestSSHHostRequest,
  type UpdateSSHHostRequest,
} from '@/types/v1/openssh'

defineOptions({ name: 'SshConnectionCreate' })

const emits = defineEmits<{ refresh: [type: 'create' | 'update'] }>()
const { t } = useI18n()

const open = ref(false)
const loading = ref(false)
const testing = ref(false)
const editingId = ref('')
const showPassword = ref(false)
const showPassphrase = ref(false)
const errors = ref<Record<string, string>>({})

const defaultForm = () => ({
  name: '',
  host: '',
  port: 22,
  username: '',
  authType: 'SSH_AUTH_TYPE_PASSWORD' as string,
  password: '',
  privateKey: '',
  passphrase: '',
  fingerprint: '',
  tags: '',
  sharedUserIds: [] as string[],
  remark: '',
})

const form = ref(defaultForm())
const originalForm = ref(defaultForm())

const close = () => {
  open.value = false
  loading.value = false
  testing.value = false
  editingId.value = ''
  errors.value = {}
  showPassword.value = false
  showPassphrase.value = false
  form.value = defaultForm()
}

// 用当前表单草稿测连通性；编辑时空凭据走库中已存配置
const testDraft = async () => {
  if (!form.value.host.trim() || !form.value.username.trim()) {
    feedback.warning(t('ssh.common.testNeedHostUser'))
    return
  }
  if (!editingId.value) {
    if (form.value.authType === 'SSH_AUTH_TYPE_PASSWORD' && !form.value.password.trim()) {
      feedback.warning(t('ssh.common.passwordPlaceholder'))
      return
    }
    if (form.value.authType === 'SSH_AUTH_TYPE_KEY' && !form.value.privateKey.trim()) {
      feedback.warning(t('ssh.common.privateKeyRequired'))
      return
    }
  }

  testing.value = true
  try {
    const payload: TestSSHHostRequest = {
      host: form.value.host.trim(),
      port: form.value.port,
      username: form.value.username.trim(),
      authType: form.value.authType as SSHAuthType,
      fingerprint: form.value.fingerprint || undefined,
    }
    if (editingId.value) payload.id = editingId.value
    if (form.value.password.trim()) payload.password = form.value.password
    if (form.value.privateKey.trim()) {
      payload.privateKey = form.value.privateKey
      if (form.value.passphrase) payload.passphrase = form.value.passphrase
    }

    const { data } = await testSshHost(payload)
    if (data?.ok) {
      if (data.fingerprint && !form.value.fingerprint) {
        form.value.fingerprint = data.fingerprint
      }
      feedback.success(
        t('ssh.common.connectionSuccess', { message: data.message || t('ssh.common.online') }),
      )
    } else {
      feedback.warning(
        t('ssh.common.connectionFailedWithMessage', {
          message: data?.message || t('ssh.common.offline'),
        }),
      )
    }
  } finally {
    testing.value = false
  }
}

const validate = () => {
  const e: Record<string, string> = {}
  if (!form.value.name.trim()) e.name = t('ssh.common.namePlaceholder')
  if (!form.value.host.trim()) e.host = t('ssh.common.hostPlaceholder')
  if (!form.value.username.trim()) e.username = t('ssh.common.usernamePlaceholder')
  if (form.value.authType === 'SSH_AUTH_TYPE_PASSWORD' && !editingId.value && !form.value.password.trim())
    e.password = t('ssh.common.passwordPlaceholder')
  if (form.value.authType === 'SSH_AUTH_TYPE_KEY' && !editingId.value && !form.value.privateKey.trim())
    e.privateKey = t('ssh.common.privateKeyRequired')
  errors.value = e
  return Object.keys(e).length === 0
}

const confirm = async () => {
  if (!validate()) return
  loading.value = true
  try {
    if (editingId.value) {
      // patch: 仅提交变更字段
      const patch: Record<string, unknown> = { id: editingId.value }
      const tracked: (keyof typeof form.value)[] = [
        'name', 'host', 'port', 'username', 'authType', 'fingerprint', 'tags', 'remark', 'sharedUserIds',
      ]
      for (const key of tracked) {
        if (JSON.stringify(form.value[key]) !== JSON.stringify(originalForm.value[key])) {
          patch[key] = form.value[key]
        }
      }
      // 凭据：仅当用户输入了新值才包含
      if (form.value.password) {
        patch.password = form.value.password
        if (!patch.authType) patch.authType = form.value.authType
      }
      if (form.value.privateKey) {
        patch.privateKey = form.value.privateKey
        if (form.value.passphrase) patch.passphrase = form.value.passphrase
        if (!patch.authType) patch.authType = form.value.authType
      }
      await updateSshHost(patch as unknown as UpdateSSHHostRequest)
    } else {
      await createSshHost({
        name: form.value.name,
        host: form.value.host,
        port: form.value.port,
        username: form.value.username,
        authType: form.value.authType as SSHAuthType,
        password: form.value.password,
        privateKey: form.value.privateKey,
        passphrase: form.value.passphrase,
        fingerprint: form.value.fingerprint,
        tags: form.value.tags,
        remark: form.value.remark,
        sharedUserIds: form.value.sharedUserIds,
      })
    }
    feedback.success(editingId.value ? t('ssh.common.editSuccess') : t('ssh.common.addSuccess'))
    emits('refresh', editingId.value ? 'update' : 'create')
    close()
  } finally {
    loading.value = false
  }
}

const showDialog = (payload?: SSHHostInfo) => {
  errors.value = {}
  showPassword.value = false
  showPassphrase.value = false
  if (!payload?.id) {
    editingId.value = ''
    form.value = defaultForm()
    originalForm.value = defaultForm()
    open.value = true
    return
  }

  editingId.value = payload.id
  form.value = {
    name: payload.name || '',
    host: payload.host || '',
    port: payload.port || 22,
    username: payload.username || '',
    authType: payload.authType || 'SSH_AUTH_TYPE_PASSWORD',
    password: '',
    privateKey: '',
    passphrase: '',
    fingerprint: payload.fingerprint || '',
    tags: payload.tags || '',
    sharedUserIds: payload.sharedUsers?.map((user) => user.userId) || [],
    remark: payload.remark || '',
  }
  originalForm.value = { ...form.value, sharedUserIds: [...form.value.sharedUserIds] }
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.ssh-form {
  display: grid;
  grid-template-columns: 1fr 120px;
  gap: 14px;
}
.ssh-form__full {
  grid-column: 1 / -1;
}
.ssh-form__host {
  grid-column: 1 / 2;
}
.ssh-form__port {
  grid-column: 2 / 3;
}
.ssh-form__pwd {
  display: flex;
  align-items: center;
  gap: 8px;
}
.ssh-form__pwd .app-input {
  flex: 1;
  min-width: 0;
}

/* authType seg */
.seg {
  display: inline-flex;
  padding: 3px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
}
.seg--wide {
  width: 100%;
}
.seg__btn {
  flex: 1;
  padding: 5px 12px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.seg__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
@media (width <= 768px) {
  .ssh-form {
    grid-template-columns: 1fr;
  }
  .ssh-form__host,
  .ssh-form__port {
    grid-column: 1 / -1;
  }
}
</style>
