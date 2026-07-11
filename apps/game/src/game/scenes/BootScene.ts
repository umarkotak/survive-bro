import Phaser from 'phaser'

export class BootScene extends Phaser.Scene {
  constructor() {
    super('BootScene')
  }

  preload(): void {
    for (const asset of [
      'character-ranger-attack-1',
      'character-ranger-idle',
      'character-ranger-walk-1',
      'character-ranger-walk-2',
      'character-ranger-walk-3',
      'character-frieren-attack-1',
      'character-frieren-idle',
      'character-frieren-walk-1',
      'character-frieren-walk-2',
      'character-frieren-walk-3',
      'character-catapult-attack-1',
      'character-catapult-idle',
      'character-catapult-walk-1',
      'character-catapult-walk-2',
      'character-catapult-walk-3',
      'obstacle-large-rock-1',
      'obstacle-large-rock-2',
      'obstacle-large-rock-3',
      'terrain-variant-1',
      'terrain-variant-2',
      'terrain-variant-3',
      'enemy-slime-stage-1',
      'enemy-slime-stage-2',
      'enemy-slime-stage-3',
    ]) {
      this.load.image(asset, `/assets/${asset}.png`)
    }
  }

  create(): void {
    this.createShadowTexture()
    this.createBoltTexture()
    this.createRocketTextures()
    this.createExperienceTexture()
    this.createPowerCrateTexture()
    this.scene.start('GameScene')
  }

  private createShadowTexture(): void {
    const graphics = this.make.graphics({ x: 0, y: 0 })
    graphics.fillStyle(0x10291f, 0.25)
    graphics.fillEllipse(64, 26, 112, 32)
    graphics.generateTexture('entity-shadow', 128, 52)
    graphics.destroy()
  }

  private createBoltTexture(): void {
    const graphics = this.make.graphics({ x: 0, y: 0 })
    graphics.fillStyle(0xb8f3ff, 0.25)
    graphics.fillEllipse(22, 12, 42, 22)
    graphics.fillStyle(0x74ddff, 1)
    graphics.fillTriangle(4, 12, 31, 3, 46, 12)
    graphics.fillTriangle(4, 12, 31, 21, 46, 12)
    graphics.fillStyle(0xffffff, 1)
    graphics.fillEllipse(27, 12, 21, 7)
    graphics.generateTexture('arc-bolt', 48, 24)
    graphics.destroy()
  }

  private createRocketTextures(): void {
    const rocket = this.make.graphics({ x: 0, y: 0 })
    rocket.fillStyle(0xffd36a, 1).fillRoundedRect(8, 5, 30, 12, 3)
    rocket.fillStyle(0xff6b3d, 1).fillTriangle(8, 5, 0, 11, 8, 17)
    rocket.fillStyle(0xf7f2da, 1).fillTriangle(38, 5, 46, 11, 38, 17)
    rocket.generateTexture('rocket', 46, 22)
    rocket.destroy()
  }

  private createExperienceTexture(): void {
    const graphics = this.make.graphics({ x: 0, y: 0 })
    graphics.fillStyle(0x86fff1, 0.25)
    graphics.fillCircle(14, 14, 14)
    graphics.fillStyle(0x45e6ce, 1)
    graphics.fillTriangle(14, 2, 25, 12, 14, 26)
    graphics.fillTriangle(14, 2, 3, 12, 14, 26)
    graphics.fillStyle(0xd9fff9, 1)
    graphics.fillTriangle(14, 4, 14, 14, 7, 11)
    graphics.generateTexture('experience', 28, 28)
    graphics.destroy()
  }

  private createPowerCrateTexture(): void {
    const graphics = this.make.graphics({ x: 0, y: 0 })
    graphics.fillStyle(0x18362c, 0.3)
    graphics.fillEllipse(24, 43, 42, 12)
    graphics.fillStyle(0xf0b84f, 1)
    graphics.fillRoundedRect(4, 8, 40, 34, 7)
    graphics.fillStyle(0x8b542d, 1)
    graphics.fillRect(4, 17, 40, 7)
    graphics.fillRect(18, 8, 8, 34)
    graphics.fillStyle(0xfff2a5, 1)
    graphics.fillCircle(22, 22, 5)
    graphics.generateTexture('power-crate', 48, 48)
    graphics.destroy()
  }
}
