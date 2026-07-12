<!-- 卷挂载编辑器（结构化行，替代 “vol:/data” 文本域）：类型 | 源 | 目标 | 只读 | 删除。v-model = 挂载行数组。 -->
<template>
  <div class="spec-ed">
    <div class="spec-ed__head">
      <span>{{ t('docker.container.mounts') }}</span>
      <button type="button" class="spec-ed__add" @click="add">
        <component :is="menuStore.iconComponents['HOutline:PlusIcon']" />{{ t('docker.common.add') }}
      </button>
    </div>
    <div v-if="model.length" class="spec-ed__labels">
      <span>{{ t('docker.common.type') }}</span>
      <span>{{ t('docker.common.source') }}</span>
      <span>{{ t('docker.common.target') }}</span>
      <span>{{ t('docker.container.readOnly') }}</span>
      <span />
    </div>
    <div v-for="(row, i) in model" :key="i" class="spec-ed__row">
      <AppSelect v-model="row.type" :options="types" />
      <input v-model="row.source" class="app-input" :placeholder="t('docker.common.source')" />
      <input v-model="row.target" class="app-input" placeholder="/data" />
      <AppSwitch v-model="row.readOnly" />
      <AppIconButton icon="HOutline:TrashIcon" :label="t('docker.common.delete')" :box="30" @click="remove(i)" />
    </div>
    <div v-if="!model.length" class="spec-ed__empty">{{ t('docker.container.emptyMounts') }}</div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { MountRow } from './specTypes'

const model = defineModel<MountRow[]>({ default: () => [] })
const { t } = useI18n()
const menuStore = useMenuStore()

const types = [{ label: 'bind', value: 'bind' }, { label: 'volume', value: 'volume' }, { label: 'tmpfs', value: 'tmpfs' }]
const add = () => { model.value = [...model.value, { type: 'bind', source: '', target: '', readOnly: false }] }
const remove = (i: number) => { model.value = model.value.filter((_, idx) => idx !== i) }
</script>

<style scoped lang="scss">
.spec-ed { border: 1px solid var(--el-border-color-lighter); border-radius: var(--app-radius); padding: 10px; }
.spec-ed__head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; font-size: 0.8125rem; font-weight: 600; color: var(--el-text-color-primary); }
.spec-ed__add { display: inline-flex; align-items: center; gap: 3px; border: none; background: transparent; color: var(--el-color-primary); font-size: 0.8125rem; cursor: pointer; }
.spec-ed__add :deep(svg) { width: 14px; height: 14px; }
.spec-ed__labels, .spec-ed__row { display: grid; grid-template-columns: 0.9fr 1.4fr 1.2fr auto 30px; gap: 6px; align-items: center; }
.spec-ed__labels { margin-bottom: 4px; }
.spec-ed__labels span { font-size: 0.7rem; color: var(--el-text-color-placeholder); }
.spec-ed__row { margin-bottom: 6px; }
.spec-ed__empty { padding: 8px; text-align: center; color: var(--el-text-color-placeholder); font-size: 0.78rem; }
</style>
