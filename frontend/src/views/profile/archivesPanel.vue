<!-- 详细档案 -->
<template>
  <BaseCard title="详细档案" title-icon="HOutline:IdentificationIcon">
    <div>
      <div class="info-cell">
        <label>用户名</label>
        <span>{{ userStore.userInfo?.username }}</span>
      </div>
      <div class="info-cell">
        <label>邮箱</label>
        <span>{{ userStore.userInfo?.email || '暂无邮箱~' }}</span>
      </div>
      <div class="info-cell">
        <label>账号昵称</label>
        <span>{{ userStore.userInfo?.name }}</span>
      </div>
      <div class="info-cell">
        <label>账号状态</label>
        <span>{{ userStore.userInfo?.status === UserStatus.Active ? '启用' : '禁用' }}</span>
      </div>
      <div class="info-cell">
        <label>加入时间</label>
        <span>{{ createTimeText }}</span>
      </div>
      <el-divider />

      <div>
        <div class="text-sm font-bold text-(--el-text-color-secondary) mb-2">个人标签</div>
        <div class="flex flex-wrap gap-2" v-if="skills.length">
          <BaseTag
            v-for="skill in skills"
            :key="skill.name"
            :type="skill.type"
            :text="skill.name"
          />
        </div>
        <div class="text-sm" v-else>暂无标签~</div>
      </div>
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { UserStatus } from '@/types/v1/user'
import { useUserProfileStore } from '@/stores/user/profile'

const userStore = useUserStore()
const userProfileStore = useUserProfileStore()

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
