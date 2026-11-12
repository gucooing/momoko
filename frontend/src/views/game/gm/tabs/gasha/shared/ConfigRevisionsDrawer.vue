<!-- 版本记录（PC 右侧抽屉 / 手机底部面板）：经 Momoko 发布过的版本与当前草稿；
     可以对比某个版本与服务端当前内容，或把某个已发布版本恢复为草稿（之后照常检查、发布）。 -->
<template>
  <Teleport to="body">
    <Transition :name="menuStore.isMobile ? 'crd-sheet' : 'crd-drawer'">
      <div v-if="modelValue" class="crd game-skin" :data-game="game || undefined" :class="{ 'is-sheet': menuStore.isMobile }">
        <button type="button" class="crd__mask" :aria-label="t('gmConfig.common.close')" @click="close" />
        <aside class="crd__panel" role="dialog" aria-modal="true" :aria-label="t('gmConfig.revisions.title')">
          <span v-if="menuStore.isMobile" class="crd__handle" aria-hidden="true" />
          <header class="crd__head">
            <span class="m-sticker m-sticker--p crd__icon"><UIcon name="i-lucide-history" /></span>
            <span class="crd__titles">
              <h2 class="crd__title">{{ t('gmConfig.revisions.title') }}</h2>
              <span class="crd__sub">{{ t('gmConfig.revisions.sub', { name: state.info.value?.title || state.domain, n: total }) }}</span>
            </span>
            <AppIconButton icon="HOutline:XMarkIcon" :label="t('gmConfig.common.close')" :box="40" @click="close" />
          </header>

          <div v-if="loading && !items.length" class="crd__hint">{{ t('gmConfig.common.loading') }}</div>
          <EmptyState
            v-else-if="!items.length"
            icon="HOutline:ClockIcon"
            :title="t('gmConfig.revisions.empty')"
            :description="t('gmConfig.revisions.emptyDesc')"
          />
          <ol v-else class="crd__list">
            <li v-for="r in items" :key="r.id" class="crd__item">
              <div class="crd__row">
                <span class="crd__seq m-num" :class="{ 'is-draft': r.state !== 'published' }">
                  {{ r.state === 'published' ? t('gmConfig.revisions.version', { n: r.seq }) : t('gmConfig.revisions.draft') }}
                </span>
                <span class="crd__meta">
                  <span class="crd__comment">{{ r.comment || t('gmConfig.revisions.noComment') }}</span>
                  <span class="crd__by">{{ r.authorName || '—' }} · {{ timeOf(r) }}</span>
                </span>
                <AppIconButton
                  icon="HOutline:EyeIcon"
                  :label="t('gmConfig.revisions.compare')"
                  :box="40"
                  :active="openId === r.id"
                  @click="toggle(r)"
                />
                <ActionMenu
                  v-if="r.state === 'published'"
                  :items="[{ key: 'restore', label: t('gmConfig.revisions.restore'), icon: 'i-lucide-undo-2' }]"
                  @select="restore(r)"
                />
              </div>
              <div v-if="openId === r.id" class="crd__diff">
                <div v-if="detailLoading" class="crd__hint">{{ t('gmConfig.common.loading') }}</div>
                <template v-else>
                  <p class="crd__diff-lead">{{ t('gmConfig.revisions.compareLead') }}</p>
                  <ConfigDiffView
                    :entries="detailDiff"
                    :zone="zone"
                    :group-label="groupLabel"
                    :ref-label="refLabel"
                    :empty-text="t('gmConfig.revisions.same')"
                  />
                </template>
              </div>
            </li>
          </ol>
          <UButton class="gmc-btn" v-if="items.length < total" color="neutral" variant="outline" block :loading="loading" @click="load(page + 1)">
            {{ t('gmConfig.common.more') }}
          </UButton>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmConfigRevision, listGmConfigRevisions } from '@/api/gameGm'
import type { GmConfigRevision } from '@/types/v1/game_gm'
import { Dialog } from '@/utils/dialog'
import { getRequestErrorMessage, isRequestErrorHandled } from '@/utils/request'
import { useGmContext } from '../../../context'
import ConfigDiffView from './ConfigDiffView.vue'
import { diffJson, type DiffEntry } from './diff'
import { formatZoned, toMs } from './time'
import type { ConfigDomainState } from './useConfigDomain'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    state: ConfigDomainState<unknown>
    zone: string
    groupLabel?: (path: string[]) => string
    refLabel?: (collection: string, id: string) => string
  }>(),
  { groupLabel: undefined, refLabel: undefined },
)
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()
const menuStore = useMenuStore()
const ctx = useGmContext()
const game = computed(() => ctx.game.value)

const PAGE_SIZE = 20
const items = ref<GmConfigRevision[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const openId = ref('')
const detailLoading = ref(false)
const detailDiff = shallowRef<DiffEntry[]>([])
const contents = new Map<string, string>()

const close = () => emit('update:modelValue', false)

const load = async (p: number) => {
  loading.value = true
  try {
    const { data } = await listGmConfigRevisions(props.state.serverId, props.state.domain, p, PAGE_SIZE)
    items.value = p === 1 ? data.items || [] : [...items.value, ...(data.items || [])]
    total.value = data.total || 0
    page.value = p
  } catch (e) {
    if (!isRequestErrorHandled(e)) feedback.error(getRequestErrorMessage(e))
  } finally {
    loading.value = false
  }
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      openId.value = ''
      void load(1)
    }
  },
)

const timeOf = (r: GmConfigRevision) => {
  const ms = toMs(r.publishedAt) || toMs(r.createTime)
  return ms ? formatZoned(ms, props.zone) : '—'
}

const contentOf = async (r: GmConfigRevision) => {
  const hit = contents.get(r.id)
  if (hit !== undefined) return hit
  const { data } = await getGmConfigRevision(props.state.serverId, props.state.domain, r.id)
  contents.set(r.id, data.content)
  return data.content
}

const toggle = async (r: GmConfigRevision) => {
  if (openId.value === r.id) {
    openId.value = ''
    return
  }
  openId.value = r.id
  detailLoading.value = true
  try {
    const content = await contentOf(r)
    if (openId.value !== r.id) return
    detailDiff.value = diffJson(props.state.published.value, JSON.parse(content))
  } catch (e) {
    openId.value = ''
    if (!isRequestErrorHandled(e)) feedback.error(getRequestErrorMessage(e))
  } finally {
    detailLoading.value = false
  }
}

const restore = (r: GmConfigRevision) => {
  Dialog.confirm({
    title: t('gmConfig.revisions.restoreTitle', { seq: r.seq }),
    content: props.state.dirty.value ? t('gmConfig.revisions.restoreDirty') : t('gmConfig.revisions.restoreContent'),
    cancelText: t('gmConfig.common.cancel'),
    confirmText: t('gmConfig.revisions.restore'),
    onConfirm: async () => {
      try {
        const content = await contentOf(r)
        props.state.restore(content)
        feedback.success(t('gmConfig.revisions.restored', { seq: r.seq }))
        close()
      } catch (e) {
        if (!isRequestErrorHandled(e)) feedback.error(getRequestErrorMessage(e))
      }
    },
  })
}
</script>

<style lang="scss" src="./form.scss"></style>
<style scoped lang="scss">
.crd {
  position: fixed;
  inset: 0;
  z-index: 2100;
}
.crd__mask {
  position: absolute;
  inset: 0;
  border: 0;
  padding: 0;
  background: color-mix(in srgb, var(--m-ink) 16%, transparent);
  cursor: pointer;
}
.crd__panel {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  width: 520px;
  max-width: 100%;
  overflow-y: auto;
  scrollbar-width: none;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 24px;
  background: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color-light);
  box-shadow: var(--app-shadow-lg);
}
.crd__panel::-webkit-scrollbar {
  display: none;
}
.crd.is-sheet .crd__mask {
  background: color-mix(in srgb, var(--m-ink) 50%, transparent);
}
.crd.is-sheet .crd__panel {
  top: auto;
  left: 0;
  width: 100%;
  max-height: 86%;
  padding: 10px 16px calc(28px + env(safe-area-inset-bottom, 0px));
  border-left: none;
  border-radius: 30px 30px 0 0;
}
.crd__handle {
  align-self: center;
  flex: none;
  width: 44px;
  height: 5px;
  border-radius: 999px;
  background: var(--el-border-color-light);
}
.crd__head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.crd__icon {
  flex: none;
  width: 44px;
  height: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  font-size: 20px;
}
.crd__titles {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.crd__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 20px;
  font-weight: 800;
}
.crd__sub {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.crd__hint {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  padding: 8px 0;
}
.crd__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.crd__item {
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  padding: 8px 8px 8px 12px;
}
.crd__row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.crd__seq {
  flex: none;
  min-width: 44px;
  padding: 4px 8px;
  border-radius: 10px;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-weight: 800;
  text-align: center;
  font-size: 13px;
}
.crd__seq.is-draft {
  background: var(--m-warnSoft);
  color: var(--m-warn);
}
.crd__meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.crd__comment {
  font-weight: 700;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.crd__by {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.crd__diff {
  padding: 10px 4px 4px 0;
}
.crd__diff-lead {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.crd-drawer-enter-active,
.crd-drawer-leave-active,
.crd-sheet-enter-active,
.crd-sheet-leave-active {
  transition: opacity 0.2s;
}
.crd-drawer-enter-active .crd__panel,
.crd-drawer-leave-active .crd__panel,
.crd-sheet-enter-active .crd__panel,
.crd-sheet-leave-active .crd__panel {
  transition: transform 0.22s cubic-bezier(0.4, 0, 0.2, 1);
}
.crd-drawer-enter-from,
.crd-drawer-leave-to,
.crd-sheet-enter-from,
.crd-sheet-leave-to {
  opacity: 0;
}
.crd-drawer-enter-from .crd__panel,
.crd-drawer-leave-to .crd__panel {
  transform: translateX(24px);
}
.crd-sheet-enter-from .crd__panel,
.crd-sheet-leave-to .crd__panel {
  transform: translateY(32px);
}
</style>
