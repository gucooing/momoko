import { ref, onMounted, onBeforeUnmount } from 'vue'

/**
 * 内容区布局声明。
 *
 * 「全出血」= 整页即内容本身，去掉 AppContent 的内边距与限宽（如文件管理、终端）。
 * 这是**页面自身**的表现诉求，故由页面在 setup 中声明，而不是把页面路径耦合进路由生成逻辑。
 * 静态路由仍可用 `meta.fullBleed`（见终端页）；两者在 AppContent 里合并生效。
 */
const contentFullBleed = ref(false)

// AppContent 读取：内容区是否全出血。
export const useContentFullBleed = () => contentFullBleed

// 页面调用：挂载即全出血，卸载还原。内容区无 keep-alive，导航即重挂载，故成对触发可靠。
export const useFullBleed = () => {
  onMounted(() => {
    contentFullBleed.value = true
  })
  onBeforeUnmount(() => {
    contentFullBleed.value = false
  })
}
