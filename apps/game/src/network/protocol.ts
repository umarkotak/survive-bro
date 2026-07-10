export const PROTOCOL_VERSION = 2

export type MessageType =
  | 'join_room'
  | 'leave_room'
  | 'ping'
  | 'input'
  | 'joined'
  | 'room_state'
  | 'match_started'
  | 'snapshot'
  | 'projectile_spawned'
  | 'projectile_removed'
  | 'match_ended'
  | 'pong'
  | 'error'
  | 'server_shutdown'

const typeToId: Record<MessageType, number> = {
  join_room: 1,
  leave_room: 2,
  ping: 3,
  input: 4,
  joined: 64,
  room_state: 65,
  match_started: 66,
  snapshot: 67,
  projectile_spawned: 68,
  projectile_removed: 69,
  match_ended: 70,
  pong: 71,
  error: 126,
  server_shutdown: 127,
}

const idToType = new Map(Object.entries(typeToId).map(([name, id]) => [id, name as MessageType]))

export interface Envelope<T = unknown> {
  v: number
  type: MessageType
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

export function createEnvelope<T>(type: MessageType, payload: T, requestId?: string): Envelope<T> {
  return { v: PROTOCOL_VERSION, type, requestId, payload }
}

export function encodeEnvelope(envelope: Envelope): Uint8Array<ArrayBuffer> {
  if (envelope.v !== PROTOCOL_VERSION) throw new Error(`Protocol version must be ${PROTOCOL_VERSION}`)
  const typeId = typeToId[envelope.type]
  if (typeId === undefined) throw new Error(`Unknown message type: ${envelope.type}`)

  const writer = new BinaryWriter()
  const requestId = envelope.requestId ?? ''
  const requestBytes = textEncoder.encode(requestId)
  if (requestBytes.length > 255) throw new Error('Request ID exceeds 255 bytes')
  writer.u8(PROTOCOL_VERSION)
  writer.u8(typeId)
  writer.u8(requestBytes.length)
  writer.bytes(requestBytes)
  encodePayload(writer, envelope)
  return writer.finish()
}

export function decodeEnvelope(data: ArrayBuffer | ArrayBufferView): Envelope {
  const reader = new BinaryReader(data)
  const version = reader.u8()
  if (version !== PROTOCOL_VERSION) throw new Error(`Unsupported protocol version: ${version}`)
  const typeId = reader.u8()
  const type = idToType.get(typeId)
  if (!type) throw new Error(`Unknown message type ID: ${typeId}`)
  const requestId = reader.fixedString(reader.u8())
  const payload = decodePayload(reader, type)
  reader.finish()
  return { v: version, type, requestId: requestId || undefined, payload }
}

function encodePayload(writer: BinaryWriter, envelope: Envelope): void {
  switch (envelope.type) {
    case 'join_room': {
      const payload = envelope.payload as { displayName: string; reconnectToken: string | null }
      writer.string(payload.displayName)
      writer.bool(payload.reconnectToken !== null)
      if (payload.reconnectToken !== null) writer.string(payload.reconnectToken)
      break
    }
    case 'leave_room':
    case 'ping':
    case 'pong':
      break
    case 'input': {
      const payload = envelope.payload as { sequence: number; moveX: number; moveY: number }
      writer.u32(payload.sequence)
      writer.f32(payload.moveX)
      writer.f32(payload.moveY)
      break
    }
    case 'joined': {
      const payload = envelope.payload as JoinedPayload
      writer.string(payload.playerId)
      writer.string(payload.reconnectToken)
      writer.string(payload.roomName)
      writer.bool(payload.host)
      break
    }
    case 'room_state': {
      const payload = envelope.payload as RoomStatePayload
      writer.u8(encodeRoomStatus(payload.status))
      writer.string(payload.hostPlayerId ?? '')
      writer.u8(payload.players.length)
      for (const player of payload.players) {
        writer.string(player.id)
        writer.string(player.displayName)
        writer.string(player.characterId)
        writer.u8(Number(player.ready) | (Number(player.connected) << 1))
      }
      break
    }
    case 'match_started': {
      const payload = envelope.payload as MatchStartedPayload
      writer.string(payload.roomName)
      writer.string(payload.mapId)
      writer.f32(payload.mapWidth)
      writer.f32(payload.mapHeight)
      writer.i64(payload.startedAtMs)
      writer.u16(payload.obstacles.length)
      for (const obstacle of payload.obstacles) {
        writer.string(obstacle.id)
        writer.string(obstacle.type)
        writer.f32(obstacle.x)
        writer.f32(obstacle.y)
        writer.f32(obstacle.radius)
      }
      break
    }
    case 'snapshot':
      encodeSnapshot(writer, envelope.payload as SnapshotPayload)
      break
    case 'projectile_spawned': {
      const payload = envelope.payload as ProjectileSpawnedPayload
      writer.u32(payload.projectileId)
      writer.string(payload.ownerId)
      writer.string(payload.weaponId)
      writer.f32(payload.x)
      writer.f32(payload.y)
      writer.f32(payload.velocityX)
      writer.f32(payload.velocityY)
      writer.u32(payload.spawnTick)
      break
    }
    case 'projectile_removed': {
      const payload = envelope.payload as ProjectileRemovedPayload
      writer.u32(payload.projectileId)
      writer.u8(encodeRemovalReason(payload.reason))
      break
    }
    case 'match_ended': {
      const payload = envelope.payload as MatchEndedPayload
      writer.u8(payload.outcome === 'won' ? 1 : 0)
      writer.u32(payload.survivalMs)
      writer.u16(payload.teamLevel)
      writer.u32(payload.totalKills)
      break
    }
    case 'error': {
      const payload = envelope.payload as ErrorPayload
      writer.string(payload.code)
      writer.string(payload.message)
      break
    }
    case 'server_shutdown':
      writer.string((envelope.payload as { reason: string }).reason)
      break
  }
}

function encodeSnapshot(writer: BinaryWriter, payload: SnapshotPayload): void {
  writer.u32(payload.tick)
  writer.i64(payload.serverTimeMs)
  writer.u8(payload.players.length)
  for (const player of payload.players) {
    writer.string(player.id)
    writer.string(player.displayName)
    writer.f32(player.x)
    writer.f32(player.y)
    writer.f32(player.velocityX)
    writer.f32(player.velocityY)
    writer.u8(Number(player.facing === 'left') | (Number(player.alive) << 1))
    writer.u16(player.hp)
    writer.u16(player.maxHp)
    writer.u32(player.lastProcessedInput)
    writer.u32(player.kills)
  }
  writer.u16(payload.monsters.length)
  for (const monster of payload.monsters) {
    writer.u32(monster.id)
    writer.f32(monster.x)
    writer.f32(monster.y)
    writer.u16(monster.hp)
    writer.u16(monster.maxHp)
  }
  writer.u16(payload.pickups.length)
  for (const pickup of payload.pickups) {
    writer.u32(pickup.id)
    writer.f32(pickup.x)
    writer.f32(pickup.y)
  }
  writer.u16(payload.team.level)
  writer.u16(payload.team.experience)
  writer.u16(payload.team.experienceRequired)
  writer.u32(payload.team.totalKills)
  writer.u32(payload.remainingMs)
}

function decodePayload(reader: BinaryReader, type: MessageType): unknown {
  switch (type) {
    case 'join_room': {
      const displayName = reader.string()
      const reconnectToken = reader.bool() ? reader.string() : null
      return { displayName, reconnectToken }
    }
    case 'leave_room':
    case 'ping':
    case 'pong':
      return {}
    case 'input':
      return { sequence: reader.u32(), moveX: reader.f32(), moveY: reader.f32() }
    case 'joined':
      return { playerId: reader.string(), reconnectToken: reader.string(), roomName: reader.string(), host: reader.bool() } satisfies JoinedPayload
    case 'room_state':
      return decodeRoomState(reader)
    case 'match_started':
      return decodeMatchStarted(reader)
    case 'snapshot':
      return decodeSnapshot(reader)
    case 'projectile_spawned':
      return {
        projectileId: reader.u32(), ownerId: reader.string(), weaponId: reader.string(),
        x: reader.f32(), y: reader.f32(), velocityX: reader.f32(), velocityY: reader.f32(), spawnTick: reader.u32(),
      } satisfies ProjectileSpawnedPayload
    case 'projectile_removed':
      return { projectileId: reader.u32(), reason: decodeRemovalReason(reader.u8()) } satisfies ProjectileRemovedPayload
    case 'match_ended':
      return {
        outcome: decodeOutcome(reader.u8()), survivalMs: reader.u32(), teamLevel: reader.u16(), totalKills: reader.u32(),
      } satisfies MatchEndedPayload
    case 'error':
      return { code: reader.string(), message: reader.string() } satisfies ErrorPayload
    case 'server_shutdown':
      return { reason: reader.string() }
  }
}

function decodeRoomState(reader: BinaryReader): RoomStatePayload {
  const status = decodeRoomStatus(reader.u8())
  const hostPlayerId = reader.string()
  const playerCount = reader.u8()
  if (playerCount > 4) throw new Error(`Player count exceeds limit: ${playerCount}`)
  const players: RoomStatePayload['players'] = []
  for (let index = 0; index < playerCount; index += 1) {
    const id = reader.string()
    const displayName = reader.string()
    const characterId = reader.string()
    const flags = reader.u8()
    if ((flags & ~3) !== 0) throw new Error(`Invalid player flags: ${flags}`)
    players.push({ id, displayName, characterId, ready: (flags & 1) !== 0, connected: (flags & 2) !== 0 })
  }
  return { status, hostPlayerId: hostPlayerId || undefined, players }
}

function decodeMatchStarted(reader: BinaryReader): MatchStartedPayload {
  const roomName = reader.string()
  const mapId = reader.string()
  const mapWidth = reader.f32()
  const mapHeight = reader.f32()
  const startedAtMs = reader.i64()
  const count = reader.u16()
  if (count > 256) throw new Error(`Obstacle count exceeds limit: ${count}`)
  const obstacles: ObstaclePayload[] = []
  for (let index = 0; index < count; index += 1) {
    const id = reader.string()
    const type = reader.string()
    if (type !== 'large_rock') throw new Error(`Unknown obstacle type: ${type}`)
    obstacles.push({ id, type, x: reader.f32(), y: reader.f32(), radius: reader.f32() })
  }
  return { roomName, mapId, mapWidth, mapHeight, startedAtMs, obstacles }
}

function decodeSnapshot(reader: BinaryReader): SnapshotPayload {
  const tick = reader.u32()
  const serverTimeMs = reader.i64()
  const playerCount = reader.u8()
  if (playerCount > 4) throw new Error(`Player count exceeds limit: ${playerCount}`)
  const players: SnapshotPlayer[] = []
  for (let index = 0; index < playerCount; index += 1) {
    const id = reader.string()
    const displayName = reader.string()
    const x = reader.f32()
    const y = reader.f32()
    const velocityX = reader.f32()
    const velocityY = reader.f32()
    const flags = reader.u8()
    if ((flags & ~3) !== 0) throw new Error(`Invalid snapshot player flags: ${flags}`)
    players.push({
      id, displayName, x, y, velocityX, velocityY, facing: (flags & 1) !== 0 ? 'left' : 'right',
      alive: (flags & 2) !== 0, hp: reader.u16(), maxHp: reader.u16(),
      lastProcessedInput: reader.u32(), kills: reader.u32(),
    })
  }
  const monsterCount = reader.u16()
  if (monsterCount > 1024) throw new Error(`Monster count exceeds limit: ${monsterCount}`)
  const monsters: SnapshotMonster[] = []
  for (let index = 0; index < monsterCount; index += 1) {
    monsters.push({ id: reader.u32(), x: reader.f32(), y: reader.f32(), hp: reader.u16(), maxHp: reader.u16() })
  }
  const pickupCount = reader.u16()
  if (pickupCount > 2048) throw new Error(`Pickup count exceeds limit: ${pickupCount}`)
  const pickups: SnapshotPickup[] = []
  for (let index = 0; index < pickupCount; index += 1) {
    pickups.push({ id: reader.u32(), x: reader.f32(), y: reader.f32() })
  }
  return {
    tick,
    serverTimeMs,
    players,
    monsters,
    pickups,
    team: {
      level: reader.u16(), experience: reader.u16(), experienceRequired: reader.u16(), totalKills: reader.u32(),
    },
    remainingMs: reader.u32(),
  }
}

class BinaryWriter {
  private buffer = new ArrayBuffer(256)
  private view = new DataView(this.buffer)
  private offset = 0

  finish(): Uint8Array<ArrayBuffer> {
    return new Uint8Array(this.buffer.slice(0, this.offset))
  }

  bytes(value: Uint8Array): void {
    this.ensure(value.length)
    new Uint8Array(this.buffer, this.offset, value.length).set(value)
    this.offset += value.length
  }

  u8(value: number): void {
    assertInteger(value, 0xff, 'u8')
    this.ensure(1)
    this.view.setUint8(this.offset, value)
    this.offset += 1
  }

  u16(value: number): void {
    assertInteger(value, 0xffff, 'u16')
    this.ensure(2)
    this.view.setUint16(this.offset, value, true)
    this.offset += 2
  }

  u32(value: number): void {
    assertInteger(value, 0xffffffff, 'u32')
    this.ensure(4)
    this.view.setUint32(this.offset, value, true)
    this.offset += 4
  }

  i64(value: number): void {
    if (!Number.isSafeInteger(value)) throw new Error(`i64 value is not a safe integer: ${value}`)
    this.ensure(8)
    this.view.setBigInt64(this.offset, BigInt(value), true)
    this.offset += 8
  }

  f32(value: number): void {
    if (!Number.isFinite(value) || Math.abs(value) > 3.4028234663852886e38) throw new Error(`Invalid f32 value: ${value}`)
    this.ensure(4)
    this.view.setFloat32(this.offset, value, true)
    this.offset += 4
  }

  bool(value: boolean): void {
    this.u8(value ? 1 : 0)
  }

  string(value: string): void {
    const encoded = textEncoder.encode(value)
    this.u16(encoded.length)
    this.bytes(encoded)
  }

  private ensure(length: number): void {
    if (this.offset + length <= this.buffer.byteLength) return
    let capacity = this.buffer.byteLength
    while (capacity < this.offset + length) capacity *= 2
    const expanded = new ArrayBuffer(capacity)
    new Uint8Array(expanded).set(new Uint8Array(this.buffer, 0, this.offset))
    this.buffer = expanded
    this.view = new DataView(expanded)
  }
}

class BinaryReader {
  private readonly bytes: Uint8Array
  private readonly view: DataView
  private offset = 0

  constructor(data: ArrayBuffer | ArrayBufferView) {
    this.bytes = data instanceof ArrayBuffer
      ? new Uint8Array(data)
      : new Uint8Array(data.buffer, data.byteOffset, data.byteLength)
    this.view = new DataView(this.bytes.buffer, this.bytes.byteOffset, this.bytes.byteLength)
  }

  finish(): void {
    if (this.offset !== this.bytes.byteLength) throw new Error(`Binary frame has ${this.bytes.byteLength - this.offset} trailing bytes`)
  }

  u8(): number {
    this.require(1)
    const value = this.view.getUint8(this.offset)
    this.offset += 1
    return value
  }

  u16(): number {
    this.require(2)
    const value = this.view.getUint16(this.offset, true)
    this.offset += 2
    return value
  }

  u32(): number {
    this.require(4)
    const value = this.view.getUint32(this.offset, true)
    this.offset += 4
    return value
  }

  i64(): number {
    this.require(8)
    const value = Number(this.view.getBigInt64(this.offset, true))
    this.offset += 8
    if (!Number.isSafeInteger(value)) throw new Error(`i64 exceeds JavaScript safe integer: ${value}`)
    return value
  }

  f32(): number {
    this.require(4)
    const value = this.view.getFloat32(this.offset, true)
    this.offset += 4
    if (!Number.isFinite(value)) throw new Error('Binary frame contains a non-finite float')
    return value
  }

  bool(): boolean {
    const value = this.u8()
    if (value > 1) throw new Error(`Invalid boolean value: ${value}`)
    return value === 1
  }

  string(): string {
    return this.fixedString(this.u16())
  }

  fixedString(length: number): string {
    this.require(length)
    const value = textDecoder.decode(this.bytes.subarray(this.offset, this.offset + length))
    this.offset += length
    return value
  }

  private require(length: number): void {
    if (length < 0 || this.offset + length > this.bytes.byteLength) {
      throw new Error(`Binary frame is truncated at byte ${this.offset}`)
    }
  }
}

function assertInteger(value: number, maximum: number, type: string): void {
  if (!Number.isInteger(value) || value < 0 || value > maximum) throw new Error(`${type} value is out of range: ${value}`)
}

function encodeRoomStatus(status: RoomStatePayload['status']): number {
  return status === 'lobby' ? 0 : status === 'running' ? 1 : 2
}

function decodeRoomStatus(value: number): RoomStatePayload['status'] {
  if (value === 0) return 'lobby'
  if (value === 1) return 'running'
  if (value === 2) return 'finished'
  throw new Error(`Unknown room status: ${value}`)
}

function encodeRemovalReason(reason: ProjectileRemovedPayload['reason']): number {
  return reason === 'enemy_hit' ? 0 : reason === 'obstacle_hit' ? 1 : reason === 'range_expired' ? 2 : 3
}

function decodeRemovalReason(value: number): ProjectileRemovedPayload['reason'] {
  if (value === 0) return 'enemy_hit'
  if (value === 1) return 'obstacle_hit'
  if (value === 2) return 'range_expired'
  if (value === 3) return 'match_ended'
  throw new Error(`Unknown projectile removal reason: ${value}`)
}

function decodeOutcome(value: number): MatchEndedPayload['outcome'] {
  if (value === 0) return 'lost'
  if (value === 1) return 'won'
  throw new Error(`Unknown match outcome: ${value}`)
}

const textEncoder = new TextEncoder()
const textDecoder = new TextDecoder('utf-8', { fatal: true })
