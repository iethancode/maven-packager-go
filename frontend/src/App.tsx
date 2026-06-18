import { useState, useEffect, useCallback } from 'react'
import type { InitialStateDTO, Commit } from './types'
import { api, isDevMode, onEvent, offEvent } from './wails'
import TopBar from './components/TopBar'
import CommitList from './components/CommitList'
import DiffPanel from './components/DiffPanel'
import BuildControl from './components/BuildControl'
import LogPanel from './components/LogPanel'
import { useBuildEvents } from './hooks/useBuildEvents'

const defaultConfig = {
  theme: 'Light',
  output_dir: '',
  build_speed: '快速模式',
  build_scope_mode: '稳妥模式',
  last_branch: '',
  smart_dependency: true,
  project_root: '',
}

function normalizeInitialState(s: Partial<InitialStateDTO> | null | undefined): InitialStateDTO {
  return {
    projectRoot: s?.projectRoot || '',
    config: { ...defaultConfig, ...(s?.config || {}) },
    hasGit: !!s?.hasGit,
    hasPom: !!s?.hasPom,
    hasMaven: !!s?.hasMaven,
    branches: s?.branches || [],
    currentBranch: s?.currentBranch || '',
    commits: s?.commits || [],
    moduleCount: s?.moduleCount || 0,
    defaultOutputDir: s?.defaultOutputDir || '',
  }
}

function App() {
  const [dark, setDark] = useState(false)
  const [state, setState] = useState<InitialStateDTO | null>(null)
  const [loading, setLoading] = useState(true)
  const [selectedHashes, setSelectedHashes] = useState<string[]>([])
  const [diffHash, setDiffHash] = useState<string | null>(null)
  const [commitsLoading, setCommitsLoading] = useState(false)
  const [reloadToken, setReloadToken] = useState(0)

  const buildState = useBuildEvents()

  // 加载首屏
  useEffect(() => { loadInitial() }, [])

  const loadInitial = useCallback(async () => {
    try {
      const s = normalizeInitialState(await api.getInitialState())
      // 防御 null/undefined
      setState(s)
      setDark(s.config.theme === 'Dark')
    } catch (e) {
      console.warn('Failed to load initial state, using mock data:', e)
      setState({
        projectRoot: 'D:/Projects/demo',
        config: {
          theme: 'Light', output_dir: '', build_speed: '快速模式',
          build_scope_mode: '稳妥模式', last_branch: '',
          smart_dependency: true, project_root: '',
        },
        hasGit: true, hasPom: true, hasMaven: true,
        branches: ['main', 'dev', 'feature/test'],
        currentBranch: 'main',
        commits: Array.from({ length: 20 }, (_, i) => ({
          hash: `a${i.toString(16).padStart(6, '0')}`,
          author: ['张三', '李四', '王五', '赵六'][i % 4],
          date: `2025-12-${String(20 - i).padStart(2, '0')} 10:${String(i).padStart(2, '0')}:00`,
          msg: ['修复登录页', '添加用户管理', '优化查询性能', '重构鉴权模块'][i % 4],
        })),
        moduleCount: 12,
        defaultOutputDir: 'D:/Projects/demo/dist_jars',
      })
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!state?.hasGit || isDevMode()) return
    let cancelled = false
    setCommitsLoading(true)
    setState(prev => prev ? { ...prev, commits: [] } : prev)
    api.listBranches().then(branches => {
      if (!cancelled) {
        setState(prev => prev ? { ...prev, branches: branches || [] } : prev)
      }
    }).catch(() => {})
    api.currentBranch().then(currentBranch => {
      if (!cancelled) {
        setState(prev => prev ? { ...prev, currentBranch: currentBranch || prev.currentBranch } : prev)
      }
    }).catch(() => {})
    offEvent('commits:batch')
    offEvent('commits:complete')
    onEvent('commits:batch', (payload: string) => {
      if (cancelled) return
      let parsed: { commits?: Commit[] } = {}
      try {
        parsed = typeof payload === 'string' ? JSON.parse(payload || '{}') : (payload || {})
      } catch (e) {
        console.warn('Invalid commits batch payload:', e)
        return
      }
      const batch = parsed.commits || []
      setState(prev => {
        if (!prev) return prev
        const byHash = new Map(prev.commits.map(c => [c.hash, c]))
        batch.forEach((c: Commit) => byHash.set(c.hash, c))
        const commits = Array.from(byHash.values())
          .sort((a, b) => b.date.localeCompare(a.date))
          .slice(0, 300)
        return { ...prev, commits }
      })
    })
    onEvent('commits:complete', () => {
      if (!cancelled) setCommitsLoading(false)
    })
    api.startLoadCommits(300).catch(() => {
      if (!cancelled) setCommitsLoading(false)
    })
    return () => {
      cancelled = true
      offEvent('commits:batch')
      offEvent('commits:complete')
    }
  }, [state?.projectRoot, state?.hasGit, reloadToken])

  useEffect(() => {
    if (!state?.hasPom || isDevMode()) return
    let cancelled = false
    api.countModules().then(moduleCount => {
      if (!cancelled) {
        setState(prev => prev ? { ...prev, moduleCount: moduleCount || 0 } : prev)
      }
    }).catch(() => {})
    return () => { cancelled = true }
  }, [state?.projectRoot, state?.hasPom, reloadToken])

  // 主题切换
  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
  }, [dark])

  const toggleTheme = useCallback(() => {
    setDark(d => {
      api.patchConfig({ theme: !d ? 'Dark' : 'Light' }).catch(() => {})
      return !d
    })
  }, [])

  // 构建选项切换
  const handleSpeedChange = useCallback((speed: string) => {
    api.patchConfig({ build_speed: speed }).catch(() => {})
    setState(prev => prev ? { ...prev, config: { ...prev.config, build_speed: speed } } : prev)
  }, [])

  const handleScopeChange = useCallback((mode: string) => {
    api.patchConfig({ build_scope_mode: mode }).catch(() => {})
    setState(prev => prev ? { ...prev, config: { ...prev.config, build_scope_mode: mode } } : prev)
  }, [])

  const handleSmartDepChange = useCallback((enabled: boolean) => {
    api.patchConfig({ smart_dependency: enabled }).catch(() => {})
    setState(prev => prev ? { ...prev, config: { ...prev.config, smart_dependency: enabled } } : prev)
  }, [])

  const handleSwitchBranch = useCallback(async (branch: string) => {
    if (isDevMode()) return
    const result = await api.switchBranch(branch)
    if (result.success) {
      setState(prev => prev ? {
        ...prev,
        branches: result.branches || prev.branches,
        currentBranch: result.currentBranch,
        commits: [],
      } : prev)
      setSelectedHashes([])
      setReloadToken(v => v + 1)
    }
  }, [])

  const handleChangeRoot = useCallback(async () => {
    const dir = await api.pickDirectory('选择项目根目录', state?.projectRoot || '')
    if (dir) {
      const newState = normalizeInitialState(await api.changeProjectRoot(dir))
      setState(newState)
      setSelectedHashes([])
      setReloadToken(v => v + 1)
    }
  }, [state?.projectRoot])

  const handlePickOutput = useCallback(async () => {
    const dir = await api.pickDirectory('选择输出目录', state?.config.output_dir || state?.defaultOutputDir || '')
    if (dir && dir.trim()) {
      await api.patchConfig({ output_dir: dir })
      setState(prev => prev ? { ...prev, config: { ...prev.config, output_dir: dir } } : prev)
    }
  }, [state?.config.output_dir, state?.defaultOutputDir])

  const handleRefresh = useCallback(async () => {
    const s = normalizeInitialState(await api.refreshProject())
    setState(s)
    setSelectedHashes([])
    setReloadToken(v => v + 1)
  }, [])

  const handleSelectCommit = useCallback((hash: string, checked: boolean) => {
    setSelectedHashes(prev => {
      if (checked) return [...prev, hash]
      return prev.filter(h => h !== hash)
    })
  }, [])

  const handleSelectAll = useCallback((checked: boolean, pageCommits: Commit[]) => {
    if (checked) {
      setSelectedHashes(prev => {
        const set = new Set(prev)
        pageCommits.forEach(c => set.add(c.hash))
        return Array.from(set)
      })
    } else {
      const pageSet = new Set(pageCommits.map(c => c.hash))
      setSelectedHashes(prev => prev.filter(h => !pageSet.has(h)))
    }
  }, [])

  const handleOpenDiff = useCallback((hash: string) => setDiffHash(hash), [])
  const handleCloseDiff = useCallback(() => setDiffHash(null), [])

  const handleStartBuild = useCallback(async (opts: {
    speedMode: string; scopeMode: string; smartDependency: boolean; outputDir: string
  }) => {
    if (selectedHashes.length === 0) return
    buildState.reset()
    try {
      await api.startPackaging({
        commits: selectedHashes,
        speedMode: opts.speedMode,
        scopeMode: opts.scopeMode,
        smartDependency: opts.smartDependency,
        outputDir: opts.outputDir,
      })
    } catch (e) {
      console.error('Start packaging failed:', e)
    }
  }, [selectedHashes, buildState])

  const handleCancelBuild = useCallback(async () => {
    await api.cancelPackaging()
    buildState.setIdle()
  }, [buildState])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-bg-secondary">
        <div className="text-center">
          <div className="inline-block w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          <p className="mt-3 text-text-secondary text-sm">加载中...</p>
        </div>
      </div>
    )
  }

  if (!state) return null

  const commits = state.commits || []

  return (
    <div className="h-screen flex flex-col bg-bg-secondary text-text-primary font-sans select-none">
      {/* ─── TopBar ─── */}
      <TopBar
        branches={state.branches || []}
        currentBranch={state.currentBranch}
        projectRoot={state.projectRoot}
        outputDir={state.config.output_dir || state.defaultOutputDir}
        moduleCount={state.moduleCount}
        buildSpeed={state.config.build_speed}
        buildScopeMode={state.config.build_scope_mode}
        smartDependency={state.config.smart_dependency}
        dark={dark}
        onSwitchBranch={handleSwitchBranch}
        onChangeRoot={handleChangeRoot}
        onPickOutput={handlePickOutput}
        onRefresh={handleRefresh}
        onToggleTheme={toggleTheme}
        onSpeedChange={handleSpeedChange}
        onScopeChange={handleScopeChange}
        onSmartDepChange={handleSmartDepChange}
      />

      {/* ─── Tabs ─── */}
      <div className="flex border-b border-border bg-bg-card px-4">
        <div className="px-4 py-2.5 text-sm font-medium border-b-2 border-primary text-primary">
          提交列表 <span className="text-xs text-text-muted">({commitsLoading ? '加载中' : commits.length})</span>
        </div>
        <div className="ml-auto flex items-center text-xs text-text-muted px-3">
          {selectedHashes.length > 0 && (
            <span className="text-primary font-medium">{selectedHashes.length} 个提交已选中</span>
          )}
        </div>
      </div>

      {/* ─── Main Content ─── */}
      <div className="flex-1 flex min-h-0">
        <div className="flex-1 flex flex-col min-w-0">
          <CommitList
            commits={commits}
            loading={commitsLoading}
            selectedHashes={selectedHashes}
            onSelectCommit={handleSelectCommit}
            onSelectAll={handleSelectAll}
            onOpenDiff={handleOpenDiff}
          />
        </div>
      </div>

      {/* ─── Log Panel ─── */}
      <LogPanel
        logs={buildState.logs}
        progress={buildState.progress}
        progressText={buildState.progressText}
        status={buildState.status}
      />

      {/* ─── Build Control ─── */}
      <BuildControl
        selectedCount={selectedHashes.length}
        status={buildState.status}
        progress={buildState.progress}
        progressText={buildState.progressText}
        outputDir={state.config.output_dir || state.defaultOutputDir}
        onStart={() => handleStartBuild({
          speedMode: state.config.build_speed,
          scopeMode: state.config.build_scope_mode,
          smartDependency: state.config.smart_dependency,
          outputDir: state.config.output_dir || state.defaultOutputDir,
        })}
        onCancel={handleCancelBuild}
      />

      {/* ─── Diff Modal ─── */}
      {diffHash && (
        <DiffPanel hash={diffHash} onClose={handleCloseDiff} />
      )}
    </div>
  )
}

export default App
