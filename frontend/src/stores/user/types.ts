import type { Component } from 'vue'
import type { LoginDevice, UpdatePasswordRequest } from '@/types/v1/auth'
import type { UpdateMeRequest } from '@/types/v1/user'

/** 右侧内容 Tab（资料不在 Tab 内，左栏常驻 — 06a） */
export type ProfileCurrentTab = 'messages' | 'permissions' | 'devices' | 'logs'

export interface ProfileTabsMenuItem {
  key: string
  label?: string
  labelKey?: string
  badge?: number | string
  disabled?: boolean
  icon?: string | Component
}

export type MessageType = 'system' | 'security' | 'notice'

/** 站内消息（标题 / 内容已按当前语言解析好，供顶栏通知与个人中心共用） */
export interface UserMessageItem {
  id: string
  title: string
  content: string
  type: MessageType
  /** 后端消息种类：login_new_device / login_failed / password_changed / password_reset / notice */
  kind: string
  params: Record<string, string>
  read: boolean
  /** 格式化后的时间 YYYY-MM-DD HH:mm:ss */
  time: string
  /** 原始时间（ISO） */
  createTime: string
  avatar?: string
  senderName?: string
}

export type UserProfileFormValue = {
  [K in keyof Pick<
    UpdateMeRequest,
    'username' | 'name' | 'email' | 'bio' | 'tags' | 'phone'
  >]-?: NonNullable<UpdateMeRequest[K]>
}

export interface UserPasswordFormValue extends UpdatePasswordRequest {
  confirmPassword: string
}

export interface LoginDeviceRow extends LoginDevice, Record<string, unknown> {}

export interface LoginLogItem {
  /** 原始时间（ISO） */
  time: string
  ip: string
  /** loopback / lan / public */
  ipScope: string
  userAgent: string
  device: string
  /** desktop / mobile / tablet / cli */
  deviceType: string
  /** password / email_code */
  method: string
  success: boolean
  error: string
  detail: string
}
