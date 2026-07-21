<!-- frps 服务端配置（重写）：FormDialog + 令牌字段 + AppSwitch。保存后 emit saved。 -->
<template>
  <FormDialog
    v-model="visible"
    :title="t('tools.tunnel.frps.title')"
    :width="640"
    :loading="saving"
    @confirm="save"
  >
    <p class="frps-intro">{{ t('tools.tunnel.frps.intro') }}</p>
    <div v-if="loading" class="frps-loading">{{ t('tools.tunnel.frps.loading') }}</div>
    <div v-else class="frps-form">
      <div class="app-field frps-form__switch frps-form__full">
        <div>
          <label class="app-label">{{ t('tools.tunnel.frps.isEnable') }}</label>
        </div>
        <AppSwitch v-model="form.isEnable" />
      </div>
      <div class="app-field">
        <label class="app-label">{{ t('tools.tunnel.frps.bindAddr') }}</label>
        <input v-model="form.bindAddr" class="app-input" placeholder="0.0.0.0" />
      </div>
      <div class="app-field">
        <label class="app-label">{{ t('tools.tunnel.frps.bindPort') }}</label>
        <input v-model.number="form.bindPort" type="number" min="0" max="65535" class="app-input" />
      </div>
      <div class="app-field frps-form__full">
        <label class="app-label">{{ t('tools.tunnel.frps.serverAddr') }}</label>
        <input v-model="form.serverAddr" class="app-input" :placeholder="t('tools.tunnel.frps.serverAddrPlaceholder')" />
      </div>
      <div class="app-field">
        <label class="app-label">{{ t('tools.tunnel.frps.vhostHttpPort') }}</label>
        <input v-model.number="form.vhostHttpPort" type="number" min="0" max="65535" class="app-input" />
      </div>
      <div class="app-field">
        <label class="app-label">{{ t('tools.tunnel.frps.vhostHttpsPort') }}</label>
        <input v-model.number="form.vhostHttpsPort" type="number" min="0" max="65535" class="app-input" />
      </div>
      <div class="app-field">
        <label class="app-label">{{ t('tools.tunnel.frps.kcpBindPort') }}</label>
        <input v-model.number="form.kcpBindPort" type="number" min="0" max="65535" class="app-input" />
      </div>
      <div class="app-field">
        <label class="app-label">{{ t('tools.tunnel.frps.quicBindPort') }}</label>
        <input v-model.number="form.quicBindPort" type="number" min="0" max="65535" class="app-input" />
      </div>
      <div class="app-field frps-form__full">
        <label class="app-label">{{ t('tools.tunnel.frps.subdomainHost') }}</label>
        <input v-model="form.subdomainHost" class="app-input" placeholder="frps.example.com" />
      </div>
      <div class="app-field frps-form__full">
        <label class="app-label">{{ t('tools.tunnel.frps.sampleInterval') }}</label>
        <div class="frps-row">
          <input v-model.number="form.statSampleInterval" type="number" min="5" max="3600" class="app-input" style="max-width: 140px" />
          <span class="frps-hint">{{ t('tools.tunnel.frps.portHint') }}</span>
        </div>
      </div>
    </div>
    <template #footer="{ close }">
      <UButton color="neutral" variant="soft" @click="close">{{ t('tools.tunnel.cancel') }}</UButton>
      <UButton color="primary" :loading="saving" @click="save">{{ t('tools.tunnel.frps.save') }}</UButton>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getFrpsConfig, updateFrpsConfig } from '@/api/tunnel'
import type { FrpsConfig } from '@/types/v1/tunnel'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [config: FrpsConfig] }>()

const { t } = useI18n()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})

const loading = ref(false)
const saving = ref(false)

const defaultForm = (): FrpsConfig => ({
  isEnable: false,
  bindAddr: '0.0.0.0',
  bindPort: 7000,
  vhostHttpPort: 0,
  vhostHttpsPort: 0,
  kcpBindPort: 0,
  quicBindPort: 0,
  subdomainHost: '',
  statSampleInterval: 30,
  serverAddr: '',
})

const form = ref<FrpsConfig>(defaultForm())

const load = async () => {
  loading.value = true
  try {
    const { data } = await getFrpsConfig()
    if (data?.config) form.value = { ...defaultForm(), ...data.config }
  } finally {
    loading.value = false
  }
}

const save = async () => {
  saving.value = true
  try {
    const { data } = await updateFrpsConfig({ config: form.value })
    feedback.success(t('tools.tunnel.frps.saveSuccess'))
    if (data?.config) emit('saved', data.config)
    visible.value = false
  } finally {
    saving.value = false
  }
}

watch(visible, (open) => {
  if (open) void load()
})
</script>

<style scoped lang="scss">
.frps-intro {
  margin: 0 0 12px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.frps-loading {
  padding: 24px;
  text-align: center;
  color: var(--el-text-color-placeholder);
  font-size: 0.8125rem;
}
.frps-form {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}
.frps-form__full { grid-column: 1 / -1; }
.frps-form__switch {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
}
.frps-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.frps-hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
@media (width <= 768px) {
  .frps-form { grid-template-columns: 1fr; }
  .frps-form__full { grid-column: 1; }
}
</style>
