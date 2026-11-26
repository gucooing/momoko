<!-- 编辑一条公告（运营管理里点日历上的公告直接打开）：左边表单（分类、类型、标题、横幅、详情页、展示时间、最后更新时间），右边游戏内列表预览。 -->
<template>
  <ItemDialog
    :model-value="modelValue"
    :title="(notice || publishedNotice)?.title || t('gmConfig.notice.untitled')"
    :width="1100"
    :state="notice ? 'ok' : publishedNotice ? 'removed' : 'missing'"
    :status="notice ? periodStatus(notice.startTime, notice.endTime, clock.now.value) : undefined"
    :draft="draftOf(publishedNotice, notice)"
    :copy-label="t('gmConfig.ops.copy')"
    @update:model-value="emit('update:modelValue', $event)"
    @copy="copy"
    @delete="remove"
    @restore="restore"
  >
    <div v-if="notice && cfg" class="ned">
      <div class="ned__form">
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
          <ImageField v-model="notice.bannerAssetId" :title="t('gmConfig.notice.pickBanner')" :sources="bannerSources" />
        </div>
        <label class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.notice.detailUrl') }}</span>
          <input v-model.trim="notice.detailUrl" class="gmc-input m-mono" type="url" spellcheck="false" placeholder="https://" />
          <span class="gmc-hint">{{ t('gmConfig.notice.detailUrlHint') }}</span>
        </label>
        <div class="gmc-grid">
          <div class="gmc-field">
            <span class="gmc-label">{{ t('gmConfig.notice.start') }}</span>
            <DateTimeField v-model="notice.startTime" :zone="zone" :label="t('gmConfig.notice.start')" />
          </div>
          <div class="gmc-field">
            <span class="gmc-label">
              {{ t('gmConfig.notice.end') }}
              <span class="gmc-label__aside ned__noend">
                {{ t('gmConfig.notice.noEnd') }}
                <AppSwitch :model-value="!notice.endTime" @update:model-value="setNoEnd" />
              </span>
            </span>
            <DateTimeField v-if="notice.endTime" v-model="notice.endTime" :zone="zone" :invalid="notice.endTime <= notice.startTime" :label="t('gmConfig.notice.end')" />
            <span v-else class="gmc-readonly">{{ t('gmConfig.notice.noEndHint') }}</span>
          </div>
        </div>
        <div class="gmc-field">
          <span class="gmc-label">
            {{ t('gmConfig.notice.lastUpdate') }}
            <button type="button" class="ned__link" @click="touch">{{ t('gmConfig.notice.touch') }}</button>
          </span>
          <DateTimeField v-model="notice.lastUpdateTime" :zone="zone" :invalid="notice.lastUpdateTime < notice.startTime" :label="t('gmConfig.notice.lastUpdate')" />
          <span class="gmc-hint">{{ t('gmConfig.notice.lastUpdateHint') }}</span>
        </div>
        <label class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.notice.pageSize') }}</span>
          <input class="gmc-input m-num" type="number" min="1" step="1" :value="cfg.pageSize" @change="cfg.pageSize = Math.max(1, intOf($event))" />
          <span class="gmc-hint">{{ t('gmConfig.notice.pageSizeHint') }}</span>
        </label>
        <ul v-if="problems.length" class="gmc-problems">
          <li v-for="(p, i) in problems" :key="i" :class="`is-${p.level}`">
            <UIcon :name="p.level === 'bad' ? 'i-lucide-circle-x' : 'i-lucide-triangle-alert'" />
            {{ t(`gmConfig.notice.problems.${p.key}`, p.params || {}) }}
          </li>
        </ul>
      </div>
      <NoticePreview :cfg="cfg" :current="notice" :now="clock.now.value" :zone="zone" />
    </div>
    <template #actions>
      <UButton v-if="notice && noticeStatus(notice, clock.now.value) === 'showing'" class="gmc-btn" color="error" variant="soft" icon="i-lucide-power" @click="takeDown">
        {{ t('gmConfig.notice.takeDown') }}
      </UButton>
    </template>
  </ItemDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useGmContext } from '../../../context'
import type { ItemRef } from '../calendar'
import { useOpsDomains } from '../domains'
import { gashaBannerSource, noticeBannerSource } from '../shared/assets'
import DateTimeField from '../shared/DateTimeField.vue'
import { cloneJson, draftOf, intOf, periodStatus } from '../shared/entry'
import ImageField from '../shared/ImageField.vue'
import ItemDialog from '../shared/ItemDialog.vue'
import { useServerClock } from '../shared/serverClock'
import NoticePreview from './NoticePreview.vue'
import { addNotice, NOTICE_CATEGORIES, NOTICE_TYPES, noticeProblems, noticeStatus } from './notice'

const props = defineProps<{ modelValue: boolean; noticeId: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; open: [ref: ItemRef] }>()

const MINUTE = 60_000

const { t } = useI18n()
const ctx = useGmContext()
const clock = useServerClock(ctx)
const zone = computed(() => clock.zone.value)
const state = useOpsDomains().notice.state

const cfg = computed(() => state.working.value)
const notice = computed(() => cfg.value?.notices.find((n) => n.id === props.noticeId))
const publishedNotice = computed(() => state.published.value?.notices.find((n) => n.id === props.noticeId))
const problems = computed(() => (cfg.value && notice.value ? noticeProblems(cfg.value, notice.value) : []))
const typeOptions = computed(() => NOTICE_TYPES.map((v) => ({ label: t(`gmConfig.notice.types.${v}`), value: v })))
const bannerSources = computed(() => [noticeBannerSource(t('gmConfig.assets.noticeBanner')), gashaBannerSource(t('gmConfig.assets.gashaBanner'))])

const minuteNow = () => Math.floor(clock.now.value / MINUTE) * MINUTE
const setNoEnd = (on: boolean) => {
  const n = notice.value
  if (!n) return
  if (on) n.endTime = 0
  else n.endTime = publishedNotice.value?.endTime && publishedNotice.value.endTime > n.startTime ? publishedNotice.value.endTime : n.startTime + 86_400_000
}
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

const copy = () => {
  const n = notice.value
  if (n && cfg.value) emit('open', { kind: 'notice', id: addNotice(cfg.value, clock.now.value, t('gmConfig.ops.copyName', { name: n.title }), n).id })
}
const remove = () => {
  const c = cfg.value
  const i = c && notice.value ? c.notices.indexOf(notice.value) : -1
  if (c && i >= 0) c.notices.splice(i, 1)
}
const restore = () => {
  if (cfg.value && publishedNotice.value) cfg.value.notices.push(cloneJson(publishedNotice.value))
}
</script>

<style lang="scss" src="../shared/form.scss"></style>
<style scoped lang="scss">
.ned {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 360px;
  gap: 14px;
  align-items: start;
}
.ned__form {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}
.ned__noend {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.ned__link {
  margin-left: auto;
  border: 0;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
}
@media (max-width: 991px) {
  .ned {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
