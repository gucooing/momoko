<template>
  <template v-if="!isButton">
    <el-sub-menu v-if="visibleChildren.length" :index="item.id">
      <template #title>
        <el-icon v-if="item.icon">
          <component :is="menuStore.iconComponents[item.icon]" />
        </el-icon>
        <span>{{ item.title }}</span>
      </template>
      <MenuItem v-for="child in visibleChildren" :key="child.id" :item="child" />
    </el-sub-menu>

    <el-menu-item v-else :index="item.path">
      <el-icon v-if="item.icon">
        <component :is="menuStore.iconComponents[item.icon]" />
      </el-icon>
      <span>{{ item.title }}</span>
    </el-menu-item>
  </template>
</template>

<script setup lang="ts">
import { MenuStatus, MenuType, type MenuInfo } from '@/types/v1/system'

const props = defineProps<{ item: MenuInfo }>()

const menuStore = useMenuStore()

const isButton = computed(() => props.item.type === MenuType.MenuType_Button)

const visibleChildren = computed(() =>
  (props.item.children ?? []).filter((child) => {
    return child.type !== MenuType.MenuType_Button && child.status === MenuStatus.MenuStatus_Active
  }),
)
</script>
