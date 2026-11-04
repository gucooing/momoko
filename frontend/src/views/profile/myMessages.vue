<!-- 我的消息：卡片（最近 3 条 + 未读数）或设置行（手机）；「消息中心」弹窗：发布通知 / 全部·未读 / 全部已读 / 清空 / 单条已读·删除。 -->
<template>
  <ProfileRow
    v-if="row"
    icon="i-lucide-bell"
    tone="solid-p"
    :tilt="-2"
    :title="t('user.tabs.messages')"
    :sub="latest ? `${latest.title} · ${relTime(latest.createTime)}` : t('layout.noMessages')"
    clickable
    chevron
    @click="openCenter"
  >
    <span v-if="unreadCount" class="msg-badge m-num">{{ badgeText }}</span>
  </ProfileRow>

  <AppPanel v-else class="msg-card">
    <template #header>
      <div class="msg-card__head">
        <h2 class="pf-side-title">{{ t('user.tabs.messages') }}</h2>
        <template v-if="unreadCount">
          <span class="msg-badge m-num">{{ badgeText }}</span>
          <span class="msg-card__unread">{{ t('profile.msg.unreadSuffix') }}</span>
        </template>
      </div>
    </template>
    <template #actions>
      <button type="button" class="pf-link" @click="openCenter">{{ t('profile.msg.center') }} →</button>
    </template>

    <EmptyState v-if="!preview.length" icon="HOutline:BellAlertIcon" :title="t('layout.noMessages')" />
    <div v-else class="msg-preview">
      <button
        v-for="m in preview"
        :key="m.id"
        type="button"
        class="msg-mini"
        @click="openFromPreview(m.id)"
      >
        <span class="m-sticker msg-mini__sk" :class="`m-sticker--${kindTone(m)}`" aria-hidden="true">
          <UIcon :name="kindIcon(m)" />
        </span>
        <span class="msg-mini__text">
          <span class="msg-mini__title">{{ m.title }}</span>
          <span class="msg-mini__sub">{{ kindLabel(m) }} · {{ relTime(m.createTime) }}</span>
        </span>
        <span v-if="!m.read" class="msg-dot" role="img" :aria-label="t('user.messages.unread')" />
      </button>
    </div>
  </AppPanel>

  <!-- 消息中心 -->
  <FormDialog v-model="centerOpen" :title="t('profile.msg.center')" :width="680" :show-footer="false">
    <div class="msg-center">
      <section v-if="canPublish" class="msg-compose" :aria-label="t('user.publishMessage')">
        <div class="msg-compose__hi">
          <AppAvatar :src="userStore.resolvedUserAvatar" :name="displayName" :size="30" />
          <span>{{ t('user.greetingMessage', { name: displayName }) }}</span>
        </div>
        <input
          v-model="messageDraftTitle"
          class="app-input"
          maxlength="100"
          :placeholder="t('profile.msg.titlePlaceholder')"
        />
        <textarea
          v-model="messageDraft"
          class="app-textarea msg-compose__input"
          rows="3"
          maxlength="2000"
          :placeholder="t('user.messagePlaceholder')"
        />
        <div class="msg-compose__foot">
          <span class="msg-compose__hint">{{ t('profile.msg.publishHint') }}</span>
          <UButton
            color="primary"
            icon="i-lucide-send"
            :loading="publishing"
            :disabled="!messageDraft.trim()"
            @click="sendMessage"
          >
            {{ t('user.publishMessage') }}
          </UButton>
        </div>
      </section>

      <div class="msg-bar">
        <div class="msg-chips" role="tablist">
          <button
            v-for="tab in messageTabs"
            :key="tab.key"
            type="button"
            role="tab"
            class="msg-chip"
            :class="{ 'is-on': activeMessageTab === tab.key }"
            :aria-selected="activeMessageTab === tab.key"
            @click="activeMessageTab = tab.key as 'all' | 'unread'"
          >
            {{ t(tab.labelKey) }}<span class="msg-chip__n m-num">{{ tab.badge }}</span>
          </button>
        </div>
        <div class="msg-bar__ops">
          <UButton
            color="neutral"
            variant="ghost"
            size="sm"
            icon="i-lucide-check-check"
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
              <UButton
                color="error"
                variant="ghost"
                size="sm"
                icon="i-lucide-trash-2"
                :disabled="!userMessages.length"
              >
                {{ t('user.clearAll') }}
              </UButton>
            </template>
          </AdaptiveConfirm>
        </div>
      </div>

      <EmptyState
        v-if="!messageList.length"
        icon="HOutline:BellAlertIcon"
        :title="activeMessageTab === 'unread' ? t('user.noUnreadMessages') : t('layout.noMessages')"
      />
      <div v-else class="msg-list">
        <article
          v-for="m in messageList"
          :key="m.id"
          class="msg-item"
          :class="{ 'is-unread': !m.read, 'is-focus': focusId === m.id }"
        >
          <AppAvatar v-if="m.kind === 'notice'" :src="m.avatar" :name="m.senderName || m.title" :size="40" />
          <span v-else class="m-sticker msg-item__sk" :class="`m-sticker--${kindTone(m)}`" aria-hidden="true">
            <UIcon :name="kindIcon(m)" />
          </span>
          <div class="msg-item__body">
            <div class="msg-item__top">
              <span class="msg-item__title">{{ m.title }}</span>
              <span v-if="!m.read" class="msg-dot" role="img" :aria-label="t('user.messages.unread')" />
            </div>
            <p v-if="m.content" class="msg-item__content">{{ m.content }}</p>
            <div class="msg-item__meta m-num">
              {{ kindLabel(m) }}<template v-if="m.senderName"> · {{ m.senderName }}</template> · {{ m.time }}
            </div>
          </div>
          <div class="msg-item__ops">
            <AppIconButton
              v-if="!m.read"
              icon="HOutline:CheckIcon"
              :label="t('user.setRead')"
              :size="17"
              :box="36"
              @click="userProfileStore.markAsRead(m.id)"
            />
            <AdaptiveConfirm :title="t('user.deleteMessageConfirm')" @confirm="doDeleteMessage(m.id)">
              <template #reference>
                <AppIconButton icon="HOutline:TrashIcon" :label="t('common.delete')" :size="17" :box="36" class="msg-del" />
              </template>
            </AdaptiveConfirm>
          </div>
        </article>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useUserProfileStore } from '@/stores/user/profile'
import { useFeedback } from '@/utils/feedback'
import { getRequestErrorMessage } from '@/utils/request'
import type { UserMessageItem } from '@/stores/user/types'
import { useI18n } from 'vue-i18n'
import ProfileRow from './components/ProfileRow.vue'
import { useProfileFormat } from './components/useProfileFormat'

withDefaults(defineProps<{ /** 手机端：渲染为设置行 */ row?: boolean }>(), { row: false })

const userStore = useUserStore()
const userProfileStore = useUserProfileStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const fb = useFeedback()
const { relTime } = useProfileFormat()

const {
  messageDraft,
  messageDraftTitle,
  publishing,
  activeMessageTab,
  messageTabs,
  messageList,
  unreadCount,
  userMessages,
} = storeToRefs(userProfileStore)

const canPublish = computed(() => menuStore.hasButtonPermission('system_config:edit'))
const displayName = computed(
  () => userStore.userInfo?.name || userStore.userInfo?.username || t('user.messages.unknownUser'),
)
const badgeText = computed(() => (unreadCount.value > 99 ? '99+' : String(unreadCount.value)))
const preview = computed(() => userMessages.value.slice(0, 3))
const latest = computed(() => userMessages.value[0])

const kindIcon = (m: UserMessageItem) => {
  switch (m.kind) {
    case 'login_new_device':
      return 'i-lucide-monitor-smartphone'
    case 'login_failed':
      return 'i-lucide-shield-alert'
    case 'password_changed':
    case 'password_reset':
      return 'i-lucide-key-round'
    case 'notice':
      return 'i-lucide-megaphone'
    default:
      return 'i-lucide-bell'
  }
}
const kindTone = (m: UserMessageItem) => {
  switch (m.kind) {
    case 'login_new_device':
      return 'a2'
    case 'login_failed':
      return 'bad'
    case 'password_changed':
    case 'password_reset':
      return 'ok'
    case 'notice':
      return 'p'
    default:
      return 'warn'
  }
}
const kindLabel = (m: UserMessageItem) => t(`profile.msg.category.${m.type}`)

const centerOpen = ref(false)
const focusId = ref('')
const openCenter = () => {
  focusId.value = ''
  centerOpen.value = true
  void userProfileStore.refreshMessages()
}
const openFromPreview = (id: string) => {
  activeMessageTab.value = 'all'
  focusId.value = id
  centerOpen.value = true
  void userProfileStore.markAsRead(id)
}

const sendMessage = async () => {
  try {
    const n = await userProfileStore.publishMessage()
    fb.success(t('profile.msg.published', { n }))
  } catch (error) {
    fb.error(getRequestErrorMessage(error, t('profile.msg.publishFailed')))
  }
}

const doClearAll = async () => {
  await userProfileStore.deleteAllMessages()
  fb.success(t('user.clearMessagesDone'))
}

const doDeleteMessage = async (id: string) => {
  await userProfileStore.deleteMessage(id)
  fb.success(t('user.messageDeleted'))
}

defineExpose({ openCenter })
</script>

<style scoped lang="scss">
.msg-card :deep(.app-panel__body) {
  padding-top: 8px;
  padding-bottom: 12px;
}
.pf-side-title {
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.msg-card__head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.msg-card__unread {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.msg-badge {
  min-width: 24px;
  height: 24px;
  padding: 0 7px;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-size: 0.75rem;
  font-weight: 800;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.pf-link {
  min-height: 40px;
  padding: 0 2px;
  border: 0;
  background: none;
  color: var(--m-pInk);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.pf-link:hover {
  color: var(--el-text-color-primary);
}
.msg-preview {
  display: flex;
  flex-direction: column;
}
.msg-mini {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  width: calc(100% + 20px);
  margin: 0 -10px;
  padding: 10px;
  border: 0;
  border-radius: 16px;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.msg-mini:hover {
  background: var(--el-fill-color-light);
}
.msg-mini__sk {
  width: 38px;
  height: 38px;
  border-radius: 13px;
  font-size: 18px;
}
.msg-mini__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.msg-mini__title {
  font-size: 0.875rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.msg-mini__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.msg-dot {
  flex: none;
  width: 9px;
  height: 9px;
  margin-top: 6px;
  border-radius: 50%;
  background: var(--m-p);
}

/* 消息中心 */
.msg-center {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.msg-compose {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
}
.msg-compose__hi {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.msg-compose__input {
  width: 100%;
  min-height: 72px;
  resize: vertical;
}
.msg-compose__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}
.msg-compose__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.msg-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}
.msg-bar__ops {
  display: flex;
  align-items: center;
  gap: 4px;
}
.msg-chips {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  scrollbar-width: none;
  &::-webkit-scrollbar {
    display: none;
  }
}
.msg-chip {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
}
.msg-chip.is-on {
  border-color: var(--m-ink);
  background: var(--m-ink);
  color: var(--m-bg);
}
.msg-chip__n {
  font-size: 0.75rem;
  opacity: 0.8;
}
.msg-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.msg-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
}
.msg-item.is-unread {
  background: color-mix(in srgb, var(--m-pSoft) 45%, var(--el-bg-color));
}
.msg-item.is-focus {
  border-color: var(--m-p);
}
.msg-item__sk {
  width: 40px;
  height: 40px;
  border-radius: 13px;
  font-size: 19px;
}
.msg-item__body {
  flex: 1;
  min-width: 0;
}
.msg-item__top {
  display: flex;
  align-items: center;
  gap: 8px;
}
.msg-item__top .msg-dot {
  margin-top: 0;
}
.msg-item__title {
  min-width: 0;
  font-size: 0.9375rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
}
.msg-item__content {
  margin: 4px 0 0;
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--el-text-color-regular);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.msg-item__meta {
  margin-top: 6px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.msg-item__ops {
  flex: none;
  display: flex;
  align-items: center;
  gap: 2px;
}
.msg-del:hover {
  color: var(--m-bad);
}
@media (width < 640px) {
  .pf-link {
    min-height: 44px;
  }
  .msg-item {
    flex-wrap: wrap;
  }
  .msg-item__ops {
    width: 100%;
    justify-content: flex-end;
    margin-top: -4px;
  }
}
</style>
