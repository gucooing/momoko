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
                tooltip="主题配置"
                @click="themeStore.themeConfigDrawerOpen = true"
              />
            </HoverAnimateWrapper>
          </div>
        </div>

        <el-steps :active="currentStep" align-center finish-status="success" class="init-steps">
          <el-step title="数据库配置" />
          <el-step title="管理员账号" />
          <el-step title="确认初始化" />
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
                <el-form-item label="数据库类型" prop="type">
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
                  <el-form-item label="数据库文件路径" prop="sqlitePath">
                    <el-input v-model="dbForm.sqlitePath" placeholder="留空使用默认路径 ./data/momoko.db">
                      <template #suffix>
                        <el-tooltip content="默认为 ./data/momoko.db">
                          <span class="text-gray-400 text-xs cursor-help">?</span>
                        </el-tooltip>
                      </template>
                    </el-input>
                  </el-form-item>
                </template>

                <template v-else>
                  <el-form-item label="连接地址" prop="address">
                    <el-input
                      v-model="dbForm.address"
                      :placeholder="addressPlaceholder"
                    />
                  </el-form-item>
                  <el-form-item label="用户名" prop="username">
                    <el-input v-model="dbForm.username" placeholder="请输入数据库用户名" />
                  </el-form-item>
                  <el-form-item label="密码" prop="password">
                    <el-input
                      v-model="dbForm.password"
                      type="password"
                      show-password
                      placeholder="请输入数据库密码"
                    />
                  </el-form-item>
                  <el-form-item label="数据库名" prop="databaseName">
                    <el-input v-model="dbForm.databaseName" placeholder="请输入数据库名" />
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
                  {{ testing ? '测试中...' : '测试连接' }}
                </el-button>
                <el-button
                  size="large"
                  :disabled="!connectionTested"
                  @click="currentStep = 1"
                >
                  下一步
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
                <el-form-item label="用户名" prop="username">
                  <el-input v-model="adminForm.username" placeholder="请输入管理员用户名" />
                </el-form-item>
                <el-form-item label="密码" prop="password">
                  <el-input
                    v-model="adminForm.password"
                    type="password"
                    show-password
                    placeholder="请输入管理员密码"
                  />
                </el-form-item>
                <el-form-item label="确认密码" prop="confirmPassword">
                  <el-input
                    v-model="adminForm.confirmPassword"
                    type="password"
                    show-password
                    placeholder="请再次输入管理员密码"
                  />
                </el-form-item>
                <el-form-item label="邮箱" prop="email">
                  <el-input v-model="adminForm.email" placeholder="请输入管理员邮箱" />
                </el-form-item>
                <el-form-item label="昵称" prop="name">
                  <el-input v-model="adminForm.name" placeholder="请输入管理员昵称" />
                </el-form-item>
              </el-form>

              <div class="step-actions">
                <el-button size="large" @click="currentStep = 0">上一步</el-button>
                <el-button type="primary" size="large" @click="handleStep2Next">下一步</el-button>
              </div>
            </div>

            <!-- Step 3: 确认初始化 -->
            <div v-else key="step3" class="step-content">
              <template v-if="!confirmed">
                <div class="summary-section">
                  <h4 class="summary-title">数据库配置</h4>
                  <div class="summary-grid">
                    <div class="summary-item">
                      <span class="summary-label">类型</span>
                      <span class="summary-value">{{ databaseTypeLabel(dbForm.type) }}</span>
                    </div>
                    <div v-if="dbForm.type === DatabaseType.DatabaseType_SQLite" class="summary-item">
                      <span class="summary-label">文件路径</span>
                      <span class="summary-value">{{ dbForm.sqlitePath || './data/momoko.db（默认）' }}</span>
                    </div>
                    <template v-else>
                      <div class="summary-item">
                        <span class="summary-label">连接地址</span>
                        <span class="summary-value">{{ dbForm.address }}</span>
                      </div>
                      <div class="summary-item">
                        <span class="summary-label">用户名</span>
                        <span class="summary-value">{{ dbForm.username }}</span>
                      </div>
                      <div class="summary-item">
                        <span class="summary-label">数据库名</span>
                        <span class="summary-value">{{ dbForm.databaseName }}</span>
                      </div>
                    </template>
                  </div>

                  <h4 class="summary-title">管理员账号</h4>
                  <div class="summary-grid">
                    <div class="summary-item">
                      <span class="summary-label">用户名</span>
                      <span class="summary-value">{{ adminForm.username }}</span>
                    </div>
                    <div class="summary-item">
                      <span class="summary-label">邮箱</span>
                      <span class="summary-value">{{ adminForm.email }}</span>
                    </div>
                    <div class="summary-item">
                      <span class="summary-label">昵称</span>
                      <span class="summary-value">{{ adminForm.name }}</span>
                    </div>
                  </div>
                </div>

                <div class="step-actions">
                  <el-button size="large" :disabled="confirming" @click="currentStep = 1">上一步</el-button>
                  <el-button type="primary" size="large" :loading="confirming" @click="handleConfirm">
                    {{ confirming ? '正在初始化...' : '确认初始化' }}
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
                        {{ polling ? '等待服务重启中...' : '前往登录' }}
                      </el-button>
                      <el-button v-if="readyForLogin" type="primary" @click="goToLogin">前往登录</el-button>
                      <el-button v-if="initError" type="primary" @click="resetInit">重新配置</el-button>
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

defineOptions({ name: 'InitializeView' })

const themeStore = useThemeStore()
const router = useRouter()

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
  if (dbForm.type === DatabaseType.DatabaseType_MySQL) return 'host 或 host:port，如 127.0.0.1:3306'
  if (dbForm.type === DatabaseType.DatabaseType_PostgreSQL) return 'host 或 host:port，如 127.0.0.1:5432'
  return 'host 或 host:port'
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
  if (initError.value) return '初始化失败'
  if (readyForLogin.value) return '初始化完成'
  return '系统初始化中'
})

const resultSubTitle = computed(() => {
  if (initError.value) return '系统初始化过程中出现错误，请检查配置后重试'
  if (readyForLogin.value) return '系统已完成初始化并重启，请使用刚才创建的管理员账号登录'
  return '服务正在自动重启，请稍候...'
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
      return type || '请选择'
  }
}

function validateDbForm(): Record<string, FormRules[0]> {
  const rules: Record<string, FormRules[0]> = {
    type: [{ required: true, message: '请选择数据库类型', trigger: 'change' }],
  }

  if (dbForm.type === DatabaseType.DatabaseType_SQLite) {
    // sqlite path is optional
  } else {
    rules.address = [{ required: true, message: '请输入连接地址', trigger: 'blur' }]
    rules.username = [{ required: true, message: '请输入用户名', trigger: 'blur' }]
    rules.databaseName = [{ required: true, message: '请输入数据库名', trigger: 'blur' }]
  }

  return rules
}

const dbRules = computed(() => validateDbForm())

const adminRules: FormRules = {
  username: [{ required: true, message: '请输入管理员用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入管理员密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入管理员密码', trigger: 'blur' },
    {
      validator: (_rule, value, callback) => {
        if (value !== adminForm.password) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
  email: [
    { required: true, message: '请输入管理员邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' },
  ],
  name: [{ required: true, message: '请输入管理员昵称', trigger: 'blur' }],
}

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
      ElMessage.success('数据库连接成功')
      connectionTested.value = true
    } else {
      ElMessage.error('数据库连接失败')
    }
  } catch (e: any) {
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
      ElMessage.error('初始化失败')
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
