<!-- 反馈桥：无渲染包裹组件。置于 <UApp> 子树内，setup 时捕获 toast 单例，
     供 axios 拦截器 / store action / util 等非组件上下文调用 `feedback.*`。 -->
<template>
  <slot />
</template>

<script setup lang="ts">
import { initFeedbackSingleton } from '@/utils/feedback'

initFeedbackSingleton()
</script>
