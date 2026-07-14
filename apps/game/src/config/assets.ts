export type CharacterFrame = 'idle' | 'walk-1' | 'walk-2' | 'walk-3' | 'attack-1'

export function characterAssetPath(characterIdOrSpriteId: string, frame: CharacterFrame = 'idle'): string {
  return `/assets/character/${characterVisualId(characterIdOrSpriteId)}/${frame}.png`
}

export function enemyAssetPath(enemyIdOrSpriteId: string): string {
  return `/assets/enemy/${stripPrefix(enemyIdOrSpriteId, 'enemy-')}/idle.png`
}

export function terrainAssetPath(variant: number): string {
  return `/assets/terrain/variant-${variant}.png`
}

export function obstacleAssetPath(variant: number): string {
  return `/assets/obstacle/large-rock-${variant}.png`
}

export function characterTextureKey(characterId: string, frame: CharacterFrame): string {
  return `character-${characterVisualId(characterId)}-${frame}`
}

export function enemyTextureKey(enemyId: string): string {
  return `enemy-${enemyId}`
}

export function terrainTextureKey(variant: number): string {
  return `terrain-variant-${variant}`
}

export function obstacleTextureKey(variant: number): string {
  return `obstacle-large-rock-${variant}`
}

function stripPrefix(value: string, prefix: string): string {
  return value.startsWith(prefix) ? value.slice(prefix.length) : value
}

function characterVisualId(value: string): string {
  const id = stripPrefix(value, 'character-')
  return id === 'dummy-tester' ? 'ranger' : id
}
