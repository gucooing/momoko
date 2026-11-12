<!-- 更换 MGP 密钥：点「生成」在前端生成新的 Key ID 与密钥（可复制）。
     托管：Momoko 写入服务端配置，运行中可选立即重启；仅 GM：保存到 Momoko，再把它写进服务端配置。 -->
<template>
  <FormDialog
    :model-value="modelValue"
    :title="t('gameServers.rotate.title')"
    :width="560"
    :loading="busy"
    :confirm-text="t('gameServers.rotate.confirm')"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
    @confirm="run"
  >
    <div class="gsr">
      <p class="gsr__desc">{{ hosted ? t('gameServers.rotate.desc') : t('gameServers.rotate.remoteDesc') }}</p>
      <MgpKeyGenerator
        v-model="key"
        :current-key-id="server?.keyId"
        :hint="hosted ? t('gameServers.key.hostedHint') : t('gameServers.key.remoteHint')"
      />
      <label v-if="hosted" class="gsr__opt">
        <span class="gsr__opt-text">
          <b>{{ t('gameServers.rotate.restart') }}</b>
          <small>{{ t('gameServers.rotate.restartHint') }}</small>
        </span>
        <AppSwitch v-model="restart" :aria-label="t('gameServers.rotate.restart')" />
      </label>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { rotateGameServerKey, updateGameServer } from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import type { GameServerInfo } from '@/types/v1/game_server'
import type { MgpKey } from '../mgpKey'
import MgpKeyGenerator from './MgpKeyGenerator.vue'

const props = defineProps<{ modelValue: boolean; server: GameServerInfo | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; done: [info: GameServerInfo] }>()
const { t } = useI18n()
const fb = useFeedback()
const busy = ref(false)
const restart = ref(true)
const key = ref<MgpKey | null>(null)
const hosted = computed(() => props.server?.brief?.mode === 'hosted')

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    restart.value = true
    key.value = null
  },
)

const run = async () => {
  const s = props.server
  const id = s?.brief?.id
  if (!s || !id) return
  if (!key.value) {
    fb.warning(t('gameServers.key.needGenerate'))
    return
  }
  busy.value = true
  try {
    if (hosted.value) {
      const { data } = await rotateGameServerKey(id, restart.value, key.value)
      fb.success(t(data.restarted ? 'gameServers.rotate.doneRestart' : 'gameServers.rotate.done', { keyId: data.keyId }))
      if (data.info) emit('done', data.info)
    } else {
      const { data } = await updateGameServer({
        id,
        name: s.brief?.name || '',
        tags: s.brief?.tags || [],
        cover: s.brief?.cover || '',
        remark: s.remark,
        mgpEndpoint: s.mgpEndpoint,
        keyId: key.value.keyId,
        secret: key.value.secret,
        passportVisibility: s.passportVisibility,
        passportInviteCode: s.passportInviteCode,
        timezone: s.timezone,
      })
      fb.success(t('gameServers.rotate.done', { keyId: key.value.keyId }))
      emit('done', data)
    }
    emit('update:modelValue', false)
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    busy.value = false
  }
}
</script>

<style scoped lang="scss">
.gsr {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.gsr__desc {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.6;
  color: var(--el-text-color-regular);
}
.gsr__opt {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
}
.gsr__opt-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gsr__opt-text small {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
</style>
