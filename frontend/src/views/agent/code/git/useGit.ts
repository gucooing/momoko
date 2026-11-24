// 右侧「Git」标签的状态和操作：几个仓库的状态、选中的仓库、下面看的那份 diff、提交说明，
// 以及右键菜单、确认框、输入框这几个面板里共用的浮层。面板里的子组件用 useGit() 取到同一份。
import type { InjectionKey } from 'vue'
import { getAgentGitStatus, type AgentGitDiffSource } from '@/api/agent'
import { translate } from '@/locales'
import type { AgentGitFile, AgentGitStatus } from '@/types/v1/agent'

export type GitView = 'changes' | 'history' | 'stashes'

export interface GitMenuItem {
  label?: string
  icon?: string
  danger?: boolean
  disabled?: boolean
  /** 右边的小字（快捷键、说明） */
  hint?: string
  /** 分隔线 */
  sep?: boolean
  /** 小标题 */
  title?: string
  run?: () => unknown
}

export interface GitAsk {
  title: string
  message?: string
  confirmText?: string
  danger?: boolean
  /** 弹窗里的一个复选框（比如「没合并过的也删」） */
  check?: string
  checked?: boolean
  run: (checked: boolean) => unknown
}

export interface GitPromptField {
  key: string
  label: string
  value?: string
  placeholder?: string
  multiline?: boolean
  mono?: boolean
  required?: boolean
}
export interface GitPrompt {
  title: string
  fields: GitPromptField[]
  checks?: { key: string; label: string; value?: boolean }[]
  /** 单选（重置方式之类）：value 是选中的 */
  choice?: { key: string; value: string; options: { value: string; label: string; hint?: string; danger?: boolean }[] }
  confirmText?: string
  danger?: boolean
  run: (values: Record<string, string>, checks: Record<string, boolean>, choice: string) => unknown
}

/** 下面那份 diff 看的是什么 */
export interface GitDiffTarget {
  /** 哪个仓库的（相对会话目录）：换了仓库就不显示别的仓库的 diff */
  repo: string
  source: AgentGitDiffSource
  /** 相对仓库 */
  path: string
  origPath?: string
  /** commit、stash 的哈希 */
  ref?: string
  /** 文件的改动种类（M、A、D、R、U……），决定能不能按行暂存 */
  kind?: string
  binary?: boolean
}

/** 在中间打开的提交（或储藏） */
export interface GitCommitTarget {
  /** 仓库相对会话目录的路径 */
  repo: string
  hash: string
  short: string
  subject: string
  stash: boolean
}

/** 给对话、文件标签用的改动：路径相对会话目录 */
export interface GitChange {
  path: string
  added: number
  deleted: number
}

export const repoPath = (repo: string, path: string) => (repo ? `${repo}/${path}` : path)
export const baseName = (p: string) => p.slice(p.lastIndexOf('/') + 1) || p
export const dirName = (p: string) => (p.includes('/') ? p.slice(0, p.lastIndexOf('/')) : '')

export function createGitStore(opts: {
  sessionId: () => string
  /** 做完会改东西的操作：文件树、打开的文件跟着重读 */
  onChanged: () => void
  /** 打开、引用文件：路径相对会话目录 */
  openFile: (path: string) => void
  /** 在中间打开一个提交 */
  openCommit: (c: GitCommitTarget) => void
  reference: (path: string) => void
}) {
  const repos = ref<AgentGitStatus[]>([])
  const loaded = ref(false)
  const loading = ref(false)
  const repoKey = ref('')
  const repo = computed(() => repos.value.find((r) => r.repo === repoKey.value) ?? repos.value[0])
  const view = ref<GitView>('changes')
  /** 历史只看这个文件（相对仓库；空 = 全部） */
  const historyPath = ref('')
  const diff = ref<GitDiffTarget | null>(null)
  /** 中间正开着的提交的哈希（历史、储藏里把它标出来） */
  const opened = ref('')
  /** 当前视图下面显示的 diff：更改里看工作区、暂存区、冲突的，历史里看提交里的，储藏里看储藏里的 */
  const diffShown = computed(() => {
    const d = diff.value
    if (!d || d.repo !== repo.value?.repo) return null
    const fits = view.value === 'changes' ? ['unstaged', 'staged', 'conflict'].includes(d.source) : view.value === 'history' ? d.source === 'commit' : d.source === 'stash'
    return fits ? d : null
  })
  /** 正在跑的写操作（按钮转圈、别的写操作先等着） */
  const busy = ref('')
  /** 每做完一次会改东西的操作加一：历史、储藏、分支列表跟着重读 */
  const rev = ref(0)
  /** 提交说明：每个会话、每个仓库各一份 */
  const messages = reactive(new Map<string, string>())
  const msgKey = () => `${opts.sessionId()}|${repo.value?.repo ?? ''}`
  const message = computed({
    get: () => messages.get(msgKey()) ?? '',
    set: (v: string) => void messages.set(msgKey(), v),
  })

  let seq = 0
  const load = async (manual = false) => {
    const id = ++seq
    loading.value = true
    try {
      const { data } = await getAgentGitStatus(opts.sessionId())
      if (id !== seq) return
      repos.value = data.repos ?? []
      loaded.value = true
      if (!repos.value.some((r) => r.repo === repoKey.value)) repoKey.value = repos.value[0]?.repo ?? ''
      if (manual) feedback.success(translate('agent.git.refreshed'))
    } catch {
      if (id === seq) loaded.value = true
    } finally {
      if (id === seq) loading.value = false
    }
  }
  const reset = () => {
    seq++
    repos.value = []
    loaded.value = false
    repoKey.value = ''
    diff.value = null
    view.value = 'changes'
    historyPath.value = ''
  }

  /** 跑一个会改东西的操作：转圈、成功了提示一句，完了不管成没成功都重读状态（失败的原因请求层已经提示过） */
  const run = async <T>(key: string, fn: () => Promise<T>, ok?: string | ((r: T) => string)): Promise<T | undefined> => {
    if (busy.value) return undefined
    busy.value = key
    try {
      const r = await fn()
      const text = typeof ok === 'function' ? ok(r) : ok
      if (text) feedback.success(text)
      return r
    } catch {
      return undefined
    } finally {
      busy.value = ''
      rev.value++
      await load()
      opts.onChanged()
    }
  }

  // —— 浮层 ——
  const menu = ref<{ x: number; y: number; items: GitMenuItem[] } | null>(null)
  /** 菜单刚被点在外面关掉时点的是什么、什么时候（点的就是打开它的按钮时，按钮这一下不再打开） */
  let closedBy: { target: EventTarget | null; at: number } = { target: null, at: 0 }
  const menuClosedBy = (target: EventTarget | null) => {
    closedBy = { target, at: performance.now() }
  }
  /** 右键时在鼠标处弹出；点按钮时挂在按钮下面，再点一次同一个按钮是关上 */
  const openMenu = (at: MouseEvent | Element, items: GitMenuItem[]) => {
    if (at instanceof MouseEvent && at.type === 'contextmenu') {
      menu.value = { x: at.clientX, y: at.clientY, items }
      return
    }
    const el = (at instanceof MouseEvent ? at.currentTarget : at) as Element | null
    if (el && closedBy.target instanceof Node && el.contains(closedBy.target) && performance.now() - closedBy.at < 500) {
      closedBy = { target: null, at: 0 }
      return
    }
    const r = el?.getBoundingClientRect()
    menu.value = r ? { x: r.left, y: r.bottom + 4, items } : { x: 0, y: 0, items }
  }
  const askState = ref<GitAsk | null>(null)
  const ask = (a: GitAsk) => {
    askState.value = a
  }
  const promptState = ref<GitPrompt | null>(null)
  const prompt = (p: GitPrompt) => {
    promptState.value = p
  }

  // —— 汇总 ——
  /** 一个仓库里有改动的文件数（同一个文件暂存了一部分也只算一个） */
  const changedCount = (r: AgentGitStatus | undefined) => {
    if (!r) return 0
    return new Set([...r.staged, ...r.unstaged, ...r.conflicted].map((f) => f.path)).size
  }
  const total = computed(() => repos.value.reduce((n, r) => n + changedCount(r), 0))
  /** 每个文件相对 HEAD 大概改了多少行（暂存的加没暂存的），路径相对会话目录 */
  const changes = computed<GitChange[]>(() => {
    const out = new Map<string, GitChange>()
    for (const r of repos.value) {
      for (const f of [...r.staged, ...r.unstaged, ...r.conflicted]) {
        const path = repoPath(r.repo, f.path)
        const c = out.get(path) ?? { path, added: 0, deleted: 0 }
        c.added += Number(f.added || 0)
        c.deleted += Number(f.deleted || 0)
        out.set(path, c)
      }
    }
    return [...out.values()]
  })

  /** 对话里点「完整改动」：找到文件所在的仓库，选中它（有没暂存的改动看没暂存的，否则看暂存的） */
  const reveal = (sessionPath: string): boolean => {
    const owners = repos.value
      .filter((r) => !r.repo || sessionPath.startsWith(`${r.repo}/`))
      .sort((a, b) => b.repo.length - a.repo.length)
    for (const r of owners) {
      const inner = r.repo ? sessionPath.slice(r.repo.length + 1) : sessionPath
      const find = (list: AgentGitFile[]) => list.find((f) => f.path === inner)
      const hit =
        (find(r.conflicted) && { f: find(r.conflicted)!, source: 'conflict' as const }) ||
        (find(r.unstaged) && { f: find(r.unstaged)!, source: 'unstaged' as const }) ||
        (find(r.staged) && { f: find(r.staged)!, source: 'staged' as const })
      if (!hit) continue
      repoKey.value = r.repo
      view.value = 'changes'
      diff.value = { repo: r.repo, source: hit.source, path: hit.f.path, origPath: hit.f.origPath, kind: hit.f.kind, binary: hit.f.binary }
      return true
    }
    return false
  }

  return {
    sessionId: opts.sessionId,
    repos,
    loaded,
    loading,
    repoKey,
    repo,
    view,
    historyPath,
    diff,
    diffShown,
    openFile: opts.openFile,
    openCommit: opts.openCommit,
    opened,
    reference: opts.reference,
    busy,
    rev,
    message,
    load,
    reset,
    run,
    menu,
    openMenu,
    menuClosedBy,
    askState,
    ask,
    promptState,
    prompt,
    changedCount,
    total,
    changes,
    reveal,
  }
}

export type GitStore = ReturnType<typeof createGitStore>
export const GIT_KEY: InjectionKey<GitStore> = Symbol('agent-git')
export const useGit = (): GitStore => {
  const store = inject(GIT_KEY)
  if (!store) throw new Error('useGit() 只能在 Git 面板里用')
  return store
}

/** 复制到剪贴板并提示 */
export const copyText = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    feedback.success(translate('agent.git.copied'))
  } catch {
    feedback.error(translate('agent.git.copyFailed'))
  }
}
