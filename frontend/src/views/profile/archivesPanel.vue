<!-- 详细档案：只读 DescriptionList + 中性标签 chips -->
<template>
  <AppPanel :title="t('user.detailArchive')" title-icon="HOutline:IdentificationIcon">
    <DescriptionList :items="archiveItems" :columns="1" />

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

const userStore = useUserStore()
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
.archive-tags {
  margin-top: 1rem;
  padding-top: 0.85rem;
  border-top: 1px solid var(--el-border-color-lighter);
}
.archive-tags__label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 0.45rem;
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
