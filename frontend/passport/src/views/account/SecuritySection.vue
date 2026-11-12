<!-- 登录与安全：密码、邮箱、两步验证（含恢复码）、邮箱验证码登录。 -->
<template>
  <section v-if="me" class="pp-card pp-section">
    <h1 class="pp-section__title">{{ t('account.security.title') }}</h1>
    <p class="pp-section__desc">{{ t('account.security.desc') }}</p>

    <div class="pp-frow">
      <span class="pp-frow__label">
        {{ t('account.security.password') }}
        <small>{{ me.passwordUpdatedAt ? t('account.security.changedAgo', { ago: fmtAgo(me.passwordUpdatedAt) }) : '' }}</small>
      </span>
      <span><PPill tone="ok">{{ t('account.security.set') }}</PPill></span>
      <div class="pp-frow__actions"><PButton size="sm" @click="openPassword">{{ t('account.security.change') }}</PButton></div>
    </div>

    <div class="pp-frow">
      <span class="pp-frow__label">{{ t('account.security.email') }}<small>{{ t('account.security.emailUse') }}</small></span>
      <span class="email">
        <span>{{ me.email || t('home.noEmail') }}</span>
        <PPill v-if="me.email" :tone="me.emailVerified ? 'ok' : 'warn'">{{ me.emailVerified ? t('home.verified') : t('home.unverified') }}</PPill>
      </span>
      <div class="pp-frow__actions">
        <PButton size="sm" :disabled="!emailEnabled" @click="openEmail">
          {{ me.email ? (me.emailVerified ? t('account.security.changeEmail') : t('account.security.verifyEmail')) : t('account.security.bindEmail') }}
        </PButton>
      </div>
    </div>

    <div class="pp-frow">
      <span class="pp-frow__label">{{ t('account.security.totp') }}<small>{{ t('account.security.totpUse') }}</small></span>
      <span class="email">
        <PPill :tone="me.totpEnabled ? 'ok' : 'warn'">{{ me.totpEnabled ? t('home.on') : t('home.off') }}</PPill>
        <span v-if="me.totpEnabled" class="pp-hint">{{ t('account.security.recoveryLeft', { n: me.recoveryCodesLeft }) }}</span>
      </span>
      <div class="pp-frow__actions">
        <template v-if="me.totpEnabled">
          <PButton size="sm" @click="openRegen">{{ t('account.security.newRecovery') }}</PButton>
          <PButton size="sm" variant="danger" @click="openDisable">{{ t('account.security.turnOff') }}</PButton>
        </template>
        <PButton v-else size="sm" variant="primary" :loading="busy" @click="startTotp">{{ t('home.security.enable') }}</PButton>
      </div>
    </div>

    <div class="pp-frow">
      <span class="pp-frow__label">{{ t('account.security.codeLogin') }}</span>
      <span class="pp-hint">{{ codeLoginHint }}</span>
      <div class="pp-frow__actions">
        <PSwitch :model-value="me.emailCodeLogin" :disabled="!codeLoginAvailable" :label="t('account.security.codeLogin')" @update:model-value="toggleCodeLogin" />
      </div>
    </div>

    <!-- 修改密码 -->
    <PDialog v-model="pwOpen" :title="t('account.security.changePassword')">
      <form id="pw-form" class="pp-form" @submit.prevent="savePassword">
        <PField :label="t('account.security.oldPassword')">
          <input v-model="pw.old" type="password" autocomplete="current-password" required />
        </PField>
        <PField :label="t('auth.forgot.password')" :hint="t('auth.register.passwordHint', { n: session.config?.minPasswordLength || 8 })">
          <input v-model="pw.next" type="password" autocomplete="new-password" required />
        </PField>
        <PField :label="t('auth.register.confirm')" :error="pw.confirm && pw.confirm !== pw.next ? t('auth.register.mismatch') : ''">
          <input v-model="pw.confirm" type="password" autocomplete="new-password" required />
        </PField>
        <p class="pp-hint">{{ t('account.security.othersOut') }}</p>
      </form>
      <template #footer>
        <PButton @click="pwOpen = false">{{ t('common.cancel') }}</PButton>
        <PButton type="submit" form="pw-form" variant="primary" :loading="busy" :disabled="pw.confirm !== pw.next">{{ t('common.save') }}</PButton>
      </template>
    </PDialog>

    <!-- 更换 / 验证邮箱 -->
    <PDialog v-model="emailOpen" :title="t('account.security.changeEmail')">
      <form id="email-form" class="pp-form" @submit.prevent="saveEmail">
        <PField :label="t('account.security.newEmail')">
          <input v-model.trim="em.email" type="email" autocomplete="email" required />
        </PField>
        <PField :label="t('auth.code')" :hint="t('account.security.codeToNew')">
          <span class="pp-code-row">
            <input v-model.trim="em.code" inputmode="numeric" maxlength="6" class="pp-num" required />
            <PButton size="sm" :disabled="left > 0 || !em.email" :loading="sending" @click="sendEmailCode">
              {{ left > 0 ? t('common.resendIn', { n: left }) : t('common.sendCode') }}
            </PButton>
          </span>
        </PField>
        <PField :label="t('account.security.currentPassword')">
          <input v-model="em.password" type="password" autocomplete="current-password" required />
        </PField>
      </form>
      <template #footer>
        <PButton @click="emailOpen = false">{{ t('common.cancel') }}</PButton>
        <PButton type="submit" form="email-form" variant="primary" :loading="busy">{{ t('common.save') }}</PButton>
      </template>
    </PDialog>

    <!-- 开启两步验证 -->
    <PDialog v-model="setupOpen" :title="t('account.security.setupTitle')" :width="520" :close-on-mask="false">
      <div v-if="recovery.length" class="pp-form">
        <p class="pp-note">{{ t('account.security.recoveryIntro') }}</p>
        <div class="codes pp-mono">
          <span v-for="c in recovery" :key="c">{{ c }}</span>
        </div>
        <PButton block @click="copyCodes">{{ t('common.copy') }}</PButton>
      </div>
      <form v-else-if="setup" id="totp-form" class="pp-form" @submit.prevent="confirmTotp">
        <p class="pp-hint">{{ t('account.security.scan') }}</p>
        <img :src="setup.qrCode" :alt="t('account.security.qrAlt')" class="qr" width="220" height="220" />
        <PField :label="t('account.security.manualKey')">
          <span class="secret pp-mono">{{ setup.secret }}</span>
        </PField>
        <PField :label="t('auth.mfa.code')">
          <input v-model.trim="totpCode" inputmode="numeric" maxlength="6" autocomplete="one-time-code" class="pp-num" required />
        </PField>
      </form>
      <template #footer>
        <PButton v-if="recovery.length" variant="primary" @click="closeSetup">{{ t('account.security.saved') }}</PButton>
        <template v-else>
          <PButton @click="closeSetup">{{ t('common.cancel') }}</PButton>
          <PButton type="submit" form="totp-form" variant="primary" :loading="busy">{{ t('account.security.confirmEnable') }}</PButton>
        </template>
      </template>
    </PDialog>

    <!-- 关闭两步验证 -->
    <PDialog v-model="disableOpen" :title="t('account.security.disableTitle')">
      <form id="disable-form" class="pp-form" @submit.prevent="disableTotp">
        <PField :label="t('account.security.currentPassword')">
          <input v-model="dis.password" type="password" autocomplete="current-password" required />
        </PField>
        <PField :label="t('account.security.codeOrRecovery')">
          <input v-model.trim="dis.code" autocomplete="one-time-code" required />
        </PField>
      </form>
      <template #footer>
        <PButton @click="disableOpen = false">{{ t('common.cancel') }}</PButton>
        <PButton type="submit" form="disable-form" variant="danger" :loading="busy">{{ t('account.security.turnOff') }}</PButton>
      </template>
    </PDialog>

    <!-- 重新生成恢复码 -->
    <PDialog v-model="regenOpen" :title="t('account.security.newRecovery')">
      <div v-if="recovery.length" class="pp-form">
        <p class="pp-note">{{ t('account.security.recoveryIntro') }}</p>
        <div class="codes pp-mono">
          <span v-for="c in recovery" :key="c">{{ c }}</span>
        </div>
        <PButton block @click="copyCodes">{{ t('common.copy') }}</PButton>
      </div>
      <form v-else id="regen-form" class="pp-form" @submit.prevent="regenerate">
        <p class="pp-hint">{{ t('account.security.regenHint') }}</p>
        <PField :label="t('auth.mfa.code')">
          <input v-model.trim="totpCode" inputmode="numeric" maxlength="6" autocomplete="one-time-code" class="pp-num" required />
        </PField>
      </form>
      <template #footer>
        <PButton v-if="recovery.length" variant="primary" @click="closeRegen">{{ t('account.security.saved') }}</PButton>
        <template v-else>
          <PButton @click="regenOpen = false">{{ t('common.cancel') }}</PButton>
          <PButton type="submit" form="regen-form" variant="primary" :loading="busy">{{ t('common.confirm') }}</PButton>
        </template>
      </template>
    </PDialog>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import PButton from '@passport/components/PButton.vue'
import PDialog from '@passport/components/PDialog.vue'
import PField from '@passport/components/PField.vue'
import PPill from '@passport/components/PPill.vue'
import PSwitch from '@passport/components/PSwitch.vue'
import * as api from '@passport/api/passport'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { errorText, fmtAgo } from '@passport/utils/format'
import { useCountdown } from '@passport/utils/countdown'
import type { SetupPassportTotpResponse } from '@/types/v1/passport'

const { t } = useI18n()
const session = useSessionStore()
const ui = useUiStore()
const { left, start } = useCountdown()

const me = computed(() => session.me)
const emailEnabled = computed(() => !!session.config?.emailEnabled)
const codeLoginAvailable = computed(() => !!session.config?.emailLoginEnabled && !!me.value?.email && !!me.value?.emailVerified)
const codeLoginHint = computed(() => {
  if (!session.config?.emailLoginEnabled) return t('account.security.codeLoginOff')
  if (!me.value?.email || !me.value.emailVerified) return t('account.security.codeLoginNeedsEmail')
  return t('account.security.codeLoginHint')
})

const busy = ref(false)
const sending = ref(false)

const pwOpen = ref(false)
const pw = reactive({ old: '', next: '', confirm: '' })
const emailOpen = ref(false)
const em = reactive({ email: '', code: '', password: '' })
const setupOpen = ref(false)
const setup = ref<SetupPassportTotpResponse | null>(null)
const totpCode = ref('')
const recovery = ref<string[]>([])
const disableOpen = ref(false)
const dis = reactive({ password: '', code: '' })
const regenOpen = ref(false)

const fail = (e: unknown) => ui.toast(errorText(e), 'bad')

const openPassword = () => {
  Object.assign(pw, { old: '', next: '', confirm: '' })
  pwOpen.value = true
}
const savePassword = async () => {
  if (pw.next !== pw.confirm) return
  busy.value = true
  try {
    await api.changePassword(pw.old, pw.next)
    pwOpen.value = false
    ui.toast(t('account.security.passwordChanged'), 'ok')
    await session.loadMe()
  } catch (e) {
    fail(e)
  } finally {
    busy.value = false
  }
}

const openEmail = () => {
  Object.assign(em, { email: me.value?.emailVerified ? '' : me.value?.email || '', code: '', password: '' })
  emailOpen.value = true
}
const sendEmailCode = async () => {
  sending.value = true
  try {
    const res = await api.sendEmailChangeCode(em.email)
    start(res.resendAfterSeconds || 60)
    ui.toast(t('common.codeSent', { n: Math.round((res.expiresInSeconds || 300) / 60) }), 'ok')
  } catch (e) {
    fail(e)
  } finally {
    sending.value = false
  }
}
const saveEmail = async () => {
  busy.value = true
  try {
    session.applyMe(await api.changeEmail({ email: em.email, code: em.code, password: em.password }))
    emailOpen.value = false
    ui.toast(t('common.saved'), 'ok')
  } catch (e) {
    fail(e)
  } finally {
    busy.value = false
  }
}

const toggleCodeLogin = async (value: boolean) => {
  try {
    session.applyMe(await api.setEmailLogin(value))
  } catch (e) {
    fail(e)
  }
}

const startTotp = async () => {
  busy.value = true
  try {
    setup.value = await api.setupTotp()
    totpCode.value = ''
    recovery.value = []
    setupOpen.value = true
  } catch (e) {
    fail(e)
  } finally {
    busy.value = false
  }
}
const confirmTotp = async () => {
  busy.value = true
  try {
    recovery.value = (await api.enableTotp(totpCode.value)).codes || []
    await session.loadMe()
  } catch (e) {
    fail(e)
  } finally {
    busy.value = false
  }
}
const closeSetup = () => {
  setupOpen.value = false
  setup.value = null
  recovery.value = []
}

const openDisable = () => {
  Object.assign(dis, { password: '', code: '' })
  disableOpen.value = true
}
const disableTotp = async () => {
  busy.value = true
  try {
    session.applyMe(await api.disableTotp(dis.password, dis.code))
    disableOpen.value = false
    ui.toast(t('account.security.totpOffDone'), 'ok')
  } catch (e) {
    fail(e)
  } finally {
    busy.value = false
  }
}

const openRegen = () => {
  totpCode.value = ''
  recovery.value = []
  regenOpen.value = true
}
const regenerate = async () => {
  busy.value = true
  try {
    recovery.value = (await api.regenerateRecoveryCodes(totpCode.value)).codes || []
    await session.loadMe()
  } catch (e) {
    fail(e)
  } finally {
    busy.value = false
  }
}
const closeRegen = () => {
  regenOpen.value = false
  recovery.value = []
}

const copyCodes = async () => {
  try {
    await navigator.clipboard.writeText(recovery.value.join('\n'))
    ui.toast(t('common.copied'), 'ok')
  } catch {
    ui.toast(t('errors.UNKNOWN'), 'bad')
  }
}
</script>

<style scoped>
.email {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  font-weight: 700;
  overflow-wrap: anywhere;
}
.qr {
  align-self: center;
  width: 220px;
  height: 220px;
  border-radius: 16px;
  border: 1px solid var(--m-line);
}
.secret {
  padding: 10px 12px;
  border-radius: 12px;
  background: var(--m-s2);
  font-size: 14px;
  letter-spacing: 0.08em;
  overflow-wrap: anywhere;
}
.codes {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  padding: 14px;
  border-radius: 14px;
  background: var(--m-s2);
  font-size: 15px;
  font-weight: 700;
  text-align: center;
}
</style>
