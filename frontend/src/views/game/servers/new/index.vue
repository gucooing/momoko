<!-- 接入游戏服务器：选择游戏（内置适配器）→ 接入方式（托管 / 仅 GM，按适配器支持的方式）→ 表单 → 完成。
     右侧是预检 / 连接测试的真实结果。 -->
<template>
  <div class="gsn" :data-game="game || undefined">
    <RouterLink to="/game/servers" class="gsn__back">
      <UIcon name="i-lucide-chevron-left" />{{ t('gameServers.new.back') }}
    </RouterLink>
    <div class="gsn__head">
      <h1>{{ t('gameServers.new.title') }}</h1>
      <ol class="gsn__steps">
        <li v-for="(s, i) in steps" :key="s.key" :class="{ 'is-done': s.done }">
          <span class="m-num">{{ i + 1 }}</span>{{ s.label }}
        </li>
      </ol>
    </div>

    <template v-if="!created">
      <section class="gsn__section">
        <h2>{{ t('gameServers.new.sectionGame') }}</h2>
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
            @click="pickGame(a.id)"
          >
            <span class="gsn__game-band"><span class="gsn__game-logo">{{ adapterName(a.id).charAt(0) }}</span></span>
            <span class="gsn__game-body">
              <b>{{ adapterName(a.id) }}</b>
              <small>{{ t(`gameServers.list.adapterDriver.${a.gmDriver}`, a.gmDriver) }}</small>
              <span class="gsn__caps">
                <span v-for="m in a.modes" :key="m" class="gsn__cap">{{ t(`gameServers.mode.${m}`) }}</span>
                <span v-for="c in a.capabilities.slice(0, 4)" :key="c.key" class="gsn__cap">{{ t(`gameServers.caps.${c.key}`) }}</span>
              </span>
            </span>
          </button>
        </div>
      </section>

      <section v-if="adapter" class="gsn__section">
        <h2>{{ t('gameServers.new.sectionMode') }}</h2>
        <div class="gsn__modes">
          <button
            v-for="m in modeCards"
            :key="m.key"
            type="button"
            class="gsn__opt gsn__mode"
            :class="{ 'is-on': mode === m.key }"
            :disabled="!m.supported"
            :aria-pressed="mode === m.key"
            @click="pickMode(m.key)"
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
      </section>

      <section v-if="adapter && mode" class="gsn__form-row">
        <div class="gsn__card gsn__form">
          <h2>{{ t('gameServers.new.sectionForm', { title: mode === 'hosted' ? t('gameServers.new.steps.hosted') : t('gameServers.new.steps.remote') }) }}</h2>
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

          <HostedForm v-if="mode === 'hosted'" v-model="hosted" :detected="detected" :detecting="checking" @detect="runCheck" />

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
            <UButton color="neutral" variant="outline" to="/game/servers">{{ t('gameServers.new.cancel') }}</UButton>
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
      </section>
    </template>

    <section v-else class="gsn__card gsn__done">
      <span class="gsn__done-ico m-sticker m-sticker--ok"><UIcon name="i-lucide-check" /></span>
      <h2>{{ t('gameServers.new.doneTitle', { name: created.brief?.name || '' }) }}</h2>
      <p class="gsn__muted">{{ t('gameServers.new.doneDesc') }}</p>
      <div class="gsn__actions gsn__actions--center">
        <UButton color="neutral" variant="outline" to="/game/servers">{{ t('gameServers.new.backToList') }}</UButton>
        <UButton color="neutral" variant="soft" @click="reset">{{ t('gameServers.new.createAnother') }}</UButton>
        <UButton color="primary" :to="`/game/servers/${encodeURIComponent(created.brief?.id || '')}`">{{ t('gameServers.new.openServer') }}</UButton>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import '@/games'
import { createGameServer, detectGameServerDir, listGameAdapters, testGameServer } from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import type { DetectGameServerDirResponse, GameAdapter, GameServerCheck, GameServerInfo } from '@/types/v1/game_server'
import CheckList from '../components/CheckList.vue'
import HostedForm from '../components/HostedForm.vue'
import MgpKeyGenerator from '../components/MgpKeyGenerator.vue'
import TagInput from '../components/TagInput.vue'
import { applyDetection, detectListens, emptyHostedDraft, toHostedSpec } from '../hosted'
import type { PillVariant } from '../shared'
import type { MgpKey } from '../mgpKey'

defineOptions({ name: 'GameServerNewView' })

const { t, te } = useI18n()
const fb = useFeedback()

const adapters = ref<GameAdapter[]>([])
const adaptersLoaded = ref(false)
const available = computed(() => adapters.value.filter((a) => a.available))
const game = ref('')
const mode = ref<'' | 'hosted' | 'remote'>('')
const adapter = computed(() => available.value.find((a) => a.id === game.value))
const name = ref('')
const tags = ref<string[]>([])
const hosted = ref(emptyHostedDraft())
const remote = reactive({ endpoint: '' })
// 仅 GM：MGP 密钥由前端生成（点「生成」），生成后复制到服务端配置
const remoteKey = ref<MgpKey | null>(null)
const detected = ref<DetectGameServerDirResponse | null>(null)
const checks = ref<GameServerCheck[]>([])
const checking = ref(false)
const submitting = ref(false)
const created = ref<GameServerInfo | null>(null)

const adapterName = (id: string) => (te(`game.adapters.${id}`) ? t(`game.adapters.${id}`) : t('game.unnamed'))

onMounted(async () => {
  try {
    const { data } = await listGameAdapters()
    adapters.value = data.items ?? []
    if (available.value.length === 1) pickGame(available.value[0]!.id)
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
const pickMode = (m: 'hosted' | 'remote') => {
  mode.value = m
}
watch(mode, () => {
  checks.value = []
  detected.value = null
})

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

const steps = computed(() => [
  { key: 'game', label: t('gameServers.new.steps.game'), done: !!game.value },
  { key: 'mode', label: t('gameServers.new.steps.mode'), done: !!mode.value },
  {
    key: 'form',
    label: mode.value === 'remote' ? t('gameServers.new.steps.remote') : t('gameServers.new.steps.hosted'),
    done: checks.value.length > 0 && !checks.value.some((c) => c.state === 'fail'),
  },
  { key: 'done', label: t('gameServers.new.steps.done'), done: !!created.value },
])

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
      applyDetection(hosted.value, data, first)
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
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    submitting.value = false
  }
}

const reset = () => {
  created.value = null
  name.value = ''
  tags.value = []
  hosted.value = emptyHostedDraft()
  remote.endpoint = ''
  remoteKey.value = null
  detected.value = null
  checks.value = []
}
</script>

<style scoped lang="scss">
.gsn {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.gsn__back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  align-self: flex-start;
  min-height: 40px;
  font-size: 0.875rem;
  font-weight: 700;
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
  gap: 12px 18px;
}
.gsn__head h1 {
  flex: 1 1 260px;
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.875rem;
  font-weight: 800;
}
.gsn__steps {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.gsn__steps li {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding: 0 14px 0 6px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
}
.gsn__steps li span {
  width: 24px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--el-border-color-light);
  font-size: 0.75rem;
}
.gsn__steps li.is-done {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.gsn__steps li.is-done span {
  background: var(--m-p);
  color: var(--m-pOn);
}
.gsn__section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.gsn__section h2,
.gsn__card h2 {
  margin: 0;
  font-size: 1.0625rem;
  font-weight: 800;
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
  transition: border-color 0.15s, box-shadow 0.15s;
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
  font-size: 1.25rem;
  font-weight: 800;
  box-shadow: 3px 3px 0 var(--m-g, var(--m-p));
  transform: rotate(-4deg);
}
.gsn__game-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 30px 16px 16px;
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
  font-weight: 700;
}
.gsn__modes {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}
.gsn__mode {
  align-items: flex-start;
  gap: 16px;
  padding: 20px;
}
.gsn__mode-ico {
  flex: none;
  width: 52px;
  height: 52px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  font-size: 1.5rem;
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
  font-size: 1.0625rem;
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
.gsn__form-row {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 20px;
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
  gap: 18px;
  padding: 22px 24px;
}
.gsn__check {
  flex: 2 1 320px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 20px 22px;
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
  font-weight: 700;
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
.gsn__keygen,
.gsn__note {
  margin: 0;
  padding: 14px 16px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  font-size: 0.8125rem;
  line-height: 1.7;
  color: var(--el-text-color-secondary);
}
.gsn__note {
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
  font-weight: 600;
}
.gsn__muted {
  margin: 0;
  font-size: 0.875rem;
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
.gsn__done {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 40px 24px;
  text-align: center;
}
.gsn__done-ico {
  width: 56px;
  height: 56px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 18px;
  font-size: 1.5rem;
}
@media (width <= 768px) {
  .gsn__modes,
  .gsn__grid,
  .gsn__feats {
    grid-template-columns: 1fr;
  }
  .gsn__form {
    padding: 18px 16px;
  }
  .gsn__actions :deep(.u-button),
  .gsn__actions > * {
    min-height: 44px;
  }
}
</style>
