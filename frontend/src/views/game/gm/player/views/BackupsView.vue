<!-- 存档备份（MGP backups，从新到旧）：种类（手动 / 指令前自动 / 回档前自动）、备注、时间；
     有 archive.backup 时可「备份当前存档」，有 archive.restore 时每条可「回档」（危险指令，走通用指令对话框）。 -->
<template>
  <div class="pv">
    <div class="pv__line">
      <span v-if="list.total.value" class="m-num">{{ t('gmData.backups.total', { n: list.total.value }) }}</span>
      <span class="pv__spacer" />
      <UButton v-if="backupCommand" color="primary" variant="soft" icon="i-lucide-archive" @click="page.run(backupCommand.id)">
        {{ backupCommand.title }}
      </UButton>
    </div>

    <div v-if="list.error.value" class="pv__error">{{ list.error.value }}</div>
    <div v-if="list.loading.value && !list.items.value.length" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState v-else-if="!list.items.value.length" icon="HOutline:ArchiveBoxIcon" :title="t('gmData.backups.empty')" />
    <div v-else class="pv__mcards">
      <article v-for="b in list.items.value" :key="b.id" class="pv__mcard bk__row">
        <div class="pv__mbody">
          <span class="pv__mline">
            <StatusPill :variant="b.kind === 1 ? 'primary' : 'neutral'" :label="t(`gmData.backups.kind.${b.kind}`)" />
            <span class="pv__strong pv__ellipsis">{{ b.note || '—' }}</span>
          </span>
          <span class="pv__dim m-num">{{ dateTime(b.createdAt) }}</span>
        </div>
        <UButton v-if="restoreCommand" size="sm" color="error" variant="soft" @click="page.run(restoreCommand.id, { backupId: b.id })">
          {{ t('gmData.backups.restore') }}
        </UButton>
      </article>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmPlayerView, mgpQuery } from '@/api/gameGm'
import type { ListPage, ManifestView } from '@/games/mgp'
import type { Backup } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import { useGmFormat } from '../shared/format'
import { usePagedList } from '../shared/paging'

const props = defineProps<{ view?: ManifestView }>()

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const { dateTime } = useGmFormat()

const list = usePagedList<Backup>(
  (q) => getGmPlayerView<ListPage<Backup>>(ctx.serverId.value, page.pid.value, props.view?.id || 'backups', mgpQuery({ ...q })),
  { pageSize: 200, append: true },
)
watch(page.version, () => void list.reload(), { immediate: true })

const backupCommand = computed(() => ctx.command('archive.backup'))
const restoreCommand = computed(() => ctx.command('archive.restore'))
</script>

<style scoped lang="scss">
@use '../shared/views';
.bk__row {
  align-items: center;
}
</style>
