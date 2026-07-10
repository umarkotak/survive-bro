import { useEffect, useRef, useState, type PointerEvent } from 'react'

import type { GameBridge } from '../bridge/GameBridge'
import { joystickState } from '../input/joystick'

interface VirtualJoystickProps {
  bridge: GameBridge
}

const knobRadius = 28

export function VirtualJoystick({ bridge }: VirtualJoystickProps) {
  const activePointer = useRef<number | null>(null)
  const [knob, setKnob] = useState({ x: 0, y: 0 })

  useEffect(() => () => bridge.setVirtualMovement(0, 0), [bridge])

  function update(event: PointerEvent<HTMLDivElement>) {
    const bounds = event.currentTarget.getBoundingClientRect()
    const radius = Math.max(1, bounds.width / 2 - knobRadius)
    const state = joystickState(
      event.clientX,
      event.clientY,
      bounds.left + bounds.width / 2,
      bounds.top + bounds.height / 2,
      radius,
    )
    bridge.setVirtualMovement(state.movement.x, state.movement.y)
    setKnob(state.knob)
  }

  function start(event: PointerEvent<HTMLDivElement>) {
    if (activePointer.current !== null || !event.isPrimary) return
    event.preventDefault()
    activePointer.current = event.pointerId
    event.currentTarget.setPointerCapture(event.pointerId)
    update(event)
  }

  function move(event: PointerEvent<HTMLDivElement>) {
    if (activePointer.current !== event.pointerId) return
    event.preventDefault()
    update(event)
  }

  function stop(event: PointerEvent<HTMLDivElement>) {
    if (activePointer.current !== event.pointerId) return
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    activePointer.current = null
    bridge.setVirtualMovement(0, 0)
    setKnob({ x: 0, y: 0 })
  }

  return (
    <div
      className="virtual-joystick"
      role="group"
      aria-label="Movement joystick"
      onPointerDown={start}
      onPointerMove={move}
      onPointerUp={stop}
      onPointerCancel={stop}
      onLostPointerCapture={stop}
    >
      <span className="virtual-joystick-ring" />
      <span
        className="virtual-joystick-knob"
        style={{ transform: `translate3d(${knob.x}px, ${knob.y}px, 0)` }}
      />
      <span className="virtual-joystick-label">Move</span>
    </div>
  )
}
