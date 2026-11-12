<!-- 删除服务器：托管服务器的托管进程（应用实例）一起停止并删除，不会留在应用列表里。 -->
<template>
  <FormDialog
    :model-value="modelValue"
    :title="t('gameServers.delete.title')"
    :width="520"
    :loading="busy"
    :confirm-text="t('gameServers.delete.confirm')"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
    @confirm="run"
  >
    <div class="gsd">
      <p class="gsd__desc">{{ t('gameServers.delete.desc', { name: server?.brief?.name || '' }) }}</p>
      <p v-if="hosted" class="gsd__desc">{{ t('gameServers.delete.hostedNote') }}</p>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { deleteGameServer } from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import type { GameServerInfo } from '@/types/v1/game_server'

const props = defineProps<{ modelValue: boolean; server: GameServerInfo | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; deleted: [id: string] }>()
const { t } = useI18n()
const fb = useFeedback()
const busy = ref(false)
const hosted = computed(() => props.server?.brief?.mode === 'hosted')

const run = async () => {
  const id = props.server?.brief?.id
  if (!id) return
  busy.value = true
  try {
    await deleteGameServer(id)
    fb.success(t('gameServers.delete.done'))
    emit('deleted', id)
    emit('update:modelValue', false)
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    busy.value = false
  }
}
</script>

<style scoped lang="scss">
.gsd {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.gsd__desc {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.6;
  color: var(--el-text-color-regular);
}
.gsd__opt-text small {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
</style>
