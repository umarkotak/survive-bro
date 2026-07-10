import Phaser from 'phaser'

import type { MultiplayerSession } from '../network/MultiplayerSession'
import { BootScene } from './scenes/BootScene'
import { GameScene } from './scenes/GameScene'

export function createGame(parent: HTMLElement, session: MultiplayerSession): Phaser.Game {
  return new Phaser.Game({
    type: Phaser.AUTO,
    parent,
    width: 1280,
    height: 720,
    backgroundColor: '#15271f',
    antialias: true,
    render: {
      roundPixels: false,
    },
    scale: {
      mode: Phaser.Scale.RESIZE,
      autoCenter: Phaser.Scale.CENTER_BOTH,
    },
    scene: [new BootScene(), new GameScene(session)],
  })
}
