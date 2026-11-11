<!-- 「需要你看一眼」：异常退出的容器、新版本、进行中的后台任务……每条带一个就地操作。
     手机端只展开第一条，其余折叠成「还有 N 条」。没有待办时显示一切正常。 -->
<template>
  <section class="att" :aria-labelledby="headId">
    <div class="att__head">
      <h2 :id="headId" class="att__title">{{ t('dashboard.home.attention') }}</h2>
      <span v-if="items.length" class="att__count m-num">{{ items.length }}</span>
    </div>

    <div v-if="!items.length" class="att__empty">
      <span class="att__sk m-sticker m-sticker--ok" aria-hidden="true">
        <component :is="ic('HOutline:CheckBadgeIcon')" />
      </span>
      <span class="att__text">
        <span class="att__t">{{ t('dashboard.home.attentionNone') }}</span>
        <span class="att__s">{{ t('dashboard.home.attentionNoneDesc') }}</span>
      </span>
    </div>

    <template v-else>
      <div v-for="it in shown" :key="it.key" class="att__row">
        <span class="att__sk m-sticker" :class="`m-sticker--${it.tone}`" aria-hidden="true">
          <component :is="ic(it.icon)" />
        </span>
        <span class="att__text">
          <span class="att__t">{{ it.title }}</span>
          <span class="att__s">{{ it.sub }}</span>
        </span>
        <UButton
          v-if="it.action"
          color="neutral"
          variant="soft"
          size="sm"
          class="att__btn"
          :loading="it.loading"
          @click="it.action.run()"
        >
          {{ it.action.label }}
        </UButton>
      </div>
      <button
        v-if="compact && items.length > 1"
        type="button"
        class="att__more"
        :aria-expanded="expanded"
        @click="expanded = !expanded"
      >
        {{ expanded ? t('dashboard.home.collapse') : t('dashboard.home.moreItems', { n: items.length - 1 }) }}
      </button>
    </template>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { AttentionItem } from './useDashboardExtras'

const props = defineProps<{ items: AttentionItem[]; compact?: boolean }>()

const { t } = useI18n()
const menuStore = useMenuStore()
const ic = (name: string) => menuStore.iconComponents[name]
const headId = `att-h-${Math.random().toString(36).slice(2, 8)}`
const expanded = ref(false)

const shown = computed(() =>
  props.compact && !expanded.value ? props.items.slice(0, 1) : props.items,
)
</script>

<style scoped lang="scss">
.att {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: calc(var(--cmp-card-unit) * 22);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.att__head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.att__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.att__count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 24px;
  padding: 0 7px;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-size: 0.8125rem;
  font-weight: 800;
}
.att__row,
.att__empty {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  margin: 0 -10px;
  border-radius: 16px;
  transition: background 0.15s;
}
.att__row:hover {
  background: var(--el-fill-color-light);
}
.att__sk {
  width: 40px;
  height: 40px;
  border-radius: 13px;
}
.att__sk :deep(svg) {
  width: 20px;
  height: 20px;
}
.att__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.att__t {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.att__s {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.att__btn {
  flex: none;
  min-height: 36px;
  padding-inline: 14px;
  border-radius: 999px;
  font-weight: 700;
}
.att__more {
  align-self: flex-start;
  min-height: 36px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
@media (width < 640px) {
  .att {
    padding: 16px;
    gap: 10px;
  }
  .att__title {
    font-size: 1rem;
  }
  .att__count {
    min-width: 22px;
    height: 22px;
    font-size: 0.75rem;
  }
  .att__row,
  .att__empty {
    padding: 4px 0;
    margin: 0;
  }
  .att__btn {
    min-height: 40px;
  }
}
</style>
