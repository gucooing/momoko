<!-- 我的消息：发布 + 列表（store 本地消息流）；确认用 AdaptiveConfirm / FormDialog -->
<template>
  <div class="msg">
    <AppPanel>
      <div class="msg-compose">
        <AppAvatar :src="userStore.resolvedUserAvatar" :name="displayName" :size="32" />
        <span class="msg-compose__hi">{{ t('user.greetingMessage', { name: displayName }) }}</span>
      </div>
      <textarea
        v-model="messageDraft"
        class="app-textarea msg-compose__input"
        rows="3"
        :placeholder="t('user.messagePlaceholder')"
      />
      <div class="msg-compose__foot">
        <span class="msg-compose__hint">{{ t('user.messagePushHint') }}</span>
        <UButton color="primary" size="sm" :disabled="!messageDraft.trim()" @click="sendMessage">
          {{ t('user.publishMessage') }}
        </UButton>
      </div>
    </AppPanel>

    <AppPanel>
      <template #header>
        <div class="msg-head">
          <div class="msg-seg" role="tablist">
            <button
              v-for="tab in messageTabs"
              :key="tab.key"
              type="button"
              role="tab"
              class="msg-seg__btn"
              :class="{ 'is-active': activeMessageTab === tab.key }"
              @click="activeMessageTab = tab.key as 'all' | 'unread'"
            >
              {{ t(tab.labelKey) }}
              <span v-if="tab.badge" class="msg-seg__badge">{{ tab.badge }}</span>
            </button>
          </div>
          <div class="msg-head__ops">
            <UButton
              color="neutral"
              variant="ghost"
              size="sm"
              :disabled="!unreadCount"
              @click="userProfileStore.markAllAsRead()"
            >
              {{ t('user.markAllRead') }}
            </UButton>
            <AdaptiveConfirm
              :title="t('user.clearMessagesContent')"
              :disabled="!userMessages.length"
              @confirm="doClearAll"
            >
              <template #reference>
                <UButton color="error" variant="ghost" size="sm" :disabled="!userMessages.length">
                  {{ t('user.clearAll') }}
                </UButton>
              </template>
            </AdaptiveConfirm>
          </div>
        </div>
      </template>

      <EmptyState
        v-if="!messageList.length"
        icon="HOutline:BellAlertIcon"
        :title="activeMessageTab === 'unread' ? t('user.noUnreadMessages') : t('layout.noMessages')"
      />
      <div v-else class="msg-list">
        <article v-for="message in messageList" :key="message.id" class="msg-item">
          <div class="msg-item__avatar">
            <AppAvatar :src="message.avatar" :name="resolveMessageTitle(message)" :size="40" />
            <span v-if="!message.read" class="msg-item__dot" />
          </div>
          <div class="msg-item__body">
            <div class="msg-item__top">
              <span class="msg-item__title">{{ resolveMessageTitle(message) }}</span>
              <div class="msg-item__ops">
                <button
                  v-if="!message.read"
                  type="button"
                  class="msg-item__op"
                  :title="t('user.setRead')"
                  @click="userProfileStore.markAsRead(message.id)"
                >
                  <component :is="menuStore.iconComponents['HOutline:CheckIcon']" />
                </button>
                <AdaptiveConfirm
                  :title="t('user.deleteMessageConfirm')"
                  @confirm="doDeleteMessage(message.id)"
                >
                  <template #reference>
                    <button type="button" class="msg-item__op msg-item__op--danger" :title="t('common.delete')">
                      <component :is="menuStore.iconComponents['HOutline:TrashIcon']" />
                    </button>
                  </template>
                </AdaptiveConfirm>
              </div>
            </div>
            <p class="msg-item__content">{{ resolveMessageContent(message) }}</p>
            <div class="msg-item__time">{{ message.time }}</div>
          </div>
        </article>
      </div>
    </AppPanel>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useUserProfileStore } from '@/stores/user/profile'
import { useFeedback } from '@/utils/feedback'
import type { UserMessageItem } from '@/stores/user/types'
import { useI18n } from 'vue-i18n'

const userStore = useUserStore()
const userProfileStore = useUserProfileStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const fb = useFeedback()

const { messageDraft, activeMessageTab, messageTabs, messageList, unreadCount, userMessages } =
  storeToRefs(userProfileStore)

const displayName = computed(
  () => userStore.userInfo?.name || userStore.userInfo?.username || t('user.messages.unknownUser'),
)

const resolveMessageTitle = (message: UserMessageItem) =>
  message.titleKey ? t(message.titleKey) : message.title || ''
const resolveMessageContent = (message: UserMessageItem) =>
  message.contentKey ? t(message.contentKey) : message.content || ''

const sendMessage = () => {
  const senderName =
    userStore.userInfo?.name || userStore.userInfo?.username || t('user.messages.unknownUser')
  const success = userProfileStore.sendMessage(senderName, userStore.resolvedUserAvatar)
  if (!success) return
  fb.success(t('user.messageSendSuccess'))
}

const doClearAll = () => {
  userProfileStore.deleteAllMessages()
  fb.success(t('user.clearMessagesDone'))
}

const doDeleteMessage = (id: string) => {
  userProfileStore.deleteMessage(id)
  fb.success(t('user.messageDeleted'))
}
</script>

<style scoped lang="scss">
.msg {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.msg-compose {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  margin-bottom: 0.65rem;
}
.msg-compose__hi {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.msg-compose__input {
  width: 100%;
  resize: vertical;
  min-height: 72px;
}
.msg-compose__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 0.65rem;
}
.msg-compose__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.msg-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  width: 100%;
  flex-wrap: wrap;
}
.msg-head__ops {
  display: flex;
  align-items: center;
  gap: 0.2rem;
}
.msg-seg {
  display: inline-flex;
  gap: 2px;
  padding: 2px;
  background: var(--el-fill-color-light);
  border-radius: calc(var(--app-radius) - 2px);
}
.msg-seg__btn {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  height: 28px;
  padding: 0 10px;
  border: 0;
  border-radius: calc(var(--app-radius) - 4px);
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
}
.msg-seg__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font-weight: 600;
  box-shadow: var(--app-shadow-sm, 0 1px 2px rgb(0 0 0 / 0.04));
}
.msg-seg__badge {
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--el-color-primary) 14%, transparent);
  color: var(--el-color-primary);
  font-size: 0.6875rem;
  font-weight: 700;
  line-height: 16px;
  text-align: center;
  font-variant-numeric: tabular-nums;
}
.msg-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.msg-item {
  display: flex;
  gap: 0.75rem;
  padding: 0.75rem;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
}
.msg-item:hover {
  border-color: var(--el-border-color);
}
.msg-item__avatar {
  position: relative;
  flex-shrink: 0;
}
.msg-item__dot {
  position: absolute;
  right: 0;
  bottom: 2px;
  width: 9px;
  height: 9px;
  border-radius: 999px;
  background: var(--el-color-danger);
  border: 2px solid var(--el-bg-color);
}
.msg-item__body {
  flex: 1;
  min-width: 0;
}
.msg-item__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}
.msg-item__title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.msg-item__ops {
  display: flex;
  align-items: center;
  gap: 0.1rem;
  opacity: 0.65;
}
.msg-item:hover .msg-item__ops {
  opacity: 1;
}
.msg-item__op {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  :deep(svg) {
    width: 15px;
    height: 15px;
  }
}
.msg-item__op:hover {
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
}
.msg-item__op--danger:hover {
  color: var(--el-color-danger);
  background: color-mix(in srgb, var(--el-color-danger) 10%, transparent);
}
.msg-item__content {
  margin: 0.35rem 0 0;
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--el-text-color-regular);
  word-break: break-word;
}
.msg-item__time {
  margin-top: 0.35rem;
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
}
@media (width < 640px) {
  .msg-item__ops {
    opacity: 1;
  }
}
</style>
