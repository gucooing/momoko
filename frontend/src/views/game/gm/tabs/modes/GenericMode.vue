<!-- 没有专门界面的玩法：读同名报表，用通用表格 / 键值显示；列表报表可「加载更多」。 -->
<template>
  <div class="pv">
    <div v-if="error" class="pv__error">{{ error }}</div>
    <div v-if="loading && data === undefined" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <GenericData v-else-if="data !== undefined" :data="data" />
    <div v-if="nextCursor" class="pv__foot">
      <span>{{ t('gmData.common.loadedN', { n: loadedCount }) }}</span>
      <button type="button" class="pv__more" :disabled="loading" @click="load(true)">
        {{ loading ? t('gmData.common.loading') : t('gmData.common.loadMore') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmReport, mgpQuery } from '@/api/gameGm'
import { getRequestErrorMessage } from '@/utils/request'
import type { ManifestEntry } from '@/games/mgp'
import { useGmContext } from '../../context'
import GenericData from '../../player/shared/GenericData.vue'

const props = defineProps<{ mode: ManifestEntry }>()

const { t } = useI18n()
const ctx = useGmContext()

const data = shallowRef<unknown>()
const loading = ref(false)
const error = ref('')
const nextCursor = ref('')

type Paged = { items: unknown[]; nextCursor?: string }
const isPaged = (v: unknown): v is Paged => !!v && typeof v === 'object' && Array.isArray((v as { items?: unknown }).items)
const loadedCount = computed(() => (isPaged(data.value) ? data.value.items.length : 0))

const load = async (more = false) => {
  loading.value = true
  error.value = ''
  try {
    const query = more && nextCursor.value ? mgpQuery({ cursor: nextCursor.value }) : ''
    const res = await getGmReport<unknown>(ctx.serverId.value, props.mode.id, query)
    if (more && isPaged(res) && isPaged(data.value)) data.value = { ...res, items: [...data.value.items, ...res.items] }
    else data.value = res
    nextCursor.value = isPaged(res) ? res.nextCursor || '' : ''
  } catch (e) {
    error.value = getRequestErrorMessage(e)
  } finally {
    loading.value = false
  }
}
void load()
</script>

<style scoped lang="scss">
@use '../../player/shared/views';
</style>
