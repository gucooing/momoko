<!-- 研修：各研修组的试题数与已通关数，组名来自研修目录；
     可以「全部通关」「重置进度」（seminar.clear / seminar.reset，走操作对话框）。 -->
<template>
  <div class="pv">
    <div class="pv__line">
      <span v-if="groups.length">
        {{ t('gmData.seminar.summary', { cleared: cleared, total: totalExams }) }}
      </span>
      <span class="pv__spacer" />
      <UButton v-for="op in operations" :key="op.id" color="neutral" variant="outline" size="sm" @click="page.run(op.id)">
        {{ op.title }}
      </UButton>
    </div>

    <div v-if="error" class="pv__error">{{ error }}</div>
    <div v-if="loading && !groups.length" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState v-else-if="!groups.length" icon="HOutline:BuildingLibraryIcon" :title="t('gmData.seminar.empty')" />
    <div v-else class="sem">
      <article v-for="(g, i) in groups" :key="g.groupId" class="sem__group">
        <span class="sem__order m-num">{{ i + 1 }}</span>
        <span class="sem__text">
          <span class="pv__strong pv__ellipsis">{{ lookup.name('seminar', g.groupId) }}</span>
        </span>
        <span class="sem__bar" role="progressbar" :aria-valuenow="pct(g.cleared, g.total)" aria-valuemin="0" aria-valuemax="100">
          <span :style="{ width: `${pct(g.cleared, g.total)}%` }" />
        </span>
        <span class="sem__count m-num">{{ g.cleared }} / {{ g.total }}</span>
        <StatusPill
          :variant="g.total > 0 && g.cleared >= g.total ? 'success' : g.cleared ? 'primary' : 'neutral'"
          :label="g.total > 0 && g.cleared >= g.total ? t('gmData.seminar.full') : g.cleared ? t('gmData.seminar.partial') : t('gmData.seminar.none')"
        />
      </article>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getPlayerSeminar } from '@/api/gakumas'
import type { OperationDef } from '@/games/mgp'
import type { SeminarProgress } from '@/games/gakumas/types'
import { getRequestErrorMessage } from '@/utils/request'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import { useGmFormat } from '../shared/format'
import { useCatalogLookup } from '../shared/lookup'

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const lookup = useCatalogLookup()
const { pct } = useGmFormat()

const groups = shallowRef<SeminarProgress[]>([])
const loading = ref(false)
const error = ref('')
let seq = 0

const load = async () => {
  const id = ++seq
  loading.value = true
  error.value = ''
  try {
    const data = await getPlayerSeminar(ctx.serverId.value, page.pid.value)
    if (id === seq) groups.value = data.groups
  } catch (e) {
    if (id === seq) {
      error.value = getRequestErrorMessage(e)
      groups.value = []
    }
  } finally {
    if (id === seq) loading.value = false
  }
}
watch(page.version, () => void load(), { immediate: true })

const totalExams = computed(() => groups.value.reduce((s, g) => s + g.total, 0))
const cleared = computed(() => groups.value.reduce((s, g) => s + g.cleared, 0))

const operations = computed(() => ['seminar.clear', 'seminar.reset'].map((id) => ctx.operation(id)).filter((c): c is OperationDef => !!c))
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
