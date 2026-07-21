<template>
  <BaseDialog
    v-model="open"
    :title="t('avatar.title')"
    :width="menuStore.isMobile ? '94%' : '800px'"
    @close="close"
  >
    <div class="avatar-container" :class="{ 'is-mobile': menuStore.isMobile }">
      <div class="avatar-menu">
        <button
          type="button"
          class="avatar-menu-item"
          :class="{ active: activeMenu === 'upload' }"
          @click="switchMenu('upload')"
        >
          <component :is="ic('HSolid:CameraIcon')" class="avatar-menu-item__icon" />
          <span>{{ t('avatar.uploadAvatar') }}</span>
        </button>
        <button
          type="button"
          class="avatar-menu-item"
          :class="{ active: activeMenu === 'cat' }"
          @click="switchMenu('cat')"
        >
          <component :is="ic('HSolid:SparklesIcon')" class="avatar-menu-item__icon" />
          <span>{{ t('avatar.systemAvatar') }}</span>
        </button>
        <button
          type="button"
          class="avatar-menu-item"
          :class="{ active: activeMenu === 'url' }"
          @click="switchMenu('url')"
        >
          <component :is="ic('HOutline:LinkIcon')" class="avatar-menu-item__icon" />
          <span>{{ t('avatar.urlAvatar') }}</span>
        </button>
      </div>

      <div class="avatar-content">
        <transition name="fade-slide" mode="out-in">
          <div :key="activeMenu" class="avatar-pane">
            <!-- 上传 -->
            <div v-if="activeMenu === 'upload'" class="upload-container">
              <label
                class="upload-drop"
                :class="{ 'has-preview': !!selectedAvatar }"
                @dragover.prevent
                @drop.prevent="onDrop"
              >
                <input
                  class="upload-drop__input"
                  type="file"
                  accept="image/*"
                  @change="onFileInput"
                />
                <div v-if="!selectedAvatar" class="upload-content">
                  <component :is="ic('HSolid:PhotoIcon')" class="upload-icon" />
                  <div class="upload-text">
                    <div class="upload-text-main">{{ t('avatar.dragUpload') }}</div>
                    <div class="upload-text-tip">{{ t('avatar.uploadTip') }}</div>
                  </div>
                </div>
                <div v-else class="upload-preview-content">
                  <AppAvatar :src="selectedAvatar || undefined" :size="previewAvatarSize" />
                  <div class="preview-hint">{{ t('avatar.reuploadHint') }}</div>
                </div>
              </label>
            </div>

            <!-- 系统头像 -->
            <div v-else-if="activeMenu === 'cat'" class="cat-container">
              <div class="icon-search">
                <component :is="ic('HOutline:MagnifyingGlassIcon')" class="icon-search__icon" />
                <input
                  v-model="avatarSearchText"
                  class="app-input icon-search__input"
                  type="search"
                  :placeholder="t('avatar.searchPlaceholder')"
                />
              </div>
              <div class="cat-scroll">
                <div class="cat-avatar-list">
                  <button
                    v-for="cat in filteredAvatars"
                    :key="cat.id"
                    type="button"
                    class="cat-avatar-item"
                    :class="{ active: selectedAvatarId === cat.id }"
                    @click="selectAvatar(cat)"
                  >
                    <div class="cat-avatar-image">
                      <img :src="cat.src" :alt="cat.alt" />
                    </div>
                    <div class="cat-avatar-name">{{ cat.title }}</div>
                  </button>
                </div>
              </div>
            </div>

            <!-- URL -->
            <div v-else class="url-container">
              <div class="url-input-block">
                <div class="url-title">{{ t('avatar.urlTitle') }}</div>
                <div class="url-tip">{{ t('avatar.urlTip') }}</div>
                <div class="icon-search">
                  <component :is="ic('HOutline:LinkIcon')" class="icon-search__icon" />
                  <input
                    v-model.trim="avatarUrlInput"
                    class="app-input icon-search__input"
                    type="url"
                    :placeholder="t('avatar.urlPlaceholder')"
                    @input="syncUrlAvatar"
                  />
                </div>
              </div>
              <div class="url-preview-card">
                <div class="url-preview-label">{{ t('avatar.preview') }}</div>
                <div class="url-preview-body">
                  <AppAvatar
                    v-if="selectedAvatar"
                    :src="selectedAvatar"
                    :size="previewAvatarSize"
                  />
                  <div v-else class="url-preview-empty">{{ t('avatar.urlPreviewEmpty') }}</div>
                </div>
              </div>
            </div>
          </div>
        </transition>
      </div>
    </div>

    <template #footer>
      <UButton color="neutral" variant="soft" @click="close">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" :disabled="!selectedAvatar" @click="updateAvatar">
        {{ t('common.confirm') }}
      </UButton>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'SelectAvatarDialog' })

interface IEmits {
  (e: 'getAvatar', avatar: string): void
}

const emits = defineEmits<IEmits>()
const menuStore = useMenuStore()
const { t } = useI18n()
const ic = (name: string) => menuStore.iconComponents[name]

const open = ref(false)
const previewAvatarSize = computed(() => (menuStore.isMobile ? 132 : 180))
const activeMenu = ref<'upload' | 'cat' | 'url'>('upload')
const selectedAvatarId = ref<string | number | null>(null)
const selectedAvatar = ref<string | null>(null)
const avatarUrlInput = ref('')
const avatarSearchText = ref('')

const allAvatars = ref(
  Array.from({ length: 100 }, (_, index) => ({
    id: index + 1,
    title: String(index + 1),
    alt: String(index + 1),
    src: `https://api.dicebear.com/7.x/avataaars/svg?seed=${index + 1}`,
  })),
)

const filteredAvatars = computed(() => {
  if (!avatarSearchText.value) return allAvatars.value
  const search = avatarSearchText.value.toLowerCase()
  return allAvatars.value.filter((avatar) => avatar.title.toLowerCase().includes(search))
})

const switchMenu = (menu: 'upload' | 'cat' | 'url') => {
  if (activeMenu.value === menu) return
  activeMenu.value = menu
  selectedAvatarId.value = null
  selectedAvatar.value = null
  avatarUrlInput.value = ''
}

const selectAvatar = (avatar: { src: string; id?: string | number }) => {
  selectedAvatarId.value = avatar.id ?? null
  selectedAvatar.value = avatar.src
}

const syncUrlAvatar = () => {
  selectedAvatarId.value = null
  selectedAvatar.value = avatarUrlInput.value.trim() || null
}

const readImageFile = (file: File) => {
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

const onFileInput = (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) readImageFile(file)
  input.value = ''
}

const onDrop = (event: DragEvent) => {
  const file = event.dataTransfer?.files?.[0]
  if (file && file.type.startsWith('image/')) readImageFile(file)
}

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

defineExpose({ showDialog })
</script>

<style scoped lang="scss">
.avatar-container {
  display: flex;
  gap: 1rem;
  min-height: 420px;
}
.avatar-menu {
  width: 10rem;
  border-right: 1px solid var(--el-border-color-lighter);
  padding: 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.avatar-menu-item {
  padding: 0.5rem 0.85rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border: none;
  border-radius: 0.75rem;
  background: transparent;
  cursor: pointer;
  color: var(--el-text-color-primary);
  font-weight: 500;
  text-align: left;
  transition: background 0.15s, color 0.15s;
}
.avatar-menu-item__icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}
.avatar-menu-item:hover {
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
}
.avatar-menu-item.active {
  background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
  color: var(--el-color-primary);
}
.avatar-content {
  flex: 1;
  min-width: 0;
  min-height: 0;
  padding: 0.5rem;
}
.avatar-pane {
  height: 100%;
}

.upload-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 360px;
}
.upload-drop {
  width: 100%;
  min-height: 280px;
  border: 2px dashed var(--el-border-color);
  border-radius: 12px;
  background: var(--el-fill-color-lighter);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: border-color 0.15s;
  position: relative;
}
.upload-drop:hover {
  border-color: var(--el-color-primary);
}
.upload-drop__input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}
.upload-content,
.upload-preview-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 1.5rem;
  pointer-events: none;
}
.upload-icon {
  width: 48px;
  height: 48px;
  color: var(--el-color-primary);
}
.upload-text {
  text-align: center;
}
.upload-text-main {
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 0.25rem;
}
.upload-text-tip {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.preview-hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}

.cat-container {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  height: 100%;
  min-height: 360px;
}
.icon-search {
  position: relative;
  display: flex;
  align-items: center;
}
.icon-search__icon {
  position: absolute;
  left: 10px;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-placeholder);
  pointer-events: none;
}
.icon-search__input {
  width: 100%;
  padding-left: 32px;
}
.cat-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.cat-avatar-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
  gap: 0.75rem;
  padding: 0.25rem 0;
}
.cat-avatar-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  border: 1px solid var(--el-border-color);
  border-radius: 0.5rem;
  background: var(--el-bg-color);
  cursor: pointer;
  padding: 0.5rem;
  transition: border-color 0.15s, transform 0.15s;
}
.cat-avatar-item:hover {
  border-color: var(--el-color-primary);
  transform: translateY(-1px);
}
.cat-avatar-item.active {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 40%, transparent);
}
.cat-avatar-image img {
  width: 72px;
  height: 72px;
  object-fit: cover;
  border-radius: 50%;
}
.cat-avatar-name {
  font-size: 0.75rem;
  margin-top: 0.25rem;
  color: var(--el-text-color-secondary);
}

.url-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  min-height: 360px;
}
.url-input-block {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.url-title {
  font-size: 1rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.url-tip {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
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
}
.url-preview-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.url-preview-body {
  flex: 1;
  min-height: 180px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.url-preview-empty {
  text-align: center;
  font-size: 0.8125rem;
  color: var(--el-text-color-placeholder);
}

.is-mobile {
  flex-direction: column;
  min-height: 0;
}
.is-mobile .avatar-menu {
  width: 100%;
  border-right: none;
  border-bottom: 1px solid var(--el-border-color-lighter);
  flex-direction: row;
  gap: 0.25rem;
}
.is-mobile .avatar-menu-item {
  flex: 1;
  justify-content: center;
  padding: 0.65rem 0.35rem;
  font-size: 0.8125rem;
}

.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: opacity 0.15s ease;
}
.fade-slide-enter-from,
.fade-slide-leave-to {
  opacity: 0;
}
</style>
