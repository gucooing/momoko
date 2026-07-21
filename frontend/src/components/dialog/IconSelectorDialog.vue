<template>
  <BaseDialog v-model="open" :title="resolvedTitle" :width="width" :show-footer="false">
    <div
      class="icon-selector-dialog-container"
      :class="{
        'is-mobile': menuStore.isMobile,
        'is-spacious': props.density === 'spacious',
      }"
    >
      <div class="icon-menu">
        <button
          v-for="menu in iconMenu"
          :key="menu.value"
          type="button"
          class="item-menu-item"
          :class="{ active: activeMenu === menu.value }"
          @click="activeMenu = menu.value"
        >
          <component :is="menuStore.iconComponents[menu.icon]" class="item-menu-item__icon" />
          <span>{{ menu.label }}</span>
        </button>
      </div>
      <div class="icon-content">
        <transition name="fade-slide" mode="out-in">
          <div :key="activeMenu" class="icon-content__pane">
            <div class="icon-search">
              <component
                :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']"
                class="icon-search__icon"
              />
              <input
                v-model="searchValue"
                class="app-input icon-search__input"
                type="search"
                :placeholder="t('iconSelector.searchPlaceholder')"
              />
              <button
                v-if="searchValue"
                type="button"
                class="icon-search__clear"
                :aria-label="t('common.close')"
                @click="searchValue = ''"
              >
                ×
              </button>
            </div>
            <div
              class="icon-list-scroll"
              :class="menuStore.isMobile ? 'is-mobile-scroll' : 'is-desktop-scroll'"
            >
              <div class="icon-list">
                <button
                  v-for="icon in filteredIconList"
                  :key="icon"
                  type="button"
                  class="icon-item"
                  :class="{ active: currentIcon === icon }"
                  :title="icon"
                  @click="selectIcon(icon)"
                >
                  <component :is="menuStore.iconComponents[icon]" class="icon-item__glyph" />
                  <span v-if="props.density === 'spacious'" class="icon-name">{{ icon }}</span>
                </button>
              </div>
            </div>
          </div>
        </transition>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { loadIconCatalog, type IconPrefix } from '@/config/iconRegistry'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'IconSelectorDialog', inheritAttrs: false })

interface IProps {
  title?: string
  width?: string | number
  density?: 'compact' | 'spacious'
}

interface IEmits {
  (e: 'selectIcon', icon: string, component: Component): void
}

type IActiveMenu = IconPrefix

interface IIconMenuItem {
  label: string
  value: IActiveMenu
  icon: string
}

const props = withDefaults(defineProps<IProps>(), {
  width: '900px',
  density: 'compact',
})

const emits = defineEmits<IEmits>()
const open = ref(false)
const menuStore = useMenuStore()
const { t } = useI18n()
const resolvedTitle = computed(() => props.title || t('iconSelector.title'))
const currentIcon = ref('')
const searchValue = ref('')
const activeMenu = ref<IActiveMenu>('Element:')
const iconMenu = ref<IIconMenuItem[]>([
  { label: 'Element Plus', value: 'Element:', icon: 'Element:Monitor' },
  { label: 'HeroIcons Outline', value: 'HOutline:', icon: 'HOutline:ShieldCheckIcon' },
  { label: 'HeroIcons Solid', value: 'HSolid:', icon: 'HSolid:ShieldCheckIcon' },
])
const activeIconList = ref<string[]>([])

const syncActiveIconList = async () => {
  activeIconList.value = await loadIconCatalog(activeMenu.value)
}

const filteredIconList = computed(() => {
  if (!searchValue.value) return activeIconList.value
  const search = searchValue.value.toLowerCase()
  return activeIconList.value.filter((name) => name.toLowerCase().includes(search))
})

const selectIcon = (icon: string) => {
  currentIcon.value = icon
  emits('selectIcon', icon, menuStore.iconComponents[icon] as Component)
  closeDialog()
}

const showDialog = (currentIconValue: string = '') => {
  currentIcon.value = currentIconValue
  if (currentIconValue.startsWith('HOutline:')) activeMenu.value = 'HOutline:'
  else if (currentIconValue.startsWith('HSolid:')) activeMenu.value = 'HSolid:'
  else activeMenu.value = 'Element:'
  open.value = true
}

const closeDialog = () => {
  open.value = false
  searchValue.value = ''
}

const clearData = () => {
  currentIcon.value = ''
  searchValue.value = ''
  activeMenu.value = 'Element:'
}

watch(
  [open, activeMenu],
  ([isOpen]) => {
    if (!isOpen) return
    void syncActiveIconList()
  },
  { immediate: true },
)

defineExpose({ showDialog, closeDialog, clearData })
</script>

<style scoped lang="scss">
.icon-selector-dialog-container {
  height: min(60vh, 560px);
  display: flex;
  gap: 1rem;
}
.icon-menu {
  width: 12.5rem;
  padding: 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  border-right: 1px solid var(--el-border-color-lighter);
}
.item-menu-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.85rem;
  border: none;
  border-radius: 0.5rem;
  background: transparent;
  cursor: pointer;
  color: var(--el-text-color-primary);
  font-weight: 500;
  text-align: left;
  transition: background 0.15s, color 0.15s;
}
.item-menu-item__icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}
.item-menu-item:hover {
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
}
.item-menu-item.active {
  background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
  color: var(--el-color-primary);
}
.icon-content {
  flex: 1;
  min-width: 0;
  padding: 0.5rem;
  display: flex;
  flex-direction: column;
}
.icon-content__pane {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.icon-search {
  position: relative;
  display: flex;
  align-items: center;
}
.icon-search__icon {
  position: absolute;
  left: 10px;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-placeholder);
  pointer-events: none;
}
.icon-search__input {
  width: 100%;
  padding-left: 32px;
  padding-right: 28px;
}
.icon-search__clear {
  position: absolute;
  right: 8px;
  border: none;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
}
.icon-list-scroll {
  margin-top: 0.75rem;
  overflow: auto;
  min-height: 0;
}
.icon-list-scroll.is-desktop-scroll {
  height: calc(100% - 2.5rem);
}
.icon-list-scroll.is-mobile-scroll {
  height: calc(100% - 2.5rem);
}
.icon-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(50px, 1fr));
  gap: 0.75rem;
  padding: 0.25rem 0;
}
.icon-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--el-border-color);
  border-radius: 0.5rem;
  background: var(--el-bg-color);
  cursor: pointer;
  padding: 0.5rem 0.25rem;
  transition: border-color 0.15s, background 0.15s, transform 0.15s;
  color: var(--el-text-color-primary);
}
.icon-item__glyph {
  width: 22px;
  height: 22px;
}
.icon-item:hover {
  border-color: var(--el-color-primary);
  background: var(--el-fill-color-light);
  transform: translateY(-1px);
}
.icon-item.active {
  border-color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
}
.icon-name {
  width: 100%;
  margin-top: 0.35rem;
  text-align: center;
  font-size: 0.75rem;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 0 0.25rem;
}
.is-spacious .icon-list {
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 1rem;
}
.is-spacious .icon-item {
  padding: 0.75rem 0.5rem;
}
.is-spacious .icon-item__glyph {
  width: 24px;
  height: 24px;
}
.is-mobile {
  flex-direction: column;
  gap: 0.5rem;
  height: min(72vh, 640px);
}
.is-mobile .icon-menu {
  width: 100%;
  border-right: none;
  flex-direction: row;
  border-bottom: 1px solid var(--el-border-color-lighter);
  gap: 0.25rem;
}
.is-mobile .item-menu-item {
  flex: 1;
  justify-content: center;
  padding: 0.65rem 0.35rem;
  font-size: 0.75rem;
}

.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: opacity 0.15s ease;
}
.fade-slide-enter-from,
.fade-slide-leave-to {
  opacity: 0;
}
</style>
