<!-- 资料编辑：桌面嵌左栏；移动由父级 FormDialog 全屏承载（embedded）。
     不含危险区「注销账户」（06a 未要求，且功能未实现）。 -->
<template>
  <AppPanel
    :title="embedded ? undefined : t('user.profileCenter')"
    :title-icon="embedded ? undefined : 'HOutline:UserIcon'"
    :class="{ 'pi-panel--embedded': embedded }"
  >
    <template v-if="!embedded" #actions>
      <UButton color="primary" size="sm" icon="i-lucide-save" :loading="saving" @click="requestSave">
        {{ t('user.saveAllChanges') }}
      </UButton>
    </template>

    <div class="avatar-block">
      <AppAvatar :src="previewAvatar" :name="profileForm.name || profileForm.username" :size="64" />
      <div class="avatar-block__meta">
        <div class="avatar-block__title">{{ t('user.avatarTitle') }}</div>
        <p class="avatar-block__desc">{{ t('user.avatarDescription') }}</p>
        <UButton color="neutral" variant="soft" size="sm" @click="selectAvatarDialogRef?.showDialog()">
          {{ t('user.changeAvatar') }}
        </UButton>
      </div>
    </div>

    <div class="form-divider" />

    <div class="form-stack">
      <div class="app-field">
        <label class="app-label" for="pf-username">{{ t('user.username') }}</label>
        <input
          id="pf-username"
          v-model="profileForm.username"
          class="app-input"
          :placeholder="t('user.loginUsernamePlaceholder')"
          autocomplete="username"
        />
      </div>
      <div class="app-field">
        <label class="app-label" for="pf-name">{{ t('user.name') }}</label>
        <input
          id="pf-name"
          v-model="profileForm.name"
          class="app-input"
          :placeholder="t('user.namePlaceholder')"
          autocomplete="name"
        />
      </div>
      <div class="app-field">
        <label class="app-label" for="pf-email">{{ t('user.email') }}</label>
        <input
          id="pf-email"
          v-model="profileForm.email"
          class="app-input"
          type="email"
          :placeholder="t('user.emailPlaceholder')"
          autocomplete="email"
        />
      </div>
      <div class="app-field">
        <label class="app-label" for="pf-bio">{{ t('user.bioLabel') }}</label>
        <textarea
          id="pf-bio"
          v-model="profileForm.bio"
          class="app-textarea"
          rows="3"
          :placeholder="t('user.bioPlaceholder')"
        />
      </div>
      <div class="app-field">
        <label class="app-label" for="pf-tags">{{ t('user.tagsLabel') }}</label>
        <textarea
          id="pf-tags"
          v-model="profileForm.tags"
          class="app-textarea"
          rows="2"
          :placeholder="t('user.tagsPlaceholder')"
        />
      </div>
    </div>

    <div class="form-divider" />

    <div class="action-row">
      <div class="action-row__icon">
        <component :is="menuStore.iconComponents['HOutline:KeyIcon']" />
      </div>
      <div class="action-row__body">
        <div class="action-row__title">{{ t('user.passwordCardTitle') }}</div>
        <div class="action-row__desc">{{ t('user.passwordCardDescription') }}</div>
      </div>
      <UButton color="primary" variant="soft" size="sm" @click="updatePasswordRef?.showDialog()">
        {{ t('user.changeNow') }}
      </UButton>
    </div>

    <div v-if="embedded" class="pi-embedded-foot">
      <UButton color="primary" block :loading="saving" icon="i-lucide-save" @click="requestSave">
        {{ t('user.saveAllChanges') }}
      </UButton>
    </div>

    <SelectAvatarDialog ref="selectAvatarDialogRef" @get-avatar="userProfileStore.setProfileAvatar" />
    <UpdatePassword ref="updatePasswordRef" />

    <FormDialog
      v-model="saveConfirmOpen"
      :title="t('user.saveProfileTitle')"
      :width="420"
      :confirm-text="t('user.saveProfileConfirm')"
      :cancel-text="t('user.saveProfileCancel')"
      :loading="saving"
      @confirm="doSave"
    >
      <p class="confirm-text">{{ t('user.saveProfileContent') }}</p>
    </FormDialog>
  </AppPanel>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useUserProfileStore } from '@/stores/user/profile'
import { resolveAvatarUrl } from '@/utils/assets'
import { getRequestErrorMessage } from '@/utils/request'
import { useFeedback } from '@/utils/feedback'
import UpdatePassword from '@/components/dialog/UpdatePassword.vue'
import SelectAvatarDialog from '@/components/dialog/SelectAvatarDialog.vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    /** 嵌在 FormDialog 内时隐藏面板外壳动作、显示底栏保存 */
    embedded?: boolean
  }>(),
  { embedded: false },
)
const emit = defineEmits<{ saved: [] }>()

const menuStore = useMenuStore()
const userStore = useUserStore()
const userProfileStore = useUserProfileStore()
const { t } = useI18n()
const fb = useFeedback()
const { profileForm } = storeToRefs(userProfileStore)

const previewAvatar = computed(() => resolveAvatarUrl(profileForm.value.avatar) || '')
const selectAvatarDialogRef = useTemplateRef('selectAvatarDialogRef')
const updatePasswordRef = useTemplateRef('updatePasswordRef')
const saving = ref(false)
const saveConfirmOpen = ref(false)

const requestSave = () => {
  saveConfirmOpen.value = true
}

const doSave = async () => {
  saving.value = true
  try {
    await userStore.updateUserProfile(profileForm.value)
    saveConfirmOpen.value = false
    emit('saved')
  } catch (error) {
    fb.error(getRequestErrorMessage(error, t('user.profileUpdateFailed')))
  } finally {
    saving.value = false
  }
}

watch(
  () => userStore.userInfo,
  (userInfo) => {
    userProfileStore.syncProfileForm(userInfo)
  },
  { immediate: true },
)

defineExpose({ save: requestSave })
void props
</script>

<style scoped lang="scss">
.pi-panel--embedded :deep(.app-panel__head) {
  display: none;
}
.avatar-block {
  display: flex;
  align-items: flex-start;
  gap: 0.85rem;
}
.avatar-block__meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
.avatar-block__title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.avatar-block__desc {
  margin: 0 0 0.25rem;
  font-size: 0.75rem;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
}
.form-divider {
  height: 1px;
  margin: 0.9rem 0;
  background: var(--el-border-color-lighter);
}
.form-stack {
  display: flex;
  flex-direction: column;
  gap: 0.7rem;
}
.action-row {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0.6rem 0.7rem;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
}
.action-row__icon {
  flex-shrink: 0;
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: calc(var(--app-radius) - 2px);
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
  color: var(--el-color-primary);
  :deep(svg) {
    width: 16px;
    height: 16px;
  }
}
.action-row__body {
  flex: 1;
  min-width: 0;
}
.action-row__title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.action-row__desc {
  margin-top: 0.1rem;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}
.pi-embedded-foot {
  margin-top: 1rem;
}
.confirm-text {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.5;
  color: var(--el-text-color-regular);
}
@media (width < 480px) {
  .action-row {
    flex-wrap: wrap;
  }
  .action-row > :last-child {
    width: 100%;
  }
}
</style>
