<template>
  <BaseDialog v-model="visible" :title="title" width="640" :show-footer="false">
    <p class="frpc-intro">{{ t('tools.tunnel.frpc.intro') }}</p>
    <pre class="frpc-content">{{ content }}</pre>
    <div class="frpc-actions">
      <el-button :icon="menuStore.iconComponents['HOutline:ClipboardDocumentIcon']" @click="copy">
        {{ t('tools.tunnel.frpc.copy') }}
      </el-button>
      <el-button type="primary" :icon="menuStore.iconComponents['HOutline:ArrowDownTrayIcon']" @click="download">
        {{ t('tools.tunnel.frpc.download') }}
      </el-button>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { TunnelType, type TunnelInfo, type FrpsConfig } from '@/types/v1/tunnel'

const props = defineProps<{
  modelValue: boolean
  row: TunnelInfo | null
  frps: FrpsConfig | null
}>()

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const menuStore = useMenuStore()
const { t } = useI18n()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})

const title = computed(() =>
  props.row ? t('tools.tunnel.frpc.titleWithName', { name: props.row.name }) : t('tools.tunnel.frpcConfig'),
)

const typeString = (type: TunnelType) => type.replace('TUNNEL_TYPE_', '').toLowerCase()

// frpc.toml 完全由前端依据隧道信息与 frps 配置生成。
const content = computed(() => {
  const row = props.row
  if (!row) return ''
  const frps = props.frps

  let serverAddr = frps?.serverAddr || frps?.bindAddr || ''
  if (!serverAddr || serverAddr === '0.0.0.0') serverAddr = 'YOUR_SERVER_ADDR'
  const serverPort = frps?.bindPort || 7000
  const type = typeString(row.type)

  const lines = [
    '# momoko 内网穿透 frpc 配置',
    `# 隧道：${row.name}`,
    `serverAddr = "${serverAddr}"`,
    `serverPort = ${serverPort}`,
    '',
    '# 每隧道独立凭证，momoko 鉴权插件据此放行',
    `metadatas.name = "${row.name}"`,
    `metadatas.credential = "${row.credential}"`,
    '',
    '[[proxies]]',
    `name = "${row.name}"`,
    `type = "${type}"`,
    `localIP = "${row.localIp || '127.0.0.1'}"`,
    `localPort = ${row.localPort || 0}`,
  ]

  if (type === 'tcp' || type === 'udp' || type === 'tcpmux') {
    lines.push(`remotePort = ${row.remotePort || 0}`)
  } else if (type === 'http' || type === 'https') {
    const domains = (row.customDomains || '')
      .split(',')
      .map((d) => d.trim())
      .filter(Boolean)
    if (domains.length) {
      lines.push(`customDomains = [${domains.map((d) => `"${d}"`).join(', ')}]`)
    }
    if (row.subdomain) lines.push(`subdomain = "${row.subdomain}"`)
  } else if (type === 'stcp' || type === 'xtcp') {
    lines.push(`secretKey = "${row.credential}"`)
  }

  return lines.join('\n') + '\n'
})

const copy = async () => {
  try {
    await navigator.clipboard.writeText(content.value)
    ElMessage.success(t('tools.tunnel.frpc.copied'))
  } catch {
    // 某些非安全上下文不支持 clipboard，回退到选中提示
    ElMessage.error(t('tools.tunnel.frpc.copy'))
  }
}

const download = () => {
  const blob = new Blob([content.value], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `frpc-${props.row?.name || 'tunnel'}.toml`
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped lang="scss">
.frpc-intro {
  margin: 0 0 0.6rem;
  font-size: 0.85rem;
  color: var(--el-text-color-secondary);
}

.frpc-content {
  margin: 0;
  padding: 0.8rem 1rem;
  max-height: 16rem;
  overflow: auto;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', monospace;
  font-size: 0.82rem;
  line-height: 1.5;
  white-space: pre;
}

.frpc-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 0.8rem;
}
</style>
