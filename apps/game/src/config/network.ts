export const DEFAULT_API_BASE_URL = 'https://survive-bro-dev-api.cabocil.com'
export const DEFAULT_WEBSOCKET_BASE_URL = 'wss://survive-bro-dev-api.cabocil.com'

export interface NetworkConfig {
  apiBaseUrl: string
  websocketBaseUrl: string
}

interface NetworkEnvironment {
  VITE_API_BASE_URL?: string
  VITE_WEBSOCKET_BASE_URL?: string
}

export function resolveNetworkConfig(environment: NetworkEnvironment): NetworkConfig {
  return {
    apiBaseUrl: normalizeBaseUrl(
      environment.VITE_API_BASE_URL,
      DEFAULT_API_BASE_URL,
      ['http:', 'https:'],
      'VITE_API_BASE_URL',
    ),
    websocketBaseUrl: normalizeBaseUrl(
      environment.VITE_WEBSOCKET_BASE_URL,
      DEFAULT_WEBSOCKET_BASE_URL,
      ['ws:', 'wss:'],
      'VITE_WEBSOCKET_BASE_URL',
    ),
  }
}

export function joinNetworkUrl(baseUrl: string, path: string): string {
  return new URL(path.replace(/^\/+/, ''), `${baseUrl}/`).toString()
}

function normalizeBaseUrl(
  configuredValue: string | undefined,
  fallback: string,
  allowedProtocols: string[],
  variableName: string,
): string {
  const value = configuredValue?.trim() || fallback
  let url: URL
  try {
    url = new URL(value)
  } catch {
    throw new Error(`${variableName} must be an absolute URL`)
  }
  if (!allowedProtocols.includes(url.protocol)) {
    throw new Error(`${variableName} must use ${allowedProtocols.join(' or ')}`)
  }
  if (url.username || url.password || url.search || url.hash) {
    throw new Error(`${variableName} cannot contain credentials, a query, or a fragment`)
  }
  return url.toString().replace(/\/$/, '')
}

export const networkConfig = resolveNetworkConfig(import.meta.env)
