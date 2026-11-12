<!-- 消息：全部 / 未读，分页加载，全部已读。 -->
<template>
  <div class="page">
    <div class="page__head">
      <h1 class="pp-title page__title">{{ t('nav.messages') }}</h1>
      <PButton v-if="unread > 0" size="sm" variant="soft" :loading="reading" @click="readAll">{{ t('home.readAll') }}</PButton>
    </div>
    <div class="pp-scroll-x" role="tablist" :aria-label="t('nav.messages')">
      <button type="button" class="pp-chip" :class="{ 'is-on': !unreadOnly }" role="tab" :aria-selected="!unreadOnly" @click="setFilter(false)">
        {{ t('common.all') }} <span class="pp-chip__n">{{ allTotal }}</span>
      </button>
      <button type="button" class="pp-chip" :class="{ 'is-on': unreadOnly }" role="tab" :aria-selected="unreadOnly" @click="setFilter(true)">
        {{ t('home.unread') }} <span class="pp-chip__n">{{ unread }}</span>
      </button>
    </div>
    <section class="pp-card list">
      <p v-if="error" class="pp-error">{{ error }}</p>
      <PEmpty v-else-if="loaded && !items.length" :icon="EnvelopeIcon" :title="unreadOnly ? t('home.noUnread') : t('home.noMessages')" />
      <div class="pp-rows">
        <MessageRow v-for="m in items" :key="m.id" :message="m" />
      </div>
      <PButton v-if="items.length < total" block variant="ghost" :loading="loading" @click="load(false)">{{ t('common.loadMore') }}</PButton>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { EnvelopeIcon } from '@heroicons/vue/24/outline'
import MessageRow from '@passport/components/MessageRow.vue'
import PButton from '@passport/components/PButton.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import * as api from '@passport/api/passport'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { errorText } from '@passport/utils/format'
import type { PassportMessage } from '@/types/v1/passport'

const { t } = useI18n()
const session = useSessionStore()
const ui = useUiStore()

const PAGE = 20
const items = ref<PassportMessage[]>([])
const total = ref(0)
const allTotal = ref(0)
const unread = ref(0)
const page = ref(0)
const unreadOnly = ref(false)
const loading = ref(false)
const loaded = ref(false)
const reading = ref(false)
const error = ref('')

const load = async (reset: boolean) => {
  loading.value = true
  error.value = ''
  try {
    const next = reset ? 1 : page.value + 1
    const res = await api.listMessages(next, PAGE, unreadOnly.value)
    items.value = reset ? res.items || [] : [...items.value, ...(res.items || [])]
    total.value = res.total
    if (!unreadOnly.value) allTotal.value = res.total
    unread.value = res.unread
    session.setUnread(res.unread)
    page.value = next
  } catch (e) {
    error.value = errorText(e)
  } finally {
    loading.value = false
    loaded.value = true
  }
}

const setFilter = (value: boolean) => {
  if (unreadOnly.value === value) return
  unreadOnly.value = value
  void load(true)
}

const readAll = async () => {
  reading.value = true
  try {
    const res = await api.readMessages([], true)
    unread.value = res.unread
    session.setUnread(res.unread)
    await load(true)
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    reading.value = false
  }
}

onMounted(() => void load(true))
</script>

<style scoped>
.page {
  width: 100%;
  max-width: 820px;
  margin: 0 auto;
  padding: 28px 32px 56px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.page__head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.page__title {
  flex: 1;
  font-size: 28px;
}
.list {
  padding: 6px 20px 12px;
}
@media (max-width: 768px) {
  .page {
    padding: 16px 16px 24px;
  }
  .page__title {
    font-size: 22px;
  }
  .list {
    padding: 4px 16px 10px;
  }
}
</style>
