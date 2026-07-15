<!-- 个人资料中心：令牌表单 + 改密入口 + 危险区提示（注销未实现） -->
<template>
  <AppPanel :title="t('user.profileCenter')" title-icon="HOutline:UserIcon">
    <template #actions>
      <UButton color="primary" size="sm" icon="i-lucide-save" :loading="saving" @click="save">
        {{ t('user.saveAllChanges') }}
      </UButton>
    </template>

    <!-- 头像 -->
    <div class="avatar-block">
      <AppAvatar :src="previewAvatar" :name="profileForm.name || profileForm.username" :size="72" />
      <div class="avatar-block__meta">
        <div class="avatar-block__title">{{ t('user.avatarTitle') }}</div>
        <p class="avatar-block__desc">{{ t('user.avatarDescription') }}</p>
        <UButton color="neutral" variant="soft" size="sm" @click="selectAvatarDialogRef?.showDialog()">
          {{ t('user.changeAvatar') }}
        </UButton>
      </div>
    </div>

    <div class="form-divider" />

    <!-- 字段 -->
    <div class="form-grid">
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
      <div class="app-field form-grid__full">
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
      <div class="app-field form-grid__full">
        <label class="app-label" for="pf-bio">{{ t('user.bioLabel') }}</label>
        <textarea
          id="pf-bio"
          v-model="profileForm.bio"
          class="app-textarea"
          rows="3"
          :placeholder="t('user.bioPlaceholder')"
        />
      </div>
      <div class="app-field form-grid__full">
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

    <!-- 改密 -->
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

    <div class="form-divider" />

    <!-- 危险区 -->
    <div class="danger-row">
      <div class="danger-row__body">
        <div class="danger-row__title">{{ t('user.dangerZone') }}</div>
        <div class="danger-row__sub">{{ t('user.deleteAccount') }}</div>
        <div class="danger-row__desc">{{ t('user.deleteAccountDescription') }}</div>
      </div>
      <UButton color="error" variant="soft" size="sm" @click="deleteUser">
        {{ t('user.deleteNow') }}
      </UButton>
    </div>

    <SelectAvatarDialog ref="selectAvatarDialogRef" @get-avatar="userProfileStore.setProfileAvatar" />
    <UpdatePassword ref="updatePasswordRef" />

    <!-- 保存确认 -->
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

const save = () => {
  saveConfirmOpen.value = true
}

const doSave = async () => {
  saving.value = true
  try {
    await userStore.updateUserProfile(profileForm.value)
    saveConfirmOpen.value = false
  } catch (error) {
    fb.error(getRequestErrorMessage(error, t('user.profileUpdateFailed')))
  } finally {
    saving.value = false
  }
}

const deleteUser = () => {
  fb.info(t('user.deleteAccountNotImplemented'))
}

watch(
  () => userStore.userInfo,
  (userInfo) => {
    userProfileStore.syncProfileForm(userInfo)
  },
  { immediate: true },
)

defineExpose({ save })
</script>

<style scoped lang="scss">
.avatar-block {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
}
.avatar-block__meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}
.avatar-block__title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.avatar-block__desc {
  margin: 0 0 0.35rem;
  font-size: 0.75rem;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
}

.form-divider {
  height: 1px;
  margin: 1rem 0;
  background: var(--el-border-color-lighter);
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.75rem 1rem;
}
@media (width >= 640px) {
  .form-grid {
    grid-template-columns: 1fr 1fr;
  }
  .form-grid__full {
    grid-column: 1 / -1;
  }
}

.action-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.65rem 0.75rem;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-fill-color-blank, var(--el-bg-color));
}
.action-row__icon {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: calc(var(--app-radius) - 2px);
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
  color: var(--el-color-primary);
  :deep(svg) {
    width: 18px;
    height: 18px;
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
  margin-top: 0.15rem;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}

.danger-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem 0.85rem;
  border: 1px dashed color-mix(in srgb, var(--el-color-danger) 35%, var(--el-border-color));
  border-radius: var(--app-radius);
  background: color-mix(in srgb, var(--el-color-danger) 5%, transparent);
}
.danger-row__title {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-color-danger);
  margin-bottom: 0.2rem;
}
.danger-row__sub {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.danger-row__desc {
  margin-top: 0.15rem;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}

.confirm-text {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.5;
  color: var(--el-text-color-regular);
}

@media (width < 480px) {
  .action-row,
  .danger-row {
    flex-wrap: wrap;
  }
  .action-row > :last-child,
  .danger-row > :last-child {
    width: 100%;
  }
}
</style>
