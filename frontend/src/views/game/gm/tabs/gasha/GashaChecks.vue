<!-- 发布前检查：按草稿实时计算（概率合计、UP 落在哪个槽位、目录里有没有这些卡、交换点是否覆盖开放时间、按钮…），
     最后一项是服务端校验（MGP 试运行，用游戏服务端启动时相同的加载逻辑校验整份配置）。 -->
<template>
  <AppPanel :title="t('gmConfig.checks.title')" title-icon="HOutline:ShieldCheckIcon">
    <ul class="gc">
      <li v-for="(c, i) in checks" :key="i" class="gc__item">
        <span class="gc__icon" :class="`is-${c.level}`"><UIcon :name="ICONS[c.level]" /></span>
        <span class="gc__text">{{ labels.problem(c, ed.cfg.value, ed.gasha.value) }}</span>
      </li>
      <li class="gc__item">
        <span class="gc__icon" :class="`is-${serverLevel}`"><UIcon :name="ICONS[serverLevel]" :class="{ 'gc__spin': state.validating.value }" /></span>
        <span class="gc__text">
          {{ serverText }}
          <span v-if="serverError" class="gc__error m-mono">{{ serverError }}</span>
        </span>
        <UButton class="gmc-btn" size="xs" color="neutral" variant="outline" :loading="state.validating.value" @click="validate">{{ t('gmConfig.checks.runServer') }}</UButton>
      </li>
    </ul>
  </AppPanel>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useGashaEditor } from './editor'
import { gashaChecks, type CheckLevel } from './gasha'
import { useGashaLabels } from './labels'

const ICONS: Record<CheckLevel, string> = {
  ok: 'i-lucide-check',
  warn: 'i-lucide-triangle-alert',
  bad: 'i-lucide-x',
  pending: 'i-lucide-loader-circle',
}

const { t } = useI18n()
const ed = useGashaEditor()
const labels = useGashaLabels()
const state = ed.state

const checks = computed(() => {
  const c = ed.cfg.value
  const g = ed.gasha.value
  if (!c || !g) return []
  return gashaChecks(c, g, { rarityOf: ed.rarityOf, cardStatus: ed.cardStatus })
})

const serverLevel = computed<CheckLevel>(() => {
  if (state.validating.value) return 'pending'
  if (!state.validationFresh.value || !state.validation.value) return 'warn'
  return state.validation.value.valid ? 'ok' : 'bad'
})
const serverText = computed(() => {
  if (state.validating.value) return t('gmConfig.publish.validating')
  const v = state.validation.value
  if (!state.validationFresh.value || !v) return t('gmConfig.checks.serverNotRun')
  if (v.failed) return t('gmConfig.publish.validateFailed')
  return v.valid ? t('gmConfig.checks.serverOk') : t('gmConfig.checks.serverBad')
})
const serverError = computed(() => (state.validationFresh.value && state.validation.value && !state.validation.value.valid ? state.validation.value.error : ''))

const validate = async () => {
  await state.flush()
  await state.validate()
}
</script>

<style lang="scss" src="./shared/form.scss"></style>
<style scoped lang="scss">
.gc {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.gc__item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  font-size: 13px;
  line-height: 1.5;
}
.gc__icon {
  flex: none;
  width: 24px;
  height: 24px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.gc__icon.is-ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.gc__icon.is-warn {
  background: var(--m-warnSoft);
  color: var(--m-warn);
}
.gc__icon.is-bad {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.gc__icon.is-pending {
  background: var(--m-s2);
  color: var(--el-text-color-secondary);
}
.gc__spin {
  animation: gc-spin 1s linear infinite;
}
@keyframes gc-spin {
  to {
    transform: rotate(360deg);
  }
}
.gc__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow-wrap: anywhere;
}
.gc__error {
  font-size: 12px;
  color: var(--m-bad);
}
</style>
