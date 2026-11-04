<!-- frpc 客户端配置：贴纸头（放到哪台客户端）→ 范围分段（仅此隧道 / 该客户端全部 / 批量所选）+ 显示凭证
     → 深色代码块（行号 + 高亮，凭证默认打码）→ 启动命令 → 重新生成凭证 / 下载 / 复制。
     合并多条隧道时：登录凭证取第一条，每条 [[proxies]] 再带自己的 metadatas.credential（后端按 proxy 校验）。 -->
<template>
  <FormDialog v-model="visible" :title="title" :width="720" :show-footer="false">
    <div class="frpc">
      <div class="frpc-head">
        <span class="m-sticker m-sticker--a frpc-head__sticker"><component :is="menuStore.iconComponents['HOutline:KeyIcon']" /></span>
        <span class="frpc-head__text">
          <i18n-t v-if="clientHost" keypath="tools.tunnel.frpcDialog.fileHintClient" tag="span" scope="global">
            <template #host><b class="m-mono">{{ clientHost }}</b></template>
          </i18n-t>
          <span v-else>{{ t('tools.tunnel.frpc.fileHint') }}</span>
        </span>
      </div>

      <div class="frpc-opts">
        <div v-if="scopes.length > 1" class="frpc-seg" role="radiogroup" :aria-label="t('tools.tunnel.frpcDialog.scopeLabel')">
          <button
            v-for="s in scopes"
            :key="s.value"
            type="button"
            role="radio"
            class="frpc-seg__btn"
            :class="{ 'is-on': scope === s.value }"
            :aria-checked="scope === s.value"
            @click="scope = s.value"
          >{{ s.label }}</button>
        </div>
        <span class="frpc-spacer" />
        <label class="frpc-reveal">
          <input v-model="reveal" type="checkbox" />
          {{ t('tools.tunnel.frpcDialog.reveal') }}
          <span v-if="reveal || menuStore.isMobile" class="frpc-reveal__warn">{{ t('tools.tunnel.frpcDialog.revealHint') }}</span>
        </label>
      </div>

      <div class="frpc-code-wrap">
        <div class="frpc-code m-mono" role="region" :aria-label="t('tools.tunnel.frpcConfig')" tabindex="0">
          <div v-for="(line, i) in highlighted" :key="i" class="frpc-line">
            <span class="frpc-line__no">{{ i + 1 }}</span>
            <span class="frpc-line__text">
              <template v-if="line.kind === 'comment'"><span class="c-com">{{ line.text }}</span></template>
              <template v-else-if="line.kind === 'section'"><span class="c-sec">{{ line.text }}</span></template>
              <template v-else-if="line.kind === 'kv'"><span class="c-key">{{ line.key }}</span><span class="c-eq"> = </span><span :class="line.str ? 'c-str' : 'c-num'">{{ line.value }}</span></template>
              <template v-else>{{ line.text || ' ' }}</template>
            </span>
          </div>
        </div>
        <span class="frpc-code__file m-mono">frpc.toml</span>
      </div>

      <div class="frpc-cmd">
        <component :is="menuStore.iconComponents['HOutline:CommandLineIcon']" class="frpc-cmd__ico" />
        <span class="frpc-cmd__text m-mono">{{ START_CMD }}</span>
        <span class="frpc-cmd__hint">{{ t('tools.tunnel.frpc.windowsHint') }}</span>
        <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('tools.tunnel.frpc.copyCmd')" :box="menuStore.isMobile ? 44 : 40" :size="18" @click="copyCmd" />
      </div>

      <div class="frpc-foot">
        <button v-if="scope === 'one' && row" type="button" class="frpc-reset" :disabled="resetting" @click="confirmReset">
          <component :is="menuStore.iconComponents['HOutline:ArrowPathIcon']" />{{ t('tools.tunnel.frpcDialog.resetToken') }}
        </button>
        <span class="frpc-spacer" />
        <UButton color="neutral" variant="outline" size="lg" icon="i-lucide-download" @click="download">
          {{ menuStore.isMobile ? t('tools.tunnel.frpcDialog.download') : t('tools.tunnel.frpc.downloadFile') }}
        </UButton>
        <UButton color="primary" size="lg" :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'" @click="copy">
          {{ copied ? t('tools.tunnel.frpcDialog.copied') : t('tools.tunnel.frpc.copyConfig') }}
        </UButton>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { listTunnels, resetTunnelCredential } from '@/api/tunnel'
import { TunnelType, type TunnelInfo, type FrpsConfig } from '@/types/v1/tunnel'
import { downloadText } from '@/views/tools/components/format'
import { Dialog } from '@/utils/dialog'

type Scope = 'one' | 'client' | 'selected'
type Line =
  | { kind: 'comment' | 'section' | 'blank'; text: string }
  | { kind: 'kv'; key: string; value: string; str: boolean; text: string }

const START_CMD = './frpc -c ./frpc.toml'
const MASK = '••••••••••••••••'

const props = defineProps<{
  modelValue: boolean
  row: TunnelInfo | null
  /** 批量合并时传入所选隧道；单条打开时留空（自动查找同一客户端上的其它隧道） */
  rows?: TunnelInfo[]
  frps: FrpsConfig | null
}>()

const emit = defineEmits<{ 'update:modelValue': [value: boolean]; 'credential-reset': [info: TunnelInfo] }>()

const { t } = useI18n()
const menuStore = useMenuStore()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})

const scope = ref<Scope>('one')
const reveal = ref(false)
const copied = ref(false)
const resetting = ref(false)
const peers = ref<TunnelInfo[]>([])
const current = ref<TunnelInfo | null>(null)

const isBatch = computed(() => (props.rows?.length || 0) > 1)
const clientKey = (r: TunnelInfo | null | undefined) => (r?.client ? `${r.client.hostname}|${r.client.address}` : '')
const clientHost = computed(() => {
  const list = included.value
  const hosts = new Set(list.map((r) => r.client?.hostname).filter(Boolean))
  return hosts.size === 1 ? [...hosts][0]! : ''
})

const scopes = computed(() => {
  if (isBatch.value) return [{ value: 'selected' as Scope, label: t('tools.tunnel.frpcDialog.scopeSelected', { n: props.rows?.length || 0 }) }]
  const out = [{ value: 'one' as Scope, label: t('tools.tunnel.frpcDialog.scopeOne') }]
  if (peers.value.length > 1) out.push({ value: 'client', label: t('tools.tunnel.frpcDialog.scopeClient', { n: peers.value.length }) })
  return out
})

const included = computed<TunnelInfo[]>(() => {
  if (isBatch.value) return props.rows || []
  if (scope.value === 'client' && peers.value.length > 1) return peers.value
  return current.value ? [current.value] : []
})

const title = computed(() => {
  const list = included.value
  if (list.length > 1) return t('tools.tunnel.frpcDialog.titleMerged', { n: list.length })
  return list[0] ? t('tools.tunnel.frpc.titleWithName', { name: list[0].name }) : t('tools.tunnel.frpcConfig')
})

const typeString = (type: TunnelType) => type.replace('TUNNEL_TYPE_', '').toLowerCase()

const buildLines = (masked: boolean): string[] => {
  const list = included.value
  const first = list[0]
  if (!first) return []
  const cred = (v: string) => (masked ? MASK : v)
  const frps = props.frps
  let serverAddr = frps?.serverAddr || frps?.bindAddr || ''
  if (!serverAddr || serverAddr === '0.0.0.0') serverAddr = 'YOUR_SERVER_ADDR'
  const serverPort = frps?.bindPort || 7000
  const merged = list.length > 1

  const lines = [t('tools.tunnel.frpc.commentHeader')]
  if (merged) {
    if (clientHost.value) lines.push(t('tools.tunnel.frpcDialog.commentClient', { host: clientHost.value }))
    lines.push(t('tools.tunnel.frpcDialog.commentMerged', { n: list.length }))
  } else {
    lines.push(t('tools.tunnel.frpc.commentTunnel', { name: first.name }))
  }
  lines.push(`serverAddr = "${serverAddr}"`, `serverPort = ${serverPort}`)
  if (frps?.tlsForce) lines.push('', t('tools.tunnel.frpcDialog.commentTls'), 'transport.tls.enable = true')
  lines.push(
    '',
    t('tools.tunnel.frpc.commentCredential'),
    `metadatas.name = "${first.name}"`,
    `metadatas.credential = "${cred(first.credential)}"`,
  )

  for (const row of list) {
    const type = typeString(row.type)
    lines.push(
      '',
      '[[proxies]]',
      `name = "${row.name}"`,
      `type = "${type}"`,
      `localIP = "${row.localIp || '127.0.0.1'}"`,
      `localPort = ${row.localPort || 0}`,
    )
    if (type === 'tcp' || type === 'udp' || type === 'tcpmux') {
      lines.push(`remotePort = ${row.remotePort || 0}`)
    } else if (type === 'http' || type === 'https') {
      const domains = (row.customDomains || '').split(',').map((d) => d.trim()).filter(Boolean)
      if (domains.length) lines.push(`customDomains = [${domains.map((d) => `"${d}"`).join(', ')}]`)
      if (row.subdomain) lines.push(`subdomain = "${row.subdomain}"`)
    } else if (type === 'stcp' || type === 'xtcp') {
      lines.push(`secretKey = "${cred(row.credential)}"`)
    }
    if (merged) lines.push(`metadatas.credential = "${cred(row.credential)}"`)
    if (row.maxBandwidth) {
      lines.push(t('tools.tunnel.frpc.commentBandwidth'), `transport.bandwidthLimit = "${row.maxBandwidth}"`)
    }
  }
  return lines
}

const content = computed(() => buildLines(false).join('\n') + '\n')

const highlighted = computed<Line[]>(() =>
  buildLines(!reveal.value).map((text): Line => {
    if (!text) return { kind: 'blank', text: '' }
    if (text.startsWith('#')) return { kind: 'comment', text }
    if (text.startsWith('[')) return { kind: 'section', text }
    const idx = text.indexOf(' = ')
    if (idx < 0) return { kind: 'blank', text }
    const value = text.slice(idx + 3)
    return { kind: 'kv', key: text.slice(0, idx), value, str: value.startsWith('"') || value.startsWith('['), text }
  }),
)

const loadPeers = async () => {
  peers.value = []
  const row = current.value
  if (!row || isBatch.value || !row.client) return
  try {
    const { data } = await listTunnels({ page: 1, pageSize: 0 })
    const key = clientKey(row)
    const same = (data?.infos || []).filter((r) => clientKey(r) === key)
    // 当前隧道放第一位：登录凭证用它
    peers.value = [row, ...same.filter((r) => r.id !== row.id)]
  } catch {
    /* 拿不到同客户端隧道时只提供单条 */
  }
}

let copiedTimer: ReturnType<typeof setTimeout> | null = null
const copy = async () => {
  try {
    await navigator.clipboard.writeText(content.value)
    copied.value = true
    if (copiedTimer) clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => (copied.value = false), 2000)
    feedback.success(t('tools.tunnel.frpc.copied'))
  } catch {
    feedback.error(t('tools.tunnel.frpc.copyFailed'))
  }
}
const copyCmd = async () => {
  try {
    await navigator.clipboard.writeText(START_CMD)
    feedback.success(t('tools.tunnel.frpc.copied'))
  } catch {
    feedback.error(t('tools.tunnel.frpc.copyFailed'))
  }
}
const download = () => {
  const list = included.value
  const name = list.length > 1 ? clientHost.value || 'merged' : list[0]?.name || 'tunnel'
  downloadText(`frpc-${name}.toml`, content.value)
}

const confirmReset = () => {
  const row = current.value
  if (!row) return
  Dialog.confirm({
    title: t('tools.tunnel.resetCredential'),
    content: t('tools.tunnel.resetCredentialConfirm'),
    confirmText: t('tools.tunnel.resetCredentialOk'),
    cancelText: t('tools.tunnel.cancel'),
    onConfirm: async () => {
      resetting.value = true
      try {
        const { data } = await resetTunnelCredential({ id: row.id })
        if (data?.info) {
          current.value = data.info
          peers.value = peers.value.map((p) => (p.id === data.info!.id ? data.info! : p))
          emit('credential-reset', data.info)
        }
        feedback.success(t('tools.tunnel.resetCredentialDone'))
      } finally {
        resetting.value = false
      }
    },
  })
}

watch(visible, (open) => {
  if (!open) return
  current.value = props.row
  reveal.value = false
  copied.value = false
  scope.value = isBatch.value ? 'selected' : 'one'
  void loadPeers()
})
watch(() => props.row, (r) => { if (visible.value && r && r.id === current.value?.id) current.value = r })
</script>

<style scoped lang="scss">
.frpc { display: flex; flex-direction: column; gap: 14px; }
.frpc-head { display: flex; align-items: center; gap: 12px; min-width: 0; }
.frpc-head__sticker {
  width: 48px;
  height: 48px;
  flex: none;
  border-radius: 16px;
  transform: rotate(-5deg);
  :deep(svg) { width: 24px; height: 24px; }
}
.frpc-head__text {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  b { color: var(--el-text-color-primary); font-weight: 700; }
}
.frpc-opts { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; }
.frpc-spacer { flex: 1; }
.frpc-seg {
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  max-width: 100%;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.frpc-seg__btn {
  min-height: 34px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.frpc-seg__btn.is-on { background: var(--el-bg-color); color: var(--el-text-color-primary); box-shadow: var(--app-shadow-sm); }
.frpc-reveal {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  user-select: none;
  input { width: 18px; height: 18px; margin: 0; accent-color: var(--m-p); }
}
.frpc-reveal__warn { font-size: 0.75rem; font-weight: 600; color: var(--m-warn); }

/* 代码块：固定终端色 */
.frpc-code-wrap { position: relative; }
.frpc-code {
  max-height: 340px;
  overflow: auto;
  padding: 14px 0;
  border-radius: var(--app-radius-lg);
  background: #0f0b14;
  color: #e8e3ee;
  font-size: 0.78rem;
  line-height: 1.75;
  scrollbar-width: none;
}
.frpc-code::-webkit-scrollbar { display: none; }
.frpc-code:focus-visible { outline: 3px solid var(--m-a2); outline-offset: 2px; }
.frpc-line { display: flex; white-space: pre; padding-right: 16px; }
.frpc-line__no { width: 44px; flex: none; padding-right: 14px; text-align: right; color: #8a8197; user-select: none; }
.frpc-line__text { min-width: 0; }
.c-com { color: #9c93a8; }
.c-sec { color: #e8e3ee; font-weight: 800; }
.c-key { color: #e8e3ee; }
.c-eq { color: #9c93a8; }
.c-str { color: #ffd25a; }
.c-num { color: #79c0ff; }
.frpc-code__file {
  position: absolute;
  top: 10px;
  right: 12px;
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  background: #2a2233;
  color: #e8e3ee;
  font-size: 0.6875rem;
  font-weight: 700;
}
.frpc-cmd {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 4px 4px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.frpc-cmd__ico { width: 16px; height: 16px; flex: none; color: var(--el-text-color-secondary); }
.frpc-cmd__text { flex: 1; min-width: 0; font-size: 0.8125rem; font-weight: 600; color: var(--el-text-color-primary); }
.frpc-cmd__hint { font-size: 0.75rem; color: var(--el-text-color-secondary); white-space: nowrap; }
.frpc-foot { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; }
.frpc-reset {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 8px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--m-warn);
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
  :deep(svg) { width: 16px; height: 16px; }
}
.frpc-reset:hover { background: var(--m-warnSoft); }
.frpc-reset:disabled { opacity: 0.6; cursor: default; }

@media (width <= 768px) {
  .frpc-seg { width: 100%; }
  .frpc-seg__btn { flex: 1; min-height: 44px; padding: 0 8px; }
  .frpc-spacer { display: none; }
  .frpc-reveal { width: 100%; }
  .frpc-reveal__warn { margin-left: auto; }
  .frpc-code { max-height: 290px; font-size: 0.72rem; }
  .frpc-line__no { width: 34px; padding-right: 10px; }
  .frpc-cmd__hint { display: none; }
  .frpc-foot > :deep(button:not(.frpc-reset)) { flex: 1; justify-content: center; min-height: 48px; }
  .frpc-reset { width: 100%; justify-content: center; }
}
</style>
