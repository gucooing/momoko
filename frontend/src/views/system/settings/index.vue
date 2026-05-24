<template>
  <div>
    <el-card shadow="never">
      <el-tabs v-model="activeTab" type="border-card" @tab-change="onTabChange">
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
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">注册邮箱验证</span>
                  <span class="setting-item-desc">开启后注册时需通过邮箱验证码完成验证</span>
                </div>
                <el-switch
                  v-model="loginForm.registerEmailVerificationRequired"
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

          <div class="setting-module">
            <div class="setting-group">
              <div class="setting-group-header">模板配置</div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">模板类型</span>
                  <span class="setting-item-desc">选择要编辑的邮件模板</span>
                </div>
                <el-select
                  v-model="templateType"
                  :disabled="!canEdit"
                  style="width: 200px"
                  @change="handleTemplateTypeChange"
                >
                  <el-option
                    v-for="tpl in templateTypeOptions"
                    :key="tpl.value"
                    :label="tpl.label"
                    :value="tpl.value"
                  />
                </el-select>
              </div>
              <div class="setting-item">
                <div class="setting-item-info">
                  <span class="setting-item-label">邮件主题</span>
                  <span class="setting-item-desc"><span v-pre>支持 Go text/template 语法，如 {{.name}}</span></span>
                </div>
                <el-input
                  v-model="templateForm.subject"
                  :disabled="!canEdit"
                  placeholder="邮件主题模板"
                  style="width: 400px"
                  @focus="onInputFocus"
                />
              </div>
              <div class="setting-item setting-item-vertical">
                <div class="setting-item-info">
                  <span class="setting-item-label">邮件内容</span>
                  <span class="setting-item-desc"><span v-pre>支持 Go html/template 语法，如 {{.email}}、{{.code}}</span></span>
                </div>
                <el-input
                  v-model="templateForm.template"
                  :disabled="!canEdit"
                  type="textarea"
                  :rows="12"
                  placeholder="<html><body>...</body></html>"
                  @focus="onInputFocus"
                />
              </div>
              <div class="placeholder-strip">
                <span class="placeholder-strip-label">快捷插入：</span>
                <el-tag
                  v-for="p in placeholders"
                  :key="p"
                  size="small"
                  class="placeholder-tag"
                  :class="{ 'tag-disabled': !canEdit }"
                  @click="canEdit && insertPlaceholder(p)"
                >
                  {{ p }}
                </el-tag>
              </div>
            </div>

            <div class="setting-footer">
              <el-button
                type="primary"
                :loading="templateSaving"
                :disabled="!canEdit"
                @click="handleTemplateSave"
              >
                保存模板
              </el-button>
              <el-button
                :loading="templateTesting"
                :disabled="!canEdit"
                @click="openTemplateTestDialog"
              >
                测试发送
              </el-button>
              <el-button
                :disabled="!canEdit"
                @click="openPreviewDialog"
              >
                预览
              </el-button>
            </div>
          </div>

          <!-- 邮件服务测试对话框 -->
          <BaseDialog v-model="testDialogVisible" title="测试邮件发送" width="420">
            <el-form label-position="top">
              <el-form-item label="收件邮箱">
                <el-input v-model="testRecipient" placeholder="输入收件邮箱地址" />
              </el-form-item>
            </el-form>
            <template #footer>
              <el-button @click="testDialogVisible = false">取消</el-button>
              <el-button type="primary" :loading="emailTesting" :disabled="!testRecipient" @click="handleEmailTest">发送</el-button>
            </template>
          </BaseDialog>

          <!-- 模板测试发送对话框 -->
          <BaseDialog v-model="templateTestDialogVisible" title="测试模板邮件发送" width="420">
            <el-form label-position="top">
              <el-form-item label="收件邮箱">
                <el-input v-model="templateTestRecipient" placeholder="输入收件邮箱地址" />
              </el-form-item>
              <el-form-item v-for="field in templateTestFields" :key="field.name" :label="'{{.' + field.name + '}}'">
                <el-input v-model="field.value" :placeholder="`输入 {{.${field.name}}} 的值`" />
              </el-form-item>
            </el-form>
            <template #footer>
              <el-button @click="templateTestDialogVisible = false">取消</el-button>
              <el-button type="primary" :loading="templateTesting" :disabled="!templateTestRecipient" @click="handleTemplateTest">发送</el-button>
            </template>
          </BaseDialog>

          <!-- 模板预览对话框 -->
          <BaseDialog v-model="previewDialogVisible" title="邮件模板预览" width="750" resizable>
            <el-scrollbar max-height="60vh">
              <div class="preview-subject">
                <span class="preview-label">主题：</span>
                <span>{{ renderedPreviewSubject }}</span>
              </div>
              <div class="preview-divider" />
              <iframe :srcdoc="renderedPreviewBody" class="preview-iframe" sandbox="allow-same-origin" />
            </el-scrollbar>
            <template #footer>
              <el-button @click="previewDialogVisible = false">关闭</el-button>
            </template>
          </BaseDialog>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { getLoginConfig, updateLoginConfig } from '@/api/login'
import { getEmailConfig, updateEmailConfig, testEmailConfig, updateEmailTemplate, getEmailTemplate } from '@/api/system'
import { EmailTemplateType } from '@/types/v1/system'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import BaseDialog from '@/components/dialog/BaseDialog.vue'

defineOptions({ name: 'SystemSettingsView' })

const activeTab = ref('security')
const loginSaving = ref(false)
const emailSaving = ref(false)
const emailTesting = ref(false)
const testRecipient = ref('')
const testDialogVisible = ref(false)

const templateSaving = ref(false)
const templateTesting = ref(false)
const templateTestDialogVisible = ref(false)
const templateTestRecipient = ref('')
const previewDialogVisible = ref(false)

const canEdit = useButtonPermission([PERM.SYSTEM_CONFIG_EDIT], [])

const placeholders = ['{{.name}}', '{{.email}}', '{{.code}}']

const lastFocusedEl = ref<HTMLInputElement | HTMLTextAreaElement | null>(null)

const onInputFocus = (e: FocusEvent) => {
  lastFocusedEl.value = e.target as HTMLInputElement | HTMLTextAreaElement
}

const insertPlaceholder = (text: string) => {
  const el = lastFocusedEl.value
  if (!el) return
  const start = el.selectionStart ?? 0
  const end = el.selectionEnd ?? 0

  el.value = el.value.slice(0, start) + text + el.value.slice(end)
  el.setSelectionRange(start + text.length, start + text.length)
  el.focus()
  el.dispatchEvent(new Event('input', { bubbles: true }))
}

const TEMPLATE_TYPE_LABELS: Record<string, string> = {
  [EmailTemplateType.EmailTemplateType_Register]: '注册邮件模板',
  [EmailTemplateType.EmailTemplateType_Login]: '登录邮件模板',
}

const templateTypeOptions = Object.entries(TEMPLATE_TYPE_LABELS).map(([value, label]) => ({
  value,
  label,
}))

const loginForm = reactive({
  registerEnabled: false,
  usernameLoginEnabled: true,
  emailLoginEnabled: false,
  registerEmailVerificationRequired: false,
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

const templateType = ref(EmailTemplateType.EmailTemplateType_Register)
const templateForm = reactive({
  subject: '',
  template: '',
})

interface TemplateField {
  name: string
  value: string
}
const templateTestFields = ref<TemplateField[]>([])

const renderedPreviewSubject = ref('')
const renderedPreviewBody = ref('')

const extractPlaceholders = (content: string): string[] => {
  const regex = /\{\{\.(\w+)\}\}/g
  const names = new Set<string>()
  let match: RegExpExecArray | null
  while ((match = regex.exec(content)) !== null) {
    if (match[1]) names.add(match[1])
  }
  return Array.from(names)
}

const renderTemplate = (source: string, data: Record<string, string>): string => {
  return source.replace(/\{\{\.(\w+)\}\}/g, (_, name) => {
    return data[name] ?? `{{.${name}}}`
  })
}

const highlightPlaceholders = (source: string): string => {
  return source.replace(/\{\{\.(\w+)\}\}/g, (_, name) => {
    return `<span style="background:#ffe58f;padding:0 3px;border-radius:2px;font-family:monospace;">${name}</span>`
  })
}

const handleTemplateTypeChange = async () => {
  try {
    const { data } = await getEmailTemplate({ type: templateType.value as EmailTemplateType })
    if (data?.template) {
      templateForm.subject = data.template.subject
      templateForm.template = data.template.template
    } else {
      templateForm.subject = ''
      templateForm.template = ''
    }
  } catch {
    templateForm.subject = ''
    templateForm.template = ''
  }
}

const handleTemplateSave = async () => {
  templateSaving.value = true
  try {
    await updateEmailTemplate({
      type: templateType.value as EmailTemplateType,
      subject: templateForm.subject,
      template: templateForm.template,
    })
    ElMessage.success('保存成功')
  }finally {
    templateSaving.value = false
  }
}

const openTemplateTestDialog = () => {
  const placeholders = extractPlaceholders(
    templateForm.subject + ' ' + templateForm.template
  )
  templateTestFields.value = placeholders.map((name) => ({ name, value: '' }))
  templateTestRecipient.value = ''
  templateTestDialogVisible.value = true
}

const handleTemplateTest = async () => {
  const data: Record<string, string> = {}
  for (const field of templateTestFields.value) {
    if (field.value) {
      data[field.name] = field.value
    }
  }

  templateTesting.value = true
  try {
    await testEmailConfig({
      recipient: templateTestRecipient.value,
      config:  undefined, // 使用全局邮件配置
      messages: {
        subject: templateForm.subject,
        template: templateForm.template,
        type: templateType.value as EmailTemplateType,
      },
      Data: data,
    })
    ElMessage.success('测试邮件发送成功')
    templateTestDialogVisible.value = false
  } catch {
    ElMessage.error('测试邮件发送失败')
  } finally {
    templateTesting.value = false
  }
}

const openPreviewDialog = () => {
  const subjectPlaceholders = extractPlaceholders(templateForm.subject)
  const bodyPlaceholders = extractPlaceholders(templateForm.template)
  const data: Record<string, string> = {}
  for (const field of templateTestFields.value) {
    data[field.name] = field.value
  }
  for (const name of [...subjectPlaceholders, ...bodyPlaceholders]) {
    if (!(name in data)) {
      data[name] = ''
    }
  }
  renderedPreviewSubject.value = renderTemplate(templateForm.subject, data)
  renderedPreviewBody.value = highlightPlaceholders(templateForm.template)
  previewDialogVisible.value = true
}

const loadLoginConfig = async () => {
  try {
    const { data } = await getLoginConfig()
    if (data?.config) {
      loginForm.registerEnabled = data.config.registerEnabled
      loginForm.usernameLoginEnabled = data.config.usernameLoginEnabled
      loginForm.emailLoginEnabled = data.config.emailLoginEnabled
      loginForm.registerEmailVerificationRequired = data.config.registerEmailVerificationRequired
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
      registerEmailVerificationRequired: loginForm.registerEmailVerificationRequired,
    })
    ElMessage.success('保存成功')
  }finally {
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
  }finally {
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
      Data: {}, // 使用全局邮件模板测试发送
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

const loadedTabs = ref(new Set<string>())

const onTabChange = (name: string | number) => {
  const tab = String(name)
  if (loadedTabs.value.has(tab)) return
  loadedTabs.value.add(tab)
  if (tab === 'security') {
    loadLoginConfig()
  } else if (tab === 'email') {
    loadEmailConfig()
    handleTemplateTypeChange()
  }
}

onMounted(() => {
  onTabChange(activeTab.value)
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

.setting-item-vertical {
  flex-direction: column;
  align-items: flex-start;
  gap: 0.5rem;

  .setting-item-info {
    margin-right: 0;
  }
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

.preview-subject {
  font-size: 0.9rem;
  color: var(--el-text-color-primary);
  padding: 0.5rem 0;
}

.preview-label {
  font-weight: 600;
}

.preview-divider {
  height: 1px;
  background: var(--el-border-color-lighter);
  margin: 0.5rem 0;
}

.preview-iframe {
  width: 100%;
  height: 400px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}

.placeholder-strip {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  flex-wrap: wrap;
  padding: 0.35rem 0.75rem;
  background: var(--el-bg-color-overlay);
  border-top: 1px solid var(--el-border-color-lighter);
}

.placeholder-strip-label {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  margin-right: 0.15rem;
  white-space: nowrap;
}

.placeholder-tag {
  cursor: pointer;
  user-select: none;

  &:hover { opacity: 0.8; }
  &.tag-disabled { cursor: not-allowed; opacity: 0.6; }
}

/* mobile */
@media (width <= 768px) {
  .setting-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
    padding: 0.6rem 0.75rem;
  }

  .setting-item-info {
    margin-right: 0;
  }

  .setting-item :deep(.el-input),
  .setting-item :deep(.el-input-number),
  .setting-item :deep(.el-select) {
    width: 100% !important;
  }

  .setting-footer {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .setting-footer :deep(.el-button) {
    flex: 1;
    min-width: 0;
  }

  .preview-iframe {
    height: 260px;
  }
}
</style>
