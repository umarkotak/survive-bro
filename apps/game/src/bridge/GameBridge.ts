import type { ConnectionState } from '../network/NetworkClient'

export type MatchOutcome = 'playing' | 'won' | 'lost'

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
  connection: ConnectionState
  outcome: MatchOutcome
}

type HudListener = (state: GameHudState) => void

const initialState: GameHudState = {
  hp: 100,
  maxHp: 100,
  level: 1,
  experience: 0,
  experienceRequired: 13,
  remainingMs: 5 * 60 * 1000,
  kills: 0,
  enemies: 0,
  playerCount: 1,
  roomName: '',
  playerId: '',
  displayName: '',
  connection: 'connecting',
  outcome: 'playing',
}

export class GameBridge {
  private state = { ...initialState }
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
}
