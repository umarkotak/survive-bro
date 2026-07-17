export const WORLD_WIDTH = 3200
export const WORLD_HEIGHT = 1800
export const MATCH_DURATION_MS = 15 * 60 * 1000

export interface DifficultyStep {
  spawnRate: number
  maxLiving: number
}

export interface Point {
  x: number
  y: number
}

export interface Circle extends Point {
  radius: number
}

export interface EdgeIndicator {
  visible: boolean
  x: number
  y: number
  angle: number
  distance: number
}

export function difficultyAt(elapsedMs: number): DifficultyStep {
  const seconds = elapsedMs / 1000
  if (seconds < 60) return { spawnRate: 1, maxLiving: 60 }
  if (seconds < 150) return { spawnRate: 1.8, maxLiving: 110 }
  if (seconds < 240) return { spawnRate: 2.7, maxLiving: 170 }
  return { spawnRate: 3.5, maxLiving: 240 }
}

export function requiredExperience(level: number): number {
  return Math.round(8 + 5 * Math.pow(level, 1.45))
}

export function normalizeMovement(x: number, y: number): Point {
  const length = Math.hypot(x, y)
  if (length === 0) return { x: 0, y: 0 }
  if (length <= 1) return { x, y }
  return { x: x / length, y: y / length }
}

export function circlesOverlap(a: Circle, b: Circle): boolean {
  const distanceX = a.x - b.x
  const distanceY = a.y - b.y
  const radii = a.radius + b.radius
  return distanceX * distanceX + distanceY * distanceY < radii * radii
}

export function resolveCircleOverlap(moving: Circle, fixed: Circle): Point {
  const distanceX = moving.x - fixed.x
  const distanceY = moving.y - fixed.y
  const minimumDistance = moving.radius + fixed.radius
  const distance = Math.hypot(distanceX, distanceY)

  if (distance >= minimumDistance) return { x: moving.x, y: moving.y }
  if (distance === 0) return { x: fixed.x + minimumDistance, y: fixed.y }

  const scale = minimumDistance / distance
  return {
    x: fixed.x + distanceX * scale,
    y: fixed.y + distanceY * scale,
  }
}

export function formatRemainingTime(remainingMs: number): string {
  const totalSeconds = Math.max(0, Math.ceil(remainingMs / 1000))
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}

export function teammateEdgeIndicator(
  cameraX: number,
  cameraY: number,
  viewportWidth: number,
  viewportHeight: number,
  targetX: number,
  targetY: number,
  inset = 48,
): EdgeIndicator {
  const screenX = targetX - cameraX
  const screenY = targetY - cameraY
  const centerX = viewportWidth / 2
  const centerY = viewportHeight / 2
  const directionX = screenX - centerX
  const directionY = screenY - centerY
  const distance = Math.hypot(directionX, directionY)
  const visible = screenX >= inset && screenX <= viewportWidth - inset && screenY >= inset && screenY <= viewportHeight - inset
  if (visible || distance === 0) return { visible: false, x: screenX, y: screenY, angle: 0, distance }

  const scaleX = directionX === 0 ? Number.POSITIVE_INFINITY : (centerX - inset) / Math.abs(directionX)
  const scaleY = directionY === 0 ? Number.POSITIVE_INFINITY : (centerY - inset) / Math.abs(directionY)
  const scale = Math.min(scaleX, scaleY)
  return {
    visible: true,
    x: centerX + directionX * scale,
    y: centerY + directionY * scale,
    angle: Math.atan2(directionY, directionX),
    distance,
  }
}
