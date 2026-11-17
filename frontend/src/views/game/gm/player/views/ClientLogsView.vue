<!-- 客户端日志（Momoko）：玩家的游戏客户端上传的错误日志，每个游戏账号保留最新 5 份（从新到旧）。
     与「客户端日志」页是同一个列表（按这台服务器与这个玩家筛选）和同一个查看器：点一行读全文、在日志里查找、复制全部，
     行内复制摘要、下载 .log 与删除。 -->
<template>
  <div class="pv">
    <div class="pv__line">
      <span>{{ t('clientLogs.playerHint', { n: CLIENT_LOG_KEEP }) }}</span>
      <span class="pv__spacer" />
      <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('gmData.common.refresh')" @click="load" />
    </div>
    <div v-if="error" class="pv__error">{{ error }}</div>
    <ClientLogTable :items="items" :loading="loading" :empty-text="t('clientLogs.emptyPlayer')" @changed="load" />
  </div>
</template>

<script setup lang="ts">
import { ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listGmClientLogs } from '@/api/gameGm'
import type { GmClientLog } from '@/types/v1/game_gm'
import { getRequestErrorMessage } from '@/utils/request'
import { CLIENT_LOG_KEEP } from '@/views/game/clientLogs'
import ClientLogTable from '@/views/game/components/ClientLogTable.vue'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()

const items = shallowRef<GmClientLog[]>([])
const loading = ref(false)
const error = ref('')
let seq = 0

const load = async () => {
  const id = ++seq
  loading.value = true
  error.value = ''
  try {
    const { data } = await listGmClientLogs({ serverId: ctx.serverId.value, pid: page.pid.value, pageSize: 100 })
    if (id === seq) items.value = data.items || []
  } catch (e) {
    if (id === seq) {
      error.value = getRequestErrorMessage(e)
      items.value = []
    }
  } finally {
    if (id === seq) loading.value = false
  }
}
watch(page.version, () => void load(), { immediate: true })
</script>

<style scoped lang="scss">
@use '../shared/views';
</style>
