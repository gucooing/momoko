<!-- 接入游戏服务器：分步向导，一步一屏 —— 选择游戏（内置适配器）→ 接入方式（托管 / 仅 GM，按适配器支持的方式）
     → 目录与端口 / 地址与密钥 → 完成。点卡片即选中并进入下一步；顶部步骤条可以回到走过的步骤修改。
     第 3 步右侧是预检 / 连接测试的真实结果。 -->
<template>
  <div class="gsn" :data-game="game || undefined">
    <RouterLink to="/game/servers" class="gsn__back">
      <UIcon name="i-lucide-chevron-left" />{{ t('gameServers.new.back') }}
    </RouterLink>
    <div class="gsn__head">
      <h1>{{ t('gameServers.new.title') }}</h1>
      <ol class="gsn__steps" :aria-label="t('gameServers.new.title')">
        <li v-for="(s, i) in steps" :key="s.key">
          <button
            type="button"
            class="gsn__step"
            :class="{ 'is-done': stepDone(i), 'is-on': step === i }"
            :aria-current="step === i ? 'step' : undefined"
            :aria-label="s.label"
            :title="s.label"
            :disabled="!canJumpTo(i)"
            @click="step = i"
          >
            <span class="gsn__step-num">
              <UIcon v-if="stepDone(i) && (step !== i || !!created)" name="i-lucide-check" />
              <span v-else class="m-num">{{ i + 1 }}</span>
            </span>
            <span class="gsn__step-label">{{ s.label }}</span>
          </button>
        </li>
      </ol>
    </div>

    <Transition name="gsn-step" mode="out-in">
      <!-- 第 1 步：选择游戏 -->
      <section v-if="step === 0" key="game" class="gsn__pane">
        <header class="gsn__pane-head">
          <h2>{{ steps[0]!.label }}</h2>
          <p>{{ t('gameServers.new.stepDesc.game') }}</p>
        </header>
        <EmptyState v-if="adaptersLoaded && !available.length" icon="HOutline:PuzzlePieceIcon" :title="t('gameServers.new.noAdapter')" />
        <div class="gsn__games">
          <button
            v-for="a in available"
            :key="a.id"
            type="button"
            class="gsn__opt gsn__game"
            :class="{ 'is-on': game === a.id }"
            :aria-pressed="game === a.id"
            :data-game="a.id"
            @click="chooseGame(a.id)"
          >
            <span v-if="gameCover(a.id)" class="gsn__game-art">
              <GameImage :src="gameCover(a.id)" alt="" eager />
              <span v-if="game === a.id" class="gsn__game-check"><UIcon name="i-lucide-check" /></span>
            </span>
            <span v-else class="gsn__game-band"><span class="gsn__game-logo">{{ adapterName(a.id).charAt(0) }}</span></span>
            <span class="gsn__game-body">
              <b>{{ adapterName(a.id) }}</b>
              <small>{{ t(`gameServers.list.adapterDriver.${a.gmDriver}`, a.gmDriver) }}</small>
              <span class="gsn__caps">
                <span v-for="m in a.modes" :key="m" class="gsn__cap">{{ t(`gameServers.mode.${m}`) }}</span>
                <!-- 「托管」能力和上面的托管模式是一回事，不重复显示 -->
                <span v-for="c in gameCaps(a)" :key="c.key" class="gsn__cap">{{ t(`gameServers.caps.${c.key}`) }}</span>
              </span>
            </span>
          </button>
        </div>
        <div class="gsn__nav">
          <UButton color="neutral" variant="outline" to="/game/servers">{{ t('gameServers.new.cancel') }}</UButton>
          <UButton color="primary" trailing-icon="i-lucide-arrow-right" :disabled="!adapter" @click="step = 1">
            {{ t('gameServers.new.next') }}
          </UButton>
        </div>
      </section>

      <!-- 第 2 步：接入方式 -->
      <section v-else-if="step === 1" key="mode" class="gsn__pane">
        <header class="gsn__pane-head">
          <h2>{{ steps[1]!.label }} <span class="gsn__ctx">{{ adapterName(game) }}</span></h2>
          <p>{{ t('gameServers.new.stepDesc.mode') }}</p>
        </header>
        <div class="gsn__modes">
          <button
            v-for="m in modeCards"
            :key="m.key"
            type="button"
            class="gsn__opt gsn__mode"
            :class="{ 'is-on': mode === m.key }"
            :disabled="!m.supported"
            :aria-pressed="mode === m.key"
            @click="chooseMode(m.key)"
          >
            <span class="gsn__mode-ico m-sticker" :class="m.key === 'hosted' ? 'm-sticker--p' : 'm-sticker--a2'">
              <UIcon :name="m.key === 'hosted' ? 'i-lucide-cpu' : 'i-lucide-cloud'" />
            </span>
            <span class="gsn__mode-body">
              <span class="gsn__mode-title">
                <b>{{ m.title }}</b><span class="gsn__tag">{{ m.tag }}</span>
              </span>
              <small>{{ m.supported ? m.desc : t('gameServers.new.modeUnsupported') }}</small>
              <span class="gsn__feats">
                <span v-for="f in m.feats" :key="f.key" :class="{ 'is-off': !f.on }">
                  <UIcon :name="f.on ? 'i-lucide-check' : 'i-lucide-minus'" />{{ t(`gameServers.caps.${f.key}`) }}
                </span>
              </span>
            </span>
          </button>
        </div>
        <div class="gsn__nav">
          <UButton color="neutral" variant="outline" icon="i-lucide-arrow-left" @click="step = 0">{{ t('gameServers.new.prev') }}</UButton>
          <UButton color="primary" trailing-icon="i-lucide-arrow-right" :disabled="!modeOk" @click="step = 2">
            {{ t('gameServers.new.next') }}
          </UButton>
        </div>
      </section>

      <!-- 第 3 步：目录与端口 / 地址与密钥 -->
      <section v-else-if="step === 2" key="form" class="gsn__pane">
        <header class="gsn__pane-head">
          <h2>{{ steps[2]!.label }} <span class="gsn__ctx">{{ adapterName(game) }} · {{ modeTitle }}</span></h2>
          <p>{{ mode === 'hosted' ? t('gameServers.new.stepDesc.hosted') : t('gameServers.new.stepDesc.remote') }}</p>
        </header>
        <div class="gsn__form-row">
          <div class="gsn__card gsn__form">
            <div class="gsn__grid">
              <label class="gsn__field">
                <span>{{ t('gameServers.new.name') }}</span>
                <UInput v-model="name" size="lg" :placeholder="t('gameServers.new.namePh')" :maxlength="64" />
              </label>
              <div class="gsn__field">
                <span>{{ t('gameServers.new.tags') }} <small>{{ t('gameServers.new.tagsHint') }}</small></span>
                <TagInput v-model="tags" :placeholder="t('gameServers.new.tagsPh')" :label="t('gameServers.new.tags')" />
              </div>
            </div>

            <HostedForm
              v-if="mode === 'hosted'"
              v-model="hosted"
              :detected="detected"
              :detecting="checking"
              :listen-keys="listenKeys"
              :resource="hostedResource"
              @detect="runCheck"
            />

            <div v-else class="gsn__remote">
              <label class="gsn__field">
                <span>{{ t('gameServers.new.mgpEndpoint') }}</span>
                <UInput v-model="remote.endpoint" class="m-mono" icon="i-lucide-globe" :placeholder="t('gameServers.new.mgpEndpointPh')" />
                <small>{{ t('gameServers.new.mgpEndpointHint') }}</small>
              </label>
              <div class="gsn__field">
                <span>{{ t('gameServers.key.title') }}</span>
                <MgpKeyGenerator v-model="remoteKey" :hint="t('gameServers.key.remoteHint')" />
              </div>
            </div>

            <div class="gsn__actions">
              <UButton color="neutral" variant="outline" icon="i-lucide-arrow-left" @click="step = 1">{{ t('gameServers.new.prev') }}</UButton>
              <span class="gsn__spacer" />
              <UButton color="neutral" variant="soft" icon="i-lucide-zap" :loading="checking" @click="runCheck">
                {{ mode === 'hosted' ? t('gameServers.new.precheck') : t('gameServers.new.test') }}
              </UButton>
              <UButton color="primary" icon="i-lucide-check" :loading="submitting" :disabled="!canSubmit" @click="submit">
                {{ mode === 'hosted' && hosted.start ? t('gameServers.new.submitStart') : t('gameServers.new.submit') }}
              </UButton>
            </div>
          </div>

          <aside class="gsn__card gsn__check">
            <div class="gsn__check-head">
              <h2>{{ mode === 'hosted' ? t('gameServers.new.checkHosted') : t('gameServers.new.checkRemote') }}</h2>
              <StatusPill :variant="pill.variant" :label="pill.label" />
            </div>
            <p v-if="!checks.length" class="gsn__muted">
              {{ mode === 'hosted' ? t('gameServers.new.checkNoneHosted') : t('gameServers.new.checkNoneRemote') }}
            </p>
            <CheckList v-else :checks="checks" />
            <p v-if="manifestNote" class="gsn__note">{{ manifestNote }}</p>
          </aside>
        </div>
      </section>

      <!-- 第 4 步：完成 -->
      <section v-else key="done" class="gsn__card gsn__done">
        <span class="gsn__done-ico m-sticker m-sticker--ok"><UIcon name="i-lucide-check" /></span>
        <h2>{{ t('gameServers.new.doneTitle', { name: created?.brief?.name || '' }) }}</h2>
        <p class="gsn__muted">{{ t('gameServers.new.doneDesc') }}</p>
        <div class="gsn__actions gsn__actions--center">
          <UButton color="neutral" variant="outline" to="/game/servers">{{ t('gameServers.new.backToList') }}</UButton>
          <UButton color="neutral" variant="soft" @click="reset">{{ t('gameServers.new.createAnother') }}</UButton>
          <UButton color="primary" :to="`/game/servers/${encodeURIComponent(created?.brief?.id || '')}`">{{ t('gameServers.new.openServer') }}</UButton>
        </div>
      </section>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { gameCover } from '@/games'
import GameImage from '@/components/game/GameImage.vue'
import { createGameServer, detectGameServerDir, listGameAdapters, testGameServer } from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import type { DetectGameServerDirResponse, GameAdapter, GameServerCheck, GameServerInfo } from '@/types/v1/game_server'
import CheckList from '../components/CheckList.vue'
import HostedForm from '../components/HostedForm.vue'
import MgpKeyGenerator from '../components/MgpKeyGenerator.vue'
import TagInput from '../components/TagInput.vue'
import { applyDetection, detectListens, emptyHostedDraft, mainListenKeys, toHostedSpec } from '../hosted'
import type { PillVariant } from '../shared'
import type { MgpKey } from '../mgpKey'

defineOptions({ name: 'GameServerNewView' })

const { t, te } = useI18n()
const fb = useFeedback()

const adapters = ref<GameAdapter[]>([])
const adaptersLoaded = ref(false)
const available = computed(() => adapters.value.filter((a) => a.available))
// 当前步骤：0 选择游戏 · 1 接入方式 · 2 表单 · 3 完成
const step = ref(0)
const game = ref('')
const mode = ref<'' | 'hosted' | 'remote'>('')
const adapter = computed(() => available.value.find((a) => a.id === game.value))
const listenKeys = computed(() => mainListenKeys(adapter.value))
const hostedResource = computed(() => !!adapter.value?.capabilities.some((c) => c.key === 'resource' && c.enabled))
const modeOk = computed(() => !!mode.value && !!adapter.value?.modes.includes(mode.value))
const name = ref('')
const tags = ref<string[]>([])
const hosted = ref(emptyHostedDraft())
const remote = reactive({ endpoint: '' })
// 仅 GM：MGP 密钥可填写服务端配置里已有的，也可点「生成」新建一组（新建的要写进服务端配置）
const remoteKey = ref<MgpKey | null>(null)
const detected = ref<DetectGameServerDirResponse | null>(null)
const checks = ref<GameServerCheck[]>([])
const checking = ref(false)
const submitting = ref(false)
const created = ref<GameServerInfo | null>(null)

const adapterName = (id: string) => (te(`game.adapters.${id}`) ? t(`game.adapters.${id}`) : t('game.unnamed'))
const gameCaps = (a: GameAdapter) => a.capabilities.filter((c) => c.key !== 'hosting').slice(0, 4)

// 只有一款游戏时直接选中，从「接入方式」开始
const firstStep = () => (available.value.length === 1 ? 1 : 0)

onMounted(async () => {
  try {
    const { data } = await listGameAdapters()
    adapters.value = data.items ?? []
    if (available.value.length === 1) {
      pickGame(available.value[0]!.id)
      step.value = firstStep()
    }
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    adaptersLoaded.value = true
  }
})

const pickGame = (id: string) => {
  game.value = id
  const a = adapter.value
  if (a && !a.modes.includes(mode.value)) mode.value = (a.modes[0] as 'hosted' | 'remote') || ''
}
const chooseGame = (id: string) => {
  pickGame(id)
  step.value = 1
}
const chooseMode = (m: 'hosted' | 'remote') => {
  if (!adapter.value?.modes.includes(m)) return
  mode.value = m
  step.value = 2
}
watch([game, mode], () => {
  checks.value = []
  detected.value = null
})
// 监听的键随游戏不同：换游戏时清空已填的监听
watch(game, () => {
  hosted.value.listens = {}
})
// 换步骤时回到页面顶部
watch(step, () => document.querySelector('.app-content')?.scrollTo({ top: 0, behavior: 'smooth' }))

const HOSTED_FEATS = ['console', 'files', 'settings', 'update', 'backup', 'logs']
const modeCards = computed(() => {
  const a = adapter.value
  const caps = new Set((a?.capabilities || []).map((c) => c.key))
  const mgp = a?.gmDriver === 'mgp'
  return [
    {
      key: 'hosted' as const,
      title: t('gameServers.new.hostedTitle'),
      tag: t('gameServers.new.hostedTag'),
      desc: t('gameServers.new.hostedDesc'),
      supported: !!a?.modes.includes('hosted'),
      feats: [...HOSTED_FEATS.map((k) => ({ key: k, on: caps.has(k) })), { key: 'gm', on: mgp }],
    },
    {
      key: 'remote' as const,
      title: t('gameServers.new.remoteTitle'),
      tag: t('gameServers.new.remoteTag'),
      desc: t('gameServers.new.remoteDesc'),
      supported: !!a?.modes.includes('remote'),
      feats: [
        ...HOSTED_FEATS.slice(0, 4).map((k) => ({ key: k, on: false })),
        { key: 'gm', on: mgp },
        { key: 'passport', on: mgp },
      ],
    },
  ]
})
const modeTitle = computed(() => (mode.value === 'hosted' ? t('gameServers.new.hostedTitle') : t('gameServers.new.remoteTitle')))

const steps = computed(() => [
  { key: 'game', label: t('gameServers.new.steps.game') },
  { key: 'mode', label: t('gameServers.new.steps.mode') },
  { key: 'form', label: mode.value === 'remote' ? t('gameServers.new.steps.remote') : t('gameServers.new.steps.hosted') },
  { key: 'done', label: t('gameServers.new.steps.done') },
])
const stepDone = (i: number) => {
  if (created.value) return true
  if (i === 0) return !!adapter.value
  if (i === 1) return modeOk.value
  return false
}
// 能走到的最远一步：前面的选择都做好了才能往后跳；接入完成后步骤条不再可点
const maxStep = computed(() => (!adapter.value ? 0 : !modeOk.value ? 1 : 2))
const canJumpTo = (i: number) => !created.value && i !== step.value && i <= maxStep.value

const pill = computed<{ variant: PillVariant; label: string }>(() => {
  if (!checks.value.length) return { variant: 'neutral', label: t('gameServers.new.pillNone') }
  const fail = checks.value.filter((c) => c.state === 'fail').length
  const warn = checks.value.filter((c) => c.state === 'warn').length
  if (fail) return { variant: 'error', label: t('gameServers.new.pillFail', { n: fail }) }
  if (warn) return { variant: 'warning', label: t('gameServers.new.pillWarn', { n: warn }) }
  return { variant: 'success', label: t('gameServers.new.pillOk') }
})

const manifestNote = computed(() => {
  const m = checks.value.find((c) => c.key === 'manifest' && c.state === 'ok')
  if (!m) return ''
  return t('gameServers.new.afterManifest', { gameName: m.params.gameName, protocol: m.params.protocol })
})

const runCheck = async () => {
  if (!game.value) return
  checking.value = true
  try {
    if (mode.value === 'hosted') {
      const { data } = await detectGameServerDir({
        game: game.value,
        installDir: hosted.value.installDir.trim(),
        configFile: hosted.value.configFile,
        listens: detectListens(hosted.value),
      })
      const first = !detected.value || detected.value.installDir !== data.installDir
      detected.value = data
      checks.value = data.checks ?? []
      applyDetection(hosted.value, data, first, listenKeys.value)
    } else {
      if (!remoteKey.value) {
        fb.warning(t('gameServers.key.needGenerate'))
        return
      }
      const { data } = await testGameServer({
        mgpEndpoint: remote.endpoint.trim(),
        keyId: remoteKey.value.keyId,
        secret: remoteKey.value.secret,
      })
      checks.value = data.checks ?? []
    }
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    checking.value = false
  }
}

const canSubmit = computed(() => {
  if (!name.value.trim() || !game.value || !mode.value) return false
  if (mode.value === 'hosted') return !!detected.value?.ok && !!hosted.value.installDir.trim()
  return !!remote.endpoint.trim() && !!remoteKey.value
})

const submit = async () => {
  if (!canSubmit.value) {
    fb.warning(t(mode.value === 'hosted' ? 'gameServers.new.needDetect' : 'gameServers.new.needTest'))
    return
  }
  submitting.value = true
  try {
    const { data } = await createGameServer({
      game: game.value,
      name: name.value.trim(),
      tags: tags.value,
      mode: mode.value,
      ...(mode.value === 'hosted'
        ? { hosted: toHostedSpec(hosted.value) }
        : { mgpEndpoint: remote.endpoint.trim(), keyId: remoteKey.value?.keyId, secret: remoteKey.value?.secret }),
    })
    created.value = data
    step.value = 3
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    submitting.value = false
  }
}

// 再接入一台：保留已选的游戏与接入方式，表单清空，回到第一个要选的步骤
const reset = () => {
  created.value = null
  name.value = ''
  tags.value = []
  hosted.value = emptyHostedDraft()
  remote.endpoint = ''
  remoteKey.value = null
  detected.value = null
  checks.value = []
  step.value = firstStep()
}
</script>

<style scoped lang="scss">
.gsn {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.gsn__back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  align-self: flex-start;
  min-height: 40px;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  text-decoration: none;
}
.gsn__back:hover {
  color: var(--el-text-color-primary);
}
.gsn__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px 14px;
}
.gsn__head h1 {
  flex: 1 1 260px;
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.375rem;
  font-weight: 600;
}
.gsn__steps {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.gsn__step {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding: 0 14px 0 6px;
  border: 0;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
  transition:
    background 0.15s,
    color 0.15s,
    box-shadow 0.15s;
}
.gsn__step:disabled {
  cursor: default;
}
.gsn__step:focus-visible {
  outline: none;
  box-shadow: var(--app-focus-ring, 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 25%, transparent));
}
.gsn__step-num {
  width: 24px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--el-border-color-light);
  font-size: 0.75rem;
}
.gsn__step.is-done {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.gsn__step.is-done .gsn__step-num {
  background: var(--m-p);
  color: var(--m-pOn);
}
.gsn__step:not(:disabled):hover {
  box-shadow: 0 0 0 2px var(--m-p);
}
.gsn__step.is-on {
  background: var(--m-p);
  color: var(--m-pOn);
  box-shadow: 0 0 0 4px var(--m-pSoft);
}
.gsn__step.is-on .gsn__step-num {
  background: var(--m-pOn);
  color: var(--m-p);
}
.gsn__pane {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gsn__pane-head {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.gsn__pane-head h2 {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
}
.gsn__pane-head p {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.gsn__ctx {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 600;
}
.gsn__card h2 {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 600;
}
.gsn__opt {
  display: flex;
  padding: 0;
  border-radius: var(--app-radius-lg);
  border: 2px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  overflow: hidden;
  transition:
    border-color 0.15s,
    box-shadow 0.15s;
}
.gsn__opt:hover:not(:disabled) {
  border-color: var(--m-p);
}
.gsn__opt.is-on {
  border-color: var(--m-p);
  box-shadow: 0 0 0 4px var(--m-pSoft);
}
.gsn__opt:disabled {
  cursor: not-allowed;
  opacity: 0.62;
}
.gsn__games {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 14px;
}
.gsn__game {
  flex-direction: column;
}
.gsn__game-band {
  position: relative;
  display: block;
  height: 84px;
  background: var(--m-gSoft, var(--m-s2));
}
.gsn__game-logo {
  position: absolute;
  left: 14px;
  bottom: -22px;
  width: 52px;
  height: 52px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--m-gInk, var(--m-pInk));
  font-family: var(--app-font-display);
  font-size: 1rem;
  font-weight: 600;
  box-shadow: 3px 3px 0 var(--m-g, var(--m-p));
  transform: rotate(-4deg);
}
.gsn__game-art {
  position: relative;
  display: block;
  aspect-ratio: 2 / 1;
  background: var(--m-gSoft, var(--m-s2));
  overflow: hidden;
}
.gsn__game-art :deep(img) {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: 50% 30%;
  transition: transform 0.3s ease;
}
.gsn__game:hover .gsn__game-art :deep(img) {
  transform: scale(1.04);
}
.gsn__game-check {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--m-p);
  color: var(--m-pOn);
  box-shadow: 0 0 0 3px var(--el-bg-color);
}
.gsn__game-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 20px 16px 16px;
}
.gsn__game-art + .gsn__game-body {
  padding-top: 14px;
}
.gsn__game-body small,
.gsn__mode-body small {
  font-size: 0.75rem;
  line-height: 1.55;
  color: var(--el-text-color-secondary);
}
.gsn__caps {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}
.gsn__cap,
.gsn__tag {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.6875rem;
  font-weight: 500;
}
.gsn__modes {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}
.gsn__mode {
  align-items: flex-start;
  gap: 16px;
  padding: 14px;
}
.gsn__mode-ico {
  flex: none;
  width: 52px;
  height: 52px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  font-size: 1.1875rem;
  transform: rotate(-4deg);
}
.gsn__mode-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.gsn__mode-title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  font-size: 0.875rem;
}
.gsn__feats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px 14px;
  font-size: 0.8125rem;
}
.gsn__feats span {
  display: flex;
  align-items: center;
  gap: 6px;
}
.gsn__feats span.is-off {
  color: var(--el-text-color-secondary);
}
.gsn__nav {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 10px;
  padding-top: 4px;
}
.gsn__form-row {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 14px;
}
.gsn__card {
  border-radius: var(--app-radius-lg);
  border: var(--cmp-card-border-width, 1px) solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.gsn__form {
  flex: 3 1 540px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px 16px;
}
.gsn__check {
  flex: 2 1 320px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 14px 16px;
}
.gsn__check-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.gsn__check-head h2 {
  flex: 1;
}
.gsn__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}
.gsn__field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.gsn__field > span {
  font-size: 0.8125rem;
  font-weight: 600;
}
.gsn__field small {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.gsn__remote {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gsn__note {
  margin: 0;
  padding: 14px 16px;
  border-radius: var(--app-radius);
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.7;
}
.gsn__muted {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.gsn__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 6px;
}
.gsn__actions--center {
  justify-content: center;
}
.gsn__spacer {
  flex: 1;
}
.gsn__done {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 30px 16px;
  text-align: center;
}
.gsn__done-ico {
  width: 56px;
  height: 56px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  font-size: 1.1875rem;
}
.gsn-step-enter-active,
.gsn-step-leave-active {
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}
.gsn-step-enter-from {
  opacity: 0;
  transform: translateY(4px);
}
.gsn-step-leave-to {
  opacity: 0;
}
@media (prefers-reduced-motion: reduce) {
  .gsn-step-enter-active,
  .gsn-step-leave-active,
  .gsn__game-art :deep(img) {
    transition: none;
  }
}
@media (width <= 768px) {
  .gsn__modes,
  .gsn__grid,
  .gsn__feats {
    grid-template-columns: 1fr;
  }
  .gsn__form {
    padding: 14px 16px;
  }
  // 手机上只有当前步显示文字，其余只留序号
  .gsn__step:not(.is-on) {
    padding: 0 6px;
  }
  .gsn__step:not(.is-on) .gsn__step-label {
    display: none;
  }
  .gsn__actions :deep(.u-button),
  .gsn__actions > *,
  .gsn__nav > * {
    min-height: 40px;
  }
}
</style>
