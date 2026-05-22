<template>
  <div class="form-content-inner">
    <h2 class="title">找回密码</h2>
    <p class="subtitle">请输入您的邮箱地址来重置密码</p>

    <el-form :model="forgotPasswordForm" label-position="top" class="forgot-password-form">
      <el-form-item>
        <el-input v-model="forgotPasswordForm.email" placeholder="请输入注册时的邮箱地址" />
      </el-form-item>
      <el-button type="primary" class="submit-btn" @click="handleAction"> 发送重置链接 </el-button>
      <div class="back-link">
        <el-link :underline="false" @click="emits('goToMode', 'login')">
          <el-icon><component :is="menuStore.iconComponents['Element:ArrowLeft']" /></el-icon>
          返回登录
        </el-link>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'

const emits = defineEmits<{
  (e: 'goToMode', mode: 'login' | 'forgot' | 'register'): void
}>()
const menuStore = useMenuStore()

const forgotPasswordForm = ref({
  email: '',
})

const handleAction = () => {
  ElMessage.success('敬请期待👀')
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
