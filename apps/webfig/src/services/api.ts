import type { ApiResponse, AuthSession, BandwidthProfile, BandwidthStatistic, DashboardData, IPAddress, NetworkInterface, PPPoESession, PPPoEStatus, ResourceUsage, Route, ServiceStatus, SystemInfo } from '../types/api'

const apiBase = import.meta.env.VITE_API_URL ?? ''

export class ApiError extends Error {
  constructor(message: string, public status: number, public code?: string) { super(message) }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, { ...init, headers: { Accept: 'application/json', ...init.headers }, credentials: 'include' })
  let payload: ApiResponse<T>
  try { payload = await response.json() as ApiResponse<T> } catch { throw new Error('Respons API tidak valid') }
  if (!response.ok || !payload.success) {
    if (response.status === 401 && path !== '/api/v1/auth/login') window.dispatchEvent(new Event('codenet:unauthorized'))
    throw new ApiError(payload.message || 'Permintaan gagal', response.status, payload.error?.code)
  }
  return payload.data
}

const get = <T>(path: string) => request<T>(path)

export const api = {
  health: () => get<{ status: string }>('/health'),
  login: (username: string, password: string) => request<AuthSession>('/api/v1/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, password }) }),
  session: () => get<AuthSession>('/api/v1/auth/session'),
  logout: (csrfToken: string) => request<{ logged_out: boolean }>('/api/v1/auth/logout', { method: 'POST', headers: { 'X-CSRF-Token': csrfToken } }),
  dashboard: () => get<DashboardData>('/api/v1/dashboard'),
  systemInfo: () => get<SystemInfo>('/api/v1/system/info'),
  resources: () => get<ResourceUsage>('/api/v1/system/resources'),
  services: () => get<ServiceStatus[]>('/api/v1/system/services'),
  interfaces: () => get<NetworkInterface[]>('/api/v1/interfaces'),
  ipAddresses: () => get<IPAddress[]>('/api/v1/ip/addresses'),
  routes: () => get<Route[]>('/api/v1/ip/routes'),
  pppoeStatus: () => get<PPPoEStatus>('/api/v1/pppoe/status'),
  pppoeSessions: () => get<PPPoESession[]>('/api/v1/pppoe/sessions'),
  bandwidthProfiles: () => get<BandwidthProfile[]>('/api/v1/bandwidth/profiles'),
  bandwidthStatistics: () => get<BandwidthStatistic[]>('/api/v1/bandwidth/statistics'),
}
