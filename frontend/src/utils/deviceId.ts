const STORAGE_KEY = 'poleaxe_device_id'

function getCanvasFingerprint(): string {
  try {
    const canvas = document.createElement('canvas')
    canvas.width = 200
    canvas.height = 50
    const ctx = canvas.getContext('2d')
    if (!ctx) return ''

    ctx.textBaseline = 'top'
    ctx.font = '14px Arial'
    ctx.fillStyle = '#f60'
    ctx.fillRect(125, 1, 62, 20)
    ctx.fillStyle = '#069'
    ctx.fillText('PoleaxeAdmin,\ud83d\ude43', 2, 15)
    ctx.fillStyle = 'rgba(102, 204, 0, 0.7)'
    ctx.fillText('device-fingerprint', 4, 37)

    return canvas.toDataURL()
  } catch {
    return ''
  }
}

function getWebGLFingerprint(): string {
  try {
    const canvas = document.createElement('canvas')
    const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl')
    if (!gl || !(gl instanceof WebGLRenderingContext)) return ''

    const debugInfo = gl.getExtension('WEBGL_debug_renderer_info')
    const vendor = debugInfo ? gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) : ''
    const renderer = debugInfo ? gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) : ''

    return `${vendor}~${renderer}`
  } catch {
    return ''
  }
}

function collectBrowserTraits(): string[] {
  const traits: string[] = []

  traits.push(`ua:${navigator.userAgent}`)
  traits.push(`lang:${navigator.language}`)
  traits.push(`langs:${(navigator.languages || []).join(',')}`)
  traits.push(`screen:${screen.width}x${screen.height}x${screen.colorDepth}`)
  traits.push(`pixelRatio:${window.devicePixelRatio}`)
  traits.push(`cores:${navigator.hardwareConcurrency || 'unknown'}`)
  traits.push(`maxTouchPoints:${navigator.maxTouchPoints || 0}`)

  try {
    traits.push(`tz:${Intl.DateTimeFormat().resolvedOptions().timeZone}`)
  } catch {
    traits.push(`tzOffset:${new Date().getTimezoneOffset()}`)
  }

  try {
    const ua = (navigator as { userAgentData?: { platform?: string } }).userAgentData
    if (ua?.platform) {
      traits.push(`platform:${ua.platform}`)
    } else {
      traits.push(`platform:${navigator.platform}`)
    }
  } catch {
    traits.push(`platform:${navigator.platform}`)
  }

  traits.push(`canvas:${getCanvasFingerprint()}`)
  traits.push(`webgl:${getWebGLFingerprint()}`)

  return traits
}

/**
 * 纯 JS 实现的 FNV-1a 哈希（32 位），不依赖 Web Crypto API，全平台可用。
 */
function fnv1aHash(str: string): string {
  let hash = 0x811c9dc5
  for (let i = 0; i < str.length; i++) {
    hash ^= str.charCodeAt(i)
    hash = Math.imul(hash, 0x01000193)
  }
  return (hash >>> 0).toString(16).padStart(8, '0')
}

/**
 * 将输入切分为多段分别计算 FNV-1a，拼接为 64 字符的十六进制字符串，
 * 在效果上近似 SHA-256 的长度，但完全不依赖 crypto.subtle。
 */
function hashFingerprint(raw: string): string {
  const segments = 8
  const segLen = Math.ceil(raw.length / segments)
  const parts: string[] = []
  for (let i = 0; i < segments; i++) {
    const chunk = raw.slice(i * segLen, (i + 1) * segLen) + `|seg${i}`
    parts.push(fnv1aHash(chunk))
  }
  return parts.join('')
}

function generateUUID(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

let cachedId: string | null = null

/**
 * 获取当前浏览器/设备的唯一 ID。
 *
 * 优先从 localStorage 读取已生成的 ID；
 * 若无缓存，则基于浏览器指纹计算确定性哈希；
 * 若指纹采集失败（如受限 WebView），则回退为随机 UUID 并持久化。
 */
export function getDeviceId(): string {
  if (cachedId) return cachedId

  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored) {
    cachedId = stored
    return stored
  }

  let id: string
  try {
    const traits = collectBrowserTraits()
    const raw = traits.join('||')
    id = hashFingerprint(raw)
  } catch {
    id = generateUUID().replace(/-/g, '')
  }

  cachedId = id
  localStorage.setItem(STORAGE_KEY, id)
  return id
}
