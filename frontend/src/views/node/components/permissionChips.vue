<!-- 节点权限标签组：按权限目录顺序展示；空时提示“未授予任何权限”。 -->
<template>
  <div class="perm-chips">
    <span v-for="item in items" :key="item.code" class="perm-chip" :title="item.code">
      {{ t(item.labelKey) }}
    </span>
    <span v-if="!items.length" class="perm-chips__empty">{{ t('node.permission.none') }}</span>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NODE_PERMISSION_ITEMS } from '@/config/nodePermission'

const props = defineProps<{ permissions?: string[] }>()
const { t } = useI18n()

const items = computed(() => {
  const granted = new Set(props.permissions || [])
  return NODE_PERMISSION_ITEMS.filter((item) => granted.has(item.code))
})
</script>

<style scoped lang="scss">
.perm-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.perm-chip {
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 0.75rem;
  line-height: 1.6;
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
  white-space: nowrap;
}
.perm-chips__empty {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
</style>
