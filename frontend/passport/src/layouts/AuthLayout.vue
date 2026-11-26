<!-- 登录、注册与找回密码的布局：无外壳，居中卡片。 -->
<template>
  <div class="auth">
    <header class="auth__top">
      <RouterLink to="/" class="auth__brand">
        <span class="auth__logo pp-sticker">M</span>
        <span class="auth__name">{{ session.config?.siteName || t('nav.brand') }}</span>
      </RouterLink>
      <span class="auth__spacer" />
      <select class="auth__lang" :value="locale" :aria-label="t('auth.language')" @change="onLang">
        <option v-for="l in LOCALES" :key="l" :value="l">{{ t(`auth.languages.${l}`) }}</option>
      </select>
      <button type="button" class="auth__ib" :aria-label="t('nav.toggleDark')" @click="ui.toggleDark()">
        <component :is="ui.dark ? SunIcon : MoonIcon" aria-hidden="true" />
      </button>
    </header>
    <main class="auth__main">
      <section class="auth__card pp-card">
        <span class="auth__deco" aria-hidden="true" />
        <h1 class="auth__title pp-title">{{ title }}</h1>
        <p v-if="subtitle" class="auth__sub">{{ subtitle }}</p>
        <slot />
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { MoonIcon, SunIcon } from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'
import { LOCALES, setLocale } from '@passport/locales'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'

defineProps<{ title: string; subtitle?: string }>()

const { t, locale } = useI18n()
const ui = useUiStore()
const session = useSessionStore()

const onLang = (e: Event) => setLocale((e.target as HTMLSelectElement).value)
</script>

<style scoped>
.auth {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--m-bg);
}
.auth__top {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px;
}
.auth__brand {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--m-ink);
  text-decoration: none;
}
.auth__logo {
  width: 30px;
  height: 30px;
  border-radius: 12px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-size: 16px;
  transform: rotate(-6deg);
}
.auth__name {
  font-size: 15px;
  font-weight: 600;
}
.auth__spacer {
  flex: 1;
}
.auth__lang {
  min-height: 40px;
  padding: 0 10px;
  border-radius: 999px;
  border: 1px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  font-size: 13px;
}
.auth__ib {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--m-ink2);
  cursor: pointer;
}
.auth__ib svg {
  width: 22px;
  height: 22px;
}
.auth__main {
  flex: 1;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 20px 16px 42px;
}
.auth__card {
  position: relative;
  width: 100%;
  max-width: 440px;
  padding: 20px 18px 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow: hidden;
}
.auth__deco {
  position: absolute;
  top: -40px;
  right: -40px;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  background: var(--m-aSoft);
  border: 10px solid var(--m-a2Soft);
}
.auth__title {
  position: relative;
  font-size: 22px;
}
.auth__sub {
  position: relative;
  margin: -6px 0 4px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--m-ink2);
}
@media (max-width: 768px) {
  .auth__top {
    padding: 10px 12px 10px 16px;
  }
  .auth__main {
    padding: 12px 12px 30px;
  }
  .auth__card {
    padding: 16px 14px 16px;
  }
}
</style>
