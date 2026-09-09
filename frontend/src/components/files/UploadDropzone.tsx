import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type DragEvent,
} from 'react'
import { toApiError } from '../../api/client'
import { createFolder, listRootFolders, type FolderItem } from '../../api/folders'
import { uploadFile } from '../../api/files'
import { formatBytes } from '../../lib/formatBytes'

type UploadState = 'queued' | 'uploading' | 'done' | 'error'

type UploadTask = {
  id: string
  file: File
  progress: number
  status: UploadState
  error?: string
}

function taskID(file: File): string {
  return `${file.name}-${file.size}-${file.lastModified}-${Math.random().toString(36).slice(2, 8)}`
}

type UploadDropzoneProps = {
  onUploaded?: () => void
}

export function UploadDropzone({ onUploaded }: UploadDropzoneProps) {
  const [isDragging, setIsDragging] = useState(false)
  const [tasks, setTasks] = useState<UploadTask[]>([])
  const [folders, setFolders] = useState<FolderItem[]>([])
  const [selectedFolderID, setSelectedFolderID] = useState<number | null>(null)
  const [folderError, setFolderError] = useState<string | null>(null)
  const [creatingFolder, setCreatingFolder] = useState(false)
  const [busy, setBusy] = useState(false)
  const inputRef = useRef<HTMLInputElement | null>(null)

  const loadFolders = useCallback(async () => {
    setFolderError(null)
    try {
      const rootFolders = await listRootFolders()
      setFolders(rootFolders)
      setSelectedFolderID((prev) => {
        if (prev && rootFolders.some((f) => f.id === prev)) return prev
        return rootFolders[0]?.id ?? null
      })
    } catch (err) {
      setFolderError(toApiError(err).message)
    }
  }, [])

  useEffect(() => {
    void loadFolders()
  }, [loadFolders])

  const queueFiles = useCallback(
    async (files: File[]) => {
      if (!selectedFolderID) {
        setFolderError('Create or choose a folder before uploading')
        return
      }

      const nextTasks = files.map<UploadTask>((file) => ({
        id: taskID(file),
        file,
        progress: 0,
        status: 'queued',
      }))
      setTasks((prev) => [...nextTasks, ...prev].slice(0, 12))
      setBusy(true)

      let hasSuccess = false
      for (const task of nextTasks) {
        setTasks((prev) =>
          prev.map((t) => (t.id === task.id ? { ...t, status: 'uploading', progress: 0, error: undefined } : t)),
        )
        try {
          await uploadFile(task.file, selectedFolderID, (loaded, total) => {
            const progress = total && total > 0 ? Math.round((loaded / total) * 100) : 0
            setTasks((prev) => prev.map((t) => (t.id === task.id ? { ...t, progress } : t)))
          })
          hasSuccess = true
          setTasks((prev) =>
            prev.map((t) => (t.id === task.id ? { ...t, status: 'done', progress: 100, error: undefined } : t)),
          )
        } catch (err) {
          setTasks((prev) =>
            prev.map((t) =>
              t.id === task.id
                ? { ...t, status: 'error', progress: 0, error: toApiError(err).message }
                : t,
            ),
          )
        }
      }

      setBusy(false)
      if (hasSuccess) onUploaded?.()
    },
    [onUploaded, selectedFolderID],
  )

  const onDrop = useCallback(
    (event: DragEvent<HTMLDivElement>) => {
      event.preventDefault()
      setIsDragging(false)
      const files = Array.from(event.dataTransfer.files ?? [])
      if (files.length > 0) {
        void queueFiles(files)
      }
    },
    [queueFiles],
  )

  const onInputChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(event.target.files ?? [])
      if (files.length > 0) {
        void queueFiles(files)
      }
      event.target.value = ''
    },
    [queueFiles],
  )

  const completedCount = useMemo(
    () => tasks.filter((t) => t.status === 'done').length,
    [tasks],
  )

  async function handleCreateUploadsFolder() {
    setCreatingFolder(true)
    setFolderError(null)
    try {
      const created = await createFolder('Uploads')
      await loadFolders()
      setSelectedFolderID(created.id)
    } catch (err) {
      setFolderError(toApiError(err).message)
    } finally {
      setCreatingFolder(false)
    }
  }

  return (
    <section className="space-y-4 rounded-2xl border border-slate-200/80 bg-white/90 p-4 shadow-sm sm:p-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-base font-semibold text-slate-900">Upload files</h2>
        <div className="flex items-center gap-2">
          <label className="text-xs font-medium uppercase tracking-wide text-slate-500" htmlFor="upload-folder-select">
            Target folder
          </label>
          <select
            id="upload-folder-select"
            className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-700"
            value={selectedFolderID ?? ''}
            onChange={(e) => setSelectedFolderID(Number(e.target.value))}
            disabled={folders.length === 0}
          >
            {folders.length === 0 ? <option value="">No folders</option> : null}
            {folders.map((folder) => (
              <option key={folder.id} value={folder.id}>
                {folder.name}
              </option>
            ))}
          </select>
          <button
            type="button"
            onClick={() => void handleCreateUploadsFolder()}
            disabled={creatingFolder}
            className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50 disabled:opacity-50"
          >
            {creatingFolder ? 'Creating...' : '+ Uploads folder'}
          </button>
        </div>
      </div>

      <div
        className={`rounded-2xl border-2 border-dashed px-6 py-10 text-center transition ${
          isDragging ? 'border-blue-500 bg-blue-50/70' : 'border-slate-200 bg-slate-50/50'
        }`}
        onDragOver={(e) => {
          e.preventDefault()
          setIsDragging(true)
        }}
        onDragLeave={() => setIsDragging(false)}
        onDrop={onDrop}
      >
        <p className="text-sm font-medium text-slate-800">
          Drag and drop files here
        </p>
        <p className="mt-1 text-sm text-slate-500">or click to choose multiple files</p>
        <button
          type="button"
          className="mt-4 rounded-lg bg-slate-900 px-4 py-2 text-sm font-semibold text-white transition hover:bg-slate-800"
          onClick={() => inputRef.current?.click()}
        >
          Select files
        </button>
        <input ref={inputRef} type="file" multiple hidden onChange={onInputChange} />
      </div>

      {folderError ? (
        <p className="text-sm text-red-700">{folderError}</p>
      ) : null}

      {tasks.length > 0 ? (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium text-slate-700">
              Upload queue {completedCount > 0 ? `· done ${completedCount}/${tasks.length}` : ''}
            </p>
            {busy ? <span className="text-xs text-slate-500">Uploading...</span> : null}
          </div>
          <ul className="space-y-2">
            {tasks.map((task) => (
              <li key={task.id} className="rounded-xl border border-slate-200 bg-white px-3 py-2">
                <div className="flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-slate-900">{task.file.name}</p>
                    <p className="text-xs text-slate-500">{formatBytes(task.file.size)}</p>
                  </div>
                  <span
                    className={`shrink-0 text-xs font-semibold ${
                      task.status === 'done'
                        ? 'text-emerald-600'
                        : task.status === 'error'
                          ? 'text-red-600'
                          : 'text-slate-500'
                    }`}
                  >
                    {task.status === 'queued' ? 'Queued' : task.status === 'uploading' ? `${task.progress}%` : task.status === 'done' ? 'Done' : 'Failed'}
                  </span>
                </div>
                <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-slate-100">
                  <div
                    className={`h-full transition-all ${
                      task.status === 'error'
                        ? 'bg-red-500'
                        : task.status === 'done'
                          ? 'bg-emerald-500'
                          : 'bg-blue-500'
                    }`}
                    style={{ width: `${Math.max(task.progress, task.status === 'done' ? 100 : 3)}%` }}
                  />
                </div>
                {task.error ? <p className="mt-1 text-xs text-red-700">{task.error}</p> : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  )
}
