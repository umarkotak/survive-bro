import { describe, expect, it } from 'vitest'

import { describeHTTPError, describeWebSocketError } from './NetworkClient'

describe('network diagnostics', () => {
  it('reports the HTTP target, status, and server message', () => {
    expect(describeHTTPError('https://api.example/rooms/TEST', 403, 'Forbidden', 'denied')).toBe(
      'Room API failed: HTTP 403 Forbidden from https://api.example/rooms/TEST — denied',
    )
  })

  it('reports the WebSocket target, browser origin, and close code', () => {
    expect(describeWebSocketError(
      'wss://api.example/ws/TEST',
      'https://game.example',
      1006,
    )).toContain('wss://api.example/ws/TEST; frontend origin: https://game.example; close code 1006')
  })
})
