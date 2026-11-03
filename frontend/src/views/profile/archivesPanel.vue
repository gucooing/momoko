<!-- 个人中心 hero：主色实底 + 黄圆 + 紫环装饰；头像（可更换）/ 昵称 / 邮箱 / 角色 / 标签 / 档案摘要 -->
<template>
  <section class="pf-hero" :aria-label="t('profile.hero.label')">
    <span class="pf-hero__blob" aria-hidden="true" />
    <span class="pf-hero__ring" aria-hidden="true" />

    <div class="pf-hero__body">
      <div class="pf-hero__avatar-col">
        <span class="pf-hero__avatar">
          <AppAvatar :src="userStore.resolvedUserAvatar" :name="displayName" :size="avatarSize" />
        </span>
        <button
          type="button"
          class="pf-hero__cam"
          :class="{ 'is-round': compact }"
          :aria-label="t('user.changeAvatar')"
          :disabled="savingAvatar"
          @click="selectAvatarDialogRef?.showDialog()"
        >
          <UIcon :name="savingAvatar ? 'i-lucide-loader-circle' : 'i-lucide-camera'" :class="{ 'is-spin': savingAvatar }" />
          <span v-if="!compact">{{ t('user.changeAvatar') }}</span>
        </button>
      </div>

      <div class="pf-hero__text">
        <h2 class="pf-hero__name" :title="displayName">{{ displayName }}</h2>
        <div class="pf-hero__mail">
          <UIcon name="i-lucide-mail" aria-hidden="true" />
          <span>{{ userStore.userInfo?.email || t('user.noEmail') }}</span>
        </div>
        <div class="pf-hero__chips">
          <span class="pf-hero__role">
            <UIcon name="i-lucide-shield-check" aria-hidden="true" />{{ roleLabel }}
          </span>
          <span v-if="!isActive" class="pf-hero__chip">{{ t('common.disabled') }}</span>
          <span v-for="name in tagNames" :key="name" class="pf-hero__chip">{{ name }}</span>
        </div>
        <p v-if="bioText" class="pf-hero__bio">{{ bioText }}</p>
        <div class="pf-hero__meta m-num">
          <span>{{ t('profile.hero.username', { name: userStore.userInfo?.username || '—' }) }}</span>
          <span v-if="joinDate">{{ t('profile.hero.joined', { date: joinDate }) }}</span>
          <span v-if="lastLogin && !compact">{{ t('profile.hero.lastLogin', { time: lastLogin }) }}</span>
        </div>
      </div>
    </div>

    <SelectAvatarDialog ref="selectAvatarDialogRef" @get-avatar="onPickAvatar" />
  </section>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { UserStatus } from '@/types/v1/user'
import { useUserProfileStore } from '@/stores/user/profile'
import SelectAvatarDialog from '@/components/dialog/SelectAvatarDialog.vue'
import { getRequestErrorMessage } from '@/utils/request'
import { useFeedback } from '@/utils/feedback'
import { useI18n } from 'vue-i18n'
import { useProfileFormat } from './components/useProfileFormat'

const userStore = useUserStore()
const menuStore = useMenuStore()
const userProfileStore = useUserProfileStore()
const { t } = useI18n()
const fb = useFeedback()
const { relTime } = useProfileFormat()

const compact = computed(() => menuStore.isMobile)
const avatarSize = computed(() => (compact.value ? 76 : 94))
const selectAvatarDialogRef = useTemplateRef('selectAvatarDialogRef')
const savingAvatar = ref(false)

const displayName = computed(
  () => userStore.userInfo?.name || userStore.userInfo?.username || t('user.messages.unknownUser'),
)
const roleLabel = computed(() => userStore.getUserRoleName() || t('user.noPermission'))
const isActive = computed(() => userStore.userInfo?.status === UserStatus.Active)
const bioText = computed(() => {
  const bio = userStore.userInfo?.bio?.trim()
  return bio && bio !== t('user.defaultBio') ? bio : ''
})
const joinDate = computed(() =>
  userStore.userInfo?.createTime ? dayjs(userStore.userInfo.createTime).format('YYYY-MM-DD') : '',
)
const lastLogin = computed(() =>
  userStore.userInfo?.lastLoginTime ? relTime(userStore.userInfo.lastLoginTime) : '',
)
const tagNames = computed(() => {
  const raw = userStore.userInfo?.tags?.trim()
  if (!raw) return [] as string[]
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
})

/** 选中头像后直接保存（其余资料字段沿用当前值） */
const onPickAvatar = async (avatar: string) => {
  userProfileStore.syncProfileForm(userStore.userInfo)
  userProfileStore.setProfileAvatar(avatar)
  savingAvatar.value = true
  try {
    await userStore.updateUserProfile(userProfileStore.profileForm)
  } catch (error) {
    fb.error(getRequestErrorMessage(error, t('user.profileUpdateFailed')))
  } finally {
    savingAvatar.value = false
  }
}
</script>

<style scoped lang="scss">
.pf-hero {
  position: relative;
  overflow: hidden;
  padding: 26px 28px;
  border-radius: var(--app-radius-xl);
  background: var(--m-p);
  color: var(--m-pOn);
}
.pf-hero__blob {
  position: absolute;
  right: -60px;
  top: -80px;
  width: 220px;
  height: 220px;
  border-radius: 50%;
  background: var(--m-a);
}
.pf-hero__ring {
  position: absolute;
  right: 96px;
  bottom: -34px;
  width: 74px;
  height: 74px;
  border-radius: 50%;
  border: 12px solid var(--m-a2);
}
.pf-hero__body {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 24px;
}
.pf-hero__avatar-col {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  flex: none;
}
.pf-hero__avatar {
  display: inline-flex;
  padding: 0;
  border: 5px solid var(--m-pOn);
  border-radius: 34px;
  overflow: hidden;
  background: var(--m-a);
  transform: rotate(-5deg);
  :deep(.app-avatar) {
    border-radius: 28px;
  }
  :deep(img) {
    border-radius: 0;
  }
}
.pf-hero__cam {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: var(--m-pOn);
  color: var(--m-p);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  transition: transform 0.15s;
}
.pf-hero__cam:active {
  transform: scale(0.96);
}
.pf-hero__cam:disabled {
  opacity: 0.75;
  cursor: progress;
}
.pf-hero__cam.is-round {
  position: absolute;
  right: -10px;
  bottom: -10px;
  width: 44px;
  height: 44px;
  padding: 0;
  border: 3px solid var(--m-p);
}
.is-spin {
  animation: pf-spin 0.9s linear infinite;
}
@keyframes pf-spin {
  to {
    transform: rotate(360deg);
  }
}
.pf-hero__text {
  flex: 1 1 280px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.pf-hero__name {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.875rem;
  font-weight: 800;
  line-height: 1.15;
  overflow-wrap: anywhere;
}
.pf-hero__mail {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  font-size: 0.9375rem;
  opacity: 0.95;
  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
.pf-hero__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 4px;
}
.pf-hero__role,
.pf-hero__chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 30px;
  padding: 0 12px;
  border-radius: 999px;
  font-size: 0.8125rem;
}
.pf-hero__role {
  background: var(--m-pOn);
  color: var(--m-p);
  font-weight: 800;
}
.pf-hero__chip {
  background: color-mix(in srgb, var(--m-pOn) 18%, transparent);
  color: var(--m-pOn);
  font-weight: 700;
}
.pf-hero__bio {
  margin: 2px 0 0;
  max-width: 560px;
  font-size: 0.875rem;
  line-height: 1.5;
  opacity: 0.92;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.pf-hero__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 0;
  margin-top: 2px;
  font-size: 0.8125rem;
  opacity: 0.92;
  span + span::before {
    content: '·';
    margin: 0 8px;
  }
}

@media (width < 640px) {
  .pf-hero {
    padding: 22px 20px 20px;
    border-radius: 26px;
  }
  .pf-hero__blob {
    right: -62px;
    top: -72px;
    width: 128px;
    height: 128px;
  }
  .pf-hero__ring {
    right: -14px;
    bottom: -26px;
    width: 56px;
    height: 56px;
    border-width: 10px;
  }
  .pf-hero__body {
    flex-wrap: nowrap;
    align-items: flex-start;
    gap: 18px;
  }
  .pf-hero__avatar {
    border-width: 4px;
    border-radius: 28px;
  }
  .pf-hero__text {
    flex-basis: auto;
  }
  .pf-hero__name {
    font-size: 1.5rem;
  }
  .pf-hero__mail {
    font-size: 0.8125rem;
  }
  .pf-hero__meta {
    font-size: 0.75rem;
  }
}
</style>
