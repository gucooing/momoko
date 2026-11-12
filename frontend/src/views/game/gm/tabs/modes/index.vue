<!-- GM 运营 · 玩法：服务端实际配置了的玩法（manifest modes）各一个标签：
     竞赛排名 + 阵容、全服培育列表、研修通关、塔排名。 -->
<template>
  <div class="gmo game-skin">
    <PageHeader :title="t('gm.tabs.modes')" :back="false" :theme-toggle="false" />

    <EmptyState v-if="!modes.length" icon="HOutline:TrophyIcon" :title="t('gmData.modes.empty')" />
    <template v-else>
      <nav class="gmo__tabs" role="tablist" :aria-label="t('gm.tabs.modes')">
        <button
          v-for="m in modes"
          :key="m.id"
          type="button"
          role="tab"
          class="gmo__tab"
          :class="{ 'is-on': m.id === current?.id }"
          :aria-selected="m.id === current?.id"
          @click="active = m.id"
        >
          {{ m.title }}
        </button>
      </nav>

      <KeepAlive>
        <component :is="VIEWS[current.id]" v-if="current" :key="current.id" />
      </KeepAlive>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { enabledModes, type GakumasMode } from '@/games/gakumas/modes'
import { useGmContext } from '../../context'

defineOptions({ name: 'GmModesTab' })

const { t } = useI18n()
const ctx = useGmContext()

const VIEWS: Record<GakumasMode, Component> = {
  pvp: defineAsyncComponent(() => import('./PvpMode.vue')),
  produce: defineAsyncComponent(() => import('./ProduceMode.vue')),
  seminar: defineAsyncComponent(() => import('./SeminarMode.vue')),
  tower: defineAsyncComponent(() => import('./TowerMode.vue')),
}
const modes = computed(() => enabledModes(ctx.modes.value))
const active = ref('')
const current = computed(() => modes.value.find((m) => m.id === active.value) || modes.value[0])
</script>

<style scoped lang="scss">
.gmo {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-width: 0;
}
.gmo__tabs {
  display: flex;
  gap: 28px;
  overflow-x: auto;
  border-bottom: 1px solid var(--el-border-color-light);
  scrollbar-width: none;
}
.gmo__tabs::-webkit-scrollbar {
  display: none;
}
.gmo__tab {
  flex: none;
  min-height: 44px;
  padding: 0 2px;
  border: 0;
  border-bottom: 3px solid transparent;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 15px;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.gmo__tab:hover {
  color: var(--el-text-color-primary);
}
.gmo__tab.is-on {
  color: var(--el-text-color-primary);
  border-bottom-color: var(--m-g);
}
@media (max-width: 760px) {
  .gmo__tabs {
    gap: 20px;
  }
}
</style>
