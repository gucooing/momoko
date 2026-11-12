<!-- 没有专门界面的玩家视图：读 MGP 视图并用通用表格 / 键值显示；列表视图可「加载更多」。 -->
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
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmPlayerView, mgpQuery } from '@/api/gameGm'
import { getRequestErrorMessage } from '@/utils/request'
import type { ManifestView } from '@/games/mgp'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import GenericData from '../shared/GenericData.vue'

const props = defineProps<{ view?: ManifestView }>()

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()

const data = shallowRef<unknown>()
const loading = ref(false)
const error = ref('')
const nextCursor = ref('')

type Paged = { items: unknown[]; nextCursor?: string }
const isPaged = (v: unknown): v is Paged => !!v && typeof v === 'object' && Array.isArray((v as { items?: unknown }).items)
const loadedCount = computed(() => (isPaged(data.value) ? data.value.items.length : 0))

const load = async (more = false) => {
  if (!props.view) return
  loading.value = true
  error.value = ''
  try {
    const query = more && nextCursor.value ? mgpQuery({ cursor: nextCursor.value }) : ''
    const res = await getGmPlayerView<unknown>(ctx.serverId.value, page.pid.value, props.view.id, query)
    if (more && isPaged(res) && isPaged(data.value)) {
      data.value = { ...res, items: [...data.value.items, ...res.items] }
    } else {
      data.value = res
    }
    nextCursor.value = isPaged(res) ? res.nextCursor || '' : ''
  } catch (e) {
    error.value = getRequestErrorMessage(e)
  } finally {
    loading.value = false
  }
}
watch(page.version, () => void load(), { immediate: true })
</script>

<style scoped lang="scss">
@use '../shared/views';
</style>
