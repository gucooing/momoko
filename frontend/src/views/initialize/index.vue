<template>
  <div class="initialize-container">
    <div class="card-wrapper">
      <div class="bg-decoration-orange"></div>
      <div class="bg-decoration-blue"></div>

      <div class="init-card">
        <div class="init-card-top">
          <div class="brand">
            <img :src="APP_CONFIG.logoSrc" alt="logo" class="logo" />
            <span class="brand-name">{{ APP_CONFIG.name }}</span>
          </div>
          <div class="top-actions">
            <HoverAnimateWrapper name="rotate">
              <IconButton
                icon="HOutline:Cog6ToothIcon"
                :tooltip="t('layout.themeConfig')"
                @click="themeStore.themeConfigDrawerOpen = true"
              />
            </HoverAnimateWrapper>
          </div>
        </div>

        <el-steps :active="currentStep" align-center finish-status="success" class="init-steps">
          <el-step :title="t('initialize.steps.database')" />
          <el-step :title="t('initialize.steps.admin')" />
          <el-step :title="t('initialize.steps.confirm')" />
        </el-steps>

        <div class="init-form-wrap">
          <!-- Step 1: 数据库配置 -->
          <Transition name="fade-slide" mode="out-in">
            <div v-if="currentStep === 0" key="step1" class="step-content">
              <el-form
                ref="dbFormRef"
                :model="dbForm"
                :rules="dbRules"
                label-position="top"
                size="large"
              >
                <el-form-item :label="t('initialize.databaseType')" prop="type">
                  <el-select v-model="dbForm.type" class="w-full" @change="onDbTypeChange">
                    <el-option
                      v-for="dt in supportedDatabaseTypes"
                      :key="dt"
                      :label="databaseTypeLabel(dt)"
                      :value="dt"
                    />
                  </el-select>
                </el-form-item>

                <template v-if="dbForm.type === DatabaseType.DatabaseType_SQLite">
                  <el-form-item :label="t('initialize.sqlitePath')" prop="sqlitePath">
                    <el-input v-model="dbForm.sqlitePath" :placeholder="t('initialize.sqlitePathPlaceholder')">
                      <template #suffix>
                        <el-tooltip :content="t('initialize.sqlitePathTip')">
                          <span class="text-gray-400 text-xs cursor-help">?</span>
                        </el-tooltip>
                      </template>
                    </el-input>
                  </el-form-item>
                </template>

                <template v-else>
                  <el-form-item :label="t('initialize.address')" prop="address">
                    <el-input
                      v-model="dbForm.address"
                      :placeholder="addressPlaceholder"
                    />
                  </el-form-item>
                  <el-form-item :label="t('initialize.databaseUsername')" prop="username">
                    <el-input v-model="dbForm.username" :placeholder="t('initialize.databaseUsernamePlaceholder')" />
                  </el-form-item>
                  <el-form-item :label="t('initialize.databasePassword')" prop="password">
                    <el-input
                      v-model="dbForm.password"
                      type="password"
                      show-password
                      :placeholder="t('initialize.databasePasswordPlaceholder')"
                    />
                  </el-form-item>
                  <el-form-item :label="t('initialize.databaseName')" prop="databaseName">
                    <el-input v-model="dbForm.databaseName" :placeholder="t('initialize.databaseNamePlaceholder')" />
                  </el-form-item>
                </template>
              </el-form>

              <div class="step-actions">
                <el-button
                  type="primary"
                  size="large"
                  :loading="testing"
                  :disabled="!isDbTypeSelected"
                  @click="handleTestConnection"
                >
                  {{ testing ? t('initialize.testConnecting') : t('initialize.testConnection') }}
                </el-button>
                <el-button
                  size="large"
                  :disabled="!connectionTested"
                  @click="currentStep = 1"
                >
                  {{ t('initialize.nextStep') }}
                </el-button>
              </div>
            </div>

            <!-- Step 2: 管理员账号 -->
            <div v-else-if="currentStep === 1" key="step2" class="step-content">
              <el-form
                ref="adminFormRef"
                :model="adminForm"
                :rules="adminRules"
                label-position="top"
                size="large"
              >
                <el-form-item :label="t('initialize.adminUsername')" prop="username">
                  <el-input v-model="adminForm.username" :placeholder="t('initialize.adminUsernamePlaceholder')" />
                </el-form-item>
                <el-form-item :label="t('initialize.adminPassword')" prop="password">
                  <el-input
                    v-model="adminForm.password"
                    type="password"
                    show-password
                    :placeholder="t('initialize.adminPasswordPlaceholder')"
                  />
                </el-form-item>
                <el-form-item :label="t('initialize.adminConfirmPassword')" prop="confirmPassword">
                  <el-input
                    v-model="adminForm.confirmPassword"
                    type="password"
                    show-password
                    :placeholder="t('initialize.adminConfirmPasswordPlaceholder')"
                  />
                </el-form-item>
                <el-form-item :label="t('initialize.adminEmail')" prop="email">
                  <el-input v-model="adminForm.email" :placeholder="t('initialize.adminEmailPlaceholder')" />
                </el-form-item>
                <el-form-item :label="t('initialize.adminName')" prop="name">
                  <el-input v-model="adminForm.name" :placeholder="t('initialize.adminNamePlaceholder')" />
                </el-form-item>
              </el-form>

              <div class="step-actions">
                <el-button size="large" @click="currentStep = 0">{{ t('initialize.previousStep') }}</el-button>
                <el-button type="primary" size="large" @click="handleStep2Next">{{ t('initialize.nextStep') }}</el-button>
              </div>
            </div>

            <!-- Step 3: 确认初始化 -->
            <div v-else key="step3" class="step-content">
              <template v-if="!confirmed">
                <div class="summary-section">
                  <h4 class="summary-title">{{ t('initialize.steps.database') }}</h4>
                  <div class="summary-grid">
                    <div class="summary-item">
                      <span class="summary-label">{{ t('initialize.type') }}</span>
                      <span class="summary-value">{{ databaseTypeLabel(dbForm.type) }}</span>
                    </div>
                    <div v-if="dbForm.type === DatabaseType.DatabaseType_SQLite" class="summary-item">
                      <span class="summary-label">{{ t('initialize.filePath') }}</span>
                      <span class="summary-value">{{ dbForm.sqlitePath || `./data/momoko.db${t('initialize.defaultMark')}` }}</span>
                    </div>
                    <template v-else>
                      <div class="summary-item">
                        <span class="summary-label">{{ t('initialize.address') }}</span>
                        <span class="summary-value">{{ dbForm.address }}</span>
                      </div>
                      <div class="summary-item">
                        <span class="summary-label">{{ t('initialize.databaseUsername') }}</span>
                        <span class="summary-value">{{ dbForm.username }}</span>
                      </div>
                      <div class="summary-item">
                        <span class="summary-label">{{ t('initialize.databaseName') }}</span>
                        <span class="summary-value">{{ dbForm.databaseName }}</span>
                      </div>
                    </template>
                  </div>

                  <h4 class="summary-title">{{ t('initialize.steps.admin') }}</h4>
                  <div class="summary-grid">
                    <div class="summary-item">
                      <span class="summary-label">{{ t('initialize.adminUsername') }}</span>
                      <span class="summary-value">{{ adminForm.username }}</span>
                    </div>
                    <div class="summary-item">
                      <span class="summary-label">{{ t('initialize.adminEmail') }}</span>
                      <span class="summary-value">{{ adminForm.email }}</span>
                    </div>
                    <div class="summary-item">
                      <span class="summary-label">{{ t('initialize.adminName') }}</span>
                      <span class="summary-value">{{ adminForm.name }}</span>
                    </div>
                  </div>
                </div>

                <div class="step-actions">
                  <el-button size="large" :disabled="confirming" @click="currentStep = 1">{{ t('initialize.previousStep') }}</el-button>
                  <el-button type="primary" size="large" :loading="confirming" @click="handleConfirm">
                    {{ confirming ? t('initialize.initializing') : t('initialize.confirmInitialize') }}
                  </el-button>
                </div>
              </template>

              <template v-else>
                <div class="result-section">
                  <el-result
                    :icon="initError ? 'error' : (readyForLogin ? 'success' : 'warning')"
                    :title="resultTitle"
                    :sub-title="resultSubTitle"
                  >
                    <template #extra>
                      <el-button v-if="!readyForLogin" type="primary" :loading="polling" @click="goToLogin">
                        {{ polling ? t('initialize.waitingRestart') : t('initialize.goLogin') }}
                      </el-button>
                      <el-button v-if="readyForLogin" type="primary" @click="goToLogin">{{ t('initialize.goLogin') }}</el-button>
                      <el-button v-if="initError" type="primary" @click="resetInit">{{ t('initialize.reconfigure') }}</el-button>
                    </template>
                  </el-result>
                </div>
              </template>
            </div>
          </Transition>
        </div>
      </div>
    </div>

    <ThemeConfig />
    <div class="init-copyright">Copyright &copy; 2025 DFANNN</div>
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
import type { FormInstance, FormRules } from 'element-plus'
import ThemeConfig from '@/components/ThemeConfig.vue'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'InitializeView' })

const themeStore = useThemeStore()
const router = useRouter()
const { t } = useI18n()

const currentStep = ref(0)
const testing = ref(false)
const connectionTested = ref(false)
const confirming = ref(false)
const confirmed = ref(false)
const polling = ref(false)
const readyForLogin = ref(false)
const initError = ref(false)
const supportedDatabaseTypes = ref<DatabaseType[]>([])

const dbFormRef = ref<FormInstance>()
const adminFormRef = ref<FormInstance>()

const isDbTypeSelected = computed(() => !!dbForm.type)

const addressPlaceholder = computed(() => {
  if (dbForm.type === DatabaseType.DatabaseType_MySQL) return t('initialize.mysqlAddressPlaceholder')
  if (dbForm.type === DatabaseType.DatabaseType_PostgreSQL) return t('initialize.postgresAddressPlaceholder')
  return t('initialize.addressPlaceholder')
})

interface DbForm {
  type: DatabaseType
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
  type: '' as DatabaseType,
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

function databaseTypeLabel(type: DatabaseType): string {
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

function validateDbForm(): Record<string, FormRules[0]> {
  const rules: Record<string, FormRules[0]> = {
    type: [{ required: true, message: t('initialize.databaseTypeRequired'), trigger: 'change' }],
  }

  if (dbForm.type === DatabaseType.DatabaseType_SQLite) {
    // sqlite path is optional
  } else {
    rules.address = [{ required: true, message: t('initialize.addressRequired'), trigger: 'blur' }]
    rules.username = [{ required: true, message: t('initialize.databaseUsernameRequired'), trigger: 'blur' }]
    rules.databaseName = [{ required: true, message: t('initialize.databaseNameRequired'), trigger: 'blur' }]
  }

  return rules
}

const dbRules = computed(() => validateDbForm())

const adminRules = computed<FormRules>(() => ({
  username: [{ required: true, message: t('initialize.adminUsernameRequired'), trigger: 'blur' }],
  password: [
    { required: true, message: t('initialize.adminPasswordRequired'), trigger: 'blur' },
    { min: 6, message: t('initialize.passwordMin'), trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: t('initialize.adminConfirmPasswordRequired'), trigger: 'blur' },
    {
      validator: (_rule, value, callback) => {
        if (value !== adminForm.password) {
          callback(new Error(t('initialize.passwordMismatch')))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
  email: [
    { required: true, message: t('initialize.adminEmailRequired'), trigger: 'blur' },
    { type: 'email', message: t('initialize.adminEmailInvalid'), trigger: 'blur' },
  ],
  name: [{ required: true, message: t('initialize.adminNameRequired'), trigger: 'blur' }],
}))

function onDbTypeChange() {
  connectionTested.value = false
  dbForm.sqlitePath = ''
  dbForm.address = ''
  dbForm.username = ''
  dbForm.password = ''
  dbForm.databaseName = ''
  dbFormRef.value?.clearValidate()
}

async function handleTestConnection() {
  const valid = await dbFormRef.value?.validate().catch(() => false)
  if (!valid) return

  testing.value = true
  connectionTested.value = false

  try {
    const payload = buildTestPayload()
    const { data } = await testDatabaseConnection(payload)
    if (data?.success) {
      ElMessage.success(t('initialize.databaseConnected'))
      connectionTested.value = true
    } else {
      ElMessage.error(t('initialize.databaseConnectFailed'))
    }
  } catch {
    // error already shown by request interceptor
  } finally {
    testing.value = false
  }
}

function buildTestPayload(): TestInitializeDatabaseRequest {
  return {
    database: {
      type: dbForm.type,
      sqlitePath: dbForm.type === DatabaseType.DatabaseType_SQLite ? dbForm.sqlitePath : '',
      address: dbForm.type === DatabaseType.DatabaseType_SQLite ? '' : dbForm.address,
      username: dbForm.type === DatabaseType.DatabaseType_SQLite ? '' : dbForm.username,
      password: dbForm.type === DatabaseType.DatabaseType_SQLite ? '' : dbForm.password,
      databaseName: dbForm.type === DatabaseType.DatabaseType_SQLite ? '' : dbForm.databaseName,
    },
  }
}

function buildConfirmPayload(): ConfirmInitializeRequest {
  return {
    database: {
      type: dbForm.type,
      sqlitePath: dbForm.type === DatabaseType.DatabaseType_SQLite ? dbForm.sqlitePath : '',
      address: dbForm.type === DatabaseType.DatabaseType_SQLite ? '' : dbForm.address,
      username: dbForm.type === DatabaseType.DatabaseType_SQLite ? '' : dbForm.username,
      password: dbForm.type === DatabaseType.DatabaseType_SQLite ? '' : dbForm.password,
      databaseName: dbForm.type === DatabaseType.DatabaseType_SQLite ? '' : dbForm.databaseName,
    },
    admin: {
      username: adminForm.username,
      password: adminForm.password,
      email: adminForm.email,
      name: adminForm.name,
    },
  }
}

async function handleStep2Next() {
  const valid = await adminFormRef.value?.validate().catch(() => false)
  if (!valid) return
  currentStep.value = 2
}

async function handleConfirm() {
  confirming.value = true
  initError.value = false

  try {
    const payload = buildConfirmPayload()
    const { data } = await confirmInitialize(payload)
    if (data?.initialized) {
      confirmed.value = true
      if (data.restartRequired) {
        startPolling()
      } else {
        readyForLogin.value = true
      }
    } else {
      initError.value = true
      ElMessage.error(t('initialize.resultFailed'))
    }
  } catch {
    initError.value = true
  } finally {
    confirming.value = false
  }
}

function startPolling() {
  polling.value = true
  let attempts = 0
  const maxAttempts = 60

  const poll = setInterval(async () => {
    attempts++
    try {
      const { data } = await getInitializeStatus()
      if (data?.initialized) {
        clearInterval(poll)
        polling.value = false
        readyForLogin.value = true
        return
      }
    } catch {
      // 接口可能因重启而暂时不可用
    }

    if (attempts >= maxAttempts) {
      clearInterval(poll)
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
    // 接口不可用时保持在初始化页面
  }
}

onMounted(loadInitStatus)
</script>

<style scoped lang="scss">
.initialize-container {
  min-height: 100vh;
  width: 100%;
  background-color: var(--el-bg-color-page);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  padding: 20px;

  .card-wrapper {
    width: 100%;
    position: relative;
    z-index: 10;
    display: flex;
    align-items: center;
    justify-content: center;

    .bg-decoration-orange {
      position: absolute;
      bottom: -100px;
      left: -100px;
      width: 400px;
      height: 400px;
      background-color: #f99c7d;
      border-radius: 50%;
      opacity: 0.8;
      z-index: -1;
      animation: float-orange 20s infinite ease-in-out;
      filter: blur(20px);
    }

    .bg-decoration-blue {
      position: absolute;
      top: -120px;
      right: -100px;
      width: 350px;
      height: 450px;
      background-color: #5bbff9;
      border-radius: 40% 60% 70% 30% / 40% 50% 60% 50%;
      opacity: 0.8;
      z-index: -1;
      transform: rotate(15deg);
      animation: float-blue 25s infinite ease-in-out;
      filter: blur(20px);
    }
  }

  .init-card {
    width: min(100%, 44rem);
    max-width: 95%;
    background: var(--el-bg-color-overlay);
    border-radius: 16px;
    box-shadow: var(--el-box-shadow-light);
    display: flex;
    flex-direction: column;
    z-index: 10;
    overflow: hidden;
    padding: 2.5rem;

    .init-card-top {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 1.5rem;

      .brand {
        display: flex;
        align-items: center;
        gap: 1rem;
        .logo {
          width: 2.5rem;
          height: 2.5rem;
        }
        .brand-name {
          font-size: 1.5rem;
          font-weight: 600;
          color: var(--el-text-color-primary);
        }
      }

      .top-actions {
        display: flex;
        align-items: center;
      }
    }

    .init-steps {
      margin-bottom: 2rem;
    }

    .init-form-wrap {
      width: 100%;
      min-height: 22rem;

      .step-content {
        width: 100%;
      }

      .step-actions {
        display: flex;
        justify-content: center;
        gap: 1rem;
        margin-top: 2rem;
      }
    }
  }

  .init-copyright {
    position: absolute;
    bottom: 20px;
    left: 0;
    right: 0;
    text-align: center;
    font-size: 0.85rem;
    color: var(--el-text-color-placeholder);
    z-index: 20;
  }
}

.summary-section {
  margin-bottom: 1.5rem;

  .summary-title {
    font-size: 1rem;
    font-weight: 600;
    color: var(--el-text-color-primary);
    margin-bottom: 0.75rem;
    padding-bottom: 0.5rem;
    border-bottom: 1px solid var(--el-border-color-light);
  }

  .summary-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 0.5rem;
  }

  .summary-item {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;

    .summary-label {
      font-size: 0.85rem;
      color: var(--el-text-color-secondary);
    }

    .summary-value {
      font-size: 0.95rem;
      color: var(--el-text-color-primary);
    }
  }
}

.result-section {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 18rem;
}

@keyframes float-orange {
  0%,
  100% {
    transform: translate(0, 0);
  }
  50% {
    transform: translate(30px, -20px);
  }
}

@keyframes float-blue {
  0%,
  100% {
    transform: rotate(15deg) translate(0, 0);
  }
  50% {
    transform: rotate(20deg) translate(-20px, 30px);
  }
}

@media (max-width: 992px) {
  .initialize-container {
    padding: 10px;

    .card-wrapper {
      width: 100%;
    }

    .init-card {
      width: 98%;
      max-width: 98%;
      padding: 2rem 1.5rem;
    }
  }
}
</style>
