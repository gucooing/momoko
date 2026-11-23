import { h } from 'vue'
import UpdateReleaseContent from '@/components/update/UpdateReleaseContent.vue'
import { checkUpdateRequest } from '@/api/login'
import { startUpdate } from '@/api/system'
import { pageTabTo } from '@/config/pageGroups'
import { refreshDownloads } from '@/layouts/app/downloads/useDownloads'
import { translate } from '@/locales'
import router from '@/router'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import type { CheckUpdateResponse } from '@/types/v1/system'

const PROJECT_RELEASES_LINK = 'https://github.com/gucooing/momoko/releases'

interface CheckForUpdateOptions {
  // 自动检查时不提示“已是最新版本”，只在发现更新时打扰用户。
  silentNoUpdate?: boolean
  // 点了「立即更新」由调用方接手（系统设置页自己显示进度）；不传时在这里开始更新并跳到设置页。
  onStartUpdate?: () => void
}

// 不在设置页时点「立即更新」：开始下载更新，有设置页权限就跳过去看进度。
const startUpdateHere = async () => {
  try {
    await startUpdate()
    void refreshDownloads()
    feedback.success(translate('system.settings.updater.started'))
    const menuStore = useMenuStore()
    if (menuStore.hasButtonPermission('system_config:view')) {
      void router.push(pageTabTo('settings', 'general', { sec: 'update' }))
    } else {
      feedback.info(translate('system.settings.updater.viewDownloads'))
    }
  } catch (e) {
    showRequestError(e)
  }
}

// 统一展示更新弹窗，避免手动检查和工作台自动检查出现两套文案与样式。
// 最新发行版里有本机系统的包、且有更新权限时，确认按钮是「立即更新」（在线下载并替换），否则去发行版页面手动下载。
const showUpdateDialog = (update: CheckUpdateResponse, options: CheckForUpdateOptions) => {
  const menuStore = useMenuStore()
  const oneClick = update.canAutoUpdate && menuStore.hasButtonPermission('system:update')
  return Dialog.info({
    showCancelButton: true,
    showClose: true,
    showFullscreenButton: true,
    mobileAdaptive: true,
    width: '680px',
    mobileWidth: 'calc(100vw - 24px)',
    title: translate('layout.updateAvailableTitle'),
    content: () =>
      h(UpdateReleaseContent, {
        update,
        note: oneClick ? translate('system.settings.updater.dialogNote') : undefined,
      }),
    confirmText: oneClick ? translate('system.settings.updater.updateNow') : translate('layout.updateGoDownload'),
    cancelText: translate('layout.logoutCancelText'),
    onConfirm: () => {
      if (!oneClick) {
        window.open(update.releaseUrl || PROJECT_RELEASES_LINK, '_blank')
        return
      }
      if (options.onStartUpdate) options.onStartUpdate()
      else void startUpdateHere()
    },
  })
}

// 执行更新检查；返回接口数据，调用方可继续根据角色、页面等条件做外层控制。
export const checkForUpdate = async (options: CheckForUpdateOptions = {}) => {
  const { data } = await checkUpdateRequest()

  if (data?.hasUpdate) {
    void showUpdateDialog(data, options).catch(() => undefined)
    return data
  }

  if (!options.silentNoUpdate) {
    feedback.success(
      translate('layout.updateUpToDate', { version: data?.currentVersion || 'dev' }),
    )
  }

  return data
}
