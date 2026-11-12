<!-- 文件：服务端目录（复用应用实例的文件管理组件，接口为游戏服务器的同构文件接口）。 -->
<template>
  <div class="fs">
    <FileBrowser :key="id" :scope="{ kind: 'instance', id }" :client="client" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import FileBrowser from '@/components/file/FileBrowser.vue'
import type { GameAdapter, GameServerInfo } from '@/types/v1/game_server'
import { createGameServerFileClient } from '../../fileClient'

const props = defineProps<{ server: GameServerInfo; adapter?: GameAdapter | null }>()
const id = computed(() => props.server.brief?.id || '')
const client = computed(() => createGameServerFileClient(id.value))
</script>

<style scoped lang="scss">
.fs {
  min-height: 560px;
  display: flex;
  flex-direction: column;
}
.fs > :deep(*) {
  flex: 1;
}
</style>
