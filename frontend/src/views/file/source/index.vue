<template>
  <div>
    <el-card shadow="never" class="card-clear-mb">
      <div class="fs-header">
        <el-button type="primary" @click="openCreate">{{ t('fileSource.add') }}</el-button>
      </div>
    </el-card>

    <el-card shadow="never" class="card-mt-16">
      <el-table v-loading="loading" :data="items" stripe>
        <el-table-column :label="t('fileSource.name')" prop="name" min-width="140" />
        <el-table-column :label="t('fileSource.type')" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="typeTagType(row.type)">{{ typeLabel(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('fileSource.enabled')" width="90">
          <template #default="{ row }">
            <el-switch
              :model-value="row.enabled"
              size="small"
              @change="(v: string | number | boolean) => toggleEnabled(row, !!v)"
            />
          </template>
        </el-table-column>
        <el-table-column :label="t('fileSource.redirect302')" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.caps?.presign && row.redirect302" type="success" size="small">
              {{ t('fileSource.redirectOn') }}
            </el-tag>
            <span v-else class="fs-muted">{{ t('fileSource.redirectOff') }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('fileSource.creator')" prop="creatorName" width="120" />
        <el-table-column :label="t('fileSource.createTime')" width="170">
          <template #default="{ row }">{{ formatDateTime(row.createTime) }}</template>
        </el-table-column>
        <el-table-column :label="t('system.common.operation')" width="220" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="testExisting(row)">
              {{ t('fileSource.test') }}
            </el-button>
            <el-button type="primary" link @click="openEdit(row)">
              {{ t('system.common.edit') }}
            </el-button>
            <el-popconfirm
              :title="t('fileSource.confirmDelete', { name: row.name })"
              @confirm="remove(row)"
            >
              <template #reference>
                <el-button type="danger" link>{{ t('system.common.delete') }}</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
        <template #empty>{{ t('fileSource.empty') }}</template>
      </el-table>
    </el-card>

    <!-- 新增/编辑弹窗 -->
    <el-dialog
      v-model="dialog.open"
      :title="dialog.isEdit ? t('fileSource.editTitle') : t('fileSource.addTitle')"
      width="560px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
        <el-form-item :label="t('fileSource.name')" prop="name">
          <el-input v-model="form.name" :placeholder="t('fileSource.namePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('fileSource.type')" prop="type">
          <el-select v-model="form.type" :disabled="dialog.isEdit" @change="onTypeChange">
            <el-option :label="t('fileSource.typeOss')" value="oss" />
            <el-option :label="t('fileSource.typeFtp')" value="ftp" />
            <el-option :label="t('fileSource.typeWebdav')" value="webdav" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('fileSource.enabled')">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item :label="t('fileSource.redirect302')">
          <el-switch v-model="form.redirect302" :disabled="!supportsRedirect" />
          <span class="fs-field-hint">
            {{ supportsRedirect ? t('fileSource.redirectHint') : t('fileSource.redirectUnsupported') }}
          </span>
        </el-form-item>

        <!-- OSS / S3 -->
        <template v-if="form.type === 'oss'">
          <el-form-item :label="t('fileSource.endpoint')" prop="config.endpoint">
            <el-input v-model="form.config.endpoint" placeholder="oss-cn-hangzhou.aliyuncs.com" />
          </el-form-item>
          <el-form-item :label="t('fileSource.bucket')" prop="config.bucket">
            <el-input v-model="form.config.bucket" />
          </el-form-item>
          <el-form-item :label="t('fileSource.region')">
            <el-input v-model="form.config.region" placeholder="cn-hangzhou" />
          </el-form-item>
          <el-form-item :label="t('fileSource.accessKey')">
            <el-input v-model="form.config.accessKey" />
          </el-form-item>
          <el-form-item :label="t('fileSource.secretKey')">
            <el-input
              v-model="form.config.secretKey"
              type="password"
              show-password
              :placeholder="dialog.isEdit ? t('fileSource.secretKeep') : ''"
            />
          </el-form-item>
          <el-form-item :label="t('fileSource.prefix')">
            <el-input v-model="form.config.prefix" placeholder="momoko/" />
          </el-form-item>
          <el-form-item :label="t('fileSource.options')">
            <el-checkbox v-model="form.config.useSsl">HTTPS</el-checkbox>
            <el-checkbox v-model="form.config.pathStyle">Path-Style</el-checkbox>
          </el-form-item>
        </template>

        <!-- FTP -->
        <template v-else-if="form.type === 'ftp'">
          <el-form-item :label="t('fileSource.host')" prop="config.host">
            <el-input v-model="form.config.host" />
          </el-form-item>
          <el-form-item :label="t('fileSource.port')">
            <el-input v-model.number="form.config.port" type="number" placeholder="21" />
          </el-form-item>
          <el-form-item :label="t('fileSource.username')">
            <el-input v-model="form.config.username" />
          </el-form-item>
          <el-form-item :label="t('fileSource.password')">
            <el-input
              v-model="form.config.password"
              type="password"
              show-password
              :placeholder="dialog.isEdit ? t('fileSource.secretKeep') : ''"
            />
          </el-form-item>
          <el-form-item :label="t('fileSource.basePath')">
            <el-input v-model="form.config.basePath" placeholder="/" />
          </el-form-item>
          <el-form-item :label="t('fileSource.options')">
            <el-checkbox v-model="form.config.tls">FTPS (TLS)</el-checkbox>
          </el-form-item>
        </template>

        <!-- WebDAV -->
        <template v-else-if="form.type === 'webdav'">
          <el-form-item :label="t('fileSource.url')" prop="config.url">
            <el-input v-model="form.config.url" placeholder="https://dav.example.com/dav" />
          </el-form-item>
          <el-form-item :label="t('fileSource.username')">
            <el-input v-model="form.config.username" />
          </el-form-item>
          <el-form-item :label="t('fileSource.password')">
            <el-input
              v-model="form.config.password"
              type="password"
              show-password
              :placeholder="dialog.isEdit ? t('fileSource.secretKeep') : ''"
            />
          </el-form-item>
          <el-form-item :label="t('fileSource.basePath')">
            <el-input v-model="form.config.basePath" placeholder="/" />
          </el-form-item>
        </template>
      </el-form>

      <template #footer>
        <el-button :loading="testing" @click="testForm">{{ t('fileSource.test') }}</el-button>
        <el-button @click="dialog.open = false">{{ t('system.common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="save">
          {{ t('system.common.confirm') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'FileSourceView' })
import { useI18n } from 'vue-i18n'
import type { FormInstance, FormRules } from 'element-plus'
import { showRequestError } from '@/utils/request'
import { formatDateTime } from '@/utils/file'
import {
  listFileSourcesRequest,
  createFileSourceRequest,
  updateFileSourceRequest,
  deleteFileSourceRequest,
  testFileSourceRequest,
} from '@/api/fileSource'
import type { FileSourceInfo, FileSourceConfig } from '@/types/v1/file'

const { t } = useI18n()

const loading = ref(false)
const items = ref<FileSourceInfo[]>([])

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

const dialog = reactive({ open: false, isEdit: false, id: '' })
const form = reactive({
  name: '',
  type: 'oss',
  enabled: true,
  redirect302: false,
  config: emptyConfig(),
})
const formRef = ref<FormInstance>()
const saving = ref(false)
const testing = ref(false)

const rules: FormRules = {
  name: [{ required: true, message: t('fileSource.namePlaceholder'), trigger: 'blur' }],
  type: [{ required: true, trigger: 'change' }],
}

// 仅对象存储支持预签名直链(302)。
const supportsRedirect = computed(() => form.type === 'oss')

const typeLabel = (type: string) =>
  type === 'oss'
    ? t('fileSource.typeOss')
    : type === 'ftp'
      ? t('fileSource.typeFtp')
      : type === 'webdav'
        ? t('fileSource.typeWebdav')
        : type
const typeTagType = (type: string) =>
  type === 'oss' ? 'primary' : type === 'ftp' ? 'warning' : 'success'

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listFileSourcesRequest()
    items.value = data.items ?? []
  } catch (error) {
    showRequestError(error, t('fileSource.loadFailed'))
  } finally {
    loading.value = false
  }
}

const onTypeChange = () => {
  if (!supportsRedirect.value) form.redirect302 = false
}

const openCreate = () => {
  dialog.isEdit = false
  dialog.id = ''
  form.name = ''
  form.type = 'oss'
  form.enabled = true
  form.redirect302 = false
  Object.assign(form.config, emptyConfig())
  dialog.open = true
}

const openEdit = (row: FileSourceInfo) => {
  dialog.isEdit = true
  dialog.id = row.id
  form.name = row.name
  form.type = row.type
  form.enabled = row.enabled
  form.redirect302 = row.redirect302
  // 密钥不回显（留空=保留原值）
  Object.assign(form.config, emptyConfig(), row.config ?? {}, { secretKey: '', password: '' })
  dialog.open = true
}

const buildPayloadConfig = (): FileSourceConfig => ({ ...form.config, port: Number(form.config.port) || 0 })

const save = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    if (dialog.isEdit) {
      await updateFileSourceRequest({
        id: dialog.id,
        name: form.name,
        enabled: form.enabled,
        redirect302: form.redirect302,
        config: buildPayloadConfig(),
      })
      ElMessage.success(t('fileSource.updateSuccess'))
    } else {
      await createFileSourceRequest({
        name: form.name,
        type: form.type,
        enabled: form.enabled,
        redirect302: form.redirect302,
        config: buildPayloadConfig(),
      })
      ElMessage.success(t('fileSource.createSuccess'))
    }
    dialog.open = false
    getList()
  } catch (error) {
    showRequestError(error, t('fileSource.saveFailed'))
  } finally {
    saving.value = false
  }
}

const runTest = async (payload: { id?: string; type?: string; config?: FileSourceConfig }) => {
  testing.value = true
  try {
    const { data } = await testFileSourceRequest({
      id: payload.id ?? '',
      type: payload.type ?? '',
      config: payload.config,
    })
    if (data.ok) ElMessage.success(data.message || t('fileSource.testOk'))
    else ElMessage.error(data.message || t('fileSource.testFailed'))
  } catch (error) {
    showRequestError(error, t('fileSource.testFailed'))
  } finally {
    testing.value = false
  }
}

const testForm = () => runTest({ type: form.type, config: buildPayloadConfig() })
const testExisting = (row: FileSourceInfo) => runTest({ id: row.id })

const toggleEnabled = async (row: FileSourceInfo, enabled: boolean) => {
  try {
    await updateFileSourceRequest({
      id: row.id,
      name: row.name,
      enabled,
      redirect302: row.redirect302,
      config: { ...emptyConfig(), ...(row.config ?? {}), secretKey: '', password: '' },
    })
    row.enabled = enabled
    ElMessage.success(t('fileSource.updateSuccess'))
  } catch (error) {
    showRequestError(error, t('fileSource.saveFailed'))
  }
}

const remove = async (row: FileSourceInfo) => {
  try {
    await deleteFileSourceRequest(row.id)
    ElMessage.success(t('fileSource.deleteSuccess'))
    getList()
  } catch (error) {
    showRequestError(error, t('fileSource.deleteFailed'))
  }
}

onMounted(getList)
</script>

<style scoped>
.fs-header {
  display: flex;
  align-items: center;
  gap: 1rem;
}
.fs-muted {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}
.fs-field-hint {
  margin-left: 0.75rem;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
