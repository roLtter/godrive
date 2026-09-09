import { useCallback, useEffect, useState } from 'react'
import { toApiError } from '../../api/client'
import { listFiles, type FileListItem, type FileListResponse } from '../../api/files'
import { formatBytes } from '../../lib/formatBytes'
import { getFileTypeCategory } from './fileTypeCategory'
import { FileTypeIcon } from './FileTypeIcon'

const VIEW_STORAGE_KEY = 'godrive_files_view_mode'

type ViewMode = 'grid' | 'list'

function readStoredView(): ViewMode {
  try {
    const v = localStorage.getItem(VIEW_STORAGE_KEY)
    if (v === 'grid' || v === 'list') return v
  } catch {
    /* ignore */
  }
  return 'grid'
}

function formatShortDate(iso: string): string {
  try {
    const d = new Date(iso)
    return d.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      year: d.getFullYear() !== new Date().getFullYear() ? 'numeric' : undefined,
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return iso
  }
}

function ViewToggle({ mode, onChange }: { mode: ViewMode; onChange: (m: ViewMode) => void }) {
  const btn =
    'inline-flex items-center justify-center rounded-lg p-2 text-slate-500 transition hover:bg-slate-100 hover:text-slate-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-400'
  const active = 'bg-slate-900 text-white shadow-sm hover:bg-slate-800 hover:text-white'

  return (
    <div
      className="inline-flex rounded-xl border border-slate-200/80 bg-white p-0.5 shadow-sm"
      role="group"
      aria-label="View mode"
    >
      <button
        type="button"
        aria-pressed={mode === 'grid'}
        className={`${btn} ${mode === 'grid' ? active : ''}`}
        onClick={() => onChange('grid')}
        title="Grid view"
      >
        <svg className="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
          <path d="M4 4h7v7H4V4zm9 0h7v7h-7V4zM4 13h7v7H4v-7zm9 0h7v7h-7v-7z" />
        </svg>
      </button>
      <button
        type="button"
        aria-pressed={mode === 'list'}
        className={`${btn} ${mode === 'list' ? active : ''}`}
        onClick={() => onChange('list')}
        title="List view"
      >
        <svg className="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
          <path d="M4 6h16v2H4V6zm0 5h16v2H4v-2zm0 5h16v2H4v-2z" />
        </svg>
      </button>
    </div>
  )
}

function FileGrid({ items }: { items: FileListItem[] }) {
  return (
    <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
      {items.map((file) => {
        const cat = getFileTypeCategory(file.mime, file.name)
        return (
          <li key={file.id}>
            <div className="flex flex-col rounded-2xl border border-slate-200/80 bg-white/90 p-4 shadow-sm ring-1 ring-transparent transition hover:border-slate-300 hover:shadow-md hover:ring-slate-200/60">
              <div className="mx-auto">
                <FileTypeIcon category={cat} size="lg" />
              </div>
              <p className="mt-3 line-clamp-2 text-center text-sm font-medium text-slate-900" title={file.name}>
                {file.name}
              </p>
              <p className="mt-1 text-center text-xs text-slate-500">{formatBytes(file.size)}</p>
            </div>
          </li>
        )
      })}
    </ul>
  )
}

function FileTable({ items }: { items: FileListItem[] }) {
  return (
    <div className="overflow-hidden rounded-2xl border border-slate-200/80 bg-white/90 shadow-sm">
      <table className="w-full min-w-[32rem] text-left text-sm">
        <thead className="border-b border-slate-100 bg-slate-50/80 text-xs font-semibold uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-4 py-3" scope="col">
              Name
            </th>
            <th className="hidden px-4 py-3 sm:table-cell" scope="col">
              Type
            </th>
            <th className="px-4 py-3 text-right" scope="col">
              Size
            </th>
            <th className="hidden px-4 py-3 md:table-cell" scope="col">
              Added
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {items.map((file) => {
            const cat = getFileTypeCategory(file.mime, file.name)
            return (
              <tr key={file.id} className="transition hover:bg-slate-50/80">
                <td className="px-4 py-3">
                  <div className="flex min-w-0 items-center gap-3">
                    <FileTypeIcon category={cat} size="sm" className="hidden sm:flex" />
                    <span className="truncate font-medium text-slate-900" title={file.name}>
                      {file.name}
                    </span>
                  </div>
                </td>
                <td className="hidden px-4 py-3 text-slate-600 sm:table-cell">
                  <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-700">
                    {file.mime || 'unknown'}
                  </span>
                </td>
                <td className="whitespace-nowrap px-4 py-3 text-right text-slate-600">{formatBytes(file.size)}</td>
                <td className="hidden px-4 py-3 text-slate-500 md:table-cell">{formatShortDate(file.created_at)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

type FileListPanelProps = {
  refreshVersion?: number
}

export function FileListPanel({ refreshVersion = 0 }: FileListPanelProps) {
  const [viewMode, setViewModeState] = useState<ViewMode>(readStoredView)
  const [page, setPage] = useState(1)
  const [data, setData] = useState<FileListResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const setViewMode = useCallback((m: ViewMode) => {
    setViewModeState(m)
    try {
      localStorage.setItem(VIEW_STORAGE_KEY, m)
    } catch {
      /* ignore */
    }
  }, [])

  const load = useCallback(async (p: number) => {
    setLoading(true)
    setError(null)
    try {
      const res = await listFiles({ page: p, limit: 24 })
      setData(res)
    } catch (e) {
      setError(toApiError(e).message)
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load(page)
  }, [load, page, refreshVersion])

  const items = data?.items ?? []
  const totalPages = data?.total_pages ?? 0

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-slate-900">My Drive</h1>
          <p className="mt-1 text-sm text-slate-600">
            {data != null ? (
              <>
                {data.total} {data.total === 1 ? 'file' : 'files'}
                {totalPages > 1 ? ` · page ${data.page} of ${totalPages}` : null}
              </>
            ) : (
              'Loading your library…'
            )}
          </p>
        </div>
        <ViewToggle mode={viewMode} onChange={setViewMode} />
      </div>

      {error ? (
        <div className="rounded-2xl border border-red-200 bg-red-50/80 px-4 py-3 text-sm text-red-800">
          <p className="font-medium">Could not load files</p>
          <p className="mt-1 text-red-700">{error}</p>
          <button
            type="button"
            className="mt-3 rounded-lg bg-red-900 px-3 py-1.5 text-xs font-semibold text-white hover:bg-red-800"
            onClick={() => void load(page)}
          >
            Retry
          </button>
        </div>
      ) : null}

      {loading && !data ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="h-40 animate-pulse rounded-2xl bg-slate-200/70" />
          ))}
        </div>
      ) : null}

      {!loading && items.length === 0 && !error ? (
        <div className="rounded-2xl border border-dashed border-slate-200 bg-white/60 px-6 py-16 text-center">
          <p className="text-lg font-medium text-slate-800">No files yet</p>
          <p className="mx-auto mt-2 max-w-md text-sm text-slate-600">
            Upload something with the API (or the upcoming uploader) — it will show up here with the right icon
            and size.
          </p>
        </div>
      ) : null}

      {items.length > 0 ? (
        <div className={loading && data ? 'pointer-events-none opacity-55 transition-opacity' : 'transition-opacity'}>
          {viewMode === 'grid' ? (
            <FileGrid items={items} />
          ) : (
            <div className="overflow-x-auto">
              <FileTable items={items} />
            </div>
          )}
        </div>
      ) : null}

      {totalPages > 1 ? (
        <div className="flex items-center justify-center gap-3 pt-2">
          <button
            type="button"
            disabled={page <= 1 || loading}
            className="rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
            onClick={() => setPage((p) => Math.max(1, p - 1))}
          >
            Previous
          </button>
          <span className="text-sm text-slate-600">
            Page {page} / {totalPages}
          </span>
          <button
            type="button"
            disabled={page >= totalPages || loading}
            className="rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
            onClick={() => setPage((p) => p + 1)}
          >
            Next
          </button>
        </div>
      ) : null}
    </div>
  )
}
