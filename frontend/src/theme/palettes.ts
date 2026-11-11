// 六套内置配色的完整色值（与 styles/design-tokens.css 中的 data-palette 定义保持一致），
// 以及主题编辑器里展示的 token 清单。自定义主题只存「改过的 token」，其余取所基于的内置配色。

export type PaletteKey = 'sakura' | 'mint' | 'volt' | 'grape' | 'sunset' | 'classic'
export type SchemeKey = 'light' | 'dark'

/** 界面 token（对应 CSS 变量 --m-*） */
export const UI_TOKEN_KEYS = [
  'p', 'pOn', 'pInk', 'pSoft', 'a', 'aOn', 'aSoft', 'a2', 'a2Soft',
  'bg', 's', 's2', 'line', 'ink', 'ink2',
  'ok', 'okSoft', 'warn', 'warnSoft', 'bad', 'badSoft',
] as const
export type UiTokenKey = (typeof UI_TOKEN_KEYS)[number]

/** 终端专用 token（只影响控制台 / SSH 终端的深色终端） */
export const TERM_TOKEN_KEYS = [
  'termBg', 'termFg', 'termCursor',
  'ansi0', 'ansi1', 'ansi2', 'ansi3', 'ansi4', 'ansi5', 'ansi6', 'ansi7',
  'ansi8', 'ansi9', 'ansi10', 'ansi11', 'ansi12', 'ansi13', 'ansi14', 'ansi15',
] as const
export type TermTokenKey = (typeof TERM_TOKEN_KEYS)[number]
export type TokenKey = UiTokenKey | TermTokenKey
export type TokenMap = Record<UiTokenKey, string>

const P = (s: string): TokenMap => {
  const out = {} as TokenMap
  for (const pair of s.trim().split(/\s*;\s*/)) {
    if (!pair) continue
    const [k, v] = pair.split(':').map((x) => x.trim())
    out[k as UiTokenKey] = v!.toUpperCase()
  }
  return out
}

export const PALETTE_TOKENS: Record<PaletteKey, Record<SchemeKey, TokenMap>> = {
  sakura: {
    light: P(`p:#d6245f;pOn:#ffffff;pInk:#c8195a;pSoft:#ffe3ee;a:#ffc93c;aOn:#2a1630;aSoft:#fff3cc;a2:#7b5cff;a2Soft:#ece6ff;
      bg:#fff6f9;s:#ffffff;s2:#fff0f5;line:#f6dce6;ink:#2a1630;ink2:#6e5a72;ok:#0b7f57;okSoft:#ddf5ea;warn:#b25e00;warnSoft:#fff0d9;bad:#d21f3c;badSoft:#ffe2e6`),
    dark: P(`p:#ff4f8b;pOn:#1a0b14;pInk:#ff85ae;pSoft:#3a1a2c;a:#ffd25a;aOn:#1a0b14;aSoft:#3a3016;a2:#a08bff;a2Soft:#2a2245;
      bg:#140e17;s:#1e1622;s2:#271d2c;line:#33263a;ink:#f8eef4;ink2:#b9a6be;ok:#3ddc97;okSoft:#123326;warn:#ffb547;warnSoft:#3a2a12;bad:#ff6b81;badSoft:#3a1720`),
  },
  mint: {
    light: P(`p:#00805f;pOn:#ffffff;pInk:#007a5a;pSoft:#d9f7ec;a:#ffe14d;aOn:#0f2a21;aSoft:#fff8cc;a2:#2e9bff;a2Soft:#e0f0ff;
      bg:#f3fbf7;s:#ffffff;s2:#eaf7f1;line:#d3ede2;ink:#0f2a21;ink2:#4f6b61;ok:#0b7f57;okSoft:#ddf5ea;warn:#a35b00;warnSoft:#fff2d6;bad:#d2263b;badSoft:#ffe3e6`),
    dark: P(`p:#1fd3a0;pOn:#0c1512;pInk:#4be3b6;pSoft:#0f3329;a:#ffe14d;aOn:#0c1512;aSoft:#33300f;a2:#5bb4ff;a2Soft:#12283a;
      bg:#0c1512;s:#13201b;s2:#192a23;line:#22362e;ink:#eaf7f2;ink2:#9db8ae;ok:#4be3b6;okSoft:#0f3329;warn:#ffc14d;warnSoft:#3a2e12;bad:#ff6b7d;badSoft:#3a1820`),
  },
  volt: {
    light: P(`p:#3450ff;pOn:#ffffff;pInk:#2a44e6;pSoft:#e3e8ff;a:#ff7a1a;aOn:#11163a;aSoft:#ffe9d9;a2:#00b8a9;a2Soft:#d9f6f3;
      bg:#f4f6ff;s:#ffffff;s2:#edf0ff;line:#dde2f7;ink:#11163a;ink2:#555c80;ok:#0b7f57;okSoft:#ddf5ea;warn:#b4530a;warnSoft:#ffebd9;bad:#d32448;badSoft:#ffe2e8`),
    dark: P(`p:#5b74ff;pOn:#0d1024;pInk:#8c9dff;pSoft:#1c2350;a:#ff8a33;aOn:#0d1024;aSoft:#3a2312;a2:#2fd9c8;a2Soft:#10302f;
      bg:#0d1024;s:#161a35;s2:#1d2242;line:#272d52;ink:#eef0ff;ink2:#a3a9cc;ok:#3edc9a;okSoft:#113024;warn:#ffaa55;warnSoft:#3a2812;bad:#ff6b86;badSoft:#3a1626`),
  },
  grape: {
    light: P(`p:#7c3aed;pOn:#ffffff;pInk:#6d28d9;pSoft:#efe6ff;a:#ffd23f;aOn:#1f1436;aSoft:#fff6cc;a2:#e8457a;a2Soft:#ffe3ec;
      bg:#f8f5ff;s:#ffffff;s2:#f2ecff;line:#e4dafa;ink:#1f1436;ink2:#625878;ok:#0b7f57;okSoft:#ddf5ea;warn:#a35b00;warnSoft:#fff2d6;bad:#d2263b;badSoft:#ffe3e6`),
    dark: P(`p:#a78bfa;pOn:#120c1f;pInk:#c4b1ff;pSoft:#2a1f4a;a:#ffd23f;aOn:#120c1f;aSoft:#37300f;a2:#ff7ba6;a2Soft:#3a1a2a;
      bg:#120c1f;s:#1b1430;s2:#241b3d;line:#30264d;ink:#f2eeff;ink2:#b2a8cc;ok:#3ddc97;okSoft:#123326;warn:#ffb547;warnSoft:#3a2a12;bad:#ff6b81;badSoft:#3a1720`),
  },
  sunset: {
    light: P(`p:#c2410c;pOn:#ffffff;pInk:#b23a0a;pSoft:#ffe6d6;a:#ffc93c;aOn:#2b1708;aSoft:#fff3cc;a2:#2f6fed;a2Soft:#e1ebff;
      bg:#fff7f0;s:#ffffff;s2:#fff0e5;line:#f7dfcc;ink:#2b1708;ink2:#735a4b;ok:#0b7f57;okSoft:#ddf5ea;warn:#9a5a00;warnSoft:#fff2d6;bad:#c81e3a;badSoft:#ffe3e6`),
    dark: P(`p:#ff7a3d;pOn:#1a0f08;pInk:#ffa06e;pSoft:#3a2114;a:#ffd25a;aOn:#1a0f08;aSoft:#3a3016;a2:#6ea0ff;a2Soft:#172440;
      bg:#150e09;s:#22160e;s2:#2c1d13;line:#3a281b;ink:#fff1e8;ink2:#c7ae9e;ok:#3ddc97;okSoft:#123326;warn:#ffc14d;warnSoft:#3a2e12;bad:#ff6b7d;badSoft:#3a1820`),
  },
  classic: {
    light: P(`p:#0f7f72;pOn:#ffffff;pInk:#0b7266;pSoft:#ddf3ef;a:#ffb020;aOn:#1b2333;aSoft:#fff1d6;a2:#5b6cff;a2Soft:#e6e9ff;
      bg:#f3f5f9;s:#ffffff;s2:#eef1f6;line:#e2e7ee;ink:#1b2333;ink2:#5f6878;ok:#0b7f57;okSoft:#ddf5ea;warn:#a35b00;warnSoft:#fff2d6;bad:#d2263b;badSoft:#ffe3e6`),
    dark: P(`p:#2cc4ae;pOn:#0e1116;pInk:#5ed8c5;pSoft:#12302c;a:#ffb020;aOn:#0e1116;aSoft:#3a2e12;a2:#8c98ff;a2Soft:#1e2348;
      bg:#0e1116;s:#171b22;s2:#1c212b;line:#2b323d;ink:#e8ebf1;ink2:#9aa2b1;ok:#3ddc97;okSoft:#123326;warn:#ffc14d;warnSoft:#3a2e12;bad:#ff6b7d;badSoft:#3a1820`),
  },
}

/** 终端默认色（深色终端，与 components/terminal/useTerminalX.ts 的 DARK_THEME 一致） */
export const TERM_DEFAULTS: Record<TermTokenKey, string> = {
  termBg: '#0F0B14',
  termFg: '#E8E3EE',
  termCursor: '#E8E3EE',
  ansi0: '#484F58',
  ansi1: '#FF7B72',
  ansi2: '#3FB950',
  ansi3: '#D29922',
  ansi4: '#58A6FF',
  ansi5: '#BC8CFF',
  ansi6: '#39C5CF',
  ansi7: '#B1BAC4',
  ansi8: '#6E7681',
  ansi9: '#FFA198',
  ansi10: '#56D364',
  ansi11: '#E3B341',
  ansi12: '#79C0FF',
  ansi13: '#D2A8FF',
  ansi14: '#56D4DD',
  ansi15: '#F0F6FC',
}

/** 编辑器里列出的 token（名称 / 用途走 i18n：theme.tokens.<key>.name / .use） */
export const EDITOR_TOKENS: UiTokenKey[] = ['p', 'pOn', 'a', 'a2', 'bg', 's', 's2', 'line', 'ink', 'ink2', 'ok', 'warn', 'bad']
export const EDITOR_TERM_TOKENS: TermTokenKey[] = ['termBg', 'termFg', 'termCursor']

/**
 * 派生 token：只改了「父」token、没单独改它时，按父色自动算出（避免改了主色后淡底还是原来的颜色）。
 * 例如改了 p 而没改 pSoft，pSoft 取 p 与卡片色的混合。
 */
export const DERIVED_TOKENS: { key: UiTokenKey; from: UiTokenKey }[] = [
  { key: 'pInk', from: 'p' },
  { key: 'pSoft', from: 'p' },
  { key: 'aOn', from: 'a' },
  { key: 'aSoft', from: 'a' },
  { key: 'a2Soft', from: 'a2' },
  { key: 'okSoft', from: 'ok' },
  { key: 'warnSoft', from: 'warn' },
  { key: 'badSoft', from: 'bad' },
]

export interface PaletteOption {
  key: PaletteKey
  /** i18n 名称键 */
  labelKey: string
  /** 英文短名（展示用，不翻译） */
  en: string
  /** 预览色：主色 / 强调色 / 第二强调色（浅色版），用于选择器色块 */
  swatches: [string, string, string]
}

export const PALETTE_OPTIONS: PaletteOption[] = (
  [
    ['sakura', 'SAKURA SODA'],
    ['mint', 'MINT LEMON'],
    ['volt', 'VOLT'],
    ['grape', 'GRAPE FIZZ'],
    ['sunset', 'SUNSET'],
    ['classic', 'CLASSIC'],
  ] as [PaletteKey, string][]
).map(([key, en]) => {
  const t = PALETTE_TOKENS[key].light
  return { key, labelKey: `theme.palettes.${key}`, en, swatches: [t.p, t.a, t.a2] }
})

export const isPaletteKey = (v: unknown): v is PaletteKey => PALETTE_OPTIONS.some((p) => p.key === v)
