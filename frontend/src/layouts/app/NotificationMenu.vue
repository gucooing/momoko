<!-- 通知（复用 userProfileStore 逻辑，重写视觉）。 -->
<template>
  <AppDropdown align="end" :width="336">
    <template #trigger>
      <AppIconButton
        icon="HOutline:BellIcon"
        label="通知"
        :badge="unreadCount > 0 ? (unreadCount > 99 ? '99+' : unreadCount) : undefined"
      />
    </template>
    <template #default="{ close }">
      <div class="notif">
        <header class="notif__head">
          <span class="notif__title">{{ t('layout.notificationTitle') }}</span>
          <button
            v-if="unreadCount > 0"
            type="button"
            class="notif__action"
            @click="userProfileStore.markAllAsRead()"
          >
            {{ t('layout.markAllRead') }}
          </button>
        </header>

        <div class="notif__list">
          <EmptyState
            v-if="unreadMessageList.length === 0"
            icon="HOutline:BellSlashIcon"
            :title="t('layout.noMessages')"
          />
          <button
            v-for="message in unreadMessageList"
            :key="message.id"
            type="button"
            class="notif__item"
            @click="userProfileStore.markAsRead(message.id)"
          >
            <AppAvatar :src="message.avatar" :size="34" :name="message.title" />
            <div class="notif__content">
              <div class="notif__item-title">{{ message.title }}</div>
              <div class="notif__item-text">{{ message.content }}</div>
              <div class="notif__item-time">{{ message.time }}</div>
            </div>
          </button>
        </div>

        <footer class="notif__foot">
          <button type="button" class="notif__viewall" @click="goToProfile(close)">
            {{ t('layout.viewAllMessages') }}
          </button>
        </footer>
      </div>
    </template>
  </AppDropdown>
</template>

<script setup lang="ts">
import { useUserProfileStore } from '@/stores/user/profile'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const userProfileStore = useUserProfileStore()
const { unreadCount, userMessages } = storeToRefs(userProfileStore)
const { t } = useI18n()

const unreadMessageList = computed(() => userMessages.value.filter((msg) => !msg.read))

const goToProfile = (close: () => void) => {
  userProfileStore.currentTab = 'messages'
  router.push('/profile')
  close()
}
</script>

<style scoped lang="scss">
.notif__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 13px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.notif__title {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.notif__action {
  border: none;
  background: transparent;
  color: var(--el-color-primary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
}
.notif__list {
  max-height: 22rem;
  overflow-y: auto;
}
.notif__item {
  display: flex;
  gap: 12px;
  width: 100%;
  padding: 12px 16px;
  border: none;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s;
}
.notif__item:hover {
  background: var(--el-fill-color-light);
}
.notif__content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.notif__item-title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.notif__item-text {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.notif__item-time {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
}
.notif__foot {
  padding: 10px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
  text-align: center;
}
.notif__viewall {
  border: none;
  background: transparent;
  color: var(--el-color-primary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
}
</style>
