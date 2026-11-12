<!-- 托管接入表单：运行节点（本机）、服务端目录（识别结果）、监听地址、资源服务器、开关。
     识别（预检）由父组件触发 detect()；识别出的监听地址、资源服务器与密钥状态回填表单。 -->
<template>
  <div class="hf">
    <div class="hf__grid">
      <div class="hf__field">
        <span>{{ t('gameServers.new.node') }}</span>
        <div class="hf__static">
          <UIcon name="i-lucide-server" />
          {{ hostName ? t('gameServers.common.localHost', { host: hostName }) : t('gameServers.common.local') }}
        </div>
        <small>{{ t('gameServers.new.nodeHint') }}</small>
      </div>
    </div>
    <label class="hf__field">
      <span>{{ t('gameServers.new.dir') }}</span>
      <div class="hf__row">
        <UInput v-model="model.installDir" class="hf__grow m-mono" icon="i-lucide-folder" :placeholder="t('gameServers.new.dirPh')" @keyup.enter="emit('detect')" />
        <UButton color="neutral" variant="soft" :loading="detecting" @click="emit('detect')">{{ t('gameServers.new.detect') }}</UButton>
      </div>
      <small v-if="dirHint">{{ dirHint }}</small>
    </label>
    <div class="hf__field">
      <span>{{ t('gameServers.new.listens') }}</span>
      <div class="hf__grid hf__grid--3">
        <label v-for="k in ['grpc', 'http', 'mgp']" :key="k" class="hf__sub">
          <small>{{ t(`gameServers.role.${k}`) }}</small>
          <UInput v-model="model.listens[k]" class="m-mono" :placeholder="detected?.listens?.[k] || 'host:port'" />
        </label>
      </div>
    </div>
    <template v-if="model.resourceServer">
      <div class="hf__grid hf__grid--3">
        <label class="hf__sub">
          <small>{{ t('gameServers.role.resource') }}</small>
          <UInput v-model="model.resourceListen" class="m-mono" :placeholder="detected?.listens?.resource || 'host:port'" />
        </label>
        <label class="hf__sub hf__sub--wide">
          <small>{{ t('gameServers.new.resourceOrigin') }}</small>
          <UInput v-model="model.resourceOrigin" class="m-mono" placeholder="http://host:port" />
        </label>
      </div>
      <small class="hf__hint">{{ t('gameServers.new.resourceOriginHint') }}</small>
      <label class="hf__sub">
        <small>{{ t('gameServers.new.resourceRoot') }}</small>
        <UInput v-model="model.resourceRoot" class="m-mono" />
        <small class="hf__hint">{{ t('gameServers.new.resourceRootHint') }}</small>
      </label>
    </template>
    <div class="hf__field">
      <span>{{ t('gameServers.key.title') }}</span>
      <MgpKeyGenerator
        v-model="model.key"
        :current-key-id="detected?.mgpKeyPresent ? detected.mgpKeyId : ''"
        :hint="t('gameServers.key.hostedHint')"
        clearable
      />
      <small>{{ keyHint }}</small>
    </div>
    <div class="hf__toggles">
      <label class="hf__toggle" :class="{ 'is-disabled': !canResource }">
        <span>
          <b>{{ t('gameServers.new.tgResource') }}</b>
          <small>{{ canResource ? t('gameServers.new.tgResourceDesc') : t('gameServers.new.tgResourceMissing') }}</small>
        </span>
        <AppSwitch v-model="model.resourceServer" :disabled="!canResource" :aria-label="t('gameServers.new.tgResource')" />
      </label>
      <label class="hf__toggle">
        <span>
          <b>{{ t('gameServers.new.tgAutoStart') }}</b>
          <small>{{ t('gameServers.new.tgAutoStartDesc') }}</small>
        </span>
        <AppSwitch v-model="model.autoStart" :aria-label="t('gameServers.new.tgAutoStart')" />
      </label>
      <label class="hf__toggle">
        <span>
          <b>{{ t('gameServers.new.tgAutoRestart') }}</b>
          <small>{{ t('gameServers.new.tgAutoRestartDesc') }}</small>
        </span>
        <AppSwitch v-model="model.autoRestart" :aria-label="t('gameServers.new.tgAutoRestart')" />
      </label>
      <label class="hf__toggle">
        <span>
          <b>{{ t('gameServers.new.tgStart') }}</b>
          <small>{{ t('gameServers.new.tgStartDesc') }}</small>
        </span>
        <AppSwitch v-model="model.start" :aria-label="t('gameServers.new.tgStart')" />
      </label>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DetectGameServerDirResponse } from '@/types/v1/game_server'
import type { HostedDraft } from '../hosted'
import MgpKeyGenerator from './MgpKeyGenerator.vue'

const model = defineModel<HostedDraft>({ required: true })
const props = defineProps<{ detected: DetectGameServerDirResponse | null; detecting?: boolean }>()
const emit = defineEmits<{ detect: [] }>()
const { t } = useI18n()

const hostName = computed(() => props.detected?.hostName || '')
const canResource = computed(() => !!props.detected?.hasResourceServer)
const dirHint = computed(() => {
  const d = props.detected
  if (!d) return ''
  const build = d.build
  const parts: string[] = []
  if (build?.valid) parts.push(build.path.split(/[\\/]/).pop() || '')
  if (d.configFile) parts.push(d.configExists ? d.configFile : `${d.configTemplate} → ${d.configFile}`)
  if (d.hasResourceServer) parts.push('resource-server')
  return parts.filter(Boolean).join(' · ')
})
const keyHint = computed(() => {
  const d = props.detected
  if (model.value.key) return t('gameServers.new.tgKeyDesc')
  if (d?.mgpKeyPresent) return t('gameServers.new.tgKeyPresent', { keyId: d.mgpKeyId })
  if (d) return t('gameServers.new.tgKeyNone')
  return t('gameServers.new.tgKeyDesc')
})
</script>

<style scoped lang="scss">
.hf {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.hf__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 16px;
}
.hf__grid--3 {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.hf__field,
.hf__sub {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.hf__sub {
  gap: 6px;
}
.hf__sub--wide {
  grid-column: span 2;
}
.hf__field > span {
  font-size: 0.8125rem;
  font-weight: 700;
}
.hf__field small,
.hf__sub small,
.hf__hint {
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.hf__row {
  display: flex;
  gap: 8px;
}
.hf__grow {
  flex: 1;
  min-width: 0;
}
.hf__static {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  font-size: 0.875rem;
  font-weight: 700;
}
.hf__toggles {
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--el-border-color-light);
}
.hf__toggle {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 62px;
  border-bottom: 1px solid var(--el-border-color-light);
  cursor: pointer;
}
.hf__toggle.is-disabled {
  cursor: not-allowed;
  opacity: 0.7;
}
.hf__toggle > span {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 0.875rem;
}
.hf__toggle small {
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
@media (width <= 768px) {
  .hf__grid,
  .hf__grid--3 {
    grid-template-columns: 1fr;
  }
  .hf__sub--wide {
    grid-column: auto;
  }
  .hf__toggle {
    min-height: 64px;
  }
}
</style>
