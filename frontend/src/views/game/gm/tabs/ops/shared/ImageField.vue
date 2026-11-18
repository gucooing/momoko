<!-- 图片字段（横幅、标题图、背景）：显示当前的图，点「更换」从图里选；不显示、也不需要填资源名。
     前端素材包里没有的图，显示它在配置里被谁用着（「和「XXX」相同」）。 -->
<template>
  <div class="gif">
    <GameImage v-if="url" :src="url" class="gif__img" />
    <span v-else class="gif__img gif__img--empty">{{ emptyText }}</span>
    <UButton class="gmc-btn" size="sm" color="neutral" variant="soft" icon="i-lucide-image" :disabled="!sources.length" @click="open = true">
      {{ t('gmConfig.ops.changeImage') }}
    </UButton>
    <AssetPickerDialog v-model="open" :title="title" :sources="sources" :selected="modelValue" @pick="emit('update:modelValue', $event)" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AssetPickerDialog from './AssetPickerDialog.vue'
import { bannerImage, type AssetSource } from './assets'

const props = defineProps<{ modelValue: string | undefined; title: string; sources: AssetSource[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const { t } = useI18n()
const open = ref(false)
const url = computed(() => bannerImage(props.modelValue || ''))
const emptyText = computed(() => {
  if (!props.modelValue) return t('gmConfig.ops.noImage')
  const owner = props.sources.flatMap((s) => s.items || []).find((x) => x.assetId === props.modelValue)?.label
  return owner ? t('gmConfig.ops.imageSameAs', { name: owner }) : t('gmConfig.ops.imageUnknown')
})
</script>

<style scoped lang="scss">
.gif {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.gif .gif__img {
  width: 240px;
  max-width: 100%;
  height: 72px;
  object-fit: contain;
  border-radius: 12px;
  background: transparent;
}
.gif__img--empty {
  display: flex;
  padding: 0 10px;
  text-align: center;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color-light) !important;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
