import { Link } from 'react-router-dom'
import { useBreadcrumbs } from '../../contexts/BreadcrumbContext'

function Chevron() {
  return (
    <svg
      className="mx-1 h-4 w-4 shrink-0 text-slate-300"
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden
    >
      <path d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" />
    </svg>
  )
}

export function BreadcrumbsBar() {
  const { items } = useBreadcrumbs()

  return (
    <nav aria-label="Breadcrumb" className="flex min-w-0 items-center text-sm">
      <ol className="flex min-w-0 items-center">
        {items.map((item, i) => {
          const isLast = i === items.length - 1
          return (
            <li key={`${item.label}-${i}`} className="flex min-w-0 items-center">
              {i > 0 ? <Chevron /> : null}
              {isLast || !item.href ? (
                <span
                  className={`truncate font-medium ${isLast ? 'text-slate-900' : 'text-slate-500'}`}
                >
                  {item.label}
                </span>
              ) : (
                <Link
                  to={item.href}
                  className="truncate text-slate-500 transition hover:text-slate-800"
                >
                  {item.label}
                </Link>
              )}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}
