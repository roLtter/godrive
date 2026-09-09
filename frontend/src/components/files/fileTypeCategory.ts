export type FileTypeCategory =
  | 'image'
  | 'video'
  | 'audio'
  | 'pdf'
  | 'word'
  | 'sheet'
  | 'slides'
  | 'archive'
  | 'code'
  | 'text'
  | 'file'

export function getFileTypeCategory(mime: string, filename: string): FileTypeCategory {
  const m = mime.toLowerCase()
  const ext = filename.includes('.') ? (filename.split('.').pop()?.toLowerCase() ?? '') : ''

  if (m.startsWith('image/')) return 'image'
  if (m.startsWith('video/')) return 'video'
  if (m.startsWith('audio/')) return 'audio'
  if (m === 'application/pdf') return 'pdf'

  if (
    m.includes('wordprocessingml') ||
    m.includes('msword') ||
    ext === 'doc' ||
    ext === 'docx' ||
    ext === 'odt'
  ) {
    return 'word'
  }

  if (
    m.includes('spreadsheetml') ||
    m.includes('ms-excel') ||
    ext === 'xls' ||
    ext === 'xlsx' ||
    ext === 'csv' ||
    ext === 'ods'
  ) {
    return 'sheet'
  }

  if (
    m.includes('presentationml') ||
    m.includes('ms-powerpoint') ||
    ext === 'ppt' ||
    ext === 'pptx' ||
    ext === 'odp'
  ) {
    return 'slides'
  }

  if (
    m.includes('zip') ||
    m.includes('compressed') ||
    m.includes('x-rar') ||
    m.includes('x-7z') ||
    ['zip', 'rar', '7z', 'tar', 'gz'].includes(ext)
  ) {
    return 'archive'
  }

  if (
    m.startsWith('text/html') ||
    m.startsWith('text/xml') ||
    m.includes('javascript') ||
    m.includes('typescript') ||
    ['ts', 'tsx', 'js', 'jsx', 'py', 'go', 'rs', 'java', 'c', 'cpp', 'h', 'css', 'scss'].includes(ext)
  ) {
    return 'code'
  }

  if (m.startsWith('text/') || m === 'application/json' || ext === 'json' || ext === 'md' || ext === 'txt') {
    return 'text'
  }

  return 'file'
}
