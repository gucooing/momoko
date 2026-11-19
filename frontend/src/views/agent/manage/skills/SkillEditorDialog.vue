<!-- 技能编辑器：名称、说明；左边是技能里的文件（按目录分组），右边是当前文件的路径和内容；可以增删文件、改路径。
     不是文本的文件只保留、不编辑（保存时不带内容，后端沿用原来的字节），所以它的路径也不能改。 -->
<template>
  <AgDialog
    :model-value="modelValue"
    :title="form.id ? t('agent.skills.editTitle') : t('agent.skills.create')"
    :width="800"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="ag-sk-cols ag-grid-2">
      <label class="ag-field">
        <span class="ag-fl">{{ t('agent.common.name') }}</span>
        <span class="ag-fld">
          <input v-model="form.name" class="ag-mono" type="text" autocomplete="off" spellcheck="false" :placeholder="t('agent.skills.editor.namePh')" :aria-label="t('agent.common.name')" />
        </span>
      </label>
      <label class="ag-field">
        <span class="ag-fl">{{ t('agent.common.description') }}</span>
        <span class="ag-fld">
          <input v-model="form.description" type="text" autocomplete="off" :placeholder="t('agent.skills.editor.descPh')" :aria-label="t('agent.common.description')" />
        </span>
      </label>
    </div>

    <div class="ag-sk-panes ag-grid-2">
      <div class="ag-sk-tree">
        <div class="ag-lbl ag-sk-tree__title">{{ t('agent.skills.editor.files') }}</div>
        <div class="ag-sk-tree__list ag-noscroll">
          <template v-for="x in tree" :key="x.key">
            <div v-if="x.dir" class="ag-sk-dir">
              <UIcon name="i-lucide-folder" /><span class="ag-mono ag-ellipsis">{{ x.text }}</span>
            </div>
            <button
              v-else
              type="button"
              class="ag-sk-fr"
              :class="{ 'is-on': x.index === form.cur, 'is-nested': x.nested }"
              :aria-current="x.index === form.cur"
              @click="form.cur = x.index"
            >
              <UIcon name="i-lucide-file" /><span class="ag-mono ag-sk-fr__name">{{ x.text || '—' }}</span>
            </button>
          </template>
        </div>
        <button type="button" class="ag-btn ag-btn--sm" @click="addFile">
          <UIcon name="i-lucide-plus" />{{ t('agent.skills.editor.addFile') }}
        </button>
      </div>

      <div v-if="cur" class="ag-sk-edit">
        <div class="ag-sk-edit__bar">
          <input
            v-model="cur.path"
            class="ag-sk-edit__path"
            type="text"
            autocomplete="off"
            spellcheck="false"
            :readonly="cur.binary"
            :aria-label="t('agent.skills.editor.path')"
          />
          <button
            v-if="form.files.length > 1"
            type="button"
            class="ag-ib ag-ib--sm"
            :aria-label="t('agent.skills.editor.deleteFile')"
            :title="t('agent.skills.editor.deleteFile')"
            @click="dropFile"
          >
            <UIcon name="i-lucide-trash-2" />
          </button>
        </div>
        <div v-if="cur.binary" class="ag-sk-binary">{{ t('agent.skills.editor.binary') }}</div>
        <textarea
          v-else
          :key="cur.uid"
          v-model="cur.content"
          class="ag-sk-text ag-noscroll"
          spellcheck="false"
          :aria-label="t('agent.skills.editor.content')"
        />
      </div>
    </div>

    <label class="ag-sk-on">
      <span>{{ t('agent.skills.editor.enabled') }}</span>
      <AgSwitch v-model="form.enabled" :label="t('agent.skills.editor.enabled')" />
    </label>

    <template #footer>
      <button type="button" class="ag-btn" @click="emit('update:modelValue', false)">{{ t('agent.common.cancel') }}</button>
      <button type="button" class="ag-btn ag-btn--p" :disabled="saving" @click="save">
        <UIcon v-if="saving" name="i-lucide-loader-circle" class="ag-spin" />{{ t('agent.common.save') }}
      </button>
    </template>
  </AgDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { createAgentSkill, updateAgentSkill } from '@/api/agent'
import type { AgentSkill, AgentSkillFile } from '@/types/v1/agent'
import AgDialog from '@/views/agent/components/AgDialog.vue'
import AgSwitch from '@/views/agent/components/AgSwitch.vue'

defineOptions({ name: 'AgentSkillEditorDialog' })

const props = defineProps<{
  modelValue: boolean
  /** 要编辑的技能和它的文件；null 表示新建 */
  detail: { skill: AgentSkill; files: AgentSkillFile[] } | null
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [] }>()
const { t } = useI18n()

interface FileRow {
  uid: number
  path: string
  content: string
  binary: boolean
}

const ROOT_DOC = 'SKILL.md'
const NAME_RE = /^[a-z0-9][a-z0-9._-]*$/

let uid = 0
const form = reactive({
  id: '',
  name: '',
  description: '',
  enabled: true,
  files: [] as FileRow[],
  cur: 0,
})
const saving = ref(false)
const cur = computed(() => form.files[form.cur])

/** 新技能的 SKILL.md */
const skillDoc = (name: string, description: string) =>
  [
    '---',
    `name: ${name}`,
    `description: ${description}`,
    '---',
    '',
    `# ${name || t('agent.skills.editor.templateTitle')}`,
    '',
    t('agent.skills.editor.templateBody'),
  ].join('\n')

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    const d = props.detail
    const files: FileRow[] = d
      ? d.files.map((f) => ({ uid: ++uid, path: f.path, content: f.binary ? '' : f.content, binary: f.binary }))
      : [{ uid: ++uid, path: ROOT_DOC, content: skillDoc('', ''), binary: false }]
    Object.assign(form, {
      id: d?.skill.id ?? '',
      name: d?.skill.name ?? '',
      description: d?.skill.description ?? '',
      enabled: d ? d.skill.enabled : true,
      files,
      // 先打开 SKILL.md
      cur: Math.max(0, files.findIndex((f) => f.path === ROOT_DOC)),
    })
    saving.value = false
  },
)

// SKILL.md 还是模板原样的时候，开头的 name / description 跟着上面两格走
watch(
  () => [form.name.trim(), form.description.trim()] as const,
  ([name, description], [prevName, prevDescription]) => {
    const doc = form.files.find((f) => f.path === ROOT_DOC && !f.binary)
    if (doc && doc.content === skillDoc(prevName, prevDescription)) doc.content = skillDoc(name, description)
  },
)

// —— 左边的文件列表：根下的文件在前，然后每个目录一个标题，目录里的文件缩进 ——
interface TreeRow {
  key: string
  dir: boolean
  text: string
  /** 在 form.files 里的位置；目录标题是 -1 */
  index: number
  nested: boolean
}
const tree = computed<TreeRow[]>(() => {
  const rows: TreeRow[] = []
  const dirs = new Map<string, TreeRow[]>()
  form.files.forEach((f, index) => {
    const at = f.path.lastIndexOf('/')
    const dir = at < 0 ? '' : f.path.slice(0, at)
    const row: TreeRow = { key: `f${f.uid}`, dir: false, text: f.path.slice(at + 1), index, nested: !!dir }
    if (!dir) rows.push(row)
    else if (dirs.has(dir)) dirs.get(dir)?.push(row)
    else dirs.set(dir, [row])
  })
  dirs.forEach((files, dir) => {
    rows.push({ key: `d${dir}`, dir: true, text: dir, index: -1, nested: false }, ...files)
  })
  return rows
})

const normPath = (path: string) => path.trim().replace(/\\/g, '/').replace(/^\/+/, '')

const addFile = () => {
  const taken = new Set(form.files.map((f) => normPath(f.path)))
  let n = form.files.length
  while (taken.has(`references/new-${n}.md`)) n += 1
  form.files.push({ uid: ++uid, path: `references/new-${n}.md`, content: '', binary: false })
  form.cur = form.files.length - 1
}
const dropFile = () => {
  form.files.splice(form.cur, 1)
  form.cur = 0
}

const save = async () => {
  if (saving.value) return
  const name = form.name.trim()
  if (!name) {
    feedback.warning(t('agent.skills.toast.needName'))
    return
  }
  if (!NAME_RE.test(name)) {
    feedback.warning(t('agent.skills.toast.badName'))
    return
  }
  // 每个文件都发；不是文本的只发路径，后端沿用原来的内容
  const files: AgentSkillFile[] = form.files.map((f) => ({
    path: normPath(f.path),
    content: f.binary ? '' : f.content,
    binary: f.binary,
  }))
  const paths = files.map((f) => f.path)
  if (paths.some((p) => !p || p.endsWith('/')) || new Set(paths).size !== paths.length) {
    feedback.warning(t('agent.skills.toast.badPaths'))
    return
  }
  if (!paths.includes(ROOT_DOC)) {
    feedback.warning(t('agent.skills.toast.needSkillMd'))
    return
  }
  saving.value = true
  try {
    const body = { name, description: form.description.trim(), enabled: form.enabled, files }
    if (form.id) await updateAgentSkill(form.id, body)
    else await createAgentSkill(body)
    feedback.success(t(form.id ? 'agent.skills.toast.saved' : 'agent.skills.toast.created', { name }))
    emit('saved')
    emit('update:modelValue', false)
  } catch {
    // 请求层已经提示过，弹窗留着
  } finally {
    saving.value = false
  }
}
</script>

<style lang="scss">
.ag-sk-cols {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 2.4fr);
  gap: 8px;
}
.ag-sk-panes {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: 8px;
  height: 400px;
}
.ag-sk-tree {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-height: 0;
  padding: 6px;
  border-radius: 10px;
  border: 1px solid var(--m-line);
}
.ag-sk-tree__title {
  padding: 4px 8px;
}
.ag-sk-tree__list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.ag-sk-dir {
  flex: none;
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 26px;
  padding: 0 10px;
  font-size: 12px;
  color: var(--m-ink2);
}
.ag-sk-fr {
  flex: none;
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 8px;
  border: 0;
  background: transparent;
  font: inherit;
  font-size: 13px;
  color: var(--m-ink);
  text-align: left;
  cursor: pointer;
}
.ag-sk-fr:hover {
  background: var(--m-s2);
}
.ag-sk-fr.is-on,
.ag-sk-fr.is-on:hover {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.ag-sk-fr.is-nested {
  padding-left: 26px;
}
.ag-sk-fr__name {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ag-sk-dir .iconify,
.ag-sk-fr .iconify {
  width: 15px;
  height: 15px;
  flex: none;
}
.ag-sk-edit {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-radius: 10px;
  border: 1px solid var(--m-line);
  overflow: hidden;
}
.ag-sk-edit__bar {
  flex: none;
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 0 4px 0 12px;
  background: var(--m-s2);
}
.ag-sk-edit__path {
  flex: 1;
  min-width: 0;
  height: 26px;
  border: 0;
  background: transparent;
  font-family: var(--ag-mono);
  font-size: 12px;
  font-weight: 600;
  color: var(--m-ink);
  outline: none;
}
.ag-sk-edit__path:read-only {
  color: var(--m-ink2);
}
.ag-sk-text {
  flex: 1;
  min-height: 0;
  width: 100%;
  padding: 10px 12px;
  border: 0;
  background: var(--m-bg);
  font-family: var(--ag-mono);
  font-size: 12.5px;
  line-height: 20px;
  color: var(--m-ink);
  resize: none;
  outline: none;
}
.ag-sk-binary {
  flex: 1;
  padding: 10px 12px;
  background: var(--m-bg);
  font-size: 12px;
  color: var(--m-ink2);
}
.ag-sk-on {
  align-self: flex-start;
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 32px;
  padding: 0 10px 0 12px;
  border-radius: 10px;
  border: 1px solid var(--m-line);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
// 手机：两栏上下叠，文件列表限高，编辑区固定一个够用的高度
@media (width < 992px) {
  .ag-sk-panes {
    height: auto;
  }
  .ag-sk-tree__list {
    flex: none;
    max-height: 168px;
  }
  .ag-sk-fr {
    min-height: 36px;
  }
  .ag-sk-edit {
    height: 320px;
  }
  .ag-sk-edit__bar {
    min-height: 44px;
  }
  .ag-sk-on {
    min-height: 40px;
  }
}
</style>
