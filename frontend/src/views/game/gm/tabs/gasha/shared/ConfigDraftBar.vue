<!-- 配置草稿条（页面底部悬浮）：草稿状态与改动数、自动保存状态；查看改动、放弃草稿、发布。
     发布前先用服务端的加载逻辑校验（MGP 试运行），必须填原因；服务端配置在此期间被别人改过时提示冲突。 -->
<template>
  <div class="cdb-wrap" :class="{ 'is-mobile': menuStore.isMobile }">
    <div v-if="state.stale.value" class="cdb-stale" role="alert">
      <UIcon name="i-lucide-triangle-alert" class="cdb-stale__icon" />
      <span class="cdb-stale__text">{{ t('gmConfig.bar.stale') }}</span>
      <UButton class="gmc-btn" size="sm" color="neutral" variant="outline" :loading="rebasing" @click="rebase">{{ t('gmConfig.bar.rebase') }}</UButton>
      <UButton class="gmc-btn" size="sm" color="neutral" variant="ghost" @click="confirmDiscard">{{ t('gmConfig.bar.discard') }}</UButton>
    </div>

    <div class="cdb" :class="{ 'is-dirty': state.dirty.value }">
      <span class="cdb__dot" :class="dotClass" aria-hidden="true" />
      <span class="cdb__text">
        <span class="cdb__title">{{ title }}</span>
        <span class="cdb__sub">{{ subline }}</span>
      </span>
      <span class="cdb__actions">
        <button type="button" class="cdb__btn" :disabled="!state.dirty.value" @click="diffOpen = true">{{ t('gmConfig.bar.viewChanges') }}</button>
        <button type="button" class="cdb__btn" :disabled="!state.dirty.value && !state.info.value?.draft" @click="confirmDiscard">
          {{ t('gmConfig.bar.discard') }}
        </button>
        <UButton class="gmc-btn cdb__publish" icon="i-lucide-send" :disabled="!state.dirty.value" @click="openPublish">{{ t('gmConfig.bar.publish') }}</UButton>
      </span>
    </div>

    <FormDialog v-model="diffOpen" :title="t('gmConfig.bar.changesTitle', { n: state.changes.value.length })" :width="760" :show-footer="false">
      <div class="game-skin" :data-game="ctx.game.value || undefined">
        <ConfigDiffView :entries="state.changes.value" :zone="zone" :group-label="groupLabel" :ref-label="refLabel" />
      </div>
    </FormDialog>

    <FormDialog v-model="publishOpen" :title="t('gmConfig.publish.title', { name: domainTitle })" :width="600" :show-footer="false">
      <div class="cpd game-skin" :data-game="ctx.game.value || undefined">
        <p class="cpd__lead">{{ t('gmConfig.publish.lead', { n: state.changes.value.length }) }}</p>

        <div class="cpd__checks">
          <div v-for="(b, i) in blockers" :key="i" class="cpd__check is-bad">
            <UIcon name="i-lucide-circle-x" class="cpd__icon" />
            <span>{{ b }}</span>
          </div>
          <div class="cpd__check" :class="validationClass">
            <UIcon :name="validationIcon" class="cpd__icon" :class="{ 'is-spin': state.validating.value }" />
            <span class="cpd__vtext">
              <span>{{ validationText }}</span>
              <span v-if="state.validation.value && !state.validation.value.valid && state.validationFresh.value" class="cpd__error m-mono">
                {{ state.validation.value.error }}
              </span>
            </span>
            <UButton class="gmc-btn"
              v-if="!state.validating.value && (!state.validationFresh.value || state.validation.value?.failed)"
              size="xs"
              color="neutral"
              variant="outline"
              @click="state.validate()"
            >
              {{ t('gmConfig.publish.validate') }}
            </UButton>
          </div>
          <div v-if="state.stale.value" class="cpd__check is-bad">
            <UIcon name="i-lucide-triangle-alert" class="cpd__icon" />
            <span>{{ t('gmConfig.publish.stale') }}</span>
          </div>
        </div>

        <label class="cpd__field">
          <span class="cpd__label">{{ t('gmConfig.publish.comment') }}</span>
          <input v-model="comment" class="cpd__input" :placeholder="t('gmConfig.publish.commentHint')" :disabled="publishing" />
        </label>
        <p v-if="state.info.value?.hotReload" class="cpd__hint">{{ t('gmConfig.publish.hotReload') }}</p>

        <div class="cpd__actions">
          <UButton class="gmc-btn" color="neutral" variant="ghost" :disabled="publishing" @click="publishOpen = false">{{ t('gmConfig.common.cancel') }}</UButton>
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
import { useGmContext } from '../../../context'
import ConfigDiffView from './ConfigDiffView.vue'
import { formatZoned, toMs } from './time'
import type { ConfigDomainState } from './useConfigDomain'

const props = withDefaults(
  defineProps<{
    state: ConfigDomainState<unknown>
    zone: string
    /** 本地检查发现的阻止发布的问题 */
    blockers?: string[]
    groupLabel?: (path: string[]) => string
    /** 配置里 ID 的名称（见 ConfigDiffView） */
    refLabel?: (collection: string, id: string) => string
  }>(),
  { blockers: () => [], groupLabel: undefined, refLabel: undefined },
)
const emit = defineEmits<{ published: [seq: number] }>()

const { t } = useI18n()
const menuStore = useMenuStore()
const ctx = useGmContext()

const diffOpen = ref(false)
const publishOpen = ref(false)
const comment = ref('')
const publishing = ref(false)
const rebasing = ref(false)

const domainTitle = computed(() => props.state.info.value?.title || props.state.domain)

const groupNames = computed(() => {
  const seen: string[] = []
  for (const e of props.state.changes.value) {
    const head = e.path.slice(0, 2)
    const label = props.groupLabel?.(head) || ''
    if (label && !seen.includes(label)) seen.push(label)
    if (seen.length >= 3) break
  }
  return seen
})

const title = computed(() => {
  const s = props.state
  if (s.dirty.value) return t('gmConfig.bar.dirty', { n: s.changes.value.length })
  return t('gmConfig.bar.clean')
})

const subline = computed(() => {
  const s = props.state
  const parts: string[] = []
  if (s.saving.value) parts.push(t('gmConfig.bar.saving'))
  else if (s.saveError.value) parts.push(t('gmConfig.bar.saveFailed', { error: s.saveError.value }))
  if (s.dirty.value) {
    if (groupNames.value.length) parts.push(groupNames.value.join('、'))
    const draft = s.info.value?.draft
    const at = toMs(draft?.updateTime) || s.savedAt.value
    if (!s.saving.value && !s.saveError.value && at) {
      parts.push(t('gmConfig.bar.savedBy', { name: draft?.authorName || '—', time: formatZoned(at, props.zone) }))
    }
    if (s.info.value?.hotReload) parts.push(t('gmConfig.bar.hotReload'))
  } else {
    const info = s.info.value
    const at = toMs(info?.lastPublishedAt)
    if (info?.lastSeq && at) {
      parts.push(t('gmConfig.bar.lastPublished', { seq: info.lastSeq, name: info.lastPublishedBy || '—', time: formatZoned(at, props.zone) }))
    } else {
      parts.push(t('gmConfig.bar.neverPublished'))
    }
  }
  return parts.join(' · ')
})

const dotClass = computed(() => {
  const s = props.state
  if (s.saveError.value || s.stale.value) return 'is-bad'
  if (s.dirty.value) return 'is-warn'
  return 'is-ok'
})

const confirmDiscard = () => {
  Dialog.confirm({
    title: t('gmConfig.bar.discardTitle'),
    content: t('gmConfig.bar.discardContent'),
    cancelText: t('gmConfig.common.cancel'),
    confirmText: t('gmConfig.bar.discard'),
    onConfirm: async () => {
      try {
        await props.state.discard()
        feedback.success(t('gmConfig.bar.discarded'))
      } catch (e) {
        if (!isRequestErrorHandled(e)) feedback.error(getRequestErrorMessage(e))
      }
    },
  })
}

const rebase = async () => {
  rebasing.value = true
  try {
    await props.state.rebase()
  } finally {
    rebasing.value = false
  }
}

const openPublish = () => {
  comment.value = ''
  publishOpen.value = true
  void props.state.flush().then(() => {
    if (!props.state.validationFresh.value) void props.state.validate()
  })
}

const validationClass = computed(() => {
  const s = props.state
  if (s.validating.value || !s.validationFresh.value) return 'is-idle'
  return s.validation.value?.valid ? 'is-ok' : 'is-bad'
})
const validationIcon = computed(() => {
  const s = props.state
  if (s.validating.value) return 'i-lucide-loader-circle'
  if (!s.validationFresh.value) return 'i-lucide-shield-check'
  return s.validation.value?.valid ? 'i-lucide-circle-check' : 'i-lucide-circle-x'
})
const validationText = computed(() => {
  const s = props.state
  if (s.validating.value) return t('gmConfig.publish.validating')
  if (!s.validationFresh.value || !s.validation.value) return t('gmConfig.publish.notValidated')
  if (s.validation.value.failed) return t('gmConfig.publish.validateFailed')
  return s.validation.value.valid ? t('gmConfig.publish.valid') : t('gmConfig.publish.invalid')
})

const canPublish = computed(
  () =>
    !publishing.value &&
    props.blockers.length === 0 &&
    !props.state.stale.value &&
    props.state.dirty.value &&
    props.state.validationFresh.value &&
    !!props.state.validation.value?.valid,
)

const publish = async () => {
  if (!canPublish.value) return
  publishing.value = true
  try {
    const res = await props.state.publish(comment.value.trim())
    if (res === 'conflict') {
      feedback.warning(t('gmConfig.publish.conflict'))
      return
    }
    publishOpen.value = false
    feedback.success(t('gmConfig.publish.done', { seq: res }))
    emit('published', res)
  } catch (e) {
    if (!isRequestErrorHandled(e)) feedback.error(getRequestErrorMessage(e))
  } finally {
    publishing.value = false
  }
}
</script>

<style lang="scss" src="./form.scss"></style>
<style scoped lang="scss">
.cdb-wrap {
  position: sticky;
  bottom: 16px;
  z-index: 20;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.cdb-stale {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: var(--app-radius-lg);
  background: var(--m-warnSoft);
  border: 1px solid color-mix(in srgb, var(--m-warn) 40%, transparent);
}
.cdb-stale__icon {
  width: 18px;
  height: 18px;
  color: var(--m-warn);
  flex: none;
}
.cdb-stale__text {
  flex: 1 1 220px;
  font-size: 13px;
  line-height: 1.5;
}
.cdb {
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
.cdb__dot {
  flex: none;
  width: 10px;
  height: 10px;
  border-radius: 50%;
}
.cdb__dot.is-ok {
  background: var(--m-ok);
}
.cdb__dot.is-warn {
  background: var(--m-g, var(--m-warn));
}
.cdb__dot.is-bad {
  background: var(--m-bad);
}
.cdb__text {
  flex: 1 1 260px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.cdb__title {
  font-size: 14px;
  font-weight: 800;
}
.cdb__sub {
  font-size: 12px;
  opacity: 0.78;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cdb__actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}
.cdb__btn {
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
.cdb__btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--m-bg) 14%, transparent);
}
.cdb__btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.cdb__publish {
  min-height: 40px;
  border-radius: 999px;
  padding-inline: 18px;
}

.cpd {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.cpd__lead {
  margin: 0;
  font-size: 14px;
}
.cpd__checks {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.cpd__check {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  font-size: 13px;
  line-height: 1.5;
  background: var(--m-s2);
}
.cpd__check.is-ok {
  background: var(--m-okSoft);
}
.cpd__check.is-bad {
  background: var(--m-badSoft);
}
.cpd__icon {
  flex: none;
  width: 18px;
  height: 18px;
  margin-top: 1px;
}
.is-ok > .cpd__icon {
  color: var(--m-ok);
}
.is-bad > .cpd__icon {
  color: var(--m-bad);
}
.cpd__icon.is-spin {
  animation: cpd-spin 1s linear infinite;
}
@keyframes cpd-spin {
  to {
    transform: rotate(360deg);
  }
}
.cpd__vtext {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.cpd__error {
  font-size: 12px;
  overflow-wrap: anywhere;
  color: var(--m-bad);
}
.cpd__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.cpd__label {
  font-size: 13px;
  font-weight: 700;
}
.cpd__req {
  color: var(--m-bad);
  margin-left: 2px;
}
.cpd__input {
  min-height: 44px;
  padding: 0 14px;
  border-radius: var(--app-radius);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  outline: none;
}
.cpd__input:focus {
  border-color: var(--m-p);
}
.cpd__hint {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.cpd__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

.cdb-wrap.is-mobile {
  bottom: calc(84px + env(safe-area-inset-bottom, 0px));
}
.is-mobile .cdb {
  border-radius: var(--app-radius-xl);
  padding: 12px 14px;
}
.is-mobile .cdb__actions {
  width: 100%;
  justify-content: space-between;
}
.is-mobile .cdb__btn,
.is-mobile .cdb__publish {
  min-height: 44px;
}
.is-mobile .cdb__sub {
  white-space: normal;
}
</style>
