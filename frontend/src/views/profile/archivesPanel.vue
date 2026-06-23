<!-- 详细档案 -->
<template>
  <BaseCard :title="t('user.detailArchive')" title-icon="HOutline:IdentificationIcon">
    <div>
      <div class="info-cell">
        <label>{{ t('user.username') }}</label>
        <span>{{ userStore.userInfo?.username }}</span>
      </div>
      <div class="info-cell">
        <label>{{ t('user.email') }}</label>
        <span>{{ userStore.userInfo?.email || t('user.noEmail') }}</span>
      </div>
      <div class="info-cell">
        <label>{{ t('user.nickname') }}</label>
        <span>{{ userStore.userInfo?.name }}</span>
      </div>
      <div class="info-cell">
        <label>{{ t('user.accountStatus') }}</label>
        <span>{{ userStore.userInfo?.status === UserStatus.Active ? t('common.enabled') : t('common.disabled') }}</span>
      </div>
      <div class="info-cell">
        <label>{{ t('user.joinTime') }}</label>
        <span>{{ createTimeText }}</span>
      </div>
      <el-divider />

      <div>
        <div class="text-sm font-bold text-(--el-text-color-secondary) mb-2">{{ t('user.personalTags') }}</div>
        <div class="flex flex-wrap gap-2" v-if="skills.length">
          <BaseTag
            v-for="skill in skills"
            :key="skill.name"
            :type="skill.type"
            :text="skill.name"
          />
        </div>
        <div class="text-sm" v-else>{{ t('user.noTags') }}</div>
      </div>
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { UserStatus } from '@/types/v1/user'
import { useUserProfileStore } from '@/stores/user/profile'
import { useI18n } from 'vue-i18n'

const userStore = useUserStore()
const userProfileStore = useUserProfileStore()
const { t } = useI18n()

const createTimeText = computed(() => {
  if (!userStore.userInfo?.createTime) return ''
  return dayjs(userStore.userInfo.createTime).format('YYYY-MM-DD HH:mm:ss')
})

const skills = computed(() => {
  return userProfileStore.buildSkillTags(userStore.userInfo?.tags)
})
</script>

<style scoped lang="scss">
.info-cell {
  margin-bottom: 1.25rem;
  label {
    display: block;
    font-size: 14px;
    color: var(--el-text-color-secondary);
    margin-bottom: 0.25rem;
  }
  span {
    font-size: 14px;
    font-weight: 600;
  }
  &:last-child {
    margin-bottom: 0;
  }
}
</style>
