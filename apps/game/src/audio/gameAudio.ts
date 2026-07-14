type ToneOptions = {
  frequency: number
  endFrequency?: number
  duration: number
  volume: number
  type?: OscillatorType
  delay?: number
}

class GameAudio {
  private context: AudioContext | null = null
  private lastShotAt = 0
  private lastMenuHoverAt = 0

  unlock = (): void => {
    const context = this.getContext()
    if (context?.state === 'suspended') void context.resume()
  }

  fireball(): void {
    const now = performance.now()
    if (now - this.lastShotAt < 90) return
    this.lastShotAt = now
    this.tone({ frequency: 310, endFrequency: 150, duration: 0.11, volume: 0.035, type: 'sawtooth' })
  }

  soulTrack(): void {
    this.tone({ frequency: 760, endFrequency: 260, duration: 0.28, volume: 0.045, type: 'sawtooth' })
    this.tone({ frequency: 980, endFrequency: 520, duration: 0.42, volume: 0.025, type: 'sine', delay: 0.04 })
  }

  damage(): void {
    this.tone({ frequency: 120, endFrequency: 70, duration: 0.15, volume: 0.055, type: 'square' })
  }

  levelUp(): void {
    this.tone({ frequency: 440, endFrequency: 660, duration: 0.12, volume: 0.05 })
    this.tone({ frequency: 660, endFrequency: 880, duration: 0.16, volume: 0.045, delay: 0.1 })
  }

  treasure(): void {
    this.tone({ frequency: 520, duration: 0.09, volume: 0.05, type: 'triangle' })
    this.tone({ frequency: 780, duration: 0.12, volume: 0.045, type: 'triangle', delay: 0.08 })
    this.tone({ frequency: 1040, duration: 0.15, volume: 0.04, type: 'triangle', delay: 0.16 })
  }

  menuHover(): void {
    const now = performance.now()
    if (now - this.lastMenuHoverAt < 90) return
    this.lastMenuHoverAt = now
    this.tone({ frequency: 180, endFrequency: 260, duration: 0.065, volume: 0.022, type: 'square' })
    this.tone({ frequency: 920, endFrequency: 720, duration: 0.085, volume: 0.012, type: 'sine', delay: 0.018 })
  }

  menuClick(): void {
    this.tone({ frequency: 240, endFrequency: 130, duration: 0.09, volume: 0.035, type: 'square' })
    this.tone({ frequency: 680, endFrequency: 980, duration: 0.12, volume: 0.022, type: 'triangle', delay: 0.025 })
  }

  private getContext(): AudioContext | null {
    if (typeof window === 'undefined' || !window.AudioContext) return null
    this.context ??= new AudioContext()
    return this.context
  }

  private tone(options: ToneOptions): void {
    const context = this.getContext()
    if (!context || context.state !== 'running') return
    const start = context.currentTime + (options.delay ?? 0)
    const end = start + options.duration
    const oscillator = context.createOscillator()
    const gain = context.createGain()
    oscillator.type = options.type ?? 'sine'
    oscillator.frequency.setValueAtTime(options.frequency, start)
    oscillator.frequency.exponentialRampToValueAtTime(Math.max(1, options.endFrequency ?? options.frequency), end)
    gain.gain.setValueAtTime(0.0001, start)
    gain.gain.exponentialRampToValueAtTime(options.volume, start + 0.012)
    gain.gain.exponentialRampToValueAtTime(0.0001, end)
    oscillator.connect(gain).connect(context.destination)
    oscillator.start(start)
    oscillator.stop(end + 0.01)
  }
}

export const gameAudio = new GameAudio()

export function installAudioUnlock(): () => void {
  const unlock = () => gameAudio.unlock()
  window.addEventListener('pointerdown', unlock, { passive: true })
  window.addEventListener('keydown', unlock)
  return () => {
    window.removeEventListener('pointerdown', unlock)
    window.removeEventListener('keydown', unlock)
  }
}
