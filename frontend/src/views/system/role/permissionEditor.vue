<!-- 角色权限编辑器：把菜单树整理成「模块（顶层目录）→ 页面（菜单）→ 操作（按钮）」三层卡片。
     勾选会自动补齐上级（否则侧边栏看不到），取消会连带取消下级；目录在没有任何下级时自动移除。
     v-model 为角色 menuIds；locked 时只读。 -->
<template>
  <div class="pe">
    <section v-for="(m, mi) in modules" :key="m.id" class="pe-mod" :aria-label="m.title">
      <header class="pe-mod__head">
        <span class="m-sticker pe-mod__ico" :class="modTone(mi)" :style="{ transform: `rotate(${ROTS[mi % ROTS.length]}deg)` }">
          <component :is="menuStore.iconComponents[m.icon || 'HOutline:Squares2X2Icon']" />
        </span>
        <h3 class="pe-mod__title">{{ m.title }}</h3>
        <span class="pe-mod__count tnum">{{ onCount(m.leafIds) }} / {{ m.leafIds.length }}</span>
        <span class="pe-mod__spacer" />
        <label class="pe-mod__all">
          <span>{{ t('accessAdmin.role.selectAll') }}</span>
          <AppSwitch
            :model-value="allOn(m.leafIds)"
            :disabled="locked"
            @update:model-value="(v) => setMany(m.allIds, v)"
          />
        </label>
      </header>

      <div v-for="p in m.pages" :key="p.id" class="pe-page">
        <label class="pe-page__main">
          <input
            type="checkbox"
            class="pe-cb"
            :checked="allOn(p.ids)"
            :indeterminate.prop="!allOn(p.ids) && onCount(p.ids) > 0"
            :disabled="locked"
            :aria-label="p.title"
            @change="setMany(p.ids, !allOn(p.ids))"
          />
          <span class="pe-page__text">
            <span class="pe-page__title">
              {{ p.title }}
              <span class="pe-page__n tnum">{{ onCount(p.ids) }}/{{ p.ids.length }}</span>
            </span>
            <span v-if="p.path" class="pe-page__path m-mono">{{ p.path }}</span>
          </span>
        </label>
        <div v-if="p.actions.length > 1" class="pe-page__acts">
          <label
            v-for="a in p.actions"
            :key="a.id"
            class="pe-chip"
            :class="{ 'is-on': selected.has(a.id), 'is-bad': selected.has(a.id) && a.danger, 'is-locked': locked }"
          >
            <input
              type="checkbox"
              class="pe-cb"
              :checked="selected.has(a.id)"
              :disabled="locked"
              :aria-label="`${p.title} · ${a.title}`"
              @change="toggle(a.id)"
            />
            {{ a.title }}
          </label>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { translateKnownText } from '@/locales'
import { MenuType, type MenuInfo } from '@/types/v1/system'

const props = defineProps<{ menus: MenuInfo[]; locked?: boolean }>()
const model = defineModel<string[]>({ default: () => [] })

const { t } = useI18n()
const menuStore = useMenuStore()
const ROTS = [-4, 3, -2, 4, -3, 2]
const TONES = ['m-sticker--p', 'm-sticker--a', 'm-sticker--a2', 'm-sticker--ok']
const modTone = (i: number) => TONES[i % TONES.length]

interface ActionItem {
  id: string
  title: string
  danger: boolean
}
interface PageItem {
  id: string
  title: string
  path: string
  ids: string[]
  actions: ActionItem[]
}
interface ModuleItem {
  id: string
  title: string
  icon: string
  pages: PageItem[]
  allIds: string[]
  leafIds: string[]
}

const tr = (s: string) => translateKnownText(s) || s
const isDanger = (m: MenuInfo) => /delete|删除|刪除/i.test(`${m.permissions} ${m.title}`)

// 树索引：父级、子级、类型
const index = computed(() => {
  const parent = new Map<string, string>()
  const children = new Map<string, string[]>()
  const type = new Map<string, MenuType>()
  const walk = (list: MenuInfo[], pid: string) => {
    for (const m of list) {
      type.set(m.id, m.type)
      if (pid) parent.set(m.id, pid)
      children.set(m.id, (m.children || []).map((c) => c.id))
      if (m.children?.length) walk(m.children, m.id)
    }
  }
  walk(props.menus, '')
  return { parent, children, type }
})

const collectButtons = (m: MenuInfo, acc: MenuInfo[]) => {
  for (const c of m.children || []) {
    if (c.type === MenuType.MenuType_Button) acc.push(c)
    if (c.children?.length && c.type === MenuType.MenuType_Button) collectButtons(c, acc)
  }
  return acc
}

const toPage = (m: MenuInfo): PageItem => {
  const buttons = collectButtons(m, [])
  return {
    id: m.id,
    title: tr(m.title),
    path: m.path || '',
    ids: [m.id, ...buttons.map((b) => b.id)],
    actions: [
      { id: m.id, title: t('accessAdmin.role.access'), danger: false },
      ...buttons.map((b) => ({ id: b.id, title: tr(b.title), danger: isDanger(b) })),
    ],
  }
}

const modules = computed<ModuleItem[]>(() =>
  props.menus.map((top) => {
    const pages: PageItem[] = []
    const allIds: string[] = []
    const visit = (node: MenuInfo) => {
      allIds.push(node.id)
      if (node.type === MenuType.MenuType_Directory) {
        const loose = (node.children || []).filter((c) => c.type === MenuType.MenuType_Button)
        if (loose.length) {
          // 目录下直接挂的按钮：归到以目录本身为「页面」的一行
          pages.push({
            id: node.id,
            title: tr(node.title),
            path: '',
            ids: loose.map((b) => b.id),
            actions: loose.map((b) => ({ id: b.id, title: tr(b.title), danger: isDanger(b) })),
          })
          loose.forEach((b) => allIds.push(b.id))
        }
        for (const c of node.children || []) if (c.type !== MenuType.MenuType_Button) visit(c)
        return
      }
      const page = toPage(node)
      pages.push(page)
      page.ids.slice(1).forEach((id) => allIds.push(id))
      for (const c of node.children || []) if (c.type === MenuType.MenuType_Menu) visit(c)
    }
    visit(top)
    const leafIds = allIds.filter((id) => index.value.type.get(id) !== MenuType.MenuType_Directory)
    return { id: top.id, title: tr(top.title), icon: top.icon, pages, allIds, leafIds }
  }).filter((m) => m.leafIds.length),
)

const selected = computed(() => new Set(model.value))
const onCount = (ids: string[]) => ids.filter((id) => selected.value.has(id)).length
const allOn = (ids: string[]) => ids.length > 0 && ids.every((id) => selected.value.has(id))

const ancestors = (id: string) => {
  const out: string[] = []
  let cur = index.value.parent.get(id)
  while (cur) {
    out.push(cur)
    cur = index.value.parent.get(cur)
  }
  return out
}
const descendants = (id: string, acc: string[] = []) => {
  for (const c of index.value.children.get(id) || []) {
    acc.push(c)
    descendants(c, acc)
  }
  return acc
}

// 没有任何已选下级的目录自动移除
const prune = (set: Set<string>) => {
  let changed = true
  while (changed) {
    changed = false
    for (const id of [...set]) {
      if (index.value.type.get(id) !== MenuType.MenuType_Directory) continue
      if (!descendants(id).some((d) => set.has(d))) {
        set.delete(id)
        changed = true
      }
    }
  }
}

const setMany = (ids: string[], on: boolean) => {
  if (props.locked) return
  const next = new Set(model.value)
  for (const id of ids) {
    if (on) {
      next.add(id)
      ancestors(id).forEach((a) => next.add(a))
    } else {
      next.delete(id)
      descendants(id).forEach((d) => next.delete(d))
    }
  }
  if (!on) prune(next)
  model.value = [...next]
}
const toggle = (id: string) => setMany([id], !selected.value.has(id))
</script>

<style scoped lang="scss">
.pe {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.pe-mod {
  padding: 18px 24px 8px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-xl, var(--app-radius-lg));
  background: var(--el-bg-color);
}
.pe-mod__head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-bottom: 12px;
}
.pe-mod__ico {
  width: 40px;
  height: 40px;
  border-radius: 13px;
}
.pe-mod__ico :deep(svg) {
  width: 20px;
  height: 20px;
}
.pe-mod__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.pe-mod__count {
  padding: 2px 10px;
  border-radius: 999px;
  background: var(--m-s2);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
}
.pe-mod__spacer {
  flex: 1;
}
.pe-mod__all {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.pe-page {
  display: grid;
  grid-template-columns: minmax(180px, 240px) 1fr;
  align-items: center;
  gap: 10px 20px;
  padding: 12px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.pe-page__main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  cursor: pointer;
}
.pe-page__text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}
.pe-page__title {
  font-size: 0.9375rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.pe-page__n {
  margin-left: 4px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.pe-page__path {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pe-page__acts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.pe-cb {
  width: 20px;
  height: 20px;
  margin: 0;
  flex: none;
  accent-color: var(--m-p);
  cursor: pointer;
}
.pe-cb:disabled {
  cursor: not-allowed;
}
.pe-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 14px 0 10px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  user-select: none;
}
.pe-chip:hover:not(.is-locked) {
  border-color: var(--m-p);
}
.pe-chip .pe-cb {
  width: 16px;
  height: 16px;
}
.pe-chip.is-on {
  border-color: transparent;
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.pe-chip.is-bad {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.pe-chip.is-bad .pe-cb {
  accent-color: var(--m-bad);
}
.pe-chip.is-locked {
  cursor: not-allowed;
}
@media (width < 768px) {
  .pe-mod {
    padding: 14px 14px 6px 16px;
  }
  .pe-mod__head {
    gap: 10px;
    padding-bottom: 6px;
  }
  .pe-mod__title {
    font-size: 1rem;
  }
  .pe-mod__all {
    gap: 8px;
    font-size: 0.8125rem;
  }
  .pe-page {
    grid-template-columns: 1fr;
    gap: 8px;
    padding: 10px 0;
  }
  .pe-chip {
    min-height: 44px;
  }
}
</style>
