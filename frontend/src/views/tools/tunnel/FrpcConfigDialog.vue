<!-- frpc 客户端配置预览（重写）：FormDialog + 代码块 + 复制/下载。内容由前端根据隧道+frps 生成。 -->
<template>
  <FormDialog v-model="visible" :title="title" :width="640" :show-footer="false">
    <p class="frpc-intro">{{ t('tools.tunnel.frpc.intro') }}</p>
    <pre class="frpc-content">{{ content }}</pre>
    <div class="frpc-actions">
      <UButton color="neutral" variant="soft" icon="i-lucide-copy" @click="copy">
        {{ t('tools.tunnel.frpc.copy') }}
      </UButton>
      <UButton color="primary" icon="i-lucide-download" @click="download">
        {{ t('tools.tunnel.frpc.download') }}
      </UButton>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { TunnelType, type TunnelInfo, type FrpsConfig } from '@/types/v1/tunnel'

const props = defineProps<{
  modelValue: boolean
  row: TunnelInfo | null
  frps: FrpsConfig | null
}>()

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})

const title = computed(() =>
  props.row ? t('tools.tunnel.frpc.titleWithName', { name: props.row.name }) : t('tools.tunnel.frpcConfig'),
)

const typeString = (type: TunnelType) => type.replace('TUNNEL_TYPE_', '').toLowerCase()

const content = computed(() => {
  const row = props.row
  if (!row) return ''
  const frps = props.frps

  let serverAddr = frps?.serverAddr || frps?.bindAddr || ''
  if (!serverAddr || serverAddr === '0.0.0.0') serverAddr = 'YOUR_SERVER_ADDR'
  const serverPort = frps?.bindPort || 7000
  const type = typeString(row.type)

  const lines = [
    t('tools.tunnel.frpc.commentHeader'),
    t('tools.tunnel.frpc.commentTunnel', { name: row.name }),
    `serverAddr = "${serverAddr}"`,
    `serverPort = ${serverPort}`,
    '',
    t('tools.tunnel.frpc.commentCredential'),
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

  if (row.maxBandwidth) {
    lines.push('', t('tools.tunnel.frpc.commentBandwidth'))
    lines.push(`transport.bandwidthLimit = "${row.maxBandwidth}"`)
  }

  return lines.join('\n') + '\n'
})

const copy = async () => {
  try {
    await navigator.clipboard.writeText(content.value)
    feedback.success(t('tools.tunnel.frpc.copied'))
  } catch {
    feedback.error(t('tools.tunnel.frpc.copyFailed'))
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
  margin: 0 0 10px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.frpc-content {
  margin: 0;
  padding: 10px 12px;
  max-height: 16rem;
  overflow: auto;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', Consolas, monospace;
  font-size: 0.78rem;
  line-height: 1.5;
  white-space: pre;
  color: var(--el-text-color-regular);
}
.frpc-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}
</style>
