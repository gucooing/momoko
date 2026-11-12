<!-- 接入设置：接入方式（托管 ↔ 仅 GM 双向切换）、MGP 接口（仅 GM 可改地址与密钥并测试；托管来自服务端配置）、
     用户通行证可见性与邀请码、时区、托管选项（自启动、崩溃重启、资源服务器）。 -->
<template>
  <div class="cn">
    <section class="cn__card">
      <div class="cn__head">
        <h2>{{ t('gameServers.connection.mode') }}</h2>
        <span class="cn__tag" :class="hosted ? 'is-hosted' : 'is-remote'">{{ t(`gameServers.mode.${mode}`) }}</span>
        <span class="cn__spacer" />
        <UButton
          v-if="hosted && canRemote"
          size="sm"
          color="neutral"
          variant="soft"
          icon="i-lucide-cloud"
          @click="openRemote"
        >
          {{ t('gameServers.connection.switchToRemote') }}
        </UButton>
        <UButton v-if="!hosted && canHosted" size="sm" color="neutral" variant="soft" icon="i-lucide-cpu" @click="openHosted">
          {{ t('gameServers.connection.switchToHosted') }}
        </UButton>
      </div>
      <dl class="cn__dl">
        <div class="is-wide">
          <dt>{{ t('gameServers.connection.endpoint') }}</dt>
          <dd class="m-mono">{{ server.mgpEndpoint || t('gameServers.list.mgpNone') }}</dd>
        </div>
        <div>
          <dt>{{ t('gameServers.connection.keyId') }}</dt>
          <dd class="m-mono">{{ server.keyId || '—' }}</dd>
        </div>
        <div>
          <dt>{{ t('gameServers.connection.manifest') }}</dt>
          <dd class="m-num">
            {{ server.protocol ? t('gameServers.connection.manifestValue', { protocol: server.protocol, views: server.viewCount, commands: server.commandCount, configs: server.configCount }) : '—' }}
          </dd>
        </div>
      </dl>
      <p class="cn__hint">
        <template v-if="server.keyRotatedAt">{{ t('gameServers.connection.rotatedAt', { t: fmtTime(server.keyRotatedAt, true) }) }} · </template>
        <template v-if="server.probeAt">{{ t('gameServers.connection.probeAt', { t: fmtTime(server.probeAt, true) }) }}</template>
      </p>
      <p v-if="server.probeError && !server.brief?.mgpOk" class="cn__error">{{ t('gameServers.connection.probeError', { error: server.probeError }) }}</p>
      <p v-if="hosted" class="cn__hint">{{ t('gameServers.connection.hostedMgp') }}</p>
    </section>

    <section v-if="!hosted" class="cn__card">
      <h2>{{ t('gameServers.connection.mgp') }}</h2>
      <label class="cn__field">
        <span>{{ t('gameServers.connection.endpoint') }}</span>
        <UInput v-model="mgp.endpoint" class="m-mono" icon="i-lucide-globe" />
      </label>
      <div class="cn__field">
        <span>{{ t('gameServers.key.title') }}</span>
        <MgpKeyGenerator v-model="mgpKey" :current-key-id="server.keyId" :hint="t('gameServers.key.remoteHint')" clearable />
      </div>
      <CheckList v-if="checks.length" :checks="checks" />
      <div class="cn__actions">
        <UButton color="neutral" variant="soft" icon="i-lucide-zap" :loading="testing" @click="test">{{ t('gameServers.connection.test') }}</UButton>
        <UButton color="primary" :loading="saving" @click="save">{{ t('gameServers.connection.save') }}</UButton>
      </div>
    </section>

    <section class="cn__card">
      <h2>{{ t('gameServers.connection.passport') }}</h2>
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
      <div class="cn__actions">
        <UButton color="primary" :loading="saving" @click="save">{{ t('gameServers.connection.save') }}</UButton>
      </div>
    </section>

    <section v-if="hosted && server.hosting" class="cn__card">
      <h2>{{ t('gameServers.connection.hosting') }}</h2>
      <div class="cn__procs">
        <div v-for="p in server.processes" :key="p.role" class="cn__proc">
          <b>{{ t(`gameServers.role.${p.role}`, p.role) }}</b>
          <small>{{ p.instanceName || t('game.unnamed') }}</small>
          <small v-if="p.ownerName">{{ t('gameServers.detail.ownerHint', { owner: p.ownerName }) }}</small>
          <small v-for="(v, k) in p.listens" :key="k" class="m-mono">{{ t(`gameServers.role.${k}`, String(k)) }} · {{ v }}</small>
        </div>
      </div>
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
      <label class="cn__toggle">
        <span>
          <b>{{ t('gameServers.new.tgResource') }}</b>
          <small>{{ t('gameServers.new.tgResourceDesc') }}</small>
        </span>
        <AppSwitch v-model="host.resourceServer" :aria-label="t('gameServers.new.tgResource')" />
      </label>
      <div v-if="host.resourceServer" class="cn__grid">
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
      <div class="cn__actions">
        <UButton color="primary" :loading="savingHost" @click="saveHosting">{{ t('gameServers.connection.updateHosting') }}</UButton>
      </div>
    </section>

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
        <label class="cn__toggle">
          <span>
            <b>{{ t('gameServers.connection.deleteInstances') }}</b>
            <small>{{ t('gameServers.connection.deleteInstancesDesc') }}</small>
          </span>
          <AppSwitch v-model="sw.deleteInstances" :aria-label="t('gameServers.connection.deleteInstances')" />
        </label>
      </div>
    </FormDialog>

    <FormDialog v-model="hostedOpen" :title="t('gameServers.connection.switchHostedTitle')" :width="760" :loading="switching" @confirm="switchToHosted">
      <div class="cn__dialog">
        <HostedForm v-model="draft" :detected="detected" :detecting="detecting" @detect="detect" />
        <CheckList v-if="detected?.checks.length" :checks="detected.checks" />
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { detectGameServerDir, switchGameServerMode, testGameServer, updateGameServer, updateGameServerHosting } from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import type { DetectGameServerDirResponse, GameAdapter, GameServerCheck, GameServerInfo } from '@/types/v1/game_server'
import CheckList from '../../components/CheckList.vue'
import HostedForm from '../../components/HostedForm.vue'
import { applyDetection, detectListens, emptyHostedDraft, toHostedSpec } from '../../hosted'
import { fmtTime } from '../../shared'
import type { MgpKey } from '../../mgpKey'
import MgpKeyGenerator from '../../components/MgpKeyGenerator.vue'

const props = defineProps<{ server: GameServerInfo; adapter?: GameAdapter | null }>()
const emit = defineEmits<{ changed: [] }>()
const { t } = useI18n()
const fb = useFeedback()

const VISIBILITY = ['public', 'invite', 'hidden'] as const
type Visibility = (typeof VISIBILITY)[number]

const id = computed(() => props.server.brief?.id || '')
const mode = computed(() => props.server.brief?.mode || 'remote')
const hosted = computed(() => mode.value === 'hosted')
const canHosted = computed(() => !!props.adapter?.modes.includes('hosted'))
const canRemote = computed(() => !!props.adapter?.modes.includes('remote'))

const mgp = reactive({ endpoint: '' })
// 新的 MGP 密钥（点「生成」得到）；为空时沿用当前密钥
const mgpKey = ref<MgpKey | null>(null)
const form = reactive<{ visibility: Visibility; inviteCode: string; timezone: string }>({ visibility: 'public', inviteCode: '', timezone: '' })
const host = reactive({ autoStart: true, autoRestart: true, resourceServer: false, resourceListen: '', resourceOrigin: '', resourceRoot: '' })
const checks = ref<GameServerCheck[]>([])
const testing = ref(false)
const saving = ref(false)
const savingHost = ref(false)

const sync = () => {
  const s = props.server
  mgp.endpoint = s.mgpEndpoint
  mgpKey.value = null
  form.visibility = (VISIBILITY as readonly string[]).includes(s.passportVisibility) ? (s.passportVisibility as Visibility) : 'public'
  form.inviteCode = s.passportInviteCode
  form.timezone = s.timezone
  if (s.hosting) {
    Object.assign(host, {
      autoStart: s.hosting.autoStart,
      autoRestart: s.hosting.autoRestart,
      resourceServer: s.hosting.resourceServer,
      resourceListen: s.hosting.resourceListen,
      resourceOrigin: s.hosting.resourceOrigin,
      resourceRoot: s.hosting.resourceRoot,
    })
  }
}
watch(() => props.server.updateTime, sync, { immediate: true })

const timezones = computed(() => {
  const fn = (Intl as unknown as { supportedValuesOf?: (k: string) => string[] }).supportedValuesOf
  return fn ? fn('timeZone') : []
})

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

const save = async () => {
  const s = props.server
  saving.value = true
  try {
    await updateGameServer({
      id: id.value,
      name: s.brief?.name || '',
      tags: s.brief?.tags || [],
      cover: s.brief?.cover || '',
      remark: s.remark,
      mgpEndpoint: hosted.value ? s.mgpEndpoint : mgp.endpoint.trim(),
      keyId: hosted.value ? s.keyId : mgpKey.value?.keyId || s.keyId,
      secret: hosted.value ? '' : mgpKey.value?.secret || '',
      passportVisibility: form.visibility,
      passportInviteCode: form.inviteCode.trim(),
      timezone: form.timezone.trim(),
    })
    fb.success(t('gameServers.common.saved'))
    emit('changed')
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    saving.value = false
  }
}

const saveHosting = async () => {
  savingHost.value = true
  try {
    await updateGameServerHosting({ id: id.value, ...host })
    fb.success(t('gameServers.common.saved'))
    emit('changed')
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    savingHost.value = false
  }
}

// ---- 切换接入方式 ----
const switching = ref(false)
const remoteOpen = ref(false)
const sw = reactive({ keep: true, endpoint: '', deleteInstances: false })
const swKey = ref<MgpKey | null>(null)
const openRemote = () => {
  Object.assign(sw, { keep: !!props.server.mgpEndpoint, endpoint: '', deleteInstances: false })
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
      deleteInstances: sw.deleteInstances,
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
    applyDetection(draft.value, data, first)
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
.cn {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
  gap: 16px;
  align-items: start;
}
.cn__card {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 20px 22px;
  border-radius: var(--app-radius-lg);
  border: var(--cmp-card-border-width, 1px) solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.cn__card h2 {
  margin: 0;
  font-size: 1.0625rem;
  font-weight: 800;
}
.cn__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.cn__spacer {
  flex: 1;
}
.cn__tag {
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 10px;
  border-radius: 8px;
  font-size: 0.75rem;
  font-weight: 700;
}
.cn__tag.is-hosted {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.cn__tag.is-remote {
  background: var(--m-a2Soft);
  color: var(--m-ink);
}
.cn__dl {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 14px;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.cn__dl div {
  min-width: 0;
}
.cn__dl .is-wide {
  grid-column: 1 / -1;
}
.cn__dl dt {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.cn__dl dd {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 700;
  overflow-wrap: anywhere;
}
.cn__hint,
.cn__field small,
.cn__toggle small,
.cn__proc small {
  margin: 0;
  font-size: 0.75rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.cn__error {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--m-bad);
  overflow-wrap: anywhere;
}
.cn__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 16px;
}
.cn__wide {
  grid-column: 1 / -1;
}
.cn__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.cn__field > span {
  font-size: 0.8125rem;
  font-weight: 700;
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
  min-height: 36px;
  padding: 0 16px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.cn__seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.cn__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}
.cn__procs {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 10px;
}
.cn__proc {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  font-size: 0.875rem;
}
.cn__proc small {
  overflow-wrap: anywhere;
}
.cn__toggle {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 56px;
  border-top: 1px solid var(--el-border-color-light);
  cursor: pointer;
}
.cn__toggle > span {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 0.875rem;
}
.cn__dialog {
  display: flex;
  flex-direction: column;
  gap: 14px;
  font-size: 0.875rem;
  line-height: 1.6;
}
.cn__dialog p {
  margin: 0;
}
@media (width <= 768px) {
  .cn {
    grid-template-columns: 1fr;
  }
  .cn__grid,
  .cn__dl {
    grid-template-columns: 1fr;
  }
  .cn__card {
    padding: 16px;
  }
}
</style>
