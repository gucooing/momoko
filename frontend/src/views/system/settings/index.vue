<!-- 系统设置（重写 · P3 配置型）：PageHeader + 令牌 Tab 条（安全/邮件）+ AppPanel 分组，每组分区保存。
     控件用 AppSwitch / 令牌 .app-input/.app-textarea / AppSelect；对话框用 FormDialog（测试/模板测试/预览）。
     保留全部 API（login/email/template 配置）、PERM.SYSTEM_CONFIG_EDIT、i18n、占位符插入与预览逻辑。 -->
<template>
  <div class="settings-page">
    <PageHeader :title="t('system.settings.pageTitle')" :description="t('system.settings.pageDesc')" />

    <!-- Tab 条 -->
    <div class="settings-tabs" role="tablist">
      <button
        v-for="tab in TABS"
        :key="tab.name"
        type="button"
        role="tab"
        class="settings-tabs__btn"
        :class="{ 'is-active': activeTab === tab.name }"
        :aria-selected="activeTab === tab.name"
        @click="setTab(tab.name)"
      >
        <component :is="menuStore.iconComponents[tab.icon]" />
        {{ t(tab.labelKey) }}
      </button>
    </div>

    <!-- 安全与认证 -->
    <div v-show="activeTab === 'security'" class="settings-tab">
      <AppPanel :title="t('system.settings.loginSettings')" :padded="false">
        <div class="set-rows">
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.registerFeature') }}</span>
              <span class="set-row__desc">{{ t('system.settings.registerFeatureDesc') }}</span>
            </div>
            <AppSwitch v-model="loginForm.registerEnabled" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.usernameLogin') }}</span>
              <span class="set-row__desc">{{ t('system.settings.usernameLoginDesc') }}</span>
            </div>
            <AppSwitch v-model="loginForm.usernameLoginEnabled" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.emailLogin') }}</span>
              <span class="set-row__desc">{{ t('system.settings.emailLoginDesc') }}</span>
            </div>
            <AppSwitch v-model="loginForm.emailLoginEnabled" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.registerEmailVerification') }}</span>
              <span class="set-row__desc">{{ t('system.settings.registerEmailVerificationDesc') }}</span>
            </div>
            <AppSwitch v-model="loginForm.registerEmailVerificationRequired" :disabled="!canEdit" />
          </div>
        </div>
        <template #footer>
          <UButton color="primary" :loading="loginSaving" :disabled="!canEdit" @click="handleLoginSave">
            {{ t('system.settings.saveConfig') }}
          </UButton>
        </template>
      </AppPanel>
    </div>

    <!-- 邮件配置 -->
    <div v-show="activeTab === 'email'" class="settings-tab">
      <AppPanel :title="t('system.settings.emailService')" :padded="false">
        <div class="set-rows">
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.enableEmailService') }}</span>
              <span class="set-row__desc">{{ t('system.settings.enableEmailServiceDesc') }}</span>
            </div>
            <AppSwitch v-model="emailForm.enabled" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.smtpHost') }}</span>
              <span class="set-row__desc">{{ t('system.settings.smtpHostDesc') }}</span>
            </div>
            <input v-model="emailForm.host" class="app-input set-input" :disabled="!canEdit" placeholder="smtp.example.com" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.smtpPort') }}</span>
              <span class="set-row__desc">{{ t('system.settings.smtpPortDesc') }}</span>
            </div>
            <input v-model.number="emailForm.port" type="number" min="1" max="65535" class="app-input set-num" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.smtpUsername') }}</span>
              <span class="set-row__desc">{{ t('system.settings.smtpUsernameDesc') }}</span>
            </div>
            <input v-model="emailForm.username" class="app-input set-input" :disabled="!canEdit" placeholder="user@example.com" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.smtpPassword') }}</span>
              <span class="set-row__desc">{{ t('system.settings.smtpPasswordDesc') }}</span>
            </div>
            <div class="set-input set-pwd">
              <input
                v-model="emailForm.password"
                :type="showPwd ? 'text' : 'password'"
                class="app-input"
                :disabled="!canEdit"
                :placeholder="t('system.settings.smtpPasswordPlaceholder')"
              />
              <AppIconButton
                :icon="showPwd ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon'"
                :label="t('system.settings.smtpPassword')"
                :box="30"
                @click="showPwd = !showPwd"
              />
            </div>
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.fromEmail') }}</span>
              <span class="set-row__desc">{{ t('system.settings.fromEmailDesc') }}</span>
            </div>
            <input v-model="emailForm.from" class="app-input set-input" :disabled="!canEdit" placeholder="noreply@example.com" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.fromName') }}</span>
              <span class="set-row__desc">{{ t('system.settings.fromNameDesc') }}</span>
            </div>
            <input v-model="emailForm.fromName" class="app-input set-input" :disabled="!canEdit" placeholder="Momoko" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.useTls') }}</span>
              <span class="set-row__desc">{{ t('system.settings.useTlsDesc') }}</span>
            </div>
            <AppSwitch v-model="emailForm.useTls" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.timeoutSeconds') }}</span>
              <span class="set-row__desc">{{ t('system.settings.timeoutSecondsDesc') }}</span>
            </div>
            <input v-model.number="emailForm.timeoutSeconds" type="number" min="1" max="60" class="app-input set-num" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.concurrency') }}</span>
              <span class="set-row__desc">{{ t('system.settings.concurrencyDesc') }}</span>
            </div>
            <input v-model.number="emailForm.ccsN" type="number" min="1" max="50" class="app-input set-num" :disabled="!canEdit" />
          </div>
        </div>
        <template #footer>
          <UButton color="primary" :loading="emailSaving" :disabled="!canEdit" @click="handleEmailSave">
            {{ t('system.settings.saveConfig') }}
          </UButton>
          <UButton color="neutral" variant="soft" :loading="emailTesting" :disabled="!canEdit" @click="testDialogVisible = true">
            {{ t('system.settings.testEmail') }}
          </UButton>
        </template>
      </AppPanel>

      <AppPanel :title="t('system.settings.templateConfig')" :padded="false">
        <div class="set-rows">
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.templateType') }}</span>
              <span class="set-row__desc">{{ t('system.settings.templateTypeDesc') }}</span>
            </div>
            <AppSelect v-model="templateType" :options="templateTypeOptions" :disabled="!canEdit" class="set-input" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.emailSubject') }}</span>
              <span class="set-row__desc">{{ t('system.settings.subjectDesc', subjectDescParams) }}</span>
            </div>
            <input
              v-model="templateForm.subject"
              class="app-input set-wide"
              :disabled="!canEdit"
              :placeholder="t('system.settings.subjectPlaceholder')"
              @focus="onInputFocus"
            />
          </div>
          <div class="set-row set-row--col">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('system.settings.emailContent') }}</span>
              <span class="set-row__desc">{{ t('system.settings.contentDesc', contentDescParams) }}</span>
            </div>
            <textarea
              v-model="templateForm.template"
              class="app-textarea set-code"
              :disabled="!canEdit"
              rows="12"
              placeholder="<html><body>...</body></html>"
              @focus="onInputFocus"
            />
          </div>
          <div class="set-row set-row--chips">
            <span class="set-chips__label">{{ t('system.settings.quickInsert') }}</span>
            <button
              v-for="p in placeholders"
              :key="p"
              type="button"
              class="set-chip"
              :disabled="!canEdit"
              @click="insertPlaceholder(p)"
            >
              {{ p }}
            </button>
          </div>
        </div>
        <template #footer>
          <UButton color="primary" :loading="templateSaving" :disabled="!canEdit" @click="handleTemplateSave">
            {{ t('system.settings.saveTemplate') }}
          </UButton>
          <UButton color="neutral" variant="soft" :loading="templateTesting" :disabled="!canEdit" @click="openTemplateTestDialog">
            {{ t('system.settings.testSend') }}
          </UButton>
          <UButton color="neutral" variant="ghost" :disabled="!canEdit" @click="openPreviewDialog">
            {{ t('system.settings.preview') }}
          </UButton>
        </template>
      </AppPanel>
    </div>

    <!-- 邮件服务测试 -->
    <FormDialog v-model="testDialogVisible" :title="t('system.settings.testEmailTitle')" :width="440">
      <div class="app-field">
        <label class="app-label">{{ t('system.settings.recipientEmail') }}</label>
        <input v-model="testRecipient" class="app-input" :placeholder="t('system.settings.recipientEmailPlaceholder')" />
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('system.common.cancel') }}</UButton>
        <UButton color="primary" :loading="emailTesting" :disabled="!testRecipient" @click="handleEmailTest">
          {{ t('system.settings.send') }}
        </UButton>
      </template>
    </FormDialog>

    <!-- 模板测试发送 -->
    <FormDialog v-model="templateTestDialogVisible" :title="t('system.settings.testTemplateEmailTitle')" :width="440">
      <div class="settings-form">
        <div class="app-field">
          <label class="app-label">{{ t('system.settings.recipientEmail') }}</label>
          <input v-model="templateTestRecipient" class="app-input" :placeholder="t('system.settings.recipientEmailPlaceholder')" />
        </div>
        <div v-for="field in templateTestFields" :key="field.name" class="app-field">
          <label class="app-label">{{ fieldToken(field.name) }}</label>
          <input
            v-model="field.value"
            class="app-input"
            :placeholder="t('system.settings.templateFieldPlaceholder', { placeholder: fieldToken(field.name) })"
          />
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('system.common.cancel') }}</UButton>
        <UButton color="primary" :loading="templateTesting" :disabled="!templateTestRecipient" @click="handleTemplateTest">
          {{ t('system.settings.send') }}
        </UButton>
      </template>
    </FormDialog>

    <!-- 模板预览 -->
    <FormDialog v-model="previewDialogVisible" :title="t('system.settings.templatePreviewTitle')" :width="760">
      <div class="preview">
        <div class="preview__subject">
          <span class="preview__label">{{ t('system.settings.subjectLabel') }}</span>
          <span>{{ renderedPreviewSubject }}</span>
        </div>
        <div class="preview__divider" />
        <iframe :srcdoc="renderedPreviewBody" class="preview__iframe" sandbox="allow-same-origin" />
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('system.common.close') }}</UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getLoginConfig, updateLoginConfig } from '@/api/login'
import { getEmailConfig, updateEmailConfig, testEmailConfig, updateEmailTemplate, getEmailTemplate } from '@/api/system'
import { EmailTemplateType } from '@/types/v1/system'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'

defineOptions({ name: 'SystemSettingsView' })

const { t } = useI18n()
const menuStore = useMenuStore()

const TABS = [
  { name: 'security', labelKey: 'system.settings.securityTab', icon: 'HOutline:ShieldCheckIcon' },
  { name: 'email', labelKey: 'system.settings.emailTab', icon: 'HOutline:EnvelopeIcon' },
] as const

const activeTab = ref<'security' | 'email'>('security')
const loginSaving = ref(false)
const emailSaving = ref(false)
const emailTesting = ref(false)
const showPwd = ref(false)
const testRecipient = ref('')
const testDialogVisible = ref(false)

const templateSaving = ref(false)
const templateTesting = ref(false)
const templateTestDialogVisible = ref(false)
const templateTestRecipient = ref('')
const previewDialogVisible = ref(false)

const canEdit = useButtonPermission([PERM.SYSTEM_CONFIG_EDIT], [])

const placeholders = ['{{.name}}', '{{.email}}', '{{.code}}']
const subjectDescParams = { placeholder: placeholders[0] }
const contentDescParams = { email: placeholders[1], code: placeholders[2] }

// 模板占位符字面量（避免在模板文本插值里写字面 {{ 触发 Vue 解析错误）
const fieldToken = (name: string) => `{{.${name}}}`

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

const TEMPLATE_TYPE_LABEL_KEYS: Record<string, string> = {
  [EmailTemplateType.EmailTemplateType_Register]: 'system.settings.registerEmailTemplate',
  [EmailTemplateType.EmailTemplateType_Login]: 'system.settings.loginEmailTemplate',
}

const templateTypeOptions = computed<{ label: string; value: EmailTemplateType }[]>(() =>
  Object.entries(TEMPLATE_TYPE_LABEL_KEYS).map(([value, labelKey]) => ({
    value: value as EmailTemplateType,
    label: t(labelKey),
  })),
)

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

const templateType = ref<EmailTemplateType>(EmailTemplateType.EmailTemplateType_Register)
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
    const { data } = await getEmailTemplate({ type: templateType.value })
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
      type: templateType.value,
      subject: templateForm.subject,
      template: templateForm.template,
    })
    feedback.success(t('system.common.saveSuccess'))
  } finally {
    templateSaving.value = false
  }
}

const openTemplateTestDialog = () => {
  const names = extractPlaceholders(templateForm.subject + ' ' + templateForm.template)
  templateTestFields.value = names.map((name) => ({ name, value: '' }))
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
      config: undefined, // 使用全局邮件配置
      messages: {
        subject: templateForm.subject,
        template: templateForm.template,
        type: templateType.value,
      },
      Data: data,
    })
    feedback.success(t('system.settings.emailTestSuccess'))
    templateTestDialogVisible.value = false
  } catch {
    feedback.error(t('system.settings.emailTestFailed'))
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
    feedback.error(t('system.settings.getLoginConfigFailed'))
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
    feedback.error(t('system.settings.getEmailConfigFailed'))
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
    feedback.success(t('system.common.saveSuccess'))
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
    feedback.success(t('system.common.saveSuccess'))
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
      Data: {}, // 使用全局邮件模板测试发送
    })
    feedback.success(t('system.settings.emailTestSuccess'))
    testDialogVisible.value = false
    testRecipient.value = ''
  } catch {
    feedback.error(t('system.settings.emailTestFailed'))
  } finally {
    emailTesting.value = false
  }
}

const loadedTabs = ref(new Set<string>())

const onTabChange = (name: string) => {
  if (loadedTabs.value.has(name)) return
  loadedTabs.value.add(name)
  if (name === 'security') {
    loadLoginConfig()
  } else if (name === 'email') {
    loadEmailConfig()
    handleTemplateTypeChange()
  }
}

const setTab = (name: 'security' | 'email') => {
  activeTab.value = name
  onTabChange(name)
}

// 切换模板类型即拉取对应模板（AppSelect 仅 emit update:modelValue，用 watch 承接）
watch(templateType, handleTemplateTypeChange)

onMounted(() => {
  onTabChange(activeTab.value)
})
</script>

<style scoped lang="scss">
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* Tab 条 —— 令牌分段（沿用 list 页 .seg 观感，页级放大） */
.settings-tabs {
  display: inline-flex;
  align-self: flex-start;
  padding: 3px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
}
.settings-tabs__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 16px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.settings-tabs__btn :deep(svg) {
  width: 16px;
  height: 16px;
}
.settings-tabs__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}

.settings-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* 设置行 */
.set-rows {
  display: flex;
  flex-direction: column;
}
.set-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 11px 20px;
}
.set-row + .set-row {
  border-top: 1px solid var(--el-border-color-lighter);
}
.set-row__info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}
.set-row__label {
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}
.set-row__desc {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.set-row--col {
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
}
.set-row--col .set-row__info {
  flex: none;
}

/* 控件宽度 */
.set-input {
  width: 260px;
  max-width: 100%;
  flex-shrink: 0;
}
.set-num {
  width: 150px;
  flex-shrink: 0;
}
.set-wide {
  width: 400px;
  max-width: 100%;
  flex-shrink: 0;
}
.set-code {
  width: 100%;
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.8125rem;
  line-height: 1.5;
}
.set-pwd {
  display: flex;
  align-items: center;
  gap: 6px;
}
.set-pwd .app-input {
  flex: 1;
  min-width: 0;
}

/* 快捷插入占位符 */
.set-row--chips {
  justify-content: flex-start;
  flex-wrap: wrap;
  gap: 6px;
  background: var(--el-fill-color-lighter);
}
.set-chips__label {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  margin-right: 2px;
}
.set-chip {
  padding: 2px 9px;
  border: 1px solid var(--el-border-color);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 0.75rem;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s;
}
.set-chip:hover:not(:disabled) {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.set-chip:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

/* 对话框内多字段间距 */
.settings-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* 预览 */
.preview__subject {
  display: flex;
  gap: 8px;
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
}
.preview__label {
  font-weight: 600;
  flex-shrink: 0;
}
.preview__divider {
  height: 1px;
  background: var(--el-border-color-lighter);
  margin: 12px 0;
}
.preview__iframe {
  width: 100%;
  height: 420px;
  border: 1px solid var(--el-border-color);
  border-radius: var(--app-radius-sm);
  background: #fff;
}

@media (width <= 768px) {
  .settings-tabs {
    align-self: stretch;
    display: flex;
  }
  .settings-tabs__btn {
    flex: 1;
    justify-content: center;
  }
  .set-row {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
  .set-row__info {
    flex: none;
  }
  .set-row :deep(.app-input),
  .set-input,
  .set-num,
  .set-wide {
    width: 100%;
  }
  .set-row .app-switch {
    align-self: flex-start;
  }
  .preview__iframe {
    height: 280px;
  }
}
</style>
