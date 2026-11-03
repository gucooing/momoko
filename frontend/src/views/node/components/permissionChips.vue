<!-- 节点权限标签组。
     variant="items"（默认）：按权限目录顺序逐项展示已授予的功能；空时提示“未授予任何功能”。
     variant="groups"：按功能分组展示（应用 / 文件 / 容器 / 工具…），整组授予打勾、部分授予显示 n/m、未授予置灰，
     用于卡片与表格里的紧凑展示；鼠标悬停可看到组内已授予的具体项。 -->
<template>
  <div class="perm-chips">
    <template v-if="variant === 'groups'">
      <span
        v-for="g in shownGroups"
        :key="g.key"
        class="perm-chip perm-chip--group"
        :class="`is-${g.state}`"
        :title="g.title"
      >
        <UIcon :name="g.state === 'off' ? 'i-lucide-x' : 'i-lucide-check'" class="perm-chip__icon" />
        {{ t(g.labelKey) }}
        <span v-if="g.state === 'part'" class="perm-chip__n m-num">{{ g.granted }}/{{ g.total }}</span>
      </span>
      <span v-if="!shownGroups.length" class="perm-chips__empty">{{ t('node.permission.none') }}</span>
    </template>
    <template v-else>
      <span v-for="item in items" :key="item.code" class="perm-chip" :title="item.code">
        {{ t(item.labelKey) }}
      </span>
      <span v-if="!items.length" class="perm-chips__empty">{{ t('node.permission.none') }}</span>
    </template>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NODE_PERMISSION_GROUPS, NODE_PERMISSION_ITEMS } from '@/config/nodePermission'

const props = withDefaults(
  defineProps<{ permissions?: string[]; variant?: 'items' | 'groups'; onlyGranted?: boolean }>(),
  { permissions: () => [], variant: 'items', onlyGranted: false },
)
const { t } = useI18n()

const granted = computed(() => new Set(props.permissions || []))

const items = computed(() => NODE_PERMISSION_ITEMS.filter((item) => granted.value.has(item.code)))

const groups = computed(() =>
  NODE_PERMISSION_GROUPS.map((g) => {
    const on = g.items.filter((i) => granted.value.has(i.code))
    const state = on.length === 0 ? 'off' : on.length === g.items.length ? 'on' : 'part'
    return {
      key: g.key,
      labelKey: g.labelKey,
      state,
      granted: on.length,
      total: g.items.length,
      title: on.length ? on.map((i) => t(i.labelKey)).join('、') : t('node.permission.none'),
    }
  }),
)

// onlyGranted：紧凑展示（表格 / 手机卡片），只列出已授予（含部分授予）的分组
const shownGroups = computed(() => (props.onlyGranted ? groups.value.filter((g) => g.state !== 'off') : groups.value))
</script>

<style scoped lang="scss">
.perm-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}
.perm-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--m-pInk);
  background: var(--m-pSoft);
  white-space: nowrap;
}
.perm-chip--group.is-on {
  color: var(--m-ok);
  background: var(--m-okSoft);
}
.perm-chip--group.is-part {
  color: var(--m-ink);
  background: var(--m-aSoft);
}
.perm-chip--group.is-off {
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  text-decoration: line-through;
  text-decoration-color: color-mix(in srgb, currentColor 45%, transparent);
}
.perm-chip__icon {
  width: 13px;
  height: 13px;
  flex: none;
}
.perm-chip__n {
  font-size: 0.6875rem;
  opacity: 0.8;
}
.perm-chips__empty {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
</style>
