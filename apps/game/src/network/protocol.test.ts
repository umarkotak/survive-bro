import { describe, expect, it } from 'vitest'

import { createEnvelope, decodeEnvelope, encodeEnvelope, type SnapshotPayload } from './protocol'

describe('binary protocol v2', () => {
  it('matches the Go input golden frame', () => {
    const encoded = encodeEnvelope(createEnvelope('input', { sequence: 154, moveX: 0.5, moveY: -1 }))
    expect(toHex(encoded)).toBe('0204009a0000000000003f000080bf')
    expect(decodeEnvelope(encoded)).toEqual(createEnvelope('input', { sequence: 154, moveX: 0.5, moveY: -1 }))
  })

  it('round trips snapshots with entity arrays', () => {
    const payload: SnapshotPayload = {
      tick: 42,
      serverTimeMs: 1_780_000_000_000,
      players: [{
        id: 'p_1', displayName: 'Umar', characterId: 'ranger', x: 1600.25, y: 900.5, velocityX: 220, velocityY: 0,
        movementSpeed: 240, armorPercent: 0.125, healthRegeneration: 1, attackBuffPercent: 0.1, cooldownPercent: 0.08,
        spellDamage: 24, projectileSpeed: 770, spellBurst: 1, spellDirections: 2,
        facing: 'left', hp: 90, maxHp: 100, alive: true, lastProcessedInput: 154, kills: 2,
        resurrectionDurationMs: 2000, resurrectionRadius: 120, resurrectionImmunityDurationMs: 5000,
        resurrectionProgress: 0, resurrectionPending: false, immunityRemainingMs: 0,
      }],
      monsters: [{ id: 7, typeId: 'slime-stage-1', x: 2100, y: 840, hp: 20, maxHp: 40, isBoss: false }],
      beams: [], explosions: [], meteors: [],
      pickups: [{ id: 4, kind: 'experience', x: 2020, y: 810 }],
      team: { level: 2, experience: 3, experienceRequired: 22, totalKills: 4, lives: 2 },
      remainingMs: 286_400,
    }
    const decoded = decodeEnvelope(encodeEnvelope(createEnvelope('snapshot', payload)))
    expect(decoded.type).toBe('snapshot')
    expect(decoded.payload).toEqual(payload)
  })

  it('rejects unsupported, truncated, and trailing frames', () => {
    expect(() => decodeEnvelope(new Uint8Array([1, 3, 0]))).toThrow('Unsupported protocol version')
    expect(() => decodeEnvelope(new Uint8Array([2, 4, 0]))).toThrow('truncated')
    expect(() => decodeEnvelope(new Uint8Array([2, 3, 0, 1]))).toThrow('trailing')
  })
})

function toHex(bytes: Uint8Array): string {
  return Array.from(bytes, (value) => value.toString(16).padStart(2, '0')).join('')
}
