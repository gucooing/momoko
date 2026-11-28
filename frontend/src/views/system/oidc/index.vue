<!-- 系统设置 · OIDC 身份提供方（第二期）：标题 + 服务状态 pill → SettingsSubNav → 页内服务端配置卡（configDialog.vue）
     → 客户端区：标题计数 + 新建 → 启用状态 chips(带数) + 搜索 → 新 Secret 提示条 → PC 表格（贴纸头像、Client ID、回调、Scopes、启用开关、创建/最近使用、行内小图标）/ 手机卡片。
     子弹窗：clientForm(创建编辑) / 客户端配置只读展示（完整 secret 仅一次）。行内 编辑/刷新密钥/删除（Dialog.info 确认）。 -->
<template>
  <div class="oidc-page">
    <PageHeader>
      <template #title>
        <span class="oidc-title">
          {{ t('system.oidcx.pageTitle') }}
          <StatusPill
            v-if="config"
            :variant="config.enabled ? 'success' : 'neutral'"
            :label="config.enabled ? t('system.oidcx.serving') : t('system.oidcx.stopped')"
          />
        </span>
      </template>
      <template #description>{{ t('system.oidcx.pageDesc') }}</template>
      <template #actions>
        <UButton v-if="!menuStore.isMobile" color="neutral" variant="outline" icon="i-lucide-book-open" @click="docsVisible = true">
          {{ t('systemV2.oidc.docs') }}
        </UButton>
        <AppIconButton v-else icon="HOutline:BookOpenIcon" :label="t('systemV2.oidc.docs')" :box="44" @click="docsVisible = true" />
      </template>
    </PageHeader>

    <SettingsSubNav />

    <OIDCConfigCard :can-edit="canEdit" @change="onConfigChange" />

    <div class="oidc-section">
      <div class="oidc-section__text">
        <h2>{{ t('system.oidcx.clients') }}</h2>
        <p>
          <span class="m-num">{{ t('system.oidcx.clientsCount', { n: counts.all }) }}</span>
          · <b class="oidc-tone-ok m-num">{{ t('system.oidcx.clientsActive', { n: counts.active }) }}</b>
        </p>
      </div>
      <UButton v-if="!menuStore.isMobile" v-permission="[PERM.OIDC_EDIT]" color="primary" icon="i-lucide-plus" @click="openClient()">
        {{ t('system.oidcx.newClient') }}
      </UButton>
    </div>

    <ListToolbar
      v-model:status="activeFilter"
      v-model:keyword="queryForm.keywords"
      :chips="filterChips"
      :chips-label="t('system.common.status')"
      :placeholder="t('system.oidcx.searchPlaceholder')"
      @search="reload"
    />

    <!-- 新 Secret 提示条 -->
    <div v-if="revealClient" class="oidc-secret" role="status">
      <span class="m-sticker m-sticker--solid-ink oidc-secret__icon">
        <component :is="menuStore.iconComponents['HOutline:KeyIcon']" />
      </span>
      <span class="oidc-secret__text">
        <b>{{ t('system.oidcx.secretIssued', { name: revealClient.name }) }}</b>
        <span>
          {{ t('system.oidcx.secretOnce') }}
          <code class="m-mono">{{ revealClient.clientSecret }}</code>
        </span>
      </span>
      <UButton color="neutral" variant="outline" size="sm" :icon="secretCopied ? 'i-lucide-check' : 'i-lucide-copy'" @click="copySecret">
        {{ secretCopied ? t('system.oidc.copied') : t('system.oidcx.copySecret') }}
      </UButton>
      <UButton color="neutral" variant="ghost" size="sm" @click="revealVisible = true">
        {{ t('system.oidcx.viewConfig') }}
      </UButton>
      <AppIconButton icon="HOutline:XMarkIcon" :label="t('common.close')" :box="36" @click="revealClient = null" />
    </div>

    <!-- 桌面：表格 -->
    <DataTable
      v-if="!menuStore.isMobile"
      :columns="columns"
      :rows="clients"
      row-key="id"
      :loading="loading"
      :empty-text="t('system.oidc.emptyTitle')"
    >
      <template #cell-name="{ row }">
        <span class="oidc-client" :class="{ 'is-off': !row.active }">
          <span class="m-sticker oidc-avatar" :class="`m-sticker--${tone(String(row.id))}`" :style="{ transform: `rotate(${rot(String(row.id))})` }">
            {{ letters(String(row.name)) }}
          </span>
          <span class="oidc-client__text">
            <b>{{ row.name }}</b>
            <span v-if="row.description">{{ row.description }}</span>
          </span>
        </span>
      </template>
      <template #cell-clientId="{ row }">
        <span class="oidc-chip is-cell m-mono" :title="String(row.clientId)">{{ row.clientId }}</span>
      </template>
      <template #cell-redirectUris="{ row }">
        <span class="oidc-uris">
          <span v-for="u in (row.redirectUris as string[])" :key="u" class="m-mono">{{ u }}</span>
        </span>
      </template>
      <template #cell-scopes="{ row }">
        <span class="oidc-scopes">
          <span v-for="sp in (row.scopes as string[])" :key="sp" class="oidc-scope m-mono">{{ sp }}</span>
        </span>
      </template>
      <template #cell-active="{ row }">
        <AppSwitch
          :model-value="Boolean(row.active)"
          :disabled="!canEdit || toggling === row.id"
          :aria-label="t('system.oidcx.toggleClient', { name: row.name })"
          @update:model-value="(v: boolean) => toggleActive(String(row.id), v)"
        />
      </template>
      <template #cell-createTime="{ row }">
        <span class="oidc-time">
          <b class="m-num">{{ fmtDate(row.createTime) }}</b>
          <span>{{ usedText(row.lastUsedTime) }}</span>
        </span>
      </template>
      <template #cell-operation="{ row }">
        <span v-if="canEdit" class="oidc-acts">
          <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('system.oidc.refreshSecret')" :box="36" @click="onRowAction('refresh', row)" />
          <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('system.common.edit')" :box="36" @click="onRowAction('edit', row)" />
          <AppIconButton icon="HOutline:TrashIcon" class="is-danger" :label="t('system.common.delete')" :box="36" @click="onRowAction('delete', row)" />
        </span>
        <span v-else class="text-dim">—</span>
      </template>
    </DataTable>

    <!-- 移动：卡片 -->
    <template v-else>
      <div v-if="loading && !clients.length" class="oidc-cards">
        <div v-for="i in 3" :key="i" class="oidc-skeleton" />
      </div>
      <EmptyState
        v-else-if="!clients.length"
        icon="HOutline:KeyIcon"
        :title="t('system.oidc.emptyTitle')"
        :description="t('system.oidcx.emptyDesc')"
      />
      <div v-else class="oidc-cards">
        <article v-for="row in clients" :key="row.id" class="oidc-card" :class="{ 'is-off': !row.active }">
          <header class="oidc-card__head">
            <span class="m-sticker oidc-avatar" :class="`m-sticker--${tone(row.id)}`" :style="{ transform: `rotate(${rot(row.id)})` }">
              {{ letters(row.name) }}
            </span>
            <span class="oidc-client__text">
              <b>{{ row.name }}</b>
              <span>
                <template v-if="row.description">{{ row.description }} · </template>
                <em :class="row.active ? 'is-on' : 'is-off'">{{ row.active ? t('system.oidcx.filterOn') : t('system.oidcx.filterOff') }}</em>
              </span>
            </span>
            <AppSwitch
              :model-value="row.active"
              :disabled="!canEdit || toggling === row.id"
              :aria-label="t('system.oidcx.toggleClient', { name: row.name })"
              @update:model-value="(v: boolean) => toggleActive(row.id, v)"
            />
          </header>
          <dl class="oidc-dl">
            <div class="is-full"><dt>Client ID</dt><dd class="m-mono">{{ row.clientId }}</dd></div>
            <div class="is-full">
              <dt>{{ t('system.oidcx.redirectCount', { n: row.redirectUris.length }) }}</dt>
              <dd v-for="u in row.redirectUris.slice(0, 3)" :key="u" class="m-mono oidc-dl__uri">{{ u }}</dd>
              <dd v-if="!row.redirectUris.length" class="m-mono">—</dd>
              <dd v-if="row.redirectUris.length > 3" class="oidc-dl__more">+{{ row.redirectUris.length - 3 }}</dd>
            </div>
            <div><dt>{{ t('system.oidcx.createdAt') }}</dt><dd class="m-num">{{ fmtDate(row.createTime) }}</dd></div>
            <div><dt>{{ t('system.oidcx.lastUsed') }}</dt><dd>{{ usedText(row.lastUsedTime) }}</dd></div>
          </dl>
          <footer class="oidc-card__foot">
            <span class="oidc-scopes">
              <span v-for="sp in row.scopes" :key="sp" class="oidc-scope m-mono">{{ sp }}</span>
            </span>
            <template v-if="canEdit">
              <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('system.oidc.refreshSecret')" :box="44" @click="refreshSecret(row)" />
              <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('system.common.edit')" :box="44" @click="openClient(row)" />
              <AppIconButton icon="HOutline:TrashIcon" class="is-danger" :label="t('system.common.delete')" :box="44" @click="confirmDelete(row)" />
            </template>
          </footer>
        </article>
      </div>
    </template>

    <Pagination
      v-if="pagination.total > pagination.pageSize || pagination.page > 1"
      v-model:page="pagination.page"
      v-model:page-size="pagination.pageSize"
      :total="pagination.total"
      @change="loadClients"
    />

    <p class="oidc-note">
      <component :is="menuStore.iconComponents['HOutline:InformationCircleIcon']" />
      {{ t('system.oidcx.callbackNote') }}
    </p>

    <!-- 手机：悬浮新建 -->
    <button v-if="menuStore.isMobile && canEdit" type="button" class="oidc-fab" @click="openClient()">
      <component :is="menuStore.iconComponents['HOutline:PlusIcon']" />
      {{ t('system.oidcx.newClient') }}
    </button>

    <OIDCClientForm ref="clientFormRef" @refresh="loadClients" @reveal="openReveal" />

    <!-- 接入文档 -->
    <FormDialog v-model="docsVisible" :title="t('systemV2.oidc.docsTitle')" :width="680">
      <div class="oidc-docs">
        <ol class="oidc-docs__steps">
          <li v-for="(step, idx) in docSteps" :key="idx">
            <span class="oidc-docs__no m-num">{{ idx + 1 }}</span>
            <span>{{ step }}</span>
          </li>
        </ol>
        <section class="oidc-docs__sec">
          <h3 class="oidc-docs__label">{{ t('systemV2.oidc.docsParams') }}</h3>
          <div v-for="item in docItems" :key="item.label" class="oidc-docs__item">
            <span class="oidc-docs__text">
              <span class="oidc-docs__k">{{ item.label }}</span>
              <span class="oidc-docs__v m-mono">{{ item.value || '—' }}</span>
            </span>
            <AppIconButton
              v-if="item.value"
              icon="HOutline:ClipboardDocumentIcon"
              :label="t('system.oidcx.copyEndpoint', { name: item.label })"
              :box="40"
              @click="copyText(item.value)"
            />
          </div>
        </section>
        <section class="oidc-docs__sec">
          <div class="oidc-docs__head">
            <h3 class="oidc-docs__label">{{ t('systemV2.oidc.docsExample') }}</h3>
            <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('common.copy')" :box="40" @click="copyText(docExample)" />
          </div>
          <pre class="oidc-docs__code m-mono">{{ docExample }}</pre>
        </section>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('common.close') }}</UButton>
      </template>
    </FormDialog>

    <!-- 客户端配置（完整 secret 仅一次） -->
    <FormDialog v-model="revealVisible" :title="t('system.oidc.revealTitle')" :width="720">
      <div class="oidc-reveal">
        <div class="oidc-reveal__warn">
          {{ t('system.oidc.secretOnceWarn') }}
        </div>
        <div class="oidc-reveal__grid">
          <div v-for="item in clientConfigItems" :key="item.label" class="app-field">
            <label class="app-label">{{ item.label }}</label>
            <div class="oidc-config__inline">
              <textarea v-if="item.multiline" class="app-textarea" :value="item.value" readonly rows="2" />
              <input v-else class="app-input" :value="item.value" readonly />
              <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('common.copy')" :box="36" @click="copyText(item.value)" />
            </div>
          </div>
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('common.close') }}</UButton>
        <UButton color="primary" icon="i-lucide-copy" @click="copyText(clientConfigText)">
          {{ t('system.oidc.copyAll') }}
        </UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'
import 'dayjs/locale/zh-hk'
import { listOIDCClients, deleteOIDCClient, refreshOIDCClientSecret, updateOIDCClient } from '@/api/oidc'
import { PERM } from '@/config/permission'
import { Dialog } from '@/utils/dialog'
import type { OIDCClientInfo, OIDCConfig } from '@/types/v1/oidc'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import OIDCConfigCard from '@/views/system/oidc/configDialog.vue'
import OIDCClientForm from '@/views/system/oidc/clientForm.vue'
import SettingsSubNav from '@/views/system/components/SettingsSubNav.vue'

dayjs.extend(relativeTime)

defineOptions({ name: 'SystemOIDCView' })

type ActiveKey = '' | 'on' | 'off'

const menuStore = useMenuStore()
const { t, locale } = useI18n()

const clientFormRef = useTemplateRef<InstanceType<typeof OIDCClientForm> | null>('clientFormRef')

const canEdit = computed(() => menuStore.hasButtonPermission(PERM.OIDC_EDIT))

const loading = ref(false)
const clients = ref<OIDCClientInfo[]>([])
const config = ref<OIDCConfig | null>(null)
const queryForm = ref({ keywords: '' })
const activeFilter = ref<ActiveKey>('')
const counts = ref({ all: 0, active: 0 })
const pagination = ref({ page: 1, pageSize: 20, total: 0 })
const toggling = ref('')

const filterChips = computed<ToolbarChip<ActiveKey>[]>(() => [
  { label: t('system.common.all'), value: '', count: counts.value.all },
  { label: t('system.oidcx.filterOn'), value: 'on', count: counts.value.active },
  { label: t('system.oidcx.filterOff'), value: 'off', count: counts.value.all - counts.value.active },
])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('system.oidcx.colClient'), minWidth: 180, ellipsis: false },
  { key: 'clientId', title: t('system.oidc.colClientId'), minWidth: 150 },
  { key: 'redirectUris', title: t('system.oidc.colRedirectUris'), minWidth: 210, ellipsis: false },
  { key: 'scopes', title: t('system.oidc.colScopes'), minWidth: 150, ellipsis: false },
  { key: 'active', title: t('system.common.enabled'), width: 80 },
  { key: 'createTime', title: t('system.oidcx.createdAt'), width: 140, ellipsis: false },
  { key: 'operation', title: t('system.common.operation'), width: 136, align: 'right' },
])

// —— 贴纸头像 ——
const TONES = ['a', 'ok', 'a2', 'p', 'warn']
const ROTS = ['-4deg', '3deg', '-2deg', '4deg', '-3deg']
const hash = (s: string) => [...s].reduce((a, c) => a + c.charCodeAt(0), 0)
const tone = (id: string) => TONES[hash(id) % TONES.length]
const rot = (id: string) => ROTS[hash(id) % ROTS.length]
const letters = (name: string) => {
  const chars = [...name.trim()]
  if (!chars.length) return '?'
  return /^[a-z]/i.test(chars[0] ?? '') ? (chars[0] ?? '').toUpperCase() + (chars[1] ?? '').toLowerCase() : chars[0]
}

const fmtDate = (v: unknown) => (v ? dayjs(v as string | Date).format('YYYY-MM-DD') : '—')
const dayjsLocale = computed(() => (locale.value === 'en-US' ? 'en' : locale.value === 'zh-HK' ? 'zh-hk' : 'zh-cn'))
const usedText = (v: unknown) => {
  if (!v) return t('system.oidcx.neverUsed')
  return t('system.oidcx.usedAgo', { ago: dayjs(v as string | Date).locale(dayjsLocale.value).fromNow() })
}

const loadClients = async () => {
  loading.value = true
  try {
    const { data } = await listOIDCClients({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.value.keywords || undefined,
      active: activeFilter.value === '' ? undefined : activeFilter.value === 'on',
    })
    clients.value = data?.clients || []
    pagination.value.total = Number(data?.total || 0)
    counts.value = { all: Number(data?.allCount) || 0, active: Number(data?.activeCount) || 0 }
  } finally {
    loading.value = false
  }
}

const reload = () => {
  pagination.value.page = 1
  loadClients()
}

const onConfigChange = (value: OIDCConfig) => {
  config.value = value
}

const openClient = (row?: OIDCClientInfo) => clientFormRef.value?.showDialog(row)

const onRowAction = (key: string, row: Record<string, unknown>) => {
  const client = clients.value.find((c) => c.id === String(row.id))
  if (!client) return
  if (key === 'edit') openClient(client)
  else if (key === 'refresh') refreshSecret(client)
  else if (key === 'delete') confirmDelete(client)
}

const toggleActive = async (id: string, active: boolean) => {
  const client = clients.value.find((c) => c.id === id)
  if (!client) return
  toggling.value = id
  try {
    await updateOIDCClient({
      id: client.id,
      name: client.name,
      redirectUris: client.redirectUris,
      scopes: client.scopes,
      active,
      description: client.description,
    })
    client.active = active
    counts.value.active += active ? 1 : -1
    feedback.success(t('system.oidc.editSuccess'))
    if (activeFilter.value) loadClients()
  } finally {
    toggling.value = ''
  }
}

const refreshSecret = (client: OIDCClientInfo) => {
  Dialog.info({
    showCancelButton: true,
    content: t('system.oidc.refreshSecretConfirm'),
    confirmText: t('system.oidc.refresh'),
    cancelText: t('common.cancel'),
    onConfirm: async () => {
      const { data } = await refreshOIDCClientSecret({ id: client.id })
      if (data?.client) openReveal(data.client, false)
      loadClients()
    },
  })
}

const confirmDelete = (client: OIDCClientInfo) => {
  Dialog.info({
    showCancelButton: true,
    content: t('system.oidc.deleteConfirm', { name: client.name }),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    onConfirm: async () => {
      await deleteOIDCClient({ id: client.id })
      feedback.success(t('common.deleteSuccess'))
      if (revealClient.value?.id === client.id) revealClient.value = null
      loadClients()
    },
  })
}

// —— 新 Secret：提示条 + 完整配置弹窗 ——
const revealVisible = ref(false)
const revealClient = ref<OIDCClientInfo | null>(null)
const secretCopied = ref(false)

const clientConfigItems = computed<{ label: string; value: string; multiline?: boolean }[]>(() => {
  const client = revealClient.value
  if (!client) return []
  const e = config.value?.endpoints
  return [
    { label: t('system.oidc.colName'), value: client.name },
    { label: t('system.oidc.colClientId'), value: client.clientId },
    { label: t('system.oidc.colClientSecret'), value: client.clientSecret },
    { label: t('system.oidc.colRedirectUris'), value: client.redirectUris.join('\n'), multiline: true },
    { label: t('system.oidc.colScopes'), value: client.scopes.join(' ') },
    { label: t('system.oidc.issuerUrl'), value: e?.issuerUrl || '' },
    { label: t('system.oidc.discoveryUrl'), value: e?.discoveryUrl || '' },
    { label: t('system.oidc.authorizeUrl'), value: e?.authorizeUrl || '' },
    { label: t('system.oidc.tokenUrl'), value: e?.tokenUrl || '' },
    { label: t('system.oidc.userinfoUrl'), value: e?.userinfoUrl || '' },
    { label: t('system.oidc.jwksUrl'), value: e?.jwksUrl || '' },
    { label: t('system.oidc.tokenAuthMethod'), value: e?.tokenEndpointAuthMethod || 'client_secret_post' },
    { label: t('system.oidc.signingAlgs'), value: e?.allowedIdTokenSigningAlgs || 'RS256' },
  ].filter((item) => item.value)
})
const clientConfigText = computed(() =>
  clientConfigItems.value.map((item) => `${item.label}: ${item.value}`).join('\n'),
)

// —— 接入文档：步骤 + 关键参数 + 通用 OIDC 客户端配置示例（以 Grafana generic_oauth 为例）——
const docsVisible = ref(false)
const docSteps = computed(() => [
  t('systemV2.oidc.step1'),
  t('systemV2.oidc.step2'),
  t('systemV2.oidc.step3'),
  t('systemV2.oidc.step4'),
])
const docItems = computed(() => {
  const e = config.value?.endpoints
  return [
    { label: t('system.oidc.issuerUrl'), value: e?.issuerUrl || config.value?.issuerUrl || '' },
    { label: 'Discovery', value: e?.discoveryUrl || '' },
    { label: t('system.oidcx.epAuthorize'), value: e?.authorizeUrl || '' },
    { label: t('system.oidcx.epToken'), value: e?.tokenUrl || '' },
    { label: t('system.oidcx.epUserinfo'), value: e?.userinfoUrl || '' },
    { label: 'JWKS', value: e?.jwksUrl || '' },
    { label: t('system.oidc.tokenAuthMethod'), value: e?.tokenEndpointAuthMethod || 'client_secret_post' },
    { label: 'Scopes', value: 'openid profile email' },
  ]
})
const docExample = computed(() => {
  const e = config.value?.endpoints
  return [
    '[auth.generic_oauth]',
    'enabled = true',
    'name = Momoko',
    'client_id = <Client ID>',
    'client_secret = <Client Secret>',
    'scopes = openid profile email',
    `auth_url = ${e?.authorizeUrl || '<authorize>'}`,
    `token_url = ${e?.tokenUrl || '<token>'}`,
    `api_url = ${e?.userinfoUrl || '<userinfo>'}`,
  ].join('\n')
})

// 新建：直接弹完整配置；刷新：顶部提示条（可再展开完整配置）
const openReveal = (client: OIDCClientInfo, showDialog = true) => {
  revealClient.value = client
  secretCopied.value = false
  revealVisible.value = showDialog
}

const copyText = async (value: string) => {
  await navigator.clipboard.writeText(value)
  feedback.success(t('system.oidc.copied'))
}
const copySecret = async () => {
  if (!revealClient.value) return
  await copyText(revealClient.value.clientSecret)
  secretCopied.value = true
}

onMounted(loadClients)
</script>

<style scoped lang="scss">
.oidc-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.oidc-title {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}
.oidc-title :deep(.status-pill) {
  font-family: inherit;
  font-size: 0.75rem;
  letter-spacing: 0;
}
.oidc-section {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 12px 16px;
  margin-top: 8px;
}
.oidc-section > :deep(button) {
  min-height: 40px;
}
.oidc-section__text {
  flex: 1 1 260px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.oidc-section__text h2 {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.oidc-section__text p {
  margin: 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.oidc-tone-ok {
  color: var(--m-ok);
  font-weight: 700;
}
.text-dim {
  color: var(--el-text-color-placeholder);
}

/* Secret 提示条 */
.oidc-secret {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 14px;
  padding: 14px 14px 14px 18px;
  border-radius: var(--app-radius-lg);
  background: var(--m-warnSoft);
}
.oidc-secret__icon {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 13px;
  transform: rotate(-5deg);
}
.oidc-secret__icon :deep(svg) {
  width: 20px;
  height: 20px;
}
.oidc-secret__text {
  flex: 1 1 320px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}
.oidc-secret__text b {
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--m-warn);
}
.oidc-secret__text code {
  display: inline-block;
  margin-top: 2px;
  padding: 2px 8px;
  border-radius: var(--app-radius-xs);
  background: var(--el-bg-color);
  font-weight: 700;
  word-break: break-all;
}

/* 表格单元 */
.oidc-client {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.oidc-client.is-off,
.oidc-card.is-off .oidc-client__text {
  opacity: 0.6;
}
.oidc-avatar {
  flex: none;
  width: 42px;
  height: 42px;
  border-radius: 14px;
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
}
.oidc-client__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.oidc-client__text b {
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.oidc-client__text span {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.oidc-chip {
  display: inline-block;
  max-width: 100%;
  padding: 3px 8px;
  border-radius: var(--app-radius-xs);
  background: var(--el-fill-color-light);
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: middle;
}
/* 表格里 Client ID 很长：限宽省略，悬停看全（编辑弹窗里可完整查看） */
.oidc-chip.is-cell {
  max-width: 200px;
}
.oidc-uris {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 340px;
}
.oidc-uris span {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.oidc-scopes {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.oidc-scope {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 7px;
  border-radius: 7px;
  background: var(--m-a2Soft);
  color: var(--m-ink);
  font-size: 0.6875rem;
  font-weight: 600;
}
.oidc-time {
  display: flex;
  flex-direction: column;
  gap: 2px;
  white-space: nowrap;
}
.oidc-time b {
  font-size: 0.8125rem;
  font-weight: 700;
}
.oidc-time span {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.oidc-acts {
  display: inline-flex;
  justify-content: flex-end;
  gap: 2px;
}
.oidc-acts :deep(.is-danger),
.oidc-card__foot :deep(.is-danger) {
  color: var(--m-bad);
}
.oidc-acts :deep(.is-danger:hover),
.oidc-card__foot :deep(.is-danger:hover) {
  background: var(--m-badSoft);
}
/* 手机悬浮新建 */
.oidc-fab {
  position: sticky;
  bottom: 0;
  z-index: 6;
  align-self: flex-end;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 56px;
  padding: 0 22px;
  border: none;
  border-radius: var(--app-radius-pill);
  background: var(--m-p);
  color: var(--m-pOn);
  font: inherit;
  font-size: 0.9375rem;
  font-weight: 700;
  box-shadow: var(--app-shadow-brand);
  cursor: pointer;
}
.oidc-fab :deep(svg) {
  width: 20px;
  height: 20px;
}
.oidc-fab:focus-visible {
  outline: 3px solid var(--m-a2);
  outline-offset: 2px;
}
.oidc-client__text em {
  font-style: normal;
  font-weight: 700;
}
.oidc-client__text em.is-on {
  color: var(--m-ok);
}
.oidc-dl dd.oidc-dl__uri {
  font-size: 0.6875rem;
  font-weight: 400;
}
.oidc-dl dd.oidc-dl__more {
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
}

/* 接入文档 */
.oidc-docs {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.oidc-docs__steps {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.oidc-docs__steps li {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  font-size: 0.875rem;
  line-height: 1.55;
  color: var(--el-text-color-primary);
}
.oidc-docs__no {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-size: 0.75rem;
  font-weight: 800;
}
.oidc-docs__sec {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.oidc-docs__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.oidc-docs__label {
  margin: 0;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.oidc-docs__item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 4px 6px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.oidc-docs__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.oidc-docs__k {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.oidc-docs__v {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.oidc-docs__code {
  margin: 0;
  padding: 14px 16px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
  font-size: 0.75rem;
  line-height: 1.65;
  color: var(--el-text-color-primary);
  overflow-x: auto;
  scrollbar-width: none;
  white-space: pre;
}
.oidc-docs__code::-webkit-scrollbar {
  display: none;
}

.oidc-note {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.oidc-note :deep(svg) {
  flex: none;
  width: 16px;
  height: 16px;
}

/* 客户端配置只读展示 */
.oidc-reveal {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.oidc-reveal__warn {
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-warnSoft);
  color: var(--m-warn);
  font-size: 0.8125rem;
  font-weight: 600;
}
.oidc-reveal__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.oidc-config__inline {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.oidc-config__inline .app-input,
.oidc-config__inline .app-textarea {
  flex: 1;
  min-width: 0;
}

/* 移动卡片 */
.oidc-cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.oidc-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: calc(var(--cmp-card-unit) * 16) calc(var(--cmp-card-unit) * 12) calc(var(--cmp-card-unit) * 6) calc(var(--cmp-card-unit) * 16);
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.oidc-card__head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-right: 4px;
}
.oidc-dl {
  margin: 0 4px 0 0;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 14px;
}
.oidc-dl > div {
  min-width: 0;
}
.oidc-dl .is-full {
  grid-column: 1 / -1;
}
.oidc-dl dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.oidc-dl dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.oidc-card__foot {
  display: flex;
  align-items: center;
  gap: 2px;
  padding-top: 6px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.oidc-card__foot .oidc-scopes {
  flex: 1;
  min-width: 0;
}
.oidc-skeleton {
  height: 180px;
  border-radius: var(--app-radius-lg);
  background: linear-gradient(100deg, var(--el-fill-color-light) 30%, var(--el-fill-color) 50%, var(--el-fill-color-light) 70%);
  background-size: 200% 100%;
  animation: oidc-shimmer 1.4s ease-in-out infinite;
}
@keyframes oidc-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
@media (width <= 768px) {
  .oidc-page {
    /* 悬浮新建按钮贴在底部导航上方 88px 处；留出同样高度，滚到底时按钮不压住最后一张卡片 */
    padding-bottom: 88px;
  }
  .oidc-reveal__grid {
    grid-template-columns: 1fr;
  }
  .oidc-secret__text {
    flex-basis: calc(100% - 60px);
  }
  .oidc-secret__text code {
    display: block;
    padding: 8px 10px;
  }
  .oidc-note {
    align-items: flex-start;
  }
}
</style>
