import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { TrafficChart } from './TrafficChart'

describe('TrafficChart', () => {
  it('menampilkan label RX dan TX', () => {
    render(<TrafficChart data={[{ timestamp: '2026-01-01T00:00:00Z', rx_bps: 2_000_000, tx_bps: 1_000_000 }]} />)
    expect(screen.getByText('RX')).toBeInTheDocument()
    expect(screen.getByText('TX')).toBeInTheDocument()
    expect(screen.getByRole('img')).toHaveAttribute('aria-label', 'Grafik traffic RX dan TX')
  })
})
