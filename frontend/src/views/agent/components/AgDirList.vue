<!-- 服务器目录选择：一层一层点进去，点到哪一层就选中哪一层（modelValue 是选中的绝对路径）。
     每次列出目录后发 loaded，父组件可以从里面拿 cloneBase（克隆仓库的默认上级目录）。 -->
<template>
  <div
    ref="box"
    class="ag-dir ag-noscroll"
    role="listbox"
    :aria-label="t('agent.workspaces.dirs.label')"
    :aria-busy="loading"
  >
    <button v-if="canUp" type="button" class="ag-mi" :aria-label="t('agent.workspaces.dirs.up')" @click="goUp">
      <UIcon name="i-lucide-corner-left-up" />
      <span class="ag-mono ag-dir__up">..</span>
      <span class="ag-mono ag-ellipsis ag-dir__cur">{{ listing?.path }}</span>
    </button>
    <button
      v-for="d in dirs"
      :key="d.path"
      type="button"
      class="ag-mi"
      role="option"
      :class="{ 'is-on': same(d.path, modelValue) }"
      :aria-selected="same(d.path, modelValue)"
      @click="pick(d)"
    >
      <UIcon name="i-lucide-folder" />
      <span class="ag-mono ag-ellipsis ag-dir__name">{{ d.name || d.path }}</span>
      <span v-if="d.isGit" class="ag-dir__branch">
        <UIcon name="i-lucide-git-branch" />
        <span class="ag-ellipsis">{{ d.branch }}</span>
      </span>
    </button>
    <div v-if="!dirs.length" class="ag-dir__empty">
      {{ listing ? t('agent.workspaces.dirs.empty') : t('agent.common.loading') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { listAgentDirs } from '@/api/agent'
import type { AgentDir, ListAgentDirsResponse } from '@/types/v1/agent'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{
  'update:modelValue': [value: string]
  loaded: [data: ListAgentDirsResponse]
}>()
const { t } = useI18n()

const box = useTemplateRef<HTMLElement>('box')
const listing = ref<ListAgentDirsResponse | null>(null)
const loading = ref(false)
let seq = 0

const dirs = computed(() => listing.value?.dirs ?? [])
// 根（盘符列表）没有上一级
const canUp = computed(() => !!listing.value && (listing.value.parent !== '' || listing.value.path !== ''))

// 手输的路径末尾可能多一个分隔符
const trimSep = (p: string) => (p.length > 1 ? p.replace(/[\\/]+$/, '') : p)
const same = (a: string, b: string) => !!b && trimSep(a) === trimSep(b)

const load = async (path: string) => {
  const id = ++seq
  loading.value = true
  try {
    const { data } = await listAgentDirs(path)
    if (id !== seq) return true
    listing.value = data
    emit('loaded', data)
    await nextTick()
    if (box.value) box.value.scrollTop = 0
    return true
  } catch {
    // 请求层已经提示过
    return false
  } finally {
    if (id === seq) loading.value = false
  }
}

const goUp = () => void load(listing.value?.parent ?? '')

const pick = (dir: AgentDir) => {
  emit('update:modelValue', dir.path)
  void load(dir.path)
}

onMounted(async () => {
  const start = props.modelValue.trim()
  // 已经填了目录就从它开始；打不开（比如手输了一个不存在的路径）就回到根
  if (start && (await load(start))) return
  await load('')
})
</script>

<style lang="scss">
.ag-dir {
  display: flex;
  flex-direction: column;
  gap: 1px;
  max-height: 216px;
  overflow-y: auto;
  padding: 4px;
  border-radius: 10px;
  border: 1px solid var(--m-line);
  background: var(--m-s);
}
.ag-dir .ag-mi {
  flex: none;
}
.ag-dir .ag-mi > .iconify:first-child {
  color: var(--m-ink2);
}
.ag-dir__up {
  flex: none;
  font-size: 12px;
}
.ag-dir__cur {
  flex: 1;
  text-align: right;
  font-size: 11.5px;
  color: var(--m-ink2);
}
.ag-dir__name {
  flex: 1;
  font-size: 12px;
}
.ag-dir__branch {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 45%;
  min-width: 0;
  font-size: 11.5px;
  color: var(--m-ink2);
}
.ag-dir .ag-dir__branch .iconify {
  width: 12px;
  height: 12px;
}
.ag-dir__empty {
  padding: 6px 9px;
  font-size: 12px;
  color: var(--m-ink2);
}
</style>
