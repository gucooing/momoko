<!-- 端口转发 新建/编辑（交互重写）：路由心智优先。
     人想的是「把本机某端口转到某地址」，不是一堆平铺字段。
     布局：名称 → 协议 → 可视化路由条（监听 → 目标）→ 启用。
     默认监听 0.0.0.0 / 目标 127.0.0.1；底部实时预览即将创建的规则。 -->
<template>
  <FormDialog
    v-model="open"
    :title="editingId ? t('tools.portForward.editPortForward') : t('tools.portForward.addPortForward')"
    :width="560"
    :loading="loading"
    @close="close"
    @confirm="confirm"
  >
    <div class="pf-form">
      <!-- 名称：次要，但必填；放顶部避免滚动后忘记 -->
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('tools.portForward.ruleName') }}</label>
        <input
          ref="nameRef"
          v-model="form.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          :placeholder="namePlaceholder"
          maxlength="64"
          @keyup.enter="confirm"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
      </div>

      <!-- 协议：紧凑分段，紧贴路由（这是规则的第一属性） -->
      <div class="app-field">
        <label class="app-label">{{ t('tools.portForward.forwardType') }}</label>
        <div class="seg">
          <button
            type="button"
            class="seg__btn"
            :class="{ 'is-active': form.type === 'PORT_FORWARD_TYPE_TCP' }"
            @click="form.type = 'PORT_FORWARD_TYPE_TCP'"
          >TCP</button>
          <button
            type="button"
            class="seg__btn"
            :class="{ 'is-active': form.type === 'PORT_FORWARD_TYPE_UDP' }"
            @click="form.type = 'PORT_FORWARD_TYPE_UDP'"
          >UDP</button>
        </div>
      </div>

      <!-- 核心：路由条 —— 监听端 → 目标端（人类读规则的顺序） -->
      <div class="route" :class="{ 'is-error': hasRouteError }">
        <div class="route__side">
          <div class="route__side-head">
            <span class="route__badge">{{ t('tools.portForward.listenSide') }}</span>
            <span class="route__hint">{{ t('tools.portForward.listenSideHint') }}</span>
          </div>
          <div class="route__endpoint">
            <input
              v-model="form.listenAddress"
              class="app-input route__host"
              :class="{ 'is-error': errors.listenAddress }"
              :placeholder="t('tools.portForward.listenAddressPh')"
              spellcheck="false"
            />
            <span class="route__colon">:</span>
            <input
              v-model.number="form.listenPort"
              type="number"
              min="1"
              max="65535"
              class="app-input route__port"
              :class="{ 'is-error': errors.listenPort }"
              :placeholder="t('tools.portForward.portPh')"
            />
          </div>
          <span v-if="errors.listenAddress || errors.listenPort" class="app-field__error">
            {{ errors.listenAddress || errors.listenPort }}
          </span>
        </div>

        <div class="route__arrow" aria-hidden="true">
          <span class="route__arrow-line" />
          <span class="route__arrow-label">{{ typeShort }}</span>
          <span class="route__arrow-chev">→</span>
        </div>

        <div class="route__side">
          <div class="route__side-head">
            <span class="route__badge route__badge--target">{{ t('tools.portForward.targetSide') }}</span>
            <span class="route__hint">{{ t('tools.portForward.targetSideHint') }}</span>
          </div>
          <div class="route__endpoint">
            <input
              v-model="form.targetAddress"
              class="app-input route__host"
              :class="{ 'is-error': errors.targetAddress }"
              :placeholder="t('tools.portForward.targetAddressPh')"
              spellcheck="false"
            />
            <span class="route__colon">:</span>
            <input
              v-model.number="form.targetPort"
              type="number"
              min="1"
              max="65535"
              class="app-input route__port"
              :class="{ 'is-error': errors.targetPort }"
              :placeholder="t('tools.portForward.portPh')"
              @keyup.enter="confirm"
            />
          </div>
          <span v-if="errors.targetAddress || errors.targetPort" class="app-field__error">
            {{ errors.targetAddress || errors.targetPort }}
          </span>
        </div>
      </div>

      <!-- 启用：做成一句「创建后立刻生效」的选择，而不是孤立开关 -->
      <div class="pf-form__enable">
        <div class="pf-form__enable-text">
          <span class="pf-form__enable-title">{{ t('tools.portForward.enableNow') }}</span>
          <span class="pf-form__enable-desc">{{ t('tools.portForward.enableNowDesc') }}</span>
        </div>
        <AppSwitch v-model="form.isEnable" />
      </div>

      <!-- 实时预览：确认前看见最终规则 -->
      <div class="preview" :class="{ 'is-ready': previewReady }">
        <span class="preview__label">{{ t('tools.portForward.preview') }}</span>
        <code class="preview__code">{{ previewText }}</code>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { createPortForward, updatePortForward } from '@/api/network'
import { PortForwardType, type PortForwardInfo } from '@/types/v1/network'

defineOptions({ name: 'PortForwardCreate' })

const emits = defineEmits<{ refresh: [type: 'create' | 'update'] }>()
const { t } = useI18n()

const open = ref(false)
const loading = ref(false)
const editingId = ref('')
const errors = ref<Record<string, string>>({})
const nameRef = ref<HTMLInputElement | null>(null)

const defaultForm = () => ({
  name: '',
  type: 'PORT_FORWARD_TYPE_TCP' as string,
  listenAddress: '0.0.0.0',
  listenPort: undefined as number | undefined,
  targetAddress: '127.0.0.1',
  targetPort: undefined as number | undefined,
  isEnable: true,
})
const form = ref(defaultForm())

const typeShort = computed(() => (form.value.type === 'PORT_FORWARD_TYPE_UDP' ? 'UDP' : 'TCP'))
const hasRouteError = computed(() =>
  !!(errors.value.listenAddress || errors.value.listenPort || errors.value.targetAddress || errors.value.targetPort),
)

// 名称未填时用路由生成建议占位，降低起名摩擦
const namePlaceholder = computed(() => {
  const lp = form.value.listenPort
  const tp = form.value.targetPort
  if (lp && tp) return t('tools.portForward.ruleNameAutoPh', { listen: lp, target: tp })
  return t('tools.portForward.ruleNamePlaceholder')
})

const previewReady = computed(() =>
  !!(form.value.listenAddress.trim() && form.value.listenPort && form.value.targetAddress.trim() && form.value.targetPort),
)
const previewText = computed(() => {
  const name = form.value.name.trim() || '—'
  const L = form.value.listenAddress.trim() || '?'
  const lp = form.value.listenPort || '?'
  const T = form.value.targetAddress.trim() || '?'
  const tp = form.value.targetPort || '?'
  const on = form.value.isEnable ? t('tools.portForward.enabled') : t('tools.portForward.disabled')
  return `${name} · ${typeShort.value}  ${L}:${lp} → ${T}:${tp}  · ${on}`
})

const close = () => {
  open.value = false
  loading.value = false
  editingId.value = ''
  errors.value = {}
  form.value = defaultForm()
}

const validate = () => {
  const e: Record<string, string> = {}
  if (!form.value.name.trim()) e.name = t('tools.portForward.ruleNameRequired')
  if (!form.value.listenAddress.trim()) e.listenAddress = t('tools.portForward.listenAddressRequired')
  if (!form.value.listenPort || form.value.listenPort < 1 || form.value.listenPort > 65535) {
    e.listenPort = t('tools.portForward.listenPortRequired')
  }
  if (!form.value.targetAddress.trim()) e.targetAddress = t('tools.portForward.targetAddressRequired')
  if (!form.value.targetPort || form.value.targetPort < 1 || form.value.targetPort > 65535) {
    e.targetPort = t('tools.portForward.targetPortRequired')
  }
  errors.value = e
  return Object.keys(e).length === 0
}

const confirm = async () => {
  if (!validate()) return
  loading.value = true
  try {
    let info: PortForwardInfo | undefined
    if (editingId.value) {
      const { data } = await updatePortForward({
        id: editingId.value,
        name: form.value.name || undefined,
        type: form.value.type as PortForwardType,
        listenAddress: form.value.listenAddress || undefined,
        listenPort: form.value.listenPort,
        targetAddress: form.value.targetAddress || undefined,
        targetPort: form.value.targetPort,
        isEnable: form.value.isEnable,
      })
      info = data?.info
    } else {
      const { data } = await createPortForward({
        name: form.value.name,
        type: form.value.type as PortForwardType,
        listenAddress: form.value.listenAddress,
        listenPort: form.value.listenPort!,
        targetAddress: form.value.targetAddress,
        targetPort: form.value.targetPort!,
        isEnable: form.value.isEnable,
      })
      info = data?.info
    }
    if (info?.error) ElMessage.error(info.error)
    else ElMessage.success(editingId.value ? t('tools.portForward.editSuccess') : t('tools.portForward.addSuccess'))
    emits('refresh', editingId.value ? 'update' : 'create')
    close()
  } finally {
    loading.value = false
  }
}

const showDialog = (payload?: PortForwardInfo) => {
  open.value = true
  errors.value = {}
  if (!payload?.id) {
    editingId.value = ''
    form.value = defaultForm()
    nextTick(() => nameRef.value?.focus())
    return
  }
  editingId.value = payload.id
  form.value = {
    name: payload.name || '',
    type: payload.type || 'PORT_FORWARD_TYPE_TCP',
    listenAddress: payload.listenAddress || '0.0.0.0',
    listenPort: payload.listenPort || undefined,
    targetAddress: payload.targetAddress || '127.0.0.1',
    targetPort: payload.targetPort || undefined,
    isEnable: payload.isEnable ?? true,
  }
  nextTick(() => nameRef.value?.focus())
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.pf-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* 协议分段 —— 不铺满全宽，像芯片组 */
.seg {
  display: inline-flex;
  padding: 2px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
}
.seg__btn {
  min-width: 64px;
  padding: 5px 14px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.seg__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}

/* 路由条：两端 endpoint + 中间箭头 */
.route {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 10px;
  align-items: start;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-lighter);
}
.route.is-error {
  border-color: color-mix(in srgb, var(--el-color-danger) 45%, transparent);
}
.route__side {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.route__side-head {
  display: flex;
  align-items: baseline;
  gap: 6px;
  flex-wrap: wrap;
}
.route__badge {
  display: inline-flex;
  padding: 1px 7px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--el-color-primary) 14%, transparent);
  color: var(--el-color-primary);
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.02em;
}
.route__badge--target {
  background: color-mix(in srgb, var(--el-color-success, #16a34a) 14%, transparent);
  color: var(--el-color-success, #16a34a);
}
.route__hint {
  font-size: 0.7rem;
  color: var(--el-text-color-placeholder);
}
.route__endpoint {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}
.route__host {
  flex: 1;
  min-width: 0;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', Consolas, monospace;
  font-size: 0.8125rem;
}
.route__port {
  width: 78px;
  flex-shrink: 0;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', Consolas, monospace;
  font-size: 0.8125rem;
  font-variant-numeric: tabular-nums;
}
.route__colon {
  color: var(--el-text-color-placeholder);
  font-weight: 600;
  flex-shrink: 0;
}
.route__arrow {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding-top: 22px;
  color: var(--el-text-color-secondary);
  user-select: none;
}
.route__arrow-label {
  font-size: 0.65rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: var(--el-text-color-placeholder);
}
.route__arrow-chev {
  font-size: 1.1rem;
  line-height: 1;
  color: var(--el-color-primary);
  font-weight: 700;
}
.route__arrow-line { display: none; }

.pf-form__enable {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
}
.pf-form__enable-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.pf-form__enable-title {
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
  font-weight: 500;
}
.pf-form__enable-desc {
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
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', Consolas, monospace;
  font-size: 0.78rem;
  color: var(--el-text-color-regular);
  word-break: break-all;
  line-height: 1.45;
}

/* 与 FormDialog 移动断点一致：窄屏改纵向全宽，禁止再走三列 grid */
@media (width <= 768px) {
  .route {
    display: flex !important;
    flex-direction: column !important;
    grid-template-columns: none !important;
    gap: 12px;
    align-items: stretch !important;
  }
  .route__side {
    width: 100% !important;
    max-width: none !important;
  }
  .route__arrow {
    flex-direction: row;
    justify-content: center;
    padding: 0;
    gap: 8px;
    width: 100%;
  }
  .route__arrow-chev {
    transform: rotate(90deg);
  }
  /* 地址 + 端口同一行占满宽度，端口略窄 */
  .route__endpoint {
    display: flex;
    flex-direction: row;
    align-items: center;
    width: 100%;
    gap: 6px;
  }
  .route__host {
    flex: 1 1 auto !important;
    width: auto !important;
    min-width: 0;
  }
  .route__port {
    flex: 0 0 96px !important;
    width: 96px !important;
  }
  .route__colon {
    display: inline;
  }
  .seg {
    width: 100%;
  }
  .seg__btn {
    flex: 1;
  }
}
</style>
