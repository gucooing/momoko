<!-- GM 运营 · GM 操作：「执行操作」先选玩家再执行（RunPanel），「玩家权限」管理玩家在通行证自助服务里能执行哪些操作
     （PermissionsPanel）；当前页面记在 ?view=（run / permissions）。 -->
<template>
  <div class="gop game-skin">
    <div class="gop__head">
      <h2 class="gop__h">{{ t('gmOps.run.title') }}</h2>
      <nav class="gop__seg" role="tablist">
        <button
          v-for="v in views"
          :key="v.key"
          type="button"
          role="tab"
          class="gop__seg-btn"
          :class="{ 'is-on': view === v.key }"
          :aria-selected="view === v.key"
          @click="setView(v.key)"
        >
          {{ v.label }}
        </button>
      </nav>
    </div>
    <RunPanel v-if="view === 'run'" />
    <PermissionsPanel v-else />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import PermissionsPanel from './PermissionsPanel.vue'
import RunPanel from './RunPanel.vue'

defineOptions({ name: 'GmOperationsTab' })

type View = 'run' | 'permissions'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const views = computed<{ key: View; label: string }[]>(() => [
  { key: 'run', label: t('gmOps.run.runView') },
  { key: 'permissions', label: t('gmOps.run.permissionsView') },
])
const view = computed<View>(() => (route.query.view === 'permissions' ? 'permissions' : 'run'))
const setView = (key: View) => void router.replace({ query: { ...route.query, view: key === 'run' ? undefined : key } })
</script>

<style scoped>
.gop {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.gop__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px 16px;
}
.gop__h {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 30px;
  font-weight: 800;
  letter-spacing: -0.01em;
}
.gop__seg {
  display: inline-flex;
  gap: 4px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.gop__seg-btn {
  min-height: 34px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
}
.gop__seg-btn.is-on {
  background: var(--m-ink);
  color: var(--m-bg);
}
@media (width < 768px) {
  .gop__h {
    font-size: 24px;
  }
}
</style>
