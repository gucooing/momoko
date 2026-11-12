<!-- 游戏适配器（弹窗）：随 Momoko 版本内置，决定一款游戏能被托管到什么程度、GM 能做什么；能力由已实现的驱动推出。 -->
<template>
  <FormDialog
    :model-value="modelValue"
    :title="t('gameServers.list.adapters')"
    :width="720"
    :show-footer="false"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="gad">
      <p class="gad__desc">{{ t('gameServers.list.adaptersDesc') }}</p>
      <p v-if="loaded && !adapters.length" class="gad__desc">{{ t('gameServers.list.noAdapters') }}</p>
      <div class="gad__grid">
        <div v-for="a in adapters" :key="a.id" class="gad-card" :data-game="a.id">
          <div class="gad-card__head">
            <span class="gad-card__logo" aria-hidden="true">{{ adapterName(a.id).charAt(0) }}</span>
            <span class="gad-card__title">
              <b>{{ adapterName(a.id) }}</b>
              <small>
                {{ t(`gameServers.list.adapterDriver.${a.gmDriver}`, a.gmDriver) }} ·
                {{ t('gameServers.list.adapterModes', { modes: modesOf(a) }) }}
              </small>
            </span>
          </div>
          <ul class="gad-card__caps">
            <li v-for="c in a.capabilities" :key="c.key" :class="{ 'is-off': !c.enabled }">
              <UIcon :name="c.enabled ? 'i-lucide-check' : 'i-lucide-minus'" />
              {{ t(`gameServers.capDesc.${c.key}`, c.key) }}
            </li>
          </ul>
        </div>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { GameAdapter } from '@/types/v1/game_server'

defineProps<{
  modelValue: boolean
  adapters: GameAdapter[]
  loaded: boolean
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t, te } = useI18n()

const adapterName = (id: string) => (te(`game.adapters.${id}`) ? t(`game.adapters.${id}`) : t('game.unnamed'))
const modesOf = (a: GameAdapter) => a.modes.map((m) => t(`gameServers.mode.${m}`)).join(' / ')
</script>

<style scoped lang="scss">
.gad {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.gad__desc {
  margin: 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.gad__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 14px;
}
.gad-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 18px;
  border-radius: var(--app-radius-lg);
  border: var(--cmp-card-border-width, 1px) solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.gad-card__head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.gad-card__logo {
  flex: none;
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 13px;
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
  font-family: var(--app-font-display);
  font-weight: 800;
  transform: rotate(-4deg);
}
.gad-card__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gad-card__title small {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.gad-card__caps {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 0.8125rem;
}
.gad-card__caps li {
  display: flex;
  align-items: center;
  gap: 8px;
}
.gad-card__caps li.is-off {
  color: var(--el-text-color-secondary);
}
@media (width <= 560px) {
  .gad__grid {
    grid-template-columns: 1fr;
  }
}
</style>
