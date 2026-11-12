<!-- 通行证详情（右侧面板）：资料、停用 / 启用、绑定的游戏账号（按最近使用，超过 5 个折叠）、替他绑定、玩家端能做什么。 -->
<template>
  <section class="pud">
    <div v-if="loading && !detail" class="pud__state">{{ t('passportAdmin.common.loading') }}</div>
    <div v-else-if="error && !detail" class="pud__state">
      <EmptyState icon="HOutline:ExclamationTriangleIcon" :title="t('passportAdmin.detail.loadFailed')" :description="error" />
      <UButton color="neutral" variant="soft" @click="load">{{ t('passportAdmin.common.retry') }}</UButton>
    </div>
    <template v-else-if="detail && user">
      <header class="pud__head">
        <PassportAvatar :avatar="user.avatar" :avatar-asset="user.avatarAsset" :name="user.name || user.username" :size="72" />
        <span class="pud__id">
          <span class="pud__name">{{ user.name || user.username }}</span>
          <span class="pud__sub">@{{ user.username }}</span>
          <span class="pud__sub">
            {{ user.emailMasked || t('passportAdmin.detail.noEmail') }}
            <template v-if="user.emailMasked && !user.emailVerified"> · {{ t('passportAdmin.detail.emailUnverified') }}</template>
          </span>
        </span>
        <StatusPill :variant="statusOf(user).variant" :label="statusOf(user).label" />
      </header>

      <ul class="pud__facts">
        <li>{{ t('passportAdmin.detail.joined', { date: date(user.createTime) }) }}</li>
        <li>{{ user.lastLoginAt ? t('passportAdmin.detail.lastLogin', { time: dateTime(user.lastLoginAt) }) : t('passportAdmin.never') }}</li>
        <li v-if="detail.lastLoginIp || detail.lastLoginDevice" class="m-mono">{{ [detail.lastLoginDevice, detail.lastLoginIp].filter(Boolean).join(' · ') }}</li>
        <li>{{ t('passportAdmin.detail.devices', { n: detail.deviceCount }) }} · {{ t('passportAdmin.detail.apps', { n: detail.appCount }) }}</li>
        <li v-if="user.totpEnabled">{{ t('passportAdmin.detail.totp') }}</li>
        <li v-if="detail.inviteCode" class="m-mono">{{ t('passportAdmin.detail.invite', { code: detail.inviteCode }) }}</li>
        <li v-if="user.deletionScheduledAt" class="pud__warn">{{ t('passportAdmin.detail.deletionAt', { time: dateTime(user.deletionScheduledAt) }) }}</li>
        <li v-if="user.status === 'disabled' && detail.disabledAt" class="pud__bad">
          {{ detail.disabledByName ? t('passportAdmin.detail.disabledBy', { name: detail.disabledByName, time: dateTime(detail.disabledAt) }) : t('passportAdmin.detail.disabledAt', { time: dateTime(detail.disabledAt) }) }}
        </li>
      </ul>

      <div class="pud__actions">
        <UButton
          v-if="user.status === 'disabled'"
          color="success"
          variant="soft"
          icon="i-lucide-circle-check"
          :loading="statusBusy"
          @click="toggleStatus"
        >
          {{ t('passportAdmin.detail.enable') }}
        </UButton>
        <UButton v-else color="error" variant="soft" icon="i-lucide-ban" :loading="statusBusy" @click="toggleStatus">
          {{ t('passportAdmin.detail.disable') }}
        </UButton>
      </div>

      <div class="pud__block">
        <div class="pud__block-head">
          <span class="pud__label">{{ t('passportAdmin.detail.linksTitle', { n: detail.links.length }) }}</span>
          <span v-if="detail.links.length > 1" class="pud__hint">{{ t('passportAdmin.detail.sortHint') }}</span>
        </div>
        <p v-if="!detail.links.length" class="pud__hint">{{ t('passportAdmin.detail.noLinks') }}</p>
        <div v-else class="pud__links">
          <LinkRow v-for="l in shownLinks" :key="l.id" :link="l" @unbind="confirmUnbind" />
        </div>
        <div class="pud__links-foot">
          <UButton color="neutral" variant="outline" icon="i-lucide-plus" @click="bindOpen = true">{{ t('passportAdmin.detail.bindFor') }}</UButton>
          <UButton v-if="detail.links.length > MAX_SHOWN" color="neutral" variant="ghost" @click="allOpen = true">
            {{ t('passportAdmin.detail.viewAll', { n: detail.links.length }) }}
          </UButton>
          <span v-if="detail.links.length > MAX_SHOWN" class="pud__hint">{{ t('passportAdmin.detail.moreHint') }}</span>
        </div>
      </div>

      <div v-if="abilities.length" class="pud__block">
        <span class="pud__label">{{ t('passportAdmin.detail.abilities') }}</span>
        <ul class="pud__abilities">
          <li v-for="a in abilities" :key="a.id">{{ a.text }}</li>
        </ul>
        <span class="pud__hint">{{ t('passportAdmin.detail.abilityHint') }}</span>
      </div>
    </template>

    <BindDialog v-if="detail && user" v-model="bindOpen" :user-id="user.id" :name="user.name || user.username" @bound="onDetail" />
    <LinksDialog v-if="detail && user" v-model="allOpen" :name="user.name || user.username" :links="detail.links" @unbind="confirmUnbind" />
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminUnbindPassportLink, getPassportUser, setPassportUserStatus } from '@/api/passportAdmin'
import type { GetPassportUserResponse, PassportAdminLink, PassportAdminUser } from '@/types/v1/passport_admin'
import { feedback } from '@/utils/feedback'
import { Dialog } from '@/utils/dialog'
import { getRequestErrorMessage } from '@/utils/request'
import PassportAvatar from './PassportAvatar.vue'
import LinkRow from './LinkRow.vue'
import BindDialog from './BindDialog.vue'
import LinksDialog from './LinksDialog.vue'
import { notifyError, usePassportTime } from '../shared'
import { usePassportStatus } from '../status'

const props = defineProps<{ userId: string }>()
const emit = defineEmits<{ changed: [user: PassportAdminUser] }>()

const MAX_SHOWN = 5

const { t } = useI18n()
const { date, dateTime } = usePassportTime()
const statusOf = usePassportStatus()

const detail = ref<GetPassportUserResponse>()
const loading = ref(false)
const error = ref('')
const statusBusy = ref(false)
const bindOpen = ref(false)
const allOpen = ref(false)
let seq = 0

const user = computed(() => detail.value?.user)
const shownLinks = computed(() => (detail.value?.links || []).slice(0, MAX_SHOWN))

const abilities = computed(() =>
  (detail.value?.links || []).map((l) => {
    const server = [l.server?.gameName, l.server?.name].filter(Boolean).join(' ')
    return {
      id: l.id,
      text: l.selfServiceEnabled > 0
        ? t('passportAdmin.detail.ability', { server, n: l.selfServiceEnabled })
        : t('passportAdmin.detail.abilityNone', { server }),
    }
  }),
)

const onDetail = (res: GetPassportUserResponse) => {
  detail.value = res
  if (res.user) emit('changed', res.user)
}

const load = async () => {
  const id = ++seq
  loading.value = true
  error.value = ''
  try {
    const res = await getPassportUser(props.userId)
    if (id === seq) detail.value = res.data
  } catch (e) {
    if (id === seq) error.value = getRequestErrorMessage(e)
  } finally {
    if (id === seq) loading.value = false
  }
}

watch(
  () => props.userId,
  () => {
    detail.value = undefined
    void load()
  },
  { immediate: true },
)

const toggleStatus = () => {
  const u = user.value
  if (!u) return
  const disable = u.status !== 'disabled'
  const name = u.name || u.username
  Dialog.confirm({
    title: disable ? t('passportAdmin.detail.disableTitle') : t('passportAdmin.detail.enableTitle'),
    content: disable ? t('passportAdmin.detail.disableConfirm', { name }) : t('passportAdmin.detail.enableConfirm', { name }),
    confirmText: disable ? t('passportAdmin.detail.disable') : t('passportAdmin.detail.enable'),
    cancelText: t('passportAdmin.common.cancel'),
    onConfirm: async () => {
      statusBusy.value = true
      try {
        const res = await setPassportUserStatus(u.id, disable)
        onDetail(res.data)
        feedback.success(disable ? t('passportAdmin.detail.disabled') : t('passportAdmin.detail.enabled'))
      } catch (e) {
        notifyError(e)
      } finally {
        statusBusy.value = false
      }
    },
  })
}

const confirmUnbind = (l: PassportAdminLink) => {
  const u = user.value
  if (!u) return
  Dialog.confirm({
    title: t('passportAdmin.detail.unbindTitle'),
    content: t('passportAdmin.detail.unbindConfirm', {
      name: u.name || u.username,
      server: l.server?.name || '',
      nickname: l.nickname || t('gm.player.noName'),
    }),
    confirmText: t('passportAdmin.detail.unbindAction'),
    cancelText: t('passportAdmin.common.cancel'),
    onConfirm: async () => {
      try {
        const res = await adminUnbindPassportLink(u.id, l.id)
        onDetail(res.data)
        feedback.success(t('passportAdmin.detail.unbound'))
      } catch (e) {
        notifyError(e)
      }
    },
  })
}

defineExpose({ reload: load })
</script>

<style scoped>
.pud {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.pud__state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 24px 0;
  color: var(--el-text-color-secondary);
}
.pud__head {
  display: flex;
  align-items: center;
  gap: 14px;
}
.pud__id {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.pud__name {
  font-family: var(--app-font-display);
  font-size: 20px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pud__sub,
.pud__hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.pud__hint {
  margin: 0;
}
.pud__facts {
  margin: 0;
  padding: 12px 14px;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 4px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  font-size: 13px;
}
.pud__warn {
  color: var(--m-warn);
  font-weight: 700;
}
.pud__bad {
  color: var(--m-bad);
  font-weight: 700;
}
.pud__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.pud__block {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.pud__block-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.pud__label {
  flex: 1;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.02em;
  color: var(--el-text-color-secondary);
}
.pud__links {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.pud__links-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.pud__abilities {
  margin: 0;
  padding-left: 18px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
  line-height: 1.6;
}
</style>
