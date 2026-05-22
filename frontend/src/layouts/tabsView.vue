<template>
  <div class="tabs-container">
    <HoverAnimateWrapper name="rubber">
      <IconButton
        icon="HOutline:ChevronLeftIcon"
        size="1.75rem"
        tooltip="向左滑动"
        @click="slideLeft"
      />
    </HoverAnimateWrapper>

    <div class="tabs-pages" ref="tabsPagesRef">
      <div
        v-for="tab in tabsStore.tabs"
        :key="tab.fullPath"
        class="tabs-page-item"
        :class="getTabClassState(tab.fullPath)"
        :style="getTabStyle(tab.fullPath)"
        :ref="(el) => setTabRef(el, tab.fullPath)"
        @pointerdown="handlePointerDown($event, tab.fullPath)"
        @click="navigation(tab.fullPath)"
      >
        <HoverAnimateWrapper name="wobble" :duration="700">
          <div class="tabs-page-content">
            <el-icon class="tabs-page-icon" size="18">
              <component :is="menuStore.iconComponents[tab.icon as string]" />
            </el-icon>
            <div>{{ tab.title }}</div>
            <el-icon
              v-if="tab.closable"
              class="close-icon"
              @pointerdown.stop
              @click.stop="handleClose(tab)"
            >
              <component :is="menuStore.iconComponents['HSolid:XMarkIcon']" />
            </el-icon>
          </div>
        </HoverAnimateWrapper>
      </div>
    </div>

    <HoverAnimateWrapper name="rubber">
      <IconButton
        icon="HOutline:ChevronRightIcon"
        size="1.75rem"
        tooltip="向右滑动"
        @click="slideRight"
      />
    </HoverAnimateWrapper>

    <div class="tabs-refresh">
      <HoverAnimateWrapper name="rotate">
        <IconButton
          icon="HOutline:ArrowPathIcon"
          size="1.75rem"
          tooltip="刷新当前页面"
          @click="handleRefreshCurrentPage"
        />
      </HoverAnimateWrapper>
    </div>

    <div class="tabs-dropdown">
      <el-dropdown
        trigger="click"
        :show-arrow="false"
        class="tabs-dropdown-wrapper"
        popper-class="tabs-dropdown-popper"
      >
        <div class="tabs-dropdown-icon">
          <HoverAnimateWrapper name="rubber">
            <IconButton icon="HOutline:EllipsisHorizontalIcon" size="1.75rem" />
          </HoverAnimateWrapper>
        </div>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item
              :icon="menuStore.iconComponents['HOutline:MinusCircleIcon']"
              @click="tabsStore.closeOtherTabs(tabsStore.activePath)"
            >
              关闭其他标签页
            </el-dropdown-item>
            <el-dropdown-item
              :icon="menuStore.iconComponents['HOutline:TrashIcon']"
              @click="(tabsStore.closeAllTabs(), router.push(tabsStore.activePath))"
            >
              关闭所有标签页
            </el-dropdown-item>
            <el-dropdown-item
              :icon="menuStore.iconComponents['HOutline:ChevronDoubleRightIcon']"
              @click="tabsStore.closeRightTabs(tabsStore.activePath)"
            >
              关闭右侧标签页
            </el-dropdown-item>
            <el-dropdown-item
              :icon="menuStore.iconComponents['HOutline:ChevronDoubleLeftIcon']"
              @click="tabsStore.closeLeftTabs(tabsStore.activePath)"
            >
              关闭左侧标签页
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { TabItem } from '@/stores/tabs'

defineOptions({ name: 'TabsView' })

const router = useRouter()
const menuStore = useMenuStore()
const tabsStore = useTabsStore()
const tabsPagesRef = useTemplateRef<HTMLDivElement>('tabsPagesRef')
const DRAG_TRIGGER_DISTANCE = 6
const EDGE_SCROLL_THRESHOLD = 72
const EDGE_SCROLL_STEP = 18
const SCROLL_STEP_RATIO = 0.8

const tabRefs = new Map<string, HTMLDivElement>()
const pressingPath = ref('')
const draggingPath = ref('')
const dragOffsetX = ref(0)
const dragCompensationX = ref(0)
const suppressClick = ref(false)

let activePointerId: number | null = null
let pointerStartX = 0

const setTabRef = (el: Element | ComponentPublicInstance | null, fullPath: string) => {
  if (el && el instanceof HTMLElement) {
    tabRefs.set(fullPath, el as HTMLDivElement)
  } else {
    tabRefs.delete(fullPath)
  }
}

const scrollToActiveTab = () => {
  nextTick(() => {
    const activeTab = tabRefs.get(tabsStore.activePath)
    const container = tabsPagesRef.value
    if (!activeTab || !container) return

    const containerRect = container.getBoundingClientRect()
    const tabRect = activeTab.getBoundingClientRect()
    const isVisible = tabRect.left >= containerRect.left && tabRect.right <= containerRect.right

    if (!isVisible) {
      if (tabRect.left < containerRect.left) {
        container.scrollTo({
          left: container.scrollLeft + (tabRect.left - containerRect.left) - 10,
          behavior: 'smooth',
        })
      } else if (tabRect.right > containerRect.right) {
        container.scrollTo({
          left: container.scrollLeft + (tabRect.right - containerRect.right) + 10,
          behavior: 'smooth',
        })
      }
    }
  })
}

watch(
  () => tabsStore.activePath,
  () => {
    scrollToActiveTab()
  },
  { immediate: true },
)

watch(
  () => tabsStore.tabs.length,
  () => {
    scrollToActiveTab()
  },
)

const getTabClassState = (fullPath: string) => ({
  active: fullPath === tabsStore.activePath,
  'is-pressing': fullPath === pressingPath.value,
  'is-dragging': fullPath === draggingPath.value,
  'is-fixed': !tabsStore.tabs.find((tab) => tab.fullPath === fullPath)?.closable,
})

const getTabStyle = (fullPath: string) => ({
  '--tabs-drag-x':
    fullPath === draggingPath.value ? `${dragOffsetX.value + dragCompensationX.value}px` : '0px',
})

const setDraggingBodyState = (dragging: boolean) => {
  document.body.classList.toggle('tabs-is-dragging', dragging)
}

const removeGlobalPointerListeners = () => {
  window.removeEventListener('pointermove', handleGlobalPointerMove)
  window.removeEventListener('pointerup', handleGlobalPointerUp)
  window.removeEventListener('pointercancel', handleGlobalPointerUp)
}

const resetPointerState = () => {
  removeGlobalPointerListeners()
  setDraggingBodyState(false)
  pressingPath.value = ''
  draggingPath.value = ''
  dragOffsetX.value = 0
  dragCompensationX.value = 0
  activePointerId = null
}

const autoScrollWhenDragging = (clientX: number) => {
  if (!draggingPath.value) return

  const container = tabsPagesRef.value
  if (!container) return

  const rect = container.getBoundingClientRect()

  if (clientX < rect.left + EDGE_SCROLL_THRESHOLD) {
    container.scrollLeft -= EDGE_SCROLL_STEP
  } else if (clientX > rect.right - EDGE_SCROLL_THRESHOLD) {
    container.scrollLeft += EDGE_SCROLL_STEP
  }
}

const reorderTabsByPointer = async (clientX: number) => {
  if (!draggingPath.value) return

  const draggingElement = tabRefs.get(draggingPath.value)
  const previousLeft = draggingElement?.getBoundingClientRect().left ?? 0
  const otherTabs = tabsStore.tabs.filter((tab) => tab.fullPath !== draggingPath.value)
  let targetIndex = otherTabs.length

  for (let index = 0; index < otherTabs.length; index += 1) {
    const fullPath = otherTabs[index]!.fullPath
    const element = tabRefs.get(fullPath)
    if (!element) continue

    const rect = element.getBoundingClientRect()
    if (clientX < rect.left + rect.width / 2) {
      targetIndex = index
      break
    }
  }

  const currentIndex = tabsStore.tabs.findIndex((tab) => tab.fullPath === draggingPath.value)
  if (currentIndex === targetIndex) return

  tabsStore.moveTab(draggingPath.value, targetIndex)
  await nextTick()

  const nextDraggingElement = tabRefs.get(draggingPath.value)
  const nextLeft = nextDraggingElement?.getBoundingClientRect().left ?? previousLeft
  dragCompensationX.value += previousLeft - nextLeft
}

const handleGlobalPointerMove = (event: PointerEvent) => {
  if (event.pointerId !== activePointerId) return

  const deltaX = event.clientX - pointerStartX

  if (!draggingPath.value) {
    if (Math.abs(deltaX) < DRAG_TRIGGER_DISTANCE) return

    draggingPath.value = pressingPath.value
    suppressClick.value = true
    setDraggingBodyState(true)
  }

  dragOffsetX.value = deltaX
  autoScrollWhenDragging(event.clientX)
  void reorderTabsByPointer(event.clientX)
}

const finishDragging = () => {
  if (draggingPath.value) {
    dragOffsetX.value = 0
    scrollToActiveTab()
  }

  resetPointerState()
  window.setTimeout(() => {
    suppressClick.value = false
  }, 0)
}

const handleGlobalPointerUp = (event: PointerEvent) => {
  if (event.pointerId !== activePointerId) return
  finishDragging()
}

const handlePointerDown = (event: PointerEvent, fullPath: string) => {
  if (event.button !== 0) return
  if (!tabsStore.tabs.find((tab) => tab.fullPath === fullPath)?.closable) return

  suppressClick.value = false
  activePointerId = event.pointerId
  pointerStartX = event.clientX
  pressingPath.value = fullPath

  removeGlobalPointerListeners()
  window.addEventListener('pointermove', handleGlobalPointerMove)
  window.addEventListener('pointerup', handleGlobalPointerUp)
  window.addEventListener('pointercancel', handleGlobalPointerUp)
}

const navigation = (fullPath: string) => {
  if (suppressClick.value) return

  router.push(fullPath)
  tabsStore.activePath = fullPath
  scrollToActiveTab()
}

const handleClose = (item: TabItem) => {
  tabsStore.removeTab(item.fullPath)
  router.push(tabsStore.activePath)
  scrollToActiveTab()
}

const handleRefreshCurrentPage = () => {
  tabsStore.refreshTab(tabsStore.activePath || router.currentRoute.value.fullPath)
}

const getScrollInfo = () => {
  const container = tabsPagesRef.value
  if (!container) return null

  return {
    container,
    containerWidth: container.offsetWidth,
    contentWidth: container.scrollWidth,
    scrollLeft: container.scrollLeft,
    maxScrollLeft: container.scrollWidth - container.offsetWidth,
  }
}

const slideLeft = () => {
  const info = getScrollInfo()
  if (!info) return
  if (info.containerWidth >= info.contentWidth) return

  const scrollDistance = info.containerWidth * SCROLL_STEP_RATIO
  const targetScrollLeft = Math.max(0, info.scrollLeft - scrollDistance)
  if (info.scrollLeft <= 0) return

  info.container.scrollTo({
    left: targetScrollLeft,
    behavior: 'smooth',
  })
}

const slideRight = () => {
  const info = getScrollInfo()
  if (!info) return
  if (info.containerWidth >= info.contentWidth) return

  const scrollDistance = info.containerWidth * SCROLL_STEP_RATIO
  const targetScrollLeft = Math.min(info.maxScrollLeft, info.scrollLeft + scrollDistance)
  if (info.scrollLeft >= info.maxScrollLeft) return

  info.container.scrollTo({
    left: targetScrollLeft,
    behavior: 'smooth',
  })
}

onBeforeUnmount(() => {
  resetPointerState()
})
</script>

<style scoped lang="scss">
:deep(.el-dropdown-menu__item .el-icon) {
  font-size: 1.125rem;
}

.tabs-container {
  --tabs-bar-bg: transparent;
  --tabs-active-bg: color-mix(in srgb, var(--el-color-primary) 20%, transparent);
  --tabs-active-text-color: var(--el-color-primary);

  background-color: var(--tabs-bar-bg);
  padding-top: 0.1rem;
  height: 2.5rem;
  padding-left: 1.25rem;
  display: flex;
  align-items: center;

  .tabs-pages {
    padding: 0 1.25rem;
    height: 2.5rem;
    flex: 1;
    display: flex;
    font-size: 0.875rem;
    overflow-x: auto;
    gap: 0.2rem;

    &::-webkit-scrollbar {
      display: none;
    }

    .tabs-page-item {
      padding: 0 0.75rem;
      display: flex;
      align-items: center;
      flex-shrink: 0;
      gap: 0.5rem;
      cursor: grab;
      color: var(--el-text-color-regular);
      user-select: none;
      transform: translateX(var(--tabs-drag-x, 0));
      transition:
        transform 0.2s ease,
        box-shadow 0.2s ease,
        background-color 0.2s ease,
        opacity 0.2s ease;

      &.is-fixed {
        cursor: default;
      }

      &.is-pressing {
        background-color: color-mix(in srgb, var(--el-color-primary) 7%, transparent);
        box-shadow: 0 6px 18px rgb(59 130 246 / 10%);
      }

      &.is-dragging {
        cursor: grabbing;
        z-index: 8;
        opacity: 0.96;
        box-shadow: 0 14px 30px rgb(15 23 42 / 18%);
        transition: none;
      }

      .tabs-page-content {
        display: flex;
        align-items: center;
        gap: 0.5rem;
      }

      .close-icon {
        margin-left: 0.25rem;
        font-size: 0.75rem;
        width: 0.875rem;
        height: 0.875rem;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 20%;
        transition: all 0.2s ease;
        flex-shrink: 0;
        color: var(--el-text-color-regular);
        cursor: pointer;

        &:hover {
          background-color: var(--el-fill-color-darker);
          color: var(--el-color-danger);
          transform: scale(1.1);
        }
      }

      &:hover {
        background-color: var(--el-fill-color-light);
        border-radius: 0.625rem 0.625rem 0.875rem 0.875rem;
        font-weight: bold;
      }

      &.active {
        position: relative;
        border-radius: 0.625rem 0.625rem 0 0;
        background-color: var(--tabs-active-bg);
        color: var(--tabs-active-text-color);
        font-weight: bold;

        &::before {
          content: '';
          position: absolute;
          width: 20px;
          height: 20px;
          left: -20px;
          bottom: 0;
          background: #000;
          background: radial-gradient(
            circle at 0 0,
            transparent 20px,
            var(--tabs-active-bg) 21px
          );
        }

        &::after {
          content: '';
          position: absolute;
          width: 20px;
          height: 20px;
          right: -20px;
          bottom: 0;
          background: #000;
          background: radial-gradient(
            circle at 100% 0,
            transparent 20px,
            var(--tabs-active-bg) 21px
          );
        }

        &.is-dragging {
          border-radius: 0.625rem;

          &::before,
          &::after {
            display: none;
          }
        }
      }
    }
  }

  .tabs-dropdown {
    height: 100%;

    .tabs-dropdown-wrapper {
      height: 100%;
      cursor: pointer;
      margin-right: 0.5rem;

      .tabs-dropdown-icon {
        display: flex;
        align-items: center;
        color: var(--el-text-color-regular);

        &:hover {
          color: var(--el-color-primary);
        }
      }
    }
  }

  .tabs-refresh {
    display: flex;
    align-items: center;
  }
}

:global(html.dark .tabs-container) {
  --tabs-bar-bg: var(--el-bg-color);
  --tabs-active-bg: var(--el-fill-color-light);
  --tabs-active-text-color: var(--el-text-color-primary);
}

:deep(.el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  transition:
    background-color 0.2s,
    color 0.2s;

  &:hover {
    background: var(--el-fill-color-light) !important;
    color: var(--el-color-primary);
  }

  &:focus,
  &:focus-visible {
    background: var(--el-fill-color-light) !important;
    color: var(--el-color-primary);
  }
}
</style>

<style lang="scss">
.tabs-dropdown-popper {
  border-radius: 8px !important;

  .el-dropdown-menu {
    border-radius: 8px !important;
  }
}

body.tabs-is-dragging {
  cursor: grabbing;
  user-select: none;
}
</style>
