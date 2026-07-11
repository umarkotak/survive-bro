import type { ConnectionState } from '../network/NetworkClient'

export type MatchOutcome = 'playing' | 'won' | 'lost'

export interface GameDiagnostics {
  fps: number
  activeSprites: number
  visibleSprites: number
  projectiles: number
  snapshotIntervalMs: number
  decodeMs: number
  roundTripMs: number
  lastMessageBytes: number
}

export interface GameHudState {
  hp: number
  maxHp: number
  level: number
  experience: number
  experienceRequired: number
  remainingMs: number
  kills: number
  enemies: number
  playerCount: number
  roomName: string
  playerId: string
  displayName: string
  armorPercent: number
  movementSpeed: number
  healthRegeneration: number
  attackBuffPercent: number
  cooldownPercent: number
  spellDamage: number
  projectileSpeed: number
  spellBurst: number
  spellDirections: number
  connection: ConnectionState
  outcome: MatchOutcome
  score: number
  diagnostics: GameDiagnostics
}

type HudListener = (state: GameHudState) => void

const initialState: GameHudState = {
  hp: 100,
  maxHp: 100,
  level: 1,
  experience: 0,
  experienceRequired: 13,
  remainingMs: 6 * 60 * 1000,
  kills: 0,
  enemies: 0,
  playerCount: 1,
  roomName: '',
  playerId: '',
  displayName: '',
  armorPercent: 0,
  movementSpeed: 220,
  healthRegeneration: 0,
  attackBuffPercent: 0,
  cooldownPercent: 0,
  spellDamage: 20,
  projectileSpeed: 700,
  spellBurst: 1,
  spellDirections: 1,
  connection: 'connecting',
  outcome: 'playing',
  score: 0,
  diagnostics: {
    fps: 0,
    activeSprites: 0,
    visibleSprites: 0,
    projectiles: 0,
    snapshotIntervalMs: 0,
    decodeMs: 0,
    roundTripMs: 0,
    lastMessageBytes: 0,
  },
}

export class GameBridge {
  private state = { ...initialState }
  private virtualMovement = { x: 0, y: 0 }
  private readonly listeners = new Set<HudListener>()

  getSnapshot = (): GameHudState => this.state

  subscribe = (listener: HudListener): (() => void) => {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  patch(patch: Partial<GameHudState>): void {
    this.state = { ...this.state, ...patch }
    for (const listener of this.listeners) listener(this.state)
  }

  getVirtualMovement = (): { x: number; y: number } => this.virtualMovement

  setVirtualMovement(x: number, y: number): void {
    this.virtualMovement = { x, y }
  }
}
