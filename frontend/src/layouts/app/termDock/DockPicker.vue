<!-- 新建终端的选择列表：本机终端（shell:use）+ SSH 主机（ssh:manage）。「+」菜单和空窗口共用。 -->
<template>
  <div class="tdp">
    <button v-if="canLocal" type="button" class="ag-mi" role="menuitem" @click="pickLocal">
      <UIcon name="i-lucide-square-terminal" />
      <span class="ag-ellipsis">{{ t('termDock.local') }}</span>
    </button>
    <template v-if="canSsh">
      <div v-if="canLocal" class="ag-mi-sep" />
      <div class="ag-mi-label">{{ t('termDock.sshHosts') }}</div>
      <div class="tdp__list">
        <button v-for="h in hosts" :key="h.id" type="button" class="ag-mi" role="menuitem" :title="`${h.username}@${h.host}:${h.port}`" @click="pickSsh(h)">
          <UIcon name="i-lucide-server" />
          <span class="tdp__name ag-ellipsis">{{ h.name }}</span>
          <span class="tdp__sub ag-muted ag-mono ag-ellipsis">{{ h.username }}@{{ h.host }}</span>
        </button>
      </div>
      <div v-if="loading && !hosts.length" class="tdp__note ag-muted">{{ t('termDock.loading') }}</div>
      <div v-else-if="!hosts.length" class="tdp__note ag-muted">{{ t('termDock.noHosts') }}</div>
      <RouterLink to="/openssh/management" class="ag-mi" role="menuitem" @click="emit('pick')">
        <UIcon name="i-lucide-settings-2" />
        <span class="ag-ellipsis">{{ t('termDock.manageHosts') }}</span>
      </RouterLink>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getSshHosts } from '@/api/openssh'
import { SHELL_PERMISSION } from '@/api/shell'
import type { SSHHostInfo } from '@/types/v1/openssh'
import { addTab } from './state'

defineOptions({ name: 'DockPicker' })

const emit = defineEmits<{ pick: [] }>()

const { t } = useI18n()
const menuStore = useMenuStore()

const canLocal = computed(() => menuStore.hasButtonPermission(SHELL_PERMISSION))
const canSsh = computed(() => menuStore.hasButtonPermission('ssh:manage'))

const hosts = ref<SSHHostInfo[]>([])
const loading = ref(false)

const loadHosts = async () => {
  if (!canSsh.value) return
  loading.value = true
  try {
    const { data } = await getSshHosts({ page: 1, pageSize: 50 })
    hosts.value = data?.infos || []
  } catch {
    hosts.value = []
  } finally {
    loading.value = false
  }
}

const pickLocal = () => {
  addTab('local')
  emit('pick')
}

const pickSsh = (h: SSHHostInfo) => {
  addTab('ssh', h.id, `${h.username}@${h.name || h.host}`)
  emit('pick')
}

onMounted(loadHosts)
</script>

<style scoped lang="scss">
.tdp {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.tdp__list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 240px;
  overflow-y: auto;
}
.tdp__name {
  flex: 0 1 auto;
}
.tdp__sub {
  flex: 1 1 0;
  margin-left: auto;
  font-size: 11.5px;
  text-align: right;
}
.tdp__note {
  padding: 4px 10px 6px;
  font-size: 12px;
}
</style>
