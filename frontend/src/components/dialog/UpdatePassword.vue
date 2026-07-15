<!-- 修改密码：FormDialog + 令牌字段 + 内联校验（替代 BaseDialog/el-form） -->
<template>
  <FormDialog
    v-model="open"
    :title="t('password.title')"
    :width="480"
    :loading="submitting"
    @confirm="submit"
    @close="reset"
  >
    <div class="pwd-form">
      <div class="app-field">
        <label class="app-label" for="pwd-old">{{ t('password.oldPassword') }}</label>
        <div class="pwd-wrap">
          <input
            id="pwd-old"
            v-model.trim="passwordForm.oldPassword"
            class="app-input pwd-input"
            :class="{ 'is-error': errors.oldPassword }"
            :type="show.old ? 'text' : 'password'"
            :placeholder="t('password.oldPasswordPlaceholder')"
            autocomplete="current-password"
          />
          <button type="button" class="pwd-eye" @click="show.old = !show.old">
            <component
              :is="menuStore.iconComponents[show.old ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon']"
            />
          </button>
        </div>
        <span v-if="errors.oldPassword" class="app-field__error">{{ errors.oldPassword }}</span>
      </div>

      <div class="app-field">
        <label class="app-label" for="pwd-new">{{ t('password.newPassword') }}</label>
        <div class="pwd-wrap">
          <input
            id="pwd-new"
            v-model.trim="passwordForm.newPassword"
            class="app-input pwd-input"
            :class="{ 'is-error': errors.newPassword }"
            :type="show.next ? 'text' : 'password'"
            :placeholder="t('password.newPasswordPlaceholder')"
            autocomplete="new-password"
          />
          <button type="button" class="pwd-eye" @click="show.next = !show.next">
            <component
              :is="menuStore.iconComponents[show.next ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon']"
            />
          </button>
        </div>
        <span v-if="errors.newPassword" class="app-field__error">{{ errors.newPassword }}</span>
      </div>

      <div class="app-field">
        <label class="app-label" for="pwd-confirm">{{ t('password.confirmPassword') }}</label>
        <div class="pwd-wrap">
          <input
            id="pwd-confirm"
            v-model.trim="passwordForm.confirmPassword"
            class="app-input pwd-input"
            :class="{ 'is-error': errors.confirmPassword }"
            :type="show.confirm ? 'text' : 'password'"
            :placeholder="t('password.confirmPasswordPlaceholder')"
            autocomplete="new-password"
          />
          <button type="button" class="pwd-eye" @click="show.confirm = !show.confirm">
            <component
              :is="
                menuStore.iconComponents[show.confirm ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon']
              "
            />
          </button>
        </div>
        <span v-if="errors.confirmPassword" class="app-field__error">{{
          errors.confirmPassword
        }}</span>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import type { UserPasswordFormValue } from '@/stores/user/types'
import { getRequestErrorMessage } from '@/utils/request'
import { useFeedback } from '@/utils/feedback'
import { useI18n } from 'vue-i18n'

const userStore = useUserStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const fb = useFeedback()

const open = ref(false)
const submitting = ref(false)
const show = reactive({ old: false, next: false, confirm: false })

const createDefaultPasswordForm = (): UserPasswordFormValue => ({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const passwordForm = ref<UserPasswordFormValue>(createDefaultPasswordForm())
const errors = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const reset = () => {
  passwordForm.value = createDefaultPasswordForm()
  errors.oldPassword = ''
  errors.newPassword = ''
  errors.confirmPassword = ''
  show.old = false
  show.next = false
  show.confirm = false
}

const validate = () => {
  let ok = true
  errors.oldPassword = ''
  errors.newPassword = ''
  errors.confirmPassword = ''

  if (!passwordForm.value.oldPassword.trim()) {
    errors.oldPassword = t('password.oldPasswordPlaceholder')
    ok = false
  }
  const next = passwordForm.value.newPassword
  if (!next.trim()) {
    errors.newPassword = t('password.newPasswordRequired')
    ok = false
  } else if (next.length < 6) {
    errors.newPassword = t('password.newPasswordMin')
    ok = false
  }
  const confirm = passwordForm.value.confirmPassword
  if (!confirm.trim()) {
    errors.confirmPassword = t('password.confirmPasswordRequired')
    ok = false
  } else if (confirm !== next) {
    errors.confirmPassword = t('password.confirmPasswordMismatch')
    ok = false
  }
  return ok
}

const submit = async () => {
  if (!validate()) return
  submitting.value = true
  try {
    await userStore.updatePassword(passwordForm.value)
    open.value = false
    reset()
  } catch (error) {
    fb.error(getRequestErrorMessage(error, t('password.updateFailed')))
  } finally {
    submitting.value = false
  }
}

const showDialog = () => {
  reset()
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.pwd-form {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}
.pwd-wrap {
  position: relative;
  display: flex;
  align-items: center;
}
.pwd-input {
  width: 100%;
  padding-inline-end: 2.25rem;
}
.pwd-input.is-error {
  border-color: var(--el-color-danger);
}
.pwd-eye {
  position: absolute;
  right: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  :deep(svg) {
    width: 16px;
    height: 16px;
  }
}
.pwd-eye:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.app-field__error {
  display: block;
  margin-top: 0.25rem;
  font-size: 0.75rem;
  color: var(--el-color-danger);
}
</style>
