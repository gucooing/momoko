<!-- 检测结果详情弹窗：元信息格（模型/推理强度/状态/时间/耗时/Token）+ 请求提示词 + 渲染结果（沙箱）+ 回复原文，可复制。 -->
<template>
  <FormDialog v-model="open" :title="t('iqcheck.public.detailTitle')" :width="960" :show-footer="false" @close="open = false">
    <div v-if="result" class="iqd">
      <dl class="iqd__meta">
        <div v-for="item in metaItems" :key="item.label" class="iqd__meta-item">
          <dt>{{ item.label }}</dt>
          <dd :class="[item.cls, { 'm-mono': item.mono }]">{{ item.value }}</dd>
        </div>
      </dl>

      <section class="iqd__sec">
        <div class="iqd__sec-head">
          <h4>{{ t('iqcheck.public.prompt') }}</h4>
          <span v-if="result.promptTitle" class="iqd__ptitle">{{ result.promptTitle }}</span>
          <AppIconButton
            icon="HOutline:ClipboardDocumentIcon"
            :box="36"
            :size="17"
            :label="t('publicPage.iq.copyPrompt')"
            @click="copy(result.promptContent)"
          />
        </div>
        <pre class="iqd__pre">{{ result.promptContent }}</pre>
      </section>

      <section v-if="failed" class="iqd__sec">
        <div class="iqd__sec-head">
          <h4>{{ t('iqcheck.public.error') }}</h4>
          <AppIconButton
            icon="HOutline:ClipboardDocumentIcon"
            :box="36"
            :size="17"
            :label="t('publicPage.iq.copyRaw')"
            @click="copy(result.error)"
          />
        </div>
        <pre class="iqd__pre is-bad">{{ result.error }}</pre>
      </section>

      <template v-else>
        <section class="iqd__sec">
          <div class="iqd__sec-head">
            <h4>{{ t('iqcheck.public.rendered') }}</h4>
            <span class="iqd__sandbox">
              <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />
              {{ t('publicPage.iq.sandbox') }}
            </span>
          </div>
          <div class="iqd__frame">
            <ReplyFrame :text="result.responseText" :dark="dark" :title="result.model" />
          </div>
        </section>
        <section class="iqd__sec">
          <div class="iqd__sec-head">
            <h4>{{ t('iqcheck.public.raw') }}</h4>
            <AppIconButton
              icon="HOutline:ClipboardDocumentIcon"
              :box="36"
              :size="17"
              :label="t('publicPage.iq.copyRaw')"
              @click="copy(result.responseText)"
            />
          </div>
          <pre class="iqd__pre">{{ result.responseText }}</pre>
        </section>
      </template>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { IQCheckResultStatus, type IQCheckResultInfo } from '@/types/v1/iqcheck'
import { copyTextToClipboard } from '@/utils/file'
import { useFeedback } from '@/utils/feedback'
import ReplyFrame from '@/views/public/iqcheck/ReplyFrame.vue'

const props = withDefaults(defineProps<{ result: IQCheckResultInfo | null; dark?: boolean }>(), { dark: false })
const open = defineModel<boolean>({ default: false })
const { t } = useI18n()
const menuStore = useMenuStore()
const fb = useFeedback()

const failed = computed(() => props.result?.status === IQCheckResultStatus.IQ_CHECK_RESULT_STATUS_FAILED)

const metaItems = computed(() => {
  const r = props.result
  if (!r) return []
  const items: { label: string; value: string; cls?: string; mono?: boolean }[] = [
    { label: t('iqcheck.public.model'), value: r.model, mono: true },
    { label: t('iqcheck.public.effort'), value: r.effort || t('iqcheck.public.noEffort') },
    {
      label: t('iqcheck.public.status'),
      value: failed.value ? t('iqcheck.public.failed') : t('iqcheck.public.success'),
      cls: failed.value ? 'is-bad' : 'is-ok',
    },
    {
      label: t('iqcheck.public.time'),
      value: r.requestAt ? dayjs(r.requestAt as unknown as string).format('YYYY-MM-DD HH:mm:ss') : '—',
    },
    {
      label: t('iqcheck.public.duration'),
      value: t('iqcheck.public.seconds', { seconds: (Number(r.durationMs) / 1000).toFixed(1) }),
    },
  ]
  if (Number(r.inputTokens) > 0 || Number(r.outputTokens) > 0) {
    items.push({ label: t('iqcheck.public.tokens'), value: `${r.inputTokens} / ${r.outputTokens}` })
  }
  return items
})

const copy = async (text: string) => {
  try {
    await copyTextToClipboard(text || '')
    fb.success(t('file.share.copied'))
  } catch {
    fb.error(t('file.share.copyFailed'))
  }
}
</script>

<style scoped lang="scss">
.iqd {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.iqd__meta {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px 16px;
  margin: 0;
  padding: 14px 16px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  @media (max-width: 640px) {
    grid-template-columns: 1fr 1fr;
  }
}
.iqd__meta-item {
  min-width: 0;
  dt {
    font-size: 0.6875rem;
    font-weight: 700;
    color: var(--el-text-color-secondary);
  }
  dd {
    margin: 2px 0 0;
    font-size: 0.875rem;
    font-weight: 700;
    overflow-wrap: anywhere;
  }
  dd.is-ok {
    color: var(--m-ok);
  }
  dd.is-bad {
    color: var(--m-bad);
  }
}
.iqd__sec {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.iqd__sec-head {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 36px;
  h4 {
    margin: 0;
    font-size: 0.9375rem;
    font-weight: 800;
  }
}
.iqd__ptitle {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iqd__sec-head h4 + :deep(.app-icon-btn) {
  margin-left: auto;
}
.iqd__sandbox {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  margin-left: auto;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  svg {
    width: 13px;
    height: 13px;
  }
}
.iqd__pre {
  max-height: 260px;
  margin: 0;
  padding: 12px 14px;
  overflow: auto;
  scrollbar-width: none;
  border-radius: 14px;
  background: var(--el-fill-color-light);
  font-family: var(--app-font-mono);
  font-size: 0.78rem;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  &::-webkit-scrollbar {
    display: none;
  }
  &.is-bad {
    color: var(--m-bad);
  }
}
.iqd__frame {
  height: 480px;
  overflow: hidden;
  border: 1.5px dashed var(--el-border-color-light);
  border-radius: var(--app-radius);
  @media (max-width: 640px) {
    height: 360px;
  }
}
</style>
