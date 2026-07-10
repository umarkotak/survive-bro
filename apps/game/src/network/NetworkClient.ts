import {
  createEnvelope,
  decodeEnvelope,
  encodeEnvelope,
  type Envelope,
  type ErrorPayload,
  type JoinedPayload,
} from './protocol'
import { joinNetworkUrl, networkConfig } from '../config/network'

export type ConnectionState = 'connecting' | 'connected' | 'disconnected'
export type MessageListener = (message: Envelope) => void
export type ConnectionListener = (state: ConnectionState) => void

const webSocketConnectTimeoutMs = 8_000

export function describeHTTPError(url: string, status: number, statusText: string, serverMessage = ''): string {
  const response = `HTTP ${status}${statusText ? ` ${statusText}` : ''}`
  return `Room API failed: ${response} from ${url}${serverMessage ? ` — ${serverMessage}` : ''}`
}

export function describeWebSocketError(url: string, frontendOrigin: string, code?: number, reason = ''): string {
  const closeDetail = code === undefined
    ? 'the browser did not expose the handshake HTTP response or a close code'
    : `close code ${code}${reason ? ` (${reason})` : ''}`
  return `WebSocket failed: ${url}; frontend origin: ${frontendOrigin}; ${closeDetail}. Check Cloudflare WebSocket forwarding and ensure backend ALLOWED_ORIGINS contains ${frontendOrigin}.`
}

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
    const roomPath = `/api/v1/rooms/${encodeURIComponent(roomName)}`
    const roomUrl = joinNetworkUrl(networkConfig.apiBaseUrl, roomPath)
    let ensureResponse: Response
    try {
      ensureResponse = await fetch(roomUrl, { method: 'PUT' })
    } catch (error) {
      const detail = error instanceof Error ? error.message : String(error)
      throw new Error(`Room API network request failed: PUT ${roomUrl} — ${detail}`)
    }
    if (!ensureResponse.ok) {
      const body = (await ensureResponse.json().catch(() => null)) as { error?: ErrorPayload } | null
      throw new Error(describeHTTPError(roomUrl, ensureResponse.status, ensureResponse.statusText, body?.error?.message))
    }
    let ensured: { roomName: string }
    try {
      ensured = (await ensureResponse.json()) as { roomName: string }
    } catch {
      throw new Error(`Room API returned invalid JSON: PUT ${roomUrl} (HTTP ${ensureResponse.status})`)
    }
    this.roomName = ensured.roomName

    return new Promise<JoinedPayload>((resolve, reject) => {
      const socketPath = `/ws/v2/rooms/${encodeURIComponent(this.roomName)}`
      const socketUrl = joinNetworkUrl(networkConfig.websocketBaseUrl, socketPath)
      const frontendOrigin = window.location.origin
      const socket = new WebSocket(socketUrl)
      socket.binaryType = 'arraybuffer'
      this.socket = socket
      let settled = false
      let errorFallback: number | null = null
      const connectTimeout = window.setTimeout(() => {
        fail(describeWebSocketError(socketUrl, frontendOrigin, undefined, `timed out after ${webSocketConnectTimeoutMs}ms`))
        socket.close()
      }, webSocketConnectTimeoutMs)

      const clearConnectTimers = () => {
        window.clearTimeout(connectTimeout)
        if (errorFallback !== null) window.clearTimeout(errorFallback)
      }
      const fail = (message: string) => {
        if (settled) return
        settled = true
        clearConnectTimers()
        reject(new Error(message))
      }

      socket.addEventListener('open', () => {
        socket.send(encodeEnvelope(createEnvelope('join_room', { displayName, reconnectToken: null }, this.nextRequestId('join'))))
      })
      socket.addEventListener('message', (event) => {
        try {
          if (!(event.data instanceof ArrayBuffer)) throw new Error('Server sent a non-binary WebSocket frame')
          const message = decodeEnvelope(event.data)
          if (message.type === 'joined') {
            const joined = message.payload as JoinedPayload
            this.playerId = joined.playerId
            settled = true
            clearConnectTimers()
            this.setConnectionState('connected')
            this.startHeartbeat()
            resolve(joined)
          } else if (message.type === 'error' && !settled) {
            const error = message.payload as ErrorPayload
            fail(`Game server rejected the WebSocket join [${error.code}]: ${error.message} (${socketUrl})`)
          }
          if (message.type === 'match_started' || message.type === 'room_state' || message.type === 'snapshot' || message.type === 'match_ended') {
            this.replayMessages.set(message.type, message)
          }
          for (const listener of this.messageListeners) listener(message)
        } catch (error) {
          if (!settled) {
            const detail = error instanceof Error ? error.message : String(error)
            fail(`Invalid WebSocket message from ${socketUrl}: ${detail}`)
          }
        }
      })
      socket.addEventListener('error', () => {
        if (!settled) {
          // Browsers hide failed-handshake status and body. Give the close event a
          // moment to provide its code/reason before falling back to diagnostics.
          errorFallback = window.setTimeout(() => {
            fail(describeWebSocketError(socketUrl, frontendOrigin))
          }, 250)
        }
      })
      socket.addEventListener('close', (event) => {
        this.stopHeartbeat()
        this.setConnectionState('disconnected')
        if (!settled) {
          fail(describeWebSocketError(socketUrl, frontendOrigin, event.code, event.reason))
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
    if (this.socket?.readyState === WebSocket.OPEN) this.socket.send(encodeEnvelope(message))
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
