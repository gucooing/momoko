// Imagine API 的 sub2api 凭据上下文（运行期由 store bootstrap 写入）。
// 独立成模块，避免 api client 与 pinia store 之间的循环依赖。
export const imagineContext = {
  token: '',
  srcHost: '',
}
