<template>
  <div class="form-content-inner">
    <h2 class="title">创建账号</h2>
    <p class="subtitle">加入 {{ APP_CONFIG.name }}，开始您的管理之旅</p>

    <el-form :model="registerForm" label-position="top" class="register-form">
      <el-form-item>
        <el-input v-model="registerForm.username" placeholder="设置用户名" />
      </el-form-item>
      <el-form-item>
        <el-input v-model="registerForm.email" placeholder="输入电子邮箱" />
      </el-form-item>
      <el-form-item>
        <el-input
          v-model="registerForm.password"
          type="password"
          show-password
          placeholder="设置登录密码"
        />
      </el-form-item>
      <el-form-item>
        <el-input
          v-model="registerForm.confirmPassword"
          type="password"
          show-password
          placeholder="确认您的密码"
        />
      </el-form-item>
      <el-button type="primary" class="submit-btn" @click="handleRegister"> 立即注册 </el-button>
      <div class="back-link">
        <span class="have-account">已有账号？</span>
        <el-link :underline="false" @click="emits('goToMode', 'login')">
          <el-icon><component :is="menuStore.iconComponents['Element:ArrowLeft']" /></el-icon>
          返回登录
        </el-link>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { APP_CONFIG } from '@/config/app.config'
import { ElMessage } from 'element-plus'

defineOptions({ name: 'RegisterComponent' })

const emits = defineEmits<{
  (e: 'goToMode', mode: 'login' | 'forgot' | 'register'): void
}>()
const menuStore = useMenuStore()

const registerForm = ref({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
})

const handleRegister = () => {
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

  .register-form {
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
      gap: 0.5rem;

      .have-account {
        font-size: 0.875rem;
        color: var(--el-text-color-secondary);
      }

      .el-link {
        font-size: 0.9rem;
        font-weight: 600;
        transition: all 0.3s;
        color: var(--el-text-color-secondary);

        &:hover {
          color: var(--el-color-primary);
          transform: translateX(-4px);
        }
      }
    }
  }
}
</style>
