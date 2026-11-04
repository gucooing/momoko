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

/** 主机路径：绝对 / 相对路径、家目录或 Windows 盘符；否则视为命名卷 */
export const isHostPath = (value: string) => /^([/.~]|[a-zA-Z]:[\\/])/.test(value)
