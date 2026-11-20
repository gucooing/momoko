// 顶栏「终端」悬浮窗（本机终端 / SSH）的文案，以及本机终端的节点权限名。
export default {
  'zh-CN': {
    termDock: {
      title: '终端',
      new: '新建终端',
      local: '本机终端',
      localN: '本机终端 {n}',
      sshHosts: 'SSH 主机',
      noHosts: '还没有 SSH 主机',
      manageHosts: '管理 SSH 主机',
      loading: '正在加载…',
      closeTab: '关闭 {name}',
      minimize: '最小化',
      close: '关闭（断开全部终端）',
      closed: '已断开',
      failed: '连接失败',
      reconnect: '重新连接',
    },
    node: {
      permission: {
        items: { shell: '本机终端（在节点上执行任意命令）' },
      },
    },
  },
  'zh-HK': {
    termDock: {
      title: '終端',
      new: '新建終端',
      local: '本機終端',
      localN: '本機終端 {n}',
      sshHosts: 'SSH 主機',
      noHosts: '還沒有 SSH 主機',
      manageHosts: '管理 SSH 主機',
      loading: '正在載入…',
      closeTab: '關閉 {name}',
      minimize: '最小化',
      close: '關閉（斷開全部終端）',
      closed: '已斷開',
      failed: '連接失敗',
      reconnect: '重新連接',
    },
    node: {
      permission: {
        items: { shell: '本機終端（在節點上執行任意命令）' },
      },
    },
  },
  'en-US': {
    termDock: {
      title: 'Terminal',
      new: 'New terminal',
      local: 'Local terminal',
      localN: 'Local terminal {n}',
      sshHosts: 'SSH hosts',
      noHosts: 'No SSH hosts yet',
      manageHosts: 'Manage SSH hosts',
      loading: 'Loading…',
      closeTab: 'Close {name}',
      minimize: 'Minimize',
      close: 'Close (disconnects all terminals)',
      closed: 'Disconnected',
      failed: 'Connection failed',
      reconnect: 'Reconnect',
    },
    node: {
      permission: {
        items: { shell: 'Local terminal (run any command on the node)' },
      },
    },
  },
}
