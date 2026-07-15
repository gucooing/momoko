/**
 * 应用全局配置
 * 用户可以在这里自定义项目的各种配置项
 */

// 打包后的 logo URL（相对当前 origin，随页面协议；避免写死 http 触发 Mixed Content）
const LOGO_URL = new URL('@/assets/logo.png', import.meta.url).href

export const APP_CONFIG = {
  // 项目名称
  name: 'Momoko',

  // Favicon：用 Vite 解析的资源，不要拼绝对 http://host/logo.png
  faviconSrc: LOGO_URL,

  // Logo src
  logoSrc: LOGO_URL,

  // 是否展示主题配置
  showThemeConfig: true,
}
