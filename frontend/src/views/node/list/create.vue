<!-- 节点新建/编辑弹窗：节点 gRPC 地址 + 节点上创建的 API Key（签名种子）+ TLS。
     底部「测试连接」用当前表单值实时探测（编辑时未填 API Key 沿用已保存的值）。
     ref 契约 showDialog(row?) + @refresh('create'|'update')。 -->
<template>
  <FormDialog
    v-model="open"
    :title="form.id ? t('node.list.edit') : t('node.list.add')"
    :width="560"
    :loading="submitLoading"
    @close="close"
    @confirm="confirm"
  >
    <div class="node-form">
      <details v-if="!form.id" class="node-form__guide">
        <summary>{{ t('node.page.guideHint') }}</summary>
        <ol>
          <li>{{ t('node.list.guideStep1') }}</li>
          <li>{{ t('node.list.guideStep2') }}</li>
          <li>{{ t('node.list.guideStep3') }}</li>
        </ol>
      </details>

      <div class="app-field">
        <label class="app-label app-label--required">{{ t('node.list.name') }}</label>
        <input
          v-model="form.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          :placeholder="t('node.list.namePlaceholder')"
          maxlength="50"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
      </div>

      <div class="app-field">
        <label class="app-label app-label--required">{{ t('node.list.address') }}</label>
        <input
          v-model="form.address"
          class="app-input node-mono"
          :class="{ 'is-error': errors.address }"
          :placeholder="t('node.list.addressPlaceholder')"
        />
        <span v-if="errors.address" class="app-field__error">{{ errors.address }}</span>
        <span v-else class="node-form__hint">{{ t('node.list.addressHint') }}</span>
      </div>

      <div class="app-field">
        <label class="app-label" :class="{ 'app-label--required': !form.id }">API Key</label>
        <input
          v-model="form.apiKey"
          type="password"
          autocomplete="new-password"
          class="app-input node-mono"
          :class="{ 'is-error': errors.apiKey }"
          :placeholder="form.id ? t('node.list.apiKeyKeep') : t('node.list.apiKeyPlaceholder')"
        />
        <span v-if="errors.apiKey" class="app-field__error">{{ errors.apiKey }}</span>
        <span v-else class="node-form__hint">{{ t('node.list.apiKeyHint') }}</span>
      </div>

      <div class="app-field node-form__switch">
        <div class="node-form__switch-text">
          <label class="app-label">{{ t('node.list.tls') }}</label>
          <span class="node-form__hint">{{ t('node.list.tlsHint') }}</span>
        </div>
        <AppSwitch v-model="form.tls" />
      </div>

      <div v-if="form.tls" class="app-field node-form__switch">
        <div class="node-form__switch-text">
          <label class="app-label">{{ t('node.list.tlsSkipVerify') }}</label>
          <span class="node-form__hint">{{ t('node.list.tlsSkipVerifyHint') }}</span>
        </div>
        <AppSwitch v-model="form.tlsSkipVerify" />
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('node.list.remark') }}</label>
        <textarea
          v-model="form.remark"
          class="app-input node-form__textarea"
          rows="2"
          maxlength="200"
          :placeholder="t('node.list.remarkPlaceholder')"
        />
      </div>

      <NodeTestResult v-if="testResult" :status="testResult" />
    </div>

    <template #footer="{ close: closeDialog }">
      <div class="node-form__footer">
        <UButton
          color="neutral"
          variant="soft"
          icon="i-lucide-plug-zap"
          :loading="testing"
          @click="runTest"
        >
          {{ t('node.list.test') }}
        </UButton>
        <span class="node-form__spacer" />
        <UButton color="neutral" variant="soft" @click="closeDialog">{{ t('node.list.cancel') }}</UButton>
        <UButton color="primary" :loading="submitLoading" @click="confirm">{{ t('node.list.confirm') }}</UButton>
      </div>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { createNode, testNode, updateNode } from '@/api/node'
import type { NodeInfo, NodeStatus } from '@/types/v1/node'
import NodeTestResult from '@/views/node/list/testResult.vue'

defineOptions({ name: 'NodeCreate' })

const emits = defineEmits<{ refresh: [type: 'create' | 'update'] }>()
const { t } = useI18n()

const open = ref(false)
const submitLoading = ref(false)
const testing = ref(false)
const testResult = ref<NodeStatus | null>(null)
const errors = ref<Record<string, string>>({})

const getDefaultForm = () => ({
  id: undefined as string | undefined,
  name: '',
  address: '',
  apiKey: '',
  tls: false,
  tlsSkipVerify: false,
  remark: '',
})
const form = ref(getDefaultForm())

const ADDRESS_PATTERN = /^(\[[0-9a-fA-F:.]+\]|[^\s:/]+):(\d{1,5})$/

const close = () => {
  open.value = false
  submitLoading.value = false
  testing.value = false
  testResult.value = null
  errors.value = {}
  form.value = getDefaultForm()
}

const validate = (forTest = false) => {
  const e: Record<string, string> = {}
  if (!forTest && !form.value.name.trim()) e.name = t('node.list.namePlaceholder')
  const match = ADDRESS_PATTERN.exec(form.value.address.trim())
  if (!match || Number(match[2]) < 1 || Number(match[2]) > 65535) {
    e.address = t('node.list.addressInvalid')
  }
  if (!form.value.id && !form.value.apiKey.trim()) e.apiKey = t('node.list.apiKeyPlaceholder')
  errors.value = e
  return Object.keys(e).length === 0
}

const runTest = async () => {
  if (!validate(true)) return
  testing.value = true
  try {
    const { data } = await testNode({
      id: form.value.id,
      address: form.value.address.trim(),
      apiKey: form.value.apiKey.trim() || undefined,
      tls: form.value.tls,
      tlsSkipVerify: form.value.tls && form.value.tlsSkipVerify,
    })
    testResult.value = data?.status || null
  } finally {
    testing.value = false
  }
}

const confirm = async () => {
  if (!validate()) return
  submitLoading.value = true
  try {
    const payload = {
      name: form.value.name.trim(),
      address: form.value.address.trim(),
      tls: form.value.tls,
      tlsSkipVerify: form.value.tls && form.value.tlsSkipVerify,
      remark: form.value.remark.trim(),
    }
    if (form.value.id) {
      await updateNode({
        id: form.value.id,
        ...payload,
        apiKey: form.value.apiKey.trim() || undefined,
      })
    } else {
      await createNode({ ...payload, apiKey: form.value.apiKey.trim() })
    }
    feedback.success(form.value.id ? t('node.list.editSuccess') : t('node.list.addSuccess'))
    emits('refresh', form.value.id ? 'update' : 'create')
    close()
  } finally {
    submitLoading.value = false
  }
}

const showDialog = (record?: NodeInfo) => {
  errors.value = {}
  testResult.value = null
  const next = getDefaultForm()
  if (record) {
    next.id = record.id
    next.name = record.name
    next.address = record.address
    next.tls = record.tls
    next.tlsSkipVerify = record.tlsSkipVerify
    next.remark = record.remark
  }
  form.value = next
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.node-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.node-form__guide {
  padding: 10px 14px;
  border-radius: var(--app-radius);
  background: var(--m-a2Soft);
  color: var(--m-ink);
  font-size: 0.8125rem;
  line-height: 1.6;
}
.node-form__guide summary {
  min-height: 24px;
  font-weight: 700;
  cursor: pointer;
}
.node-form__guide ol {
  margin: 6px 0 0;
  padding-left: 1.2rem;
  list-style: decimal;
}
.node-form__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.node-form__switch {
  flex-direction: row;
  padding: 10px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.node-form__switch-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.node-form__textarea {
  height: auto;
  min-height: 56px;
  padding-top: 8px;
  padding-bottom: 8px;
  resize: vertical;
}
.node-form__footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.node-form__spacer {
  flex: 1;
}
@media (width < 480px) {
  .node-form__spacer {
    flex-basis: 100%;
    height: 0;
  }
  .node-form__footer > :deep(button) {
    flex: 1 1 0;
    justify-content: center;
    min-height: 44px;
  }
}
.node-mono {
  font-family: var(--app-font-mono);
  font-size: 0.8125rem;
}
</style>
