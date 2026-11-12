<!-- GM 运营 · 公告（?tab=notices）：配置域 notice（data/Notice.json）的编辑。
     只编辑真实字段：ID、分类、类型、标题、横幅、详情页地址、开始 / 结束展示时间、最后更新时间，以及每页条数。
     卡池公告由服务端按卡池自动生成（notice-<卡池 ID>），不在这个文件里。横幅从前端素材包选；右侧是游戏内列表预览。 -->
<template>
  <div class="gn game-skin">
    <PageHeader :title="t('gmConfig.notice.title')" :back="false" :theme-toggle="false">
      <template #description>{{ countLine }}</template>
      <template #actions>
        <UButton
          class="gmc-btn"
          color="neutral"
          variant="outline"
          icon="i-lucide-history"
          :aria-label="t('gmConfig.revisions.title')"
          :disabled="!state.info.value"
          @click="revisionsOpen = true"
        >
          <template v-if="!menuStore.isMobile">{{ t('gmConfig.revisions.title') }}</template>
        </UButton>
        <UButton class="gmc-btn" icon="i-lucide-plus" :aria-label="t('gmConfig.notice.create')" :disabled="!cfg" @click="openCreate()">
          <template v-if="!menuStore.isMobile">{{ t('gmConfig.notice.create') }}</template>
        </UButton>
      </template>
    </PageHeader>

    <EmptyState
      v-if="unsupported"
      icon="HOutline:PuzzlePieceIcon"
      :title="t('gmConfig.common.unsupportedTitle')"
      :description="t('gmConfig.common.unsupportedDesc', { game: ctx.gameName.value })"
    />
    <EmptyState v-else-if="state.error.value && !cfg" icon="HOutline:ExclamationTriangleIcon" :title="t('gmConfig.common.loadFailed')" :description="state.error.value">
      <template #action>
        <UButton class="gmc-btn" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="state.loading.value" @click="state.load()">
          {{ t('gmConfig.common.retry') }}
        </UButton>
      </template>
    </EmptyState>
    <div v-else-if="!cfg" class="gn__loading">{{ t('gmConfig.common.loading') }}</div>
    <template v-else>
      <div class="gn__body">
        <section class="gn__list" :aria-label="t('gmConfig.notice.listLabel')">
          <div class="gn__filters">
            <div class="gmc-chips" role="tablist" :aria-label="t('gmConfig.notice.category')">
              <button
                v-for="f in categoryChips"
                :key="f.key"
                type="button"
                role="tab"
                class="gmc-chip"
                :class="{ 'is-on': category === f.key }"
                :aria-selected="category === f.key"
                @click="category = f.key"
              >
                {{ f.label }} <span class="gmc-chip__n m-num">{{ f.n }}</span>
              </button>
            </div>
            <div class="gmc-chips" role="tablist" :aria-label="t('gmConfig.common.filterStatus')">
              <button
                v-for="f in statusChips"
                :key="f.key"
                type="button"
                role="tab"
                class="gmc-chip is-soft"
                :class="{ 'is-on': status === f.key }"
                :aria-selected="status === f.key"
                @click="status = f.key"
              >
                {{ f.label }} <span class="gmc-chip__n m-num">{{ f.n }}</span>
              </button>
            </div>
          </div>
          <button
            v-for="row in rows"
            :key="row.index"
            type="button"
            class="gn__row"
            :class="{ 'is-on': row.index === selectedIndex }"
            :aria-pressed="row.index === selectedIndex"
            @click="select(row.index)"
          >
            <GameImage v-if="row.image" :src="row.image" class="gn__thumb" width="112" height="52" />
            <span v-else class="gn__thumb gn__thumb--empty"><UIcon name="i-lucide-megaphone" /></span>
            <span class="gn__row-text">
              <span class="gn__row-title">{{ row.n.title || t('gmConfig.notice.untitled') }}</span>
              <span class="gn__row-meta">
                <span class="gmc-tag gmc-tag--g">{{ t(`gmConfig.notice.categories.${row.n.category}`) }}</span>
                <span class="gmc-tag">{{ t(`gmConfig.notice.types.${row.n.type}`) }}</span>
                <StatusPill :variant="noticeStatusVariant(row.status)" :label="t(`gmConfig.notice.status.${row.status}`)" />
                <span v-if="row.isNew" class="gmc-tag gmc-tag--g">{{ t('gmConfig.common.new') }}</span>
                <span v-else-if="row.changed" class="gmc-tag gmc-tag--warn">{{ t('gmConfig.common.changed') }}</span>
                <span v-if="row.bad" class="gmc-tag gmc-tag--bad">{{ t('gmConfig.notice.needsCheck') }}</span>
              </span>
              <span class="gn__row-span m-num">{{ row.span }}</span>
            </span>
          </button>
          <EmptyState
            v-if="!rows.length"
            icon="HOutline:MegaphoneIcon"
            :title="cfg.notices.length ? t('gmConfig.notice.emptyFilter') : t('gmConfig.notice.empty')"
            :description="cfg.notices.length ? '' : t('gmConfig.notice.emptyDesc')"
          />
          <div class="gn__settings">
            <label class="gn__pagesize">
              <span class="gmc-label">{{ t('gmConfig.notice.pageSize') }}</span>
              <input
                class="gmc-input gmc-input--sm m-num"
                type="number"
                min="1"
                step="1"
                :value="cfg.pageSize"
                :class="{ 'is-invalid': !(cfg.pageSize > 0) }"
                @change="setPageSize(($event.target as HTMLInputElement).valueAsNumber)"
              />
            </label>
            <p class="gmc-hint">{{ t('gmConfig.notice.gashaNoticeHint') }}</p>
          </div>
        </section>

        <section ref="mainRef" class="gn__main">
          <EmptyState v-if="!notice" icon="HOutline:MegaphoneIcon" :title="t('gmConfig.notice.pick')" />
          <div v-else class="gn__editor">
            <div class="gn__form">
              <div class="gn__form-head">
                <span class="gn__form-title">{{ t('gmConfig.notice.edit') }}</span>
                <StatusPill :variant="noticeStatusVariant(currentStatus)" :label="t(`gmConfig.notice.status.${currentStatus}`)" />
                <ActionMenu :items="menuItems" @select="onMenu" />
              </div>

              <div class="gmc-field">
                <span class="gmc-label">{{ t('gmConfig.notice.category') }}</span>
                <div class="gmc-seg" role="radiogroup" :aria-label="t('gmConfig.notice.category')">
                  <button
                    v-for="c in NOTICE_CATEGORIES"
                    :key="c"
                    type="button"
                    role="radio"
                    class="gmc-seg__btn"
                    :class="{ 'is-on': notice.category === c }"
                    :aria-checked="notice.category === c"
                    @click="notice.category = c"
                  >
                    {{ t(`gmConfig.notice.categories.${c}`) }}
                  </button>
                </div>
              </div>
              <div class="gmc-field">
                <span class="gmc-label">{{ t('gmConfig.notice.type') }}</span>
                <AppSelect v-model="notice.type" :options="typeOptions" />
              </div>
              <label class="gmc-field">
                <span class="gmc-label">{{ t('gmConfig.notice.titleField') }}</span>
                <input v-model="notice.title" class="gmc-input" />
              </label>
              <div class="gmc-field">
                <span class="gmc-label">{{ t('gmConfig.notice.banner') }}</span>
                <div class="gn__banner">
                  <GameImage v-if="currentBanner" :src="currentBanner" class="gn__banner-img" />
                  <span v-else class="gn__banner-img gn__banner-img--empty">{{ notice.bannerAssetId ? t('gmConfig.notice.bannerUnknown') : t('gmConfig.notice.noBanner') }}</span>
                  <span class="gn__banner-actions">
                    <UButton class="gmc-btn" size="sm" color="neutral" variant="soft" icon="i-lucide-search" @click="bannerOpen = true">{{ t('gmConfig.notice.pickBanner') }}</UButton>
                    <UButton class="gmc-btn" v-if="notice.bannerAssetId" size="sm" color="neutral" variant="ghost" icon="i-lucide-eraser" @click="notice.bannerAssetId = ''">
                      {{ t('gmConfig.notice.clearBanner') }}
                    </UButton>
                  </span>
                </div>
                <span class="gmc-hint">{{ t('gmConfig.notice.bannerHint') }}</span>
              </div>
              <label class="gmc-field">
                <span class="gmc-label">{{ t('gmConfig.notice.detailUrl') }}</span>
                <input v-model.trim="notice.detailUrl" class="gmc-input m-mono" type="url" spellcheck="false" placeholder="https://" />
                <span class="gmc-hint">{{ t('gmConfig.notice.detailUrlHint') }}</span>
              </label>
              <div class="gmc-grid">
                <div class="gmc-field">
                  <span class="gmc-label">{{ t('gmConfig.notice.start') }}</span>
                  <DateTimeField v-model="notice.startTime" :zone="clock.zone.value" :label="t('gmConfig.notice.start')" />
                </div>
                <div class="gmc-field">
                  <span class="gmc-label">
                    {{ t('gmConfig.notice.end') }}
                    <span class="gn__noend">
                      {{ t('gmConfig.notice.noEnd') }}
                      <AppSwitch :model-value="!notice.endTime" @update:model-value="setNoEnd" />
                    </span>
                  </span>
                  <DateTimeField
                    v-if="notice.endTime"
                    v-model="notice.endTime"
                    :zone="clock.zone.value"
                    :invalid="notice.endTime <= notice.startTime"
                    :label="t('gmConfig.notice.end')"
                  />
                  <span v-else class="gmc-readonly">{{ t('gmConfig.notice.noEndHint') }}</span>
                </div>
              </div>
              <div class="gmc-field">
                <span class="gmc-label">
                  {{ t('gmConfig.notice.lastUpdate') }}
                  <button type="button" class="gn__link" @click="touch">{{ t('gmConfig.notice.touch') }}</button>
                </span>
                <DateTimeField
                  v-model="notice.lastUpdateTime"
                  :zone="clock.zone.value"
                  :invalid="notice.lastUpdateTime < notice.startTime"
                  :label="t('gmConfig.notice.lastUpdate')"
                />
                <span class="gmc-hint">{{ t('gmConfig.notice.lastUpdateHint') }}</span>
              </div>

              <ul v-if="problems.length" class="gn__problems">
                <li v-for="(p, i) in problems" :key="i" :class="`is-${p.level}`">
                  <UIcon :name="p.level === 'bad' ? 'i-lucide-circle-x' : 'i-lucide-triangle-alert'" />
                  {{ t(`gmConfig.notice.problems.${p.key}`, p.params || {}) }}
                </li>
              </ul>

              <div class="gn__actions">
                <UButton class="gmc-btn" v-if="currentStatus === 'showing'" color="error" variant="soft" icon="i-lucide-power" @click="takeDown">{{ t('gmConfig.notice.takeDown') }}</UButton>
                <span class="gn__hint-inline">{{ t('gmConfig.notice.autosaveHint') }}</span>
              </div>
            </div>

            <NoticePreview class="gn__preview" :cfg="cfg" :current="notice" :now="clock.now.value" :zone="clock.zone.value" />
          </div>
          <ConfigDraftBar :state="state" :zone="clock.zone.value" :blockers="blockers" :group-label="groupLabel" :ref-label="refLabel" />
        </section>
      </div>
    </template>

    <FormDialog v-model="createOpen" :title="t('gmConfig.notice.create')" :width="480" :show-footer="false">
      <div class="gn__dialog game-skin" :data-game="ctx.game.value || undefined">
        <label class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.notice.titleField') }}</span>
          <input v-model.trim="createTitle" class="gmc-input" :class="{ 'is-invalid': !createTitle }" />
          <span class="gmc-hint">{{ t('gmConfig.notice.newTitleHint') }}</span>
        </label>
        <div class="gn__dialog-actions">
          <UButton class="gmc-btn" color="neutral" variant="ghost" @click="createOpen = false">{{ t('gmConfig.common.cancel') }}</UButton>
          <UButton class="gmc-btn" :disabled="!createTitle" @click="create">{{ t('gmConfig.common.ok') }}</UButton>
        </div>
      </div>
    </FormDialog>

    <AssetPickerDialog
      v-model="bannerOpen"
      :title="t('gmConfig.notice.pickBanner')"
      :sources="bannerSources"
      :selected="notice?.bannerAssetId"
      @pick="setBanner"
    />
    <ConfigRevisionsDrawer v-model="revisionsOpen" :state="state" :zone="clock.zone.value" :group-label="groupLabel" :ref-label="refLabel" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onActivated, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import type { NoticeConfig, NoticeEntry } from '@/games/gakumas/types'
import { Dialog } from '@/utils/dialog'
import { useGmContext } from '../../context'
import AssetPickerDialog from '../gasha/shared/AssetPickerDialog.vue'
import { bannerImage, gashaBannerSource, noticeBannerSource } from '../gasha/shared/assets'
import ConfigDraftBar from '../gasha/shared/ConfigDraftBar.vue'
import ConfigRevisionsDrawer from '../gasha/shared/ConfigRevisionsDrawer.vue'
import DateTimeField from '../gasha/shared/DateTimeField.vue'
import { deepEqual } from '../gasha/shared/diff'
import { useServerClock } from '../gasha/shared/serverClock'
import { formatShort } from '../gasha/shared/time'
import { useConfigDomain } from '../gasha/shared/useConfigDomain'
import NoticePreview from './NoticePreview.vue'
import {
  configProblems,
  NOTICE_CATEGORIES,
  NOTICE_STATUSES,
  NOTICE_TYPES,
  noticeProblems,
  noticeStatus,
  noticeStatusVariant,
  type NoticeStatus,
} from './notice'

defineOptions({ name: 'GmNoticesTab' })

const DOMAIN = 'notice'
const MINUTE = 60_000

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const menuStore = useMenuStore()
const ctx = useGmContext()
const clock = useServerClock(ctx)
const state = useConfigDomain<NoticeConfig>(DOMAIN)

const revisionsOpen = ref(false)
const bannerOpen = ref(false)
const unsupported = computed(() => !!ctx.game.value && ctx.game.value !== 'gakumas')
const cfg = computed(() => state.working.value)

// 用下标选中（新建的公告 ID 可以改，按 ID 选会丢）
const selectedIndex = ref(-1)
const notice = computed(() => (selectedIndex.value >= 0 ? cfg.value?.notices[selectedIndex.value] : undefined))
const publishedNotice = computed(() => (notice.value ? state.published.value?.notices.find((n) => n.id === notice.value!.id) : undefined))
const currentStatus = computed(() => (notice.value ? noticeStatus(notice.value, clock.now.value) : 'showing'))
const currentBanner = computed(() => (notice.value ? bannerImage(notice.value.bannerAssetId) : ''))

onMounted(async () => {
  if (unsupported.value) return
  await state.load()
  applyQuery()
})
onActivated(() => applyQuery())
watch(() => route.query, () => applyQuery())

// ?notice=<id> 打开某条公告；?new=1 新建（运营日历跳过来）
const applyQuery = () => {
  const c = cfg.value
  if (!c) return
  const wanted = route.query.notice
  if (typeof wanted === 'string') {
    const i = c.notices.findIndex((n) => n.id === wanted)
    if (i >= 0) selectedIndex.value = i
  }
  if (route.query.new === '1') {
    void router.replace({ query: { ...route.query, new: undefined } })
    openCreate()
  }
}

const category = ref<0 | 1 | 2 | 3>(0)
const status = ref<'all' | NoticeStatus>('all')
const statusOf = (n: NoticeEntry) => noticeStatus(n, clock.now.value)

const categoryChips = computed(() => {
  const list = cfg.value?.notices || []
  return [
    { key: 0 as const, label: t('gmConfig.common.all'), n: list.length },
    ...NOTICE_CATEGORIES.map((c) => ({ key: c, label: t(`gmConfig.notice.categories.${c}`), n: list.filter((n) => n.category === c).length })),
  ]
})
const statusChips = computed(() => {
  const list = (cfg.value?.notices || []).filter((n) => !category.value || n.category === category.value)
  return [
    { key: 'all' as const, label: t('gmConfig.notice.allStatus'), n: list.length },
    ...NOTICE_STATUSES.map((s) => ({ key: s, label: t(`gmConfig.notice.status.${s}`), n: list.filter((n) => statusOf(n) === s).length })),
  ]
})

const rows = computed(() => {
  const c = cfg.value
  if (!c) return []
  const zone = clock.zone.value
  const published = state.published.value?.notices || []
  return c.notices
    .map((n, index) => ({ n, index }))
    .filter(({ n }) => (!category.value || n.category === category.value) && (status.value === 'all' || statusOf(n) === status.value))
    .sort((a, b) => b.n.startTime - a.n.startTime)
    .map(({ n, index }) => {
      const before = published.find((p) => p.id === n.id)
      return {
        n,
        index,
        status: statusOf(n),
        image: bannerImage(n.bannerAssetId),
        span: `${n.startTime ? formatShort(n.startTime, zone) : '—'} → ${n.endTime ? formatShort(n.endTime, zone) : t('gmConfig.notice.noEnd')}`,
        isNew: !before,
        changed: !!before && !deepEqual(before, n),
        bad: noticeProblems(c, n).some((p) => p.level === 'bad'),
      }
    })
})

// 选中的公告不存在了（删除、重新加载）时选列表里的第一条
watch(
  () => cfg.value?.notices.length,
  () => {
    const list = cfg.value?.notices || []
    if (selectedIndex.value >= list.length || selectedIndex.value < 0) selectedIndex.value = rows.value[0]?.index ?? (list.length ? 0 : -1)
  },
  { immediate: true },
)

const countLine = computed(() => {
  const list = cfg.value?.notices || []
  const parts = [t('gmConfig.notice.count', { n: list.length }), t('gmConfig.notice.countShowing', { n: list.filter((n) => statusOf(n) === 'showing').length })]
  if (state.dirty.value) parts.push(t('gmConfig.gasha.countDraft', { n: state.changes.value.length }))
  return parts.join(' · ')
})

const typeOptions = computed(() => NOTICE_TYPES.map((v) => ({ label: t(`gmConfig.notice.types.${v}`), value: v })))
const problems = computed(() => (cfg.value && notice.value ? noticeProblems(cfg.value, notice.value) : []))

const blockers = computed(() => {
  const c = cfg.value
  if (!c) return []
  const out = configProblems(c).map((p) => t(`gmConfig.notice.problems.${p.key}`))
  for (const n of c.notices) {
    for (const p of noticeProblems(c, n)) {
      if (p.level === 'bad') out.push(`${n.title || t('gmConfig.notice.untitled')}：${t(`gmConfig.notice.problems.${p.key}`, p.params || {})}`)
    }
  }
  return out
})

const groupLabel = (path: string[]) => {
  const [section, id] = path
  if (section === 'notices' && id) {
    const n = cfg.value?.notices.find((x) => x.id === id) || state.published.value?.notices.find((x) => x.id === id)
    return `${t('gmConfig.notice.title')} · ${n?.title || t('gmConfig.notice.untitled')}`
  }
  if (section === 'pageSize') return t('gmConfig.notice.pageSize')
  return ''
}

// 配置改动里的公告引用显示公告标题
const refLabel = (collection: string, id: string) => {
  if (collection !== 'notices') return ''
  const n = cfg.value?.notices.find((x) => x.id === id) || state.published.value?.notices.find((x) => x.id === id)
  return n ? n.title || t('gmConfig.notice.untitled') : ''
}

const mainRef = ref<HTMLElement>()
const select = (index: number) => {
  selectedIndex.value = index
  if (menuStore.isMobile) void nextTick(() => mainRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
}

const setPageSize = (v: number) => {
  if (cfg.value) cfg.value.pageSize = Number.isFinite(v) && v > 0 ? Math.trunc(v) : 0
}

const setNoEnd = (on: boolean) => {
  const n = notice.value
  if (!n) return
  if (on) n.endTime = 0
  else {
    const before = publishedNotice.value
    n.endTime = before?.endTime && before.endTime > n.startTime ? before.endTime : n.startTime + 86_400_000
  }
}

const minuteNow = () => Math.floor(clock.now.value / MINUTE) * MINUTE

const touch = () => {
  const n = notice.value
  if (n) n.lastUpdateTime = Math.max(minuteNow(), n.startTime)
}

// 立即下线：结束时间改成现在（发布后生效）
const takeDown = () => {
  const n = notice.value
  if (!n) return
  const now = minuteNow()
  n.endTime = now > n.startTime ? now : n.startTime + MINUTE
}

const bannerSources = computed(() => [noticeBannerSource(t('gmConfig.assets.noticeBanner')), gashaBannerSource(t('gmConfig.assets.gashaBanner'))])
const setBanner = (assetId: string) => {
  if (notice.value) notice.value.bannerAssetId = assetId
}

// ---- 新建 / 复制 / 删除 ----
// 新公告只填标题；配置里的公告 ID 自动生成（gm-notice-<时间>，不会和服务端给卡池生成的 notice-<卡池> 重复），界面上不出现。
const createOpen = ref(false)
const createTitle = ref('')
const createFrom = ref<NoticeEntry>()
const openCreate = (from?: NoticeEntry) => {
  createTitle.value = from?.title ? t('gmConfig.gasha.copyName', { name: from.title }) : ''
  createFrom.value = from
  createOpen.value = true
}
const newNoticeId = (c: NoticeConfig) => {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  const base = `gm-notice-${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`
  let id = base
  for (let i = 2; c.notices.some((n) => n.id === id); i++) id = `${base}-${i}`
  return id
}
const create = () => {
  const c = cfg.value
  if (!c || !createTitle.value) return
  const id = newNoticeId(c)
  const start = minuteNow()
  const src = createFrom.value
  const entry: NoticeEntry = src
    ? { ...(JSON.parse(JSON.stringify(src)) as NoticeEntry), id, title: createTitle.value }
    : {
        id,
        title: createTitle.value,
        category: 1,
        type: 6,
        startTime: start,
        endTime: 0,
        lastUpdateTime: start,
        bannerAssetId: '',
        detailUrl: '',
      }
  c.notices.push(entry)
  createOpen.value = false
  category.value = 0
  status.value = 'all'
  select(c.notices.length - 1)
}

const menuItems = computed(() => [
  { key: 'copy', label: t('gmConfig.notice.copy'), icon: 'i-lucide-copy' },
  { key: 'delete', label: t('gmConfig.notice.delete'), icon: 'i-lucide-trash-2', danger: true },
])
const onMenu = (key: string) => {
  const n = notice.value
  if (!n) return
  if (key === 'copy') openCreate(n)
  if (key === 'delete') {
    Dialog.confirm({
      title: t('gmConfig.notice.deleteTitle', { name: n.title || t('gmConfig.notice.untitled') }),
      content: t('gmConfig.notice.deleteContent'),
      cancelText: t('gmConfig.common.cancel'),
      confirmText: t('gmConfig.notice.delete'),
      onConfirm: () => {
        const c = cfg.value
        if (!c) return
        const i = c.notices.indexOf(n)
        if (i >= 0) c.notices.splice(i, 1)
        selectedIndex.value = rows.value[0]?.index ?? -1
      },
    })
  }
}
</script>

<style lang="scss" src="../gasha/shared/form.scss"></style>
<style scoped lang="scss">
.gn {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-width: 0;
}
.gn__loading {
  padding: 40px 0;
  text-align: center;
  color: var(--el-text-color-secondary);
}
.gn__body {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-start;
}
.gn__list {
  flex: 1 1 380px;
  max-width: 480px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  overflow: hidden;
}
.gn__filters {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
}
.gn__row {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 12px 14px;
  border: 0;
  border-top: 1px solid var(--el-border-color-light);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gn__row:hover {
  background: var(--m-s2);
}
.gn__row.is-on {
  background: var(--m-gSoft, var(--m-pSoft));
  box-shadow: inset 3px 0 0 var(--m-g, var(--m-p));
}
.gn__row .gn__thumb {
  flex: none;
  width: 112px;
  height: 52px;
  border-radius: 12px;
  object-fit: cover;
}
.gn__thumb--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--m-s2);
  color: var(--el-text-color-secondary);
}
.gn__row-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.gn__row-title {
  font-size: 14px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gn__row-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}
.gn__row-span {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gn__settings {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  border-top: 1px solid var(--el-border-color-light);
}
.gn__pagesize {
  display: flex;
  align-items: center;
  gap: 10px;
}
.gn__pagesize .gmc-input {
  width: 96px;
}
.gn__main {
  flex: 2 1 640px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gn__editor {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  padding: 20px 24px;
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.gn__form {
  flex: 1 1 300px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gn__form-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.gn__form-title {
  flex: 1;
  font-size: 20px;
  font-weight: 800;
}
.gn__preview {
  flex: 0 0 300px;
  max-width: 100%;
  margin: 0 auto;
}
.gn__banner {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}
.gn__banner .gn__banner-img {
  width: 184px;
  height: 64px;
  border-radius: 14px;
  object-fit: cover;
}
.gn__banner-img--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 10px;
  text-align: center;
  background: var(--m-s2);
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.gn__banner-actions {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gn__noend {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.gn__link {
  margin-left: auto;
  min-height: 40px;
  border: 0;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}
.gn__problems {
  list-style: none;
  margin: 0;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--m-s2);
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
}
.gn__problems li {
  display: flex;
  align-items: center;
  gap: 8px;
}
.gn__problems li.is-bad {
  color: var(--m-bad);
}
.gn__problems li.is-warn {
  color: var(--m-warn);
}
.gn__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.gn__hint-inline {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gn__dialog {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.gn__dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
@media (max-width: 991px) {
  .gn__list {
    max-width: none;
  }
  .gn__editor {
    padding: 16px 14px;
  }
  .gn__row .gn__thumb {
    width: 88px;
    height: 42px;
  }
}
</style>
