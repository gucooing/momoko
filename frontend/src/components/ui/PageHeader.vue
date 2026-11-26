<!-- 页头：标题 + 一句描述 + 右侧主操作（05 P1/P2）。页面第一焦点是标题，不再包大卡。
     手机端导航在顶栏的菜单抽屉里；:back 的页面（如外观设置）在手机上标题左侧带返回按钮。 -->
<template>
  <header class="page-header">
    <AppIconButton
      v-if="showBack"
      class="page-header__back"
      icon="HOutline:ChevronLeftIcon"
      :label="t('shell.back')"
      :box="40"
      :size="20"
      @click="goBack"
    />
    <div class="page-header__text">
      <h1 class="page-header__title"><slot name="title">{{ title }}</slot></h1>
      <p v-if="description || $slots.description" class="page-header__desc">
        <slot name="description">{{ description }}</slot>
      </p>
    </div>
    <div v-if="$slots.actions" class="page-header__actions"><slot name="actions" /></div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  title?: string
  description?: string
  /** 手机端标题左侧的返回按钮 */
  back?: boolean
}>()

const { t } = useI18n()
const router = useRouter()
const menuStore = useMenuStore()

const showBack = computed(() => menuStore.isMobile && props.back)

const goBack = () => {
  const state = window.history.state as { back?: string | null } | null
  if (state?.back) router.back()
  else void router.push('/')
}
</script>

<style scoped lang="scss">
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.page-header__text {
  min-width: 0;
  flex: 1;
}
.page-header__title {
  font-family: var(--app-font-display);
  font-size: var(--cmp-page-title-size);
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
  line-height: 1.2;
}
.page-header__desc {
  margin-top: 6px;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
}
.page-header__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  flex-wrap: wrap;
  justify-content: flex-end;
}
/* 动作区按钮：统一 40px 全圆角药丸（含带 to 的 UButton，它渲染成 <a data-slot="base">） */
.page-header__actions :deep(button),
.page-header__actions :deep(a:where([data-slot='base'])) {
  min-height: 40px;
  height: 40px;
  padding-inline: 16px;
  border-radius: 999px;
  font-size: 0.875rem;
  font-weight: 700;
}
.page-header__actions :deep(button svg),
.page-header__actions :deep(a:where([data-slot='base']) svg) {
  width: 16px;
  height: 16px;
}
.page-header__back {
  flex: none;
  margin: -6px 0 0 -10px;
}
@media (width <= 768px) {
  .page-header {
    align-items: flex-start;
    gap: 8px;
  }
  .page-header__title {
    font-size: calc(var(--cmp-page-title-size) * 0.7);
  }
  .page-header__desc {
    display: -webkit-box;
    -webkit-line-clamp: 1;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .page-header__actions :deep(button),
  .page-header__actions :deep(a:where([data-slot='base'])) {
    min-height: 38px;
    height: 38px;
    padding-inline: 14px;
  }
}
</style>
