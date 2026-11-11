import request from '@/utils/request'
import type {
  CreateThemeRequest,
  CreateThemeResponse,
  DeleteThemeResponse,
  DuplicateThemeRequest,
  DuplicateThemeResponse,
  ExportThemePackageRequest,
  ExportThemePackageResponse,
  GetSiteThemeResponse,
  GetThemeResponse,
  ImportThemePackageRequest,
  ImportThemePackageResponse,
  InspectThemePackageRequest,
  InspectThemePackageResponse,
  ListThemesResponse,
  SetSiteThemeRequest,
  SetSiteThemeResponse,
  UpdateThemeRequest,
  UpdateThemeResponse,
  UploadThemeAssetRequest,
  UploadThemeAssetResponse,
} from '@/types/v1/theme'

// 自定义外观主题（只对自己生效；「设为全站默认」需系统设置编辑权限）。二进制内容用 base64 字符串传输。

export const listThemes = () => request.get<ListThemesResponse>('/user/me/themes')

export const getTheme = (id: string) => request.get<GetThemeResponse>(`/user/me/themes/${encodeURIComponent(id)}`)

export const createTheme = (data: CreateThemeRequest) => request.post<CreateThemeResponse>('/user/me/themes', data)

export const updateTheme = (data: UpdateThemeRequest) =>
  request.put<UpdateThemeResponse>(`/user/me/themes/${encodeURIComponent(data.id)}`, data)

export const deleteTheme = (id: string) =>
  request.delete<DeleteThemeResponse>(`/user/me/themes/${encodeURIComponent(id)}`)

export const duplicateTheme = (data: DuplicateThemeRequest) =>
  request.post<DuplicateThemeResponse>(`/user/me/themes/${encodeURIComponent(data.id)}/duplicate`, data)

/** 上传背景图 / 字体（图片服务端转 WebP），大文件给足超时 */
export const uploadThemeAsset = (data: UploadThemeAssetRequest) =>
  request.post<UploadThemeAssetResponse>('/user/me/theme-assets', data, { timeout: 120_000 })

export const inspectThemePackage = (data: InspectThemePackageRequest) =>
  request.post<InspectThemePackageResponse>('/user/me/themes/inspect', data, { timeout: 120_000 })

export const importThemePackage = (data: ImportThemePackageRequest) =>
  request.post<ImportThemePackageResponse>('/user/me/themes/import', data, { timeout: 120_000 })

export const exportThemePackage = (data: ExportThemePackageRequest) =>
  request.post<ExportThemePackageResponse>('/user/me/themes/export', data, { timeout: 120_000 })

/** 全站默认主题（公开接口，登录页使用） */
export const getSiteTheme = () => request.get<GetSiteThemeResponse>('/public/site-theme')

export const setSiteTheme = (data: SetSiteThemeRequest) => request.put<SetSiteThemeResponse>('/system/site-theme', data)

// ---- 文件 ↔ base64 ----

export const fileToBase64 = (file: Blob): Promise<string> =>
  new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const s = String(reader.result || '')
      resolve(s.slice(s.indexOf(',') + 1))
    }
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })

export const downloadBase64 = (base64: string, fileName: string, mime = 'application/zip') => {
  const bin = atob(base64)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  const url = URL.createObjectURL(new Blob([bytes], { type: mime }))
  const a = document.createElement('a')
  a.href = url
  a.download = fileName
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
