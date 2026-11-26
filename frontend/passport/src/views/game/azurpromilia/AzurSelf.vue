<!-- 蓝色星原 游戏空间 · 自助服务：管理员开放的操作（规则与使用情况）、我的申请、操作记录。操作只作用于我绑定的游戏账号。 -->
<template>
  <div class="azs">
    <p class="azs__banner pp-hint">{{ t('game.self.banner', { server: serverName, nickname }) }}</p>

    <div class="azs__main">
      <section class="azs__fns">
        <div v-for="f in items" :key="f.command" class="pp-card azs__fn">
          <span class="azs__fn-title">{{ opTitle(f.command, f.title) }}</span>
          <span v-if="f.playerNote" class="pp-hint">{{ f.playerNote }}</span>
          <div class="azs__rules">
            <span v-for="r in rules(f)" :key="r" class="pp-tag">{{ r }}</span>
          </div>
          <div class="azs__fn-foot">
            <span class="pp-hint">{{ stateText(f) }}</span>
            <PButton size="sm" :variant="usable(f) ? 'primary' : 'outline'" :disabled="!usable(f)" @click="open(f)">
              {{ f.needApproval ? t('game.self.apply') : t('game.self.run') }}
            </PButton>
          </div>
        </div>
        <PEmpty v-if="loaded && !items.length" class="pp-card" :title="error || t('game.self.empty')" :description="error ? '' : t('game.self.emptyDesc')" />
      </section>

      <aside class="azs__aside">
        <div v-for="box in boxes" :key="box.key" class="pp-card azs__box">
          <span class="azs__box-head">{{ t(`game.self.${box.key}`) }}</span>
          <ul v-if="box.list.items.length" class="azs__records">
            <li v-for="r in box.list.items" :key="r.id">
              <span class="azs__rec-title">{{ opTitle(r.command, r.commandTitle) }}</span>
              <span class="pp-hint">{{ statusText(r.status) }} · {{ fmtTime(r.createTime) }}</span>
              <span v-if="r.reviewNote" class="pp-hint">{{ r.reviewNote }}</span>
            </li>
          </ul>
          <span v-else-if="box.list.loaded" class="pp-hint">{{ box.list.error || t(`game.self.${box.key}Empty`) }}</span>
          <PPager
            v-if="box.list.loaded && box.list.total > PAGE"
            :page="box.list.page"
            :total="box.list.total"
            :page-size="PAGE"
            @update:page="(p) => loadBox(box.key, p)"
          />
        </div>
      </aside>
    </div>

    <PDialog :model-value="!!active" :title="active ? opTitle(active.command, active.title) : ''" :width="560" @update:model-value="(v) => !v && (active = null)">
      <form v-if="active" class="azs__form" @submit.prevent="submit">
        <template v-if="active.command === 'player.mail'">
          <PField :label="t('azur.form.title')"><input v-model="form.title" maxlength="64" /></PField>
          <PField :label="t('azur.form.content')"><textarea v-model="form.content" rows="3" /></PField>
          <PField :label="t('azur.form.rewards')"><AzurRewardPicker v-model="form.rewards" /></PField>
        </template>
        <template v-else-if="active.command === 'item.adjust'">
          <PField :label="t('azur.form.grant')"><AzurRewardPicker v-model="form.grant" /></PField>
          <PField :label="t('azur.form.take')"><AzurRewardPicker v-model="form.take" :types="takeTypes" /></PField>
        </template>
        <template v-else-if="active.command === 'player.ban'">
          <label class="azs__check"><input v-model="form.permanent" type="checkbox" /> {{ t('azur.form.permanent') }}</label>
          <PField v-if="!form.permanent" :label="t('azur.form.until')"><input v-model="form.until" type="datetime-local" /></PField>
        </template>
        <PField v-else-if="active.command === 'player.password'" :label="t('game.self.password')">
          <input v-model="form.password" type="password" autocomplete="new-password" maxlength="64" />
        </PField>
        <PField v-else-if="active.command === 'player.rename'" :label="t('game.self.newName')"><input v-model="form.name" maxlength="32" /></PField>

        <PField v-if="active.needApproval" :label="t('game.self.reason')" :hint="t('game.self.reasonHint')">
          <textarea v-model="form.reason" rows="2" maxlength="200" />
        </PField>
        <PField v-if="active.needTotp" :label="t('game.self.totpCode')" :hint="t('game.self.totpHint')">
          <input v-model="form.totp" inputmode="numeric" maxlength="6" autocomplete="one-time-code" />
        </PField>
        <p v-if="formError" class="pp-error">{{ formError }}</p>
        <PButton type="submit" variant="primary" block :loading="running">
          {{ active.needApproval ? t('game.self.submitApply') : t('game.self.submit') }}
        </PButton>
      </form>
    </PDialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import PButton from '@passport/components/PButton.vue'
import PDialog from '@passport/components/PDialog.vue'
import PEmpty from '@passport/components/PEmpty.vue'
import PField from '@passport/components/PField.vue'
import PPager from '@passport/components/PPager.vue'
import { listGameRecords, listGameRequests, listSelfService } from '@passport/api/passport'
import * as azur from '@passport/api/azurpromilia'
import { useUiStore } from '@passport/stores/ui'
import { errorText, fmtTime, toDate } from '@passport/utils/format'
import { REWARD_KINDS, type AzurReward } from '@/games/azurpromilia/names'
import type { PassportGameRecord, PassportSelfService } from '@/types/v1/passport'
import type { MySelfServiceResponse } from '@/types/v1/azurpromilia/passport'
import AzurRewardPicker from './AzurRewardPicker.vue'

const props = defineProps<{ serverId: string; serverName: string; nickname: string }>()
const { t, te } = useI18n()
const ui = useUiStore()
const PAGE = 10

// 扣除只能是货币与普通道具（游戏服务端规则）
const takeTypes = REWARD_KINDS.filter((k) => k.kind === 'currency' || k.kind === 'common_item').map((k) => k.type)

const opTitle = (command: string, title: string) => (command && te(`azur.ops.${command}`) ? t(`azur.ops.${command}`) : title || command)
const statusText = (s: string) => (te(`game.self.status.${s}`) ? t(`game.self.status.${s}`) : s)

// ---------- 已开放的操作 ----------
const items = ref<PassportSelfService[]>([])
const loaded = ref(false)
const error = ref('')

const loadItems = async () => {
  try {
    items.value = (await listSelfService(props.serverId)).items ?? []
    error.value = ''
  } catch (e) {
    error.value = errorText(e)
  } finally {
    loaded.value = true
  }
}

const usable = (f: PassportSelfService) => f.remaining !== 0 && !f.hasPending

const rules = (f: PassportSelfService) => {
  const out: string[] = []
  if (f.limit > 0) {
    if (!f.periodDays) out.push(t('game.self.perForever', { n: f.limit }))
    else if (f.periodDays === 1) out.push(t('game.self.perDay', { n: f.limit }))
    else out.push(t('game.self.perDays', { d: f.periodDays, n: f.limit }))
  }
  if (f.needApproval) out.push(t('game.self.needApproval'))
  if (f.needTotp) out.push(t('game.self.needTotp'))
  if (!out.length) out.push(t('game.self.unlimited'))
  return out
}

const stateText = (f: PassportSelfService) => {
  if (f.hasPending) return t('game.self.pending')
  const resets = toDate(f.resetsAt)
  if (f.remaining === 0) return resets ? t('game.self.usedUpUntil', { time: fmtTime(resets) }) : t('game.self.usedUp')
  if (f.remaining > 0) return f.periodDays === 1 ? t('game.self.leftToday', { n: f.remaining }) : t('game.self.left', { n: f.remaining })
  if (f.lastUsedAt) return t('game.self.last', { time: fmtTime(f.lastUsedAt), status: statusText(f.lastStatus) })
  return t('game.self.never')
}

// ---------- 我的申请 / 操作记录 ----------
type Box = { items: PassportGameRecord[]; total: number; page: number; loaded: boolean; error: string }
const newBox = (): Box => ({ items: [], total: 0, page: 1, loaded: false, error: '' })
const requests = reactive(newBox())
const records = reactive(newBox())
const boxes = [
  { key: 'requests' as const, list: requests },
  { key: 'records' as const, list: records },
]

const loadBox = async (key: 'requests' | 'records', page = 1) => {
  const box = key === 'requests' ? requests : records
  try {
    const res = await (key === 'requests' ? listGameRequests : listGameRecords)(props.serverId, page, PAGE)
    box.items = res.items
    box.total = res.total
    box.page = page
    box.error = ''
  } catch (e) {
    box.error = errorText(e)
  } finally {
    box.loaded = true
  }
}

// ---------- 执行 ----------
const active = ref<PassportSelfService | null>(null)
const running = ref(false)
const formError = ref('')
const form = reactive({
  title: '',
  content: '',
  rewards: [] as AzurReward[],
  grant: [] as AzurReward[],
  take: [] as AzurReward[],
  permanent: false,
  until: '',
  password: '',
  name: '',
  reason: '',
  totp: '',
})

const open = (f: PassportSelfService) => {
  Object.assign(form, { title: '', content: '', rewards: [], grant: [], take: [], permanent: false, until: '', password: '', name: '', reason: '', totp: '' })
  formError.value = ''
  active.value = f
}

const required = (field: string) => t('game.self.required', { field })

const call = (f: PassportSelfService): Promise<MySelfServiceResponse> | string => {
  const base = { serverId: props.serverId, reason: form.reason.trim(), totpCode: form.totp.trim() }
  switch (f.command) {
    case 'player.mail':
      if (!form.title.trim()) return required(t('azur.form.title'))
      return azur.mailMe({ ...base, title: form.title.trim(), content: form.content, rewards: form.rewards })
    case 'item.adjust':
      if (!form.grant.length && !form.take.length) return required(t('azur.form.rewards'))
      return azur.adjustMyItems({ ...base, grant: form.grant, take: form.take })
    case 'player.kick':
      return azur.kickMe(base)
    case 'player.ban': {
      const until = form.permanent ? 0 : new Date(form.until).getTime() || 0
      if (!form.permanent && until <= Date.now()) return required(t('azur.form.until'))
      return azur.banMe({ ...base, until })
    }
    case 'player.unban':
      return azur.unbanMe(base)
    case 'player.password':
      if (form.password.length < 6) return required(t('game.self.password'))
      return azur.setMyPassword({ ...base, password: form.password })
    case 'player.rename':
      if (!form.name.trim()) return required(t('game.self.newName'))
      return azur.renameMe({ ...base, name: form.name.trim() })
  }
  return f.command
}

const submit = async () => {
  const f = active.value
  if (!f) return
  formError.value = ''
  if (f.needTotp && !/^\d{6}$/.test(form.totp.trim())) {
    formError.value = required(t('game.self.totpCode'))
    return
  }
  const pending = call(f)
  if (typeof pending === 'string') {
    formError.value = pending
    return
  }
  running.value = true
  try {
    const res = await pending
    const status = res.result?.status || ''
    if (status === 'failed') {
      formError.value = res.result?.errorMessage || res.result?.errorCode || statusText(status)
    } else {
      ui.toast(status === 'pending' ? t('game.self.resultPending') : t('game.self.resultSuccess'), 'ok')
      active.value = null
    }
    void loadItems()
    void loadBox('requests')
    void loadBox('records')
  } catch (e) {
    formError.value = errorText(e)
  } finally {
    running.value = false
  }
}

void loadItems()
void loadBox('requests')
void loadBox('records')
</script>

<style scoped>
.azs {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.azs__banner {
  margin: 0;
}
.azs__main {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(240px, 1fr);
  gap: 14px;
}
@media (max-width: 860px) {
  .azs__main {
    grid-template-columns: 1fr;
  }
}
.azs__fns {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
  align-content: start;
}
.azs__fn {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.azs__fn-title {
  font-weight: 600;
}
.azs__rules {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.azs__fn-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: auto;
}
.azs__aside {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.azs__box {
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.azs__box-head {
  font-weight: 600;
}
.azs__records {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.azs__records li {
  display: flex;
  flex-direction: column;
}
.azs__rec-title {
  font-size: 14px;
}
.azs__form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.azs__form input,
.azs__form textarea {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid var(--m-line);
  border-radius: 10px;
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
}
.azs__check {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
</style>
