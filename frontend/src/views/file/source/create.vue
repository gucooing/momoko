<!-- 文件来源 新建/编辑弹窗（重写 · P1）：FormDialog 外壳 + 令牌字段 + AppSelect(类型) + AppSwitch(开关) +
     按类型动态字段(OSS/FTP/WebDAV)。页脚含 测试连接。保留 ref 契约 showDialog(row?) + @refresh('create'|'update')。
     记忆约束：来源只是储存方式，前端不特判本地/远程细节，逻辑路径 + sourceID 由后端处理。 -->
<template>
  <FormDialog
    v-model="open"
    :title="form.id ? t('fileSource.editTitle') : t('fileSource.addTitle')"
    :width="560"
    :loading="saving"
    @close="close"
    @confirm="confirm"
  >
    <div class="src-form">
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('fileSource.name') }}</label>
        <input
          v-model="form.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          :placeholder="t('fileSource.namePlaceholder')"
          maxlength="50"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
      </div>

      <div class="src-form__grid">
        <div class="app-field">
          <label class="app-label">{{ t('fileSource.type') }}</label>
          <AppSelect v-model="form.type" :options="typeOptions" :disabled="!!form.id" @update:model-value="onTypeChange" />
        </div>
        <div class="app-field src-form__switch">
          <label class="app-label">{{ t('fileSource.enabled') }}</label>
          <AppSwitch v-model="form.enabled" />
        </div>
      </div>

      <div v-if="supportsRedirect" class="app-field src-form__switch src-form__switch--wide">
        <div class="src-form__switch-text">
          <label class="app-label">{{ t('fileSource.redirect302') }}</label>
          <span class="src-form__hint">{{ t('fileSource.redirectHint') }}</span>
        </div>
        <AppSwitch v-model="form.redirect302" />
      </div>

      <!-- OSS / S3 -->
      <template v-if="form.type === 'oss'">
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('fileSource.endpoint') }}</label>
          <input
            v-model="form.config.endpoint"
            class="app-input"
            :class="{ 'is-error': errors.endpoint }"
            placeholder="oss-cn-hangzhou.aliyuncs.com"
          />
          <span v-if="errors.endpoint" class="app-field__error">{{ errors.endpoint }}</span>
        </div>
        <div class="src-form__grid">
          <div class="app-field">
            <label class="app-label app-label--required">{{ t('fileSource.bucket') }}</label>
            <input
              v-model="form.config.bucket"
              class="app-input"
              :class="{ 'is-error': errors.bucket }"
            />
            <span v-if="errors.bucket" class="app-field__error">{{ errors.bucket }}</span>
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('fileSource.region') }}</label>
            <input v-model="form.config.region" class="app-input" placeholder="cn-hangzhou" />
          </div>
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('fileSource.accessKey') }}</label>
          <input v-model="form.config.accessKey" class="app-input" autocomplete="off" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('fileSource.secretKey') }}</label>
          <input
            v-model="form.config.secretKey"
            class="app-input"
            type="password"
            autocomplete="new-password"
            :placeholder="form.id ? t('fileSource.secretKeep') : ''"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('fileSource.prefix') }}</label>
          <input v-model="form.config.prefix" class="app-input" placeholder="momoko/" />
        </div>
        <div class="src-form__opts">
          <span class="app-label">{{ t('fileSource.options') }}</span>
          <label class="src-form__opt"><AppSwitch v-model="form.config.useSsl" /> HTTPS</label>
          <label class="src-form__opt"><AppSwitch v-model="form.config.pathStyle" /> Path-Style</label>
        </div>
      </template>

      <!-- FTP -->
      <template v-else-if="form.type === 'ftp'">
        <div class="src-form__grid">
          <div class="app-field">
            <label class="app-label app-label--required">{{ t('fileSource.host') }}</label>
            <input
              v-model="form.config.host"
              class="app-input"
              :class="{ 'is-error': errors.host }"
            />
            <span v-if="errors.host" class="app-field__error">{{ errors.host }}</span>
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('fileSource.port') }}</label>
            <input v-model.number="form.config.port" class="app-input" type="number" placeholder="21" />
          </div>
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('fileSource.username') }}</label>
          <input v-model="form.config.username" class="app-input" autocomplete="off" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('fileSource.password') }}</label>
          <input
            v-model="form.config.password"
            class="app-input"
            type="password"
            autocomplete="new-password"
            :placeholder="form.id ? t('fileSource.secretKeep') : ''"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('fileSource.basePath') }}</label>
          <input v-model="form.config.basePath" class="app-input" placeholder="/" />
        </div>
        <div class="src-form__opts">
          <span class="app-label">{{ t('fileSource.options') }}</span>
          <label class="src-form__opt"><AppSwitch v-model="form.config.tls" /> FTPS (TLS)</label>
        </div>
      </template>

      <!-- WebDAV -->
      <template v-else-if="form.type === 'webdav'">
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('fileSource.url') }}</label>
          <input
            v-model="form.config.url"
            class="app-input"
            :class="{ 'is-error': errors.url }"
            placeholder="https://dav.example.com/dav"
          />
          <span v-if="errors.url" class="app-field__error">{{ errors.url }}</span>
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('fileSource.username') }}</label>
          <input v-model="form.config.username" class="app-input" autocomplete="off" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('fileSource.password') }}</label>
          <input
            v-model="form.config.password"
            class="app-input"
            type="password"
            autocomplete="new-password"
            :placeholder="form.id ? t('fileSource.secretKeep') : ''"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('fileSource.basePath') }}</label>
          <input v-model="form.config.basePath" class="app-input" placeholder="/" />
        </div>
      </template>
    </div>

    <template #footer="{ close: doClose, confirm: doConfirm }">
      <UButton class="src-form__test" color="neutral" variant="soft" size="sm" :loading="testing" @click="testForm">
        {{ t('fileSource.test') }}
      </UButton>
      <UButton color="neutral" variant="soft" @click="doClose">{{ t('system.common.cancel') }}</UButton>
      <UButton color="primary" :loading="saving" @click="doConfirm">{{ t('system.common.confirm') }}</UButton>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  createFileSourceRequest,
  updateFileSourceRequest,
  testFileSourceRequest,
} from '@/api/fileSource'
import type { FileSourceInfo, FileSourceConfig } from '@/types/v1/file'

defineOptions({ name: 'FileSourceCreate' })

const emits = defineEmits<{ refresh: [type: 'create' | 'update'] }>()
const { t } = useI18n()

const open = ref(false)
const saving = ref(false)
const testing = ref(false)
const errors = ref<Record<string, string>>({})

const typeOptions = computed(() => [
  { label: t('fileSource.typeOss'), value: 'oss' },
  { label: t('fileSource.typeFtp'), value: 'ftp' },
  { label: t('fileSource.typeWebdav'), value: 'webdav' },
])

const emptyConfig = (): FileSourceConfig => ({
  endpoint: '',
  region: '',
  bucket: '',
  accessKey: '',
  secretKey: '',
  prefix: '',
  useSsl: true,
  pathStyle: false,
  host: '',
  port: 21,
  username: '',
  password: '',
  basePath: '',
  tls: false,
  url: '',
})

const getDefaultForm = () => ({
  id: undefined as string | undefined,
  name: '',
  type: 'oss',
  enabled: true,
  redirect302: false,
  config: emptyConfig(),
})
const form = ref(getDefaultForm())

// 仅对象存储支持预签名直链(302)。
const supportsRedirect = computed(() => form.value.type === 'oss')

const onTypeChange = () => {
  if (!supportsRedirect.value) form.value.redirect302 = false
}

const close = () => {
  open.value = false
  saving.value = false
  errors.value = {}
  form.value = getDefaultForm()
}

const validate = () => {
  const e: Record<string, string> = {}
  if (!form.value.name.trim()) e.name = t('fileSource.namePlaceholder')
  if (form.value.type === 'oss') {
    if (!form.value.config.endpoint.trim()) e.endpoint = t('fileSource.endpoint')
    if (!form.value.config.bucket.trim()) e.bucket = t('fileSource.bucket')
  } else if (form.value.type === 'ftp') {
    if (!form.value.config.host.trim()) e.host = t('fileSource.host')
  } else if (form.value.type === 'webdav') {
    if (!form.value.config.url.trim()) e.url = t('fileSource.url')
  }
  errors.value = e
  return Object.keys(e).length === 0
}

const buildPayloadConfig = (): FileSourceConfig => ({
  ...form.value.config,
  port: Number(form.value.config.port) || 0,
})

const confirm = async () => {
  if (!validate()) return
  saving.value = true
  try {
    if (form.value.id) {
      await updateFileSourceRequest({
        id: form.value.id,
        name: form.value.name,
        enabled: form.value.enabled,
        redirect302: form.value.redirect302,
        config: buildPayloadConfig(),
      })
      feedback.success(t('fileSource.updateSuccess'))
      emits('refresh', 'update')
    } else {
      await createFileSourceRequest({
        name: form.value.name,
        type: form.value.type,
        enabled: form.value.enabled,
        redirect302: form.value.redirect302,
        config: buildPayloadConfig(),
      })
      feedback.success(t('fileSource.createSuccess'))
      emits('refresh', 'create')
    }
    close()
  } finally {
    saving.value = false
  }
}

const testForm = async () => {
  testing.value = true
  try {
    const { data } = await testFileSourceRequest({
      id: '',
      type: form.value.type,
      config: buildPayloadConfig(),
    })
    if (data.ok) feedback.success(data.message || t('fileSource.testOk'))
    else feedback.error(data.message || t('fileSource.testFailed'))
  } finally {
    testing.value = false
  }
}

const showDialog = (record?: FileSourceInfo) => {
  errors.value = {}
  const next = getDefaultForm()
  if (record) {
    next.id = record.id
    next.name = record.name
    next.type = record.type
    next.enabled = record.enabled
    next.redirect302 = record.redirect302
    // 密钥不回显（留空=保留原值）
    Object.assign(next.config, emptyConfig(), record.config ?? {}, { secretKey: '', password: '' })
  }
  form.value = next
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.src-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.src-form__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}
.src-form__switch {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.src-form__switch--wide {
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-sm);
}
.src-form__switch-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.src-form__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  line-height: 1.35;
}
.src-form__opts {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 18px;
}
.src-form__opt {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.src-form__test {
  margin-right: auto;
}
@media (width <= 768px) {
  .src-form__grid {
    grid-template-columns: 1fr;
  }
}
</style>
