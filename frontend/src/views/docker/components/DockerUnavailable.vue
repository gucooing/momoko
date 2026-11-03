<!-- 引擎不可用时列表页的空状态：说明原因 + 去配置。disabled（未启用）/ disconnected（连不上，附错误）。 -->
<template>
  <section class="dk-off" :class="`is-${state}`">
    <span class="dk-off__sticker m-sticker" :class="state === 'disconnected' ? 'm-sticker--bad' : 'm-sticker--a2'">
      <component :is="menuStore.iconComponents[state === 'disconnected' ? 'HOutline:SignalSlashIcon' : 'HOutline:CubeTransparentIcon']" />
    </span>
    <h2 class="dk-off__title">{{ state === 'disconnected' ? t('docker.engine.offTitle') : t('docker.engine.disabledTitle') }}</h2>
    <p class="dk-off__desc">{{ state === 'disconnected' ? t('docker.engine.offDesc') : t('docker.engine.disabledDesc') }}</p>
    <code v-if="state === 'disconnected' && error" class="dk-off__err m-mono">{{ error }}</code>
    <div class="dk-off__acts">
      <UButton v-if="canConfig" color="primary" icon="i-lucide-settings" @click="router.push(pageTabTo('docker', 'config'))">
        {{ t('docker.engine.goConfig') }}
      </UButton>
      <UButton color="neutral" variant="soft" icon="i-lucide-refresh-cw" :loading="loading" @click="emit('retry')">
        {{ t('docker.common.refresh') }}
      </UButton>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { pageTabTo } from '@/config/pageGroups'
import { useDockerStatus } from './useDockerStatus'

const emit = defineEmits<{ retry: [] }>()
const { t } = useI18n()
const router = useRouter()
const menuStore = useMenuStore()
const { status, state, loading } = useDockerStatus()
const error = computed(() => status.value?.error || '')
const canConfig = computed(() => menuStore.hasButtonPermission('docker_config:edit'))
</script>

<style scoped lang="scss">
.dk-off {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 56px 24px 52px;
  border-radius: var(--app-radius-xl);
  border: 1.5px dashed var(--el-border-color-light);
  background: var(--el-bg-color);
  text-align: center;
}
.dk-off__sticker {
  width: 64px;
  height: 64px;
  border-radius: 20px;
  transform: rotate(-6deg);
  margin-bottom: 6px;
}
.dk-off__sticker :deep(svg) {
  width: 30px;
  height: 30px;
}
.dk-off__title {
  font-family: var(--app-font-display);
  font-size: 1.375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.dk-off__desc {
  max-width: 460px;
  font-size: 0.875rem;
  line-height: 1.55;
  color: var(--el-text-color-secondary);
}
.dk-off__err {
  max-width: min(560px, 100%);
  padding: 8px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-badSoft);
  color: var(--m-bad);
  font-size: 0.75rem;
  word-break: break-all;
}
.dk-off__acts {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  margin-top: 8px;
}
.dk-off__acts :deep(button) {
  min-height: 40px;
  padding-inline: 16px;
  border-radius: 999px;
  font-weight: 700;
}
</style>
