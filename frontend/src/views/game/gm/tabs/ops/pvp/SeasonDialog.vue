<!-- 编辑竞赛赛季（运营管理里点日历上的赛季直接打开）：赛季内容（竞赛配置）与准备、开赛、结束时间；
     规则与段位奖励所有赛季共用，在「规则与奖励」里改。 -->
<template>
  <ItemDialog
    :model-value="modelValue"
    :title="season || before ? seasonLabel(t, (season || before)!, ctx.name) : t('gmConfig.ops.views.pvp')"
    :width="680"
    :state="season ? 'ok' : before ? 'removed' : 'missing'"
    :status="season ? periodStatus(season.prepareStartTime || season.startTime, season.endTime, clock.now.value) : undefined"
    :status-label="season ? t(`gmConfig.pvp.status.${seasonStatus(season, clock.now.value)}`) : ''"
    :draft="draftOf(before, season)"
    @update:model-value="emit('update:modelValue', $event)"
    @delete="remove"
    @restore="restore"
  >
    <template v-if="season && cfg">
      <GmSchemaForm :schema="schema" :model-value="season" keep-empty @update:model-value="assignInPlace(season, $event)" />
      <div class="sdl__rules">
        <span class="sdl__rules-text">{{ t('gmConfig.pvp.rulesLine', { n: cfg.maxDailyPlayCount, grades: cfg.grades.length }) }}</span>
        <UButton class="gmc-btn" size="sm" color="neutral" variant="soft" icon="i-lucide-list-ordered" @click="emit('rules')">{{ t('gmConfig.pvp.rulesAndRewards') }}</UButton>
      </div>
      <ul v-if="seasonProblems(cfg, season).length" class="gmc-problems">
        <li v-for="p in seasonProblems(cfg, season)" :key="p.key"><UIcon name="i-lucide-circle-x" />{{ t(`gmConfig.pvp.problems.${p.key}`, p.params || {}) }}</li>
      </ul>
    </template>
  </ItemDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import GmSchemaForm from '../../../components/GmSchemaForm.vue'
import { useGmContext } from '../../../context'
import type { ItemRef } from '../calendar'
import { useOpsDomains } from '../domains'
import { seasonSchema } from '../schemas'
import { assignInPlace, cloneJson, draftOf, periodStatus, useTracked } from '../shared/entry'
import ItemDialog from '../shared/ItemDialog.vue'
import { useServerClock } from '../shared/serverClock'
import { seasonLabel, seasonProblems, seasonStatus } from './pvp'

const props = defineProps<{ modelValue: boolean; id: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; open: [ref: ItemRef]; rules: [] }>()

const { t, te } = useI18n()
const ctx = useGmContext()
const clock = useServerClock(ctx)
const state = useOpsDomains().pvpRate.state
const cfg = computed(() => state.working.value)
const schema = computed(() => seasonSchema(t, te))

const season = useTracked(() => cfg.value?.seasons, (s) => s.pvpRateConfigId, props.id)
const before = computed(() => state.published.value?.seasons.find((s) => s.pvpRateConfigId === props.id))

const remove = () => {
  const c = cfg.value
  const i = c && season.value ? c.seasons.indexOf(season.value) : -1
  if (c && i >= 0) c.seasons.splice(i, 1)
}
const restore = () => {
  if (cfg.value && before.value) cfg.value.seasons.push(cloneJson(before.value))
}
</script>

<style lang="scss" src="../shared/form.scss"></style>
<style scoped lang="scss">
.sdl__rules {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 12px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
}
.sdl__rules-text {
  flex: 1 1 200px;
  font-size: 13px;
}
</style>
