import { apiClient } from './client'

export type FileListItem = {
  id: number
  folder_id: number
  name: string
  size: number
  mime: string
  created_at: string
}

export type FileListResponse = {
  items: FileListItem[]
  page: number
  limit: number
  total: number
  total_pages: number
}

export type ListFilesParams = {
  page?: number
  limit?: number
  folder_id?: number
  sort_by?: 'name' | 'size' | 'created_at'
  sort_order?: 'asc' | 'desc'
}

export type UploadResult = {
  id: number
  folder_id: number
  name: string
  size: number
  mime: string
  s3_key: string
  created_at: string
}

export async function listFiles(params: ListFilesParams = {}): Promise<FileListResponse> {
  const { data } = await apiClient.get<FileListResponse>('/api/files', {
    params: {
      page: params.page ?? 1,
      limit: params.limit ?? 20,
      ...(params.folder_id != null ? { folder_id: params.folder_id } : {}),
      sort_by: params.sort_by ?? 'created_at',
      sort_order: params.sort_order ?? 'desc',
    },
  })
  return data
}

export async function uploadFile(
  file: File,
  folderID: number,
  onProgress?: (loaded: number, total?: number) => void,
): Promise<UploadResult> {
  const form = new FormData()
  form.append('folder_id', String(folderID))
  form.append('file', file)

  const { data } = await apiClient.post<UploadResult>('/api/files/upload', form, {
    onUploadProgress: (event) => {
      onProgress?.(event.loaded, event.total)
    },
  })
  return data
}
