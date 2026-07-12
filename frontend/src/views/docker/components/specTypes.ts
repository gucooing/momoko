// 容器 端口/挂载/环境变量 结构化行类型（编辑器组件 + 容器页共用）
export interface PortRow {
  hostIp: string
  hostPort: string
  containerPort: string
  protocol: 'tcp' | 'udp'
}
export interface MountRow {
  type: string
  source: string
  target: string
  readOnly: boolean
}
export interface EnvRow {
  key: string
  value: string
}
