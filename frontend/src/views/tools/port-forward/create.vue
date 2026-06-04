<template>
  <BaseDialog
    v-model="open"
    :title="editingId ? '编辑端口转发' : '新增端口转发'"
    width="640"
    @close="close"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="formRules"
      label-width="100px"
      label-position="right"
    >
      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" placeholder="请输入规则名称" />
      </el-form-item>

      <el-form-item label="转发类型" prop="type">
        <el-radio-group v-model="form.type">
          <el-radio value="PORT_FORWARD_TYPE_TCP">TCP</el-radio>
          <el-radio value="PORT_FORWARD_TYPE_UDP">UDP</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-row :gutter="10">
        <el-col :span="14">
          <el-form-item label="监听地址" prop="listenAddress" label-width="100px">
            <el-input v-model="form.listenAddress" placeholder="0.0.0.0" />
          </el-form-item>
        </el-col>
        <el-col :span="10">
          <el-form-item label="端口" prop="listenPort" label-width="60px">
            <el-input-number v-model="form.listenPort" :min="1" :max="65535" style="width: 100%" />
          </el-form-item>
        </el-col>
      </el-row>

      <el-row :gutter="10">
        <el-col :span="14">
          <el-form-item label="目标地址" prop="targetAddress" label-width="100px">
            <el-input v-model="form.targetAddress" placeholder="127.0.0.1" />
          </el-form-item>
        </el-col>
        <el-col :span="10">
          <el-form-item label="端口" prop="targetPort" label-width="60px">
            <el-input-number v-model="form.targetPort" :min="1" :max="65535" style="width: 100%" />
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item label="启用" prop="isEnable">
        <el-switch v-model="form.isEnable" />
      </el-form-item>

      <el-form-item label="标签" prop="tags">
        <el-input v-model="form.tags" placeholder="多个标签用逗号分隔" />
      </el-form-item>

      <el-form-item label="备注" prop="remark">
        <el-input
          v-model="form.remark"
          type="textarea"
          :rows="2"
          placeholder="请输入备注信息"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" :loading="loading" @click="confirm">确定</el-button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { createPortForward, updatePortForward } from '@/api/network'
import { PortForwardType, type PortForwardInfo } from '@/types/v1/network'
import type { FormInstance, FormRules } from 'element-plus'

defineOptions({ name: 'PortForwardCreate' })

const emits = defineEmits(['refresh'])
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
  tags: '',
  remark: '',
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
        tags: form.value.tags || undefined,
        remark: form.value.remark || undefined,
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
        tags: form.value.tags,
        remark: form.value.remark,
      })
      info = data?.info
    }

    if (info?.error) {
      ElMessage.error(info.error)
    } else {
      ElMessage.success(editingId.value ? '编辑成功' : '新增成功')
    }
    emits('refresh', editingId.value ? 'update' : 'create')
    close()
  } finally {
    loading.value = false
  }
}

const formRules: FormRules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择转发类型', trigger: 'change' }],
  listenAddress: [{ required: true, message: '请输入监听地址', trigger: 'blur' }],
  listenPort: [{ required: true, message: '请输入监听端口', trigger: 'blur' }],
  targetAddress: [{ required: true, message: '请输入目标地址', trigger: 'blur' }],
  targetPort: [{ required: true, message: '请输入目标端口', trigger: 'blur' }],
}

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
    tags: payload.tags || '',
    remark: payload.remark || '',
  }
}

defineExpose({ showDialog })
</script>
