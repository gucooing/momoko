// 通行证套用全站默认外观主题（管理后台「设为全站默认」的同一份，随公开配置下发）：用管理后台同一个主题引擎
// （src/theme/engine.ts）写 <html> 的配色、细节与背景类名，背景层由 ThemeBackdrop 绘制；没设置全站默认时是默认配色。
// 上次的主题存在本机，下次打开先套上，免得先闪一下默认配色。深浅色仍由通行证自己的开关决定（stores/ui.ts）。
import { computed, shallowRef } from 'vue'
import type { SiteTheme, ThemeContent } from '@/types/v1/theme'
import { ASSET_ID_RE, emptyContent, normalizeContent, resolveBackground } from '@/theme/content'
import { applyThemeToDocument } from '@/theme/engine'

const KEY = 'passport.site-theme'
const API_BASE = ((import.meta.env.VITE_PASSPORT_API_BASE_URL as string | undefined) || '/api/v1').replace(/\/$/, '')

/** 主题引用的背景图 / 字体：由通行证 server 提供（与管理后台同一份资源）。 */
export const themeResolver = (ref: string) => (ASSET_ID_RE.test(ref) ? `${API_BASE}/theme-asset/${ref}` : '')

const content = shallowRef<ThemeContent>(emptyContent())

/** 当前背景（给 ThemeBackdrop） */
export const themeBackground = computed(() => resolveBackground(content.value, null, null).bg)

function apply(c: ThemeContent) {
  content.value = c
  applyThemeToDocument({ content: c, pageKey: null, override: null, radiusPref: 'round', resolver: themeResolver })
}

/** 套用公开配置里的全站默认主题（为空时恢复默认配色），并记在本机。 */
export function applySiteTheme(t: SiteTheme | null | undefined) {
  apply(t?.content ? normalizeContent(t.content) : emptyContent())
  try {
    if (t?.content) localStorage.setItem(KEY, JSON.stringify(t.content))
    else localStorage.removeItem(KEY)
  } catch {
    // 隐私模式：不记
  }
}

/** 启动时先套上本机记着的主题。 */
export function initSiteTheme() {
  let saved: ThemeContent | null = null
  try {
    const raw = localStorage.getItem(KEY)
    saved = raw ? (JSON.parse(raw) as ThemeContent) : null
  } catch {
    saved = null
  }
  apply(saved ? normalizeContent(saved) : emptyContent())
}
