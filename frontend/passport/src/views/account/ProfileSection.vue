<!-- 个人资料：头像（上传 / 用游戏账号的偶像卡）、昵称、用户名、简介、语言与时区。 -->
<template>
  <section v-if="me" class="pp-card pp-section">
    <h1 class="pp-section__title">{{ t('account.profile.title') }}</h1>
    <p class="pp-section__desc">{{ t('account.profile.desc') }}</p>

    <div class="pp-frow">
      <span class="pp-frow__label">{{ t('account.profile.avatar') }}</span>
      <div class="avatar">
        <PAvatar :src="form.avatarAsset ? '' : me.avatar" :asset="form.avatarAsset" :name="form.name || me.username" :size="64" shape="rounded" tilt />
        <span class="pp-hint">
          {{ me.avatarUploadEnabled ? t('account.profile.avatarHint', { mb: maxMb }) : t('account.profile.avatarUploadOff') }}
          <template v-if="assetChoices.length"><br />{{ t('account.profile.avatarAssetHint') }}</template>
        </span>
      </div>
      <div class="pp-frow__actions">
        <PButton v-if="me.avatarUploadEnabled" size="sm" :loading="uploading" @click="pickFile">{{ t('account.profile.upload') }}</PButton>
        <PButton v-if="assetChoices.length" size="sm" variant="soft" @click="assetOpen = true">{{ t('account.profile.pickAsset') }}</PButton>
        <PButton v-if="me.avatar || form.avatarAsset" size="sm" variant="ghost" @click="clearAvatar">{{ t('account.profile.removeAvatar') }}</PButton>
        <input ref="fileInput" type="file" accept="image/png,image/jpeg,image/webp,image/gif" hidden @change="onFile" />
      </div>
    </div>
    <div class="pp-frow">
      <span class="pp-frow__label">{{ t('account.profile.name') }}</span>
      <PField><input v-model.trim="form.name" maxlength="32" :aria-label="t('account.profile.name')" /></PField>
      <span />
    </div>
    <div class="pp-frow">
      <span class="pp-frow__label">{{ t('account.profile.username') }}<small>{{ t('account.profile.usernameHint') }}</small></span>
      <span class="pp-mono username">{{ me.username }}</span>
      <span />
    </div>
    <div class="pp-frow">
      <span class="pp-frow__label">{{ t('account.profile.bio') }}</span>
      <PField><input v-model.trim="form.bio" maxlength="100" :aria-label="t('account.profile.bio')" /></PField>
      <span />
    </div>
    <div class="pp-frow">
      <span class="pp-frow__label">{{ t('account.profile.langTz') }}</span>
      <div class="two">
        <PField>
          <select v-model="form.language" :aria-label="t('account.profile.language')">
            <option value="">{{ t('account.profile.followBrowser') }}</option>
            <option v-for="l in LOCALES" :key="l" :value="l">{{ t(`auth.languages.${l}`) }}</option>
          </select>
        </PField>
        <PField>
          <select v-model="form.timezone" :aria-label="t('account.profile.timezone')">
            <option value="">{{ t('account.profile.followBrowser') }}</option>
            <option v-for="z in timezones" :key="z" :value="z">{{ z }}</option>
          </select>
        </PField>
      </div>
      <span />
    </div>
    <div class="pp-actions">
      <PButton :disabled="!dirty" @click="reset">{{ t('common.cancel') }}</PButton>
      <PButton variant="primary" :loading="saving" :disabled="!dirty" @click="save">{{ t('common.save') }}</PButton>
    </div>

    <PDialog v-model="assetOpen" :title="t('account.profile.pickAsset')" :width="520">
      <p class="pp-hint">{{ t('account.profile.assetDialogHint') }}</p>
      <div class="assets">
        <button
          v-for="c in assetChoices"
          :key="c.asset"
          type="button"
          class="asset"
          :class="{ 'is-on': form.avatarAsset === c.asset }"
          :aria-pressed="form.avatarAsset === c.asset"
          @click="pickAsset(c.asset)"
        >
          <PAvatar :asset="c.asset" :name="c.label" :size="72" shape="rounded" />
          <span class="asset__label">{{ c.label }}</span>
        </button>
      </div>
    </PDialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import PAvatar from '@passport/components/PAvatar.vue'
import PButton from '@passport/components/PButton.vue'
import PDialog from '@passport/components/PDialog.vue'
import PField from '@passport/components/PField.vue'
import * as api from '@passport/api/passport'
import { LOCALES, setLocale } from '@passport/locales'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { errorText } from '@passport/utils/format'
import { gakumas } from '@/games'
import type { PassportLink } from '@/types/v1/passport'

const { t } = useI18n()
const session = useSessionStore()
const ui = useUiStore()

const me = computed(() => session.me)
const form = reactive({ name: '', bio: '', language: '', timezone: '', avatarAsset: '' })
const saving = ref(false)
const uploading = ref(false)
const assetOpen = ref(false)
const links = ref<PassportLink[]>([])
const fileInput = ref<HTMLInputElement | null>(null)

const maxMb = computed(() => Math.round((me.value?.avatarMaxSize || 0) / 1024 / 1024))

const timezones = computed(() => {
  const intl = Intl as unknown as { supportedValuesOf?: (key: string) => string[] }
  const list = intl.supportedValuesOf ? intl.supportedValuesOf('timeZone') : []
  return form.timezone && !list.includes(form.timezone) ? [form.timezone, ...list] : list
})

const reset = () => {
  const m = me.value
  if (!m) return
  Object.assign(form, { name: m.name, bio: m.bio, language: m.language, timezone: m.timezone, avatarAsset: m.avatarAsset })
}
watch(me, reset, { immediate: true })

const dirty = computed(() => {
  const m = me.value
  return !!m && (form.name !== m.name || form.bio !== m.bio || form.language !== m.language || form.timezone !== m.timezone || form.avatarAsset !== m.avatarAsset)
})

// 可选的游戏素材头像：已绑定的学マス 账号所出借的偶像卡。
const assetChoices = computed(() => {
  const out: { asset: string; label: string }[] = []
  const seen = new Set<string>()
  for (const l of links.value) {
    if (l.game !== 'gakumas' || !l.avatarAsset) continue
    const asset = `gakumas/${gakumas.idolCardKey(l.avatarAsset, 'sq')}`
    if (seen.has(asset)) continue
    seen.add(asset)
    out.push({ asset, label: `${l.server?.name || ''} · ${l.nickname}` })
  }
  return out
})

onMounted(async () => {
  try {
    links.value = (await api.listLinks()).items || []
  } catch {
    links.value = []
  }
})

const save = async () => {
  saving.value = true
  try {
    const next = await api.updateProfile({ ...form, clearAvatar: false })
    session.applyMe(next)
    setLocale(next.language)
    ui.toast(t('common.saved'), 'ok')
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    saving.value = false
  }
}

const pickAsset = (asset: string) => {
  form.avatarAsset = asset
  assetOpen.value = false
}

const clearAvatar = async () => {
  try {
    const next = await api.updateProfile({ ...form, avatarAsset: '', clearAvatar: true })
    session.applyMe(next)
    form.avatarAsset = ''
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  }
}

const pickFile = () => fileInput.value?.click()

// 头像直传：申请预签名地址 → 浏览器直接 PUT 到对象存储 → 确认。
const onFile = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (me.value && file.size > me.value.avatarMaxSize) {
    ui.toast(t('account.profile.tooLarge', { mb: maxMb.value }), 'bad')
    return
  }
  uploading.value = true
  try {
    const ticket = await api.presignAvatar(file.type, file.size)
    const res = await fetch(ticket.uploadUrl, { method: 'PUT', body: file, headers: { 'Content-Type': file.type } })
    if (!res.ok) throw new Error(String(res.status))
    session.applyMe(await api.completeAvatar(ticket.token))
    form.avatarAsset = ''
    ui.toast(t('account.profile.avatarDone'), 'ok')
  } catch (err) {
    ui.toast(err instanceof Error && /^\d+$/.test(err.message) ? t('account.profile.uploadFailed') : errorText(err), 'bad')
  } finally {
    uploading.value = false
  }
}
</script>

<style scoped>
.avatar {
  display: flex;
  align-items: center;
  gap: 14px;
}
.username {
  font-weight: 700;
}
.two {
  display: flex;
  gap: 10px;
}
.two > * {
  flex: 1;
}
.assets {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 10px;
  margin-top: 10px;
}
.asset {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 12px 8px;
  border-radius: 18px;
  border: 2px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  cursor: pointer;
}
.asset.is-on {
  border-color: var(--m-p);
  background: var(--m-pSoft);
}
.asset__label {
  font-size: 12px;
  font-weight: 700;
  text-align: center;
  overflow-wrap: anywhere;
}
@media (max-width: 768px) {
  .two {
    flex-direction: column;
  }
}
</style>
