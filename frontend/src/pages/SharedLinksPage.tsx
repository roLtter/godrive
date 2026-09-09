import { useEffect } from 'react'
import { useBreadcrumbs } from '../contexts/BreadcrumbContext'

export function SharedLinksPage() {
  const { setBreadcrumbs } = useBreadcrumbs()

  useEffect(() => {
    setBreadcrumbs([
      { label: 'My Drive', href: '/' },
      { label: 'Shared links' },
    ])
  }, [setBreadcrumbs])

  return (
    <div className="p-6 sm:p-8">
      <h1 className="text-2xl font-semibold tracking-tight text-slate-900">Shared links</h1>
      <p className="mt-2 text-slate-600">Manage public links once the shares API is connected in the UI.</p>
    </div>
  )
}
