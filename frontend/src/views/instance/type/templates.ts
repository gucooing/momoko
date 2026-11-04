// 应用类型：按名称/命令猜卡片图标；新建类型时可选的常用模板（Go · .NET · Deno · Bun 等）。

const ICON_RULES: [RegExp, string][] = [
  [/java|jar|minecraft|spring/i, 'i-lucide-coffee'],
  [/\.net|dotnet|c#/i, 'i-lucide-box'],
  [/deno/i, 'i-lucide-feather'],
  [/\bbun\b/i, 'i-lucide-croissant'],
  [/node|npm|pnpm|js\b/i, 'i-lucide-hexagon'],
  [/python|py\b|django|flask/i, 'i-lucide-code-xml'],
  [/\bgo\b|golang|rust|二进制|二進制|binary|exe|bin/i, 'i-lucide-cpu'],
  [/game|游戏|遊戲|palworld|terraria|steam/i, 'i-lucide-gamepad-2'],
  [/php|web|nginx|http/i, 'i-lucide-globe'],
  [/shell|bash|脚本|腳本|script|cmd/i, 'i-lucide-terminal'],
]

export const typeIconOf = (text: string) => ICON_RULES.find(([re]) => re.test(text))?.[1] || 'i-lucide-layers'

export interface TypeTemplate {
  key: string
  name: string
  /** 说明文案的 i18n 键（instance.ui.tplXxx） */
  descKey: string
  startCommand: string
  stopCommand: string
  quickCommands: string[]
}

export const TYPE_TEMPLATES: TypeTemplate[] = [
  { key: 'go', name: 'Go', descKey: 'instance.ui.tplGo', startCommand: './app', stopCommand: '', quickCommands: [] },
  { key: 'dotnet', name: '.NET', descKey: 'instance.ui.tplDotnet', startCommand: 'dotnet App.dll', stopCommand: '', quickCommands: [] },
  { key: 'deno', name: 'Deno', descKey: 'instance.ui.tplDeno', startCommand: 'deno run -A main.ts', stopCommand: '', quickCommands: [] },
  { key: 'bun', name: 'Bun', descKey: 'instance.ui.tplBun', startCommand: 'bun run index.ts', stopCommand: '', quickCommands: [] },
  {
    key: 'java',
    name: 'Java',
    descKey: 'instance.ui.tplJava',
    startCommand: 'java -Xms1G -Xmx4G -jar server.jar nogui',
    stopCommand: 'stop',
    quickCommands: ['list', 'save-all'],
  },
  { key: 'node', name: 'Node.js', descKey: 'instance.ui.tplNode', startCommand: 'node index.js', stopCommand: '', quickCommands: [] },
  { key: 'python', name: 'Python', descKey: 'instance.ui.tplPython', startCommand: 'python -u main.py', stopCommand: '', quickCommands: [] },
]
