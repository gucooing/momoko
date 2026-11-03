<!-- 卷挂载编辑器（结构化行）：类型 | 源 → 目标 | 读写/只读 分段 | 删除；底部虚线「添加」。
     v-model = 挂载行数组；创建页与编辑弹窗共用。手机端每行折成三排。 -->
<template>
  <div class="spec-ed" :class="{ 'is-bare': bare }">
    <div v-if="!bare" class="spec-ed__title">{{ t('docker.container.mounts') }}</div>
    <div v-if="model.length" class="spec-ed__labels spec-ed__grid">
      <span>{{ t('docker.common.type') }}</span>
      <span>{{ t('docker.common.source') }}</span>
      <span />
      <span>{{ t('docker.common.target') }}</span>
      <span>{{ t('docker.container.accessMode') }}</span>
      <span />
    </div>
    <div v-for="(row, i) in model" :key="i" class="spec-ed__row spec-ed__grid">
      <div class="spec-ed__type"><AppSelect v-model="row.type" :options="types" /></div>
      <input
        v-model="row.source"
        class="app-input m-mono spec-ed__src"
        :placeholder="row.type === 'volume' ? 'my-volume' : '/srv/data'"
        :aria-label="t('docker.common.source')"
      />
      <span class="spec-ed__arrow" aria-hidden="true">→</span>
      <input
        v-model="row.target"
        class="app-input m-mono spec-ed__dst"
        placeholder="/data"
        :aria-label="t('docker.common.target')"
      />
      <div class="spec-seg spec-ed__rw" role="radiogroup" :aria-label="t('docker.container.accessMode')">
        <button
          type="button"
          role="radio"
          :aria-checked="!row.readOnly"
          class="spec-seg__btn"
          :class="{ 'is-on': !row.readOnly }"
          @click="row.readOnly = false"
        >{{ t('docker.container.readWrite') }}</button>
        <button
          type="button"
          role="radio"
          :aria-checked="row.readOnly"
          class="spec-seg__btn"
          :class="{ 'is-on': row.readOnly }"
          @click="row.readOnly = true"
        >{{ t('docker.container.readOnly') }}</button>
      </div>
      <AppIconButton
        class="spec-ed__del"
        icon="HOutline:TrashIcon"
        :label="t('docker.common.delete')"
        :size="17"
        @click="remove(i)"
      />
    </div>
    <div v-if="!model.length && emptyText" class="spec-ed__empty">{{ emptyText }}</div>
    <button type="button" class="spec-ed__add" @click="add">
      <component :is="menuStore.iconComponents['HOutline:PlusIcon']" />{{ addText || t('docker.common.add') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { MountRow } from './specTypes'

withDefaults(
  defineProps<{
    bare?: boolean
    emptyText?: string
    addText?: string
  }>(),
  { bare: false, emptyText: '', addText: '' },
)
const model = defineModel<MountRow[]>({ default: () => [] })
const { t } = useI18n()
const menuStore = useMenuStore()

const types = [{ label: 'bind', value: 'bind' }, { label: 'volume', value: 'volume' }, { label: 'tmpfs', value: 'tmpfs' }]
const add = () => { model.value = [...model.value, { type: 'bind', source: '', target: '', readOnly: false }] }
const remove = (i: number) => { model.value = model.value.filter((_, idx) => idx !== i) }
</script>

<style scoped lang="scss">
@use './spec-editor.scss';

.spec-ed__grid {
  grid-template-columns: 104px minmax(0, 1.3fr) 18px minmax(0, 1fr) 124px 40px;
}
.spec-ed__type { min-width: 0; }
@media (width < 640px) {
  .spec-ed__labels { display: none; }
  .spec-ed__row {
    grid-template-columns: 104px minmax(0, 1fr) 40px;
    grid-template-areas:
      'src src src'
      'dst dst dst'
      'type rw del';
    padding: 10px;
    border-radius: var(--app-radius);
    background: var(--el-fill-color-light);
  }
  .spec-ed__src { grid-area: src; }
  .spec-ed__arrow { display: none; }
  .spec-ed__dst { grid-area: dst; }
  .spec-ed__type { grid-area: type; }
  .spec-ed__rw { grid-area: rw; }
  .spec-ed__del { grid-area: del; justify-self: end; }
}
</style>
