import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { api } from '../services/api'
import { PPPoEPage } from './PPPoEPage'

vi.mock('../services/api', () => ({ api: { pppoeStatus: vi.fn(), pppoeSessions: vi.fn() } }))

describe('PPPoEPage', () => {
  it('menampilkan sesi aktif dan rate pelanggan', async () => {
    vi.mocked(api.pppoeStatus).mockResolvedValue({ service_name: 'CODENET-PPPOE', access_concentrator: 'CODENET', status: 'active', active_sessions: 1, session_count_ready: true, control_status: 'available', ip_pool: '10.60.0.10-10.60.0.50' })
    vi.mocked(api.pppoeSessions).mockResolvedValue([{ username: 'alice', ip_address: '10.60.0.10', interface: 'ppp0', connected_at: '2026-01-01T00:00:00Z', uptime_seconds: 60, rx_bytes: 1000, tx_bytes: 2000, upload_bps: 100, download_bps: 200, status: 'online' }])
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={client}><PPPoEPage /></QueryClientProvider>)
    expect(await screen.findByText('alice')).toBeInTheDocument()
    expect(screen.getAllByText('Active sessions').length).toBeGreaterThan(0)
    expect(api.pppoeSessions).toHaveBeenCalledOnce()
  })
})
