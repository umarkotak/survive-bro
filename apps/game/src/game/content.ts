export const RANGER = {
  id: 'ranger',
  name: 'Ranger',
  baseHp: 100,
  movementSpeed: 220,
  pickupRadius: 120,
  collisionRadius: 30,
  weapon: {
    id: 'arc_bolt',
    damage: 20,
    cooldownMs: 750,
    projectileSpeed: 700,
    range: 700,
    projectileRadius: 10,
  },
} as const

export const CRAWLER = {
  id: 'crawler',
  name: 'Crawler',
  baseHp: 40,
  movementSpeed: 80,
  contactDamage: 10,
  contactCooldownMs: 800,
  collisionRadius: 24,
  experienceValue: 1,
} as const

export const LARGE_ROCK = {
  id: 'large_rock',
  collisionRadius: 65,
} as const

export const MEADOW_ROCKS = [
  { x: 480, y: 360 },
  { x: 930, y: 280 },
  { x: 1380, y: 420 },
  { x: 2140, y: 330 },
  { x: 2750, y: 430 },
  { x: 580, y: 1260 },
  { x: 1080, y: 1480 },
  { x: 2030, y: 1390 },
  { x: 2600, y: 1250 },
  { x: 2920, y: 900 },
] as const
