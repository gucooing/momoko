<!-- Sub2API 配置（重写 · P3 配置型）：PageHeader + 令牌 Tab 条（连接/首页/生图）+ AppPanel 分组，每组分区保存。
     控件用 AppSwitch / 令牌 .app-input/.app-textarea / 令牌 toggle-chip；toast 用 useFeedback。
     保留全部逻辑与接口（useSub2APIStore.loadAdmin/testConfig/saveConfig、configForm、groups、PERM.SUB2API_EDIT、i18n）。 -->
<template>
  <div class="s2a-config" :class="{ 'is-loading': store.configLoading }">
    <PageHeader :title="t('sub2api.admin.configPageTitle')" :description="t('sub2api.admin.configSubtitle')">
      <template #actions>
        <StatusPill :variant="statusVariant" :label="store.statusText(snapshot?.status)" />
      </template>
    </PageHeader>

    <!-- Tab 条 -->
    <div class="s2a-tabs" role="tablist">
      <button
        v-for="tab in TABS"
        :key="tab.name"
        type="button"
        role="tab"
        class="s2a-tabs__btn"
        :class="{ 'is-active': activeTab === tab.name }"
        :aria-selected="activeTab === tab.name"
        @click="activeTab = tab.name"
      >
        <component :is="menuStore.iconComponents[tab.icon]" />
        {{ t(tab.labelKey) }}
      </button>
    </div>

    <!-- 连接配置 -->
    <div v-show="activeTab === 'connection'" class="s2a-tab">
      <AppPanel :title="t('sub2api.common.connectionConfig')" :padded="false">
        <div class="set-rows">
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.autoSync') }}</span>
            </div>
            <AppSwitch v-model="form.syncEnabled" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.baseUrl') }}</span>
            </div>
            <input v-model="form.baseUrl" class="app-input set-wide" :disabled="!canEdit" placeholder="https://your-sub2api.example.com" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.adminApiKey') }}</span>
            </div>
            <div class="set-wide set-pwd">
              <input
                v-model="form.adminApiKey"
                :type="showKey ? 'text' : 'password'"
                class="app-input"
                :disabled="!canEdit"
                placeholder="sk-..."
              />
              <AppIconButton
                :icon="showKey ? 'HOutline:EyeSlashIcon' : 'HOutline:EyeIcon'"
                :label="t('sub2api.admin.adminApiKey')"
                :box="30"
                @click="showKey = !showKey"
              />
            </div>
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.consoleUrl') }}</span>
              <span class="set-row__desc">{{ t('sub2api.admin.consoleUrlHint') }}</span>
            </div>
            <input v-model="form.consoleUrl" class="app-input set-wide" :disabled="!canEdit" placeholder="https://your-sub2api.example.com" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.syncInterval') }}</span>
            </div>
            <input v-model.number="form.syncIntervalMinutes" type="number" min="1" max="1440" class="app-input set-num" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.historyDays') }}</span>
            </div>
            <input v-model.number="form.historyDays" type="number" min="1" max="365" class="app-input set-num" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.pageSize') }}</span>
            </div>
            <input v-model.number="form.pageSize" type="number" min="50" max="1000" step="50" class="app-input set-num" :disabled="!canEdit" />
          </div>
        </div>
        <template v-if="canEdit" #footer>
          <UButton color="primary" :loading="store.saving" @click="onSave">{{ t('sub2api.common.saveConfig') }}</UButton>
          <UButton color="neutral" variant="soft" :loading="store.testing" @click="onTest">{{ t('sub2api.common.testConnection') }}</UButton>
        </template>
      </AppPanel>
    </div>

    <!-- 首页设置 -->
    <div v-show="activeTab === 'home'" class="s2a-tab">
      <AppPanel :title="t('sub2api.common.homeSettings')" :padded="false">
        <div class="set-rows">
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.enablePublicHome') }}</span>
              <span class="set-row__desc">{{ t('sub2api.admin.publicHomeHint') }}</span>
            </div>
            <AppSwitch v-model="form.homeEnabled" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.siteTitle') }}</span>
            </div>
            <input v-model="form.title" class="app-input set-wide" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.subtitleLabel') }}</span>
            </div>
            <input v-model="form.subtitle" class="app-input set-wide" :disabled="!canEdit" />
          </div>
          <div class="set-row set-row--col">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.introduction') }}</span>
            </div>
            <textarea v-model="form.introduction" class="app-textarea" rows="3" :disabled="!canEdit" />
          </div>
          <div class="set-row set-row--col">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.publicGroups') }}</span>
              <span class="set-row__desc">{{ t('sub2api.admin.publicGroupsHint') }}</span>
            </div>
            <div class="grp">
              <button
                v-for="name in groupOptions"
                :key="name"
                type="button"
                class="grp-chip"
                :class="{ 'is-on': form.publicGroups.includes(name) }"
                :disabled="!canEdit"
                @click="toggleGroup(name)"
              >
                <component :is="menuStore.iconComponents['HOutline:CheckIcon']" v-if="form.publicGroups.includes(name)" class="grp-chip__tick" />
                {{ groupLabel(name) }}
              </button>
              <p v-if="!groupOptions.length" class="grp-empty">{{ t('sub2api.admin.noSyncedGroups') }}</p>
            </div>
          </div>
        </div>
        <template v-if="canEdit" #footer>
          <UButton color="primary" :loading="store.saving" @click="onSave">{{ t('sub2api.common.saveConfig') }}</UButton>
        </template>
      </AppPanel>
    </div>

    <!-- 生图设置 -->
    <div v-show="activeTab === 'image'" class="s2a-tab">
      <AppPanel :title="t('sub2api.common.imageSettings')" :padded="false">
        <div class="set-rows">
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.enableImageGen') }}</span>
              <span class="set-row__desc">{{ t('sub2api.admin.imageGenHint') }}</span>
            </div>
            <AppSwitch v-model="form.imageEnabled" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.hostWhitelist') }}</span>
              <span class="set-row__desc">{{ t('sub2api.admin.hostWhitelistHint') }}</span>
            </div>
            <AppSwitch v-model="form.srcHostWhitelistEnabled" :disabled="!canEdit" />
          </div>
          <div class="set-row set-row--col">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.admin.allowedHosts') }}</span>
            </div>
            <div class="hosts">
              <span v-for="(host, i) in form.allowedSrcHosts" :key="i" class="host-chip">
                {{ host }}
                <button v-if="canEdit" type="button" class="host-chip__x" :aria-label="host" @click="removeHost(i)">
                  <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
                </button>
              </span>
              <input
                v-if="hostInputVisible"
                ref="hostInputRef"
                v-model="hostInputValue"
                class="app-input host-add"
                @keyup.enter="addHost"
                @blur="addHost"
              />
              <button v-else-if="canEdit" type="button" class="host-add-btn" @click="showHostInput">
                {{ t('sub2api.admin.addHost') }}
              </button>
            </div>
          </div>
        </div>
        <template v-if="canEdit" #footer>
          <UButton color="primary" :loading="store.saving" @click="onSave">{{ t('sub2api.common.saveConfig') }}</UButton>
        </template>
      </AppPanel>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { useSub2APIStore } from '@/stores/sub2api'
import { useFeedback } from '@/utils/feedback'

defineOptions({ name: 'Sub2APIConfig' })

const store = useSub2APIStore()
const { snapshot, configForm: form, groups } = storeToRefs(store)
const { t } = useI18n()
const menuStore = useMenuStore()
const fb = useFeedback()
const canEdit = useButtonPermission([PERM.SUB2API_EDIT], [])

const TABS = [
  { name: 'connection', labelKey: 'sub2api.common.connectionConfig', icon: 'HOutline:LinkIcon' },
  { name: 'home', labelKey: 'sub2api.common.homeSettings', icon: 'HOutline:HomeIcon' },
  { name: 'image', labelKey: 'sub2api.common.imageSettings', icon: 'HOutline:PhotoIcon' },
] as const

const activeTab = ref<'connection' | 'home' | 'image'>('connection')
const showKey = ref(false)

// 同步状态 → StatusPill 变体（store.statusType 沿用 EP 语义：danger → error）
const STATUS_VARIANT = { warning: 'warning', success: 'success', danger: 'error', info: 'info' } as const
const statusVariant = computed(
  () => STATUS_VARIANT[store.statusType(snapshot.value?.status) as keyof typeof STATUS_VARIANT] ?? 'neutral',
)

const DELETED_GROUP_KEY = '__deleted__'
// 活跃分组逐项展示；已删除合并为单一选项
const groupOptions = computed(() => {
  const active = (groups.value || []).filter((g) => g && !g.deleted && g.name).map((g) => g.name)
  const hasDeleted = (groups.value || []).some((g) => g?.deleted)
  return hasDeleted ? [...active, DELETED_GROUP_KEY] : active
})
const groupLabel = (name: string) => (name === DELETED_GROUP_KEY ? t('sub2api.admin.deletedGroups') : name)
const toggleGroup = (name: string) => {
  if (!canEdit.value) return
  const arr = form.value.publicGroups
  const idx = arr.indexOf(name)
  if (idx >= 0) arr.splice(idx, 1)
  else arr.push(name)
}

const onTest = async () => {
  if (!canEdit.value) return
  const result = await store.testConfig()
  if (!result) return
  if (result.connected) fb.success(result.message || t('sub2api.common.connected'))
  else fb.error(result.message || t('sub2api.common.disconnected'))
}

const onSave = async () => {
  if (!canEdit.value) return
  const ok = await store.saveConfig()
  if (ok) fb.success(t('sub2api.common.saved'))
}

// 允许站点动态输入
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
const removeHost = (i: number) => {
  if (!canEdit.value) return
  form.value.allowedSrcHosts.splice(i, 1)
}

onMounted(() => {
  store.loadAdmin()
})
</script>

<style scoped lang="scss">
.s2a-config {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.s2a-config.is-loading .s2a-tab {
  opacity: 0.55;
  pointer-events: none;
}

/* Tab 条 —— 令牌分段 */
.s2a-tabs {
  display: inline-flex;
  align-self: flex-start;
  padding: 3px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
}
.s2a-tabs__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 16px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.s2a-tabs__btn :deep(svg) {
  width: 16px;
  height: 16px;
}
.s2a-tabs__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}

.s2a-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
  transition: opacity 0.15s;
  max-width: 760px;
}

/* 设置行 */
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
  line-height: 1.45;
}
.set-row--col {
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
}
.set-row--col .set-row__info {
  flex: none;
}

/* 控件宽度 */
.set-wide {
  width: 360px;
  max-width: 100%;
  flex-shrink: 0;
}
.set-num {
  width: 150px;
  flex-shrink: 0;
}
.set-pwd {
  display: flex;
  align-items: center;
  gap: 6px;
}
.set-pwd .app-input {
  flex: 1;
  min-width: 0;
}

/* 展示分组：toggle chip 多选 */
.grp {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.grp-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}
.grp-chip:hover:not(:disabled) {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.grp-chip.is-on {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
}
.grp-chip.is-on:hover:not(:disabled) {
  background: color-mix(in srgb, var(--el-color-primary) 16%, transparent);
}
.grp-chip:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
.grp-chip__tick {
  width: 14px;
  height: 14px;
}
.grp-empty {
  margin: 0;
  color: var(--el-text-color-placeholder);
  font-size: 0.75rem;
}

/* 允许站点：可移除 chip + 行内追加输入 */
.hosts {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.host-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 6px 3px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 999px;
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
  font-variant-numeric: tabular-nums;
}
.host-chip__x {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.host-chip__x:hover {
  background: color-mix(in srgb, var(--el-color-danger) 14%, transparent);
  color: var(--el-color-danger);
}
.host-chip__x :deep(svg) {
  width: 12px;
  height: 12px;
}
.host-add {
  width: 220px;
}
.host-add-btn {
  padding: 4px 12px;
  border: 1px dashed var(--el-border-color);
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s;
}
.host-add-btn:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}

@media (width <= 768px) {
  .s2a-tabs {
    align-self: stretch;
    display: flex;
  }
  .s2a-tabs__btn {
    flex: 1;
    justify-content: center;
  }
  .set-row {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
  .set-row__info {
    flex: none;
  }
  .set-row :deep(.app-input),
  .set-wide,
  .set-num,
  .host-add {
    width: 100%;
  }
  .set-row .app-switch {
    align-self: flex-start;
  }
}
</style>
