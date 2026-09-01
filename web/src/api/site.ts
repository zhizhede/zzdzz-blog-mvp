import { http, type ApiResponse } from './http'

export interface IconMeta {
  custom: boolean
  updated_at: string // unix 秒字符串; 空串表示无自定义图标
}

export const siteApi = {
  getIconMeta: () => http.get<any, ApiResponse<IconMeta>>('/site/icon/meta'),
  uploadIcon: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return http.put<any, ApiResponse<{ updated_at: number }>>('/site/icon', form)
  },
  resetIcon: () => http.delete<any, ApiResponse<null>>('/site/icon'),
}
