<!-- 编辑服务器：名称、标签、备注、封面（从游戏素材里选横版卡面：搜索 + 最多显示 24 张，图片懒加载）。 -->
<template>
  <FormDialog
    :model-value="modelValue"
    :title="t('gameServers.edit.title')"
    :width="620"
    :loading="saving"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
    @confirm="save"
  >
    <div class="gse">
      <label class="gse__field">
        <span>{{ t('gameServers.edit.name') }}</span>
        <UInput v-model="form.name" size="lg" :maxlength="64" />
      </label>
      <div class="gse__field">
        <span>{{ t('gameServers.edit.tags') }} <small>{{ t('gameServers.edit.tagsHint') }}</small></span>
        <TagInput v-model="form.tags" :label="t('gameServers.edit.tags')" />
      </div>
      <label class="gse__field">
        <span>{{ t('gameServers.edit.remark') }}</span>
        <UTextarea v-model="form.remark" :rows="2" :maxlength="500" />
      </label>
      <div class="gse__field">
        <span>{{ t('gameServers.edit.cover') }} <small>{{ t('gameServers.edit.coverHint') }}</small></span>
        <UInput v-model="coverQuery" icon="i-lucide-search" :placeholder="t('gameServers.edit.coverSearch')" />
        <div class="gse__covers" :data-game="game">
          <button type="button" class="gse__cover gse__cover--none" :class="{ 'is-on': !form.cover }" @click="form.cover = ''">
            {{ t('gameServers.edit.coverNone') }}
          </button>
          <button
            v-for="a in shownCovers"
            :key="a.key"
            type="button"
            class="gse__cover"
            :class="{ 'is-on': form.cover === a.key }"
            :title="a.key"
            :aria-pressed="form.cover === a.key"
            @click="form.cover = a.key"
          >
            <GameImage :src="a.url" :alt="a.key" />
          </button>
        </div>
        <span v-if="moreCovers > 0" class="gse__hint">{{ t('gameServers.edit.coverMore', { n: moreCovers }) }}</span>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listGameAssets, type GameAssetEntry } from '@/games'
import { updateGameServer } from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import type { GameServerInfo } from '@/types/v1/game_server'
import TagInput from './TagInput.vue'

const props = defineProps<{ modelValue: boolean; server: GameServerInfo | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [info: GameServerInfo] }>()
const { t } = useI18n()
const fb = useFeedback()

const form = reactive({ name: '', tags: [] as string[], remark: '', cover: '' })
const saving = ref(false)
const coverQuery = ref('')
const covers = ref<GameAssetEntry[]>([])
const game = computed(() => props.server?.brief?.game || '')

watch(
  () => [props.modelValue, props.server] as const,
  async ([open, s]) => {
    if (!open || !s) return
    form.name = s.brief?.name || ''
    form.tags = [...(s.brief?.tags || [])]
    form.remark = s.remark || ''
    form.cover = s.brief?.cover || ''
    coverQuery.value = ''
    // 只加载这一组的索引（很小）；图片按需懒加载
    const list = await listGameAssets(game.value, 'cards/idol')
    covers.value = list.filter((a) => a.key.endsWith('-land'))
  },
  { immediate: true },
)

const LIMIT = 24
const matched = computed(() => {
  const q = coverQuery.value.trim().toLowerCase()
  return q ? covers.value.filter((a) => a.key.toLowerCase().includes(q)) : covers.value
})
const shownCovers = computed(() => matched.value.slice(0, LIMIT))
const moreCovers = computed(() => Math.max(0, matched.value.length - LIMIT))

const save = async () => {
  const s = props.server
  if (!s?.brief) return
  if (!form.name.trim()) {
    fb.warning(t('gameServers.edit.nameRequired'))
    return
  }
  saving.value = true
  try {
    const { data } = await updateGameServer({
      id: s.brief.id,
      name: form.name.trim(),
      tags: form.tags,
      cover: form.cover,
      remark: form.remark,
      mgpEndpoint: s.mgpEndpoint,
      keyId: s.keyId,
      secret: '',
      passportVisibility: s.passportVisibility,
      passportInviteCode: s.passportInviteCode,
      timezone: s.timezone,
    })
    fb.success(t('gameServers.common.saved'))
    emit('saved', data)
    emit('update:modelValue', false)
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped lang="scss">
.gse {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gse__field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.gse__field > span {
  font-size: 0.8125rem;
  font-weight: 700;
}
.gse__field small {
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.gse__covers {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(118px, 1fr));
  gap: 8px;
  max-height: 260px;
  overflow-y: auto;
}
.gse__cover {
  height: 66px;
  padding: 0;
  border-radius: var(--app-radius-sm);
  border: 2px solid transparent;
  background: var(--el-fill-color-light);
  overflow: hidden;
  cursor: pointer;
}
.gse__cover :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.gse__cover.is-on {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.gse__cover--none {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gse__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
</style>
