import Phaser from 'phaser'

import { gameAudio } from '../../audio/gameAudio'
import { diagnosticsEnabled } from '../../config/diagnostics'
import type { MultiplayerSession } from '../../network/MultiplayerSession'
import type {
  Envelope,
  MatchEndedPayload,
  MatchStartedPayload,
  ProjectileRemovedPayload,
  ProjectileSpawnedPayload,
  RoomStatePayload,
  SnapshotMonster,
  SnapshotBeam,
  SnapshotExplosion,
  SnapshotMeteor,
  SnapshotPayload,
  SnapshotPickup,
  SnapshotPlayer,
} from '../../network/protocol'
import { RANGER, SLIME_STAGES } from '../content'
import { normalizeMovement, resolveCircleOverlap, teammateEdgeIndicator } from '../model'

interface IndicatorView {
  badge: Phaser.GameObjects.Container
  arrow: Phaser.GameObjects.Triangle
}

interface PlayerView {
  sprite: Phaser.GameObjects.Image
  shadow: Phaser.GameObjects.Image
  name: Phaser.GameObjects.Text
  indicator: IndicatorView | null
  targetX: number
  targetY: number
  alive: boolean
  movementSpeed: number
  moving: boolean
  walkFrame: number
  walkElapsed: number
  attackUntil: number
  characterId: string
}

interface MonsterView {
  sprite: Phaser.GameObjects.Image
  shadow: Phaser.GameObjects.Image
  targetX: number
  targetY: number
  hp: number
  typeId: SnapshotMonster['typeId']
  shadowOffset: number
}

interface ProjectileView {
  sprite: Phaser.GameObjects.Image
  velocityX: number
  velocityY: number
}
interface BeamView { shape: Phaser.GameObjects.Rectangle }
interface ExplosionView { shape: Phaser.GameObjects.Arc }
interface MeteorView { zone: Phaser.GameObjects.Arc; marker: Phaser.GameObjects.Arc }

interface PickupView {
  sprite: Phaser.GameObjects.Image
  kind: SnapshotPickup['kind']
  targetX: number
  targetY: number
}

interface PickupAbsorption {
  sprite: Phaser.GameObjects.Image
  elapsed: number
}

const rangerDisplaySize = 132
const rockDisplaySize = 180
const walkFrameDurationMs = 160
const attackFrameDurationMs = 140
const walkFrames = [1, 2, 3, 2] as const

export class GameScene extends Phaser.Scene {
  private readonly session: MultiplayerSession
  private cursors: Phaser.Types.Input.Keyboard.CursorKeys | null = null
  private keys: Record<'W' | 'A' | 'S' | 'D', Phaser.Input.Keyboard.Key> | null = null
  private readonly players = new Map<string, PlayerView>()
  private readonly monsters = new Map<number, MonsterView>()
  private readonly pickups = new Map<number, PickupView>()
  private readonly projectiles = new Map<number, ProjectileView>()
  private readonly beams = new Map<number, BeamView>()
  private readonly explosions = new Map<number, ExplosionView>()
  private readonly meteors = new Map<number, MeteorView>()
  private readonly pickupAbsorptions: PickupAbsorption[] = []
  private readonly obstacleSprites: Phaser.GameObjects.Image[] = []
  private obstacles: MatchStartedPayload['obstacles'] = []
  private inputSequence = 0
  private lastInputSentAt = -50
  private lastMoveX = 0
  private lastMoveY = 0
  private cameraTargetId = ''
  private unsubscribeNetwork: (() => void) | null = null
  private unsubscribeConnection: (() => void) | null = null
  private unsubscribeDiagnostics: (() => void) | null = null
  private diagnosticsElapsed = 0
  private smoothedFps = 0
  private lastLocalHP: number | null = null

  constructor(session: MultiplayerSession) {
    super('GameScene')
    this.session = session
  }

  create(): void {
    this.createTerrain()
    this.cameras.main.setBounds(0, 0, 3200, 1800)

    if (this.input.keyboard) {
      this.cursors = this.input.keyboard.createCursorKeys()
      this.keys = this.input.keyboard.addKeys('W,A,S,D') as Record<'W' | 'A' | 'S' | 'D', Phaser.Input.Keyboard.Key>
    }

    this.unsubscribeNetwork = this.session.network.subscribe((message) => this.handleMessage(message))
    this.unsubscribeConnection = this.session.network.subscribeConnection((connection) => this.session.bridge.patch({ connection }))
    if (diagnosticsEnabled) {
      this.unsubscribeDiagnostics = this.session.network.subscribeDiagnostics((network) => {
        const current = this.session.bridge.getSnapshot().diagnostics
        this.session.bridge.patch({ diagnostics: { ...current, ...network } })
      })
    }
    this.events.once(Phaser.Scenes.Events.SHUTDOWN, () => this.cleanup())
  }

  update(time: number, delta: number): void {
    const seconds = Math.min(delta, 50) / 1000
    this.updateLocalInput(time, seconds)
    this.updatePlayerAnimations(time, delta)
    this.interpolatePlayers(seconds)
    this.interpolateMonsters(seconds)
    this.interpolatePickups(time, seconds)
    this.updatePickupAbsorptions(seconds)
    this.updateProjectiles(seconds)
    this.updateIndicators()
    if (diagnosticsEnabled) this.updateDiagnostics(delta)
  }

  private handleMessage(message: Envelope): void {
    switch (message.type) {
      case 'match_started':
        this.handleMatchStarted(message.payload as MatchStartedPayload)
        break
      case 'room_state':
        this.handleRoomState(message.payload as RoomStatePayload)
        break
      case 'snapshot':
        this.handleSnapshot(message.payload as SnapshotPayload)
        break
      case 'projectile_spawned':
        this.handleProjectileSpawned(message.payload as ProjectileSpawnedPayload)
        break
      case 'projectile_removed':
        this.removeProjectile((message.payload as ProjectileRemovedPayload).projectileId)
        break
      case 'match_ended':
        this.handleMatchEnded(message.payload as MatchEndedPayload)
        break
      case 'server_shutdown':
        this.session.bridge.patch({ connection: 'disconnected' })
        break
      default:
        break
    }
  }

  private handleMatchStarted(payload: MatchStartedPayload): void {
    for (const sprite of this.obstacleSprites) sprite.destroy()
    this.obstacleSprites.length = 0
    for (const view of this.monsters.values()) {
      view.sprite.destroy()
      view.shadow.destroy()
    }
    this.monsters.clear()
    for (const pickup of this.pickups.values()) pickup.sprite.destroy()
    this.pickups.clear()
    for (const absorption of this.pickupAbsorptions) absorption.sprite.destroy()
    this.pickupAbsorptions.length = 0
    for (const projectile of this.projectiles.values()) projectile.sprite.destroy()
    this.projectiles.clear()
    for (const beam of this.beams.values()) beam.shape.destroy()
    this.beams.clear()
    for (const explosion of this.explosions.values()) explosion.shape.destroy()
    this.explosions.clear()
    for (const meteor of this.meteors.values()) { meteor.zone.destroy(); meteor.marker.destroy() }
    this.meteors.clear()
    this.obstacles = payload.obstacles
    this.cameras.main.setBounds(0, 0, payload.mapWidth, payload.mapHeight)
    for (const obstacle of payload.obstacles) {
      const number = Number(obstacle.id.split('-').at(-1)) || 1
      const variant = ((number - 1) % 3) + 1
      this.obstacleSprites.push(
        this.add.image(obstacle.x, obstacle.y, `obstacle-large-rock-${variant}`)
          .setDisplaySize(rockDisplaySize, rockDisplaySize)
          .setDepth(obstacle.y)
          .setData('obstacle-id', obstacle.id),
      )
    }
    this.session.bridge.patch({ roomName: payload.roomName, outcome: 'playing', levelDurationMs: payload.durationMs, timelineEvents: payload.events, bosses: [] })
  }

  private handleRoomState(payload: RoomStatePayload): void {
    this.session.bridge.patch({ playerCount: payload.players.length })
  }

  private handleSnapshot(snapshot: SnapshotPayload): void {
    const localPlayerId = this.session.network.playerId
    const activePlayerIds = new Set(snapshot.players.map((player) => player.id))

    for (const player of snapshot.players) {
      const view = this.ensurePlayerView(player)
      view.alive = player.alive
      view.sprite.setFlipX(player.facing === 'left').setAlpha(player.alive ? 1 : 0.42)
      view.shadow.setAlpha(player.alive ? 0.9 : 0.25)
      view.targetX = player.x
      view.targetY = player.y
      view.movementSpeed = player.movementSpeed
      view.moving = Math.hypot(player.velocityX, player.velocityY) > 1

      if (player.id === localPlayerId) {
        const error = Phaser.Math.Distance.Between(view.sprite.x, view.sprite.y, player.x, player.y)
        if (error > 80) view.sprite.setPosition(player.x, player.y)
        else view.sprite.setPosition(Phaser.Math.Linear(view.sprite.x, player.x, 0.35), Phaser.Math.Linear(view.sprite.y, player.y, 0.35))
      }
    }
    for (const [playerId, view] of this.players) {
      if (!activePlayerIds.has(playerId)) {
        this.destroyPlayerView(view)
        this.players.delete(playerId)
      }
    }

    this.syncMonsters(snapshot.monsters)
    this.syncBeams(snapshot.beams)
    this.syncExplosions(snapshot.explosions)
    this.syncMeteors(snapshot.meteors)
    this.syncPickups(snapshot.pickups)
    this.updateCameraTarget(snapshot.players)

    const local = snapshot.players.find((player) => player.id === localPlayerId)
    if (local) {
      if (this.lastLocalHP !== null && local.hp < this.lastLocalHP) gameAudio.damage()
      this.lastLocalHP = local.hp
      this.session.bridge.patch({
        hp: local.hp,
        maxHp: local.maxHp,
        level: snapshot.team.level,
        experience: snapshot.team.experience,
        experienceRequired: snapshot.team.experienceRequired,
        remainingMs: snapshot.remainingMs,
        kills: snapshot.team.totalKills,
        enemies: snapshot.monsters.length,
        bosses: snapshot.monsters.filter((monster) => monster.isBoss).map((monster) => ({ id: monster.id, name: enemyDisplayName(monster.typeId), spriteId: `enemy-${monster.typeId}`, hp: monster.hp, maxHp: monster.maxHp })),
        playerCount: snapshot.players.length,
        armorPercent: local.armorPercent,
        movementSpeed: local.movementSpeed,
        healthRegeneration: local.healthRegeneration,
        attackBuffPercent: local.attackBuffPercent,
        cooldownPercent: local.cooldownPercent,
        spellDamage: local.spellDamage,
        projectileSpeed: local.projectileSpeed,
        spellBurst: local.spellBurst,
        spellDirections: local.spellDirections,
      })
    }
  }

  private handleProjectileSpawned(payload: ProjectileSpawnedPayload): void {
    const owner = this.players.get(payload.ownerId)
    if (owner) owner.attackUntil = this.time.now + attackFrameDurationMs
    if (payload.ownerId === this.session.network.playerId) gameAudio.fireball()
    const angle = Math.atan2(payload.velocityY, payload.velocityX)
    const sprite = this.add.image(payload.x, payload.y, payload.weaponId === 'rocket' ? 'rocket' : 'arc-bolt').setRotation(angle).setDepth(payload.y + 2)
    this.projectiles.set(payload.projectileId, { sprite, velocityX: payload.velocityX, velocityY: payload.velocityY })
  }

  private handleMatchEnded(payload: MatchEndedPayload): void {
    this.session.bridge.patch({
      outcome: payload.outcome,
      level: payload.teamLevel,
      kills: payload.totalKills,
      remainingMs: Math.max(0, 6 * 60 * 1000 - payload.survivalMs),
      score: payload.score,
    })
  }

  private updateLocalInput(time: number, seconds: number): void {
    const local = this.players.get(this.session.network.playerId)
    if (!local || !local.alive) return

    const horizontal = Number(this.cursors?.right.isDown || this.keys?.D.isDown) - Number(this.cursors?.left.isDown || this.keys?.A.isDown)
    const vertical = Number(this.cursors?.down.isDown || this.keys?.S.isDown) - Number(this.cursors?.up.isDown || this.keys?.W.isDown)
    const virtual = this.session.bridge.getVirtualMovement()
    const movement = normalizeMovement(horizontal + virtual.x, vertical + virtual.y)
    local.moving = movement.x !== 0 || movement.y !== 0
    const changed = movement.x !== this.lastMoveX || movement.y !== this.lastMoveY

    if (changed || time-this.lastInputSentAt >= 50) {
      this.inputSequence += 1
      this.session.network.sendInput(this.inputSequence, movement.x, movement.y)
      this.lastInputSentAt = time
      this.lastMoveX = movement.x
      this.lastMoveY = movement.y
    }

    if (movement.x !== 0) local.sprite.setFlipX(movement.x < 0)
    let predicted = {
      x: Phaser.Math.Clamp(local.sprite.x + movement.x * local.movementSpeed * seconds, RANGER.collisionRadius, 3200 - RANGER.collisionRadius),
      y: Phaser.Math.Clamp(local.sprite.y + movement.y * local.movementSpeed * seconds, RANGER.collisionRadius, 1800 - RANGER.collisionRadius),
    }
    for (const obstacle of this.obstacles) {
      predicted = resolveCircleOverlap({ ...predicted, radius: RANGER.collisionRadius }, obstacle)
    }
    local.sprite.setPosition(predicted.x, predicted.y)
  }

  private interpolatePlayers(seconds: number): void {
    const blend = 1 - Math.exp(-12 * seconds)
    for (const [playerId, view] of this.players) {
      if (playerId !== this.session.network.playerId) {
        view.sprite.setPosition(
          Phaser.Math.Linear(view.sprite.x, view.targetX, blend),
          Phaser.Math.Linear(view.sprite.y, view.targetY, blend),
        )
      }
      view.sprite.setDepth(view.sprite.y)
      view.shadow.setPosition(view.sprite.x, view.sprite.y + 48).setDepth(view.sprite.y - 1)
      view.name.setPosition(view.sprite.x, view.sprite.y - 60).setDepth(view.sprite.y + 1)
    }
  }

  private updatePlayerAnimations(time: number, delta: number): void {
    for (const view of this.players.values()) {
      if (time < view.attackUntil) {
        view.sprite.setTexture(`character-${view.characterId}-attack-1`)
        continue
      }
      if (!view.moving) {
        view.walkFrame = 0
        view.walkElapsed = 0
        view.sprite.setTexture(`character-${view.characterId}-idle`)
        continue
      }
      view.walkElapsed += delta
      while (view.walkElapsed >= walkFrameDurationMs) {
        view.walkElapsed -= walkFrameDurationMs
        view.walkFrame = (view.walkFrame + 1) % walkFrames.length
      }
      view.sprite.setTexture(`character-${view.characterId}-walk-${walkFrames[view.walkFrame]}`)
    }
  }

  private interpolateMonsters(seconds: number): void {
    const blend = 1 - Math.exp(-14 * seconds)
    for (const view of this.monsters.values()) {
      const previousX = view.sprite.x
      view.sprite.setPosition(
        Phaser.Math.Linear(view.sprite.x, view.targetX, blend),
        Phaser.Math.Linear(view.sprite.y, view.targetY, blend),
      )
      if (view.sprite.x !== previousX) view.sprite.setFlipX(view.sprite.x < previousX)
      view.sprite.setDepth(view.sprite.y)
      view.shadow.setPosition(view.sprite.x, view.sprite.y + view.shadowOffset).setDepth(view.sprite.y - 1)
    }
  }

  private updateProjectiles(seconds: number): void {
    for (const view of this.projectiles.values()) {
      view.sprite.x += view.velocityX * seconds
      view.sprite.y += view.velocityY * seconds
      view.sprite.setDepth(view.sprite.y + 2)
    }
  }

  private interpolatePickups(time: number, seconds: number): void {
    const blend = 1 - Math.exp(-20 * seconds)
    for (const view of this.pickups.values()) {
      const distance = Phaser.Math.Distance.Between(view.sprite.x, view.sprite.y, view.targetX, view.targetY)
      view.sprite.setPosition(
        Phaser.Math.Linear(view.sprite.x, view.targetX, blend),
        Phaser.Math.Linear(view.sprite.y, view.targetY, blend),
      )
      if (view.kind === 'experience') {
        view.sprite.rotation += seconds * (distance > 4 ? 8 : 2)
        const pulse = 1 + Math.sin(time / 130) * 0.08
        view.sprite.setScale(distance > 4 ? pulse * 1.2 : pulse)
      } else {
        view.sprite.setY(view.sprite.y + Math.sin(time / 180) * 0.08)
      }
      view.sprite.setDepth(view.sprite.y)
    }
  }

  private updatePickupAbsorptions(seconds: number): void {
    for (let index = this.pickupAbsorptions.length - 1; index >= 0; index -= 1) {
      const absorption = this.pickupAbsorptions[index]
      absorption.elapsed += seconds
      const progress = Math.min(1, absorption.elapsed / 0.14)
      absorption.sprite.setScale(1.2 * (1 - progress)).setAlpha(1 - progress)
      if (progress === 1) {
        absorption.sprite.destroy()
        this.pickupAbsorptions.splice(index, 1)
      }
    }
  }

  private updateIndicators(): void {
    const camera = this.cameras.main
    for (const [playerId, view] of this.players) {
      if (playerId === this.session.network.playerId || !view.indicator || !view.alive) {
        view.indicator?.badge.setVisible(false)
        view.indicator?.arrow.setVisible(false)
        continue
      }
      const edge = teammateEdgeIndicator(camera.worldView.x, camera.worldView.y, camera.width, camera.height, view.sprite.x, view.sprite.y)
      view.indicator.badge.setVisible(edge.visible).setPosition(edge.x, edge.y)
      view.indicator.arrow
        .setVisible(edge.visible)
        .setPosition(edge.x + Math.cos(edge.angle) * 28, edge.y + Math.sin(edge.angle) * 28)
        .setRotation(edge.angle)
    }
  }

  private updateDiagnostics(delta: number): void {
    const instantFps = delta > 0 ? 1000 / delta : 0
    this.smoothedFps = this.smoothedFps === 0 ? instantFps : Phaser.Math.Linear(this.smoothedFps, instantFps, 0.08)
    this.diagnosticsElapsed += delta
    if (this.diagnosticsElapsed < 250) return
    this.diagnosticsElapsed = 0

    const camera = this.cameras.main.worldView
    let activeSprites = 0
    let visibleSprites = 0
    const count = (sprite: Phaser.GameObjects.Image) => {
      if (!sprite.active) return
      activeSprites++
      if (sprite.visible && sprite.x >= camera.x && sprite.x <= camera.x + camera.width && sprite.y >= camera.y && sprite.y <= camera.y + camera.height) {
        visibleSprites++
      }
    }
    for (const view of this.players.values()) count(view.sprite)
    for (const view of this.monsters.values()) count(view.sprite)
    for (const view of this.pickups.values()) count(view.sprite)
    for (const view of this.projectiles.values()) count(view.sprite)
    for (const absorption of this.pickupAbsorptions) count(absorption.sprite)
    for (const obstacle of this.obstacleSprites) count(obstacle)

    const current = this.session.bridge.getSnapshot().diagnostics
    this.session.bridge.patch({
      diagnostics: {
        ...current,
        fps: this.smoothedFps,
        activeSprites,
        visibleSprites,
        projectiles: this.projectiles.size,
      },
    })
  }

  private ensurePlayerView(player: SnapshotPlayer): PlayerView {
    const existing = this.players.get(player.id)
    if (existing) return existing

    const isLocal = player.id === this.session.network.playerId
    const color = playerColor(player.id)
    const shadow = this.add.image(player.x, player.y + 48, 'entity-shadow').setDisplaySize(94, 38)
    const sprite = this.add.image(player.x, player.y, `character-${player.characterId}-idle`).setDisplaySize(rangerDisplaySize, rangerDisplaySize)
    if (!isLocal) sprite.setTint(color)
    const name = this.add.text(player.x, player.y - 60, player.displayName, {
      fontFamily: 'Inter, system-ui, sans-serif',
      fontSize: '13px',
      fontStyle: 'bold',
      color: '#f6fff7',
      backgroundColor: 'rgba(7, 19, 13, 0.68)',
      padding: { x: 7, y: 3 },
    }).setOrigin(0.5)

    const indicator = isLocal ? null : this.createIndicator(player.displayName, color)
    const view: PlayerView = {
      sprite,
      shadow,
      name,
      indicator,
      targetX: player.x,
      targetY: player.y,
      alive: player.alive,
      movementSpeed: player.movementSpeed,
      moving: Math.hypot(player.velocityX, player.velocityY) > 1,
      walkFrame: 0,
      walkElapsed: 0,
      attackUntil: 0,
      characterId: player.characterId,
    }
    this.players.set(player.id, view)
    return view
  }

  private createIndicator(displayName: string, color: number): IndicatorView {
    const circle = this.add.circle(0, 0, 21, color, 0.94).setStrokeStyle(2, 0xffffff, 0.8)
    const label = this.add.text(0, 0, initials(displayName), {
      fontFamily: 'Inter, system-ui, sans-serif',
      fontSize: '11px',
      fontStyle: 'bold',
      color: '#102018',
    }).setOrigin(0.5)
    const badge = this.add.container(0, 0, [circle, label]).setScrollFactor(0).setDepth(20_000).setVisible(false)
    const arrow = this.add.triangle(0, 0, -7, -7, 10, 0, -7, 7, color).setScrollFactor(0).setDepth(19_999).setVisible(false)
    return { badge, arrow }
  }

  private syncMonsters(monsters: SnapshotMonster[]): void {
    const active = new Set(monsters.map((monster) => monster.id))
    for (const monster of monsters) {
      let view = this.monsters.get(monster.id)
      if (!view) {
        const content = SLIME_STAGES[monster.typeId]
        const displaySize = content.displaySize
        view = {
          shadow: this.add.image(monster.x, monster.y + displaySize * 0.3, 'entity-shadow').setDisplaySize(displaySize * 0.72, displaySize * 0.26),
          sprite: this.add.image(monster.x, monster.y, content.texture).setDisplaySize(displaySize, displaySize),
          targetX: monster.x,
          targetY: monster.y,
          hp: monster.hp,
          typeId: monster.typeId,
          shadowOffset: displaySize * 0.3,
        }
        this.monsters.set(monster.id, view)
      }
      view.targetX = monster.x
      view.targetY = monster.y
      if (monster.hp < view.hp) {
        view.sprite.setTint(0xe8ffff).setTintMode(Phaser.TintModes.FILL)
        this.time.delayedCall(55, () => view?.sprite.active && view.sprite.clearTint())
      }
      view.hp = monster.hp
    }
    for (const [id, view] of this.monsters) {
      if (!active.has(id)) {
        view.sprite.destroy()
        view.shadow.destroy()
        this.monsters.delete(id)
      }
    }
  }

  private syncBeams(beams: SnapshotBeam[]): void {
    const active = new Set(beams.map((beam) => beam.id))
    for (const beam of beams) {
      let view = this.beams.get(beam.id)
      if (!view) {
        view = { shape: this.add.rectangle(beam.x, beam.y, beam.length, beam.width, 0xb8f3ff, 0.72).setOrigin(0, 0.5).setStrokeStyle(2, 0xffffff, 0.88).setBlendMode(Phaser.BlendModes.ADD) }
        this.beams.set(beam.id, view)
        const owner = this.players.get(beam.ownerId)
        if (owner) owner.attackUntil = this.time.now + Math.min(beam.remainingMs, 1000)
        if (beam.ownerId === this.session.network.playerId) gameAudio.soulTrack()
      }
      view.shape.setPosition(beam.x, beam.y).setRotation(beam.angle).setDisplaySize(beam.length, beam.width).setDepth(beam.y + 1)
    }
    for (const [id, view] of this.beams) {
      if (!active.has(id)) { view.shape.destroy(); this.beams.delete(id) }
    }
  }

  private syncExplosions(explosions: SnapshotExplosion[]): void {
    const active = new Set(explosions.map((explosion) => explosion.id))
    for (const explosion of explosions) {
      let view = this.explosions.get(explosion.id)
      if (!view) {
        view = { shape: this.add.circle(explosion.x, explosion.y, explosion.radius, 0xff7b35, 0.34).setStrokeStyle(5, 0xffd36a, 0.8).setBlendMode(Phaser.BlendModes.ADD) }
        this.explosions.set(explosion.id, view)
      }
      const pulse = 0.92 + Math.sin(this.time.now / 70) * 0.08
      view.shape.setPosition(explosion.x, explosion.y).setRadius(explosion.radius).setScale(pulse).setDepth(explosion.y + 1)
    }
    for (const [id, view] of this.explosions) {
      if (!active.has(id)) { view.shape.destroy(); this.explosions.delete(id) }
    }
  }

  private syncMeteors(meteors: SnapshotMeteor[]): void {
	const active = new Set(meteors.map((meteor) => meteor.id))
	for (const meteor of meteors) {
		let view = this.meteors.get(meteor.id)
		if (!view) {
			view = {
				zone: this.add.circle(meteor.x, meteor.y, meteor.radius, 0xff3d18, 0.08).setStrokeStyle(5, 0xffc247, 0.9),
				marker: this.add.circle(meteor.x, meteor.y, 12, 0xfff0b0, 0.95).setStrokeStyle(3, 0xff4b1f, 1),
			}
			this.meteors.set(meteor.id, view)
		}
		const warning = meteor.impactInMs > 0
		const pulse = 0.94 + Math.sin(this.time.now / (warning ? 90 : 55)) * 0.06
		view.zone.setPosition(meteor.x, meteor.y).setRadius(meteor.radius).setScale(pulse).setFillStyle(warning ? 0xff3d18 : 0xff6a18, warning ? 0.08 : 0.38).setStrokeStyle(warning ? 5 : 2, warning ? 0xffc247 : 0xff7b35, warning ? 0.9 : 0.5).setDepth(meteor.y + 1)
		view.marker.setPosition(meteor.x, warning ? meteor.y - Math.min(180, 30 + meteor.impactInMs * 0.1) : meteor.y).setScale(warning ? 1 : 2.2).setAlpha(warning ? 1 : 0.35).setDepth(meteor.y + 2)
	}
	for (const [id, view] of this.meteors) {
		if (!active.has(id)) { view.zone.destroy(); view.marker.destroy(); this.meteors.delete(id) }
	}
  }

  private syncPickups(pickups: SnapshotPickup[]): void {
    const active = new Set(pickups.map((pickup) => pickup.id))
    for (const pickup of pickups) {
      let view = this.pickups.get(pickup.id)
      if (!view) {
        const texture = pickup.kind === 'power_crate' ? 'power-crate' : 'experience'
        view = { sprite: this.add.image(pickup.x, pickup.y, texture).setDepth(pickup.y), kind: pickup.kind, targetX: pickup.x, targetY: pickup.y }
        this.pickups.set(pickup.id, view)
      }
      view.targetX = pickup.x
      view.targetY = pickup.y
    }
    for (const [id, view] of this.pickups) {
      if (!active.has(id)) {
        this.pickups.delete(id)
        this.pickupAbsorptions.push({ sprite: view.sprite, elapsed: 0 })
      }
    }
  }

  private updateCameraTarget(players: SnapshotPlayer[]): void {
    const localId = this.session.network.playerId
    const local = players.find((player) => player.id === localId)
    let targetId = local?.alive ? localId : players.find((player) => player.alive)?.id ?? ''
    if (!targetId) targetId = localId
    if (targetId !== this.cameraTargetId) {
      const target = this.players.get(targetId)
      if (target) {
        this.cameras.main.startFollow(target.sprite, true, 0.12, 0.12)
        this.cameraTargetId = targetId
      }
    }
  }

  private removeProjectile(id: number): void {
    this.projectiles.get(id)?.sprite.destroy()
    this.projectiles.delete(id)
  }

  private destroyPlayerView(view: PlayerView): void {
    view.sprite.destroy()
    view.shadow.destroy()
    view.name.destroy()
    view.indicator?.badge.destroy(true)
    view.indicator?.arrow.destroy()
  }

  private cleanup(): void {
    this.session.bridge.setVirtualMovement(0, 0)
    this.lastLocalHP = null
    for (const beam of this.beams.values()) beam.shape.destroy()
    this.beams.clear()
    for (const explosion of this.explosions.values()) explosion.shape.destroy()
    this.explosions.clear()
    for (const meteor of this.meteors.values()) { meteor.zone.destroy(); meteor.marker.destroy() }
    this.meteors.clear()
    for (const absorption of this.pickupAbsorptions) absorption.sprite.destroy()
    this.pickupAbsorptions.length = 0
    this.unsubscribeNetwork?.()
    this.unsubscribeConnection?.()
    this.unsubscribeDiagnostics?.()
    this.unsubscribeNetwork = null
    this.unsubscribeConnection = null
    this.unsubscribeDiagnostics = null
  }

  private createTerrain(): void {
    const tileSize = 256
    const columns = Math.ceil(3200 / tileSize)
    const rows = Math.ceil(1800 / tileSize)
    for (let row = 0; row < rows; row += 1) {
      for (let column = 0; column < columns; column += 1) {
        const variant = ((column * 7 + row * 11) % 3) + 1
        this.add.image(
          column * tileSize + tileSize / 2,
          row * tileSize + tileSize / 2,
          `terrain-variant-${variant}`,
        ).setDepth(-20)
      }
    }
  }
}

function enemyDisplayName(typeId: SnapshotMonster['typeId']): string {
  if (typeId === 'slime-stage-3') return 'Slime King'
  return typeId.split('-').map((part) => part[0]?.toUpperCase() + part.slice(1)).join(' ')
}

function initials(name: string): string {
  return name.trim().split(/\s+/).slice(0, 2).map((part) => part[0]?.toUpperCase() ?? '').join('') || '?'
}

function playerColor(id: string): number {
  const colors = [0x74ddff, 0xffb66e, 0xc79cff, 0xff8fa2]
  let hash = 0
  for (const character of id) hash = (hash * 31 + character.charCodeAt(0)) >>> 0
  return colors[hash % colors.length]
}
