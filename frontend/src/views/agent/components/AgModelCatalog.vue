<!-- 模型目录：一行一个模型（名字 + 收起时的摘要），点开改它的设置。第三方接口的配置和官方账户的配置用的都是它，按场景显示：
       - editable（第三方接口）：摘要是推理等级 + 上下文和输出；点开改上下文、输出和推理等级；能删模型、在最后一行手动添加；
       - 不带 editable（官方账户）：模型是 Agent 给的，摘要只有上下文和输出（没设写 defaultLabel）；点开只改这两项。 -->
<template>
  <div class="ag-pv-models">
    <div v-for="m in models" :key="m.id" class="ag-pv-model">
      <div class="ag-pv-model__head">
        <!-- 点这一行展开或收起它的配置 -->
        <button type="button" class="ag-pv-model__toggle" :aria-expanded="open === m.id" @click="open = open === m.id ? '' : m.id">
          <UIcon :name="open === m.id ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="ag-pv-model__chev" />
          <span class="ag-pv-model__n" :class="{ 'ag-mono': !m.name }" :title="m.id">{{ m.name || m.id }}</span>
          <span class="ag-pv-model__sum">{{ summary(m) }}</span>
        </button>
        <button
          v-if="editable"
          type="button"
          class="ag-ib ag-ib--sm"
          :aria-label="t('agent.providers.form.removeModel', { name: m.id })"
          @click="remove(m.id)"
        >
          <UIcon name="i-lucide-x" />
        </button>
      </div>
      <div v-if="open === m.id" class="ag-pv-model__body">
        <span v-if="editable" class="ag-pv-model__lt">{{ t('agent.providers.form.limits') }}</span>
        <div class="ag-pv-levels ag-pv-levels--head" aria-hidden="true">
          <span>{{ t('agent.providers.form.contextWindow') }}</span>
          <span>{{ withOutput ? t('agent.providers.form.maxOutput') : '' }}</span>
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
          <span v-if="withOutput" class="ag-fld ag-fld--plain ag-pv-lv">
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

        <template v-if="editable">
          <div class="ag-pv-row">
            <span class="ag-pv-model__lt">{{ t('agent.providers.form.levels') }}</span>
            <button type="button" class="ag-btn ag-btn--sm ag-btn--plain" @click="addLevel(m)">
              <UIcon name="i-lucide-plus" />{{ t('agent.providers.form.addLevel') }}
            </button>
          </div>
          <div class="ag-pv-levels ag-pv-levels--head" aria-hidden="true">
            <span>{{ t('agent.providers.form.level') }}</span>
            <span>{{ t('agent.providers.form.levelValue') }}</span>
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
              <input
                v-model="r.value"
                class="ag-mono"
                type="text"
                :placeholder="valuePlaceholder(r)"
                :aria-label="t('agent.providers.form.levelValueOf', { name: r.name })"
              />
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
          <span class="ag-fh">{{ t('agent.providers.form.levelsHint') }}</span>
        </template>
      </div>
    </div>
    <span v-if="editable" class="ag-fld ag-pv-add">
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
  /** 能删模型、改推理等级、手动添加（第三方接口） */
  editable?: boolean
  /** 有最长输出这一项（给 Claude 系的） */
  withOutput?: boolean
  /** 上下文、输出没设时显示的字（官方账户是「Agent 默认」） */
  defaultLabel?: string
  /** 上下文、输出下面那一行说明 */
  limitsHint: string
  /** 手动添加时新建一行 */
  create?: (id: string) => CatalogRow
}>()
const models = defineModel<CatalogRow[]>({ required: true })
const { t } = useI18n()

const open = ref('')
const draft = ref('')

// 收起时一行看完：最长上下文（和给 Claude 系时的最长输出）；第三方接口前面还有它的推理等级
const summary = (m: CatalogRow) => {
  const fmt = (v: number | '') => (Number(v) > 0 ? fmtLimit(v) : props.defaultLabel || fmtLimit(v))
  const limits = props.withOutput
    ? t('agent.providers.form.limitsSummary', { ctx: fmt(m.contextWindow), out: fmt(m.maxOutput) })
    : t('agent.providers.form.limitsSummaryCtx', { ctx: fmt(m.contextWindow) })
  if (!props.editable) return limits
  const names = m.levels.map((r) => r.name.trim()).filter(Boolean)
  const levels = names.length ? t('agent.providers.form.levelsSummary', { list: names.join(' · ') }) : t('agent.providers.form.noLevels')
  return `${levels} · ${limits}`
}
const valuePlaceholder = (r: CatalogLevel) =>
  r.name === 'off' ? t('agent.providers.form.offPlaceholder') : r.name || t('agent.providers.form.valuePlaceholder')

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
