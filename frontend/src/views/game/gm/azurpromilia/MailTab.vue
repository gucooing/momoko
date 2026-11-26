<!-- 邮件补偿：发送记录（搜索、一封一行）；「写邮件」在弹窗里编写（全服可定时与限定领取截止，或指定玩家），可先试运行；
     点一封在同一弹窗里只读查看，全服邮件可撤回。 -->
<template>
  <div class="azm">
    <ListToolbar v-model:keyword="q" :placeholder="t('azurGm.mail.search')" searchable @search="pages.reset()">
      <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('azurGm.common.refresh')" :box="32" @click="pages.load()" />
      <UButton icon="i-heroicons-pencil-square" @click="compose">{{ t('azurGm.mail.compose') }}</UButton>
    </ListToolbar>

    <p v-if="pages.error.value" class="azm__error">{{ t('azurGm.common.loadFailed', { msg: pages.error.value }) }}</p>
    <DataTable :columns="columns" :rows="rows" row-key="id" :loading="pages.loading.value" :empty-text="t('azurGm.common.empty')" @row-click="view">
      <template #cell-state="{ row }">
        <StatusPill :variant="stateVariant(String(row.state))" :label="t(`azurGm.mail.state.${row.state}`)" />
      </template>
    </DataTable>
    <CursorPager
      :page="pages.page.value"
      :total="pages.total.value"
      :has-prev="pages.hasPrev.value"
      :has-next="pages.hasNext.value"
      :loading="pages.loading.value"
      @prev="pages.prev()"
      @next="pages.next()"
    />

    <FormDialog v-model="editing" :title="t('azurGm.mail.compose')" :width="680" :show-footer="false">
      <div class="azm__form">
        <div class="azm__seg" role="radiogroup">
          <button v-for="k in ['all', 'players']" :key="k" type="button" :class="{ 'is-on': form.target === k }" @click="form.target = k as Target">
            {{ t(`azurGm.mail.target.${k}`) }}
          </button>
        </div>
        <label v-if="form.target === 'players'" class="azm__field">
          <span>{{ t('azurGm.mail.pids') }}</span>
          <textarea v-model="form.pids" rows="2" />
        </label>
        <div v-else class="azm__row">
          <label class="azm__field">
            <span>{{ t('azurGm.mail.schedule') }}</span>
            <input v-model="form.startAt" type="datetime-local" />
          </label>
          <label class="azm__field">
            <span>{{ t('azurGm.mail.endAt') }}</span>
            <input v-model="form.endAt" type="datetime-local" />
          </label>
        </div>
        <label class="azm__field">
          <span>{{ t('azurGm.form.title') }}</span>
          <input v-model="form.title" maxlength="64" />
        </label>
        <label class="azm__field">
          <span>{{ t('azurGm.form.sender') }}</span>
          <input v-model="form.sender" maxlength="32" />
        </label>
        <label class="azm__field">
          <span>{{ t('azurGm.form.content') }}</span>
          <textarea v-model="form.content" rows="5" />
        </label>
        <div class="azm__field">
          <span>{{ t('azurGm.form.rewards') }}</span>
          <RewardEditor v-model="form.rewards" />
        </div>
        <label class="azm__field azm__field--short">
          <span>{{ t('azurGm.form.effectDays') }}</span>
          <input v-model.number="form.days" type="number" min="0" max="3650" />
        </label>
        <p v-if="formError" class="azm__error">{{ formError }}</p>
        <p v-if="preview" class="azm__hint">{{ preview }}</p>
        <div class="azm__actions">
          <UButton color="neutral" variant="soft" :loading="sending === 'dry'" :disabled="!!sending" @click="send(true)">{{ t('azurGm.common.dryRun') }}</UButton>
          <UButton :loading="sending === 'run'" :disabled="!!sending" @click="send(false)">{{ t('azurGm.mail.send') }}</UButton>
        </div>
      </div>
    </FormDialog>

    <FormDialog v-model="viewing" :title="t('azurGm.mail.detail')" :width="600" :show-footer="false">
      <div v-if="current" class="azm__form">
        <DescriptionList :items="detail" :columns="1" />
        <p class="azm__body">{{ current.content }}</p>
        <template v-if="current.target === 'all' && current.state !== 'revoked'">
          <p class="azm__hint">{{ t('azurGm.mail.revokeHint') }}</p>
          <div class="azm__actions">
            <UButton color="error" variant="soft" :loading="revoking" @click="revoke">{{ t('azurGm.mail.revoke') }}</UButton>
          </div>
        </template>
        <p v-if="formError" class="azm__error">{{ formError }}</p>
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import * as api from '@/api/azurpromilia'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { GmMail, Reward } from '@/types/v1/azurpromilia/types'
import { feedback } from '@/utils/feedback'
import { useGmContext } from '../context'
import CursorPager from './CursorPager.vue'
import RewardEditor from './RewardEditor.vue'
import { errorMessage, fmtTime, fromInput, parsePids, rewardsText, useCursorPages } from './shared'

type Target = 'all' | 'players'

const { t } = useI18n()
const ctx = useGmContext()
const q = ref('')

const pages = useCursorPages<GmMail>((cursor) => api.listMails({ serverId: ctx.serverId.value, q: q.value.trim(), cursor, limit: 20 }))

const columns = computed<DataTableColumn[]>(() => [
  { key: 'title', title: t('azurGm.mail.cols.title'), minWidth: 160, ellipsis: true },
  { key: 'targetText', title: t('azurGm.mail.cols.target'), width: 120 },
  { key: 'rewardsText', title: t('azurGm.mail.cols.rewards'), minWidth: 180, ellipsis: true },
  { key: 'state', title: t('azurGm.mail.cols.state'), width: 90 },
  { key: 'time', title: t('azurGm.mail.cols.time'), width: 150 },
  { key: 'result', title: t('azurGm.mail.cols.result'), width: 150 },
  { key: 'operator', title: t('azurGm.mail.cols.operator'), width: 120, ellipsis: true },
])

const targetText = (m: GmMail) => (m.target === 'all' ? t('azurGm.mail.target.all') : t('azurGm.mail.targetPlayers', { n: m.pids.length }))

const rows = computed(() =>
  pages.items.value.map((m) => ({
    ...m,
    targetText: targetText(m),
    rewardsText: rewardsText(m.rewards),
    time: fmtTime(m.target === 'all' ? m.startAt : m.createdAt),
    result: m.target === 'players' ? t('azurGm.mail.delivered', { ok: m.delivered, failed: m.failed }) : '—',
  })),
)

const stateVariant = (s: string) => (s === 'revoked' ? 'neutral' : s === 'scheduled' ? 'warning' : 'success')

const editing = ref(false)
const form = reactive({ target: 'all' as Target, pids: '', startAt: '', endAt: '', title: '', sender: '', content: '', rewards: [] as Reward[], days: 30 })
const formError = ref('')
const preview = ref('')
const sending = ref<'' | 'dry' | 'run'>('')

const compose = () => {
  Object.assign(form, { target: 'all', pids: '', startAt: '', endAt: '', title: '', sender: '', content: '', rewards: [], days: 30 })
  formError.value = ''
  preview.value = ''
  editing.value = true
}

const send = async (dryRun: boolean) => {
  formError.value = ''
  preview.value = ''
  if (!form.title.trim()) {
    formError.value = t('azurGm.form.required', { field: t('azurGm.form.title') })
    return
  }
  const pids = form.target === 'players' ? parsePids(form.pids) : []
  if (form.target === 'players' && !pids.length) {
    formError.value = t('azurGm.form.required', { field: t('azurGm.mail.pids') })
    return
  }
  sending.value = dryRun ? 'dry' : 'run'
  try {
    const res = await api.sendMail({
      serverId: ctx.serverId.value,
      target: form.target,
      pids,
      mail: {
        sender: form.sender.trim(),
        title: form.title.trim(),
        content: form.content,
        rewards: form.rewards,
        effect: Math.max(0, Math.floor(form.days || 0)) * 86400,
      },
      startAt: form.target === 'all' ? fromInput(form.startAt) : 0,
      endAt: form.target === 'all' ? fromInput(form.endAt) : 0,
      dryRun,
    })
    const r = res.result
    if (dryRun) {
      preview.value = t('azurGm.mail.targets', { n: r?.targets ?? 0 })
      return
    }
    const failed = r?.failures || []
    if (failed.length) formError.value = `${t('azurGm.mail.failures')}：${failed.map((f) => `${f.pid} ${f.message || f.code}`).join('；')}`
    else editing.value = false
    feedback.success(t('azurGm.mail.send'))
    void pages.reset()
  } catch (e) {
    formError.value = errorMessage(e)
  } finally {
    sending.value = ''
  }
}

const viewing = ref(false)
const current = ref<GmMail>()
const revoking = ref(false)

const view = (row: Record<string, unknown>) => {
  current.value = pages.items.value.find((m) => m.id === row.id)
  formError.value = ''
  viewing.value = true
}

const detail = computed(() => {
  const m = current.value
  if (!m) return []
  const items = [
    { label: t('azurGm.form.title'), value: m.title },
    { label: t('azurGm.form.sender'), value: m.sender || '—' },
    { label: t('azurGm.mail.cols.target'), value: m.target === 'all' ? t('azurGm.mail.target.all') : m.pids.join(', ') },
    { label: t('azurGm.form.rewards'), value: rewardsText(m.rewards) },
    { label: t('azurGm.mail.cols.state'), value: t(`azurGm.mail.state.${m.state}`) },
    { label: t('azurGm.mail.cols.operator'), value: m.operator || '—' },
  ]
  if (m.target === 'all') {
    items.push({ label: t('azurGm.mail.schedule'), value: fmtTime(m.startAt) }, { label: t('azurGm.mail.endAt'), value: fmtTime(m.endAt) })
  } else {
    items.push({ label: t('azurGm.mail.cols.result'), value: t('azurGm.mail.delivered', { ok: m.delivered, failed: m.failed }) })
  }
  return items
})

const revoke = async () => {
  if (!current.value) return
  revoking.value = true
  formError.value = ''
  try {
    const res = await api.revokeMail({ serverId: ctx.serverId.value, id: current.value.id, dryRun: false })
    if (res.mail) current.value = res.mail
    feedback.success(t('azurGm.mail.revoke'))
    void pages.load()
  } catch (e) {
    formError.value = errorMessage(e)
  } finally {
    revoking.value = false
  }
}

void pages.reset()
onActivated(() => void pages.load())
</script>

<style scoped lang="scss">
.azm {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.azm__form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.azm__row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.azm__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  > span {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }
  input,
  textarea {
    padding: 6px 10px;
    border: 1px solid var(--el-border-color-light);
    border-radius: var(--app-radius-sm);
    background: var(--el-bg-color);
    color: var(--el-text-color-primary);
    font: inherit;
  }
  input {
    height: 32px;
  }
}
.azm__field--short {
  max-width: 320px;
}
.azm__seg {
  display: inline-flex;
  gap: 4px;
  padding: 3px;
  border-radius: var(--app-radius-pill);
  background: var(--el-fill-color-light);
  align-self: flex-start;
  button {
    padding: 4px 14px;
    border: 0;
    border-radius: var(--app-radius-pill);
    background: transparent;
    color: var(--el-text-color-secondary);
    font: inherit;
    cursor: pointer;
    &.is-on {
      background: var(--el-bg-color);
      color: var(--el-text-color-primary);
    }
  }
}
.azm__actions {
  display: flex;
  gap: 8px;
}
.azm__error {
  margin: 0;
  color: var(--m-bad);
  font-size: 13px;
}
.azm__hint {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.azm__body {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
}
</style>
