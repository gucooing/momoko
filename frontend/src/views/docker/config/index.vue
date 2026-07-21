<!-- Docker 配置（重写 · P3 配置型）：PageHeader + 令牌 Tab 条 + AppPanel 分组。
     四个分区：连接状态 / 连接配置 / 默认设置 / 仓库认证。控件用 AppSwitch + .app-input。
     保留全部 API（status/config/test/update）、PERM.DOCKER_CONFIG_EDIT、i18n。 -->
<template>
  <div class="dk-cfg">
    <PageHeader :title="t('docker.config.pageTitle')" :description="t('docker.config.pageDesc')" />

    <div class="settings-tabs" role="tablist">
      <button
        v-for="tab in TABS"
        :key="tab.name"
        type="button"
        role="tab"
        class="settings-tabs__btn"
        :class="{ 'is-active': activeTab === tab.name }"
        :aria-selected="activeTab === tab.name"
        @click="setTab(tab.name)"
      >
        <component :is="menuStore.iconComponents[tab.icon]" />
        {{ t(tab.labelKey) }}
      </button>
    </div>

    <!-- 连接状态 -->
    <div v-show="activeTab === 'status'" class="settings-tab">
      <AppPanel :title="t('docker.config.engineStatus')" :padded="false">
        <div v-if="statusLoading" class="dk-cfg__loading">{{ t('docker.common.refresh') }}…</div>
        <div v-else class="set-rows">
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('docker.config.connectionStatus') }}</span>
            </div>
            <StatusPill
              v-if="statusInfo?.connected"
              variant="success"
              :label="t('docker.common.connected')"
            />
            <StatusPill
              v-else-if="statusInfo?.enabled"
              variant="error"
              :label="t('docker.common.notConnected')"
            />
            <StatusPill
              v-else
              variant="neutral"
              :label="t('docker.common.notEnabled')"
            />
          </div>
          <template v-if="statusInfo?.connected && statusInfo.info">
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('docker.common.name') }}</span></div>
              <span class="set-value">{{ statusInfo.info.name }}</span>
            </div>
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('docker.common.version') }}</span></div>
              <span class="set-value">{{ statusInfo.info.serverVersion }}</span>
            </div>
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('docker.common.os') }}</span></div>
              <span class="set-value">{{ statusInfo.info.operatingSystem }} ({{ statusInfo.info.osType }})</span>
            </div>
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('docker.common.architecture') }}</span></div>
              <span class="set-value">{{ statusInfo.info.architecture }}</span>
            </div>
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('docker.common.containers') }}</span></div>
              <span class="set-value">{{ t('docker.config.containerSummary', {
                total: statusInfo.info.containers,
                running: statusInfo.info.containersRunning,
                paused: statusInfo.info.containersPaused,
                stopped: statusInfo.info.containersStopped,
              }) }}</span>
            </div>
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('docker.common.images') }}</span></div>
              <span class="set-value">{{ statusInfo.info.images }}</span>
            </div>
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('docker.common.driver') }}</span></div>
              <span class="set-value">{{ statusInfo.info.driver }}</span>
            </div>
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('docker.common.cpu') }} / {{ t('docker.common.memory') }}</span></div>
              <span class="set-value">{{ t('docker.config.cpuMemory', { cpus: statusInfo.info.cpus, memory: formatBytes(statusInfo.info.memoryTotal) }) }}</span>
            </div>
          </template>
          <div v-if="statusInfo?.error" class="set-row">
            <div class="set-row__info"><span class="set-row__label">{{ t('docker.config.error') }}</span></div>
            <span class="set-value set-value--err">{{ statusInfo.error }}</span>
          </div>
        </div>
        <template #footer>
          <UButton color="neutral" variant="soft" icon="i-lucide-refresh-cw" :loading="statusLoading" @click="loadStatus">
            {{ t('docker.common.refresh') }}
          </UButton>
        </template>
      </AppPanel>
    </div>

    <!-- 连接配置 -->
    <div v-show="activeTab === 'connection'" class="settings-tab">
      <AppPanel :title="t('docker.config.connectionParams')" :padded="false">
        <div class="set-rows">
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('docker.config.enableDocker') }}</span>
              <span class="set-row__desc">{{ t('docker.config.enableDockerDesc') }}</span>
            </div>
            <AppSwitch v-model="configForm.enabled" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('docker.config.dockerHost') }}</span>
              <span class="set-row__desc">{{ t('docker.config.dockerHostDesc') }}</span>
            </div>
            <input v-model="configForm.host" class="app-input set-input" :disabled="!canEdit" placeholder="unix:///var/run/docker.sock" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('docker.config.apiVersion') }}</span>
              <span class="set-row__desc">{{ t('docker.config.apiVersionDesc') }}</span>
            </div>
            <input v-model="configForm.apiVersion" class="app-input set-num" :disabled="!canEdit" placeholder="auto" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('docker.config.requestTimeoutSeconds') }}</span>
            </div>
            <input v-model.number="configForm.requestTimeoutSeconds" type="number" min="1" max="300" class="app-input set-num" :disabled="!canEdit" />
          </div>
        </div>
      </AppPanel>

      <AppPanel :title="t('docker.config.tls')" :padded="false">
        <div class="set-rows">
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('docker.config.enableTls') }}</span>
            </div>
            <AppSwitch v-model="configForm.tlsEnabled" :disabled="!canEdit" />
          </div>
          <template v-if="configForm.tlsEnabled">
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('docker.config.tlsCaPath') }}</span></div>
              <input v-model="configForm.tlsCaPath" class="app-input set-input" :disabled="!canEdit" />
            </div>
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('docker.config.tlsCertPath') }}</span></div>
              <input v-model="configForm.tlsCertPath" class="app-input set-input" :disabled="!canEdit" />
            </div>
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('docker.config.tlsKeyPath') }}</span></div>
              <input v-model="configForm.tlsKeyPath" class="app-input set-input" :disabled="!canEdit" />
            </div>
          </template>
        </div>
        <template #footer>
          <UButton color="primary" :loading="configSaving" :disabled="!canEdit" @click="handleSaveConfig">
            {{ t('docker.config.saveConfig') }}
          </UButton>
          <UButton color="neutral" variant="soft" :loading="configTesting" :disabled="!canEdit" @click="handleTestConfig">
            {{ t('docker.config.testConnection') }}
          </UButton>
        </template>
      </AppPanel>
    </div>

    <!-- 默认设置 -->
    <div v-show="activeTab === 'defaults'" class="settings-tab">
      <AppPanel :title="t('docker.config.defaultParams')" :padded="false">
        <div class="set-rows">
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('docker.config.defaultPlatform') }}</span>
            </div>
            <input v-model="configForm.defaultPlatform" class="app-input set-input" :disabled="!canEdit" placeholder="linux/amd64" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('docker.config.taskTimeoutSeconds') }}</span>
            </div>
            <input v-model.number="configForm.taskTimeoutSeconds" type="number" min="60" max="86400" class="app-input set-num" :disabled="!canEdit" />
          </div>
        </div>
        <template #footer>
          <UButton color="primary" :loading="configSaving" :disabled="!canEdit" @click="handleSaveConfig">
            {{ t('docker.config.saveSettings') }}
          </UButton>
        </template>
      </AppPanel>
    </div>

    <!-- 仓库认证 -->
    <div v-show="activeTab === 'registries'" class="settings-tab">
      <AppPanel :title="t('docker.config.authList')" :padded="false">
        <div v-if="!configForm.registryAuths.length" class="dk-cfg__empty">{{ t('docker.config.emptyRegistryAuths') }}</div>
        <div v-else class="set-rows">
          <div v-for="(auth, idx) in configForm.registryAuths" :key="idx" class="set-row set-row--col">
            <div class="registry-fields">
              <input v-model="auth.serverAddress" class="app-input" :disabled="!canEdit" :placeholder="t('docker.config.registryAddress')" />
              <input v-model="auth.username" class="app-input" :disabled="!canEdit" :placeholder="t('docker.config.username')" />
              <input v-model="auth.password" class="app-input" type="password" :disabled="!canEdit" :placeholder="t('docker.config.passwordToken')" />
              <AppIconButton
                icon="HOutline:TrashIcon"
                :label="t('docker.common.delete')"
                :box="30"
                :disabled="!canEdit"
                @click="removeRegistryAuth(idx)"
              />
            </div>
          </div>
        </div>
        <template #footer>
          <UButton color="neutral" variant="soft" icon="i-lucide-plus" :disabled="!canEdit" @click="addRegistryAuth">
            {{ t('docker.config.addAuth') }}
          </UButton>
          <UButton color="primary" :loading="configSaving" :disabled="!canEdit" @click="handleSaveConfig">
            {{ t('docker.common.save') }}
          </UButton>
        </template>
      </AppPanel>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getDockerConfig, getDockerStatus, testDockerConfig, updateDockerConfig } from '@/api/docker'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { showRequestError } from '@/utils/request'
import type { DockerConfigInfo, DockerStatusResponse } from '@/types/v1/docker'

defineOptions({ name: 'DockerConfigView' })

type TabName = 'status' | 'connection' | 'defaults' | 'registries'
const TABS: { name: TabName; labelKey: string; icon: string }[] = [
  { name: 'status', labelKey: 'docker.config.statusTab', icon: 'HOutline:ServerStackIcon' },
  { name: 'connection', labelKey: 'docker.config.connectionTab', icon: 'HOutline:LinkIcon' },
  { name: 'defaults', labelKey: 'docker.config.defaultsTab', icon: 'HOutline:Cog6ToothIcon' },
  { name: 'registries', labelKey: 'docker.config.registriesTab', icon: 'HOutline:KeyIcon' },
]

const menuStore = useMenuStore()
const { t } = useI18n()
const canEdit = useButtonPermission([PERM.DOCKER_CONFIG_EDIT], [])

const activeTab = ref<TabName>('status')
const statusLoading = ref(false)
const configLoading = ref(false)
const configSaving = ref(false)
const configTesting = ref(false)
const statusInfo = ref<DockerStatusResponse | null>(null)

const defaultConfig = (): DockerConfigInfo => ({
  enabled: false,
  host: 'unix:///var/run/docker.sock',
  tlsEnabled: false,
  tlsCaPath: '',
  tlsCertPath: '',
  tlsKeyPath: '',
  apiVersion: '',
  requestTimeoutSeconds: 30,
  defaultPlatform: 'linux/amd64',
  taskTimeoutSeconds: 3600,
  registryAuths: [],
})

const configForm = reactive<DockerConfigInfo>(defaultConfig())

const formatBytes = (bytes: number | string) => {
  const n = Number(bytes)
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

const loadStatus = async () => {
  statusLoading.value = true
  try {
    const { data } = await getDockerStatus()
    statusInfo.value = data || null
  } catch (e) {
    showRequestError(e, t('docker.config.loadStatusFailed'))
  } finally {
    statusLoading.value = false
  }
}

const loadConfig = async () => {
  configLoading.value = true
  try {
    const { data } = await getDockerConfig()
    if (data?.config) Object.assign(configForm, data.config)
  } catch (e) {
    showRequestError(e, t('docker.config.loadConfigFailed'))
  } finally {
    configLoading.value = false
  }
}

const handleSaveConfig = async () => {
  configSaving.value = true
  try {
    await updateDockerConfig({ config: { ...configForm } })
    feedback.success(t('docker.config.saveSuccess'))
  } catch (e) {
    showRequestError(e, t('docker.config.saveFailed'))
  } finally {
    configSaving.value = false
  }
}

const handleTestConfig = async () => {
  configTesting.value = true
  try {
    const { data } = await testDockerConfig({ config: { ...configForm } })
    if (data?.status?.connected) feedback.success(t('docker.config.testSuccess'))
    else feedback.error(data?.status?.error || t('docker.common.connectionFailed'))
  } catch (e) {
    showRequestError(e, t('docker.config.testFailed'))
  } finally {
    configTesting.value = false
  }
}

const addRegistryAuth = () => {
  configForm.registryAuths.push({ serverAddress: '', username: '', password: '', token: '' })
}
const removeRegistryAuth = (idx: number) => {
  configForm.registryAuths.splice(idx, 1)
}

const loadedTabs = ref(new Set<string>())
const onTabChange = (name: string) => {
  if (loadedTabs.value.has(name)) return
  loadedTabs.value.add(name)
  if (name === 'status') loadStatus()
  else loadConfig()
}
const setTab = (name: TabName) => {
  activeTab.value = name
  onTabChange(name)
}

onMounted(() => {
  onTabChange(activeTab.value)
})
</script>

<style scoped lang="scss">
.dk-cfg {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.dk-cfg__loading,
.dk-cfg__empty {
  padding: 28px 20px;
  text-align: center;
  color: var(--el-text-color-placeholder);
  font-size: 0.8125rem;
}

.settings-tabs {
  display: inline-flex;
  align-self: flex-start;
  padding: 3px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
  flex-wrap: wrap;
}
.settings-tabs__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.settings-tabs__btn :deep(svg) {
  width: 16px;
  height: 16px;
}
.settings-tabs__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}

.settings-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.set-rows {
  display: flex;
  flex-direction: column;
}
.set-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 11px 20px;
}
.set-row + .set-row {
  border-top: 1px solid var(--el-border-color-lighter);
}
.set-row__info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}
.set-row__label {
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}
.set-row__desc {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.set-row--col {
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
}
.set-value {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  text-align: right;
  word-break: break-all;
}
.set-value--err {
  color: var(--el-color-danger);
}
.set-input {
  width: 280px;
  max-width: 100%;
  flex-shrink: 0;
}
.set-num {
  width: 140px;
  flex-shrink: 0;
}
.registry-fields {
  display: grid;
  grid-template-columns: 1.4fr 1fr 1.2fr 30px;
  gap: 8px;
  align-items: center;
  width: 100%;
}

@media (width <= 768px) {
  .set-row {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
  .set-input,
  .set-num {
    width: 100%;
  }
  .set-value {
    text-align: left;
  }
  .registry-fields {
    grid-template-columns: 1fr;
  }
}
</style>
