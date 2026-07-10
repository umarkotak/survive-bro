import { describe, expect, it } from 'vitest'

import {
  DEFAULT_API_BASE_URL,
  DEFAULT_WEBSOCKET_BASE_URL,
  joinNetworkUrl,
  resolveNetworkConfig,
} from './network'

describe('network configuration', () => {
  it('uses the Cabocil development API by default', () => {
    expect(resolveNetworkConfig({})).toEqual({
      apiBaseUrl: DEFAULT_API_BASE_URL,
      websocketBaseUrl: DEFAULT_WEBSOCKET_BASE_URL,
    })
  })

  it('accepts local overrides and removes trailing slashes', () => {
    expect(resolveNetworkConfig({
      VITE_API_BASE_URL: 'http://localhost:3701/',
      VITE_WEBSOCKET_BASE_URL: 'ws://localhost:3701/',
    })).toEqual({
      apiBaseUrl: 'http://localhost:3701',
      websocketBaseUrl: 'ws://localhost:3701',
    })
  })

  it('joins protocol paths without duplicate slashes', () => {
    expect(joinNetworkUrl(DEFAULT_API_BASE_URL, '/api/v1/rooms/FRIDAY-SQUAD')).toBe(
      'https://survive-bro-dev-api.cabocil.com/api/v1/rooms/FRIDAY-SQUAD',
    )
  })

  it('rejects a WebSocket base URL with an HTTP protocol', () => {
    expect(() => resolveNetworkConfig({
      VITE_WEBSOCKET_BASE_URL: 'https://survive-bro-dev-api.cabocil.com',
    })).toThrow('VITE_WEBSOCKET_BASE_URL must use ws: or wss:')
  })
})
