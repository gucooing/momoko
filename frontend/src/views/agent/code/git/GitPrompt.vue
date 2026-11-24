<!-- Git 面板共用的输入框弹窗：几个输入框（分支名、标签说明……）、几个复选框、一组单选（重置方式）。必填的没填时确定按钮不能点。 -->
<template>
  <AgDialog :model-value="!!p" :title="p?.title || ''" :width="420" @update:model-value="(v) => !v && close()">
    <form v-if="p" class="agg-prompt" @submit.prevent="submit">
      <label v-for="f in p.fields" :key="f.key" class="ag-field">
        <span class="ag-fl">{{ f.label }}</span>
        <textarea
          v-if="f.multiline"
          v-model="values[f.key]"
          class="ag-ta"
          rows="3"
          :placeholder="f.placeholder"
          @keydown.enter.ctrl.prevent="submit"
        />
        <span v-else class="ag-fld">
          <input
            v-model="values[f.key]"
            type="text"
            spellcheck="false"
            :class="{ 'ag-mono': f.mono }"
            :placeholder="f.placeholder"
          />
        </span>
      </label>
      <div v-if="p.choice" class="agg-prompt__choice" role="radiogroup">
        <button
          v-for="o in p.choice.options"
          :key="o.value"
          type="button"
          class="ag-opt"
          :class="{ 'is-on': choice === o.value }"
          role="radio"
          :aria-checked="choice === o.value"
          @click="choice = o.value"
        >
          <span class="ag-rd" />
          <span class="agg-prompt__opt">
            <span :class="{ 'agc-bad': o.danger }">{{ o.label }}</span>
            <span v-if="o.hint" class="ag-muted agc-small">{{ o.hint }}</span>
          </span>
        </button>
      </div>
      <label v-for="c in p.checks ?? []" :key="c.key" class="agg-check">
        <input v-model="checks[c.key]" type="checkbox" class="ag-cb" />
        <span>{{ c.label }}</span>
      </label>
      <div class="ag-confirm__acts">
        <button type="button" class="ag-btn" @click="close">{{ t('agent.common.cancel') }}</button>
        <button type="submit" class="ag-btn" :class="danger ? 'ag-btn--stop' : 'ag-btn--p'" :disabled="busy || !ready">
          <UIcon v-if="busy" name="i-lucide-loader-circle" class="ag-spin" />
          {{ p.confirmText || t('agent.common.confirm') }}
        </button>
      </div>
    </form>
  </AgDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import { useGit } from './useGit'

const { t } = useI18n()
const git = useGit()
const p = computed(() => git.promptState.value)
const values = reactive<Record<string, string>>({})
const checks = reactive<Record<string, boolean>>({})
const choice = ref('')
const busy = ref(false)

watch(
  p,
  async (next) => {
    if (!next) return
    for (const k of Object.keys(values)) delete values[k]
    for (const k of Object.keys(checks)) delete checks[k]
    for (const f of next.fields) values[f.key] = f.value ?? ''
    for (const c of next.checks ?? []) checks[c.key] = !!c.value
    choice.value = next.choice?.value ?? ''
    await nextTick()
    const input = document.querySelector<HTMLInputElement>('.agg-prompt input[type="text"], .agg-prompt textarea')
    input?.focus()
    input?.select()
  },
  { immediate: true },
)
const ready = computed(() => !!p.value && p.value.fields.every((f) => !f.required || !!values[f.key]?.trim()))
const danger = computed(() => !!p.value?.danger || !!p.value?.choice?.options.find((o) => o.value === choice.value)?.danger)
const close = () => {
  git.promptState.value = null
}
const submit = async () => {
  const cur = p.value
  if (!cur || busy.value || !ready.value) return
  busy.value = true
  const trimmed = Object.fromEntries(Object.entries(values).map(([k, v]) => [k, v.trim()]))
  try {
    close()
    await cur.run(trimmed, { ...checks }, choice.value)
  } finally {
    busy.value = false
  }
}
</script>
