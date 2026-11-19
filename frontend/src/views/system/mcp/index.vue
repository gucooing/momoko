<!-- Momoko MCP：对外的 MCP 服务器（让 Agent 等 MCP 客户端操作 Momoko）。
     页头（工具数 · 凭证数 + 新建凭证）→ 连接（地址、传输方式、请求头）→ 凭证（自己的 API Key 里授予了工具权限的：
     能调用的工具、状态、最后使用，行尾查看 / 复制 MCP 配置）→ 工具（名称、用途、需要的权限）。
     凭证就是 API Key：新建用 API Key 的弹窗（预填工具要的权限），改权限、吊销到「节点与 API Key」。 -->
<template>
  <div class="mcp-page">
    <PageHeader title="Momoko MCP">
      <template #description>
        <template v-if="loaded">{{ t('mcp.summary', { tools: tools.length, credentials: credentials.length }) }}</template>
        <template v-else>{{ t('system.common.loading') }}</template>
      </template>
      <template #actions>
        <UButton color="primary" icon="i-lucide-plus" :disabled="!loaded" @click="createCredential">{{ t('mcp.credential.add') }}</UButton>
      </template>
    </PageHeader>

    <AppPanel :title="t('mcp.connection.title')" title-icon="HOutline:LinkIcon">
      <dl class="mcp-conn">
        <div class="mcp-conn__row">
          <dt>{{ t('mcp.connection.endpoint') }}</dt>
          <dd class="m-mono">{{ endpoint || '—' }}</dd>
          <AppIconButton
            icon="HOutline:ClipboardDocumentIcon"
            :box="34"
            :size="17"
            :label="t('mcp.connection.copyEndpoint')"
            @click="copyEndpoint"
          />
        </div>
        <div class="mcp-conn__row">
          <dt>{{ t('mcp.connection.transport') }}</dt>
          <dd>{{ t('mcp.connection.transportValue') }}</dd>
        </div>
        <div class="mcp-conn__row">
          <dt>{{ t('mcp.connection.header') }}</dt>
          <dd class="m-mono">Authorization: Bearer &lt;API Key&gt;</dd>
        </div>
      </dl>
    </AppPanel>

    <AppPanel :title="t('mcp.credential.title')" title-icon="HOutline:KeyIcon" :caption="loaded ? String(credentials.length) : ''">
      <template v-if="canManageKeys" #actions>
        <UButton color="neutral" variant="soft" size="sm" :to="pageTabTo('node', 'key')">{{ t('mcp.credential.manage') }}</UButton>
      </template>
      <ul v-if="credentials.length" class="mcp-list">
        <li v-for="(c, idx) in credentials" :key="c.id" class="mcp-cred">
          <span class="m-sticker mcp-cred__av" :class="`m-sticker--${TONES[idx % TONES.length]}`">
            <component :is="menuStore.iconComponents['HOutline:KeyIcon']" />
          </span>
          <div class="mcp-cred__main">
            <div class="mcp-cred__title">
              <span class="mcp-cred__name">{{ c.name }}</span>
              <StatusPill :variant="statusPill(c).variant" :label="statusPill(c).label" />
            </div>
            <span class="mcp-cred__sub m-mono">{{ c.keyId }}</span>
          </div>
          <div class="mcp-cred__tools">
            <span class="mcp-cred__label">{{ t('mcp.credential.tools') }}</span>
            <div class="mcp-chips">
              <span v-for="name in c.tools" :key="name" class="mcp-chip m-mono">{{ name }}</span>
              <span v-if="!c.tools.length" class="mcp-cred__none" :class="{ 'is-warn': c.status !== APIKeyStatus.APIKeyStatus_Expired }">
                {{ c.status === APIKeyStatus.APIKeyStatus_Expired ? '—' : t('mcp.credential.noTools') }}
              </span>
            </div>
          </div>
          <div class="mcp-cred__used">
            <span class="mcp-cred__label">{{ t('node.keyx.lastUsed') }}</span>
            <span class="m-num" :title="c.lastUsedTime ? dateTime(c.lastUsedTime) : ''">
              {{ c.lastUsedTime ? ago(c.lastUsedTime) : t('node.keyx.neverUsed') }}
            </span>
            <span v-if="c.lastUsedIp" class="mcp-cred__sub m-mono">{{ c.lastUsedIp }}</span>
          </div>
          <AppIconButton
            class="mcp-cred__op"
            icon="HOutline:DocumentDuplicateIcon"
            :box="menuStore.isMobile ? 44 : 34"
            :size="17"
            :label="t('mcp.credential.configOf', { name: c.name })"
            @click="openConfig(c)"
          />
        </li>
      </ul>
      <EmptyState v-else-if="loaded" icon="HOutline:KeyIcon" :title="t('mcp.credential.empty')">
        <template #action>
          <UButton color="primary" variant="soft" icon="i-lucide-plus" @click="createCredential">{{ t('mcp.credential.add') }}</UButton>
        </template>
      </EmptyState>
    </AppPanel>

    <AppPanel :title="t('mcp.tool.title')" title-icon="HOutline:WrenchScrewdriverIcon" :caption="loaded ? String(tools.length) : ''">
      <ul class="mcp-list">
        <li v-for="tool in tools" :key="tool.name" class="mcp-tool">
          <div class="mcp-tool__main">
            <span class="mcp-tool__name m-mono">{{ tool.name }}</span>
            <p v-if="te(`mcp.tool.items.${tool.name}`)" class="mcp-tool__desc">{{ t(`mcp.tool.items.${tool.name}`) }}</p>
          </div>
          <div class="mcp-tool__perm">
            <span class="mcp-cred__label">{{ t('mcp.tool.permission') }}</span>
            <span>{{ permissionLabel(tool.permission) }}</span>
            <span class="mcp-cred__sub m-mono">{{ tool.permission }}</span>
          </div>
        </li>
      </ul>
    </AppPanel>

    <ApiKeyCreate ref="apiKeyCreateRef" @refresh="onKeyCreated" />

    <FormDialog v-model="configOpen" :title="t('mcp.config.title', { name: configName })" :width="600">
      <pre class="mcp-config m-mono">{{ configText }}</pre>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('system.common.close') }}</UButton>
        <UButton color="primary" icon="i-lucide-copy" @click="copyConfig">{{ t('mcp.config.copy') }}</UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getMcpOverview } from '@/api/mcp'
import { copyAPIKey } from '@/api/node'
import { pageTabTo } from '@/config/pageGroups'
import { nodePermissionLabelKey } from '@/config/nodePermission'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import type { McpCredential, McpTool } from '@/types/v1/mcp'
import { APIKeyStatus, type APIKeyInfo } from '@/types/v1/node'
import { copyTextToClipboard } from '@/utils/file'
import ApiKeyCreate from '@/views/node/key/create.vue'
import { useNodeTime } from '@/views/node/components/useNodeTime'

defineOptions({ name: 'SystemMcpView' })

const { t, te } = useI18n()
const menuStore = useMenuStore()
const accessible = useAccessibleRoutes()
const { ago, dateTime } = useNodeTime()
const apiKeyCreateRef = useTemplateRef<InstanceType<typeof ApiKeyCreate> | null>('apiKeyCreateRef')

const TONES = ['p', 'a2', 'a', 'ok']

// —— 数据 ——
const loaded = ref(false)
const path = ref('')
const tools = ref<McpTool[]>([])
const credentials = ref<McpCredential[]>([])
const load = async () => {
  try {
    const { data } = await getMcpOverview()
    path.value = data.path || ''
    tools.value = data.tools || []
    credentials.value = data.credentials || []
  } catch {
    // 请求层已经提示过
  } finally {
    loaded.value = true
  }
}
onMounted(load)
// 菜单页是 keep-alive 的：切回来时刷新（在 API Key 页改了权限、Agent 用过凭证）
onActivated(() => {
  if (loaded.value) void load()
})

const endpoint = computed(() => (path.value ? `${window.location.origin}${path.value}` : ''))
const canManageKeys = computed(() => accessible.value.has('/node/key'))
const permissionLabel = (code: string) => {
  const key = nodePermissionLabelKey(code)
  return key ? t(key) : code
}

type PillVariant = 'success' | 'warning' | 'error'
const statusPill = (c: McpCredential): { variant: PillVariant; label: string } => {
  if (c.status === APIKeyStatus.APIKeyStatus_Expired) return { variant: 'error', label: t('access.key.expired') }
  if (c.status === APIKeyStatus.APIKeyStatus_Expiring) return { variant: 'warning', label: t('access.key.expiring') }
  return { variant: 'success', label: t('access.key.valid') }
}

const copyEndpoint = async () => {
  if (!endpoint.value) return
  await copyTextToClipboard(endpoint.value)
  feedback.success(t('mcp.connection.endpointCopied'))
}

// —— 新建凭证：API Key 的新建弹窗，预填名称、工具要的权限和一年有效期（弹窗里都能改，清空有效期即永久有效） ——
const YEAR_MS = 365 * 24 * 3600 * 1000
const createCredential = () => {
  const permissions = [...new Set(tools.value.map((tool) => tool.permission))]
  apiKeyCreateRef.value?.showDialog({
    name: 'Momoko MCP',
    permissions,
    expiresAt: new Date(Date.now() + YEAR_MS),
  } as unknown as APIKeyInfo)
}
const onKeyCreated = (type: 'create' | 'update', info?: APIKeyInfo) => {
  void load()
  if (type === 'create' && info?.apiKey) showConfig(info.name, info.apiKey)
}

// —— MCP 配置：mcpServers JSON，可粘贴到「Agent 管理 → MCP」或其它 MCP 客户端 ——
const configOpen = ref(false)
const configName = ref('')
const configKey = ref('')
const configText = computed(() =>
  JSON.stringify(
    {
      mcpServers: {
        momoko: { type: 'http', url: endpoint.value, headers: { Authorization: `Bearer ${configKey.value}` } },
      },
    },
    null,
    2,
  ),
)
const showConfig = (name: string, apiKey: string) => {
  configName.value = name
  configKey.value = apiKey
  configOpen.value = true
}
const openConfig = async (c: McpCredential) => {
  const { data } = await copyAPIKey({ id: c.id })
  if (data?.info?.apiKey) showConfig(c.name, data.info.apiKey)
}
const copyConfig = async () => {
  await copyTextToClipboard(configText.value)
  feedback.success(t('mcp.config.copied'))
}
</script>

<style scoped lang="scss">
.mcp-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

// —— 连接 ——
.mcp-conn {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.mcp-conn__row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 38px;
  padding: 2px 4px 2px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  font-size: 0.8125rem;
}
.mcp-conn__row dt {
  flex: none;
  width: 72px;
  color: var(--el-text-color-secondary);
  font-weight: 600;
}
.mcp-conn__row dd {
  flex: 1;
  min-width: 0;
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
}

// —— 列表（凭证、工具共用） ——
.mcp-list {
  display: flex;
  flex-direction: column;
}
.mcp-list > li + li {
  border-top: 1px solid var(--el-border-color-lighter);
}
.mcp-cred__label {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.mcp-cred__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

// —— 凭证 ——
.mcp-cred {
  display: grid;
  grid-template-columns: auto minmax(0, 1.2fr) minmax(0, 1.4fr) minmax(0, 0.8fr) auto;
  align-items: center;
  gap: 14px;
  padding: 12px 0;
}
.mcp-cred__av {
  width: 38px;
  height: 38px;
}
.mcp-cred__av :deep(svg) {
  width: 19px;
  height: 19px;
}
.mcp-cred__main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.mcp-cred__title {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.mcp-cred__name {
  min-width: 0;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mcp-cred__tools,
.mcp-cred__used {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}
.mcp-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.mcp-chip {
  padding: 2px 8px;
  border-radius: var(--app-radius-pill);
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-size: 0.75rem;
  font-weight: 600;
}
.mcp-cred__none {
  color: var(--el-text-color-secondary);
}
.mcp-cred__none.is-warn {
  color: var(--m-warn);
  font-weight: 600;
}

// —— 工具 ——
.mcp-tool {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
  align-items: center;
  gap: 14px;
  padding: 12px 0;
}
.mcp-tool__main {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.mcp-tool__name {
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.mcp-tool__desc {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
}
.mcp-tool__perm {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}

// —— MCP 配置 ——
.mcp-config {
  margin: 0;
  padding: 14px 16px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
  line-height: 1.55;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

@media (width < 768px) {
  .mcp-conn__row {
    flex-wrap: wrap;
    gap: 2px 12px;
    padding: 8px 4px 8px 12px;
  }
  .mcp-conn__row dt {
    width: auto;
  }
  .mcp-conn__row dd {
    flex-basis: 100%;
    order: 3;
  }
  .mcp-conn__row :deep(.app-icon-btn) {
    margin-left: auto;
  }
  .mcp-cred {
    grid-template-columns: auto minmax(0, 1fr) auto;
    grid-template-areas:
      'av main op'
      'tools tools tools'
      'used used used';
    gap: 10px 12px;
  }
  .mcp-cred__av {
    grid-area: av;
  }
  .mcp-cred__main {
    grid-area: main;
  }
  .mcp-cred__op {
    grid-area: op;
  }
  .mcp-cred__tools {
    grid-area: tools;
  }
  .mcp-cred__used {
    grid-area: used;
  }
  .mcp-tool {
    grid-template-columns: minmax(0, 1fr);
    gap: 8px;
  }
}
</style>
