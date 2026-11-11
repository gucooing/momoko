// 主题「页面单独设置」的页面清单：直接从路由表生成 —— 菜单页面（按菜单顺序）+ 外壳里的静态页面 + 公开页，
// 以后新增页面（加菜单或加路由）会自动出现在主题编辑器里，不需要在这里登记。
// 页面键 = 匹配到的路由模板（如 /dashboard/home、/instance/app/:instanceId），App.vue 按它套用页面单独设置。
//
// 路由 meta 约定（router/route.ts、menuToRoute）：
//   title / titleKey  页面名（菜单页面来自菜单标题；titleKey 为 i18n 键，优先）
//   theme: false      不出现在主题编辑器里（错误页、编辑器自己、初始化向导…）
//   siteTheme: true   访客看到的页面，套用全站默认主题（登录页）
//   public / authOnly 公开页 / 只需登录的独立页（归到「其它页面」）
import { computed } from 'vue'
import { useRouter, type RouteRecordNormalized, type Router } from 'vue-router'
import type { MenuInfo } from '@/types/v1/system'
import { MenuType } from '@/types/v1/system'
import { translate, translateKnownText } from '@/locales'
import { useMenuStore } from '@/stores/menu'

export interface ThemePageDef {
  /** 页面单独设置的键（路由模板） */
  key: string
  label: string
  /** menu：菜单页面；shell：外壳里的其它页面；public：公开 / 独立页 */
  group: 'menu' | 'shell' | 'public'
  /** 路由带参数（如应用详情），不能直接打开，要在预览里点进去 */
  params: boolean
  /** 访客页面（套用全站默认主题） */
  siteTheme: boolean
}

/** 路由模板 → 页面键 */
export const pageKeyOf = (matchedPath: string | undefined | null) => {
  const p = (matchedPath || '').replace(/\/+$/, '')
  return p ? (p.startsWith('/') ? p : `/${p}`) : '/'
}

const flatMenus = (list: MenuInfo[]): MenuInfo[] =>
  list.flatMap((m) =>
    m.type === MenuType.MenuType_Directory ? flatMenus(m.children || []) : m.type === MenuType.MenuType_Menu ? [m] : [],
  )

const routeLabel = (r: RouteRecordNormalized) => {
  const key = r.meta?.titleKey
  if (typeof key === 'string' && key) return translate(key)
  const title = r.meta?.title
  if (typeof title === 'string' && title) return translateKnownText(title)
  return String(r.name || r.path)
}

/** 路由表里所有可以单独设置主题的页面 */
export const listThemePages = (router: Router, menus: MenuInfo[]): ThemePageDef[] => {
  const menuOrder = new Map(flatMenus(menus).map((m, i) => [pageKeyOf(m.path), i]))
  const seen = new Set<string>()
  const out: (ThemePageDef & { order: number })[] = []
  for (const r of router.getRoutes()) {
    if (!r.components || r.children.length || r.redirect) continue
    if (r.meta?.theme === false || r.path.includes('(.*)')) continue
    const key = pageKeyOf(r.path)
    if (seen.has(key)) continue
    seen.add(key)
    const isPublic = !!(r.meta?.public || r.meta?.authOnly || r.meta?.siteTheme)
    const inMenu = menuOrder.has(key)
    out.push({
      key,
      label: routeLabel(r),
      group: inMenu ? 'menu' : isPublic ? 'public' : 'shell',
      params: key.includes(':'),
      siteTheme: !!r.meta?.siteTheme,
      order: inMenu ? menuOrder.get(key)! : isPublic ? 2000 + out.length : 1000 + out.length,
    })
  }
  return out
    .sort((a, b) => a.order - b.order)
    .map((p) => ({ key: p.key, label: p.label, group: p.group, params: p.params, siteTheme: p.siteTheme }))
}

/** 组件里用：页面清单（菜单加载后自动更新）与页面名 */
export const useThemePages = () => {
  const router = useRouter()
  const menuStore = useMenuStore()
  const pages = computed(() => {
    // 依赖菜单：动态路由与菜单同时加入路由表
    void menuStore.menuList
    return listThemePages(router, menuStore.menuList)
  })
  const labelOf = (key: string) => pages.value.find((p) => p.key === key)?.label || key
  /** 访客页面（登录页）的键：导入预览、默认页面单独设置用 */
  const siteThemeKey = computed(() => pages.value.find((p) => p.siteTheme)?.key ?? null)
  return { pages, labelOf, siteThemeKey }
}
