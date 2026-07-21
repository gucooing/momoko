<template>
  <BaseDialog
    v-model="open"
    :title="t('avatar.title')"
    :width="menuStore.isMobile ? '94%' : '800px'"
    :close-on-click-modal="false"
    @close="close"
    :style="dialogStyle"
  >
    <div class="avatar-container" :class="{ 'is-mobile': menuStore.isMobile }">
      <div class="avatar-menu">
        <div
          class="avatar-menu-item"
          :class="{ active: activeMenu === 'upload' }"
          @click="switchMenu('upload')"
        >
          <el-icon size="20"
            ><component :is="menuStore.iconComponents['HSolid:CameraIcon']"
          /></el-icon>
          <span>{{ t('avatar.uploadAvatar') }}</span>
        </div>
        <div
          class="avatar-menu-item"
          :class="{ active: activeMenu === 'cat' }"
          @click="switchMenu('cat')"
        >
          <el-icon size="20"
            ><component :is="menuStore.iconComponents['HSolid:SparklesIcon']"
          /></el-icon>
          <span>{{ t('avatar.systemAvatar') }}</span>
        </div>
        <div
          class="avatar-menu-item"
          :class="{ active: activeMenu === 'url' }"
          @click="switchMenu('url')"
        >
          <el-icon size="20"
            ><component :is="menuStore.iconComponents['HOutline:LinkIcon']"
          /></el-icon>
          <span>{{ t('avatar.urlAvatar') }}</span>
        </div>
      </div>
      <div class="avatar-content">
        <transition name="fade-slide" mode="out-in">
          <div :key="activeMenu" style="height: 100%">
            <div v-if="activeMenu === 'upload'" class="upload-container">
              <el-upload
                drag
                accept="image/*"
                :show-file-list="false"
                :auto-upload="false"
                :on-change="uploadFile"
                class="upload-drag"
              >
                <div v-if="!selectedAvatar" class="upload-content">
                  <el-icon class="upload-icon">
                    <component :is="menuStore.iconComponents['HSolid:PhotoIcon']" />
                  </el-icon>
                  <div class="upload-text">
                    <div class="upload-text-main">{{ t('avatar.dragUpload') }}</div>
                    <div class="upload-text-tip">
                      {{ t('avatar.uploadTip') }}
                    </div>
                  </div>
                </div>
                <div v-else class="upload-preview-content">
                  <el-avatar :size="previewAvatarSize" :src="selectedAvatar" />
                  <div class="preview-hint">{{ t('avatar.reuploadHint') }}</div>
                </div>
              </el-upload>
            </div>
            <div v-else-if="activeMenu === 'cat'" class="cat-container">
              <el-input v-model="avatarSearchText" :placeholder="t('avatar.searchPlaceholder')" clearable>
                <template #prefix>
                  <el-icon><component :is="menuStore.iconComponents['Search']" /></el-icon>
                </template>
              </el-input>

              <el-scrollbar>
                <div class="cat-avatar-list">
                  <div
                    class="cat-avatar-item"
                    v-for="cat in filteredAvatars"
                    :key="cat.id"
                    :class="{ active: selectedAvatarId === cat.id }"
                    @click="selectAvatar(cat)"
                  >
                    <div class="cat-avatar-image">
                      <img :src="cat.src" :alt="cat.alt" />
                    </div>
                    <div class="cat-avatar-name">{{ cat.title }}</div>
                  </div>
                </div>
              </el-scrollbar>
            </div>
            <div v-else-if="activeMenu === 'url'" class="url-container">
              <div class="url-input-block">
                <div class="url-title">{{ t('avatar.urlTitle') }}</div>
                <div class="url-tip">{{ t('avatar.urlTip') }}</div>
                <el-input
                  v-model.trim="avatarUrlInput"
                  :placeholder="t('avatar.urlPlaceholder')"
                  clearable
                  @input="syncUrlAvatar"
                >
                  <template #prefix>
                    <el-icon><component :is="menuStore.iconComponents['HOutline:LinkIcon']" /></el-icon>
                  </template>
                </el-input>
              </div>

              <div class="url-preview-card">
                <div class="url-preview-label">{{ t('avatar.preview') }}</div>
                <div class="url-preview-body">
                  <el-avatar v-if="selectedAvatar" :size="previewAvatarSize" :src="selectedAvatar" />
                  <div v-else class="url-preview-empty">
                    {{ t('avatar.urlPreviewEmpty') }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </transition>
      </div>
    </div>

    <template #footer>
      <el-button @click="close">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" :disabled="!selectedAvatar" @click="updateAvatar">
        {{ t('common.confirm') }}
      </el-button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import type { UploadFile } from 'element-plus'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'SelectAvatarDialog' })

interface IEmits {
  (e: 'getAvatar', avatar: string): void
}

const emits = defineEmits<IEmits>()

const menuStore = useMenuStore()
const { t } = useI18n()

const open = ref(false)
const previewAvatarSize = computed(() => (menuStore.isMobile ? 132 : 180))
const dialogStyle = computed(() => ({
  height: menuStore.isMobile ? '82vh' : '60vh',
}))

// 当前选中的菜单
const activeMenu = ref<'upload' | 'cat' | 'url'>('upload')
// 当前选中的头像id
const selectedAvatarId = ref<string | number | null>(null)
// 当前选中的头像
const selectedAvatar = ref<string | null>(null)
const avatarUrlInput = ref('')

// 搜索框内容
const avatarSearchText = ref('')

// 获取所有头像
const allAvatars = ref(
  Array.from({ length: 100 }, (_, index) => {
    return {
      id: index + 1,
      title: String(index + 1),
      alt: String(index + 1),
      src: `https://api.dicebear.com/7.x/avataaars/svg?seed=${index + 1}`,
    }
  }),
)

// 过滤后的头像列表
const filteredAvatars = computed(() => {
  if (!avatarSearchText.value) {
    return allAvatars.value
  }
  const search = avatarSearchText.value.toLowerCase()
  return allAvatars.value.filter((avatar) => avatar.title.toLowerCase().includes(search))
})

// 切换菜单
const switchMenu = (menu: 'upload' | 'cat' | 'url') => {
  if (activeMenu.value === menu) return
  activeMenu.value = menu
  selectedAvatarId.value = null
  selectedAvatar.value = null
  avatarUrlInput.value = ''
}
// 选择头像
const selectAvatar = (avatar: { src: string; id?: string | number }) => {
  selectedAvatarId.value = avatar.id ?? null
  selectedAvatar.value = avatar.src
}

const syncUrlAvatar = () => {
  selectedAvatarId.value = null
  selectedAvatar.value = avatarUrlInput.value.trim() || null
}

// 上传头像
const uploadFile = (uploadFile: UploadFile) => {
  const file = uploadFile.raw
  if (!file) return

  // 验证文件大小（2MB）
  const maxSize = 2 * 1024 * 1024
  if (file.size > maxSize) {
    feedback.warning(t('avatar.imageSizeLimit'))
    return
  }

  const reader = new FileReader()
  reader.onload = () => {
    selectedAvatar.value = reader.result as string
  }
  reader.readAsDataURL(file)
}

// 修改头像
const updateAvatar = async () => {
  if (!selectedAvatar.value) return
  emits('getAvatar', selectedAvatar.value)
  close()
}

const close = () => {
  open.value = false
  avatarSearchText.value = ''
  selectedAvatarId.value = null
  selectedAvatar.value = null
  avatarUrlInput.value = ''
}

const showDialog = () => {
  open.value = true
}

defineExpose({
  showDialog,
})
</script>

<style scoped lang="scss">
.avatar-container {
  display: flex;
  gap: 1rem;
  height: 100%;

  // 移动端布局
  &.is-mobile {
    height: 100%;
    flex-direction: column;
    gap: 0.5rem;
    min-height: auto;
    overflow: hidden;

    .avatar-menu {
      width: 100%;
      border-right: none;
      border-bottom: 1px solid var(--el-border-color-lighter);
      padding: 0.5rem;
      flex-direction: row;
      justify-content: space-around;
      gap: 0.25rem;

      .avatar-menu-item {
        flex: 1;
        justify-content: center;
        padding: 0.75rem 0.5rem;
        font-size: 0.875rem;
      }
    }

    .avatar-content {
      padding: 0.25rem 0;
      overflow-y: auto;
    }
  }

  .avatar-menu {
    width: 10rem;
    border-right: 1px solid var(--el-border-color-lighter);
    padding: 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    .avatar-menu-item {
      padding: 0.5rem 1rem;
      display: flex;
      align-items: center;
      gap: 0.5rem;
      border-radius: 0.75rem;
      cursor: pointer;
      color: var(--el-text-color-primary);
      font-weight: 500;
      transition: all 0.3s ease;

      &:hover {
        background: var(--el-fill-color-light);
        color: var(--el-color-primary);
      }

      &.active {
        background: linear-gradient(
          135deg,
          color-mix(in srgb, var(--el-color-primary) 20%, transparent) 0%,
          color-mix(in srgb, var(--el-color-primary) 20%, transparent) 100%
        );
        color: var(--el-color-primary);
      }
    }
  }
  .avatar-content {
    flex: 1;
    padding: 0.5rem;
    height: 100%;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;

    .upload-container {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      min-height: 410px;
      gap: 1.5rem;

      // 移动端优化
      @media (max-width: 992px) {
        gap: 1rem;
        min-height: 0;
        justify-content: flex-start;
      }

      .upload-drag {
        width: 100%;
        :deep(.el-upload) {
          width: 100%;
        }
        :deep(.el-upload-dragger) {
          width: 100%;
          height: 300px;
          border: 2px dashed var(--el-border-color);
          border-radius: 12px;
          background: var(--el-fill-color-lighter);
          transition: all 0.3s ease;
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          gap: 1rem;
          padding: 2rem;
          position: relative;

          // 移动端优化
          @media (max-width: 992px) {
            height: clamp(220px, 34vh, 260px);
            padding: 1rem;
            gap: 0.75rem;
          }

          &:hover {
            border-color: var(--el-color-primary);
          }

          .upload-content {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            gap: 1rem;

            @media (max-width: 992px) {
              gap: 0.75rem;
            }

            .upload-icon {
              font-size: 4rem;
              color: var(--el-color-primary);

              @media (max-width: 992px) {
                font-size: 3rem;
              }
            }

            .upload-text {
              text-align: center;
              .upload-text-main {
                font-size: 1rem;
                color: var(--el-text-color-primary);
                font-weight: 500;
                margin-bottom: 0.5rem;

                @media (max-width: 992px) {
                  font-size: 0.875rem;
                  margin-bottom: 0.25rem;
                }
              }
              .upload-text-tip {
                font-size: 0.875rem;
                color: var(--el-text-color-secondary);

                @media (max-width: 992px) {
                  font-size: 0.75rem;
                }
              }
            }
          }

          .upload-preview-content {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            gap: 1rem;
            width: 100%;
            height: 100%;

            @media (max-width: 992px) {
              gap: 0.75rem;
            }

            .preview-hint {
              font-size: 0.875rem;
              color: var(--el-text-color-secondary);
              text-align: center;

              @media (max-width: 992px) {
                font-size: 0.75rem;
                padding: 0 0.5rem;
              }
            }
          }
        }
      }
    }

    .cat-container {
      display: flex;
      flex-direction: column;
      gap: 1rem;
      height: 100%;

      @media (max-width: 992px) {
        gap: 0.75rem;
      }

      .cat-avatar-list {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
        gap: 1rem;
        padding: 0.5rem 0;

        @media (max-width: 992px) {
          grid-template-columns: repeat(auto-fill, minmax(90px, 1fr));
          gap: 0.75rem;
          padding: 0.25rem 0;
        }

        .cat-avatar-item {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          border: 2px solid var(--el-border-color);
          border-radius: 0.5rem;
          cursor: pointer;
          transition: all 0.3s ease;
          padding: 0.5rem;

          @media (max-width: 992px) {
            padding: 0.375rem;
            border-width: 1.5px;
          }

          &:hover {
            border-color: var(--el-color-primary);
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
          }

          // 移动端点击效果
          @media (max-width: 992px) {
            &:active {
              transform: scale(0.95);
            }
          }

          &.active {
            border-color: var(--el-color-primary);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
          }
          .cat-avatar-image {
            img {
              width: 100%;
              height: 100%;
              object-fit: cover;
            }
          }

          .cat-avatar-name {
            font-size: 0.75rem;
            margin-top: 0.25rem;
            text-align: center;

            @media (max-width: 992px) {
              font-size: 0.625rem;
              margin-top: 0.125rem;
            }
          }
        }
      }
    }

    .url-container {
      display: flex;
      flex-direction: column;
      gap: 1rem;
      min-height: 410px;

      @media (max-width: 992px) {
        min-height: 0;
        gap: 0.75rem;
      }

      .url-input-block {
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
      }

      .url-title {
        font-size: 1rem;
        font-weight: 700;
        color: var(--el-text-color-primary);
      }

      .url-tip {
        font-size: 0.875rem;
        color: var(--el-text-color-secondary);
        line-height: 1.5;
      }

      .url-preview-card {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
        border: 1px dashed var(--el-border-color);
        border-radius: 12px;
        background: var(--el-fill-color-lighter);
        padding: 1rem;

        @media (max-width: 992px) {
          min-height: 220px;
          padding: 0.75rem;
        }
      }

      .url-preview-label {
        font-size: 0.875rem;
        font-weight: 600;
        color: var(--el-text-color-secondary);
      }

      .url-preview-body {
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 0.75rem;
        background: color-mix(in srgb, var(--el-bg-color) 72%, transparent);
        padding: 1rem;

        @media (max-width: 992px) {
          min-height: 180px;
          padding: 0.75rem;
        }
      }

      .url-preview-empty {
        text-align: center;
        font-size: 0.875rem;
        color: var(--el-text-color-placeholder);
        line-height: 1.6;
      }
    }
  }
}
</style>
