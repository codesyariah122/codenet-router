import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'

export function StatusBadge({ status }: { status: string }) {
  const active = ['up', 'active', 'online'].includes(status.toLowerCase())
  return <span className={`inline-flex items-center gap-2 rounded-full border px-2.5 py-1 text-xs font-semibold capitalize ${active ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-950/40 dark:text-emerald-300' : 'border-slate-200 bg-slate-100 text-slate-600 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300'}`}><span className={`h-1.5 w-1.5 rounded-full ${active ? 'bg-emerald-500 shadow-[0_0_8px_#10b981]' : 'bg-slate-400'}`} />{status}</span>
}

export function PageHeader({ eyebrow, title, description, action }: { eyebrow: string; title: string; description: string; action?: ReactNode }) {
  return <div className="mb-7 flex flex-col justify-between gap-4 md:flex-row md:items-end"><div><p className="mb-2 text-xs font-bold uppercase tracking-[.2em] text-cyan-600 dark:text-cyan-400">{eyebrow}</p><h1 className="text-2xl font-bold tracking-tight text-slate-950 dark:text-white md:text-3xl">{title}</h1><p className="mt-2 max-w-2xl text-sm leading-6 text-slate-500 dark:text-slate-400">{description}</p></div>{action}</div>
}

export function StatCard({ icon: Icon, label, value, detail, tone = 'cyan' }: { icon: LucideIcon; label: string; value: string; detail: string; tone?: 'cyan' | 'violet' | 'amber' | 'emerald' }) {
  const tones = { cyan: 'bg-cyan-50 text-cyan-600 dark:bg-cyan-950/40 dark:text-cyan-300', violet: 'bg-violet-50 text-violet-600 dark:bg-violet-950/40 dark:text-violet-300', amber: 'bg-amber-50 text-amber-600 dark:bg-amber-950/40 dark:text-amber-300', emerald: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-300' }
  return <div className="panel p-5"><div className="flex items-start justify-between"><div className={`rounded-xl p-2.5 ${tones[tone]}`}><Icon size={19} /></div><span className="text-[10px] font-bold uppercase tracking-widest text-slate-400">Live</span></div><p className="mt-5 text-xs font-semibold uppercase tracking-wider text-slate-500">{label}</p><p className="mt-1.5 text-2xl font-bold tracking-tight text-slate-950 dark:text-white">{value}</p><p className="mt-2 text-xs text-slate-500 dark:text-slate-400">{detail}</p></div>
}

export function LoadingState() { return <div className="grid min-h-[420px] place-items-center"><div className="text-center"><div className="mx-auto h-9 w-9 animate-spin rounded-full border-2 border-cyan-500 border-t-transparent" /><p className="mt-4 text-sm text-slate-500">Menghubungkan ke router…</p></div></div> }
export function ErrorState({ error, retry }: { error: Error; retry: () => void }) { return <div className="panel grid min-h-[360px] place-items-center p-8 text-center"><div><p className="font-semibold text-slate-900 dark:text-white">Data router tidak tersedia</p><p className="mt-2 text-sm text-slate-500">{error.message}</p><button onClick={retry} className="mt-5 rounded-lg bg-cyan-500 px-4 py-2 text-sm font-bold text-slate-950 hover:bg-cyan-400">Coba lagi</button></div></div> }
export function EmptyState({ title, description }: { title: string; description: string }) { return <div className="panel p-12 text-center"><p className="font-semibold text-slate-900 dark:text-white">{title}</p><p className="mt-2 text-sm text-slate-500">{description}</p></div> }
