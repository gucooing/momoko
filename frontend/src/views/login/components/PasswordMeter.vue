<!-- 密码强度：四段条 + 「强度：x」+ 说明；仅作提示，最少位数由表单校验（与后端 MinPasswordLength 一致）。 -->
<template>
  <div class="pm">
    <div class="pm__row" aria-live="polite">
      <span class="pm__bars" aria-hidden="true">
        <span v-for="n in 4" :key="n" :class="password && n <= score ? `is-${tone}` : ''" />
      </span>
      <span class="pm__label" :class="password ? `is-${tone}` : ''">
        {{ t('authPage.strength.label', { level: password ? label : '—' }) }}
      </span>
    </div>
    <span v-if="hint" class="pm__hint">
      <span :class="{ 'pm__long': hintShort }">{{ hint }}</span>
      <span v-if="hintShort" class="pm__short">{{ hintShort }}</span>
    </span>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { scorePassword } from '@/views/login/composables/passwordStrength'

const props = defineProps<{
  password: string
  hint?: string
  /** 手机上的短说明 */
  hintShort?: string
}>()
const { t } = useI18n()

const LEVELS = ['veryWeak', 'weak', 'fair', 'strong', 'veryStrong'] as const
const TONES = ['bad', 'bad', 'warn', 'ok', 'ok'] as const

const raw = computed(() => scorePassword(props.password).score)
const score = computed(() => Math.max(raw.value, 1))
const tone = computed(() => TONES[raw.value] ?? 'bad')
const label = computed(() => t(`authPage.strength.levels.${LEVELS[raw.value] ?? 'weak'}`))
</script>

<style scoped lang="scss">
.pm {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.pm__row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.pm__bars {
  display: flex;
  flex: 1;
  gap: 6px;
}
.pm__bars span {
  flex: 1;
  height: 6px;
  border-radius: 999px;
  background: var(--m-line);
  transition: background-color 0.2s;
}
.pm__bars .is-bad {
  background: var(--m-bad);
}
.pm__bars .is-warn {
  background: var(--m-warn);
}
.pm__bars .is-ok {
  background: var(--m-ok);
}
.pm__label {
  flex: none;
  min-width: 56px;
  text-align: right;
  font-size: 0.8125rem;
  font-weight: 800;
  color: var(--el-text-color-secondary);
}
.pm__label.is-bad {
  color: var(--m-bad);
}
.pm__label.is-warn {
  color: var(--m-warn);
}
.pm__label.is-ok {
  color: var(--m-ok);
}
.pm__hint {
  font-size: 0.75rem;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
}
.pm__short {
  display: none;
}
@media (width < 900px) {
  .pm__long {
    display: none;
  }
  .pm__short {
    display: inline;
  }
}
</style>
