<!-- 快捷方式 -->
<template>
  <BaseCard title="便捷工具" title-icon="HOutline:WrenchScrewdriverIcon">
    <div class="shortcuts-grid">
      <div
        v-for="item in shortcuts"
        :key="item.label"
        @click="router.push(item.routePath)"
      >
        <HoverAnimateWrapper name="jelly" intensity="normal">
          <div
            class="group flex flex-col items-center gap-3 py-3 rounded-2xl transition-all duration-300 hover:bg-(--el-bg-color-page)"
          >
            <div
              class="w-14 h-14 bg-(--el-bg-color-page) flex items-center justify-center rounded-2xl transition-all duration-300 group-hover:bg-(--el-bg-color)"
            >
              <el-icon size="26" :style="{ color: item.color }">
                <component :is="menuStore.iconComponents[item.icon]" />
              </el-icon>
            </div>
            <div class="text-[13px] font-bold">
              {{ item.label }}
            </div>
          </div>
        </HoverAnimateWrapper>
      </div>
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useDashboardHomeStore } from '@/stores/dashboard/home'

const router = useRouter()
const menuStore = useMenuStore()
const dashboardHomeStore = useDashboardHomeStore()
const { shortcuts } = storeToRefs(dashboardHomeStore)
</script>

<style scoped lang="scss">
.shortcuts-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;

  @media (width >= 480px) {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  @media (width >= 768px) {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 1.5rem;
  }
}
</style>
