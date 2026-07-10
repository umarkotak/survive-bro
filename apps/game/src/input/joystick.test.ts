import { describe, expect, it } from 'vitest'

import { joystickState } from './joystick'

describe('virtual joystick', () => {
  it('returns zero movement inside the dead zone', () => {
    expect(joystickState(104, 100, 100, 100, 50).movement).toEqual({ x: 0, y: 0 })
  })

  it('returns analog movement within the joystick radius', () => {
    const state = joystickState(125, 100, 100, 100, 50)
    expect(state.movement.x).toBeCloseTo(0.4318, 3)
    expect(state.movement.y).toBeCloseTo(0)
    expect(state.knob).toEqual({ x: 25, y: 0 })
  })

  it('clamps diagonal movement and knob travel at the edge', () => {
    const state = joystickState(200, 200, 100, 100, 50)
    expect(Math.hypot(state.movement.x, state.movement.y)).toBeCloseTo(1)
    expect(Math.hypot(state.knob.x, state.knob.y)).toBeCloseTo(50)
  })
})
