import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { api } from '../services/api'
import { IPAddressesPage } from './IPAddressesPage'

vi.mock('../services/api', () => ({ api: { ipAddresses: vi.fn() } }))

describe('IPAddressesPage', () => {
  it('menampilkan dan memfilter address IPv4/IPv6', async () => {
    vi.mocked(api.ipAddresses).mockResolvedValue([
      { address: '10.0.2.15/24', network: '10.0.2.0/24', interface: 'enp0s3', ip_version: 4, scope: 'site', flags: ['up'] },
      { address: 'fe80::1/64', network: 'fe80::/64', interface: 'enp0s3', ip_version: 6, scope: 'link', flags: ['up'] },
    ])
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={client}><IPAddressesPage /></QueryClientProvider>)
    expect(await screen.findByText('10.0.2.15/24')).toBeInTheDocument()
    expect(screen.getByText('fe80::1/64')).toBeInTheDocument()
    expect(api.ipAddresses).toHaveBeenCalledOnce()
  })
})
