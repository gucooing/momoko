<!-- 模型目录：一行一个模型（名字 + 收起时的摘要），点开改它的设置。有哪些设置由调用方打开：
       addable 手动添加、删除模型；levels 推理等级（给了 fetchLevels 时还能按 Agent 给的同步）；context 最长上下文；output 最长输出。
       第三方接口全开（输出只给 Claude 系）；官方账户的模型是检查时 Agent 给的，不能增删，上下文和输出看这一系 Agent 有没有这个配置。 -->
<template>
  <div class="ag-pv-models">
    <div v-for="m in models" :key="m.id" class="ag-pv-model">
      <div class="ag-pv-model__head">
        <!-- 点这一行展开或收起它的设置 -->
        <button type="button" class="ag-pv-model__toggle" :aria-expanded="open === m.id" @click="open = open === m.id ? '' : m.id">
          <UIcon :name="open === m.id ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="ag-pv-model__chev" />
          <span class="ag-pv-model__n" :class="{ 'ag-mono': !m.name }" :title="m.id">{{ m.name || m.id }}</span>
          <span class="ag-pv-model__sum">{{ summary(m) }}</span>
        </button>
        <button
          v-if="addable"
          type="button"
          class="ag-ib ag-ib--sm"
          :aria-label="t('agent.providers.form.removeModel', { name: m.id })"
          @click="remove(m.id)"
        >
          <UIcon name="i-lucide-x" />
        </button>
      </div>
      <div v-if="open === m.id" class="ag-pv-model__body">
        <template v-if="context">
          <span v-if="levels" class="ag-pv-model__lt">{{ t('agent.providers.form.limits') }}</span>
          <div class="ag-pv-levels ag-pv-levels--head" aria-hidden="true">
            <span>{{ t('agent.providers.form.contextWindow') }}</span>
            <span>{{ output ? t('agent.providers.form.maxOutput') : '' }}</span>
            <span />
          </div>
          <div class="ag-pv-levels" role="group" :aria-label="t('agent.providers.form.limitsOf', { name: m.name || m.id })">
            <span class="ag-fld ag-fld--plain ag-pv-lv">
              <input
                v-model.number="m.contextWindow"
                class="ag-mono"
                type="number"
                min="1"
                step="1000"
                :placeholder="defaultLabel"
                :aria-label="t('agent.providers.form.contextWindow')"
              />
              <span v-if="Number(m.contextWindow) > 0" class="ag-pv-unit ag-num">{{ fmtLimit(m.contextWindow) }}</span>
            </span>
            <span v-if="output" class="ag-fld ag-fld--plain ag-pv-lv">
              <input
                v-model.number="m.maxOutput"
                class="ag-mono"
                type="number"
                min="1"
                step="1000"
                :placeholder="defaultLabel"
                :aria-label="t('agent.providers.form.maxOutput')"
              />
              <span v-if="Number(m.maxOutput) > 0" class="ag-pv-unit ag-num">{{ fmtLimit(m.maxOutput) }}</span>
            </span>
          </div>
          <span class="ag-fh">{{ limitsHint }}</span>
        </template>

        <template v-if="levels">
          <div class="ag-pv-row">
            <span class="ag-pv-model__lt">{{ t('agent.providers.form.levels') }}</span>
            <button type="button" class="ag-btn ag-btn--sm ag-btn--plain" @click="addLevel(m)">
              <UIcon name="i-lucide-plus" />{{ t('agent.providers.form.addLevel') }}
            </button>
            <button
              v-if="fetchLevels"
              type="button"
              class="ag-btn ag-btn--sm ag-btn--plain"
              :title="t('agent.providers.form.syncLevelsTitle')"
              :disabled="!!syncing"
              @click="syncLevels(m)"
            >
              <UIcon :name="syncing === m.id ? 'i-lucide-loader-circle' : 'i-lucide-refresh-cw'" :class="{ 'ag-spin': syncing === m.id }" />{{
                t('agent.providers.form.syncLevels')
              }}
            </button>
          </div>
          <div class="ag-pv-levels ag-pv-levels--head" aria-hidden="true">
            <span>{{ t('agent.providers.form.level') }}</span>
            <span>{{ valueLabel }}</span>
            <span />
          </div>
          <div v-for="(r, j) in m.levels" :key="r.uid" class="ag-pv-levels">
            <span class="ag-fld ag-fld--plain ag-pv-lv">
              <input
                v-model="r.name"
                class="ag-mono"
                type="text"
                :placeholder="t('agent.providers.form.levelName')"
                :aria-label="t('agent.providers.form.level')"
              />
            </span>
            <span class="ag-fld ag-fld--plain ag-pv-lv">
              <input v-model="r.value" class="ag-mono" type="text" :placeholder="r.name" :aria-label="`${r.name} · ${valueLabel}`" />
            </span>
            <button
              type="button"
              class="ag-ib ag-ib--sm"
              :aria-label="t('agent.providers.form.removeLevel', { name: r.name })"
              @click="m.levels.splice(j, 1)"
            >
              <UIcon name="i-lucide-x" />
            </button>
          </div>
          <span class="ag-fh">{{ levelsHint }}</span>
        </template>
      </div>
    </div>
    <span v-if="addable" class="ag-fld ag-pv-add">
      <input
        v-model="draft"
        class="ag-mono"
        type="text"
        :placeholder="t('agent.providers.form.addModelPlaceholder')"
        :aria-label="t('agent.providers.form.addModel')"
        @keydown.enter.prevent="flush"
      />
    </span>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { type CatalogLevel, type CatalogRow, fmtLimit, LADDER, levelRow, presetLevel } from '@/views/agent/utils/catalog'

const props = defineProps<{
  /** 能手动添加、删除模型 */
  addable?: boolean
  /** 有推理等级 */
  levels?: boolean
  /** 有最长上下文 */
  context?: boolean
  /** 有最长输出 */
  output?: boolean
  /** 上下文、输出没设时显示的字 */
  defaultLabel?: string
  /** 上下文、输出下面那一行说明 */
  limitsHint?: string
  /** 推理等级右边那一列（选中这一级时交出去的值）的名字 */
  valueLabel?: string
  /** 推理等级下面那一行说明 */
  levelsHint?: string
  /** 给了就有「同步」：取这个模型现在的推理等级（name 显示的名字，value 交出去的值），换掉表单里的 */
  fetchLevels?: (model: string) => Promise<{ name: string; value: string }[]>
  /** 手动添加时新建一行 */
  create?: (id: string) => CatalogRow
}>()
const models = defineModel<CatalogRow[]>({ required: true })
const { t } = useI18n()

const open = ref('')
const draft = ref('')
// 正在同步推理等级的模型；同一时间只同步一个
const syncing = ref('')

// 收起时一行看完：推理等级，最长上下文（和最长输出）
const summary = (m: CatalogRow) => {
  const parts: string[] = []
  if (props.levels) {
    const names = m.levels.map((r) => r.name.trim()).filter(Boolean)
    parts.push(names.length ? t('agent.providers.form.levelsSummary', { list: names.join(' · ') }) : t('agent.providers.form.noLevels'))
  }
  if (props.context) {
    const fmt = (v: number | '') => (Number(v) > 0 ? fmtLimit(v) : props.defaultLabel || fmtLimit(v))
    parts.push(
      props.output
        ? t('agent.providers.form.limitsSummary', { ctx: fmt(m.contextWindow), out: fmt(m.maxOutput) })
        : t('agent.providers.form.limitsSummaryCtx', { ctx: fmt(m.contextWindow) }),
    )
  }
  return parts.join(' · ')
}

// 带出比当前最高一档更高的那一档；中间有缺的按顺序补进对应位置；六档全有之后才是空白行
const addLevel = (m: CatalogRow) => {
  const rank = (r: CatalogLevel) => LADDER.indexOf(r.name.trim().toLowerCase())
  const have = m.levels.map((r) => r.name.trim().toLowerCase())
  const top = Math.max(-1, ...m.levels.map(rank))
  const next = LADDER.find((x, q) => q > top && !have.includes(x)) || LADDER.find((x) => !have.includes(x))
  if (!next) {
    m.levels.push(levelRow('', ''))
    return
  }
  const at = m.levels.findIndex((r) => rank(r) > LADDER.indexOf(next))
  if (at < 0) m.levels.push(presetLevel(next))
  else m.levels.splice(at, 0, presetLevel(next))
}

const syncLevels = async (m: CatalogRow) => {
  if (!props.fetchLevels || syncing.value) return
  syncing.value = m.id
  try {
    const levels = await props.fetchLevels(m.id)
    m.levels = levels.map((l) => levelRow(l.name, l.value))
    feedback.success(t(levels.length ? 'agent.providers.form.synced' : 'agent.providers.form.syncedNone', { name: m.name || m.id }))
  } catch {
    /* shown */
  } finally {
    syncing.value = ''
  }
}

const remove = (id: string) => {
  models.value = models.value.filter((m) => m.id !== id)
}

/** 把手动添加框里的模型加进目录（回车时，或者外面要往下走之前） */
const flush = () => {
  const id = draft.value.trim()
  if (!id || !props.create) return
  if (!models.value.some((m) => m.id === id)) models.value = [...models.value, props.create(id)]
  draft.value = ''
  open.value = id
}

/** 收起所有展开的模型、清空添加框（重开弹窗、换供应商类型时） */
const reset = () => {
  open.value = ''
  draft.value = ''
}

defineExpose({ flush, reset })
</script>
