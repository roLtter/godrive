import { Outlet } from 'react-router-dom'
import { BreadcrumbProvider } from '../../contexts/BreadcrumbContext'
import { AppHeader } from './AppHeader'
import { Sidebar } from './Sidebar'

export function AppShell() {
  return (
    <BreadcrumbProvider>
      <div className="flex min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/90 text-slate-900">
        <Sidebar />
        <div className="flex min-w-0 flex-1 flex-col">
          <AppHeader />
          <main className="flex-1 overflow-auto">
            <Outlet />
          </main>
        </div>
      </div>
    </BreadcrumbProvider>
  )
}
