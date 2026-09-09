import type { FileTypeCategory } from './fileTypeCategory'

const categoryStyles: Record<
  FileTypeCategory,
  { bg: string; ring: string; label: string }
> = {
  image: { bg: 'bg-fuchsia-500/15', ring: 'ring-fuchsia-500/20', label: 'Image' },
  video: { bg: 'bg-violet-500/15', ring: 'ring-violet-500/20', label: 'Video' },
  audio: { bg: 'bg-pink-500/15', ring: 'ring-pink-500/20', label: 'Audio' },
  pdf: { bg: 'bg-red-500/15', ring: 'ring-red-500/20', label: 'PDF' },
  word: { bg: 'bg-blue-500/15', ring: 'ring-blue-500/20', label: 'Document' },
  sheet: { bg: 'bg-emerald-500/15', ring: 'ring-emerald-500/20', label: 'Sheet' },
  slides: { bg: 'bg-amber-500/15', ring: 'ring-amber-500/20', label: 'Slides' },
  archive: { bg: 'bg-orange-500/15', ring: 'ring-orange-500/20', label: 'Archive' },
  code: { bg: 'bg-cyan-500/15', ring: 'ring-cyan-500/20', label: 'Code' },
  text: { bg: 'bg-slate-500/15', ring: 'ring-slate-500/20', label: 'Text' },
  file: { bg: 'bg-indigo-500/12', ring: 'ring-indigo-500/15', label: 'File' },
}

function IconGlyph({ category }: { category: FileTypeCategory }) {
  const common = 'h-[55%] w-[55%] text-current'
  switch (category) {
    case 'image':
      return (
        <svg className={common} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <rect x="3" y="5" width="18" height="14" rx="2" />
          <circle cx="8.5" cy="10" r="1.5" fill="currentColor" stroke="none" />
          <path d="M21 15l-5-5-5 5M3 17l6-6 4 4 3-3 5 5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      )
    case 'video':
      return (
        <svg className={common} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <rect x="2" y="6" width="15" height="12" rx="2" />
          <path d="M17 10l5-3v10l-5-3V10z" fill="currentColor" stroke="none" />
        </svg>
      )
    case 'audio':
      return (
        <svg className={common} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <path d="M9 18V5l12-2v13" strokeLinecap="round" />
          <circle cx="6" cy="18" r="3" />
          <circle cx="18" cy="16" r="3" />
        </svg>
      )
    case 'pdf':
      return (
        <svg className={common} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <path d="M7 3h7l5 5v13a2 2 0 01-2 2H7a2 2 0 01-2-2V5a2 2 0 012-2z" />
          <path d="M14 3v4h4M8 13h8M8 17h6" strokeLinecap="round" />
        </svg>
      )
    case 'word':
      return (
        <svg className={common} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <path d="M7 3h8l4 4v14a2 2 0 01-2 2H7a2 2 0 01-2-2V5a2 2 0 012-2z" />
          <path d="M15 3v4h4M9 12h6M9 16h4" strokeLinecap="round" />
        </svg>
      )
    case 'sheet':
      return (
        <svg className={common} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <rect x="3" y="4" width="18" height="16" rx="2" />
          <path d="M3 9h18M9 4v16M3 14h18" strokeLinecap="round" />
        </svg>
      )
    case 'slides':
      return (
        <svg className={common} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <rect x="3" y="4" width="18" height="14" rx="2" />
          <path d="M7 21h10" strokeLinecap="round" />
          <path d="M9 10h6M9 14h4" strokeLinecap="round" />
        </svg>
      )
    case 'archive':
      return (
        <svg className={common} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <path d="M8 3h8v4H8V3zM6 7h12v14a2 2 0 01-2 2H8a2 2 0 01-2-2V7z" />
          <path d="M10 11h4" strokeLinecap="round" />
        </svg>
      )
    case 'code':
      return (
        <svg className={common} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <path d="M8 9l-4 3 4 3M16 9l4 3-4 3M14 5l-4 14" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      )
    case 'text':
      return (
        <svg className={common} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <path d="M7 3h8l4 4v14a2 2 0 01-2 2H7a2 2 0 01-2-2V5a2 2 0 012-2z" />
          <path d="M9 12h6M9 16h6M9 8h4" strokeLinecap="round" />
        </svg>
      )
    default:
      return (
        <svg className={common} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <path d="M7 3h7l5 5v11a2 2 0 01-2 2H7a2 2 0 01-2-2V5a2 2 0 012-2z" />
          <path d="M14 3v4h4" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      )
  }
}

type FileTypeIconProps = {
  category: FileTypeCategory
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const sizeClass: Record<NonNullable<FileTypeIconProps['size']>, string> = {
  sm: 'h-9 w-9',
  md: 'h-12 w-12',
  lg: 'h-16 w-16',
}

export function FileTypeIcon({ category, size = 'md', className = '' }: FileTypeIconProps) {
  const styles = categoryStyles[category]
  return (
    <div
      className={`flex ${sizeClass[size]} shrink-0 items-center justify-center rounded-2xl ring-1 ${styles.bg} ${styles.ring} text-slate-700 ${className}`}
      title={styles.label}
      aria-hidden
    >
      <IconGlyph category={category} />
    </div>
  )
}
