// 右侧面板的宽度：中间和右栏之间的缝可以左右拖，双击回到默认宽度；键盘上左右方向键也能调。
// 会话页和新会话页共用一个宽度，记在本机。
const KEY = 'agent.code.rightWidth'
const DEFAULT = 400
const MIN = 300
const STEP = 24

/** 最宽多少：中间至少留 560 */
const maxWidth = () => Math.max(MIN, Math.min(1200, window.innerWidth - 560))
const clamp = (w: number) => Math.round(Math.max(MIN, Math.min(maxWidth(), w)))
const read = () => {
  try {
    const n = Number(localStorage.getItem(KEY))
    return n >= MIN ? n : DEFAULT
  } catch {
    return DEFAULT
  }
}
const save = () => {
  try {
    if (rightWidth.value === DEFAULT) localStorage.removeItem(KEY)
    else localStorage.setItem(KEY, String(rightWidth.value))
  } catch {
    /* 存不了就只在这次有效 */
  }
}

export const rightWidth = ref(read())

/** 按下拖动条：之后的移动都交给它（盖在 iframe 上也收得到） */
export const startRightResize = (e: PointerEvent) => {
  if (e.button !== 0) return
  e.preventDefault()
  const grip = e.currentTarget as HTMLElement
  const startX = e.clientX
  const startW = rightWidth.value
  grip.setPointerCapture(e.pointerId)
  document.body.classList.add('agc-resizing')
  const move = (ev: PointerEvent) => {
    rightWidth.value = clamp(startW + startX - ev.clientX)
  }
  const up = () => {
    grip.removeEventListener('pointermove', move)
    grip.removeEventListener('pointerup', up)
    grip.removeEventListener('pointercancel', up)
    document.body.classList.remove('agc-resizing')
    save()
  }
  grip.addEventListener('pointermove', move)
  grip.addEventListener('pointerup', up)
  grip.addEventListener('pointercancel', up)
}

export const resetRightWidth = () => {
  rightWidth.value = clamp(DEFAULT)
  save()
}

export const onRightGripKey = (e: KeyboardEvent) => {
  // 右栏在右边：往左是变宽
  const d = e.key === 'ArrowLeft' ? STEP : e.key === 'ArrowRight' ? -STEP : 0
  if (!d) return
  e.preventDefault()
  rightWidth.value = clamp(rightWidth.value + d)
  save()
}

/** 右栏的 style：手机上右栏是底部抽屉，不设宽度 */
export const rightStyle = (mobile: boolean) => (mobile ? undefined : { width: `${rightWidth.value}px` })
