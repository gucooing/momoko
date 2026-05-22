export interface AnsiTextSegment {
  text: string
  style: Record<string, string | number>
}

export interface AnsiRenderedLine {
  segments: AnsiTextSegment[]
}

interface AnsiState {
  foreground?: string
  background?: string
  bold: boolean
  faint: boolean
  italic: boolean
  underline: boolean
  strikethrough: boolean
  inverse: boolean
  hidden: boolean
}

const DEFAULT_FOREGROUND = '#e8edf7'
const DEFAULT_BACKGROUND = '#141922'

const ANSI_COLOR_PALETTE = [
  '#1f2430',
  '#ff6b6b',
  '#b8e986',
  '#ffd866',
  '#6fb3ff',
  '#d29bff',
  '#4dd0e1',
  '#d8dee9',
  '#5c6773',
  '#ff8787',
  '#c3e88d',
  '#ffcb6b',
  '#82aaff',
  '#c792ea',
  '#89ddff',
  '#ffffff',
]

const ANSI_ESCAPE_PATTERN = /\u001b\[([0-9;?]*)([A-Za-z])/gu

const createDefaultState = (): AnsiState => ({
  foreground: undefined,
  background: undefined,
  bold: false,
  faint: false,
  italic: false,
  underline: false,
  strikethrough: false,
  inverse: false,
  hidden: false,
})

const cloneState = (state: AnsiState): AnsiState => ({ ...state })

const getAnsiColor = (index: number) => ANSI_COLOR_PALETTE[index] || undefined

const getAnsi256Color = (index: number) => {
  if (index < 0 || index > 255) return undefined
  if (index < 16) return getAnsiColor(index)

  if (index <= 231) {
    const cubeIndex = index - 16
    const levels = [0, 95, 135, 175, 215, 255]
    const red = levels[Math.floor(cubeIndex / 36) % 6]
    const green = levels[Math.floor(cubeIndex / 6) % 6]
    const blue = levels[cubeIndex % 6]
    return `rgb(${red}, ${green}, ${blue})`
  }

  const gray = 8 + (index - 232) * 10
  return `rgb(${gray}, ${gray}, ${gray})`
}

const parseCodes = (rawValue: string) => {
  if (!rawValue.trim()) return [0]

  return rawValue
    .split(';')
    .filter((value) => value.trim() !== '')
    .map((value) => Number(value))
    .filter((value) => Number.isFinite(value))
}

const parseExtendedColor = (codes: number[], startIndex: number) => {
  const mode = codes[startIndex + 1]

  if (mode === 5) {
    const color = getAnsi256Color(codes[startIndex + 2] ?? -1)
    return {
      color,
      nextIndex: startIndex + 2,
    }
  }

  if (mode === 2) {
    const red = codes[startIndex + 2]
    const green = codes[startIndex + 3]
    const blue = codes[startIndex + 4]

    if ([red, green, blue].every((value) => Number.isFinite(value))) {
      return {
        color: `rgb(${red}, ${green}, ${blue})`,
        nextIndex: startIndex + 4,
      }
    }
  }

  return {
    color: undefined,
    nextIndex: mode === 5 ? startIndex + 2 : mode === 2 ? startIndex + 4 : startIndex,
  }
}

const applyAnsiCodes = (codes: number[], state: AnsiState) => {
  const nextCodes = codes.length ? codes : [0]

  for (let index = 0; index < nextCodes.length; index += 1) {
    const code = nextCodes[index]
    if (code === undefined) continue

    if (code === 0) {
      Object.assign(state, createDefaultState())
      continue
    }

    if (code === 1) {
      state.bold = true
      continue
    }

    if (code === 2) {
      state.faint = true
      continue
    }

    if (code === 3) {
      state.italic = true
      continue
    }

    if (code === 4) {
      state.underline = true
      continue
    }

    if (code === 7) {
      state.inverse = true
      continue
    }

    if (code === 8) {
      state.hidden = true
      continue
    }

    if (code === 9) {
      state.strikethrough = true
      continue
    }

    if (code === 22) {
      state.bold = false
      state.faint = false
      continue
    }

    if (code === 23) {
      state.italic = false
      continue
    }

    if (code === 24) {
      state.underline = false
      continue
    }

    if (code === 27) {
      state.inverse = false
      continue
    }

    if (code === 28) {
      state.hidden = false
      continue
    }

    if (code === 29) {
      state.strikethrough = false
      continue
    }

    if (code === 39) {
      state.foreground = undefined
      continue
    }

    if (code === 49) {
      state.background = undefined
      continue
    }

    if (code >= 30 && code <= 37) {
      state.foreground = getAnsiColor(code - 30)
      continue
    }

    if (code >= 40 && code <= 47) {
      state.background = getAnsiColor(code - 40)
      continue
    }

    if (code >= 90 && code <= 97) {
      state.foreground = getAnsiColor(code - 90 + 8)
      continue
    }

    if (code >= 100 && code <= 107) {
      state.background = getAnsiColor(code - 100 + 8)
      continue
    }

    if (code === 38 || code === 48) {
      const { color, nextIndex } = parseExtendedColor(nextCodes, index)

      if (color) {
        if (code === 38) {
          state.foreground = color
        } else {
          state.background = color
        }
      }

      index = nextIndex
    }
  }
}

const buildSegmentStyle = (state: AnsiState) => {
  const style: Record<string, string | number> = {}

  const originalForeground = state.foreground
  const originalBackground = state.background
  let foreground = originalForeground
  let background = originalBackground

  if (state.inverse) {
    foreground = originalBackground || DEFAULT_BACKGROUND
    background = originalForeground || DEFAULT_FOREGROUND
  }

  if (foreground) {
    style.color = foreground
  }

  if (background) {
    style.backgroundColor = background
  }

  if (state.bold) {
    style.fontWeight = 700
  }

  if (state.italic) {
    style.fontStyle = 'italic'
  }

  const decorations: string[] = []

  if (state.underline) {
    decorations.push('underline')
  }

  if (state.strikethrough) {
    decorations.push('line-through')
  }

  if (decorations.length) {
    style.textDecorationLine = decorations.join(' ')
  }

  if (state.hidden) {
    style.visibility = 'hidden'
  } else if (state.faint) {
    style.opacity = 0.72
  }

  return style
}

const parseAnsiLine = (line: string, carryState: AnsiState) => {
  const segments: AnsiTextSegment[] = []
  const lineState = cloneState(carryState)
  let cursor = 0

  for (const match of line.matchAll(ANSI_ESCAPE_PATTERN)) {
    const matchIndex = match.index ?? 0
    const [rawToken, rawCodes = '', command] = match

    if (matchIndex > cursor) {
      segments.push({
        text: line.slice(cursor, matchIndex),
        style: buildSegmentStyle(lineState),
      })
    }

    cursor = matchIndex + rawToken.length

    if (command === 'm') {
      applyAnsiCodes(parseCodes(rawCodes), lineState)
    }
  }

  if (cursor < line.length) {
    segments.push({
      text: line.slice(cursor),
      style: buildSegmentStyle(lineState),
    })
  }

  return {
    segments,
    nextState: lineState,
  }
}

export const parseAnsiOutputLines = (lines: string[]): AnsiRenderedLine[] => {
  let carryState = createDefaultState()

  return lines.map((line) => {
    const { segments, nextState } = parseAnsiLine(line, carryState)
    carryState = nextState

    return { segments }
  })
}
