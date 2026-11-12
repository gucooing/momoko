# Momoko · 游戏服务端管理与 GM（原型）

- `*.dc.html`：页面原型，格式和 `../project/` 一样（自定义模板语法的 HTML 源码，不能直接用浏览器打开；直接读文本：模板部分是布局，`renderVals()` / `pageVals()` 里的数据是页面要展示的信息）。
  - `Main.dc.html`：架构总览（托管 / 仅 GM / 用户通行证独立 server）。
  - `SpecPC.dc.html`：接口与数据的权威说明（MGP v1、数据表、权限、学マス GM 目录、游戏素材来源）。
  - 其余页面的清单、路由和要点见 `IMPL_GAME.md`。
- `canvas.json`：画布上的画板排布与标题。
- `momoko-game.css`：原型共用样式，包含 Momoko 的 6 套配色和学マス 游戏皮肤（`.gk-light / .gk-dark`，令牌 `--g --gOn --gInk --gSoft --gLine --g2* --g3*`）。只用来读懂原型，实现时用项目自己的令牌和组件。
- `assets/*.webp`：原型用到的 47 张学マス 素材（从客户端资源导出）。实现时不要引用这个目录，按 `IMPL_GAME.md` 第 7 节用脚本导出到前端。
- `tools/extract_gakumas_assets.py`：素材导出脚本（已在 48 个资源上实测可用）。
- `IMPL_GAME.md`：给本机 Claude Code 的实现规范。

管理端页面侧边栏里指向非游戏页面的链接已改成 `../project/XxxPC.dc.html`。

原型来源：claude.ai 上的 Design 画布，导出于 2026-11-11。
