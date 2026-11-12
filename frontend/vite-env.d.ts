interface ImportMetaEnv {
  // 接口基础URL
  readonly VITE_API_BASE_URL: string
  // 静态资源URL
  readonly VITE_STATIC_URL: string
  // 用户通行证（玩家端）接口基础 URL
  readonly VITE_PASSPORT_API_BASE_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
