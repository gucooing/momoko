<!-- 工具页：无业务 UI，仅短暂占位后 replace 到目标路径。 -->
<template>
  <div class="redir-page" role="status" aria-live="polite">
    <img :src="APP_CONFIG.logoSrc" alt="" class="redir-logo" />
    <p class="redir-text">{{ t('utils.redirecting') }}</p>
  </div>
</template>

<script setup lang="ts">
import { APP_CONFIG } from '@/config/app.config'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'RedirectView' })

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const { params, query, hash } = route
const path = params.path as string

nextTick(() => {
  setTimeout(() => {
    router.replace({
      path: '/' + (path || ''),
      query,
      hash,
    })
  }, 80)
})
</script>

<style scoped lang="scss">
.redir-page {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  background: var(--el-bg-color-page);
}

.redir-logo {
  width: 40px;
  height: 40px;
  object-fit: contain;
  opacity: 0.9;
  animation: redir-pulse 1.4s ease-in-out infinite;
}

.redir-text {
  margin: 0;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}

@keyframes redir-pulse {
  0%,
  100% {
    opacity: 0.95;
    transform: scale(1);
  }
  50% {
    opacity: 0.55;
    transform: scale(1.04);
  }
}
</style>
