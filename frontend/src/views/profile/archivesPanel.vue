<!-- 详细档案：顶栏横铺（horizontal）或侧栏单列；只读 DescriptionList + 标签 -->
<template>
  <AppPanel
    :title="t('user.detailArchive')"
    title-icon="HOutline:IdentificationIcon"
    class="archive"
    :class="{ 'archive--horizontal': horizontal }"
  >
    <DescriptionList :items="archiveItems" :columns="listColumns" />

    <div class="archive-tags">
      <div class="archive-tags__label">{{ t('user.personalTags') }}</div>
      <div v-if="tagNames.length" class="archive-tags__list">
        <span v-for="name in tagNames" :key="name" class="archive-chip">{{ name }}</span>
      </div>
      <div v-else class="archive-tags__empty">{{ t('user.noTags') }}</div>
    </div>
  </AppPanel>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { UserStatus } from '@/types/v1/user'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    /** 顶栏横铺：多列；侧栏竖排：1 列 */
    horizontal?: boolean
  }>(),
  { horizontal: false },
)

const userStore = useUserStore()
const menuStore = useMenuStore()
const { t } = useI18n()

const createTimeText = computed(() => {
  if (!userStore.userInfo?.createTime) return ''
  return dayjs(userStore.userInfo.createTime).format('YYYY-MM-DD HH:mm:ss')
})

const archiveItems = computed(() => [
  { label: t('user.username'), value: userStore.userInfo?.username },
  { label: t('user.email'), value: userStore.userInfo?.email || t('user.noEmail') },
  { label: t('user.nickname'), value: userStore.userInfo?.name },
  {
    label: t('user.accountStatus'),
    value:
      userStore.userInfo?.status === UserStatus.Active ? t('common.enabled') : t('common.disabled'),
  },
  { label: t('user.joinTime'), value: createTimeText.value },
])

const listColumns = computed(() => {
  if (!props.horizontal) return 1
  // 横铺：宽屏 3～4 列，窄屏 1～2 列
  if (menuStore.isMobile) return 1
  return 3
})

const tagNames = computed(() => {
  const raw = userStore.userInfo?.tags?.trim()
  if (!raw) return [] as string[]
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
})
</script>

<style scoped lang="scss">
.archive {
  width: 100%;
  min-width: 0;
}
.archive--horizontal :deep(.app-panel__body) {
  padding-bottom: 16px;
}
.archive-tags {
  margin-top: 0.85rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--el-border-color-lighter);
}
.archive-tags__label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 0.4rem;
}
.archive-tags__list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}
.archive-chip {
  display: inline-flex;
  align-items: center;
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-extra-light);
}
.archive-tags__empty {
  font-size: 0.8125rem;
  color: var(--el-text-color-placeholder);
}
</style>
