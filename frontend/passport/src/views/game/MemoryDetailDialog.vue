<!-- 回忆详情弹窗（培育记录、偶像卡详情里点开）：GakumasMemoryDetail 展示回忆详情与培育记录，
     回忆照片经 GetMemoryPhoto 读取（同一张只读一次）；培育记录里编成的自己的回忆可以打开，「返回」回到之前的回忆。
     「设为援助回忆」与游戏里的资料设置相同（立即生效，原来的援助回忆不再出借），成功后通知列表刷新。 -->
<template>
  <PDialog :model-value="!!memoryId" :title="t('game.memory.title')" :width="720" @update:model-value="onToggle">
    <p v-if="error" class="pp-hint md__err">{{ error }}</p>
    <p v-else-if="!detail" class="pp-hint">{{ t('game.memory.loading') }}</p>
    <GakumasMemoryDetail
      v-else
      :detail="detail"
      :game="ctx.game.value"
      :lookup="lookup"
      :time="fmtTime"
      :photo="photo"
      @open-memory="openMemory"
    />
    <template #footer>
      <PButton v-if="history.length" variant="ghost" @click="back">{{ t('game.memory.back') }}</PButton>
      <PButton :disabled="!detail || detail.lent" :loading="saving" @click="setRental">
        {{ detail?.lent ? t('game.memory.lent') : t('game.memory.setRental') }}
      </PButton>
    </template>
  </PDialog>
</template>

<script setup lang="ts">
import { ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import GakumasMemoryDetail from '@/components/game/GakumasMemoryDetail.vue'
import { imageDataUrl, type GakumasDetailLookup } from '@/games/gakumas/display'
import type { MemoryDetail } from '@/types/v1/gakumas/types'
import PButton from '@passport/components/PButton.vue'
import PDialog from '@passport/components/PDialog.vue'
import { getMemoryPhoto, getMyMemory, setMyRentalMemory } from '@passport/api/gakumas'
import { useUiStore } from '@passport/stores/ui'
import { errorText, fmtTime } from '@passport/utils/format'
import { useGameContext } from './context'

const props = defineProps<{ memoryId: string }>()
const emit = defineEmits<{ 'update:memoryId': [value: string]; changed: [] }>()

const { t } = useI18n()
const ctx = useGameContext()
const ui = useUiStore()
const lookup: GakumasDetailLookup = { name: ctx.nameOf, entry: ctx.entry }

const detail = shallowRef<MemoryDetail>()
const error = ref('')
const saving = ref(false)
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
    const d = await getMyMemory({ serverId: ctx.serverId.value, memoryId: id })
    if (n === seq) detail.value = d
  } catch (e) {
    if (n === seq) error.value = errorText(e)
  }
}
watch(() => props.memoryId, () => void load(), { immediate: true })

const onToggle = (open: boolean) => {
  if (open) return
  history.value = []
  emit('update:memoryId', '')
}

const setRental = async () => {
  const d = detail.value
  if (!d || saving.value) return
  saving.value = true
  try {
    await setMyRentalMemory({ serverId: ctx.serverId.value, memoryId: d.memoryId })
    detail.value = { ...d, lent: true }
    ui.toast(t('game.memory.rentalDone'), 'ok')
    emit('changed')
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.md__err {
  color: var(--m-bad);
}
</style>
