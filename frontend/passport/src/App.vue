<template>
  <UApp>
    <ThemeBackdrop :bg="themeBackground" :scheme="ui.dark ? 'dark' : 'light'" :resolver="themeResolver" />
    <RouterView />
    <PToaster />
  </UApp>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import ThemeBackdrop from '@/components/theme/ThemeBackdrop.vue'
import PToaster from '@passport/components/PToaster.vue'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { themeBackground, themeResolver } from '@passport/theme'

const session = useSessionStore()
const ui = useUiStore()
onMounted(() => {
  void session.loadConfig().catch(() => undefined)
})
</script>
