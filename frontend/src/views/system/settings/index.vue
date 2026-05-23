<template>
  <div>
    <el-card shadow="never">
      <el-tabs v-model="activeTab" type="border-card">
        <el-tab-pane label="安全与认证" name="security">
          <div class="setting-module">
            <div class="setting-group">
              <div class="setting-group-header">登录设置</div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">注册功能</span>
                  <span class="setting-item-desc">开启后用户可通过注册页面创建新账号</span>
                </div>
                <el-switch
                  v-model="form.registerEnabled"
                  :disabled="!canEdit"
                  inline-prompt
                  active-text="开启"
                  inactive-text="关闭"
                />
              </div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">用户名登录</span>
                  <span class="setting-item-desc">使用用户名和密码登录系统</span>
                </div>
                <el-switch
                  v-model="form.usernameLoginEnabled"
                  :disabled="!canEdit"
                  inline-prompt
                  active-text="开启"
                  inactive-text="关闭"
                />
              </div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">邮箱登录</span>
                  <span class="setting-item-desc">使用邮箱和验证码登录系统</span>
                </div>
                <el-switch
                  v-model="form.emailLoginEnabled"
                  :disabled="!canEdit"
                  inline-prompt
                  active-text="开启"
                  inactive-text="关闭"
                />
              </div>
            </div>

            <div class="setting-footer">
              <el-button
                type="primary"
                :loading="saving"
                :disabled="!canEdit"
                @click="handleSave"
              >
                保存配置
              </el-button>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { getLoginConfig, updateLoginConfig } from '@/api/login'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'

defineOptions({ name: 'SystemSettingsView' })

const activeTab = ref('security')
const saving = ref(false)

const canEdit = useButtonPermission([PERM.SYSTEM_CONFIG_EDIT], [])

const form = reactive({
  registerEnabled: false,
  usernameLoginEnabled: true,
  emailLoginEnabled: false,
})

const loadConfig = async () => {
  try {
    const { data } = await getLoginConfig()
    if (data?.config) {
      form.registerEnabled = data.config.registerEnabled
      form.usernameLoginEnabled = data.config.usernameLoginEnabled
      form.emailLoginEnabled = data.config.emailLoginEnabled
    }
  } catch {
    ElMessage.error('获取登录配置失败')
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    await updateLoginConfig({
      registerEnabled: form.registerEnabled,
      usernameLoginEnabled: form.usernameLoginEnabled,
      emailLoginEnabled: form.emailLoginEnabled,
    })
    ElMessage.success('保存成功')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped lang="scss">
.setting-module {
  & + .setting-module {
    margin-top: 1.5rem;
  }
}

.setting-group {
  display: flex;
  flex-direction: column;
  gap: 1px;
  background: var(--el-border-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 0.5rem;
  overflow: hidden;
}

.setting-group-header {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  padding: 0.5rem 0.75rem;
  background: var(--el-bg-color-overlay);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.45rem 0.75rem;
  background: var(--el-bg-color-overlay);
}

.setting-item-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 0;
  flex: 1;
  margin-right: 1rem;
}

.setting-item-label {
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
}

.setting-item-desc {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}

.setting-footer {
  margin-top: 1rem;
}
</style>
