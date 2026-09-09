import type { InputHTMLAttributes } from 'react'

type AuthInputProps = InputHTMLAttributes<HTMLInputElement> & {
  label: string
  error?: string
}

export function AuthInput({ label, error, ...inputProps }: AuthInputProps) {
  return (
    <label className="block">
      <span className="mb-2 block text-sm font-medium text-slate-700">{label}</span>
      <input
        className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none transition placeholder:text-slate-500/90 focus:border-blue-600/80 focus:ring-4 focus:ring-blue-600/15"
        {...inputProps}
      />
      {error ? <span className="mt-2 block text-xs text-rose-600">{error}</span> : null}
    </label>
  )
}
