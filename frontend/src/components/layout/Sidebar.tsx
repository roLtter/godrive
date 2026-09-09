import { NavLink } from 'react-router-dom'

const linkBase =
  'flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors'
const linkIdle = 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'
const linkActive = 'bg-slate-900 text-white shadow-sm shadow-slate-900/10'

export function Sidebar() {
  return (
    <aside className="flex w-64 shrink-0 flex-col border-r border-slate-200/80 bg-white/90 backdrop-blur-md">
      <div className="flex h-14 items-center border-b border-slate-100 px-5">
        <span className="text-lg font-semibold tracking-tight text-slate-900">GoDrive</span>
      </div>

      <nav className="flex flex-1 flex-col gap-1 p-3" aria-label="Main navigation">
        <NavLink to="/" end className={({ isActive }) => `${linkBase} ${isActive ? linkActive : linkIdle}`}>
          <span className="text-lg leading-none" aria-hidden>
            📁
          </span>
          My Drive
        </NavLink>
        <NavLink
          to="/shared"
          className={({ isActive }) => `${linkBase} ${isActive ? linkActive : linkIdle}`}
        >
          <span className="text-lg leading-none" aria-hidden>
            🔗
          </span>
          Shared links
        </NavLink>
        <NavLink
          to="/trash"
          className={({ isActive }) => `${linkBase} ${isActive ? linkActive : linkIdle}`}
        >
          <span className="text-lg leading-none" aria-hidden>
            🗑️
          </span>
          Trash
        </NavLink>
      </nav>

      <div className="border-t border-slate-100 p-4">
        <p className="text-xs font-medium uppercase tracking-wide text-slate-400">Storage</p>
        <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-slate-100">
          <div className="h-full w-0 rounded-full bg-gradient-to-r from-blue-500 to-indigo-500" />
        </div>
        <p className="mt-2 text-xs text-slate-500">Quota will appear here</p>
      </div>
    </aside>
  )
}
