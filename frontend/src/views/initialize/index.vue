<!-- 初始化向导：左侧主色品牌区（步骤列表 + 贴纸）+ 右侧表单；手机上品牌区在上（当前步标题 + 进度条），
     表单卡片上浮，底部固定操作条。四步：选择数据库 → 连接信息（测试连接）→ 创建超级管理员 → 完成（等待重启 / 结果）。 -->
<template>
  <div class="init" :class="{ 'is-final': currentStep === 3 }">
    <section class="init-brand">
      <span class="init-brand__blob" aria-hidden="true" />
      <span class="init-brand__ring" aria-hidden="true" />
      <svg class="init-brand__star" viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 1.5l2.6 7.9 7.9 2.6-7.9 2.6L12 22.5l-2.6-7.9L1.5 12l7.9-2.6z" />
      </svg>

      <div class="init-brand__logo">
        <img :src="APP_CONFIG.logoSrc" alt="" />
        <span>{{ APP_CONFIG.name }}</span>
        <span class="init-brand__badge">{{ t('initialize.firstRun') }}</span>
      </div>

      <div class="init-brand__copy">
        <h1>
          {{ t('initialize.heroLine1') }}<br />
          {{ t('initialize.heroLine2', { name: APP_CONFIG.name }) }}
        </h1>
        <p>{{ t('initialize.heroDesc') }}</p>
      </div>

      <!-- 手机：当前步标题 + 进度条 -->
      <div class="init-brand__mstep">
        <span class="init-brand__mcount">
          {{ t('initialize.firstRun') }} · {{ t('initialize.stepOf', { step: currentStep + 1, total: steps.length }) }}
        </span>
        <h1 class="init-brand__mtitle">{{ steps[currentStep]?.title }}</h1>
        <ol class="init-brand__bars" :aria-label="t('initialize.pageTitle')">
          <li
            v-for="(s, i) in steps"
            :key="s.key"
            :class="{ 'is-on': i <= currentStep }"
            :aria-label="s.title"
          />
        </ol>
      </div>

      <ol class="init-steps" :aria-label="t('initialize.pageTitle')">
        <li v-for="(s, i) in steps" :key="s.key">
          <button
            type="button"
            class="init-steps__btn"
            :class="{ 'is-done': stepDone(i), 'is-on': currentStep === i }"
            :aria-current="currentStep === i ? 'step' : undefined"
            :disabled="!canJumpTo(i)"
            @click="jumpTo(i)"
          >
            <span class="init-steps__num">
              <component
                :is="menuStore.iconComponents['HOutline:CheckIcon']"
                v-if="stepDone(i)"
                class="init-steps__check"
              />
              <span v-else class="m-num">{{ i + 1 }}</span>
            </span>
            <span class="init-steps__text">
              <span class="init-steps__title">{{ s.title }}</span>
              <span class="init-steps__desc">{{ s.desc }}</span>
            </span>
          </button>
        </li>
      </ol>

      <ul class="init-brand__tags" aria-hidden="true">
        <li v-for="(tag, i) in heroTags" :key="tag" :class="`is-${i % 3}`">{{ tag }}</li>
      </ul>
    </section>

    <section class="init-panel">
      <div class="init-panel__top">
        <span class="init-panel__count">
          <i18n-t scope="global" keypath="initialize.stepOfRich" tag="span">
            <template #step>
              <b class="m-num">{{ currentStep + 1 }}</b>
            </template>
            <template #total>{{ steps.length }}</template>
          </i18n-t>
        </span>
        <span class="init-panel__spacer" />
        <div class="init-panel__chrome">
          <LanguageMenu />
          <AppIconButton
            v-if="APP_CONFIG.showThemeConfig"
            icon="HOutline:SwatchIcon"
            :label="t('layout.themeConfig')"
            @click="themeStore.themeConfigDrawerOpen = true"
          />
          <AppIconButton
            :icon="themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
            :label="t('login.toggleTheme')"
            @click="toggleTheme"
          />
        </div>
      </div>
      <div class="init-panel__bars" aria-hidden="true">
        <span v-for="(s, i) in steps" :key="s.key" :class="{ 'is-on': i <= currentStep }" />
      </div>

      <div class="init-panel__center">
        <form class="init-form" novalidate @submit.prevent="onNext">
          <Transition name="init-fade" mode="out-in">
            <!-- 第 1 步：选择数据库 -->
            <div v-if="currentStep === 0" key="type" class="init-step">
              <header class="init-head">
                <h2>{{ t('initialize.typeTitle') }}</h2>
                <p>{{ t('initialize.typeDesc', { name: APP_CONFIG.name }) }}</p>
              </header>

              <div
                v-if="dbOptions.length"
                class="init-dbs"
                role="radiogroup"
                :aria-label="t('initialize.databaseType')"
              >
                <button
                  v-for="d in dbOptions"
                  :key="d.value"
                  type="button"
                  role="radio"
                  class="init-db"
                  :class="{ 'is-on': dbForm.type === d.value }"
                  :aria-checked="dbForm.type === d.value"
                  @click="selectDbType(d.value)"
                >
                  <span class="init-db__top">
                    <span class="m-sticker init-db__av" :class="`m-sticker--${d.tone}`" :style="{ rotate: d.rot }">
                      {{ d.short }}
                    </span>
                    <span class="init-db__chk" :class="{ 'is-on': dbForm.type === d.value }">
                      <component
                        :is="menuStore.iconComponents['HOutline:CheckIcon']"
                        v-if="dbForm.type === d.value"
                      />
                    </span>
                  </span>
                  <span class="init-db__body">
                    <span class="init-db__name">
                      {{ d.name }}
                      <span v-if="d.recommended" class="init-db__rec">{{ t('initialize.recommended') }}</span>
                    </span>
                    <span class="init-db__sub">{{ d.sub }}</span>
                    <span class="init-db__desc">{{ d.desc }}</span>
                  </span>
                </button>
              </div>
              <div v-else class="init-note init-note--warn">
                <component :is="menuStore.iconComponents['HOutline:ExclamationTriangleIcon']" />
                <span>{{ t('initialize.noDatabaseTypes') }}</span>
              </div>
              <span v-if="dbErrors.type" class="app-field__error">{{ dbErrors.type }}</span>

              <div class="init-note">
                <component :is="menuStore.iconComponents['HOutline:InformationCircleIcon']" />
                <span>{{ t('initialize.typeTip') }}</span>
              </div>
            </div>

            <!-- 第 2 步：连接信息 -->
            <div v-else-if="currentStep === 1" key="conn" class="init-step">
              <header class="init-head">
                <h2>{{ t('initialize.connTitle') }}</h2>
                <i18n-t scope="global" keypath="initialize.connSelected" tag="p">
                  <template #db>
                    <b>{{ databaseTypeLabel(dbForm.type) }}</b>
                  </template>
                  <template #tip>{{ isSqlite ? t('initialize.connSqliteTip') : t('initialize.connRemoteTip') }}</template>
                </i18n-t>
              </header>

              <template v-if="isSqlite">
                <label class="init-field">
                  <span class="init-field__label">{{ t('initialize.sqlitePath') }}</span>
                  <span class="init-fld">
                    <component :is="menuStore.iconComponents['HOutline:DocumentIcon']" class="init-fld__ico" />
                    <input
                      v-model="dbForm.sqlitePath"
                      class="init-fld__input m-mono"
                      :placeholder="defaultSqlitePath"
                      autocomplete="off"
                      spellcheck="false"
                      @input="connectionTested = false"
                    />
                  </span>
                  <span class="init-field__hint">{{ t('initialize.sqlitePathHint') }}</span>
                </label>
              </template>

              <template v-else>
                <div class="init-grid init-grid--host">
                  <label class="init-field">
                    <span class="init-field__label">{{ t('initialize.host') }}</span>
                    <span class="init-fld" :class="{ 'is-error': dbErrors.address }">
                      <component :is="menuStore.iconComponents['HOutline:ServerIcon']" class="init-fld__ico" />
                      <input
                        v-model="dbForm.host"
                        class="init-fld__input m-mono"
                        placeholder="127.0.0.1"
                        autocomplete="off"
                        spellcheck="false"
                        @input="connectionTested = false"
                      />
                    </span>
                    <span v-if="dbErrors.address" class="app-field__error">{{ dbErrors.address }}</span>
                  </label>
                  <label class="init-field">
                    <span class="init-field__label">{{ t('initialize.port') }}</span>
                    <span class="init-fld">
                      <component :is="menuStore.iconComponents['HOutline:HashtagIcon']" class="init-fld__ico" />
                      <input
                        v-model="dbForm.port"
                        class="init-fld__input m-mono"
                        inputmode="numeric"
                        :placeholder="defaultPort"
                        autocomplete="off"
                        @input="connectionTested = false"
                      />
                    </span>
                  </label>
                </div>
                <label class="init-field">
                  <span class="init-field__label">{{ t('initialize.databaseName') }}</span>
                  <span class="init-fld" :class="{ 'is-error': dbErrors.databaseName }">
                    <component :is="menuStore.iconComponents['HOutline:CircleStackIcon']" class="init-fld__ico" />
                    <input
                      v-model="dbForm.databaseName"
                      class="init-fld__input m-mono"
                      placeholder="momoko"
                      autocomplete="off"
                      spellcheck="false"
                      @input="connectionTested = false"
                    />
                  </span>
                  <span v-if="dbErrors.databaseName" class="app-field__error">{{ dbErrors.databaseName }}</span>
                </label>
                <div class="init-grid">
                  <label class="init-field">
                    <span class="init-field__label">{{ t('initialize.databaseUsername') }}</span>
                    <span class="init-fld" :class="{ 'is-error': dbErrors.username }">
                      <component :is="menuStore.iconComponents['HOutline:UserIcon']" class="init-fld__ico" />
                      <input
                        v-model="dbForm.username"
                        class="init-fld__input m-mono"
                        placeholder="momoko"
                        autocomplete="off"
                        spellcheck="false"
                        @input="connectionTested = false"
                      />
                    </span>
                    <span v-if="dbErrors.username" class="app-field__error">{{ dbErrors.username }}</span>
                  </label>
                  <label class="init-field">
                    <span class="init-field__label">{{ t('initialize.databasePassword') }}</span>
                    <span class="init-fld">
                      <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" class="init-fld__ico" />
                      <input
                        v-model="dbForm.password"
                        class="init-fld__input"
                        :type="showDbPwd ? 'text' : 'password'"
                        :placeholder="t('initialize.databasePasswordPlaceholder')"
                        autocomplete="new-password"
                        @input="connectionTested = false"
                      />
                      <button
                        type="button"
                        class="init-fld__eye"
                        :aria-label="showDbPwd ? t('login.hidePassword') : t('login.showPassword')"
                        @click="showDbPwd = !showDbPwd"
                      >
                        <component
                          :is="menuStore.iconComponents[showDbPwd ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon']"
                        />
                      </button>
                    </span>
                  </label>
                </div>
              </template>

              <div class="init-test">
                <div class="init-test__row">
                  <UButton
                    color="neutral"
                    variant="outline"
                    class="init-btn init-btn--outline"
                    icon="i-lucide-plug"
                    :loading="testing"
                    @click="handleTestConnection"
                  >
                    {{ testing ? t('initialize.testConnecting') : t('initialize.testConnection') }}
                  </UButton>
                  <span v-if="!connectionTested && !testing" class="init-field__hint">
                    {{ t('initialize.testFirst') }}
                  </span>
                </div>
                <div v-if="connectionTested" class="init-ok" role="status">
                  <span class="init-ok__ico">
                    <component :is="menuStore.iconComponents['HOutline:CheckIcon']" />
                  </span>
                  <span class="init-ok__text">
                    <b>{{ t('initialize.databaseConnected') }}</b>
                    <span>{{ testSummary }}</span>
                  </span>
                </div>
              </div>
            </div>

            <!-- 第 3 步：超级管理员 -->
            <div v-else-if="currentStep === 2" key="admin" class="init-step">
              <header class="init-head">
                <h2>{{ t('initialize.adminTitle') }}</h2>
                <p>{{ t('initialize.adminDesc') }}</p>
              </header>

              <div class="init-grid">
                <label class="init-field">
                  <span class="init-field__label">{{ t('initialize.adminUsername') }}</span>
                  <span class="init-fld" :class="{ 'is-error': adminErrors.username }">
                    <component :is="menuStore.iconComponents['HOutline:UserIcon']" class="init-fld__ico" />
                    <input
                      v-model="adminForm.username"
                      class="init-fld__input"
                      placeholder="admin"
                      autocomplete="username"
                    />
                  </span>
                  <span v-if="adminErrors.username" class="app-field__error">{{ adminErrors.username }}</span>
                </label>
                <label class="init-field">
                  <span class="init-field__label">{{ t('initialize.adminEmail') }}</span>
                  <span class="init-fld" :class="{ 'is-error': adminErrors.email }">
                    <component :is="menuStore.iconComponents['HOutline:EnvelopeIcon']" class="init-fld__ico" />
                    <input
                      v-model="adminForm.email"
                      class="init-fld__input"
                      type="email"
                      placeholder="name@example.com"
                      autocomplete="email"
                      inputmode="email"
                    />
                  </span>
                  <span v-if="adminErrors.email" class="app-field__error">{{ adminErrors.email }}</span>
                </label>
              </div>

              <label class="init-field">
                <span class="init-field__label">{{ t('initialize.adminName') }}</span>
                <span class="init-fld" :class="{ 'is-error': adminErrors.name }">
                  <component :is="menuStore.iconComponents['HOutline:IdentificationIcon']" class="init-fld__ico" />
                  <input
                    v-model="adminForm.name"
                    class="init-fld__input"
                    :placeholder="t('initialize.adminNamePlaceholder')"
                    autocomplete="nickname"
                  />
                </span>
                <span v-if="adminErrors.name" class="app-field__error">{{ adminErrors.name }}</span>
              </label>

              <div class="init-field">
                <label class="init-field__label" for="init-admin-pass">{{ t('initialize.adminPassword') }}</label>
                <span class="init-fld" :class="{ 'is-error': adminErrors.password }">
                  <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" class="init-fld__ico" />
                  <input
                    id="init-admin-pass"
                    v-model="adminForm.password"
                    class="init-fld__input"
                    :type="showAdminPwd ? 'text' : 'password'"
                    :placeholder="t('initialize.passwordPlaceholder')"
                    autocomplete="new-password"
                  />
                  <button
                    type="button"
                    class="init-fld__eye"
                    :aria-label="showAdminPwd ? t('login.hidePassword') : t('login.showPassword')"
                    @click="showAdminPwd = !showAdminPwd"
                  >
                    <component
                      :is="menuStore.iconComponents[showAdminPwd ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon']"
                    />
                  </button>
                </span>
                <div class="init-meter" aria-live="polite">
                  <span class="init-meter__bars" aria-hidden="true">
                    <span
                      v-for="n in 4"
                      :key="n"
                      :class="adminForm.password && n <= strength.score ? `is-${strength.tone}` : ''"
                    />
                  </span>
                  <span class="init-meter__label" :class="adminForm.password ? `is-${strength.tone}` : ''">
                    {{ t('initialize.strength', { level: adminForm.password ? strength.label : '—' }) }}
                  </span>
                </div>
                <ul class="init-rules">
                  <li v-for="r in strength.rules" :key="r.key" :class="{ 'is-ok': r.ok }">
                    <component :is="menuStore.iconComponents[r.ok ? 'HOutline:CheckIcon' : 'HOutline:MinusIcon']" />
                    {{ r.label }}
                  </li>
                </ul>
                <span v-if="adminErrors.password" class="app-field__error">{{ adminErrors.password }}</span>
              </div>

              <div class="init-field">
                <label class="init-field__label" for="init-admin-pass2">{{ t('initialize.adminConfirmPassword') }}</label>
                <span class="init-fld" :class="{ 'is-error': adminErrors.confirmPassword }">
                  <component :is="menuStore.iconComponents['HOutline:ShieldCheckIcon']" class="init-fld__ico" />
                  <input
                    id="init-admin-pass2"
                    v-model="adminForm.confirmPassword"
                    class="init-fld__input"
                    :type="showAdminPwd ? 'text' : 'password'"
                    :placeholder="t('initialize.confirmPlaceholder')"
                    autocomplete="new-password"
                  />
                </span>
                <span class="init-match" :class="`is-${matchState}`">
                  <component
                    :is="
                      menuStore.iconComponents[
                        matchState === 'ok'
                          ? 'HOutline:CheckIcon'
                          : matchState === 'bad'
                            ? 'HOutline:XMarkIcon'
                            : 'HOutline:MinusIcon'
                      ]
                    "
                  />
                  {{
                    matchState === 'ok'
                      ? t('initialize.passwordMatch')
                      : matchState === 'bad'
                        ? t('initialize.passwordMismatch')
                        : t('initialize.adminConfirmPasswordRequired')
                  }}
                </span>
              </div>
            </div>

            <!-- 第 4 步：完成 / 等待重启 / 失败 -->
            <div v-else key="done" class="init-step">
              <div class="init-art" :class="`is-${resultTone}`" aria-hidden="true">
                <span class="init-art__disc" />
                <span class="init-art__ring" />
                <span class="init-art__tile">
                  <component
                    :is="
                      menuStore.iconComponents[
                        resultTone === 'error'
                          ? 'HOutline:XMarkIcon'
                          : resultTone === 'success'
                            ? 'HOutline:CheckIcon'
                            : 'HOutline:ArrowPathIcon'
                      ]
                    "
                    :class="{ 'is-spin': resultTone === 'pending' }"
                  />
                </span>
                <span class="init-art__tag">{{ resultSticker }}</span>
              </div>

              <header class="init-head init-head--done">
                <h2>{{ resultTitle }}</h2>
                <p>{{ resultSubTitle }}</p>
              </header>

              <dl class="init-summary">
                <div>
                  <dt>{{ t('initialize.summaryDatabase') }}</dt>
                  <dd>{{ databaseTypeLabel(dbForm.type) }}</dd>
                </div>
                <div>
                  <dt>{{ t('initialize.summaryAdmin') }}</dt>
                  <dd>{{ adminForm.username }}</dd>
                </div>
                <div class="is-wide">
                  <dt>{{ t('initialize.summaryTarget') }}</dt>
                  <dd class="m-mono is-mono">{{ dbTarget }}</dd>
                </div>
                <div v-if="status.hostname">
                  <dt>{{ t('initialize.summaryNode') }}</dt>
                  <dd>{{ status.hostname }}</dd>
                </div>
                <div v-if="status.version">
                  <dt>{{ t('initialize.summaryVersion') }}</dt>
                  <dd class="m-mono is-mono">{{ status.version }}</dd>
                </div>
                <div class="is-wide">
                  <dt>{{ t('initialize.summaryAddress') }}</dt>
                  <dd class="m-mono is-mono">{{ accessAddress }}</dd>
                </div>
              </dl>

              <div v-if="readyForLogin" class="init-next">
                <span class="init-next__label">{{ t('initialize.nextUp') }}</span>
                <div class="init-next__links">
                  <RouterLink
                    v-for="l in nextLinks"
                    :key="l.path"
                    class="init-next__link"
                    :to="{ path: '/login', query: { redirect: l.path } }"
                  >
                    <component :is="menuStore.iconComponents[l.icon]" />
                    {{ l.label }}
                  </RouterLink>
                </div>
              </div>

              <div class="init-done-actions">
                <UButton
                  v-if="initError"
                  color="primary"
                  class="init-btn init-btn--primary init-btn--lg"
                  icon="i-lucide-arrow-left"
                  @click="resetInit"
                >
                  {{ t('initialize.reconfigure') }}
                </UButton>
                <UButton
                  v-else
                  color="primary"
                  class="init-btn init-btn--primary init-btn--lg"
                  trailing-icon="i-lucide-arrow-right"
                  :loading="polling && !readyForLogin"
                  @click="goToLogin"
                >
                  {{ polling && !readyForLogin ? t('initialize.waitingRestart') : t('initialize.goLogin') }}
                </UButton>
              </div>
            </div>
          </Transition>

          <!-- 底部导航（第 4 步不显示）；手机上固定在底部 -->
          <div v-if="currentStep < 3" class="init-foot">
            <UButton
              v-if="currentStep > 0"
              color="neutral"
              variant="outline"
              class="init-btn init-btn--outline init-btn--back"
              icon="i-lucide-arrow-left"
              :aria-label="t('initialize.previousStep')"
              :disabled="confirming"
              @click="currentStep--"
            >
              <span class="init-btn__text">{{ t('initialize.previousStep') }}</span>
            </UButton>
            <span class="init-foot__spacer" />
            <UButton
              type="submit"
              color="primary"
              class="init-btn init-btn--primary init-btn--next"
              trailing-icon="i-lucide-arrow-right"
              :loading="confirming"
              :disabled="nextDisabled"
            >
              {{ nextLabel }}
            </UButton>
          </div>
        </form>
      </div>

      <p class="init-panel__copy">
        {{ footerText }}
      </p>
    </section>

    <ThemeConfig />
  </div>
</template>

<script setup lang="ts">
import { APP_CONFIG } from '@/config/app.config'
import { getInitializeStatus, testDatabaseConnection, confirmInitialize } from '@/api/initialize'
import { DatabaseType } from '@/types/v1/initialize'
import type { TestInitializeDatabaseRequest, ConfirmInitializeRequest } from '@/types/v1/initialize'
import { useFeedback } from '@/utils/feedback'
import ThemeConfig from '@/components/ThemeConfig.vue'
import LanguageMenu from '@/layouts/app/LanguageMenu.vue'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'InitializeView' })

const EMAIL_REGEXP = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const FALLBACK_SQLITE_PATH = './data/momoko.db'

const themeStore = useThemeStore()
const menuStore = useMenuStore()
const router = useRouter()
const { t } = useI18n()
const fb = useFeedback()

const currentYear = new Date().getFullYear()
const currentStep = ref(0)
const testing = ref(false)
const connectionTested = ref(false)
const testResult = reactive({ serverVersion: '', latencyMs: 0 })
const confirming = ref(false)
const confirmed = ref(false)
const polling = ref(false)
const readyForLogin = ref(false)
const initError = ref(false)
const showDbPwd = ref(false)
const showAdminPwd = ref(false)

const status = reactive({
  supportedDatabaseTypes: [] as DatabaseType[],
  version: '',
  hostname: '',
  defaultSqlitePath: '',
})

const dbErrors = ref<Record<string, string>>({})
const adminErrors = ref<Record<string, string>>({})

interface DbForm {
  type: DatabaseType | ''
  sqlitePath: string
  host: string
  port: string
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
  host: '',
  port: '',
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

const footerText = computed(() =>
  [
    status.version ? `${APP_CONFIG.name} ${status.version}` : APP_CONFIG.name,
    t('login.copyright', { year: currentYear, name: APP_CONFIG.name }),
  ].join(' · '),
)

const heroTags = computed(() => t('initialize.heroTags').split(',').filter(Boolean))

const steps = computed(() => [
  { key: 'type', title: t('initialize.typeTitle'), desc: t('initialize.typeStepDesc') },
  { key: 'conn', title: t('initialize.connTitle'), desc: t('initialize.connStepDesc') },
  { key: 'admin', title: t('initialize.adminTitle'), desc: t('initialize.adminStepDesc') },
  { key: 'done', title: t('initialize.doneTitle'), desc: t('initialize.doneStepDesc') },
])

const isSqlite = computed(() => dbForm.type === DatabaseType.DatabaseType_SQLite)
const defaultSqlitePath = computed(() => status.defaultSqlitePath || FALLBACK_SQLITE_PATH)
const defaultPort = computed(() => (dbForm.type === DatabaseType.DatabaseType_PostgreSQL ? '5432' : '3306'))

const DB_META: Record<string, { short: string; name: string; tone: string; rot: string; key: string }> = {
  [DatabaseType.DatabaseType_SQLite]: { short: 'S', name: 'SQLite', tone: 'a', rot: '-5deg', key: 'sqlite' },
  [DatabaseType.DatabaseType_MySQL]: { short: 'My', name: 'MySQL', tone: 'a2', rot: '4deg', key: 'mysql' },
  [DatabaseType.DatabaseType_PostgreSQL]: { short: 'Pg', name: 'PostgreSQL', tone: 'p', rot: '-3deg', key: 'postgres' },
}

const dbOptions = computed(() =>
  status.supportedDatabaseTypes
    .filter((dt) => DB_META[dt])
    .map((dt) => {
      const meta = DB_META[dt]!
      return {
        value: dt,
        short: meta.short,
        name: meta.name,
        tone: meta.tone,
        rot: meta.rot,
        recommended: dt === DatabaseType.DatabaseType_SQLite,
        sub: t(`initialize.db.${meta.key}.sub`),
        desc: t(`initialize.db.${meta.key}.desc`),
      }
    }),
)

const remoteAddress = computed(() => {
  const host = dbForm.host.trim()
  const port = dbForm.port.trim()
  if (!host) return ''
  return port ? `${host}:${port}` : host
})

const dbTarget = computed(() => {
  if (isSqlite.value) return dbForm.sqlitePath.trim() || defaultSqlitePath.value
  const addr = remoteAddress.value.includes(':') ? remoteAddress.value : `${remoteAddress.value}:${defaultPort.value}`
  const user = dbForm.username.trim()
  return `${user ? `${user}@` : ''}${addr}/${dbForm.databaseName.trim()}`
})

const accessAddress = computed(() => (typeof window !== 'undefined' ? window.location.origin : ''))

const testSummary = computed(() => {
  const parts: string[] = []
  if (testResult.serverVersion) parts.push(`${databaseTypeLabel(dbForm.type)} ${testResult.serverVersion}`)
  parts.push(t('initialize.latency', { ms: testResult.latencyMs }))
  parts.push(isSqlite.value ? t('initialize.sqliteWritable') : t('initialize.remoteWritable', { db: dbForm.databaseName.trim() }))
  return parts.join(' · ')
})

// 密码强度：仅提示，不参与校验（后端规则仍以 passwordMin 为准）
const strength = computed(() => {
  const pw = adminForm.password
  const checks = [
    { key: 'len', ok: pw.length >= 8, label: t('initialize.rules.length') },
    { key: 'case', ok: /[a-z]/.test(pw) && /[A-Z]/.test(pw), label: t('initialize.rules.case') },
    { key: 'digit', ok: /\d/.test(pw), label: t('initialize.rules.digit') },
    { key: 'symbol', ok: /[^A-Za-z0-9]/.test(pw), label: t('initialize.rules.symbol') },
  ]
  let score = checks.filter((c) => c.ok).length
  if (pw.length < 6) score = Math.min(score, 1)
  const tones = ['bad', 'bad', 'warn', 'ok', 'ok'] as const
  const levels = ['veryWeak', 'weak', 'fair', 'strong', 'veryStrong'] as const
  return {
    score: Math.max(score, 1),
    tone: tones[score] ?? 'bad',
    label: t(`initialize.levels.${levels[score] ?? 'weak'}`),
    rules: checks,
  }
})

const matchState = computed<'idle' | 'ok' | 'bad'>(() => {
  if (!adminForm.confirmPassword) return 'idle'
  return adminForm.confirmPassword === adminForm.password ? 'ok' : 'bad'
})

const nextLabel = computed(() => {
  if (currentStep.value === 2) return confirming.value ? t('initialize.initializing') : t('initialize.finish')
  return t('initialize.nextStep')
})

const nextDisabled = computed(() => {
  if (currentStep.value === 0) return !dbForm.type
  if (currentStep.value === 1) return !connectionTested.value || testing.value
  return false
})

const resultTone = computed<'error' | 'success' | 'pending'>(() => {
  if (initError.value) return 'error'
  if (readyForLogin.value) return 'success'
  return 'pending'
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

const resultSticker = computed(() => {
  if (initError.value) return t('initialize.stickerFailed')
  if (readyForLogin.value) return t('initialize.stickerDone')
  return t('initialize.stickerRunning')
})

const nextLinks = computed(() => [
  { path: '/instance/list', icon: 'HOutline:CubeIcon', label: t('initialize.linkApp') },
  { path: '/docker?tab=config', icon: 'HOutline:ServerStackIcon', label: t('initialize.linkDocker') },
  { path: '/node', icon: 'HOutline:ServerIcon', label: t('initialize.linkNode') },
])

function stepDone(i: number) {
  if (i === 3) return confirmed.value && readyForLogin.value && !initError.value
  return currentStep.value > i
}

// 只能回到已完成的步骤（提交后不再允许跳转）
function canJumpTo(i: number) {
  return !confirmed.value && !confirming.value && i < currentStep.value
}

function jumpTo(i: number) {
  if (canJumpTo(i)) currentStep.value = i
}

function databaseTypeLabel(type: DatabaseType | ''): string {
  if (type && DB_META[type]) return DB_META[type]!.name
  return type || t('initialize.pleaseSelect')
}

function toggleTheme() {
  themeStore.toggleThemeMode(themeStore.isDarkTheme ? 'light' : 'dark')
}

function selectDbType(type: DatabaseType) {
  if (dbForm.type === type) return
  dbForm.type = type
  connectionTested.value = false
  dbForm.sqlitePath = ''
  dbForm.host = ''
  dbForm.port = ''
  dbForm.username = ''
  dbForm.password = ''
  dbForm.databaseName = ''
  dbErrors.value = {}
}

function validateDbForm(): boolean {
  const next: Record<string, string> = {}
  if (!dbForm.type) {
    next.type = t('initialize.databaseTypeRequired')
  } else if (!isSqlite.value) {
    if (!dbForm.host.trim()) next.address = t('initialize.addressRequired')
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
  const sqlite = isSqlite.value
  return {
    type: dbForm.type as DatabaseType,
    sqlitePath: sqlite ? dbForm.sqlitePath.trim() : '',
    address: sqlite ? '' : remoteAddress.value,
    username: sqlite ? '' : dbForm.username.trim(),
    password: sqlite ? '' : dbForm.password,
    databaseName: sqlite ? '' : dbForm.databaseName.trim(),
  }
}

function buildTestPayload(): TestInitializeDatabaseRequest {
  return { database: buildDatabaseConfig() }
}

function buildConfirmPayload(): ConfirmInitializeRequest {
  return {
    database: buildDatabaseConfig(),
    admin: {
      username: adminForm.username.trim(),
      password: adminForm.password,
      email: adminForm.email.trim(),
      name: adminForm.name.trim(),
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
      testResult.serverVersion = data.serverVersion || ''
      testResult.latencyMs = Number(data.latencyMs) || 0
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

function onNext() {
  if (nextDisabled.value || confirming.value) return
  if (currentStep.value === 0) {
    if (!dbForm.type) {
      dbErrors.value = { type: t('initialize.databaseTypeRequired') }
      return
    }
    currentStep.value = 1
    return
  }
  if (currentStep.value === 1) {
    if (connectionTested.value) currentStep.value = 2
    return
  }
  if (currentStep.value === 2) {
    if (!validateAdminForm()) return
    handleConfirm()
  }
}

async function handleConfirm() {
  confirming.value = true
  initError.value = false

  try {
    const { data } = await confirmInitialize(buildConfirmPayload())
    confirmed.value = true
    currentStep.value = 3
    if (data?.initialized) {
      if (data.restartRequired) {
        startPolling()
      } else {
        readyForLogin.value = true
      }
    } else {
      initError.value = true
      fb.error(t('initialize.resultFailed'))
    }
  } catch {
    confirmed.value = true
    initError.value = true
    currentStep.value = 3
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
    status.supportedDatabaseTypes = data?.supportedDatabaseTypes || []
    status.version = data?.version || ''
    status.hostname = data?.hostname || ''
    status.defaultSqlitePath = data?.defaultSqlitePath || ''
    if (data?.initialized) {
      router.replace({ path: '/login' })
      return
    }
    // 默认选中推荐的 SQLite，减少一步操作
    if (!dbForm.type && status.supportedDatabaseTypes.includes(DatabaseType.DatabaseType_SQLite)) {
      dbForm.type = DatabaseType.DatabaseType_SQLite
    }
  } catch {
    // 接口不可用时保持在初始化页
  }
}

onMounted(loadInitStatus)
onBeforeUnmount(clearPoll)
</script>

<style scoped lang="scss">
.init {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  min-height: 100vh;
  min-height: 100dvh;
  width: 100%;
  background: var(--el-bg-color);
}

/* ---- 品牌区 ---- */
.init-brand {
  position: relative;
  flex: 1 1 460px;
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  gap: 36px;
  padding: 48px 52px;
  overflow: hidden;
  background: var(--m-p);
  color: var(--m-pOn);
}
.init-brand__blob {
  position: absolute;
  right: -140px;
  bottom: -170px;
  width: 440px;
  height: 440px;
  border-radius: 50%;
  background: var(--m-a);
}
.init-brand__ring {
  position: absolute;
  right: 70px;
  top: 46%;
  width: 130px;
  height: 130px;
  border-radius: 50%;
  border: 20px solid var(--m-a2);
}
.init-brand__star {
  position: absolute;
  right: 70px;
  top: 96px;
  width: 56px;
  height: 56px;
  fill: var(--m-pOn);
}
.init-brand__logo {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  font-family: var(--app-font-display);
  font-size: 1.625rem;
  font-weight: 800;
}
.init-brand__logo img {
  width: 48px;
  height: 48px;
  border-radius: 15px;
  object-fit: cover;
  transform: rotate(-6deg);
  box-shadow: 0 0 0 3px var(--m-pOn);
}
.init-brand__badge {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 12px;
  border-radius: 999px;
  background: var(--m-a);
  color: var(--m-aOn);
  font-size: 0.8125rem;
  font-weight: 800;
  transform: rotate(3deg);
}
.init-brand__copy {
  position: relative;
  max-width: 480px;
}
.init-brand__copy h1 {
  font-family: var(--app-font-display);
  font-size: clamp(2.25rem, 3.6vw, 3rem);
  line-height: 1.1;
  font-weight: 800;
  letter-spacing: -0.02em;
}
.init-brand__copy p {
  margin-top: 14px;
  font-size: 1.0625rem;
  line-height: 1.6;
  opacity: 0.94;
}
.init-brand__mstep {
  display: none;
}

.init-steps {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-width: 360px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.init-steps__btn {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  min-height: 56px;
  padding: 6px 10px;
  border: 0;
  border-radius: 18px;
  background: transparent;
  color: var(--m-pOn);
  font: inherit;
  text-align: left;
  cursor: default;
  transition: background-color 0.15s;
}
.init-steps__btn:disabled {
  opacity: 1;
}
.init-steps__btn.is-done:not(:disabled) {
  cursor: pointer;
}
.init-steps__btn.is-done:not(:disabled):hover {
  background: color-mix(in srgb, var(--m-pOn) 12%, transparent);
}
.init-steps__btn.is-on {
  background: color-mix(in srgb, var(--m-pOn) 16%, transparent);
}
.init-steps__num {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 14px;
  border: 2px solid var(--m-pOn);
  font-size: 1rem;
  font-weight: 800;
}
.init-steps__btn.is-done .init-steps__num {
  border-color: transparent;
  background: var(--m-pOn);
  color: var(--m-p);
}
.init-steps__btn.is-on .init-steps__num {
  border-color: transparent;
  background: var(--m-a);
  color: var(--m-aOn);
  transform: rotate(-6deg);
}
.init-steps__check {
  width: 20px;
  height: 20px;
  stroke-width: 3;
}
.init-steps__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.init-steps__title {
  font-size: 1rem;
  font-weight: 800;
}
.init-steps__desc {
  font-size: 0.8125rem;
  font-weight: 500;
  opacity: 0.92;
}
.init-brand__tags {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  gap: 12px 14px;
  margin: auto 0 0;
  padding: 0;
  list-style: none;
}
.init-brand__tags li {
  display: inline-flex;
  align-items: center;
  height: 44px;
  padding: 0 18px;
  border-radius: 999px;
  font-size: 1rem;
  font-weight: 800;
  box-shadow: 0 10px 24px -12px rgba(0, 0, 0, 0.35);
}
.init-brand__tags .is-0 {
  background: var(--m-pOn);
  color: var(--m-p);
  transform: rotate(-5deg);
}
.init-brand__tags .is-1 {
  background: var(--m-ink);
  color: var(--m-bg);
  transform: rotate(4deg);
}
.init-brand__tags .is-2 {
  background: var(--m-a2);
  color: #fff;
  transform: rotate(-3deg);
}

/* ---- 表单区 ---- */
.init-panel {
  position: relative;
  flex: 1 1 600px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 24px 40px 28px;
  background: var(--el-bg-color);
}
.init-panel__top {
  display: flex;
  align-items: center;
  gap: 6px;
}
.init-panel__count {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.init-panel__count b {
  color: var(--el-text-color-primary);
}
.init-panel__spacer {
  flex: 1;
}
.init-panel__chrome {
  display: flex;
  align-items: center;
  gap: 4px;
}
.init-panel__bars {
  display: flex;
  gap: 8px;
}
.init-panel__bars span {
  flex: 1;
  height: 8px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  transition: background-color 0.2s;
}
.init-panel__bars span.is-on {
  background: var(--m-p);
}
.init-panel__center {
  flex: 1;
  display: flex;
  justify-content: center;
  padding-top: 20px;
}
.init-panel__copy {
  margin: 0;
  text-align: center;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

.init-form {
  width: 100%;
  max-width: 640px;
  display: flex;
  flex-direction: column;
  gap: 22px;
}
.init-step {
  display: flex;
  flex-direction: column;
  gap: 22px;
}
.init-head {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.init-head h2 {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 2.125rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}
.init-head p {
  margin: 0;
  font-size: 0.9375rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.init-head p b {
  color: var(--el-text-color-primary);
}

/* 数据库卡片 */
.init-dbs {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 14px;
}
.init-db {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 18px;
  border: 2px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition:
    border-color 0.15s,
    box-shadow 0.15s,
    transform 0.15s;
}
.init-db:hover {
  border-color: var(--el-text-color-secondary);
  transform: translateY(-2px);
}
.init-db.is-on,
.init-db.is-on:hover {
  border-color: var(--m-p);
  box-shadow: 0 0 0 4px var(--m-pSoft);
}
.init-db__top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}
.init-db__av {
  width: 56px;
  height: 56px;
  border-radius: 18px;
  font-family: var(--app-font-display);
  font-size: 1.25rem;
}
.init-db__chk {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  border: 2px solid var(--el-border-color-light);
}
.init-db__chk.is-on {
  border-color: transparent;
  background: var(--m-p);
  color: var(--m-pOn);
}
.init-db__chk svg {
  width: 16px;
  height: 16px;
  stroke-width: 3;
}
.init-db__body {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.init-db__name {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-family: var(--app-font-display);
  font-size: 1.1875rem;
  font-weight: 800;
}
.init-db__rec {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 9px;
  border-radius: 999px;
  background: var(--m-a);
  color: var(--m-aOn);
  font-size: 0.75rem;
  font-weight: 800;
}
.init-db__sub {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--m-pInk);
}
.init-db__desc {
  font-size: 0.8125rem;
  line-height: 1.55;
  color: var(--el-text-color-secondary);
}

.init-note {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 14px 16px;
  border-radius: var(--app-radius);
  background: var(--m-a2Soft);
  color: var(--el-text-color-primary);
  font-size: 0.875rem;
  line-height: 1.55;
}
.init-note svg {
  flex: none;
  width: 20px;
  height: 20px;
  margin-top: 1px;
}
.init-note--warn {
  background: var(--m-warnSoft);
  color: var(--m-warn);
}

/* 字段 */
.init-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}
.init-grid--host {
  grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
}
.init-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.init-field__label {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.init-field__hint {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.init-fld {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 0 6px 0 16px;
  border-radius: var(--app-radius);
  border: 1.5px solid transparent;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  transition:
    border-color 0.15s,
    background-color 0.15s;
}
.init-fld:focus-within {
  border-color: var(--m-p);
  background: var(--el-bg-color);
}
.init-fld.is-error {
  border-color: var(--m-bad);
}
.init-fld__ico {
  flex: none;
  width: 20px;
  height: 20px;
}
.init-fld__input {
  flex: 1;
  min-width: 0;
  height: 48px;
  padding: 0 10px 0 0;
  border: 0;
  background: transparent;
  font: inherit;
  font-size: 0.9375rem;
  font-weight: 500;
  color: var(--el-text-color-primary);
  outline: none;
}
.init-fld__input.m-mono {
  font-size: 0.875rem;
}
.init-fld__input::placeholder {
  color: var(--el-text-color-placeholder);
}
.init-fld__eye {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.init-fld__eye:hover {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
}
.init-fld__eye svg {
  width: 20px;
  height: 20px;
}

/* 测试连接 */
.init-test {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.init-test__row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}
.init-ok {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border-radius: var(--app-radius);
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.init-ok__ico {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 34px;
  height: 34px;
  border-radius: 11px;
  background: var(--m-ok);
  color: var(--el-bg-color);
  transform: rotate(-5deg);
}
.init-ok__ico svg {
  width: 18px;
  height: 18px;
  stroke-width: 3;
}
.init-ok__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.init-ok__text b {
  font-size: 0.9375rem;
  font-weight: 800;
}
.init-ok__text span {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
}

/* 密码强度 */
.init-meter {
  display: flex;
  align-items: center;
  gap: 12px;
}
.init-meter__bars {
  display: flex;
  flex: 1;
  gap: 6px;
}
.init-meter__bars span {
  flex: 1;
  height: 6px;
  border-radius: 999px;
  background: var(--el-border-color-light);
  transition: background-color 0.2s;
}
.init-meter__bars .is-bad {
  background: var(--m-bad);
}
.init-meter__bars .is-warn {
  background: var(--m-warn);
}
.init-meter__bars .is-ok {
  background: var(--m-ok);
}
.init-meter__label {
  min-width: 64px;
  text-align: right;
  font-size: 0.8125rem;
  font-weight: 800;
  color: var(--el-text-color-secondary);
}
.init-meter__label.is-bad {
  color: var(--m-bad);
}
.init-meter__label.is-warn {
  color: var(--m-warn);
}
.init-meter__label.is-ok {
  color: var(--m-ok);
}
.init-rules {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 14px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.init-rules li,
.init-match {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.init-rules li svg,
.init-match svg {
  width: 15px;
  height: 15px;
  stroke-width: 2.6;
}
.init-rules li.is-ok,
.init-match.is-ok {
  color: var(--m-ok);
}
.init-match {
  font-weight: 700;
}
.init-match.is-bad {
  color: var(--m-bad);
}

/* 完成步 */
.init-art {
  position: relative;
  height: 150px;
}
.init-art__disc {
  position: absolute;
  left: 6px;
  top: 10px;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  background: var(--m-a);
}
.init-art__ring {
  position: absolute;
  left: 96px;
  top: 70px;
  width: 70px;
  height: 70px;
  border-radius: 50%;
  border: 12px solid var(--m-a2);
}
.init-art__tile {
  position: absolute;
  left: 30px;
  top: 26px;
  width: 96px;
  height: 96px;
  border-radius: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--m-p);
  color: var(--m-pOn);
  transform: rotate(-8deg);
  box-shadow: var(--app-shadow-brand);
}
.init-art.is-error .init-art__tile {
  background: var(--m-bad);
}
.init-art__tile svg {
  width: 50px;
  height: 50px;
  stroke-width: 2.6;
}
.init-art__tile svg.is-spin {
  width: 44px;
  height: 44px;
  stroke-width: 2.2;
  animation: init-spin 1.1s linear infinite;
}
@keyframes init-spin {
  to {
    transform: rotate(360deg);
  }
}
.init-art__tag {
  position: absolute;
  left: 190px;
  top: 40px;
  display: inline-flex;
  align-items: center;
  height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  font-size: 0.9375rem;
  font-weight: 800;
  white-space: nowrap;
  transform: rotate(4deg);
}
.init-head--done h2 {
  font-size: 2.25rem;
}
.init-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px 20px;
  margin: 0;
  padding: 18px 20px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
}
.init-summary > div {
  min-width: 0;
}
.init-summary .is-wide {
  grid-column: 1 / -1;
}
.init-summary dt {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.init-summary dd {
  margin: 4px 0 0;
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.init-summary dd.is-mono {
  font-size: 0.8125rem;
  font-weight: 600;
}
.init-next {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.init-next__label {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.init-next__links {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.init-next__link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 16px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.875rem;
  font-weight: 700;
  text-decoration: none;
  transition:
    background-color 0.15s,
    color 0.15s;
}
.init-next__link:hover {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.init-next__link svg {
  width: 16px;
  height: 16px;
}
.init-done-actions {
  display: flex;
}

/* 按钮 */
.init-btn {
  justify-content: center;
  min-height: 44px;
  padding: 0 18px;
  border-radius: 999px !important;
  font-weight: 700 !important;
}
.init-btn--outline {
  background: var(--el-bg-color) !important;
  box-shadow: inset 0 0 0 1.5px var(--el-border-color-light) !important;
}
.init-btn--outline:hover:not(:disabled) {
  color: var(--m-pInk) !important;
  box-shadow: inset 0 0 0 1.5px var(--m-p) !important;
}
.init-btn--back,
.init-btn--next {
  min-height: 52px;
  padding: 0 22px;
}
.init-btn--next {
  padding: 0 26px;
  font-size: 0.9375rem !important;
  box-shadow: var(--app-shadow-brand);
}
.init-btn--lg {
  min-height: 56px;
  padding: 0 28px;
  font-size: 1rem !important;
  box-shadow: var(--app-shadow-brand);
}

.init-foot {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 4px;
  padding-top: 18px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.init-foot__spacer {
  flex: 1;
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

/* ---- 窄屏：品牌区在上（步骤标题 + 进度条），表单卡片上浮，底部操作条固定 ---- */
@media (width < 900px) {
  .init {
    overflow-x: clip;
  }
  /* 品牌区不建立定位上下文：装饰相对整页定位，表单卡片的背景与圆角才能盖住品牌区底部 */
  .init-brand {
    position: static;
    flex: 1 1 100%;
    min-height: 0;
    padding: 16px 20px 56px;
    gap: 0;
  }
  .init-brand__blob {
    right: -60px;
    top: -70px;
    bottom: auto;
    width: 200px;
    height: 200px;
  }
  .init-brand__ring {
    display: none;
  }
  .init-brand__star,
  .init-brand__copy,
  .init-steps,
  .init-brand__tags,
  .init-brand__badge {
    display: none;
  }
  .init-brand__logo {
    font-size: 1.25rem;
    padding-right: 150px;
  }
  .init-brand__logo img {
    width: 38px;
    height: 38px;
    border-radius: 12px;
  }
  .init-brand__mstep {
    position: relative;
    display: flex;
    flex-direction: column;
    margin-top: 22px;
  }
  .init-brand__mcount {
    font-size: 0.8125rem;
    font-weight: 700;
  }
  .init-brand__mtitle {
    margin: 4px 0 14px;
    font-family: var(--app-font-display);
    font-size: 1.875rem;
    line-height: 1.15;
    font-weight: 800;
    letter-spacing: -0.01em;
  }
  .init-brand__bars {
    display: flex;
    gap: 6px;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .init-brand__bars li {
    flex: 1;
    height: 8px;
    border-radius: 999px;
    background: color-mix(in srgb, var(--m-pOn) 30%, transparent);
  }
  .init-brand__bars li.is-on {
    background: var(--m-a);
  }

  .init-panel {
    position: static;
    flex: 1 1 100%;
    margin-top: -32px;
    padding: 24px 20px 0;
    gap: 0;
    border-radius: 30px 30px 0 0;
  }
  .init-panel__top {
    position: absolute;
    top: 12px;
    right: 12px;
    z-index: 2;
    padding: 2px;
    border-radius: 999px;
    background: var(--el-bg-color);
    box-shadow: var(--app-shadow-sm);
  }
  .init-panel__count,
  .init-panel__spacer,
  .init-panel__bars,
  .init-head h2 {
    display: none;
  }
  .init-head--done h2 {
    display: block;
    font-size: 1.75rem;
  }
  .init-panel__center {
    padding-top: 0;
  }
  .init-panel__copy {
    padding: 8px 0 20px;
  }
  .init-form,
  .init-step {
    gap: 18px;
  }
  .init-head p {
    font-size: 0.875rem;
  }

  .init-dbs {
    grid-template-columns: 1fr;
    gap: 12px;
  }
  .init-db {
    position: relative;
    flex-direction: row;
    align-items: flex-start;
    gap: 14px;
    padding: 16px;
  }
  .init-db:hover {
    transform: none;
  }
  .init-db__top {
    display: contents;
  }
  .init-db__av {
    order: 0;
    width: 50px;
    height: 50px;
    border-radius: 16px;
    font-size: 1.125rem;
  }
  .init-db__body {
    order: 1;
    flex: 1;
    min-width: 0;
    gap: 3px;
  }
  .init-db__chk {
    order: 2;
  }
  .init-db__name {
    font-size: 1.0625rem;
  }
  .init-note {
    padding: 12px 14px;
    font-size: 0.8125rem;
  }

  .init-grid {
    grid-template-columns: 1fr;
  }
  .init-grid--host {
    grid-template-columns: minmax(0, 1.7fr) minmax(0, 1fr);
    gap: 10px;
  }

  .init-art {
    height: 120px;
    flex: none;
  }
  .init-art__disc {
    left: 0;
    top: 6px;
    width: 100px;
    height: 100px;
  }
  .init-art__ring {
    left: 78px;
    top: 58px;
    width: 56px;
    height: 56px;
    border-width: 10px;
  }
  .init-art__tile {
    left: 18px;
    top: 18px;
    width: 80px;
    height: 80px;
    border-radius: 26px;
  }
  .init-art__tile svg {
    width: 42px;
    height: 42px;
  }
  .init-art__tag {
    left: 150px;
    top: 34px;
    height: 36px;
    padding: 0 14px;
    font-size: 0.875rem;
  }
  .init-summary {
    padding: 14px;
    gap: 12px;
    border-radius: 18px;
  }
  .init-summary dd {
    font-size: 0.875rem;
  }
  .init-done-actions :deep(.init-btn) {
    width: 100%;
  }

  .init-foot {
    position: sticky;
    bottom: 0;
    z-index: 3;
    margin: 0 -20px;
    padding: 12px 16px calc(16px + env(safe-area-inset-bottom));
    background: var(--el-bg-color);
  }
  .init-foot__spacer {
    display: none;
  }
  .init-btn--back {
    width: 56px;
    min-height: 56px;
    padding: 0 !important;
    flex: none;
  }
  .init-btn--back .init-btn__text {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
  }
  .init-btn--next {
    flex: 1;
    min-height: 56px;
  }
}
</style>
