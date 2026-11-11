<!-- 文件源 新增/编辑表单（原型：PC 为列表右侧的表单卡，手机为弹窗）：存储类型 2×2 贴纸单选卡（本地磁盘 / OSS-S3 / FTP / WebDAV）+
     按类型动态字段 + 302 直链 / 保存后立即启用 开关 + 测试结果内联提示 + 测试连接 / 保存。
     ref 契约：showDialog(row?) 打开（新增/编辑）；@refresh('create'|'update') 保存成功后通知列表刷新。inline=true 渲染为表单卡。 -->
<template>
  <component
    :is="inline ? InlineShell : FormDialog"
    v-if="!inline || open"
    v-model="open"
    :title="title"
    :width="560"
    :show-footer="false"
    @close="close"
  >
    <form class="sf" @submit.prevent="confirm">
      <div class="sf-group">
        <span class="sf-lbl">{{ t('fileSource.storageType') }}</span>
        <div class="sf-types" role="radiogroup" :aria-label="t('fileSource.storageType')">
          <button
            v-for="(opt, i) in typeOptions"
            :key="opt.value"
            type="button"
            role="radio"
            class="sf-type"
            :class="{ 'is-on': form.type === opt.value }"
            :aria-checked="form.type === opt.value"
            :disabled="!!form.id && form.type !== opt.value"
            @click="pickType(opt.value)"
          >
            <span class="sf-type__sticker m-sticker" :class="`m-sticker--${opt.tone}`" :style="{ transform: `rotate(${ROTS[i]})` }">
              <UIcon :name="opt.icon" />
            </span>
            <span class="sf-type__text">
              <span class="sf-type__name">{{ opt.label }}</span>
              <span class="sf-type__desc">{{ opt.desc }}</span>
            </span>
          </button>
        </div>
        <span v-if="form.id" class="sf-hint">{{ t('fileSource.typeLocked') }}</span>
      </div>

      <label class="sf-field">
        <span class="sf-field__label is-required">{{ t('fileSource.name') }}</span>
        <span class="sf-fld" :class="{ 'is-error': errors.name }">
          <input v-model="form.name" :placeholder="namePlaceholder" maxlength="50" />
        </span>
        <span v-if="errors.name" class="sf-err">{{ errors.name }}</span>
      </label>

      <!-- 本地磁盘（本机目录挂载） -->
      <template v-if="form.type === 'local'">
        <label class="sf-field">
          <span class="sf-field__label is-required">{{ t('fileSource.rootDir') }}</span>
          <span class="sf-fld" :class="{ 'is-error': errors.basePath }">
            <UIcon name="i-lucide-folder" class="sf-fld__ico" />
            <input v-model="form.config.basePath" class="m-mono" placeholder="/srv/data" autocomplete="off" />
          </span>
          <span v-if="errors.basePath" class="sf-err">{{ errors.basePath }}</span>
          <span v-else class="sf-hint">{{ t('fileSource.rootDirHint') }}</span>
        </label>
      </template>

      <!-- OSS / S3 -->
      <template v-else-if="form.type === 'oss'">
        <label class="sf-field">
          <span class="sf-field__label is-required">{{ t('fileSource.endpoint') }}</span>
          <span class="sf-fld" :class="{ 'is-error': errors.endpoint }">
            <input v-model="form.config.endpoint" class="m-mono" placeholder="https://oss-cn-hangzhou.aliyuncs.com" autocomplete="off" />
          </span>
          <span v-if="errors.endpoint" class="sf-err">{{ errors.endpoint }}</span>
        </label>
        <div class="sf-grid">
          <label class="sf-field">
            <span class="sf-field__label is-required">{{ t('fileSource.bucket') }}</span>
            <span class="sf-fld" :class="{ 'is-error': errors.bucket }">
              <input v-model="form.config.bucket" class="m-mono" placeholder="momoko-assets" autocomplete="off" />
            </span>
            <span v-if="errors.bucket" class="sf-err">{{ errors.bucket }}</span>
          </label>
          <label class="sf-field">
            <span class="sf-field__label">{{ t('fileSource.region') }}</span>
            <span class="sf-fld"><input v-model="form.config.region" class="m-mono" placeholder="cn-hangzhou" autocomplete="off" /></span>
          </label>
        </div>
        <label class="sf-field">
          <span class="sf-field__label">{{ t('fileSource.accessKey') }}</span>
          <span class="sf-fld">
            <UIcon name="i-lucide-key-round" class="sf-fld__ico" />
            <input v-model="form.config.accessKey" class="m-mono" autocomplete="off" />
          </span>
        </label>
        <label class="sf-field">
          <span class="sf-field__label">{{ t('fileSource.secretKey') }}</span>
          <span class="sf-fld">
            <UIcon name="i-lucide-lock" class="sf-fld__ico" />
            <input
              v-model="form.config.secretKey"
              class="m-mono"
              type="password"
              autocomplete="new-password"
              :placeholder="form.id ? t('fileSource.secretKeep') : t('fileSource.secretNeverShown')"
            />
          </span>
        </label>
        <label class="sf-field">
          <span class="sf-field__label">{{ t('fileSource.prefix') }}</span>
          <span class="sf-fld"><input v-model="form.config.prefix" class="m-mono" placeholder="momoko/" autocomplete="off" /></span>
        </label>
        <div class="sf-opts">
          <label class="sf-opt"><AppSwitch v-model="form.config.useSsl" /> HTTPS</label>
          <label class="sf-opt"><AppSwitch v-model="form.config.pathStyle" /> Path-Style</label>
        </div>
      </template>

      <!-- FTP / WebDAV -->
      <template v-else>
        <div v-if="form.type === 'ftp'" class="sf-grid sf-grid--host">
          <label class="sf-field">
            <span class="sf-field__label is-required">{{ t('fileSource.host') }}</span>
            <span class="sf-fld" :class="{ 'is-error': errors.host }">
              <input v-model="form.config.host" class="m-mono" placeholder="192.168.1.20" autocomplete="off" />
            </span>
            <span v-if="errors.host" class="sf-err">{{ errors.host }}</span>
          </label>
          <label class="sf-field">
            <span class="sf-field__label">{{ t('fileSource.port') }}</span>
            <span class="sf-fld"><input v-model.number="form.config.port" class="m-mono" type="number" placeholder="21" /></span>
          </label>
        </div>
        <label v-else class="sf-field">
          <span class="sf-field__label is-required">{{ t('fileSource.url') }}</span>
          <span class="sf-fld" :class="{ 'is-error': errors.url }">
            <input v-model="form.config.url" class="m-mono" placeholder="https://dav.jianguoyun.com/dav/" autocomplete="off" />
          </span>
          <span v-if="errors.url" class="sf-err">{{ errors.url }}</span>
        </label>
        <div class="sf-grid">
          <label class="sf-field">
            <span class="sf-field__label">{{ t('fileSource.username') }}</span>
            <span class="sf-fld"><input v-model="form.config.username" autocomplete="off" /></span>
          </label>
          <label class="sf-field">
            <span class="sf-field__label">{{ t('fileSource.password') }}</span>
            <span class="sf-fld">
              <input
                v-model="form.config.password"
                type="password"
                autocomplete="new-password"
                :placeholder="form.id ? t('fileSource.secretKeep') : ''"
              />
            </span>
          </label>
        </div>
        <label class="sf-field">
          <span class="sf-field__label">{{ t('fileSource.rootDir') }}</span>
          <span class="sf-fld"><input v-model="form.config.basePath" class="m-mono" placeholder="/" autocomplete="off" /></span>
        </label>
        <div v-if="form.type === 'ftp'" class="sf-opts">
          <label class="sf-opt"><AppSwitch v-model="form.config.tls" /> FTPS (TLS)</label>
        </div>
      </template>

      <div class="sf-rows">
        <div class="sf-row" :class="{ 'is-off': !supportsRedirect }">
          <span class="sf-row__text">
            <span class="sf-row__title">{{ t('fileSource.redirect302') }}</span>
            <span class="sf-row__desc">{{ supportsRedirect ? t('fileSource.redirectHint') : t('fileSource.redirectUnsupported') }}</span>
          </span>
          <AppSwitch v-model="form.redirect302" :disabled="!supportsRedirect" :aria-label="t('fileSource.redirect302')" />
        </div>
        <div class="sf-row">
          <span class="sf-row__text">
            <span class="sf-row__title">{{ form.id ? t('fileSource.enabled') : t('fileSource.enableOnSave') }}</span>
            <span class="sf-row__desc">{{ t('fileSource.enableHint') }}</span>
          </span>
          <AppSwitch v-model="form.enabled" :aria-label="t('fileSource.enabled')" />
        </div>
      </div>

      <div v-if="testResult" class="sf-result" :class="testResult.ok ? 'is-ok' : 'is-bad'" role="status">
        <UIcon :name="testResult.ok ? 'i-lucide-circle-check' : 'i-lucide-circle-alert'" class="sf-result__ico" />
        <span class="sf-result__text">
          <strong>{{ testResult.ok ? t('fileSource.testOk') : t('fileSource.testFailed') }}</strong>
          <span v-if="testResult.latencyMs" class="m-num"> · {{ t('fileSource.latency', { ms: testResult.latencyMs }) }}</span>
          <span v-if="testResult.ok && usageText" class="m-num"> · {{ usageText }}</span>
          <span v-if="!testResult.ok && testResult.message" class="sf-result__msg m-mono">{{ testResult.message }}</span>
        </span>
      </div>

      <div class="sf-actions">
        <UButton class="sf-actions__btn" color="neutral" variant="outline" icon="i-lucide-plug-zap" :loading="testing" @click="testForm">
          {{ t('fileSource.testConnection') }}
        </UButton>
        <UButton class="sf-actions__btn" type="submit" color="primary" :loading="saving">
          {{ t('common.save') }}
        </UButton>
      </div>
    </form>
  </component>
</template>

<script setup lang="ts">
import { defineComponent, h } from 'vue'
import { useI18n } from 'vue-i18n'
import FormDialog from '@/components/ui/FormDialog.vue'
import AppIconButton from '@/components/ui/AppIconButton.vue'
import { formatFileSize } from '@/utils/file'
import {
  createFileSourceRequest,
  updateFileSourceRequest,
  testFileSourceRequest,
} from '@/api/fileSource'
import type { FileSourceInfo, FileSourceConfig, FileSourceTestResult } from '@/types/v1/file'

defineOptions({ name: 'FileSourceCreate' })

withDefaults(defineProps<{ inline?: boolean }>(), { inline: false })
const emits = defineEmits<{ refresh: [type: 'create' | 'update'] }>()
const { t } = useI18n()

// PC 内联表单卡外壳：标题 + 关闭 + 内容（与 FormDialog 同一组 props/emits，便于 <component :is> 切换）。
const InlineShell = defineComponent({
  name: 'SourceFormCard',
  inheritAttrs: false,
  props: { modelValue: Boolean, title: String },
  emits: ['update:modelValue', 'close'],
  setup(props, { slots, emit }) {
    const onClose = () => {
      emit('update:modelValue', false)
      emit('close')
    }
    return () =>
      h('aside', { class: 'sf-card', 'aria-label': props.title }, [
        h('div', { class: 'sf-card__head' }, [
          h('h2', { class: 'sf-card__title' }, props.title),
          h(AppIconButton, { icon: 'HOutline:XMarkIcon', label: t('system.common.close'), box: 36, size: 16, onClick: onClose }),
        ]),
        slots.default?.(),
      ])
  },
})

const ROTS = ['-4deg', '3deg', '-2deg', '4deg']
const open = ref(false)
const saving = ref(false)
const testing = ref(false)
const errors = ref<Record<string, string>>({})

type SourceType = 'local' | 'oss' | 'ftp' | 'webdav'
const typeOptions = computed(() => [
  { label: t('fileSource.localDisk'), value: 'local' as SourceType, desc: t('fileSource.typeDescLocal'), icon: 'i-lucide-hard-drive', tone: 'a' },
  { label: t('fileSource.typeShort.oss'), value: 'oss' as SourceType, desc: t('fileSource.typeDescOss'), icon: 'i-lucide-cloud', tone: 'p' },
  { label: t('fileSource.typeShort.ftp'), value: 'ftp' as SourceType, desc: t('fileSource.typeDescFtp'), icon: 'i-lucide-arrow-left-right', tone: 'ok' },
  { label: t('fileSource.typeShort.webdav'), value: 'webdav' as SourceType, desc: t('fileSource.typeDescWebdav'), icon: 'i-lucide-globe', tone: 'a2' },
])
const NAME_PLACEHOLDER: Record<SourceType, string> = {
  local: 'fileSource.namePlaceholderLocal',
  oss: 'fileSource.namePlaceholderOss',
  ftp: 'fileSource.namePlaceholderFtp',
  webdav: 'fileSource.namePlaceholderWebdav',
}
const namePlaceholder = computed(() => t(NAME_PLACEHOLDER[form.value.type as SourceType] ?? 'fileSource.namePlaceholder'))

const testResult = ref<FileSourceTestResult | null>(null)
const usageText = computed(() => {
  const r = testResult.value
  if (!r) return ''
  const used = Number(r.usedBytes) || 0
  const total = Number(r.totalBytes) || 0
  if (total) return t('fileManager.diskUsed', { used: formatFileSize(used), total: formatFileSize(total) })
  return used ? t('fileManager.usedOnly', { used: formatFileSize(used) }) : ''
})

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
  type: 'oss' as string,
  enabled: true,
  redirect302: true,
  config: emptyConfig(),
})
const form = ref(getDefaultForm())
const title = computed(() => (form.value.id ? t('fileSource.editTitle') : t('fileSource.addTitle')))

// 仅对象存储支持预签名直链(302)。
const supportsRedirect = computed(() => form.value.type === 'oss')

const pickType = (type: SourceType) => {
  if (form.value.id || form.value.type === type) return
  form.value.type = type
  testResult.value = null
  errors.value = {}
  form.value.redirect302 = type === 'oss'
}

const close = () => {
  open.value = false
  saving.value = false
  errors.value = {}
  testResult.value = null
  form.value = getDefaultForm()
}

const isAbsolutePath = (p: string) => /^(\/|[a-zA-Z]:[\\/])/.test(p.trim())
const validate = () => {
  const e: Record<string, string> = {}
  const c = form.value.config
  if (!form.value.name.trim()) e.name = t('fileSource.namePlaceholder')
  if (form.value.type === 'local') {
    if (!c.basePath.trim()) e.basePath = t('fileSource.rootDirRequired')
    else if (!isAbsolutePath(c.basePath)) e.basePath = t('fileSource.rootDirAbsolute')
  } else if (form.value.type === 'oss') {
    if (!c.endpoint.trim()) e.endpoint = t('fileSource.endpoint')
    if (!c.bucket.trim()) e.bucket = t('fileSource.bucket')
  } else if (form.value.type === 'ftp') {
    if (!c.host.trim()) e.host = t('fileSource.host')
  } else if (form.value.type === 'webdav') {
    if (!c.url.trim()) e.url = t('fileSource.url')
  }
  errors.value = e
  return Object.keys(e).length === 0
}

const buildPayloadConfig = (): FileSourceConfig => ({
  ...form.value.config,
  basePath: form.value.config.basePath.trim(),
  port: Number(form.value.config.port) || 0,
})

const confirm = async () => {
  if (!validate()) return
  saving.value = true
  try {
    const redirect302 = supportsRedirect.value && form.value.redirect302
    if (form.value.id) {
      await updateFileSourceRequest({
        id: form.value.id,
        name: form.value.name,
        enabled: form.value.enabled,
        redirect302,
        config: buildPayloadConfig(),
      })
      feedback.success(t('fileSource.updateSuccess'))
      emits('refresh', 'update')
    } else {
      await createFileSourceRequest({
        name: form.value.name,
        type: form.value.type,
        enabled: form.value.enabled,
        redirect302,
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
  if (form.value.type === 'local' && !validate()) return
  testing.value = true
  try {
    // 编辑已保存的来源且未改密钥时，用 id 测试（服务端用已存密钥）；否则测当前填写的配置。
    const useSaved = !!form.value.id && !form.value.config.secretKey && !form.value.config.password
    const { data } = await testFileSourceRequest(
      useSaved
        ? { id: form.value.id!, type: '', config: undefined }
        : { id: '', type: form.value.type, config: buildPayloadConfig() },
    )
    testResult.value = data.result ?? null
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

defineExpose({ showDialog, isOpen: () => open.value })
</script>

<style scoped lang="scss">
.sf {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.sf-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.sf-lbl {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.sf-hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}
.sf-types {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}
.sf-type {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  min-height: 64px;
  padding: 10px;
  border: 2px solid var(--el-border-color-lighter);
  border-radius: 18px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.sf-type:hover:not(:disabled) {
  border-color: var(--el-text-color-secondary);
}
.sf-type.is-on,
.sf-type.is-on:hover {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.sf-type:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.sf-type__sticker {
  flex: none;
  width: 38px;
  height: 38px;
  border-radius: 13px;
}
.sf-type__sticker :deep(svg),
.sf-type__sticker :deep(.iconify) {
  width: 18px;
  height: 18px;
}
.sf-type__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.sf-type__name {
  font-size: 0.875rem;
  font-weight: 800;
}
.sf-type__desc {
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sf-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.sf-field__label {
  font-size: 0.8125rem;
  font-weight: 700;
}
.sf-field__label.is-required::after {
  content: ' *';
  color: var(--m-bad);
}
.sf-fld {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 48px;
  padding: 0 14px;
  border: 1.5px solid transparent;
  border-radius: 14px;
  background: var(--m-s2);
  color: var(--el-text-color-secondary);
  transition: border-color 0.15s, background 0.15s;
}
.sf-fld:focus-within {
  border-color: var(--m-p);
  background: var(--el-bg-color);
}
.sf-fld.is-error {
  border-color: var(--m-bad);
}
.sf-fld__ico {
  flex: none;
  width: 16px;
  height: 16px;
}
.sf-fld input {
  flex: 1;
  min-width: 0;
  height: 44px;
  border: 0;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 500;
}
.sf-fld input.m-mono {
  font-size: 0.8125rem;
}
.sf-fld input::placeholder {
  color: var(--el-text-color-secondary);
}
.sf-err {
  font-size: 0.75rem;
  color: var(--m-bad);
}
.sf-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}
.sf-grid--host {
  grid-template-columns: minmax(0, 1fr) 110px;
}
.sf-opts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 18px;
}
.sf-opt {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.sf-rows {
  display: flex;
  flex-direction: column;
  margin-top: 4px;
}
.sf-row {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 56px;
  padding: 6px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.sf-row__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.sf-row__title {
  font-size: 0.875rem;
  font-weight: 700;
}
.sf-row.is-off .sf-row__title {
  color: var(--el-text-color-secondary);
}
.sf-row__desc {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.sf-result {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  font-size: 0.8125rem;
}
.sf-result.is-ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.sf-result.is-bad {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.sf-result__ico {
  flex: none;
  width: 18px;
  height: 18px;
  margin-top: 1px;
}
.sf-result__text {
  min-width: 0;
}
.sf-result__msg {
  display: block;
  margin-top: 2px;
  font-size: 0.75rem;
  word-break: break-all;
}
.sf-actions {
  display: flex;
  gap: 10px;
}
.sf-actions__btn {
  flex: 1;
  justify-content: center;
  min-height: 44px;
  border-radius: 999px;
  font-weight: 700;
}

@media (width <= 768px) {
  .sf-grid,
  .sf-grid--host {
    grid-template-columns: minmax(0, 1fr);
  }
  .sf-type__desc {
    white-space: normal;
  }
}
</style>

<!-- PC 表单卡外壳由脚本内的渲染函数组件生成（不带 scoped 标记），样式放非 scoped 块；类名以 sf-card 前缀隔离。 -->
<style lang="scss">
/* PC 表单卡（文件源列表右侧，吸顶） */
.sf-card {
  position: sticky;
  top: 0;
  flex: 0 1 380px;
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-width: 0;
  padding: calc(var(--cmp-card-unit) * 22);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-lg);
}
.sf-card__head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.sf-card__title {
  flex: 1;
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
}
</style>
