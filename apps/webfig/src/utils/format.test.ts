import { describe, expect, it } from 'vitest'
import { formatBytes, formatUptime } from './format'
describe('format utilitas',()=>{it('memformat bytes',()=>expect(formatBytes(1024*1024)).toBe('1.0 MB'));it('memformat uptime',()=>expect(formatUptime(90061)).toBe('1h 1j 1m'))})
