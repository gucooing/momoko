// xterm.js 封装：创建终端 + fit 自适应 + ResizeObserver + 手动主题（明暗两套 ITheme）。
// 实例控制台 / SSH 终端共用。主题由 useTerminalTheme 的 ref 驱动，切换即热更新 canvas。
import { onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { Terminal, type ITheme } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import type { TerminalThemeMode } from './useTerminalTheme'

const FONT_FAMILY =
  "'Cascadia Code', 'JetBrains Mono', 'Fira Code', 'Source Code Pro', Consolas, Menlo, Monaco, monospace"

// 暗色（默认）：深板岩底 + 薄荷光标，ANSI 取 GitHub-dark 近似
const DARK_THEME: ITheme = {
  background: '#0d1117',
  foreground: '#c9d1d9',
  cursor: '#2dd4bf',
  cursorAccent: '#0d1117',
  selectionBackground: 'rgba(45, 212, 191, 0.28)',
  black: '#484f58',
  red: '#ff7b72',
  green: '#3fb950',
  yellow: '#d29922',
  blue: '#58a6ff',
  magenta: '#bc8cff',
  cyan: '#39c5cf',
  white: '#b1bac4',
  brightBlack: '#6e7681',
  brightRed: '#ffa198',
  brightGreen: '#56d364',
  brightYellow: '#e3b341',
  brightBlue: '#79c0ff',
  brightMagenta: '#d2a8ff',
  brightCyan: '#56d4dd',
  brightWhite: '#f0f6fc',
}

// 浅色：冷灰底 + 薄荷光标，ANSI 取 GitHub-light 近似
const LIGHT_THEME: ITheme = {
  background: '#f8fafc',
  foreground: '#1f2933',
  cursor: '#0d9488',
  cursorAccent: '#f8fafc',
  selectionBackground: 'rgba(13, 148, 136, 0.20)',
  black: '#24292f',
  red: '#cf222e',
  green: '#116329',
  yellow: '#7d4e00',
  blue: '#0969da',
  magenta: '#8250df',
  cyan: '#1b7c83',
  white: '#6e7781',
  brightBlack: '#57606a',
  brightRed: '#a40e26',
  brightGreen: '#1a7f37',
  brightYellow: '#633c01',
  brightBlue: '#218bff',
  brightMagenta: '#a475f9',
  brightCyan: '#3192aa',
  brightWhite: '#8c959f',
}

export const terminalThemeFor = (mode: TerminalThemeMode): ITheme =>
  mode === 'light' ? LIGHT_THEME : DARK_THEME

interface UseTerminalXOptions {
  /** 手动主题 ref（来自 useTerminalTheme） */
  theme: Ref<TerminalThemeMode>
  /** 键盘输入回调：原始键盘流原样发往后端（SSH / 实例 PTY 同构） */
  onData?: (data: string) => void
  /** 二进制输入回调：legacy 鼠标编码等含高位字节的输入走此通道（需按二进制帧发送） */
  onBinary?: (data: Uint8Array) => void
  /** 尺寸变化回调（用于向后端发送 resize） */
  onResize?: (size: { cols: number; rows: number }) => void
  fontSize?: number
}

export function useTerminalX(options: UseTerminalXOptions) {
  const containerRef = ref<HTMLElement>()
  const cols = ref(0)
  const rows = ref(0)

  let terminal: Terminal | null = null
  let fitAddon: FitAddon | null = null
  let resizeObserver: ResizeObserver | null = null

  const applyFit = () => {
    if (!terminal || !fitAddon || !containerRef.value) return
    try {
      fitAddon.fit()
    } catch {
      return // 容器尚不可见时忽略
    }
    if (terminal.cols === cols.value && terminal.rows === rows.value) return
    cols.value = terminal.cols
    rows.value = terminal.rows
    options.onResize?.({ cols: terminal.cols, rows: terminal.rows })
  }

  const mount = () => {
    if (terminal || !containerRef.value) return

    terminal = new Terminal({
      cursorBlink: true,
      convertEol: true,
      fontFamily: FONT_FAMILY,
      fontSize: options.fontSize ?? 13,
      lineHeight: 1.2,
      scrollback: 5000,
      theme: terminalThemeFor(options.theme.value),
    })

    fitAddon = new FitAddon()
    terminal.loadAddon(fitAddon)
    terminal.open(containerRef.value)
    applyFit()

    if (options.onData) terminal.onData(options.onData)
    if (options.onBinary) {
      terminal.onBinary((data) => {
        // xterm 用 latin1 字符串承载原始字节，逐字符还原为字节数组
        const bytes = new Uint8Array(data.length)
        for (let i = 0; i < data.length; i += 1) bytes[i] = data.charCodeAt(i) & 0xff
        options.onBinary?.(bytes)
      })
    }

    resizeObserver = new ResizeObserver(() => applyFit())
    resizeObserver.observe(containerRef.value)
    window.addEventListener('resize', applyFit)
  }

  const write = (data: string) => terminal?.write(data)
  const writeln = (data: string) => terminal?.writeln(data)
  const clear = () => terminal?.clear()
  const reset = () => terminal?.reset()
  const focus = () => terminal?.focus()

  watch(
    () => options.theme.value,
    (mode) => {
      if (terminal) terminal.options.theme = terminalThemeFor(mode)
    },
  )

  onBeforeUnmount(() => {
    window.removeEventListener('resize', applyFit)
    resizeObserver?.disconnect()
    resizeObserver = null
    terminal?.dispose()
    terminal = null
    fitAddon = null
  })

  return { containerRef, cols, rows, mount, write, writeln, clear, reset, focus, fit: applyFit }
}
