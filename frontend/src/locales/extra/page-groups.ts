// 标签页分组（config/pageGroups.ts）的分组名与标签名，与原型的页内标签一致。
export default {
  'zh-CN': {
    pageGroup: {
      noAccessTitle: '没有可查看的内容',
      noAccessDesc: '当前账号在这里没有任何可访问的标签，请联系管理员分配权限。',
      file: { label: '文件', index: '文件管理', share: '分享', source: '文件源' },
      docker: { label: 'Docker', container: '容器', image: '镜像', network: '网络', config: '连接配置' },
      node: { label: '节点与 API Key', list: '节点', key: 'API Key' },
      access: { label: '用户与权限', user: '用户', role: '角色', menu: '菜单' },
      logs: { label: '任务与日志', task: '后台任务', operation: '操作记录' },
      settings: { label: '系统设置', general: '常规 · 登录 · 邮件 · 储存', oidc: 'OIDC 身份提供方' },
      iqcheck: { label: '智商检测', site: '检测站点', prompt: '提示词' },
    },
  },
  'zh-HK': {
    pageGroup: {
      noAccessTitle: '沒有可查看的內容',
      noAccessDesc: '當前帳號在這裡沒有任何可訪問的標籤，請聯絡管理員分配權限。',
      file: { label: '文件', index: '文件管理', share: '分享', source: '文件源' },
      docker: { label: 'Docker', container: '容器', image: '鏡像', network: '網絡', config: '連接配置' },
      node: { label: '節點與 API Key', list: '節點', key: 'API Key' },
      access: { label: '用戶與權限', user: '用戶', role: '角色', menu: '選單' },
      logs: { label: '任務與日誌', task: '後台任務', operation: '操作記錄' },
      settings: { label: '系統設定', general: '常規 · 登入 · 郵件 · 儲存', oidc: 'OIDC 身份提供方' },
      iqcheck: { label: '智商檢測', site: '檢測站點', prompt: '提示詞' },
    },
  },
  'en-US': {
    pageGroup: {
      noAccessTitle: 'Nothing to show here',
      noAccessDesc: 'Your account has no access to any tab on this page. Ask an administrator for permission.',
      file: { label: 'Files', index: 'File manager', share: 'Shares', source: 'File sources' },
      docker: { label: 'Docker', container: 'Containers', image: 'Images', network: 'Networks', config: 'Connection' },
      node: { label: 'Nodes and API keys', list: 'Nodes', key: 'API keys' },
      access: { label: 'Users and access', user: 'Users', role: 'Roles', menu: 'Menus' },
      logs: { label: 'Tasks and logs', task: 'Background tasks', operation: 'Activity log' },
      settings: { label: 'Settings', general: 'General · Sign-in · Email · Storage', oidc: 'OIDC provider' },
      iqcheck: { label: 'IQ check', site: 'Sites', prompt: 'Prompts' },
    },
  },
}
