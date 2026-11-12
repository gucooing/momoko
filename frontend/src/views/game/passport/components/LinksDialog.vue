<!-- 全部绑定：已经读到的完整列表，按服务器、昵称、游戏 ID 本地筛选。 -->
<template>
  <FormDialog :model-value="modelValue" :title="t('passportAdmin.allLinks.title', { name })" :width="560" :show-footer="false" @update:model-value="emit('update:modelValue', $event)">
    <label class="plinks__search">
      <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" class="plinks__icon" />
      <input v-model="q" type="search" :placeholder="t('passportAdmin.allLinks.search')" :aria-label="t('passportAdmin.allLinks.search')" />
    </label>
    <div v-if="filtered.length" class="plinks__list">
      <LinkRow v-for="l in filtered" :key="l.id" :link="l" @unbind="emit('unbind', $event)" />
    </div>
    <EmptyState v-else icon="HOutline:LinkIcon" :title="t('passportAdmin.allLinks.empty')" />
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PassportAdminLink } from '@/types/v1/passport_admin'
import LinkRow from './LinkRow.vue'

const props = defineProps<{ modelValue: boolean; name: string; links: PassportAdminLink[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; unbind: [link: PassportAdminLink] }>()

const { t } = useI18n()
const menuStore = useMenuStore()
const q = ref('')

watch(
  () => props.modelValue,
  (open) => {
    if (open) q.value = ''
  },
)

const filtered = computed(() => {
  const k = q.value.trim().toLowerCase()
  if (!k) return props.links
  return props.links.filter((l) =>
    [l.server?.name, l.server?.gameName, l.server?.game, l.nickname, l.displayId, l.account, l.playerId]
      .filter(Boolean)
      .some((v) => String(v).toLowerCase().includes(k)),
  )
})
</script>

<style scoped>
.plinks__search {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 42px;
  padding: 0 14px;
  margin-bottom: 12px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.plinks__search:focus-within {
  border-color: var(--m-p);
}
.plinks__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
}
.plinks__icon {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.plinks__list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
