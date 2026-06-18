import { useState, useMemo, useCallback } from 'react'
import type { Commit } from '../types'

interface CommitListProps {
  commits: Commit[]
  loading?: boolean
  selectedHashes: string[]
  onSelectCommit: (hash: string, checked: boolean) => void
  onSelectAll: (checked: boolean, pageCommits: Commit[]) => void
  onOpenDiff: (hash: string) => void
}

const PAGE_SIZE = 20

const shortHash = (hash: string) => {
  const parts = hash.split('::')
  return parts[parts.length - 1] || hash
}

export default function CommitList({
  commits, loading = false, selectedHashes, onSelectCommit, onSelectAll, onOpenDiff,
}: CommitListProps) {
  const [search, setSearch] = useState('')
  const [dateStart, setDateStart] = useState('')
  const [dateEnd, setDateEnd] = useState('')
  const [page, setPage] = useState(0)

  // 过滤
  const filtered = useMemo(() => {
    return commits.filter(c => {
      if (search) {
        const s = search.toLowerCase()
        if (
          !c.msg.toLowerCase().includes(s) &&
          !c.author.toLowerCase().includes(s) &&
          !c.hash.toLowerCase().includes(s)
        ) return false
      }
      if (dateStart && c.date < dateStart) return false
      if (dateEnd && c.date > dateEnd) return false
      return true
    })
  }, [commits, search, dateStart, dateEnd])

  // 分页
  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const safePage = Math.min(page, totalPages - 1)
  const start = safePage * PAGE_SIZE
  const pageCommits = filtered.slice(start, start + PAGE_SIZE)

  // 全选状态
  const pageSelectedCount = pageCommits.filter(c => selectedHashes.includes(c.hash)).length
  const allChecked = pageCommits.length > 0 && pageSelectedCount === pageCommits.length
  const someChecked = pageSelectedCount > 0 && pageSelectedCount < pageCommits.length

  const goTo = useCallback((p: number) => {
    setPage(Math.max(0, Math.min(p, totalPages - 1)))
  }, [totalPages])

  const displayStart = start + 1
  const displayEnd = Math.min(start + PAGE_SIZE, filtered.length)

  return (
    <div className="flex flex-col min-h-0">
      {/* Search bar */}
      <div className="flex items-center gap-2 px-4 py-2.5 bg-bg-card border-b border-border">
        <div className="relative flex-1 max-w-sm">
          <span className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted text-[10px]">🔍</span>
          <input
            type="text"
            placeholder="搜索信息、作者、哈希..."
            value={search}
            onChange={e => { setSearch(e.target.value); setPage(0) }}
            className="w-full pl-7 pr-3 h-7 text-sm rounded-lg border border-border bg-bg-secondary text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-1 focus:ring-primary focus:border-primary transition-colors"
          />
        </div>
        <div className="flex items-center gap-1.5">
          <input
            type="datetime-local"
            value={dateStart}
            onChange={e => { setDateStart(e.target.value.replace('T', ' ')); setPage(0) }}
            className="h-7 px-2 text-xs rounded-lg border border-border bg-bg-secondary text-text-primary focus:outline-none focus:ring-1 focus:ring-primary transition-colors"
          />
          <span className="text-text-muted text-xs">至</span>
          <input
            type="datetime-local"
            value={dateEnd}
            onChange={e => { setDateEnd(e.target.value.replace('T', ' ')); setPage(0) }}
            className="h-7 px-2 text-xs rounded-lg border border-border bg-bg-secondary text-text-primary focus:outline-none focus:ring-1 focus:ring-primary transition-colors"
          />
        </div>
        {(search || dateStart || dateEnd) && (
          <button
            onClick={() => { setSearch(''); setDateStart(''); setDateEnd(''); setPage(0) }}
            className="text-xs text-text-muted hover:text-text-primary px-2 py-1 rounded-md hover:bg-bg-surface transition-colors"
          >
            ✕ 清除
          </button>
        )}
        <div className="flex-1" />
        <span className="text-xs text-text-muted bg-bg-surface px-2 py-0.5 rounded-md">
          共 {filtered.length} 条
        </span>
      </div>

      {/* Header */}
      <div className="grid grid-cols-12 gap-0 px-4 py-2 bg-bg-surface/80 text-[11px] font-medium text-text-secondary border-b border-border uppercase tracking-wider">
        <div className="col-span-1 flex items-center">
          <input
            type="checkbox"
            checked={allChecked}
            ref={el => { if (el) el.indeterminate = someChecked }}
            onChange={e => onSelectAll(e.target.checked, pageCommits)}
            className="w-3.5 h-3.5 rounded cursor-pointer accent-primary"
          />
          <span className="ml-1.5 text-[10px]">版本</span>
        </div>
        <div className="col-span-2">作者</div>
        <div className="col-span-2">日期</div>
        <div className="col-span-6">信息</div>
        <div className="col-span-1 text-center">Diff</div>
      </div>

      {/* Rows */}
      <div className="flex-1 overflow-y-auto">
        {pageCommits.length === 0 && loading && (
          <div className="divide-y divide-border-light">
            {Array.from({ length: 10 }).map((_, i) => (
              <div key={i} className="grid grid-cols-12 gap-0 px-4 py-2 text-sm border-b border-border-light">
                <div className="col-span-1 flex items-center gap-2">
                  <span className="w-3.5 h-3.5 rounded bg-bg-surface animate-pulse" />
                  <span className="w-14 h-5 rounded bg-bg-surface animate-pulse" />
                </div>
                <div className="col-span-2 flex items-center gap-2">
                  <span className="w-5 h-5 rounded-full bg-bg-surface animate-pulse" />
                  <span className="w-20 h-4 rounded bg-bg-surface animate-pulse" />
                </div>
                <div className="col-span-2 flex items-center">
                  <span className="w-32 h-4 rounded bg-bg-surface animate-pulse" />
                </div>
                <div className="col-span-6 flex items-center">
                  <span className="w-3/4 h-4 rounded bg-bg-surface animate-pulse" />
                </div>
                <div className="col-span-1 flex justify-center items-center">
                  <span className="w-7 h-7 rounded-md bg-bg-surface animate-pulse" />
                </div>
              </div>
            ))}
            <div className="px-4 py-3 text-xs text-text-muted bg-bg-card">
              正在加载提交记录，已优先读取最新仓库，列表会自动刷新
            </div>
          </div>
        )}
        {pageCommits.length === 0 && !loading && (
          <div className="flex items-center justify-center h-full min-h-[220px] text-sm text-text-muted">
            暂无提交记录
          </div>
        )}
        {pageCommits.map((c) => {
          const isSelected = selectedHashes.includes(c.hash)
          return (
            <div
              key={c.hash}
              className={`grid grid-cols-12 gap-0 px-4 py-2 text-sm border-b border-border-light transition-colors cursor-default ${
                isSelected
                  ? 'bg-primary/8 border-l-2 border-l-primary'
                  : 'hover:bg-bg-surface/50'
              }`}
            >
              <div className="col-span-1 flex items-center gap-2 min-w-0">
                <input
                  type="checkbox"
                  checked={isSelected}
                  onChange={e => onSelectCommit(c.hash, e.target.checked)}
                  className="w-3.5 h-3.5 rounded cursor-pointer accent-primary"
                />
                <code
                  className="text-[11px] font-mono text-primary font-semibold bg-primary/5 px-1 py-0.5 rounded truncate"
                  title={c.repo ? `${c.repo} / ${shortHash(c.hash)}` : c.hash}
                >
                  {shortHash(c.hash)}
                </code>
              </div>
              <div className="col-span-2 text-text-secondary truncate flex items-center">
                <span className="w-5 h-5 rounded-full bg-bg-surface flex items-center justify-center text-[10px] mr-2 flex-shrink-0">
                  {c.author[0]}
                </span>
                <span className="truncate">{c.author}</span>
              </div>
              <div className="col-span-2 text-xs font-mono text-text-muted truncate">{c.date}</div>
              <div className="col-span-6 truncate text-sm leading-tight text-text-primary" title={c.repo || undefined}>
                {c.repo && <span className="mr-1.5 text-[11px] text-text-muted">[{c.repo}]</span>}
                {c.msg}
              </div>
              <div className="col-span-1 flex justify-center items-center">
                <button
                  onClick={() => onOpenDiff(c.hash)}
                  className="inline-flex items-center justify-center w-7 h-7 rounded-md text-text-muted hover:text-primary hover:bg-primary/10 transition-colors"
                  title="查看差异"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 3v18"/><path d="M18 9l-6 6"/><path d="M6 9l6 6"/><rect x="3" y="3" width="18" height="18" rx="2"/></svg>
                </button>
              </div>
            </div>
          )
        })}
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between px-4 py-2 bg-bg-card border-t border-border text-xs text-text-muted">
        <span>显示 {displayStart}-{displayEnd} / 共 {filtered.length}</span>
        <div className="flex items-center gap-1">
          <button disabled={safePage === 0} onClick={() => goTo(0)}
            className="px-2 py-1 rounded-md hover:bg-bg-surface disabled:opacity-30 transition-colors text-xs">{'<<'}</button>
          <button disabled={safePage === 0} onClick={() => goTo(safePage - 1)}
            className="px-2 py-1 rounded-md hover:bg-bg-surface disabled:opacity-30 transition-colors text-xs">{'<'}</button>
          <span className="px-2 py-1 bg-bg-surface rounded-md font-medium">{safePage + 1}/{totalPages}</span>
          <button disabled={safePage >= totalPages - 1} onClick={() => goTo(safePage + 1)}
            className="px-2 py-1 rounded-md hover:bg-bg-surface disabled:opacity-30 transition-colors text-xs">{'>'}</button>
          <button disabled={safePage >= totalPages - 1} onClick={() => goTo(totalPages - 1)}
            className="px-2 py-1 rounded-md hover:bg-bg-surface disabled:opacity-30 transition-colors text-xs">{'>>'}</button>
        </div>
      </div>
    </div>
  )
}
