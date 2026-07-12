<!-- 隧道 新建/编辑（交互重写）：类型决定字段，本地服务是终点。
     心智：① 叫什么 ② 走哪种穿透（TCP/HTTP…）③ 公网入口 ④ 落到本地哪 ⑤ 可选限制。
     TCP/UDP：remotePort 与 local 用路由条；HTTP/HTTPS：域名优先；高级限制默认折叠。 -->
<template>
  <FormDialog
    v-model="open"
    :title="editingId ? t('tools.tunnel.editTunnel') : t('tools.tunnel.addTunnel')"
    :width="600"
    :loading="loading"
    @close="close"
    @confirm="confirm"
  >
    <div class="tn-form">
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('tools.tunnel.name') }}</label>
        <input
          ref="nameRef"
          v-model="form.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          :placeholder="t('tools.tunnel.namePlaceholder')"
          spellcheck="false"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
        <span class="tn-form__tip">{{ t('tools.tunnel.nameTip') }}</span>
      </div>

      <!-- 类型：大按钮网格，一眼看懂差异（不是下拉埋信息） -->
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('tools.tunnel.proxyType') }}</label>
        <div class="type-grid" role="radiogroup">
          <button
            v-for="opt in typeCards"
            :key="opt.value"
            type="button"
            role="radio"
            class="type-card"
            :class="{ 'is-active': form.type === opt.value }"
            :aria-checked="form.type === opt.value"
            @click="form.type = opt.value"
          >
            <strong>{{ opt.label }}</strong>
            <span>{{ opt.desc }}</span>
          </button>
        </div>
      </div>

      <!-- 公网入口：按类型切换 -->
      <div v-if="isPortType" class="block">
        <div class="block__title">{{ t('tools.tunnel.publicEntry') }}</div>
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('tools.tunnel.remotePort') }}</label>
          <div class="port-row">
            <span class="port-row__prefix">:</span>
            <input
              v-model.number="form.remotePort"
              type="number"
              min="1"
              max="65535"
              class="app-input port-row__input"
              :class="{ 'is-error': errors.remotePort }"
              :placeholder="t('tools.tunnel.remotePortPh')"
            />
          </div>
          <span v-if="errors.remotePort" class="app-field__error">{{ errors.remotePort }}</span>
          <span class="tn-form__tip">{{ t('tools.tunnel.remotePortTip') }}</span>
        </div>
      </div>

      <div v-else-if="isHttpType" class="block">
        <div class="block__title">{{ t('tools.tunnel.publicEntry') }}</div>
        <div class="app-field">
          <label class="app-label">{{ t('tools.tunnel.customDomains') }}</label>
          <input
            v-model="form.customDomains"
            class="app-input"
            :class="{ 'is-error': errors.domains }"
            :placeholder="t('tools.tunnel.customDomainsPlaceholder')"
            spellcheck="false"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('tools.tunnel.subdomain') }}</label>
          <input
            v-model="form.subdomain"
            class="app-input"
            :class="{ 'is-error': errors.domains }"
            placeholder="my-app"
            spellcheck="false"
          />
          <span v-if="errors.domains" class="app-field__error">{{ errors.domains }}</span>
          <span class="tn-form__tip">{{ t('tools.tunnel.httpEntryTip') }}</span>
        </div>
      </div>

      <div v-else class="block block--soft">
        <div class="block__title">{{ t('tools.tunnel.publicEntry') }}</div>
        <p class="tn-form__note">{{ t('tools.tunnel.p2pEntryTip') }}</p>
      </div>

      <!-- 本地终点：所有类型共用，视觉上像「落到这里」 -->
      <div class="block">
        <div class="block__title">{{ t('tools.tunnel.localTarget') }}</div>
        <div class="route-local">
          <div class="route-local__field">
            <label class="app-label">{{ t('tools.tunnel.localIp') }}</label>
            <input v-model="form.localIp" class="app-input mono" placeholder="127.0.0.1" spellcheck="false" />
          </div>
          <span class="route-local__colon">:</span>
          <div class="route-local__field route-local__port">
            <label class="app-label app-label--required">{{ t('tools.tunnel.localPort') }}</label>
            <input
              v-model.number="form.localPort"
              type="number"
              min="1"
              max="65535"
              class="app-input mono"
              :class="{ 'is-error': errors.localPort }"
              :placeholder="t('tools.tunnel.localPortPh')"
            />
          </div>
        </div>
        <span v-if="errors.localPort" class="app-field__error">{{ errors.localPort }}</span>
      </div>

      <!-- 高级：默认收起，避免干扰主路径 -->
      <button type="button" class="adv-toggle" @click="showAdvanced = !showAdvanced">
        <span>{{ t('tools.tunnel.advanced') }}</span>
        <span class="adv-toggle__chev" :class="{ 'is-open': showAdvanced }">▾</span>
      </button>
      <div v-if="showAdvanced" class="block block--adv">
        <div class="app-field">
          <label class="app-label">{{ t('tools.tunnel.allowUsers') }}</label>
          <input v-model="form.allowUsers" class="app-input" :placeholder="t('tools.tunnel.allowUsersPlaceholder')" />
        </div>
        <div class="adv-grid">
          <div class="app-field">
            <label class="app-label">{{ t('tools.tunnel.maxBandwidth') }}</label>
            <input v-model="form.maxBandwidth" class="app-input" :placeholder="t('tools.tunnel.maxBandwidthPlaceholder')" />
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('tools.tunnel.maxActiveConns') }}</label>
            <input v-model.number="form.maxActiveConns" type="number" min="0" max="100000" class="app-input" />
          </div>
        </div>
      </div>

      <div class="tn-form__enable">
        <div>
          <div class="tn-form__enable-title">{{ t('tools.tunnel.enableNow') }}</div>
          <div class="tn-form__enable-desc">{{ t('tools.tunnel.enableNowDesc') }}</div>
        </div>
        <AppSwitch v-model="form.isEnable" />
      </div>

      <div class="preview" :class="{ 'is-ready': previewReady }">
        <span class="preview__label">{{ t('tools.tunnel.preview') }}</span>
        <code class="preview__code">{{ previewText }}</code>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { createTunnel, updateTunnel } from '@/api/tunnel'
import { TunnelType, type TunnelInfo } from '@/types/v1/tunnel'

defineOptions({ name: 'TunnelCreate' })

const emits = defineEmits<{ refresh: [type: 'create' | 'update'] }>()
const { t } = useI18n()

const open = ref(false)
const loading = ref(false)
const editingId = ref('')
const errors = ref<Record<string, string>>({})
const showAdvanced = ref(false)
const nameRef = ref<HTMLInputElement | null>(null)

const typeCards = computed(() => [
  { value: TunnelType.TUNNEL_TYPE_TCP, label: 'TCP', desc: t('tools.tunnel.typeDesc.tcp') },
  { value: TunnelType.TUNNEL_TYPE_UDP, label: 'UDP', desc: t('tools.tunnel.typeDesc.udp') },
  { value: TunnelType.TUNNEL_TYPE_HTTP, label: 'HTTP', desc: t('tools.tunnel.typeDesc.http') },
  { value: TunnelType.TUNNEL_TYPE_HTTPS, label: 'HTTPS', desc: t('tools.tunnel.typeDesc.https') },
  { value: TunnelType.TUNNEL_TYPE_STCP, label: 'STCP', desc: t('tools.tunnel.typeDesc.stcp') },
  { value: TunnelType.TUNNEL_TYPE_XTCP, label: 'XTCP', desc: t('tools.tunnel.typeDesc.xtcp') },
  { value: TunnelType.TUNNEL_TYPE_TCPMUX, label: 'TCPMUX', desc: t('tools.tunnel.typeDesc.tcpmux') },
])

const defaultForm = () => ({
  name: '',
  type: TunnelType.TUNNEL_TYPE_TCP as TunnelType,
  remotePort: undefined as number | undefined,
  customDomains: '',
  subdomain: '',
  localIp: '127.0.0.1',
  localPort: undefined as number | undefined,
  allowUsers: '',
  maxBandwidth: '',
  maxActiveConns: 0,
  isEnable: true,
})
const form = ref(defaultForm())

const isPortType = computed(() =>
  [TunnelType.TUNNEL_TYPE_TCP, TunnelType.TUNNEL_TYPE_UDP, TunnelType.TUNNEL_TYPE_TCPMUX].includes(form.value.type),
)
const isHttpType = computed(() =>
  [TunnelType.TUNNEL_TYPE_HTTP, TunnelType.TUNNEL_TYPE_HTTPS].includes(form.value.type),
)
const typeLabel = (type: TunnelType) => type.replace('TUNNEL_TYPE_', '')

const previewReady = computed(() => {
  if (!form.value.name.trim() || !form.value.localPort) return false
  if (isPortType.value) return !!form.value.remotePort
  if (isHttpType.value) return !!(form.value.customDomains.trim() || form.value.subdomain.trim())
  return true
})
const previewText = computed(() => {
  const name = form.value.name.trim() || '—'
  const kind = typeLabel(form.value.type)
  const local = `${form.value.localIp || '127.0.0.1'}:${form.value.localPort || '?'}`
  let entry = '—'
  if (isPortType.value) entry = `:${form.value.remotePort || '?'}`
  else if (isHttpType.value) entry = form.value.customDomains.trim() || form.value.subdomain.trim() || '—'
  else entry = t('tools.tunnel.p2pShort')
  const on = form.value.isEnable ? t('tools.tunnel.enabled') : t('tools.tunnel.disabled')
  return `${name} · ${kind}  ${entry} → ${local}  · ${on}`
})

const close = () => {
  open.value = false
  loading.value = false
  editingId.value = ''
  errors.value = {}
  showAdvanced.value = false
  form.value = defaultForm()
}

const validate = () => {
  const e: Record<string, string> = {}
  if (!form.value.name.trim()) e.name = t('tools.tunnel.nameRequired')
  if (isPortType.value && !form.value.remotePort) e.remotePort = t('tools.tunnel.remotePortRequired')
  if (!form.value.localPort) e.localPort = t('tools.tunnel.localPortRequired')
  if (isHttpType.value && !form.value.customDomains.trim() && !form.value.subdomain.trim()) {
    e.domains = t('tools.tunnel.domainsRequired')
  }
  errors.value = e
  return Object.keys(e).length === 0
}

const confirm = async () => {
  if (!validate()) return
  loading.value = true
  try {
    if (editingId.value) {
      await updateTunnel({
        id: editingId.value,
        name: form.value.name || undefined,
        type: form.value.type,
        remotePort: form.value.remotePort,
        customDomains: form.value.customDomains,
        subdomain: form.value.subdomain,
        localIp: form.value.localIp || undefined,
        localPort: form.value.localPort,
        allowUsers: form.value.allowUsers,
        maxBandwidth: form.value.maxBandwidth,
        maxActiveConns: form.value.maxActiveConns,
        isEnable: form.value.isEnable,
      })
      ElMessage.success(t('tools.tunnel.editSuccess'))
    } else {
      await createTunnel({
        name: form.value.name,
        type: form.value.type,
        remotePort: form.value.remotePort ?? 0,
        customDomains: form.value.customDomains,
        subdomain: form.value.subdomain,
        localIp: form.value.localIp,
        localPort: form.value.localPort ?? 0,
        allowUsers: form.value.allowUsers,
        maxBandwidth: form.value.maxBandwidth,
        maxActiveConns: form.value.maxActiveConns,
        isEnable: form.value.isEnable,
      })
      ElMessage.success(t('tools.tunnel.addSuccess'))
    }
    emits('refresh', editingId.value ? 'update' : 'create')
    close()
  } finally {
    loading.value = false
  }
}

const showDialog = (payload?: TunnelInfo) => {
  open.value = true
  errors.value = {}
  showAdvanced.value = false
  if (!payload?.id) {
    editingId.value = ''
    form.value = defaultForm()
    nextTick(() => nameRef.value?.focus())
    return
  }
  editingId.value = payload.id
  form.value = {
    name: payload.name || '',
    type: payload.type || TunnelType.TUNNEL_TYPE_TCP,
    remotePort: payload.remotePort || undefined,
    customDomains: payload.customDomains || '',
    subdomain: payload.subdomain || '',
    localIp: payload.localIp || '127.0.0.1',
    localPort: payload.localPort || undefined,
    allowUsers: payload.allowUsers || '',
    maxBandwidth: payload.maxBandwidth || '',
    maxActiveConns: payload.maxActiveConns || 0,
    isEnable: payload.isEnable ?? true,
  }
  // 编辑时若已有高级字段，自动展开，避免「有值看不见」
  if (payload.allowUsers || payload.maxBandwidth || payload.maxActiveConns) showAdvanced.value = true
  nextTick(() => nameRef.value?.focus())
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.tn-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.tn-form__tip {
  margin-top: 4px;
  font-size: 0.72rem;
  color: var(--el-text-color-placeholder);
  line-height: 1.4;
}
.tn-form__note {
  margin: 0;
  font-size: 0.78rem;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.type-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px;
}
.type-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s, box-shadow 0.15s;
  strong {
    font-size: 0.8125rem;
    font-weight: 700;
    color: var(--el-text-color-primary);
    font-variant-numeric: tabular-nums;
  }
  span {
    font-size: 0.65rem;
    color: var(--el-text-color-placeholder);
    line-height: 1.3;
  }
}
.type-card.is-active {
  border-color: color-mix(in srgb, var(--el-color-primary) 50%, transparent);
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 25%, transparent);
  strong { color: var(--el-color-primary); }
}

.block {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-lighter);
}
.block--soft { background: var(--el-fill-color-light); }
.block--adv { background: var(--el-bg-color); }
.block__title {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  letter-spacing: 0.02em;
}

.port-row {
  display: flex;
  align-items: center;
  gap: 4px;
  max-width: 180px;
}
.port-row__prefix {
  font-family: 'Cascadia Code', Consolas, monospace;
  font-size: 1rem;
  font-weight: 700;
  color: var(--el-text-color-placeholder);
}
.port-row__input {
  font-family: 'Cascadia Code', Consolas, monospace;
  font-variant-numeric: tabular-nums;
}

.route-local {
  display: flex;
  align-items: flex-end;
  gap: 6px;
}
.route-local__field { flex: 1; min-width: 0; }
.route-local__port { flex: 0 0 110px; }
.route-local__colon {
  padding-bottom: 8px;
  font-weight: 700;
  color: var(--el-text-color-placeholder);
}
.mono {
  font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
  font-size: 0.8125rem;
}

.adv-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  align-self: flex-start;
  border: none;
  background: transparent;
  color: var(--el-color-primary);
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
  padding: 0;
}
.adv-toggle__chev {
  display: inline-block;
  transition: transform 0.15s;
  font-size: 0.75rem;
}
.adv-toggle__chev.is-open { transform: rotate(180deg); }
.adv-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.tn-form__enable {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
}
.tn-form__enable-title {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--el-text-color-primary);
}
.tn-form__enable-desc {
  margin-top: 2px;
  font-size: 0.72rem;
  color: var(--el-text-color-placeholder);
}

.preview {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 10px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  border: 1px dashed var(--el-border-color-lighter);
}
.preview.is-ready {
  border-style: solid;
  border-color: color-mix(in srgb, var(--el-color-primary) 28%, transparent);
  background: color-mix(in srgb, var(--el-color-primary) 6%, transparent);
}
.preview__label {
  font-size: 0.7rem;
  color: var(--el-text-color-placeholder);
  font-weight: 600;
}
.preview__code {
  font-family: 'Cascadia Code', Consolas, monospace;
  font-size: 0.78rem;
  color: var(--el-text-color-regular);
  word-break: break-all;
  line-height: 1.45;
}

@media (width <= 640px) {
  .type-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .adv-grid { grid-template-columns: 1fr; }
  .route-local {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
  .route-local__port { flex: none; width: 100%; }
  .route-local__colon { display: none; }
  .port-row { max-width: none; width: 100%; }
}
</style>
