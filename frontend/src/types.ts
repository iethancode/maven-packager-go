// ─── 与 Go 后端类型映射 ──────────────────────────────

export interface Commit {
  hash: string;
  repo?: string;
  author: string;
  date: string;
  msg: string;
}

export interface Config {
  theme: string;
  output_dir: string;
  build_speed: string;
  build_scope_mode: string;
  last_branch: string;
  smart_dependency: boolean;
  project_root: string;
}

export interface InitialStateDTO {
  projectRoot: string;
  config: Config;
  hasGit: boolean;
  hasPom: boolean;
  hasMaven: boolean;
  branches: string[];
  currentBranch: string;
  commits: Commit[];
  moduleCount: number;
  defaultOutputDir: string;
}

export interface SwitchBranchResult {
  success: boolean;
  message: string;
  pullOutput: string;
  branches: string[];
  currentBranch: string;
  commits: Commit[];
}

export interface FileDiff {
  path: string;
  status: string;
  diff: string;
  truncate: boolean;
}

export interface CommitDiff {
  hash: string;
  files: FileDiff[];
}

export interface GraphNode {
  id: string;
  artifactId: string;
  groupId: string;
  version: string;
  packaging: string;
  name: string;
}

export interface GraphEdge {
  from: string;
  to: string;
}

export interface ModuleGraph {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface ImpactDTO {
  changedModules: string[];
  changedFiles: string[];
  autoAddedModules: string[];
  buildPlan: string[];
}

export interface StartPackagingOpts {
  commits: string[];
  speedMode: string;
  scopeMode: string;
  smartDependency: boolean;
  outputDir: string;
}

export interface BuildResult {
  success: boolean;
  builtModules: string[];
  changedModules: string[];
  autoAddedModules: string[];
  collectedJars: string[];
  totalMillis: number;
}

export interface HistoryRecord {
  id: string;
  startedAt: string;
  branch: string;
  success: boolean;
  commits: string[];
  changedModules: string[];
  autoAddedModules: string[];
  builtModules: string[];
  collectedJars: string[];
  elapsedMs: number;
  speedMode: string;
  scopeMode: string;
  outputDir: string;
}

export interface TimingSummary {
  total: number;
  modules: ModuleTiming[];
  bottleneck: string;
  bottleneckMs: number;
}

export interface ModuleTiming {
  module: string;
  elapsedMs: number;
  success: boolean;
}

// ─── 前端事件流类型 ──────────────────────────────────

export type BuildStatus = 'idle' | 'building' | 'done'

export interface LogEvent {
  line: string;
  level: string;
  ts: number;
}

export interface ProgressEvent {
  value: number;
  text: string;
}

export interface ModuleDoneEvent {
  module: string;
  elapsedMs: number;
  success: boolean;
}

export interface CompleteEvent {
  success: boolean;
  message: string;
  result?: BuildResult;
}
