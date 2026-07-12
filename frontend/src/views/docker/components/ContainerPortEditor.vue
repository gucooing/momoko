<!-- 端口映射编辑器（结构化行，替代 “8080:80” 文本域）：主机IP | 主机端口 | → | 容器端口 | 协议 | 删除。
     v-model = 端口行数组；创建/编辑共用。 -->
<template>
  <div class="spec-ed">
    <div class="spec-ed__head">
      <span>{{ t('docker.container.ports') }}</span>
      <button type="button" class="spec-ed__add" @click="add">
        <component :is="menuStore.iconComponents['HOutline:PlusIcon']" />{{ t('docker.common.add') }}
      </button>
    </div>
    <div v-if="model.length" class="spec-ed__labels">
      <span>{{ t('docker.common.hostIp') }}</span>
      <span>{{ t('docker.common.hostPort') }}</span>
      <span />
      <span>{{ t('docker.common.containerPort') }}</span>
      <span>{{ t('docker.common.protocol') }}</span>
      <span />
    </div>
    <div v-for="(row, i) in model" :key="i" class="spec-ed__row">
      <input v-model="row.hostIp" class="app-input" placeholder="0.0.0.0" />
      <input v-model="row.hostPort" class="app-input" placeholder="8080" />
      <span class="spec-ed__arrow">→</span>
      <input v-model="row.containerPort" class="app-input" placeholder="80" />
      <AppSelect v-model="row.protocol" :options="protocols" />
      <AppIconButton icon="HOutline:TrashIcon" :label="t('docker.common.delete')" :box="30" @click="remove(i)" />
    </div>
    <div v-if="!model.length" class="spec-ed__empty">{{ t('docker.container.emptyPorts') }}</div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { PortRow } from './specTypes'

const model = defineModel<PortRow[]>({ default: () => [] })
const { t } = useI18n()
const menuStore = useMenuStore()

const protocols = [{ label: 'tcp', value: 'tcp' as const }, { label: 'udp', value: 'udp' as const }]
const add = () => { model.value = [...model.value, { hostIp: '', hostPort: '', containerPort: '', protocol: 'tcp' }] }
const remove = (i: number) => { model.value = model.value.filter((_, idx) => idx !== i) }
</script>

<style scoped lang="scss">
.spec-ed { border: 1px solid var(--el-border-color-lighter); border-radius: var(--app-radius); padding: 10px; }
.spec-ed__head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; font-size: 0.8125rem; font-weight: 600; color: var(--el-text-color-primary); }
.spec-ed__add { display: inline-flex; align-items: center; gap: 3px; border: none; background: transparent; color: var(--el-color-primary); font-size: 0.8125rem; cursor: pointer; }
.spec-ed__add :deep(svg) { width: 14px; height: 14px; }
.spec-ed__labels, .spec-ed__row { display: grid; grid-template-columns: 1.2fr 0.9fr auto 0.9fr 0.8fr 30px; gap: 6px; align-items: center; }
.spec-ed__labels { margin-bottom: 4px; }
.spec-ed__labels span { font-size: 0.7rem; color: var(--el-text-color-placeholder); }
.spec-ed__row { margin-bottom: 6px; }
.spec-ed__arrow { color: var(--el-text-color-placeholder); text-align: center; }
.spec-ed__empty { padding: 8px; text-align: center; color: var(--el-text-color-placeholder); font-size: 0.78rem; }
</style>
