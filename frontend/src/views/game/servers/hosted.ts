// 托管接入表单的草稿与转换（接入页与「改为托管」共用）。
import type { DetectGameServerDirResponse, GameServerHostedSpec } from '@/types/v1/game_server'
import type { MgpKey } from './mgpKey'

export interface HostedDraft {
  installDir: string
  configFile: string
  configTemplate: string
  listens: Record<string, string>
  resourceServer: boolean
  resourceListen: string
  resourceOrigin: string
  resourceRoot: string
  /** 点「生成」得到的新 MGP 密钥（前端生成，Momoko 写入服务端配置）；为空时沿用配置里已有的密钥 */
  key: MgpKey | null
  autoStart: boolean
  autoRestart: boolean
  start: boolean
}

export const emptyHostedDraft = (): HostedDraft => ({
  installDir: '',
  configFile: '',
  configTemplate: '',
  listens: { grpc: '', http: '', mgp: '' },
  resourceServer: false,
  resourceListen: '',
  resourceOrigin: '',
  resourceRoot: '',
  key: null,
  autoStart: true,
  autoRestart: true,
  start: true,
})

/** 用识别结果回填还没填的字段（已填的保持用户输入）。 */
export const applyDetection = (d: HostedDraft, r: DetectGameServerDirResponse, first: boolean) => {
  if (r.installDir) d.installDir = r.installDir
  d.configFile = r.configFile
  d.configTemplate = r.configTemplate
  for (const k of ['grpc', 'http', 'mgp']) {
    if (!d.listens[k] && r.listens?.[k]) d.listens[k] = r.listens[k]!
  }
  if (!d.resourceListen && r.listens?.resource) d.resourceListen = r.listens.resource
  if (!d.resourceOrigin && r.resourceOrigin) d.resourceOrigin = r.resourceOrigin
  if (!d.resourceRoot && r.resourceRoot) d.resourceRoot = r.resourceRoot
  if (first) {
    d.resourceServer = r.hasResourceServer
  }
  if (!r.hasResourceServer) d.resourceServer = false
}

export const toHostedSpec = (d: HostedDraft): GameServerHostedSpec => ({
  installDir: d.installDir.trim(),
  configFile: d.configFile,
  configTemplate: d.configTemplate,
  listens: Object.fromEntries(Object.entries(d.listens).map(([k, v]) => [k, v.trim()])),
  resourceServer: d.resourceServer,
  resourceListen: d.resourceListen.trim(),
  resourceOrigin: d.resourceOrigin.trim(),
  resourceRoot: d.resourceRoot.trim(),
  generateKey: !!d.key,
  keyId: d.key?.keyId || '',
  secret: d.key?.secret || '',
  autoStart: d.autoStart,
  autoRestart: d.autoRestart,
  start: d.start,
})

export const detectListens = (d: HostedDraft): Record<string, string> => {
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(d.listens)) if (v.trim()) out[k] = v.trim()
  if (d.resourceListen.trim()) out.resource = d.resourceListen.trim()
  return out
}
