<!-- 检测结果详情弹窗：元信息（模型/推理强度/状态/时间/耗时/Token）+ 请求提示词 + 渲染结果 + 回复原文。 -->
<template>
  <FormDialog v-model="open" :title="t('iqcheck.public.detailTitle')" :width="960" :show-footer="false" @close="open = false">
    <div v-if="result" class="iq-detail">
      <div class="iq-detail__meta">
        <div v-for="item in metaItems" :key="item.label" class="iq-detail__meta-item">
          <span class="iq-detail__meta-label">{{ item.label }}</span>
          <span class="iq-detail__meta-value">{{ item.value }}</span>
        </div>
      </div>

      <section class="iq-detail__section">
        <h4 class="iq-detail__heading">{{ t('iqcheck.public.prompt') }}</h4>
        <div v-if="result.promptTitle" class="iq-detail__prompt-title">{{ result.promptTitle }}</div>
        <pre class="iq-detail__pre">{{ result.promptContent }}</pre>
      </section>

      <section v-if="failed" class="iq-detail__section">
        <h4 class="iq-detail__heading">{{ t('iqcheck.public.error') }}</h4>
        <pre class="iq-detail__pre iq-detail__pre--error">{{ result.error }}</pre>
      </section>

      <template v-else>
        <section class="iq-detail__section">
          <h4 class="iq-detail__heading">{{ t('iqcheck.public.rendered') }}</h4>
          <div class="iq-detail__frame">
            <ReplyFrame :text="result.responseText" :dark="dark" :title="result.model" />
          </div>
        </section>
        <section class="iq-detail__section">
          <h4 class="iq-detail__heading">{{ t('iqcheck.public.raw') }}</h4>
          <pre class="iq-detail__pre">{{ result.responseText }}</pre>
        </section>
      </template>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { IQCheckResultStatus, type IQCheckResultInfo } from '@/types/v1/iqcheck'
import ReplyFrame from '@/views/public/iqcheck/ReplyFrame.vue'

const props = withDefaults(defineProps<{ result: IQCheckResultInfo | null; dark?: boolean }>(), { dark: false })
const open = defineModel<boolean>({ default: false })
const { t } = useI18n()

const failed = computed(() => props.result?.status === IQCheckResultStatus.IQ_CHECK_RESULT_STATUS_FAILED)

const metaItems = computed(() => {
  const r = props.result
  if (!r) return []
  const items = [
    { label: t('iqcheck.public.model'), value: r.model },
    { label: t('iqcheck.public.effort'), value: r.effort || t('iqcheck.public.noEffort') },
    { label: t('iqcheck.public.status'), value: failed.value ? t('iqcheck.public.failed') : t('iqcheck.public.success') },
    { label: t('iqcheck.public.time'), value: r.requestAt ? dayjs(r.requestAt as unknown as string).format('YYYY-MM-DD HH:mm:ss') : '—' },
    { label: t('iqcheck.public.duration'), value: t('iqcheck.public.seconds', { seconds: (Number(r.durationMs) / 1000).toFixed(1) }) },
  ]
  if (Number(r.inputTokens) > 0 || Number(r.outputTokens) > 0) {
    items.push({ label: t('iqcheck.public.tokens'), value: `${r.inputTokens} / ${r.outputTokens}` })
  }
  return items
})
</script>

<style scoped lang="scss">
.iq-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.iq-detail__meta {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  @media (max-width: 640px) {
    grid-template-columns: 1fr 1fr;
  }
}
.iq-detail__meta-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--el-fill-color-light, #f6f8f9);
}
.iq-detail__meta-label {
  font-size: 0.72rem;
  color: var(--el-text-color-secondary, #8c959f);
}
.iq-detail__meta-value {
  font-size: 0.85rem;
  font-weight: 600;
  overflow-wrap: anywhere;
}
.iq-detail__section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.iq-detail__heading {
  margin: 0;
  font-size: 0.85rem;
  font-weight: 600;
}
.iq-detail__prompt-title {
  font-size: 0.8rem;
  color: var(--el-text-color-secondary, #57606a);
}
.iq-detail__pre {
  max-height: 260px;
  margin: 0;
  padding: 10px 12px;
  overflow: auto;
  border-radius: 8px;
  background: var(--el-fill-color-light, #f6f8f9);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.78rem;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  &--error {
    color: var(--el-color-danger, #e5484d);
  }
}
.iq-detail__frame {
  height: 480px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter, #e4e8ee);
  border-radius: 10px;
  background: #fff;
}
</style>
