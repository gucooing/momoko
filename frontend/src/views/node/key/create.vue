<!-- API Key 新建/编辑弹窗：名称 + 用途备注 + 有效期（快速设置 30 天 / 90 天 / 1 年，原生 datetime-local，编辑时可设为永久有效）
     + 节点权限（远程主控用该 Key 调用本节点时的授权范围）。
     ref 契约 showDialog(row?) + @refresh('create'|'update', info?)；新建时 info 带完整密钥，列表页据此展示「密钥已创建」横幅。 -->
<template>
  <FormDialog
    v-model="open"
    :title="submitForm.id ? t('node.key.edit') : t('node.key.add')"
    :width="680"
    :loading="submitLoading"
    @close="close"
    @confirm="confirm"
  >
    <div class="key-form">
      <div class="key-form__grid">
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('node.key.name') }}</label>
          <input
            v-model="submitForm.name"
            class="app-input"
            :class="{ 'is-error': errors.name }"
            :placeholder="t('node.key.namePlaceholder')"
            maxlength="50"
            @keyup.enter="confirm"
          />
          <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
        </div>

        <div class="app-field">
          <label class="app-label">{{ t('node.keyx.remark') }}</label>
          <input
            v-model="submitForm.remark"
            class="app-input"
            :placeholder="t('node.keyx.remarkPlaceholder')"
            maxlength="200"
          />
        </div>
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('node.key.expiresAt') }}</label>
        <div class="key-form__presets" role="group" :aria-label="t('node.keyx.presetLabel')">
          <button
            v-for="p in presets"
            :key="p.days"
            type="button"
            class="key-form__preset"
            @click="applyPreset(p.days)"
          >
            {{ p.label }}
          </button>
        </div>
        <input
          v-model="submitForm.expiresAt"
          type="datetime-local"
          class="app-input"
          :disabled="submitForm.neverExpires"
        />
        <span class="key-form__hint">{{ t('node.key.expiresPlaceholder') }}</span>
      </div>

      <div v-if="submitForm.id" class="app-field key-form__switch">
        <div class="key-form__switch-text">
          <label class="app-label">{{ t('node.key.neverExpires') }}</label>
          <span class="key-form__hint">{{ t('node.key.neverExpiresTip') }}</span>
        </div>
        <AppSwitch v-model="submitForm.neverExpires" />
      </div>

      <div class="app-field">
        <label class="app-label">{{ t('node.key.permissions') }}</label>
        <span class="key-form__hint">{{ t('node.key.permissionsHint') }}</span>
        <NodePermissionPicker v-model="submitForm.permissions" />
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { createAPIKey, updateAPIKey } from '@/api/node'
import type { APIKeyInfo } from '@/types/v1/node'
import NodePermissionPicker from '@/views/node/components/permissionPicker.vue'

defineOptions({ name: 'ApiKeyCreate' })

const emits = defineEmits<{ refresh: [type: 'create' | 'update', info?: APIKeyInfo] }>()
const { t } = useI18n()

const open = ref(false)
const submitLoading = ref(false)
const errors = ref<Record<string, string>>({})

const getDefaultForm = () => ({
  id: undefined as string | undefined,
  name: '',
  remark: '',
  expiresAt: '' as string,
  neverExpires: false,
  permissions: [] as string[],
})
const submitForm = ref(getDefaultForm())

const presets = computed(() => [
  { days: 30, label: t('node.keyx.preset30') },
  { days: 90, label: t('node.keyx.preset90') },
  { days: 365, label: t('node.keyx.preset365') },
])

// Date → datetime-local 字符串（YYYY-MM-DDTHH:mm，本地时区）
const toLocalInput = (value: Date | string): string => {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const applyPreset = (days: number) => {
  submitForm.value.neverExpires = false
  submitForm.value.expiresAt = toLocalInput(new Date(Date.now() + days * 24 * 3600 * 1000))
}

const close = () => {
  open.value = false
  submitLoading.value = false
  errors.value = {}
  submitForm.value = getDefaultForm()
}

const validate = () => {
  const e: Record<string, string> = {}
  if (!submitForm.value.name.trim()) e.name = t('node.key.namePlaceholder')
  errors.value = e
  return Object.keys(e).length === 0
}

const confirm = async () => {
  if (!validate()) return

  const expiresDate = submitForm.value.expiresAt ? new Date(submitForm.value.expiresAt) : undefined

  submitLoading.value = true
  try {
    if (submitForm.value.id) {
      const { data } = await updateAPIKey({
        id: submitForm.value.id,
        name: submitForm.value.name || undefined,
        remark: submitForm.value.remark.trim(),
        expiresAt: submitForm.value.neverExpires ? undefined : expiresDate,
        neverExpires: submitForm.value.neverExpires || undefined,
        permissions: { items: submitForm.value.permissions },
      })
      feedback.success(t('node.key.editSuccess'))
      emits('refresh', 'update', data?.info)
    } else {
      const { data } = await createAPIKey({
        name: submitForm.value.name,
        remark: submitForm.value.remark.trim(),
        expiresAt: expiresDate,
        permissions: submitForm.value.permissions,
      })
      feedback.success(t('node.key.addSuccess'))
      emits('refresh', 'create', data?.info)
    }
    close()
  } finally {
    submitLoading.value = false
  }
}

const showDialog = (record?: APIKeyInfo) => {
  errors.value = {}
  const form = getDefaultForm()
  if (record) {
    form.id = record.id
    form.name = record.name
    form.remark = record.remark || ''
    form.permissions = [...(record.permissions || [])]
    if (record.expiresAt) {
      form.expiresAt = toLocalInput(record.expiresAt)
      form.neverExpires = false
    } else {
      form.neverExpires = true
    }
  }
  submitForm.value = form
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.key-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.key-form__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}
@media (width < 640px) {
  .key-form__grid {
    grid-template-columns: 1fr;
  }
}
.key-form__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.key-form__presets {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.key-form__preset {
  min-height: 36px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s;
}
.key-form__preset:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.key-form__switch {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.key-form__switch-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
</style>
