// Wails bindings — 使用 Wails 自动生成的 ES modules
import * as runtime from '../wailsjs/runtime/runtime'
import * as AppAPI from '../wailsjs/go/main/App'
import type {
  Commit, Config, InitialStateDTO, SwitchBranchResult,
  FileDiff, CommitDiff, ImpactDTO,
  StartPackagingOpts,
} from './types'

// ─── 事件订阅 ──────────────────────────────────────

export function onEvent(name: string, cb: (...args: any[]) => void) {
  runtime.EventsOn(name, cb)
}

export function offEvent(name: string) {
  runtime.EventsOff(name)
}

// ─── API 封装 ─────────────────────────────────────

export const api = {
  getInitialState: () => AppAPI.GetInitialState() as Promise<InitialStateDTO>,
  pickDirectory: (title: string, defaultDir: string) => {
    if (!AppAPI.PickDirectory) throw new Error('Wails runtime not ready')
    return AppAPI.PickDirectory(title, defaultDir) as Promise<string>
  },
  changeProjectRoot: (newRoot: string) => AppAPI.ChangeProjectRoot(newRoot) as Promise<InitialStateDTO>,
  refreshProject: () => AppAPI.RefreshProject() as Promise<InitialStateDTO>,
  countModules: () => AppAPI.CountModules() as Promise<number>,
  listBranches: () => AppAPI.ListBranches() as Promise<string[]>,
  currentBranch: () => AppAPI.CurrentBranch() as Promise<string>,
  switchBranch: (branch: string) => AppAPI.SwitchBranch(branch) as Promise<SwitchBranchResult>,
  listCommits: (limit: number) => AppAPI.ListCommits(limit) as Promise<Commit[]>,
  startLoadCommits: (limit: number) => AppAPI.StartLoadCommits(limit) as Promise<void>,
  getChangedFiles: (hash: string) => AppAPI.GetChangedFiles(hash) as Promise<string[]>,
  getCommitDiff: (hash: string) => AppAPI.GetCommitDiff(hash) as Promise<CommitDiff>,
  getFileDiff: (hash: string, path: string) => AppAPI.GetFileDiff(hash, path) as Promise<FileDiff>,
  resolveImpactedModules: (commits: string[], smartDependency: boolean) =>
    AppAPI.ResolveImpactedModules(commits, smartDependency) as Promise<ImpactDTO>,
  startPackaging: (opts: StartPackagingOpts) => AppAPI.StartPackaging(opts) as Promise<void>,
  cancelPackaging: () => AppAPI.CancelPackaging() as Promise<boolean>,
  getConfig: () => AppAPI.GetConfig() as Promise<Config>,
  patchConfig: (patch: Record<string, any>) => AppAPI.PatchConfig(patch) as Promise<Config>,
  hasMaven: () => AppAPI.HasMaven() as Promise<boolean>,
  validateOutputDir: (dir: string) => AppAPI.ValidateOutputDir(dir) as Promise<void>,
}

// ─── Dev 模式检测 ──────────────────────────────────

export function isDevMode(): boolean {
  return !AppAPI.GetInitialState
}
