<template>
  <div class="sub2api-config" v-loading="store.configLoading">
    <header class="page-head">
      <div>
        <h1>{{ t('sub2api.common.connectionConfig') }}</h1>
        <p>{{ t('sub2api.admin.configSubtitle') }}</p>
      </div>
      <el-tag :type="store.statusType(snapshot?.status)" effect="light" round>
        {{ store.statusText(snapshot?.status) }}
      </el-tag>
    </header>

    <el-tabs v-model="activeTab" class="config-tabs">
      <!-- 连接配置 -->
      <el-tab-pane :label="t('sub2api.common.connectionConfig')" name="connection">
        <el-card shadow="never" class="form-card">
          <el-form :model="form" label-width="130px" label-position="right" :disabled="!canEdit">
            <el-form-item :label="t('sub2api.admin.autoSync')">
              <el-switch v-model="form.syncEnabled" />
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.baseUrl')">
              <el-input v-model="form.baseUrl" placeholder="https://your-sub2api.example.com" />
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.adminApiKey')">
              <el-input
                v-model="form.adminApiKey"
                type="password"
                show-password
                placeholder="sk-..."
              />
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.consoleUrl')">
              <el-input v-model="form.consoleUrl" placeholder="https://your-sub2api.example.com" />
              <span class="form-hint">{{ t('sub2api.admin.consoleUrlHint') }}</span>
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.syncInterval')">
              <el-input-number v-model="form.syncIntervalMinutes" :min="1" :max="1440" />
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.historyDays')">
              <el-input-number v-model="form.historyDays" :min="1" :max="365" />
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.pageSize')">
              <el-input-number v-model="form.pageSize" :min="50" :max="1000" :step="50" />
            </el-form-item>
            <el-form-item v-if="canEdit">
              <el-button :loading="store.testing" @click="onTest">{{
                t('sub2api.common.testConnection')
              }}</el-button>
              <el-button type="primary" :loading="store.saving" @click="onSave">{{
                t('sub2api.common.saveConfig')
              }}</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <!-- 首页设置 -->
      <el-tab-pane :label="t('sub2api.common.homeSettings')" name="home">
        <el-card shadow="never" class="form-card">
          <el-form :model="form" label-width="130px" label-position="right" :disabled="!canEdit">
            <el-form-item :label="t('sub2api.admin.enablePublicHome')">
              <el-switch v-model="form.homeEnabled" />
              <span class="form-hint">{{ t('sub2api.admin.publicHomeHint') }}</span>
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.siteTitle')">
              <el-input v-model="form.title" />
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.subtitleLabel')">
              <el-input v-model="form.subtitle" />
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.introduction')">
              <el-input v-model="form.introduction" type="textarea" :rows="3" />
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.publicGroups')">
              <div class="group-picker">
                <el-checkbox-group v-model="form.publicGroups" class="group-checkboxes">
                  <el-checkbox v-for="name in groupOptions" :key="name" :label="name">
                    {{ groupLabel(name) }}
                  </el-checkbox>
                </el-checkbox-group>
                <p v-if="!groupOptions.length" class="form-empty">
                  {{ t('sub2api.admin.noSyncedGroups') }}
                </p>
                <span class="form-hint block">{{ t('sub2api.admin.publicGroupsHint') }}</span>
              </div>
            </el-form-item>
            <el-form-item v-if="canEdit">
              <el-button type="primary" :loading="store.saving" @click="onSave">{{
                t('sub2api.common.saveConfig')
              }}</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <!-- 生图设置 -->
      <el-tab-pane :label="t('sub2api.common.imageSettings')" name="image">
        <el-card shadow="never" class="form-card">
          <el-form :model="form" label-width="130px" label-position="right" :disabled="!canEdit">
            <el-form-item :label="t('sub2api.admin.enableImageGen')">
              <el-switch v-model="form.imageEnabled" />
              <span class="form-hint">{{ t('sub2api.admin.imageGenHint') }}</span>
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.hostWhitelist')">
              <el-switch v-model="form.srcHostWhitelistEnabled" />
              <span class="form-hint">{{ t('sub2api.admin.hostWhitelistHint') }}</span>
            </el-form-item>
            <el-form-item :label="t('sub2api.admin.allowedHosts')">
              <div class="host-tags">
                <el-tag
                  v-for="(host, i) in form.allowedSrcHosts"
                  :key="i"
                  :closable="canEdit"
                  @close="canEdit && form.allowedSrcHosts.splice(i, 1)"
                  >{{ host }}</el-tag
                >
                <el-input
                  v-if="hostInputVisible"
                  ref="hostInputRef"
                  v-model="hostInputValue"
                  size="small"
                  class="host-input"
                  @keyup.enter="addHost"
                  @blur="addHost"
                />
                <el-button v-else-if="canEdit" size="small" @click="showHostInput">{{
                  t('sub2api.admin.addHost')
                }}</el-button>
              </div>
            </el-form-item>
            <el-form-item v-if="canEdit">
              <el-button type="primary" :loading="store.saving" @click="onSave">{{
                t('sub2api.common.saveConfig')
              }}</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { useSub2APIStore } from '@/stores/sub2api'

defineOptions({ name: 'Sub2APIConfig' })

const store = useSub2APIStore()
const { snapshot, configForm: form, groups } = storeToRefs(store)
const { t } = useI18n()
const canEdit = useButtonPermission([PERM.SUB2API_EDIT], [])

const activeTab = ref('connection')

const DELETED_GROUP_KEY = '__deleted__'
// 活跃分组逐项展示；已删除合并为单一选项
const groupOptions = computed(() => {
  const active = (groups.value || [])
    .filter((g) => g && !g.deleted && g.name)
    .map((g) => g.name)
  const hasDeleted = (groups.value || []).some((g) => g?.deleted)
  return hasDeleted ? [...active, DELETED_GROUP_KEY] : active
})
const groupLabel = (name: string) =>
  name === DELETED_GROUP_KEY ? t('sub2api.admin.deletedGroups') : name

const onTest = async () => {
  if (!canEdit.value) return
  const result = await store.testConfig()
  if (!result) return
  if (result.connected) ElMessage.success(result.message || t('sub2api.common.connected'))
  else ElMessage.error(result.message || t('sub2api.common.disconnected'))
}

const onSave = async () => {
  if (!canEdit.value) return
  const ok = await store.saveConfig()
  if (ok) ElMessage.success(t('sub2api.common.saved'))
}

// 允许站点动态 tag 输入
const hostInputVisible = ref(false)
const hostInputValue = ref('')
const hostInputRef = ref<{ focus: () => void } | null>(null)
const showHostInput = () => {
  if (!canEdit.value) return
  hostInputVisible.value = true
  nextTick(() => hostInputRef.value?.focus())
}
const addHost = () => {
  if (!canEdit.value) return
  const v = hostInputValue.value.trim().replace(/\/+$/, '')
  if (v && !form.value.allowedSrcHosts.includes(v)) form.value.allowedSrcHosts.push(v)
  hostInputVisible.value = false
  hostInputValue.value = ''
}

onMounted(() => {
  store.loadAdmin()
})
</script>

<style scoped lang="scss">
.sub2api-config {
  padding: 16px;
}

.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 16px;

  h1 {
    margin: 0;
    font-size: 20px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  p {
    margin: 6px 0 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
}

.form-card {
  max-width: 720px;
  border-radius: 12px;
}

.form-hint {
  margin-left: 10px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;

  &.block {
    display: block;
    margin: 8px 0 0;
    margin-left: 0;
    line-height: 1.45;
  }
}

.form-empty {
  margin: 0;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.group-picker {
  width: 100%;
}

.group-checkboxes {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 12px;
}

.host-tags {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.host-input {
  width: 220px;
}

@media (max-width: 640px) {
  .sub2api-config {
    padding: 10px;
  }

  .form-card {
    max-width: none;
  }

  .form-card :deep(.el-form-item) {
    display: block;
    margin-bottom: 13px;
  }

  .form-card :deep(.el-form-item__label) {
    display: block;
    width: auto !important;
    margin-bottom: 5px;
    text-align: left;
  }

  .form-card :deep(.el-form-item__content) {
    width: 100%;
    margin-left: 0 !important;
  }

  .form-card :deep(.el-input),
  .form-card :deep(.el-input-number) {
    width: 100%;
  }

  .form-hint {
    display: block;
    width: 100%;
    margin: 4px 0 0;

    &.block {
      margin-top: 8px;
    }
  }

  .host-input {
    width: 100%;
  }
}
</style>
