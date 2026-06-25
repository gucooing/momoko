<template>
  <BaseDialog
    v-model="open"
    :title="editingId ? t('tools.portForward.editPortForward') : t('tools.portForward.addPortForward')"
    width="640"
    @close="close"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="formRules"
      class="port-forward-form"
      label-width="100px"
      label-position="right"
    >
      <el-form-item :label="t('common.name')" prop="name">
        <el-input v-model="form.name" :placeholder="t('tools.portForward.ruleNamePlaceholder')" />
      </el-form-item>

      <el-form-item :label="t('tools.portForward.forwardType')" prop="type">
        <el-radio-group v-model="form.type">
          <el-radio value="PORT_FORWARD_TYPE_TCP">TCP</el-radio>
          <el-radio value="PORT_FORWARD_TYPE_UDP">UDP</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-row :gutter="10" class="address-port-row">
        <el-col :xs="24" :sm="14">
          <el-form-item :label="t('tools.portForward.listenAddress')" prop="listenAddress" label-width="100px">
            <el-input v-model="form.listenAddress" placeholder="0.0.0.0" />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="10">
          <el-form-item :label="t('tools.portForward.port')" prop="listenPort" label-width="60px">
            <el-input-number
              v-model="form.listenPort"
              :min="1"
              :max="65535"
              controls-position="right"
              class="port-input"
            />
          </el-form-item>
        </el-col>
      </el-row>

      <el-row :gutter="10" class="address-port-row">
        <el-col :xs="24" :sm="14">
          <el-form-item :label="t('tools.portForward.targetAddress')" prop="targetAddress" label-width="100px">
            <el-input v-model="form.targetAddress" placeholder="127.0.0.1" />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="10">
          <el-form-item :label="t('tools.portForward.port')" prop="targetPort" label-width="60px">
            <el-input-number
              v-model="form.targetPort"
              :min="1"
              :max="65535"
              controls-position="right"
              class="port-input"
            />
          </el-form-item>
        </el-col>
      </el-row>

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
import { createPortForward, updatePortForward } from '@/api/network'
import { PortForwardType, type PortForwardInfo } from '@/types/v1/network'
import type { FormInstance, FormRules } from 'element-plus'

defineOptions({ name: 'PortForwardCreate' })

const emits = defineEmits(['refresh'])
const { t } = useI18n()
const formRef = useTemplateRef<FormInstance>('formRef')

const open = ref(false)
const loading = ref(false)
const editingId = ref('')

const defaultForm = () => ({
  name: '',
  type: 'PORT_FORWARD_TYPE_TCP' as string,
  listenAddress: '',
  listenPort: undefined as number | undefined,
  targetAddress: '',
  targetPort: undefined as number | undefined,
  isEnable: true,
})

const form = ref(defaultForm())

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
    let info: PortForwardInfo | undefined
    if (editingId.value) {
      const { data } = await updatePortForward({
        id: editingId.value,
        name: form.value.name || undefined,
        type: form.value.type as PortForwardType,
        listenAddress: form.value.listenAddress || undefined,
        listenPort: form.value.listenPort,
        targetAddress: form.value.targetAddress || undefined,
        targetPort: form.value.targetPort,
        isEnable: form.value.isEnable,
      })
      info = data?.info
    } else {
      const { data } = await createPortForward({
        name: form.value.name,
        type: form.value.type as PortForwardType,
        listenAddress: form.value.listenAddress,
        listenPort: form.value.listenPort!,
        targetAddress: form.value.targetAddress,
        targetPort: form.value.targetPort!,
        isEnable: form.value.isEnable,
      })
      info = data?.info
    }

    if (info?.error) {
      ElMessage.error(info.error)
    } else {
      ElMessage.success(editingId.value ? t('system.common.editSuccess') : t('system.common.addSuccess'))
    }
    emits('refresh', editingId.value ? 'update' : 'create')
    close()
  } finally {
    loading.value = false
  }
}

const formRules = computed<FormRules>(() => ({
  name: [{ required: true, message: t('tools.portForward.ruleNameRequired'), trigger: 'blur' }],
  type: [{ required: true, message: t('tools.portForward.forwardTypeRequired'), trigger: 'change' }],
  listenAddress: [{ required: true, message: t('tools.portForward.listenAddressRequired'), trigger: 'blur' }],
  listenPort: [{ required: true, message: t('tools.portForward.listenPortRequired'), trigger: 'blur' }],
  targetAddress: [{ required: true, message: t('tools.portForward.targetAddressRequired'), trigger: 'blur' }],
  targetPort: [{ required: true, message: t('tools.portForward.targetPortRequired'), trigger: 'blur' }],
}))

const showDialog = (payload?: PortForwardInfo) => {
  open.value = true
  if (!payload?.id) return

  editingId.value = payload.id
  form.value = {
    name: payload.name || '',
    type: payload.type || 'PORT_FORWARD_TYPE_TCP',
    listenAddress: payload.listenAddress || '',
    listenPort: payload.listenPort || undefined,
    targetAddress: payload.targetAddress || '',
    targetPort: payload.targetPort || undefined,
    isEnable: payload.isEnable ?? true,
  }
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.port-input {
  width: 100%;
}

@media (max-width: 767px) {
  .address-port-row + .address-port-row {
    margin-top: 0;
  }

  .port-forward-form :deep(.address-port-row .el-form-item) {
    margin-bottom: 18px;
  }

  .port-forward-form :deep(.address-port-row .el-col:last-child .el-form-item) {
    margin-top: -2px;
  }

  .port-forward-form :deep(.address-port-row .el-col:last-child .el-form-item__label) {
    width: 100px !important;
  }
}
</style>
