<template>
  <BaseDialog v-model="visible" :title="t('tools.tunnel.frps.title')" width="640" @opened="load">
    <p class="frps-intro">{{ t('tools.tunnel.frps.intro') }}</p>
    <el-form v-loading="loading" :model="form" label-width="120px" label-position="right">
      <el-form-item :label="t('tools.tunnel.frps.isEnable')">
        <el-switch v-model="form.isEnable" />
      </el-form-item>
      <el-row :gutter="10">
        <el-col :xs="24" :sm="14">
          <el-form-item :label="t('tools.tunnel.frps.bindAddr')">
            <el-input v-model="form.bindAddr" placeholder="0.0.0.0" />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="10">
          <el-form-item :label="t('tools.tunnel.frps.bindPort')" label-width="90px">
            <el-input-number v-model="form.bindPort" :min="0" :max="65535" controls-position="right" class="full-input" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item :label="t('tools.tunnel.frps.serverAddr')">
        <el-input v-model="form.serverAddr" :placeholder="t('tools.tunnel.frps.serverAddrPlaceholder')" />
      </el-form-item>
      <el-row :gutter="10">
        <el-col :xs="24" :sm="12">
          <el-form-item :label="t('tools.tunnel.frps.vhostHttpPort')" label-width="120px">
            <el-input-number v-model="form.vhostHttpPort" :min="0" :max="65535" controls-position="right" class="full-input" />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="12">
          <el-form-item :label="t('tools.tunnel.frps.vhostHttpsPort')" label-width="120px">
            <el-input-number v-model="form.vhostHttpsPort" :min="0" :max="65535" controls-position="right" class="full-input" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-row :gutter="10">
        <el-col :xs="24" :sm="12">
          <el-form-item :label="t('tools.tunnel.frps.kcpBindPort')" label-width="120px">
            <el-input-number v-model="form.kcpBindPort" :min="0" :max="65535" controls-position="right" class="full-input" />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="12">
          <el-form-item :label="t('tools.tunnel.frps.quicBindPort')" label-width="120px">
            <el-input-number v-model="form.quicBindPort" :min="0" :max="65535" controls-position="right" class="full-input" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item :label="t('tools.tunnel.frps.subdomainHost')">
        <el-input v-model="form.subdomainHost" placeholder="frps.example.com" />
      </el-form-item>
      <el-form-item :label="t('tools.tunnel.frps.sampleInterval')">
        <el-input-number v-model="form.statSampleInterval" :min="5" :max="3600" controls-position="right" />
        <span class="frps-hint">{{ t('tools.tunnel.frps.portHint') }}</span>
      </el-form-item>

      <el-divider content-position="left">{{ t('tools.tunnel.frps.optimizeTitle') }}</el-divider>
      <p class="frps-intro">{{ t('tools.tunnel.frps.optimizeHint') }}</p>
      <el-row :gutter="10">
        <el-col :xs="24" :sm="12">
          <el-form-item :label="t('tools.tunnel.frps.tlsForce')" label-width="120px">
            <el-switch v-model="form.tlsForce" />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="12">
          <el-form-item :label="t('tools.tunnel.frps.tcpMux')" label-width="120px">
            <el-switch v-model="form.tcpMux" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-row :gutter="10">
        <el-col :xs="24" :sm="12">
          <el-form-item :label="t('tools.tunnel.frps.maxPoolCount')" label-width="120px">
            <el-input-number v-model="form.maxPoolCount" :min="0" :max="1000" controls-position="right" class="full-input" />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="12">
          <el-form-item :label="t('tools.tunnel.frps.maxPortsPerClient')" label-width="120px">
            <el-input-number v-model="form.maxPortsPerClient" :min="0" :max="65535" controls-position="right" class="full-input" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-row :gutter="10">
        <el-col :xs="24" :sm="12">
          <el-form-item :label="t('tools.tunnel.frps.heartbeatTimeout')" label-width="120px">
            <el-input-number v-model="form.heartbeatTimeout" :min="-1" :max="86400" controls-position="right" class="full-input" />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="12">
          <el-form-item :label="t('tools.tunnel.frps.udpPacketSize')" label-width="120px">
            <el-input-number v-model="form.udpPacketSize" :min="0" :max="65507" controls-position="right" class="full-input" />
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>

    <template #footer>
      <el-button @click="visible = false">{{ t('system.common.cancel') }}</el-button>
      <el-button type="primary" :loading="saving" @click="save">{{ t('tools.tunnel.frps.save') }}</el-button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
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
  tlsForce: false,
  maxPoolCount: 5,
  maxPortsPerClient: 0,
  heartbeatTimeout: -1,
  tcpMux: true,
  udpPacketSize: 1500,
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
    ElMessage.success(t('tools.tunnel.frps.saveSuccess'))
    if (data?.config) emit('saved', data.config)
    visible.value = false
  } finally {
    saving.value = false
  }
}
</script>

<style scoped lang="scss">
.frps-intro {
  margin: 0 0 0.8rem;
  font-size: 0.85rem;
  color: var(--el-text-color-secondary);
}

.full-input {
  width: 100%;
}

.frps-hint {
  margin-left: 0.6rem;
  font-size: 0.78rem;
  color: var(--el-text-color-secondary);
}
</style>
