<!-- 自助操作里挑一个游戏目录条目（奖励、道具）：按名称搜索（服务端筛选），页码分页；不能作为礼物发放的条目不列出。 -->
<template>
  <div class="scs">
    <input v-model="keyword" type="search" class="scs__q" :placeholder="t('game.self.search')" @input="onInput" />
    <div v-if="shown.length" class="scs__grid">
      <button v-for="e in shown" :key="e.id" type="button" class="scs__item" @click="emit('pick', e)">
        <span class="scs__img"><GameImage :src="gameCatalogImage(ctx.game.value, kind, e)" /></span>
        <span class="scs__name">{{ e.name || t('game.unnamed') }}</span>
      </button>
    </div>
    <p v-else-if="!list.loading && list.loaded" class="pp-hint scs__empty">{{ t('game.self.noMatch') }}</p>
    <p v-if="list.error" class="pp-hint scs__err">{{ list.error }}</p>
    <PPager v-if="list.loaded" :page="list.page" :total="list.total" :page-size="PAGE" :disabled="list.loading" @update:page="(p) => void paged.load(p)" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import GameImage from '@/components/game/GameImage.vue'
import { gameCatalogImage } from '@/games'
import type { CatalogEntry } from '@/games/mgp'
import PPager from '@passport/components/PPager.vue'
import { offsetCursor, usePagedList } from '@passport/utils/paged'
import { useGameContext } from './context'

const props = withDefaults(defineProps<{ kind: string; excludeIds?: string[] }>(), { excludeIds: () => [] })
const emit = defineEmits<{ pick: [entry: CatalogEntry] }>()

const PAGE = 20

const { t } = useI18n()
const ctx = useGameContext()

const keyword = ref('')
const paged = usePagedList<CatalogEntry>(
  (page, size) => ctx.catalog(props.kind, { q: keyword.value.trim(), cursor: offsetCursor(page, size), limit: size }),
  PAGE,
)
const list = paged.list

const shown = computed(() => list.items.filter((e) => e.resourceType > 0 && !props.excludeIds.includes(e.id)))

const onInput = useDebounceFn(() => void paged.reload(), 300)
watch(
  () => props.kind,
  () => {
    keyword.value = ''
    void paged.reload()
  },
  { immediate: true },
)
</script>

<style scoped>
.scs {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.scs__q {
  width: 100%;
  min-height: 40px;
  padding: 0 12px;
}
.scs__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(96px, 1fr));
  gap: 8px;
}
.scs__item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  min-width: 0;
  padding: 8px 6px;
  border: 1px solid var(--m-line);
  border-radius: 12px;
  background: var(--m-s);
  color: inherit;
  font: inherit;
  cursor: pointer;
}
.scs__item:hover {
  border-color: var(--m-p);
}
.scs__img {
  width: 48px;
  height: 48px;
}
.scs__img :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.scs__name {
  width: 100%;
  font-size: 12px;
  font-weight: 700;
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.scs__empty,
.scs__err {
  margin: 0;
}
.scs__err {
  color: var(--m-bad);
}
</style>
