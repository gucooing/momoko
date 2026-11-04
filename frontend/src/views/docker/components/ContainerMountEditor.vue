<!-- 挂载编辑器：主机路径（可浏览主机目录）→ 容器路径 | 读写/只读 | 删除。
     主机路径不是路径（不以 / . ~ 或盘符开头）时按命名卷挂载，留空为匿名卷。PC 一行一条，手机每条一张小卡片。 -->
<template>
  <div class="spec-ed">
    <template v-if="!menuStore.isMobile">
      <div v-if="model.length" class="spec-grid spec-head me-cols">
        <span>{{ t('docker.wizard.hostPath') }}</span><span /><span>{{ t('docker.wizard.containerPath') }}</span><span>{{ t('docker.wizard.access') }}</span><span />
      </div>
      <div v-for="(row, i) in model" :key="i" class="spec-grid me-cols">
        <span class="spec-fld">
          <UIcon :name="row.type === 'volume' && row.source ? 'i-lucide-database' : 'i-lucide-folder'" />
          <input :value="row.source" :placeholder="t('docker.wizard.hostPathPlaceholder')" :aria-label="t('docker.wizard.hostPath')" @input="setSource(row, ($event.target as HTMLInputElement).value)" />
          <AppIconButton v-if="browsable" icon="HOutline:EllipsisHorizontalIcon" :label="t('docker.wizard.browseHost')" :box="32" :size="16" @click="browse(i)" />
        </span>
        <UIcon name="i-lucide-arrow-right" class="spec-arrow" />
        <span class="spec-fld"><input v-model.trim="row.target" placeholder="/data" :aria-label="t('docker.wizard.containerPath')" /></span>
        <div class="spec-seg" role="radiogroup" :aria-label="t('docker.wizard.access')">
          <button type="button" role="radio" :aria-checked="!row.readOnly" :class="{ 'is-on': !row.readOnly }" @click="row.readOnly = false">{{ t('docker.wizard.rw') }}</button>
          <button type="button" role="radio" :aria-checked="row.readOnly" :class="{ 'is-on': row.readOnly }" @click="row.readOnly = true">{{ t('docker.wizard.ro') }}</button>
        </div>
        <AppIconButton class="spec-del" icon="HOutline:TrashIcon" :label="t('docker.wizard.removeMount')" :box="36" :size="17" @click="remove(i)" />
      </div>
      <div v-if="!model.length && emptyText" class="spec-empty">{{ emptyText }}</div>
    </template>

    <template v-else>
      <div v-for="(row, i) in model" :key="i" class="spec-card">
        <div class="spec-card__body">
          <label class="spec-card__label">
            {{ t('docker.wizard.hostPath') }}
            <span class="spec-fld">
              <UIcon name="i-lucide-folder" />
              <input :value="row.source" :placeholder="t('docker.wizard.hostPathPlaceholder')" @input="setSource(row, ($event.target as HTMLInputElement).value)" />
              <AppIconButton v-if="browsable" icon="HOutline:EllipsisHorizontalIcon" :label="t('docker.wizard.browseHost')" :box="40" :size="18" @click="browse(i)" />
            </span>
          </label>
          <label class="spec-card__label">
            {{ t('docker.wizard.containerPath') }}
            <span class="spec-fld"><UIcon name="i-lucide-box" /><input v-model.trim="row.target" placeholder="/data" /></span>
          </label>
        </div>
        <div class="spec-card__foot">
          <div class="spec-seg" role="radiogroup" :aria-label="t('docker.wizard.access')">
            <button type="button" role="radio" :aria-checked="!row.readOnly" :class="{ 'is-on': !row.readOnly }" @click="row.readOnly = false">{{ t('docker.wizard.rw') }}</button>
            <button type="button" role="radio" :aria-checked="row.readOnly" :class="{ 'is-on': row.readOnly }" @click="row.readOnly = true">{{ t('docker.wizard.ro') }}</button>
          </div>
          <AppIconButton class="spec-del" icon="HOutline:TrashIcon" :label="t('docker.wizard.removeMount')" :box="44" @click="remove(i)" />
        </div>
      </div>
      <button type="button" class="spec-add" @click="add"><UIcon name="i-lucide-plus" />{{ t('docker.wizard.addMount') }}</button>
    </template>

    <FilePicker v-if="browsable" v-model="pickerOpen" :title="t('docker.wizard.browseHost')" @confirm="onPicked" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import FilePicker from '@/components/file/FilePicker.vue'
import type { PickedFile } from '@/components/file/types'
import { isHostPath, type MountRow } from './specTypes'

defineProps<{
  emptyText?: string
  /** 允许浏览主机目录（Docker 跑在本机时） */
  browsable?: boolean
}>()
const model = defineModel<MountRow[]>({ required: true })
const { t } = useI18n()
const menuStore = useMenuStore()

const setSource = (row: MountRow, value: string) => {
  row.source = value.trim()
  row.type = row.source && isHostPath(row.source) ? 'bind' : 'volume'
}
const add = () => {
  model.value = [...model.value, { type: 'bind', source: '', target: '', readOnly: false }]
}
const remove = (i: number) => {
  model.value = model.value.filter((_, idx) => idx !== i)
}

const pickerOpen = ref(false)
const pickingIndex = ref(-1)
const browse = (i: number) => {
  pickingIndex.value = i
  pickerOpen.value = true
}
const onPicked = (items: PickedFile[]) => {
  const row = model.value[pickingIndex.value]
  const path = items[0]?.path
  if (row && path) setSource(row, path)
}
defineExpose({ add })
</script>

<style scoped lang="scss">
@use './spec-editor.scss';

.me-cols {
  grid-template-columns: minmax(0, 1.15fr) 20px minmax(0, 1fr) 132px 36px;
}
</style>
