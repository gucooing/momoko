// 通行证状态：正常 / 已停用 / 待注销（申请了注销、7 天内可撤销）。
import { useI18n } from 'vue-i18n'
import type { PassportAdminUser } from '@/types/v1/passport_admin'

export type PassportStatusVariant = 'success' | 'error' | 'warning'

export const usePassportStatus = () => {
  const { t } = useI18n()
  return (u: PassportAdminUser): { variant: PassportStatusVariant; label: string } => {
    if (u.status === 'disabled') return { variant: 'error', label: t('passportAdmin.status.disabled') }
    if (u.deletionScheduledAt) return { variant: 'warning', label: t('passportAdmin.status.deleting') }
    return { variant: 'success', label: t('passportAdmin.status.active') }
  }
}
