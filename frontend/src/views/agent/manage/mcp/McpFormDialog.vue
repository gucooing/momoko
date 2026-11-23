<!-- MCP 服务器的添加 / 编辑弹窗：名称、说明、传输方式（stdio：命令、参数、环境变量；HTTP：地址、请求头）、
     新会话默认开启、测试连接。共享的（Agent 管理页里）多一项「共享给」。密钥只回脱敏后的后几位：已保存的值留空表示不修改。 -->
<template>
  <AgDialog :model-value="modelValue" :title="form.id ? t('agent.mcp.editTitle') : t('agent.mcp.add')" @update:model-value="emit('update:modelValue', $event)">
    <div class="ag-mcp-cols ag-mcp-cols--name ag-grid-2">
      <label class="ag-field">
        <span class="ag-fl">{{ t('agent.common.name') }}</span>
        <span class="ag-fld">
          <input v-model="form.name" class="ag-mono" type="text" autocomplete="off" spellcheck="false" :placeholder="t('agent.mcp.form.namePh')" :aria-label="t('agent.common.name')" />
        </span>
      </label>
      <label class="ag-field">
        <span class="ag-fl">{{ t('agent.mcp.form.descOptional') }}</span>
        <span class="ag-fld">
          <input v-model="form.description" type="text" autocomplete="off" :placeholder="t('agent.mcp.form.descPh')" :aria-label="t('agent.mcp.form.descOptional')" />
        </span>
      </label>
    </div>

    <div class="ag-field">
      <span class="ag-fl">{{ t('agent.mcp.form.transport') }}</span>
      <div class="ag-seg ag-mcp-seg" role="radiogroup" :aria-label="t('agent.mcp.form.transport')">
        <button
          v-for="k in kinds"
          :key="k"
          type="button"
          role="radio"
          :class="{ 'is-on': form.transport === k }"
          :aria-checked="form.transport === k"
          @click="form.transport = k"
        >
          {{ t(`agent.mcp.form.${k}`) }}
        </button>
      </div>
    </div>

    <template v-if="form.transport === 'stdio'">
      <div class="ag-mcp-cols ag-mcp-cols--cmd ag-grid-2">
        <label class="ag-field">
          <span class="ag-fl">{{ t('agent.mcp.form.command') }}</span>
          <span class="ag-fld">
            <input v-model="form.command" class="ag-mono" type="text" autocomplete="off" spellcheck="false" :placeholder="t('agent.mcp.form.commandPh')" :aria-label="t('agent.mcp.form.command')" />
          </span>
        </label>
        <label class="ag-field">
          <span class="ag-fl">{{ t('agent.mcp.form.args') }}</span>
          <span class="ag-fld">
            <input v-model="form.args" class="ag-mono" type="text" autocomplete="off" spellcheck="false" :placeholder="t('agent.mcp.form.argsPh')" :aria-label="t('agent.mcp.form.args')" />
          </span>
        </label>
      </div>
      <div class="ag-field">
        <div class="ag-mcp-kvhead">
          <span class="ag-fl">{{ t('agent.mcp.form.env') }}</span>
          <button type="button" class="ag-btn ag-btn--sm" @click="form.env.push(blankRow())">
            <UIcon name="i-lucide-plus" />{{ t('agent.common.addRow') }}
          </button>
        </div>
        <div v-for="(r, idx) in form.env" :key="r.uid" class="ag-kv ag-mcp-kv">
          <span class="ag-fld">
            <input v-model="r.key" class="ag-mono" type="text" autocomplete="off" spellcheck="false" :placeholder="t('agent.mcp.form.envKey')" :aria-label="t('agent.mcp.form.envKey')" />
          </span>
          <span class="ag-fld">
            <input v-model="r.value" class="ag-mono" type="password" autocomplete="new-password" :placeholder="valuePh(r, 'envValue')" :aria-label="t('agent.mcp.form.envValue')" />
          </span>
          <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.common.deleteRow')" :title="t('agent.common.deleteRow')" @click="form.env.splice(idx, 1)">
            <UIcon name="i-lucide-trash-2" />
          </button>
        </div>
        <span class="ag-fh">{{ t('agent.mcp.form.envHint') }}</span>
      </div>
    </template>

    <template v-else>
      <label class="ag-field">
        <span class="ag-fl">{{ t('agent.mcp.form.url') }}</span>
        <span class="ag-fld">
          <input v-model="form.url" class="ag-mono" type="url" autocomplete="off" spellcheck="false" placeholder="https://" :aria-label="t('agent.mcp.form.url')" />
        </span>
      </label>
      <div class="ag-field">
        <div class="ag-mcp-kvhead">
          <span class="ag-fl">{{ t('agent.mcp.form.headers') }}</span>
          <button type="button" class="ag-btn ag-btn--sm" @click="form.headers.push(blankRow())">
            <UIcon name="i-lucide-plus" />{{ t('agent.common.addRow') }}
          </button>
        </div>
        <div v-for="(r, idx) in form.headers" :key="r.uid" class="ag-kv ag-mcp-kv">
          <span class="ag-fld">
            <input v-model="r.key" class="ag-mono" type="text" autocomplete="off" spellcheck="false" :placeholder="t('agent.mcp.form.headerKey')" :aria-label="t('agent.mcp.form.headerKey')" />
          </span>
          <span class="ag-fld">
            <input v-model="r.value" class="ag-mono" type="password" autocomplete="new-password" :placeholder="valuePh(r, 'headerValue')" :aria-label="t('agent.mcp.form.headerValue')" />
          </span>
          <button type="button" class="ag-ib ag-ib--sm" :aria-label="t('agent.common.deleteRow')" :title="t('agent.common.deleteRow')" @click="form.headers.splice(idx, 1)">
            <UIcon name="i-lucide-trash-2" />
          </button>
        </div>
        <span class="ag-fh">{{ t('agent.mcp.form.headersHint') }}</span>
      </div>
    </template>

    <AgShareField v-if="shared" v-model="share" />

    <label class="ag-mcp-on">
      <span class="ag-mcp-on__text">
        <span class="ag-mcp-on__title">{{ t('agent.mcp.defaultOn') }}</span>
        <span class="ag-fh">{{ t('agent.mcp.defaultOnHint') }}</span>
      </span>
      <AgSwitch v-model="form.defaultOn" :label="t('agent.mcp.defaultOn')" />
    </label>

    <div class="ag-mcp-test">
      <button type="button" class="ag-btn ag-btn--sm ag-btn--plain" :disabled="test.busy" @click="runTest">
        <UIcon :name="test.busy ? 'i-lucide-loader-circle' : 'i-lucide-zap'" :class="{ 'ag-spin': test.busy }" />{{ t('agent.common.testConnection') }}
      </button>
      <span class="ag-pill" :class="testPill.cls">{{ testPill.text }}</span>
      <span class="ag-mcp-test__note" :title="testNote">{{ testNote }}</span>
    </div>

    <template #footer>
      <button type="button" class="ag-btn" @click="emit('update:modelValue', false)">{{ t('agent.common.cancel') }}</button>
      <button type="button" class="ag-btn ag-btn--p" :disabled="saving" @click="save">
        <UIcon v-if="saving" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.common.save') }}
      </button>
    </template>
  </AgDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { type AgentShare, createAgentMcpServer, testAgentMcpServer, updateAgentMcpServer } from '@/api/agent'
import type { AgentMcpServer, AgentSecretKV } from '@/types/v1/agent'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import AgShareField from '@/views/agent/components/AgShareField.vue'
import AgSwitch from '@/views/agent/components/AgSwitch.vue'
import { joinArgs, splitArgs } from './args'

defineOptions({ name: 'AgentMcpFormDialog' })

const props = defineProps<{
  modelValue: boolean
  /** 要编辑的服务器；null 表示添加 */
  server: AgentMcpServer | null
  /** 共享的（Agent 管理页里加、改）：多一项「共享给」 */
  shared?: boolean
}>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
}>()
const { t } = useI18n()

type Transport = 'stdio' | 'http'
const kinds: Transport[] = ['stdio', 'http']

interface KvRow {
  uid: number
  key: string
  value: string
  /** 已保存的值脱敏后的样子；origKey 是保存时的名字，改了名字就当成新的一行 */
  tail: string
  origKey: string
}

let uid = 0
const blankRow = (): KvRow => ({ uid: ++uid, key: '', value: '', tail: '', origKey: '' })
const rowsOf = (list: AgentSecretKV[] | undefined): KvRow[] =>
  (list || []).map((r) => ({ uid: ++uid, key: r.key, value: '', tail: r.tail, origKey: r.key }))
const isSaved = (r: KvRow) => !!r.tail && r.key.trim() === r.origKey
const valuePh = (r: KvRow, fallback: 'envValue' | 'headerValue') =>
  isSaved(r) ? t('agent.mcp.form.savedSecret', { masked: `••••${r.tail}` }) : t(`agent.mcp.form.${fallback}`)
// 留空的已保存行发 value: ''，后端按名字保留原值；被删掉的行不发
const kvOut = (rows: KvRow[]): AgentSecretKV[] =>
  rows.filter((r) => r.key.trim()).map((r) => ({ key: r.key.trim(), value: r.value, tail: '' }))

const form = reactive({
  id: '',
  name: '',
  description: '',
  transport: 'stdio' as Transport,
  command: '',
  args: '',
  env: [] as KvRow[],
  url: '',
  headers: [] as KvRow[],
  defaultOn: true,
})
const saving = ref(false)
// 共享范围，只有 shared 时用
const share = ref<AgentShare>({ share: 'all', shareUserIds: [] })

// —— 测试连接 ——
const test = reactive({ status: '' as '' | 'ok' | 'fail', message: '', tools: [] as string[], busy: false })
let testSeq = 0
const resetTest = () => {
  testSeq += 1
  Object.assign(test, { status: '', message: '', tools: [], busy: false })
}

const draft = (): Partial<AgentMcpServer> => {
  const http = form.transport === 'http'
  return {
    name: form.name.trim(),
    description: form.description.trim(),
    transport: form.transport,
    command: http ? '' : form.command.trim(),
    args: http ? [] : splitArgs(form.args),
    env: http ? [] : kvOut(form.env),
    url: http ? form.url.trim() : '',
    headers: http ? kvOut(form.headers) : [],
    defaultOn: form.defaultOn,
    ...(props.shared ? share.value : {}),
  }
}

// 换了传输方式，或者改了连接用到的任何一项，之前的测试结果就不算数了
const connection = computed(() =>
  JSON.stringify([form.transport, form.command, form.args, form.url, kvOut(form.env), kvOut(form.headers)]),
)
watch(connection, resetTest)

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    const s = props.server
    Object.assign(form, {
      id: s?.id ?? '',
      name: s?.name ?? '',
      description: s?.description ?? '',
      transport: s?.transport === 'http' ? 'http' : 'stdio',
      command: s?.command ?? '',
      args: joinArgs(s?.args ?? []),
      env: rowsOf(s?.env),
      url: s?.url ?? '',
      headers: rowsOf(s?.headers),
      defaultOn: s ? s.defaultOn : true,
    })
    share.value = {
      share: s?.share === 'users' ? 'users' : 'all',
      shareUserIds: [...(s?.shareUserIds ?? [])],
    }
    saving.value = false
    resetTest()
  },
)

const checkTarget = (): boolean => {
  const http = form.transport === 'http'
  if (http ? form.url.trim() : form.command.trim()) return true
  feedback.warning(t(http ? 'agent.mcp.toast.needUrl' : 'agent.mcp.toast.needCommand'))
  return false
}

const runTest = async () => {
  if (test.busy || !checkTarget()) return
  resetTest()
  const seq = testSeq
  test.busy = true
  try {
    const { data } = await testAgentMcpServer({ id: form.id || undefined, draft: draft() })
    if (seq !== testSeq) return
    Object.assign(test, { status: data.ok ? 'ok' : 'fail', message: data.message || '', tools: data.tools || [] })
  } catch {
    // 请求层已经提示过
  } finally {
    if (seq === testSeq) test.busy = false
  }
}

const testPill = computed(() => {
  if (test.status === 'ok') return { cls: 'ag-pill--ok', text: t('agent.mcp.status.ok') }
  if (test.status === 'fail') return { cls: 'ag-pill--bad', text: t('agent.mcp.status.fail') }
  return { cls: '', text: t('agent.common.notTested') }
})
const testNote = computed(() => {
  if (test.busy) return t('agent.mcp.test.running')
  if (test.status === 'fail') return test.message
  if (test.status !== 'ok') return t('agent.mcp.test.idle')
  const n = test.tools.length
  if (!n) return t('agent.mcp.test.ok')
  const names = test.tools.slice(0, 3).join(t('agent.mcp.test.sep'))
  return t(n > 3 ? 'agent.mcp.test.okToolsMore' : 'agent.mcp.test.okTools', { n, names })
})

const save = async () => {
  if (saving.value) return
  if (!form.name.trim()) {
    feedback.warning(t('agent.mcp.toast.needName'))
    return
  }
  if (!checkTarget()) return
  if (props.shared && share.value.share === 'users' && !share.value.shareUserIds.length) {
    feedback.warning(t('agent.share.needUsers'))
    return
  }
  saving.value = true
  try {
    const name = form.name.trim()
    await (form.id ? updateAgentMcpServer(form.id, draft()) : createAgentMcpServer(draft()))
    feedback.success(t(form.id ? 'agent.mcp.toast.saved' : 'agent.mcp.toast.added', { name }))
    emit('saved')
    emit('update:modelValue', false)
  } catch {
    // 请求层已经提示过，弹窗留着
  } finally {
    saving.value = false
  }
}
</script>

<style lang="scss">
.ag-mcp-cols {
  display: grid;
  gap: 8px;
}
.ag-mcp-cols--name {
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr);
}
.ag-mcp-cols--cmd {
  grid-template-columns: minmax(0, 1fr) minmax(0, 2.2fr);
}
.ag-mcp-kvhead {
  display: flex;
  align-items: center;
  gap: 6px;
}
.ag-mcp-kvhead .ag-fl {
  flex: 1;
}
.ag-mcp-on {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 10px;
  border: 1px solid var(--m-line);
  cursor: pointer;
}
.ag-mcp-on__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ag-mcp-on__title {
  font-size: 13px;
  font-weight: 600;
}
.ag-mcp-test {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 10px;
  background: var(--m-s2);
}
.ag-mcp-test__note {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--m-ink2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
@media (width < 992px) {
  // 两个选项的字比较长，手机上叠成两行
  .ag-seg.ag-mcp-seg {
    flex-direction: column;
    border-radius: 14px;
  }
  .ag-seg.ag-mcp-seg button {
    min-height: 36px;
    border-radius: 11px;
  }
  // 行尾的删除按钮在手机上是 40 宽
  .ag-kv.ag-mcp-kv {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1.6fr) 40px;
  }
  .ag-mcp-test {
    flex-wrap: wrap;
  }
  .ag-mcp-test__note {
    flex: 1 1 100%;
    white-space: normal;
  }
}
</style>
