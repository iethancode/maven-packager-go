import { useState, useEffect, useCallback } from 'react'
import { api } from '../wails'
import type { CommitDiff } from '../types'

interface DiffPanelProps {
  hash: string
  onClose: () => void
}

export default function DiffPanel({ hash, onClose }: DiffPanelProps) {
  const [diff, setDiff] = useState<CommitDiff | null>(null)
  const [loading, setLoading] = useState(true)
  const [selectedFile, setSelectedFile] = useState(0)

  useEffect(() => {
    let cancelled = false
    api.getCommitDiff(hash).then(d => {
      if (!cancelled) { setDiff(d); setLoading(false) }
    }).catch(() => {
      if (!cancelled) { setLoading(false) }
    })
    return () => { cancelled = true }
  }, [hash])

  // ESC 关闭
  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [onClose])

  const renderDiffLine = (line: string, i: number) => {
    if (line.startsWith('@@')) {
      return <div key={i} className="diff-hunk bg-bg-surface/50 px-2">{line}</div>
    }
    if (line.startsWith('---') || line.startsWith('+++')) {
      return <div key={i} className="diff-header px-2">{line}</div>
    }
    if (line.startsWith('+')) {
      return <div key={i} className="diff-added px-2">{line}</div>
    }
    if (line.startsWith('-')) {
      return <div key={i} className="diff-removed px-2">{line}</div>
    }
    if (line.startsWith('diff ')) {
      return <div key={i} className="text-text-muted font-bold px-2">{line}</div>
    }
    if (line.startsWith('index ') || line.startsWith('new file') || line.startsWith('deleted')) {
      return <div key={i} className="text-text-muted px-2 text-xs">{line}</div>
    }
    return <div key={i} className="px-2 text-text-secondary">{line}</div>
  }

  const statusIcon = (status: string) => {
    switch (status[0]) {
      case 'A': return <span className="text-success text-xs font-bold">ADD</span>
      case 'D': return <span className="text-error text-xs font-bold">DEL</span>
      case 'M': return <span className="text-warning text-xs font-bold">MOD</span>
      case 'R': return <span className="text-info text-xs font-bold">REN</span>
      default: return <span className="text-text-muted text-xs">{status}</span>
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 animate-fade-in" onClick={onClose}>
      <div
        className="w-[90vw] h-[85vh] bg-bg-card rounded-xl shadow-2xl border border-border flex flex-col animate-slide-in"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-3 border-b border-border">
          <h3 className="font-semibold text-sm">
            提交差异 <code className="ml-2 text-primary font-mono">{hash}</code>
          </h3>
          <button onClick={onClose} className="text-text-muted hover:text-text-primary text-lg leading-none px-2">
            ×
          </button>
        </div>

        <div className="flex-1 flex min-h-0">
          {/* File list */}
          <div className="w-56 border-r border-border overflow-y-auto bg-bg-secondary flex-shrink-0">
            {loading ? (
              <div className="p-4 text-sm text-text-muted">加载中...</div>
            ) : diff?.files ? (
              diff.files.map((f, i) => (
                <button
                  key={f.path}
                  onClick={() => setSelectedFile(i)}
                  className={`w-full text-left px-3 py-1.5 text-xs border-b border-border-light flex items-center gap-2 transition-colors ${
                    i === selectedFile ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-bg-surface'
                  }`}
                >
                  {statusIcon(f.status)}
                  <span className="truncate" title={f.path}>{f.path}</span>
                </button>
              ))
            ) : (
              <div className="p-4 text-sm text-text-muted">无文件变更</div>
            )}
          </div>

          {/* Diff content */}
          <div className="flex-1 overflow-auto bg-bg-card">
            {loading ? (
              <div className="flex items-center justify-center h-full text-text-muted">加载中...</div>
            ) : diff?.files?.[selectedFile] ? (
              <div className="font-mono text-xs leading-relaxed whitespace-pre">
                {diff.files[selectedFile].diff.split('\n').map(renderDiffLine)}
                {diff.files[selectedFile].truncate && (
                  <div className="text-warning px-2 italic">... diff 已截断 ...</div>
                )}
              </div>
            ) : (
              <div className="flex items-center justify-center h-full text-text-muted">选择文件查看差异</div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="px-5 py-2 border-t border-border text-xs text-text-muted flex justify-between">
          <span>{diff?.files?.length || 0} 个文件变更</span>
          <span>ESC 关闭</span>
        </div>
      </div>
    </div>
  )
}
