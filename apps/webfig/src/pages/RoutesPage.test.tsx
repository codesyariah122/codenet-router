import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { api } from '../services/api'
import { RoutesPage } from './RoutesPage'

vi.mock('../services/api', () => ({ api: { routes: vi.fn() } }))

describe('RoutesPage', () => {
  it('menampilkan default route IPv4 dan route IPv6', async () => {
    vi.mocked(api.routes).mockResolvedValue([
      { destination: 'default', gateway: '10.0.2.2', interface: 'enp0s3', metric: 100, protocol: 'dhcp', scope: 'global', routing_table: 'main', ip_version: 4, route_type: 'unicast', status: 'active' },
      { destination: '::/0', gateway: 'fe80::1', interface: 'enp0s3', metric: 100, protocol: 'ra', scope: 'global', routing_table: 'main', ip_version: 6, route_type: 'unicast', status: 'active' },
    ])
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={client}><RoutesPage /></QueryClientProvider>)
    expect(await screen.findByText('default')).toBeInTheDocument()
    expect(screen.getByText('::/0')).toBeInTheDocument()
    expect(api.routes).toHaveBeenCalledOnce()
  })
})
