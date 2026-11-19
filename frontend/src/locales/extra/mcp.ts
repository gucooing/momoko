// Momoko MCP（管理 → Momoko MCP）文案。凭证状态、最后使用沿用 access.key.* / node.keyx.*。
export default {
  'zh-CN': {
    mcp: {
      summary: '{tools} 个工具 · {credentials} 个凭证',
      connection: {
        title: '连接',
        endpoint: '地址',
        copyEndpoint: '复制地址',
        endpointCopied: '地址已复制',
        transport: '传输方式',
        transportValue: 'HTTP（Streamable HTTP）',
        header: '请求头',
      },
      credential: {
        title: '凭证',
        add: '新建凭证',
        manage: '管理 API Key',
        tools: '能调用的工具',
        noTools: '你的账号没有工具要的权限',
        configOf: '查看「{name}」的 MCP 配置',
        empty: '还没有授予工具权限的 API Key',
      },
      tool: {
        title: '工具',
        permission: '需要的权限',
        items: {
          install_skill: '从 Git 仓库或 .zip 下载链接安装技能；一个来源里的多个技能都会装上并启用',
          add_mcp_server: '在 Agent 管理里添加一个 MCP 服务器（stdio 或 HTTP）',
        },
      },
      config: {
        title: 'MCP 配置 · {name}',
        copy: '复制配置',
        copied: 'MCP 配置已复制',
      },
    },
  },
  'zh-HK': {
    mcp: {
      summary: '{tools} 個工具 · {credentials} 個憑證',
      connection: {
        title: '連接',
        endpoint: '地址',
        copyEndpoint: '複製地址',
        endpointCopied: '地址已複製',
        transport: '傳輸方式',
        transportValue: 'HTTP（Streamable HTTP）',
        header: '請求標頭',
      },
      credential: {
        title: '憑證',
        add: '新建憑證',
        manage: '管理 API Key',
        tools: '可調用的工具',
        noTools: '你的帳戶沒有工具需要的權限',
        configOf: '查看「{name}」的 MCP 配置',
        empty: '還沒有授予工具權限的 API Key',
      },
      tool: {
        title: '工具',
        permission: '需要的權限',
        items: {
          install_skill: '從 Git 倉庫或 .zip 下載連結安裝技能；一個來源裏的多個技能都會裝上並啟用',
          add_mcp_server: '在 Agent 管理裏添加一個 MCP 伺服器（stdio 或 HTTP）',
        },
      },
      config: {
        title: 'MCP 配置 · {name}',
        copy: '複製配置',
        copied: 'MCP 配置已複製',
      },
    },
  },
  'en-US': {
    mcp: {
      summary: '{tools} tools · {credentials} credentials',
      connection: {
        title: 'Connection',
        endpoint: 'URL',
        copyEndpoint: 'Copy URL',
        endpointCopied: 'URL copied',
        transport: 'Transport',
        transportValue: 'HTTP (Streamable HTTP)',
        header: 'Header',
      },
      credential: {
        title: 'Credentials',
        add: 'New credential',
        manage: 'Manage API keys',
        tools: 'Callable tools',
        noTools: 'Your account lacks the permissions these tools need',
        configOf: 'Show the MCP config of {name}',
        empty: 'No API key grants a tool permission yet',
      },
      tool: {
        title: 'Tools',
        permission: 'Required permission',
        items: {
          install_skill: 'Install skills from a Git repository or a .zip download link; every skill in the source is installed and enabled',
          add_mcp_server: 'Add an MCP server (stdio or HTTP) to Agent management',
        },
      },
      config: {
        title: 'MCP config · {name}',
        copy: 'Copy config',
        copied: 'MCP config copied',
      },
    },
  },
}
