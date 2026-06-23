<template>
  <div>
    <el-card shadow="never">
      <el-tabs v-model="activeTab" type="border-card" @tab-change="onTabChange">
        <el-tab-pane label="连接状态" name="status">
          <div class="setting-module" v-loading="statusLoading">
            <div class="setting-group">
              <div class="setting-group-header">Docker Engine 状态</div>
              <div class="setting-item">
                <div class="setting-item-info"><span class="setting-item-label">连接状态</span></div>
                <div class="status-value">
                  <BaseTag v-if="statusInfo?.connected" text="已连接" type="success" />
                  <BaseTag v-else-if="statusInfo?.enabled" text="未连接" type="danger" />
                  <BaseTag v-else text="未启用" type="info" />
                </div>
              </div>
              <template v-if="statusInfo?.connected && statusInfo.info">
                <div class="setting-item"><div class="setting-item-info"><span class="setting-item-label">名称</span></div><span class="setting-item-value">{{ statusInfo.info.name }}</span></div>
                <div class="setting-item"><div class="setting-item-info"><span class="setting-item-label">版本</span></div><span class="setting-item-value">{{ statusInfo.info.serverVersion }}</span></div>
                <div class="setting-item"><div class="setting-item-info"><span class="setting-item-label">操作系统</span></div><span class="setting-item-value">{{ statusInfo.info.operatingSystem }} ({{ statusInfo.info.osType }})</span></div>
                <div class="setting-item"><div class="setting-item-info"><span class="setting-item-label">架构</span></div><span class="setting-item-value">{{ statusInfo.info.architecture }}</span></div>
                <div class="setting-item"><div class="setting-item-info"><span class="setting-item-label">容器</span></div><span class="setting-item-value">总计 {{ statusInfo.info.containers }} / 运行 {{ statusInfo.info.containersRunning }} / 暂停 {{ statusInfo.info.containersPaused }} / 停止 {{ statusInfo.info.containersStopped }}</span></div>
                <div class="setting-item"><div class="setting-item-info"><span class="setting-item-label">镜像数</span></div><span class="setting-item-value">{{ statusInfo.info.images }}</span></div>
                <div class="setting-item"><div class="setting-item-info"><span class="setting-item-label">存储驱动</span></div><span class="setting-item-value">{{ statusInfo.info.driver }}</span></div>
                <div class="setting-item"><div class="setting-item-info"><span class="setting-item-label">CPU / 内存</span></div><span class="setting-item-value">{{ statusInfo.info.cpus }} 核 / {{ formatBytes(statusInfo.info.memoryTotal) }}</span></div>
              </template>
              <div v-if="statusInfo?.error" class="setting-item"><div class="setting-item-info"><span class="setting-item-label">错误</span></div><span class="status-error">{{ statusInfo.error }}</span></div>
              <div class="setting-footer">
                <el-button :loading="statusLoading" @click="loadStatus">刷新状态</el-button>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="连接配置" name="connection">
          <div class="setting-module" v-loading="configLoading">
            <div class="setting-group">
              <div class="setting-group-header">连接参数</div>
              <div class="setting-item">
                <div class="setting-item-info"><span class="setting-item-label">启用 Docker</span><span class="setting-item-desc">开启后系统将连接到 Docker Engine</span></div>
                <el-switch v-model="configForm.enabled" :disabled="!canEdit" inline-prompt active-text="开启" inactive-text="关闭" />
              </div>
              <div class="setting-item">
                <div class="setting-item-info"><span class="setting-item-label">Docker Host</span><span class="setting-item-desc">如 unix:///var/run/docker.sock 或 tcp://localhost:2375</span></div>
                <el-input v-model="configForm.host" :disabled="!canEdit" placeholder="unix:///var/run/docker.sock" style="width: 320px" />
              </div>
              <div class="setting-item">
                <div class="setting-item-info"><span class="setting-item-label">API 版本</span><span class="setting-item-desc">留空则自动协商</span></div>
                <el-input v-model="configForm.apiVersion" :disabled="!canEdit" placeholder="auto" style="width: 160px" />
              </div>
              <div class="setting-item">
                <div class="setting-item-info"><span class="setting-item-label">请求超时（秒）</span></div>
                <el-input-number v-model="configForm.requestTimeoutSeconds" :disabled="!canEdit" :min="1" :max="300" style="width: 160px" />
              </div>
            </div>

            <div class="setting-group" style="margin-top: 1rem">
              <div class="setting-group-header">TLS</div>
              <div class="setting-item">
                <div class="setting-item-info"><span class="setting-item-label">启用 TLS</span></div>
                <el-switch v-model="configForm.tlsEnabled" :disabled="!canEdit" inline-prompt active-text="开启" inactive-text="关闭" />
              </div>
              <template v-if="configForm.tlsEnabled">
                <div class="setting-item"><div class="setting-item-info"><span class="setting-item-label">CA 证书路径</span></div><el-input v-model="configForm.tlsCaPath" :disabled="!canEdit" style="width: 320px" /></div>
                <div class="setting-item"><div class="setting-item-info"><span class="setting-item-label">客户端证书路径</span></div><el-input v-model="configForm.tlsCertPath" :disabled="!canEdit" style="width: 320px" /></div>
                <div class="setting-item"><div class="setting-item-info"><span class="setting-item-label">客户端私钥路径</span></div><el-input v-model="configForm.tlsKeyPath" :disabled="!canEdit" style="width: 320px" /></div>
              </template>
            </div>

            <div class="setting-footer">
              <el-button type="primary" :loading="configSaving" :disabled="!canEdit" @click="handleSaveConfig">保存配置</el-button>
              <el-button :loading="configTesting" :disabled="!canEdit" @click="handleTestConfig">测试连接</el-button>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="默认设置" name="defaults">
          <div class="setting-module" v-loading="configLoading">
            <div class="setting-group">
              <div class="setting-group-header">默认参数</div>
              <div class="setting-item">
                <div class="setting-item-info"><span class="setting-item-label">默认拉取平台</span></div>
                <el-input v-model="configForm.defaultPlatform" :disabled="!canEdit" placeholder="linux/amd64" style="width: 200px" />
              </div>
              <div class="setting-item">
                <div class="setting-item-info"><span class="setting-item-label">任务超时（秒）</span></div>
                <el-input-number v-model="configForm.taskTimeoutSeconds" :disabled="!canEdit" :min="60" :max="86400" style="width: 160px" />
              </div>
            </div>
            <div class="setting-footer">
              <el-button type="primary" :loading="configSaving" :disabled="!canEdit" @click="handleSaveConfig">保存设置</el-button>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="仓库认证" name="registries">
          <div class="setting-module">
            <div class="setting-group">
              <div class="setting-group-header">认证列表</div>
              <div v-if="!configForm.registryAuths.length" class="setting-item"><span class="text-muted">暂无仓库认证配置</span></div>
              <div v-for="(auth, idx) in configForm.registryAuths" :key="idx" class="setting-item">
                <div class="registry-fields">
                  <el-input v-model="auth.serverAddress" :disabled="!canEdit" placeholder="仓库地址" style="width: 180px" size="small" />
                  <el-input v-model="auth.username" :disabled="!canEdit" placeholder="用户名" style="width: 130px" size="small" />
                  <el-input v-model="auth.password" :disabled="!canEdit" placeholder="密码/Token" type="password" show-password style="width: 180px" size="small" />
                  <el-button type="danger" :disabled="!canEdit" size="small" @click="removeRegistryAuth(idx)">删除</el-button>
                </div>
              </div>
              <div class="setting-footer">
                <el-button :disabled="!canEdit" @click="addRegistryAuth">添加认证</el-button>
                <el-button type="primary" :loading="configSaving" :disabled="!canEdit" @click="handleSaveConfig">保存</el-button>
              </div>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { getDockerConfig, getDockerStatus, testDockerConfig, updateDockerConfig } from '@/api/docker'
import BaseTag from '@/components/tag/BaseTag.vue'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { showRequestError } from '@/utils/request'
import type { DockerConfigInfo, DockerStatusResponse } from '@/types/v1/docker'

defineOptions({ name: 'DockerConfigView' })

const canEdit = useButtonPermission([PERM.DOCKER_CONFIG_EDIT], [])

const activeTab = ref('status')
const statusLoading = ref(false)
const configLoading = ref(false)
const configSaving = ref(false)
const configTesting = ref(false)
const statusInfo = ref<DockerStatusResponse | null>(null)

const defaultConfig = (): DockerConfigInfo => ({
  enabled: false, host: 'unix:///var/run/docker.sock',
  tlsEnabled: false, tlsCaPath: '', tlsCertPath: '', tlsKeyPath: '',
  apiVersion: '', requestTimeoutSeconds: 30,
  defaultPlatform: 'linux/amd64', taskTimeoutSeconds: 3600,
  registryAuths: [],
})

const configForm = reactive<DockerConfigInfo>(defaultConfig())

const formatBytes = (bytes: number | string) => {
  const n = Number(bytes)
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0; let v = n
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

const loadStatus = async () => {
  statusLoading.value = true
  try { const { data } = await getDockerStatus(); statusInfo.value = data || null }
  catch (e) { showRequestError(e, '获取状态失败') }
  finally { statusLoading.value = false }
}

const loadConfig = async () => {
  configLoading.value = true
  try { const { data } = await getDockerConfig(); if (data?.config) Object.assign(configForm, data.config) }
  catch (e) { showRequestError(e, '获取配置失败') }
  finally { configLoading.value = false }
}

const handleSaveConfig = async () => {
  configSaving.value = true
  try { await updateDockerConfig({ config: { ...configForm } }); ElMessage.success('Docker 配置保存成功') }
  catch (e) { showRequestError(e, '保存配置失败') }
  finally { configSaving.value = false }
}

const handleTestConfig = async () => {
  configTesting.value = true
  try {
    const { data } = await testDockerConfig({ config: { ...configForm } })
    if (data?.status?.connected) ElMessage.success('Docker 连接测试成功')
    else ElMessage.error(data?.status?.error || '连接失败')
  } catch (e) { showRequestError(e, '测试连接失败') }
  finally { configTesting.value = false }
}

const addRegistryAuth = () => { configForm.registryAuths.push({ serverAddress: '', username: '', password: '', token: '' }) }
const removeRegistryAuth = (idx: number) => { configForm.registryAuths.splice(idx, 1) }

const loadedTabs = ref(new Set<string>())
const onTabChange = (name: string | number) => {
  const tab = String(name)
  if (loadedTabs.value.has(tab)) return
  loadedTabs.value.add(tab)
  if (tab === 'status') loadStatus()
  else if (tab !== 'status') loadConfig()
}

onMounted(() => { onTabChange(activeTab.value) })
</script>

<style scoped lang="scss">
.setting-module { & + & { margin-top: 1.5rem; } }
.setting-group {
  display: flex; flex-direction: column; gap: 1px;
  background: var(--el-border-color-lighter); border: 1px solid var(--el-border-color-lighter);
  border-radius: 0.5rem; overflow: hidden;
}
.setting-group-header {
  font-size: 0.9rem; font-weight: 600; color: var(--el-text-color-primary);
  padding: 0.5rem 0.75rem; background: var(--el-bg-color-overlay);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.setting-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 0.45rem 0.75rem; background: var(--el-bg-color-overlay);
}
.setting-item-info { display: flex; flex-direction: column; gap: 0.15rem; flex: 1; margin-right: 1rem; min-width: 0; }
.setting-item-label { font-size: 0.875rem; color: var(--el-text-color-primary); }
.setting-item-desc { font-size: 0.75rem; color: var(--el-text-color-placeholder); }
.setting-item-value { font-size: 0.875rem; color: var(--el-text-color-secondary); text-align: right; }
.setting-footer { margin-top: 1rem; }
.setting-group > .setting-footer {
  display: flex;
  gap: 0.5rem;
  margin-top: 0;
  padding: 0.5rem 0.75rem;
  background: var(--el-bg-color-overlay);
}
.status-value { display: flex; align-items: center; }
.status-error { font-size: 0.85rem; color: var(--el-color-danger); }
.text-muted { color: var(--el-text-color-placeholder); font-size: 0.85rem; }
.registry-fields { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; width: 100%; }

@media (width <= 768px) {
  .setting-item { flex-direction: column; align-items: flex-start; gap: 0.5rem; padding: 0.6rem 0.75rem; }
  .setting-item-info { margin-right: 0; }
  .setting-item :deep(.el-input), .setting-item :deep(.el-input-number) { width: 100% !important; }
  .registry-fields { flex-direction: column; align-items: stretch; }
  .registry-fields :deep(.el-input) { width: 100% !important; }
}
</style>
