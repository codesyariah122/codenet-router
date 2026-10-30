import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { LoadingState } from '../components/ui'
import { useAuth } from './AuthContext'

export function ProtectedRoute() {
  const { session, loading } = useAuth()
  const location = useLocation()
  if (loading) return <div className="min-h-screen bg-slate-50 dark:bg-[#07101d]"><LoadingState /></div>
  if (!session) return <Navigate to="/login" state={{ from: location }} replace />
  return <Outlet />
}
