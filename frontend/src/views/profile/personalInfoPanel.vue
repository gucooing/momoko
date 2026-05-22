<template>
  <!-- 个人资料中心 -->
  <BaseCard title="个人资料中心" title-icon="HOutline:UserIcon">
    <template #header-right>
      <el-button type="primary" @click="saveUser">保存全部更改</el-button>
    </template>
    <div>
      <div class="flex flex-col items-start gap-4 sm:flex-row sm:items-center">
        <el-avatar :size="110" :src="previewAvatar" class="shrink-0" />
        <div class="flex w-full flex-col items-start gap-2">
          <h4>您的头像</h4>
          <p class="text-sm text-(--el-text-color-secondary)">
            内置多种头像。支持 JPG、PNG、GIF 等格式头像上传，建议大小不超过 2MB 。
          </p>
          <el-button size="small" type="primary" @click="selectAvatarDialogRef?.showDialog()"
            >修改头像</el-button
          >
        </div>
      </div>

      <el-divider />

      <el-form :model="profileForm" label-position="top" class="custom-form">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="用户名">
              <el-input v-model="profileForm.username" placeholder="请输入登录用户名" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="姓名">
              <el-input v-model="profileForm.name" placeholder="请输入姓名" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="邮箱">
              <el-input v-model="profileForm.email" placeholder="请输入邮箱" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="个人简介/座右铭">
          <el-input
            v-model="profileForm.bio"
            type="textarea"
            :rows="4"
            placeholder="写点什么来展示你自己..."
          />
        </el-form-item>
        <el-form-item label="个人标签（用几个关键词介绍一下自己）">
          <el-input
            v-model="profileForm.tags"
            type="textarea"
            :rows="4"
            placeholder="多个标签请用英文逗号分隔, 例如：写代码的, 爱咖啡, 偶尔健身, 长期学习中"
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
                <div class="mb-1 text-sm font-bold text-(--el-text-color-primary)">修改密码</div>
                <div class="text-xs text-(--el-text-color-secondary)">
                  定期更换强密码能显著提升账户安全性，建议包含字母与数字。
                </div>
              </div>
            </div>

            <el-button type="primary" plain @click="updatePasswordRef?.showDialog()">
              立即修改
            </el-button>
          </div>
        </div>
      </HoverAnimateWrapper>

      <el-divider />
      <HoverAnimateWrapper name="lift" class="w-full">
        <div
          class="p-4 bg-(--el-color-danger-light-9) border border-dashed border-(--el-color-danger-light-5) rounded-xl cursor-pointer transition-all duration-300 hover:bg-(--el-color-danger-light-7) hover:border-(--el-color-danger)"
        >
          <h4 class="mb-2 text-(--el-color-danger) font-bold">危险区域</h4>
          <div class="flex items-center justify-between gap-4">
            <div>
              <div class="mb-1 text-sm font-bold">注销账户</div>
              <div class="text-sm text-(--el-text-color-secondary)">
                一旦注销，所有数据将无法恢复，请谨慎操作。
              </div>
            </div>
            <el-button type="danger" plain @click="deleteUser">立即注销</el-button>
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

const menuStore = useMenuStore()
const userStore = useUserStore()
const userProfileStore = useUserProfileStore()
const { profileForm } = storeToRefs(userProfileStore)
const previewAvatar = computed(() => resolveAvatarUrl(profileForm.value.avatar) || '')

const selectAvatarDialogRef = useTemplateRef('selectAvatarDialogRef')
const updatePasswordRef = useTemplateRef('updatePasswordRef')

// 保存修改
const saveUser = () => {
  Dialog.confirm({
    title: '确认保存个人资料',
    content: '小改动，大影响～确认保存当前修改吗？保存后将立即生效。',
    cancelText: '我再看看',
    confirmText: '确认保存',
    onConfirm: async () => {
      try {
        await userStore.updateUserProfile(profileForm.value)
      } catch (error) {
        showRequestError(error, '修改个人资料失败')
        throw error
      }
    },
  })
}

// 注销用户
const deleteUser = () => {
  ElMessage.info('注销账户功能暂未实现')
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
