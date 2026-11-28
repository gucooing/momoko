<!-- OIDC 服务端配置卡（第二期，原弹窗改为页内卡片）：贴纸标题 + 启用开关 → Issuer（当前域名一键填）+ 三个有效期（分钟）
     → 右侧端点列表（逐条复制）→ 底部签名密钥信息 + 轮换签名密钥 + 保存配置。
     挂载即拉取配置；加载/保存后 emit change(config) 供父级（客户端配置展示、标题状态）使用。 -->
<template>
  <section class="oc-card" :aria-label="t('system.oidc.configTitle')">
    <header class="oc-card__head">
      <span class="m-sticker m-sticker--a oc-card__icon">
        <component :is="menuStore.iconComponents['HOutline:IdentificationIcon']" />
      </span>
      <span class="oc-card__title">
        <b>{{ t('system.oidcx.heroTitle') }}</b>
        <span>{{ t('system.oidcx.heroDesc') }}</span>
      </span>
      <label class="oc-card__switch">
        <span>{{ t('system.common.enabled') }}</span>
        <AppSwitch v-model="form.enabled" :disabled="!canEdit" :aria-label="t('system.oidc.enableServer')" />
      </label>
    </header>

    <div class="oc-card__grid">
      <div class="oc-card__form">
        <div class="app-field">
          <label class="app-label" for="oc-issuer">{{ t('system.oidc.issuerUrl') }}</label>
          <div class="oc-inline">
            <span class="oc-fld">
              <component :is="menuStore.iconComponents['HOutline:GlobeAltIcon']" />
              <input
                id="oc-issuer"
                v-model="form.issuerUrl"
                class="m-mono"
                type="url"
                :disabled="!canEdit"
                placeholder="https://example.com"
              />
            </span>
            <UButton color="neutral" variant="soft" :disabled="!canEdit" @click="useCurrentOrigin">
              {{ t('system.oidc.useCurrentOrigin') }}
            </UButton>
          </div>
          <span class="oc-hint">{{ t('system.oidcx.issuerHint') }}</span>
        </div>
        <div class="oc-ttls">
          <div v-for="f in ttlFields" :key="f.key" class="app-field">
            <label class="app-label" :for="`oc-${f.key}`">{{ f.label }}</label>
            <span class="oc-fld oc-fld--num">
              <input
                :id="`oc-${f.key}`"
                v-model.number="minutes[f.key]"
                class="m-num"
                type="number"
                :min="f.min"
                :max="f.max"
                :disabled="!canEdit"
              />
              <span class="oc-unit">{{ t('system.oidcx.minutes') }}</span>
            </span>
            <span class="oc-hint">{{ f.hint }}</span>
          </div>
        </div>
      </div>

      <div class="oc-card__eps">
        <div class="oc-eps__head">
          <span class="oc-label">{{ t('system.oidcx.endpoints') }}</span>
          <span class="oc-hint">{{ t('system.oidcx.endpointsHint') }}</span>
        </div>
        <div v-for="item in endpointItems" :key="item.key" class="oc-ep">
          <span class="oc-ep__text">
            <span class="oc-ep__label">{{ item.label }}</span>
            <span class="oc-ep__url m-mono">{{ item.value }}</span>
          </span>
          <AppIconButton
            :icon="copiedKey === item.key ? 'HOutline:CheckIcon' : 'HOutline:ClipboardDocumentIcon'"
            :label="t('system.oidcx.copyEndpoint', { name: item.label })"
            :class="{ 'is-copied': copiedKey === item.key }"
            :box="menuStore.isMobile ? 44 : 36"
            @click="copyText(item.key, item.value)"
          />
        </div>
        <p v-if="!endpointItems.length" class="oc-hint">{{ t('system.common.loading') }}</p>
      </div>
    </div>

    <footer class="oc-card__foot">
      <span class="oc-sign">
        <component :is="menuStore.iconComponents['HOutline:ShieldCheckIcon']" />
        <span>
          {{ t('system.oidcx.signAlg') }} <b class="m-mono">{{ signingKey?.algorithm || 'RS256' }}</b>
          <template v-if="signingKey?.keyId">
            · {{ t('system.oidcx.signKey') }} <b class="m-mono">kid {{ signingKey.keyId.slice(0, 10) }}</b>
          </template>
          <template v-if="signingKey?.rotatedAt">
            · {{ t('system.oidcx.rotatedAt', { date: fmtDate(signingKey.rotatedAt) }) }}
          </template>
          <template v-if="!signingKey?.keyId"> · {{ t('system.oidcx.keyNotGenerated') }}</template>
        </span>
      </span>
      <span class="oc-card__spacer" />
      <UButton
        color="neutral"
        variant="outline"
        icon="i-lucide-refresh-cw"
        :loading="rotating"
        :disabled="!canEdit"
        @click="confirmRotate"
      >
        {{ t('system.oidcx.rotateKey') }}
      </UButton>
      <UButton color="primary" icon="i-lucide-check" :loading="saving" :disabled="!canEdit" @click="save">
        {{ t('system.oidc.saveConfig') }}
      </UButton>
    </footer>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { getOIDCConfig, updateOIDCConfig, rotateOIDCSigningKey } from '@/api/oidc'
import { Dialog } from '@/utils/dialog'
import type { OIDCConfig, OIDCSigningKey } from '@/types/v1/oidc'

defineOptions({ name: 'OIDCConfigCard' })

defineProps<{ canEdit: boolean }>()
const emit = defineEmits<{ change: [config: OIDCConfig] }>()
const { t } = useI18n()
const menuStore = useMenuStore()

const saving = ref(false)
const rotating = ref(false)
const config = ref<OIDCConfig | null>(null)
const signingKey = ref<OIDCSigningKey | null>(null)
const copiedKey = ref('')

const form = ref({ enabled: false, issuerUrl: '' })
// 后端以秒存储，界面按分钟编辑
const minutes = reactive<Record<'at' | 'idt' | 'code', number>>({ at: 60, idt: 60, code: 5 })

const ttlFields = computed(() => [
  { key: 'at' as const, label: t('system.oidcx.atTtl'), hint: t('system.oidcx.atHint'), min: 1, max: 1440 },
  { key: 'idt' as const, label: t('system.oidcx.idtTtl'), hint: t('system.oidcx.idtHint'), min: 1, max: 1440 },
  { key: 'code' as const, label: t('system.oidcx.codeTtl'), hint: t('system.oidcx.codeHint'), min: 1, max: 30 },
])

const endpointItems = computed(() => {
  const e = config.value?.endpoints
  if (!e) return []
  return [
    { key: 'disc', label: 'Discovery', value: e.discoveryUrl },
    { key: 'jwks', label: 'JWKS', value: e.jwksUrl },
    { key: 'auth', label: t('system.oidcx.epAuthorize'), value: e.authorizeUrl },
    { key: 'token', label: t('system.oidcx.epToken'), value: e.tokenUrl },
    { key: 'ui', label: t('system.oidcx.epUserinfo'), value: e.userinfoUrl },
  ].filter((item) => item.value)
})

const fmtDate = (v: Date) => dayjs(v).format('YYYY-MM-DD')

const setForm = (value: OIDCConfig) => {
  config.value = value
  signingKey.value = value.signingKey ?? null
  form.value = { enabled: value.enabled, issuerUrl: value.issuerUrl }
  minutes.at = Math.max(1, Math.round(Number(value.accessTokenTtlSeconds) / 60))
  minutes.idt = Math.max(1, Math.round(Number(value.idTokenTtlSeconds) / 60))
  minutes.code = Math.max(1, Math.round(Number(value.authorizationCodeTtlSeconds) / 60))
  emit('change', value)
}

const loadConfig = async () => {
  const { data } = await getOIDCConfig()
  if (data?.config) setForm(data.config)
}

const useCurrentOrigin = () => {
  form.value.issuerUrl = window.location.origin
}

const save = async () => {
  saving.value = true
  try {
    const { data } = await updateOIDCConfig({
      enabled: form.value.enabled,
      issuerUrl: form.value.issuerUrl,
      accessTokenTtlSeconds: Math.round(minutes.at * 60),
      idTokenTtlSeconds: Math.round(minutes.idt * 60),
      authorizationCodeTtlSeconds: Math.round(minutes.code * 60),
    })
    if (data?.config) setForm(data.config)
    feedback.success(t('system.oidc.saveSuccess'))
  } finally {
    saving.value = false
  }
}

const confirmRotate = () => {
  Dialog.info({
    showCancelButton: true,
    content: t('system.oidcx.rotateConfirm'),
    confirmText: t('system.oidcx.rotateKey'),
    cancelText: t('common.cancel'),
    onConfirm: async () => {
      rotating.value = true
      try {
        const { data } = await rotateOIDCSigningKey()
        if (data?.signingKey) signingKey.value = data.signingKey
        feedback.success(t('system.oidcx.rotateSuccess'))
      } finally {
        rotating.value = false
      }
    },
  })
}

let copiedTimer: ReturnType<typeof setTimeout> | undefined
const copyText = async (key: string, value: string) => {
  await navigator.clipboard.writeText(value)
  copiedKey.value = key
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => (copiedKey.value = ''), 1600)
  feedback.success(t('system.oidc.copied'))
}

onMounted(loadConfig)
onBeforeUnmount(() => copiedTimer && clearTimeout(copiedTimer))

defineExpose({ reload: loadConfig })
</script>

<style scoped lang="scss">
.oc-card {
  display: flex;
  flex-direction: column;
  gap: 22px;
  padding: 24px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.oc-card__head {
  display: flex;
  align-items: center;
  gap: 16px;
}
.oc-card__icon {
  flex: none;
  width: 52px;
  height: 52px;
  border-radius: 17px;
  transform: rotate(-5deg);
}
.oc-card__icon :deep(svg) {
  width: 26px;
  height: 26px;
}
.oc-card__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.oc-card__title b {
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.oc-card__title span {
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.oc-card__switch {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  min-height: 44px;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  cursor: pointer;
}
.oc-card__grid {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  align-items: flex-start;
}
.oc-card__form {
  flex: 1 1 420px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.oc-inline {
  display: flex;
  align-items: center;
  gap: 8px;
}
/* 「当前域名」与输入框同高 */
.oc-inline > :deep(button) {
  flex: none;
  min-height: 44px;
}
.oc-fld {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 0 14px;
  border-radius: var(--app-radius-sm);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  transition: border-color 0.15s;
}
.oc-fld:focus-within {
  border-color: var(--m-p);
}
.oc-fld :deep(svg) {
  flex: none;
  width: 16px;
  height: 16px;
}
.oc-fld input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.8125rem;
}
.oc-fld--num {
  padding-right: 0;
  overflow: hidden;
}
.oc-fld--num input {
  font-family: var(--app-font-display);
  font-size: 1rem;
  font-weight: 800;
}
.oc-unit {
  align-self: stretch;
  display: flex;
  align-items: center;
  padding: 0 12px;
  background: var(--el-fill-color-light);
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.oc-hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
.oc-ttls {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}
.oc-card__eps {
  flex: 1 1 400px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.oc-eps__head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.oc-label {
  flex: 1;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.oc-ep {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 6px 8px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.oc-ep__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.oc-ep__label {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.oc-ep__url {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.oc-ep :deep(.is-copied) {
  color: var(--m-ok);
}
.oc-card__foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  padding-top: 18px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.oc-card__foot > :deep(button) {
  min-height: 40px;
}
.oc-sign {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  min-width: 0;
}
.oc-sign :deep(svg) {
  flex: none;
  width: 16px;
  height: 16px;
}
.oc-sign b {
  color: var(--el-text-color-primary);
}
.oc-card__spacer {
  flex: 1;
}
@media (width <= 768px) {
  .oc-card {
    padding: 18px 16px;
    gap: 18px;
    border-radius: var(--app-radius-lg);
  }
  .oc-card__head {
    flex-wrap: wrap;
    gap: 12px;
  }
  .oc-card__icon {
    width: 44px;
    height: 44px;
  }
  .oc-card__title b {
    font-size: 1.0625rem;
  }
  .oc-card__switch {
    margin-left: auto;
  }
  .oc-card__form,
  .oc-card__eps {
    flex-basis: 100%;
  }
  .oc-ttls {
    grid-template-columns: 1fr;
    gap: 0;
  }
  /* 手机：有效期按「标题+说明 | 输入框」一行一项 */
  .oc-ttls > .app-field {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 116px;
    grid-template-areas: 'label fld' 'hint fld';
    align-items: center;
    column-gap: 12px;
    row-gap: 2px;
    padding: 12px 0;
    border-top: 1px solid var(--el-border-color-lighter);
  }
  .oc-ttls > .app-field:first-child {
    border-top: none;
  }
  .oc-ttls > .app-field > .app-label {
    grid-area: label;
    margin: 0;
  }
  .oc-ttls > .app-field > .oc-fld {
    grid-area: fld;
  }
  .oc-ttls > .app-field > .oc-hint {
    grid-area: hint;
  }
  .oc-card__foot > :deep(button) {
    flex: 1 1 100%;
    justify-content: center;
    min-height: 44px;
  }
  .oc-card__spacer {
    display: none;
  }
}
</style>
