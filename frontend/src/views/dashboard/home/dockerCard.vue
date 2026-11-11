<!-- Docker 概况：运行中 / 已停止 / 已退出（非零退出码）三个数字 + 比例条 + 镜像 / 网络 / 引擎状态。 -->
<template>
  <section class="dk" aria-labelledby="dk-h">
    <div class="dk__head">
      <h2 id="dk-h" class="dk__title">Docker</h2>
      <span class="dk__spacer" />
      <RouterLink to="/docker" class="dk__link">{{ t('dashboard.home.manage') }} →</RouterLink>
    </div>

    <template v-if="status?.connected">
      <div class="dk__nums">
        <div>
          <div class="dk__n m-num" style="color: var(--m-ok)">{{ counts.running }}</div>
          <div class="dk__l">{{ t('dashboard.home.dockerRunning') }}</div>
        </div>
        <div>
          <div class="dk__n m-num">{{ counts.stopped }}</div>
          <div class="dk__l">{{ t('dashboard.home.dockerStopped') }}</div>
        </div>
        <div>
          <div class="dk__n m-num" :style="{ color: counts.exited ? 'var(--m-bad)' : undefined }">{{ counts.exited }}</div>
          <div class="dk__l">{{ t('dashboard.home.dockerExited') }}</div>
        </div>
      </div>
      <div
        class="dk__bar"
        role="img"
        :aria-label="t('dashboard.home.dockerBar', { running: counts.running, stopped: counts.stopped, exited: counts.exited })"
      >
        <span v-if="counts.running" :style="{ flex: counts.running, background: 'var(--m-ok)' }" />
        <span v-if="counts.stopped" class="is-off" :style="{ flex: counts.stopped }" />
        <span v-if="counts.exited" :style="{ flex: counts.exited, background: 'var(--m-bad)' }" />
        <span v-if="!counts.total" class="is-off" style="flex: 1" />
      </div>
      <div class="dk__foot m-num">
        {{ t('dashboard.home.dockerFoot', { images: status.info?.images ?? 0, networks: status.networks ?? 0 }) }}
      </div>
    </template>

    <div v-else class="dk__off">
      <span class="dk__sk m-sticker m-sticker--warn" aria-hidden="true">
        <component :is="ic('HOutline:ExclamationTriangleIcon')" />
      </span>
      <span class="dk__off-text">
        <b>{{ status?.enabled === false ? t('dashboard.home.dockerDisabled') : t('dashboard.home.dockerDisconnected') }}</b>
        <small>{{ status?.error || t('dashboard.home.dockerDisconnectedDesc') }}</small>
      </span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { DockerStatusResponse } from '@/types/v1/docker'

defineProps<{
  status: DockerStatusResponse | null
  counts: { running: number; stopped: number; exited: number; total: number }
}>()

const { t } = useI18n()
const menuStore = useMenuStore()
const ic = (name: string) => menuStore.iconComponents[name]
</script>

<style scoped lang="scss">
.dk {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
  padding: calc(var(--cmp-card-unit) * 22);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.dk__head {
  display: flex;
  align-items: center;
}
.dk__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.dk__spacer {
  flex: 1;
}
.dk__link {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--m-pInk);
  text-decoration: none;
}
.dk__link:hover {
  color: var(--el-text-color-primary);
}
.dk__nums {
  display: flex;
  gap: 20px;
}
.dk__n {
  font-size: 2rem;
  font-weight: 800;
  line-height: 1.1;
  color: var(--el-text-color-primary);
}
.dk__l {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.dk__bar {
  display: flex;
  gap: 3px;
  height: 12px;
  border-radius: 999px;
  overflow: hidden;
}
.dk__bar span.is-off {
  background: color-mix(in srgb, var(--m-ink2) 40%, transparent);
}
.dk__foot {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.dk__off {
  display: flex;
  align-items: center;
  gap: 12px;
}
.dk__sk {
  width: 40px;
  height: 40px;
  border-radius: 13px;
}
.dk__sk :deep(svg) {
  width: 20px;
  height: 20px;
}
.dk__off-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  b {
    font-size: 0.875rem;
    font-weight: 700;
  }
  small {
    font-size: 0.75rem;
    color: var(--el-text-color-secondary);
    overflow-wrap: anywhere;
  }
}
</style>
