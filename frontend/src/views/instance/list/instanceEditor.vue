<!-- 应用新建/编辑：PC 右侧抽屉，手机全屏长页（顶部返回 + 底部「仅保存 / 保存并启动」）。三段编号分区：基本信息 / 运行 / 环境变量。
     类型带模板：新建时切换类型自动预填默认启动/停止命令；标签为 chips 输入 + 常用标签；工作目录可从文件管理选择文件夹；
     环境变量逐条编辑（可按密钥保存：只写不读、控制台打码）或文本编辑，可从 .env 导入。
     受控契约：props(visible/mode/loading/submitting/form/typeOptions/allowRun/running) + emits(close/submit)。
     allowRun：显示「保存并启动 / 保存并重启」，submit 第二参数带 { run: true }，由父级执行启动。 -->
<template>
  <Teleport to="body">
    <Transition name="inst-drawer">
      <div v-if="visible" class="inst-drawer" :class="{ 'is-mobile': isMobile }" @mousedown.self="handleClose">
        <section class="inst-drawer__panel" role="dialog" aria-modal="true" aria-labelledby="inst-drawer-title">
          <header class="inst-drawer__head">
            <AppIconButton
              v-if="isMobile"
              icon="HOutline:ChevronLeftIcon"
              :box="44"
              :label="t('instance.ui.backToList')"
              @click="handleClose"
            />
            <span
              v-else
              class="inst-drawer__av m-sticker"
              :class="mode === 'create' ? 'm-sticker--a' : 'm-sticker--p'"
            >
              <UIcon v-if="mode === 'create'" name="i-lucide-plus" class="inst-drawer__av-ico" />
              <template v-else>{{ initial }}</template>
            </span>
            <span class="inst-drawer__titles">
              <h2 id="inst-drawer-title" class="inst-drawer__title">
                {{ mode === 'create' ? t('instance.ui.editorCreateTitle') : t('instance.ui.editorEditTitle') }}
              </h2>
              <span class="inst-drawer__sub">{{ subtitle }}</span>
            </span>
            <AppIconButton v-if="!isMobile" icon="HOutline:XMarkIcon" :label="t('system.common.close')" @click="handleClose" />
          </header>

          <div v-if="loading" class="inst-drawer__loading">{{ t('common.loading') }}…</div>

          <div v-else class="inst-drawer__body">
            <!-- 1 基本信息 -->
            <section class="inst-sec" aria-labelledby="inst-sec-1">
              <div class="inst-sec__head">
                <span class="inst-sec__n is-1">1</span>
                <h3 id="inst-sec-1" class="inst-sec__title">{{ t('instance.ui.secBasic') }}</h3>
              </div>
              <div class="inst-grid">
                <div class="app-field">
                  <label class="app-label app-label--required" for="inst-f-name">{{ t('instance.ui.name') }}</label>
                  <input
                    id="inst-f-name"
                    v-model="localForm.name"
                    class="app-input inst-input"
                    :class="{ 'is-error': errors.name }"
                    maxlength="100"
                    autocomplete="off"
                  />
                  <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
                  <span v-else class="inst-hint">{{ t('instance.ui.nameHint') }}</span>
                </div>
                <div class="app-field">
                  <label class="app-label" :class="{ 'app-label--required': mode === 'create' }">
                    {{ t('instance.ui.appType') }}
                  </label>
                  <AppSelect
                    v-model="localForm.type"
                    :options="typeSelectOptions"
                    :placeholder="t('instance.selectInstanceType')"
                    :error="!!errors.type"
                  />
                  <span v-if="errors.type" class="app-field__error">{{ errors.type }}</span>
                  <span v-else class="inst-hint">
                    {{ t('instance.ui.noFitType') }}
                    <RouterLink to="/instance/type" class="inst-link" @click="handleClose">
                      {{ t('instance.ui.manageTypes') }}
                    </RouterLink>
                  </span>
                </div>
              </div>
              <div class="app-field">
                <label class="app-label" for="inst-f-remark">
                  {{ t('common.remark') }}<span class="inst-opt">{{ t('instance.ui.optional') }}</span>
                </label>
                <textarea id="inst-f-remark" v-model="localForm.remark" class="app-textarea inst-input" rows="2" maxlength="500" />
              </div>
              <div class="app-field">
                <label class="app-label" for="inst-f-tag">{{ t('instance.tags') }}</label>
                <div class="inst-tags" @click="tagInputRef?.focus()">
                  <span v-for="tag in tagList" :key="tag" class="inst-tchip">
                    #{{ tag }}
                    <button type="button" :aria-label="t('instance.ui.removeTag', { tag })" @click.stop="removeTag(tag)">
                      <UIcon name="i-lucide-x" />
                    </button>
                  </span>
                  <input
                    id="inst-f-tag"
                    ref="tagInputRef"
                    v-model="tagDraft"
                    type="text"
                    class="inst-tags__input"
                    enterkeyhint="done"
                    :placeholder="isMobile ? t('instance.ui.tagPlaceholderShort') : t('instance.ui.tagPlaceholder')"
                    @keydown.enter.prevent="commitTag"
                    @keydown="onTagKey"
                    @blur="commitTag"
                  />
                </div>
                <div v-if="tagSuggestions.length" class="inst-sugs">
                  <span class="inst-hint">{{ t('instance.ui.commonTags') }}</span>
                  <button v-for="tag in tagSuggestions" :key="tag" type="button" class="inst-sug" @click="addTag(tag)">
                    <UIcon name="i-lucide-plus" />{{ tag }}
                  </button>
                </div>
              </div>
            </section>

            <!-- 2 运行 -->
            <section class="inst-sec" aria-labelledby="inst-sec-2">
              <div class="inst-sec__head">
                <span class="inst-sec__n is-2">2</span>
                <h3 id="inst-sec-2" class="inst-sec__title">{{ t('instance.ui.secRun') }}</h3>
                <span v-if="prefilledFrom" class="inst-sec__badge">
                  <UIcon name="i-lucide-sparkles" />
                  {{ t('instance.ui.prefilledFrom', { name: prefilledFrom }) }}
                </span>
              </div>
              <div class="app-field">
                <label class="app-label app-label--required" for="inst-f-dir">{{ t('instance.ui.workdir') }}</label>
                <div class="inst-row">
                  <input
                    id="inst-f-dir"
                    v-model="localForm.instancePath"
                    class="app-input inst-input inst-mono"
                    :class="{ 'is-error': errors.instancePath }"
                    autocomplete="off"
                    spellcheck="false"
                  />
                  <UButton
                    color="neutral"
                    variant="outline"
                    icon="i-lucide-folder-open"
                    class="inst-pick"
                    :class="{ 'is-icon': isMobile }"
                    :aria-label="t('instance.ui.pickFolder')"
                    @click="pickerOpen = true"
                  >
                    <template v-if="!isMobile">{{ t('instance.ui.pickFolder') }}</template>
                  </UButton>
                </div>
                <span v-if="errors.instancePath" class="app-field__error">{{ errors.instancePath }}</span>
                <span v-else class="inst-hint">{{ workdirHint }}</span>
              </div>
              <div class="app-field">
                <label class="app-label app-label--required" for="inst-f-start">{{ t('instance.startCommand') }}</label>
                <textarea
                  id="inst-f-start"
                  v-model="localForm.startCommand"
                  class="app-textarea inst-input inst-mono"
                  :class="{ 'is-error': errors.startCommand }"
                  rows="2"
                  spellcheck="false"
                />
                <span v-if="errors.startCommand" class="app-field__error">{{ errors.startCommand }}</span>
                <span v-else class="inst-hint">{{ t('instance.ui.startCmdHint') }}</span>
              </div>
              <div class="inst-grid inst-grid--stop">
                <div class="app-field">
                  <label class="app-label" for="inst-f-stop">{{ t('instance.stopCommand') }}</label>
                  <input id="inst-f-stop" v-model="localForm.stopCommand" class="app-input inst-input inst-mono" spellcheck="false" />
                  <span class="inst-hint">{{ t('instance.ui.stopCmdHint') }}</span>
                </div>
                <div class="app-field">
                  <label class="app-label" for="inst-f-timeout">{{ t('instance.ui.stopTimeout') }}</label>
                  <div class="inst-suffix">
                    <input
                      id="inst-f-timeout"
                      v-model.number="localForm.stopTimeout"
                      class="app-input inst-input m-num"
                      type="number"
                      inputmode="numeric"
                      min="0"
                      max="3600"
                      :placeholder="String(DEFAULT_STOP_TIMEOUT)"
                    />
                    <span class="inst-suffix__u">{{ t('instance.ui.seconds') }}</span>
                  </div>
                  <span class="inst-hint">{{ t('instance.ui.stopTimeoutHint', { n: DEFAULT_STOP_TIMEOUT }) }}</span>
                </div>
              </div>
              <div class="inst-switches">
                <label class="inst-swr">
                  <span class="inst-swr__text">
                    <span class="inst-swr__t">{{ t('instance.ui.bootStart') }}</span>
                    <span class="inst-swr__d">{{ t('instance.ui.autoStartDesc') }}</span>
                  </span>
                  <AppSwitch v-model="localForm.autoStart" />
                </label>
                <label class="inst-swr">
                  <span class="inst-swr__text">
                    <span class="inst-swr__t">{{ t('instance.ui.autoRestart') }}</span>
                    <span class="inst-swr__d">{{ t('instance.ui.autoRestartDesc') }}</span>
                  </span>
                  <AppSwitch v-model="localForm.autoRestart" />
                </label>
              </div>
            </section>

            <!-- 3 环境变量 -->
            <section class="inst-sec" aria-labelledby="inst-sec-3">
              <div class="inst-sec__head">
                <span class="inst-sec__n is-3">3</span>
                <h3 id="inst-sec-3" class="inst-sec__title">
                  {{ t('instance.ui.secEnv') }}
                  <span class="inst-sec__count m-num">{{ t('instance.ui.envCount', { count: envCount }) }}</span>
                </h3>
                <span class="inst-sec__acts">
                  <UButton
                    color="neutral"
                    variant="soft"
                    size="sm"
                    icon="i-lucide-upload"
                    :class="{ 'inst-ico-only': isMobile }"
                    :aria-label="t('instance.ui.importEnv')"
                    @click="envFileRef?.click()"
                  >
                    <template v-if="!isMobile">{{ t('instance.ui.importEnv') }}</template>
                  </UButton>
                  <AppIconButton
                    :icon="envMode === 'rows' ? 'HOutline:DocumentTextIcon' : 'HOutline:Bars3BottomLeftIcon'"
                    :label="envMode === 'rows' ? t('instance.ui.envTextMode') : t('instance.ui.envTableMode')"
                    :size="18"
                    :box="isMobile ? 44 : 40"
                    @click="toggleEnvMode"
                  />
                </span>
                <input ref="envFileRef" type="file" class="inst-hidden" accept=".env,text/plain,*/*" @change="onEnvFile" />
              </div>

              <template v-if="envMode === 'rows'">
                <div class="inst-env">
                  <div v-if="envRows.length && !isMobile" class="inst-env__row inst-env__row--head">
                    <span>{{ t('instance.ui.envKey') }}</span>
                    <span>{{ t('instance.ui.envValue') }}</span>
                    <span />
                  </div>
                  <div v-for="(row, idx) in envRows" :key="row.id" class="inst-env__row">
                    <input
                      v-model="row.k"
                      class="app-input inst-input inst-mono inst-env__key"
                      :placeholder="isMobile ? t('instance.ui.envKey') : 'KEY'"
                      :aria-label="`${t('instance.ui.envKey')} ${idx + 1}`"
                      autocomplete="off"
                      spellcheck="false"
                    />
                    <div class="inst-env__val" :class="{ 'is-secret': row.secret }">
                      <button
                        type="button"
                        class="inst-env__lock"
                        :class="{ 'is-on': row.secret }"
                        :aria-pressed="row.secret"
                        :aria-label="row.secret ? t('instance.ui.envUnlock') : t('instance.ui.envLock')"
                        :title="row.secret ? t('instance.ui.envUnlock') : t('instance.ui.envLock')"
                        @click="toggleSecret(row)"
                      >
                        <UIcon :name="row.secret ? 'i-lucide-lock' : 'i-lucide-lock-open'" />
                      </button>
                      <input
                        v-model="row.v"
                        class="inst-env__input m-mono"
                        :type="row.secret && !revealed.has(row.id) ? 'password' : 'text'"
                        :placeholder="row.secret && row.saved ? t('instance.ui.envSecretKeep') : t('instance.ui.envValue')"
                        :aria-label="`${t('instance.ui.envValue')} ${row.k || idx + 1}`"
                        autocomplete="new-password"
                        spellcheck="false"
                      />
                      <button
                        v-if="row.secret"
                        type="button"
                        class="inst-env__eye"
                        :aria-pressed="revealed.has(row.id)"
                        :aria-label="revealed.has(row.id) ? t('instance.ui.hideValue') : t('instance.ui.showValue')"
                        @click="toggleReveal(row.id)"
                      >
                        <UIcon :name="revealed.has(row.id) ? 'i-lucide-eye-off' : 'i-lucide-eye'" />
                      </button>
                    </div>
                    <AppIconButton
                      :icon="isMobile ? 'HOutline:TrashIcon' : 'HOutline:MinusCircleIcon'"
                      class="inst-env__del"
                      :box="isMobile ? 44 : 40"
                      :label="t('instance.ui.removeEnv', { name: row.k || idx + 1 })"
                      @click="removeEnv(row.id)"
                    />
                  </div>
                  <button type="button" class="inst-env__add" @click="addEnv">
                    <UIcon name="i-lucide-plus" />{{ t('instance.ui.addEnv') }}
                  </button>
                  <span class="inst-hint">{{ t('instance.ui.envSecretHint') }}</span>
                </div>
              </template>
              <template v-else>
                <textarea
                  v-model="envText"
                  class="app-textarea inst-input inst-mono"
                  rows="6"
                  spellcheck="false"
                  :placeholder="t('instance.envPlaceholder')"
                />
                <span class="inst-hint">
                  {{ t('instance.ui.envHint') }}
                  <template v-if="secretRowCount">{{ t('instance.ui.envTextSecretNote', { n: secretRowCount }) }}</template>
                </span>
              </template>
            </section>
          </div>

          <footer class="inst-drawer__foot">
            <span v-if="!isMobile" class="inst-drawer__dirty" :class="{ 'is-dirty': dirtyCount > 0 }">
              <span class="inst-drawer__dirty-dot" />
              {{ dirtyText }}
            </span>
            <UButton v-if="!isMobile" color="neutral" variant="soft" class="inst-drawer__btn" @click="handleClose">
              {{ t('common.cancel') }}
            </UButton>
            <UButton
              :color="isMobile && allowRun ? 'neutral' : 'primary'"
              :variant="isMobile && allowRun ? 'soft' : 'solid'"
              class="inst-drawer__btn"
              :class="{ 'inst-drawer__btn--main': !(isMobile && allowRun) }"
              :icon="isMobile && allowRun ? undefined : 'i-lucide-check'"
              :disabled="loading"
              :loading="submitting && !runIntent"
              @click="handleSubmit(false)"
            >
              {{ isMobile && allowRun ? t('instance.ui.saveOnly') : t('common.save') }}
            </UButton>
            <UButton
              v-if="allowRun"
              :color="isMobile ? 'primary' : 'neutral'"
              :variant="isMobile ? 'solid' : 'outline'"
              class="inst-drawer__btn"
              :class="{ 'inst-drawer__btn--main': isMobile }"
              :icon="running && mode === 'edit' ? 'i-lucide-rotate-cw' : 'i-lucide-play'"
              :disabled="loading"
              :loading="submitting && runIntent"
              @click="handleSubmit(true)"
            >
              {{ running && mode === 'edit' ? t('instance.ui.saveAndRestart') : t('instance.ui.saveAndStart') }}
            </UButton>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>

  <!-- 选择工作目录：系统文件管理里的本机文件夹 -->
  <FilePicker v-model="pickerOpen" :title="t('instance.ui.pickFolder')" @confirm="onPickFolder" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getInstanceStatsRequest, getInstanceTagsRequest } from '@/api/instance'
import FilePicker from '@/components/file/FilePicker.vue'
import type { PickedFile } from '@/components/file/types'
import { formatFileSize } from '@/utils/file'
import type {
  InstanceEditorFormValue,
  InstanceEditorMode,
  InstanceEnvRow,
  InstanceTypeOption,
} from '@/stores/instance/types'

defineOptions({ name: 'InstanceEditor' })

const DEFAULT_STOP_TIMEOUT = 3

const props = withDefaults(
  defineProps<{
    visible: boolean
    mode: InstanceEditorMode
    loading: boolean
    submitting: boolean
    form: InstanceEditorFormValue
    typeOptions: InstanceTypeOption[]
    /** 显示「保存并启动/重启」按钮（父级在 submit 的第二参数里拿到 run=true 后执行） */
    allowRun?: boolean
    /** 应用当前是否在运行（决定按钮文案是启动还是重启） */
    running?: boolean
  }>(),
  { allowRun: false, running: false },
)

const emit = defineEmits<{
  close: []
  submit: [value: InstanceEditorFormValue, options?: { run: boolean }]
}>()

const { t } = useI18n()
const menuStore = useMenuStore()
const isMobile = computed(() => menuStore.isMobile)

type FormFields = Omit<InstanceEditorFormValue, 'env'>
const localForm = reactive<FormFields>({
  id: '',
  name: '',
  remark: '',
  tags: '',
  type: '',
  startCommand: '',
  stopCommand: '',
  instancePath: '',
  autoStart: false,
  autoRestart: false,
  stopTimeout: 0,
})

const errors = ref<Record<string, string>>({})
const runIntent = ref(false)
const snapshot = ref<Record<string, string> | null>(null)

const initial = computed(() => (localForm.name || '?').trim().charAt(0).toUpperCase())
const subtitle = computed(() => {
  if (props.mode === 'create') return t('instance.ui.editorCreateSub')
  const name = localForm.name || props.form.name
  const state = props.running ? t('instance.running') : t('instance.stopped')
  return `${name} · ${state}，${t('instance.ui.editorEditSubShort')}`
})

// —— 类型选项：编辑未分类的应用时可保持「未分类」 ——
const typeSelectOptions = computed(() =>
  props.mode === 'edit' ? [{ label: t('instance.ui.uncategorized'), value: '' }, ...props.typeOptions] : props.typeOptions,
)

// —— 标签 chips（存储仍是逗号分隔字符串）+ 常用标签 ——
const tagInputRef = ref<HTMLInputElement>()
const tagDraft = ref('')
const knownTags = ref<string[]>([])
const tagList = computed(() =>
  localForm.tags
    .split(/[,，]/u)
    .map((s) => s.trim().replace(/^#/u, ''))
    .filter(Boolean),
)
const tagSuggestions = computed(() => knownTags.value.filter((tag) => !tagList.value.includes(tag)).slice(0, isMobile.value ? 8 : 5))
const setTags = (list: string[]) => {
  localForm.tags = Array.from(new Set(list)).join(',')
}
const addTag = (tag: string) => setTags([...tagList.value, tag])
const commitTag = () => {
  const parts = tagDraft.value
    .split(/[,，\s]+/u)
    .map((s) => s.trim().replace(/^#/u, ''))
    .filter(Boolean)
  if (parts.length) setTags([...tagList.value, ...parts])
  tagDraft.value = ''
}
const removeTag = (tag: string) => setTags(tagList.value.filter((x) => x !== tag))
const onTagKey = (e: KeyboardEvent) => {
  if (e.key === ',' || e.key === '，') {
    e.preventDefault()
    commitTag()
  } else if (e.key === 'Backspace' && !tagDraft.value && tagList.value.length) {
    removeTag(tagList.value[tagList.value.length - 1]!)
  }
}
const loadKnownTags = async () => {
  try {
    const { data } = await getInstanceTagsRequest()
    knownTags.value = (data?.tags || []).map((tag) => tag.name)
  } catch {
    knownTags.value = []
  }
}

// —— 工作目录：选择文件夹 + 状态提示 ——
const pickerOpen = ref(false)
const dirUsage = ref('')
const onPickFolder = (items: PickedFile[]) => {
  const picked = items[0]
  if (!picked) return
  if (picked.sourceId) {
    feedback.warning(t('instance.ui.pickFolderLocalOnly'))
    return
  }
  localForm.instancePath = picked.path
}
const workdirHint = computed(() => {
  if (props.mode === 'edit' && localForm.instancePath.trim() === props.form.instancePath.trim() && dirUsage.value) {
    return t('instance.ui.workdirExists', { size: dirUsage.value })
  }
  return t('instance.ui.workdirAutoCreate')
})
const loadDirUsage = async (id: string) => {
  dirUsage.value = ''
  if (!id) return
  try {
    const { data } = await getInstanceStatsRequest({ id })
    if (data?.diskScanTime) dirUsage.value = formatFileSize(Number(data.diskUsageBytes || 0))
  } catch {
    dirUsage.value = ''
  }
}

// —— 环境变量：逐条（可按密钥保存）⇄ 文本（只编辑普通变量）——
interface EnvRow {
  id: number
  k: string
  v: string
  secret: boolean
  /** 服务器上已保存的密钥：值不回显，留空保持不变 */
  saved: boolean
}
let envSeq = 0
const envMode = ref<'rows' | 'text'>('rows')
const envRows = ref<EnvRow[]>([])
const envText = ref('')
const envFileRef = ref<HTMLInputElement>()
const revealed = ref(new Set<number>())

const parseEnvText = (text: string): EnvRow[] =>
  text
    .split(/\r?\n/u)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('#'))
    .map((line) => {
      const body = line.replace(/^export\s+/u, '')
      const i = body.indexOf('=')
      const k = i < 0 ? body : body.slice(0, i).trim()
      let v = i < 0 ? '' : body.slice(i + 1).trim()
      if (v.length >= 2 && ((v.startsWith('"') && v.endsWith('"')) || (v.startsWith("'") && v.endsWith("'")))) {
        v = v.slice(1, -1)
      }
      return { id: ++envSeq, k, v, secret: false, saved: false }
    })
const plainToText = (rows: EnvRow[]) =>
  rows
    .filter((r) => !r.secret && r.k.trim())
    .map((r) => `${r.k.trim()}=${r.v}`)
    .join('\n')
const secretRowCount = computed(() => envRows.value.filter((r) => r.secret && r.k.trim()).length)

// 文本模式下改的是普通变量；密钥变量原样保留在逐条列表里
const syncFromText = () => {
  const secrets = envRows.value.filter((r) => r.secret)
  const secretKeys = new Set(secrets.map((r) => r.k.trim()))
  envRows.value = [...parseEnvText(envText.value).filter((r) => !secretKeys.has(r.k.trim())), ...secrets]
}
const toggleEnvMode = () => {
  if (envMode.value === 'rows') {
    envText.value = plainToText(envRows.value)
    envMode.value = 'text'
  } else {
    syncFromText()
    envMode.value = 'rows'
  }
}
const addEnv = () => {
  envRows.value.push({ id: ++envSeq, k: '', v: '', secret: false, saved: false })
}
const removeEnv = (id: number) => {
  envRows.value = envRows.value.filter((r) => r.id !== id)
}
const toggleSecret = (row: EnvRow) => {
  row.secret = !row.secret
  // 已保存的密钥改成普通变量时，旧值不可见，需要重新填写
  if (!row.secret) row.saved = false
}
const toggleReveal = (id: number) => {
  const next = new Set(revealed.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  revealed.value = next
}
const currentRows = () => (envMode.value === 'text' ? (syncFromText(), envRows.value) : envRows.value)
const envCount = computed(() =>
  envMode.value === 'rows'
    ? envRows.value.filter((r) => r.k.trim()).length
    : parseEnvText(envText.value).length + secretRowCount.value,
)

const onEnvFile = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  const incoming = parseEnvText(await file.text())
  if (!incoming.length) {
    feedback.warning(t('instance.ui.importEnvEmpty'))
    return
  }
  // 同名变量以导入的为准（已是密钥的保持密钥），其余追加
  const rows = currentRows()
  const map = new Map(rows.filter((r) => r.k.trim()).map((r) => [r.k.trim(), r] as const))
  incoming.forEach((r) => {
    const old = map.get(r.k)
    map.set(r.k, old ? { ...old, v: r.v } : r)
  })
  envRows.value = Array.from(map.values())
  if (envMode.value === 'text') envText.value = plainToText(envRows.value)
  feedback.success(t('instance.ui.importEnvDone', { count: incoming.length }))
}

const toFormEnv = (rows: EnvRow[]): InstanceEnvRow[] =>
  rows
    .filter((r) => r.k.trim())
    .map((r) => ({ key: r.k.trim(), value: r.v, secret: r.secret, saved: r.saved }))

// —— 类型模板预填（仅新建）——
const typeOf = (id: string) => props.typeOptions.find((o) => o.value === id)
const prefilledFrom = computed(() => {
  const opt = typeOf(localForm.type)
  if (!opt?.startCommand) return ''
  return localForm.startCommand.trim() === opt.startCommand.trim() ? opt.label : ''
})
watch(
  () => localForm.type,
  (next, prev) => {
    if (!props.visible || props.mode !== 'create' || props.loading || next === prev) return
    const nextOpt = typeOf(next)
    const prevOpt = prev ? typeOf(prev) : undefined
    if (!nextOpt) return
    const start = localForm.startCommand.trim()
    if (nextOpt.startCommand && (!start || start === (prevOpt?.startCommand || '').trim())) {
      localForm.startCommand = nextOpt.startCommand
    }
    const stop = localForm.stopCommand.trim()
    if (nextOpt.stopCommand && (!stop || stop === 'exit' || stop === (prevOpt?.stopCommand || '').trim())) {
      localForm.stopCommand = nextOpt.stopCommand
    }
  },
)

// —— 脏检查：逐项比较，显示改了几处 ——
const fieldValues = (): Record<string, string> => {
  const rows = envMode.value === 'text' ? [...parseEnvText(envText.value), ...envRows.value.filter((r) => r.secret)] : envRows.value
  return {
    name: localForm.name.trim(),
    type: localForm.type,
    remark: localForm.remark.trim(),
    tags: tagList.value.join(','),
    instancePath: localForm.instancePath.trim(),
    startCommand: localForm.startCommand.trim(),
    stopCommand: localForm.stopCommand.trim(),
    stopTimeout: String(Number(localForm.stopTimeout) || 0),
    autoStart: String(localForm.autoStart),
    autoRestart: String(localForm.autoRestart),
    env: JSON.stringify(
      rows.filter((r) => r.k.trim()).map((r) => [r.k.trim(), r.secret ? (r.v ? `*${r.v}` : '*') : r.v]),
    ),
  }
}
const dirtyCount = computed(() => {
  const base = snapshot.value
  if (!base) return 0
  const now = fieldValues()
  return Object.keys(now).filter((k) => now[k] !== base[k]).length
})
const dirtyText = computed(() => {
  if (props.mode === 'create') return dirtyCount.value ? t('instance.ui.notSavedYet') : t('instance.ui.noChanges')
  return dirtyCount.value ? t('instance.ui.unsavedCount', { n: dirtyCount.value }) : t('instance.ui.noChanges')
})

const validate = (): boolean => {
  const e: Record<string, string> = {}
  if (!localForm.name.trim()) e.name = t('instance.instanceNameRequired')
  if (props.mode === 'create' && !localForm.type.trim()) e.type = t('instance.instanceTypeRequired')
  if (!localForm.instancePath.trim()) e.instancePath = t('instance.instancePathRequired')
  if (!localForm.startCommand.trim()) e.startCommand = t('instance.startCommandRequired')
  errors.value = e
  return Object.keys(e).length === 0
}

const handleClose = () => {
  errors.value = {}
  emit('close')
}

const handleSubmit = (run: boolean) => {
  commitTag()
  if (!validate()) return
  runIntent.value = run
  const value: InstanceEditorFormValue = {
    ...localForm,
    stopTimeout: Math.max(0, Math.round(Number(localForm.stopTimeout) || 0)),
    env: toFormEnv(currentRows()),
  }
  if (run) emit('submit', value, { run: true })
  else emit('submit', value)
}

const resetFromProps = (formValue: InstanceEditorFormValue) => {
  const { env, ...fields } = formValue
  Object.assign(localForm, fields)
  if (!Number.isFinite(Number(localForm.stopTimeout))) localForm.stopTimeout = 0
  envRows.value = (env || []).map((r) => ({ id: ++envSeq, k: r.key, v: r.value, secret: r.secret, saved: r.saved }))
  envText.value = ''
  revealed.value = new Set()
  tagDraft.value = ''
  // 新建时启动命令为空：按当前类型模板预填
  if (props.mode === 'create') {
    const opt = typeOf(localForm.type)
    if (opt?.startCommand && !localForm.startCommand.trim()) localForm.startCommand = opt.startCommand
    if (opt?.stopCommand && (!localForm.stopCommand.trim() || localForm.stopCommand.trim() === 'exit')) {
      localForm.stopCommand = opt.stopCommand
    }
  }
  nextTick(() => {
    snapshot.value = fieldValues()
  })
}

watch(
  () => props.visible,
  (visible) => {
    if (!visible) {
      errors.value = {}
      runIntent.value = false
      snapshot.value = null
      dirUsage.value = ''
    } else {
      envMode.value = 'rows'
      resetFromProps(props.form)
      void loadKnownTags()
      if (props.mode === 'edit') void loadDirUsage(props.form.id)
    }
  },
)

watch(
  () => props.form,
  (formValue) => {
    resetFromProps(formValue)
    if (props.visible && props.mode === 'edit' && formValue.id) void loadDirUsage(formValue.id)
  },
  { immediate: true, deep: true },
)

// Esc 关闭 + 打开时锁定页面滚动
const onKey = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.visible && !pickerOpen.value) handleClose()
}
watch(
  () => props.visible,
  (visible) => {
    if (typeof document === 'undefined') return
    document.body.style.overflow = visible ? 'hidden' : ''
  },
)
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  if (props.visible) document.body.style.overflow = ''
})
</script>

<style scoped lang="scss">
.inst-drawer {
  position: fixed;
  inset: 0;
  /* 高于手机底部导航（2050），否则底部「保存」被挡住 */
  z-index: 2100;
  display: flex;
  justify-content: flex-end;
  background: color-mix(in srgb, var(--m-ink) 48%, transparent);
  backdrop-filter: blur(3px);
  -webkit-backdrop-filter: blur(3px);
}
.inst-drawer__panel {
  width: 640px;
  max-width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--m-bg);
  border-radius: 32px 0 0 32px;
  box-shadow: -24px 0 60px -20px color-mix(in srgb, var(--m-ink) 45%, transparent);
}
.inst-drawer__head {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 20px 18px 18px 28px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-light);
}
.inst-drawer__av {
  width: 48px;
  height: 48px;
  border-radius: 16px;
  font-size: 1.25rem;
  transform: rotate(-5deg);
  flex: none;
}
.inst-drawer__av-ico {
  width: 22px;
  height: 22px;
}
.inst-drawer__titles {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.inst-drawer__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.inst-drawer__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.inst-drawer__loading {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
}
.inst-drawer__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 20px 24px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  scrollbar-width: none;
}
.inst-drawer__body::-webkit-scrollbar {
  display: none;
}
.inst-drawer__foot {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  padding: 16px 24px calc(20px + env(safe-area-inset-bottom));
  background: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color-light);
}
.inst-drawer__dirty {
  flex: 1;
  min-width: 120px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.inst-drawer__dirty-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--el-border-color);
}
.inst-drawer__dirty.is-dirty .inst-drawer__dirty-dot {
  background: var(--m-warn);
}
.inst-drawer__btn {
  min-height: 44px;
  border-radius: 999px;
  padding: 0 18px;
  justify-content: center;
}
.inst-drawer__btn--main {
  padding: 0 26px;
}

/* 分区 */
.inst-sec {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 18px 20px 20px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.inst-sec__head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.inst-sec__n {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 11px;
  font-family: var(--app-font-display);
  font-size: 0.9375rem;
  font-weight: 800;
  flex: none;
}
.inst-sec__n.is-1 {
  background: var(--m-pSoft);
  color: var(--m-pInk);
  transform: rotate(-4deg);
}
.inst-sec__n.is-2 {
  background: var(--m-a2Soft);
  color: var(--m-ink);
  transform: rotate(3deg);
}
.inst-sec__n.is-3 {
  background: var(--m-aSoft);
  color: var(--m-ink);
  transform: rotate(-3deg);
}
.inst-sec__title {
  flex: 1;
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.inst-sec__count {
  margin-left: 4px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.inst-sec__badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: var(--m-aSoft);
  color: var(--m-ink);
  font-size: 0.75rem;
  font-weight: 700;
}
.inst-sec__acts {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.inst-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 16px;
}
.inst-grid--stop {
  grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr);
}
.inst-input {
  min-height: 46px;
  border-radius: 14px;
  background: var(--el-bg-color);
}
/* 类型下拉与同表单的输入框同高同圆角 */
.inst-drawer :deep(.app-sel__trigger) {
  height: 46px;
  padding: 0 12px 0 14px;
  border-radius: 14px;
  font-size: 0.875rem;
}
textarea.inst-input {
  min-height: 0;
}
.inst-mono {
  font-family: var(--app-font-mono);
  font-size: 0.8125rem;
}
.inst-hint {
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.inst-link {
  font-weight: 700;
  color: var(--m-pInk);
}
.inst-opt {
  margin-left: 2px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.inst-hidden {
  display: none;
}
.inst-row {
  display: flex;
  gap: 8px;
}
.inst-row > .inst-input {
  flex: 1;
  min-width: 0;
}
.inst-pick {
  flex: none;
  min-height: 46px;
  border-radius: 14px;
  padding: 0 16px;
}
.inst-pick.is-icon {
  width: 50px;
  padding: 0;
  justify-content: center;
}
.inst-ico-only {
  width: 44px;
  height: 44px;
  padding: 0;
  justify-content: center;
}

/* 标签 chips 输入 */
.inst-tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-height: 46px;
  padding: 7px 8px;
  border-radius: 14px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  cursor: text;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.inst-tags:focus-within {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.inst-tchip {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  height: 30px;
  padding: 0 4px 0 11px;
  border-radius: 999px;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-size: 0.8125rem;
  font-weight: 700;
}
.inst-tchip button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: inherit;
  cursor: pointer;
}
.inst-tchip button:hover {
  background: color-mix(in srgb, var(--m-p) 16%, transparent);
}
.inst-tags__input {
  flex: 1;
  min-width: 140px;
  min-height: 30px;
  padding: 0 6px;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
}
.inst-sugs {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.inst-sug {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 30px;
  padding: 0 10px;
  border-radius: 999px;
  border: 1.5px dashed var(--el-border-color-light);
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
  white-space: nowrap;
}
.inst-sug:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.inst-sug :deep(.iconify) {
  width: 12px;
  height: 12px;
}

/* 停止超时：带单位后缀 */
.inst-suffix {
  position: relative;
}
.inst-suffix input {
  width: 100%;
  padding-right: 44px;
}
.inst-suffix__u {
  position: absolute;
  right: 14px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  pointer-events: none;
}

/* 开关行 */
.inst-switches {
  display: flex;
  flex-direction: column;
  margin-top: -4px;
}
.inst-swr {
  display: flex;
  align-items: center;
  gap: 16px;
  min-height: 60px;
  padding: 10px 0;
  border-top: 1px solid var(--el-border-color-lighter);
  cursor: pointer;
}
.inst-swr__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.inst-swr__t {
  font-size: 0.9375rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.inst-swr__d {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}

/* 环境变量 */
.inst-env {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.inst-env__row {
  display: grid;
  grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.3fr) 40px;
  align-items: center;
  gap: 8px;
}
.inst-env__row--head {
  padding: 0 4px;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.inst-env__val {
  display: flex;
  align-items: center;
  gap: 2px;
  min-width: 0;
  min-height: 46px;
  padding: 0 4px;
  border-radius: 14px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.inst-env__val:focus-within {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.inst-env__input {
  flex: 1;
  min-width: 0;
  min-height: 40px;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
}
.inst-env__input::placeholder {
  color: var(--el-text-color-secondary);
}
.inst-env__lock,
.inst-env__eye {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.inst-env__lock:hover,
.inst-env__eye:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.inst-env__lock.is-on {
  color: var(--m-warn);
}
.inst-env__lock :deep(.iconify),
.inst-env__eye :deep(.iconify) {
  width: 16px;
  height: 16px;
}
.inst-env__del {
  color: var(--m-bad) !important;
}
.inst-env__del:hover {
  background: var(--m-badSoft) !important;
}
.inst-env__add {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 46px;
  border-radius: 14px;
  border: 1.5px dashed var(--el-border-color-light);
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.inst-env__add:hover {
  border-color: var(--m-p);
  background: var(--m-pSoft);
}

/* 过渡 */
.inst-drawer-enter-active,
.inst-drawer-leave-active {
  transition: opacity 0.2s ease;
}
.inst-drawer-enter-active .inst-drawer__panel,
.inst-drawer-leave-active .inst-drawer__panel {
  transition: transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
}
.inst-drawer-enter-from,
.inst-drawer-leave-to {
  opacity: 0;
}
.inst-drawer-enter-from .inst-drawer__panel,
.inst-drawer-leave-to .inst-drawer__panel {
  transform: translateX(40px);
}

/* ========== 手机：全屏长页 ========== */
.inst-drawer.is-mobile .inst-drawer__panel {
  width: 100%;
  border-radius: 0;
  box-shadow: none;
}
.inst-drawer.is-mobile .inst-drawer__head {
  gap: 4px;
  padding: calc(10px + env(safe-area-inset-top)) 8px 10px 4px;
  background: var(--m-bg);
}
.inst-drawer.is-mobile .inst-drawer__title {
  font-size: 1.25rem;
}
.inst-drawer.is-mobile .inst-drawer__sub {
  font-size: 0.75rem;
}
.inst-drawer.is-mobile .inst-drawer__body {
  padding: 16px 16px 24px;
  gap: 14px;
}
.inst-drawer.is-mobile .inst-sec {
  padding: 18px 16px;
}
.inst-drawer.is-mobile .inst-grid,
.inst-drawer.is-mobile .inst-grid--stop {
  grid-template-columns: minmax(0, 1fr);
}
.inst-drawer.is-mobile .inst-input,
.inst-drawer.is-mobile .inst-pick {
  min-height: 50px;
  font-size: 1rem;
}
.inst-drawer.is-mobile :deep(.app-sel__trigger) {
  height: 50px;
  font-size: 1rem;
}
.inst-drawer.is-mobile .inst-mono {
  font-size: 0.875rem;
}
.inst-drawer.is-mobile .inst-sec__badge {
  order: 5;
  flex-basis: 100%;
  min-height: 40px;
  padding: 10px 12px;
  border-radius: 14px;
  font-size: 0.8125rem;
}
.inst-drawer.is-mobile .inst-sugs {
  flex-wrap: nowrap;
  overflow-x: auto;
  margin: 0 -16px;
  padding: 0 16px;
  scrollbar-width: none;
}
.inst-drawer.is-mobile .inst-sugs::-webkit-scrollbar {
  display: none;
}
.inst-drawer.is-mobile .inst-sug {
  min-height: 36px;
}
.inst-drawer.is-mobile .inst-tags__input {
  font-size: 1rem;
  min-width: 100px;
}
/* 手机环境变量：每条一张小卡（变量名 + 删除 / 值） */
.inst-drawer.is-mobile .inst-env__row {
  grid-template-columns: minmax(0, 1fr) 44px;
  grid-template-areas:
    'key del'
    'val val';
  gap: 8px 6px;
  padding: 12px;
  border-radius: 18px;
  background: var(--el-fill-color-light);
}
.inst-drawer.is-mobile .inst-env__key {
  grid-area: key;
}
.inst-drawer.is-mobile .inst-env__val {
  grid-area: val;
  min-height: 50px;
}
.inst-drawer.is-mobile .inst-env__del {
  grid-area: del;
}
.inst-drawer.is-mobile .inst-env__input {
  font-size: 0.9375rem;
}
.inst-drawer.is-mobile .inst-env__lock,
.inst-drawer.is-mobile .inst-env__eye {
  width: 44px;
  height: 44px;
}
.inst-drawer.is-mobile .inst-env__add {
  min-height: 50px;
}
.inst-drawer.is-mobile .inst-drawer__foot {
  flex-wrap: nowrap;
  padding: 12px 16px calc(14px + env(safe-area-inset-bottom));
  box-shadow: 0 -12px 30px -18px color-mix(in srgb, var(--m-ink) 30%, transparent);
}
.inst-drawer.is-mobile .inst-drawer__btn {
  flex: 1 1 0;
  min-height: 52px;
  font-size: 0.9375rem;
}
.inst-drawer.is-mobile .inst-drawer__btn--main {
  flex: 1.5 1 0;
}
.inst-drawer.is-mobile .inst-swr {
  min-height: 64px;
}
</style>
