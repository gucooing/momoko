// 服务器的应用图标 / 应用背景：服务器「接入设置」里上传到对象存储的链接（GameServerBrief.appIcon / appBackground），
// 管理端、用户通行证与 PC 启动器同步使用；没有上传时各处退回游戏名首字与封面。背景可以是视频，按链接的扩展名判断
// （对象沿用原文件的扩展名）。

/** 链接是不是视频（应用背景可以是视频）。 */
export const isVideoLink = (link?: string | null): boolean => /\.(mp4|webm|mov|m4v|mkv|ogv|avi)(\?|#|$)/i.test(link || '')
