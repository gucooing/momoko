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

export type MessageType = 'system' | 'user' | 'todo'

export interface UserMessageItem {
  id: string
  title?: string
  titleKey?: string
  content?: string
  contentKey?: string
  type: MessageType
  read: boolean
  time: string
  avatar?: string
}

export type UserProfileFormValue = {
  [K in keyof Pick<
    UpdateMeRequest,
    'username' | 'name' | 'email' | 'avatar' | 'bio' | 'tags'
  >]-?: NonNullable<UpdateMeRequest[K]>
}

export interface UserPasswordFormValue extends UpdatePasswordRequest {
  confirmPassword: string
}

export interface LoginDeviceRow extends LoginDevice, Record<string, unknown> {}

export interface LoginLogItem {
  detail: string
  userAgent: string
  ip: string
  operationTime: string
  success: boolean
}
