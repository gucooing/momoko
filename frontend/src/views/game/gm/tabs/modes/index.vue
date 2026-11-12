<!-- GM 运营 · 玩法：每个服务端实际配置了的玩法一个标签（manifest modes，标题由服务端给出）。
     已知玩法有专门界面（竞赛排名 + 阵容、全服培育列表、研修通关、塔排名），其他玩法读同名报表用通用表格显示。 -->
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
          {{ m.title || t('game.unnamed') }}
        </button>
      </nav>

      <KeepAlive>
        <component :is="viewOf(current.id)" v-if="current" :key="current.id" :mode="current" />
      </KeepAlive>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useGmContext } from '../../context'

defineOptions({ name: 'GmModesTab' })

const { t } = useI18n()
const ctx = useGmContext()

// 已知玩法 → 专门界面；其他玩法用同名报表的通用显示
const VIEWS: Record<string, Component> = {
  pvp: defineAsyncComponent(() => import('./PvpMode.vue')),
  produce: defineAsyncComponent(() => import('./ProduceMode.vue')),
  seminar: defineAsyncComponent(() => import('./SeminarMode.vue')),
  tower: defineAsyncComponent(() => import('./TowerMode.vue')),
}
const generic = defineAsyncComponent(() => import('./GenericMode.vue'))
const viewOf = (id: string) => VIEWS[id] || generic

const modes = computed(() => ctx.manifest.value?.modes || [])
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
