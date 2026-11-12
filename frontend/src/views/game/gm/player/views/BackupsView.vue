<!-- 存档备份（从新到旧）：种类（手动 / 操作前自动 / 回档前自动）、备注、时间；
     可「备份当前存档」（archive.backup），每条可「回档」（archive.restore，危险操作，走操作对话框）。 -->
<template>
  <div class="pv">
    <div class="pv__line">
      <span v-if="backups.length" class="m-num">{{ t('gmData.backups.total', { n: backups.length }) }}</span>
      <span class="pv__spacer" />
      <UButton v-if="backupOperation" color="primary" variant="soft" icon="i-lucide-archive" @click="page.run(backupOperation.id)">
        {{ backupOperation.title }}
      </UButton>
    </div>

    <div v-if="error" class="pv__error">{{ error }}</div>
    <div v-if="loading && !backups.length" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState v-else-if="!backups.length" icon="HOutline:ArchiveBoxIcon" :title="t('gmData.backups.empty')" />
    <div v-else class="pv__mcards">
      <article v-for="b in backups" :key="b.id" class="pv__mcard bk__row">
        <div class="pv__mbody">
          <span class="pv__mline">
            <StatusPill :variant="b.kind === 1 ? 'primary' : 'neutral'" :label="t(`gmData.backups.kind.${b.kind}`)" />
            <span class="pv__strong pv__ellipsis">{{ b.note || '—' }}</span>
          </span>
          <span class="pv__dim m-num">{{ dateTime(b.createdAt) }}</span>
        </div>
        <UButton v-if="restoreOperation" size="sm" color="error" variant="soft" @click="page.run(restoreOperation.id, { backupId: b.id })">
          {{ t('gmData.backups.restore') }}
        </UButton>
      </article>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listPlayerBackups } from '@/api/gakumas'
import type { Backup } from '@/games/gakumas/types'
import { getRequestErrorMessage } from '@/utils/request'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import { useGmFormat } from '../shared/format'

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const { dateTime } = useGmFormat()

const backups = shallowRef<Backup[]>([])
const loading = ref(false)
const error = ref('')
let seq = 0

const load = async () => {
  const id = ++seq
  loading.value = true
  error.value = ''
  try {
    const data = await listPlayerBackups(ctx.serverId.value, page.pid.value)
    if (id === seq) backups.value = data.backups
  } catch (e) {
    if (id === seq) {
      error.value = getRequestErrorMessage(e)
      backups.value = []
    }
  } finally {
    if (id === seq) loading.value = false
  }
}
watch(page.version, () => void load(), { immediate: true })

const backupOperation = computed(() => ctx.operation('archive.backup'))
const restoreOperation = computed(() => ctx.operation('archive.restore'))
</script>

<style scoped lang="scss">
@use '../shared/views';
.bk__row {
  align-items: center;
}
</style>
