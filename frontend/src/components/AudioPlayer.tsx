import { useRef, useState } from 'react'

import { Icon } from './ui'

type AudioPlayerProps = {
  src: string
  label: string
  title: string
  download?: {
    filename: string
    label: string
  }
}

function formatAudioTime(seconds: number) {
  if (!Number.isFinite(seconds) || seconds < 0) {
    return '00:00'
  }

  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = Math.floor(seconds % 60)
  return `${String(minutes).padStart(2, '0')}:${String(remainingSeconds).padStart(2, '0')}`
}

export function AudioPlayer({ src, label, title, download }: AudioPlayerProps) {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const [isPlaying, setIsPlaying] = useState(false)
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)
  const [volume, setVolume] = useState(100)
  const [isMuted, setIsMuted] = useState(false)

  async function togglePlayback() {
    const audio = audioRef.current
    if (!audio) {
      return
    }

    if (audio.paused) {
      try {
        await audio.play()
        setIsPlaying(true)
      } catch {
        setIsPlaying(false)
      }
      return
    }

    audio.pause()
    setIsPlaying(false)
  }

  function handleSeek(value: string) {
    const audio = audioRef.current
    if (!audio) {
      return
    }

    const nextTime = Number(value)
    audio.currentTime = nextTime
    setCurrentTime(nextTime)
  }

  function handleVolumeChange(value: string) {
    const audio = audioRef.current
    const nextVolume = Number(value)
    setVolume(nextVolume)
    setIsMuted(nextVolume === 0)

    if (audio) {
      audio.volume = nextVolume / 100
      audio.muted = nextVolume === 0
    }
  }

  function toggleMute() {
    const audio = audioRef.current
    const nextMuted = !isMuted
    setIsMuted(nextMuted)

    if (audio) {
      audio.muted = nextMuted
    }
  }

  return (
    <div
      role="group"
      aria-label={`${label}控制`}
      className="rounded-xl border border-outline-variant bg-surface-container-lowest p-md shadow-calm"
    >
      <audio
        ref={audioRef}
        aria-label={label}
        src={src}
        preload="metadata"
        onLoadedMetadata={(event) => setDuration(event.currentTarget.duration)}
        onTimeUpdate={(event) => setCurrentTime(event.currentTarget.currentTime)}
        onEnded={() => setIsPlaying(false)}
        className="hidden"
      />
      <div className="mb-sm flex items-center justify-between gap-md">
        <p className="flex min-w-0 items-center gap-xs text-label-md font-bold text-on-surface-variant">
          <Icon name="graphic_eq" className="text-[18px] text-primary" />
          <span>{title}</span>
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-sm rounded-lg bg-surface-bright px-sm py-sm">
        <button
          type="button"
          onClick={togglePlayback}
          aria-label={`${isPlaying ? '暫停' : '播放'} ${label}`}
          className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary text-on-primary shadow-calm transition-all hover:bg-primary-container hover:text-on-primary-container active:scale-[0.98]"
        >
          <Icon name={isPlaying ? 'pause' : 'play_arrow'} className="text-[22px]" />
        </button>
        <input
          type="range"
          min="0"
          max={duration || 0}
          step="0.1"
          value={currentTime}
          disabled={!duration}
          onChange={(event) => handleSeek(event.currentTarget.value)}
          aria-label={`${label}播放進度`}
          className="h-2 min-w-0 flex-1 accent-primary disabled:opacity-50"
        />
        <span className="min-w-[5.75rem] text-right text-label-md font-bold tabular-nums text-on-surface-variant">
          {formatAudioTime(currentTime)} / {formatAudioTime(duration)}
        </span>
        <div
          role="group"
          aria-label={`${label}右側工具`}
          className="ml-auto flex min-w-[12rem] shrink-0 items-center justify-end gap-xs"
        >
          <button
            type="button"
            onClick={toggleMute}
            aria-label={`${isMuted ? '取消靜音' : '靜音'} ${label}`}
            className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-primary transition-all hover:bg-primary/5 active:scale-[0.98]"
          >
            <Icon name={isMuted || volume === 0 ? 'volume_off' : 'volume_up'} className="text-[20px]" />
          </button>
          <input
            type="range"
            min="0"
            max="100"
            step="1"
            value={volume}
            onChange={(event) => handleVolumeChange(event.currentTarget.value)}
            aria-label={`${label}音量`}
            className="h-2 w-20 accent-primary md:w-24"
          />
          {download ? (
            <a
              href={src}
              download={download.filename}
              aria-label={download.label}
              className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-outline-variant bg-surface-container-low text-primary transition-all hover:bg-primary/5 active:scale-[0.98]"
            >
              <Icon name="download" className="text-[18px]" />
            </a>
          ) : null}
        </div>
      </div>
    </div>
  )
}
