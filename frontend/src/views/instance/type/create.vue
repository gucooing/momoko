<!-- 应用类型 新建/编辑弹窗（第二期）：名称 + 说明 + 默认启动/停止命令（新建应用时预填）+ 启用开关。
     保留 ref 契约 showDialog(row?) + @refresh；第二参数 duplicateFrom 用于「复制为新类型」。 -->
<template>
  <FormDialog
    v-model="open"
    :title="submitForm.id ? t('instance.editInstanceType') : t('instance.ui.newType')"
    :width="520"
    :loading="submitLoading"
    @close="close"
    @confirm="confirm"
  >
    <div class="type-form">
      <!-- 新建时可从模板开始 -->
      <div v-if="showTemplates" class="type-form__tpls">
        <span class="type-form__hint">{{ t('instance.ui.fromTemplate') }}</span>
        <div class="type-form__tpl-row">
          <button
            v-for="tpl in TYPE_TEMPLATES"
            :key="tpl.key"
            type="button"
            class="type-form__tpl"
            :class="{ 'is-on': pickedTpl === tpl.key }"
            :aria-pressed="pickedTpl === tpl.key"
            @click="applyTemplate(tpl)"
          >
            <UIcon :name="typeIconOf(tpl.name)" />{{ tpl.name }}
          </button>
        </div>
      </div>

      <div class="app-field">
        <label class="app-label app-label--required" for="itype-name">{{ t('instance.typeName') }}</label>
        <input
          id="itype-name"
          v-model="submitForm.name"
          class="app-input"
          :class="{ 'is-error': errors.name }"
          :placeholder="t('instance.typeNamePlaceholder')"
          maxlength="50"
          @keyup.enter="confirm"
        />
        <span v-if="errors.name" class="app-field__error">{{ errors.name }}</span>
      </div>

      <div class="app-field">
        <label class="app-label" for="itype-desc">
          {{ t('instance.ui.typeDesc') }}<span class="type-form__opt">{{ t('instance.ui.optional') }}</span>
        </label>
        <textarea
          id="itype-desc"
          v-model="submitForm.description"
          class="app-textarea"
          rows="2"
          maxlength="200"
          :placeholder="t('instance.ui.typeDescPlaceholder')"
        />
      </div>

      <div class="app-field">
        <label class="app-label" for="itype-start">{{ t('instance.ui.defaultStartCmd') }}</label>
        <textarea
          id="itype-start"
          v-model="submitForm.startCommand"
          class="app-textarea m-mono type-form__mono"
          rows="2"
          :placeholder="t('instance.ui.defaultStartCmdPlaceholder')"
        />
        <span class="type-form__hint">{{ t('instance.ui.defaultCmdHint') }}</span>
      </div>

      <div class="app-field">
        <label class="app-label" for="itype-stop">{{ t('instance.ui.defaultStopCmd') }}</label>
        <input
          id="itype-stop"
          v-model="submitForm.stopCommand"
          class="app-input m-mono type-form__mono"
          :placeholder="t('instance.ui.defaultStopCmdPlaceholder')"
        />
        <span class="type-form__hint">{{ t('instance.ui.stopCmdHint') }}</span>
      </div>

      <div class="app-field">
        <label class="app-label" for="itype-quick">{{ t('instanceConsole.quickTitle') }}</label>
        <textarea
          id="itype-quick"
          v-model="submitForm.quickCommands"
          class="app-textarea m-mono type-form__mono"
          rows="4"
          :placeholder="t('instanceConsole.quickPlaceholder')"
        />
        <span class="type-form__hint">{{ t('instanceConsole.quickHint') }}</span>
      </div>

      <div class="type-form__switch">
        <span class="type-form__switch-text">
          <span class="type-form__switch-title">{{ t('instance.isEnabled') }}</span>
          <span class="type-form__hint">{{ t('instance.ui.enableTypeDesc') }}</span>
        </span>
        <AppSwitch v-model="submitForm.isEnable" />
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { createInstanceType, updateInstanceType } from '@/api/instance'
import type { InstanceTypeInfo } from '@/types/v1/instance'
import { useI18n } from 'vue-i18n'
import { TYPE_TEMPLATES, typeIconOf, type TypeTemplate } from '@/views/instance/type/templates'

defineOptions({ name: 'InstanceTypeCreate' })

const emits = defineEmits<{ refresh: [] }>()
const { t } = useI18n()

const open = ref(false)
const submitLoading = ref(false)
const errors = ref<Record<string, string>>({})

const getDefaultForm = () => ({
  id: undefined as string | undefined,
  name: '',
  description: '',
  startCommand: '',
  stopCommand: '',
  quickCommands: '',
  isEnable: true,
})
// 快捷命令在表单里一行一条
const toLines = (items?: string[]) => (items || []).join('\n')
const fromLines = (text: string) =>
  text
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
const submitForm = ref(getDefaultForm())

// —— 模板：仅「从空白新建」时显示；点一下按模板填好名称、说明与命令 ——
const showTemplates = ref(false)
const pickedTpl = ref('')
const applyTemplate = (tpl: TypeTemplate) => {
  pickedTpl.value = tpl.key
  submitForm.value = {
    ...submitForm.value,
    name: tpl.name,
    description: t(tpl.descKey),
    startCommand: tpl.startCommand,
    stopCommand: tpl.stopCommand,
    quickCommands: toLines(tpl.quickCommands),
  }
  errors.value = {}
}

const close = () => {
  open.value = false
  submitLoading.value = false
  errors.value = {}
  submitForm.value = getDefaultForm()
}

const validate = () => {
  const e: Record<string, string> = {}
  if (!submitForm.value.name.trim()) e.name = t('instance.instanceTypeNameRequired')
  errors.value = e
  return Object.keys(e).length === 0
}

const confirm = async () => {
  if (!validate()) return

  const payload = {
    name: submitForm.value.name.trim(),
    isEnable: submitForm.value.isEnable,
    description: submitForm.value.description.trim(),
    startCommand: submitForm.value.startCommand.trim(),
    stopCommand: submitForm.value.stopCommand.trim(),
  }
  const quickCommands = fromLines(submitForm.value.quickCommands)

  submitLoading.value = true
  try {
    if (submitForm.value.id) {
      await updateInstanceType({ id: submitForm.value.id, ...payload, quickCommands: { items: quickCommands } })
    } else {
      await createInstanceType({ ...payload, quickCommands })
    }
    feedback.success(submitForm.value.id ? t('instance.editSuccess') : t('instance.addSuccess'))
    emits('refresh')
    close()
  } finally {
    submitLoading.value = false
  }
}

/** instanceType：编辑该类型；duplicateFrom：以它为模板新建（名称加「副本」） */
const showDialog = (instanceType?: InstanceTypeInfo, duplicateFrom?: InstanceTypeInfo) => {
  errors.value = {}
  pickedTpl.value = ''
  showTemplates.value = !instanceType && !duplicateFrom
  if (instanceType) {
    submitForm.value = {
      id: instanceType.id,
      name: instanceType.name,
      description: instanceType.description || '',
      startCommand: instanceType.startCommand || '',
      stopCommand: instanceType.stopCommand || '',
      quickCommands: toLines(instanceType.quickCommands),
      isEnable: instanceType.isEnable,
    }
  } else if (duplicateFrom) {
    submitForm.value = {
      id: undefined,
      name: t('instance.ui.copyName', { name: duplicateFrom.name }),
      description: duplicateFrom.description || '',
      startCommand: duplicateFrom.startCommand || '',
      stopCommand: duplicateFrom.stopCommand || '',
      quickCommands: toLines(duplicateFrom.quickCommands),
      isEnable: true,
    }
  } else {
    submitForm.value = getDefaultForm()
  }
  open.value = true
}

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.type-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.type-form__tpls {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.type-form__tpl-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.type-form__tpl {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}
.type-form__tpl:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.type-form__tpl.is-on {
  background: var(--m-pSoft);
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.type-form__tpl :deep(.iconify) {
  width: 16px;
  height: 16px;
}
@media (width < 768px) {
  .type-form__tpl-row {
    flex-wrap: nowrap;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .type-form__tpl-row::-webkit-scrollbar {
    display: none;
  }
  .type-form__tpl {
    flex: none;
    min-height: 44px;
  }
}
.type-form__opt {
  margin-left: 2px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.type-form__mono {
  font-family: var(--app-font-mono);
  font-size: 0.8125rem;
}
.type-form__hint {
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.type-form__switch {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
}
.type-form__switch-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.type-form__switch-title {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
</style>
