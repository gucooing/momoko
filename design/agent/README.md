# Momoko · Agent（Code 与 Agent 管理）原型

通过 ACP（Agent Client Protocol）在 Momoko 网页里控制服务器上的 Claude Code、Codex 等 Agent 写代码，并管理会话、工作区、模型供应商、MCP、技能、运行时和通知。

- `*.dc.html`：页面原型，格式和 `../project/` 一样（自定义模板语法的 HTML 源码，不能直接用浏览器打开；直接读文本：模板部分是布局，`renderVals()` 里的数据和处理函数是页面要展示的信息和每个控件的行为）。
  - `Main.dc.html`：Code · 会话进行中（PC）。注意这里的 `Main` 是 Code 页，不是配色总览。
  - `AgentNewPC.dc.html`：Code · 新会话（PC）。
  - `AgentMobile.dc.html`：Code · 手机。
  - `AgentSessionsPC.dc.html`、`AgentWorkspacesPC.dc.html`、`AgentProvidersPC.dc.html`、`AgentMcpPC.dc.html`、`AgentSkillsPC.dc.html`、`AgentRuntimesPC.dc.html`、`AgentNotifyPC.dc.html`：「Agent 管理」的七个标签页（会话、工作区、模型供应商、MCP、技能、运行时、通知）。弹窗都画在各自的页面里，点对应按钮打开。
- `canvas.json`：画布上的画板排布与标题。
- `IMPL_AGENT.md`：给本机 Claude Code 的实现规范，包含用户反馈过的硬性要求、架构、ACP 能力对照和需要实测的清单。

原型里的会话、路径、账号、数字都是示例数据；`[版本号]`、`[模型名]`、`[授权链接]` 这类方括号是占位，实现时用真实值。侧边栏里指向 Momoko 其它页面的链接是 `#`，只有 Code 和 Agent 管理之间可以互相跳转。管理页还没有手机版原型。

原型来源：claude.ai 上的 Design 画布「Momoko Agent 原型」，导出于 2026-11-19。
