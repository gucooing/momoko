<!-- 文件来源 新建/编辑弹窗（第二期改版）：FormDialog 外壳 + 存储类型贴纸单选卡 + 按类型动态字段(OSS/FTP/WebDAV) + 启用开关 + 测试结果内联提示。
     页脚含 测试连接。保留 ref 契约 showDialog(row?) + @refresh('create'|'update')。
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

      <div class="app-field">
        <label class="app-label">{{ t('fileSource.storageType') }}</label>
        <div class="src-types" role="radiogroup" :aria-label="t('fileSource.storageType')">
          <button
            v-for="(opt, i) in typeOptions"
            :key="opt.value"
            type="button"
            role="radio"
            class="src-type"
            :class="{ 'is-on': form.type === opt.value }"
            :aria-checked="form.type === opt.value"
            :disabled="!!form.id && form.type !== opt.value"
            @click="pickType(opt.value)"
          >
            <span class="src-type__sticker m-sticker" :class="`m-sticker--${opt.tone}`" :style="{ transform: `rotate(${i % 2 ? 3 : -4}deg)` }">
              <UIcon :name="opt.icon" />
            </span>
            <span class="src-type__text">
              <span class="src-type__name">{{ opt.label }}</span>
              <span class="src-type__desc">{{ opt.desc }}</span>
            </span>
          </button>
        </div>
        <span v-if="form.id" class="src-form__hint">{{ t('fileSource.typeLocked') }}</span>
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

      <div class="app-field src-form__switch src-form__switch--wide">
        <div class="src-form__switch-text">
          <label class="app-label">{{ t('fileSource.enabled') }}</label>
          <span class="src-form__hint">{{ t('fileSource.enableHint') }}</span>
        </div>
        <AppSwitch v-model="form.enabled" />
      </div>

      <div v-if="testResult" class="src-result" :class="testResult.ok ? 'is-ok' : 'is-bad'" role="status">
        <UIcon :name="testResult.ok ? 'i-lucide-circle-check' : 'i-lucide-circle-alert'" class="src-result__ico" />
        <span class="src-result__text">
          <strong>{{ testResult.ok ? t('fileSource.testOk') : t('fileSource.testFailed') }}</strong>
          <span v-if="testResult.latencyMs" class="m-num"> · {{ t('fileSource.latency', { ms: testResult.latencyMs }) }}</span>
          <span v-if="!testResult.ok && testResult.message" class="src-result__msg m-mono">{{ testResult.message }}</span>
        </span>
      </div>
    </div>

    <template #footer="{ close: doClose, confirm: doConfirm }">
      <UButton class="src-form__test" color="neutral" variant="soft" icon="i-lucide-plug-zap" :loading="testing" @click="testForm">
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
  { label: t('fileSource.typeShort.oss'), value: 'oss', desc: t('fileSource.typeDescOss'), icon: 'i-lucide-cloud', tone: 'p' },
  { label: t('fileSource.typeShort.ftp'), value: 'ftp', desc: t('fileSource.typeDescFtp'), icon: 'i-lucide-arrow-left-right', tone: 'ok' },
  { label: t('fileSource.typeShort.webdav'), value: 'webdav', desc: t('fileSource.typeDescWebdav'), icon: 'i-lucide-globe', tone: 'a2' },
])
const pickType = (type: string) => {
  if (form.value.id || form.value.type === type) return
  form.value.type = type
  testResult.value = null
  onTypeChange()
}
const testResult = ref<{ ok: boolean; message: string; latencyMs: number } | null>(null)

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
  testResult.value = null
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
    testResult.value = { ok: data.ok, message: data.message, latencyMs: Number(data.latencyMs) || 0 }
  } finally {
    testing.value = false
  }
}

const showDialog = (record?: FileSourceInfo) => {
  errors.value = {}
  testResult.value = null
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
.src-types {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}
.src-type {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 64px;
  padding: 10px 12px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.src-type:hover:not(:disabled) {
  border-color: var(--m-p);
}
.src-type.is-on {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.src-type:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.src-type__sticker {
  width: 38px;
  height: 38px;
  border-radius: 13px;
}
.src-type__sticker :deep(svg),
.src-type__sticker :deep(.iconify) {
  width: 18px;
  height: 18px;
}
.src-type__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.src-type__name {
  font-size: 0.875rem;
  font-weight: 800;
}
.src-type__desc {
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
  line-height: 1.3;
}
.src-result {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  font-size: 0.8125rem;
}
.src-result.is-ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.src-result.is-bad {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.src-result__ico {
  flex: none;
  width: 18px;
  height: 18px;
  margin-top: 1px;
}
.src-result__text {
  min-width: 0;
}
.src-result__msg {
  display: block;
  margin-top: 2px;
  font-size: 0.75rem;
  word-break: break-all;
}
.src-form__test {
  margin-right: auto;
}
@media (width <= 768px) {
  .src-form__grid {
    grid-template-columns: 1fr;
  }
  .src-types {
    grid-template-columns: 1fr;
  }
}
</style>
