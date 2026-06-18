import { useRef, useEffect, useCallback, useState } from 'react'
import type { LogEvent } from '../types'

interface LogPanelProps {
  logs: LogEvent[]
  progress: number
  progressText: string
  status: 'idle' | 'building' | 'done'
}

const levelColor: Record<string, string> = {
  header: 'text-primary font-semibold',
  success: 'text-success',
  warning: 'text-warning',
  error: 'text-error',
  info: 'text-text-secondary',
}

const MIN_HEIGHT = 80
const DEFAULT_HEIGHT = 140

export default function LogPanel({ logs, status }: LogPanelProps) {
  const ref = useRef<HTMLDivElement>(null)
  const [height, setHeight] = useState(DEFAULT_HEIGHT)
  const resizingRef = useRef(false)

  useEffect(() => {
    if (ref.current) {
      ref.current.scrollTop = ref.current.scrollHeight
    }
  }, [logs])

  const onMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    resizingRef.current = true
    const startY = e.clientY
    const startH = height
    const onMove = (ev: MouseEvent) => {
      if (!resizingRef.current) return
      const delta = startY - ev.clientY
      setHeight(Math.max(MIN_HEIGHT, startH + delta))
    }
    const onUp = () => {
      resizingRef.current = false
      document.removeEventListener('mousemove', onMove)
      document.removeEventListener('mouseup', onUp)
    }
    document.addEventListener('mousemove', onMove)
    document.addEventListener('mouseup', onUp)
  }, [height])

  if (logs.length === 0 && status === 'idle') return null

  return (
    <div
      className="border-t border-border bg-bg-card flex flex-col relative"
      style={{ height: `${height}px`, minHeight: `${MIN_HEIGHT}px` }}
    >
      {/* Resize handle */}
      <div
        className="absolute -top-[3px] left-0 right-0 h-1.5 cursor-ns-resize group z-10"
        onMouseDown={onMouseDown}
      >
        <div className="h-full w-10 mx-auto rounded-full bg-border group-hover:bg-primary transition-colors" />
      </div>

      <div className="px-4 py-1 border-b border-border text-xs text-text-muted flex items-center gap-2">
        <span>日志</span>
        {status === 'building' && (
          <div className="w-1.5 h-1.5 rounded-full bg-warning animate-pulse" />
        )}
      </div>
      <div
        ref={ref}
        className="flex-1 overflow-y-auto font-mono text-xs px-3 py-1.5 space-y-0.5"
      >
        {logs.map((log, i) => (
          <div key={i} className={`leading-5 ${levelColor[log.level] || 'text-text-secondary'}`}>
            {log.line}
          </div>
        ))}
      </div>
    </div>
  )
}
