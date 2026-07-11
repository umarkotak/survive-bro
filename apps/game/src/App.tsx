import { useCallback, useEffect, useState, type FormEvent } from 'react'

import type { GameHudState } from './bridge/GameBridge'
import { gameAudio } from './audio/gameAudio'
import { GameCanvas } from './components/GameCanvas'
import { diagnosticsEnabled } from './config/diagnostics'
import { joinNetworkUrl, networkConfig } from './config/network'
import { MultiplayerSession } from './network/MultiplayerSession'
import type { UpgradeAppliedPayload, UpgradeAttribute } from './network/protocol'

const displayNameKey = 'survive-bro-display-name'
const roomAlphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZ'

interface RoomSummary {
  roomName: string
  status: 'lobby' | 'running' | 'finished'
  playerCount: number
  maxPlayers: number
  joinable: boolean
  levelId: string
}
interface LevelSummary { id: string; name: string; durationSeconds: number }

interface UpgradeHistoryEntry extends UpgradeAppliedPayload { id: number; occurredAt: Date }

export function App() {
  const [displayName, setDisplayName] = useState(() => localStorage.getItem(displayNameKey) ?? '')
  const [nameDraft, setNameDraft] = useState(() => localStorage.getItem(displayNameKey) ?? '')
  const [rooms, setRooms] = useState<RoomSummary[]>([])
  const [levels, setLevels] = useState<LevelSummary[]>([])
  const [loadingRooms, setLoadingRooms] = useState(false)
  const [createCode, setCreateCode] = useState('')
  const [createLevelId, setCreateLevelId] = useState('level-1')
  const [session, setSession] = useState<MultiplayerSession | null>(null)
  const [hud, setHud] = useState<GameHudState | null>(null)
  const [connectingRoom, setConnectingRoom] = useState('')
  const [error, setError] = useState('')
  const [menuOpen, setMenuOpen] = useState(false)
  const [statsOpen, setStatsOpen] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [upgradeHistory, setUpgradeHistory] = useState<UpgradeHistoryEntry[]>([])
  const [upgradeToast, setUpgradeToast] = useState<UpgradeHistoryEntry | null>(null)

  const loadRooms = useCallback(async () => {
    if (!displayName) return
    setLoadingRooms(true)
    setError('')
    try {
      const roomsUrl = joinNetworkUrl(networkConfig.apiBaseUrl, '/api/v1/rooms')
      const levelsUrl = joinNetworkUrl(networkConfig.apiBaseUrl, '/api/v1/levels')
      const [response, levelResponse] = await Promise.all([fetch(roomsUrl), fetch(levelsUrl)])
      if (!response.ok) throw new Error(`Room list returned HTTP ${response.status} ${response.statusText}`)
      if (!levelResponse.ok) throw new Error(`Level list returned HTTP ${levelResponse.status} ${levelResponse.statusText}`)
      const data = await response.json() as { rooms?: RoomSummary[] }
      const levelData = await levelResponse.json() as { levels?: LevelSummary[] }
      if (!Array.isArray(data.rooms)) throw new Error('Room list returned an invalid response')
      if (!Array.isArray(levelData.levels)) throw new Error('Level list returned an invalid response')
      setRooms(data.rooms)
      setLevels(levelData.levels)
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Could not load rooms.')
    } finally {
      setLoadingRooms(false)
    }
  }, [displayName])

  useEffect(() => { void loadRooms() }, [loadRooms])
  useEffect(() => {
    if (!session) return
    setHud(session.bridge.getSnapshot())
    return session.bridge.subscribe(setHud)
  }, [session])
  useEffect(() => {
    if (!session) return
    return session.network.subscribe((message) => {
      if (message.type !== 'upgrade_applied') return
      const upgrade = message.payload as UpgradeAppliedPayload
      if (upgrade.playerId !== session.network.playerId) return
      const entry = { ...upgrade, id: Date.now() + Math.random(), occurredAt: new Date() }
      if (upgrade.source === 'level_up') gameAudio.levelUp()
      else gameAudio.treasure()
      setUpgradeHistory((current) => [entry, ...current].slice(0, 100))
      setUpgradeToast(entry)
    })
  }, [session])
  useEffect(() => {
    if (!upgradeToast) return
    const timeout = window.setTimeout(() => setUpgradeToast(null), 3200)
    return () => window.clearTimeout(timeout)
  }, [upgradeToast])

  function saveName(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const name = nameDraft.trim()
    if (name.length < 1 || name.length > 20) return setError('Username must contain 1–20 characters.')
    localStorage.setItem(displayNameKey, name)
    setDisplayName(name)
    setError('')
  }

  async function joinRoom(roomName: string, levelId?: string) {
    if (!displayName || connectingRoom) return
    setError('')
    setConnectingRoom(roomName)
    const nextSession = new MultiplayerSession()
    try {
      await nextSession.connect(roomName, displayName, levelId)
      setCreateCode('')
      setSession(nextSession)
    } catch (connectionError) {
      nextSession.close()
      setError(connectionError instanceof Error ? connectionError.message : 'Could not enter room.')
    } finally {
      setConnectingRoom('')
    }
  }

  function leaveRoom() {
    session?.close()
    setMenuOpen(false)
    setStatsOpen(false)
    setHistoryOpen(false)
    setUpgradeHistory([])
    setUpgradeToast(null)
    setSession(null)
    setHud(null)
    void loadRooms()
  }

  if (!displayName) {
    return <main className="lobby-shell"><section className="name-card">
      <div className="entry-brand"><span className="brand-mark">SB</span><span>Survive Bro</span></div>
      <div><span className="eyebrow">Player setup</span><h1>Choose your name.</h1><p>Your username is saved on this device and shown to players in every room.</p></div>
      <form className="name-form" onSubmit={saveName}>
        <label><span>Username</span><input value={nameDraft} onChange={(event) => setNameDraft(event.target.value)} maxLength={20} autoFocus autoComplete="nickname" placeholder="Umar" /></label>
        {error && <p className="form-error" role="alert">{error}</p>}
        <button type="submit">Continue</button>
      </form>
    </section></main>
  }

  if (!session || !hud) {
    return <main className="lobby-shell"><section className="room-browser">
      <header className="room-header">
        <div className="entry-brand"><span className="brand-mark">SB</span><span>Survive Bro</span></div>
        <div className="player-chip"><span>Playing as</span><strong>{displayName}</strong><button type="button" onClick={() => { localStorage.removeItem(displayNameKey); setDisplayName(''); setNameDraft(displayName) }}>Change</button></div>
      </header>
      <div className="room-title"><div><span className="eyebrow">Live rooms</span><h1>Pick a meadow.</h1><p>Join any available squad or create a new five-letter room.</p></div><button className="create-room-button" type="button" onClick={() => setCreateCode(generateRoomCode())}>+ Create room</button></div>
      {error && <p className="room-error" role="alert">{error}</p>}
      <div className="room-list">
        <div className="room-list-heading"><strong>Available rooms</strong><button type="button" onClick={() => void loadRooms()} disabled={loadingRooms}>{loadingRooms ? 'Refreshing…' : 'Refresh'}</button></div>
        {rooms.length === 0 && !loadingRooms ? <div className="empty-rooms"><strong>No active rooms yet.</strong><span>Create one and invite your bros.</span></div> : rooms.map((room) => (
          <article className="room-row" key={room.roomName}>
            <div className="room-code"><span>{room.status} · {room.levelId === 'level-1' ? 'Slime Meadow' : room.levelId}</span><strong>{room.roomName}</strong></div>
            <div className="room-capacity"><span>Players</span><strong>{room.playerCount}/{room.maxPlayers}</strong></div>
            <button type="button" disabled={!room.joinable || Boolean(connectingRoom)} onClick={() => void joinRoom(room.roomName)}>{connectingRoom === room.roomName ? 'Joining…' : room.joinable ? 'Join room' : 'Full'}</button>
          </article>
        ))}
      </div>
      {createCode && <div className="menu-backdrop" onClick={() => setCreateCode('')}><section className="create-modal" role="dialog" aria-modal="true" aria-labelledby="create-title" onClick={(event) => event.stopPropagation()}>
        <button type="button" className="menu-close" aria-label="Close" onClick={() => setCreateCode('')}>×</button><span className="eyebrow">New room</span><h2 id="create-title">Your room is ready.</h2><p>Choose a level, then share the five-letter ID with your squad.</p><label><span>Room ID</span><input readOnly value={createCode} /></label><label><span>Level</span><select value={createLevelId} onChange={(event) => setCreateLevelId(event.target.value)}>{(levels.length ? levels : [{ id: 'level-1', name: 'Slime Meadow', durationSeconds: 360 }]).map((level) => <option key={level.id} value={level.id}>{level.name} · {Math.round(level.durationSeconds / 60)} min</option>)}</select></label><button className="start-room-button" type="button" disabled={Boolean(connectingRoom)} onClick={() => void joinRoom(createCode, createLevelId)}>{connectingRoom ? 'Starting…' : 'Start game'}</button>
      </section></div>}
    </section></main>
  }

  const healthPercent = Math.max(0, (hud.hp / hud.maxHp) * 100)
  const experiencePercent = Math.min(100, (hud.experience / hud.experienceRequired) * 100)
  return <main className="app-shell"><section className="game-frame">
    <GameCanvas session={session} />
    <div className="game-hud"><div className="experience-strip"><i style={{ width: `${experiencePercent}%` }} /></div>
      <aside className="health-panel">
        <button className="player-portrait-button" type="button" aria-label="Open your character and spell statistics" onClick={() => setStatsOpen(true)}><img src="/assets/character-ranger-idle.png" alt="" /><span>YOU</span></button>
        <div className="health-content"><div className="health-heading"><strong>{hud.displayName || 'Ranger'}</strong><b>LV {hud.level}</b></div><div className="health-value"><span>HP</span><b>{hud.hp}/{hud.maxHp}</b></div><div className="health-meter"><i style={{ width: `${healthPercent}%` }} /></div></div>
      </aside>
      <button className="menu-toggle" type="button" onClick={() => setMenuOpen(true)}><span className="menu-icon"><i /><i /><i /></span><span>Menu</span></button>
      {diagnosticsEnabled && <aside className="diagnostics-panel"><strong>Diagnostics</strong><dl><div><dt>FPS</dt><dd>{hud.diagnostics.fps.toFixed(0)}</dd></div><div><dt>Sprites</dt><dd>{hud.diagnostics.visibleSprites}/{hud.diagnostics.activeSprites}</dd></div><div><dt>RTT</dt><dd>{formatMetric(hud.diagnostics.roundTripMs, 'ms')}</dd></div></dl></aside>}
      {upgradeToast && <div className="upgrade-toast" role="status"><span>{upgradeToast.source === 'level_up' ? 'LEVEL UP' : 'TREASURE CHEST'}</span><strong>{upgradeLabel(upgradeToast.attribute)} upgraded</strong><small>+{formatUpgradeValue(upgradeToast.attribute, upgradeToast.addedValue)} · now {formatUpgradeValue(upgradeToast.attribute, upgradeToast.finalValue)}</small></div>}
    </div>
    {statsOpen && <div className="menu-backdrop" onClick={() => setStatsOpen(false)}><section className="stats-modal" role="dialog" aria-modal="true" aria-labelledby="stats-title" onClick={(event) => event.stopPropagation()}><div className="modal-actions"><button type="button" className="history-button" onClick={() => { setStatsOpen(false); setHistoryOpen(true) }}>History</button><button type="button" className="menu-close" aria-label="Close statistics" onClick={() => setStatsOpen(false)}>×</button></div><header className="stats-identity"><img src="/assets/character-ranger-idle.png" alt="Ranger" /><div><span>YOUR CHARACTER</span><h2 id="stats-title">{hud.displayName}</h2><p>Ranger · Team level {hud.level}</p></div></header><div className="stats-columns"><section><h3>Character stats</h3><dl className="attribute-list"><div><dt>Current health</dt><dd>{hud.hp}</dd></div><div><dt>Max health</dt><dd>{statLine(100, hud.maxHp, integer)}</dd></div><div><dt>Armor</dt><dd>{statLine(0, hud.armorPercent, percent)}</dd></div><div><dt>Movement speed</dt><dd>{statLine(220, hud.movementSpeed, integer)}</dd></div><div><dt>Regeneration</dt><dd>{statLine(0, hud.healthRegeneration, integer)}</dd></div><div><dt>Attack buff</dt><dd>{statLine(0, hud.attackBuffPercent, percent)}</dd></div><div><dt>Cooldown reduction</dt><dd>{statLine(0, hud.cooldownPercent, percent)}</dd></div></dl></section><section><h3>Fireball stats</h3><dl className="attribute-list"><div><dt>Damage</dt><dd>{statLine(20, hud.spellDamage, integer)}</dd></div><div><dt>Projectile speed</dt><dd>{statLine(700, hud.projectileSpeed, integer)}</dd></div><div><dt>Burst</dt><dd>{statLine(1, hud.spellBurst, integer)}</dd></div><div><dt>Directions</dt><dd>{statLine(1, hud.spellDirections, integer)}</dd></div><div><dt>Final damage</dt><dd>{Math.round(hud.spellDamage * (1 + hud.attackBuffPercent))}</dd></div><div><dt>Current cooldown</dt><dd>{Math.round(750 * (1 - hud.cooldownPercent))} ms</dd></div><div><dt>Volley size</dt><dd>{hud.spellBurst * hud.spellDirections}</dd></div></dl></section></div></section></div>}
    {historyOpen && <div className="menu-backdrop" onClick={() => setHistoryOpen(false)}><section className="stats-modal history-modal" role="dialog" aria-modal="true" aria-labelledby="history-title" onClick={(event) => event.stopPropagation()}><div className="modal-actions"><button type="button" className="history-button" onClick={() => { setHistoryOpen(false); setStatsOpen(true) }}>Stats</button><button type="button" className="menu-close" aria-label="Close history" onClick={() => setHistoryOpen(false)}>×</button></div><header className="history-heading"><span>THIS RUN</span><h2 id="history-title">Upgrade history</h2><p>Level-up and treasure upgrades received by {hud.displayName}.</p></header><div className="history-list">{upgradeHistory.length === 0 ? <div className="history-empty">No upgrades received yet.</div> : upgradeHistory.map((entry) => <article key={entry.id}><i className={entry.source} aria-hidden="true">{entry.source === 'level_up' ? 'LV' : '▣'}</i><div><span>{entry.source === 'level_up' ? 'Level up' : 'Treasure chest'} · {entry.occurredAt.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span><strong>{upgradeLabel(entry.attribute)}</strong><small>+{formatUpgradeValue(entry.attribute, entry.addedValue)} · {formatUpgradeValue(entry.attribute, entry.finalValue)} total</small></div></article>)}</div></section></div>}
    {menuOpen && <div className="menu-backdrop" onClick={() => setMenuOpen(false)}><section className="game-menu" role="dialog" aria-modal="true" onClick={(event) => event.stopPropagation()}><button type="button" className="menu-close" aria-label="Close menu" onClick={() => setMenuOpen(false)}>×</button><span className="brand-mark">SB</span><h2>Game menu</h2><p>Room {hud.roomName} · {hud.playerCount}/6 players</p><button className="leave-button" type="button" onClick={leaveRoom}>Leave room</button></section></div>}
    {hud.outcome !== 'playing' && <div className="result-backdrop"><section className="result-card"><span className="eyebrow">Final score</span><h1>{hud.score.toLocaleString()}</h1><p>{hud.kills} slimes defeated · team level {hud.level}</p><button type="button" onClick={leaveRoom}>Back to rooms</button></section></div>}
  </section></main>
}

function generateRoomCode(): string { return Array.from({ length: 5 }, () => roomAlphabet[Math.floor(Math.random() * roomAlphabet.length)]).join('') }
function percent(value: number): string { return `${Math.round(value * 100)}%` }
function integer(value: number): string { return Math.round(value).toString() }
function statLine(base: number, final: number, formatter: (value: number) => string): string { return `${formatter(base)} (+${formatter(Math.max(0, final - base))}) ${formatter(final)}` }
function upgradeLabel(attribute: UpgradeAttribute): string {
  return ({ max_health: 'Max health', armor: 'Armor', movement_speed: 'Movement speed', health_regeneration: 'Health regeneration', attack_buff: 'Attack buff', cooldown: 'Cooldown reduction', spell_damage: 'Fireball damage', projectile_speed: 'Projectile speed', spell_burst: 'Fireball burst', spell_directions: 'Fireball directions' })[attribute]
}
function formatUpgradeValue(attribute: UpgradeAttribute, value: number): string {
  return attribute === 'armor' || attribute === 'attack_buff' || attribute === 'cooldown' ? percent(value) : integer(value)
}
function formatMetric(value: number, suffix: string): string { return value > 0 ? `${value.toFixed(0)} ${suffix}` : '—' }
