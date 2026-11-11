// 主题资源（背景图、字体）：上传、以及「能选的图片」列表（我的主题里用过的 + 这次上传的）。
import { computed, reactive, ref } from 'vue'
import type { ThemeAsset } from '@/types/v1/theme'
import { fileToBase64, uploadThemeAsset } from '@/api/theme'
import { useThemeStore } from '@/stores/theme'
import { formatFileSize } from '@/utils/file'
import { translate } from '@/locales'
import { feedback } from '@/utils/feedback'

const sessionAssets = reactive<ThemeAsset[]>([])
const uploading = ref(false)

const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp']
const MAX_IMAGE = 10 * 1024 * 1024
const MAX_FONT = 4 * 1024 * 1024

export const assetMeta = (a?: ThemeAsset | null) =>
  a && a.width ? `${a.width} × ${a.height} · ${formatFileSize(a.size)}` : a ? formatFileSize(a.size) : ''

export const useThemeAssets = () => {
  const themeStore = useThemeStore()

  const all = computed(() => {
    const seen = new Set<string>()
    const out: ThemeAsset[] = []
    for (const a of [...sessionAssets, ...themeStore.myThemes.flatMap((t) => t.assets || [])]) {
      if (!a?.id || seen.has(a.id)) continue
      seen.add(a.id)
      out.push(a)
    }
    return out
  })
  const images = computed(() => all.value.filter((a) => a.kind === 'image'))
  const findAsset = (id: string) => all.value.find((a) => a.id === id) ?? null

  /** 上传背景图（JPG / PNG / WebP ≤ 10 MB，服务端转 WebP）或 woff2 字体 */
  const upload = async (file: File, kind: 'image' | 'font' = 'image'): Promise<ThemeAsset | null> => {
    if (kind === 'image') {
      if (!IMAGE_TYPES.includes(file.type)) {
        feedback.error(translate('theme.uploadType'))
        return null
      }
      if (file.size > MAX_IMAGE) {
        feedback.error(translate('theme.uploadTooLarge'))
        return null
      }
    } else if (!/\.woff2$/i.test(file.name) || file.size > MAX_FONT) {
      feedback.error(translate('theme.editor.fontInvalid'))
      return null
    }
    uploading.value = true
    try {
      const data = await fileToBase64(file)
      const { data: res } = await uploadThemeAsset({ kind, fileName: file.name, data })
      if (!res?.asset) return null
      sessionAssets.unshift(res.asset)
      return res.asset
    } catch {
      return null
    } finally {
      uploading.value = false
    }
  }

  return { images, all, findAsset, upload, uploading }
}
