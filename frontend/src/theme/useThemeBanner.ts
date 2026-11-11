// 大图区域（工作台欢迎横幅、登录页插画区、手机首页顶部卡片）的背景：主色 / 渐变 / 自定义图片。
// 返回一组 CSS 变量，组件用 var(--banner-*) 取色；文字颜色「自动」时按图片左侧亮度判断。
import { computed } from 'vue'
import { useThemeStore } from '@/stores/theme'
import { assetUrl, bannerLook } from './content'
import { imageStats } from './imageStats'

export const useThemeBanner = () => {
  const themeStore = useThemeStore()
  const banner = computed(() => themeStore.banner)
  const url = computed(() => (banner.value.type === 'image' ? assetUrl(banner.value.image, themeStore.activeResolver) : ''))
  const look = computed(() =>
    bannerLook(banner.value, themeStore.activeResolver, url.value ? (imageStats(url.value)?.leftLuma ?? null) : null),
  )
  const style = computed(() => ({
    '--banner-bg': look.value.background,
    '--banner-fg': look.value.fg,
    '--banner-shade': look.value.shade,
    '--banner-btn-bg': look.value.btnBg,
    '--banner-btn-fg': look.value.btnFg,
    '--banner-soft': look.value.soft,
  }))
  return { banner, look, style }
}
