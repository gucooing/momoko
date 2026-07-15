<!-- 404：克制居中态，无花哨插画。 -->
<template>
  <div class="exc-page">
    <div class="exc-card">
      <p class="exc-code">404</p>
      <h1 class="exc-title">{{ t('exception.notFoundTitle') }}</h1>
      <p class="exc-desc">{{ t('exception.notFound') }}</p>
      <div class="exc-actions">
        <UButton color="primary" class="exc-btn" @click="goHome">
          {{ t('exception.goHome') }}
        </UButton>
        <UButton color="neutral" variant="ghost" class="exc-btn" @click="goBack">
          {{ t('exception.goBack') }}
        </UButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'NotFoundException' })

const { t } = useI18n()
const router = useRouter()

function goHome() {
  router.replace({ path: '/' })
}

function goBack() {
  if (window.history.length > 1) router.back()
  else goHome()
}
</script>

<style scoped lang="scss">
.exc-page {
  flex: 1;
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 16px;
  box-sizing: border-box;
}

.exc-card {
  width: min(100%, 400px);
  text-align: center;
}

.exc-code {
  margin: 0 0 8px;
  font-size: 3.5rem;
  font-weight: 700;
  letter-spacing: -0.04em;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  color: var(--el-text-color-primary);
  opacity: 0.92;
}

.exc-title {
  margin: 0 0 8px;
  font-size: 1.125rem;
  font-weight: 650;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}

.exc-desc {
  margin: 0 0 22px;
  font-size: 0.875rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.exc-actions {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 8px;
}

.exc-btn {
  height: 36px !important;
  min-width: 6.5rem;
  font-size: 0.8125rem !important;
  font-weight: 600;
}
</style>
