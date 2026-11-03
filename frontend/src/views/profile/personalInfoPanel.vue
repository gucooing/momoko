<!-- 个人资料表单：PC 直接作为卡片内联编辑（两列）；embedded 时放进 FormDialog（单列，手机端用）。 -->
<template>
  <component :is="embedded ? 'div' : AppPanel" class="pi" :class="{ 'pi--embedded': embedded }">
    <template v-if="!embedded" #header>
      <div class="pi-head">
        <h2 class="pi-head__title">{{ t('profile.info.title') }}</h2>
        <p class="pi-head__desc">{{ t('profile.info.desc') }}</p>
      </div>
    </template>

    <form class="pi-grid" @submit.prevent="save">
      <div class="app-field">
        <label class="app-label" for="pf-name">{{ t('user.nickname') }}</label>
        <div class="pi-input">
          <UIcon name="i-lucide-user-round" class="pi-input__ico" aria-hidden="true" />
          <input
            id="pf-name"
            v-model="profileForm.name"
            class="app-input"
            :placeholder="t('user.namePlaceholder')"
            autocomplete="nickname"
          />
        </div>
      </div>
      <div class="app-field">
        <label class="app-label" for="pf-username">{{ t('user.username') }}</label>
        <div class="pi-input">
          <UIcon name="i-lucide-at-sign" class="pi-input__ico" aria-hidden="true" />
          <input
            id="pf-username"
            v-model="profileForm.username"
            class="app-input"
            :placeholder="t('user.loginUsernamePlaceholder')"
            autocomplete="username"
            aria-describedby="pf-username-hint"
          />
        </div>
        <span id="pf-username-hint" class="pi-hint">{{ t('profile.info.usernameHint') }}</span>
      </div>
      <div class="app-field">
        <label class="app-label" for="pf-email">{{ t('user.email') }}</label>
        <div class="pi-input">
          <UIcon name="i-lucide-mail" class="pi-input__ico" aria-hidden="true" />
          <input
            id="pf-email"
            v-model="profileForm.email"
            class="app-input"
            type="email"
            :placeholder="t('user.emailPlaceholder')"
            autocomplete="email"
          />
        </div>
      </div>
      <div class="app-field">
        <label class="app-label" for="pf-phone">{{ t('profile.info.phone') }}</label>
        <div class="pi-input">
          <UIcon name="i-lucide-phone" class="pi-input__ico" aria-hidden="true" />
          <input
            id="pf-phone"
            v-model="profileForm.phone"
            class="app-input"
            type="tel"
            :placeholder="t('profile.info.optional')"
            autocomplete="tel"
          />
        </div>
      </div>
      <div class="app-field pi-grid__wide">
        <label class="app-label" for="pf-bio">{{ t('user.bioLabel') }}</label>
        <textarea
          id="pf-bio"
          v-model="profileForm.bio"
          class="app-textarea"
          rows="2"
          :placeholder="t('user.bioPlaceholder')"
        />
      </div>
      <div class="app-field pi-grid__wide">
        <label class="app-label" for="pf-tags">{{ t('user.tagsLabel') }}</label>
        <input
          id="pf-tags"
          v-model="profileForm.tags"
          class="app-input"
          :placeholder="t('user.tagsPlaceholder')"
        />
      </div>
      <button type="submit" hidden />
    </form>

    <div class="pi-foot">
      <span class="pi-foot__state" :class="{ 'is-dirty': dirty }">
        {{ dirty ? t('profile.info.dirty') : t('profile.info.saved') }}
      </span>
      <UButton color="neutral" variant="soft" :disabled="!dirty || saving" @click="reset">
        {{ t('profile.info.revert') }}
      </UButton>
      <UButton color="primary" icon="i-lucide-check" :loading="saving" :disabled="!dirty" @click="save">
        {{ t('profile.info.save') }}
      </UButton>
    </div>
  </component>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import AppPanel from '@/components/ui/AppPanel.vue'
import { useUserProfileStore } from '@/stores/user/profile'
import { getRequestErrorMessage } from '@/utils/request'
import { useFeedback } from '@/utils/feedback'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    /** 嵌在 FormDialog 内：单列、无卡片外壳 */
    embedded?: boolean
  }>(),
  { embedded: false },
)
const emit = defineEmits<{ saved: [] }>()

const userStore = useUserStore()
const userProfileStore = useUserProfileStore()
const { t } = useI18n()
const fb = useFeedback()
const { profileForm } = storeToRefs(userProfileStore)
const saving = ref(false)

const normBio = (bio?: string) => (bio && bio !== t('user.defaultBio') ? bio : '')

const reset = () => {
  userProfileStore.syncProfileForm(userStore.userInfo)
  profileForm.value.bio = normBio(profileForm.value.bio)
}

const dirty = computed(() => {
  const u = userStore.userInfo
  const f = profileForm.value
  if (!u) return false
  return (
    (f.name || '') !== (u.name || '') ||
    (f.username || '') !== (u.username || '') ||
    (f.email || '') !== (u.email || '') ||
    (f.phone || '') !== (u.phone || '') ||
    (f.bio || '') !== normBio(u.bio) ||
    (f.tags || '') !== (u.tags || '')
  )
})

const save = async () => {
  if (!dirty.value || saving.value) return
  saving.value = true
  try {
    await userStore.updateUserProfile({ ...profileForm.value, avatar: userStore.userInfo?.avatar || '' })
    reset()
    emit('saved')
  } catch (error) {
    fb.error(getRequestErrorMessage(error, t('user.profileUpdateFailed')))
  } finally {
    saving.value = false
  }
}

watch(() => userStore.userInfo, reset, { immediate: true })

defineExpose({ save })
void props
</script>

<style scoped lang="scss">
.pi-head {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.pi-head__title {
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.pi-head__desc {
  margin: 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.pi-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px 18px;
}
.pi-grid__wide {
  grid-column: 1 / -1;
}
.pi--embedded .pi-grid {
  grid-template-columns: minmax(0, 1fr);
}
.pi-input {
  position: relative;
  display: flex;
  align-items: center;
  .app-input {
    width: 100%;
    padding-left: 38px;
  }
}
.pi-input__ico {
  position: absolute;
  left: 13px;
  width: 17px;
  height: 17px;
  color: var(--el-text-color-secondary);
  pointer-events: none;
}
.pi-hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.app-textarea {
  width: 100%;
  resize: vertical;
}
.pi-foot {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 20px;
}
.pi-foot__state {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.pi-foot__state.is-dirty {
  color: var(--m-warn);
}
@media (width < 640px) {
  .pi-grid {
    grid-template-columns: minmax(0, 1fr);
  }
  .pi-foot {
    flex-wrap: wrap;
  }
  .pi-foot__state {
    flex-basis: 100%;
  }
  .pi-foot :deep(button) {
    flex: 1;
    justify-content: center;
    min-height: 44px;
  }
}
</style>
