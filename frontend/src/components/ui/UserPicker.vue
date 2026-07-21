<!-- 用户多选选择器（令牌驱动）：一次性预载全部用户 → 已选 chips（可移除）+ 可搜索 AppSelect 追加。
     v-model = 选中的 userId[]；用于 SSH 分享用户、连接分享等需要多选用户的场景（替代 el-select multiple remote）。 -->
<template>
  <div class="user-picker">
    <div v-if="modelValue.length" class="user-picker__chips">
      <span v-for="id in modelValue" :key="id" class="user-picker__chip">
        {{ nameOf(id) }}
        <button type="button" class="user-picker__chip-x" :aria-label="t('ssh.common.delete')" @click="remove(id)">
          <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
        </button>
      </span>
    </div>
    <span v-else class="user-picker__empty">{{ emptyText || t('ssh.common.noSharedUsers') }}</span>

    <AppSelect
      :model-value="''"
      :options="availableOptions"
      searchable
      :placeholder="placeholder || t('ssh.common.addUser')"
      :search-placeholder="t('ssh.common.searchUser')"
      @update:model-value="onAdd"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { userPage } from '@/api/user'
import type { UserInfo } from '@/types/v1/user'

defineOptions({ name: 'UserPicker' })

const props = defineProps<{
  modelValue: string[]
  placeholder?: string
  emptyText?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()

const menuStore = useMenuStore()
const { t } = useI18n()

const allUsers = ref<UserInfo[]>([])

const nameOf = (userId: string): string => {
  const u = allUsers.value.find((x) => x.userId === userId)
  return u?.name || u?.username || userId
}

const availableOptions = computed(() =>
  allUsers.value
    .filter((u) => !props.modelValue.includes(u.userId))
    .map((u) => ({ label: `${u.name || u.username} (${u.username})`, value: u.userId })),
)

const onAdd = (userId: string) => {
  if (!userId || props.modelValue.includes(userId)) return
  emit('update:modelValue', [...props.modelValue, userId])
}

const remove = (userId: string) => {
  emit(
    'update:modelValue',
    props.modelValue.filter((id) => id !== userId),
  )
}

const loadUsers = async () => {
  const { data } = await userPage({ page: 1, pageSize: 1000 })
  allUsers.value = data?.users || []
}

onMounted(loadUsers)
</script>

<style scoped lang="scss">
.user-picker {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}
.user-picker__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.user-picker__chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px 2px 10px;
  font-size: 0.75rem;
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
  border-radius: 999px;
}
.user-picker__chip-x {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: none;
  background: transparent;
  color: inherit;
  cursor: pointer;
  border-radius: 999px;
  opacity: 0.7;
  transition: opacity 0.15s, background 0.15s, box-shadow 0.15s;
}
.user-picker__chip-x::before {
  content: '';
  position: absolute;
  inset: -10px;
}
.user-picker__chip-x:hover {
  opacity: 1;
  background: color-mix(in srgb, var(--el-color-primary) 20%, transparent);
}
.user-picker__chip-x:focus-visible {
  outline: none;
  opacity: 1;
  box-shadow: var(--app-focus-ring, 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 25%, transparent));
}
.user-picker__chip-x :deep(svg) {
  width: 12px;
  height: 12px;
}
.user-picker__empty {
  font-size: 0.8125rem;
  color: var(--el-text-color-placeholder);
}
</style>
