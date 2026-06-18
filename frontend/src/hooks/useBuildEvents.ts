import { useState, useEffect, useCallback, useRef } from 'react'
import { onEvent, offEvent, api, isDevMode } from '../wails'
import type { LogEvent, ProgressEvent, CompleteEvent, ModuleDoneEvent, BuildStatus } from '../types'

type BuildState = {
  status: BuildStatus
  progress: number
  progressText: string
  logs: LogEvent[]
  buildResult: any | null
}

export function useBuildEvents() {
  const [state, setState] = useState<BuildState>({
    status: 'idle',
    progress: 0,
    progressText: '',
    logs: [],
    buildResult: null,
  })

  const logRef = useRef<LogEvent[]>([])

  const appendLog = useCallback((ev: LogEvent) => {
    logRef.current = [...logRef.current, ev]
    setState(s => ({ ...s, logs: [...logRef.current] }))
  }, [])

  useEffect(() => {
    if (isDevMode()) return

    const onLog = (payload: string) => {
      try {
        const ev: LogEvent = JSON.parse(payload)
        appendLog(ev)
      } catch { /* skip */ }
    }
    const onProgress = (payload: string) => {
      try {
        const ev: ProgressEvent = JSON.parse(payload)
        setState(s => ({ ...s, progress: ev.value, progressText: ev.text }))
      } catch { /* skip */ }
    }
    const onComplete = (payload: string) => {
      try {
        const ev: CompleteEvent = JSON.parse(payload)
        setState(s => ({ ...s, status: 'done', progress: ev.success ? 1 : 0, buildResult: ev }))
      } catch { /* skip */ }
    }
    const onModuleDone = (payload: string) => {
      try {
        const ev: ModuleDoneEvent = JSON.parse(payload)
        const tag = ev.success ? 'success' : 'error'
        appendLog({
          line: `  ${ev.success ? '✓' : '✗'} ${ev.module} (${(ev.elapsedMs / 1000).toFixed(1)}s)`,
          level: tag,
          ts: Date.now(),
        })
      } catch { /* skip */ }
    }

    onEvent('build:log', onLog)
    onEvent('build:progress', onProgress)
    onEvent('build:complete', onComplete)
    onEvent('build:module-done', onModuleDone)

    return () => {
      offEvent('build:log')
      offEvent('build:progress')
      offEvent('build:complete')
      offEvent('build:module-done')
    }
  }, [appendLog])

  const reset = useCallback(() => {
    logRef.current = []
    setState({
      status: 'building',
      progress: 0,
      progressText: '开始...',
      logs: [],
      buildResult: null,
    })
  }, [])

  const setIdle = useCallback(() => {
    logRef.current = []
    setState({
      status: 'idle',
      progress: 0,
      progressText: '',
      logs: [],
      buildResult: null,
    })
  }, [])

  return { ...state, reset, setIdle }
}
