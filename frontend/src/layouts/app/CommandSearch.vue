<!-- 全局搜索/快速跳转（⌘K）。首版做菜单快速跳转（04 §3）。 -->
<template>
  <AppDropdown ref="dropdownRef" align="start" :width="360">
    <template #trigger>
      <button type="button" class="search-box">
        <component
          :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']"
          class="search-box__icon"
        />
        <span class="search-box__ph">{{ searchPlaceholder }}</span>
        <kbd class="search-box__kbd">{{ isMac ? '⌘' : 'Ctrl' }} K</kbd>
      </button>
    </template>

    <template #default="{ close }">
      <div class="cmd">
        <div class="cmd__input-row">
          <component
            :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']"
            class="cmd__input-icon"
          />
          <input
            :ref="(el) => el && (el as HTMLInputElement).focus()"
            v-model="query"
            class="cmd__input"
            :placeholder="searchPlaceholder"
            @keydown.enter="results[0] && jump(results[0].path, close)"
            @keydown.esc="close()"
          />
        </div>
        <div class="cmd__list">
          <button
            v-for="item in results"
            :key="item.path"
            type="button"
            class="cmd__item"
            @click="jump(item.path, close)"
          >
            <span class="cmd__item-icon">
              <component :is="menuStore.iconComponents[item.icon]" v-if="item.icon" />
            </span>
            <span class="cmd__item-title">{{ item.title }}</span>
            <span class="cmd__item-path">{{ item.path }}</span>
          </button>
          <div v-if="results.length === 0" class="cmd__empty">
            {{ t('common.noData') }}
          </div>
        </div>
      </div>
    </template>
  </AppDropdown>
</template>

<script setup lang="ts">
import { MenuType, type MenuInfo } from '@/types/v1/system'
import { translateKnownText } from '@/locales'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const menuStore = useMenuStore()
const { t } = useI18n()

const dropdownRef = ref<{ open: () => void } | null>(null)
const query = ref('')
const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)
// 样板占位文案；正式接入时并入 messages.ts 的 layout.searchPlaceholder
const searchPlaceholder = '搜索菜单…'

interface FlatItem {
  title: string
  path: string
  icon?: string
}
const flatItems = computed(() => {
  const out: FlatItem[] = []
  const walk = (nodes?: MenuInfo[]) => {
    nodes?.forEach((n) => {
      if (n.type === MenuType.MenuType_Menu && n.path) {
        out.push({ title: translateKnownText(n.title), path: n.path, icon: n.icon })
      }
      if (n.children?.length) walk(n.children)
    })
  }
  walk(menuStore.menuList)
  return out
})
const results = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return flatItems.value.slice(0, 8)
  return flatItems.value.filter((i) => i.title.toLowerCase().includes(q)).slice(0, 12)
})

const jump = (path: string, close: () => void) => {
  if (router.currentRoute.value.path !== path) router.push(path)
  query.value = ''
  close()
}

const onKeydown = (e: KeyboardEvent) => {
  if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'K')) {
    e.preventDefault()
    dropdownRef.value?.open()
  }
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped lang="scss">
.search-box {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 220px;
  height: 36px;
  padding: 0 10px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-placeholder);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.search-box:hover {
  border-color: var(--el-border-color);
  background: var(--el-bg-color);
}
.search-box__icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
.search-box__ph {
  flex: 1;
  text-align: left;
  font-size: 0.8125rem;
}
.search-box__kbd {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--el-text-color-placeholder);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  padding: 1px 5px;
  font-family: inherit;
}

.cmd__input-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.cmd__input-icon {
  width: 18px;
  height: 18px;
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}
.cmd__input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font-size: 0.9375rem;
  color: var(--el-text-color-primary);
}
.cmd__list {
  max-height: 20rem;
  overflow-y: auto;
  padding: 6px;
}
.cmd__item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  cursor: pointer;
  text-align: left;
  transition: background 0.15s;
}
.cmd__item:hover {
  background: var(--el-fill-color-light);
}
.cmd__item-icon {
  display: flex;
  width: 18px;
  height: 18px;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}
.cmd__item-icon :deep(svg) {
  width: 18px;
  height: 18px;
}
.cmd__item-title {
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}
.cmd__item-path {
  margin-left: auto;
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
}
.cmd__empty {
  padding: 1.5rem;
  text-align: center;
  font-size: 0.8125rem;
  color: var(--el-text-color-placeholder);
}
</style>
