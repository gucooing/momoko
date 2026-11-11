<!-- 内容区：router-view + keep-alive 页面缓存 + 内边距/限宽 + renderKey。
     同组子页（如 容器/镜像/网络/配置）之间切换不重挂载、不重新拉数据、不丢筛选与未保存的表单；
     同组切换也不做淡入淡出，像真正的页内标签一样即时切换。renderKey 变化（标签页「刷新」）仍会重建页面。 -->
<template>
  <main
    ref="mainRef"
    class="app-content"
    :class="{
      'app-content--bleed': isBleed,
      'app-content--tabbar': menuStore.isMobile && !route.meta?.hideTabbar,
      'app-content--notabbar': menuStore.isMobile && !!route.meta?.hideTabbar,
    }"
  >
    <RouterView v-slot="{ Component, route: r }">
      <Transition :name="transitionName(r)" mode="out-in">
        <KeepAlive :max="PAGE_CACHE_MAX">
          <PageFrame :key="tabsStore.getRouteRenderKey(r.fullPath)" :page="Component" />
        </KeepAlive>
      </Transition>
    </RouterView>
  </main>
</template>

<script setup lang="ts">
import { defineComponent, h, markRaw, type Component as VueComponent, type PropType } from 'vue'
import type { RouteLocationNormalizedLoaded } from 'vue-router'
import { useContentFullBleed } from '@/composables/useAppLayout'

// 最多缓存的页面数：超出后按最近最少使用淘汰，避免长时间使用后内存一直涨
const PAGE_CACHE_MAX = 12

// keep-alive 只缓存组件，且 Transition 需要单一根元素：用一个带 .app-page 根节点的壳组件包住页面，
// 以 renderKey 作为缓存键（每个路由一个实例；页面文件同名 index.vue 也不会互相冲突）。
const PageFrame = markRaw(
  defineComponent({
    name: 'PageFrame',
    props: { page: { type: [Object, Function] as PropType<VueComponent>, default: undefined } },
    setup(props) {
      return () => h('div', { class: 'app-page' }, props.page ? [h(props.page)] : [])
    },
  }),
)

// 同一组子页（第一段路径相同，如 /docker/*、/system/*）之间切换不做过渡动画；
// 只在路径变化时重新计算，避免过渡进行中重渲染把动画名改掉
const groupOf = (path: string) => path.split('/')[1] || ''
let currentPath = ''
let currentName = 'fade-slide'
const transitionName = (r: RouteLocationNormalizedLoaded) => {
  if (r.path !== currentPath) {
    const sameGroup = !!currentPath && groupOf(r.path) === groupOf(currentPath)
    currentName = r.meta?.disableTransition === true || sameGroup ? '' : 'fade-slide'
    currentPath = r.path
  }
  return currentName
}

const route = useRoute()

// 每个缓存页记住自己的滚动位置：切走时保存，切回时恢复（新页面从顶部开始）
const mainRef = useTemplateRef<HTMLElement>('mainRef')
const scrollPositions = new Map<string, number>()
watch(
  () => route.fullPath,
  (to, from) => {
    const el = mainRef.value
    if (!el) return
    if (from) scrollPositions.set(from, el.scrollTop)
    // 等 out-in 过渡把新页面挂上后再恢复
    const target = scrollPositions.get(to) ?? 0
    const restore = () => {
      if (mainRef.value) mainRef.value.scrollTop = target
    }
    nextTick(restore)
    setTimeout(restore, 260)
  },
)
const tabsStore = useTabsStore()
const menuStore = useMenuStore()
// 全出血：静态路由用 meta.fullBleed（终端页）；动态菜单页由页面自身声明（见 useFullBleed）。
const contentFullBleed = useContentFullBleed()
const isBleed = computed(() => !!route.meta?.fullBleed || contentFullBleed.value)
</script>

<style scoped lang="scss">
.app-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--app-surface-page);
  padding: 16px;
}
@media (width >= 992px) {
  .app-content {
    /* 页面边距跟随主题的组件尺寸（theme/sizeTokens.ts · layout） */
    padding: var(--cmp-page-padding-top) var(--cmp-page-padding-x) 40px;
  }
}
/* 手机底部导航高度 ≈ 70px + 安全区，内容区留出空间 */
.app-content--tabbar {
  padding-top: calc(18px + env(safe-area-inset-top));
  padding-bottom: calc(88px + env(safe-area-inset-bottom));
}
.app-content--bleed.app-content--tabbar {
  padding-bottom: calc(71px + env(safe-area-inset-bottom));
}
/* 手机整屏页面（route.meta.hideTabbar）：没有底部导航，只留安全区 */
.app-content--bleed.app-content--notabbar {
  padding-top: env(safe-area-inset-top);
  padding-bottom: env(safe-area-inset-bottom);
}
.app-page {
  display: flex;
  flex-direction: column;
  min-height: 100%;
  min-width: 0;
  max-width: 1600px;
  margin: 0 auto;
  width: 100%;
}
/* 全出血工具页（文件管理/终端）：内容区自身不滚，由页内双栏/终端主体各自滚，避免树变高把整页（含列表）一起拖走。 */
.app-content--bleed {
  padding: 0;
  overflow: hidden;
}
.app-content--bleed .app-page {
  max-width: none;
  height: 100%;
  min-height: 0;
}
</style>
