<!-- 运营管理的发布条（页面底部悬浮，只在有未发布的改动时出现）：哪几份配置改了多少处、自动保存状态；
     查看改动、放弃、发布。发布时把改过的每份配置都先用服务端的加载逻辑校验，全部通过才发布；
     服务端的配置在编辑期间被别人发布过时提示冲突。 -->
<template>
  <div class="opb" :class="{ 'is-mobile': menuStore.isMobile }">
    <div v-for="d in staleDomains" :key="d.key" class="opb-stale" role="alert">
      <UIcon name="i-lucide-triangle-alert" class="opb-stale__icon" />
      <span class="opb-stale__text">{{ t('gmConfig.ops.stale', { name: titleOf(d) }) }}</span>
      <UButton class="gmc-btn" size="sm" color="neutral" variant="outline" @click="d.state.rebase()">{{ t('gmConfig.bar.rebase') }}</UButton>
    </div>

    <div v-if="shown" class="opb__bar">
      <span class="opb__dot" :class="dotClass" aria-hidden="true" />
      <span class="opb__text">
        <span class="opb__title">{{ t('gmConfig.ops.unpublished', { n: totalChanges }) }}</span>
        <span class="opb__sub">{{ subline }}</span>
      </span>
      <span class="opb__actions">
        <button type="button" class="opb__btn" :disabled="!dirty.length" @click="diffOpen = true">{{ t('gmConfig.bar.viewChanges') }}</button>
        <button type="button" class="opb__btn" @click="confirmDiscard">{{ t('gmConfig.bar.discard') }}</button>
        <UButton class="gmc-btn opb__publish" icon="i-lucide-send" :disabled="!dirty.length" @click="openPublish">{{ t('gmConfig.bar.publish') }}</UButton>
      </span>
    </div>

    <FormDialog v-model="diffOpen" :title="t('gmConfig.bar.changesTitle', { n: totalChanges })" :width="760" :show-footer="false">
      <div class="opb__diffs game-skin" :data-game="ctx.game.value || undefined">
        <section v-for="d in dirty" :key="d.key" class="opb__diff">
          <h4 class="opb__diff-title">{{ titleOf(d) }} · {{ t('gmConfig.ops.changesN', { n: d.state.changes.value.length }) }}</h4>
          <ConfigDiffView :entries="d.state.changes.value" :zone="zone" :group-label="d.groupLabel" :ref-label="d.refLabel" />
        </section>
      </div>
    </FormDialog>

    <FormDialog v-model="publishOpen" :title="t('gmConfig.ops.publishTitle')" :width="640" :show-footer="false">
      <div class="opd game-skin" :data-game="ctx.game.value || undefined">
        <p class="opd__lead">{{ t('gmConfig.ops.publishLead', { n: totalChanges }) }}</p>
        <div v-for="d in dirty" :key="d.key" class="opd__item">
          <div class="opd__head">
            <span class="opd__name">{{ titleOf(d) }}</span>
            <span class="gmc-hint">{{ t('gmConfig.ops.changesN', { n: d.state.changes.value.length }) }}</span>
            <span class="opd__spacer" />
            <span class="opd__state" :class="`is-${checkOf(d).level}`">
              <UIcon :name="checkOf(d).icon" class="opd__icon" :class="{ 'is-spin': d.state.validating.value }" />
              {{ checkOf(d).text }}
            </span>
          </div>
          <ul v-if="d.blockers.value.length" class="gmc-problems">
            <li v-for="(b, i) in d.blockers.value" :key="i"><UIcon name="i-lucide-circle-x" />{{ b }}</li>
          </ul>
          <p v-if="d.state.validationFresh.value && d.state.validation.value && !d.state.validation.value.valid" class="opd__error m-mono">
            {{ d.state.validation.value.error }}
          </p>
        </div>
        <label class="opd__field">
          <span class="opd__label">{{ t('gmConfig.publish.comment') }}</span>
          <input v-model="comment" class="gmc-input" :placeholder="t('gmConfig.publish.commentHint')" :disabled="publishing" />
        </label>
        <p class="gmc-hint">{{ t('gmConfig.publish.hotReload') }}</p>
        <div class="opd__actions">
          <UButton class="gmc-btn" color="neutral" variant="ghost" :disabled="publishing" @click="publishOpen = false">{{ t('gmConfig.common.cancel') }}</UButton>
          <UButton class="gmc-btn" color="neutral" variant="outline" :disabled="publishing || anyValidating" @click="validateAll">{{ t('gmConfig.publish.validate') }}</UButton>
          <UButton class="gmc-btn" icon="i-lucide-send" :loading="publishing" :disabled="!canPublish" @click="publish">{{ t('gmConfig.publish.confirm') }}</UButton>
        </div>
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Dialog } from '@/utils/dialog'
import { getRequestErrorMessage, isRequestErrorHandled } from '@/utils/request'
import { useGmContext } from '../../context'
import { useOpsDomains, type OpsDomain } from './domains'
import ConfigDiffView from './shared/ConfigDiffView.vue'

defineProps<{ zone: string }>()

const { t } = useI18n()
const menuStore = useMenuStore()
const ctx = useGmContext()
const domains = useOpsDomains()

const titleOf = (d: OpsDomain) => t(`gmConfig.ops.domains.${d.key}`)

const dirty = computed(() => domains.list.filter((d) => d.state.dirty.value))
const drafted = computed(() => domains.list.filter((d) => d.state.dirty.value || !!d.state.info.value?.draft))
const staleDomains = computed(() => drafted.value.filter((d) => d.state.stale.value))
const saving = computed(() => domains.list.some((d) => d.state.saving.value))
const saveError = computed(() => domains.list.find((d) => d.state.saveError.value)?.state.saveError.value || '')
const shown = computed(() => drafted.value.length > 0 || saving.value || !!saveError.value)
const totalChanges = computed(() => dirty.value.reduce((n, d) => n + d.state.changes.value.length, 0))

const subline = computed(() => {
  const parts = dirty.value.map((d) => `${titleOf(d)} ${d.state.changes.value.length}`)
  if (saving.value) parts.push(t('gmConfig.bar.saving'))
  else if (saveError.value) parts.push(t('gmConfig.bar.saveFailed', { error: saveError.value }))
  else parts.push(t('gmConfig.ops.savedDraft'))
  return parts.join(' · ')
})
const dotClass = computed(() => (saveError.value || staleDomains.value.length ? 'is-bad' : 'is-warn'))

// ---- 查看改动 / 放弃 ----
const diffOpen = ref(false)
const confirmDiscard = () => {
  Dialog.confirm({
    title: t('gmConfig.bar.discardTitle'),
    content: t('gmConfig.ops.discardContent'),
    cancelText: t('gmConfig.common.cancel'),
    confirmText: t('gmConfig.bar.discard'),
    onConfirm: async () => {
      try {
        for (const d of drafted.value) await d.state.discard()
        feedback.success(t('gmConfig.bar.discarded'))
      } catch (e) {
        if (!isRequestErrorHandled(e)) feedback.error(getRequestErrorMessage(e))
      }
    },
  })
}

// ---- 发布 ----
const publishOpen = ref(false)
const publishing = ref(false)
const comment = ref('')
const anyValidating = computed(() => dirty.value.some((d) => d.state.validating.value))

const validateAll = async () => {
  for (const d of dirty.value) await d.state.flush()
  await Promise.all(dirty.value.map((d) => d.state.validate()))
}
const openPublish = () => {
  comment.value = ''
  publishOpen.value = true
  void Promise.all(dirty.value.map((d) => d.state.flush())).then(() =>
    Promise.all(dirty.value.filter((d) => !d.state.validationFresh.value).map((d) => d.state.validate())),
  )
}

const checkOf = (d: OpsDomain) => {
  const s = d.state
  if (s.stale.value) return { level: 'bad', icon: 'i-lucide-triangle-alert', text: t('gmConfig.publish.stale') }
  if (d.blockers.value.length) return { level: 'bad', icon: 'i-lucide-circle-x', text: t('gmConfig.ops.hasProblems') }
  if (s.validating.value) return { level: 'idle', icon: 'i-lucide-loader-circle', text: t('gmConfig.publish.validating') }
  if (!s.validationFresh.value || !s.validation.value) return { level: 'idle', icon: 'i-lucide-shield-check', text: t('gmConfig.publish.notValidated') }
  if (s.validation.value.failed) return { level: 'bad', icon: 'i-lucide-circle-x', text: t('gmConfig.publish.validateFailed') }
  return s.validation.value.valid
    ? { level: 'ok', icon: 'i-lucide-circle-check', text: t('gmConfig.publish.valid') }
    : { level: 'bad', icon: 'i-lucide-circle-x', text: t('gmConfig.publish.invalid') }
}

const canPublish = computed(() => !publishing.value && dirty.value.length > 0 && dirty.value.every((d) => checkOf(d).level === 'ok'))

const publish = async () => {
  if (!canPublish.value) return
  publishing.value = true
  const done: string[] = []
  try {
    for (const d of dirty.value.slice()) {
      const res = await d.state.publish(comment.value.trim())
      if (res === 'conflict') {
        feedback.warning(t('gmConfig.ops.conflict', { name: titleOf(d) }))
        return
      }
      done.push(titleOf(d))
    }
    publishOpen.value = false
    feedback.success(t('gmConfig.ops.published', { names: done.join('、') }))
  } catch (e) {
    if (!isRequestErrorHandled(e)) feedback.error(getRequestErrorMessage(e))
  } finally {
    publishing.value = false
  }
}

</script>

<style lang="scss" src="./shared/form.scss"></style>
<style scoped lang="scss">
.opb {
  position: sticky;
  bottom: 16px;
  z-index: 20;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.opb-stale {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: var(--app-radius-lg);
  background: var(--m-warnSoft);
  border: 1px solid color-mix(in srgb, var(--m-warn) 40%, transparent);
}
.opb-stale__icon {
  width: 18px;
  height: 18px;
  color: var(--m-warn);
}
.opb-stale__text {
  flex: 1 1 220px;
  font-size: 13px;
  line-height: 1.5;
}
.opb__bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 10px 10px 10px 18px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: var(--app-shadow-lg);
}
.opb__dot {
  flex: none;
  width: 10px;
  height: 10px;
  border-radius: 50%;
}
.opb__dot.is-warn {
  background: var(--m-g, var(--m-warn));
}
.opb__dot.is-bad {
  background: var(--m-bad);
}
.opb__text {
  flex: 1 1 260px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.opb__title {
  font-size: 14px;
  font-weight: 800;
}
.opb__sub {
  font-size: 12px;
  opacity: 0.78;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.opb__actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}
.opb__btn {
  min-height: 40px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--m-bg);
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
}
.opb__btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--m-bg) 14%, transparent);
}
.opb__btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.opb__publish {
  border-radius: 999px;
  padding-inline: 18px;
}
.opb__diffs {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.opb__diff-title {
  margin: 0 0 8px;
  font-size: 14px;
  font-weight: 800;
}
.opd {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.opd__lead {
  margin: 0;
  font-size: 14px;
}
.opd__item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
}
.opd__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.opd__name {
  font-weight: 800;
}
.opd__spacer {
  flex: 1;
}
.opd__state {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 700;
}
.opd__state.is-ok {
  color: var(--m-ok);
}
.opd__state.is-bad {
  color: var(--m-bad);
}
.opd__state.is-idle {
  color: var(--el-text-color-secondary);
}
.opd__icon {
  width: 16px;
  height: 16px;
}
.opd__icon.is-spin {
  animation: opd-spin 1s linear infinite;
}
@keyframes opd-spin {
  to {
    transform: rotate(360deg);
  }
}
.opd__error {
  margin: 0;
  font-size: 12px;
  color: var(--m-bad);
  overflow-wrap: anywhere;
}
.opd__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.opd__label {
  font-size: 13px;
  font-weight: 700;
}
.opd__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}
.opb.is-mobile {
  bottom: calc(84px + env(safe-area-inset-bottom, 0px));
}
.is-mobile .opb__bar {
  border-radius: var(--app-radius-xl);
  padding: 12px 14px;
}
.is-mobile .opb__actions {
  width: 100%;
  justify-content: space-between;
}
</style>
