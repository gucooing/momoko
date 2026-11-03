<!-- frps 服务端配置：运行状态条 → 分组（基础 / HTTP·HTTPS 虚拟主机 / 传输协议 / 统计），每组贴纸小标题 + 两列表单（手机单列）。
     保存后 emit saved。 -->
<template>
  <FormDialog
    v-model="visible"
    :title="t('tools.tunnel.frps.title')"
    :width="680"
    :loading="saving"
    @confirm="save"
  >
    <div v-if="loading" class="frps-loading">{{ t('tools.tunnel.frps.loading') }}</div>
    <div v-else class="frps">
      <div class="frps-status" :class="{ 'is-on': status?.running }">
        <span class="frps-status__dot" />
        <span class="frps-status__text">{{ statusText }}</span>
        <span v-if="status?.version" class="frps-status__ver m-mono">frp v{{ status.version }}</span>
      </div>

      <section class="frps-sec">
        <h4 class="frps-sec__title">
          <span class="m-sticker m-sticker--p frps-sec__ico"><component :is="menuStore.iconComponents['HOutline:ServerStackIcon']" /></span>
          {{ t('tools.tunnel.frps.sectionBasic') }}
        </h4>
        <div class="frps-grid">
          <label class="frps-switch frps-full">
            <span class="frps-switch__text">
              <b>{{ t('tools.tunnel.frps.isEnable') }}</b>
              <small>{{ t('tools.tunnel.frps.isEnableDesc') }}</small>
            </span>
            <AppSwitch v-model="form.isEnable" />
          </label>
          <div class="app-field">
            <label class="app-label">{{ t('tools.tunnel.frps.bindAddr') }}</label>
            <input v-model="form.bindAddr" class="app-input" placeholder="0.0.0.0" />
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('tools.tunnel.frps.bindPort') }}</label>
            <input v-model.number="form.bindPort" type="number" min="0" max="65535" class="app-input" />
          </div>
          <div class="app-field frps-full">
            <label class="app-label">{{ t('tools.tunnel.frps.serverAddr') }}</label>
            <input v-model="form.serverAddr" class="app-input" :placeholder="t('tools.tunnel.frps.serverAddrPlaceholder')" />
            <span class="frps-hint">{{ t('tools.tunnel.frpsDialog.serverAddrDesc') }}</span>
          </div>
        </div>
      </section>

      <section class="frps-sec">
        <h4 class="frps-sec__title">
          <span class="m-sticker m-sticker--a2 frps-sec__ico"><component :is="menuStore.iconComponents['HOutline:GlobeAltIcon']" /></span>
          {{ t('tools.tunnel.frps.sectionVhost') }}
          <span class="frps-sec__hint">{{ t('tools.tunnel.frps.portHint') }}</span>
        </h4>
        <div class="frps-grid">
          <div class="app-field">
            <label class="app-label">{{ t('tools.tunnel.frps.vhostHttpPort') }}</label>
            <input v-model.number="form.vhostHttpPort" type="number" min="0" max="65535" class="app-input" />
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('tools.tunnel.frps.vhostHttpsPort') }}</label>
            <input v-model.number="form.vhostHttpsPort" type="number" min="0" max="65535" class="app-input" />
          </div>
          <div class="app-field frps-full">
            <label class="app-label">{{ t('tools.tunnel.frps.subdomainHost') }}</label>
            <input v-model="form.subdomainHost" class="app-input" placeholder="frps.example.com" />
          </div>
        </div>
      </section>

      <section class="frps-sec">
        <h4 class="frps-sec__title">
          <span class="m-sticker m-sticker--a frps-sec__ico"><component :is="menuStore.iconComponents['HOutline:BoltIcon']" /></span>
          {{ t('tools.tunnel.frps.sectionTransport') }}
          <span class="frps-sec__hint">{{ t('tools.tunnel.frps.portHint') }}</span>
        </h4>
        <div class="frps-grid">
          <div class="app-field">
            <label class="app-label">{{ t('tools.tunnel.frps.kcpBindPort') }}</label>
            <input v-model.number="form.kcpBindPort" type="number" min="0" max="65535" class="app-input" />
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('tools.tunnel.frps.quicBindPort') }}</label>
            <input v-model.number="form.quicBindPort" type="number" min="0" max="65535" class="app-input" />
          </div>
          <label class="frps-switch frps-full">
            <span class="frps-switch__text">
              <b>{{ t('tools.tunnel.frpsDialog.tlsForce') }}</b>
              <small>{{ t('tools.tunnel.frpsDialog.tlsForceDesc') }}</small>
            </span>
            <AppSwitch v-model="form.tlsForce" />
          </label>
        </div>
      </section>

      <section class="frps-sec">
        <h4 class="frps-sec__title">
          <span class="m-sticker m-sticker--ok frps-sec__ico"><component :is="menuStore.iconComponents['HOutline:ChartBarIcon']" /></span>
          {{ t('tools.tunnel.frps.sectionStats') }}
        </h4>
        <div class="frps-grid">
          <div class="app-field">
            <label class="app-label">{{ t('tools.tunnel.frps.sampleInterval') }}</label>
            <input v-model.number="form.statSampleInterval" type="number" min="5" max="3600" class="app-input" />
          </div>
          <span class="frps-hint frps-hint--side">{{ t('tools.tunnel.frpsDialog.sampleDesc') }}</span>
        </div>
      </section>
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
import type { FrpsConfig, FrpsStatus } from '@/types/v1/tunnel'
import { formatDuration, toDate } from '@/views/tools/components/format'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [config: FrpsConfig] }>()

const { t } = useI18n()
const menuStore = useMenuStore()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})

const loading = ref(false)
const saving = ref(false)
const status = ref<FrpsStatus | null>(null)

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
  tlsForce: false,
})

const form = ref<FrpsConfig>(defaultForm())

const statusText = computed(() => {
  const st = status.value
  if (st?.error) return t('tools.tunnel.hero.startFailed', { err: st.error })
  if (!st?.running) return t('tools.tunnel.frpsDialog.statusStopped')
  const start = toDate(st.startTime)
  return t('tools.tunnel.frpsDialog.statusRunning', { d: start ? formatDuration(Date.now() - start.getTime(), t) : '—' })
})

const load = async () => {
  loading.value = true
  try {
    const { data } = await getFrpsConfig()
    if (data?.config) form.value = { ...defaultForm(), ...data.config }
    status.value = data?.status || null
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
.frps-loading {
  padding: 24px;
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
}
.frps { display: flex; flex-direction: column; gap: 18px; }
.frps-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
}
.frps-status.is-on { background: var(--m-okSoft); color: var(--m-ok); }
.frps-status__dot { width: 8px; height: 8px; flex: none; border-radius: 50%; background: currentColor; }
.frps-status__text { flex: 1; min-width: 0; }
.frps-status__ver { font-size: 0.75rem; opacity: 0.85; }
.frps-sec { display: flex; flex-direction: column; gap: 12px; }
.frps-sec__title {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.frps-sec__ico {
  width: 30px;
  height: 30px;
  border-radius: 10px;
  transform: rotate(-4deg);
  :deep(svg) { width: 16px; height: 16px; }
}
.frps-sec__hint { margin-left: auto; font-size: 0.75rem; font-weight: 600; color: var(--el-text-color-secondary); }
.frps-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; align-items: start; }
.frps-full { grid-column: 1 / -1; }
.frps-switch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 56px;
  padding: 10px 14px;
  border-radius: var(--app-radius);
  border: 1px solid var(--el-border-color-lighter);
  cursor: pointer;
}
.frps-switch__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  b { font-size: 0.875rem; font-weight: 700; color: var(--el-text-color-primary); }
  small { font-size: 0.75rem; color: var(--el-text-color-secondary); }
}
.frps-hint { font-size: 0.75rem; color: var(--el-text-color-secondary); }
.frps-hint--side { align-self: center; }
@media (width <= 768px) {
  .frps-grid { grid-template-columns: 1fr; }
  .frps-full { grid-column: 1; }
  .frps-hint--side { align-self: start; margin-top: -6px; }
}
</style>
