import Phaser from 'phaser'

export class BootScene extends Phaser.Scene {
  constructor() {
    super('BootScene')
  }

  create(): void {
    this.createGroundTexture()
    this.createShadowTexture()
    this.createRangerTexture()
    this.createCrawlerTexture()
    this.createRockTexture()
    this.createBoltTexture()
    this.createExperienceTexture()
    this.scene.start('GameScene')
  }

  private createGroundTexture(): void {
    const graphics = this.make.graphics({ x: 0, y: 0 })
    graphics.fillStyle(0x6e9f68, 1)
    graphics.fillRect(0, 0, 256, 256)
    graphics.fillStyle(0x78aa71, 0.65)
    for (const [x, y, radius] of [
      [28, 34, 18],
      [180, 42, 24],
      [116, 154, 28],
      [232, 208, 20],
      [45, 224, 16],
    ] as const) {
      graphics.fillCircle(x, y, radius)
    }
    graphics.lineStyle(2, 0x527d50, 0.3)
    for (let index = 0; index < 18; index += 1) {
      const x = (index * 47) % 256
      const y = (index * 83) % 256
      graphics.lineBetween(x, y, x + 5, y - 10)
      graphics.lineBetween(x, y, x - 4, y - 7)
    }
    graphics.generateTexture('meadow-ground', 256, 256)
    graphics.destroy()
  }

  private createShadowTexture(): void {
    const graphics = this.make.graphics({ x: 0, y: 0 })
    graphics.fillStyle(0x10291f, 0.25)
    graphics.fillEllipse(64, 26, 112, 32)
    graphics.generateTexture('entity-shadow', 128, 52)
    graphics.destroy()
  }

  private createRangerTexture(): void {
    const graphics = this.make.graphics({ x: 0, y: 0 })
    graphics.fillStyle(0x203a35, 1)
    graphics.fillEllipse(46, 75, 44, 20)
    graphics.fillStyle(0x2f6f57, 1)
    graphics.fillRoundedRect(26, 36, 42, 46, 15)
    graphics.fillStyle(0x75c98b, 1)
    graphics.fillTriangle(23, 72, 49, 29, 70, 74)
    graphics.fillStyle(0xf1c7a4, 1)
    graphics.fillCircle(49, 29, 14)
    graphics.fillStyle(0x5a3424, 1)
    graphics.fillRoundedRect(37, 14, 25, 11, 5)
    graphics.fillStyle(0xf5d46a, 1)
    graphics.fillTriangle(60, 41, 88, 48, 61, 54)
    graphics.lineStyle(3, 0x754728, 1)
    graphics.strokeCircle(72, 48, 19)
    graphics.lineBetween(72, 29, 72, 67)
    graphics.generateTexture('ranger', 96, 96)
    graphics.destroy()
  }

  private createCrawlerTexture(): void {
    const graphics = this.make.graphics({ x: 0, y: 0 })
    graphics.fillStyle(0x2c3e32, 1)
    graphics.fillEllipse(32, 51, 48, 22)
    graphics.fillStyle(0x6a3f62, 1)
    graphics.fillEllipse(34, 36, 54, 38)
    graphics.fillStyle(0x8f5a7f, 1)
    graphics.fillCircle(50, 34, 18)
    graphics.fillStyle(0xf1e6b0, 1)
    graphics.fillCircle(55, 29, 5)
    graphics.fillStyle(0x251c2a, 1)
    graphics.fillCircle(57, 29, 2)
    graphics.fillTriangle(66, 35, 72, 39, 65, 42)
    graphics.lineStyle(4, 0x3d2b3b, 1)
    graphics.lineBetween(18, 50, 10, 62)
    graphics.lineBetween(33, 52, 29, 68)
    graphics.lineBetween(48, 50, 53, 66)
    graphics.generateTexture('crawler', 72, 72)
    graphics.destroy()
  }

  private createRockTexture(): void {
    const graphics = this.make.graphics({ x: 0, y: 0 })
    graphics.fillStyle(0x21352f, 0.2)
    graphics.fillEllipse(110, 145, 170, 34)
    graphics.fillStyle(0x53625c, 1)
    graphics.fillRoundedRect(30, 45, 160, 105, 42)
    graphics.fillStyle(0x718078, 1)
    graphics.fillRoundedRect(47, 32, 125, 83, 35)
    graphics.fillStyle(0x91a097, 0.8)
    graphics.fillEllipse(92, 55, 67, 25)
    graphics.fillStyle(0x496b54, 0.7)
    graphics.fillCircle(65, 105, 18)
    graphics.fillCircle(145, 122, 14)
    graphics.generateTexture('large-rock', 220, 180)
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
}
