<!-- 404 / 403 共用场景：大号数字 + 中间贴纸插画（404 黄色笑脸 / 403 主色锁）+ 标题说明 + 详情胶囊 + 返回按钮。
     在外壳内容区居中；背景用淡色圆、圆环、星星点缀（不用渐变）。 -->
<template>
  <div class="err" :class="`err--${code}`">
    <span class="err__blob" aria-hidden="true" />
    <span class="err__ring" aria-hidden="true" />
    <svg class="err__star err__star--a" viewBox="0 0 24 24" aria-hidden="true">
      <path :d="STAR" />
    </svg>
    <svg class="err__star err__star--p" viewBox="0 0 24 24" aria-hidden="true">
      <path :d="STAR" />
    </svg>

    <div class="err__main">
      <div class="err__art" role="img" :aria-label="`HTTP ${code}`">
        <span class="err__digit err__digit--first" aria-hidden="true">4</span>

        <span v-if="is404" class="err__face" aria-hidden="true">
          <span class="err__face-disc" />
          <span class="err__face-ring" />
          <span class="err__face-eye err__face-eye--l" />
          <span class="err__face-eye err__face-eye--r" />
          <svg class="err__face-mouth" viewBox="0 0 60 24"><path d="M6 18c8-10 40-10 48 0" /></svg>
          <span class="err__face-q">?</span>
        </span>
        <span v-else class="err__lock" aria-hidden="true">
          <span class="err__lock-disc" />
          <span class="err__lock-tile">
            <component :is="menuStore.iconComponents['HOutline:LockClosedIcon']" />
          </span>
          <span class="err__lock-ring" />
        </span>

        <span class="err__digit err__digit--last" aria-hidden="true">{{ is404 ? '4' : '3' }}</span>
        <span class="err__tag err__tag--word" aria-hidden="true">{{ sticker }}</span>
        <span class="err__tag err__tag--code m-mono" aria-hidden="true">HTTP {{ code }}</span>
      </div>

      <div class="err__copy">
        <h1 class="err__title">{{ title }}</h1>
        <p class="err__desc">{{ desc }}</p>
        <div v-if="detailValue" class="err__detail">
          <component :is="menuStore.iconComponents[detailIcon]" class="err__detail-ico" />
          <span>{{ detailLabel }}</span>
          <span class="err__detail-val m-mono">{{ detailValue }}</span>
        </div>
      </div>

      <div class="err__actions">
        <UButton color="primary" class="err__btn" icon="i-lucide-house" @click="goHome">
          {{ t('exception.goHome') }}
        </UButton>
        <UButton
          color="neutral"
          variant="outline"
          class="err__btn err__btn--ghost"
          icon="i-lucide-arrow-left"
          @click="goBack"
        >
          {{ t('exception.goBack') }}
        </UButton>
      </div>

      <p v-if="!is404" class="err__switch">
        {{ t('exception.switchAccountHint') }}
        <button type="button" class="err__link" @click="switchAccount">
          {{ t('exception.switchAccount') }}
        </button>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const props = defineProps<{ code: '403' | '404' }>()

const STAR = 'M12 1.5l2.6 7.9 7.9 2.6-7.9 2.6L12 22.5l-2.6-7.9L1.5 12l7.9-2.6z'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const menuStore = useMenuStore()
const userStore = useUserStore()

const is404 = computed(() => props.code === '404')
const title = computed(() =>
  is404.value ? t('exception.notFoundTitle') : t('exception.forbiddenTitle'),
)
const desc = computed(() => (is404.value ? t('exception.notFoundDesc') : t('exception.forbiddenDesc')))
const sticker = computed(() =>
  is404.value ? t('exception.notFoundSticker') : t('exception.forbiddenSticker'),
)
const detailIcon = computed(() => (is404.value ? 'HOutline:LinkIcon' : 'HOutline:UserIcon'))
const detailLabel = computed(() =>
  is404.value ? t('exception.requestPath') : t('exception.currentRole'),
)
const detailValue = computed(() => {
  if (is404.value) return route.fullPath || ''
  return userStore.userInfo ? userStore.getUserRoleName() : ''
})

function goHome() {
  router.replace({ path: '/' })
}

function goBack() {
  if (window.history.length > 1) router.back()
  else goHome()
}

function switchAccount() {
  void userStore.logout()
}
</script>

<style scoped lang="scss">
.err {
  position: relative;
  flex: 1;
  min-height: max(520px, calc(100dvh - 160px));
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 16px 48px;
  overflow: hidden;
  border-radius: var(--app-radius-xl);
}

/* ---- 装饰 ---- */
.err__blob {
  position: absolute;
  right: -150px;
  bottom: -190px;
  width: 480px;
  height: 480px;
  border-radius: 50%;
  background: var(--m-pSoft);
}
.err__ring {
  position: absolute;
  left: 7%;
  top: 14%;
  width: 96px;
  height: 96px;
  border-radius: 50%;
  border: 16px solid var(--m-a2Soft);
}
.err__star {
  position: absolute;
}
.err__star--a {
  right: 14%;
  top: 12%;
  width: 48px;
  height: 48px;
  fill: var(--m-a);
}
.err__star--p {
  left: 16%;
  bottom: 14%;
  width: 28px;
  height: 28px;
  fill: var(--m-p);
}

.err__main {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 30px;
  text-align: center;
  max-width: 100%;
}

/* ---- 数字插画 ---- */
.err__art {
  position: relative;
  display: flex;
  align-items: center;
  gap: 18px;
}
.err__digit {
  font-family: var(--app-font-display);
  font-size: 200px;
  line-height: 0.9;
  font-weight: 800;
  letter-spacing: -0.04em;
  font-variant-numeric: tabular-nums;
  color: var(--m-ink);
}
.err__digit--first {
  transform: rotate(-4deg);
}
.err__digit--last {
  color: var(--m-p);
  transform: rotate(5deg);
}
.err--403 .err__digit--last {
  color: var(--m-a2);
}

.err__face,
.err__lock {
  position: relative;
  flex: none;
  width: 220px;
  height: 220px;
}
.err__face-disc {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  background: var(--m-a);
}
.err__face-ring {
  position: absolute;
  right: -26px;
  bottom: -14px;
  width: 92px;
  height: 92px;
  border-radius: 50%;
  border: 16px solid var(--m-a2);
}
.err__face-eye {
  position: absolute;
  top: 86px;
  width: 20px;
  height: 26px;
  border-radius: 50%;
  background: var(--m-aOn);
}
.err__face-eye--l {
  left: 66px;
}
.err__face-eye--r {
  right: 66px;
}
.err__face-mouth {
  position: absolute;
  left: 80px;
  top: 138px;
  width: 60px;
  height: 24px;
  fill: none;
  stroke: var(--m-aOn);
  stroke-width: 5;
  stroke-linecap: round;
}
.err__face-q {
  position: absolute;
  right: 4px;
  top: -10px;
  width: 58px;
  height: 58px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 18px;
  background: var(--m-ink);
  color: var(--m-bg);
  font-family: var(--app-font-display);
  font-size: 34px;
  font-weight: 800;
  transform: rotate(12deg);
}

.err__lock-disc {
  position: absolute;
  left: -16px;
  top: 20px;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  background: var(--m-a);
}
.err__lock-tile {
  position: absolute;
  inset: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 56px;
  background: var(--m-p);
  color: var(--m-pOn);
  transform: rotate(-8deg);
  box-shadow: var(--app-shadow-brand);
}
.err__lock-tile :deep(svg) {
  width: 104px;
  height: 104px;
  stroke-width: 1.6;
}
.err__lock-ring {
  position: absolute;
  right: -22px;
  bottom: -6px;
  width: 84px;
  height: 84px;
  border-radius: 50%;
  border: 15px solid var(--m-a2);
}

.err__tag {
  position: absolute;
  display: inline-flex;
  align-items: center;
  height: 44px;
  padding: 0 18px;
  border-radius: 999px;
  font-size: 1rem;
  font-weight: 800;
  white-space: nowrap;
  box-shadow: 0 10px 24px -12px color-mix(in srgb, var(--m-ink) 40%, transparent);
}
.err__tag--word {
  left: -70px;
  top: 18px;
  background: var(--m-p);
  color: var(--m-pOn);
  transform: rotate(-9deg);
}
.err--403 .err__tag--word {
  background: var(--m-ink);
  color: var(--m-bg);
}
.err__tag--code {
  right: -40px;
  bottom: 6px;
  background: var(--m-s);
  color: var(--m-ink);
  border: 1.5px solid var(--m-line);
  font-size: 0.875rem;
  transform: rotate(6deg);
}

/* ---- 文案 ---- */
.err__copy {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  max-width: 620px;
}
.err__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 2.625rem;
  line-height: 1.2;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--m-ink);
}
.err__desc {
  margin: 0;
  font-size: 1.0625rem;
  line-height: 1.65;
  color: var(--m-ink2);
}
.err__detail {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  max-width: 100%;
  margin-top: 4px;
  padding: 8px 14px;
  border-radius: 14px;
  background: var(--m-s2);
  font-size: 0.8125rem;
  color: var(--m-ink2);
}
.err__detail-ico {
  width: 16px;
  height: 16px;
  flex: none;
}
.err__detail-val {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 700;
  color: var(--m-ink);
}

.err__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 12px;
}
.err__btn {
  min-height: 56px;
  padding: 0 28px;
  border-radius: 999px;
  font-size: 1rem;
  font-weight: 700;
}
.err__btn:not(.err__btn--ghost) {
  box-shadow: var(--app-shadow-brand);
}
.err__btn--ghost {
  background: var(--m-s);
}
.err__switch {
  margin: -12px 0 0;
  font-size: 0.875rem;
  color: var(--m-ink2);
}
.err__link {
  min-height: 40px;
  padding: 0 4px;
  border: 0;
  background: transparent;
  font: inherit;
  font-weight: 700;
  color: var(--m-pInk);
  cursor: pointer;
}
.err__link:hover {
  text-decoration: underline;
  text-underline-offset: 3px;
}

/* ---- 手机 ---- */
@media (width < 720px) {
  .err {
    min-height: max(480px, calc(100dvh - 200px));
    padding: 24px 4px 36px;
    align-items: flex-start;
  }
  .err__blob {
    right: -120px;
    bottom: -140px;
    width: 300px;
    height: 300px;
  }
  .err__ring {
    left: -26px;
    top: 4%;
    width: 72px;
    height: 72px;
    border-width: 12px;
  }
  .err__star--a {
    right: 8%;
    top: 3%;
    width: 30px;
    height: 30px;
  }
  .err__star--p {
    left: 8%;
    bottom: 6%;
  }
  .err__main {
    gap: 24px;
    width: 100%;
    padding-top: 28px;
  }
  .err__art {
    gap: 6px;
  }
  .err__digit {
    font-size: 104px;
  }
  .err__face,
  .err__lock {
    width: 120px;
    height: 120px;
  }
  .err__face-ring {
    right: -14px;
    bottom: -8px;
    width: 50px;
    height: 50px;
    border-width: 9px;
  }
  .err__face-eye {
    top: 46px;
    width: 11px;
    height: 15px;
  }
  .err__face-eye--l {
    left: 36px;
  }
  .err__face-eye--r {
    right: 36px;
  }
  .err__face-mouth {
    left: 44px;
    top: 74px;
    width: 33px;
    height: 13px;
    stroke-width: 4;
  }
  .err__face-q {
    right: 0;
    top: -8px;
    width: 34px;
    height: 34px;
    border-radius: 11px;
    font-size: 20px;
  }
  .err__lock-disc {
    left: -8px;
    top: 12px;
    width: 66px;
    height: 66px;
  }
  .err__lock-tile {
    inset: 8px;
    border-radius: 32px;
  }
  .err__lock-tile :deep(svg) {
    width: 56px;
    height: 56px;
  }
  .err__lock-ring {
    right: -12px;
    bottom: -4px;
    width: 46px;
    height: 46px;
    border-width: 9px;
  }
  .err__tag {
    height: 34px;
    padding: 0 12px;
    font-size: 0.8125rem;
  }
  .err__tag--word {
    left: -6px;
    top: -26px;
  }
  .err__tag--code {
    right: -4px;
    bottom: -18px;
    font-size: 0.75rem;
  }
  .err__copy {
    padding: 0 8px;
  }
  .err__title {
    font-size: 1.75rem;
  }
  .err__desc {
    font-size: 0.9375rem;
  }
  .err__detail {
    max-width: calc(100vw - 64px);
  }
  .err__actions {
    flex-direction: column;
    width: 100%;
    padding: 0 8px;
  }
  .err__btn {
    width: 100%;
    justify-content: center;
    min-height: 52px;
  }
}
</style>
