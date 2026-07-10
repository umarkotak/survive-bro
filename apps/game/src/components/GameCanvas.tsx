import { memo, useEffect, useRef } from 'react'

import { createGame } from '../game/createGame'
import type { MultiplayerSession } from '../network/MultiplayerSession'

interface GameCanvasProps {
  session: MultiplayerSession
}

export const GameCanvas = memo(function GameCanvas({ session }: GameCanvasProps) {
  const parentRef = useRef<HTMLDivElement>(null)
  const sessionRef = useRef(session)

  useEffect(() => {
    if (!parentRef.current) return
    const game = createGame(parentRef.current, sessionRef.current)
    return () => game.destroy(true)
  }, [])

  return <div ref={parentRef} className="game-canvas" aria-label="Survive Bro game canvas" />
})
