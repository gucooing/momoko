<template>
  <UApp>
    <RouterView />
    <PToaster />
  </UApp>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import PToaster from '@passport/components/PToaster.vue'
import { useSessionStore } from '@passport/stores/session'

const session = useSessionStore()
onMounted(() => {
  void session.loadConfig().catch(() => undefined)
})
</script>
