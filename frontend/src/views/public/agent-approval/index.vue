<!-- 公开批准页（无需登录）：通知消息里的批准链接打开这里，凭链接里的一次性令牌查看要批准的操作并回答。
     链接里的 ?option= 只预先选中那个选项，要点一下才提交（聊天软件预览链接时不会替人做决定）。 -->
<template>
  <div class="ag agp">
    <div class="agp__card">
      <div class="agp__brand">
        <img :src="APP_CONFIG.logoSrc" alt="" class="agp__logo" />
        <span class="agp__name">{{ APP_CONFIG.name }}</span>
        <span class="ag-tag">{{ t('agent.approval.title') }}</span>
      </div>

      <div v-if="!data" class="agp__note">{{ failedLoad || t('agent.approval.loading') }}</div>

      <template v-else-if="data.state === 'expired'">
        <div class="agp__note">{{ t('agent.approval.expired') }}</div>
      </template>

      <template v-else>
        <dl class="agp__meta">
          <template v-if="data.sessionTitle">
            <dt>{{ t('agent.approval.session') }}</dt>
            <dd>{{ data.sessionTitle }}</dd>
          </template>
          <template v-if="data.runtimeName">
            <dt>{{ t('agent.approval.agent') }}</dt>
            <dd>{{ data.runtimeName }}</dd>
          </template>
          <template v-if="data.workspaceName">
            <dt>{{ t('agent.approval.workspace') }}</dt>
            <dd>{{ data.workspaceName }}</dd>
          </template>
        </dl>

        <div class="agp__ask">
          <div class="agp__ask-head"><UIcon name="i-lucide-shield" />{{ t('agent.code.perm.need') }}</div>
          <div v-if="data.title" class="agp__ask-title">{{ data.title }}</div>
          <div v-if="data.detail" class="agp__cmd ag-mono"><span class="ag-muted">$</span> {{ data.detail }}</div>
        </div>

        <div v-if="data.state === 'resolved' || chosen" class="agp__done">
          <UIcon name="i-lucide-check" />
          <span>{{ chosen ? t('agent.approval.done', { option: chosen }) : data.outcome ? t('agent.approval.resolved', { outcome: data.outcome }) : t('agent.approval.resolvedNone') }}</span>
        </div>
        <div v-else class="agp__acts">
          <button
            v-for="o in data.options"
            :key="o.id"
            type="button"
            class="ag-btn"
            :class="[isRejectOption(o) ? 'ag-btn--o' : o.id === preset || (!preset && o === data.options[0]) ? 'ag-btn--p' : '', { 'is-disabled': busy }]"
            :disabled="busy"
            @click="choose(o)"
          >
            {{ permOptionLabel(o, data.options) }}
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getPublicAgentApproval, resolvePublicAgentApproval } from '@/api/agent'
import { APP_CONFIG } from '@/config/app.config'
import type { AgentChoice, PublicAgentApprovalResponse } from '@/types/v1/agent'
import { isRejectOption, permOptionLabel } from '@/views/agent/utils/labels'

const { t } = useI18n()
const route = useRoute()
const token = computed(() => String(route.params.token || ''))
const preset = computed(() => String(route.query.option || ''))

const data = ref<PublicAgentApprovalResponse | null>(null)
const failedLoad = ref('')
const busy = ref(false)
const chosen = ref('')

const load = async () => {
  try {
    const { data: res } = await getPublicAgentApproval(token.value)
    data.value = res
  } catch {
    failedLoad.value = t('agent.approval.expired')
  }
}

const choose = async (o: AgentChoice) => {
  if (busy.value) return
  busy.value = true
  try {
    await resolvePublicAgentApproval(token.value, o.id)
    chosen.value = permOptionLabel(o, data.value?.options ?? [])
  } catch {
    feedback.error(t('agent.approval.failed'))
    await load()
  } finally {
    busy.value = false
  }
}

onMounted(() => void load())
</script>

<style lang="scss">
@use '@/views/agent/styles/agent.scss';

.agp {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 16px;
  background: var(--m-bg);
}
.agp__card {
  width: 480px;
  max-width: 100%;
  padding: 18px 20px 20px;
  border-radius: 16px;
  background: var(--m-s);
  border: 1px solid var(--m-line);
  box-shadow: 0 30px 70px -40px rgba(20, 10, 30, 0.45);
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.agp__brand {
  display: flex;
  align-items: center;
  gap: 8px;
}
.agp__logo {
  width: 26px;
  height: 26px;
  border-radius: 8px;
}
.agp__name {
  flex: 1;
  font-size: 14px;
  font-weight: 600;
}
.agp__note {
  font-size: 13px;
  line-height: 1.6;
  color: var(--m-ink2);
}
.agp__meta {
  margin: 0;
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 6px 12px;
  font-size: 13px;
}
.agp__meta dt {
  color: var(--m-ink2);
}
.agp__meta dd {
  margin: 0;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.agp__ask {
  border: 1.5px solid var(--m-warn);
  border-radius: 12px;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.agp__ask-head {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--m-warn);
  font-size: 13px;
  font-weight: 600;
}
.agp__ask-head svg,
.agp__ask-head .iconify {
  width: 17px;
  height: 17px;
}
.agp__ask-title {
  font-size: 13px;
}
.agp__cmd {
  padding: 9px 11px;
  border-radius: 10px;
  background: var(--m-s2);
  font-size: 12.5px;
  white-space: pre-wrap;
  word-break: break-all;
}
.agp__acts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.agp__done {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 11px;
  border-radius: 10px;
  background: var(--m-okSoft);
  color: var(--m-ok);
  font-size: 13px;
  font-weight: 500;
}
.agp__done svg,
.agp__done .iconify {
  width: 16px;
  height: 16px;
  flex: none;
}
</style>
