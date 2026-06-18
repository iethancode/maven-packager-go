interface TopBarProps {
  branches: string[]
  currentBranch: string
  projectRoot: string
  outputDir: string
  moduleCount: number
  buildSpeed: string
  buildScopeMode: string
  smartDependency: boolean
  dark: boolean
  onSwitchBranch: (branch: string) => void
  onChangeRoot: () => void
  onPickOutput: () => void
  onRefresh: () => void
  onToggleTheme: () => void
  onSpeedChange: (speed: string) => void
  onScopeChange: (mode: string) => void
  onSmartDepChange: (enabled: boolean) => void
}

export default function TopBar({
  branches, currentBranch, projectRoot, outputDir, moduleCount,
  buildSpeed, buildScopeMode, smartDependency, dark,
  onSwitchBranch, onChangeRoot, onPickOutput, onRefresh, onToggleTheme,
  onSpeedChange, onScopeChange, onSmartDepChange,
}: TopBarProps) {

  return (
    <div className="bg-bg-card border-b border-border">
      {/* Row 1: Title + Branch + Refresh + Theme */}
      <div className="flex items-center justify-between px-5 py-2 border-b border-border-light">
        <div className="flex items-center gap-4">
          <h1 className="text-sm font-bold text-text-primary">Git 增量打包工具</h1>
          <div className="w-px h-4 bg-border" />
          <div className="flex items-center gap-2">
            <span className="text-xs text-text-secondary">分支</span>
            <select
              value={currentBranch}
              onChange={e => onSwitchBranch(e.target.value)}
              className="h-7 pl-2 pr-6 text-sm rounded-md border border-border bg-bg-secondary text-text-primary focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer"
            >
              {branches.map(b => (
                <option key={b} value={b}>{b}</option>
              ))}
              {branches.length === 0 && <option value="">无分支</option>}
            </select>
          </div>
          <button
            onClick={onRefresh}
            className="h-7 px-3 text-xs rounded-md border border-border bg-bg-secondary text-text-secondary hover:text-text-primary hover:border-primary transition-colors"
          >
            ↻ 刷新
          </button>
        </div>

        <div className="flex items-center gap-3">
          <span className="text-xs text-text-muted">模块: {moduleCount}</span>
          <button
            onClick={onToggleTheme}
            className="h-7 w-7 flex items-center justify-center rounded-md border border-border bg-bg-secondary hover:bg-bg-surface transition-colors text-sm"
            title={dark ? '切换亮色' : '切换暗色'}
          >
            {dark ? '☀' : '☾'}
          </button>
        </div>
      </div>

      {/* Row 2: Build Controls */}
      <div className="flex items-center gap-5 px-5 py-1.5 border-b border-border-light">
        <div className="flex items-center gap-1.5">
          <span className="text-xs text-text-secondary">速度</span>
          <div className="flex gap-1">
            {['快速模式', '标准模式', '兼容模式'].map(s => (
              <button
                key={s}
                onClick={() => onSpeedChange(s)}
                className={`px-2.5 py-0.5 text-xs rounded-md border transition-colors ${
                  buildSpeed === s
                    ? 'border-primary bg-primary/10 text-primary font-medium'
                    : 'border-border bg-bg-secondary text-text-secondary hover:border-text-muted'
                }`}
              >
                {s === '快速模式' ? '🚀' : s === '标准模式' ? '⚡' : '🐌'} {s.replace('模式', '')}
              </button>
            ))}
          </div>
        </div>

        <div className="w-px h-5 bg-border" />

        <div className="flex items-center gap-1.5">
          <span className="text-xs text-text-secondary">范围</span>
          <div className="flex gap-1">
            {['稳妥模式', '严格增量模式'].map(s => (
              <button
                key={s}
                onClick={() => onScopeChange(s)}
                className={`px-2.5 py-0.5 text-xs rounded-md border transition-colors ${
                  buildScopeMode === s
                    ? 'border-primary bg-primary/10 text-primary font-medium'
                    : 'border-border bg-bg-secondary text-text-secondary hover:border-text-muted'
                }`}
              >
                {s === '稳妥模式' ? '🛡 稳妥' : '🎯 严格增量'}
              </button>
            ))}
          </div>
        </div>

        <div className="w-px h-5 bg-border" />

        <label className="flex items-center gap-1.5 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={smartDependency}
            onChange={e => onSmartDepChange(e.target.checked)}
            className="w-3.5 h-3.5 rounded accent-primary cursor-pointer"
          />
          <span className="text-xs text-text-secondary">智能补齐依赖</span>
        </label>
      </div>

      {/* Row 3: 项目根 */}
      <div className="flex items-center gap-2 px-5 py-1.5 border-b border-border-light">
        <span className="text-xs text-text-secondary whitespace-nowrap w-14">项目根</span>
        <div
          className="flex-1 flex items-center gap-2 px-2.5 py-1 rounded-md border border-border bg-bg-card cursor-pointer hover:border-primary transition-colors truncate"
          onClick={onChangeRoot}
        >
          <span className="truncate text-xs flex-1">{projectRoot}</span>
          <span className="text-xs text-text-muted whitespace-nowrap">点击切换 ▸</span>
        </div>
      </div>

      {/* Row 4: 输出目录 */}
      <div className="flex items-center gap-2 px-5 py-1.5 bg-bg-secondary/50">
        <span className="text-xs text-text-secondary whitespace-nowrap w-14">输出</span>
        <div
          className="flex-1 flex items-center gap-2 px-2.5 py-1 rounded-md border border-border bg-bg-card cursor-pointer hover:border-primary transition-colors truncate"
          onClick={onPickOutput}
        >
          <span className="truncate text-xs flex-1">{outputDir}</span>
          <span className="text-xs text-text-muted whitespace-nowrap">点击切换 ▸</span>
        </div>
      </div>
    </div>
  )
}
