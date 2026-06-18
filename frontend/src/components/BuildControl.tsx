interface BuildControlProps {
  selectedCount: number
  status: string
  progress: number
  progressText: string
  outputDir: string
  onStart: () => void
  onCancel: () => void
}

export default function BuildControl({
  selectedCount, status, progress, progressText, outputDir,
  onStart, onCancel,
}: BuildControlProps) {
  const isBuilding = status === 'building'

  return (
    <div className="border-t border-border bg-bg-card px-5 py-2">
      {/* Progress bar */}
      {isBuilding && (
        <div className="mb-2">
          <div className="flex justify-between text-xs text-text-muted mb-1">
            <span>{progressText}</span>
            <span>{Math.round(progress * 100)}%</span>
          </div>
          <div className="w-full h-1.5 bg-bg-surface rounded-full overflow-hidden">
            <div
              className="h-full rounded-full transition-all duration-300 bg-primary"
              style={{ width: `${Math.round(progress * 100)}%` }}
            />
          </div>
        </div>
      )}

      <div className="flex items-center gap-3">
        {/* Status indicator */}
        <div className="flex items-center gap-1.5 text-xs">
          {isBuilding ? (
            <>
              <div className="w-2 h-2 rounded-full bg-warning animate-pulse" />
              <span className="text-warning">构建中</span>
            </>
          ) : (
            <>
              <div className="w-2 h-2 rounded-full bg-success" />
              <span className="text-text-secondary">就绪</span>
            </>
          )}
        </div>

        {selectedCount > 0 && !isBuilding && (
          <span className="text-xs text-text-muted">
            已选 {selectedCount} 个提交 → <span className="text-text-secondary">{outputDir.split(/[\\/]/).pop()}</span>
          </span>
        )}

        <div className="flex-1" />

        {/* Action button */}
        {isBuilding ? (
          <button
            onClick={onCancel}
            className="px-5 h-7 rounded-lg text-xs font-medium bg-error text-white hover:bg-red-600 transition-colors"
          >
            取消构建
          </button>
        ) : (
          <button
            onClick={onStart}
            disabled={selectedCount === 0}
            className="px-5 h-7 rounded-lg text-xs font-medium bg-primary text-white hover:bg-primary-hover transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            开始打包 {selectedCount > 0 ? `(${selectedCount})` : '(请先选择提交)'}
          </button>
        )}
      </div>
    </div>
  )
}
