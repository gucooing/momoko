<template>
  <div>
    <el-card shadow="never">
      <div class="operation-container">
        <el-form :model="queryForm" inline @keyup.enter="loadClients">
          <el-form-item>
            <el-input v-model="queryForm.keywords" placeholder="客户端名称" clearable />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :icon="menuStore.iconComponents.Search" @click="loadClients">搜索</el-button>
            <el-button :icon="menuStore.iconComponents.Refresh" @click="resetQuery">重置</el-button>
          </el-form-item>
        </el-form>
        <div class="operation-actions">
          <el-button :icon="menuStore.iconComponents['HOutline:Cog6ToothIcon']" @click="openConfigDialog">OIDC 配置</el-button>
          <el-button type="primary" :icon="menuStore.iconComponents.Plus" :disabled="!canEdit" @click="openClientDialog()">
            生成客户端配置
          </el-button>
        </div>
      </div>

      <el-table v-loading="clientsLoading" :data="clients" border row-key="id" show-overflow-tooltip>
        <el-table-column type="index" label="序号" width="60" />
        <el-table-column prop="name" label="Provider 名称" min-width="150" />
        <el-table-column prop="clientId" label="Client ID" min-width="230" />
        <el-table-column prop="clientSecret" label="Client Secret" min-width="180" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }: { row: OIDCClientInfo }">
            <BaseTag :type="row.active ? 'success' : 'danger'" :text="row.active ? '启用' : '停用'" />
          </template>
        </el-table-column>
        <el-table-column label="回调地址" min-width="240">
          <template #default="{ row }: { row: OIDCClientInfo }">
            {{ row.redirectUris.join(', ') }}
          </template>
        </el-table-column>
        <el-table-column label="Scopes" min-width="160">
          <template #default="{ row }: { row: OIDCClientInfo }">
            {{ row.scopes.join(' ') }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }: { row: OIDCClientInfo }">
            <el-button type="primary" link :icon="menuStore.iconComponents.Edit" :disabled="!canEdit" @click="openClientDialog(row)">编辑</el-button>
            <el-button type="primary" link :icon="menuStore.iconComponents.Refresh" :disabled="!canEdit" @click="refreshSecret(row)">刷新密钥</el-button>
            <el-button type="danger" link :icon="menuStore.iconComponents.Delete" :disabled="!canEdit" @click="deleteClient(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <TablePagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :is-mobile="menuStore.isMobile"
        @change="loadClients"
      />
    </el-card>

    <BaseDialog v-model="configDialogVisible" title="OIDC 配置" width="760">
      <div class="setting-group" v-loading="configLoading">
        <div class="setting-item">
          <div class="setting-item-info">
            <span class="setting-item-label">启用 OIDC 服务端</span>
            <span class="setting-item-desc">开启后可通过 Discovery、Authorize、Token、UserInfo、JWKS 端点接入第三方系统</span>
          </div>
          <el-switch v-model="configForm.enabled" :disabled="!canEdit" />
        </div>
        <div class="setting-item">
          <div class="setting-item-info">
            <span class="setting-item-label">Issuer URL</span>
            <span class="setting-item-desc">OIDC Provider 的对外访问域名</span>
          </div>
          <el-input v-model="configForm.issuerUrl" :disabled="!canEdit" placeholder="https://example.com" class="wide-input">
            <template #append>
              <el-button :disabled="!canEdit" @click="useCurrentOrigin">当前域名</el-button>
            </template>
          </el-input>
        </div>
        <div class="ttl-grid">
          <div class="setting-item">
            <div class="setting-item-info"><span class="setting-item-label">Access Token 有效期（秒）</span></div>
            <el-input-number v-model="configForm.accessTokenTtlSeconds" :disabled="!canEdit" :min="60" :max="86400" />
          </div>
          <div class="setting-item">
            <div class="setting-item-info"><span class="setting-item-label">ID Token 有效期（秒）</span></div>
            <el-input-number v-model="configForm.idTokenTtlSeconds" :disabled="!canEdit" :min="60" :max="86400" />
          </div>
          <div class="setting-item">
            <div class="setting-item-info"><span class="setting-item-label">授权码有效期（秒）</span></div>
            <el-input-number v-model="configForm.authorizationCodeTtlSeconds" :disabled="!canEdit" :min="60" :max="1800" />
          </div>
        </div>
        <div class="endpoint-grid" v-if="config?.endpoints">
          <div v-for="item in endpointItems" :key="item.label" class="endpoint-item">
            <span class="endpoint-label">{{ item.label }}</span>
            <el-input :model-value="item.value" readonly>
              <template #append>
                <el-button :icon="menuStore.iconComponents['HOutline:ClipboardDocumentIcon']" @click="copyText(item.value)" />
              </template>
            </el-input>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="closeConfigDialog">取消</el-button>
        <el-button type="primary" :loading="configSaving" :disabled="!canEdit" @click="saveConfig">保存配置</el-button>
      </template>
    </BaseDialog>

    <BaseDialog v-model="clientDialogVisible" :title="clientForm.id ? '编辑 OIDC 客户端' : '生成 OIDC 客户端配置'" width="620">
      <el-form label-position="top">
        <el-form-item label="Provider 名称">
          <el-input v-model="clientForm.name" placeholder="OIDC" />
        </el-form-item>
        <el-form-item label="回调地址">
          <el-input v-model="clientForm.redirectUrisText" type="textarea" :rows="4" placeholder="每行一个 Redirect URI" />
        </el-form-item>
        <el-form-item label="Scopes">
          <el-input v-model="clientForm.scopesText" placeholder="openid email profile" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="clientForm.active" active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="clientDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="clientSaving" @click="saveClient">确定</el-button>
      </template>
    </BaseDialog>

    <BaseDialog v-model="clientConfigDialogVisible" title="OIDC 客户端配置" width="760">
      <el-alert title="Client Secret 只会完整显示一次，请在关闭前复制保存" type="warning" show-icon :closable="false" class="secret-alert" />
      <div class="client-config-grid">
        <div v-for="item in clientConfigItems" :key="item.label" class="client-config-item">
          <span class="client-config-label">{{ item.label }}</span>
          <div class="client-config-control" :class="{ 'is-multiline': item.multiline }">
            <el-input :model-value="item.value" readonly :type="item.multiline ? 'textarea' : 'text'" :autosize="item.multiline ? { minRows: 2, maxRows: 4 } : undefined" />
            <el-button :icon="menuStore.iconComponents['HOutline:ClipboardDocumentIcon']" @click="copyText(item.value)">复制</el-button>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="clientConfigDialogVisible = false">关闭</el-button>
        <el-button type="primary" :icon="menuStore.iconComponents['HOutline:ClipboardDocumentIcon']" @click="copyText(clientConfigText)">
          复制全部
        </el-button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import {
  createOIDCClient,
  deleteOIDCClient,
  getOIDCConfig,
  listOIDCClients,
  refreshOIDCClientSecret,
  updateOIDCClient,
  updateOIDCConfig,
} from '@/api/oidc'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import TablePagination from '@/components/pagination/TablePagination.vue'
import BaseTag from '@/components/tag/BaseTag.vue'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { PERM } from '@/config/permission'
import type { OIDCClientInfo, OIDCConfig } from '@/types/v1/oidc'
import { Dialog } from '@/utils/dialog'

defineOptions({ name: 'SystemOIDCView' })

const menuStore = useMenuStore()
const canEdit = useButtonPermission([PERM.OIDC_EDIT], [])

const config = ref<OIDCConfig | null>(null)
const configLoading = ref(false)
const configSaving = ref(false)
const clientsLoading = ref(false)
const clientSaving = ref(false)
const configDialogVisible = ref(false)
const clientDialogVisible = ref(false)
const clientConfigDialogVisible = ref(false)
const clientConfigResult = ref<OIDCClientInfo | null>(null)
const clients = ref<OIDCClientInfo[]>([])

const configForm = reactive({
  enabled: false,
  issuerUrl: '',
  accessTokenTtlSeconds: 3600,
  idTokenTtlSeconds: 3600,
  authorizationCodeTtlSeconds: 300,
})

const queryForm = reactive({
  keywords: '',
})

const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0,
})

const clientForm = reactive({
  id: '',
  name: 'OIDC',
  redirectUrisText: '',
  scopesText: 'openid email profile',
  active: true,
})

const endpointItems = computed(() => {
  const endpoints = config.value?.endpoints
  if (!endpoints) return []
  return [
    { label: 'Issuer URL', value: endpoints.issuerUrl },
    { label: 'Discovery URL', value: endpoints.discoveryUrl },
    { label: 'Authorize URL', value: endpoints.authorizeUrl },
    { label: 'Token URL', value: endpoints.tokenUrl },
    { label: 'UserInfo URL', value: endpoints.userinfoUrl },
    { label: 'JWKS URL', value: endpoints.jwksUrl },
    { label: 'Scopes', value: endpoints.scopes },
    { label: 'Token 鉴权方式', value: endpoints.tokenEndpointAuthMethod },
    { label: '允许的签名算法', value: endpoints.allowedIdTokenSigningAlgs },
  ]
})

const clientConfigItems = computed(() => {
  const client = clientConfigResult.value
  const endpoints = config.value?.endpoints
  if (!client) return []
  return [
    { label: 'Provider 名称', value: client.name },
    { label: 'Client ID', value: client.clientId },
    { label: 'Client Secret', value: client.clientSecret },
    { label: '回调地址', value: client.redirectUris.join('\n'), multiline: true },
    { label: 'Scopes', value: client.scopes.join(' ') },
    { label: 'Issuer URL', value: endpoints?.issuerUrl || configForm.issuerUrl },
    { label: 'Discovery URL', value: endpoints?.discoveryUrl || '' },
    { label: 'Authorize URL', value: endpoints?.authorizeUrl || '' },
    { label: 'Token URL', value: endpoints?.tokenUrl || '' },
    { label: 'UserInfo URL', value: endpoints?.userinfoUrl || '' },
    { label: 'JWKS URL', value: endpoints?.jwksUrl || '' },
    { label: 'Token 鉴权方式', value: endpoints?.tokenEndpointAuthMethod || 'client_secret_post' },
    { label: '允许的签名算法', value: endpoints?.allowedIdTokenSigningAlgs || 'RS256,ES256,PS256' },
    { label: '时钟偏移（秒）', value: String(endpoints?.clockSkewSeconds || 120) },
  ].filter((item) => item.value)
})

const clientConfigText = computed(() => clientConfigItems.value.map((item) => `${item.label}: ${item.value}`).join('\n'))

const splitLines = (value: string) =>
  value
    .split(/[\n,\s]+/)
    .map((item) => item.trim())
    .filter(Boolean)

// 将接口返回的 OIDC 配置回填到弹窗表单，取消编辑和重新加载时都使用同一份字段映射。
const setConfigForm = (value: OIDCConfig) => {
  configForm.enabled = value.enabled
  configForm.issuerUrl = value.issuerUrl
  configForm.accessTokenTtlSeconds = value.accessTokenTtlSeconds
  configForm.idTokenTtlSeconds = value.idTokenTtlSeconds
  configForm.authorizationCodeTtlSeconds = value.authorizationCodeTtlSeconds
}

const loadConfig = async () => {
  configLoading.value = true
  try {
    const { data } = await getOIDCConfig()
    config.value = data?.config || null
    if (data?.config) {
      setConfigForm(data.config)
    }
  } finally {
    configLoading.value = false
  }
}

const saveConfig = async () => {
  configSaving.value = true
  try {
    const { data } = await updateOIDCConfig({ ...configForm })
    config.value = data?.config || null
    if (data?.config) {
      setConfigForm(data.config)
    }
    configDialogVisible.value = false
    ElMessage.success('保存成功')
  } finally {
    configSaving.value = false
  }
}

const loadClients = async () => {
  clientsLoading.value = true
  try {
    const { data } = await listOIDCClients({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keywords: queryForm.keywords || undefined,
    })
    clients.value = data?.clients || []
    pagination.total = Number(data?.total || 0)
  } finally {
    clientsLoading.value = false
  }
}

const resetQuery = () => {
  queryForm.keywords = ''
  pagination.page = 1
  loadClients()
}

const useCurrentOrigin = () => {
  configForm.issuerUrl = window.location.origin
}

// 配置表单只在用户点击入口后展示；初始化未取到配置时，打开弹窗后补一次读取。
const openConfigDialog = async () => {
  configDialogVisible.value = true
  if (!config.value && !configLoading.value) {
    await loadConfig()
  }
}

const closeConfigDialog = () => {
  if (config.value) {
    setConfigForm(config.value)
  }
  configDialogVisible.value = false
}

const openClientDialog = (row?: OIDCClientInfo) => {
  clientForm.id = row?.id || ''
  clientForm.name = row?.name || 'OIDC'
  clientForm.redirectUrisText = row?.redirectUris.join('\n') || ''
  clientForm.scopesText = row?.scopes.join(' ') || 'openid email profile'
  clientForm.active = row?.active ?? true
  clientDialogVisible.value = true
}

const saveClient = async () => {
  clientSaving.value = true
  try {
    const payload = {
      name: clientForm.name,
      redirectUris: splitLines(clientForm.redirectUrisText),
      scopes: splitLines(clientForm.scopesText),
      active: clientForm.active,
    }
    const { data } = clientForm.id
      ? await updateOIDCClient({ id: clientForm.id, ...payload })
      : await createOIDCClient(payload)
    clientDialogVisible.value = false
    ElMessage.success(clientForm.id ? '编辑成功' : '生成成功')
    if (data?.client?.clientSecret && !data.client.clientSecret.includes('*')) {
      clientConfigResult.value = data.client
      clientConfigDialogVisible.value = true
    }
    loadClients()
  } finally {
    clientSaving.value = false
  }
}

const refreshSecret = async (row: OIDCClientInfo) => {
  try {
    await Dialog.confirm({
      title: '刷新 Client Secret',
      content: '刷新后旧 Client Secret 将立即失效。',
      confirmText: '确定',
      cancelText: '取消',
    })
  } catch {
    return
  }
  const { data } = await refreshOIDCClientSecret({ id: row.id })
  if (data?.client) {
    clientConfigResult.value = data.client
    clientConfigDialogVisible.value = true
  }
  loadClients()
}

const deleteClient = async (row: OIDCClientInfo) => {
  try {
    await Dialog.confirm({
      title: '删除 OIDC 客户端',
      content: `确定删除 ${row.name} 吗？`,
      confirmText: '删除',
      cancelText: '取消',
    })
  } catch {
    return
  }
  await deleteOIDCClient({ id: row.id })
  ElMessage.success('删除成功')
  loadClients()
}

const copyText = async (value: string) => {
  await navigator.clipboard.writeText(value)
  ElMessage.success('已复制')
}

onMounted(() => {
  loadConfig()
  loadClients()
})
</script>

<style scoped lang="scss">
.setting-group {
  display: flex;
  flex-direction: column;
  gap: 1px;
  background: var(--el-border-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 0.5rem;
  overflow: hidden;
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 0.75rem;
  background: var(--el-bg-color-overlay);
}

.setting-item-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  flex: 1;
  min-width: 0;
  margin-right: 1rem;
}

.setting-item-label {
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
}

.setting-item-desc {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}

.wide-input {
  width: min(520px, 100%);
}

.ttl-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
}

.endpoint-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1px;
}

.endpoint-item {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.5rem 0.75rem;
  background: var(--el-bg-color-overlay);
}

.endpoint-label {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

.operation-container {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.operation-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.operation-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

.secret-alert {
  margin-bottom: 0.75rem;
}

.client-config-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

.client-config-item {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  min-width: 0;
}

.client-config-label {
  font-size: 0.78rem;
  color: var(--el-text-color-secondary);
}

.client-config-control {
  display: flex;
  gap: 0.5rem;

  .el-input,
  .el-textarea {
    min-width: 0;
  }
}

.client-config-control.is-multiline {
  align-items: flex-start;
}

@media (width <= 768px) {
  .setting-item,
  .operation-container {
    flex-direction: column;
    align-items: stretch;
  }

  .operation-actions {
    flex-direction: column;
    width: 100%;
  }

  .operation-actions :deep(.el-button) {
    width: 100%;
  }

  .client-config-control {
    flex-direction: column;
  }

  .setting-item-info {
    margin-right: 0;
  }

  .ttl-grid,
  .endpoint-grid,
  .client-config-grid {
    grid-template-columns: 1fr;
  }

  .wide-input,
  .setting-item :deep(.el-input),
  .setting-item :deep(.el-input-number) {
    width: 100%;
  }
}
</style>
