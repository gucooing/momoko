<!-- 主题的小预览（外观页主题卡片、当前主题卡片、导入前预览）：把主题的完整色表写成局部 --m-* 变量，
     画一个简化的界面（侧栏 + 主色横幅 + 卡片），背景 / 遮罩 / 毛玻璃与真实界面同一套规则。
     variant：mini（卡片缩略图）/ dash（工作台）/ login（登录页）/ term（终端）/ phone（手机）。 -->
<template>
  <div class="tsc" :class="[`tsc--${variant}`, { 'is-glass': glass, 'is-lg': large }]" :style="rootStyle">
    <span class="tsc__bg" :style="bgStyle" />
    <span v-if="dimPct > 0" class="tsc__dim" :style="{ background: `color-mix(in srgb, var(--m-bg) ${dimPct}%, transparent)` }" />

    <template v-if="variant === 'mini'">
      <span v-if="!compact" class="tsc-mini__sb tsc-glass"><i class="is-on" /><i /><i /></span>
      <span class="tsc-mini__main">
        <span class="tsc-mini__hero" :style="heroStyle"><i v-if="look.deco" /></span>
        <span class="tsc-mini__row"><i class="tsc-glass" /><i class="tsc-glass" /><i class="is-acc" /></span>
      </span>
    </template>

    <template v-else-if="variant === 'dash'">
      <span class="tsc-dash">
        <span class="tsc-dash__sb tsc-glass">
          <b>Momoko</b><i class="is-on" /><i /><i /><i />
        </span>
        <span class="tsc-dash__main">
          <span class="tsc-dash__hero" :style="heroStyle">
            <span class="tsc-dash__shade" :style="{ background: look.shade }" />
            <i v-if="look.deco" class="tsc-dash__blob" />
            <span class="tsc-dash__copy">
              <b>{{ t('theme.scene.allGood') }}</b>
              <span><i :style="{ background: look.btnBg }" /><i :style="{ background: `color-mix(in srgb, ${look.soft} 22%, transparent)` }" /></span>
            </span>
          </span>
          <span class="tsc-dash__cards">
            <span class="tsc-glass">CPU<b class="m-num">16.5%</b></span>
            <span class="tsc-glass">{{ t('theme.scene.memory') }}<b class="m-num">46%</b></span>
            <span class="tsc-glass">{{ t('theme.scene.apps') }}<b class="m-num is-ok">{{ t('theme.scene.running', { n: 5 }) }}</b></span>
          </span>
        </span>
      </span>
    </template>

    <template v-else-if="variant === 'login'">
      <span class="tsc-login tsc-glass">
        <span class="tsc-login__brand"><i />Momoko</span>
        <i class="tsc-login__inp" />
        <i class="tsc-login__inp" />
        <span class="tsc-login__btn">{{ t('theme.scene.login') }}</span>
      </span>
    </template>

    <template v-else-if="variant === 'term'">
      <span class="tsc-term m-mono">
        <span><i :style="{ color: tok.ansi2 }">[Server]</i> Done (6.214s)! For help, type "help"</span>
        <span><i :style="{ color: tok.ansi6 }">[Server]</i> momo joined the game</span>
        <span><i :style="{ color: tok.ansi3 }">[Server]</i> Can't keep up! Running 2041ms behind</span>
        <span><i :style="{ color: tok.ansi1 }">[Server]</i> Error: Could not pass event to plugin</span>
        <span><i :style="{ color: tok.termCursor }">❯</i> list</span>
      </span>
    </template>

    <template v-else>
      <span class="tsc-phone">
        <b>{{ t('theme.scene.greeting') }}</b>
        <span class="tsc-phone__hero" :style="heroStyle"><i v-if="look.deco" /></span>
        <span class="tsc-phone__row"><i class="tsc-glass" /><i class="tsc-glass" /></span>
        <i class="tsc-phone__card tsc-glass" />
      </span>
    </template>

    <slot />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ThemeBackground, ThemeContent } from '@/types/v1/theme'
import {
  GLASS_TYPES,
  assetUrl,
  backgroundCss,
  bannerLook,
  effectiveDim,
  resolveBackground,
  resolveBanner,
  resolveTokens,
  tokensToCssVars,
  type AssetResolver,
  type BgType,
} from '@/theme/content'
import { imageStats } from '@/theme/imageStats'

const props = withDefaults(
  defineProps<{
    content: ThemeContent
    scheme: 'light' | 'dark'
    variant?: 'mini' | 'dash' | 'login' | 'term' | 'phone'
    /** 页面单独设置按这个页面取（导入预览里登录页 / 终端用） */
    pageKey?: string | null
    /** 外观页未保存的背景修改 */
    override?: ThemeBackground | null
    resolver?: AssetResolver
    /** mini 不画侧栏 */
    compact?: boolean
    /** mini 的大尺寸（外观页「当前主题」） */
    large?: boolean
  }>(),
  { variant: 'mini', pageKey: null, override: null, resolver: undefined, compact: false, large: false },
)

const { t } = useI18n()

const tok = computed(() => resolveTokens(props.content, props.scheme, props.pageKey))
const resolved = computed(() => resolveBackground(props.content, props.pageKey, props.override))
const glass = computed(() => GLASS_TYPES.includes(resolved.value.bg.type as BgType))
const dimPct = computed(() => effectiveDim(resolved.value.bg, props.scheme))

const banner = computed(() => resolveBanner(props.content, props.pageKey))
const bannerUrl = computed(() => (banner.value.type === 'image' ? assetUrl(banner.value.image, props.resolver) : ''))
const look = computed(() => bannerLook(banner.value, props.resolver, bannerUrl.value ? imageStats(bannerUrl.value)?.leftLuma : null))

const rootStyle = computed(() => {
  const vars = tokensToCssVars(tok.value)
  return `${vars} --tsc-card-a: ${resolved.value.cardOpacity}%; color: var(--m-ink);`
})
const bgStyle = computed(() => ({
  background: props.variant === 'term' ? tok.value.termBg : backgroundCss(resolved.value.bg, props.resolver, 1.6),
  filter: resolved.value.bg.blur > 0 && props.variant !== 'term' ? `blur(${Math.max(1, resolved.value.bg.blur / 3)}px)` : undefined,
}))
const heroStyle = computed(() => ({ background: look.value.background, color: look.value.fg }))
</script>

<style scoped lang="scss">
.tsc {
  position: relative;
  display: flex;
  overflow: hidden;
  border-radius: 14px;
  border: 1px solid var(--m-line);
  isolation: isolate;
  font-size: 11px;
}
.tsc__bg {
  position: absolute;
  inset: -12px;
  z-index: -2;
}
.tsc__dim {
  position: absolute;
  inset: 0;
  z-index: -1;
}
.tsc-glass {
  background: var(--m-s);
}
.tsc.is-glass .tsc-glass {
  background: color-mix(in srgb, var(--m-s) var(--tsc-card-a), transparent);
  -webkit-backdrop-filter: blur(8px);
  backdrop-filter: blur(8px);
}

/* ---- mini ---- */
.tsc--mini {
  gap: 6px;
  height: 84px;
  padding: 8px;
}
.tsc--mini.is-lg {
  height: 150px;
  gap: 10px;
  padding: 14px;
  border-radius: 18px;
  .tsc-mini__sb,
  .tsc-mini__hero,
  .tsc-mini__row i {
    border-radius: 12px;
  }
  .tsc-mini__main,
  .tsc-mini__row {
    gap: 8px;
  }
  .tsc-mini__hero i {
    right: -16px;
    bottom: -26px;
    width: 70px;
    height: 70px;
  }
  .tsc-mini__row i.is-acc {
    display: none;
  }
}
.tsc-mini__sb {
  width: 24%;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 5px;
  border-radius: 8px;
  i {
    height: 5px;
    border-radius: 3px;
    background: var(--m-line);
  }
  i.is-on {
    height: 8px;
    background: var(--m-pSoft);
  }
}
.tsc-mini__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.tsc-mini__hero {
  position: relative;
  flex: 1.3;
  overflow: hidden;
  border-radius: 9px;
  i {
    position: absolute;
    right: -12px;
    bottom: -18px;
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: var(--m-a);
  }
}
.tsc-mini__row {
  flex: 1;
  display: flex;
  gap: 5px;
  i {
    flex: 1;
    border-radius: 7px;
  }
  i.is-acc {
    flex: none;
    width: 14px;
    background: var(--m-a2);
  }
}

/* ---- dash ---- */
.tsc-dash {
  position: absolute;
  inset: 52px 14px 14px;
  display: flex;
  gap: 10px;
}
.tsc-dash__sb {
  width: 24%;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, var(--m-line) 70%, transparent);
  b {
    font-size: 12px;
  }
  i {
    height: 8px;
    border-radius: 4px;
    background: var(--m-line);
  }
  i.is-on {
    height: 18px;
    border-radius: 6px;
    background: var(--m-pSoft);
  }
}
.tsc-dash__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.tsc-dash__hero {
  position: relative;
  flex: 1.3;
  overflow: hidden;
  padding: 12px;
  border-radius: 12px;
}
.tsc-dash__shade {
  position: absolute;
  inset: 0;
}
.tsc-dash__blob {
  position: absolute;
  right: -18px;
  bottom: -28px;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: var(--m-a);
}
.tsc-dash__copy {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 6px;
  b {
    font-size: 13px;
  }
  span {
    display: flex;
    gap: 5px;
  }
  i {
    height: 20px;
    width: 60px;
    border-radius: 999px;
  }
  i + i {
    width: 48px;
  }
}
.tsc-dash__cards {
  flex: 1;
  display: flex;
  gap: 8px;
  > span {
    flex: 1;
    min-width: 0;
    padding: 8px;
    border-radius: 12px;
    border: 1px solid color-mix(in srgb, var(--m-line) 70%, transparent);
    color: var(--m-ink2);
    overflow: hidden;
    white-space: nowrap;
  }
  b {
    display: block;
    font-size: 15px;
    color: var(--m-ink);
  }
  b.is-ok {
    color: var(--m-ok);
  }
}

/* ---- login ---- */
.tsc-login {
  position: absolute;
  left: 50%;
  top: 56px;
  width: 230px;
  max-width: calc(100% - 28px);
  translate: -50% 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, var(--m-line) 70%, transparent);
}
.tsc-login__brand {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 800;
  i {
    width: 22px;
    height: 22px;
    border-radius: 7px;
    background: var(--m-p);
    rotate: -6deg;
  }
}
.tsc-login__inp {
  height: 24px;
  border-radius: 999px;
  background: var(--m-s2);
}
.tsc-login__btn {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 26px;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-weight: 800;
}

/* ---- term ---- */
.tsc-term {
  position: absolute;
  inset: 52px 14px 14px;
  display: flex;
  flex-direction: column;
  padding: 10px 12px;
  border-radius: 12px;
  background: var(--m-termBg);
  color: var(--m-termFg);
  font-size: 10.5px;
  line-height: 1.6;
  overflow: hidden;
  span {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  i {
    font-style: normal;
  }
}

/* ---- phone ---- */
.tsc-phone {
  position: absolute;
  left: 50%;
  top: 48px;
  width: 120px;
  height: 200px;
  translate: -50% 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 8px;
  border-radius: 20px;
  border: 3px solid color-mix(in srgb, var(--m-ink) 22%, transparent);
  b {
    font-size: 11px;
  }
}
.tsc-phone__hero {
  position: relative;
  height: 50px;
  flex: none;
  overflow: hidden;
  border-radius: 10px;
  i {
    position: absolute;
    right: -10px;
    bottom: -16px;
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: var(--m-a);
  }
}
.tsc-phone__row {
  display: flex;
  gap: 5px;
  i {
    flex: 1;
    height: 34px;
    border-radius: 8px;
  }
}
.tsc-phone__card {
  height: 40px;
  border-radius: 8px;
}
</style>
