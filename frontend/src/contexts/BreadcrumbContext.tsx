import { createContext, useCallback, useContext, useMemo, useState, type PropsWithChildren } from 'react'

export type BreadcrumbItem = {
  label: string
  href?: string
}

type BreadcrumbContextValue = {
  items: BreadcrumbItem[]
  setBreadcrumbs: (items: BreadcrumbItem[]) => void
}

const BreadcrumbContext = createContext<BreadcrumbContextValue | null>(null)

const defaultCrumbs: BreadcrumbItem[] = [{ label: 'My Drive', href: '/' }]

export function BreadcrumbProvider({ children }: PropsWithChildren) {
  const [items, setItems] = useState<BreadcrumbItem[]>(defaultCrumbs)

  const setBreadcrumbs = useCallback((next: BreadcrumbItem[]) => {
    setItems(next.length > 0 ? next : defaultCrumbs)
  }, [])

  const value = useMemo(() => ({ items, setBreadcrumbs }), [items, setBreadcrumbs])

  return <BreadcrumbContext.Provider value={value}>{children}</BreadcrumbContext.Provider>
}

export function useBreadcrumbs() {
  const ctx = useContext(BreadcrumbContext)
  if (!ctx) {
    throw new Error('useBreadcrumbs must be used within BreadcrumbProvider')
  }
  return ctx
}
