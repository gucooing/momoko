<!-- 按操作参数的 JSON Schema 渲染表单（games/<游戏>/operations.ts 里声明；x-picker 提示用选择器代替原始输入）：
     reward → 奖励选择，item → 道具目录，backup → 该玩家的存档备份。
     已经选好的道具只显示图和名称（可「更换」）；locked 里的字段（从某一行点进来时已定好的道具、备份）不能改，只填其余的。 -->
<template>
  <div class="gsf">
    <div v-for="field in fields" :key="field.key" class="gsf__field">
      <span class="gsf__label">
        {{ field.schema.title || field.key }}
        <span v-if="field.required" class="gsf__req" aria-hidden="true">*</span>
      </span>

      <GmRewardPicker
        v-if="field.schema.type === 'array' && field.schema['x-picker'] === 'reward'"
        :model-value="rewardsOf(field.key)"
        :max="field.schema.maxItems || 20"
        :exclude="excludeRewards"
        :disabled="disabled"
        @update:model-value="set(field.key, $event)"
      />

      <textarea
        v-else-if="field.schema.type === 'array'"
        class="gsf__input gsf__textarea"
        rows="3"
        :placeholder="t('gm.form.listHint')"
        :value="listOf(field.key).join('\n')"
        :disabled="disabled"
        @input="setList(field.key, field.schema, ($event.target as HTMLTextAreaElement).value)"
      />

      <fieldset v-else-if="field.schema.type === 'object'" class="gsf__group">
        <GmSchemaForm
          :schema="field.schema"
          :model-value="objectOf(field.key)"
          :pid="pid"
          :disabled="disabled"
          @update:model-value="set(field.key, $event)"
        />
      </fieldset>

      <AppSwitch
        v-else-if="field.schema.type === 'boolean'"
        :model-value="!!value(field.key)"
        :disabled="disabled"
        @update:model-value="set(field.key, $event)"
      />

      <AppSelect
        v-else-if="field.schema.enum?.length"
        :model-value="scalarOf(field.key)"
        :options="enumOptions(field.schema)"
        :disabled="disabled"
        @update:model-value="set(field.key, $event === '' ? undefined : $event)"
      />

      <div v-else-if="field.schema['x-picker'] === 'item'" class="gsf__picker">
        <div v-if="value(field.key)" class="gsf__chosen">
          <GameImage v-if="itemImage(field.key)" :src="itemImage(field.key)" class="gsf__chosen-img" width="44" height="44" />
          <span class="gsf__chosen-name">{{ ctx.label('item', String(value(field.key))) }}</span>
          <UButton
            v-if="!disabled && !locked.includes(field.key) && !changing.has(field.key)"
            size="xs"
            color="neutral"
            variant="outline"
            @click="changing.add(field.key)"
          >
            {{ t('gm.form.change') }}
          </UButton>
        </div>
        <GmCatalogSearch
          v-if="!value(field.key) || changing.has(field.key)"
          kind="item"
          :selected-id="String(value(field.key) ?? '')"
          @pick="pickItem(field.key, $event.id)"
        />
      </div>

      <div v-else-if="field.schema['x-picker'] === 'backup' && locked.includes(field.key) && value(field.key) !== undefined" class="gsf__chosen">
        <span class="gsf__chosen-name">{{ backupOptions.find((o) => o.value === value(field.key))?.label || t('gm.form.loading') }}</span>
        <span class="gsf__chosen-sub">{{ backupOptions.find((o) => o.value === value(field.key))?.description }}</span>
      </div>

      <AppSelect
        v-else-if="field.schema['x-picker'] === 'backup'"
        :model-value="scalarOf(field.key)"
        :options="backupOptions"
        :placeholder="backupsLoading ? t('gm.form.loading') : backupOptions.length ? undefined : t('gm.form.noBackups')"
        :disabled="disabled || !backupOptions.length"
        @update:model-value="set(field.key, $event === '' ? undefined : Number($event))"
      />

      <input
        v-else-if="isTimeField(field.schema)"
        class="gsf__input"
        type="datetime-local"
        :value="toLocalInput(numberOf(field.key))"
        :disabled="disabled"
        @input="set(field.key, fromLocalInput(($event.target as HTMLInputElement).value))"
      />

      <input
        v-else-if="field.schema.type === 'integer' || field.schema.type === 'number'"
        class="gsf__input m-num"
        type="number"
        :min="field.schema.minimum"
        :max="field.schema.maximum"
        :step="field.schema.type === 'integer' ? 1 : 'any'"
        :value="numberOf(field.key)"
        :disabled="disabled"
        @input="setNumber(field.key, ($event.target as HTMLInputElement).value, field.schema.type === 'integer')"
      />

      <input
        v-else
        class="gsf__input"
        :type="field.schema.format === 'password' ? 'password' : 'text'"
        :autocomplete="field.schema.format === 'password' ? 'new-password' : 'off'"
        :maxlength="field.schema.maxLength"
        :value="String(value(field.key) ?? '')"
        :disabled="disabled"
        @input="set(field.key, ($event.target as HTMLInputElement).value || undefined)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { listPlayerBackups } from '@/api/gakumas'
import type { Backup } from '@/games/gakumas/types'
import type { JSONSchema } from '@/games/mgp'
import { useGmContext } from '../context'
import GmCatalogSearch from './GmCatalogSearch.vue'
import GmRewardPicker from './GmRewardPicker.vue'
import { catalogLookupFor } from '../player/shared/lookup'
import type { GmReward } from './types'

defineOptions({ name: 'GmSchemaForm' })

const props = withDefaults(
  defineProps<{
    schema: JSONSchema
    modelValue: Record<string, unknown>
    /** 单个目标玩家（备份选择器用） */
    pid?: string
    /** 奖励里不允许的资源类型 */
    excludeRewards?: number[]
    disabled?: boolean
    /** 不渲染的字段（由使用方单独提供） */
    hidden?: string[]
    /** 已经定好、不能再改的字段（从某一行点进来时带的道具、备份） */
    locked?: string[]
  }>(),
  { pid: undefined, excludeRewards: () => [], disabled: false, hidden: () => [], locked: () => [] },
)
const emit = defineEmits<{ 'update:modelValue': [value: Record<string, unknown>] }>()

const { t } = useI18n()
const ctx = useGmContext()

const fields = computed(() =>
  Object.entries(props.schema.properties || {})
    .filter(([key]) => !props.hidden.includes(key))
    .map(([key, schema]) => ({ key, schema, required: !!props.schema.required?.includes(key) })),
)

const value = (key: string) => props.modelValue[key]

// 道具：选好后只显示图和名称；点「更换」才展开目录搜索
const changing = reactive(new Set<string>())
const lookup = catalogLookupFor(ctx)
const itemImage = (key: string) => {
  const v = value(key)
  return typeof v === 'string' && v ? lookup.image('item', v) : ''
}
const pickItem = (key: string, id: string) => {
  set(key, id)
  changing.delete(key)
}
const rewardsOf = (key: string) => (Array.isArray(value(key)) ? (value(key) as GmReward[]) : [])
const listOf = (key: string) => (Array.isArray(value(key)) ? (value(key) as unknown[]) : [])
const objectOf = (key: string) => {
  const v = value(key)
  return v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : {}
}
const scalarOf = (key: string): string | number => {
  const v = value(key)
  return typeof v === 'string' || typeof v === 'number' ? v : ''
}
const numberOf = (key: string) => {
  const v = value(key)
  return typeof v === 'number' ? v : undefined
}

const set = (key: string, v: unknown) => {
  const next = { ...props.modelValue }
  if (v === undefined) delete next[key]
  else next[key] = v
  emit('update:modelValue', next)
}

const setNumber = (key: string, raw: string, integer: boolean) => {
  if (raw === '') return set(key, undefined)
  const n = Number(raw)
  if (!Number.isFinite(n)) return
  set(key, integer ? Math.trunc(n) : n)
}

const setList = (key: string, schema: JSONSchema, raw: string) => {
  const parts = raw
    .split(/[\n,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  const numeric = schema.items?.type === 'integer' || schema.items?.type === 'number'
  set(key, parts.length ? parts.map((p) => (numeric ? Number(p) : p)) : undefined)
}

const enumOptions = (schema: JSONSchema) => [
  { label: t('gm.form.unset'), value: '' as string | number },
  ...(schema.enum || []).map((v) => ({ label: String(v), value: v })),
]

// 服务端用 Unix 毫秒表示时间的整数字段（标题里注明 Unix）：用日期时间输入。
const isTimeField = (schema: JSONSchema) => schema.type === 'integer' && /unix/i.test(schema.title || '')
const toLocalInput = (ms: number | undefined) => (ms ? dayjs(ms).format('YYYY-MM-DDTHH:mm') : '')
const fromLocalInput = (raw: string) => (raw ? dayjs(raw).valueOf() : undefined)

// 备份选择器：读取目标玩家的存档备份。
const backups = ref<Backup[]>([])
const backupsLoading = ref(false)
const needsBackups = computed(() => fields.value.some((f) => f.schema['x-picker'] === 'backup'))
const backupOptions = computed(() =>
  backups.value.map((b) => ({
    label: b.createdAt ? dayjs(b.createdAt).format('YYYY-MM-DD HH:mm') : t('game.unnamed'),
    value: b.id,
    description: [t(`gmData.backups.kind.${b.kind}`), b.note].filter(Boolean).join(' · '),
  })),
)
watch(
  () => [needsBackups.value, props.pid] as const,
  async ([need, pid]) => {
    if (!need || !pid) return
    backupsLoading.value = true
    try {
      backups.value = (await listPlayerBackups(ctx.serverId.value, pid)).backups
    } catch {
      backups.value = []
    } finally {
      backupsLoading.value = false
    }
  },
  { immediate: true },
)
</script>

<style scoped>
.gsf {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}
.gsf__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.gsf__label {
  font-size: 13px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.gsf__req {
  color: var(--m-bad);
  margin-left: 2px;
}
.gsf__input {
  min-height: 40px;
  padding: 0 12px;
  border-radius: var(--app-radius-sm);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
}
.gsf__textarea {
  padding: 8px 12px;
  resize: vertical;
}
.gsf__group {
  margin: 0;
  padding: 12px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-sm);
}
.gsf__picker {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gsf__chosen {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 4px 12px 4px 4px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
}
.gsf__chosen-img {
  flex: none;
  width: 44px;
  height: 44px;
  object-fit: contain;
}
.gsf__chosen-name {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 700;
}
.gsf__chosen-sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
