import { describe, expect, it } from 'vitest'

import {
  circlesOverlap,
  difficultyAt,
  formatRemainingTime,
  normalizeMovement,
  requiredExperience,
  resolveCircleOverlap,
  teammateEdgeIndicator,
} from './model'

describe('offline game model', () => {
  it('normalizes diagonal movement', () => {
    const movement = normalizeMovement(1, -1)
    expect(Math.hypot(movement.x, movement.y)).toBeCloseTo(1)
    expect(movement.x).toBeCloseTo(Math.SQRT1_2)
  })

  it('uses the configured difficulty curve', () => {
    expect(difficultyAt(0)).toEqual({ spawnRate: 1, maxLiving: 60 })
    expect(difficultyAt(60_000)).toEqual({ spawnRate: 1.8, maxLiving: 110 })
    expect(difficultyAt(150_000)).toEqual({ spawnRate: 2.7, maxLiving: 170 })
    expect(difficultyAt(240_000)).toEqual({ spawnRate: 3.5, maxLiving: 240 })
  })

  it('uses the shared experience threshold formula', () => {
    expect(requiredExperience(1)).toBe(13)
    expect(requiredExperience(4)).toBe(45)
  })

  it('resolves a moving circle out of a rock', () => {
    const resolved = resolveCircleOverlap(
      { x: 10, y: 0, radius: 10 },
      { x: 0, y: 0, radius: 10 },
    )
    expect(resolved).toEqual({ x: 20, y: 0 })
    expect(circlesOverlap({ ...resolved, radius: 10 }, { x: 0, y: 0, radius: 10 })).toBe(false)
  })

  it('formats remaining time by rounding up', () => {
    expect(formatRemainingTime(300_000)).toBe('5:00')
    expect(formatRemainingTime(299_001)).toBe('5:00')
    expect(formatRemainingTime(0)).toBe('0:00')
  })

  it('places off-screen teammates on the viewport edge', () => {
    const indicator = teammateEdgeIndicator(1000, 700, 800, 600, 2100, 1000, 50)
    expect(indicator.visible).toBe(true)
    expect(indicator.x).toBe(750)
    expect(indicator.y).toBe(300)
    expect(indicator.angle).toBeCloseTo(0)
  })

  it('hides teammate indicators while the teammate is on screen', () => {
    expect(teammateEdgeIndicator(1000, 700, 800, 600, 1300, 900).visible).toBe(false)
  })
})
