import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { api } from '../services/api'
import { BandwidthPage } from './BandwidthPage'

vi.mock('../services/api', () => ({ api: { bandwidthProfiles: vi.fn(), bandwidthStatistics: vi.fn() } }))

describe('BandwidthPage', () => {
  it('menampilkan limiter yang terdeteksi tanpa mengarang limit', async () => {
    vi.mocked(api.bandwidthProfiles).mockResolvedValue([])
    vi.mocked(api.bandwidthStatistics).mockResolvedValue([{ username: '', ip_address: '', interface: 'ppp0', download_limit_bps: 0, upload_limit_bps: 0, download_usage_bps: 0, upload_usage_bps: 0, limiter_status: 'active' }])
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={client}><BandwidthPage /></QueryClientProvider>)
    expect((await screen.findAllByText('Tidak tersedia')).length).toBeGreaterThan(0)
    expect(screen.getByText('active')).toBeInTheDocument()
  })
})
