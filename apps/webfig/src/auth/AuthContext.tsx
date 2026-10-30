import { useQueryClient } from '@tanstack/react-query'
import { createContext, type ReactNode, useContext, useEffect, useMemo, useState } from 'react'
import { ApiError, api } from '../services/api'
import type { AuthSession } from '../types/api'

interface AuthState {
  session: AuthSession | null
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<AuthSession | null>(null)
  const [loading, setLoading] = useState(true)
  const queryClient = useQueryClient()

  useEffect(() => {
    let active = true
    api.session().then((value) => { if (active) setSession(value) }).catch((error: unknown) => {
      if (!(error instanceof ApiError && error.status === 401)) console.error('Pemeriksaan session gagal')
    }).finally(() => { if (active) setLoading(false) })
    const unauthorized = () => { setSession(null); queryClient.clear() }
    window.addEventListener('codenet:unauthorized', unauthorized)
    return () => { active = false; window.removeEventListener('codenet:unauthorized', unauthorized) }
  }, [queryClient])

  const value = useMemo<AuthState>(() => ({
    session,
    loading,
    login: async (username, password) => { const value = await api.login(username, password); setSession(value); queryClient.clear() },
    logout: async () => { if (session) await api.logout(session.csrf_token); setSession(null); queryClient.clear() },
  }), [loading, queryClient, session])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) throw new Error('useAuth harus digunakan di dalam AuthProvider')
  return value
}
