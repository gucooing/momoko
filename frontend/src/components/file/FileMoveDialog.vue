<!-- 「移动到… / 复制到…」目标目录选择：令牌 FormDialog + 只含文件夹的懒加载目录树（同一来源内）。 -->
<template>
  <FormDialog
    v-model="visible"
    :title="mode === 'move' ? t('fileManager.moveTo') : t('fileManager.copyTo')"
    :width="480"
    :loading="confirming"
    @close="target = null"
  >
    <div class="fmv">
      <p class="fmv__desc">{{ description }}</p>
      <div class="fmv__tree">
        <button type="button" class="fmv__root" :class="{ 'is-on': target === '' }" @click="target = ''">
          <UIcon name="i-lucide-hard-drive" />
          <span>{{ rootLabel || t('fileManager.sourceRoot') }}</span>
        </button>
        <FileTree
          v-if="visible"
          :client="client"
          root-path=""
          :active-path="target ?? ''"
          selectable="all"
          dirs-only
          @select="(path: string) => (target = path)"
        />
      </div>
      <div class="fmv__target">
        <span class="fmv__label">{{ t('fileManager.targetDir') }}</span>
        <span class="fmv__path m-mono" :title="target ?? ''">{{ target === null ? '—' : target && target !== '.' ? displayDirPath(target) : rootLabel || t('fileManager.sourceRoot') }}</span>
      </div>
    </div>

    <template #footer="{ close }">
      <UButton color="neutral" variant="soft" @click="close">{{ t('system.common.cancel') }}</UButton>
      <UButton color="primary" :loading="confirming" :disabled="target === null" @click="target !== null && emit('confirm', target)">
        {{ mode === 'move' ? t('fileManager.moveHere') : t('fileManager.copyHere') }}
      </UButton>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import FileTree from './FileTree.vue'
import { displayDirPath } from './fileClient'
import type { FileClient } from './types'

const props = defineProps<{
  modelValue: boolean
  client: FileClient
  mode: 'move' | 'copy'
  count: number
  // 预选目录（通常为当前目录）
  initialPath?: string
  // 来源根目录的展示名（如「本地磁盘」「阿里云 OSS」）
  rootLabel?: string
  confirming?: boolean
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; confirm: [target: string] }>()

const { t } = useI18n()
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
// null=尚未选择；''=来源根目录
const target = ref<string | null>(null)
watch(visible, (open) => {
  if (open) target.value = props.initialPath ?? null
})
const description = computed(() =>
  props.mode === 'move'
    ? t('fileManager.moveToDesc', { count: props.count })
    : t('fileManager.copyToDesc', { count: props.count }),
)
</script>

<style scoped>
.fmv {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.fmv__desc {
  margin: 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.fmv__tree {
  max-height: 340px;
  overflow-y: auto;
  padding: 6px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
  scrollbar-width: thin;
  --ft-active-bg: var(--m-pSoft);
  --ft-active-fg: var(--m-pInk);
  --ft-hover: var(--m-s2);
}
.fmv__root {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 40px;
  padding: 0 10px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
  font-weight: 700;
  text-align: left;
  cursor: pointer;
}
.fmv__root:hover {
  background: var(--m-s2);
}
.fmv__root.is-on {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.fmv__tree :deep(.ftn-row) {
  min-height: 40px;
  border-radius: 10px;
}
.fmv__tree :deep(.ftn-icon.is-folder) {
  color: var(--m-a);
}
.fmv__target {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-s2);
  font-size: 0.8125rem;
}
.fmv__label {
  flex: none;
  color: var(--el-text-color-secondary);
  font-weight: 700;
}
.fmv__path {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
