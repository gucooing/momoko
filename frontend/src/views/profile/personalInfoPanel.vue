<template>
  <!-- 个人资料中心 -->
  <BaseCard :title="t('user.profileCenter')" title-icon="HOutline:UserIcon">
    <template #header-right>
      <el-button type="primary" @click="saveUser">{{ t('user.saveAllChanges') }}</el-button>
    </template>
    <div>
      <div class="flex flex-col items-start gap-4 sm:flex-row sm:items-center">
        <el-avatar :size="110" :src="previewAvatar" class="shrink-0" />
        <div class="flex w-full flex-col items-start gap-2">
          <h4>{{ t('user.avatarTitle') }}</h4>
          <p class="text-sm text-(--el-text-color-secondary)">
            {{ t('user.avatarDescription') }}
          </p>
          <el-button size="small" type="primary" @click="selectAvatarDialogRef?.showDialog()"
            >{{ t('user.changeAvatar') }}</el-button
          >
        </div>
      </div>

      <el-divider />

      <el-form :model="profileForm" label-position="top" class="custom-form">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('user.username')">
              <el-input v-model="profileForm.username" :placeholder="t('user.loginUsernamePlaceholder')" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('user.name')">
              <el-input v-model="profileForm.name" :placeholder="t('user.namePlaceholder')" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('user.email')">
              <el-input v-model="profileForm.email" :placeholder="t('user.emailPlaceholder')" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item :label="t('user.bioLabel')">
          <el-input
            v-model="profileForm.bio"
            type="textarea"
            :rows="4"
            :placeholder="t('user.bioPlaceholder')"
          />
        </el-form-item>
        <el-form-item :label="t('user.tagsLabel')">
          <el-input
            v-model="profileForm.tags"
            type="textarea"
            :rows="4"
            :placeholder="t('user.tagsPlaceholder')"
          />
        </el-form-item>
      </el-form>

      <el-divider />

      <HoverAnimateWrapper name="lift" class="w-full">
        <div
          class="p-5 border border-solid border-(--el-border-color-light) rounded-xl transition-all duration-300 hover:border-(--el-border-color) hover:bg-(--el-bg-color-page) cursor-pointer"
        >
          <div class="flex items-center justify-between gap-4">
            <div class="flex items-center gap-4">
              <div
                class="shrink-0 flex items-center justify-center w-12 h-12 rounded-lg bg-(--el-color-info-light-7) text-(--el-color-primary) transition-colors duration-300"
              >
                <el-icon size="20">
                  <component :is="menuStore.iconComponents['HOutline:KeyIcon']" />
                </el-icon>
              </div>

              <div>
                <div class="mb-1 text-sm font-bold text-(--el-text-color-primary)">{{ t('user.passwordCardTitle') }}</div>
                <div class="text-xs text-(--el-text-color-secondary)">
                  {{ t('user.passwordCardDescription') }}
                </div>
              </div>
            </div>

            <el-button type="primary" plain @click="updatePasswordRef?.showDialog()">
              {{ t('user.changeNow') }}
            </el-button>
          </div>
        </div>
      </HoverAnimateWrapper>

      <el-divider />
      <HoverAnimateWrapper name="lift" class="w-full">
        <div
          class="p-4 bg-(--el-color-danger-light-9) border border-dashed border-(--el-color-danger-light-5) rounded-xl cursor-pointer transition-all duration-300 hover:bg-(--el-color-danger-light-7) hover:border-(--el-color-danger)"
        >
          <h4 class="mb-2 text-(--el-color-danger) font-bold">{{ t('user.dangerZone') }}</h4>
          <div class="flex items-center justify-between gap-4">
            <div>
              <div class="mb-1 text-sm font-bold">{{ t('user.deleteAccount') }}</div>
              <div class="text-sm text-(--el-text-color-secondary)">
                {{ t('user.deleteAccountDescription') }}
              </div>
            </div>
            <el-button type="danger" plain @click="deleteUser">{{ t('user.deleteNow') }}</el-button>
          </div>
        </div>
      </HoverAnimateWrapper>
    </div>

    <SelectAvatarDialog ref="selectAvatarDialogRef" @get-avatar="userProfileStore.setProfileAvatar" />
    <UpdatePassword ref="updatePasswordRef" />
  </BaseCard>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { storeToRefs } from 'pinia'
import { useUserProfileStore } from '@/stores/user/profile'
import { resolveAvatarUrl } from '@/utils/assets'
import { showRequestError } from '@/utils/request'
import { Dialog } from '@/utils/dialog'
import { useI18n } from 'vue-i18n'

const menuStore = useMenuStore()
const userStore = useUserStore()
const userProfileStore = useUserProfileStore()
const { t } = useI18n()
const { profileForm } = storeToRefs(userProfileStore)
const previewAvatar = computed(() => resolveAvatarUrl(profileForm.value.avatar) || '')

const selectAvatarDialogRef = useTemplateRef('selectAvatarDialogRef')
const updatePasswordRef = useTemplateRef('updatePasswordRef')

// 保存修改
const saveUser = () => {
  Dialog.confirm({
    title: t('user.saveProfileTitle'),
    content: t('user.saveProfileContent'),
    cancelText: t('user.saveProfileCancel'),
    confirmText: t('user.saveProfileConfirm'),
    onConfirm: async () => {
      try {
        await userStore.updateUserProfile(profileForm.value)
      } catch (error) {
        showRequestError(error, t('user.profileUpdateFailed'))
        throw error
      }
    },
  })
}

// 注销用户
const deleteUser = () => {
  ElMessage.info(t('user.deleteAccountNotImplemented'))
}

// 监听用户信息 赋值
watch(
  () => userStore.userInfo,
  (userInfo) => {
    userProfileStore.syncProfileForm(userInfo)
  },
  {
    immediate: true,
  },
)
</script>
