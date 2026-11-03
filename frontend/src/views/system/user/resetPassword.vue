<!-- 管理员重置用户密码：新密码 + 确认 + 随机生成/复制；保存后该用户所有会话下线（后端 ResetUserPassword）。 -->
<template>
  <FormDialog
    v-model="open"
    :title="t('accessAdmin.reset.title', { name: target.name })"
    :width="480"
    :loading="submitting"
    @confirm="confirm"
    @close="onClose"
  >
    <div class="rp-form">
      <div class="rp-who">
        <span class="m-sticker m-sticker--p rp-who__av">{{ initial }}</span>
        <div class="rp-who__text">
          <strong>{{ target.name }}</strong>
          <span>{{ target.username }}</span>
        </div>
      </div>

      <div class="app-field">
        <label class="app-label app-label--required">{{ t('accessAdmin.reset.newPassword') }}</label>
        <div class="rp-row">
          <input
            v-model="form.password"
            :type="reveal ? 'text' : 'password'"
            class="app-input m-mono"
            :class="{ 'is-error': errors.password }"
            :placeholder="t('accessAdmin.reset.placeholder')"
            autocomplete="new-password"
          />
          <UButton color="neutral" variant="soft" icon="i-lucide-sparkles" @click="generate">
            {{ t('accessAdmin.reset.generate') }}
          </UButton>
          <UButton
            v-if="reveal && form.password"
            color="neutral"
            variant="soft"
            icon="i-lucide-copy"
            :aria-label="t('accessAdmin.reset.copy')"
            @click="copy"
          />
        </div>
        <span v-if="errors.password" class="app-field__error">{{ errors.password }}</span>
      </div>

      <div class="app-field">
        <label class="app-label app-label--required">{{ t('accessAdmin.reset.confirmPassword') }}</label>
        <input
          v-model="form.confirm"
          :type="reveal ? 'text' : 'password'"
          class="app-input m-mono"
          :class="{ 'is-error': errors.confirm }"
          :placeholder="t('accessAdmin.reset.confirmPlaceholder')"
          autocomplete="new-password"
        />
        <span v-if="errors.confirm" class="app-field__error">{{ errors.confirm }}</span>
      </div>

      <p class="rp-hint">
        <component :is="menuStore.iconComponents['HOutline:InformationCircleIcon']" />
        {{ t('accessAdmin.reset.hint') }}
      </p>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { resetUserPassword } from '@/api/user'
import { initialOf } from '@/views/system/components/accessTone'

defineOptions({ name: 'UserResetPassword' })

const { t } = useI18n()
const menuStore = useMenuStore()

const open = ref(false)
const submitting = ref(false)
const reveal = ref(false)
const target = ref({ userId: '', name: '', username: '' })
const form = ref({ password: '', confirm: '' })
const errors = ref<Record<string, string>>({})
const initial = computed(() => initialOf(target.value.name || target.value.username))

const generate = () => {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789!@#%'
  const buf = new Uint32Array(14)
  crypto.getRandomValues(buf)
  const pwd = Array.from(buf, (n) => chars[n % chars.length]).join('')
  form.value = { password: pwd, confirm: pwd }
  reveal.value = true
  errors.value = {}
}

const copy = async () => {
  try {
    await navigator.clipboard.writeText(form.value.password)
    feedback.success(t('accessAdmin.reset.copied'))
  } catch {
    /* 剪贴板不可用时忽略 */
  }
}

const validate = () => {
  const e: Record<string, string> = {}
  if (form.value.password.length < 8) e.password = t('accessAdmin.reset.tooShort')
  if (form.value.confirm !== form.value.password) e.confirm = t('accessAdmin.reset.mismatch')
  errors.value = e
  return !Object.keys(e).length
}

const confirm = async () => {
  if (!validate()) return
  submitting.value = true
  try {
    await resetUserPassword({ userId: target.value.userId, password: form.value.password })
    feedback.success(t('accessAdmin.reset.success'))
    open.value = false
    onClose()
  } finally {
    submitting.value = false
  }
}

const onClose = () => {
  form.value = { password: '', confirm: '' }
  errors.value = {}
  reveal.value = false
}

const showDialog = (user: { userId: string; name?: string; username: string }) => {
  target.value = { userId: user.userId, name: user.name || user.username, username: user.username }
  onClose()
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.rp-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.rp-who {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.rp-who__av {
  width: 40px;
  height: 40px;
  font-size: 1rem;
  transform: rotate(-4deg);
}
.rp-who__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.rp-who__text strong {
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.rp-who__text span {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.rp-row {
  display: flex;
  gap: 8px;
}
.rp-row .app-input {
  flex: 1;
  min-width: 0;
}
.rp-hint {
  display: flex;
  gap: 8px;
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.rp-hint :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
  margin-top: 2px;
}
@media (width < 480px) {
  .rp-row {
    flex-wrap: wrap;
  }
  .rp-row .app-input {
    flex: 1 1 100%;
  }
}
</style>
