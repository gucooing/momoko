<!-- Docker 连接配置（第二期）：标题(引擎状态) → 子导航 → 引擎卡（地址 / 探测结果 / 测试连接 / 8 格引擎信息）→
     左列：连接（启用 / 连接方式 / 地址 / API 版本 / 超时）、TLS（证书文件 PEM 上传 / 跳过校验）、默认（平台 / 任务超时）；
     右列：仓库认证（验证 / 编辑 / 移除 / 验证并添加）、最近连接事件 → 底部深色保存条（N 项修改 · 保存并重新连接）。
     权限：PERM.DOCKER_CONFIG_EDIT。 -->
<template>
  <div class="dc-page">
    <DockerHeader @tasks="tasksVisible = true">
      <template #actions>
        <UButton v-if="canCreate" color="primary" icon="i-lucide-plus" :disabled="!engineReady" @click="router.push(pageTabTo('docker', 'container', { create: '1' }))">
          {{ t('docker.container.createContainer') }}
        </UButton>
      </template>
    </DockerHeader>

    <DockerSubNav />

    <!-- 引擎卡 -->
    <section class="dc-engine" :aria-label="t('docker.config.engineTitle')">
      <div class="dc-engine__head">
        <span class="dc-engine__sticker m-sticker" :class="engineSticker"><UIcon name="i-lucide-container" /></span>
        <div class="dc-engine__id">
          <div class="dc-engine__name">
            <h2>{{ t('docker.config.engineTitle') }}</h2>
            <StatusPill :variant="engineVariant" :label="engineLabel" />
          </div>
          <div class="dc-engine__addr m-mono">{{ savedHost || '—' }}</div>
        </div>
        <div class="dc-engine__probe" :class="`is-${probe.tone}`">
          <span class="dc-engine__probe-t">
            <UIcon :name="probe.icon" />
            {{ probe.title }}
          </span>
          <span v-if="probe.sub" class="dc-engine__probe-s">{{ probe.sub }}</span>
        </div>
        <UButton class="dc-engine__test" color="neutral" variant="outline" icon="i-lucide-plug-zap" :loading="testing" @click="handleTest">
          {{ t('docker.config.testConnection') }}
        </UButton>
      </div>
      <dl v-if="engineGrid.length" class="dc-engine__grid">
        <div v-for="g in engineGrid" :key="g.k" class="dc-engine__cell">
          <dt>{{ g.k }}</dt>
          <dd class="dc-engine__v">{{ g.v }}</dd>
          <dd v-if="g.sub" class="dc-engine__sub">{{ g.sub }}</dd>
        </div>
      </dl>
      <code v-else-if="engineError" class="dc-engine__err m-mono">{{ engineError }}</code>
    </section>

    <div v-if="!canEdit" class="dc-readonly">
      <UIcon name="i-lucide-lock" />
      {{ t('docker.config.readonly') }}
    </div>

    <div class="dc-cols">
      <div class="dc-col">
        <!-- 连接 -->
        <section class="dc-card" aria-labelledby="dc-h-conn">
          <header class="dc-card__head">
            <span class="dc-card__sticker m-sticker m-sticker--p" style="--rot: -4deg"><UIcon name="i-lucide-cable" /></span>
            <div class="dc-card__title">
              <h2 id="dc-h-conn">{{ t('docker.config.connTitle') }}</h2>
              <p>{{ t('docker.config.connDesc') }}</p>
            </div>
          </header>

          <div class="dc-row dc-row--switch">
            <div class="dc-row__l">
              <span class="dc-row__t">{{ t('docker.config.enableTitle') }}</span>
              <span class="dc-row__d">{{ t('docker.config.enableDesc2') }}</span>
            </div>
            <AppSwitch v-model="form.enabled" :disabled="!canEdit" :aria-label="t('docker.config.enableTitle')" />
          </div>

          <div class="dc-row">
            <div class="dc-row__l">
              <span class="dc-row__t">{{ t('docker.config.mode') }}</span>
              <span class="dc-row__d">{{ t('docker.config.modeDesc') }}</span>
            </div>
            <div class="dc-row__c">
              <div class="dc-seg" role="radiogroup" :aria-label="t('docker.config.mode')">
                <button
                  v-for="m in modeOptions"
                  :key="m.value"
                  type="button"
                  role="radio"
                  :aria-checked="mode === m.value"
                  :class="{ 'is-on': mode === m.value }"
                  :disabled="!canEdit"
                  @click="setMode(m.value)"
                >
                  {{ m.label }}
                </button>
              </div>
            </div>
          </div>

          <div class="dc-row">
            <div class="dc-row__l">
              <label class="dc-row__t" for="dc-addr">{{ t('docker.config.address') }}</label>
              <span class="dc-row__d">{{ mode === 'unix' ? t('docker.config.addrUnixHint') : t('docker.config.addrTcpHint') }}</span>
            </div>
            <div class="dc-row__c">
              <span class="dc-field">
                <UIcon :name="mode === 'unix' ? 'i-lucide-plug' : 'i-lucide-server'" class="dc-field__ico" />
                <input
                  id="dc-addr"
                  v-model.trim="form.host"
                  class="m-mono"
                  :placeholder="mode === 'unix' ? UNIX_DEFAULT : 'tcp://192.168.1.10:2376'"
                  :disabled="!canEdit"
                />
              </span>
            </div>
          </div>

          <div class="dc-row">
            <div class="dc-row__l">
              <span class="dc-row__t">{{ t('docker.config.apiVersion') }}</span>
              <span class="dc-row__d">{{ t('docker.config.apiDesc') }}</span>
            </div>
            <div class="dc-row__c">
              <AppCombobox
                v-model="form.apiVersion"
                class="dc-combo"
                :options="API_VERSIONS"
                :placeholder="engineApi ? t('docker.config.apiAutoCur', { v: engineApi }) : t('docker.config.apiAuto')"
                :disabled="!canEdit"
              />
            </div>
          </div>

          <div class="dc-row">
            <div class="dc-row__l">
              <label class="dc-row__t" for="dc-timeout">{{ t('docker.config.timeout') }}</label>
              <span class="dc-row__d">{{ t('docker.config.timeoutDesc') }}</span>
            </div>
            <div class="dc-row__c">
              <span class="dc-unit">
                <input id="dc-timeout" v-model.number="form.requestTimeoutSeconds" type="number" min="1" max="300" inputmode="numeric" :disabled="!canEdit" />
                <span class="dc-unit__u">{{ t('docker.common.seconds') }}</span>
              </span>
            </div>
          </div>
        </section>

        <!-- TLS -->
        <section class="dc-card" aria-labelledby="dc-h-tls">
          <header class="dc-card__head">
            <span class="dc-card__sticker m-sticker m-sticker--a2" style="--rot: 3deg"><UIcon name="i-lucide-shield-check" /></span>
            <div class="dc-card__title">
              <h2 id="dc-h-tls">{{ t('docker.config.tlsTitle') }}</h2>
              <p>{{ t('docker.config.tlsDesc') }}</p>
            </div>
            <AppSwitch v-model="form.tlsEnabled" :disabled="!canEdit || mode === 'unix'" :aria-label="t('docker.config.enableTls')" />
          </header>

          <template v-if="tlsShown">
            <div v-for="f in tlsFiles" :key="f.key" class="dc-row dc-row--file">
              <div class="dc-row__l">
                <span class="dc-row__t">
                  {{ f.label }}
                  <span v-if="f.pending" class="dc-flag dc-flag--a">{{ t('docker.config.pendingSave') }}</span>
                  <span v-else-if="f.expired" class="dc-flag dc-flag--bad">{{ t('docker.config.expired') }}</span>
                </span>
                <span class="dc-row__d m-mono">{{ TLS_FLAGS[f.key] }}</span>
              </div>
              <div class="dc-row__c">
                <div class="dc-file" :class="{ 'is-empty': !f.has, 'is-err': !!f.error }">
                  <span class="dc-file__ico m-sticker" :class="f.has ? (f.error ? 'm-sticker--bad' : 'm-sticker--ok') : 'm-sticker--a'">
                    <UIcon :name="f.has ? 'i-lucide-file-key' : 'i-lucide-upload'" />
                  </span>
                  <span class="dc-file__text">
                    <span class="dc-file__name">{{ f.name }}</span>
                    <span class="dc-file__meta">{{ f.meta }}</span>
                  </span>
                  <UButton
                    size="sm"
                    :color="f.has ? 'neutral' : 'primary'"
                    :variant="f.has ? 'outline' : 'solid'"
                    class="dc-file__btn"
                    :disabled="!canEdit"
                    :aria-label="`${f.has ? t('docker.config.replaceFile') : t('docker.config.pickFile')} ${f.label}`"
                    @click="pickFile(f.key)"
                  >
                    <UIcon v-if="menuStore.isMobile" :name="f.has ? 'i-lucide-replace' : 'i-lucide-upload'" />
                    <span v-else>{{ f.has ? t('docker.config.replaceFile') : t('docker.config.pickFile') }}</span>
                  </UButton>
                  <AppIconButton
                    v-if="f.has"
                    class="is-bad"
                    :label="t('docker.config.removeFile', { name: f.label })"
                    :box="40"
                    :disabled="!canEdit"
                    @click="removeFile(f.key)"
                  >
                    <UIcon name="i-lucide-x" class="dc-ico" />
                  </AppIconButton>
                </div>
              </div>
            </div>
            <div class="dc-row dc-row--switch">
              <div class="dc-row__l">
                <span class="dc-row__t">{{ t('docker.config.skipVerify') }}</span>
                <span class="dc-row__d">{{ t('docker.config.skipVerifyDesc') }}</span>
              </div>
              <AppSwitch v-model="form.tlsSkipVerify" :disabled="!canEdit" :aria-label="t('docker.config.skipVerify')" />
            </div>
          </template>
          <div v-else class="dc-note" :class="{ 'is-warn': mode === 'tcp' }">
            <UIcon :name="mode === 'unix' ? 'i-lucide-info' : 'i-lucide-triangle-alert'" />
            <span>{{ mode === 'unix' ? t('docker.config.unixNoTls') : t('docker.config.tlsOffHint') }}</span>
          </div>
          <input ref="fileInput" type="file" class="dc-hidden" accept=".pem,.crt,.cer,.key,text/plain" @change="onFilePicked" />
        </section>

        <!-- 默认 -->
        <section class="dc-card" aria-labelledby="dc-h-def">
          <header class="dc-card__head">
            <span class="dc-card__sticker m-sticker m-sticker--a" style="--rot: -2deg"><UIcon name="i-lucide-sliders-horizontal" /></span>
            <div class="dc-card__title">
              <h2 id="dc-h-def">{{ t('docker.config.defTitle') }}</h2>
              <p>{{ t('docker.config.defDesc') }}</p>
            </div>
          </header>
          <div class="dc-row">
            <div class="dc-row__l">
              <span class="dc-row__t">{{ t('docker.config.defaultPlatform') }}</span>
              <span class="dc-row__d">{{ t('docker.config.platformDesc') }}</span>
            </div>
            <div class="dc-row__c">
              <AppCombobox
                v-model="form.defaultPlatform"
                class="dc-combo"
                :options="PLATFORMS"
                :placeholder="t('docker.config.platformAuto')"
                :disabled="!canEdit"
              />
            </div>
          </div>
          <div class="dc-row">
            <div class="dc-row__l">
              <label class="dc-row__t" for="dc-task">{{ t('docker.config.taskTimeout') }}</label>
              <span class="dc-row__d">{{ t('docker.config.taskTimeoutDesc') }}</span>
            </div>
            <div class="dc-row__c">
              <span class="dc-unit">
                <input id="dc-task" v-model.number="taskTimeoutMinutes" type="number" min="1" max="1440" inputmode="numeric" :disabled="!canEdit" />
                <span class="dc-unit__u">{{ t('system.settings.unit.minutes') }}</span>
              </span>
            </div>
          </div>
        </section>
      </div>

      <div class="dc-col">
        <!-- 仓库认证 -->
        <section class="dc-card" aria-labelledby="dc-h-reg">
          <header class="dc-card__head">
            <span class="dc-card__sticker m-sticker m-sticker--ok" style="--rot: 4deg"><UIcon name="i-lucide-key-round" /></span>
            <div class="dc-card__title">
              <h2 id="dc-h-reg">
                {{ t('docker.config.regTitle') }}
                <span v-if="form.registryAuths.length" class="dc-count m-num">{{ form.registryAuths.length }}</span>
              </h2>
              <p>{{ t('docker.config.regDesc') }}</p>
            </div>
            <UButton v-if="canEdit && !adding" color="primary" variant="soft" size="sm" icon="i-lucide-plus" class="dc-card__btn" @click="openAdd">
              {{ t('docker.config.regAdd') }}
            </UButton>
          </header>

          <div class="dc-regs">
            <div v-for="(r, idx) in form.registryAuths" :key="`${r.serverAddress}|${r.username}|${idx}`" class="dc-reg">
              <span class="dc-reg__av m-sticker" :class="stickerOf(r.serverAddress)" :style="{ '--rot': ['-4deg', '3deg', '-2deg', '4deg'][idx % 4] }">
                {{ (hostOf(r.serverAddress)[0] || '?').toUpperCase() }}
              </span>
              <span class="dc-reg__main">
                <span class="dc-reg__host m-mono">{{ hostOf(r.serverAddress) || '—' }}</span>
                <span class="dc-reg__sub">
                  {{ t('docker.config.regUser', { name: r.username || '—' }) }}
                  <template v-if="fmtWhen(r.verifiedAt)"> · {{ t('docker.config.regVerifiedAt', { time: fmtWhen(r.verifiedAt) }) }}</template>
                </span>
              </span>
              <StatusPill
                class="dc-reg__pill"
                :variant="r.verifiedAt ? (r.verified ? 'success' : 'error') : 'neutral'"
                :label="r.verifiedAt ? (r.verified ? t('docker.config.regValid') : t('docker.config.regInvalid')) : t('docker.config.regUnverified')"
                :title="r.verifyError || undefined"
              />
              <span class="dc-reg__acts">
                <AppIconButton
                  :label="t('docker.config.regVerify', { host: hostOf(r.serverAddress) })"
                  :box="40"
                  :disabled="!canEdit || !engineReady || verifyingIdx === idx"
                  :title="engineReady ? undefined : t('docker.config.regNeedEngine')"
                  @click="verifyRegistry(idx)"
                >
                  <UIcon :name="verifyingIdx === idx ? 'i-lucide-loader-circle' : 'i-lucide-badge-check'" class="dc-ico" :class="{ 'is-spin': verifyingIdx === idx }" />
                </AppIconButton>
                <AppIconButton class="dc-hide-m" :label="t('docker.config.regEdit', { host: hostOf(r.serverAddress) })" :box="40" :disabled="!canEdit" @click="openEdit(idx)">
                  <UIcon name="i-lucide-pencil" class="dc-ico" />
                </AppIconButton>
                <AppIconButton class="is-bad" :label="t('docker.config.regRemove', { host: hostOf(r.serverAddress) })" :box="40" :disabled="!canEdit" @click="removeRegistry(idx)">
                  <UIcon name="i-lucide-trash-2" class="dc-ico" />
                </AppIconButton>
              </span>
            </div>
            <div v-if="!form.registryAuths.length && !adding" class="dc-empty">{{ t('docker.config.regNone') }}</div>
          </div>

          <div v-if="adding" class="dc-add">
            <div class="dc-add__title">{{ t('docker.config.regAddTitle') }}</div>
            <label class="dc-add__f">
              <span>{{ t('docker.config.registryAddress') }}</span>
              <input v-model.trim="draft.serverAddress" class="app-input m-mono" placeholder="registry.example.com" />
            </label>
            <div class="dc-add__pair">
              <label class="dc-add__f">
                <span>{{ t('docker.config.username') }}</span>
                <input v-model.trim="draft.username" class="app-input" placeholder="robot$ci" autocomplete="off" />
              </label>
              <label class="dc-add__f">
                <span>{{ t('docker.config.passwordToken') }}</span>
                <input v-model="draft.password" class="app-input" type="password" autocomplete="new-password" />
              </label>
            </div>
            <p v-if="draftError" class="dc-add__err">{{ t('docker.config.regVerifyFail', { error: draftError }) }}</p>
            <div class="dc-add__acts">
              <UButton color="neutral" variant="ghost" @click="closeAdd">{{ t('docker.common.cancel') }}</UButton>
              <UButton v-if="draftError || !engineReady" color="neutral" variant="soft" :disabled="!draft.serverAddress" @click="commitDraft(null)">
                {{ t('docker.config.regAddAnyway') }}
              </UButton>
              <UButton color="primary" :loading="draftVerifying" :disabled="!draft.serverAddress || !engineReady" @click="verifyAndAdd">
                {{ t('docker.config.regVerifyAdd') }}
              </UButton>
            </div>
          </div>
        </section>

        <!-- 最近连接事件 -->
        <section class="dc-card dc-card--events" aria-labelledby="dc-h-ev">
          <h2 id="dc-h-ev" class="dc-events__title">{{ t('docker.config.eventsTitle') }}</h2>
          <div v-for="(ev, idx) in events" :key="idx" class="dc-ev">
            <span class="dc-ev__dot" :class="`is-${ev.level}`" />
            <span class="dc-ev__main">
              <span class="dc-ev__t">{{ eventLabel(ev.kind) }}</span>
              <span v-if="ev.detail" class="dc-ev__d">{{ ev.detail }}</span>
            </span>
            <span class="dc-ev__at m-num">{{ fmtWhen(ev.time) }}</span>
          </div>
          <div v-if="!events.length" class="dc-empty">{{ t('docker.config.noEvents') }}</div>
        </section>
      </div>
    </div>

    <!-- 保存条 -->
    <Transition name="savebar">
      <div v-if="changeCount > 0 || saving || justSaved" class="dc-bar" :class="{ 'is-dirty': changeCount > 0 }" role="status">
        <UIcon :name="changeCount > 0 ? 'i-lucide-triangle-alert' : 'i-lucide-circle-check'" class="dc-bar__ico" />
        <span class="dc-bar__text">
          <span class="dc-bar__t">
            {{ changeCount > 0 ? t('docker.config.bar.dirty', { n: changeCount }) : justSaved ? t('docker.config.bar.saved') : t('docker.config.bar.clean') }}
          </span>
          <span class="dc-bar__s">
            {{ changeCount > 0 ? (missingTls ? t('docker.config.bar.subMissingTls', { n: missingTls }) : t('docker.config.bar.subDirty')) : t('docker.config.bar.subClean') }}
          </span>
        </span>
        <button type="button" class="dc-bar__btn dc-bar__btn--ghost" :disabled="!changeCount || saving" @click="discard">
          {{ t('docker.config.bar.discard') }}
        </button>
        <button type="button" class="dc-bar__btn dc-bar__btn--primary" :disabled="!changeCount || saving || !canEdit" @click="save">
          <UIcon :name="saving ? 'i-lucide-loader-circle' : 'i-lucide-refresh-cw'" :class="{ 'is-spin': saving }" />
          <span class="dc-bar__full">{{ t('docker.config.bar.save') }}</span>
          <span class="dc-bar__short">{{ t('docker.config.bar.saveShort') }}</span>
        </button>
      </div>
    </Transition>

    <DockerTaskDialogs v-model="tasksVisible" />

    <!-- 编辑仓库凭据 -->
    <FormDialog v-model="editVisible" :title="t('docker.config.regEditTitle')" :width="460">
      <div class="dc-form">
        <div class="app-field">
          <label class="app-label">{{ t('docker.config.registryAddress') }}</label>
          <input v-model.trim="editDraft.serverAddress" class="app-input m-mono" placeholder="registry.example.com" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.config.username') }}</label>
          <input v-model.trim="editDraft.username" class="app-input" autocomplete="off" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.config.passwordToken') }}</label>
          <input v-model="editDraft.password" class="app-input" type="password" autocomplete="new-password" :placeholder="t('docker.config.regKeepPwd')" />
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :disabled="!editDraft.serverAddress" @click="commitEdit">{{ t('docker.common.save') }}</UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import DockerSubNav from '@/views/docker/components/DockerSubNav.vue'
import DockerHeader from '@/views/docker/components/DockerHeader.vue'
import DockerTaskDialogs from '@/views/docker/components/DockerTaskDialogs.vue'
import { pageTabTo } from '@/config/pageGroups'
import { useDockerStatus } from '@/views/docker/components/useDockerStatus'
import { formatBytes, formatDate, stickerFor } from '@/views/docker/components/format'
import { getDockerConfig, listDockerImages, testDockerConfig, testDockerRegistryAuth, updateDockerConfig } from '@/api/docker'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { getRequestErrorMessage, showRequestError } from '@/utils/request'
import type { DockerConfigInfo, DockerRegistryAuth, DockerStatusResponse, DockerTLSFileMeta } from '@/types/v1/docker'

defineOptions({ name: 'DockerConfigView' })

const { t } = useI18n()
const menuStore = useMenuStore()
const canEdit = useButtonPermission([PERM.DOCKER_CONFIG_EDIT], [])
const canCreate = useButtonPermission([PERM.DOCKER_CONTAINER_MANAGE], [])
const router = useRouter()
const tasksVisible = ref(false)
const TLS_FLAGS: Record<string, string> = { ca: '--tlscacert', cert: '--tlscert', key: '--tlskey' }
/** 镜像总占用（引擎卡「镜像」格的副标题） */
const imagesSize = ref<number | null>(null)
const loadImagesSize = async () => {
  if (!status.value?.connected) return
  try {
    const { data } = await listDockerImages({ all: false, page: 1, pageSize: 1, keyword: '', usage: '', sort: 'created', dangling: undefined, labels: {} })
    imagesSize.value = data?.stats ? Number(data.stats.totalSize) : null
  } catch { imagesSize.value = null }
}
const { status, refresh: refreshStatus } = useDockerStatus()

const UNIX_DEFAULT = 'unix:///var/run/docker.sock'
const API_VERSIONS = ['1.47', '1.46', '1.45', '1.44', '1.43', '1.41']
const PLATFORMS = ['linux/amd64', 'linux/arm64', 'linux/arm/v7']

/* ───────── 时间 ───────── */
const pad = (n: number) => String(n).padStart(2, '0')
const fmtWhen = (v: unknown) => {
  if (!v) return ''
  const d = v instanceof Date ? v : new Date(String(v))
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 1971) return ''
  const now = new Date()
  const hm = `${pad(d.getHours())}:${pad(d.getMinutes())}`
  if (d.toDateString() === now.toDateString()) return hm
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${hm}`
}

/* ───────── 表单 ───────── */
const emptyConfig = (): DockerConfigInfo => ({
  enabled: false,
  host: '',
  tlsEnabled: false,
  tlsCa: '',
  tlsCert: '',
  tlsKey: '',
  tlsKeyClear: false,
  tlsCaMeta: undefined,
  tlsCertMeta: undefined,
  tlsKeyMeta: undefined,
  apiVersion: '',
  requestTimeoutSeconds: 30,
  defaultPlatform: '',
  taskTimeoutSeconds: 1800,
  registryAuths: [],
  tlsSkipVerify: false,
})
const form = reactive<DockerConfigInfo>(emptyConfig())
const saved = ref<DockerConfigInfo>(emptyConfig())
const savedHost = computed(() => saved.value.host || (saved.value.enabled ? UNIX_DEFAULT : ''))
/** 本次在页面里选中的文件名（仅展示） */
const fileNames = reactive<Record<TlsKey, string>>({ ca: '', cert: '', key: '' })

const cloneAuths = (list: DockerRegistryAuth[]) => list.map((a) => ({ ...a }))
const applyConfig = (cfg: DockerConfigInfo) => {
  saved.value = { ...cfg, registryAuths: cloneAuths(cfg.registryAuths || []) }
  Object.assign(form, { ...cfg, tlsKey: '', tlsKeyClear: false, registryAuths: cloneAuths(cfg.registryAuths || []) })
  fileNames.ca = fileNames.cert = fileNames.key = ''
}

const loadConfig = async () => {
  try {
    const { data } = await getDockerConfig()
    if (data?.config) applyConfig(data.config)
  } catch (e) {
    showRequestError(e, t('docker.config.loadConfigFailed'))
  }
}

/* 连接方式：由地址前缀推断 */
type Mode = 'unix' | 'tcp'
const mode = computed<Mode>(() => (!form.host || form.host.startsWith('unix://') || form.host.startsWith('npipe://') ? 'unix' : 'tcp'))
const modeOptions = computed(() => [
  { value: 'unix' as Mode, label: t('docker.config.modeUnix') },
  { value: 'tcp' as Mode, label: t('docker.config.modeTcp') },
])
const lastHost: Record<Mode, string> = { unix: '', tcp: '' }
const setMode = (m: Mode) => {
  if (m === mode.value) return
  lastHost[mode.value] = form.host
  form.host = lastHost[m] || (m === 'unix' ? UNIX_DEFAULT : 'tcp://')
  if (m === 'unix') form.tlsEnabled = false
}

const taskTimeoutMinutes = computed<number>({
  get: () => Math.round((Number(form.taskTimeoutSeconds) || 0) / 60),
  set: (v) => {
    form.taskTimeoutSeconds = Math.max(0, Math.round(Number(v) || 0)) * 60
  },
})

/* ───────── TLS 文件 ───────── */
type TlsKey = 'ca' | 'cert' | 'key'
const tlsShown = computed(() => form.tlsEnabled && mode.value === 'tcp')
const hasFile = (k: TlsKey) => {
  if (k === 'ca') return !!form.tlsCa.trim()
  if (k === 'cert') return !!form.tlsCert.trim()
  return !!form.tlsKey.trim() || (!!saved.value.tlsKeyMeta?.present && !form.tlsKeyClear)
}
const isPending = (k: TlsKey) => {
  if (k === 'ca') return form.tlsCa !== saved.value.tlsCa
  if (k === 'cert') return form.tlsCert !== saved.value.tlsCert
  return !!form.tlsKey || form.tlsKeyClear
}
const savedMeta = (k: TlsKey): DockerTLSFileMeta | undefined =>
  k === 'ca' ? saved.value.tlsCaMeta : k === 'cert' ? saved.value.tlsCertMeta : saved.value.tlsKeyMeta
const tlsFiles = computed(() =>
  (['ca', 'cert', 'key'] as TlsKey[]).map((k) => {
    const label = t(k === 'ca' ? 'docker.config.tlsCa' : k === 'cert' ? 'docker.config.tlsCert' : 'docker.config.tlsKey')
    const has = hasFile(k)
    const pending = isPending(k) && has
    const meta = !pending ? savedMeta(k) : undefined
    const notAfter = meta?.notAfter ? new Date(String(meta.notAfter)) : null
    const expired = !!notAfter && !Number.isNaN(notAfter.getTime()) && notAfter.getTime() < Date.now()
    let name = t('docker.config.noFile')
    let metaText = t('docker.config.fileHint')
    if (has) {
      if (pending) {
        name = fileNames[k] || label
        metaText = t('docker.config.pendingSave')
      } else if (meta?.error) {
        name = label
        metaText = meta.error
      } else {
        name = meta?.subject ? `CN=${meta.subject}` : k === 'key' ? t('docker.config.keyStored') : label
        const parts = [meta?.algorithm, notAfter ? t('docker.config.validTo', { date: formatDate(notAfter) }) : '', meta?.size ? formatBytes(meta.size) : '']
        metaText = parts.filter(Boolean).join(' · ') || '—'
      }
    }
    return { key: k, label, has, pending, expired, error: meta?.error || '', name, meta: metaText }
  }),
)
const missingTls = computed(() => (tlsShown.value ? (['ca', 'cert', 'key'] as TlsKey[]).filter((k) => !hasFile(k)).length : 0))

const fileInput = ref<HTMLInputElement>()
const pickingKey = ref<TlsKey>('ca')
const pickFile = (k: TlsKey) => {
  pickingKey.value = k
  if (fileInput.value) {
    fileInput.value.value = ''
    fileInput.value.click()
  }
}
const onFilePicked = async (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (file.size > 64 * 1024) {
    feedback.error(t('docker.config.fileTooLarge'))
    return
  }
  const text = await file.text()
  if (!text.includes('-----BEGIN')) {
    feedback.error(t('docker.config.fileNotPem'))
    return
  }
  const k = pickingKey.value
  fileNames[k] = file.name
  if (k === 'ca') form.tlsCa = text
  else if (k === 'cert') form.tlsCert = text
  else {
    form.tlsKey = text
    form.tlsKeyClear = false
  }
}
const removeFile = (k: TlsKey) => {
  fileNames[k] = ''
  if (k === 'ca') form.tlsCa = ''
  else if (k === 'cert') form.tlsCert = ''
  else {
    form.tlsKey = ''
    form.tlsKeyClear = !!saved.value.tlsKeyMeta?.present
  }
}

/* ───────── 修改计数 ───────── */
const authSig = (list: DockerRegistryAuth[]) => JSON.stringify(list.map((a) => [a.serverAddress, a.username, a.password, a.token]))
const changeCount = computed(() => {
  const s = saved.value
  let n = 0
  const simple: (keyof DockerConfigInfo)[] = ['enabled', 'host', 'tlsEnabled', 'tlsSkipVerify', 'apiVersion', 'requestTimeoutSeconds', 'defaultPlatform', 'taskTimeoutSeconds', 'tlsCa', 'tlsCert']
  for (const k of simple) if (form[k] !== s[k]) n++
  if (form.tlsKey || form.tlsKeyClear) n++
  if (authSig(form.registryAuths) !== authSig(s.registryAuths)) n++
  return n
})
const justSaved = ref(false)
// 保存成功的提示只停留片刻，之后保存条随之收起
let justSavedTimer: ReturnType<typeof setTimeout> | undefined
watch(justSaved, (v) => {
  clearTimeout(justSavedTimer)
  if (v) justSavedTimer = setTimeout(() => (justSaved.value = false), 2400)
})
onBeforeUnmount(() => clearTimeout(justSavedTimer))
watch(changeCount, (n) => {
  if (n > 0) justSaved.value = false
})

const payload = (): DockerConfigInfo => ({
  ...form,
  tlsCaMeta: undefined,
  tlsCertMeta: undefined,
  tlsKeyMeta: undefined,
  registryAuths: cloneAuths(form.registryAuths),
})

/* ───────── 保存 / 放弃 / 测试 ───────── */
const saving = ref(false)
const save = async () => {
  if (!changeCount.value || saving.value) return
  saving.value = true
  try {
    const { data } = await updateDockerConfig({ config: payload() })
    if (data?.config) applyConfig(data.config)
    else await loadConfig()
    justSaved.value = true
    testResult.value = null
    feedback.success(t('docker.config.saveSuccess'))
    await loadStatus()
  } catch (e) {
    showRequestError(e, t('docker.config.saveFailed'))
  } finally {
    saving.value = false
  }
}
const discard = () => {
  applyConfig(saved.value)
}

const testing = ref(false)
const testResult = ref<DockerStatusResponse | null>(null)
const testAt = ref<Date | null>(null)
const handleTest = async () => {
  testing.value = true
  try {
    const { data } = await testDockerConfig({ config: payload() })
    testResult.value = data?.status || null
    testAt.value = new Date()
    if (data?.status?.connected) feedback.success(t('docker.config.testSuccess'))
    else feedback.error(data?.status?.error || t('docker.common.connectionFailed'))
  } catch (e) {
    showRequestError(e, t('docker.config.testFailed'))
  } finally {
    testing.value = false
    void loadStatus()
  }
}

/* ───────── 引擎卡 ───────── */
const statusAt = ref<Date | null>(null)
const loadStatus = async () => {
  await refreshStatus(true)
  statusAt.value = new Date()
  void loadImagesSize()
}
const engineReady = computed(() => !!status.value?.connected)
const engineApi = computed(() => status.value?.version?.apiVersion || '')
const engineVariant = computed<'success' | 'error' | 'neutral'>(() => (status.value?.connected ? 'success' : status.value?.enabled ? 'error' : 'neutral'))
const engineLabel = computed(() =>
  status.value?.connected ? t('docker.engine.connected') : status.value?.enabled ? t('docker.common.notConnected') : t('docker.config.probeDisabled'),
)
const engineSticker = computed(() => (status.value?.connected ? 'm-sticker--solid-p' : status.value?.enabled ? 'm-sticker--bad' : 'm-sticker--a2'))
const engineError = computed(() => (status.value?.enabled && !status.value.connected ? status.value.error : ''))

const probe = computed(() => {
  // 优先展示刚做的「测试连接」结果（用的是表单里的配置），否则展示当前引擎状态
  const r = testResult.value || status.value
  const at = testResult.value ? testAt.value : statusAt.value
  const sub = at ? t('docker.config.probeAt', { time: fmtWhen(at) }) : ''
  if (!r) return { tone: 'muted', icon: 'i-lucide-clock', title: '—', sub: '' }
  if (r.connected) return { tone: 'ok', icon: 'i-lucide-circle-check', title: t('docker.config.probeOk', { ms: Number(r.latencyMs) || 0 }), sub }
  if (!r.enabled && !testResult.value) return { tone: 'muted', icon: 'i-lucide-power', title: t('docker.config.probeDisabled'), sub: '' }
  return { tone: 'bad', icon: 'i-lucide-circle-x', title: t('docker.config.probeFail'), sub: r.error ? `${sub ? sub + ' · ' : ''}${r.error}` : sub }
})

const engineGrid = computed(() => {
  const s = status.value
  if (!s?.connected || !s.info) return []
  const info = s.info
  const ver = s.version
  return [
    { k: t('docker.config.grid.version'), v: ver?.version || info.serverVersion || '—', sub: ver?.gitCommit ? `git ${ver.gitCommit}` : '' },
    { k: t('docker.config.grid.api'), v: ver?.apiVersion || '—', sub: ver?.minApiVersion ? t('docker.config.grid.apiMin', { v: ver.minApiVersion }) : '' },
    { k: t('docker.config.grid.storage'), v: info.driver || '—', sub: info.cgroupDriver ? `cgroup ${info.cgroupDriver}${info.cgroupVersion ? ' v' + info.cgroupVersion : ''}` : '' },
    { k: t('docker.config.grid.os'), v: info.operatingSystem || info.osType || '—', sub: ver?.kernelVersion ? t('docker.config.grid.kernel', { v: ver.kernelVersion }) : '' },
    { k: t('docker.config.grid.cpu'), v: t('docker.config.grid.cores', { n: info.cpus }), sub: info.architecture },
    { k: t('docker.config.grid.memory'), v: formatBytes(info.memoryTotal), sub: info.name },
    {
      k: t('docker.config.grid.containers'),
      v: String(info.containers),
      sub: t('docker.config.grid.containersSub', { running: info.containersRunning, stopped: info.containersStopped }),
    },
    {
      k: t('docker.config.grid.images'),
      v: String(info.images),
      sub: imagesSize.value != null ? t('docker.cfgx.imagesSize', { size: formatBytes(imagesSize.value), n: s.networks }) : t('docker.config.grid.imagesSub', { n: s.networks }),
    },
  ]
})

const events = computed(() => status.value?.events || [])
const eventLabel = (kind: string) => {
  const key = `docker.config.eventKind.${kind}`
  const label = t(key)
  return label === key ? kind : label
}

/* ───────── 仓库认证 ───────── */
const hostOf = (addr: string) => String(addr || '').replace(/^https?:\/\//, '').replace(/\/+$/, '')
const stickerOf = (addr: string) => stickerFor(hostOf(addr) || 'x')

const blankAuth = (): DockerRegistryAuth => ({ serverAddress: '', username: '', password: '', token: '', verifiedAt: undefined, verified: false, verifyError: '' })
const adding = ref(false)
const draft = reactive<DockerRegistryAuth>(blankAuth())
const draftError = ref('')
const draftVerifying = ref(false)
const openAdd = () => {
  Object.assign(draft, blankAuth())
  draftError.value = ''
  adding.value = true
}
const closeAdd = () => {
  adding.value = false
  draftError.value = ''
}
const commitDraft = (verified: DockerRegistryAuth | null) => {
  form.registryAuths.push({
    ...draft,
    verifiedAt: verified?.verifiedAt,
    verified: !!verified?.verified,
    verifyError: verified?.verifyError || '',
  })
  closeAdd()
}
const verifyAndAdd = async () => {
  if (!draft.serverAddress) return
  draftVerifying.value = true
  draftError.value = ''
  try {
    const { data } = await testDockerRegistryAuth({ auth: { ...draft } })
    const res = data?.auth
    if (res?.verified) {
      feedback.success(t('docker.config.regVerifyOk'))
      commitDraft(res)
    } else {
      draftError.value = res?.verifyError || t('docker.config.regInvalid')
    }
  } catch (e) {
    draftError.value = getRequestErrorMessage(e, t('docker.config.regInvalid'))
  } finally {
    draftVerifying.value = false
  }
}

const verifyingIdx = ref(-1)
const verifyRegistry = async (idx: number) => {
  const item = form.registryAuths[idx]
  if (!item) return
  verifyingIdx.value = idx
  try {
    const { data } = await testDockerRegistryAuth({ auth: { ...item } })
    const res = data?.auth
    if (res) {
      item.verifiedAt = res.verifiedAt
      item.verified = res.verified
      item.verifyError = res.verifyError
      // 已保存的同一凭据由后端同步记下结果，这里同步到快照，避免算作未保存修改
      const s = saved.value.registryAuths.find((a) => a.serverAddress === item.serverAddress && a.username === item.username)
      if (s) Object.assign(s, { verifiedAt: res.verifiedAt, verified: res.verified, verifyError: res.verifyError })
      if (res.verified) feedback.success(t('docker.config.regVerifyOk'))
      else feedback.error(t('docker.config.regVerifyFail', { error: res.verifyError || '—' }))
    }
  } catch (e) {
    showRequestError(e)
  } finally {
    verifyingIdx.value = -1
    void refreshStatus(true)
  }
}
const removeRegistry = (idx: number) => {
  form.registryAuths.splice(idx, 1)
}

const editVisible = ref(false)
const editIdx = ref(-1)
const editDraft = reactive<DockerRegistryAuth>(blankAuth())
const openEdit = (idx: number) => {
  const item = form.registryAuths[idx]
  if (!item) return
  editIdx.value = idx
  Object.assign(editDraft, { ...item })
  editVisible.value = true
}
const commitEdit = () => {
  const item = form.registryAuths[editIdx.value]
  if (!item) return
  const changedIdentity = item.serverAddress !== editDraft.serverAddress || item.username !== editDraft.username
  Object.assign(item, {
    serverAddress: editDraft.serverAddress,
    username: editDraft.username,
    password: editDraft.password,
    token: editDraft.token,
  })
  if (changedIdentity || editDraft.password) Object.assign(item, { verifiedAt: undefined, verified: false, verifyError: '' })
  editVisible.value = false
}

onMounted(() => {
  void loadConfig()
  void loadStatus()
})
</script>

<style scoped lang="scss">
.dc-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.dc-hidden {
  display: none;
}
.dc-ico {
  width: 18px;
  height: 18px;
}
.is-spin {
  animation: dc-spin 0.9s linear infinite;
}
@keyframes dc-spin {
  to {
    transform: rotate(360deg);
  }
}
.dc-readonly {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 600;
}

/* ── 引擎卡 ── */
.dc-engine {
  padding: 22px 26px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.dc-engine__head {
  display: flex;
  align-items: center;
  gap: 16px;
}
.dc-engine__sticker {
  width: 52px;
  height: 52px;
  border-radius: 16px;
  transform: rotate(-5deg);
}
.dc-engine__sticker > * {
  width: 26px;
  height: 26px;
}
.dc-engine__id {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.dc-engine__name {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.dc-engine__name h2 {
  font-family: var(--app-font-display);
  font-size: 1.375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.dc-engine__addr {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.dc-engine__probe {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  min-width: 0;
  max-width: 340px;
  text-align: right;
}
.dc-engine__probe-t {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.dc-engine__probe.is-ok .dc-engine__probe-t {
  color: var(--m-ok);
}
.dc-engine__probe.is-bad .dc-engine__probe-t {
  color: var(--m-bad);
}
.dc-engine__probe-s {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.dc-engine__test {
  min-height: 42px;
  padding-inline: 16px;
  border-radius: 999px;
  font-weight: 700;
  flex: none;
}
.dc-engine__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-top: 18px;
}
.dc-engine__cell {
  min-width: 0;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.dc-engine__cell dt {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.dc-engine__v {
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.dc-engine__sub {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.dc-engine__err {
  display: block;
  margin-top: 14px;
  padding: 8px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-badSoft);
  color: var(--m-bad);
  font-size: 0.75rem;
  word-break: break-all;
}

/* ── 两列 ── */
.dc-cols {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}
.dc-col {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.dc-card {
  padding: 22px 24px 8px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  min-width: 0;
}
.dc-card__head {
  display: flex;
  align-items: center;
  gap: 14px;
  padding-bottom: 16px;
}
.dc-card__sticker {
  width: 42px;
  height: 42px;
  border-radius: 14px;
  transform: rotate(var(--rot, 0deg));
}
.dc-card__sticker > * {
  width: 21px;
  height: 21px;
}
.dc-card__title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.dc-card__title h2 {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-family: var(--app-font-display);
  font-size: 1.1875rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.dc-card__title p {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.dc-card__btn {
  min-height: 36px;
  border-radius: 999px;
  font-weight: 700;
  flex: none;
}
.dc-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 22px;
  padding: 0 7px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
}

.dc-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 24px;
  padding: 16px 0;
  border-top: 1px solid var(--el-border-color-light);
}
.dc-row--switch {
  flex-wrap: nowrap;
  justify-content: space-between;
}
.dc-row__l {
  flex: 1 1 200px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.dc-row__t {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 0.9375rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.dc-row__d {
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.dc-row__c {
  flex: 1 1 280px;
  max-width: 400px;
  min-width: 0;
  display: flex;
  align-items: center;
}
.dc-row--file .dc-row__l {
  flex: 0 1 130px;
}
.dc-row--file .dc-row__c {
  flex: 1 1 320px;
  max-width: none;
}
.dc-row :deep(.app-switch) {
  flex: none;
}

.dc-flag {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 8px;
  font-size: 0.6875rem;
  font-weight: 700;
}
.dc-flag--a {
  background: var(--m-aSoft);
  color: var(--m-aOn);
}
.dc-flag--bad {
  background: var(--m-badSoft);
  color: var(--m-bad);
}

.dc-seg {
  display: inline-flex;
  padding: 4px;
  gap: 2px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.dc-seg button {
  min-height: 36px;
  padding: 0 16px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.dc-seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: 0 1px 3px color-mix(in srgb, var(--m-ink) 14%, transparent);
}
.dc-seg button:disabled {
  cursor: not-allowed;
}

.dc-field {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 44px;
  padding: 0 12px;
  border-radius: 14px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-secondary);
  transition: border-color 0.15s;
}
.dc-field:focus-within {
  border-color: var(--m-p);
  background: var(--el-bg-color);
}
.dc-field__ico {
  width: 17px;
  height: 17px;
  flex: none;
}
.dc-field input {
  flex: 1;
  min-width: 0;
  height: 41px;
  border: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 0.875rem;
  outline: none;
}
.dc-combo {
  width: 100%;
}
.dc-combo :deep(.app-input) {
  height: 44px;
  border-radius: 14px;
}

.dc-unit {
  display: inline-flex;
  align-items: stretch;
  min-height: 44px;
  border-radius: 14px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  overflow: hidden;
}
.dc-unit:focus-within {
  border-color: var(--m-p);
}
.dc-unit input {
  width: 96px;
  padding: 0 12px;
  border: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
  font-variant-numeric: tabular-nums;
  outline: none;
}
.dc-unit__u {
  display: flex;
  align-items: center;
  padding: 0 12px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
}

/* TLS 文件 */
.dc-file {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-width: 0;
  padding: 8px 8px 8px 10px;
  border-radius: 16px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.dc-file.is-empty {
  border-style: dashed;
  background: var(--el-fill-color-lighter);
}
.dc-file.is-err {
  border-color: color-mix(in srgb, var(--m-bad) 45%, transparent);
}
.dc-file__ico {
  width: 36px;
  height: 36px;
  border-radius: 11px;
}
.dc-file__ico > * {
  width: 18px;
  height: 18px;
}
.dc-file__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.dc-file__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.dc-file__meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.dc-file__btn {
  min-height: 36px;
  border-radius: 999px;
  font-weight: 700;
  flex: none;
}
.is-bad {
  color: var(--m-bad);
}
.is-bad:hover:not(:disabled) {
  background: var(--m-badSoft);
}

.dc-note {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin: 0 0 16px;
  padding: 12px 14px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
  line-height: 1.5;
}
.dc-note.is-warn {
  background: var(--m-warnSoft);
}
.dc-note > :first-child {
  width: 17px;
  height: 17px;
  flex: none;
  margin-top: 1px;
}

/* 仓库 */
.dc-regs {
  display: flex;
  flex-direction: column;
}
.dc-reg {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-top: 1px solid var(--el-border-color-light);
}
.dc-reg__av {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  font-family: var(--app-font-display);
  font-size: 1rem;
  transform: rotate(var(--rot, 0deg));
}
.dc-reg__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.dc-reg__host {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.dc-reg__sub {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.dc-reg__pill {
  flex: none;
}
.dc-reg__acts {
  display: flex;
  align-items: center;
  flex: none;
}
.dc-reg__acts :deep(button:disabled) {
  opacity: 0.35;
  cursor: not-allowed;
}
.dc-empty {
  padding: 18px 0 20px;
  border-top: 1px solid var(--el-border-color-light);
  text-align: center;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.dc-add {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin: 4px 0 16px;
  padding: 16px;
  border-radius: 18px;
  background: var(--el-fill-color-light);
}
.dc-add__title {
  font-size: 0.875rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.dc-add__f {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.dc-add__f .app-input {
  height: 42px;
  border-radius: 12px;
  background: var(--el-bg-color);
}
.dc-add__pair {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.dc-add__err {
  font-size: 0.75rem;
  color: var(--m-bad);
  word-break: break-word;
}
.dc-add__acts {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}
.dc-add__acts :deep(button) {
  min-height: 40px;
  border-radius: 999px;
  font-weight: 700;
}

/* 事件 */
.dc-card--events {
  padding-bottom: 12px;
}
.dc-events__title {
  padding-bottom: 8px;
  font-family: var(--app-font-display);
  font-size: 1rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.dc-ev {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 10px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.dc-ev__dot {
  width: 9px;
  height: 9px;
  margin-top: 6px;
  border-radius: 50%;
  background: var(--el-text-color-secondary);
  flex: none;
}
.dc-ev__dot.is-ok {
  background: var(--m-ok);
}
.dc-ev__dot.is-warn {
  background: var(--m-warn);
}
.dc-ev__dot.is-error {
  background: var(--m-bad);
}
.dc-ev__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.dc-ev__t {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.dc-ev__d {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.dc-ev__at {
  flex: none;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

/* 保存条 */
.savebar-enter-active,
.savebar-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.savebar-enter-from,
.savebar-leave-to {
  opacity: 0;
  transform: translateY(16px);
}
.dc-bar {
  position: sticky;
  bottom: 18px;
  z-index: 20;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 10px 10px 18px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: 0 10px 30px color-mix(in srgb, var(--m-ink) 28%, transparent);
}
.dc-bar__ico {
  width: 20px;
  height: 20px;
  flex: none;
  color: var(--m-ok);
}
.dc-bar.is-dirty .dc-bar__ico {
  color: var(--m-a);
}
.dc-bar__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.dc-bar__t {
  font-size: 0.875rem;
  font-weight: 700;
}
.dc-bar__s {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.75rem;
  opacity: 0.7;
}
.dc-bar__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 42px;
  padding: 0 18px;
  border: none;
  border-radius: 999px;
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.dc-bar__btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.dc-bar__btn--ghost {
  background: transparent;
  color: var(--m-bg);
}
.dc-bar__btn--ghost:hover:not(:disabled) {
  background: color-mix(in srgb, var(--m-bg) 12%, transparent);
}
.dc-bar__btn--primary {
  background: var(--m-p);
  color: var(--m-pOn);
}
.dc-bar__short {
  display: none;
}

.dc-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

@media (width <= 1200px) {
  .dc-cols {
    grid-template-columns: minmax(0, 1fr);
  }
  .dc-engine__grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (width <= 900px) {
  .dc-engine__head {
    flex-wrap: wrap;
  }
  .dc-engine__probe {
    order: 3;
    flex: 1 1 100%;
    align-items: flex-start;
    max-width: none;
    text-align: left;
  }
  .dc-engine__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 768px) {
  .dc-page {
    gap: 12px;
  }
  .dc-engine {
    padding: 16px;
    border-radius: var(--app-radius-lg);
  }
  .dc-engine__head {
    gap: 12px;
  }
  .dc-engine__sticker {
    width: 44px;
    height: 44px;
    border-radius: 14px;
  }
  .dc-engine__name h2 {
    font-size: 1.125rem;
  }
  .dc-engine__test {
    order: 4;
    width: 100%;
    justify-content: center;
    min-height: 44px;
  }
  .dc-engine__grid {
    gap: 8px;
    margin-top: 14px;
  }
  .dc-engine__cell {
    padding: 10px 12px;
  }
  .dc-engine__v {
    font-size: 1rem;
  }
  .dc-card {
    padding: 16px 16px 4px;
    border-radius: var(--app-radius-lg);
  }
  .dc-card__title p {
    display: none;
  }
  .dc-card__sticker {
    width: 38px;
    height: 38px;
    border-radius: 12px;
  }
  .dc-row {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
    padding: 14px 0;
  }
  .dc-row--switch {
    flex-direction: row;
    align-items: center;
  }
  .dc-row__l,
  .dc-row--file .dc-row__l {
    flex: none;
  }
  .dc-row--switch .dc-row__l {
    flex: 1 1 auto;
  }
  .dc-row__c,
  .dc-row--file .dc-row__c {
    flex: none;
    max-width: none;
  }
  .dc-seg {
    display: flex;
    width: 100%;
  }
  .dc-seg button {
    flex: 1;
    min-height: 40px;
  }
  .dc-unit {
    width: 100%;
  }
  .dc-unit input {
    flex: 1;
    width: 0;
  }
  .dc-file__btn {
    width: 40px;
    min-height: 40px;
    justify-content: center;
    padding: 0;
  }
  .dc-reg {
    gap: 10px;
  }
  .dc-reg__pill,
  .dc-hide-m {
    display: none;
  }
  .dc-add__pair {
    grid-template-columns: minmax(0, 1fr);
  }
  .dc-bar {
    bottom: calc(78px + env(safe-area-inset-bottom));
    padding: 8px 8px 8px 14px;
    gap: 6px;
    border-radius: 22px;
  }
  .dc-bar__s {
    display: none;
  }
  .dc-bar__btn {
    min-height: 44px;
    padding: 0 14px;
  }
  .dc-bar__full {
    display: none;
  }
  .dc-bar__short {
    display: inline;
  }
}
</style>
