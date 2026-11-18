<!-- 新建卡池：填名称、选一个卡池作模板（复制它的全部设置，不含交换点）。卡池 ID 由模板 ID 自动生成，界面上不出现。 -->
<template>
  <FormDialog :model-value="modelValue" :title="t('gmConfig.gasha.create')" :width="520" :show-footer="false" @update:model-value="emit('update:modelValue', $event)">
    <div class="gcd game-skin" :data-game="ctx.game.value || undefined">
      <label class="gmc-field">
        <span class="gmc-label">{{ t('gmConfig.gasha.newName') }}</span>
        <input v-model.trim="name" class="gmc-input" :class="{ 'is-invalid': !name }" />
        <span class="gmc-hint">{{ t('gmConfig.gasha.newNameHint') }}</span>
      </label>
      <div class="gmc-field">
        <span class="gmc-label">{{ t('gmConfig.gasha.template') }}</span>
        <AppSelect v-model="from" :options="templateOptions" searchable />
        <span class="gmc-hint">{{ t('gmConfig.gasha.templateHint') }}</span>
      </div>
      <div class="gcd__actions">
        <UButton class="gmc-btn" color="neutral" variant="ghost" @click="emit('update:modelValue', false)">{{ t('gmConfig.common.cancel') }}</UButton>
        <UButton class="gmc-btn" :disabled="!name || !from" @click="create">{{ t('gmConfig.common.ok') }}</UButton>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GashaDef } from '@/games/gakumas/types'
import { useGmContext } from '../../../context'
import { useOpsDomains } from '../domains'
import { cloneJson, freeId, nextHour } from '../shared/entry'
import { useServerClock } from '../shared/serverClock'
import { formatDay } from '../shared/time'
import { useGashaLabels } from './labels'

const props = withDefaults(defineProps<{ modelValue: boolean; template?: string }>(), { template: '' })
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; created: [id: string] }>()

const { t } = useI18n()
const ctx = useGmContext()
const clock = useServerClock(ctx)
const labels = useGashaLabels()
const state = useOpsDomains().gasha.state

const name = ref('')
const from = ref('')
const ordered = computed(() => [...(state.working.value?.gashas || [])].sort((a, b) => b.startTime - a.startTime))
const templateOptions = computed(() =>
  ordered.value.map((g) => ({ label: labels.gasha(g), value: g.id, description: g.endTime ? `${formatDay(g.startTime, clock.zone.value)}–${formatDay(g.endTime, clock.zone.value)}` : '' })),
)

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    from.value = props.template || ordered.value[0]?.id || ''
    const src = state.working.value?.gashas.find((g) => g.id === from.value)
    name.value = props.template && src?.name ? t('gmConfig.ops.copyName', { name: src.name }) : ''
  },
)

// 新卡池从下一个整点开始，开放时长沿用模板（常驻的仍是常驻）；交换点按卡池各自计数，不沿用模板的
const create = () => {
  const c = state.working.value
  const src = c?.gashas.find((g) => g.id === from.value)
  if (!c || !src || !name.value) return
  const copy = cloneJson(src) as GashaDef
  copy.id = freeId((v) => c.gashas.some((g) => g.id === v), `${src.id}-copy`)
  copy.name = name.value
  copy.gashaPointId = ''
  const start = nextHour(clock.now.value)
  copy.startTime = start
  copy.endTime = src.endTime ? start + (src.endTime - src.startTime) : 0
  c.gashas.splice(c.gashas.indexOf(src) + 1, 0, copy)
  emit('update:modelValue', false)
  emit('created', copy.id)
}
</script>

<style lang="scss" src="../shared/form.scss"></style>
<style scoped lang="scss">
.gcd {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.gcd__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
