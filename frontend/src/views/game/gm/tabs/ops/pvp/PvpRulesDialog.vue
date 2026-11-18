<!-- 编辑竞赛的规则与奖励（所有赛季共用）：每日对战次数与对战记录条数、胜负得分、各段位的升段 / 保段分数、粉丝与奖励。 -->
<template>
  <FormDialog :model-value="modelValue" :title="t('gmConfig.pvp.rulesAndRewards')" :width="900" @update:model-value="emit('update:modelValue', $event)">
    <div v-if="cfg" class="prd game-skin" :data-game="ctx.game.value || undefined">
      <GmSchemaForm :schema="schema" :model-value="cfg" keep-empty @update:model-value="assignInPlace(cfg, $event)" />
      <ul v-if="problems.length" class="gmc-problems">
        <li v-for="(p, i) in problems" :key="i"><UIcon name="i-lucide-circle-x" />{{ p }}</li>
      </ul>
    </div>
    <template #footer="{ close }">
      <span class="prd__foot">{{ t('gmConfig.ops.autosaveHint') }}</span>
      <UButton class="gmc-btn" color="primary" @click="close">{{ t('gmConfig.ops.done') }}</UButton>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import GmSchemaForm from '../../../components/GmSchemaForm.vue'
import { useGmContext } from '../../../context'
import { useOpsDomains } from '../domains'
import { pvpRulesSchema } from '../schemas'
import { assignInPlace } from '../shared/entry'
import { gradeProblems, ruleProblems } from './pvp'

defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t, te } = useI18n()
const ctx = useGmContext()
const state = useOpsDomains().pvpRate.state
const cfg = computed(() => state.working.value)
const schema = computed(() => pvpRulesSchema(t, te))

const problems = computed(() => {
  const c = cfg.value
  if (!c) return []
  const p = (key: string) => t(`gmConfig.pvp.problems.${key}`)
  return [...ruleProblems(c).map((x) => p(x.key)), ...c.grades.flatMap((g) => gradeProblems(g).map((x) => `${t('gmConfig.pvp.gradeN', { n: g.grade })}：${p(x.key)}`))]
})
</script>

<style lang="scss" src="../shared/form.scss"></style>
<style scoped lang="scss">
.prd {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.prd__foot {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
