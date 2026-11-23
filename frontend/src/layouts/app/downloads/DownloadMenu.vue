<!-- 顶栏「下载」：服务器有下载记录时才出现，角标是进行中的下载数；点开是下载列表。 -->
<template>
  <AppDropdown v-if="downloads.length > 0" align="end" :width="380">
    <template #trigger>
      <AppIconButton
        icon="HOutline:ArrowDownTrayIcon"
        :label="active > 0 ? t('downloads.titleActive', { n: active }) : t('downloads.title')"
        :box="44"
        :badge="active || undefined"
      />
    </template>
    <template #default>
      <DownloadList />
    </template>
  </AppDropdown>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import DownloadList from './DownloadList.vue'
import { useDownloads } from './useDownloads'

const { t } = useI18n()
const { downloads, active } = useDownloads()
</script>
