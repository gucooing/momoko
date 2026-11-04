// 工具组第三期（按原型逐页对齐）新增文案：端口转发 / 内网穿透 / SSH 主机 / Web 终端
export default {
  'zh-CN': {
    tools: {
      portForward: {
        autoStart: '开机自启',
        autoStartDesc: '服务重启后自动运行这条规则；关闭则重启后保持停用',
        stats: {
          activeConnTitle: '活动连接 · {n} 个',
          viewAll: '查看全部',
          collapse: '收起',
        },
      },
      tunnel: {
        statsDialog: {
          todayShort: '今日 {n}',
        },
      },
    },
    ssh: {
      form: {
        quickCommands: '快捷命令',
        quickCommandsPlaceholder: 'docker ps\nsystemctl status nginx',
        quickCommandsHint: '每行一条；在 Web 终端顶部「快捷命令」里一键发送到当前会话',
      },
      term: {
        webTitle: 'Web 终端',
        quickCommands: '快捷命令',
        quickCommandsOf: '{name} 的快捷命令',
        quickCommandsEmpty: '还没有快捷命令',
        editQuickCommands: '编辑快捷命令',
        mobileMenu: '快捷命令与会话',
        latency: '延迟 {ms}ms',
        sessionDuration: '会话时长 {time}',
        disconnectShort: '断开',
        reconnectShort: '重连',
        notConnected: '会话未连接，无法发送命令',
      },
    },
  },
  'zh-HK': {
    tools: {
      portForward: {
        autoStart: '開機自啟',
        autoStartDesc: '服務重啟後自動執行這條規則；關閉則重啟後保持停用',
        stats: {
          activeConnTitle: '活動連線 · {n} 個',
          viewAll: '查看全部',
          collapse: '收起',
        },
      },
      tunnel: {
        statsDialog: {
          todayShort: '今日 {n}',
        },
      },
    },
    ssh: {
      form: {
        quickCommands: '快捷指令',
        quickCommandsPlaceholder: 'docker ps\nsystemctl status nginx',
        quickCommandsHint: '每行一條；在 Web 終端頂部「快捷指令」一鍵發送到目前工作階段',
      },
      term: {
        webTitle: 'Web 終端',
        quickCommands: '快捷指令',
        quickCommandsOf: '{name} 的快捷指令',
        quickCommandsEmpty: '還沒有快捷指令',
        editQuickCommands: '編輯快捷指令',
        mobileMenu: '快捷指令與工作階段',
        latency: '延遲 {ms}ms',
        sessionDuration: '工作階段時長 {time}',
        disconnectShort: '斷開',
        reconnectShort: '重連',
        notConnected: '工作階段未連接，無法發送指令',
      },
    },
  },
  'en-US': {
    tools: {
      portForward: {
        autoStart: 'Start on boot',
        autoStartDesc: 'Run this rule automatically after the service restarts; when off it stays stopped',
        stats: {
          activeConnTitle: 'Active connections · {n}',
          viewAll: 'View all',
          collapse: 'Collapse',
        },
      },
      tunnel: {
        statsDialog: {
          todayShort: 'Today {n}',
        },
      },
    },
    ssh: {
      form: {
        quickCommands: 'Quick commands',
        quickCommandsPlaceholder: 'docker ps\nsystemctl status nginx',
        quickCommandsHint: 'One per line. Send them to the current session from Quick commands at the top of the web terminal',
      },
      term: {
        webTitle: 'Web terminal',
        quickCommands: 'Quick commands',
        quickCommandsOf: 'Quick commands for {name}',
        quickCommandsEmpty: 'No quick commands yet',
        editQuickCommands: 'Edit quick commands',
        mobileMenu: 'Quick commands and sessions',
        latency: 'Latency {ms}ms',
        sessionDuration: 'Session {time}',
        disconnectShort: 'Disconnect',
        reconnectShort: 'Reconnect',
        notConnected: 'The session is not connected, so the command was not sent',
      },
    },
  },
}
