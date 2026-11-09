import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { PPPoEUsersPage } from './PPPoEUsersPage'

vi.mock('../auth/AuthContext', () => ({ useAuth: () => ({ session: { csrf_token: 'csrf', admin: { username: 'admin' } } }) }))
vi.mock('../services/api', () => ({ api: { pppoeUsers: vi.fn(), createPPPoEUser: vi.fn(), setPPPoEUserEnabled: vi.fn(), deletePPPoEUser: vi.fn() } }))
import { api } from '../services/api'

describe('PPPoEUsersPage', () => {
  it('menampilkan user tanpa password dan status sesi', async () => {
    vi.mocked(api.pppoeUsers).mockResolvedValue([{ username: 'alice', server: '*', ip_pool: 'pool-main', download_bps: 10000000, upload_bps: 5000000, enabled: true, session_active: true, session_status: 'active' }])
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={client}><PPPoEUsersPage /></QueryClientProvider>)
    expect(await screen.findByText('alice')).toBeInTheDocument()
    expect(screen.getByText('active')).toBeInTheDocument()
    expect(screen.queryByText('password123')).not.toBeInTheDocument()
  })

  it('menggunakan pattern browser-valid dan menerima IP pool wildcard', async () => {
    vi.mocked(api.pppoeUsers).mockResolvedValue([])
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(<QueryClientProvider client={client}><PPPoEUsersPage /></QueryClientProvider>)
    fireEvent.click(await screen.findByRole('button', { name: 'Tambah user' }))

    const username = screen.getByPlaceholderText('Username') as HTMLInputElement
    const ipPool = screen.getByPlaceholderText('IP pool') as HTMLInputElement
    expect(username.getAttribute('pattern')).toBe('(?:[A-Za-z0-9._]|-){1,64}')
    expect(ipPool.getAttribute('pattern')).toContain('[*]')
    fireEvent.change(ipPool, { target: { value: '*' } })
    expect(ipPool.checkValidity()).toBe(true)
  })
})
