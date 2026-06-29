import { h } from 'vue'
import { ElMessage } from 'element-plus'
import UpdateReleaseContent from '@/components/update/UpdateReleaseContent.vue'
import { checkUpdateRequest } from '@/api/login'
import { translate } from '@/locales'
import { Dialog } from '@/utils/dialog'
import type { CheckUpdateResponse } from '@/types/v1/system'

const PROJECT_RELEASES_LINK = 'https://github.com/gucooing/momoko/releases'

interface CheckForUpdateOptions {
  // 自动检查时不提示“已是最新版本”，只在发现更新时打扰用户。
  silentNoUpdate?: boolean
}

// 统一展示更新弹窗，避免手动检查和工作台自动检查出现两套文案与样式。
const showUpdateDialog = (update: CheckUpdateResponse) => {
  return Dialog.info({
    showCancelButton: true,
    showClose: true,
    showFullscreenButton: true,
    mobileAdaptive: true,
    width: '680px',
    mobileWidth: 'calc(100vw - 24px)',
    title: translate('layout.updateAvailableTitle'),
    content: () => h(UpdateReleaseContent, { update }),
    confirmText: translate('layout.updateGoDownload'),
    cancelText: translate('layout.logoutCancelText'),
    onConfirm: () => {
      window.open(update.releaseUrl || PROJECT_RELEASES_LINK, '_blank')
    },
  })
}

// 执行更新检查；返回接口数据，调用方可继续根据角色、页面等条件做外层控制。
export const checkForUpdate = async (options: CheckForUpdateOptions = {}) => {
  const { data } = await checkUpdateRequest()

  if (data?.hasUpdate) {
    void showUpdateDialog(data).catch(() => undefined)
    return data
  }

  if (!options.silentNoUpdate) {
    ElMessage.success(
      translate('layout.updateUpToDate', { version: data?.currentVersion || 'dev' }),
    )
  }

  return data
}
