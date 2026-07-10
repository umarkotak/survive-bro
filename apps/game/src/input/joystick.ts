import type { Point } from '../game/model'

export interface JoystickState {
  movement: Point
  knob: Point
}

export function joystickState(
  pointerX: number,
  pointerY: number,
  centerX: number,
  centerY: number,
  radius: number,
  deadZone = 0.12,
): JoystickState {
  if (radius <= 0) return { movement: { x: 0, y: 0 }, knob: { x: 0, y: 0 } }

  const deltaX = pointerX - centerX
  const deltaY = pointerY - centerY
  const distance = Math.hypot(deltaX, deltaY)
  if (distance === 0) return { movement: { x: 0, y: 0 }, knob: { x: 0, y: 0 } }

  const directionX = deltaX / distance
  const directionY = deltaY / distance
  const clampedDistance = Math.min(distance, radius)
  const rawMagnitude = clampedDistance / radius
  const movementMagnitude = rawMagnitude <= deadZone
    ? 0
    : Math.min(1, (rawMagnitude - deadZone) / (1 - deadZone))

  return {
    movement: {
      x: directionX * movementMagnitude,
      y: directionY * movementMagnitude,
    },
    knob: {
      x: directionX * clampedDistance,
      y: directionY * clampedDistance,
    },
  }
}
