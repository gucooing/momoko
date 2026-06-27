<template>
  <div class="ftree">
    <div v-if="loading" class="ftree-state">{{ t('fileManager.directoryLoading') }}</div>
    <div v-else-if="!nodes.length" class="ftree-state">{{ t('fileManager.directoryEmpty') }}</div>
    <FileTreeNode v-for="node in nodes" v-else :key="node.path" :node="node" :depth="0" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { showRequestError } from '@/utils/request'
import type { FileTreeNode as FileTreeNodeType } from '@/types/v1/file'
import { FILE_TREE_CONTEXT, type FileClient } from './types'
import FileTreeNode from './FileTreeNode.vue'

const props = withDefaults(
  defineProps<{
    client: FileClient
    rootPath: string
    activePath: string
    selectable?: 'file' | 'all'
  }>(),
  {
    selectable: 'file',
  },
)

const emit = defineEmits<{ select: [path: string, name: string, isDir: boolean] }>()

const { t } = useI18n()

const nodes = ref<FileTreeNodeType[]>([])
const loading = ref(false)

provide(FILE_TREE_CONTEXT, {
  client: props.client,
  activePath: toRef(props, 'activePath'),
  selectable: props.selectable,
  select: (path: string, name: string, isDir: boolean) => emit('select', path, name, isDir),
})

const loadRoot = async () => {
  loading.value = true
  try {
    nodes.value = await props.client.tree(props.rootPath)
  } catch (error) {
    showRequestError(error, t('fileManager.treeLoadFailed'))
  } finally {
    loading.value = false
  }
}

watch(() => props.rootPath, loadRoot)
onMounted(loadRoot)

defineExpose({ reload: loadRoot })
</script>

<style scoped>
.ftree {
  padding: 0.25rem 0;
}
.ftree-state {
  padding: 1.5rem 0.75rem;
  text-align: center;
  font-size: 12.5px;
  color: var(--fm-text-3);
}
</style>
