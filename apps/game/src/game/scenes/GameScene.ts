import Phaser from 'phaser'

import type { MultiplayerSession } from '../../network/MultiplayerSession'
import type {
  Envelope,
  MatchEndedPayload,
  MatchStartedPayload,
  ProjectileRemovedPayload,
  ProjectileSpawnedPayload,
  RoomStatePayload,
  SnapshotMonster,
  SnapshotPayload,
  SnapshotPickup,
  SnapshotPlayer,
} from '../../network/protocol'
import { RANGER } from '../content'
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
}

interface MonsterView {
  sprite: Phaser.GameObjects.Image
  shadow: Phaser.GameObjects.Image
  targetX: number
  targetY: number
  hp: number
}

interface ProjectileView {
  sprite: Phaser.GameObjects.Image
  velocityX: number
  velocityY: number
}

export class GameScene extends Phaser.Scene {
  private readonly session: MultiplayerSession
  private cursors: Phaser.Types.Input.Keyboard.CursorKeys | null = null
  private keys: Record<'W' | 'A' | 'S' | 'D', Phaser.Input.Keyboard.Key> | null = null
  private readonly players = new Map<string, PlayerView>()
  private readonly monsters = new Map<number, MonsterView>()
  private readonly pickups = new Map<number, Phaser.GameObjects.Image>()
  private readonly projectiles = new Map<number, ProjectileView>()
  private readonly obstacleSprites: Phaser.GameObjects.Image[] = []
  private obstacles: MatchStartedPayload['obstacles'] = []
  private inputSequence = 0
  private lastInputSentAt = -50
  private lastMoveX = 0
  private lastMoveY = 0
  private cameraTargetId = ''
  private unsubscribeNetwork: (() => void) | null = null
  private unsubscribeConnection: (() => void) | null = null

  constructor(session: MultiplayerSession) {
    super('GameScene')
    this.session = session
  }

  create(): void {
    this.add.tileSprite(1600, 900, 3200, 1800, 'meadow-ground').setDepth(-20)
    this.cameras.main.setBounds(0, 0, 3200, 1800)

    if (this.input.keyboard) {
      this.cursors = this.input.keyboard.createCursorKeys()
      this.keys = this.input.keyboard.addKeys('W,A,S,D') as Record<'W' | 'A' | 'S' | 'D', Phaser.Input.Keyboard.Key>
    }

    this.unsubscribeNetwork = this.session.network.subscribe((message) => this.handleMessage(message))
    this.unsubscribeConnection = this.session.network.subscribeConnection((connection) => this.session.bridge.patch({ connection }))
    this.events.once(Phaser.Scenes.Events.SHUTDOWN, () => this.cleanup())
  }

  update(time: number, delta: number): void {
    const seconds = Math.min(delta, 50) / 1000
    this.updateLocalInput(time, seconds)
    this.interpolatePlayers(seconds)
    this.interpolateMonsters(seconds)
    this.updateProjectiles(seconds)
    this.updateIndicators()
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
    for (const pickup of this.pickups.values()) pickup.destroy()
    this.pickups.clear()
    for (const projectile of this.projectiles.values()) projectile.sprite.destroy()
    this.projectiles.clear()
    this.obstacles = payload.obstacles
    this.cameras.main.setBounds(0, 0, payload.mapWidth, payload.mapHeight)
    for (const obstacle of payload.obstacles) {
      this.obstacleSprites.push(this.add.image(obstacle.x, obstacle.y, 'large-rock').setDepth(obstacle.y).setData('obstacle-id', obstacle.id))
    }
    this.session.bridge.patch({ roomName: payload.roomName, outcome: 'playing' })
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
    this.syncPickups(snapshot.pickups)
    this.updateCameraTarget(snapshot.players)

    const local = snapshot.players.find((player) => player.id === localPlayerId)
    if (local) {
      this.session.bridge.patch({
        hp: local.hp,
        maxHp: local.maxHp,
        level: snapshot.team.level,
        experience: snapshot.team.experience,
        experienceRequired: snapshot.team.experienceRequired,
        remainingMs: snapshot.remainingMs,
        kills: snapshot.team.totalKills,
        enemies: snapshot.monsters.length,
        playerCount: snapshot.players.length,
      })
    }
  }

  private handleProjectileSpawned(payload: ProjectileSpawnedPayload): void {
    const angle = Math.atan2(payload.velocityY, payload.velocityX)
    const sprite = this.add.image(payload.x, payload.y, 'arc-bolt').setRotation(angle).setDepth(payload.y + 2)
    this.projectiles.set(payload.projectileId, { sprite, velocityX: payload.velocityX, velocityY: payload.velocityY })
  }

  private handleMatchEnded(payload: MatchEndedPayload): void {
    this.session.bridge.patch({
      outcome: payload.outcome,
      level: payload.teamLevel,
      kills: payload.totalKills,
      remainingMs: Math.max(0, 5 * 60 * 1000 - payload.survivalMs),
    })
  }

  private updateLocalInput(time: number, seconds: number): void {
    const local = this.players.get(this.session.network.playerId)
    if (!local || !local.alive) return

    const horizontal = Number(this.cursors?.right.isDown || this.keys?.D.isDown) - Number(this.cursors?.left.isDown || this.keys?.A.isDown)
    const vertical = Number(this.cursors?.down.isDown || this.keys?.S.isDown) - Number(this.cursors?.up.isDown || this.keys?.W.isDown)
    const virtual = this.session.bridge.getVirtualMovement()
    const movement = normalizeMovement(horizontal + virtual.x, vertical + virtual.y)
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
      x: Phaser.Math.Clamp(local.sprite.x + movement.x * RANGER.movementSpeed * seconds, RANGER.collisionRadius, 3200 - RANGER.collisionRadius),
      y: Phaser.Math.Clamp(local.sprite.y + movement.y * RANGER.movementSpeed * seconds, RANGER.collisionRadius, 1800 - RANGER.collisionRadius),
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
      view.shadow.setPosition(view.sprite.x, view.sprite.y + 35).setDepth(view.sprite.y - 1)
      view.name.setPosition(view.sprite.x, view.sprite.y - 60).setDepth(view.sprite.y + 1)
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
      view.shadow.setPosition(view.sprite.x, view.sprite.y + 25).setDepth(view.sprite.y - 1)
    }
  }

  private updateProjectiles(seconds: number): void {
    for (const view of this.projectiles.values()) {
      view.sprite.x += view.velocityX * seconds
      view.sprite.y += view.velocityY * seconds
      view.sprite.setDepth(view.sprite.y + 2)
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

  private ensurePlayerView(player: SnapshotPlayer): PlayerView {
    const existing = this.players.get(player.id)
    if (existing) return existing

    const isLocal = player.id === this.session.network.playerId
    const color = playerColor(player.id)
    const shadow = this.add.image(player.x, player.y + 35, 'entity-shadow').setDisplaySize(94, 38)
    const sprite = this.add.image(player.x, player.y, 'ranger')
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
    const view: PlayerView = { sprite, shadow, name, indicator, targetX: player.x, targetY: player.y, alive: player.alive }
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
        view = {
          shadow: this.add.image(monster.x, monster.y + 25, 'entity-shadow').setDisplaySize(66, 26),
          sprite: this.add.image(monster.x, monster.y, 'crawler'),
          targetX: monster.x,
          targetY: monster.y,
          hp: monster.hp,
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

  private syncPickups(pickups: SnapshotPickup[]): void {
    const active = new Set(pickups.map((pickup) => pickup.id))
    for (const pickup of pickups) {
      if (!this.pickups.has(pickup.id)) {
        this.pickups.set(pickup.id, this.add.image(pickup.x, pickup.y, 'experience').setDepth(pickup.y))
      }
    }
    for (const [id, sprite] of this.pickups) {
      if (!active.has(id)) {
        sprite.destroy()
        this.pickups.delete(id)
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
    this.unsubscribeNetwork?.()
    this.unsubscribeConnection?.()
    this.unsubscribeNetwork = null
    this.unsubscribeConnection = null
  }
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
