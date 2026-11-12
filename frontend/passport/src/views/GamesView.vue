<!-- 我的游戏：全部绑定按游戏分组；没有绑定时引导去绑定。 -->
<template>
  <div class="page">
    <div class="page__head">
      <div class="page__titles">
        <h1 class="pp-title page__title">{{ t('nav.games') }}</h1>
        <span v-if="links.length" class="pp-hint">{{ t('home.games.summary', { n: links.length, g: gameCount }) }}</span>
      </div>
      <PButton size="sm" variant="primary" :icon="PlusIcon" to="/bind">{{ t('home.games.bind') }}</PButton>
    </div>
    <p v-if="error" class="pp-error">{{ error }}</p>
    <section v-else-if="loaded && !links.length" class="pp-card">
      <PEmpty :icon="PuzzlePieceIcon" :title="t('home.games.emptyTitle')" :description="t('home.games.emptyDesc')">
        <PButton variant="primary" :icon="PlusIcon" to="/bind">{{ t('home.games.bind') }}</PButton>
      </PEmpty>
    </section>
    <MyGames v-else-if="loaded" :links="links" :compact="ui.isMobile" :show-add="!ui.isMobile" />
    <p v-else class="pp-hint">{{ t('common.loading') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { PlusIcon, PuzzlePieceIcon } from '@heroicons/vue/24/outline'
import MyGames from '@passport/components/MyGames.vue'
import PButton from '@passport/components/PButton.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import * as api from '@passport/api/passport'
import { useUiStore } from '@passport/stores/ui'
import { errorText } from '@passport/utils/format'
import type { PassportLink } from '@/types/v1/passport'

const { t } = useI18n()
const ui = useUiStore()

const links = ref<PassportLink[]>([])
const loaded = ref(false)
const error = ref('')
const gameCount = computed(() => new Set(links.value.map((l) => l.game)).size)

onMounted(async () => {
  try {
    links.value = (await api.listLinks()).items || []
  } catch (e) {
    error.value = errorText(e)
  } finally {
    loaded.value = true
  }
})
</script>

<style scoped>
.page {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
  padding: 28px 32px 56px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.page__head {
  display: flex;
  align-items: flex-end;
  gap: 12px;
}
.page__titles {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.page__title {
  font-size: 28px;
}
@media (max-width: 768px) {
  .page {
    padding: 16px 16px 24px;
  }
  .page__title {
    font-size: 22px;
  }
}
</style>
