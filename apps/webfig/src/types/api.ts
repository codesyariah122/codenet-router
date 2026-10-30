export interface ApiResponse<T> { success: boolean; message: string; data: T; error?: { code: string } }
export interface SystemInfo { hostname: string; os: string; version: string; kernel: string; uptime_seconds: number }
export interface ResourceUsage { cpu_percent: number; memory_percent: number; memory_used_bytes: number; memory_total_bytes: number }
export interface ServiceStatus { name: string; display_name: string; status: string; description?: string }
export interface NetworkInterface { name: string; role: string; type: string; mac_address: string; ipv4_addresses: string[]; status: string; mtu: number; rx_bytes: number; tx_bytes: number; rx_packets: number; tx_packets: number }
export interface IPAddress { address: string; network: string; interface: string; ip_version: 4 | 6; scope: string; flags: string[] }
export interface Route { destination: string; gateway?: string; interface?: string; metric: number; protocol: string; scope: string; routing_table: string; ip_version: 4 | 6; route_type: string; status: string }
export interface PPPoEStatus { service_name: string; access_concentrator: string; server_interface?: string; status: string; active_sessions: number; session_count_ready: boolean; ip_pool: string; pool_source?: string; control_status: string; monitoring_error?: string }
export interface PPPoESession { username: string; ip_address: string; interface: string; connected_at: string; uptime_seconds: number; rx_bytes: number; tx_bytes: number; download_bps: number; upload_bps: number; status: string }
export interface BandwidthProfile { name: string; download_bps: number; upload_bps: number; description?: string }
export interface BandwidthStatistic { username: string; ip_address: string; interface: string; download_limit_bps: number; upload_limit_bps: number; download_usage_bps: number; upload_usage_bps: number; limiter_status: string }
export interface TrafficPoint { timestamp: string; rx_bps: number; tx_bps: number }
export interface DashboardData { system: SystemInfo; resources: ResourceUsage; interfaces: NetworkInterface[]; pppoe: PPPoEStatus; traffic_history: TrafficPoint[] }
export interface AuthSession { admin: { username: string }; csrf_token: string; expires_at: string }
