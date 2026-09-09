import { apiClient } from './client'

export type FolderItem = {
  id: number
  user_id: string
  parent_id?: number | null
  name: string
}

type ListFoldersResponse = {
  items: FolderItem[]
}

export async function listRootFolders(): Promise<FolderItem[]> {
  const { data } = await apiClient.get<ListFoldersResponse>('/api/folders')
  return data.items
}

export async function createFolder(name: string, parentID?: number): Promise<FolderItem> {
  const { data } = await apiClient.post<FolderItem>('/api/folders', {
    name,
    ...(parentID != null ? { parent_id: parentID } : {}),
  })
  return data
}
