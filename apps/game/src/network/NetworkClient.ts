import {
  createEnvelope,
  parseEnvelope,
  type Envelope,
  type ErrorPayload,
  type JoinedPayload,
} from './protocol'

export type ConnectionState = 'connecting' | 'connected' | 'disconnected'
export type MessageListener = (message: Envelope) => void
export type ConnectionListener = (state: ConnectionState) => void

export class NetworkClient {
  private socket: WebSocket | null = null
  private readonly messageListeners = new Set<MessageListener>()
  private readonly connectionListeners = new Set<ConnectionListener>()
  private readonly replayMessages = new Map<string, Envelope>()
  private heartbeat: number | null = null
  private requestSequence = 0
  playerId = ''
  roomName = ''

  async connect(roomName: string, displayName: string): Promise<JoinedPayload> {
    this.setConnectionState('connecting')
    const ensureResponse = await fetch(`/api/v1/rooms/${encodeURIComponent(roomName)}`, { method: 'PUT' })
    if (!ensureResponse.ok) {
      const body = (await ensureResponse.json().catch(() => null)) as { error?: ErrorPayload } | null
      throw new Error(body?.error?.message ?? 'Could not create or find the room')
    }
    const ensured = (await ensureResponse.json()) as { roomName: string }
    this.roomName = ensured.roomName

    return new Promise<JoinedPayload>((resolve, reject) => {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const socket = new WebSocket(`${protocol}//${window.location.host}/ws/v1/rooms/${encodeURIComponent(this.roomName)}`)
      this.socket = socket
      let settled = false

      socket.addEventListener('open', () => {
        socket.send(JSON.stringify(createEnvelope('join_room', { displayName, reconnectToken: null }, this.nextRequestId('join'))))
      })
      socket.addEventListener('message', (event) => {
        try {
          const message = parseEnvelope(String(event.data))
          if (message.type === 'joined') {
            const joined = message.payload as JoinedPayload
            this.playerId = joined.playerId
            settled = true
            this.setConnectionState('connected')
            this.startHeartbeat()
            resolve(joined)
          } else if (message.type === 'error' && !settled) {
            const error = message.payload as ErrorPayload
            settled = true
            reject(new Error(error.message))
          }
          if (message.type === 'match_started' || message.type === 'room_state' || message.type === 'snapshot' || message.type === 'match_ended') {
            this.replayMessages.set(message.type, message)
          }
          for (const listener of this.messageListeners) listener(message)
        } catch (error) {
          if (!settled) {
            settled = true
            reject(error instanceof Error ? error : new Error('Invalid server response'))
          }
        }
      })
      socket.addEventListener('error', () => {
        if (!settled) {
          settled = true
          reject(new Error('Could not connect to the game server'))
        }
      })
      socket.addEventListener('close', () => {
        this.stopHeartbeat()
        this.setConnectionState('disconnected')
        if (!settled) {
          settled = true
          reject(new Error('The game server closed the connection'))
        }
      })
    })
  }

  subscribe(listener: MessageListener): () => void {
    this.messageListeners.add(listener)
    for (const messageType of ['match_started', 'room_state', 'snapshot', 'match_ended']) {
      const message = this.replayMessages.get(messageType)
      if (message) listener(message)
    }
    return () => this.messageListeners.delete(listener)
  }

  subscribeConnection(listener: ConnectionListener): () => void {
    this.connectionListeners.add(listener)
    return () => this.connectionListeners.delete(listener)
  }

  sendInput(sequence: number, moveX: number, moveY: number): void {
    this.send(createEnvelope('input', { sequence, moveX, moveY }))
  }

  close(): void {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.send(createEnvelope('leave_room', {}, this.nextRequestId('leave')))
    }
    this.stopHeartbeat()
    this.socket?.close()
    this.socket = null
    this.replayMessages.clear()
    this.setConnectionState('disconnected')
  }

  private send(message: Envelope): void {
    if (this.socket?.readyState === WebSocket.OPEN) this.socket.send(JSON.stringify(message))
  }

  private startHeartbeat(): void {
    this.stopHeartbeat()
    this.heartbeat = window.setInterval(() => {
      this.send(createEnvelope('ping', {}, this.nextRequestId('ping')))
    }, 10_000)
  }

  private stopHeartbeat(): void {
    if (this.heartbeat !== null) window.clearInterval(this.heartbeat)
    this.heartbeat = null
  }

  private setConnectionState(state: ConnectionState): void {
    for (const listener of this.connectionListeners) listener(state)
  }

  private nextRequestId(prefix: string): string {
    this.requestSequence += 1
    return `${prefix}-${this.requestSequence}`
  }
}
