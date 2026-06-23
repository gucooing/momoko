<template>
  <BaseDialog v-model="open" :title="t('password.title')" width="500" @confirm="updatePassword">
    <el-form
      ref="passwordFormRef"
      :model="passwordForm"
      :rules="passwordRules"
      label-width="80px"
      class="password-form"
    >
      <el-form-item :label="t('password.oldPassword')" prop="oldPassword">
        <el-input
          v-model.trim="passwordForm.oldPassword"
          type="password"
          :placeholder="t('password.oldPasswordPlaceholder')"
          show-password
          clearable
        />
      </el-form-item>
      <el-form-item :label="t('password.newPassword')" prop="newPassword">
        <el-input
          v-model.trim="passwordForm.newPassword"
          type="password"
          :placeholder="t('password.newPasswordPlaceholder')"
          show-password
          clearable
        />
      </el-form-item>
      <el-form-item :label="t('password.confirmPassword')" prop="confirmPassword">
        <el-input
          v-model.trim="passwordForm.confirmPassword"
          type="password"
          :placeholder="t('password.confirmPasswordPlaceholder')"
          show-password
          clearable
        />
      </el-form-item>
    </el-form>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import type { FormInstance } from 'element-plus'
import type { UserPasswordFormValue } from '@/stores/user/types'
import { showRequestError } from '@/utils/request'
import { useI18n } from 'vue-i18n'

const userStore = useUserStore()
const passwordFormRef = useTemplateRef<FormInstance>('passwordFormRef')
const { t } = useI18n()

const open = ref(false)

// 密码表单
const createDefaultPasswordForm = (): UserPasswordFormValue => ({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const passwordForm = ref<UserPasswordFormValue>(createDefaultPasswordForm())

// 修改密码
const updatePassword = async () => {
  try {
    await passwordFormRef.value?.validate()
    await userStore.updatePassword(passwordForm.value)
    passwordForm.value = createDefaultPasswordForm()
    open.value = false
  } catch (error) {
    showRequestError(error, t('password.updateFailed'))
  }
}

// 新密码验证
/* eslint-disable @typescript-eslint/no-explicit-any */
const validateNewPassword = (rule: any, value: string, callback: any) => {
  if (value.trim() === '') return callback(new Error(t('password.newPasswordRequired')))
  if (value.length < 6) return callback(new Error(t('password.newPasswordMin')))
  callback()
}

// 确认密码验证
/* eslint-disable @typescript-eslint/no-explicit-any */
const validateConfirmPassword = (rule: any, value: string, callback: any) => {
  if (value.trim() === '') return callback(new Error(t('password.confirmPasswordRequired')))
  if (value !== passwordForm.value.newPassword) return callback(new Error(t('password.confirmPasswordMismatch')))
  callback()
}

// rules
const passwordRules = computed(() => ({
  oldPassword: [{ required: true, message: t('password.oldPasswordPlaceholder'), trigger: 'blur' }],
  newPassword: [{ required: true, validator: validateNewPassword, trigger: 'blur' }],
  confirmPassword: [{ required: true, validator: validateConfirmPassword, trigger: 'blur' }],
}))

const showDialog = () => {
  passwordForm.value = createDefaultPasswordForm()
  open.value = true
}

defineExpose({
  showDialog,
})
</script>

<style></style>
