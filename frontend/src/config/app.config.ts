/**
 * 应用全局配置
 * 用户可以在这里自定义项目的各种配置项
 */

export const APP_CONFIG = {
  // 项目名称
  name: 'Poleaxe Admin',

  // Favicon src - 根据环境动态设置 base path
  faviconSrc: `${import.meta.env.VITE_STATIC_URL}favicon.ico`,

  // Logo src
  logoSrc: new URL('@/assets/logo.svg', import.meta.url).href,

  // 是否展示主题配置
  showThemeConfig: true,
}
