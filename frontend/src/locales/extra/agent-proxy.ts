// Agent 的代理（只有 Linux 能配：官方账户的「代理」弹窗、第三方接口的「代理」这一步），
// 官方账户的「登录」（在悬浮终端里跑登录命令），悬浮终端里登录标签的名字。
export default {
  'zh-CN': {
    agent: {
      proxy: {
        address: '代理地址',
        placeholder: '留空不走代理，比如 socks5://127.0.0.1:1080',
        format: '支持 http:// 和 socks5://；有账号密码就写成 socks5://用户名:密码@主机:端口，已保存的密码显示为 ******。',
        invalid: '地址不对：要写成 http://主机:端口 或 socks5://主机:端口',
        test: '测试代理',
        testOk: '通了 · {target} · {ms} ms',
        testFail: '不通',
        unavailable: '现在用不了：{reason}',
        noteOff: '不走代理，直接联网。',
        noteAccount: '{name} 的检查、登录、会话、下载和更新都只经这个代理联网，其他连接一律断开。保存后，正在运行的会话会重启。',
        noteProvider: '用这个接口的会话、测试、拉模型、查余额都只经这个代理联网，其他连接一律断开。保存后，正在运行的会话会重启。',
        via: '经代理 {addr} 联网',
        tag: '代理',
      },
      accounts: {
        login: '登录',
        loginOf: '登录 {name}',
        loginSub: '点「登录」在服务器上打开 {agent} 的登录终端，登录完会自动检查',
        config: '配置',
        configTitle: '配置 {name}',
        proxy: '代理',
        models: '模型',
        modelsEmpty: '{name} 登录并检查后，这里会列出能用的模型，可以设最长上下文和最长输出。',
        saved: '已保存 {name} 的配置',
      },
      providers: { form: { step: { proxy: '代理' } } },
    },
    termDock: {
      login: '登录 · {name}',
    },
  },
  'zh-HK': {
    agent: {
      proxy: {
        address: '代理位址',
        placeholder: '留空不走代理，例如 socks5://127.0.0.1:1080',
        format: '支援 http:// 和 socks5://；有帳號密碼就寫成 socks5://使用者名稱:密碼@主機:連接埠，已儲存的密碼顯示為 ******。',
        invalid: '位址不對：要寫成 http://主機:連接埠 或 socks5://主機:連接埠',
        test: '測試代理',
        testOk: '通了 · {target} · {ms} ms',
        testFail: '不通',
        unavailable: '現在用不了：{reason}',
        noteOff: '不走代理，直接連網。',
        noteAccount: '{name} 的檢查、登入、工作階段、下載和更新都只經這個代理連網，其他連線一律斷開。儲存後，正在執行的工作階段會重新啟動。',
        noteProvider: '用這個介面的工作階段、測試、拉模型、查餘額都只經這個代理連網，其他連線一律斷開。儲存後，正在執行的工作階段會重新啟動。',
        via: '經代理 {addr} 連網',
        tag: '代理',
      },
      accounts: {
        login: '登入',
        loginOf: '登入 {name}',
        loginSub: '點「登入」在伺服器上打開 {agent} 的登入終端，登入完會自動檢查',
        config: '設定',
        configTitle: '設定 {name}',
        proxy: '代理',
        models: '模型',
        modelsEmpty: '{name} 登入並檢查後，這裡會列出能用的模型，可以設最長上下文和最長輸出。',
        saved: '已儲存 {name} 的設定',
      },
      providers: { form: { step: { proxy: '代理' } } },
    },
    termDock: {
      login: '登入 · {name}',
    },
  },
  'en-US': {
    agent: {
      proxy: {
        address: 'Proxy address',
        placeholder: 'Leave blank for no proxy, e.g. socks5://127.0.0.1:1080',
        format: 'http:// and socks5:// are supported; add credentials as socks5://user:password@host:port. A saved password shows as ******.',
        invalid: 'Invalid address: use http://host:port or socks5://host:port',
        test: 'Test proxy',
        testOk: 'Works · {target} · {ms} ms',
        testFail: 'Failed',
        unavailable: 'Not available right now: {reason}',
        noteOff: 'No proxy — connects directly.',
        noteAccount:
          "{name}'s checks, sign-in, sessions, downloads and updates reach the network only through this proxy; every other connection is cut off. Running sessions restart after saving.",
        noteProvider:
          'Sessions, tests, model lists and balance checks for this API reach the network only through this proxy; every other connection is cut off. Running sessions restart after saving.',
        via: 'Connects through {addr}',
        tag: 'Proxy',
      },
      accounts: {
        login: 'Sign in',
        loginOf: 'Sign in to {name}',
        loginSub: 'Click Sign in to open the {agent} sign-in terminal on the server; it checks again when you finish',
        config: 'Configure',
        configTitle: 'Configure {name}',
        proxy: 'Proxy',
        models: 'Models',
        modelsEmpty: 'Sign in and check {name} first; its models will be listed here so you can set context and output limits.',
        saved: 'Saved {name}',
      },
      providers: { form: { step: { proxy: 'Proxy' } } },
    },
    termDock: {
      login: 'Sign in · {name}',
    },
  },
}
