// 输入框里还没发出去的草稿：切到文件标签、切会话时输入框会卸下或换会话，先存在这里，回来接着写。
// 按会话 id 存，新会话页用 'new'。只在这个页面的内存里，刷新页面就没了。
import type { DraftAttachment } from './Composer.vue'

export const composerDrafts = new Map<string, { text: string; atts: DraftAttachment[] }>()
