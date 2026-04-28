let audioContext: AudioContext | null = null
let lastPlayAtMs = 0

const SOUND_COOLDOWN_MS = 250
const SOUND_DURATION_S = 0.12
const SOUND_VOLUME = 0.05
const SOUND_FREQUENCY_HZ = 880

function getAudioContext(): AudioContext | null {
  if (typeof window === "undefined") {
    return null
  }
  if (typeof window.AudioContext === "undefined") {
    return null
  }
  if (!audioContext) {
    audioContext = new window.AudioContext()
  }
  return audioContext
}

// warmupMessageSound инициирует AudioContext после user gesture.
export async function warmupMessageSound(): Promise<void> {
  const context = getAudioContext()
  if (!context) return
  if (context.state === "suspended") {
    await context.resume()
  }
}

// playNewMessageSound проигрывает короткий сигнал о новом сообщении.
export function playNewMessageSound(): void {
  const nowMs = Date.now()
  if (nowMs - lastPlayAtMs < SOUND_COOLDOWN_MS) {
    return
  }

  const context = getAudioContext()
  if (!context || context.state !== "running") {
    return
  }

  const oscillator = context.createOscillator()
  const gain = context.createGain()
  oscillator.type = "sine"
  oscillator.frequency.setValueAtTime(SOUND_FREQUENCY_HZ, context.currentTime)
  gain.gain.setValueAtTime(SOUND_VOLUME, context.currentTime)

  oscillator.connect(gain)
  gain.connect(context.destination)

  oscillator.start()
  oscillator.stop(context.currentTime + SOUND_DURATION_S)
  lastPlayAtMs = nowMs
}
