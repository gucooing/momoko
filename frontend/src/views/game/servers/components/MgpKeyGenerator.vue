<!-- MGP 密钥：可以填写已有的 Key ID 与密钥（如服务端配置里的 mgp.key_id / mgp.secret），也可以点「生成」在前端新建一组。
     两项都填且格式正确时才把密钥交给父组件，否则为 null；粘贴 `key_id: … / secret: …`、配置片段或 keygen 输出会自动拆成两项。
     仅 GM：填服务端已在用的密钥就不用改服务端配置；新生成的要写进服务端配置。托管：Momoko 把它写进服务端配置。 -->
<template>
  <div class="mkg">
    <p v-if="currentKeyId && !value" class="mkg__current">
      {{ t('gameServers.key.current') }} <code class="m-mono">{{ currentKeyId }}</code>
    </p>
    <div class="mkg__row">
      <span class="mkg__label">{{ t('gameServers.key.keyId') }}</span>
      <UInput
        v-model="draft.keyId"
        class="m-mono"
        :ui="{ root: 'mkg__input' }"
        :placeholder="currentKeyId ? t('gameServers.key.keepPh') : t('gameServers.key.keyIdPh')"
        :disabled="disabled"
        :aria-label="t('gameServers.key.keyId')"
        autocomplete="off"
        spellcheck="false"
        @paste="onPaste"
      />
      <AppIconButton
        icon="HOutline:ClipboardDocumentIcon"
        :label="t('gameServers.key.copyKeyId')"
        :box="30"
        :disabled="!parsed.keyId"
        @click="copy(parsed.keyId)"
      />
    </div>
    <div class="mkg__row">
      <span class="mkg__label">{{ t('gameServers.key.secret') }}</span>
      <UInput
        v-model="draft.secret"
        class="m-mono"
        :ui="{ root: 'mkg__input' }"
        :placeholder="currentKeyId ? t('gameServers.key.keepPh') : t('gameServers.key.secretPh')"
        :disabled="disabled"
        :aria-label="t('gameServers.key.secret')"
        autocomplete="off"
        spellcheck="false"
        @paste="onPaste"
      />
      <AppIconButton
        icon="HOutline:ClipboardDocumentIcon"
        :label="t('gameServers.key.copySecret')"
        :box="30"
        :disabled="!parsed.secret"
        @click="copy(parsed.secret)"
      />
    </div>
    <p v-if="error" class="mkg__error">{{ error }}</p>
    <p v-else-if="isGenerated && hint" class="mkg__hint">{{ hint }}</p>
    <p v-else-if="!value && !currentKeyId" class="mkg__hint">{{ t('gameServers.key.fillHint') }}</p>
    <div class="mkg__actions">
      <UButton color="neutral" variant="outline" icon="i-lucide-key-round" :disabled="disabled" @click="generate">
        {{ hasDraft ? t('gameServers.key.regenerate') : t('gameServers.key.generate') }}
      </UButton>
      <UButton v-if="value" color="neutral" variant="ghost" icon="i-lucide-copy" :disabled="disabled" @click="copyBoth">
        {{ t('gameServers.key.copyBoth') }}
      </UButton>
      <UButton v-if="hasDraft && clearable" color="neutral" variant="ghost" :disabled="disabled" @click="clear">
        {{ t('gameServers.key.discard') }}
      </UButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { copyTextToClipboard } from '@/utils/file'
import { generateMgpKey, parseMgpKeyText, validateMgpKey, type MgpKey } from '../mgpKey'

const props = defineProps<{
  modelValue: MgpKey | null
  /** 当前在用的 Key ID（没填新密钥时显示，留空即沿用） */
  currentKeyId?: string
  /** 点「生成」后显示的提示（新密钥要写进服务端配置） */
  hint?: string
  /** 可以放弃填写的新密钥（回到沿用当前密钥） */
  clearable?: boolean
  disabled?: boolean
}>()
const emit = defineEmits<{ 'update:modelValue': [value: MgpKey | null] }>()

const { t } = useI18n()
const fb = useFeedback()

const draft = reactive({ keyId: '', secret: '' })
// 最近一次点「生成」得到的密钥：输入框里仍是它时才显示「写进服务端配置」的提示
const lastGenerated = ref<MgpKey | null>(null)

// 去掉首尾空白和从配置里带出来的引号
const clean = (s: string) => s.trim().replace(/^(["'])(.*)\1$/, '$2')
const parsed = computed<MgpKey>(() => ({ keyId: clean(draft.keyId), secret: clean(draft.secret) }))
const hasDraft = computed(() => !!(draft.keyId.trim() || draft.secret.trim()))

const error = computed(() => {
  const { keyId, secret } = parsed.value
  if (!keyId && !secret) return ''
  const bad = validateMgpKey(keyId, secret)
  if (bad === 'keyId') return t('gameServers.key.badKeyId')
  if (bad === 'secret') return t('gameServers.key.badSecret')
  if (!keyId) return t('gameServers.key.needKeyId')
  if (!secret) return t('gameServers.key.needSecret')
  return ''
})
const value = computed<MgpKey | null>(() => (parsed.value.keyId && parsed.value.secret && !error.value ? { ...parsed.value } : null))

const same = (a: MgpKey | null | undefined, b: MgpKey | null | undefined) => a?.keyId === b?.keyId && a?.secret === b?.secret
const isGenerated = computed(() => !!value.value && same(value.value, lastGenerated.value))

watch(value, (v) => {
  if (!same(v, props.modelValue)) emit('update:modelValue', v)
})
// 父组件改了值（重置、回填）才覆盖输入框；值没变（包括填到一半时的 null）不动
watch(
  () => props.modelValue,
  (v) => {
    if (same(v, value.value)) return
    draft.keyId = v?.keyId ?? ''
    draft.secret = v?.secret ?? ''
  },
  { immediate: true },
)

const generate = () => {
  const k = generateMgpKey()
  lastGenerated.value = k
  draft.keyId = k.keyId
  draft.secret = k.secret
}

const clear = () => {
  draft.keyId = ''
  draft.secret = ''
}

// 一次粘贴了两项（key_id: … / secret: …、配置片段或两行输出）时拆开填
const onPaste = (e: ClipboardEvent) => {
  const k = parseMgpKeyText(e.clipboardData?.getData('text') ?? '')
  if (!k) return
  e.preventDefault()
  draft.keyId = k.keyId
  draft.secret = k.secret
}

const copy = async (text: string) => {
  if (!text) return
  try {
    await copyTextToClipboard(text)
    fb.success(t('gameServers.key.copied'))
  } catch {
    fb.error(t('gameServers.key.copyFailed'))
  }
}

const copyBoth = () => {
  if (!value.value) return
  void copy(`key_id: ${value.value.keyId}\nsecret: ${value.value.secret}`)
}
</script>

<style scoped lang="scss">
.mkg {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.mkg__row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.mkg__label {
  flex: none;
  width: 64px;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.mkg__row :deep(.mkg__input) {
  flex: 1;
  min-width: 0;
}
.mkg__row :deep(.app-icon-btn:disabled) {
  opacity: 0.4;
  cursor: default;
  background: transparent;
}
.mkg__hint,
.mkg__current,
.mkg__error {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.mkg__error {
  color: var(--m-bad);
}
.mkg__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
@media (max-width: 640px) {
  .mkg__label {
    width: 52px;
  }
}
</style>
