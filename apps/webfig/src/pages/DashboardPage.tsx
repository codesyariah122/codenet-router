import { ArrowDown, ArrowUp, Cpu, HardDrive, RadioTower, Timer } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../services/api'
import { ErrorState, LoadingState, PageHeader, StatCard, StatusBadge } from '../components/ui'
import { TrafficChart } from '../components/TrafficChart'
import { formatBytes, formatUptime } from '../utils/format'

export function DashboardPage() {
  const query = useQuery({ queryKey: ['dashboard'], queryFn: api.dashboard, refetchInterval: 15_000 })
  if (query.isLoading) return <LoadingState />
  if (query.error || !query.data) return <ErrorState error={query.error ?? new Error('Data kosong')} retry={() => query.refetch()} />
  const d = query.data
  const wan = d.interfaces.find((item) => item.role === 'WAN')
  const lan = d.interfaces.find((item) => item.role === 'LAN')
  const featured = [wan, lan].filter((item): item is NonNullable<typeof item> => Boolean(item))
  const visibleInterfaces = featured.length > 0 ? featured : d.interfaces.filter((item) => item.status === 'up').slice(0, 2)
  const pppoeSessionsLabel = d.pppoe.session_count_ready ? `${d.pppoe.active_sessions} sesi` : 'Tidak tersedia'
  return <>
    <PageHeader eyebrow="Network command center" title={`Selamat datang di ${d.system.hostname}`} description="Pantau kesehatan sistem, interface, dan pelanggan PPPoE dari satu tampilan operasional." action={<div className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-xs text-slate-500 dark:border-slate-800 dark:bg-slate-900"><span className="font-semibold text-slate-900 dark:text-white">{d.system.os}</span> · {d.system.version}</div>} />
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4"><StatCard icon={Cpu} label="CPU utilization" value={`${d.resources.cpu_percent.toFixed(1)}%`} detail={`Kernel ${d.system.kernel}`} tone="cyan" /><StatCard icon={HardDrive} label="Memory" value={`${d.resources.memory_percent.toFixed(1)}%`} detail={`${formatBytes(d.resources.memory_used_bytes)} / ${formatBytes(d.resources.memory_total_bytes)}`} tone="violet" /><StatCard icon={Timer} label="System uptime" value={formatUptime(d.system.uptime_seconds)} detail="Beroperasi tanpa interupsi" tone="amber" /><StatCard icon={RadioTower} label="PPPoE aktif" value={pppoeSessionsLabel} detail={`${d.pppoe.service_name} · ${d.pppoe.control_status}`} tone="emerald" /></div>
    <div className="mt-5 grid gap-5 xl:grid-cols-[minmax(0,1.6fr)_minmax(310px,.7fr)]"><TrafficChart data={d.traffic_history} /><div className="panel"><div className="border-b border-slate-200 px-5 py-4 dark:border-slate-800"><p className="font-semibold text-slate-900 dark:text-white">Interface aktif</p><p className="mt-1 text-xs text-slate-500">Status konektivitas router</p></div><div className="divide-y divide-slate-200 dark:divide-slate-800">{visibleInterfaces.map((item) => <div key={item.name} className="p-5"><div className="flex items-center justify-between"><div><div className="flex items-center gap-2"><span className="font-mono text-sm font-bold">{item.name}</span><span className="rounded bg-slate-100 px-1.5 py-0.5 text-[9px] font-bold uppercase text-slate-500 dark:bg-slate-800">{item.role}</span></div><p className="mt-1.5 font-mono text-xs text-slate-500">{item.ipv4_addresses[0] ?? 'Alamat tidak tersedia'}</p></div><StatusBadge status={item.status} /></div><div className="mt-4 grid grid-cols-2 gap-3"><div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-900"><p className="flex items-center gap-1 text-[10px] uppercase tracking-wider text-slate-400"><ArrowDown size={11} /> Received</p><p className="mt-1 text-sm font-semibold">{formatBytes(item.rx_bytes)}</p></div><div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-900"><p className="flex items-center gap-1 text-[10px] uppercase tracking-wider text-slate-400"><ArrowUp size={11} /> Transmitted</p><p className="mt-1 text-sm font-semibold">{formatBytes(item.tx_bytes)}</p></div></div></div>)}</div></div></div>
  </>
}
