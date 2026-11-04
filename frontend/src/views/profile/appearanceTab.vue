<!-- 个人中心 · 外观（ThemePC / ThemeMobile）
     PC：左栏 配色 / 深浅色 / 细节 / 自动保存与恢复默认，右栏吸顶的实时预览；
     手机：顶部主色预览 → 两列配色 → 深浅色分段 → 细节开关 → 状态示例与保存说明（「恢复默认」在页头）。 -->
<template>
  <div v-if="!isMobile" class="appr">
    <AppearancePanel class="appr__main" variant="page" />
    <AppearancePreview class="appr__side" variant="card" />
  </div>
  <div v-else class="appr-m">
    <AppearancePreview variant="hero" />
    <AppearancePanel variant="mobile" />
  </div>
</template>

<script setup lang="ts">
import AppearancePanel from '@/components/theme/AppearancePanel.vue'
import AppearancePreview from '@/components/theme/AppearancePreview.vue'

defineOptions({ name: 'ProfileAppearanceTab' })

const menuStore = useMenuStore()
const isMobile = computed(() => menuStore.isMobile)
</script>

<style scoped lang="scss">
.appr {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 24px;
}
.appr__main {
  flex: 3 1 560px;
  min-width: 0;
}
.appr__side {
  flex: 2 1 340px;
  position: sticky;
  top: 0;
}
.appr-m {
  display: flex;
  flex-direction: column;
  gap: 22px;
}
</style>
