<!-- 按 JSON Schema 渲染表单（GM 操作的参数、运营管理里各项配置都用这一个）。每个字段：标题、说明（description）和输入：
     x-picker：reward 奖励（数组为奖励列表、对象为一项奖励）；backup 玩家存档备份；image 图片（来源由 imageSources[x-images] 给）；
     其余为目录种类（item、character、missionGroup……）——一个选择框显示选好的图和名称，点开按名称搜索；数组为多选
     （选上的是标签，「添加」点开搜索）；属于某个角色的条目带上角色名，不出现 ID。
     format：unix-ms 时间（按服务器时区）、color 颜色、textarea 多行文字、password。x-unit permil 按百分比输入千分率。
     枚举用 x-enum-labels 显示文字，枚举数组为多选标签；对象数组是一组可增删的子表单（x-item-title 为每项的标题）；
     x-group 在字段前加分组标题。locked 里的字段（从某一行点进来时已定好的道具、备份）不能改。 -->
<template>
  <div class="gsf">
    <template v-for="(field, i) in fields" :key="field.key">
      <h4 v-if="field.schema['x-group'] && field.schema['x-group'] !== fields[i - 1]?.schema['x-group']" class="gsf__group-title">
        {{ field.schema['x-group'] }}
      </h4>
      <div class="gsf__field">
        <span v-if="field.schema.title" class="gsf__label">
          {{ field.schema.title }}
          <span v-if="field.required" class="gsf__req" aria-hidden="true">*</span>
        </span>

        <span v-if="field.schema.readOnly" class="gsf__readonly">{{ readOnlyText(field.key, field.schema) }}</span>

        <!-- 奖励列表 -->
        <GmRewardPicker
          v-else-if="field.schema.type === 'array' && field.schema['x-picker'] === 'reward'"
          :model-value="toGmRewards(rewardsOf(field.key))"
          :max="field.schema.maxItems || 20"
          :exclude="excludeRewards"
          :disabled="disabled"
          @update:model-value="set(field.key, mergeRewards(rewardsOf(field.key), $event))"
        />

        <!-- 一项奖励 -->
        <GmRewardPicker
          v-else-if="field.schema.type === 'object' && field.schema['x-picker'] === 'reward'"
          :model-value="toGmRewards(rewardOf(field.key))"
          :max="1"
          :exclude="excludeRewards"
          :disabled="disabled"
          @update:model-value="set(field.key, mergeRewards(rewardOf(field.key), $event)[0] ?? null)"
        />

        <!-- 对象数组：一组子表单 -->
        <div v-else-if="field.schema.type === 'array' && field.schema.items?.type === 'object'" class="gsf__list">
          <div v-for="(item, n) in listOf(field.key)" :key="n" class="gsf__item">
            <div class="gsf__item-head">
              <span class="gsf__item-title">{{ itemTitle(field.schema, n) }}</span>
              <AppIconButton v-if="!disabled" icon="HOutline:TrashIcon" :label="t('gm.form.removeItem')" :box="36" @click="removeItem(field.key, n)" />
            </div>
            <GmSchemaForm
              :schema="itemFormSchema(field.schema.items)"
              :model-value="itemFormValue(field.schema.items, item)"
              :image-sources="imageSources"
              :disabled="disabled"
              @update:model-value="setItem(field.key, field.schema.items, n, $event)"
            />
          </div>
          <p v-if="!listOf(field.key).length" class="gsf__empty">{{ t('gm.form.empty') }}</p>
          <UButton v-if="!disabled" class="gsf__add" size="sm" color="neutral" variant="outline" icon="i-lucide-plus" @click="addItem(field.key, field.schema.items)">
            {{ t('gm.form.addItem') }}
          </UButton>
        </div>

        <!-- 目录多选：选上的显示图和名称，「添加」弹出搜索 -->
        <div v-else-if="field.schema.type === 'array' && catalogKind(field.schema)" class="gsf__chips">
          <span v-for="id in listOf(field.key)" :key="String(id)" class="gsf__chip">
            <GameImage v-if="lookup.image(catalogKind(field.schema)!, String(id))" :src="lookup.image(catalogKind(field.schema)!, String(id))" class="gsf__chip-img" width="28" height="28" />
            <span class="gsf__chip-name">{{ entryName(catalogKind(field.schema)!, String(id)) }}</span>
            <button v-if="!disabled" type="button" class="gsf__chip-x" :aria-label="t('gm.form.removeItem')" @click="removeId(field.key, id)">×</button>
          </span>
          <AppDropdown v-if="!disabled" align="start" :width="440">
            <template #trigger>
              <button type="button" class="gsf__chip gsf__chip-add"><UIcon name="i-lucide-plus" />{{ t('gm.form.add') }}</button>
            </template>
            <div class="gsf__pop">
              <GmCatalogSearch autofocus :kind="catalogKind(field.schema)!" :exclude-ids="listOf(field.key).map(String)" @pick="addId(field.key, field.schema, $event.id)" />
            </div>
          </AppDropdown>
          <span v-else-if="!listOf(field.key).length" class="gsf__empty">{{ t('gm.form.unset') }}</span>
        </div>

        <!-- 枚举多选 -->
        <div v-else-if="field.schema.type === 'array' && field.schema.items?.enum?.length" class="gsf__chips">
          <button
            v-for="v in field.schema.items.enum"
            :key="v"
            type="button"
            class="gsf__toggle"
            :class="{ 'is-on': listOf(field.key).includes(v) }"
            :aria-pressed="listOf(field.key).includes(v)"
            :disabled="disabled"
            @click="toggleEnum(field.key, field.schema, v)"
          >
            {{ enumLabel(field.schema.items, v) }}
          </button>
        </div>

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
            :image-sources="imageSources"
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

        <ImageField
          v-else-if="field.schema['x-picker'] === 'image'"
          :model-value="String(value(field.key) ?? '')"
          :title="field.schema.title || ''"
          :sources="imageSources[field.schema['x-images'] || ''] || []"
          @update:model-value="set(field.key, $event)"
        />

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

        <!-- 目录单选：一个选择框，显示选好的图和名称（没选显示「不设置」），点开搜索 -->
        <AppDropdown v-else-if="catalogKind(field.schema) && !disabled && !locked.includes(field.key)" block align="start" :width="440">
          <template #trigger>
            <button type="button" class="gsf__select">
              <GameImage v-if="value(field.key) && lookup.image(catalogKind(field.schema)!, String(value(field.key)))" :src="lookup.image(catalogKind(field.schema)!, String(value(field.key)))" class="gsf__select-img" width="32" height="32" />
              <span class="gsf__select-text" :class="{ 'is-empty': !value(field.key) }">
                {{ value(field.key) ? entryName(catalogKind(field.schema)!, String(value(field.key))) : t('gm.form.unset') }}
              </span>
              <UIcon name="i-lucide-chevrons-up-down" class="gsf__select-icon" />
            </button>
          </template>
          <template #default="{ close }">
            <div class="gsf__pop">
              <button v-if="value(field.key) && !field.required" type="button" class="gsf__pop-unset" @click="set(field.key, field.schema.type === 'string' ? '' : undefined); close()">
                {{ t('gm.form.unset') }}
              </button>
              <GmCatalogSearch autofocus :kind="catalogKind(field.schema)!" :selected-id="String(value(field.key) ?? '')" @pick="pick(field.key, field.schema, $event.id); close()" />
            </div>
          </template>
        </AppDropdown>
        <div v-else-if="catalogKind(field.schema)" class="gsf__select is-static">
          <GameImage v-if="value(field.key) && lookup.image(catalogKind(field.schema)!, String(value(field.key)))" :src="lookup.image(catalogKind(field.schema)!, String(value(field.key)))" class="gsf__select-img" width="32" height="32" />
          <span class="gsf__select-text" :class="{ 'is-empty': !value(field.key) }">
            {{ value(field.key) ? entryName(catalogKind(field.schema)!, String(value(field.key))) : t('gm.form.unset') }}
          </span>
        </div>

        <DateTimeField
          v-else-if="isTimeField(field.schema)"
          :model-value="numberOf(field.key) || 0"
          :zone="zone"
          :label="field.schema.title"
          :disabled="disabled"
          @update:model-value="set(field.key, $event)"
        />

        <span v-else-if="field.schema['x-unit'] === 'permil'" class="gsf__unit">
          <input
            class="gsf__input m-num"
            type="number"
            step="0.1"
            :value="numberOf(field.key) === undefined ? '' : (numberOf(field.key) as number) / 10"
            :disabled="disabled"
            @input="setPermil(field.key, ($event.target as HTMLInputElement).value)"
          />
          <span class="gsf__unit-u">%</span>
        </span>

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

        <span v-else-if="field.schema.format === 'color'" class="gsf__color">
          <input type="color" :value="`#${String(value(field.key) || 'ffffff')}`" :disabled="disabled" @input="set(field.key, ($event.target as HTMLInputElement).value.slice(1).toUpperCase())" />
          <span class="m-mono">{{ value(field.key) ? `#${value(field.key)}` : t('gm.form.unset') }}</span>
          <UButton v-if="value(field.key) && !disabled" size="xs" color="neutral" variant="ghost" @click="set(field.key, '')">{{ t('gm.form.clear') }}</UButton>
        </span>

        <textarea
          v-else-if="field.schema.format === 'textarea'"
          class="gsf__input gsf__textarea"
          rows="3"
          :value="String(value(field.key) ?? '')"
          :disabled="disabled"
          @input="set(field.key, ($event.target as HTMLTextAreaElement).value)"
        />

        <input
          v-else
          class="gsf__input"
          :type="field.schema.format === 'password' ? 'password' : 'text'"
          :autocomplete="field.schema.format === 'password' ? 'new-password' : 'off'"
          :maxlength="field.schema.maxLength"
          :value="String(value(field.key) ?? '')"
          :disabled="disabled"
          @input="set(field.key, ($event.target as HTMLInputElement).value || (keepEmpty ? '' : undefined))"
        />

        <span v-if="field.schema.description" class="gsf__hint">{{ field.schema.description }}</span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { listPlayerBackups } from '@/api/gakumas'
import type { Backup, ConfigReward } from '@/games/gakumas/types'
import type { JSONSchema } from '@/games/mgp'
import { useGmContext } from '../context'
import { catalogLookupFor } from '../player/shared/lookup'
import type { AssetSource } from '../tabs/ops/shared/assets'
import DateTimeField from '../tabs/ops/shared/DateTimeField.vue'
import ImageField from '../tabs/ops/shared/ImageField.vue'
import { mergeRewards, toGmRewards } from '../tabs/ops/shared/rewards'
import { resolveZone } from '../tabs/ops/shared/time'
import GmCatalogSearch from './GmCatalogSearch.vue'
import GmRewardPicker from './GmRewardPicker.vue'

defineOptions({ name: 'GmSchemaForm' })

const props = withDefaults(
  defineProps<{
    schema: JSONSchema
    /** 正在编辑的对象（操作参数，或运营配置里的一项） */
    modelValue: object
    /** 单个目标玩家（备份选择器用） */
    pid?: string
    /** 奖励里不允许的资源类型 */
    excludeRewards?: number[]
    disabled?: boolean
    /** 不渲染的字段（由使用方单独提供） */
    hidden?: string[]
    /** 已经定好、不能再改的字段（从某一行点进来时带的道具、备份） */
    locked?: string[]
    /** x-picker image 的图片来源（按 x-images 取） */
    imageSources?: Record<string, AssetSource[]>
    /** 文字清空时写空串（配置文件保留字段）；默认去掉这个字段（操作参数） */
    keepEmpty?: boolean
  }>(),
  { pid: undefined, excludeRewards: () => [], disabled: false, hidden: () => [], locked: () => [], imageSources: () => ({}), keepEmpty: false },
)
const emit = defineEmits<{ 'update:modelValue': [value: Record<string, unknown>] }>()

const { t } = useI18n()
const ctx = useGmContext()
const lookup = catalogLookupFor(ctx)
const zone = computed(() => resolveZone(ctx.timezone.value))

const fields = computed(() =>
  Object.entries(props.schema.properties || {})
    .filter(([key]) => !props.hidden.includes(key))
    .map(([key, schema]) => ({ key, schema, required: !!props.schema.required?.includes(key) })),
)

const model = computed(() => props.modelValue as Record<string, unknown>)
const value = (key: string) => model.value[key]
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
  const next = { ...model.value }
  if (v === undefined) delete next[key]
  else next[key] = v
  emit('update:modelValue', next)
}
const setNumber = (key: string, raw: string, integer: boolean) => {
  if (raw === '') return set(key, props.keepEmpty ? 0 : undefined)
  const n = Number(raw)
  if (!Number.isFinite(n)) return
  set(key, integer ? Math.trunc(n) : n)
}
const setPermil = (key: string, raw: string) => {
  const n = Number(raw)
  set(key, raw === '' || !Number.isFinite(n) ? 0 : Math.round(n * 10))
}
const setList = (key: string, schema: JSONSchema, raw: string) => {
  const parts = raw
    .split(/[\n,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  const numeric = schema.items?.type === 'integer' || schema.items?.type === 'number'
  set(key, parts.length ? parts.map((p) => (numeric ? Number(p) : p)) : props.keepEmpty ? [] : undefined)
}

// ---- 奖励 ----
const rewardsOf = (key: string) => (Array.isArray(value(key)) ? (value(key) as ConfigReward[]) : [])
const rewardOf = (key: string) => {
  const v = value(key) as ConfigReward | null | undefined
  return v && v.resourceType ? [v] : []
}
// 选择器给的是 GmReward，用 mergeRewards 写回：没变的保留原对象（键与零值写法不变）

// ---- 枚举 ----
const enumLabel = (schema: JSONSchema, v: string | number) => schema['x-enum-labels']?.[String(v)] ?? String(v)
const enumOptions = (schema: JSONSchema) => [
  ...(schema.default === undefined && !props.keepEmpty ? [{ label: t('gm.form.unset'), value: '' as string | number }] : []),
  ...(schema.enum || []).map((v) => ({ label: enumLabel(schema, v), value: v })),
]
const toggleEnum = (key: string, schema: JSONSchema, v: string | number) => {
  const list = listOf(key)
  const order = schema.items?.enum || []
  const next = list.includes(v) ? list.filter((x) => x !== v) : [...list, v].sort((a, b) => order.indexOf(a as never) - order.indexOf(b as never))
  set(key, next)
}

// ---- 目录（按名称选游戏数据） ----
const NON_CATALOG = ['reward', 'backup', 'image']
const catalogKind = (schema: JSONSchema) => {
  const p = schema['x-picker'] || (schema.type === 'array' ? schema.items?.['x-picker'] : undefined)
  return p && !NON_CATALOG.includes(p) ? p : undefined
}
const isNumeric = (schema: JSONSchema) => {
  const s = schema.type === 'array' ? schema.items : schema
  return s?.type === 'integer' || s?.type === 'number'
}
const pick = (key: string, schema: JSONSchema, id: string) => set(key, isNumeric(schema) ? Number(id) : id)
/** 条目名称；属于某个角色的条目（各角色的亲爱度任务面板等）加上角色名，同名的也分得清 */
const entryName = (kind: string, id: string) => {
  const name = ctx.label(kind, id)
  const characterId = kind === 'character' ? '' : lookup.entry(kind, id)?.characterId
  const who = characterId ? ctx.name('character', characterId) : ''
  return who ? `${name} · ${who}` : name
}
const addId = (key: string, schema: JSONSchema, id: string) => set(key, [...listOf(key), isNumeric(schema) ? Number(id) : id])
const removeId = (key: string, id: unknown) => set(key, listOf(key).filter((x) => x !== id))

// ---- 对象数组：每项一个子表单（每项是一个奖励时包成 { value }，用同一套字段渲染） ----
const wraps = (items: JSONSchema) => items['x-picker'] === 'reward'
const itemFormSchema = (items: JSONSchema): JSONSchema => (wraps(items) ? { type: 'object', properties: { value: { ...items, title: undefined } } } : items)
const itemFormValue = (items: JSONSchema, item: unknown) => (wraps(items) ? { value: item } : ((item || {}) as Record<string, unknown>))
const itemTitle = (schema: JSONSchema, n: number) => (schema['x-item-title'] || t('gm.form.itemN')).replace('{n}', String(n + 1))
const setItem = (key: string, items: JSONSchema, n: number, v: Record<string, unknown>) => {
  const next = listOf(key).slice()
  next[n] = wraps(items) ? v.value : v
  set(key, next)
}
const removeItem = (key: string, n: number) => {
  const next = listOf(key).slice()
  next.splice(n, 1)
  set(key, next)
}
// 新的一项照上一项复制（没有时用 default 或空值），改起来最省事
const addItem = (key: string, items: JSONSchema) => {
  const list = listOf(key)
  const last = list[list.length - 1]
  const blank = items.default !== undefined ? items.default : items.type === 'object' ? (items['x-picker'] === 'reward' ? null : {}) : items.type === 'string' ? '' : 0
  set(key, [...list, JSON.parse(JSON.stringify(last ?? blank))])
}

// ---- 只读与时间 ----
const readOnlyText = (key: string, schema: JSONSchema) => {
  const v = value(key)
  if (schema.enum?.length) return enumLabel(schema, v as string | number)
  return v === undefined || v === '' ? '—' : String(v)
}
// Unix 毫秒：format unix-ms，或 GM 操作参数里标题注明 Unix 的整数
const isTimeField = (schema: JSONSchema) => schema.format === 'unix-ms' || (schema.type === 'integer' && /unix/i.test(schema.title || ''))

// ---- 备份选择器：读取目标玩家的存档备份 ----
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
.gsf__group-title {
  margin: 6px 0 0;
  padding-top: 10px;
  border-top: 1px solid var(--el-border-color-light);
  font-size: 14px;
  font-weight: 800;
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
.gsf__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.gsf__readonly {
  display: flex;
  align-items: center;
  min-height: 40px;
  padding: 0 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  font-size: 14px;
  font-weight: 600;
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
  min-width: 0;
}
.gsf__list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.gsf__item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.gsf__item-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.gsf__item-title {
  font-size: 13px;
  font-weight: 800;
}
.gsf__empty {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gsf__add {
  align-self: flex-start;
}
.gsf__select {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 40px;
  padding: 3px 12px 3px 4px;
  border-radius: var(--app-radius-sm);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gsf__select.is-static {
  cursor: default;
  background: var(--el-fill-color-light);
}
.gsf__select-img {
  flex: none;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  object-fit: contain;
}
.gsf__select-text {
  flex: 1;
  min-width: 0;
  padding-left: 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 700;
}
.gsf__select-img + .gsf__select-text {
  padding-left: 0;
}
.gsf__select-text.is-empty {
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.gsf__select-icon {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.gsf__pop {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
}
.gsf__pop-unset {
  min-height: 36px;
  padding: 0 8px;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-weight: 700;
  text-align: left;
  cursor: pointer;
}
.gsf__pop-unset:hover {
  background: var(--el-fill-color-light);
}
.gsf__chosen {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 4px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
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
.gsf__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.gsf__chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  padding: 3px 6px 3px 3px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  font-size: 12px;
  font-weight: 700;
}
.gsf__chip-img {
  flex: none;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: contain;
}
.gsf__chip-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gsf__chip-add {
  min-height: 34px;
  padding: 0 12px;
  border: 1px dashed var(--el-border-color);
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}
.gsf__chip-add:hover {
  color: var(--m-pInk);
  border-color: var(--m-p);
}
.gsf__chip-x {
  width: 24px;
  height: 24px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 16px;
  cursor: pointer;
}
.gsf__toggle {
  min-height: 36px;
  padding: 0 12px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}
.gsf__toggle.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.gsf__unit {
  display: flex;
  align-items: center;
  gap: 8px;
}
.gsf__unit .gsf__input {
  flex: 1;
}
.gsf__unit-u {
  font-size: 13px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gsf__color {
  display: flex;
  align-items: center;
  gap: 10px;
}
.gsf__color input[type='color'] {
  width: 48px;
  height: 36px;
  padding: 0;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-sm);
  background: transparent;
}
</style>
