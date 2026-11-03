<!-- 用户与权限页内标签：用户 / 角色 / 菜单（/system/access?tab=…），按权限过滤，带各标签计数。
     页面已知的计数通过 props 传入；未传的计数在有该页访问权限时自己轻量查询（pageSize=1 取 total）。 -->
<template>
  <SubNavTabs group="access" :counts="counts" />
</template>

<script setup lang="ts">
import { userPage } from '@/api/user'
import { rolePage } from '@/api/role'
import { adminPermissionsList } from '@/api/menu'

const props = defineProps<{
  userCount?: number | null
  roleCount?: number | null
  menuCount?: number | null
}>()

const menuStore = useMenuStore()
const fetched = ref<{ user: number | null; role: number | null; menu: number | null }>({
  user: null,
  role: null,
  menu: null,
})

const countMenus = (list: { children?: unknown[] }[]): number =>
  list.reduce((n, m) => n + 1 + countMenus((m.children || []) as { children?: unknown[] }[]), 0)

onMounted(async () => {
  const jobs: Promise<void>[] = []
  if (props.userCount === undefined && menuStore.hasButtonPermission('user:view'))
    jobs.push(
      userPage({ page: 1, pageSize: 1 }).then(({ data }) => {
        fetched.value.user = Number(data?.total || 0)
      }),
    )
  if (props.roleCount === undefined && menuStore.hasButtonPermission('role:view'))
    jobs.push(
      rolePage({ page: 1, pageSize: 1 }).then(({ data }) => {
        fetched.value.role = Number(data?.total || 0)
      }),
    )
  if (props.menuCount === undefined && menuStore.hasButtonPermission('menu:view'))
    jobs.push(
      adminPermissionsList({}).then(({ data }) => {
        fetched.value.menu = countMenus(data?.menus || [])
      }),
    )
  await Promise.allSettled(jobs)
})

const pick = (prop: number | null | undefined, own: number | null) => (prop === undefined ? own : prop)

const counts = computed(() => ({
  user: pick(props.userCount, fetched.value.user),
  role: pick(props.roleCount, fetched.value.role),
  menu: pick(props.menuCount, fetched.value.menu),
}))
</script>
