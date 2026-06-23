<template>
  <div class="form-content-inner">
    <h2 class="title">{{ t('forgot.title') }}</h2>
    <p class="subtitle">{{ t('forgot.subtitle') }}</p>

    <el-form :model="forgotPasswordForm" label-position="top" class="forgot-password-form">
      <el-form-item>
        <el-input v-model="forgotPasswordForm.email" :placeholder="t('forgot.emailPlaceholder')" />
      </el-form-item>
      <el-button type="primary" class="submit-btn" @click="handleAction">
        {{ t('forgot.submit') }}
      </el-button>
      <div class="back-link">
        <el-link :underline="false" @click="emits('goToMode', 'login')">
          <el-icon><component :is="menuStore.iconComponents['Element:ArrowLeft']" /></el-icon>
          {{ t('forgot.backLogin') }}
        </el-link>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'

const emits = defineEmits<{
  (e: 'goToMode', mode: 'login' | 'forgot' | 'register'): void
}>()
const menuStore = useMenuStore()
const { t } = useI18n()

const forgotPasswordForm = ref({
  email: '',
})

const handleAction = () => {
  ElMessage.success(t('forgot.comingSoon'))
}
</script>

<style scoped lang="scss">
.form-content-inner {
  .title {
    font-size: 1.75rem;
    font-weight: 700;
    color: var(--el-text-color-primary);
    margin-bottom: 0.5rem;
  }

  .subtitle {
    font-size: 0.95rem;
    color: var(--el-text-color-secondary);
    margin-bottom: 2rem;
  }

  .forgot-password-form {
    :deep(.el-input__wrapper),
    :deep(.el-select__wrapper) {
      padding: 0.5rem 1rem;
      border-radius: 0.5rem;
      box-shadow: 0 0 0 1px var(--el-border-color) inset;
      min-height: 2.75rem;

      &.is-focus {
        box-shadow: 0 0 0 1px var(--el-color-primary) inset;
      }
    }

    .submit-btn {
      width: 100%;
      height: 2.75rem;
      border-radius: 0.75rem;
      font-size: 1rem;
      font-weight: 600;
      margin-top: 0.9rem;
      margin-bottom: 1.5rem;
    }

    .back-link {
      display: flex;
      justify-content: center;
      align-items: center;

      .el-link {
        font-size: 0.875rem;
        color: var(--el-text-color-secondary);
        font-weight: 500;
        transition: all 0.3s;

        &:hover {
          color: var(--el-color-primary);
          transform: translateX(-4px);
        }
      }
    }
  }
}
</style>
