<!-- 回忆详情弹窗（回忆、培育记录、偶像卡详情里点开）：GakumasMemoryDetail 展示回忆详情与培育记录，
     回忆照片经 GetMemoryPhoto 读取（同一服务器同一张照片只读一次）；培育记录里编成的自己的回忆可以打开，「返回」回到之前的回忆。
     「设为援助回忆」先关闭弹窗，再用操作对话框执行 memory.setRental（试运行、确认与 GM 记录同其他操作）。 -->
<template>
  <FormDialog :model-value="!!memoryId" :title="t('gmData.memoryDetail.title')" :width="720" @update:model-value="onToggle">
    <div v-if="error" class="pv__error">{{ error }}</div>
    <div v-else-if="!detail" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <GakumasMemoryDetail
      v-else
      :detail="detail"
      :game="ctx.game.value"
      :lookup="lookup"
      :time="dateTime"
      :photo="photo"
      @open-memory="openMemory"
    />
    <template #footer="{ close }">
      <UButton v-if="history.length" color="neutral" variant="soft" @click="back">{{ t('gmData.memoryDetail.back') }}</UButton>
      <UButton color="neutral" variant="soft" @click="close">{{ t('gmData.common.close') }}</UButton>
      <UButton v-if="canSetRental" color="primary" :disabled="!detail || detail.lent" @click="setRental">
        {{ detail?.lent ? t('gmData.memoryDetail.lent') : t('gmData.memoryDetail.setRental') }}
      </UButton>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, shallowRef, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getMemoryPhoto, getPlayerMemory } from '@/api/gakumas'
import GakumasMemoryDetail from '@/components/game/GakumasMemoryDetail.vue'
import FormDialog from '@/components/ui/FormDialog.vue'
import { imageDataUrl } from '@/games/gakumas/display'
import type { MemoryDetail } from '@/types/v1/gakumas/types'
import { getRequestErrorMessage } from '@/utils/request'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import { useGmFormat } from './format'
import { useCatalogLookup } from './lookup'

const props = defineProps<{ memoryId: string }>()
const emit = defineEmits<{ 'update:memoryId': [value: string] }>()

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const lookup = useCatalogLookup()
const { dateTime } = useGmFormat()

const detail = shallowRef<MemoryDetail>()
const error = ref('')
/** 打开编成回忆之前看的回忆（「返回」用） */
const history = ref<string[]>([])
let seq = 0

// 回忆照片：同一服务器的同一张照片只读一次
const photos = new Map<string, Promise<string>>()
const photo = (path: string) => {
  const key = `${ctx.serverId.value}|${path}`
  let p = photos.get(key)
  if (!p) {
    p = getMemoryPhoto({ serverId: ctx.serverId.value, path }).then(imageDataUrl)
    p.catch(() => photos.delete(key))
    photos.set(key, p)
  }
  return p
}

const openMemory = (id: string) => {
  if (!id || id === props.memoryId) return
  history.value = [...history.value, props.memoryId]
  emit('update:memoryId', id)
}

const back = () => {
  const list = [...history.value]
  const prev = list.pop()
  history.value = list
  if (prev) emit('update:memoryId', prev)
}

const load = async () => {
  const id = props.memoryId
  const n = ++seq
  detail.value = undefined
  error.value = ''
  if (!id) return
  try {
    const d = await getPlayerMemory({ serverId: ctx.serverId.value, pid: page.pid.value, memoryId: id })
    if (n === seq) detail.value = d
  } catch (e) {
    if (n === seq) error.value = getRequestErrorMessage(e)
  }
}
watch(() => props.memoryId, () => void load(), { immediate: true })

const canSetRental = computed(() => !!ctx.operation('memory.setRental'))

const onToggle = (open: boolean) => {
  if (open) return
  history.value = []
  emit('update:memoryId', '')
}

const setRental = () => {
  const id = props.memoryId
  history.value = []
  emit('update:memoryId', '')
  page.run('memory.setRental', { memoryId: id })
}
</script>

<style scoped lang="scss">
@use './views';
</style>
