<!-- 编辑登录奖励（运营管理里点日历上的登录奖励直接打开）的全部设置：每日登录（7 天一循环）、累计登录、
     限时登录（期间内每登录一天领下一天的奖励）、自选登录（每天从几个候选里选一个）。 -->
<template>
  <ItemDialog
    :model-value="modelValue"
    :title="title"
    :width="860"
    :state="current ? 'ok' : before ? 'removed' : 'missing'"
    :status="timed ? periodStatus(timed.startTime, timed.endTime, clock.now.value) : undefined"
    :tags="[t(`gmConfig.login.kinds.${kind}`)]"
    :draft="draftOf(before, current)"
    :copy-label="timed ? t('gmConfig.ops.copy') : ''"
    :deletable="!!timed"
    @update:model-value="emit('update:modelValue', $event)"
    @copy="copy"
    @delete="remove"
    @restore="restore"
  >
    <template v-if="current">
      <GmSchemaForm :schema="schema" :model-value="current" keep-empty :image-sources="images" @update:model-value="assignInPlace(current, $event)" />
      <ul v-if="problems.length" class="gmc-problems">
        <li v-for="p in problems" :key="p.key"><UIcon name="i-lucide-circle-x" />{{ t(`gmConfig.login.problems.${p.key}`, p.params || {}) }}</li>
      </ul>
    </template>
  </ItemDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LoginBonusEvent, LoginBonusSelect } from '@/games/gakumas/types'
import GmSchemaForm from '../../../components/GmSchemaForm.vue'
import { useGmContext } from '../../../context'
import type { ItemRef } from '../calendar'
import { useOpsDomains } from '../domains'
import { loginSchema } from '../schemas'
import { loginTitleSource, usedSource } from '../shared/assets'
import { assignInPlace, cloneJson, draftOf, nextHour, periodStatus, useTracked } from '../shared/entry'
import ItemDialog from '../shared/ItemDialog.vue'
import { useServerClock } from '../shared/serverClock'
import { addLoginBonus, eventProblems, normalProblems, selectProblems, totalProblems } from './login'

const props = defineProps<{ modelValue: boolean; id: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; open: [ref: ItemRef] }>()

const { t, te } = useI18n()
const ctx = useGmContext()
const clock = useServerClock(ctx)
const state = useOpsDomains().loginBonus.state
const cfg = computed(() => state.working.value)
const pub = computed(() => state.published.value)

// normal（每日）、total（累计）各只有一个；限时、自选按 ID 跟着对象
const event = useTracked(() => cfg.value?.events, (e) => e.id, props.id)
const select = useTracked(() => cfg.value?.selects, (s) => s.id, props.id)
const kind = computed(() =>
  props.id === 'normal' || props.id === 'total' ? props.id : select.value || pub.value?.selects.some((s) => s.id === props.id) ? 'select' : 'event',
)
const timed = computed<LoginBonusEvent | LoginBonusSelect | undefined>(() => event.value || select.value)
const current = computed(() => (kind.value === 'normal' ? cfg.value?.normal : kind.value === 'total' ? cfg.value?.total : timed.value))
const before = computed(() => {
  const p = pub.value
  if (!p) return undefined
  if (kind.value === 'normal') return p.normal
  if (kind.value === 'total') return p.total
  return p.events.find((e) => e.id === props.id) || p.selects.find((s) => s.id === props.id)
})
const title = computed(() => current.value?.name || before.value?.name || t(`gmConfig.login.kinds.${kind.value}`))
const schema = computed(() => loginSchema(t, te, kind.value))

// 图片：标题图从素材包选，背景从配置里用过的选（显示是哪个登录奖励在用）
const images = computed(() => {
  const c = cfg.value
  const all = c ? [c.normal, ...c.events, ...c.selects] : []
  return {
    loginTitle: [loginTitleSource(t('gmConfig.schema.login.titleAssetId.title')), usedSource('usedTitle', t('gmConfig.ops.usedImages'), all.map((x) => ({ assetId: x.titleAssetId, owner: x.name })))],
    loginBackground: [usedSource('loginBackground', t('gmConfig.schema.login.backgroundAssetId.title'), all.map((x) => ({ assetId: x.backgroundAssetId, owner: x.name })))],
  }
})

const problems = computed(() => {
  const c = cfg.value
  if (!c) return []
  if (kind.value === 'normal') return normalProblems(c)
  if (kind.value === 'total') return totalProblems(c)
  if (event.value) return eventProblems(c, event.value)
  if (select.value) return selectProblems(c, select.value)
  return []
})

// ---- 复制 / 删除 / 撤销删除（限时、自选） ----
const copy = () => {
  const c = cfg.value
  const x = timed.value
  if (!c || !x) return
  emit('open', { kind: 'login', id: addLoginBonus(c, event.value ? 'event' : 'select', nextHour(clock.now.value), (from) => t('gmConfig.ops.copyName', { name: from || '' }), x) })
}
const remove = () => {
  const c = cfg.value
  if (!c) return
  if (event.value) c.events.splice(c.events.indexOf(event.value), 1)
  else if (select.value) c.selects.splice(c.selects.indexOf(select.value), 1)
}
const restore = () => {
  const c = cfg.value
  const p = pub.value
  if (!c || !p) return
  const e = p.events.find((x) => x.id === props.id)
  if (e) c.events.push(cloneJson(e))
  const s = p.selects.find((x) => x.id === props.id)
  if (s) c.selects.push(cloneJson(s))
}
</script>

<style lang="scss" src="../shared/form.scss"></style>
