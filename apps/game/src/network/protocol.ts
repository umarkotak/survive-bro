export const PROTOCOL_VERSION = 1

export interface Envelope<T = unknown> {
  v: number
  type: string
  requestId?: string
  payload: T
}

export interface JoinedPayload {
  playerId: string
  reconnectToken: string
  roomName: string
  host: boolean
}

export interface RoomStatePayload {
  status: 'lobby' | 'running' | 'finished'
  hostPlayerId?: string
  players: Array<{
    id: string
    displayName: string
    characterId: string
    ready: boolean
    connected: boolean
  }>
}

export interface ObstaclePayload {
  id: string
  type: 'large_rock'
  x: number
  y: number
  radius: number
}

export interface MatchStartedPayload {
  roomName: string
  mapId: string
  mapWidth: number
  mapHeight: number
  startedAtMs: number
  obstacles: ObstaclePayload[]
}

export interface SnapshotPlayer {
  id: string
  displayName: string
  x: number
  y: number
  velocityX: number
  velocityY: number
  facing: 'left' | 'right'
  hp: number
  maxHp: number
  alive: boolean
  lastProcessedInput: number
  kills: number
}

export interface SnapshotMonster {
  id: number
  x: number
  y: number
  hp: number
  maxHp: number
}

export interface SnapshotPickup {
  id: number
  x: number
  y: number
}

export interface SnapshotPayload {
  tick: number
  serverTimeMs: number
  players: SnapshotPlayer[]
  monsters: SnapshotMonster[]
  pickups: SnapshotPickup[]
  team: {
    level: number
    experience: number
    experienceRequired: number
    totalKills: number
  }
  remainingMs: number
}

export interface ProjectileSpawnedPayload {
  projectileId: number
  ownerId: string
  weaponId: string
  x: number
  y: number
  velocityX: number
  velocityY: number
  spawnTick: number
}

export interface ProjectileRemovedPayload {
  projectileId: number
  reason: 'enemy_hit' | 'obstacle_hit' | 'range_expired' | 'match_ended'
}

export interface MatchEndedPayload {
  outcome: 'won' | 'lost'
  survivalMs: number
  teamLevel: number
  totalKills: number
}

export interface ErrorPayload {
  code: string
  message: string
}

export function createEnvelope<T>(type: string, payload: T, requestId?: string): Envelope<T> {
  return { v: PROTOCOL_VERSION, type, requestId, payload }
}

export function parseEnvelope(raw: string): Envelope {
  const parsed: unknown = JSON.parse(raw)
  if (!parsed || typeof parsed !== 'object') throw new Error('Invalid server message')
  const candidate = parsed as Partial<Envelope>
  if (candidate.v !== PROTOCOL_VERSION || typeof candidate.type !== 'string' || !('payload' in candidate)) {
    throw new Error('Unsupported server message')
  }
  return candidate as Envelope
}
