// 本机终端：在 Momoko 所在机器（远程节点上下文里是该节点）开一个 shell，权限 shell:use。
// 与后端 internal/biz/openssh.go 的 ShellWSPath 一致。
export const SHELL_WS_PATH = '/api/v1/shell/ws'

export const SHELL_PERMISSION = 'shell:use'
