import type { ApiResponse, BandwidthProfile, BandwidthStatistic, DashboardData, NetworkInterface, PPPoESession, PPPoEStatus, ResourceUsage, ServiceStatus, SystemInfo } from '../types/api'

const apiBase = import.meta.env.VITE_API_URL ?? ''

async function get<T>(path: string): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, { headers: { Accept: 'application/json' }, credentials: 'same-origin' })
  let payload: ApiResponse<T>
  try { payload = await response.json() as ApiResponse<T> } catch { throw new Error('Respons API tidak valid') }
  if (!response.ok || !payload.success) throw new Error(payload.message || 'Permintaan gagal')
  return payload.data
}

export const api = {
  health: () => get<{ status: string }>('/health'),
  dashboard: () => get<DashboardData>('/api/v1/dashboard'),
  systemInfo: () => get<SystemInfo>('/api/v1/system/info'),
  resources: () => get<ResourceUsage>('/api/v1/system/resources'),
  services: () => get<ServiceStatus[]>('/api/v1/system/services'),
  interfaces: () => get<NetworkInterface[]>('/api/v1/interfaces'),
  pppoeStatus: () => get<PPPoEStatus>('/api/v1/pppoe/status'),
  pppoeSessions: () => get<PPPoESession[]>('/api/v1/pppoe/sessions'),
  bandwidthProfiles: () => get<BandwidthProfile[]>('/api/v1/bandwidth/profiles'),
  bandwidthStatistics: () => get<BandwidthStatistic[]>('/api/v1/bandwidth/statistics'),
}
