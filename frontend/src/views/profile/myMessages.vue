<!-- 我的消息 -->
<template>
  <div>
    <BaseCard>
      <div class="flex items-center gap-4">
        <el-avatar :size="32" :src="userStore.resolvedUserAvatar" />
        <span class="text-sm font-medium text-(--el-text-color-secondary)">
          {{ t('user.greetingMessage', { name: displayName }) }}
        </span>
      </div>
      <el-input
        v-model="messageDraft"
        :rows="3"
        type="textarea"
        :placeholder="t('user.messagePlaceholder')"
        class="mt-4"
      />
      <div class="flex items-center justify-between mt-4">
        <div class="text-xs text-(--el-text-color-secondary)">{{ t('user.messagePushHint') }}</div>
        <el-button type="primary" :disabled="!messageDraft.trim()" @click="sendMessage"
          >{{ t('user.publishMessage') }}</el-button
        >
      </div>
    </BaseCard>

    <BaseCard class="mt-4">
      <template #header>
        <div class="flex items-center justify-between">
          <div class="flex-1">
            <BadgeTabsMenu
              v-model="activeMessageTab"
              :tabs-menu-data="messageTabs"
              :tabs-item-height="30"
            />
          </div>
          <div class="flex items-center">
            <IconButton
              icon="Element:Check"
              type="primary"
              :tooltip="t('user.markAllRead')"
              size="1.5rem"
              iconSize="1rem"
              v-if="menuStore.isMobile"
              :disabled="!unreadCount"
              @click="userProfileStore.markAllAsRead()"
            />
            <el-button
              type="primary"
              link
              :disabled="!unreadCount"
              @click="userProfileStore.markAllAsRead()"
              v-else
            >
              {{ t('user.markAllRead') }}
            </el-button>
            <el-divider direction="vertical" />
            <IconButton
              icon="Element:Delete"
              type="danger"
              :tooltip="t('user.clearAll')"
              size="1.5rem"
              iconSize="1rem"
              :disabled="!userMessages.length"
              v-if="menuStore.isMobile"
              @click="clearAllMessages"
            />
            <el-button
              type="danger"
              link
              :disabled="!userMessages.length"
              @click="clearAllMessages"
              v-else
            >
              {{ t('user.clearAll') }}
            </el-button>
          </div>
        </div>
      </template>
      <div>
        <Transition name="zoom" mode="out-in">
          <el-empty
            v-if="messageList.length === 0"
            :description="activeMessageTab === 'unread' ? t('user.noUnreadMessages') : t('layout.noMessages')"
          />
          <TransitionGroup name="group-slide-right" tag="div" v-else>
            <div v-for="message in messageList" :key="message.id">
              <HoverAnimateWrapper name="lift" intensity="light" class="w-full">
                <div
                  class="group p-4 mb-3 flex items-center gap-4 border border-(--el-border-color-light) rounded-xl cursor-pointer hover:border-(--el-border-color) hover:bg-(--el-bg-color-page)"
                >
                  <div class="relative">
                    <el-avatar :size="48" :src="message.avatar" />
                    <span
                      class="absolute h-3 w-3 bottom-1.5 right-0.5 rounded-full border-3 border-(--el-bg-color) bg-(--el-color-danger)"
                      v-if="!message.read"
                    ></span>
                  </div>

                  <div class="flex-1">
                    <div class="flex justify-between">
                      <TextEllipsis :text="resolveMessageTitle(message)" :clickable="false" tooltipType="none" />
                      <div
                        class="flex items-center opacity-100 lg:opacity-0 group-hover:opacity-100"
                      >
                        <IconButton
                          icon="Element:Check"
                          type="primary"
                          :tooltip="t('user.setRead')"
                          size="1.5rem"
                          iconSize="1rem"
                          @click="userProfileStore.markAsRead(message.id)"
                          v-if="!message.read"
                        />
                        <el-divider direction="vertical" v-if="!message.read" />
                        <AdaptiveConfirm
                          :title="t('user.deleteMessageConfirm')"
                          @confirm="(userProfileStore.deleteMessage(message.id), ElMessage.success(t('user.messageDeleted')))"
                        >
                          <template #reference>
                            <div>
                              <IconButton
                                icon="Element:Delete"
                                type="danger"
                                size="1.5rem"
                                iconSize="1rem"
                                :tooltip="t('common.delete')"
                              />
                            </div>
                          </template>
                        </AdaptiveConfirm>
                      </div>
                    </div>
                    <div
                      class="mt-2 text-sm text-(--el-text-color-regular) leading-relaxed wrap-break-word"
                    >
                      {{ resolveMessageContent(message) }}
                    </div>
                    <div class="text-xs text-(--el-text-color-secondary) mt-2">
                      {{ message.time }}
                    </div>
                  </div>
                </div>
              </HoverAnimateWrapper>
            </div>
          </TransitionGroup>
        </Transition>
      </div>
    </BaseCard>
  </div>
</template>

<script setup lang="ts">
import { useUserProfileStore } from '@/stores/user/profile'
import { Dialog } from '@/utils/dialog'
import { delay } from '@/utils/utils'
import BadgeTabsMenu from '@/components/tabs/BadgeTabsMenu.vue'
import { ElMessage } from 'element-plus'
import type { UserMessageItem } from '@/stores/user/types'
import { useI18n } from 'vue-i18n'

const userStore = useUserStore()
const userProfileStore = useUserProfileStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const { messageDraft, activeMessageTab, messageTabs, messageList, unreadCount, userMessages } =
  storeToRefs(userProfileStore)

const displayName = computed(() => userStore.userInfo?.name || userStore.userInfo?.username || t('user.messages.unknownUser'))

const resolveMessageTitle = (message: UserMessageItem) => {
  return message.titleKey ? t(message.titleKey) : message.title || ''
}

const resolveMessageContent = (message: UserMessageItem) => {
  return message.contentKey ? t(message.contentKey) : message.content || ''
}

// 发送消息
const sendMessage = () => {
  const senderName = userStore.userInfo?.name || userStore.userInfo?.username || t('user.messages.unknownUser')
  const success = userProfileStore.sendMessage(senderName, userStore.resolvedUserAvatar)
  if (!success) return
  ElMessage.success(t('user.messageSendSuccess'))
}

// 清空全部消息
const clearAllMessages = () => {
  Dialog.confirm({
    title: t('user.clearMessagesTitle'),
    content: t('user.clearMessagesContent'),
    onConfirm: async () => {
      await delay(1000)
      userProfileStore.deleteAllMessages()
      ElMessage.success(t('user.clearMessagesDone'))
    },
  })
}
</script>

<style scoped lang="scss"></style>
