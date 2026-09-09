import { useNavigate } from 'react-router-dom'
import { logout } from '../../api/auth'
import { useCurrentUser } from '../../contexts/CurrentUserContext'
import { BreadcrumbsBar } from './BreadcrumbsBar'

function initialsFromEmail(email: string) {
  const local = email.split('@')[0] ?? '?'
  const parts = local.split(/[._-]+/).filter(Boolean)
  if (parts.length >= 2) {
    return (parts[0]![0]! + parts[1]![0]!).toUpperCase()
  }
  return local.slice(0, 2).toUpperCase() || '?'
}

export function AppHeader() {
  const navigate = useNavigate()
  const { user, status } = useCurrentUser()

  async function handleLogout() {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <header className="sticky top-0 z-20 flex h-14 shrink-0 items-center justify-between gap-4 border-b border-slate-200/80 bg-white/85 px-4 backdrop-blur-md sm:px-6">
      <div className="min-w-0 flex-1">
        <BreadcrumbsBar />
      </div>

      <div className="flex shrink-0 items-center gap-3">
        {status === 'loading' || !user ? (
          <div className="flex items-center gap-2">
            <div className="h-9 w-9 animate-pulse rounded-full bg-slate-200" />
            <div className="hidden h-4 w-28 animate-pulse rounded bg-slate-200 sm:block" />
          </div>
        ) : (
          <>
            <div className="hidden text-right text-sm sm:block">
              <p className="truncate font-medium text-slate-900">{user.email}</p>
              <p className="truncate text-xs text-slate-500">Signed in</p>
            </div>
            <div
              className="flex h-9 w-9 items-center justify-center rounded-full bg-gradient-to-br from-slate-800 to-slate-600 text-xs font-semibold text-white shadow-inner"
              aria-hidden
            >
              {initialsFromEmail(user.email)}
            </div>
            <button
              type="button"
              onClick={() => void handleLogout()}
              className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-semibold text-slate-700 shadow-sm transition hover:border-slate-300 hover:bg-slate-50"
            >
              Sign out
            </button>
          </>
        )}
      </div>
    </header>
  )
}
