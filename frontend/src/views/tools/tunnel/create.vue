<template>
  <BaseDialog
    v-model="open"
    :title="editingId ? t('tools.tunnel.editTunnel') : t('tools.tunnel.addTunnel')"
    width="640"
    @close="close"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="formRules"
      class="tunnel-form"
      label-width="100px"
      label-position="right"
    >
      <el-form-item :label="t('tools.tunnel.name')" prop="name">
        <el-input v-model="form.name" :placeholder="t('tools.tunnel.namePlaceholder')" />
      </el-form-item>

      <el-form-item :label="t('tools.tunnel.proxyType')" prop="type">
        <el-select v-model="form.type">
          <el-option v-for="opt in typeOptions" :key="opt" :label="typeLabel(opt)" :value="opt" />
        </el-select>
      </el-form-item>

      <el-form-item v-if="isPortType" :label="t('tools.tunnel.remotePort')" prop="remotePort">
        <el-input-number
          v-model="form.remotePort"
          :min="1"
          :max="65535"
          controls-position="right"
          class="full-input"
        />
      </el-form-item>

      <template v-if="isHttpType">
        <el-form-item :label="t('tools.tunnel.customDomains')" prop="customDomains">
          <el-input v-model="form.customDomains" :placeholder="t('tools.tunnel.customDomainsPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('tools.tunnel.subdomain')" prop="subdomain">
          <el-input v-model="form.subdomain" placeholder="my-app" />
        </el-form-item>
      </template>

      <el-row :gutter="10">
        <el-col :xs="24" :sm="14">
          <el-form-item :label="t('tools.tunnel.localIp')" prop="localIp" label-width="100px">
            <el-input v-model="form.localIp" placeholder="127.0.0.1" />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="10">
          <el-form-item :label="t('tools.tunnel.localPort')" prop="localPort" label-width="80px">
            <el-input-number
              v-model="form.localPort"
              :min="1"
              :max="65535"
              controls-position="right"
              class="full-input"
            />
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item :label="t('tools.tunnel.allowUsers')" prop="allowUsers">
        <el-input v-model="form.allowUsers" :placeholder="t('tools.tunnel.allowUsersPlaceholder')" />
      </el-form-item>

      <el-form-item :label="t('system.common.enabled')" prop="isEnable">
        <el-switch v-model="form.isEnable" />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="close">{{ t('system.common.cancel') }}</el-button>
      <el-button type="primary" :loading="loading" @click="confirm">{{ t('system.common.confirm') }}</el-button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { createTunnel, updateTunnel } from '@/api/tunnel'
import { TunnelType, type TunnelInfo } from '@/types/v1/tunnel'
import type { FormInstance, FormRules } from 'element-plus'

defineOptions({ name: 'TunnelCreate' })

const emits = defineEmits(['refresh'])
const { t } = useI18n()
const formRef = useTemplateRef<FormInstance>('formRef')

const open = ref(false)
const loading = ref(false)
const editingId = ref('')

const typeOptions = [
  TunnelType.TUNNEL_TYPE_TCP,
  TunnelType.TUNNEL_TYPE_UDP,
  TunnelType.TUNNEL_TYPE_HTTP,
  TunnelType.TUNNEL_TYPE_HTTPS,
  TunnelType.TUNNEL_TYPE_STCP,
  TunnelType.TUNNEL_TYPE_XTCP,
  TunnelType.TUNNEL_TYPE_TCPMUX,
]

const typeLabel = (type: TunnelType) => type.replace('TUNNEL_TYPE_', '')

const defaultForm = () => ({
  name: '',
  type: TunnelType.TUNNEL_TYPE_TCP as TunnelType,
  remotePort: undefined as number | undefined,
  customDomains: '',
  subdomain: '',
  localIp: '127.0.0.1',
  localPort: undefined as number | undefined,
  allowUsers: '',
  isEnable: true,
})

const form = ref(defaultForm())

const isPortType = computed(() =>
  [TunnelType.TUNNEL_TYPE_TCP, TunnelType.TUNNEL_TYPE_UDP, TunnelType.TUNNEL_TYPE_TCPMUX].includes(form.value.type),
)
const isHttpType = computed(() =>
  [TunnelType.TUNNEL_TYPE_HTTP, TunnelType.TUNNEL_TYPE_HTTPS].includes(form.value.type),
)

const formRules = computed<FormRules>(() => ({
  name: [{ required: true, message: t('tools.tunnel.nameRequired'), trigger: 'blur' }],
  type: [{ required: true, message: t('tools.tunnel.typeRequired'), trigger: 'change' }],
  remotePort: isPortType.value
    ? [{ required: true, message: t('tools.tunnel.remotePortRequired'), trigger: 'blur' }]
    : [],
  localPort: [{ required: true, message: t('tools.tunnel.localPortRequired'), trigger: 'blur' }],
  customDomains: isHttpType.value
    ? [{
        validator: (_r, _v, cb) => {
          if (!form.value.customDomains.trim() && !form.value.subdomain.trim()) {
            cb(new Error(t('tools.tunnel.domainsRequired')))
          } else {
            cb()
          }
        },
        trigger: 'blur',
      }]
    : [],
}))

const close = () => {
  open.value = false
  formRef.value?.resetFields()
  editingId.value = ''
  form.value = defaultForm()
}

const confirm = async () => {
  await formRef.value?.validate()
  loading.value = true
  try {
    if (editingId.value) {
      await updateTunnel({
        id: editingId.value,
        name: form.value.name || undefined,
        type: form.value.type,
        remotePort: form.value.remotePort,
        customDomains: form.value.customDomains,
        subdomain: form.value.subdomain,
        localIp: form.value.localIp || undefined,
        localPort: form.value.localPort,
        allowUsers: form.value.allowUsers,
        isEnable: form.value.isEnable,
      })
      ElMessage.success(t('system.common.editSuccess'))
    } else {
      await createTunnel({
        name: form.value.name,
        type: form.value.type,
        remotePort: form.value.remotePort ?? 0,
        customDomains: form.value.customDomains,
        subdomain: form.value.subdomain,
        localIp: form.value.localIp,
        localPort: form.value.localPort ?? 0,
        allowUsers: form.value.allowUsers,
        isEnable: form.value.isEnable,
      })
      ElMessage.success(t('system.common.addSuccess'))
    }
    emits('refresh', editingId.value ? 'update' : 'create')
    close()
  } finally {
    loading.value = false
  }
}

const showDialog = (payload?: TunnelInfo) => {
  open.value = true
  if (!payload?.id) return

  editingId.value = payload.id
  form.value = {
    name: payload.name || '',
    type: payload.type || TunnelType.TUNNEL_TYPE_TCP,
    remotePort: payload.remotePort || undefined,
    customDomains: payload.customDomains || '',
    subdomain: payload.subdomain || '',
    localIp: payload.localIp || '127.0.0.1',
    localPort: payload.localPort || undefined,
    allowUsers: payload.allowUsers || '',
    isEnable: payload.isEnable ?? true,
  }
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.full-input {
  width: 100%;
}
</style>
