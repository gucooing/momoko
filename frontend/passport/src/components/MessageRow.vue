<!-- 一条消息：游戏相关的用游戏标记，通行证自己的用盾牌图标。 -->
<template>
  <div class="mr" :class="{ 'is-unread': !message.read }">
    <GameMark v-if="message.server" :game="message.server.game" :name="message.server.gameName || message.server.name" :size="36" />
    <span v-else class="mr__ico pp-sticker"><ShieldCheckIcon aria-hidden="true" /></span>
    <span class="mr__body">
      <span class="mr__text">{{ messageText(message) }}</span>
      <span v-if="note" class="mr__note">{{ note }}</span>
      <span class="pp-hint">{{ messageSource(message) }} · {{ fmtAgo(message.createTime) }}</span>
    </span>
    <span v-if="!message.read" class="mr__dot" :aria-label="t('home.unread')" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ShieldCheckIcon } from '@heroicons/vue/24/outline'
import GameMark from '@passport/components/GameMark.vue'
import { fmtAgo } from '@passport/utils/format'
import { messageSource, messageText } from '@passport/utils/messages'
import type { PassportMessage } from '@/types/v1/passport'

const props = defineProps<{ message: PassportMessage }>()
const { t } = useI18n()

// 驳回原因等附带的说明（管理员写的原文）。
const note = computed(() => (props.message.kind === 'request_rejected' ? '' : props.message.params?.note || ''))
</script>

<style scoped>
.mr {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 0;
}
.mr__ico {
  width: 36px;
  height: 36px;
  border-radius: 12px;
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.mr__ico svg {
  width: 18px;
  height: 18px;
}
.mr__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.mr__text {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.5;
  overflow-wrap: anywhere;
}
.is-unread .mr__text {
  font-weight: 800;
}
.mr__note {
  font-size: 13px;
  color: var(--m-ink2);
}
.mr__dot {
  width: 9px;
  height: 9px;
  margin-top: 6px;
  border-radius: 50%;
  background: var(--m-p);
  flex: none;
}
</style>
