import { useCallback, useEffect, useState, type FormEvent } from 'react'

import type { GameHudState } from './bridge/GameBridge'
import { gameAudio } from './audio/gameAudio'
import { GameCanvas } from './components/GameCanvas'
import { MainMenu } from './components/MainMenu'
import { characterAssetPath, enemyAssetPath } from './config/assets'
import { diagnosticsEnabled } from './config/diagnostics'
import { joinNetworkUrl, networkConfig } from './config/network'
import { MultiplayerSession } from './network/MultiplayerSession'
import type { SnapshotSpell, UpgradeAppliedPayload, UpgradeAttribute, UpgradeOfferedPayload } from './network/protocol'
import type { SystemEventPayload } from './network/protocol'

const displayNameKey = 'survive-bro-display-name'
const roomAlphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZ'
const roomPollIntervalMs = 2000

interface RoomSummary {
  roomName: string
  status: 'lobby' | 'running' | 'finished'
  playerCount: number
  maxPlayers: number
  joinable: boolean
  levelId: string
}
interface LevelSummary { id: string; name: string; durationSeconds: number }
interface CharacterSummary { id: string; name: string; spriteId: string; maxHp: number; armorPercent: number; movementSpeed: number; healthRegeneration: number; attackBuffPercent: number; cooldownPercent: number; defaultSpellId: string; startingSpellIds: string[]; baseSpell: { id: string; kind: string; damage: number; impactDamage: number; cooldownMs: number; projectileSpeed: number; burst: number; directions: number; beamLength: number; beamWidth: number; durationMs: number; damageIntervalMs: number; explosionRadius: number; explosionDurationMs: number } }

interface UpgradeHistoryEntry extends UpgradeAppliedPayload { id: number; occurredAt: Date }

export function App() {
  const [pathname, setPathname] = useState(() => normalizePath(window.location.pathname))
  const [displayName, setDisplayName] = useState(() => localStorage.getItem(displayNameKey) ?? '')
  const [nameDraft, setNameDraft] = useState(() => localStorage.getItem(displayNameKey) ?? '')
  const [rooms, setRooms] = useState<RoomSummary[]>([])
  const [levels, setLevels] = useState<LevelSummary[]>([])
  const [characters, setCharacters] = useState<CharacterSummary[]>([])
  const [lobbyOpen, setLobbyOpen] = useState(() => normalizePath(window.location.pathname) === '/lobby')
  const [createCode, setCreateCode] = useState('')
  const [createLevelId, setCreateLevelId] = useState('level-1')
  const [pendingJoin, setPendingJoin] = useState<{ roomName: string; levelId?: string } | null>(null)
  const [session, setSession] = useState<MultiplayerSession | null>(null)
  const [hud, setHud] = useState<GameHudState | null>(null)
  const [connectingRoom, setConnectingRoom] = useState('')
  const [error, setError] = useState('')
  const [menuOpen, setMenuOpen] = useState(false)
  const [statsOpen, setStatsOpen] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [upgradeHistory, setUpgradeHistory] = useState<UpgradeHistoryEntry[]>([])
  const [upgradeToast, setUpgradeToast] = useState<UpgradeHistoryEntry | null>(null)
  const [upgradeOffer, setUpgradeOffer] = useState<UpgradeOfferedPayload | null>(null)
  const [upgradeSeconds, setUpgradeSeconds] = useState(0)
  const [selectedEvent, setSelectedEvent] = useState<SystemEventPayload | null>(null)
  const [selectedSpell, setSelectedSpell] = useState<SnapshotSpell | null>(null)
  const lobbyActive = lobbyOpen && Boolean(displayName)

  useEffect(() => {
    const handleNavigation = () => setPathname(normalizePath(window.location.pathname))
    window.addEventListener('popstate', handleNavigation)
    return () => window.removeEventListener('popstate', handleNavigation)
  }, [])

  useEffect(() => {
    if (pathname !== '/lobby') return
    window.history.replaceState({}, '', '/')
    setPathname('/')
    setLobbyOpen(Boolean(displayName))
  }, [displayName, pathname])

  function navigateTo(path: string) {
    if (window.location.pathname === path) return
    window.history.pushState({}, '', path)
    setPathname(path)
  }

  const loadRooms = useCallback(async (signal?: AbortSignal) => {
    if (!displayName || !lobbyActive) return
    try {
      const roomsUrl = joinNetworkUrl(networkConfig.apiBaseUrl, '/api/v1/rooms')
      const response = await fetch(roomsUrl, { signal })
      if (!response.ok) throw new Error(`Room list returned HTTP ${response.status} ${response.statusText}`)
      const data = await response.json() as { rooms?: RoomSummary[] }
      if (!Array.isArray(data.rooms)) throw new Error('Room list returned an invalid response')
      setRooms(data.rooms)
      setError('')
    } catch (loadError) {
      if (signal?.aborted) return
      setError(loadError instanceof Error ? loadError.message : 'Could not load rooms.')
    }
  }, [displayName, lobbyActive])

  useEffect(() => {
    if (!displayName || !lobbyActive) return
    let cancelled = false
    async function loadSelectionData() {
      try {
        const levelsUrl = joinNetworkUrl(networkConfig.apiBaseUrl, '/api/v1/levels')
        const charactersUrl = joinNetworkUrl(networkConfig.apiBaseUrl, '/api/v1/characters')
        const levelResponse = await fetch(levelsUrl)
        if (!levelResponse.ok) throw new Error(`Level list returned HTTP ${levelResponse.status} ${levelResponse.statusText}`)
        const levelData = await levelResponse.json() as { levels?: LevelSummary[] }
        if (!Array.isArray(levelData.levels)) throw new Error('Level list returned an invalid response')
        if (!cancelled) setLevels(levelData.levels)

        const characterResponse = await fetch(charactersUrl)
        if (!characterResponse.ok) throw new Error(`Character list returned HTTP ${characterResponse.status} ${characterResponse.statusText}`)
        const characterData = await characterResponse.json() as { characters?: CharacterSummary[] }
        if (!Array.isArray(characterData.characters)) throw new Error('Character list returned an invalid response')
        if (!cancelled) setCharacters(characterData.characters)
      } catch (loadError) {
        if (!cancelled) setError(loadError instanceof Error ? loadError.message : 'Could not load game data.')
      }
    }
    void loadSelectionData()
    return () => { cancelled = true }
  }, [displayName, lobbyActive])

  useEffect(() => {
    if (!displayName || !lobbyActive || session) return
    let cancelled = false
    let timeout: number | undefined
    let controller: AbortController | undefined
    async function pollRooms() {
      controller = new AbortController()
      await loadRooms(controller.signal)
      if (cancelled) return
      timeout = window.setTimeout(() => void pollRooms(), roomPollIntervalMs)
    }
    void pollRooms()
    return () => {
      cancelled = true
      controller?.abort()
      if (timeout !== undefined) window.clearTimeout(timeout)
    }
  }, [displayName, loadRooms, lobbyActive, session])
  useEffect(() => {
    if (!session) return
    setHud(session.bridge.getSnapshot())
    return session.bridge.subscribe(setHud)
  }, [session])
  useEffect(() => {
    if (lobbyActive || !session) return
    session.close()
    setSession(null)
    setHud(null)
  }, [lobbyActive, session])
  useEffect(() => {
    if (!session) return
    return session.network.subscribe((message) => {
      if (message.type === 'upgrade_offered') {
        setUpgradeOffer(message.payload as UpgradeOfferedPayload)
        session.bridge.patch({ upgradePaused: true })
        return
      }
      if (message.type !== 'upgrade_applied') return
      const upgrade = message.payload as UpgradeAppliedPayload
      if (upgrade.playerId !== session.network.playerId) return
      setUpgradeOffer(null)
      session.bridge.patch({ upgradePaused: false })
      const entry = { ...upgrade, id: Date.now() + Math.random(), occurredAt: new Date() }
      const bridgePatch: Partial<GameHudState> = {}
      if (upgrade.attribute === 'beam_length') bridgePatch.beamLength = upgrade.finalValue
      if (upgrade.attribute === 'beam_width') bridgePatch.beamWidth = upgrade.finalValue
      if (upgrade.attribute === 'spell_duration') bridgePatch.spellDurationMs = upgrade.finalValue
      if (upgrade.attribute === 'explosion_radius') bridgePatch.explosionRadius = upgrade.finalValue
      if (upgrade.attribute === 'explosion_duration') bridgePatch.explosionDurationMs = upgrade.finalValue
      session.bridge.patch(bridgePatch)
      setUpgradeHistory((current) => [entry, ...current].slice(0, 100))
      setUpgradeToast(entry)
    })
  }, [session])
  useEffect(() => {
    if (!upgradeOffer) return
    if (upgradeOffer.source === 'level_up') gameAudio.levelUp()
    else gameAudio.treasure()
  }, [upgradeOffer?.offerId, upgradeOffer?.source])
  useEffect(() => {
    if (!upgradeOffer) {
      setUpgradeSeconds(0)
      return
    }
    const updateCountdown = () => setUpgradeSeconds(Math.max(0, Math.ceil((upgradeOffer.deadlineMs - Date.now()) / 1000)))
    updateCountdown()
    const interval = window.setInterval(updateCountdown, 250)
    return () => window.clearInterval(interval)
  }, [upgradeOffer])
  useEffect(() => {
    if (!session || !upgradeOffer || upgradeOffer.selected) return
    const handleKeyDown = (event: KeyboardEvent) => {
      const choiceIndex = Number(event.key) - 1
      if (!Number.isInteger(choiceIndex) || choiceIndex < 0 || choiceIndex >= upgradeOffer.choices.length) return
      event.preventDefault()
      session.network.selectUpgrade(upgradeOffer.offerId, choiceIndex)
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [session, upgradeOffer])
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

  function logoutAccount() {
    localStorage.removeItem(displayNameKey)
    setNameDraft(displayName)
    setDisplayName('')
    setError('')
  }

  async function joinRoom(roomName: string, levelId: string | undefined, characterId: string) {
    if (!displayName || connectingRoom) return
    setError('')
    setConnectingRoom(roomName)
    const nextSession = new MultiplayerSession()
    try {
      await nextSession.connect(roomName, displayName, levelId, characterId)
      const character = characters.find((item) => item.id === characterId)
      if (character) nextSession.bridge.patch({ characterId: character.id, characterName: character.name, spellId: character.defaultSpellId, baseMaxHp: character.maxHp, baseArmorPercent: character.armorPercent, baseMovementSpeed: character.movementSpeed, baseHealthRegeneration: character.healthRegeneration, baseAttackBuffPercent: character.attackBuffPercent, baseCooldownPercent: character.cooldownPercent, baseSpellDamage: character.baseSpell.damage, baseProjectileSpeed: character.baseSpell.projectileSpeed, baseSpellBurst: character.baseSpell.burst, baseSpellDirections: character.baseSpell.directions, baseSpellCooldownMs: character.baseSpell.cooldownMs, beamLength: character.baseSpell.beamLength, beamWidth: character.baseSpell.beamWidth, spellDurationMs: character.baseSpell.durationMs, damageIntervalMs: character.baseSpell.damageIntervalMs, explosionRadius: character.baseSpell.explosionRadius, explosionDurationMs: character.baseSpell.explosionDurationMs, impactDamage: character.baseSpell.impactDamage })
      setCreateCode('')
      setPendingJoin(null)
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
    setUpgradeOffer(null)
    setSelectedEvent(null)
    setSelectedSpell(null)
    setSession(null)
    setHud(null)
  }

  if (pathname === '/armory') {
    return <main className="armory-placeholder"><section><span>ARMORY SYSTEM</span><h1>Access locked.</h1><p>Weapons, spells, and persistent loadouts will arrive in a future milestone.</p><button type="button" onClick={() => navigateTo('/')}>Back to main menu</button></section></main>
  }

  if (!session || !hud) {
    const closeLobby = () => {
      setLobbyOpen(false)
      setCreateCode('')
      setPendingJoin(null)
      setError('')
    }
    const lobby = lobbyActive ? <section className="room-browser lobby-modal" role="dialog" aria-modal="true" aria-label="Multiplayer lobby" onClick={(event) => event.stopPropagation()}>
      {error && <p className="room-error" role="alert">{error}</p>}
      <div className="room-list">
        <div className="room-list-heading"><strong>Active operations</strong><button className="create-room-button" type="button" onClick={() => setCreateCode(generateRoomCode())}>+ Create squad</button></div>
        {rooms.length === 0 ? <div className="empty-rooms"><strong>No active rooms yet.</strong><span>Create one and invite your bros.</span></div> : rooms.map((room) => (
          <article className="room-row" key={room.roomName}>
            <div className="room-code"><span>{room.status} · {room.levelId === 'level-1' ? 'Slime Meadow' : room.levelId}</span><strong>{room.roomName}</strong></div>
            <div className="room-capacity"><span>Players</span><strong>{room.playerCount}/{room.maxPlayers}</strong></div>
            <button type="button" disabled={!room.joinable || Boolean(connectingRoom)} onClick={() => setPendingJoin({ roomName: room.roomName })}>{connectingRoom === room.roomName ? 'Joining…' : room.joinable ? 'Join room' : 'Full'}</button>
          </article>
        ))}
      </div>
    </section> : undefined
    const createSquad = createCode ? <section className="room-browser lobby-modal create-squad-modal" role="dialog" aria-modal="true" aria-labelledby="create-title" onClick={(event) => event.stopPropagation()}><div className="create-modal create-squad-content">
      <button type="button" className="menu-close" aria-label="Close create squad" onClick={() => setCreateCode('')}>×</button><span className="eyebrow">New room</span><h2 id="create-title">Your room is ready.</h2><p>Choose a level, then share the five-letter ID with your squad.</p><label><span>Room ID</span><input readOnly value={createCode} /></label><label><span>Level</span><select value={createLevelId} onChange={(event) => setCreateLevelId(event.target.value)}>{(levels.length ? levels : [{ id: 'level-1', name: 'Slime Meadow', durationSeconds: 900 }]).map((level) => <option key={level.id} value={level.id}>{level.name} · {Math.round(level.durationSeconds / 60)} min</option>)}</select></label><button className="start-room-button" type="button" disabled={Boolean(connectingRoom)} onClick={() => { setCreateCode(''); setPendingJoin({ roomName: createCode, levelId: createLevelId }) }}>Choose character</button>
    </div></section> : undefined
    const characterSelection = pendingJoin ? <section className="create-modal character-modal" role="dialog" aria-modal="true" aria-labelledby="character-title" onClick={(event) => event.stopPropagation()}><button type="button" className="menu-close" aria-label="Close character selection" onClick={() => setPendingJoin(null)}>×</button><span className="eyebrow">Choose character</span><h2 id="character-title">Enter as who?</h2><div className="character-grid">{(characters.length ? characters : [{ id: 'ranger', name: 'Ranger', spriteId: 'character-ranger', maxHp: 100, movementSpeed: 220, baseSpell: { id: 'fireball', damage: 20, cooldownMs: 750, projectileSpeed: 700 } }]).map((character) => <button key={character.id} type="button" onClick={() => void joinRoom(pendingJoin.roomName, pendingJoin.levelId, character.id)}><img src={characterAssetPath(character.spriteId)} alt="" /><strong>{character.name}</strong><span>{character.maxHp} HP · {character.movementSpeed} speed</span><small>{character.baseSpell.id} · {character.baseSpell.damage} damage</small></button>)}</div></section> : undefined
    const overlay = characterSelection ?? createSquad ?? lobby
    const dismissOverlay = () => {
      if (pendingJoin) return setPendingJoin(null)
      if (createCode) return setCreateCode('')
      closeLobby()
    }
    return <MainMenu displayName={displayName} nameDraft={nameDraft} accountError={error} onNameDraftChange={setNameDraft} onLogin={saveName} onLogout={logoutAccount} onPlay={() => setLobbyOpen(true)} overlay={overlay} onDismissOverlay={dismissOverlay} />
  }

  const healthPercent = Math.max(0, (hud.hp / hud.maxHp) * 100)
  const experiencePercent = Math.min(100, (hud.experience / hud.experienceRequired) * 100)
  const elapsedMs = Math.max(0, hud.levelDurationMs - hud.remainingMs)
  return <main className="app-shell"><section className="game-frame">
    <GameCanvas session={session} />
    <div className={`game-hud${hud.bosses.length > 0 ? ' boss-active' : ''}`}><div className="timeline-hud"><div className="timeline-arrow">▼</div><div className="timeline-track" style={{ backgroundPositionX: `${-(elapsedMs / 1000) * 12}px` }}>{hud.timelineEvents.map((event) => { const position = 50 + ((event.atMs - elapsedMs) / hud.levelDurationMs) * 100; return <button key={event.id} type="button" className={`timeline-event ${event.type}`} style={{ left: `${position}%` }} onClick={() => setSelectedEvent(event)}><i>{event.type === 'boss' ? '♛' : event.type === 'meteor_shower' ? '☄' : event.type === 'end' ? '■' : '◆'}</i><span>{formatTimelineTime(event.atMs)}</span></button> })}</div></div>
      {hud.bosses.length > 0 && <div className="boss-bars" style={{ gridTemplateColumns: `repeat(${hud.bosses.length}, minmax(0, 1fr))` }}>{hud.bosses.map((boss) => <section className="boss-health" key={boss.id}><img src={enemyAssetPath(boss.spriteId)} alt="" /><div className="boss-health-content"><div><strong>{boss.name}</strong><span>{boss.hp.toLocaleString()}/{boss.maxHp.toLocaleString()}</span></div><div className="boss-health-track"><i style={{ width: `${Math.max(0, Math.min(100, boss.hp / boss.maxHp * 100))}%` }} /></div></div></section>)}</div>}
      <div className="experience-strip"><i style={{ width: `${experiencePercent}%` }} /><span>LV {hud.level} · {hud.experience}/{hud.experienceRequired} XP</span></div>
      <aside className="health-panel">
        <button className="player-portrait-button" type="button" aria-label="Open your character statistics" onClick={() => setStatsOpen(true)}><img src={characterAssetPath(hud.characterId)} alt="" /><span>YOU</span></button>
        <div className="health-content"><div className="health-heading"><strong>{hud.displayName || 'Ranger'}</strong><b>LV {hud.level}</b></div><div className="health-value"><span>HP</span><b>{hud.hp}/{hud.maxHp}</b></div><div className="health-meter"><i style={{ width: `${healthPercent}%` }} /></div></div>
      </aside>
      <div className="active-spell-list" aria-label="Active spells">{hud.spells.map((spell) => <button key={spell.id} type="button" onClick={() => setSelectedSpell(spell)}><strong>{spellName(spell.id)}</strong><span>LV {spell.level}</span><small>{spell.damage} DMG · {spell.cooldownMs} MS</small></button>)}</div>
      <aside className="life-panel" aria-label={`${hud.lives} shared room lives`}><span>ROOM LIFE</span><strong>{hud.lives}</strong></aside>
      <button className="menu-toggle" type="button" onClick={() => setMenuOpen(true)}><span className="menu-icon"><i /><i /><i /></span><span>Menu</span></button>
      {hud.levelId === 'test-boss' && <button className="debug-level-button" type="button" disabled={Boolean(upgradeOffer)} onClick={() => session.network.debugLevelUp()}>Auto level up</button>}
      {diagnosticsEnabled && <aside className="diagnostics-panel"><strong>Diagnostics</strong><dl><div><dt>FPS</dt><dd>{hud.diagnostics.fps.toFixed(0)}</dd></div><div><dt>Sprites</dt><dd>{hud.diagnostics.visibleSprites}/{hud.diagnostics.activeSprites}</dd></div><div><dt>RTT</dt><dd>{formatMetric(hud.diagnostics.roundTripMs, 'ms')}</dd></div></dl></aside>}
      {upgradeToast && <div className="upgrade-toast" role="status"><span>{upgradeSourceLabel(upgradeToast.source, hud.level)}</span><strong>{upgradeLabel(upgradeToast.attribute)} upgraded</strong><small>+{formatUpgradeValue(upgradeToast.attribute, upgradeToast.addedValue)} · now {formatUpgradeValue(upgradeToast.attribute, upgradeToast.finalValue)}</small></div>}
    </div>
    {selectedEvent && <div className="menu-backdrop" onClick={() => setSelectedEvent(null)}><section className="event-modal" role="dialog" aria-modal="true" aria-labelledby="event-title" onClick={(event) => event.stopPropagation()}><button type="button" className="menu-close" aria-label="Close event" onClick={() => setSelectedEvent(null)}>×</button><span className={`event-type ${selectedEvent.type}`}>{selectedEvent.type.replace('_', ' ')}</span><h2 id="event-title">{selectedEvent.title}</h2><time>{formatTimelineTime(selectedEvent.atMs)}</time><p>{selectedEvent.description}</p></section></div>}
    {selectedSpell && <SpellStatsModal spell={hud.spells.find((spell) => spell.id === selectedSpell.id) ?? selectedSpell} onClose={() => setSelectedSpell(null)} />}
    {statsOpen && <PlayerStatsModal hud={hud} onClose={() => setStatsOpen(false)} onHistory={() => { setStatsOpen(false); setHistoryOpen(true) }} />}
    {historyOpen && <div className="menu-backdrop" onClick={() => setHistoryOpen(false)}><section className="stats-modal history-modal" role="dialog" aria-modal="true" aria-labelledby="history-title" onClick={(event) => event.stopPropagation()}><div className="modal-actions"><button type="button" className="history-button" onClick={() => { setHistoryOpen(false); setStatsOpen(true) }}>Stats</button><button type="button" className="menu-close" aria-label="Close history" onClick={() => setHistoryOpen(false)}>×</button></div><header className="history-heading"><span>THIS RUN</span><h2 id="history-title">Upgrade history</h2><p>Level, treasure, and spell upgrades received by {hud.displayName}.</p></header><div className="history-list">{upgradeHistory.length === 0 ? <div className="history-empty">No upgrades received yet.</div> : upgradeHistory.map((entry) => <article key={entry.id}><i className={entry.source} aria-hidden="true">{entry.source === 'level_up' ? 'LV' : '▣'}</i><div><span>{upgradeSourceLabel(entry.source, hud.level)} · {entry.occurredAt.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span><strong>{upgradeLabel(entry.attribute)}</strong><small>+{formatUpgradeValue(entry.attribute, entry.addedValue)} · {formatUpgradeValue(entry.attribute, entry.finalValue)} total</small></div></article>)}</div></section></div>}
    {upgradeOffer && <div className="upgrade-backdrop"><section className="upgrade-modal" role="dialog" aria-modal="true" aria-labelledby="upgrade-title"><header><span>{upgradeSourceLabel(upgradeOffer.source, upgradeOffer.teamLevel)}</span><h2 id="upgrade-title">{upgradeOffer.source === 'spell_chest' ? 'Choose a new spell' : 'Choose your upgrade'}</h2><p>{upgradeOffer.selected ? `Locked in. Waiting for ${upgradeOffer.pendingCount} teammate${upgradeOffer.pendingCount === 1 ? '' : 's'}.` : upgradeOffer.source === 'spell_chest' ? 'Choose one unowned spell. All owned spells attack automatically.' : 'Pick one card. Every squad member receives their own choices.'}</p><time>{upgradeSeconds}s</time></header><div className="upgrade-card-grid">{upgradeOffer.choices.map((choice, index) => <button key={`${choice.attribute}-${index}`} type="button" disabled={upgradeOffer.selected} onClick={() => session.network.selectUpgrade(upgradeOffer.offerId, index)}><i>{index + 1}</i><span>{upgradeCategory(choice.attribute)}</span><strong>{upgradeLabel(choice.attribute)}</strong><small>{formatUpgradeValue(choice.attribute, choice.currentValue)} → {formatUpgradeValue(choice.attribute, choice.finalValue)}</small><b>+{formatUpgradeValue(choice.attribute, choice.addedValue)}</b></button>)}</div><footer>{upgradeOffer.pendingCount}/{upgradeOffer.totalCount} players choosing · unresolved choices use card 1 when time expires</footer></section></div>}
    {menuOpen && <div className="menu-backdrop" onClick={() => setMenuOpen(false)}><section className="game-menu" role="dialog" aria-modal="true" onClick={(event) => event.stopPropagation()}><button type="button" className="menu-close" aria-label="Close menu" onClick={() => setMenuOpen(false)}>×</button><span className="brand-mark">SB</span><h2>Game menu</h2><p>Room {hud.roomName} · {hud.playerCount}/6 players</p><button className="leave-button" type="button" onClick={leaveRoom}>Leave room</button></section></div>}
    {hud.outcome !== 'playing' && <div className="result-backdrop"><section className="result-card"><span className="eyebrow">Final score</span><h1>{hud.score.toLocaleString()}</h1><p>{hud.kills} slimes defeated · team level {hud.level}</p><button type="button" onClick={leaveRoom}>Back to rooms</button></section></div>}
  </section></main>
}

function PlayerStatsModal({ hud, onClose, onHistory }: { hud: GameHudState; onClose: () => void; onHistory: () => void }) {
  return <div className="menu-backdrop" onClick={onClose}><section className="stats-modal" role="dialog" aria-modal="true" aria-labelledby="stats-title" onClick={(event) => event.stopPropagation()}>
    <div className="modal-actions"><button type="button" className="history-button" onClick={onHistory}>History</button><button type="button" className="menu-close" aria-label="Close statistics" onClick={onClose}>×</button></div>
    <header className="stats-identity"><img src={characterAssetPath(hud.characterId)} alt={hud.characterName} /><div><span>YOUR CHARACTER</span><h2 id="stats-title">{hud.displayName}</h2><p>{hud.characterName} · Team level {hud.level}</p></div></header>
    <div className="stats-columns character-only"><section><h3>Character stats</h3><dl className="attribute-list"><div><dt>Current health</dt><dd>{hud.hp}</dd></div><div><dt>Max health</dt><dd>{statLine(hud.baseMaxHp, hud.maxHp, integer)}</dd></div><div><dt>Armor</dt><dd>{statLine(hud.baseArmorPercent, hud.armorPercent, percent)}</dd></div><div><dt>Movement speed</dt><dd>{statLine(hud.baseMovementSpeed, hud.movementSpeed, integer)}</dd></div><div><dt>Regeneration</dt><dd>{statLine(hud.baseHealthRegeneration, hud.healthRegeneration, integer)}</dd></div><div><dt>Resurrection time</dt><dd>{(hud.resurrectionDurationMs / 1000).toFixed(1)} s</dd></div><div><dt>Resurrection radius</dt><dd>{integer(hud.resurrectionRadius)}</dd></div><div><dt>Resurrection immunity</dt><dd>{(hud.resurrectionImmunityDurationMs / 1000).toFixed(1)} s</dd></div><div><dt>Attack buff</dt><dd>{statLine(hud.baseAttackBuffPercent, hud.attackBuffPercent, percent)}</dd></div><div><dt>Cooldown reduction</dt><dd>{statLine(hud.baseCooldownPercent, hud.cooldownPercent, percent)}</dd></div></dl></section></div>
  </section></div>
}

function SpellStatsModal({ spell, onClose }: { spell: SnapshotSpell; onClose: () => void }) {
  return <div className="menu-backdrop" onClick={onClose}><section className="stats-modal spell-stats-modal" role="dialog" aria-modal="true" aria-labelledby="spell-stats-title" onClick={(event) => event.stopPropagation()}><button type="button" className="menu-close" aria-label="Close spell statistics" onClick={onClose}>×</button><header className="history-heading"><span>ACTIVE SPELL · LEVEL {spell.level}/{spell.maxLevel}</span><h2 id="spell-stats-title">{spellName(spell.id)}</h2><p>{spell.kind.replaceAll('_', ' ')}</p></header><dl className="attribute-list"><div><dt>Damage</dt><dd>{spell.damage}</dd></div><div><dt>Cooldown</dt><dd>{spell.kind === 'aura' ? 'Always on' : `${spell.cooldownMs} ms`}</dd></div>{spell.range > 0 && <div><dt>Range</dt><dd>{integer(spell.range)}</dd></div>}{spell.projectileSpeed > 0 && <div><dt>Projectile speed</dt><dd>{integer(spell.projectileSpeed)}</dd></div>}{spell.projectileRadius > 0 && <div><dt>Projectile radius</dt><dd>{integer(spell.projectileRadius)}</dd></div>}{spell.beamLength > 0 && <div><dt>Beam length</dt><dd>{integer(spell.beamLength)}</dd></div>}{spell.beamWidth > 0 && <div><dt>Beam width</dt><dd>{integer(spell.beamWidth)}</dd></div>}{spell.explosionRadius > 0 && <div><dt>{spell.kind === 'aura' ? 'Aura radius' : 'Blast radius'}</dt><dd>{integer(spell.explosionRadius)}</dd></div>}{spell.durationMs > 0 && spell.kind !== 'aura' && <div><dt>Duration</dt><dd>{spell.durationMs} ms</dd></div>}{spell.damageIntervalMs > 0 && <div><dt>Damage interval</dt><dd>{spell.damageIntervalMs} ms</dd></div>}{spell.impactDamage > 0 && <div><dt>Impact damage</dt><dd>{spell.impactDamage}</dd></div>}<div><dt>Burst</dt><dd>{spell.burst}</dd></div><div><dt>Directions</dt><dd>{spell.directions}</dd></div></dl></section></div>
}

function generateRoomCode(): string { return Array.from({ length: 5 }, () => roomAlphabet[Math.floor(Math.random() * roomAlphabet.length)]).join('') }
function percent(value: number): string { return `${Math.round(value * 100)}%` }
function integer(value: number): string { return Math.round(value).toString() }
function statLine(base: number, final: number, formatter: (value: number) => string): string { return `${formatter(base)} (+${formatter(Math.max(0, final - base))}) ${formatter(final)}` }
function upgradeLabel(attribute: UpgradeAttribute): string {
  if (attribute.startsWith('spell:')) return `${spellName(attribute.slice(6).replace(/:level$/, ''))}${attribute.endsWith(':level') ? ' level' : ''}`
  const labels: Record<string, string> = { max_health: 'Max health', armor: 'Armor', movement_speed: 'Movement speed', health_regeneration: 'Health regeneration', attack_buff: 'Attack buff', cooldown: 'Cooldown reduction', spell_damage: 'Spell damage', projectile_speed: 'Projectile speed', spell_burst: 'Spell burst', spell_directions: 'Spell directions', beam_length: 'Beam length', beam_width: 'Beam width', spell_duration: 'Linger duration', explosion_radius: 'Blast radius', explosion_duration: 'Explosion linger' }
  return labels[attribute] ?? attribute
}
function upgradeCategory(attribute: UpgradeAttribute): string {
  if (attribute.startsWith('spell:')) return 'Spell'
  return ['max_health', 'armor', 'movement_speed', 'health_regeneration'].includes(attribute) ? 'Character' : 'Spell'
}
function formatUpgradeValue(attribute: UpgradeAttribute, value: number): string {
  if (attribute.startsWith('spell:')) return value === 0 ? 'New' : `Lv ${integer(value)}`
  return attribute === 'armor' || attribute === 'attack_buff' || attribute === 'cooldown' ? percent(value) : integer(value)
}
function upgradeSourceLabel(source: UpgradeAppliedPayload['source'], level: number): string { return source === 'level_up' ? `LEVEL ${level}` : source === 'spell_chest' ? 'SPELL CHEST' : 'TREASURE CHEST' }
function formatMetric(value: number, suffix: string): string { return value > 0 ? `${value.toFixed(0)} ${suffix}` : '—' }
function formatTimelineTime(value: number): string { const seconds = Math.floor(value / 1000); return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}` }
function spellName(id: string): string { return id.split('-').map((part) => part[0]?.toUpperCase() + part.slice(1)).join(' ') }
function normalizePath(path: string): string { return path.length > 1 ? path.replace(/\/+$/, '') : '/' }
