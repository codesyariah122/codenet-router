export const formatBytes = (value: number) => {
  if (value === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']; const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  return `${(value / 1024 ** index).toFixed(index > 1 ? 1 : 0)} ${units[index]}`
}
export const formatRate = (value: number) => `${(value / 1_000_000).toFixed(value >= 1_000_000 ? 1 : 2)} Mbps`
export const formatUptime = (seconds: number) => { const days = Math.floor(seconds / 86400); const hours = Math.floor((seconds % 86400) / 3600); const minutes = Math.floor((seconds % 3600) / 60); return `${days}h ${hours}j ${minutes}m` }
