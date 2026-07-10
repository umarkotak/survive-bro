import { GameBridge } from '../bridge/GameBridge'
import { NetworkClient } from './NetworkClient'

export class MultiplayerSession {
  readonly network = new NetworkClient()
  readonly bridge = new GameBridge()
  displayName = ''

  async connect(roomName: string, displayName: string): Promise<void> {
    this.displayName = displayName
    const joined = await this.network.connect(roomName, displayName)
    this.bridge.patch({
      connection: 'connected',
      roomName: joined.roomName,
      playerId: joined.playerId,
      displayName,
    })
  }

  close(): void {
    this.network.close()
    this.bridge.patch({ connection: 'disconnected' })
  }
}
