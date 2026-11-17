<!-- 启动器设置：PC 启动器的公告与各游戏的名称、字标、游戏页背景（存在 Momoko 系统配置里，通行证 GET /api/v1/launcher
     公开下发）。背景从游戏素材里选（登录插画、偶像卡横图、偶像卡原画；搜索 + 最多显示 48 张，图片懒加载），最多 3 张，
     按选择顺序显示。服务器图标跟随服务器封面，不在这里设置。每款支持客户端启动的游戏一张卡片，打开开关才保存它的设置。 -->
<template>
  <FormDialog :model-value="modelValue" :title="t('passportAdmin.launcher.title')" :width="860" :show-footer="false" @update:model-value="emit('update:modelValue', $event)">
    <div v-if="loading && !info" class="lset__hint lset__hint--center">{{ t('passportAdmin.common.loading') }}</div>
    <div v-else class="lset">
      <p class="lset__hint">{{ t('passportAdmin.launcher.hint') }}</p>
      <p v-if="info && !info.publicUrl" class="lset__warn">{{ t('passportAdmin.launcher.publicUrlMissing') }}</p>

      <label class="lset__field">
        <span class="lset__label">{{ t('passportAdmin.launcher.notice') }}</span>
        <UTextarea v-model="form.notice" :rows="3" :maxlength="500" :placeholder="t('passportAdmin.launcher.noticePlaceholder')" autoresize />
      </label>

      <section class="lset__section">
        <h3 class="lset__title">{{ t('passportAdmin.launcher.games') }}</h3>
        <p v-if="!form.games.length" class="lset__hint">{{ t('passportAdmin.launcher.gamesEmpty') }}</p>
        <article v-for="g in form.games" :key="g.game" class="lset__game">
          <div class="lset__game-head">
            <span class="lset__game-id m-mono">{{ g.game }}</span>
            <span class="lset__grow" />
            <span class="lset__label">{{ t('passportAdmin.launcher.enabled') }}</span>
            <AppSwitch v-model="g.enabled" />
          </div>
          <p v-if="!g.enabled" class="lset__hint">{{ t('passportAdmin.launcher.disabledHint') }}</p>
          <template v-else>
            <div class="lset__grid">
              <label class="lset__field">
                <span class="lset__label">{{ t('passportAdmin.launcher.name') }}</span>
                <UInput v-model="g.name" maxlength="40" />
              </label>
              <label class="lset__field">
                <span class="lset__label">{{ t('passportAdmin.launcher.wordmark') }}</span>
                <UInput v-model="g.wordmark" maxlength="40" :placeholder="t('passportAdmin.launcher.wordmarkPlaceholder')" />
              </label>
            </div>
            <p class="lset__hint">{{ t('passportAdmin.launcher.iconHint') }}</p>

            <div class="lset__field">
              <span class="lset__label">{{ t('passportAdmin.launcher.backgrounds') }}
                <small class="lset__hint">{{ t('passportAdmin.launcher.backgroundsHint') }}</small></span>
              <div class="lset__picked">
                <span v-if="!g.backgrounds.length" class="lset__hint">{{ t('passportAdmin.launcher.pickedEmpty') }}</span>
                <figure v-for="(key, i) in g.backgrounds" :key="key" class="lset__pick" :title="key">
                  <GameImage :src="gameAsset(g.game, key)" :alt="key" />
                  <span class="lset__order">{{ i + 1 }}</span>
                  <button type="button" class="lset__remove" :aria-label="t('passportAdmin.launcher.remove')" @click="toggle(g, key)">×</button>
                </figure>
              </div>
            </div>

            <div class="lset__picker">
              <div class="lset__picker-bar">
                <div class="lset__seg" role="radiogroup" :aria-label="t('passportAdmin.launcher.source')">
                  <button
                    v-for="s in SOURCES"
                    :key="s.key"
                    type="button"
                    role="radio"
                    class="lset__seg-btn"
                    :class="{ 'is-on': picker.source === s.key }"
                    :aria-checked="picker.source === s.key"
                    @click="picker.source = s.key"
                  >
                    {{ t(`passportAdmin.launcher.sources.${s.key}`) }}
                  </button>
                </div>
                <UInput v-model="picker.query" class="lset__grow" icon="i-lucide-search" :placeholder="t('passportAdmin.launcher.search')" />
              </div>
              <div class="lset__assets" :class="`is-${picker.source}`">
                <button
                  v-for="a in shown(g.game)"
                  :key="a.key"
                  type="button"
                  class="lset__asset"
                  :class="{ 'is-on': g.backgrounds.includes(a.key) }"
                  :title="a.key"
                  :aria-pressed="g.backgrounds.includes(a.key)"
                  :disabled="!g.backgrounds.includes(a.key) && g.backgrounds.length >= MAX_BACKGROUNDS"
                  @click="toggle(g, a.key)"
                >
                  <GameImage :src="a.url" :alt="a.key" />
                  <span v-if="g.backgrounds.includes(a.key)" class="lset__order">{{ g.backgrounds.indexOf(a.key) + 1 }}</span>
                </button>
              </div>
              <span v-if="more(g.game) > 0" class="lset__hint">{{ t('passportAdmin.launcher.more', { n: more(g.game) }) }}</span>
            </div>
          </template>
        </article>
      </section>

      <div class="lset__actions">
        <UButton :loading="saving" @click="save">{{ t('passportAdmin.launcher.save') }}</UButton>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getLauncherSettings, updateLauncherSettings } from '@/api/passportAdmin'
import { gameAsset, listGameAssets, type GameAssetEntry } from '@/games'
import type { GetLauncherSettingsResponse } from '@/types/v1/passport_admin'
import { feedback } from '@/utils/feedback'
import { notifyError } from '../shared'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()

const MAX_BACKGROUNDS = 3
const LIMIT = 48

// 背景的素材来源：登录插画（横图，适合铺满）、偶像卡横图、偶像卡原画（竖图，适合并排）。
type SourceKey = 'login' | 'land' | 'full'
const SOURCES: { key: SourceKey; group: string; match: (key: string) => boolean }[] = [
  { key: 'login', group: 'ui', match: (key) => key.startsWith('ui/login-') },
  { key: 'land', group: 'cards/idol', match: (key) => key.endsWith('-land') },
  { key: 'full', group: 'cards/idol', match: (key) => key.endsWith('-full') },
]

interface GameForm {
  game: string
  enabled: boolean
  name: string
  wordmark: string
  backgrounds: string[]
}

const info = ref<GetLauncherSettingsResponse>()
const loading = ref(false)
const saving = ref(false)
const form = reactive<{ notice: string; games: GameForm[] }>({ notice: '', games: [] })
const picker = reactive<{ source: SourceKey; query: string }>({ source: 'login', query: '' })
// 已加载的素材列表：<游戏>:<来源> → 素材
const assets = reactive(new Map<string, GameAssetEntry[]>())

const loadAssets = async (game: string, source: SourceKey) => {
  const id = `${game}:${source}`
  if (assets.has(id)) return
  const s = SOURCES.find((item) => item.key === source)
  if (!s) return
  assets.set(id, (await listGameAssets(game, s.group)).filter((a) => s.match(a.key)))
}

const matched = (game: string) => {
  const list = assets.get(`${game}:${picker.source}`) ?? []
  const q = picker.query.trim().toLowerCase()
  return q ? list.filter((a) => a.key.includes(q)) : list
}
const shown = (game: string) => matched(game).slice(0, LIMIT)
const more = (game: string) => Math.max(0, matched(game).length - LIMIT)

const toggle = (g: GameForm, key: string) => {
  const i = g.backgrounds.indexOf(key)
  if (i >= 0) g.backgrounds.splice(i, 1)
  else if (g.backgrounds.length < MAX_BACKGROUNDS) g.backgrounds.push(key)
}

watch(
  () => [picker.source, form.games.map((g) => g.game).join(',')] as const,
  ([source]) => {
    for (const g of form.games) void loadAssets(g.game, source)
  },
)

// 每款可以设置的游戏一张卡片；保存过的游戏打开开关并填入设置。
const fill = (res: GetLauncherSettingsResponse) => {
  info.value = res
  const saved = res.settings?.games ?? []
  form.notice = res.settings?.notice ?? ''
  form.games = res.games.map((game) => {
    const s = saved.find((item) => item.game === game)
    return {
      game,
      enabled: !!s,
      name: s?.name ?? '',
      wordmark: s?.wordmark ?? '',
      backgrounds: [...(s?.backgrounds ?? [])],
    }
  })
}

const load = async () => {
  loading.value = true
  try {
    fill((await getLauncherSettings()).data)
  } catch (e) {
    notifyError(e)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) void load()
  },
  { immediate: true },
)

const save = async () => {
  saving.value = true
  try {
    const res = await updateLauncherSettings({
      notice: form.notice.trim(),
      games: form.games
        .filter((g) => g.enabled)
        .map((g) => ({ game: g.game, name: g.name.trim(), wordmark: g.wordmark.trim(), backgrounds: [...g.backgrounds] })),
    })
    fill(res.data)
    feedback.success(t('passportAdmin.launcher.saved'))
  } catch (e) {
    notifyError(e)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.lset {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.lset__section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-top: 14px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.lset__title {
  margin: 0;
  font-size: 16px;
  font-weight: 800;
}
.lset__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.lset__label {
  font-size: 13px;
  font-weight: 700;
}
.lset__hint {
  margin: 0;
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.lset__hint--center {
  padding: 24px 0;
  text-align: center;
}
.lset__warn {
  margin: 0;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-warnSoft);
  color: var(--m-ink);
  font-size: 13px;
}
.lset__game {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.lset__game-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.lset__game-id {
  font-size: 14px;
  font-weight: 800;
}
.lset__grow {
  flex: 1;
  min-width: 0;
}
.lset__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.lset__picked {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-height: 72px;
}
.lset__pick {
  position: relative;
  width: 128px;
  height: 72px;
  margin: 0;
  border-radius: var(--app-radius-sm);
  overflow: hidden;
  background: var(--el-fill-color-darker);
}
.lset__pick :deep(img),
.lset__asset :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.lset__order {
  position: absolute;
  left: 6px;
  top: 6px;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 10px;
  background: var(--m-p);
  color: var(--m-onP, #fff);
  font-size: 12px;
  font-weight: 800;
  line-height: 20px;
  text-align: center;
}
.lset__remove {
  position: absolute;
  right: 4px;
  top: 4px;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 0;
  border-radius: 11px;
  background: rgb(0 0 0 / 55%);
  color: #fff;
  font-size: 15px;
  line-height: 22px;
  cursor: pointer;
}
.lset__picker {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.lset__picker-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.lset__seg {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.lset__seg-btn {
  min-height: 34px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}
.lset__seg-btn.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.lset__assets {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(128px, 1fr));
  gap: 8px;
  max-height: 300px;
  overflow-y: auto;
}
.lset__assets.is-full {
  grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
}
.lset__asset {
  position: relative;
  height: 72px;
  padding: 0;
  border-radius: var(--app-radius-sm);
  border: 2px solid transparent;
  background: var(--el-fill-color-darker);
  overflow: hidden;
  cursor: pointer;
}
.lset__assets.is-full .lset__asset {
  height: 128px;
}
.lset__asset.is-on {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.lset__asset:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.lset__actions {
  display: flex;
  justify-content: flex-end;
}
@media (width < 640px) {
  .lset__grid {
    grid-template-columns: 1fr;
  }
}
</style>
