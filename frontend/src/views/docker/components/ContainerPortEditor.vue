<!-- 端口映射编辑器（结构化行）：主机 IP | 主机端口 → 容器端口 | TCP/UDP 分段 | 删除；底部虚线「添加」。
     v-model = 端口行数组；创建页与编辑弹窗共用。手机端每行折成两排。 -->
<template>
  <div class="spec-ed" :class="{ 'is-bare': bare }">
    <div v-if="!bare" class="spec-ed__title">{{ t('docker.container.ports') }}</div>
    <div v-if="model.length" class="spec-ed__labels spec-ed__grid">
      <span>{{ t('docker.common.hostIp') }}</span>
      <span>{{ t('docker.common.hostPort') }}</span>
      <span />
      <span>{{ t('docker.common.containerPort') }}</span>
      <span>{{ t('docker.common.protocol') }}</span>
      <span />
    </div>
    <div v-for="(row, i) in model" :key="i" class="spec-ed__row spec-ed__grid">
      <input
        v-model="row.hostIp"
        class="app-input m-mono spec-ed__ip"
        placeholder="0.0.0.0"
        :aria-label="t('docker.common.hostIp')"
      />
      <input
        v-model="row.hostPort"
        class="app-input m-mono spec-ed__hp"
        inputmode="numeric"
        placeholder="8080"
        :aria-label="t('docker.common.hostPort')"
      />
      <span class="spec-ed__arrow" aria-hidden="true">→</span>
      <input
        v-model="row.containerPort"
        class="app-input m-mono spec-ed__cp"
        inputmode="numeric"
        placeholder="80"
        :aria-label="t('docker.common.containerPort')"
      />
      <div class="spec-seg spec-ed__proto" role="radiogroup" :aria-label="t('docker.common.protocol')">
        <button
          v-for="p in protocols"
          :key="p"
          type="button"
          role="radio"
          :aria-checked="row.protocol === p"
          class="spec-seg__btn"
          :class="{ 'is-on': row.protocol === p }"
          @click="row.protocol = p"
        >{{ p.toUpperCase() }}</button>
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
import type { PortRow } from './specTypes'

withDefaults(
  defineProps<{
    /** 去掉自带标题（外层分区已有标题时用） */
    bare?: boolean
    emptyText?: string
    addText?: string
  }>(),
  { bare: false, emptyText: '', addText: '' },
)
const model = defineModel<PortRow[]>({ default: () => [] })
const { t } = useI18n()
const menuStore = useMenuStore()

const protocols = ['tcp', 'udp'] as const
const add = () => { model.value = [...model.value, { hostIp: '', hostPort: '', containerPort: '', protocol: 'tcp' }] }
const remove = (i: number) => { model.value = model.value.filter((_, idx) => idx !== i) }
</script>

<style scoped lang="scss">
@use './spec-editor.scss';

.spec-ed__grid {
  grid-template-columns: minmax(0, 1fr) minmax(0, 0.8fr) 18px minmax(0, 0.8fr) 112px 40px;
}
@media (width < 640px) {
  .spec-ed__labels { display: none; }
  .spec-ed__row {
    grid-template-columns: minmax(0, 1fr) 18px minmax(0, 1fr);
    grid-template-areas:
      'hp arrow cp'
      'ip ip ip'
      'proto proto del';
    padding: 10px;
    border-radius: var(--app-radius);
    background: var(--el-fill-color-light);
  }
  .spec-ed__hp { grid-area: hp; }
  .spec-ed__arrow { grid-area: arrow; }
  .spec-ed__cp { grid-area: cp; }
  .spec-ed__ip { grid-area: ip; }
  .spec-ed__proto { grid-area: proto; justify-self: start; width: 140px; }
  .spec-ed__del { grid-area: del; justify-self: end; }
}
</style>
