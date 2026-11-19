<!-- Agent 管理 · 通知：Agent 在服务器上跑，关了网页还在继续，所以要能通知到人。
     三个时机开关 + 四种方式（Discord、Telegram、飞书、QQBot）；必填项填完才能保存，保存后自动启用。 -->
<template>
  <div class="ag ag-page">
    <AgManageHeader>{{ subtitle }}</AgManageHeader>

    <div class="ag-body">
      <h2 class="ag-sh">{{ t('agent.notify.when') }}</h2>
      <div class="ag-grp">
        <div v-for="key in EVENTS" :key="key" class="ag-rw ag-nt-ev">
          <div class="ag-rm">
            <div class="ag-rt"><span>{{ t(`agent.notify.events.${key}.title`) }}</span></div>
            <div class="ag-rs">{{ t(`agent.notify.events.${key}.desc`) }}</div>
          </div>
          <div class="ag-rc">
            <AgSwitch
              :model-value="events[key]"
              :label="t(`agent.notify.events.${key}.title`)"
              :disabled="!loaded"
              @update:model-value="setEvent(key, $event)"
            />
          </div>
        </div>
      </div>

      <h2 class="ag-sh ag-gap">{{ t('agent.notify.where') }}</h2>
      <div class="ag-grp">
        <div v-for="c in rows" :key="c.kind" class="ag-rw">
          <div class="ag-rm">
            <div class="ag-rt">
              <span>{{ c.name }}</span>
              <span class="ag-pill" :class="{ 'ag-pill--ok': c.on }">{{ c.statusText }}</span>
            </div>
            <div class="ag-rs">{{ c.sub }}</div>
          </div>
          <div class="ag-rc">
            <template v-if="c.configured">
              <AgSwitch
                class="ag-nt-sw"
                :model-value="c.on"
                :label="t('agent.notify.enable', { name: c.name })"
                @update:model-value="setEnabled(c.kind, $event)"
              />
              <button type="button" class="ag-btn ag-btn--sm ag-nt-test" :disabled="testing.has(c.kind)" @click="sendTest(c)">
                <UIcon v-if="testing.has(c.kind)" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.notify.test') }}
              </button>
            </template>
            <button type="button" class="ag-btn ag-btn--sm" :disabled="!loaded" @click="openConfig(c)">
              {{ c.configured ? t('agent.notify.edit') : t('agent.notify.setup') }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <AgDialog v-model="dlg.open" :title="t('agent.notify.dialogTitle', { name: dlgRow.name })" :width="460">
      <label v-for="f in dlgFields" :key="f.key" class="ag-field">
        <span class="ag-fl">{{ f.label }}</span>
        <span class="ag-fld">
          <input
            v-model="dlg.values[f.key]"
            class="ag-mono"
            :type="f.type"
            :placeholder="f.placeholder"
            :aria-label="f.label"
            :autocomplete="f.type === 'password' ? 'new-password' : 'off'"
          />
        </span>
      </label>
      <div class="ag-nt-acts">
        <button type="button" class="ag-btn" :disabled="dlg.testing" @click="testDraft">
          <UIcon v-if="dlg.testing" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.notify.test') }}
        </button>
        <span class="ag-nt-acts__sp" />
        <button type="button" class="ag-btn" @click="dlg.open = false">{{ t('agent.common.cancel') }}</button>
        <button type="button" class="ag-btn ag-btn--p" :disabled="dlg.saving" @click="saveDraft">
          <UIcon v-if="dlg.saving" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.common.save') }}
        </button>
      </div>
    </AgDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  getAgentNotify,
  saveAgentNotifyChannel,
  setAgentNotifyChannelEnabled,
  testAgentNotifyChannel,
  updateAgentNotifyEvents,
} from '@/api/agent'
import type { AgentNotifyChannel, AgentNotifyEvents } from '@/types/v1/agent'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import AgManageHeader from '@/views/agent/components/AgManageHeader.vue'
import AgSwitch from '@/views/agent/components/AgSwitch.vue'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'

defineOptions({ name: 'AgentManageNotifyView' })

type EventKey = keyof AgentNotifyEvents
type ChannelKind = 'discord' | 'telegram' | 'feishu' | 'qq'
interface FieldDef {
  key: string
  type: 'url' | 'password' | 'text'
  /** 密钥类：接口只返回脱敏值，已配置时留空表示不修改 */
  secret: boolean
  optional?: boolean
  /** 不随语言变的示例；没有就用文案里的 <key>Ph */
  example?: string
}

const EVENTS: EventKey[] = ['ask', 'done', 'error']
// 四种方式各自要填的。名称、说明和字段文案在 agent.notify.channels.<kind> 下。QQBot 是 QQ 官方机器人。
const CHANNELS: { kind: ChannelKind; fields: FieldDef[] }[] = [
  { kind: 'discord', fields: [{ key: 'url', type: 'url', secret: true, example: 'https://discord.com/api/webhooks/…' }] },
  {
    kind: 'telegram',
    fields: [
      { key: 'token', type: 'password', secret: true },
      { key: 'chat', type: 'text', secret: false },
    ],
  },
  {
    kind: 'feishu',
    fields: [
      { key: 'url', type: 'url', secret: true, example: 'https://open.feishu.cn/open-apis/bot/v2/hook/…' },
      { key: 'secret', type: 'password', secret: true, optional: true },
    ],
  },
  {
    kind: 'qq',
    fields: [
      { key: 'appid', type: 'text', secret: false },
      { key: 'secret', type: 'password', secret: true },
      { key: 'to', type: 'text', secret: false },
    ],
  },
]

const { t } = useI18n()
const { overview, refresh } = useAgentOverview()

const loaded = ref(false)
const events = reactive<AgentNotifyEvents>({ ask: false, done: false, error: false })
const channels = ref<AgentNotifyChannel[]>([])

const load = async () => {
  try {
    const { data } = await getAgentNotify()
    events.ask = !!data.events?.ask
    events.done = !!data.events?.done
    events.error = !!data.events?.error
    channels.value = data.channels || []
    loaded.value = true
  } catch {
    /* shown */
  }
}

// —— 什么时候通知：点了就保存，失败还原 ——
const setEvent = async (key: EventKey, on: boolean) => {
  events[key] = on
  try {
    await updateAgentNotifyEvents({ ...events })
  } catch {
    events[key] = !on
  }
}

// —— 通知发到哪 ——
const subOf = (kind: ChannelKind, fields: Record<string, string>) => {
  if (kind === 'telegram') return t('agent.notify.sub.chat', { chat: fields.chat || '' })
  if (kind === 'qq') return t('agent.notify.sub.qq', { appid: fields.appid || '', to: fields.to || '' })
  return t('agent.notify.sub.webhook', { tail: (fields.url || '').slice(-4) })
}
const rows = computed(() =>
  CHANNELS.map((def) => {
    const ch = channels.value.find((c) => c.kind === def.kind)
    const configured = !!ch?.configured
    const on = configured && !!ch?.enabled
    const fields = ch?.fields || {}
    return {
      kind: def.kind,
      def,
      name: t(`agent.notify.channels.${def.kind}.name`),
      configured,
      on,
      fields,
      statusText: t(!configured ? 'agent.notify.status.none' : on ? 'agent.notify.status.on' : 'agent.notify.status.off'),
      sub: configured ? subOf(def.kind, fields) : t(`agent.notify.channels.${def.kind}.desc`),
    }
  }),
)
type Row = (typeof rows.value)[number]

// 数量来自概览（后端算好的），改了通知方式后跟着概览刷新
const subtitle = computed(() => {
  const o = overview.value
  return o ? t('agent.notify.subtitle', { total: o.notifyChannels, on: o.notifyEnabled }) : ''
})

const setEnabled = async (kind: ChannelKind, on: boolean) => {
  const ch = channels.value.find((c) => c.kind === kind)
  if (!ch) return
  ch.enabled = on
  try {
    await setAgentNotifyChannelEnabled(kind, on)
    void refresh()
  } catch {
    ch.enabled = !on
  }
}

const testing = ref(new Set<ChannelKind>())
const setTesting = (kind: ChannelKind, on: boolean) => {
  const next = new Set(testing.value)
  if (on) next.add(kind)
  else next.delete(kind)
  testing.value = next
}
const sendTest = async (c: Row) => {
  if (testing.value.has(c.kind)) return
  setTesting(c.kind, true)
  try {
    await testAgentNotifyChannel(c.kind)
    feedback.success(t('agent.notify.tested', { name: c.name }))
  } catch {
    /* shown */
  } finally {
    setTesting(c.kind, false)
  }
}

// —— 配置弹窗 ——
const dlg = reactive({
  open: false,
  kind: 'discord' as ChannelKind,
  values: {} as Record<string, string>,
  testing: false,
  saving: false,
})
const dlgRow = computed(() => rows.value.find((c) => c.kind === dlg.kind) ?? (rows.value[0] as Row))
const dlgFields = computed(() => {
  const row = dlgRow.value
  return row.def.fields.map((f) => {
    // 已配置的密钥：输入框留空，提示里带脱敏值（后端只给后四位），留空就沿用已保存的
    const tail = row.configured && f.secret ? row.fields[f.key] || '' : ''
    const masked = tail ? `••••${tail}` : ''
    return {
      ...f,
      label: t(`agent.notify.channels.${row.kind}.${f.key}`),
      placeholder: masked
        ? t('agent.notify.keepSaved', { masked })
        : f.example || t(`agent.notify.channels.${row.kind}.${f.key}Ph`),
      kept: !!masked,
    }
  })
})
const openConfig = (c: Row) => {
  dlg.kind = c.kind
  dlg.values = Object.fromEntries(c.def.fields.map((f) => [f.key, f.secret ? '' : c.fields[f.key] || '']))
  dlg.testing = false
  dlg.saving = false
  dlg.open = true
}
const draftValues = () => Object.fromEntries(dlgFields.value.map((f) => [f.key, (dlg.values[f.key] || '').trim()]))
const draftReady = () => {
  const ok = dlgFields.value.every((f) => f.optional || f.kept || (dlg.values[f.key] || '').trim())
  if (!ok) feedback.warning(t('agent.notify.missing'))
  return ok
}
const testDraft = async () => {
  if (dlg.testing || !draftReady()) return
  dlg.testing = true
  try {
    await testAgentNotifyChannel(dlg.kind, draftValues())
    feedback.success(t('agent.notify.tested', { name: dlgRow.value.name }))
  } catch {
    /* shown */
  } finally {
    dlg.testing = false
  }
}
const saveDraft = async () => {
  if (dlg.saving || !draftReady()) return
  dlg.saving = true
  try {
    await saveAgentNotifyChannel(dlg.kind, draftValues())
    feedback.success(t('agent.notify.saved', { name: dlgRow.value.name }))
    dlg.open = false
    void refresh()
    await load()
  } catch {
    /* shown */
  } finally {
    dlg.saving = false
  }
}

onMounted(load)
// 标签页在 KeepAlive 里：切回来时静默刷新一次
onActivated(() => {
  if (loaded.value) void load()
})
</script>

<style lang="scss">
.ag-nt-sw {
  margin-right: 8px;
}
.ag-btn.ag-nt-test {
  margin-right: 4px;
}
.ag-nt-acts {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding-top: 2px;
}
.ag-nt-acts__sp {
  flex: 1;
}
@media (width < 992px) {
  // 时机开关：留在文字右边
  .ag-rw.ag-nt-ev {
    flex-wrap: nowrap;
  }
  .ag-nt-ev .ag-rm {
    flex: 1 1 0;
  }
  .ag-nt-ev .ag-rc {
    flex: none;
  }
  .ag-nt-sw {
    margin-right: auto;
  }
  .ag-btn.ag-nt-test {
    margin-right: 0;
  }
}
</style>
