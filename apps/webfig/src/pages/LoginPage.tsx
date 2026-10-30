import { LockKeyhole, Network, ShieldCheck } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { ApiError } from '../services/api'

export function LoginPage() {
  const { login, session, loading } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  if (!loading && session) return <Navigate to="/" replace />

  const submit = async (event: FormEvent) => {
    event.preventDefault(); setError(''); setSubmitting(true)
    try {
      await login(username, password)
      const destination = (location.state as { from?: { pathname?: string } } | null)?.from?.pathname ?? '/'
      navigate(destination, { replace: true })
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : 'Login tidak dapat diproses')
    } finally { setSubmitting(false) }
  }

  return <main className="relative grid min-h-screen place-items-center overflow-hidden bg-[#07101d] px-4 py-10 text-white"><div className="absolute -left-40 top-1/3 h-96 w-96 rounded-full bg-cyan-500/10 blur-3xl"/><div className="absolute -right-40 bottom-1/4 h-96 w-96 rounded-full bg-violet-500/10 blur-3xl"/><section className="relative w-full max-w-md"><div className="mb-7 flex items-center justify-center gap-3"><div className="grid h-12 w-12 place-items-center rounded-2xl bg-gradient-to-br from-cyan-300 to-cyan-600 text-slate-950 shadow-xl shadow-cyan-500/20"><Network size={25}/></div><div><p className="text-xl font-black tracking-[.14em]">CODENET</p><p className="text-[9px] font-bold uppercase tracking-[.2em] text-cyan-400">Intelligent Network OS</p></div></div><div className="rounded-3xl border border-slate-700/70 bg-[#0b1726]/95 p-7 shadow-2xl shadow-black/30 backdrop-blur md:p-9"><div className="mb-7"><div className="mb-5 grid h-11 w-11 place-items-center rounded-xl bg-cyan-400/10 text-cyan-300"><LockKeyhole size={21}/></div><h1 className="text-2xl font-bold">Administrator login</h1><p className="mt-2 text-sm leading-6 text-slate-400">Masuk untuk mengakses monitoring dan pengelolaan CODENET Router.</p></div><form onSubmit={submit} className="space-y-5"><label className="block"><span className="mb-2 block text-xs font-semibold text-slate-300">Username</span><input aria-label="Username" autoComplete="username" value={username} onChange={(event)=>setUsername(event.target.value)} required maxLength={64} className="w-full rounded-xl border border-slate-700 bg-slate-950/60 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-600 focus:border-cyan-400 focus:ring-2 focus:ring-cyan-400/20" placeholder="Administrator CODENET"/></label><label className="block"><span className="mb-2 block text-xs font-semibold text-slate-300">Password</span><input aria-label="Password" type="password" autoComplete="current-password" value={password} onChange={(event)=>setPassword(event.target.value)} required maxLength={1024} className="w-full rounded-xl border border-slate-700 bg-slate-950/60 px-4 py-3 text-sm text-white outline-none transition placeholder:text-slate-600 focus:border-cyan-400 focus:ring-2 focus:ring-cyan-400/20" placeholder="Masukkan password"/></label>{error && <div role="alert" className="rounded-xl border border-rose-800/70 bg-rose-950/40 px-4 py-3 text-sm text-rose-200">{error}</div>}<button disabled={submitting} className="flex w-full items-center justify-center gap-2 rounded-xl bg-cyan-400 px-4 py-3 text-sm font-bold text-slate-950 transition hover:bg-cyan-300 disabled:cursor-not-allowed disabled:opacity-60">{submitting ? 'Memverifikasi…' : 'Masuk ke WebFig'}</button></form><div className="mt-6 flex items-start gap-2 border-t border-slate-800 pt-5 text-xs leading-5 text-slate-500"><ShieldCheck size={16} className="mt-0.5 shrink-0 text-emerald-400"/><span>Session dilindungi cookie HttpOnly. CODENET tidak menyimpan password di browser.</span></div></div><p className="mt-6 text-center text-[10px] font-semibold uppercase tracking-[.18em] text-slate-600">CodeSyariah · CODENET WebFig v0.1.1</p></section></main>
}
