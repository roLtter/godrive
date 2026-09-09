import type { PropsWithChildren } from 'react'
import { motion } from 'framer-motion'

type AuthLayoutProps = PropsWithChildren<{
  title: string
  subtitle: string
}>

export function AuthLayout({ title, subtitle, children }: AuthLayoutProps) {
  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-gradient-to-b from-slate-50 to-slate-100 px-4 py-8 text-slate-900">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_10%_15%,rgba(191,219,254,0.45),transparent_28%),radial-gradient(circle_at_85%_85%,rgba(196,181,253,0.28),transparent_30%)]" />
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(148,163,184,0.14)_1px,transparent_1px),linear-gradient(90deg,rgba(148,163,184,0.14)_1px,transparent_1px)] bg-[length:40px_40px] opacity-35 [mask-image:radial-gradient(circle_at_center,black_35%,transparent_90%)]" />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_20%_25%,rgba(59,130,246,0.11),transparent_25%),radial-gradient(circle_at_75%_30%,rgba(99,102,241,0.10),transparent_22%),radial-gradient(circle_at_50%_80%,rgba(14,165,233,0.08),transparent_24%)] blur-md" />

      <motion.div
        className="absolute -left-24 top-16 h-64 w-64 rounded-full bg-slate-300/30 blur-3xl"
        animate={{ x: [0, 14, 0], y: [0, 20, 0] }}
        transition={{ repeat: Number.POSITIVE_INFINITY, duration: 12, ease: 'easeInOut' }}
      />
      <motion.div
        className="absolute -right-24 bottom-12 h-72 w-72 rounded-full bg-blue-200/35 blur-3xl"
        animate={{ x: [0, -18, 0], y: [0, -12, 0] }}
        transition={{ repeat: Number.POSITIVE_INFINITY, duration: 14, ease: 'easeInOut' }}
      />

      <motion.section
        initial={{ opacity: 0, y: 24, scale: 0.98 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ duration: 0.45, ease: 'easeOut' }}
        className="relative w-full max-w-md rounded-3xl bg-gradient-to-br from-slate-300/60 via-slate-200/40 to-slate-300/60 p-px"
      >
        <div className="rounded-[calc(1.5rem-1px)] border border-slate-200 bg-white p-8 shadow-[0_14px_40px_rgba(15,23,42,0.08)]">
          <h1 className="text-3xl font-semibold tracking-tight text-slate-900">{title}</h1>
          <p className="mt-2 text-sm text-slate-600">{subtitle}</p>
          <div className="mt-8">{children}</div>
        </div>
      </motion.section>
    </main>
  )
}
