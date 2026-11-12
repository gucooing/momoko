<!-- MGP 密钥：点「生成」在前端生成一组 Key ID 与密钥，生成后可复制（没有输入框）。
     仅 GM 的服务器要把它写进服务端配置（mgp.key_id / mgp.secret）；托管的由 Momoko 写入。 -->
<template>
  <div class="mkg">
    <div v-if="modelValue" class="mkg__values">
      <div class="mkg__row">
        <span class="mkg__label">{{ t('gameServers.key.keyId') }}</span>
        <code class="mkg__value m-mono">{{ modelValue.keyId }}</code>
        <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('gameServers.key.copyKeyId')" :box="36" @click="copy(modelValue.keyId)" />
      </div>
      <div class="mkg__row">
        <span class="mkg__label">{{ t('gameServers.key.secret') }}</span>
        <code class="mkg__value m-mono">{{ modelValue.secret }}</code>
        <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('gameServers.key.copySecret')" :box="36" @click="copy(modelValue.secret)" />
      </div>
      <p v-if="hint" class="mkg__hint">{{ hint }}</p>
    </div>
    <p v-else-if="currentKeyId" class="mkg__current">
      {{ t('gameServers.key.current') }} <code class="m-mono">{{ currentKeyId }}</code>
    </p>
    <div class="mkg__actions">
      <UButton color="neutral" variant="outline" icon="i-lucide-key-round" :disabled="disabled" @click="generate">
        {{ modelValue ? t('gameServers.key.regenerate') : t('gameServers.key.generate') }}
      </UButton>
      <UButton v-if="modelValue" color="neutral" variant="ghost" icon="i-lucide-copy" :disabled="disabled" @click="copyBoth">
        {{ t('gameServers.key.copyBoth') }}
      </UButton>
      <UButton v-if="modelValue && clearable" color="neutral" variant="ghost" :disabled="disabled" @click="emit('update:modelValue', null)">
        {{ t('gameServers.key.discard') }}
      </UButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { copyTextToClipboard } from '@/utils/file'
import { generateMgpKey, type MgpKey } from '../mgpKey'

const props = defineProps<{
  modelValue: MgpKey | null
  /** 当前在用的 Key ID（还没生成新密钥时显示） */
  currentKeyId?: string
  /** 生成后显示的提示 */
  hint?: string
  /** 可以放弃新生成的密钥（回到沿用当前密钥） */
  clearable?: boolean
  disabled?: boolean
}>()
const emit = defineEmits<{ 'update:modelValue': [value: MgpKey | null] }>()

const { t } = useI18n()
const fb = useFeedback()

const generate = () => emit('update:modelValue', generateMgpKey())

const copy = async (text: string) => {
  try {
    await copyTextToClipboard(text)
    fb.success(t('gameServers.key.copied'))
  } catch {
    fb.error(t('gameServers.key.copyFailed'))
  }
}

const copyBoth = () => {
  if (!props.modelValue) return
  void copy(`key_id: ${props.modelValue.keyId}\nsecret: ${props.modelValue.secret}`)
}
</script>

<style scoped lang="scss">
.mkg {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}
.mkg__values {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
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
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.mkg__value {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  word-break: break-all;
  user-select: all;
}
.mkg__hint,
.mkg__current {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.mkg__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
