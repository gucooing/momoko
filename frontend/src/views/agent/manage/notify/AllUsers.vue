<!-- Agent 管理 · 通知：所有人自己设置的通知方式，按人分组（只列设置过的人）。每个人一行写他开着哪几个时机，
     下面一种方式一行（状态、说明），管理员能停用 / 启用或移除，改不了别人的配置（密钥只有脱敏的后几位）。
     自己的通知在 Agent 设置的通知里设置（index.vue）。 -->
<template>
  <div class="ag ag-page">
    <AgManageHeader>{{ subtitle }}</AgManageHeader>

    <div class="ag-body">
      <template v-for="(u, i) in people" :key="u.userId">
        <div class="ag-sh-row" :class="{ 'ag-gap': i > 0 }">
          <h2 class="ag-sh">{{ u.name }}</h2>
          <span class="ag-fh">{{ u.events }}</span>
        </div>
        <div class="ag-grp">
          <div v-for="c in u.channels" :key="c.kind" class="ag-rw">
            <div class="ag-rm">
              <div class="ag-rt">
                <span>{{ c.name }}</span>
                <span class="ag-pill" :class="{ 'ag-pill--ok': c.enabled }">
                  {{ t(c.enabled ? 'agent.notify.status.on' : 'agent.notify.status.off') }}
                </span>
              </div>
              <div class="ag-rs">{{ c.sub }}</div>
            </div>
            <div class="ag-rc">
              <AgSwitch
                class="ag-nta-sw"
                :model-value="c.enabled"
                :label="t('agent.notify.enableOf', { user: u.name, name: c.name })"
                @update:model-value="setEnabled(u.userId, c.kind, $event)"
              />
              <button
                type="button"
                class="ag-ib ag-ib--sm"
                :aria-label="t('agent.notify.all.removeOf', { user: u.name, name: c.name })"
                :title="t('agent.notify.remove')"
                @click="askRemove(u, c)"
              >
                <UIcon name="i-lucide-trash-2" />
              </button>
            </div>
          </div>
        </div>
      </template>
      <div v-if="!loaded" class="ag-grp"><div class="ag-empty">{{ t('agent.common.loading') }}</div></div>
      <div v-else-if="!people.length" class="ag-grp"><div class="ag-empty">{{ t('agent.notify.all.empty') }}</div></div>
    </div>

    <AgConfirm
      v-model="removeOpen"
      danger
      :title="t('agent.notify.all.removeTitle', { user: removeTarget?.user || '', name: removeTarget?.name || '' })"
      :message="t('agent.notify.all.removeMsg')"
      :confirm-text="t('agent.notify.remove')"
      :on-confirm="doRemove"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { deleteAgentNotifyChannel, listAgentNotify, setAgentNotifyChannelEnabled } from '@/api/agent'
import type { AgentNotifyOwner, AgentUser } from '@/types/v1/agent'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import AgManageHeader from '@/views/agent/components/AgManageHeader.vue'
import AgSwitch from '@/views/agent/components/AgSwitch.vue'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'
import { CHANNEL_KINDS, channelSub } from './channels'

defineOptions({ name: 'AgentAdminNotifyView' })

const { t } = useI18n()
const { overview, refresh } = useAgentOverview()

const items = ref<AgentNotifyOwner[]>([])
const users = ref<AgentUser[]>([])
const loaded = ref(false)
const load = async () => {
  try {
    const { data } = await listAgentNotify()
    items.value = data.items || []
    users.value = data.users || []
  } catch {
    /* shown */
  } finally {
    loaded.value = true
  }
}

// 数量来自概览（all 时是所有人设置过的通知方式、其中启用着的）
const subtitle = computed(() => {
  const o = overview.value
  if (!o || !loaded.value) return ''
  return t('agent.notify.all.subtitle', { people: items.value.length, total: o.notifyChannels, on: o.notifyEnabled })
})

const userName = (id: string) => {
  const u = users.value.find((x) => x.id === id)
  return u ? u.name || u.username : t('agent.admin.goneUser')
}
const people = computed(() =>
  items.value.map((it) => {
    const on = (['ask', 'done', 'error'] as const).filter((k) => it.events?.[k]).map((k) => t(`agent.notify.events.${k}.title`))
    return {
      userId: it.userId,
      name: userName(it.userId),
      events: on.length ? t('agent.notify.all.events', { events: on.join(t('agent.share.sep')) }) : t('agent.notify.all.noEvents'),
      channels: CHANNEL_KINDS.flatMap((kind) => {
        const ch = (it.channels || []).find((c) => c.kind === kind)
        if (!ch) return []
        return [{ kind, enabled: ch.enabled, name: t(`agent.notify.channels.${kind}.name`), sub: channelSub(t, kind, ch.fields || {}) }]
      }),
    }
  }),
)

const setEnabled = async (userId: string, kind: string, on: boolean) => {
  const ch = items.value.find((it) => it.userId === userId)?.channels?.find((c) => c.kind === kind)
  if (!ch) return
  ch.enabled = on
  try {
    await setAgentNotifyChannelEnabled(kind, on, userId)
    void refresh()
  } catch {
    ch.enabled = !on
  }
}

// —— 移除 ——
const removeOpen = ref(false)
const removeTarget = ref<{ userId: string; user: string; kind: string; name: string } | null>(null)
const askRemove = (u: { userId: string; name: string }, c: { kind: string; name: string }) => {
  removeTarget.value = { userId: u.userId, user: u.name, kind: c.kind, name: c.name }
  removeOpen.value = true
}
const doRemove = async () => {
  const r = removeTarget.value
  if (!r) return
  await deleteAgentNotifyChannel(r.kind, r.userId)
  feedback.success(t('agent.notify.all.removed', { user: r.user, name: r.name }))
  void refresh()
  await load()
}

onMounted(load)
// 标签页在 KeepAlive 里：切回来时静默刷新一次
onActivated(() => {
  if (loaded.value) void load()
})
</script>

<style lang="scss">
.ag-nta-sw {
  margin-right: 8px;
}
</style>
