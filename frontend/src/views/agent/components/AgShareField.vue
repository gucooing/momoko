<!-- 共享范围：Agent 管理页里加、改共享的 MCP 服务器和技能时选给谁用。所有人，或者指定的人（从所有用户里点选，人多时能搜）。 -->
<template>
  <div class="ag-field">
    <span class="ag-fl">{{ t('agent.share.title') }}</span>
    <div class="ag-seg ag-share__seg" role="radiogroup" :aria-label="t('agent.share.title')">
      <button
        v-for="k in kinds"
        :key="k"
        type="button"
        role="radio"
        :class="{ 'is-on': modelValue.share === k }"
        :aria-checked="modelValue.share === k"
        @click="emit('update:modelValue', { ...modelValue, share: k })"
      >
        {{ t(`agent.share.${k}`) }}
      </button>
    </div>
    <template v-if="modelValue.share === 'users'">
      <span v-if="users.length > SEARCH_FROM" class="ag-fld">
        <UIcon name="i-lucide-search" />
        <input v-model="keyword" type="text" autocomplete="off" :placeholder="t('agent.share.search')" :aria-label="t('agent.share.search')" />
      </span>
      <div v-if="shown.length" class="ag-share__chips" role="group" :aria-label="t('agent.share.users')">
        <button
          v-for="u in shown"
          :key="u.id"
          type="button"
          class="ag-chip"
          :class="{ 'is-on': picked.has(u.id) }"
          :aria-pressed="picked.has(u.id)"
          @click="toggle(u.id)"
        >
          <UIcon :name="picked.has(u.id) ? 'i-lucide-check' : 'i-lucide-user'" />
          {{ userLabel(u) }}
        </button>
      </div>
      <span class="ag-fh">{{ usersHint }}</span>
    </template>
    <span v-else class="ag-fh">{{ t('agent.share.allHint') }}</span>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { type AgentShare, listAgentUsers } from '@/api/agent'
import type { AgentUser } from '@/types/v1/agent'

defineOptions({ name: 'AgShareField' })

const props = defineProps<{ modelValue: AgentShare }>()
const emit = defineEmits<{ 'update:modelValue': [value: AgentShare] }>()
const { t } = useI18n()

const kinds: AgentShare['share'][] = ['all', 'users']
/** 人多到这个数才给搜索框 */
const SEARCH_FROM = 12

// 用户列表各个弹窗共用，打开过一次就不再请求
const users = ref<AgentUser[]>([])
const state = ref<'loading' | 'ok' | 'failed'>('loading')
let loading: Promise<void> | null = null
const loadUsers = () => {
  loading ??= listAgentUsers()
    .then(({ data }) => {
      users.value = data.users ?? []
      state.value = 'ok'
    })
    .catch(() => {
      state.value = 'failed'
      loading = null
    })
  return loading
}
void loadUsers()

const keyword = ref('')
const picked = computed(() => new Set(props.modelValue.shareUserIds))
const userLabel = (u: AgentUser) => (u.name && u.name !== u.username ? `${u.name} (${u.username})` : u.username || u.id)
// 选中的总是显示（排在前面），没选中的按搜索词过滤
const shown = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  const match = (u: AgentUser) => !kw || `${u.name} ${u.username}`.toLowerCase().includes(kw)
  return [...users.value.filter((u) => picked.value.has(u.id)), ...users.value.filter((u) => !picked.value.has(u.id) && match(u))]
})
const usersHint = computed(() => {
  if (state.value === 'loading') return t('agent.common.loading')
  if (state.value === 'failed') return t('agent.share.loadFailed')
  const n = props.modelValue.shareUserIds.length
  return n ? t('agent.share.picked', { n }) : t('agent.share.pickHint')
})

const toggle = (id: string) => {
  const ids = props.modelValue.shareUserIds
  emit('update:modelValue', {
    ...props.modelValue,
    shareUserIds: ids.includes(id) ? ids.filter((x) => x !== id) : [...ids, id],
  })
}
</script>

<style lang="scss">
.ag-share__seg {
  align-self: flex-start;
}
.ag-share__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  max-height: 168px;
  overflow-y: auto;
  scrollbar-width: thin;
}
</style>
