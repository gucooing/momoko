<!-- 端口映射编辑器：主机端口 → 容器端口 | TCP/UDP | 删除；主机端口被占用时标红并说明被谁占用。
     主机端口可写成「127.0.0.1:8080」只绑定到某个地址。PC 一行一条，手机每条一张小卡片。 -->
<template>
  <div class="spec-ed">
    <template v-if="!menuStore.isMobile">
      <div v-if="model.length" class="spec-grid spec-head pe-cols">
        <span>{{ t('docker.common.hostPort') }}</span><span /><span>{{ t('docker.common.containerPort') }}</span><span>{{ t('docker.common.protocol') }}</span><span />
      </div>
      <div v-for="(row, i) in model" :key="i" class="spec-item">
        <div class="spec-grid pe-cols">
          <span class="spec-fld" :class="{ 'is-bad': !!conflictOf(row) }">
            <input
              :value="hostText(row)"
              inputmode="numeric"
              placeholder="8080"
              :aria-label="t('docker.common.hostPort')"
              @input="setHost(row, ($event.target as HTMLInputElement).value)"
            />
          </span>
          <UIcon name="i-lucide-arrow-right" class="spec-arrow" />
          <span class="spec-fld">
            <input v-model.trim="row.containerPort" inputmode="numeric" placeholder="80" :aria-label="t('docker.common.containerPort')" />
          </span>
          <div class="spec-seg" role="radiogroup" :aria-label="t('docker.common.protocol')">
            <button v-for="p in PROTOCOLS" :key="p" type="button" role="radio" :aria-checked="row.protocol === p" :class="{ 'is-on': row.protocol === p }" @click="row.protocol = p">
              {{ p.toUpperCase() }}
            </button>
          </div>
          <AppIconButton class="spec-del" icon="HOutline:TrashIcon" :label="t('docker.wizard.removePort')" :box="36" :size="17" @click="remove(i)" />
        </div>
        <span v-if="conflictOf(row)" class="spec-err">
          <UIcon name="i-lucide-triangle-alert" />{{ t('docker.wizard.portTaken', { port: row.hostPort, owner: conflictOf(row) }) }}
        </span>
      </div>
    </template>

    <template v-else>
      <div v-for="(row, i) in model" :key="i" class="spec-card">
        <div class="pe-mrow">
          <label class="spec-card__label">
            {{ t('docker.common.hostPort') }}
            <span class="spec-fld" :class="{ 'is-bad': !!conflictOf(row) }">
              <input :value="hostText(row)" inputmode="numeric" placeholder="8080" @input="setHost(row, ($event.target as HTMLInputElement).value)" />
            </span>
          </label>
          <UIcon name="i-lucide-arrow-right" class="spec-arrow pe-marrow" />
          <label class="spec-card__label">
            {{ t('docker.common.containerPort') }}
            <span class="spec-fld"><input v-model.trim="row.containerPort" inputmode="numeric" placeholder="80" /></span>
          </label>
        </div>
        <span v-if="conflictOf(row)" class="spec-err">
          <UIcon name="i-lucide-triangle-alert" />{{ t('docker.wizard.portTaken', { port: row.hostPort, owner: conflictOf(row) }) }}
        </span>
        <div class="spec-card__foot">
          <div class="spec-seg" role="radiogroup" :aria-label="t('docker.common.protocol')">
            <button v-for="p in PROTOCOLS" :key="p" type="button" role="radio" :aria-checked="row.protocol === p" :class="{ 'is-on': row.protocol === p }" @click="row.protocol = p">
              {{ p.toUpperCase() }}
            </button>
          </div>
          <AppIconButton class="spec-del" icon="HOutline:TrashIcon" :label="t('docker.wizard.removePort')" :box="44" @click="remove(i)" />
        </div>
      </div>
      <button type="button" class="spec-add" @click="add"><UIcon name="i-lucide-plus" />{{ t('docker.wizard.addPortLong') }}</button>
    </template>

    <div v-if="!model.length && emptyText && !menuStore.isMobile" class="spec-empty">{{ emptyText }}</div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { PortRow } from './specTypes'

const props = defineProps<{
  /** 主机端口 → 占用者（容器名等），用于冲突提示 */
  taken?: Record<string, string>
  emptyText?: string
}>()
const model = defineModel<PortRow[]>({ required: true })
const { t } = useI18n()
const menuStore = useMenuStore()
const PROTOCOLS = ['tcp', 'udp'] as const

const hostText = (row: PortRow) => (row.hostIp ? `${row.hostIp.includes(':') ? `[${row.hostIp}]` : row.hostIp}:${row.hostPort}` : row.hostPort)
const setHost = (row: PortRow, text: string) => {
  const value = text.trim()
  const idx = value.lastIndexOf(':')
  if (idx > 0) {
    row.hostIp = value.slice(0, idx).replace(/^\[|\]$/g, '')
    row.hostPort = value.slice(idx + 1)
  } else {
    row.hostIp = ''
    row.hostPort = value
  }
}
const conflictOf = (row: PortRow) => {
  const port = row.hostPort.trim()
  if (!port || !props.taken) return ''
  return props.taken[`${port}/${row.protocol}`] || ''
}
const add = () => {
  model.value = [...model.value, { hostIp: '', hostPort: '', containerPort: '', protocol: 'tcp' }]
}
const remove = (i: number) => {
  model.value = model.value.filter((_, idx) => idx !== i)
}
defineExpose({ add })
</script>

<style scoped lang="scss">
@use './spec-editor.scss';

.pe-cols {
  grid-template-columns: minmax(0, 1fr) 20px minmax(0, 1fr) 128px 36px;
}
.pe-mrow {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 20px minmax(0, 1fr);
  align-items: end;
  gap: 8px;
  padding-right: 6px;
}
.pe-marrow {
  margin-bottom: 14px;
}
</style>
