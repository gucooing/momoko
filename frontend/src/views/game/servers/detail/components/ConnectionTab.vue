<!-- 接入设置：页面上只放简要信息（一行一项：接入方式、MGP 接口、用户通行证、客户端应用凭据、客户端地址、托管选项），
     要改的点「修改」在弹窗里改：MGP 地址与密钥（可测试连接）、通行证可见性 / 邀请码 / 时区、
     客户端地址（只有这款游戏的客户端用到的，由适配器给出：学マス SDK / 游戏服务端 / 资源服务器，蓝色星原只有 SDK）、
     托管选项；切换接入方式也是弹窗。
     客户端（游戏支持通行证启动客户端时才有）：应用凭据由 Momoko 生成，重置密钥与重新生成应用 ID 都要先确认；
     应用密钥与 MGP 密钥一样只在生成时显示一次（可复制），之后只显示应用 ID。托管服务器同轮换 MGP 密钥：
     Momoko 把凭据写进服务端配置（momoko.app_id / momoko.app_secret），运行中可选立即重启，所以第一次生成也先开确认弹窗。 -->
<template>
  <div class="cn">
    <section class="cn__list">
      <!-- 接入方式 -->
      <div class="cn__row">
        <span class="cn__ico m-sticker m-sticker--a2"><UIcon :name="hosted ? 'i-lucide-cpu' : 'i-lucide-cloud'" /></span>
        <span class="cn__main">
          <b>{{ t('gameServers.connection.mode') }}</b>
          <span>{{ t(`gameServers.mode.${mode}`) }}</span>
        </span>
        <UButton v-if="hosted && canRemote" size="sm" color="neutral" variant="soft" @click="openRemote">
          {{ t('gameServers.connection.switchToRemote') }}
        </UButton>
        <UButton v-if="!hosted && canHosted" size="sm" color="neutral" variant="soft" @click="openHosted">
          {{ t('gameServers.connection.switchToHosted') }}
        </UButton>
      </div>

      <!-- MGP 接口 -->
      <div class="cn__row">
        <span class="cn__ico m-sticker m-sticker--p"><UIcon name="i-lucide-plug-zap" /></span>
        <span class="cn__main">
          <b>{{ t('gameServers.connection.mgp') }}</b>
          <span class="m-mono">{{ server.mgpEndpoint || t('gameServers.list.mgpNone') }}<template v-if="server.keyId"> · {{ server.keyId }}</template></span>
          <small v-if="server.probeError && !server.brief?.mgpOk" class="cn__error">{{ server.probeError }}</small>
        </span>
        <StatusPill :variant="mgpPill.variant" :label="mgpPill.label" :title="probeTitle" />
        <UButton v-if="!hosted" size="sm" color="neutral" variant="soft" @click="openMgp">{{ t('gameServers.connection.modify') }}</UButton>
        <small v-else class="cn__aside" :title="t('gameServers.connection.hostedMgp')">{{ t('gameServers.connection.fromConfig') }}</small>
      </div>

      <!-- 用户通行证 -->
      <div class="cn__row">
        <span class="cn__ico m-sticker m-sticker--a"><UIcon name="i-lucide-id-card" /></span>
        <span class="cn__main">
          <b>{{ t('gameServers.connection.passport') }}</b>
          <span>{{ passportSummary }}</span>
        </span>
        <UButton size="sm" color="neutral" variant="soft" @click="openPassport">{{ t('gameServers.connection.modify') }}</UButton>
      </div>

      <!-- 客户端应用凭据（通行证启动） -->
      <div v-if="clientLaunch" class="cn__row">
        <span class="cn__ico m-sticker m-sticker--a"><UIcon name="i-lucide-badge-check" /></span>
        <span class="cn__main">
          <b>{{ t('gameServers.connection.clientApp') }}</b>
          <span v-if="server.clientAppId" class="m-mono">{{ server.clientAppId }}</span>
          <span v-else>{{ t('gameServers.connection.clientAppNone') }}</span>
        </span>
        <template v-if="server.clientAppId">
          <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('gameServers.connection.clientAppCopyId')" :box="30" @click="copy(server.clientAppId)" />
          <UButton size="sm" color="neutral" variant="soft" :loading="appBusy" @click="askAppAction('secret')">{{ t('gameServers.connection.clientAppResetSecret') }}</UButton>
          <UButton size="sm" color="neutral" variant="soft" :loading="appBusy" @click="askAppAction('id')">{{ t('gameServers.connection.clientAppRegenerate') }}</UButton>
        </template>
        <UButton v-else size="sm" color="primary" variant="soft" icon="i-lucide-key-round" :loading="appBusy" @click="generateApp">
          {{ t('gameServers.connection.clientAppGenerate') }}
        </UButton>
      </div>

      <!-- 客户端地址（通行证启动） -->
      <div v-if="clientLaunch" class="cn__row">
        <span class="cn__ico m-sticker m-sticker--warn"><UIcon name="i-lucide-monitor-smartphone" /></span>
        <span class="cn__main">
          <b>{{ t('gameServers.connection.client') }}</b>
          <span :class="{ 'm-mono': clientSet }">{{ clientSummary }}</span>
        </span>
        <StatusPill v-if="clientPartial" variant="warning" :label="t('gameServers.connection.clientPartial')" :title="clientHint" />
        <UButton size="sm" color="neutral" variant="soft" @click="openClient">{{ t('gameServers.connection.modify') }}</UButton>
      </div>

      <!-- 应用图标与应用背景（上传到对象存储）：服务器页页头与启动器同步使用 -->
      <div v-for="m in APP_MEDIA" :key="m.kind" class="cn__row">
        <span class="cn__ico m-sticker m-sticker--a2"><UIcon :name="m.icon" /></span>
        <span class="cn__main">
          <b>{{ t(`gameServers.connection.media.${m.kind}`) }}</b>
          <span>{{ mediaLink(m.kind) ? t('gameServers.connection.media.hint') : `${t('gameServers.connection.media.none')} · ${t('gameServers.connection.media.hint')}` }}</span>
        </span>
        <template v-if="mediaLink(m.kind)">
          <video v-if="isVideoLink(mediaLink(m.kind))" class="cn__media" :src="mediaLink(m.kind)" muted loop autoplay playsinline />
          <img v-else class="cn__media" :class="{ 'cn__media--icon': m.kind === 'icon' }" :src="mediaLink(m.kind)" alt="" />
        </template>
        <UButton size="sm" color="neutral" variant="soft" :loading="mediaBusy === m.kind" @click="pickMedia(m.kind)">
          {{ t(mediaLink(m.kind) ? 'gameServers.connection.media.replace' : 'gameServers.connection.media.upload') }}
        </UButton>
        <UButton v-if="mediaLink(m.kind)" size="sm" color="error" variant="soft" :disabled="mediaBusy !== ''" @click="removeMedia(m.kind)">
          {{ t('gameServers.connection.media.remove') }}
        </UButton>
      </div>
      <input ref="mediaInput" type="file" hidden :accept="mediaAccept" @change="uploadMedia" />

      <!-- 托管选项 -->
      <div v-if="hosted && server.hosting" class="cn__row">
        <span class="cn__ico m-sticker m-sticker--ok"><UIcon name="i-lucide-server-cog" /></span>
        <span class="cn__main">
          <b>{{ t('gameServers.connection.hosting') }}</b>
          <span>{{ hostingSummary }}</span>
        </span>
        <UButton size="sm" color="neutral" variant="soft" @click="openHosting">{{ t('gameServers.connection.modify') }}</UButton>
      </div>
    </section>

    <!-- 弹窗：MGP 地址与密钥（仅 GM） -->
    <FormDialog
      v-model="mgpOpen"
      :title="t('gameServers.connection.editMgpTitle')"
      :width="560"
      :loading="savingMgp"
      :confirm-text="t('gameServers.connection.save')"
      @confirm="saveMgp"
    >
      <div class="cn__dialog">
        <label class="cn__field">
          <span>{{ t('gameServers.connection.endpoint') }}</span>
          <UInput v-model="mgp.endpoint" class="m-mono" icon="i-lucide-globe" />
        </label>
        <div class="cn__field">
          <span>{{ t('gameServers.key.title') }}</span>
          <MgpKeyGenerator v-model="mgpKey" :current-key-id="server.keyId" :hint="t('gameServers.key.remoteHint')" clearable />
        </div>
        <div class="cn__test">
          <UButton color="neutral" variant="soft" icon="i-lucide-zap" :loading="testing" @click="test">{{ t('gameServers.connection.test') }}</UButton>
        </div>
        <CheckList v-if="checks.length" :checks="checks" />
      </div>
    </FormDialog>

    <!-- 弹窗：用户通行证 -->
    <FormDialog
      v-model="passportOpen"
      :title="t('gameServers.connection.passport')"
      :width="520"
      :loading="savingPassport"
      :confirm-text="t('gameServers.connection.save')"
      @confirm="savePassport"
    >
      <div class="cn__dialog">
        <div class="cn__field">
          <span>{{ t('gameServers.connection.visibility') }}</span>
          <div class="cn__seg" role="radiogroup" :aria-label="t('gameServers.connection.visibility')">
            <button
              v-for="v in VISIBILITY"
              :key="v"
              type="button"
              role="radio"
              :aria-checked="form.visibility === v"
              :class="{ 'is-on': form.visibility === v }"
              @click="form.visibility = v"
            >
              {{ t(`gameServers.connection.visibilityOpts.${v}`) }}
            </button>
          </div>
          <small>{{ t(`gameServers.connection.visibilityDesc.${form.visibility}`) }}</small>
        </div>
        <label v-if="form.visibility === 'invite'" class="cn__field">
          <span>{{ t('gameServers.connection.inviteCode') }}</span>
          <UInput v-model="form.inviteCode" class="m-mono" :maxlength="32" />
          <small>{{ t('gameServers.connection.inviteCodeHint') }}</small>
        </label>
        <label class="cn__field">
          <span>{{ t('gameServers.connection.timezone') }}</span>
          <UInput v-model="form.timezone" class="m-mono" :placeholder="t('gameServers.connection.timezonePh')" list="cn-timezones" />
          <datalist id="cn-timezones">
            <option v-for="z in timezones" :key="z" :value="z" />
          </datalist>
          <small>{{ t('gameServers.connection.timezoneHint') }}</small>
        </label>
      </div>
    </FormDialog>

    <!-- 弹窗：客户端地址 -->
    <FormDialog
      v-model="clientOpen"
      :title="t('gameServers.connection.client')"
      :width="560"
      :loading="savingClient"
      :confirm-text="t('gameServers.connection.save')"
      @confirm="saveClient"
    >
      <div class="cn__dialog">
        <p>{{ clientHint }}</p>
        <label v-for="e in clientEndpoints" :key="e.key" class="cn__field">
          <span>{{ t(`gameServers.connection.clientField.${e.key}`) }}</span>
          <UInput v-model="client[e.key]" class="m-mono" placeholder="https://host:port" />
          <small v-if="hostedListen(e)">{{ t('gameServers.connection.clientListen', { key: e.listen, listen: hostedListen(e) }) }}</small>
        </label>
        <small>{{ t('gameServers.connection.clientFormat') }}</small>
      </div>
    </FormDialog>

    <!-- 弹窗：确认生成（托管）/ 重置应用密钥 / 重新生成应用 ID -->
    <FormDialog
      :model-value="!!appAction"
      :title="appActionText.title"
      :width="520"
      :loading="appBusy"
      :confirm-text="appActionText.confirm"
      @update:model-value="closeAppAction"
      @confirm="runAppAction"
    >
      <div class="cn__dialog">
        <p>{{ appActionText.desc }}</p>
        <label v-if="hosted" class="cn__toggle">
          <span>
            <b>{{ t('gameServers.connection.clientAppRestart') }}</b>
            <small>{{ t('gameServers.connection.clientAppRestartHint') }}</small>
          </span>
          <AppSwitch v-model="appRestart" :aria-label="t('gameServers.connection.clientAppRestart')" />
        </label>
      </div>
    </FormDialog>

    <!-- 弹窗：新的应用凭据（应用密钥只显示这一次） -->
    <FormDialog v-model="credsOpen" :title="t('gameServers.connection.clientAppCredsTitle')" :width="600" :close-on-overlay="false">
      <div v-if="creds" class="cn__dialog">
        <p>{{ hosted ? t('gameServers.connection.clientAppCredsHostedDesc') : t('gameServers.connection.clientAppCredsDesc') }}{{ appNote('creds') }}</p>
        <div class="cn__cred">
          <span>{{ t('gameServers.connection.clientAppId') }}</span>
          <UInput :model-value="creds.appId" class="m-mono" readonly :ui="{ root: 'cn__cred-input' }" :aria-label="t('gameServers.connection.clientAppId')" />
          <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('gameServers.connection.clientAppCopyId')" :box="30" @click="copy(creds.appId)" />
        </div>
        <div class="cn__cred">
          <span>{{ t('gameServers.connection.clientAppSecret') }}</span>
          <UInput :model-value="creds.appSecret" class="m-mono" readonly :ui="{ root: 'cn__cred-input' }" :aria-label="t('gameServers.connection.clientAppSecret')" />
          <AppIconButton icon="HOutline:ClipboardDocumentIcon" :label="t('gameServers.connection.clientAppCopySecret')" :box="30" @click="copy(creds.appSecret)" />
        </div>
        <small>{{ t('gameServers.connection.clientAppCredsHint') }}</small>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="ghost" icon="i-lucide-copy" @click="copyCreds">{{ t('gameServers.connection.clientAppCopyBoth') }}</UButton>
        <UButton color="primary" @click="close">{{ t('gameServers.connection.clientAppSaved') }}</UButton>
      </template>
    </FormDialog>

    <!-- 弹窗：托管选项 -->
    <FormDialog
      v-model="hostingOpen"
      :title="t('gameServers.connection.hosting')"
      :width="600"
      :loading="savingHost"
      :confirm-text="t('gameServers.connection.save')"
      @confirm="saveHosting"
    >
      <div class="cn__dialog">
        <label class="cn__toggle">
          <span>
            <b>{{ t('gameServers.new.tgAutoStart') }}</b>
            <small>{{ t('gameServers.new.tgAutoStartDesc') }}</small>
          </span>
          <AppSwitch v-model="host.autoStart" :aria-label="t('gameServers.new.tgAutoStart')" />
        </label>
        <label class="cn__toggle">
          <span>
            <b>{{ t('gameServers.new.tgAutoRestart') }}</b>
            <small>{{ t('gameServers.new.tgAutoRestartDesc') }}</small>
          </span>
          <AppSwitch v-model="host.autoRestart" :aria-label="t('gameServers.new.tgAutoRestart')" />
        </label>
        <label v-if="canResource" class="cn__toggle">
          <span>
            <b>{{ t('gameServers.new.tgResource') }}</b>
            <small>{{ t('gameServers.new.tgResourceDesc') }}</small>
          </span>
          <AppSwitch v-model="host.resourceServer" :aria-label="t('gameServers.new.tgResource')" />
        </label>
        <div v-if="canResource && host.resourceServer" class="cn__grid">
          <label class="cn__field">
            <span>{{ t('gameServers.role.resource') }}</span>
            <UInput v-model="host.resourceListen" class="m-mono" placeholder="host:port" />
          </label>
          <label class="cn__field">
            <span>{{ t('gameServers.new.resourceOrigin') }}</span>
            <UInput v-model="host.resourceOrigin" class="m-mono" placeholder="http://host:port" />
          </label>
          <label class="cn__field cn__wide">
            <span>{{ t('gameServers.new.resourceRoot') }}</span>
            <UInput v-model="host.resourceRoot" class="m-mono" />
          </label>
        </div>
      </div>
    </FormDialog>

    <!-- 弹窗：改为仅 GM -->
    <FormDialog
      v-model="remoteOpen"
      :title="t('gameServers.connection.switchRemoteTitle')"
      :width="560"
      :loading="switching"
      @confirm="switchToRemote"
    >
      <div class="cn__dialog">
        <p>{{ t('gameServers.connection.switchRemoteDesc') }}</p>
        <label v-if="server.mgpEndpoint" class="cn__toggle">
          <span><b>{{ t('gameServers.connection.keepEndpoint', { endpoint: server.mgpEndpoint }) }}</b></span>
          <AppSwitch v-model="sw.keep" :aria-label="t('gameServers.connection.keepEndpoint', { endpoint: server.mgpEndpoint })" />
        </label>
        <template v-if="!sw.keep || !server.mgpEndpoint">
          <label class="cn__field">
            <span>{{ t('gameServers.connection.endpoint') }}</span>
            <UInput v-model="sw.endpoint" class="m-mono" />
          </label>
          <div class="cn__field">
            <span>{{ t('gameServers.key.title') }}</span>
            <MgpKeyGenerator v-model="swKey" :hint="t('gameServers.key.remoteHint')" />
          </div>
        </template>
        <small>{{ t('gameServers.connection.removeProcessesNote') }}</small>
      </div>
    </FormDialog>

    <!-- 弹窗：改为托管 -->
    <FormDialog v-model="hostedOpen" :title="t('gameServers.connection.switchHostedTitle')" :width="760" :loading="switching" @confirm="switchToHosted">
      <div class="cn__dialog">
        <HostedForm v-model="draft" :detected="detected" :detecting="detecting" :listen-keys="listenKeys" :resource="canResource" @detect="detect" />
        <CheckList v-if="detected?.checks.length" :checks="detected.checks" />
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  deleteGameServerAppMedia,
  detectGameServerDir,
  generateGameServerClientApp,
  type AppMediaKind,
  uploadGameServerAppMedia,
  resetGameServerClientAppSecret,
  switchGameServerMode,
  testGameServer,
  updateGameServer,
  updateGameServerHosting,
} from '@/api/gameServer'
import { isVideoLink } from '@/games/serverMedia'
import { copyTextToClipboard } from '@/utils/file'
import { getRequestErrorMessage } from '@/utils/request'
import type {
  DetectGameServerDirResponse,
  GameAdapter,
  GameAdapterClientEndpoint,
  GameServerCheck,
  GameServerClientAppResponse,
  GameServerInfo,
  UpdateGameServerRequest,
} from '@/types/v1/game_server'
import CheckList from '../../components/CheckList.vue'
import HostedForm from '../../components/HostedForm.vue'
import { applyDetection, detectListens, emptyHostedDraft, mainListenKeys, toHostedSpec } from '../../hosted'
import { fmtAgo, type PillVariant } from '../../shared'
import type { MgpKey } from '../../mgpKey'
import MgpKeyGenerator from '../../components/MgpKeyGenerator.vue'

const props = defineProps<{ server: GameServerInfo; adapter?: GameAdapter | null }>()
const emit = defineEmits<{ changed: [] }>()
const { t, te } = useI18n()
const fb = useFeedback()

const VISIBILITY = ['public', 'invite', 'hidden'] as const
type Visibility = (typeof VISIBILITY)[number]

const id = computed(() => props.server.brief?.id || '')
const mode = computed(() => props.server.brief?.mode || 'remote')
const hosted = computed(() => mode.value === 'hosted')
const canHosted = computed(() => !!props.adapter?.modes.includes('hosted'))
const canRemote = computed(() => !!props.adapter?.modes.includes('remote'))
const clientLaunch = computed(() => !!props.adapter?.clientEndpoints.length)
const listenKeys = computed(() => mainListenKeys(props.adapter))
const canResource = computed(() => !!props.adapter?.capabilities.some((c) => c.key === 'resource' && c.enabled))
const serverVisibility = computed<Visibility>(() =>
  (VISIBILITY as readonly string[]).includes(props.server.passportVisibility) ? (props.server.passportVisibility as Visibility) : 'public',
)

// ---- 简要信息 ----
const mgpPill = computed<{ variant: PillVariant; label: string }>(() => {
  const s = props.server
  if (!s.mgpEndpoint) return { variant: 'neutral', label: t('gameServers.list.mgpNone') }
  if (!s.probeAt) return { variant: 'neutral', label: t('gameServers.list.mgpUnknown') }
  return s.brief?.mgpOk
    ? { variant: 'success', label: t('gameServers.list.mgpOk', { ms: s.brief.mgpLatencyMs }) }
    : { variant: 'error', label: t('gameServers.list.mgpDown') }
})
const probeTitle = computed(() => (props.server.probeAt ? t('gameServers.connection.probeAt', { t: fmtAgo(t, props.server.probeAt) }) : undefined))
const passportSummary = computed(() => {
  const tz = props.server.timezone ? t('gameServers.connection.tzValue', { tz: props.server.timezone }) : t('gameServers.connection.tzFollow')
  return `${t(`gameServers.connection.visibilityOpts.${serverVisibility.value}`)} · ${tz}`
})
// 客户端地址：只有这款游戏的客户端用到的那几项（适配器给出；学マス 三项，蓝色星原只有 SDK）
type ClientKey = 'sdk' | 'game' | 'resource'
const clientEndpoints = computed(() => (props.adapter?.clientEndpoints ?? []) as (GameAdapterClientEndpoint & { key: ClientKey })[])
const clientUrl = (key: ClientKey) =>
  ({ sdk: props.server.clientSdkUrl, game: props.server.clientGameUrl, resource: props.server.clientResourceUrl })[key] || ''
const clientSet = computed(() => clientEndpoints.value.some((e) => clientUrl(e.key)))
const clientPartial = computed(() => clientSet.value && clientEndpoints.value.some((e) => e.required && !clientUrl(e.key)))
const clientSummary = computed(() => {
  if (!clientSet.value) return t('gameServers.connection.clientNone')
  const [only, ...rest] = clientEndpoints.value
  if (only && !rest.length) return clientUrl(only.key)
  return clientEndpoints.value
    .map((e) => {
      const url = clientUrl(e.key) || t(e.required ? 'gameServers.connection.clientUnset' : 'gameServers.connection.clientOfficial')
      return `${t(`gameServers.connection.clientShort.${e.key}`)} ${url}`
    })
    .join(' · ')
})
const clientHint = computed(() => {
  const key = `gameServers.connection.clientHint.${props.server.brief?.game || ''}`
  return te(key) ? t(key) : t('gameServers.connection.clientHint.default')
})
// 托管服务器上提供这个地址的监听（如蓝色星原 sdk.listen），提示管理员填它的对外地址
const hostedListen = (e: GameAdapterClientEndpoint) =>
  hosted.value && e.listen ? props.server.processes.find((p) => p.listens?.[e.listen])?.listens[e.listen] || '' : ''
const hostingSummary = computed(() => {
  const h = props.server.hosting
  if (!h) return ''
  const on = [
    h.autoStart && t('gameServers.new.tgAutoStart'),
    h.autoRestart && t('gameServers.new.tgAutoRestart'),
    h.resourceServer && t('gameServers.role.resource'),
  ].filter(Boolean)
  return on.length ? on.join(' · ') : t('gameServers.connection.hostingNone')
})

// 保存时其余字段都用服务器当前的值，只改弹窗里那一组；secret 为空表示沿用当前密钥
const currentFields = (): UpdateGameServerRequest => {
  const s = props.server
  return {
    id: id.value,
    name: s.brief?.name || '',
    tags: s.brief?.tags || [],
    cover: s.brief?.cover || '',
    remark: s.remark,
    mgpEndpoint: s.mgpEndpoint,
    keyId: s.keyId,
    secret: '',
    passportVisibility: serverVisibility.value,
    passportInviteCode: s.passportInviteCode,
    timezone: s.timezone,
    clientSdkUrl: s.clientSdkUrl,
    clientGameUrl: s.clientGameUrl,
    clientResourceUrl: s.clientResourceUrl,
  }
}

const update = async (req: UpdateGameServerRequest, busy: { value: boolean }) => {
  busy.value = true
  try {
    await updateGameServer(req)
    fb.success(t('gameServers.common.saved'))
    emit('changed')
    return true
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
    return false
  } finally {
    busy.value = false
  }
}

// ---- 弹窗：MGP 地址与密钥 ----
const mgpOpen = ref(false)
const mgp = reactive({ endpoint: '' })
// 新的 MGP 密钥（填写或点「生成」得到）；为空时沿用当前密钥
const mgpKey = ref<MgpKey | null>(null)
const checks = ref<GameServerCheck[]>([])
const testing = ref(false)
const savingMgp = ref(false)

const openMgp = () => {
  mgp.endpoint = props.server.mgpEndpoint
  mgpKey.value = null
  checks.value = []
  mgpOpen.value = true
}

const test = async () => {
  testing.value = true
  try {
    const { data } = await testGameServer({
      id: id.value,
      mgpEndpoint: mgp.endpoint.trim(),
      keyId: mgpKey.value?.keyId || '',
      secret: mgpKey.value?.secret || '',
    })
    checks.value = data.checks ?? []
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    testing.value = false
  }
}

const saveMgp = async () => {
  const ok = await update(
    {
      ...currentFields(),
      mgpEndpoint: mgp.endpoint.trim(),
      keyId: mgpKey.value?.keyId || props.server.keyId,
      secret: mgpKey.value?.secret || '',
    },
    savingMgp,
  )
  if (ok) mgpOpen.value = false
}

// ---- 弹窗：用户通行证 ----
const passportOpen = ref(false)
const form = reactive<{ visibility: Visibility; inviteCode: string; timezone: string }>({ visibility: 'public', inviteCode: '', timezone: '' })
const savingPassport = ref(false)

const openPassport = () => {
  const s = props.server
  Object.assign(form, { visibility: serverVisibility.value, inviteCode: s.passportInviteCode, timezone: s.timezone })
  passportOpen.value = true
}

const timezones = computed(() => {
  const fn = (Intl as unknown as { supportedValuesOf?: (k: string) => string[] }).supportedValuesOf
  return fn ? fn('timeZone') : []
})

const savePassport = async () => {
  const ok = await update(
    {
      ...currentFields(),
      passportVisibility: form.visibility,
      passportInviteCode: form.inviteCode.trim(),
      timezone: form.timezone.trim(),
    },
    savingPassport,
  )
  if (ok) passportOpen.value = false
}

// ---- 弹窗：客户端地址 ----
const clientOpen = ref(false)
const client = reactive<Record<ClientKey, string>>({ sdk: '', game: '', resource: '' })
const savingClient = ref(false)

const openClient = () => {
  Object.assign(client, { sdk: clientUrl('sdk'), game: clientUrl('game'), resource: clientUrl('resource') })
  clientOpen.value = true
}

// 只提交这款游戏用到的地址，其余为空（服务端同样只保存用到的）
const saveClient = async () => {
  const used = (key: ClientKey) => (clientEndpoints.value.some((e) => e.key === key) ? client[key].trim() : '')
  const ok = await update(
    {
      ...currentFields(),
      clientSdkUrl: used('sdk'),
      clientGameUrl: used('game'),
      clientResourceUrl: used('resource'),
    },
    savingClient,
  )
  if (ok) clientOpen.value = false
}

// ---- 客户端应用凭据 ----
const appBusy = ref(false)

// ---- 应用图标与应用背景：选文件后直传对象存储并保存链接，换文件或删除时服务端删掉旧文件 ----
const APP_MEDIA: { kind: AppMediaKind; icon: string }[] = [
  { kind: 'icon', icon: 'i-lucide-image' },
  { kind: 'background', icon: 'i-lucide-panels-top-left' },
]
const mediaInput = ref<HTMLInputElement>()
const mediaKind = ref<AppMediaKind>('icon')
const mediaBusy = ref<AppMediaKind | ''>('')
const mediaAccept = computed(() => (mediaKind.value === 'icon' ? 'image/*' : 'image/*,video/*'))
const mediaLink = (kind: AppMediaKind) => (kind === 'icon' ? props.server.appIcon : props.server.appBackground)

const pickMedia = (kind: AppMediaKind) => {
  mediaKind.value = kind
  // accept 跟着种类变，等它更新后再打开文件选择
  requestAnimationFrame(() => mediaInput.value?.click())
}

const uploadMedia = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  const kind = mediaKind.value
  mediaBusy.value = kind
  try {
    await uploadGameServerAppMedia(id.value, kind, file)
    fb.success(t('gameServers.common.saved'))
    emit('changed')
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.connection.media.uploadFailed')))
  } finally {
    mediaBusy.value = ''
  }
}

const removeMedia = async (kind: AppMediaKind) => {
  mediaBusy.value = kind
  try {
    await deleteGameServerAppMedia(id.value, kind)
    fb.success(t('gameServers.common.saved'))
    emit('changed')
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    mediaBusy.value = ''
  }
}
// 待确认的操作：new 托管服务器第一次生成 / secret 只换应用密钥 / id 应用 ID 与应用密钥一起换
type AppAction = 'new' | 'secret' | 'id'
const appAction = ref<AppAction | null>(null)
// 托管：服务端运行中时立即重启使新凭据生效（同轮换 MGP 密钥，默认开）
const appRestart = ref(true)
// 刚生成的凭据（应用密钥只在这里显示一次）
const creds = ref<GameServerClientAppResponse | null>(null)
const credsOpen = ref(false)

const copy = async (text: string) => {
  if (!text) return
  try {
    await copyTextToClipboard(text)
    fb.success(t('gameServers.key.copied'))
  } catch {
    fb.error(t('gameServers.key.copyFailed'))
  }
}

const copyCreds = () => {
  if (!creds.value) return
  void copy(`app_id: ${creds.value.appId}\napp_secret: ${creds.value.appSecret}`)
}

// 只属于某款游戏客户端的补充说明（学マス 的安卓客户端要打包应用 ID），没有时为空
const appNote = (kind: 'creds' | 'secret' | 'id') => {
  const key = `gameServers.connection.clientAppNote.${props.server.brief?.game || ''}.${kind}`
  return te(key) ? t(key) : ''
}

const appActionText = computed(() => {
  const c = 'gameServers.connection.'
  switch (appAction.value) {
    case 'secret':
      return {
        title: t(`${c}clientAppResetSecretTitle`),
        confirm: t(`${c}clientAppResetSecret`),
        desc: t(hosted.value ? `${c}clientAppResetSecretHostedDesc` : `${c}clientAppResetSecretDesc`) + appNote('secret'),
      }
    case 'id':
      return {
        title: t(`${c}clientAppRegenerateTitle`),
        confirm: t(`${c}clientAppRegenerate`),
        desc: t(hosted.value ? `${c}clientAppRegenerateHostedDesc` : `${c}clientAppRegenerateDesc`) + appNote('id'),
      }
    default:
      return { title: t(`${c}clientAppGenerateTitle`), confirm: t(`${c}clientAppGenerate`), desc: t(`${c}clientAppGenerateHostedDesc`) }
  }
})

const issueApp = async (action: AppAction) => {
  const restart = hosted.value && appRestart.value
  appBusy.value = true
  try {
    const { data } =
      action === 'secret' ? await resetGameServerClientAppSecret(id.value, restart) : await generateGameServerClientApp(id.value, restart)
    creds.value = data
    credsOpen.value = true
    appAction.value = null
    if (hosted.value) fb.success(t(data.restarted ? 'gameServers.connection.clientAppDoneRestart' : 'gameServers.connection.clientAppDone'))
    emit('changed')
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    appBusy.value = false
  }
}

// 仅 GM 的第一次生成直接执行；托管服务器要选是否重启，先开确认弹窗
const generateApp = () => (hosted.value ? askAppAction('new') : issueApp('new'))

const askAppAction = (action: AppAction) => {
  appRestart.value = true
  appAction.value = action
}

const closeAppAction = (open: boolean) => {
  if (!open) appAction.value = null
}

const runAppAction = async () => {
  if (appAction.value) await issueApp(appAction.value)
}

// ---- 弹窗：托管选项 ----
const hostingOpen = ref(false)
const host = reactive({ autoStart: true, autoRestart: true, resourceServer: false, resourceListen: '', resourceOrigin: '', resourceRoot: '' })
const savingHost = ref(false)

const openHosting = () => {
  const h = props.server.hosting
  if (!h) return
  Object.assign(host, {
    autoStart: h.autoStart,
    autoRestart: h.autoRestart,
    resourceServer: h.resourceServer,
    resourceListen: h.resourceListen,
    resourceOrigin: h.resourceOrigin,
    resourceRoot: h.resourceRoot,
  })
  hostingOpen.value = true
}

const saveHosting = async () => {
  savingHost.value = true
  try {
    await updateGameServerHosting({ id: id.value, ...host })
    fb.success(t('gameServers.common.saved'))
    hostingOpen.value = false
    emit('changed')
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    savingHost.value = false
  }
}

// ---- 弹窗：切换接入方式 ----
const switching = ref(false)
const remoteOpen = ref(false)
const sw = reactive({ keep: true, endpoint: '' })
const swKey = ref<MgpKey | null>(null)
const openRemote = () => {
  Object.assign(sw, { keep: !!props.server.mgpEndpoint, endpoint: '' })
  swKey.value = null
  remoteOpen.value = true
}
const switchToRemote = async () => {
  switching.value = true
  try {
    const keep = sw.keep && !!props.server.mgpEndpoint
    if (!keep && !swKey.value) {
      fb.warning(t('gameServers.key.needGenerate'))
      return
    }
    await switchGameServerMode({
      id: id.value,
      mode: 'remote',
      mgpEndpoint: keep ? '' : sw.endpoint.trim(),
      keyId: keep ? '' : swKey.value?.keyId || '',
      secret: keep ? '' : swKey.value?.secret || '',
    })
    fb.success(t('gameServers.connection.switched'))
    remoteOpen.value = false
    emit('changed')
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    switching.value = false
  }
}

const hostedOpen = ref(false)
const draft = ref(emptyHostedDraft())
const detected = ref<DetectGameServerDirResponse | null>(null)
const detecting = ref(false)
const openHosted = () => {
  draft.value = emptyHostedDraft()
  detected.value = null
  hostedOpen.value = true
}
const detect = async () => {
  detecting.value = true
  try {
    const { data } = await detectGameServerDir({
      game: props.server.brief?.game || '',
      installDir: draft.value.installDir.trim(),
      configFile: draft.value.configFile,
      listens: detectListens(draft.value),
      serverId: id.value,
    })
    const first = !detected.value || detected.value.installDir !== data.installDir
    detected.value = data
    applyDetection(draft.value, data, first, listenKeys.value)
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    detecting.value = false
  }
}
const switchToHosted = async () => {
  if (!detected.value?.ok) {
    fb.warning(t('gameServers.new.needDetect'))
    return
  }
  switching.value = true
  try {
    await switchGameServerMode({ id: id.value, mode: 'hosted', hosted: toHostedSpec(draft.value) })
    fb.success(t('gameServers.connection.switched'))
    hostedOpen.value = false
    emit('changed')
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    switching.value = false
  }
}
</script>

<style scoped lang="scss">
// 一块白底面板（不描边），里面一行一项简要信息，行之间一条细线。
.cn__list {
  display: flex;
  flex-direction: column;
  padding: 4px 16px;
  border-radius: 16px;
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-sm);
}
.cn__row {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 58px;
  padding: 12px 0;
}
.cn__row + .cn__row {
  border-top: 1px solid var(--el-border-color-lighter, var(--el-border-color-light));
}
.cn__ico {
  width: 32px;
  height: 32px;
  border-radius: 13px;
  font-size: 1rem;
  transform: rotate(-5deg);
}
.cn__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.cn__main b {
  font-size: 0.8125rem;
  font-weight: 600;
}
.cn__media {
  flex: none;
  width: 96px;
  height: 54px;
  border-radius: var(--app-radius-sm);
  object-fit: cover;
  background: var(--el-fill-color-light);
}
.cn__media--icon {
  width: 54px;
}
.cn__main > span {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cn__error {
  font-size: 0.75rem;
  color: var(--m-bad);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cn__cred {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.cn__cred > span {
  flex: 0 0 96px;
  font-size: 0.8125rem;
  font-weight: 600;
}
:deep(.cn__cred-input) {
  flex: 1;
  min-width: 0;
}
.cn__aside {
  flex: none;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  cursor: help;
}

// ---- 弹窗内容 ----
.cn__dialog {
  display: flex;
  flex-direction: column;
  gap: 16px;
  font-size: 0.8125rem;
  line-height: 1.6;
}
.cn__dialog p {
  margin: 0;
}
.cn__dialog > small,
.cn__field small,
.cn__toggle small {
  font-size: 0.75rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.cn__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.cn__field > span {
  font-size: 0.8125rem;
  font-weight: 600;
}
.cn__test {
  display: flex;
}
.cn__seg {
  display: inline-flex;
  flex-wrap: wrap;
  align-self: flex-start;
  padding: 3px;
  gap: 2px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.cn__seg button {
  min-height: 30px;
  padding: 0 16px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
}
.cn__seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.cn__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 16px;
}
.cn__wide {
  grid-column: 1 / -1;
}
.cn__toggle {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 40px;
  cursor: pointer;
}
.cn__toggle + .cn__toggle {
  border-top: 1px solid var(--el-border-color-lighter, var(--el-border-color-light));
}
.cn__toggle > span {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
@media (width <= 768px) {
  .cn__list {
    padding: 4px 16px;
    border-radius: 14px;
  }
  .cn__row {
    flex-wrap: wrap;
  }
  .cn__main {
    flex-basis: calc(100% - 54px);
  }
  .cn__grid {
    grid-template-columns: 1fr;
  }
}
</style>
