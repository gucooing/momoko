<!-- 检查项列表（预检 / 连接测试）：状态贴纸 + 标题 + 说明，全部来自后端的检查结果。 -->
<template>
  <ul class="ckl">
    <li v-for="(c, i) in checks" :key="`${c.key}-${i}`" class="ckl__item">
      <span class="ckl__ico" :class="`is-${c.state}`" aria-hidden="true">
        <UIcon :name="icon(c.state)" />
      </span>
      <span class="ckl__text">
        <b>{{ text(c).title }}</b>
        <small v-if="text(c).desc">{{ text(c).desc }}</small>
      </span>
    </li>
  </ul>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { GameServerCheck } from '@/types/v1/game_server'
import { checkText } from '../shared'

defineProps<{ checks: GameServerCheck[] }>()
const { t, te } = useI18n()
const text = (c: GameServerCheck) => checkText(t, te, c)
const icon = (state: string) => {
  switch (state) {
    case 'ok':
      return 'i-lucide-check'
    case 'warn':
      return 'i-lucide-triangle-alert'
    case 'fail':
      return 'i-lucide-x'
    default:
      return 'i-lucide-clock'
  }
}
</script>

<style scoped lang="scss">
.ckl {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.ckl__item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.ckl__ico {
  flex: none;
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 9px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.ckl__ico.is-ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.ckl__ico.is-warn {
  background: var(--m-warnSoft);
  color: var(--m-warn);
}
.ckl__ico.is-fail {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.ckl__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 0.875rem;
}
.ckl__text small {
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
</style>
