<!-- 通行证分页：上一页、页码（首页、末页与当前页前后各两页，其余省略）、下一页与总条数；只有一页时只显示总条数。 -->
<template>
  <nav class="ppg" :aria-label="t('common.pager')">
    <template v-if="pages > 1">
      <button type="button" class="ppg__btn" :disabled="page <= 1 || disabled" :aria-label="t('common.prevPage')" @click="go(page - 1)">
        <ChevronLeftIcon aria-hidden="true" />
      </button>
      <template v-for="(p, i) in numbers" :key="`${p}-${i}`">
        <span v-if="p === 0" class="ppg__gap" aria-hidden="true">…</span>
        <button
          v-else
          type="button"
          class="ppg__btn pp-num"
          :class="{ 'is-on': p === page }"
          :aria-current="p === page ? 'page' : undefined"
          :disabled="disabled"
          @click="go(p)"
        >
          {{ p }}
        </button>
      </template>
      <button type="button" class="ppg__btn" :disabled="page >= pages || disabled" :aria-label="t('common.nextPage')" @click="go(page + 1)">
        <ChevronRightIcon aria-hidden="true" />
      </button>
    </template>
    <span class="pp-hint pp-num ppg__total">{{ t('common.pagerTotal', { n: total }) }}</span>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/vue/24/outline'

const props = withDefaults(defineProps<{ page: number; total: number; pageSize: number; disabled?: boolean }>(), { disabled: false })
const emit = defineEmits<{ 'update:page': [page: number] }>()

const { t } = useI18n()

const pages = computed(() => Math.max(1, Math.ceil(props.total / Math.max(1, props.pageSize))))

/** 显示的页码：0 为省略号 */
const numbers = computed(() => {
  const last = pages.value
  const cur = Math.min(Math.max(1, props.page), last)
  const keep = new Set([1, last])
  for (let p = cur - 2; p <= cur + 2; p++) if (p >= 1 && p <= last) keep.add(p)
  const sorted = [...keep].sort((a, b) => a - b)
  const out: number[] = []
  sorted.forEach((p, i) => {
    if (i > 0 && p - (sorted[i - 1] ?? p) > 1) out.push(0)
    out.push(p)
  })
  return out
})

const go = (p: number) => {
  const next = Math.min(Math.max(1, p), pages.value)
  if (next !== props.page) emit('update:page', next)
}
</script>

<style scoped>
.ppg {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 0 4px;
}
.ppg__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 34px;
  height: 34px;
  padding: 0 8px;
  border: 1px solid var(--m-line);
  border-radius: 999px;
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
}
.ppg__btn svg {
  width: 16px;
  height: 16px;
}
.ppg__btn:hover:not(:disabled) {
  border-color: var(--m-g, var(--m-p));
}
.ppg__btn.is-on {
  border-color: var(--m-p);
  background: var(--m-p);
  color: var(--m-pOn);
}
.ppg__btn:disabled:not(.is-on) {
  opacity: 0.45;
  cursor: default;
}
.ppg__gap {
  padding: 0 2px;
  color: var(--m-ink2);
}
.ppg__total {
  margin-left: 6px;
}
</style>
