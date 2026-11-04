import { listTasks } from '@/api/task'

// 进行中的后台任务数（「更多」里「后台任务」入口的角标、底部导航「更多」的小圆点）。
// 多处同时使用时共享一次轮询；没有 task:manage 权限时恒为 0。
const POLL_MS = 30_000
const count = ref(0)
let users = 0
let timer: ReturnType<typeof setInterval> | null = null

const refresh = async () => {
  const menuStore = useMenuStore()
  if (!menuStore.hasButtonPermission('task:manage')) {
    count.value = 0
    return
  }
  try {
    const { data } = await listTasks({ page: 1, pageSize: 1, status: 'running' })
    count.value = Number(data?.counts?.running ?? data?.total ?? 0)
  } catch {
    // 忽略：角标只是提示
  }
}

export const refreshRunningTaskCount = refresh

export const useRunningTaskCount = () => {
  onMounted(() => {
    users += 1
    if (users === 1) {
      void refresh()
      timer = setInterval(refresh, POLL_MS)
    }
  })
  onBeforeUnmount(() => {
    users -= 1
    if (users <= 0 && timer) {
      clearInterval(timer)
      timer = null
      users = 0
    }
  })
  return count
}
