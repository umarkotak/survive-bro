import { useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react'

import { gameAudio } from '../audio/gameAudio'
import { menuMedia } from '../config/menuMedia'

interface MainMenuProps {
  displayName: string
  nameDraft: string
  accountError: string
  onNameDraftChange: (value: string) => void
  onLogin: (event: FormEvent<HTMLFormElement>) => void
  onLogout: () => void
  onPlay: () => void
  overlay?: ReactNode
  onDismissOverlay?: () => void
}

export function MainMenu({ displayName, nameDraft, accountError, onNameDraftChange, onLogin, onLogout, onPlay, overlay, onDismissOverlay }: MainMenuProps) {
  const [videoReady, setVideoReady] = useState(false)
  const [audioUnlocked, setAudioUnlocked] = useState(false)
  const [soundEnabled, setSoundEnabled] = useState(menuMedia.soundEnabledByDefault)
  const videoRef = useRef<HTMLVideoElement>(null)
  const loginInputRef = useRef<HTMLInputElement>(null)

  function unlockAudio(): void {
    gameAudio.unlock()
    const video = videoRef.current
    if (video) {
      video.volume = menuMedia.videoVolume
      video.muted = !soundEnabled
      void video.play().catch(() => { video.muted = true })
    }
    setAudioUnlocked(true)
  }

  useEffect(() => {
    const unlock = () => unlockAudio()
    window.addEventListener('pointerdown', unlock, { once: true, passive: true })
    window.addEventListener('keydown', unlock, { once: true })
    return () => {
      window.removeEventListener('pointerdown', unlock)
      window.removeEventListener('keydown', unlock)
    }
  }, [])

  useEffect(() => {
    const video = videoRef.current
    if (!video) return
    video.volume = menuMedia.videoVolume
    video.muted = !audioUnlocked || !soundEnabled
    if (videoReady) void video.play().catch(() => { video.muted = true })
  }, [audioUnlocked, soundEnabled, videoReady])

  function toggleSound(): void {
    if (!audioUnlocked) unlockAudio()
    gameAudio.menuClick()
    setSoundEnabled((enabled) => !enabled)
  }

  const playHoverSound = () => { if (soundEnabled) gameAudio.menuHover() }
  const choose = (action: () => void) => {
    if (soundEnabled) gameAudio.menuClick()
    action()
  }
  const requestPlay = () => {
    if (!displayName) {
      if (soundEnabled) gameAudio.menuClick()
      loginInputRef.current?.focus()
      return
    }
    choose(onPlay)
  }
  const submitLogin = (event: FormEvent<HTMLFormElement>) => {
    if (soundEnabled) gameAudio.menuClick()
    onLogin(event)
  }
  const logout = () => {
    if (soundEnabled) gameAudio.menuClick()
    onLogout()
    window.requestAnimationFrame(() => loginInputRef.current?.focus())
  }

  return <main className="main-menu-shell">
    <div className="main-menu-media" aria-hidden="true">
      <img src={menuMedia.backgroundImage} alt="" />
      {menuMedia.preferVideo && <video
        ref={videoRef}
        className={videoReady ? 'is-ready' : ''}
        src={menuMedia.backgroundVideo}
        poster={menuMedia.backgroundImage}
        autoPlay
        loop
        muted
        playsInline
        preload="auto"
        onCanPlay={() => setVideoReady(true)}
      />}
    </div>
    <div className="main-menu-shade" aria-hidden="true" />
    {overlay ? <div className="main-menu-overlay" onClick={(event) => { if (event.target === event.currentTarget) onDismissOverlay?.() }}>{overlay}</div> : <>
      <aside className={`main-menu-account${displayName ? ' is-online' : ''}`} aria-label="Local player account">
      {displayName ? <>
        <div className="main-menu-account-avatar" aria-hidden="true">{displayName.slice(0, 2).toUpperCase()}</div>
        <div className="main-menu-account-identity"><span>Operative online</span><strong>{displayName}</strong><small>Local profile</small></div>
        <button type="button" onClick={logout} onPointerEnter={playHoverSound} onFocus={playHoverSound}>Change</button>
      </> : <form onSubmit={submitLogin}>
        <header><span>Operative login</span><small>Local callsign required</small></header>
        <div>
          <input ref={loginInputRef} value={nameDraft} onChange={(event) => onNameDraftChange(event.target.value)} maxLength={20} autoComplete="nickname" placeholder="Enter callsign" aria-label="Callsign" />
          <button type="submit" onPointerEnter={playHoverSound} onFocus={playHoverSound}>Login</button>
        </div>
        {accountError && <p role="alert">{accountError}</p>}
      </form>}
      </aside>
      <button className="main-menu-sound" type="button" onClick={toggleSound} onPointerEnter={playHoverSound} aria-pressed={soundEnabled}>
        <i aria-hidden="true">{soundEnabled ? '◖))' : '◖×'}</i>
        <span>{soundEnabled ? 'Sound on' : 'Sound off'}</span>
      </button>
      <section className="main-menu-panel" aria-label="Heavy Armament main menu">
        <img className="main-menu-logo" src={menuMedia.logo} alt="Heavy Armament" />
        <nav className="main-menu-actions" aria-label="Main navigation">
          <button className="armament-button is-primary" type="button" onClick={requestPlay} onPointerEnter={playHoverSound} onFocus={playHoverSound}>
            <i aria-hidden="true" />
            <span>Play</span>
            <small>{displayName ? 'Open multiplayer lobby' : 'Login required'}</small>
          </button>
          <button className="armament-button is-locked" type="button" disabled>
            <span>Armory</span>
            <small>System locked · unavailable</small>
          </button>
        </nav>
      </section>
      <footer className="main-menu-footer"><span>ONLINE SURVIVAL SYSTEM</span><b>V0.1.0</b></footer>
    </>}
  </main>
}
