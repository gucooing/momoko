<!-- Agent 设置 · 模型供应商：上面是大家共用的官方账户（Claude Code、Codex 在服务器上登录的账户，只能看，
     检查和编辑模型在 Agent 管理的「官方账户」里），下面是自己的第三方接口（添加、测试连接、查余额、编辑、删除），
     后面跟着管理员共享给自己的（只能用，看不到密钥）。
     在 Agent 管理页里（useAgentAll）分两组：共享的（管理员在这里加、改、删，选共享给所有人还是指定的人），
     和各人自己的（行上带主人，只能删除）。 -->
<template>
  <div class="ag ag-page">
    <AgManageHeader>
      {{ subtitle }}
      <template #actions>
        <button type="button" class="ag-btn ag-btn--p" @click="openForm(null)">
          <UIcon name="i-lucide-plus" />{{ all ? t('agent.share.addShared') : t('agent.providers.addApi') }}
        </button>
      </template>
    </AgManageHeader>

    <div class="ag-body">
      <template v-if="!all">
        <h2 class="ag-sh">{{ t('agent.providers.serverLogin') }}</h2>
        <div class="ag-grp">
          <div v-for="r in runtimes" :key="r.id" class="ag-rw">
            <div class="ag-rm">
              <div class="ag-rt">
                <span>{{ r.name }}</span>
                <AgAgentTag :kind="agentKindOf(r.kind)" :name="r.name" />
              </div>
              <div class="ag-rs">{{ loginSub(t, r, false) }}</div>
            </div>
            <div class="ag-rc">
              <span class="ag-pill ag-pv-pill" :class="loginPill(t, r).cls">{{ loginPill(t, r).text }}</span>
            </div>
          </div>
          <div v-if="!runtimes.length" class="ag-empty">{{ loaded ? t('agent.providers.view.none') : t('agent.common.loading') }}</div>
        </div>
        <p class="ag-fh ag-pv-hint">
          {{ t('agent.providers.view.hint') }}
          <RouterLink v-if="canManageLogins" to="/agent/admin?tab=accounts">{{ t('agent.providers.view.goManage') }}</RouterLink>
        </p>
      </template>

      <template v-for="g in groups" :key="g.key">
        <h2 class="ag-sh" :class="{ 'ag-gap': !all || g.key === 'personal' }">{{ g.title }}</h2>
        <div class="ag-grp">
          <div v-for="r in g.rows" :key="r.p.id" class="ag-rw">
            <div class="ag-rm">
              <div class="ag-rt">
                <span>{{ r.p.name }}</span>
                <AgAgentTag
                  v-for="pr in vendorOf(r.p)?.protocols || []"
                  :key="pr"
                  :kind="protocolAgent(pr)"
                  :name="t(`agent.family.${protocolAgent(pr)}`)"
                />
                <span class="ag-tag">{{ vendorName(r.p.vendor) }}</span>
                <span v-if="r.shareTag" class="ag-tag ag-tag--p" :title="r.scope">{{ r.shareTag }}</span>
              </div>
              <div class="ag-rs ag-pv-sub">
                <template v-if="all && !r.shared">{{ r.owner }} · </template>
                <template v-if="all && r.shared && r.scope">{{ r.scope }} · </template>
                <span class="ag-mono">{{ r.p.baseUrl || vendorOf(r.p)?.baseUrl }}</span> · {{ modelsSummary(r.p) }}
                <template v-if="r.p.apiKeyTail"> · <span class="ag-mono">sk-••••{{ r.p.apiKeyTail }}</span></template>
              </div>
            </div>
            <div class="ag-rc">
              <span
                v-if="r.editable && balanceOf(r.p)"
                class="ag-tag ag-pv-bal"
                :class="{ 'ag-tag--warn': balanceOf(r.p)?.warn }"
                :title="balanceOf(r.p)?.title"
              >
                <UIcon name="i-lucide-wallet" />{{ balanceOf(r.p)?.text }}
              </span>
              <span class="ag-pill ag-pv-pill" :class="apiPill(r.p).cls">{{ apiPill(r.p).text }}</span>
              <template v-if="r.editable">
                <button
                  type="button"
                  class="ag-ib ag-ib--sm"
                  :aria-label="`${t('agent.common.testConnection')} ${r.p.name}`"
                  :title="t('agent.common.testConnection')"
                  :disabled="testing.has(r.p.id)"
                  @click="runTest(r.p)"
                >
                  <UIcon
                    :name="testing.has(r.p.id) ? 'i-lucide-loader-circle' : 'i-lucide-zap'"
                    :class="{ 'ag-spin': testing.has(r.p.id) }"
                  />
                </button>
                <button
                  type="button"
                  class="ag-ib ag-ib--sm"
                  :aria-label="`${t('agent.common.edit')} ${r.p.name}`"
                  :title="t('agent.common.edit')"
                  @click="openForm(r.p)"
                >
                  <UIcon name="i-lucide-pencil" />
                </button>
              </template>
              <span v-else-if="!all" class="ag-pv-lock" :title="t('agent.share.sharedReadonly')">
                <UIcon name="i-lucide-lock" />
              </span>
              <button
                v-if="r.editable || all"
                type="button"
                class="ag-ib ag-ib--sm"
                :aria-label="`${t('agent.common.delete')} ${r.p.name}`"
                :title="t('agent.common.delete')"
                @click="askDelete(r.p)"
              >
                <UIcon name="i-lucide-trash-2" />
              </button>
            </div>
          </div>
          <div v-if="!loaded" class="ag-empty">{{ t('agent.common.loading') }}</div>
          <div v-else-if="!g.rows.length" class="ag-empty">{{ g.empty }}</div>
        </div>
      </template>
    </div>

    <ProviderDialog v-model="formOpen" :provider="formTarget" :vendors="vendors" :shared="formShared" @saved="reload" />
    <AgConfirm
      v-model="deleteOpen"
      danger
      :title="t('agent.providers.deleteTitle', { name: deleteTarget?.name || '' })"
      :message="t('agent.providers.deleteApiMsg')"
      :confirm-text="t('agent.common.delete')"
      :on-confirm="doDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  AGENT_SHARED_OWNER,
  deleteAgentProvider,
  getAgentProviderBalance,
  listAgentProviders,
  listAgentRuntimes,
  testAgentProvider,
} from '@/api/agent'
import type { AgentProvider, AgentRuntime, AgentUser, AgentVendor, GetAgentProviderBalanceResponse } from '@/types/v1/agent'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import AgConfirm from '@/views/agent/components/AgConfirm.vue'
import AgManageHeader from '@/views/agent/components/AgManageHeader.vue'
import { useAgentAll } from '@/views/agent/composables/useAgentAll'
import { useAgentModels } from '@/views/agent/composables/useAgentModels'
import { useAgentOverview } from '@/views/agent/composables/useAgentOverview'
import { agentKindOf, protocolAgent } from '@/views/agent/utils/format'
import { vendorName } from '@/views/agent/utils/labels'
import { loginPill, loginSub } from '../accounts/login'
import ProviderDialog from './ProviderDialog.vue'

defineOptions({ name: 'AgentManageProvidersView' })

const { t, locale } = useI18n()
/** 在 Agent 管理页里：所有人的第三方接口 */
const all = useAgentAll()
const menuStore = useMenuStore()
const { refresh } = useAgentOverview()
const { refresh: refreshModels } = useAgentModels()

const providers = ref<AgentProvider[]>([])
const vendors = ref<AgentVendor[]>([])
const users = ref<AgentUser[]>([])
/** 官方账户（只在 Agent 设置里显示，只读） */
const runtimes = ref<AgentRuntime[]>([])
const loaded = ref(false)

// 能管官方账户的人给个去 Agent 管理的链接
const canManageLogins = computed(() => menuStore.hasMenuPath('/agent/admin') && menuStore.hasButtonPermission('agent:manage'))

const subtitle = computed(() => {
  if (!loaded.value) return ''
  const apis = providers.value.length
  if (all) return t('agent.providers.subtitleAll', { n: apis })
  // 供应商 = 登录着的官方账户（和「已登录」那个标记一样：就绪且登录着）+ 第三方接口
  const logins = runtimes.value.filter((r) => r.ready && r.loggedIn).length
  return t('agent.providers.subtitle', { n: logins + apis, logins, apis })
})

const load = async () => {
  try {
    const [ps, rts] = await Promise.all([listAgentProviders(all), all ? null : listAgentRuntimes()])
    providers.value = ps.data.providers || []
    vendors.value = ps.data.vendors || []
    users.value = ps.data.users || []
    runtimes.value = rts?.data.runtimes || []
    loadBalances()
  } catch {
    /* shown */
  } finally {
    loaded.value = true
  }
}
/** 改了数据之后：重新拉列表，刷新标签上的数量和各页面共用的模型清单 */
const reload = () => {
  void refresh()
  void refreshModels()
  return load()
}

// 供应商不绑哪一条 Agent：按它的类型给的协议给哪几系用（自己添加的 Agent 检测出是哪一系也能用）
const vendorOf = (p: AgentProvider) => vendors.value.find((v) => v.id === p.vendor)
const apiPill = (p: AgentProvider) => {
  if (p.status === 'ok') {
    const ms = Number(p.testLatencyMs) || 0
    return { cls: 'ag-pill--ok', text: ms > 0 ? t('agent.providers.status.okMs', { ms }) : t('agent.providers.status.ok') }
  }
  if (p.status === 'bad') return { cls: 'ag-pill--bad', text: p.statusText || t('agent.providers.status.failed') }
  return { cls: '', text: t('agent.common.notTested') }
}
const modelsSummary = (p: AgentProvider) => {
  const ids = (p.models || []).map((m) => m.id)
  if (!ids.length) return t('agent.providers.noModels')
  if (ids.length <= 2) return ids.join(t('agent.providers.listSep'))
  return t('agent.providers.modelsMore', { first: ids[0] ?? '', n: ids.length })
}

// —— 分组：自己的列表里共享的跟在后面、只能用；Agent 管理页里共享的在前 ——
const userName = (id: string) => {
  const u = users.value.find((x) => x.id === id)
  return u ? u.name || u.username : t('agent.admin.goneUser')
}
/** 共享的行上的标记：Agent 管理页里写共享给谁，自己的列表里写「管理员共享」 */
const shareLabels = (p: AgentProvider, shared: boolean) => {
  if (!shared) return { shareTag: '', scope: '' }
  if (!all) return { shareTag: t('agent.share.tagSharedToMe'), scope: t('agent.share.sharedReadonly') }
  const ids = p.shareUserIds || []
  if (p.share !== 'users') return { shareTag: t('agent.share.tagAll'), scope: '' }
  return { shareTag: t('agent.share.tagUsers', { n: ids.length }), scope: ids.map(userName).join(t('agent.share.sep')) }
}
const rows = computed(() =>
  providers.value.map((p) => {
    const shared = p.userId === AGENT_SHARED_OWNER
    // 能改的：自己列表里自己的；Agent 管理页里共享的。别人的在 Agent 管理页里只能删
    const editable = all ? shared : !shared
    return { p, shared, editable, owner: all && !shared ? userName(p.userId) : '', ...shareLabels(p, shared) }
  }),
)
const groups = computed(() => {
  if (!all) return [{ key: 'mine', title: t('agent.providers.apis'), rows: rows.value, empty: t('agent.providers.noApis') }]
  return [
    { key: 'shared', title: t('agent.share.sharedTitle'), rows: rows.value.filter((r) => r.shared), empty: t('agent.share.noShared') },
    { key: 'personal', title: t('agent.share.personalTitle'), rows: rows.value.filter((r) => !r.shared), empty: t('agent.providers.noApis') },
  ]
})

const testing = ref(new Set<string>())
const runTest = async (p: AgentProvider) => {
  if (testing.value.has(p.id)) return
  testing.value = new Set(testing.value).add(p.id)
  try {
    const { data } = await testAgentProvider({ id: p.id })
    const failed = (data.results || []).find((r) => !r.ok)
    if (!failed) feedback.success(t('agent.providers.testOk', { name: p.name }))
    else feedback.error(t('agent.providers.withMessage', { name: p.name, message: failed.error || t('agent.providers.status.failed') }))
  } catch {
    /* shown */
  } finally {
    const next = new Set(testing.value)
    next.delete(p.id)
    testing.value = next
    await load()
  }
}

// —— 余额：能改的、而且类型能查余额的才查（要用密钥），列表每次刷新时一起重新查 ——
const balances = ref<Record<string, GetAgentProviderBalanceResponse>>({})
const loadBalances = () => {
  for (const r of rows.value) {
    if (!r.editable || !vendorOf(r.p)?.balance) continue
    getAgentProviderBalance(r.p.id)
      .then(({ data }) => {
        balances.value = { ...balances.value, [r.p.id]: data }
      })
      .catch(() => {
        /* shown */
      })
  }
}
// 金额按当前语言和币种显示（¥110.00、US$5.00）；币种认不出时原样拼上
const money = (amount: string, currency: string) => {
  try {
    return new Intl.NumberFormat(locale.value, { style: 'currency', currency }).format(Number(amount))
  } catch {
    return `${amount} ${currency}`
  }
}
const balanceOf = (p: AgentProvider) => {
  const b = balances.value[p.id]
  if (!b) return null
  if (b.error) return { text: t('agent.providers.balance.failed'), title: b.error, warn: true }
  const items = b.balances || []
  const amount = items.map((x) => money(x.total, x.currency)).join(' · ') || money('0', 'CNY')
  const title = items
    .map((x) =>
      t('agent.providers.balance.detail', {
        total: money(x.total, x.currency),
        granted: money(x.granted, x.currency),
        toppedUp: money(x.toppedUp, x.currency),
      }),
    )
    .join('\n')
  return {
    text: t(b.available ? 'agent.providers.balance.text' : 'agent.providers.balance.low', { amount }),
    title,
    warn: !b.available,
  }
}

// —— 添加 / 编辑：Agent 管理页里加的是共享的 ——
const formOpen = ref(false)
const formTarget = ref<AgentProvider | null>(null)
const formShared = ref(false)
const openForm = (p: AgentProvider | null) => {
  formTarget.value = p
  formShared.value = p ? p.userId === AGENT_SHARED_OWNER : all
  formOpen.value = true
}

// —— 删除 ——
const deleteOpen = ref(false)
const deleteTarget = ref<AgentProvider | null>(null)
const askDelete = (p: AgentProvider) => {
  deleteTarget.value = p
  deleteOpen.value = true
}
const doDelete = async () => {
  const p = deleteTarget.value
  if (!p) return
  await deleteAgentProvider(p.id)
  feedback.success(t('agent.providers.deleted', { name: p.name }))
  await reload()
}

onMounted(load)
// 标签页在 KeepAlive 里：切回来时静默刷新一次（官方账户可能刚检查过），标签上的数量一起刷新
onActivated(() => {
  if (loaded.value) void reload()
})
</script>

<style lang="scss">
.ag-pv-pill {
  margin-right: 4px;
}
.ag-pv-bal {
  font-variant-numeric: tabular-nums;
}
.ag-pv-hint {
  margin: 6px 2px 0;
}
.ag-pv-lock {
  display: inline-flex;
  align-items: center;
  padding: 0 6px;
  color: var(--m-ink2);
}
// 供应商弹窗里：两列的格子、单选卡片里的「标题 + 说明 + 给哪几系 Agent 用」
.ag-pv-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
.ag-pv-opt__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.ag-pv-opt__t {
  font-size: 13px;
  font-weight: 600;
}
.ag-pv-opt__d {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--m-ink2);
}
@media (width < 992px) {
  .ag-pv-pill {
    margin-right: auto;
  }
  // 接口地址很长时在卡片里折行
  .ag-pv-sub {
    overflow-wrap: anywhere;
  }
}
</style>
