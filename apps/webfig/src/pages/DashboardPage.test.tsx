import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../services/api'
import { DashboardPage } from './DashboardPage'

vi.mock('../services/api', () => ({ api: { dashboard: vi.fn() } }))

describe('DashboardPage', () => {
  beforeEach(() => {
    vi.mocked(api.dashboard).mockResolvedValue({
      system: { hostname: 'codenet-test', os: 'CODENET Router OS', version: '0.1.0', kernel: '6.12-test', uptime_seconds: 90061 },
      resources: { cpu_percent: 12.5, memory_percent: 25, memory_used_bytes: 1024, memory_total_bytes: 4096 },
      interfaces: [
        { name: 'enp0s3', role: 'WAN', type: 'ethernet', mac_address: '00:00:00:00:00:01', ipv4_addresses: ['10.0.2.15/24'], status: 'up', mtu: 1500, rx_bytes: 1024, tx_bytes: 2048, rx_packets: 10, tx_packets: 20 },
        { name: 'enp0s8', role: 'LAN', type: 'ethernet', mac_address: '00:00:00:00:00:02', ipv4_addresses: ['10.50.0.1/24'], status: 'up', mtu: 1500, rx_bytes: 4096, tx_bytes: 8192, rx_packets: 30, tx_packets: 40 },
      ],
      pppoe: { service_name: 'CODENET-PPPOE', access_concentrator: 'CODENET', status: 'active', active_sessions: 2, session_count_ready: true, control_status: 'available', ip_pool: '10.60.0.10-10.60.0.50' },
      traffic_history: [{ timestamp: '2026-01-01T00:00:00Z', rx_bps: 1_000_000, tx_bps: 500_000 }],
    })
  })

  it('menampilkan data dari Management API', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={client}><DashboardPage /></QueryClientProvider>)
    expect(await screen.findByText('Selamat datang di codenet-test')).toBeInTheDocument()
    expect(screen.getByText('2 sesi')).toBeInTheDocument()
    expect(api.dashboard).toHaveBeenCalledOnce()
  })
})
