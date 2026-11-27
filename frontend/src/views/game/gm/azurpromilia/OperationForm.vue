<!-- 对玩家执行一个 GM 操作：选操作（或由调用方指定）→ 填参数 → 试运行 / 执行，下面显示逐个玩家的结果。 -->
<template>
  <div class="azo">
    <div v-if="!fixed" class="azo__ops" role="radiogroup" :aria-label="t('azurGm.ops.choose')">
      <button
        v-for="o in OPS"
        :key="o.id"
        type="button"
        role="radio"
        class="azo__op"
        :class="{ 'is-on': op === o.id }"
        :aria-checked="op === o.id"
        @click="op = o.id"
      >
        <span>{{ t(`gm.commands.azurpromilia.${o.id}`) }}</span>
        <GmRiskPill :risk="o.risk" />
      </button>
    </div>

    <p v-if="single && pids.length > 1" class="azo__warn">{{ t('azurGm.ops.onePlayer') }}</p>

    <div class="azo__form">
      <template v-if="op === 'player.mail'">
        <label class="azo__field">
          <span>{{ t('azurGm.form.title') }}</span>
          <input v-model="mail.title" maxlength="64" />
        </label>
        <label class="azo__field">
          <span>{{ t('azurGm.form.sender') }}</span>
          <input v-model="mail.sender" maxlength="32" />
        </label>
        <label class="azo__field">
          <span>{{ t('azurGm.form.content') }}</span>
          <textarea v-model="mail.content" rows="4" />
        </label>
        <div class="azo__field">
          <span>{{ t('azurGm.form.rewards') }}</span>
          <RewardEditor v-model="mail.rewards" />
        </div>
        <label class="azo__field azo__field--short">
          <span>{{ t('azurGm.form.effectDays') }}</span>
          <input v-model.number="mail.days" type="number" min="0" max="3650" />
        </label>
      </template>
      <template v-else-if="op === 'item.adjust'">
        <div class="azo__field">
          <span>{{ t('azurGm.form.grant') }}</span>
          <RewardEditor v-model="grant" />
        </div>
        <div class="azo__field">
          <span>{{ t('azurGm.form.take') }}</span>
          <RewardEditor v-model="take" :types="takeTypes" />
        </div>
      </template>
      <template v-else-if="op === 'player.ban'">
        <label class="azo__check">
          <input v-model="ban.permanent" type="checkbox" />
          {{ t('azurGm.form.permanent') }}
        </label>
        <label v-if="!ban.permanent" class="azo__field azo__field--short">
          <span>{{ t('azurGm.form.until') }}</span>
          <input v-model="ban.until" type="datetime-local" />
        </label>
        <label class="azo__field">
          <span>{{ t('azurGm.form.reason') }}</span>
          <input v-model="ban.reason" maxlength="256" />
        </label>
      </template>
      <label v-else-if="op === 'player.password'" class="azo__field azo__field--short">
        <span>{{ t('azurGm.form.password') }}</span>
        <input v-model="password" type="password" autocomplete="new-password" maxlength="64" />
      </label>
      <label v-else-if="op === 'player.rename'" class="azo__field azo__field--short">
        <span>{{ t('azurGm.form.name') }}</span>
        <input v-model="name" maxlength="32" />
      </label>
    </div>

    <p v-if="invalid" class="azo__warn">{{ invalid }}</p>
    <div class="azo__actions">
      <UButton color="neutral" variant="soft" :disabled="!ready" :loading="running === 'dry'" @click="run(true)">
        {{ t('azurGm.common.dryRun') }}
      </UButton>
      <UButton :disabled="!ready" :loading="running === 'run'" @click="run(false)">{{ t('azurGm.common.run') }}</UButton>
    </div>

    <p v-if="failure" class="azo__warn">{{ failure }}</p>
    <GmOperationResult v-if="outcome" :outcome="outcome" />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import * as api from '@/api/azurpromilia'
import type { OperationOutcome } from '@/games/mgp'
import type { MailContent, OperationResponse } from '@/types/v1/azurpromilia/gm'
import type { Reward } from '@/types/v1/azurpromilia/types'
import GmOperationResult from '../components/GmOperationResult.vue'
import GmRiskPill from '../components/GmRiskPill.vue'
import RewardEditor from './RewardEditor.vue'
import { REWARD_KINDS } from '@/games/azurpromilia/names'
import { canTake, errorMessage, fromInput, outcomeOf } from './shared'

type OpID = 'player.mail' | 'item.adjust' | 'player.kick' | 'player.ban' | 'player.unban' | 'player.password' | 'player.rename'

// 与服务端 AzurPromiliaOperations 的风险一致
const OPS: { id: OpID; risk: string }[] = [
  { id: 'player.mail', risk: 'medium' },
  { id: 'item.adjust', risk: 'high' },
  { id: 'player.kick', risk: 'low' },
  { id: 'player.ban', risk: 'high' },
  { id: 'player.unban', risk: 'high' },
  { id: 'player.password', risk: 'high' },
  { id: 'player.rename', risk: 'low' },
]

const props = defineProps<{ serverId: string; pids: string[]; fixed?: OpID }>()
const emit = defineEmits<{ done: [outcome: OperationOutcome] }>()
const { t } = useI18n()

const op = ref<OpID>(props.fixed || 'player.mail')
watch(
  () => props.fixed,
  (v) => {
    if (v) op.value = v
  },
)

const takeTypes = REWARD_KINDS.filter((k) => canTake(k.type)).map((k) => k.type)

const mail = reactive({ title: '', sender: '', content: '', rewards: [] as Reward[], days: 30 })
const grant = ref<Reward[]>([])
const take = ref<Reward[]>([])
const ban = reactive({ permanent: false, until: '', reason: '' })
const password = ref('')
const name = ref('')

const single = computed(() => op.value === 'player.rename')

const invalid = computed(() => {
  switch (op.value) {
    case 'player.mail':
      return mail.title.trim() ? '' : t('azurGm.form.required', { field: t('azurGm.form.title') })
    case 'item.adjust':
      return grant.value.length + take.value.length ? '' : t('azurGm.form.required', { field: t('azurGm.form.rewards') })
    case 'player.ban':
      return ban.permanent || fromInput(ban.until) > Date.now() ? '' : t('azurGm.form.required', { field: t('azurGm.form.until') })
    case 'player.password':
      return password.value.length >= 6 ? '' : t('azurGm.form.required', { field: t('azurGm.form.password') })
    case 'player.rename':
      return name.value.trim() ? '' : t('azurGm.form.required', { field: t('azurGm.form.name') })
  }
  return ''
})

const ready = computed(() => props.pids.length > 0 && !invalid.value && !(single.value && props.pids.length > 1) && !running.value)

const running = ref<'' | 'dry' | 'run'>('')
const outcome = ref<OperationOutcome>()
const failure = ref('')

const mailContent = (): MailContent => ({
  sender: mail.sender.trim(),
  title: mail.title.trim(),
  content: mail.content,
  rewards: mail.rewards,
  effect: Math.max(0, Math.floor(mail.days || 0)) * 86400,
})

const call = (dryRun: boolean): Promise<OperationResponse> => {
  const base = { serverId: props.serverId, pids: props.pids, dryRun }
  switch (op.value) {
    case 'player.mail':
      return api.mailPlayers({ ...base, mail: mailContent() })
    case 'item.adjust':
      return api.adjustItems({ ...base, grant: grant.value, take: take.value })
    case 'player.kick':
      return api.kickPlayers(base)
    case 'player.ban':
      return api.banPlayers({ ...base, until: ban.permanent ? 0 : fromInput(ban.until), reason: ban.reason.trim() })
    case 'player.unban':
      return api.unbanPlayers(base)
    case 'player.password':
      return api.resetPassword({ ...base, password: password.value })
    case 'player.rename':
      return api.renamePlayer({ ...base, name: name.value.trim() })
  }
}

const run = async (dryRun: boolean) => {
  running.value = dryRun ? 'dry' : 'run'
  failure.value = ''
  outcome.value = undefined
  try {
    const res = await call(dryRun)
    outcome.value = outcomeOf(res.execution, res.results, dryRun)
    if (!dryRun) emit('done', outcome.value)
  } catch (e) {
    failure.value = errorMessage(e)
  } finally {
    running.value = ''
  }
}

watch(op, () => {
  outcome.value = undefined
  failure.value = ''
})
</script>

<style scoped lang="scss">
.azo {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.azo__ops {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.azo__op {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-pill);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  cursor: pointer;
  &.is-on {
    border-color: var(--m-p);
    background: var(--m-pSoft);
    color: var(--m-pInk);
  }
}
.azo__form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.azo__field {
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
.azo__field--short {
  max-width: 320px;
}
.azo__check {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
}
.azo__actions {
  display: flex;
  gap: 8px;
}
.azo__warn {
  margin: 0;
  font-size: 13px;
  color: var(--m-bad);
}
</style>
