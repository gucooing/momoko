<template>
  <UApp>
    <FeedbackBridge>
      <el-config-provider :locale="elementLocale">
        <RouterView />
      </el-config-provider>
    </FeedbackBridge>
  </UApp>
</template>

<script setup lang="ts">
import { elementPlusLocales } from '@/locales'
import { useI18n } from 'vue-i18n'

const { locale } = useI18n()

const elementLocale = computed(() => elementPlusLocales[locale.value as keyof typeof elementPlusLocales])
</script>

<style></style>
