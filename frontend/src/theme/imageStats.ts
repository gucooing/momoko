// 背景图 / 横幅图的亮度统计（画到小画布上取样），用于「文字颜色：自动」、对比度检查和「从背景图取色」。
import { reactive } from 'vue'
import { rgbToHex, luminanceRgb, type RGB } from './color'

export interface ImageStats {
  /** 左侧 60%（横幅文字所在区域）平均亮度 0-1 */
  leftLuma: number
  /** 整体平均亮度 */
  avgLuma: number
  /** 最亮的一小块区域的颜色（取亮度前 5% 的平均） */
  bright: string
  /** 主色候选（饱和度高、出现多的颜色，最多 6 个） */
  palette: string[]
}

const cache = reactive(new Map<string, ImageStats | null>())
const pending = new Set<string>()

const W = 48
const H = 30

const analyze = (img: HTMLImageElement): ImageStats | null => {
  const canvas = document.createElement('canvas')
  canvas.width = W
  canvas.height = H
  const ctx = canvas.getContext('2d', { willReadFrequently: true })
  if (!ctx) return null
  ctx.drawImage(img, 0, 0, W, H)
  let data: Uint8ClampedArray
  try {
    data = ctx.getImageData(0, 0, W, H).data
  } catch {
    return null // 跨域图片没有 CORS 头时读不了像素
  }
  const px: { rgb: RGB; l: number; x: number }[] = []
  for (let i = 0; i < W * H; i++) {
    const rgb: RGB = [data[i * 4]!, data[i * 4 + 1]!, data[i * 4 + 2]!]
    px.push({ rgb, l: luminanceRgb(rgb), x: i % W })
  }
  const avg = (list: typeof px) => list.reduce((s, p) => s + p.l, 0) / Math.max(1, list.length)
  const sorted = [...px].sort((a, b) => b.l - a.l)
  const top = sorted.slice(0, Math.max(1, Math.round(px.length * 0.05)))
  const brightRgb = [0, 1, 2].map((c) => top.reduce((s, p) => s + p.rgb[c]!, 0) / top.length) as RGB

  // 简单量化取色：按 4 bit/通道分桶，挑出现多且饱和的
  const buckets = new Map<number, { n: number; sum: [number, number, number] }>()
  for (const p of px) {
    const key = ((p.rgb[0] >> 4) << 8) | ((p.rgb[1] >> 4) << 4) | (p.rgb[2] >> 4)
    const b = buckets.get(key) || { n: 0, sum: [0, 0, 0] }
    b.n++
    b.sum[0] += p.rgb[0]
    b.sum[1] += p.rgb[1]
    b.sum[2] += p.rgb[2]
    buckets.set(key, b)
  }
  const palette = [...buckets.values()]
    .map((b) => {
      const rgb = b.sum.map((v) => v / b.n) as RGB
      const mx = Math.max(...rgb)
      const mn = Math.min(...rgb)
      const sat = mx ? (mx - mn) / mx : 0
      return { hex: rgbToHex(rgb), score: b.n * (0.35 + sat) }
    })
    .sort((a, b) => b.score - a.score)
    .map((x) => x.hex)
    .filter((hex, i, arr) => arr.indexOf(hex) === i)
    .slice(0, 6)

  return {
    leftLuma: avg(px.filter((p) => p.x < W * 0.6)),
    avgLuma: avg(px),
    bright: rgbToHex(brightRgb),
    palette,
  }
}

/** 读取统计结果（未算出时返回 undefined 并开始计算；计算完成后响应式更新） */
export const imageStats = (url: string): ImageStats | null | undefined => {
  if (!url) return null
  if (cache.has(url)) return cache.get(url)
  if (!pending.has(url) && typeof window !== 'undefined') {
    pending.add(url)
    const img = new Image()
    if (!url.startsWith('data:') && !url.startsWith('blob:')) img.crossOrigin = 'anonymous'
    img.onload = () => {
      cache.set(url, analyze(img))
      pending.delete(url)
    }
    img.onerror = () => {
      cache.set(url, null)
      pending.delete(url)
    }
    img.src = url
  }
  return undefined
}
