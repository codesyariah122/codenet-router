import { Navigate, Route, Routes } from 'react-router-dom'
import { ProtectedRoute } from './auth/ProtectedRoute'
import { AppLayout } from './layouts/AppLayout'
import { BandwidthPage } from './pages/BandwidthPage'
import { DashboardPage } from './pages/DashboardPage'
import { InterfacesPage } from './pages/InterfacesPage'
import { LoginPage } from './pages/LoginPage'
import { PPPoEPage } from './pages/PPPoEPage'
import { SystemPage } from './pages/SystemPage'
import { IPAddressesPage } from './pages/IPAddressesPage'
import { RoutesPage } from './pages/RoutesPage'

export default function App(){return <Routes><Route path="login" element={<LoginPage/>}/><Route element={<ProtectedRoute/>}><Route element={<AppLayout/>}><Route index element={<DashboardPage/>}/><Route path="interfaces" element={<InterfacesPage/>}/><Route path="ip/addresses" element={<IPAddressesPage/>}/><Route path="ip/routes" element={<RoutesPage/>}/><Route path="pppoe" element={<PPPoEPage/>}/><Route path="bandwidth" element={<BandwidthPage/>}/><Route path="system" element={<SystemPage/>}/><Route path="*" element={<Navigate to="/" replace/>}/></Route></Route></Routes>}
