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
                  v-model="loginForm.registerEnabled"
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
                  v-model="loginForm.usernameLoginEnabled"
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
                  v-model="loginForm.emailLoginEnabled"
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
                :loading="loginSaving"
                :disabled="!canEdit"
                @click="handleLoginSave"
              >
                保存配置
              </el-button>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="邮件配置" name="email">
          <div class="setting-module">
            <div class="setting-group">
              <div class="setting-group-header">邮件服务</div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">启用邮件服务</span>
                  <span class="setting-item-desc">开启后系统将使用下方配置发送邮件</span>
                </div>
                <el-switch
                  v-model="emailForm.enabled"
                  :disabled="!canEdit"
                  inline-prompt
                  active-text="开启"
                  inactive-text="关闭"
                />
              </div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">SMTP 服务地址</span>
                  <span class="setting-item-desc">邮件服务器的地址，如 smtp.example.com</span>
                </div>
                <el-input
                  v-model="emailForm.host"
                  :disabled="!canEdit"
                  placeholder="smtp.example.com"
                  style="width: 260px"
                />
              </div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">SMTP 端口</span>
                  <span class="setting-item-desc">TLS 默认 465，非 TLS 默认 25</span>
                </div>
                <el-input-number
                  v-model="emailForm.port"
                  :disabled="!canEdit"
                  :min="1"
                  :max="65535"
                  style="width: 160px"
                />
              </div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">SMTP 用户名</span>
                  <span class="setting-item-desc">邮箱账号，通常为邮箱地址</span>
                </div>
                <el-input
                  v-model="emailForm.username"
                  :disabled="!canEdit"
                  placeholder="user@example.com"
                  style="width: 260px"
                />
              </div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">SMTP 密码</span>
                  <span class="setting-item-desc">邮箱授权码或登录密码</span>
                </div>
                <el-input
                  v-model="emailForm.password"
                  :disabled="!canEdit"
                  type="password"
                  show-password
                  placeholder="授权码或密码"
                  style="width: 260px"
                />
              </div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">发件邮箱</span>
                  <span class="setting-item-desc">显示在邮件发件人栏的邮箱地址</span>
                </div>
                <el-input
                  v-model="emailForm.from"
                  :disabled="!canEdit"
                  placeholder="noreply@example.com"
                  style="width: 260px"
                />
              </div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">发件人名称</span>
                  <span class="setting-item-desc">显示在邮件发件人栏的名称</span>
                </div>
                <el-input
                  v-model="emailForm.fromName"
                  :disabled="!canEdit"
                  placeholder="Momoko"
                  style="width: 260px"
                />
              </div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">使用 TLS</span>
                  <span class="setting-item-desc">开启后将使用 TLS 加密连接 SMTP 服务器</span>
                </div>
                <el-switch
                  v-model="emailForm.useTls"
                  :disabled="!canEdit"
                  inline-prompt
                  active-text="开启"
                  inactive-text="关闭"
                />
              </div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">连接超时（秒）</span>
                  <span class="setting-item-desc">连接 SMTP 服务器的超时时间，默认 10 秒</span>
                </div>
                <el-input-number
                  v-model="emailForm.timeoutSeconds"
                  :disabled="!canEdit"
                  :min="1"
                  :max="60"
                  style="width: 160px"
                />
              </div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">并发发送数</span>
                  <span class="setting-item-desc">同时发送邮件的协程数量，默认 5</span>
                </div>
                <el-input-number
                  v-model="emailForm.ccsN"
                  :disabled="!canEdit"
                  :min="1"
                  :max="50"
                  style="width: 160px"
                />
              </div>
            </div>

            <div class="setting-footer">
              <el-button
                type="primary"
                :loading="emailSaving"
                :disabled="!canEdit"
                @click="handleEmailSave"
              >
                保存配置
              </el-button>
              <el-button
                :loading="emailTesting"
                :disabled="!canEdit"
                @click="testDialogVisible = true"
              >
                测试邮件
              </el-button>
            </div>
          </div>

          <el-dialog
            v-model="testDialogVisible"
            title="测试邮件发送"
            width="420px"
            :close-on-click-modal="false"
          >
            <el-form label-position="top">
              <el-form-item label="收件邮箱">
                <el-input
                  v-model="testRecipient"
                  placeholder="输入收件邮箱地址"
                />
              </el-form-item>
            </el-form>
            <template #footer>
              <el-button @click="testDialogVisible = false">取消</el-button>
              <el-button
                type="primary"
                :loading="emailTesting"
                :disabled="!testRecipient"
                @click="handleEmailTest"
              >
                发送
              </el-button>
            </template>
          </el-dialog>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { getLoginConfig, updateLoginConfig } from '@/api/login'
import { getEmailConfig, updateEmailConfig, testEmailConfig } from '@/api/system'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'

defineOptions({ name: 'SystemSettingsView' })

const activeTab = ref('security')
const loginSaving = ref(false)
const emailSaving = ref(false)
const emailTesting = ref(false)
const testRecipient = ref('')
const testDialogVisible = ref(false)

const canEdit = useButtonPermission([PERM.SYSTEM_CONFIG_EDIT], [])

const loginForm = reactive({
  registerEnabled: false,
  usernameLoginEnabled: true,
  emailLoginEnabled: false,
})

const emailForm = reactive({
  enabled: false,
  host: '',
  port: 465,
  username: '',
  password: '',
  from: '',
  fromName: '',
  useTls: true,
  timeoutSeconds: 10,
  ccsN: 5,
})

const loadLoginConfig = async () => {
  try {
    const { data } = await getLoginConfig()
    if (data?.config) {
      loginForm.registerEnabled = data.config.registerEnabled
      loginForm.usernameLoginEnabled = data.config.usernameLoginEnabled
      loginForm.emailLoginEnabled = data.config.emailLoginEnabled
    }
  } catch {
    ElMessage.error('获取登录配置失败')
  }
}

const loadEmailConfig = async () => {
  try {
    const { data } = await getEmailConfig()
    if (data?.config) {
      emailForm.enabled = data.config.enabled
      emailForm.host = data.config.host
      emailForm.port = data.config.port
      emailForm.username = data.config.username
      emailForm.password = data.config.password
      emailForm.from = data.config.from
      emailForm.fromName = data.config.fromName
      emailForm.useTls = data.config.useTls
      emailForm.timeoutSeconds = data.config.timeoutSeconds
      emailForm.ccsN = data.config.ccsN
    }
  } catch {
    ElMessage.error('获取邮件配置失败')
  }
}

const handleLoginSave = async () => {
  loginSaving.value = true
  try {
    await updateLoginConfig({
      registerEnabled: loginForm.registerEnabled,
      usernameLoginEnabled: loginForm.usernameLoginEnabled,
      emailLoginEnabled: loginForm.emailLoginEnabled,
    })
    ElMessage.success('保存成功')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    loginSaving.value = false
  }
}

const handleEmailSave = async () => {
  emailSaving.value = true
  try {
    await updateEmailConfig({
      enabled: emailForm.enabled,
      host: emailForm.host,
      port: emailForm.port,
      username: emailForm.username,
      password: emailForm.password,
      from: emailForm.from,
      fromName: emailForm.fromName,
      useTls: emailForm.useTls,
      timeoutSeconds: emailForm.timeoutSeconds,
      ccsN: emailForm.ccsN,
    })
    ElMessage.success('保存成功')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    emailSaving.value = false
  }
}

const handleEmailTest = async () => {
  emailTesting.value = true
  try {
    await testEmailConfig({
      recipient: testRecipient.value,
      config: {
        enabled: emailForm.enabled,
        host: emailForm.host,
        port: emailForm.port,
        username: emailForm.username,
        password: emailForm.password,
        from: emailForm.from,
        fromName: emailForm.fromName,
        useTls: emailForm.useTls,
        timeoutSeconds: emailForm.timeoutSeconds,
        ccsN: emailForm.ccsN,
      },
    })
    ElMessage.success('测试邮件发送成功')
    testDialogVisible.value = false
    testRecipient.value = ''
  } catch {
    ElMessage.error('测试邮件发送失败')
  } finally {
    emailTesting.value = false
  }
}

onMounted(() => {
  loadLoginConfig()
  loadEmailConfig()
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
