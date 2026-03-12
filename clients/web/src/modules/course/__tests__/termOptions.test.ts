import { describe, expect, it } from 'vitest'
import { buildTermOptions } from '../termOptions'

describe('buildTermOptions', () => {
  it('keeps current term first and preserves ids and names', () => {
    expect(buildTermOptions([
      { id: '2025-1', name: '2025 春', isCurrent: false },
      { id: '2025-2', name: '2025 秋', isCurrent: true },
    ])).toEqual([
      { id: '2025-2', name: '2025 秋' },
      { id: '2025-1', name: '2025 春' },
    ])
  })
})
