import { useEffect, useState, type FormEvent } from 'react'

import type { GameHudState } from './bridge/GameBridge'
import { GameCanvas } from './components/GameCanvas'
import { formatRemainingTime } from './game/model'
import { MultiplayerSession } from './network/MultiplayerSession'

export function App() {
  const [displayName, setDisplayName] = useState(() => localStorage.getItem('survive-bro-display-name') ?? '')
  const [roomName, setRoomName] = useState('')
  const [session, setSession] = useState<MultiplayerSession | null>(null)
  const [hud, setHud] = useState<GameHudState | null>(null)
  const [connecting, setConnecting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!session) return
    setHud(session.bridge.getSnapshot())
    return session.bridge.subscribe(setHud)
  }, [session])

  async function enterRoom(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    const name = displayName.trim()
    const room = roomName.trim().toUpperCase()
    if (name.length < 1 || name.length > 20) {
      setError('Display name must contain 1–20 characters.')
      return
    }
    if (!/^[A-Z0-9_-]{1,24}$/.test(room)) {
      setError('Room name uses 1–24 letters, numbers, - or _.')
      return
    }

    setConnecting(true)
    const nextSession = new MultiplayerSession()
    try {
      await nextSession.connect(room, name)
      localStorage.setItem('survive-bro-display-name', name)
      setRoomName(room)
      setSession(nextSession)
    } catch (connectionError) {
      nextSession.close()
      setError(connectionError instanceof Error ? connectionError.message : 'Could not enter room.')
    } finally {
      setConnecting(false)
    }
  }

  function leaveRoom() {
    session?.close()
    setSession(null)
    setHud(null)
  }

  if (!session || !hud) {
    return (
      <main className="entry-shell">
        <section className="entry-card">
          <div className="entry-brand"><span className="brand-mark">SB</span><span>Survive Bro</span></div>
          <div className="entry-copy">
            <span className="eyebrow">Shared meadow · 1–4 players</span>
            <h1>Bring a room.<br />Bring your bro.</h1>
            <p>Enter any room name. If it does not exist, we create it. If it does, you join the same live battlefield.</p>
          </div>

          <form className="entry-form" onSubmit={enterRoom}>
            <label>
              <span>Your name</span>
              <input
                value={displayName}
                onChange={(event) => setDisplayName(event.target.value)}
                maxLength={20}
                autoComplete="nickname"
                placeholder="Umar"
                autoFocus
              />
            </label>
            <label>
              <span>Room name</span>
              <input
                value={roomName}
                onChange={(event) => setRoomName(event.target.value.toUpperCase().replace(/[^A-Z0-9_-]/g, ''))}
                maxLength={24}
                autoComplete="off"
                placeholder="FRIDAY-SQUAD"
              />
            </label>
            {error && <p className="form-error" role="alert">{error}</p>}
            <button type="submit" disabled={connecting}>
              {connecting ? 'Opening the meadow…' : 'Create or join room'}
            </button>
          </form>

          <footer className="entry-footnote">WASD / Arrow keys · Arc Bolt fires automatically</footer>
        </section>
      </main>
    )
  }

  const healthPercent = Math.max(0, (hud.hp / hud.maxHp) * 100)
  const experiencePercent = Math.min(100, (hud.experience / hud.experienceRequired) * 100)

  return (
    <main className="app-shell">
      <section className="game-frame">
        <GameCanvas session={session} />

        <header className="game-header">
          <div className="brand">
            <span className="brand-mark">SB</span>
            <div>
              <strong>Survive Bro</strong>
              <small>{hud.roomName} · <span className={`connection ${hud.connection}`}>{hud.connection}</span></small>
            </div>
          </div>

          <div className="timer" aria-label="Time remaining">
            <span>Meadow</span>
            <strong>{formatRemainingTime(hud.remainingMs)}</strong>
          </div>

          <div className="match-stats">
            <span><b>{hud.playerCount}</b> players</span>
            <span><b>{hud.kills}</b> defeated</span>
            <span><b>{hud.enemies}</b> nearby</span>
            <button className="leave-button" type="button" onClick={leaveRoom}>Leave</button>
          </div>
        </header>

        <aside className="status-panel" aria-label="Ranger status">
          <div className="character-chip">
            <span className="character-avatar">R</span>
            <div>
              <strong>{hud.displayName || 'Ranger'}</strong>
              <small>Arc Bolt · Team level {hud.level}</small>
            </div>
          </div>

          <div className="meter-row">
            <span>HP</span>
            <div className="meter health-meter"><i style={{ width: `${healthPercent}%` }} /></div>
            <b>{hud.hp}/{hud.maxHp}</b>
          </div>
          <div className="meter-row">
            <span>XP</span>
            <div className="meter experience-meter"><i style={{ width: `${experiencePercent}%` }} /></div>
            <b>{hud.experience}/{hud.experienceRequired}</b>
          </div>
        </aside>

        <aside className="controls-panel">
          <span><kbd>WASD</kbd> or <kbd>Arrows</kbd></span>
          <small>Move · edge markers point to teammates</small>
        </aside>

        {hud.outcome !== 'playing' && (
          <div className="result-backdrop" role="dialog" aria-modal="true" aria-labelledby="result-title">
            <section className="result-card">
              <span className={`result-badge ${hud.outcome}`}>{hud.outcome === 'won' ? 'Dawn reached' : 'Run ended'}</span>
              <h1 id="result-title">{hud.outcome === 'won' ? 'Your squad survived.' : 'The meadow took the squad.'}</h1>
              <p>{hud.kills} crawlers defeated · team level {hud.level}</p>
              <button type="button" onClick={leaveRoom}>Back to rooms</button>
            </section>
          </div>
        )}
      </section>
    </main>
  )
}
