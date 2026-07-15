<!-- 初始化向导（P3）：居中卡 + 三步（数据库 → 管理员 → 确认）；
     对齐登录门面：冷灰页底、hairline 卡、令牌表单、无 EP、无彩色装饰。 -->
<template>
  <div class="init-page">
    <div class="init-page__chrome">
      <LanguageMenu />
      <AppIconButton
        v-if="APP_CONFIG.showThemeConfig"
        icon="HOutline:Cog6ToothIcon"
        :label="t('layout.themeConfig')"
        :box="32"
        @click="themeStore.themeConfigDrawerOpen = true"
      />
      <AppIconButton
        :icon="themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
        :label="t('login.toggleTheme')"
        :box="32"
        @click="toggleTheme"
      />
    </div>

    <div class="init-page__center">
      <div class="init-card">
        <div class="init-card__brand">
          <img :src="APP_CONFIG.logoSrc" alt="" class="init-card__logo" />
          <span class="init-card__name">{{ APP_CONFIG.name }}</span>
        </div>

        <header class="init-card__head">
          <h1 class="init-card__title">{{ t('initialize.pageTitle') }}</h1>
          <p class="init-card__sub">{{ t('initialize.pageDesc') }}</p>
        </header>

        <!-- 步骤条：桌面数字+文案；移动紧凑点 -->
        <nav class="init-steps" :aria-label="t('initialize.pageTitle')">
          <template v-for="(step, i) in steps" :key="step.key">
            <div
              class="init-steps__item"
              :class="{
                'is-active': currentStep === i,
                'is-done': currentStep > i || (i === 2 && confirmed && readyForLogin),
              }"
            >
              <span class="init-steps__dot">
                <component
                  :is="menuStore.iconComponents['HOutline:CheckIcon']"
                  v-if="currentStep > i || (i === 2 && confirmed && readyForLogin && !initError)"
                  class="init-steps__check"
                />
                <template v-else>{{ i + 1 }}</template>
              </span>
              <span class="init-steps__label">{{ step.label }}</span>
            </div>
            <span v-if="i < steps.length - 1" class="init-steps__line" aria-hidden="true" />
          </template>
        </nav>

        <div class="init-card__body">
          <Transition name="init-fade" mode="out-in">
            <!-- Step 0: 数据库 -->
            <div v-if="currentStep === 0" key="db" class="init-step">
              <div class="app-field">
                <label class="app-label">{{ t('initialize.databaseType') }}</label>
                <AppSelect
                  v-model="dbForm.type"
                  :options="dbTypeOptions"
                  :placeholder="t('initialize.pleaseSelect')"
                  :error="!!dbErrors.type"
                  @update:model-value="onDbTypeChange"
                />
                <span v-if="dbErrors.type" class="app-field__error">{{ dbErrors.type }}</span>
              </div>

              <template v-if="dbForm.type === DatabaseType.DatabaseType_SQLite">
                <div class="app-field">
                  <label class="app-label" for="init-sqlite-path">
                    {{ t('initialize.sqlitePath') }}
                    <span class="init-hint" :title="t('initialize.sqlitePathTip')">?</span>
                  </label>
                  <input
                    id="init-sqlite-path"
                    v-model="dbForm.sqlitePath"
                    class="app-input init-input"
                    :placeholder="t('initialize.sqlitePathPlaceholder')"
                    autocomplete="off"
                  />
                </div>
              </template>

              <template v-else-if="isRemoteDb">
                <div class="app-field">
                  <label class="app-label" for="init-address">{{ t('initialize.address') }}</label>
                  <input
                    id="init-address"
                    v-model="dbForm.address"
                    class="app-input init-input"
                    :class="{ 'is-error': dbErrors.address }"
                    :placeholder="addressPlaceholder"
                    autocomplete="off"
                  />
                  <span v-if="dbErrors.address" class="app-field__error">{{ dbErrors.address }}</span>
                </div>
                <div class="app-field">
                  <label class="app-label" for="init-db-user">{{ t('initialize.databaseUsername') }}</label>
                  <input
                    id="init-db-user"
                    v-model="dbForm.username"
                    class="app-input init-input"
                    :class="{ 'is-error': dbErrors.username }"
                    :placeholder="t('initialize.databaseUsernamePlaceholder')"
                    autocomplete="username"
                  />
                  <span v-if="dbErrors.username" class="app-field__error">{{ dbErrors.username }}</span>
                </div>
                <div class="app-field">
                  <label class="app-label" for="init-db-pass">{{ t('initialize.databasePassword') }}</label>
                  <div class="init-input-wrap">
                    <input
                      id="init-db-pass"
                      v-model="dbForm.password"
                      class="app-input init-input init-input--with-icon"
                      :type="showDbPwd ? 'text' : 'password'"
                      :placeholder="t('initialize.databasePasswordPlaceholder')"
                      autocomplete="new-password"
                    />
                    <button
                      type="button"
                      class="init-eye"
                      :aria-label="showDbPwd ? t('login.hidePassword') : t('login.showPassword')"
                      @click="showDbPwd = !showDbPwd"
                    >
                      <component
                        :is="
                          menuStore.iconComponents[
                            showDbPwd ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon'
                          ]
                        "
                        class="init-eye__icon"
                      />
                    </button>
                  </div>
                </div>
                <div class="app-field">
                  <label class="app-label" for="init-db-name">{{ t('initialize.databaseName') }}</label>
                  <input
                    id="init-db-name"
                    v-model="dbForm.databaseName"
                    class="app-input init-input"
                    :class="{ 'is-error': dbErrors.databaseName }"
                    :placeholder="t('initialize.databaseNamePlaceholder')"
                    autocomplete="off"
                  />
                  <span v-if="dbErrors.databaseName" class="app-field__error">{{
                    dbErrors.databaseName
                  }}</span>
                </div>
              </template>

              <div class="init-actions">
                <UButton
                  color="primary"
                  variant="soft"
                  class="init-btn"
                  :loading="testing"
                  :disabled="!isDbTypeSelected"
                  @click="handleTestConnection"
                >
                  {{ testing ? t('initialize.testConnecting') : t('initialize.testConnection') }}
                </UButton>
                <UButton
                  color="primary"
                  class="init-btn"
                  :disabled="!connectionTested"
                  @click="currentStep = 1"
                >
                  {{ t('initialize.nextStep') }}
                </UButton>
              </div>
            </div>

            <!-- Step 1: 管理员 -->
            <div v-else-if="currentStep === 1" key="admin" class="init-step">
              <div class="app-field">
                <label class="app-label" for="init-admin-user">{{ t('initialize.adminUsername') }}</label>
                <input
                  id="init-admin-user"
                  v-model="adminForm.username"
                  class="app-input init-input"
                  :class="{ 'is-error': adminErrors.username }"
                  :placeholder="t('initialize.adminUsernamePlaceholder')"
                  autocomplete="username"
                />
                <span v-if="adminErrors.username" class="app-field__error">{{
                  adminErrors.username
                }}</span>
              </div>
              <div class="app-field">
                <label class="app-label" for="init-admin-pass">{{ t('initialize.adminPassword') }}</label>
                <div class="init-input-wrap">
                  <input
                    id="init-admin-pass"
                    v-model="adminForm.password"
                    class="app-input init-input init-input--with-icon"
                    :class="{ 'is-error': adminErrors.password }"
                    :type="showAdminPwd ? 'text' : 'password'"
                    :placeholder="t('initialize.adminPasswordPlaceholder')"
                    autocomplete="new-password"
                  />
                  <button
                    type="button"
                    class="init-eye"
                    :aria-label="showAdminPwd ? t('login.hidePassword') : t('login.showPassword')"
                    @click="showAdminPwd = !showAdminPwd"
                  >
                    <component
                      :is="
                        menuStore.iconComponents[
                          showAdminPwd ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon'
                        ]
                      "
                      class="init-eye__icon"
                    />
                  </button>
                </div>
                <span v-if="adminErrors.password" class="app-field__error">{{
                  adminErrors.password
                }}</span>
              </div>
              <div class="app-field">
                <label class="app-label" for="init-admin-pass2">{{
                  t('initialize.adminConfirmPassword')
                }}</label>
                <div class="init-input-wrap">
                  <input
                    id="init-admin-pass2"
                    v-model="adminForm.confirmPassword"
                    class="app-input init-input init-input--with-icon"
                    :class="{ 'is-error': adminErrors.confirmPassword }"
                    :type="showAdminPwd2 ? 'text' : 'password'"
                    :placeholder="t('initialize.adminConfirmPasswordPlaceholder')"
                    autocomplete="new-password"
                  />
                  <button
                    type="button"
                    class="init-eye"
                    :aria-label="showAdminPwd2 ? t('login.hidePassword') : t('login.showPassword')"
                    @click="showAdminPwd2 = !showAdminPwd2"
                  >
                    <component
                      :is="
                        menuStore.iconComponents[
                          showAdminPwd2 ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon'
                        ]
                      "
                      class="init-eye__icon"
                    />
                  </button>
                </div>
                <span v-if="adminErrors.confirmPassword" class="app-field__error">{{
                  adminErrors.confirmPassword
                }}</span>
              </div>
              <div class="app-field">
                <label class="app-label" for="init-admin-email">{{ t('initialize.adminEmail') }}</label>
                <input
                  id="init-admin-email"
                  v-model="adminForm.email"
                  class="app-input init-input"
                  :class="{ 'is-error': adminErrors.email }"
                  :placeholder="t('initialize.adminEmailPlaceholder')"
                  autocomplete="email"
                  inputmode="email"
                />
                <span v-if="adminErrors.email" class="app-field__error">{{ adminErrors.email }}</span>
              </div>
              <div class="app-field">
                <label class="app-label" for="init-admin-name">{{ t('initialize.adminName') }}</label>
                <input
                  id="init-admin-name"
                  v-model="adminForm.name"
                  class="app-input init-input"
                  :class="{ 'is-error': adminErrors.name }"
                  :placeholder="t('initialize.adminNamePlaceholder')"
                  autocomplete="nickname"
                />
                <span v-if="adminErrors.name" class="app-field__error">{{ adminErrors.name }}</span>
              </div>

              <div class="init-actions">
                <UButton color="neutral" variant="ghost" class="init-btn" @click="currentStep = 0">
                  {{ t('initialize.previousStep') }}
                </UButton>
                <UButton color="primary" class="init-btn" @click="handleStep2Next">
                  {{ t('initialize.nextStep') }}
                </UButton>
              </div>
            </div>

            <!-- Step 2: 确认 / 结果 -->
            <div v-else key="confirm" class="init-step">
              <template v-if="!confirmed">
                <section class="init-summary">
                  <h2 class="init-summary__title">{{ t('initialize.steps.database') }}</h2>
                  <dl class="init-summary__grid">
                    <div class="init-summary__item">
                      <dt>{{ t('initialize.type') }}</dt>
                      <dd>{{ databaseTypeLabel(dbForm.type) }}</dd>
                    </div>
                    <div
                      v-if="dbForm.type === DatabaseType.DatabaseType_SQLite"
                      class="init-summary__item"
                    >
                      <dt>{{ t('initialize.filePath') }}</dt>
                      <dd>
                        {{
                          dbForm.sqlitePath ||
                          `./data/momoko.db${t('initialize.defaultMark')}`
                        }}
                      </dd>
                    </div>
                    <template v-else>
                      <div class="init-summary__item">
                        <dt>{{ t('initialize.address') }}</dt>
                        <dd>{{ dbForm.address }}</dd>
                      </div>
                      <div class="init-summary__item">
                        <dt>{{ t('initialize.databaseUsername') }}</dt>
                        <dd>{{ dbForm.username }}</dd>
                      </div>
                      <div class="init-summary__item">
                        <dt>{{ t('initialize.databaseName') }}</dt>
                        <dd>{{ dbForm.databaseName }}</dd>
                      </div>
                    </template>
                  </dl>

                  <h2 class="init-summary__title">{{ t('initialize.steps.admin') }}</h2>
                  <dl class="init-summary__grid">
                    <div class="init-summary__item">
                      <dt>{{ t('initialize.adminUsername') }}</dt>
                      <dd>{{ adminForm.username }}</dd>
                    </div>
                    <div class="init-summary__item">
                      <dt>{{ t('initialize.adminEmail') }}</dt>
                      <dd>{{ adminForm.email }}</dd>
                    </div>
                    <div class="init-summary__item">
                      <dt>{{ t('initialize.adminName') }}</dt>
                      <dd>{{ adminForm.name }}</dd>
                    </div>
                  </dl>
                </section>

                <div class="init-actions">
                  <UButton
                    color="neutral"
                    variant="ghost"
                    class="init-btn"
                    :disabled="confirming"
                    @click="currentStep = 1"
                  >
                    {{ t('initialize.previousStep') }}
                  </UButton>
                  <UButton
                    color="primary"
                    class="init-btn"
                    :loading="confirming"
                    @click="handleConfirm"
                  >
                    {{ confirming ? t('initialize.initializing') : t('initialize.confirmInitialize') }}
                  </UButton>
                </div>
              </template>

              <template v-else>
                <div class="init-result" :class="resultTone">
                  <div class="init-result__icon" aria-hidden="true">
                    <component
                      :is="
                        menuStore.iconComponents[
                          initError
                            ? 'HOutline:XCircleIcon'
                            : readyForLogin
                              ? 'HOutline:CheckCircleIcon'
                              : 'HOutline:ArrowPathIcon'
                        ]
                      "
                      class="init-result__svg"
                      :class="{ 'is-spin': polling && !readyForLogin && !initError }"
                    />
                  </div>
                  <h2 class="init-result__title">{{ resultTitle }}</h2>
                  <p class="init-result__sub">{{ resultSubTitle }}</p>
                  <div class="init-actions init-actions--center">
                    <UButton
                      v-if="!readyForLogin && !initError"
                      color="primary"
                      class="init-btn"
                      :loading="polling"
                      @click="goToLogin"
                    >
                      {{ polling ? t('initialize.waitingRestart') : t('initialize.goLogin') }}
                    </UButton>
                    <UButton
                      v-if="readyForLogin"
                      color="primary"
                      class="init-btn"
                      @click="goToLogin"
                    >
                      {{ t('initialize.goLogin') }}
                    </UButton>
                    <UButton
                      v-if="initError"
                      color="primary"
                      class="init-btn"
                      @click="resetInit"
                    >
                      {{ t('initialize.reconfigure') }}
                    </UButton>
                  </div>
                </div>
              </template>
            </div>
          </Transition>
        </div>
      </div>

      <p class="init-page__copy">
        {{ t('login.copyright', { year: currentYear, name: APP_CONFIG.name }) }}
      </p>
    </div>

    <ThemeConfig />
  </div>
</template>

<script setup lang="ts">
import { APP_CONFIG } from '@/config/app.config'
import {
  getInitializeStatus,
  testDatabaseConnection,
  confirmInitialize,
} from '@/api/initialize'
import { DatabaseType } from '@/types/v1/initialize'
import type {
  TestInitializeDatabaseRequest,
  ConfirmInitializeRequest,
} from '@/types/v1/initialize'
import { useFeedback } from '@/utils/feedback'
import ThemeConfig from '@/components/ThemeConfig.vue'
import LanguageMenu from '@/layouts/app/LanguageMenu.vue'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'InitializeView' })

const EMAIL_REGEXP = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

const themeStore = useThemeStore()
const menuStore = useMenuStore()
const router = useRouter()
const { t } = useI18n()
const fb = useFeedback()

const currentYear = new Date().getFullYear()
const currentStep = ref(0)
const testing = ref(false)
const connectionTested = ref(false)
const confirming = ref(false)
const confirmed = ref(false)
const polling = ref(false)
const readyForLogin = ref(false)
const initError = ref(false)
const supportedDatabaseTypes = ref<DatabaseType[]>([])
const showDbPwd = ref(false)
const showAdminPwd = ref(false)
const showAdminPwd2 = ref(false)

const dbErrors = ref<Record<string, string>>({})
const adminErrors = ref<Record<string, string>>({})

interface DbForm {
  type: DatabaseType | ''
  sqlitePath: string
  address: string
  username: string
  password: string
  databaseName: string
}

interface AdminForm {
  username: string
  password: string
  confirmPassword: string
  email: string
  name: string
}

const dbForm = reactive<DbForm>({
  type: '',
  sqlitePath: '',
  address: '',
  username: '',
  password: '',
  databaseName: '',
})

const adminForm = reactive<AdminForm>({
  username: '',
  password: '',
  confirmPassword: '',
  email: '',
  name: '',
})

const steps = computed(() => [
  { key: 'database', label: t('initialize.steps.database') },
  { key: 'admin', label: t('initialize.steps.admin') },
  { key: 'confirm', label: t('initialize.steps.confirm') },
])

const isDbTypeSelected = computed(() => !!dbForm.type)
const isRemoteDb = computed(
  () =>
    dbForm.type === DatabaseType.DatabaseType_MySQL ||
    dbForm.type === DatabaseType.DatabaseType_PostgreSQL,
)

const addressPlaceholder = computed(() => {
  if (dbForm.type === DatabaseType.DatabaseType_MySQL) return t('initialize.mysqlAddressPlaceholder')
  if (dbForm.type === DatabaseType.DatabaseType_PostgreSQL)
    return t('initialize.postgresAddressPlaceholder')
  return t('initialize.addressPlaceholder')
})

const dbTypeOptions = computed(() =>
  supportedDatabaseTypes.value.map((dt) => ({
    label: databaseTypeLabel(dt),
    value: dt as DatabaseType | '',
  })),
)

const resultTitle = computed(() => {
  if (initError.value) return t('initialize.resultFailed')
  if (readyForLogin.value) return t('initialize.resultDone')
  return t('initialize.resultRunning')
})

const resultSubTitle = computed(() => {
  if (initError.value) return t('initialize.resultFailedSubtitle')
  if (readyForLogin.value) return t('initialize.resultDoneSubtitle')
  return t('initialize.resultRunningSubtitle')
})

const resultTone = computed(() => {
  if (initError.value) return 'is-error'
  if (readyForLogin.value) return 'is-success'
  return 'is-pending'
})

function databaseTypeLabel(type: DatabaseType | ''): string {
  switch (type) {
    case DatabaseType.DatabaseType_SQLite:
      return 'SQLite'
    case DatabaseType.DatabaseType_MySQL:
      return 'MySQL'
    case DatabaseType.DatabaseType_PostgreSQL:
      return 'PostgreSQL'
    default:
      return type || t('initialize.pleaseSelect')
  }
}

function toggleTheme() {
  themeStore.toggleThemeMode(themeStore.isDarkTheme ? 'light' : 'dark')
}

function onDbTypeChange() {
  connectionTested.value = false
  dbForm.sqlitePath = ''
  dbForm.address = ''
  dbForm.username = ''
  dbForm.password = ''
  dbForm.databaseName = ''
  dbErrors.value = {}
}

function validateDbForm(): boolean {
  const next: Record<string, string> = {}
  if (!dbForm.type) {
    next.type = t('initialize.databaseTypeRequired')
  } else if (dbForm.type !== DatabaseType.DatabaseType_SQLite) {
    if (!dbForm.address.trim()) next.address = t('initialize.addressRequired')
    if (!dbForm.username.trim()) next.username = t('initialize.databaseUsernameRequired')
    if (!dbForm.databaseName.trim()) next.databaseName = t('initialize.databaseNameRequired')
  }
  dbErrors.value = next
  return Object.keys(next).length === 0
}

function validateAdminForm(): boolean {
  const next: Record<string, string> = {}
  if (!adminForm.username.trim()) next.username = t('initialize.adminUsernameRequired')
  if (!adminForm.password) {
    next.password = t('initialize.adminPasswordRequired')
  } else if (adminForm.password.length < 6) {
    next.password = t('initialize.passwordMin')
  }
  if (!adminForm.confirmPassword) {
    next.confirmPassword = t('initialize.adminConfirmPasswordRequired')
  } else if (adminForm.confirmPassword !== adminForm.password) {
    next.confirmPassword = t('initialize.passwordMismatch')
  }
  const email = adminForm.email.trim()
  if (!email) {
    next.email = t('initialize.adminEmailRequired')
  } else if (!EMAIL_REGEXP.test(email)) {
    next.email = t('initialize.adminEmailInvalid')
  }
  if (!adminForm.name.trim()) next.name = t('initialize.adminNameRequired')
  adminErrors.value = next
  return Object.keys(next).length === 0
}

function buildDatabaseConfig() {
  const isSqlite = dbForm.type === DatabaseType.DatabaseType_SQLite
  return {
    type: dbForm.type as DatabaseType,
    sqlitePath: isSqlite ? dbForm.sqlitePath : '',
    address: isSqlite ? '' : dbForm.address,
    username: isSqlite ? '' : dbForm.username,
    password: isSqlite ? '' : dbForm.password,
    databaseName: isSqlite ? '' : dbForm.databaseName,
  }
}

function buildTestPayload(): TestInitializeDatabaseRequest {
  return { database: buildDatabaseConfig() }
}

function buildConfirmPayload(): ConfirmInitializeRequest {
  return {
    database: buildDatabaseConfig(),
    admin: {
      username: adminForm.username,
      password: adminForm.password,
      email: adminForm.email,
      name: adminForm.name,
    },
  }
}

async function handleTestConnection() {
  if (!validateDbForm()) return

  testing.value = true
  connectionTested.value = false

  try {
    const { data } = await testDatabaseConnection(buildTestPayload())
    if (data?.success) {
      fb.success(t('initialize.databaseConnected'))
      connectionTested.value = true
    } else {
      fb.error(t('initialize.databaseConnectFailed'))
    }
  } catch {
    // 请求层已 toast
  } finally {
    testing.value = false
  }
}

function handleStep2Next() {
  if (!validateAdminForm()) return
  currentStep.value = 2
}

async function handleConfirm() {
  confirming.value = true
  initError.value = false

  try {
    const { data } = await confirmInitialize(buildConfirmPayload())
    if (data?.initialized) {
      confirmed.value = true
      if (data.restartRequired) {
        startPolling()
      } else {
        readyForLogin.value = true
      }
    } else {
      confirmed.value = true
      initError.value = true
      fb.error(t('initialize.resultFailed'))
    }
  } catch {
    confirmed.value = true
    initError.value = true
  } finally {
    confirming.value = false
  }
}

let pollTimer: ReturnType<typeof setInterval> | null = null

function clearPoll() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function startPolling() {
  polling.value = true
  clearPoll()
  let attempts = 0
  const maxAttempts = 60

  pollTimer = setInterval(async () => {
    attempts++
    try {
      const { data } = await getInitializeStatus()
      if (data?.initialized) {
        clearPoll()
        polling.value = false
        readyForLogin.value = true
        return
      }
    } catch {
      // 重启期间接口可能短暂不可用
    }

    if (attempts >= maxAttempts) {
      clearPoll()
      polling.value = false
      readyForLogin.value = true
    }
  }, 1000)
}

function goToLogin() {
  router.replace({ path: '/login' })
}

function resetInit() {
  confirmed.value = false
  initError.value = false
  readyForLogin.value = false
  polling.value = false
  clearPoll()
  currentStep.value = 2
}

async function loadInitStatus() {
  try {
    const { data } = await getInitializeStatus()
    supportedDatabaseTypes.value = data?.supportedDatabaseTypes || []
    if (data?.initialized) {
      router.replace({ path: '/login' })
    }
  } catch {
    // 接口不可用时保持在初始化页
  }
}

onMounted(loadInitStatus)
onBeforeUnmount(clearPoll)
</script>

<style scoped lang="scss">
.init-page {
  position: relative;
  min-height: 100vh;
  min-height: 100dvh;
  width: 100%;
  background: var(--el-bg-color-page);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 16px 32px;
  box-sizing: border-box;
}

.init-page__chrome {
  position: absolute;
  top: 12px;
  right: 12px;
  display: flex;
  align-items: center;
  gap: 2px;
  z-index: 2;
}

.init-page__center {
  width: min(100%, 480px);
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 16px;
}

.init-card {
  background: var(--el-bg-color);
  border: 1px solid var(--app-hairline);
  border-radius: var(--app-radius-lg);
  padding: 28px 28px 24px;
  box-shadow: var(--app-shadow-sm);
}

.init-card__brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 18px;
}

.init-card__logo {
  width: 32px;
  height: 32px;
  object-fit: contain;
  flex-shrink: 0;
}

.init-card__name {
  font-size: 1.0625rem;
  font-weight: 650;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}

.init-card__head {
  margin-bottom: 16px;
}

.init-card__title {
  margin: 0 0 4px;
  font-size: 1.25rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.3;
  color: var(--el-text-color-primary);
}

.init-card__sub {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
}

/* —— 步骤条 —— */
.init-steps {
  display: flex;
  align-items: center;
  gap: 0;
  margin-bottom: 20px;
  padding: 10px 4px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
}

.init-steps__item {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
  justify-content: center;
  padding: 0 4px;
}

.init-steps__dot {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 999px;
  font-size: 0.6875rem;
  font-weight: 700;
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
  background: var(--el-bg-color);
  box-shadow: inset 0 0 0 1px var(--app-hairline);
  font-variant-numeric: tabular-nums;
}

.init-steps__item.is-active .init-steps__dot {
  color: #fff;
  background: var(--el-color-primary);
  box-shadow: none;
}

.init-steps__item.is-done .init-steps__dot {
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 14%, transparent);
  box-shadow: none;
}

.init-steps__check {
  width: 12px;
  height: 12px;
}

.init-steps__label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.init-steps__item.is-active .init-steps__label {
  color: var(--el-color-primary);
}

.init-steps__item.is-done .init-steps__label {
  color: var(--el-text-color-regular);
}

.init-steps__line {
  width: 12px;
  height: 1px;
  flex-shrink: 0;
  background: var(--app-hairline);
}

/* —— 表单区 —— */
.init-step {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.init-input {
  height: 36px;
  font-size: 0.8125rem;
}

.init-input-wrap {
  position: relative;
}

.init-input--with-icon {
  padding-right: 36px;
}

.init-eye {
  position: absolute;
  top: 50%;
  right: 4px;
  transform: translateY(-50%);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-placeholder);
  cursor: pointer;
  padding: 0;
}

.init-eye:hover {
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
}

.init-eye__icon {
  width: 16px;
  height: 16px;
}

.init-hint {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  margin-left: 4px;
  border-radius: 999px;
  font-size: 0.625rem;
  font-weight: 700;
  color: var(--el-text-color-placeholder);
  background: var(--el-fill-color);
  cursor: help;
  vertical-align: middle;
}

.init-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.init-actions--center {
  justify-content: center;
}

.init-btn {
  height: 36px !important;
  min-width: 6.5rem;
  font-size: 0.8125rem !important;
  font-weight: 600;
}

/* —— 摘要 —— */
.init-summary {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.init-summary__title {
  margin: 0;
  padding-bottom: 6px;
  font-size: 0.8125rem;
  font-weight: 650;
  color: var(--el-text-color-primary);
  border-bottom: 1px solid var(--app-hairline);
}

.init-summary__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 12px;
  margin: 0;
}

.init-summary__item {
  min-width: 0;

  dt {
    margin: 0 0 2px;
    font-size: 0.75rem;
    color: var(--el-text-color-secondary);
  }

  dd {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--el-text-color-primary);
    word-break: break-all;
  }
}

/* —— 结果 —— */
.init-result {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 12px 0 4px;
  gap: 8px;
}

.init-result__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 999px;
  margin-bottom: 4px;
}

.init-result.is-success .init-result__icon {
  color: var(--el-color-success);
  background: color-mix(in srgb, var(--el-color-success) 12%, transparent);
}

.init-result.is-error .init-result__icon {
  color: var(--el-color-danger);
  background: color-mix(in srgb, var(--el-color-danger) 12%, transparent);
}

.init-result.is-pending .init-result__icon {
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
}

.init-result__svg {
  width: 26px;
  height: 26px;
}

.init-result__svg.is-spin {
  animation: init-spin 1s linear infinite;
}

@keyframes init-spin {
  to {
    transform: rotate(360deg);
  }
}

.init-result__title {
  margin: 0;
  font-size: 1.0625rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}

.init-result__sub {
  margin: 0 0 8px;
  max-width: 28rem;
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.init-page__copy {
  margin: 0;
  text-align: center;
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}

.init-fade-enter-active,
.init-fade-leave-active {
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}
.init-fade-enter-from {
  opacity: 0;
  transform: translateY(4px);
}
.init-fade-leave-to {
  opacity: 0;
  transform: translateY(-3px);
}

@media (max-width: 480px) {
  .init-page {
    padding: 56px 12px 32px;
    align-items: center;
  }

  .init-card {
    padding: 22px 18px 20px;
    border-radius: var(--app-radius);
  }

  .init-steps__label {
    display: none;
  }

  .init-steps__item {
    flex: 0 0 auto;
  }

  .init-steps {
    justify-content: center;
    gap: 0;
    padding: 10px 12px;
  }

  .init-steps__line {
    width: 28px;
    margin: 0 4px;
  }

  .init-summary__grid {
    grid-template-columns: 1fr;
  }

  .init-actions {
    flex-direction: column-reverse;

    .init-btn {
      width: 100%;
    }
  }
}
</style>
