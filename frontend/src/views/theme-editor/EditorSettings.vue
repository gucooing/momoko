<!-- 编辑器左栏（手机上「背景 · 字体 · 组件」面板也用它）：改动作用于 全局 / 仅当前页 + 配色 / 背景 / 字体形状 / 组件 四个标签。
     配色列出所有 token（色块、名称、用途、色值，改过的有圆点，本页覆盖的有「本页」标签）；终端页多出终端专用 token 与 ANSI 16 色。 -->
<template>
  <div class="eds">
    <div v-if="showScope" class="eds-scope" :class="{ 'is-page': ed.scope.value === 'page' }">
      <span class="eds-lbl">{{ t('theme.editor.scope') }}</span>
      <div class="eds-seg eds-seg--fill is-surface">
        <button type="button" :class="{ on: ed.scope.value === 'all' }" @click="ed.scopeChoice.value = 'all'">{{ t('theme.editor.scopeAll') }}</button>
        <button type="button" :class="{ on: ed.scope.value === 'page' }" @click="ed.scopeChoice.value = 'page'">
          {{ t('theme.editor.scopePage', { page: ed.pageName.value }) }}
        </button>
      </div>
      <small>{{ ed.scope.value === 'page' ? t('theme.editor.scopeHintPage', { page: ed.pageName.value }) : t('theme.editor.scopeHintAll') }}</small>
    </div>

    <div class="eds-seg eds-seg--fill" role="tablist">
      <button
        v-for="k in tabs"
        :key="k"
        type="button"
        role="tab"
        :class="{ on: activeTab === k }"
        :aria-selected="activeTab === k"
        @click="activeTab = k"
      >
        {{ t(`theme.editor.tabs.${k}`) }}
      </button>
    </div>

    <!-- 配色 -->
    <div v-if="activeTab === 'color'" class="eds-toks">
      <button
        v-for="k in ed.editorTokens"
        :key="k"
        type="button"
        class="eds-tok"
        :class="{ on: ed.sel.value === k }"
        :aria-pressed="ed.sel.value === k"
        @click="ed.select(k)"
      >
        <span class="eds-sw" :style="{ background: ed.tokenValue(k) }" />
        <span class="eds-tok__text">
          <b>
            {{ t(`theme.tokens.${k}.name`) }}
            <span v-if="ed.isPageToken(k)" class="eds-tagx">{{ t('theme.editor.pageTag') }}</span>
            <span v-if="ed.isEdited(k)" class="eds-dot" :title="t('theme.editor.edited')" />
          </b>
          <small>{{ t(`theme.tokens.${k}.use`) }}</small>
        </span>
        <span class="m-mono eds-hex">{{ ed.tokenValue(k) }}</span>
      </button>

      <template v-if="ed.isTermPage.value">
        <div class="eds-lbl eds-lbl--sub">{{ t('theme.editor.termTitle') }}</div>
        <button
          v-for="k in TERM_KEYS"
          :key="k"
          type="button"
          class="eds-tok"
          :class="{ on: ed.sel.value === k }"
          @click="ed.select(k)"
        >
          <span class="eds-sw" :style="{ background: ed.tokenValue(k) }" />
          <span class="eds-tok__text">
            <b>
              {{ t(`theme.tokens.${k}.name`) }}
              <span v-if="ed.isPageToken(k)" class="eds-tagx">{{ t('theme.editor.pageTag') }}</span>
            </b>
            <small>{{ t(`theme.tokens.${k}.use`) }}</small>
          </span>
          <span class="m-mono eds-hex">{{ ed.tokenValue(k) }}</span>
        </button>
        <div class="eds-ansi" :aria-label="t('theme.editor.ansi')">
          <button
            v-for="i in 16"
            :key="i"
            type="button"
            :class="{ on: ed.sel.value === `ansi${i - 1}` }"
            :title="`ANSI ${i - 1} · ${ed.tokenValue(`ansi${i - 1}`)}`"
            :aria-label="`ANSI ${i - 1}`"
            :style="{ background: ed.tokenValue(`ansi${i - 1}`) }"
            @click="ed.select(`ansi${i - 1}`)"
          />
        </div>
      </template>

      <div class="eds-two">
        <button type="button" class="eds-btn" @click="ed.pickFromImage()"><BeakerIcon aria-hidden="true" />{{ t('theme.editor.pickFromImage') }}</button>
        <button type="button" class="eds-btn" @click="ed.genDark()"><MoonIcon aria-hidden="true" />{{ t('theme.editor.genDark') }}</button>
      </div>
    </div>

    <!-- 背景 -->
    <BackgroundEditor v-else-if="activeTab === 'bg'" v-model="ed.bgModel.value" variant="editor" :scheme="ed.scheme.value" />

    <!-- 字体形状 -->
    <div v-else-if="activeTab === 'shape'" class="eds-col">
      <label class="eds-field">
        <span class="eds-lbl">{{ t('theme.editor.font') }}</span>
        <select class="app-select eds-select" :value="shape.font" @change="onFont(($event.target as HTMLSelectElement).value)">
          <option v-for="f in FONTS" :key="f" :value="f">{{ t(`theme.editor.fonts.${f}`) }}</option>
          <option v-if="shape.fontAsset || customFont" value="custom">{{ t('theme.editor.fontCustom', { name: shape.fontName || customFont?.name || 'woff2' }) }}</option>
        </select>
        <label class="eds-upfont">
          <input type="file" accept=".woff2,font/woff2" class="eds-file" @change="onFontFile" />
          <ArrowUpTrayIcon aria-hidden="true" />{{ t('theme.editor.fontUpload') }}
        </label>
        <small>{{ t('theme.editor.fontUploadHint') }}</small>
      </label>
      <div class="eds-field">
        <span class="eds-lbl">{{ t('theme.editor.fontSize') }}</span>
        <div class="eds-seg eds-seg--fill">
          <button v-for="k in FONT_SIZES" :key="k" type="button" :class="{ on: shape.fontSize === k }" @click="ed.setShape({ fontSize: k })">
            {{ t(`theme.editor.fontSizes.${k}`) }}
          </button>
        </div>
      </div>
      <label class="eds-range">
        <span><span>{{ t('theme.editor.radius') }}</span><b class="m-num">{{ radius }} px</b></span>
        <input type="range" min="0" max="32" :value="radius" :aria-label="t('theme.editor.radius')" @input="ed.setShape({ radius: Number(($event.target as HTMLInputElement).value) }, 'radius')" />
      </label>
      <div class="eds-field">
        <span class="eds-lbl">{{ t('theme.editor.shadow') }}</span>
        <div class="eds-seg eds-seg--fill">
          <button v-for="k in SHADOWS" :key="k" type="button" :class="{ on: shape.shadow === k }" @click="ed.setShape({ shadow: k })">
            {{ t(`theme.editor.shadows.${k}`) }}
          </button>
        </div>
      </div>
      <div class="eds-field">
        <span class="eds-lbl">{{ t('theme.editor.button') }}</span>
        <div class="eds-seg eds-seg--fill">
          <button v-for="k in BUTTON_SHAPES" :key="k" type="button" :class="{ on: shape.button === k }" @click="ed.setShape({ button: k })">
            {{ t(`theme.editor.buttons.${k}`) }}
          </button>
        </div>
      </div>
    </div>

    <!-- 组件 -->
    <div v-else class="eds-col">
      <button type="button" class="eds-tok eds-hero" :class="{ on: ed.sel.value === 'hero' }" @click="ed.select('hero')">
        <span class="eds-hero__sw" :style="{ background: heroCss }" />
        <span class="eds-tok__text">
          <b>{{ t('theme.tokens.hero.name') }}</b>
          <small>{{ t('theme.editor.bannerSub', { type: heroType }) }}</small>
        </span>
        <ChevronRightIcon aria-hidden="true" class="eds-chev" />
      </button>
      <div class="eds-field">
        <span class="eds-lbl">{{ t('theme.editor.sidebar') }}</span>
        <div class="eds-seg eds-seg--fill">
          <button v-for="k in SIDEBAR_STYLES" :key="k" type="button" :class="{ on: shape.sidebar === k }" @click="ed.setShape({ sidebar: k })">
            {{ t(`theme.editor.sidebars.${k}`) }}
          </button>
        </div>
      </div>
      <div class="eds-field">
        <span class="eds-lbl">{{ t('theme.editor.pill') }}</span>
        <div class="eds-seg eds-seg--fill">
          <button v-for="k in PILL_STYLES" :key="k" type="button" :class="{ on: shape.pill === k }" @click="ed.setShape({ pill: k })">
            {{ t(`theme.editor.pills.${k}`) }}
          </button>
        </div>
      </div>
      <div class="eds-field">
        <span class="eds-lbl">{{ t('theme.editor.chartColors') }}</span>
        <div class="eds-chart">
          <i v-for="c in chartColors" :key="c" :style="{ background: c }" :title="c" />
        </div>
        <small>{{ t('theme.editor.chartColorsHint') }}</small>
      </div>
      <label class="eds-switch">
        <AppSwitch :model-value="shape.cardBorder !== false" :aria-label="t('theme.editor.cardBorder')" @update:model-value="ed.setShape({ cardBorder: $event })" />
        {{ t('theme.editor.cardBorder') }}
      </label>
      <label class="eds-switch">
        <AppSwitch :model-value="shape.stickerTilt !== false" :aria-label="t('theme.editor.stickerTilt')" @update:model-value="ed.setShape({ stickerTilt: $event })" />
        {{ t('theme.editor.stickerTilt') }}
      </label>

      <!-- 组件尺寸：清单来自 theme/sizeTokens.ts；在这个标签下点预览里的元素也会选中它所属的组件 -->
      <div class="eds-field">
        <span class="eds-lbl">{{ t('theme.editor.sizesTitle') }}</span>
        <small>{{ t('theme.editor.sizesHint') }}</small>
      </div>
      <div class="eds-sizes">
        <button
          v-for="g in SIZE_GROUPS"
          :key="g.key"
          type="button"
          class="eds-tok eds-size"
          :class="{ on: ed.sel.value === `size:${g.key}` }"
          :aria-pressed="ed.sel.value === `size:${g.key}`"
          @click="ed.select(`size:${g.key}`)"
        >
          <span class="eds-tok__text">
            <b>
              {{ ed.sizeGroupLabel(g.key) }}
              <span v-if="g.tokens.some((tk) => ed.isPageSize(tk.key))" class="eds-tagx">{{ t('theme.editor.pageTag') }}</span>
            </b>
            <small>{{ g.tokens.map((tk) => ed.sizeLabel(tk.key)).join(t('theme.listSep')) }}</small>
          </span>
          <span v-if="ed.sizeGroupEdits(g.key)" class="eds-count m-num">{{ ed.sizeGroupEdits(g.key) }}</span>
          <ChevronRightIcon aria-hidden="true" class="eds-chev" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowUpTrayIcon, BeakerIcon, ChevronRightIcon, MoonIcon } from '@heroicons/vue/24/outline'
import type { ThemeAsset } from '@/types/v1/theme'
import BackgroundEditor from '@/components/theme/BackgroundEditor.vue'
import {
  BANNER_GRADIENTS,
  BUTTON_SHAPES,
  DEFAULT_RADIUS,
  FONTS,
  FONT_SIZES,
  PILL_STYLES,
  SHADOWS,
  SIDEBAR_STYLES,
  assetUrl,
} from '@/theme/content'
import { EDITOR_TERM_TOKENS } from '@/theme/palettes'
import { SIZE_GROUPS } from '@/theme/sizeTokens'
import { useThemeAssets } from '@/theme/useThemeAssets'
import { useThemeEditor, type EditorTab } from './useThemeEditor'

const props = withDefaults(defineProps<{ tabs?: EditorTab[]; showScope?: boolean }>(), {
  tabs: () => ['color', 'bg', 'shape', 'comp'],
  showScope: true,
})

const ed = useThemeEditor()
const { t } = ed
const { upload, findAsset } = useThemeAssets()
const TERM_KEYS = EDITOR_TERM_TOKENS

// 只显示部分标签时（手机面板），当前标签不在其中就用第一个
const activeTab = computed<EditorTab>({
  get: () => (props.tabs.includes(ed.tab.value) ? ed.tab.value : props.tabs[0]!),
  set: (v) => (ed.tab.value = v),
})

const shape = computed(() => ed.content.value.shape!)
const radius = computed(() => shape.value.radius ?? DEFAULT_RADIUS)
const chartColors = computed(() => ['p', 'a2', 'a', 'ok', 'warn'].map((k) => ed.tokenValue(k)))

const banner = computed(() => ed.banner.value)
const heroCss = computed(() =>
  banner.value.type === 'image' && banner.value.image
    ? `url("${assetUrl(banner.value.image)}") center / cover`
    : banner.value.type === 'gradient'
      ? BANNER_GRADIENTS[banner.value.gradient] || BANNER_GRADIENTS[0]!
      : 'var(--m-p)',
)
const heroType = computed(() => {
  const b = banner.value
  const label = t(`theme.editor.bannerTypes.${b.type}`)
  const a = b.type === 'image' ? findAsset(b.image) : null
  return a?.name ? `${label} · ${a.name}` : label
})

const customFont = ref<ThemeAsset | null>(null)
const onFont = (v: string) => {
  if (v === 'custom') {
    const a = customFont.value
    ed.setShape({ font: 'custom', fontAsset: a?.id || shape.value.fontAsset, fontName: a ? a.name.replace(/\.woff2$/i, '') : shape.value.fontName })
  } else {
    ed.setShape({ font: v })
  }
}
const onFontFile = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const f = input.files?.[0]
  input.value = ''
  if (!f) return
  const a = await upload(f, 'font')
  if (!a) return
  customFont.value = a
  ed.setShape({ font: 'custom', fontAsset: a.id, fontName: f.name.replace(/\.woff2$/i, '').slice(0, 40) })
}
</script>

<style scoped lang="scss">
.eds {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.eds-lbl {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--m-ink2);
}
.eds-lbl--sub {
  margin: 12px 0 6px;
}
.eds-scope {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  border-radius: 18px;
  background: var(--m-s2);
  small {
    font-size: 0.75rem;
    line-height: 1.5;
    color: var(--m-ink2);
  }
  &.is-page {
    background: var(--m-aSoft);
  }
}
.eds-seg {
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s2);
  button {
    min-height: 36px;
    padding: 0 12px;
    border: 0;
    border-radius: 999px;
    background: transparent;
    font: inherit;
    font-size: 0.8125rem;
    font-weight: 700;
    color: var(--m-ink2);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    cursor: pointer;
  }
  .on {
    background: var(--m-s);
    color: var(--m-ink);
    box-shadow: 0 1px 3px rgba(20, 10, 30, 0.12);
  }
  &.is-surface {
    background: var(--m-s);
    .on {
      background: var(--m-s2);
    }
  }
}
.eds-seg--fill {
  display: flex;
  button {
    flex: 1;
    min-width: 0;
    padding: 0 6px;
  }
}
.eds-toks {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.eds-tok {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 48px;
  padding: 6px 8px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: var(--m-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
  &:hover {
    background: var(--m-s2);
  }
  &.on,
  &.on:hover {
    background: var(--m-pSoft);
  }
}
.eds-sw {
  flex: none;
  width: 30px;
  height: 30px;
  border-radius: 10px;
  border: 1px solid rgba(0, 0, 0, 0.1);
}
.eds-tok__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  b {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 0.8125rem;
  }
  small {
    font-size: 0.6875rem;
    color: var(--m-ink2);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}
.eds-hex {
  font-size: 0.6875rem;
  color: var(--m-ink2);
}
.eds-tagx {
  display: inline-flex;
  align-items: center;
  height: 18px;
  padding: 0 7px;
  border-radius: 999px;
  background: var(--m-aSoft);
  color: var(--m-aOn);
  font-size: 0.625rem;
  font-weight: 800;
  white-space: nowrap;
}
.eds-dot {
  flex: none;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--m-p);
}
.eds-ansi {
  display: grid;
  grid-template-columns: repeat(8, minmax(0, 1fr));
  gap: 4px;
  padding: 6px 8px 0;
  button {
    height: 22px;
    padding: 0;
    border: 0;
    border-radius: 6px;
    cursor: pointer;
    &.on {
      box-shadow: 0 0 0 2px var(--m-s), 0 0 0 4px var(--m-p);
    }
  }
}
.eds-two {
  display: flex;
  gap: 8px;
  margin-top: 10px;
}
.eds-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 10px;
  border: 0;
  border-radius: var(--app-radius-btn);
  background: var(--m-s2);
  color: var(--m-ink);
  font: inherit;
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  svg {
    width: 16px;
    height: 16px;
  }
  &:hover {
    background: var(--m-pSoft);
    color: var(--m-pInk);
  }
}
.eds-col {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.eds-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  small {
    font-size: 0.6875rem;
    color: var(--m-ink2);
  }
}
.eds-select {
  height: 44px;
  border-radius: 14px;
}
.eds-upfont {
  position: relative;
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  gap: 6px;
  min-height: 32px;
  padding: 0 10px;
  border-radius: 999px;
  color: var(--m-pInk);
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
  svg {
    width: 14px;
    height: 14px;
  }
  &:hover {
    background: var(--m-pSoft);
  }
  &:focus-within {
    box-shadow: var(--app-focus-ring);
  }
}
.eds-file {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}
.eds-range {
  display: flex;
  flex-direction: column;
  gap: 4px;
  > span {
    display: flex;
    font-size: 0.75rem;
    font-weight: 700;
    span {
      flex: 1;
    }
    b {
      color: var(--m-pInk);
    }
  }
  input {
    width: 100%;
    height: 24px;
    margin: 0;
    accent-color: var(--m-p);
    cursor: pointer;
  }
}
.eds-hero {
  padding: 8px;
}
.eds-hero__sw {
  flex: none;
  width: 56px;
  height: 36px;
  border-radius: 10px;
}
.eds-chev {
  width: 16px;
  height: 16px;
  color: var(--m-ink2);
}
.eds-chart {
  display: flex;
  gap: 6px;
  i {
    flex: 1;
    height: 28px;
    border-radius: 8px;
  }
}
.eds-sizes {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: -6px;
}
.eds-size {
  min-height: 52px;
}
.eds-count {
  flex: none;
  min-width: 22px;
  height: 22px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-size: 0.6875rem;
  font-weight: 800;
  line-height: 22px;
  text-align: center;
}
.eds-switch {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
</style>
