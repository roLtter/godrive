import { useEffect } from 'react'
import { useBreadcrumbs } from '../contexts/BreadcrumbContext'

export function TrashPage() {
  const { setBreadcrumbs } = useBreadcrumbs()

  useEffect(() => {
    setBreadcrumbs([
      { label: 'My Drive', href: '/' },
      { label: 'Trash' },
    ])
  }, [setBreadcrumbs])

  return (
    <div className="p-6 sm:p-8">
      <h1 className="text-2xl font-semibold tracking-tight text-slate-900">Trash</h1>
      <p className="mt-2 text-slate-600">Deleted files will appear here after the file list is wired up.</p>
    </div>
  )
}
