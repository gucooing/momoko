<!-- 研修：各研修组的试题数与已通关数（MGP seminar 视图），组名来自 MGP 研修目录；
     有 seminar.clear / seminar.reset 指令时可以「全部通关」「重置进度」（走通用指令对话框）。 -->
<template>
  <div class="pv">
    <div class="pv__line">
      <span v-if="list.items.value.length">
        {{ t('gmData.seminar.summary', { cleared: cleared, total: totalExams }) }}
      </span>
      <span class="pv__spacer" />
      <UButton v-for="cmd in commands" :key="cmd.id" color="neutral" variant="outline" size="sm" @click="page.run(cmd.id)">
        {{ cmd.title }}
      </UButton>
    </div>

    <div v-if="list.error.value" class="pv__error">{{ list.error.value }}</div>
    <div v-if="list.loading.value && !list.items.value.length" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState v-else-if="!list.items.value.length" icon="HOutline:BuildingLibraryIcon" :title="t('gmData.seminar.empty')" />
    <div v-else class="sem">
      <article v-for="(g, i) in list.items.value" :key="g.GroupID" class="sem__group">
        <span class="sem__order m-num">{{ i + 1 }}</span>
        <span class="sem__text">
          <span class="pv__strong pv__ellipsis">{{ lookup.name('seminar', g.GroupID) }}</span>
        </span>
        <span class="sem__bar" role="progressbar" :aria-valuenow="pct(g.Cleared, g.Total)" aria-valuemin="0" aria-valuemax="100">
          <span :style="{ width: `${pct(g.Cleared, g.Total)}%` }" />
        </span>
        <span class="sem__count m-num">{{ g.Cleared }} / {{ g.Total }}</span>
        <StatusPill
          :variant="g.Total > 0 && g.Cleared >= g.Total ? 'success' : g.Cleared ? 'primary' : 'neutral'"
          :label="g.Total > 0 && g.Cleared >= g.Total ? t('gmData.seminar.full') : g.Cleared ? t('gmData.seminar.partial') : t('gmData.seminar.none')"
        />
      </article>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmPlayerView, mgpQuery } from '@/api/gameGm'
import type { ListPage, ManifestCommand, ManifestView } from '@/games/mgp'
import type { SeminarProgress } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import { useGmFormat } from '../shared/format'
import { useCatalogLookup } from '../shared/lookup'
import { usePagedList } from '../shared/paging'

const props = defineProps<{ view?: ManifestView }>()

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const lookup = useCatalogLookup()
const { pct } = useGmFormat()

// 研修组只有几组：一次读完（视图上限 500）
const list = usePagedList<SeminarProgress>(
  (q) => getGmPlayerView<ListPage<SeminarProgress>>(ctx.serverId.value, page.pid.value, props.view?.id || 'seminar', mgpQuery({ ...q })),
  { pageSize: 200, append: true },
)
watch(page.version, () => void list.reload(), { immediate: true })

const totalExams = computed(() => list.items.value.reduce((s, g) => s + g.Total, 0))
const cleared = computed(() => list.items.value.reduce((s, g) => s + g.Cleared, 0))

const commands = computed(() => ['seminar.clear', 'seminar.reset'].map((id) => ctx.command(id)).filter((c): c is ManifestCommand => !!c))
</script>

<style scoped lang="scss">
@use '../shared/views';
.sem {
  display: flex;
  flex-direction: column;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  overflow: hidden;
}
.sem__group {
  display: grid;
  grid-template-columns: 38px minmax(0, 1.2fr) minmax(0, 1fr) 70px 110px;
  gap: 12px;
  align-items: center;
  min-height: 64px;
  padding: 8px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.sem__group:first-child {
  border-top: 0;
}
.sem__order {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 12px;
  background: var(--m-gSoft);
  color: var(--m-gInk);
  font-weight: 800;
}
.sem__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.sem__bar {
  height: 8px;
  border-radius: 999px;
  background: var(--m-s2);
  overflow: hidden;
}
.sem__bar span {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--m-g);
}
.sem__count {
  font-weight: 700;
  text-align: right;
}
@media (max-width: 760px) {
  .sem__group {
    grid-template-columns: 38px minmax(0, 1fr) auto;
    grid-template-areas:
      'order text pill'
      'order bar count';
    row-gap: 6px;
    padding: 10px 14px;
  }
  .sem__order {
    grid-area: order;
  }
  .sem__text {
    grid-area: text;
  }
  .sem__bar {
    grid-area: bar;
  }
  .sem__count {
    grid-area: count;
  }
  .sem__group > :last-child {
    grid-area: pill;
  }
}
</style>
