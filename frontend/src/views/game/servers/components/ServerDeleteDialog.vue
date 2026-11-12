<!-- 删除服务器：托管服务器可以选择同时删除应用实例。 -->
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
      <label v-if="hosted" class="gsd__opt">
        <span class="gsd__opt-text">
          <b>{{ t('gameServers.delete.instances') }}</b>
          <small>{{ t('gameServers.delete.instancesDesc') }}</small>
        </span>
        <AppSwitch v-model="deleteInstances" :aria-label="t('gameServers.delete.instances')" />
      </label>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { deleteGameServer } from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import type { GameServerInfo } from '@/types/v1/game_server'

const props = defineProps<{ modelValue: boolean; server: GameServerInfo | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; deleted: [id: string] }>()
const { t } = useI18n()
const fb = useFeedback()
const busy = ref(false)
const deleteInstances = ref(false)
const hosted = computed(() => props.server?.brief?.mode === 'hosted')

watch(
  () => props.modelValue,
  (open) => {
    if (open) deleteInstances.value = false
  },
)

const run = async () => {
  const id = props.server?.brief?.id
  if (!id) return
  busy.value = true
  try {
    await deleteGameServer(id, hosted.value && deleteInstances.value)
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
.gsd__opt {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 56px;
  padding: 10px 14px;
  border-radius: var(--app-radius);
  background: var(--m-badSoft);
  cursor: pointer;
}
.gsd__opt-text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 0.875rem;
}
.gsd__opt-text small {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
</style>
