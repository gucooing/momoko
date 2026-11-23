<!-- 对话记录：用户消息、Agent 的回答（Markdown）、思考、工具调用（带命令输出）、改动、批准卡片的四种状态、
     每轮结束的摘要（改了哪些文件、用时、用量）、子 Agent、排着队还没发的消息。
     连着的工具调用、思考折成一行摘要（「已运行 3 个命令，已读取 2 个文件」），点开看每一项；
     改动是一行「已编辑 文件名 +增 −删」，点开看 diff。
     滚动在这里：停在底部时有新内容就跟着往下走，往上翻了就不打扰。 -->
<template>
  <div
    ref="scroller"
    class="agc-log ag-noscroll"
    role="log"
    :aria-label="sub ? t('agent.code.sub.log') : t('agent.code.log')"
    @scroll="onScroll"
  >
    <div ref="content" class="agc-log__in">
      <div v-if="!items.length && !busy" class="agc-log__empty">
        {{ empty || t('agent.code.noMessages') }}
      </div>
      <div v-if="loadingMore" class="agc-log__more ag-muted">{{ t('agent.common.loading') }}</div>

      <template v-for="block in blocks" :key="block.key">
        <div v-if="block.user" class="agc-umsg">
          <div class="agc-user" :class="{ 'agc-user--sub': sub }">
          <div v-if="sub" class="agc-user__label">{{ t('agent.code.sub.task') }}</div>
          <div class="agc-user__text">{{ block.user.text }}</div>
          <div v-if="block.user.attachments?.length" class="agc-user__atts">
            <template v-for="(a, i) in block.user.attachments" :key="i">
              <AttImage v-if="a.kind === 'image' && a.file && sessionId" :name="a.name" :session-id="sessionId" :file="a.file" />
              <span v-else class="agc-att">
                <UIcon :name="attIcon(a.kind)" />
                <span :class="{ 'ag-mono': a.kind === 'ref' || a.kind === 'snippet' }">{{ a.name }}</span>
              </span>
            </template>
          </div>
          </div>
          <div class="agc-meta">
            <span v-if="block.user.steer" class="ag-muted agc-small">{{ t('agent.code.composer.steered') }}</span>
            <span class="agc-meta__time" :title="fullTime(block.user.at)">{{ clockTime(block.user.at) }}</span>
            <button
              v-if="block.user.text"
              type="button"
              class="agc-sm"
              :aria-label="copied === block.key ? t('agent.common.copied') : t('agent.common.copy')"
              :title="copied === block.key ? t('agent.common.copied') : t('agent.common.copy')"
              @click="copy(block.key, block.user.text)"
            >
              <UIcon :name="copied === block.key ? 'i-lucide-check' : 'i-lucide-copy'" />
            </button>
          </div>
        </div>

        <div v-else class="agc-agent">
          <div v-if="!sub" class="agc-agent__head">
            <AgAgentTag :kind="agentKind" :name="agentName" />
          </div>

          <template v-for="row in rowsByBlock.get(block.key) ?? []" :key="row.key">
            <!-- 折起来的一组工具调用、思考 -->
            <button
              v-if="row.group"
              type="button"
              class="agc-fold"
              :aria-expanded="groupOpen(row.group)"
              @click="toggleGroup(row.group)"
            >
              <span v-if="row.group.active" class="agc-sd agc-sd--run agc-pulse" />
              <span class="agc-fold__t">{{ groupText(row.group) }}</span>
              <span v-if="row.group.active && !groupOpen(row.group)" class="ag-ellipsis agc-fold__now">{{
                groupNow(row.group)
              }}</span>
              <UIcon :name="groupOpen(row.group) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" />
            </button>
            <div v-else class="agc-ri" :class="{ 'agc-ri--grp': row.inGroup }">
            <template v-for="item in [row.item]" :key="item.seq">
            <!-- 思考 -->
            <div v-if="item.type === 'thought'" class="agc-think">
              <button
                type="button"
                class="agc-think__btn"
                :aria-expanded="thoughtOpen.has(item.seq)"
                @click="toggle(thoughtOpen, item.seq)"
              >
                <UIcon name="i-lucide-lightbulb" />
                <span :class="{ 'agc-pulse': item.open }">
                  {{
                    item.open
                      ? t('agent.code.thinking')
                      : t('agent.code.thought', { time: fmtDuration(item.ms || 0) })
                  }}
                </span>
                <UIcon
                  :name="
                    thoughtOpen.has(item.seq) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'
                  "
                />
              </button>
              <div v-if="thoughtOpen.has(item.seq)" class="agc-think__body">{{ item.text }}</div>
            </div>

            <!-- 回答 -->
            <MdBlock v-else-if="item.type === 'assistant'" :text="item.text || ''" />

            <!-- 工具调用 -->
            <template v-else-if="item.type === 'tool' && item.tool">
              <template v-for="d in item.tool.diffs ?? []" :key="`${item.seq}-${d.path}`">
                <PatchCard
                  v-if="d.op"
                  :diff="d"
                  :path="rel(d.path)"
                  :old-path="d.oldPath ? rel(d.oldPath) : ''"
                  @show-diff="emit('showDiff', $event)"
                />
                <DiffCard v-else :diff="d" :path="rel(d.path)" @show-diff="emit('showDiff', $event)" />
              </template>
              <div v-if="!item.tool.diffs?.length" class="agc-toolwrap">
                <div class="agc-toolrow">
                  <button
                    type="button"
                    class="agc-tool"
                    :class="{ 'is-click': !!item.tool.output || !!item.tool.child }"
                    :aria-expanded="
                      item.tool.output ? outputOpen.has(item.seq) || running(item.tool) : undefined
                    "
                    @click="onTool(item)"
                  >
                    <UIcon :name="toolIcon(item.tool.kind)" class="ag-muted" />
                    <span v-if="verb(item.tool.kind)" class="agc-tool__verb">{{
                      verb(item.tool.kind)
                    }}</span>
                    <span
                      class="ag-ellipsis agc-tool__arg"
                      :class="{ 'ag-mono': !!toolArg(item.tool).mono }"
                      >{{ toolArg(item.tool).text }}</span
                    >
                    <UIcon v-if="item.tool.child" name="i-lucide-bot" class="ag-muted" />
                    <span v-if="item.tool.exitCode" class="agc-tool__meta agc-bad ag-num">{{
                      t('agent.code.tool.exit', { code: item.tool.exitCode })
                    }}</span>
                    <span v-if="item.tool.status === 'paused'" class="agc-tool__meta ag-muted">{{
                      t('agent.code.tool.paused')
                    }}</span>
                    <span v-if="inBg(item.tool)" class="agc-tool__meta ag-muted">{{
                      t('agent.code.tool.background')
                    }}</span>
                    <UIcon
                      :name="inBg(item.tool) ? 'i-lucide-loader-circle' : statusIcon(item.tool.status)"
                      :class="inBg(item.tool) ? 'ag-spin ag-muted' : statusCls(item.tool.status)"
                    />
                  </button>
                  <button
                    v-if="item.tool.cancel"
                    type="button"
                    class="agc-sm"
                    :aria-label="t('agent.code.tool.stop')"
                    :title="t('agent.code.tool.stop')"
                    :disabled="stopping.has(`${item.sub || ''}:${item.tool.id}`)"
                    @click="stop(item.sub || '', item.tool.id)"
                  >
                    <UIcon name="i-lucide-square" />
                  </button>
                </div>
                <pre
                  v-if="item.tool.output && (outputOpen.has(item.seq) || running(item.tool))"
                  class="agc-term ag-noscroll"
                  >{{ item.tool.output }}</pre
                >
                <div v-if="item.tool.images?.length && sessionId" class="agc-toolimgs">
                  <AttImage
                    v-for="f in item.tool.images"
                    :key="f"
                    :name="f.slice(f.lastIndexOf('/') + 1)"
                    :session-id="sessionId"
                    :file="f"
                  />
                </div>
              </div>
            </template>

            <!-- 批准卡片 -->
            <template v-else-if="item.type === 'permission' && item.permission">
              <div
                v-if="!item.permission.outcome"
                class="agc-perm"
                role="group"
                :aria-label="t('agent.code.perm.need')"
              >
                <div class="agc-perm__head">
                  <UIcon name="i-lucide-shield" />
                  <span>{{ t('agent.code.perm.need') }}</span>
                  <span class="agc-perm__who">{{ wants(item.permission.kind) }}</span>
                </div>
                <div class="agc-perm__cmd ag-mono">
                  <span v-if="item.permission.command" class="ag-muted">$</span>
                  <span class="ag-ellipsis">{{
                    item.permission.command || item.permission.title
                  }}</span>
                  <span v-if="cwd" class="agc-perm__cwd ag-muted">{{ cwd }}</span>
                </div>
                <div
                  v-if="item.permission.command && item.permission.title"
                  class="agc-perm__title ag-muted"
                >
                  {{ item.permission.title }}
                </div>
                <div v-if="item.permission.description" class="agc-perm__title ag-muted">
                  {{ item.permission.description }}
                </div>
                <div class="agc-perm__acts">
                  <button
                    v-for="(o, i) in allowOptions(item.permission.options)"
                    :key="o.id"
                    type="button"
                    class="ag-btn ag-btn--sm"
                    :class="i === 0 ? 'ag-btn--p' : ''"
                    :disabled="resolving.has(item.permission.id)"
                    @click="resolve(item.permission.id, o.id)"
                  >
                    {{ permOptionLabel(o, item.permission.options) }}
                  </button>
                  <span class="agc-flex" />
                  <button
                    v-for="o in rejectOptions(item.permission.options)"
                    :key="o.id"
                    type="button"
                    class="ag-btn ag-btn--o ag-btn--sm"
                    :disabled="resolving.has(item.permission.id)"
                    @click="resolve(item.permission.id, o.id)"
                  >
                    {{ permOptionLabel(o, item.permission.options) }}
                  </button>
                </div>
              </div>
              <div v-else class="agc-tool agc-tool--flat">
                <UIcon :name="permIcon(item.permission)" :class="permCls(item.permission)" />
                <span class="agc-tool__verb">{{ permOutcome(item.permission) }}</span>
                <span
                  class="ag-ellipsis agc-tool__arg"
                  :class="{ 'ag-mono': !!item.permission.command }"
                >
                  {{
                    item.permission.outcome === 'cancelled'
                      ? t('agent.code.perm.stoppedHint')
                      : item.permission.command || item.permission.title
                  }}
                </span>
              </div>
            </template>

            <!-- 提问卡片 -->
            <QuestionCard
              v-else-if="item.type === 'question' && item.question"
              :question="item.question"
              :agent-name="agentName"
              :busy="resolving.has(item.question.id)"
              @answer="(action, answers) => answer(item.question!.id, action, answers)"
            />

            <!-- 子 Agent -->
            <div v-else-if="item.type === 'subagent' && item.subagent" class="agc-toolrow">
              <button type="button" class="agc-tool is-click" @click="emit('openSub', item.subagent.id)">
                <span v-if="item.subagent.state === 'running'" class="agc-sd agc-sd--run agc-pulse" />
                <UIcon v-else name="i-lucide-bot" class="ag-muted" />
                <span class="agc-tool__verb">{{ t('agent.code.sub.name') }}</span>
                <span class="ag-ellipsis agc-tool__arg">{{
                  item.subagent.title || item.subagent.description || item.subagent.id
                }}</span>
                <span class="ag-muted agc-tool__meta">{{ subState(item.subagent.state) }}</span>
                <UIcon name="i-lucide-chevron-right" class="ag-muted" />
              </button>
              <button
                v-if="item.subagent.cancel && item.subagent.state !== 'idle'"
                type="button"
                class="agc-sm"
                :aria-label="t('agent.code.sub.stop')"
                :title="t('agent.code.sub.stop')"
                :disabled="stopping.has(`${item.subagent.id}:`)"
                @click="stop(item.subagent.id, '')"
              >
                <UIcon name="i-lucide-square" />
              </button>
            </div>

            <!-- 会话之间的消息 -->
            <div v-else-if="item.type === 'relay'" class="agc-relay">
              <div class="agc-relay__head ag-muted">
                <UIcon name="i-lucide-arrow-left-right" />
                <span>{{ relayLabel(item) }}</span>
              </div>
              <MdBlock :text="item.text || ''" />
            </div>

            <!-- 提示 -->
            <div
              v-else-if="item.type === 'notice'"
              class="agc-notice"
              :class="`agc-notice--${item.level || 'info'}`"
            >
              <UIcon
                :name="
                  item.level === 'error'
                    ? 'i-lucide-circle-x'
                    : item.level === 'warning'
                      ? 'i-lucide-triangle-alert'
                      : 'i-lucide-info'
                "
              />
              <span v-if="item.code">{{
                t(`agent.code.notice.${item.code}`, { reason: item.text || '' })
              }}</span>
              <span v-else
                ><b v-if="item.title">{{ item.title }}</b
                ><template v-if="item.title && item.text"> · </template>{{ item.text }}</span
              >
            </div>

            <!-- 一轮结束：这一轮改了哪些文件，再是结果、用时、用量 -->
            <template v-else-if="item.type === 'turn' && item.turn">
              <div v-if="block.changed.length" class="agc-tfiles">
                <button
                  type="button"
                  class="agc-tfiles__head"
                  :aria-expanded="!filesShut.has(item.seq)"
                  @click="toggle(filesShut, item.seq)"
                >
                  <UIcon name="i-lucide-file-diff" class="ag-muted" />
                  <span class="agc-flex">{{ t('agent.code.turn.changed', { n: block.changed.length }) }}</span>
                  <span class="agc-plus ag-num">+{{ block.added }}</span>
                  <span class="agc-minus ag-num">−{{ block.deleted }}</span>
                  <UIcon
                    :name="filesShut.has(item.seq) ? 'i-lucide-chevron-right' : 'i-lucide-chevron-down'"
                    class="ag-muted"
                  />
                </button>
                <div v-if="!filesShut.has(item.seq)" class="agc-tfiles__list">
                  <button
                    v-for="f in block.changed"
                    :key="f.path"
                    type="button"
                    class="agc-fr"
                    :title="f.path"
                    @click="emit('showDiff', f.path)"
                  >
                    <span class="ag-mono agc-fr__k" :class="opCls(f.op)">{{ opLetter(f.op) }}</span>
                    <span class="ag-mono ag-ellipsis agc-flex">{{ f.path }}</span>
                    <span class="agc-plus ag-num">+{{ f.added }}</span>
                    <span class="agc-minus ag-num">−{{ f.deleted }}</span>
                  </button>
                </div>
              </div>
              <div class="agc-turn">
                <UIcon :name="turnIcon(item.turn.stop)" :class="turnCls(item.turn.stop)" />
                <span>{{ turnText(item.turn) }}</span>
                <button
                  v-if="item.turn.usage?.total"
                  type="button"
                  class="agc-turn__tok"
                  :aria-expanded="tokenOpen.has(item.seq)"
                  @click="toggle(tokenOpen, item.seq)"
                >
                  · <span class="ag-num">{{ fmtTokens(item.turn.usage.total) }}</span> token
                  <UIcon
                    :name="
                      tokenOpen.has(item.seq) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'
                    "
                  />
                </button>
              </div>
              <div v-if="item.turn.usage && tokenOpen.has(item.seq)" class="agc-turn__usage ag-num">
                <span>{{
                  t('agent.code.usage.input', { n: fmtTokens(inputOf(item.turn.usage)) })
                }}</span>
                <span>{{
                  t('agent.code.usage.cacheHit', { n: fmtTokens(item.turn.usage.cacheRead) })
                }}</span>
                <span>{{
                  t('agent.code.usage.output', { n: fmtTokens(item.turn.usage.output) })
                }}</span>
                <span v-if="item.turn.usage.thought">{{
                  t('agent.code.usage.thought', { n: fmtTokens(item.turn.usage.thought) })
                }}</span>
              </div>
              <div v-if="item.turn.code || item.turn.error" class="agc-notice agc-notice--error">
                <UIcon name="i-lucide-circle-x" />
                <span>{{
                  item.turn.code
                    ? t(`agent.code.turnError.${item.turn.code}`, { reason: item.turn.error || '' })
                    : item.turn.error
                }}</span>
              </div>
            </template>
            </template>
            </div>
          </template>
          <div v-if="!(busy && block === blocks[blocks.length - 1])" class="agc-meta agc-meta--agent">
            <button
              v-if="block.text"
              type="button"
              class="agc-sm"
              :aria-label="copied === block.key ? t('agent.common.copied') : t('agent.common.copy')"
              :title="copied === block.key ? t('agent.common.copied') : t('agent.common.copy')"
              @click="copy(block.key, block.text)"
            >
              <UIcon :name="copied === block.key ? 'i-lucide-check' : 'i-lucide-copy'" />
            </button>
            <span class="agc-meta__time" :title="fullTime(block.at)">{{ clockTime(block.at) }}</span>
          </div>
        </div>
      </template>

      <div v-if="busy && !streaming" class="agc-working">
        <span class="agc-sd agc-sd--run agc-pulse" />{{ t('agent.code.working') }}
      </div>

      <!-- 排着队还没发的消息：这一轮结束后发，或者点「立即发送」 -->
      <div v-for="q in queued" :key="`q${q.id}`" class="agc-umsg agc-umsg--queued">
        <div class="agc-user">
          <div v-if="q.text" class="agc-user__text">{{ q.text }}</div>
          <div v-if="q.attachments?.length" class="agc-user__atts">
            <span v-for="(a, i) in q.attachments" :key="i" class="agc-att">
              <UIcon :name="attIcon(a.kind)" />
              <span :class="{ 'ag-mono': a.kind === 'ref' || a.kind === 'snippet' }">{{ a.name }}</span>
            </span>
          </div>
        </div>
        <div class="agc-meta">
          <span class="agc-queued-tag" :title="t('agent.code.queue.waitingTip')">
            <UIcon name="i-lucide-clock" />{{ t('agent.code.queue.waiting') }}
          </span>
          <button
            v-if="q.text"
            type="button"
            class="agc-sm"
            :aria-label="copied === `q${q.id}` ? t('agent.common.copied') : t('agent.common.copy')"
            :title="copied === `q${q.id}` ? t('agent.common.copied') : t('agent.common.copy')"
            @click="copy(`q${q.id}`, q.text)"
          >
            <UIcon :name="copied === `q${q.id}` ? 'i-lucide-check' : 'i-lucide-copy'" />
          </button>
          <button
            type="button"
            class="agc-sm"
            :aria-label="t('agent.code.queue.remove')"
            :title="t('agent.code.queue.remove')"
            :disabled="queueBusy.has(q.id)"
            @click="onQueue('unqueue', q.id)"
          >
            <UIcon name="i-lucide-x" />
          </button>
          <button
            type="button"
            class="agc-sendnow"
            :aria-label="t('agent.code.queue.sendNow')"
            :title="t('agent.code.queue.sendNowTip')"
            :disabled="queueBusy.has(q.id)"
            @click="onQueue('sendNow', q.id)"
          >
            {{ t('agent.code.queue.sendNow') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="!atBottom" class="agc-log__jump">
      <button
        type="button"
        class="agc-jump"
        :aria-label="t('agent.code.toLatest')"
        :title="t('agent.code.toLatest')"
        @click="toLatest"
      >
        <UIcon name="i-lucide-arrow-down" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useResizeObserver } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import AgAgentTag from '@/views/agent/components/AgAgentTag.vue'
import { copyTextToClipboard } from '@/utils/file'
import { clockTime, fmtDuration, fmtTokens } from '@/views/agent/utils/format'
import { isRejectOption, permOptionLabel } from '@/views/agent/utils/labels'
import { diffDelta, diffOp } from '@/views/agent/utils/diff'
import AttImage from './AttImage.vue'
import DiffCard from './DiffCard.vue'
import MdBlock from './MdBlock.vue'
import PatchCard from './PatchCard.vue'
import QuestionCard from './QuestionCard.vue'
import type {
  AgentItem,
  AgentPermission,
  AgentPermOption,
  AgentQueued,
  AgentTool,
  AgentTurn,
  AgentTurnUsage,
} from './types'

const props = withDefaults(
  defineProps<{
    items: AgentItem[]
    /** 消息里的图片附件按它向后端取 */
    sessionId?: string
    agentKind?: string
    agentName?: string
    /** 会话目录：批准卡片上显示，工具里的绝对路径按它缩短 */
    cwd?: string
    /** 一轮正在进行 */
    busy?: boolean
    /** 转到后台还在跑的工具调用 id（工具调用本身报了完成，命令其实还在跑） */
    background?: string[]
    /** 子 Agent 的对话 */
    sub?: boolean
    empty?: string
    /** 更早的事件还有没拿；loadingMore 正在拿 */
    more?: boolean
    loadingMore?: boolean
    /** 排着队还没发的消息 */
    queued?: AgentQueued[]
  }>(),
  {
    sessionId: '',
    agentKind: '',
    agentName: '',
    cwd: '',
    busy: false,
    background: () => [],
    sub: false,
    empty: '',
    more: false,
    loadingMore: false,
    queued: () => [],
  },
)
const emit = defineEmits<{
  resolve: [requestId: string, optionId: string]
  /** 回答一张提问卡片：accept 带各字段的值，decline 是跳过 */
  answer: [questionId: string, action: 'accept' | 'decline', answers: Record<string, string[]>]
  /** 只停一项（ACP v2）：toolId 不为空停这个工具调用（sub 是它所在的子 Agent），否则停 sub 这个子 Agent */
  stop: [sub: string, toolId: string]
  showDiff: [path: string]
  openSub: [id: string]
  more: []
  /** 撤回一条排着队的消息 */
  unqueue: [queueId: string]
  /** 排着队的一条马上发 */
  sendNow: [queueId: string]
}>()
const { t } = useI18n()

// —— 分块：一条用户消息，接着是 Agent 的一段 ——
/** 一轮里改过的一个文件：同一个文件的几次改动合在一起 */
interface ChangedFile {
  path: string
  /** 第一次是新建就算新建，最后一次是删除就算删除，其它算修改 */
  op: string
  added: number
  deleted: number
}
interface Block {
  key: string
  user?: AgentItem
  items: AgentItem[]
  /** 这一段里工具调用改过的文件 */
  changed: ChangedFile[]
  added: number
  deleted: number
  /** Agent 这一段的回答文字（复制用） */
  text: string
  /** 这一段最后一条的时间 */
  at: number
}
/** 提问卡片对应的那次工具调用（AskUserQuestion）：它的输出就是卡片上的问题、答案，卡片在了就不再单独显示 */
const askTools = computed(() => new Set(props.items.flatMap((i) => (i.question?.toolId ? [i.question.toolId] : []))))
const blocks = computed<Block[]>(() => {
  const out: Block[] = []
  let cur: Block | null = null
  let files = new Map<string, ChangedFile>()
  for (const item of props.items) {
    if (item.type === 'tool' && item.tool && askTools.value.has(item.tool.id)) continue
    if (item.type === 'user') {
      out.push({ key: `u${item.seq}`, user: item, items: [], changed: [], added: 0, deleted: 0, text: '', at: item.at })
      cur = null
      continue
    }
    if (!cur) {
      cur = { key: `a${item.seq}`, items: [], changed: [], added: 0, deleted: 0, text: '', at: item.at }
      files = new Map()
      out.push(cur)
    }
    cur.items.push(item)
    cur.at = item.at
    if (item.type === 'assistant' && !item.sub && item.text) cur.text = cur.text ? `${cur.text}

${item.text}` : item.text
    const diffs = item.tool?.diffs ?? []
    if (!diffs.length) continue
    for (const d of diffs) {
      const path = rel(d.path)
      const op = diffOp(d)
      const { added, deleted } = diffDelta(d)
      const f = files.get(path)
      if (!f) {
        files.set(path, { path, op, added, deleted })
        continue
      }
      f.added += added
      f.deleted += deleted
      if (op === 'delete') f.op = 'delete'
      else if (f.op === 'delete') f.op = 'modify'
    }
    cur.changed = [...files.values()]
    cur.added = cur.changed.reduce((n, f) => n + f.added, 0)
    cur.deleted = cur.changed.reduce((n, f) => n + f.deleted, 0)
  }
  return out
})

// —— 折叠：连着的工具调用、思考、处理过的批准折成一行摘要 ——
interface Group {
  key: string
  items: AgentItem[]
  /** 还在进行：有正在跑的工具调用、正在写的思考，或者是正在进行的这一轮最后一段 */
  active: boolean
}
interface Row {
  key: string
  item: AgentItem
  /** 这一行是一组的摘要 */
  group?: Group
  /** 展开着的一组里的一项 */
  inGroup?: boolean
}
/** 能折进一组的：没有改动的工具调用（改动单独一行）、思考、已经处理过的批准和提问 */
const foldable = (item: AgentItem) =>
  (item.type === 'tool' && !!item.tool && !item.tool.diffs?.length) ||
  item.type === 'thought' ||
  (item.type === 'permission' && !!item.permission?.outcome) ||
  (item.type === 'question' && !!item.question?.outcome)
/** 手动展开、收起过的组；没动过的都收着 */
const groupState = reactive(new Map<string, boolean>())
const groupOpen = (g: Group) => groupState.get(g.key) ?? false
const toggleGroup = (g: Group) => groupState.set(g.key, !groupOpen(g))
const rowsOf = (items: AgentItem[], live: boolean): Row[] => {
  const out: Row[] = []
  let run: AgentItem[] = []
  const flush = (atEnd: boolean) => {
    if (run.length === 1) out.push({ key: `i${run[0]!.seq}`, item: run[0]! })
    else if (run.length > 1) {
      const active = run.some((i) => i.open || (i.type === 'tool' && running(i.tool))) || (atEnd && live)
      const g: Group = { key: `g${run[0]!.seq}`, items: run, active }
      out.push({ key: g.key, item: run[0]!, group: g })
      if (groupOpen(g)) for (const i of run) out.push({ key: `i${i.seq}`, item: i, inGroup: true })
    }
    run = []
  }
  for (const item of items) {
    if (foldable(item)) {
      run.push(item)
      continue
    }
    flush(false)
    out.push({ key: `i${item.seq}`, item })
  }
  flush(true)
  return out
}
const rowsByBlock = computed(() => {
  const last = blocks.value[blocks.value.length - 1]
  return new Map(blocks.value.map((b) => [b.key, rowsOf(b.items, props.busy && b === last)]))
})
const GROUP_KINDS = ['execute', 'read', 'edit', 'search', 'fetch']
/** 一组的摘要：「已运行 3 个命令，已读取 2 个文件」 */
const groupText = (g: Group) => {
  const counts = new Map<string, number>()
  let failed = 0
  let thoughts = 0
  let thoughtMs = 0
  for (const i of g.items) {
    if (i.type === 'tool' && i.tool) {
      const k = GROUP_KINDS.includes(i.tool.kind || '') ? i.tool.kind! : 'other'
      counts.set(k, (counts.get(k) ?? 0) + 1)
      if (i.tool.status === 'failed') failed++
    } else if (i.type === 'thought') {
      thoughts++
      thoughtMs += i.ms || 0
    }
  }
  const parts = [...GROUP_KINDS, 'other']
    .filter((k) => counts.get(k))
    .map((k) => t(`agent.code.group.${k}`, { n: counts.get(k) }))
  if (!parts.length && thoughts) parts.push(t('agent.code.thought', { time: fmtDuration(thoughtMs) }))
  if (!parts.length) parts.push(t('agent.code.group.steps', { n: g.items.length }))
  if (failed) parts.push(t('agent.code.group.failed', { n: failed }))
  return parts.join(t('agent.code.group.sep'))
}
/** 进行中的一组收着时，摘要后面带上正在做的那一项 */
const groupNow = (g: Group) => {
  for (let i = g.items.length - 1; i >= 0; i--) {
    const item = g.items[i]!
    if (item.type === 'tool' && item.tool) return [verb(item.tool.kind), toolArg(item.tool).text].filter(Boolean).join(' ')
    if (item.type === 'thought' && item.open) return t('agent.code.thinking')
  }
  return ''
}

// —— 改动的文件 ——
/** 收起的「这一轮改了哪些文件」（默认展开） */
const filesShut = reactive(new Set<number>())
const OP_LETTER: Record<string, string> = { add: 'A', delete: 'D', modify: 'M', move: 'R', copy: 'C' }
const opLetter = (op: string) => OP_LETTER[op] || 'M'
const opCls = (op: string) => (op === 'add' ? 'agc-ok' : op === 'delete' ? 'agc-bad' : 'agc-warn')
const ATT_ICON: Record<string, string> = { image: 'i-lucide-image', paste: 'i-lucide-file-text', snippet: 'i-lucide-text-select', ref: 'i-lucide-at-sign' }
const attIcon = (kind?: string) => ATT_ICON[kind ?? ''] ?? 'i-lucide-file'

// —— 排着队的消息 ——
const queueBusy = reactive(new Set<string>())
const onQueue = (kind: 'unqueue' | 'sendNow', id: string) => {
  queueBusy.add(id)
  if (kind === 'unqueue') emit('unqueue', id)
  else emit('sendNow', id)
  setTimeout(() => queueBusy.delete(id), 3000)
}

// —— 复制、时间 ——
/** 刚复制过的那一块：按钮短暂显示成对勾 */
const copied = ref('')
let copiedTimer: ReturnType<typeof setTimeout> | undefined
const copy = async (key: string, text: string) => {
  try {
    await copyTextToClipboard(text)
  } catch {
    feedback.error(t('agent.common.copyFailed'))
    return
  }
  copied.value = key
  clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => (copied.value = ''), 1500)
}
const fullTime = (at: number) => (at ? new Date(at).toLocaleString() : '')

/** 最后一条还在往下写：不用再显示「正在处理」 */
const streaming = computed(() => {
  const last = props.items[props.items.length - 1]
  return (
    !!last &&
    (last.open ||
      (last.type === 'tool' && running(last.tool)) ||
      (last.type === 'permission' && !last.permission?.outcome) ||
      (last.type === 'question' && !last.question?.outcome))
  )
})

const thoughtOpen = reactive(new Set<number>())
const tokenOpen = reactive(new Set<number>())
const outputOpen = reactive(new Set<number>())
const resolving = reactive(new Set<string>())
const toggle = (set: Set<number>, seq: number) => (set.has(seq) ? set.delete(seq) : set.add(seq))

const resolve = (requestId: string, optionId: string) => {
  resolving.add(requestId)
  emit('resolve', requestId, optionId)
  setTimeout(() => resolving.delete(requestId), 3000)
}
const answer = (questionId: string, action: 'accept' | 'decline', answers: Record<string, string[]>) => {
  resolving.add(questionId)
  emit('answer', questionId, action, answers)
  setTimeout(() => resolving.delete(questionId), 3000)
}

// —— 只停一项（ACP v2）——
const stopping = reactive(new Set<string>())
const stop = (sub: string, toolId: string) => {
  const key = `${sub}:${toolId}`
  stopping.add(key)
  emit('stop', sub, toolId)
  setTimeout(() => stopping.delete(key), 3000)
}

// —— 会话之间的消息（ACP v2）——
/** 子 Agent 的会话 id → 标题 */
const subTitles = computed(() => {
  const out = new Map<string, string>()
  for (const item of props.items) {
    if (item.subagent) out.set(item.subagent.id, item.subagent.title || item.subagent.description || '')
  }
  return out
})
const peerName = (id?: string) =>
  !id ? '' : id === 'main' ? t('agent.code.relay.main') : subTitles.value.get(id) || t('agent.code.sub.name')
const relayLabel = (item: AgentItem) => {
  const from = peerName(item.from)
  const to = peerName(item.to)
  if (from && to) return t('agent.code.relay.fromTo', { from, to })
  if (from) return t('agent.code.relay.from', { from })
  if (to) return t('agent.code.relay.to', { to })
  return t('agent.code.relay.message')
}

// —— 工具 ——
const KIND_ICON: Record<string, string> = {
  read: 'i-lucide-file-text',
  edit: 'i-lucide-pencil',
  delete: 'i-lucide-trash-2',
  move: 'i-lucide-move-right',
  search: 'i-lucide-search',
  execute: 'i-lucide-square-terminal',
  think: 'i-lucide-lightbulb',
  fetch: 'i-lucide-globe',
  switch_mode: 'i-lucide-toggle-right',
}
const KNOWN_KINDS = Object.keys(KIND_ICON)
const toolIcon = (kind?: string) => KIND_ICON[kind || ''] ?? 'i-lucide-wrench'
const verb = (kind?: string) =>
  kind && KNOWN_KINDS.includes(kind) ? t(`agent.code.tool.${kind}`) : ''
const running = (tool?: AgentTool) =>
  !!tool && (tool.status === 'pending' || tool.status === 'in_progress')
/** 转到后台还在跑：卡片上标出来，不显示成已完成。输出不自动展开（起的服务会一直出日志），右侧「任务」里看 */
const bgSet = computed(() => new Set(props.background))
const inBg = (tool?: AgentTool) => !!tool && bgSet.value.has(tool.id)

/** 工作目录下的绝对路径缩成相对路径 */
const rel = (path: string) => {
  const cwd = props.cwd.replace(/[\\/]+$/, '')
  if (cwd && (path.startsWith(`${cwd}/`) || path.startsWith(`${cwd}\\`)))
    return path.slice(cwd.length + 1).replace(/\\/g, '/')
  return path
}

const toolArg = (tool: AgentTool): { text: string; mono: boolean } => {
  if (tool.command) return { text: tool.command, mono: true }
  const loc = tool.locations?.[0]
  if (loc?.path && verb(tool.kind))
    return { text: rel(loc.path) + (loc.line ? `:${loc.line}` : ''), mono: true }
  return { text: tool.title, mono: false }
}

const statusIcon = (status?: string) => {
  if (status === 'completed') return 'i-lucide-check'
  if (status === 'failed') return 'i-lucide-x'
  if (status === 'cancelled') return 'i-lucide-square'
  if (status === 'paused') return 'i-lucide-pause'
  return 'i-lucide-loader-circle'
}
const statusCls = (status?: string) => {
  if (status === 'completed') return 'agc-ok'
  if (status === 'failed' || status === 'cancelled') return 'agc-bad'
  if (status === 'paused') return 'ag-muted'
  return 'ag-spin ag-muted'
}

const onTool = (item: AgentItem) => {
  const tool = item.tool
  if (!tool) return
  if (tool.child) {
    emit('openSub', tool.child)
    return
  }
  if (tool.output) toggle(outputOpen, item.seq)
}

// —— 批准 ——
const allowOptions = (options: AgentPermOption[]) => options.filter((o) => !isRejectOption(o))
const rejectOptions = (options: AgentPermOption[]) => options.filter((o) => isRejectOption(o))
const wants = (kind?: string) =>
  t(
    KNOWN_KINDS.includes(kind || '')
      ? `agent.code.perm.wants.${kind}`
      : 'agent.code.perm.wants.other',
    { agent: props.agentName },
  )
const permOutcome = (p: AgentPermission) => {
  if (p.outcome === 'cancelled') return t('agent.code.perm.stopped')
  const chosen = p.options.find((o) => o.id === p.optionId)
  if (isRejectOption({ kind: p.optionKind })) return t('agent.code.perm.denied')
  return chosen ? permOptionLabel(chosen, p.options) : t('agent.code.perm.allowed')
}
const permIcon = (p: AgentPermission) =>
  p.outcome === 'cancelled'
    ? 'i-lucide-square'
    : isRejectOption({ kind: p.optionKind })
      ? 'i-lucide-x'
      : 'i-lucide-check'
const permCls = (p: AgentPermission) =>
  p.outcome === 'cancelled' || isRejectOption({ kind: p.optionKind }) ? 'agc-bad' : 'agc-ok'

// —— 子 Agent ——
const subState = (state?: string) =>
  state === 'running'
    ? t('agent.code.sub.running')
    : state === 'requires_action'
      ? t('agent.code.sub.waiting')
      : t('agent.code.sub.done')

// —— 一轮结束 ——
const STOPS = ['end_turn', 'cancelled', 'max_tokens', 'max_turn_requests', 'refusal', 'error']
const turnIcon = (stop: string) =>
  stop === 'end_turn'
    ? 'i-lucide-check'
    : stop === 'cancelled'
      ? 'i-lucide-square'
      : 'i-lucide-circle-alert'
const turnCls = (stop: string) => (stop === 'end_turn' ? 'agc-ok' : 'agc-bad')
const inputOf = (u: AgentTurnUsage) =>
  Number(u.input || 0) + Number(u.cacheRead || 0) + Number(u.cacheWrite || 0)
const turnText = (turn: AgentTurn) =>
  [
    STOPS.includes(turn.stop) ? t(`agent.code.stop.${turn.stop}`) : turn.stop,
    t('agent.code.turn.took', { time: fmtDuration(turn.ms || 0) }),
  ].join(' · ')

// —— 跟随滚动 ——
// 停在底部（atBottom）时内容变高就跟着到底：新消息、Markdown 渲染完、看不见的块排版后变高都算；
// 往上翻了就不动，下面浮一个「回到最新」。
const scroller = ref<HTMLElement | null>(null)
const content = ref<HTMLElement | null>(null)
const atBottom = ref(true)
const onScroll = () => {
  const el = scroller.value
  if (!el) return
  atBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < 80
  // 翻到顶上附近：要更早的一页。
  if (el.scrollTop < 120 && props.more && !props.loadingMore) emit('more')
}

// 更早的一页插到前面时，保持眼前看的内容不动。
let heightBefore = 0
watch(
  () => props.items[0]?.seq,
  (seq, old) => {
    const el = scroller.value
    if (!el || seq == null || old == null || seq >= old) return
    void nextTick(() => {
      el.scrollTop += el.scrollHeight - heightBefore
    })
  },
  { flush: 'pre' },
)
watch(
  () => props.loadingMore,
  (on) => {
    if (on && scroller.value) heightBefore = scroller.value.scrollHeight
  },
)
const toBottom = () => {
  const el = scroller.value
  if (el && atBottom.value) el.scrollTop = el.scrollHeight
}
const toLatest = () => {
  atBottom.value = true
  toBottom()
}
const tail = computed(() => {
  const last = props.items[props.items.length - 1]
  return `${props.items.length}:${last?.rev ?? 0}:${last?.text?.length ?? 0}:${props.busy}:${props.queued.length}`
})
watch(tail, () => nextTick(toBottom), { flush: 'post' })
// 换了会话（记录先清空再来新的）：从最新的开始看
watch(
  () => props.items.length === 0,
  (empty) => {
    if (empty) atBottom.value = true
  },
)
useResizeObserver(content, toBottom)
onMounted(() => {
  atBottom.value = true
  void nextTick(toBottom)
})
// 页面切走再切回来（缓存的页面重新挂上），滚动位置会丢：在底部的回到底部，翻上去的回到原处
let savedTop = 0
onDeactivated(() => {
  savedTop = scroller.value?.scrollTop ?? 0
})
onActivated(() => {
  const el = scroller.value
  if (!el) return
  if (atBottom.value) el.scrollTop = el.scrollHeight
  else el.scrollTop = savedTop
})
</script>
