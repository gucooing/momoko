<!-- 通行证设置：注册与登录（存在 Momoko 系统配置里）、通行证服务的只读信息（配置文件）。 -->
<template>
  <FormDialog :model-value="modelValue" :title="t('passportAdmin.settings.title')" :width="760" :show-footer="false" @update:model-value="emit('update:modelValue', $event)">
    <div v-if="loading && !info" class="pset__hint">{{ t('passportAdmin.common.loading') }}</div>
    <div v-else class="pset">
      <section class="pset__section">
        <h3 class="pset__title">{{ t('passportAdmin.settings.general') }}</h3>
        <label class="pset__field">
          <span class="pset__label">{{ t('passportAdmin.settings.siteName') }}</span>
          <UInput v-model="form.siteName" :placeholder="t('passportAdmin.settings.siteNamePlaceholder')" maxlength="40" />
        </label>
        <div class="pset__field">
          <span class="pset__label">{{ t('passportAdmin.settings.registerMode') }}</span>
          <div class="pset__seg" role="radiogroup" :aria-label="t('passportAdmin.settings.registerMode')">
            <button
              v-for="m in MODES"
              :key="m"
              type="button"
              role="radio"
              class="pset__seg-btn"
              :class="{ 'is-on': form.registerMode === m }"
              :aria-checked="form.registerMode === m"
              @click="form.registerMode = m"
            >
              {{ t(`passportAdmin.settings.modes.${m}`) }}
            </button>
          </div>
        </div>
        <p v-if="info && !info.emailConfigured" class="pset__warn">{{ t('passportAdmin.settings.emailMissing') }}</p>
        <div class="pset__row">
          <span class="pset__label">{{ t('passportAdmin.settings.registerEmail') }}</span>
          <AppSwitch v-model="form.registerEmailVerification" />
        </div>
        <div class="pset__row">
          <span class="pset__label">{{ t('passportAdmin.settings.emailLogin') }}</span>
          <AppSwitch v-model="form.emailLoginEnabled" />
        </div>
        <label class="pset__field">
          <span class="pset__label">{{ t('passportAdmin.settings.sessionDays') }}</span>
          <UInput v-model.number="form.sessionDays" type="number" min="1" max="365" class="pset__num" />
          <span class="pset__hint">{{ t('passportAdmin.settings.sessionDaysHint') }}</span>
        </label>
        <div class="pset__actions">
          <UButton :loading="saving" @click="save">{{ t('passportAdmin.settings.save') }}</UButton>
        </div>
      </section>

      <section v-if="info" class="pset__section">
        <h3 class="pset__title">{{ t('passportAdmin.settings.server') }}</h3>
        <p v-if="!info.serverEnabled" class="pset__warn">{{ t('passportAdmin.settings.serverOff') }}</p>
        <dl class="pset__dl">
          <div><dt>{{ t('passportAdmin.settings.listen') }}</dt><dd class="m-mono">{{ info.listenAddr }}</dd></div>
          <div><dt>{{ t('passportAdmin.settings.publicUrl') }}</dt><dd class="m-mono">{{ info.publicUrl || t('passportAdmin.settings.publicUrlEmpty') }}</dd></div>
          <div><dt>{{ t('passportAdmin.settings.avatar') }}</dt><dd>{{ info.avatarUploadEnabled ? t('passportAdmin.settings.avatarOn') : t('passportAdmin.settings.avatarOff') }}</dd></div>
        </dl>
        <p class="pset__hint">{{ t('passportAdmin.settings.configHint') }}</p>
      </section>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getPassportSettings, updatePassportSettings } from '@/api/passportAdmin'
import type { GetPassportSettingsResponse } from '@/types/v1/passport_admin'
import { feedback } from '@/utils/feedback'
import { notifyError } from '../shared'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [info: GetPassportSettingsResponse] }>()

const MODES = ['open', 'invite_only', 'closed']

const { t } = useI18n()

const info = ref<GetPassportSettingsResponse>()
const loading = ref(false)
const saving = ref(false)
const form = reactive({
  siteName: '',
  registerMode: 'open',
  registerEmailVerification: false,
  emailLoginEnabled: true,
  sessionDays: 30,
})

const fill = (res: GetPassportSettingsResponse) => {
  info.value = res
  const s = res.settings
  if (!s) return
  form.siteName = s.siteName
  form.registerMode = s.registerMode || 'open'
  form.registerEmailVerification = s.registerEmailVerification
  form.emailLoginEnabled = s.emailLoginEnabled
  form.sessionDays = s.sessionDays || 30
}

const load = async () => {
  loading.value = true
  try {
    fill((await getPassportSettings()).data)
  } catch (e) {
    notifyError(e)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    void load()
  },
  { immediate: true },
)

const save = async () => {
  saving.value = true
  try {
    const res = await updatePassportSettings({
      siteName: form.siteName.trim(),
      registerMode: form.registerMode,
      registerEmailVerification: form.registerEmailVerification,
      emailLoginEnabled: form.emailLoginEnabled,
      sessionDays: Number(form.sessionDays) || 30,
    })
    fill(res.data)
    emit('saved', res.data)
    feedback.success(t('passportAdmin.settings.saved'))
  } catch (e) {
    notifyError(e)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.pset {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.pset__section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.pset__section + .pset__section {
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.pset__title {
  margin: 0;
  font-size: 16px;
  font-weight: 800;
}
.pset__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.pset__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 44px;
}
.pset__label {
  font-size: 13px;
  font-weight: 700;
}
.pset__hint {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.pset__warn {
  margin: 0;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-warnSoft);
  color: var(--m-ink);
  font-size: 13px;
}
.pset__num {
  max-width: 160px;
}
.pset__seg {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.pset__seg-btn {
  min-height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
}
.pset__seg-btn.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.pset__actions {
  display: flex;
  justify-content: flex-end;
}
.pset__dl {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.pset__dl dt {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.pset__dl dd {
  margin: 0;
  font-size: 13px;
  font-weight: 700;
  word-break: break-all;
}
.pset__hint:only-child {
  padding: 24px 0;
  text-align: center;
}
@media (width < 640px) {
  .pset__dl {
    grid-template-columns: 1fr;
  }
}
</style>
