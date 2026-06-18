package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"maven-packager-go/internal/config"
	"maven-packager-go/internal/git"
	"maven-packager-go/internal/maven"
	"maven-packager-go/internal/project"
	"maven-packager-go/internal/timing"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	config *config.Manager

	rootMu  sync.RWMutex
	rootDir string
	gitCli  *git.Workspace
	mvn     *maven.Handler
	scan    project.ScanResult

	timing *timing.Collector

	buildMu      sync.Mutex
	buildCancel  context.CancelFunc
	buildRunning bool
}

func NewApp() *App {
	return &App{
		config: config.NewManager(),
		timing: timing.NewCollector(),
	}
}

// 鈹€鈹€鈹€ 鐢熷懡鍛ㄦ湡 鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€

func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	a.ensureRootConfigured()
}

func (a *App) ensureRootConfigured() {
	cfg := a.config.Reload()
	wd, _ := os.Getwd()
	candidates := []string{project.AppBaseDir(), wd, cfg.ProjectRoot}
	if cfg.ProjectRoot != "" && fileExists(a.config.Path()) {
		candidates = []string{cfg.ProjectRoot, project.AppBaseDir(), wd}
	}

	detected := firstValidProjectRoot(candidates)
	if detected == "" {
		detected = project.FindProjectRoot(candidates)
	}
	if !project.IsValidProjectRoot(detected) {
		detected = ""
	}
	current, _, _ := a.snapshot()
	if samePath(current, detected) {
		if cfg.ProjectRoot != detected {
			a.config.Patch(map[string]any{"project_root": detected})
		}
		return
	}
	a.setRootDir(detected)
	if cfg.ProjectRoot != detected {
		a.config.Patch(map[string]any{"project_root": detected})
	}
}

func (a *App) OnShutdown(_ context.Context) {
	a.cancelBuildIfRunning()
}

// 鈹€鈹€鈹€ 鍐呴儴宸ュ叿 鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€

func (a *App) setRootDir(root string) {
	a.rootMu.Lock()
	defer a.rootMu.Unlock()
	a.rootDir = root
	if root == "" {
		a.gitCli = nil
		a.mvn = nil
		a.scan = project.ScanResult{}
		return
	}
	// 一次扫描同时得到子仓列表、pom 存在性与模块数，避免首屏多次全树 walk。
	scan := project.ScanWorkspace(root)
	a.scan = scan
	a.gitCli = git.NewWorkspaceFromScan(root, scan.ChildGitDirs, scan.HasRootGit)
	a.mvn = maven.NewHandler(root)
}

func (a *App) snapshot() (string, *git.Workspace, *maven.Handler) {
	a.rootMu.RLock()
	defer a.rootMu.RUnlock()
	return a.rootDir, a.gitCli, a.mvn
}

// scanSnapshot 返回缓存的扫描摘要（hasPom / moduleCount 等），供首屏读取，
// 避免每次都重新全树遍历。
func (a *App) scanSnapshot() project.ScanResult {
	a.rootMu.RLock()
	defer a.rootMu.RUnlock()
	return a.scan
}

func firstValidProjectRoot(candidates []string) string {
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		key := strings.ToLower(abs)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if project.IsValidProjectRoot(abs) {
			return abs
		}
	}
	return ""
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return a == b
	}
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return strings.EqualFold(filepath.Clean(aa), filepath.Clean(bb))
}

// 鈹€鈹€鈹€ 鍚戝墠绔毚闇茬殑鏂规硶 鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€

type InitialStateDTO struct {
	ProjectRoot      string        `json:"projectRoot"`
	Config           config.Config `json:"config"`
	HasGit           bool          `json:"hasGit"`
	HasPom           bool          `json:"hasPom"`
	HasMaven         bool          `json:"hasMaven"`
	Branches         []string      `json:"branches"`
	CurrentBranch    string        `json:"currentBranch"`
	Commits          []git.Commit  `json:"commits"`
	ModuleCount      int           `json:"moduleCount"`
	DefaultOutputDir string        `json:"defaultOutputDir"`
}

func (a *App) GetInitialState() InitialStateDTO {
	a.ensureRootConfigured()
	root, gc, _ := a.snapshot()
	scan := a.scanSnapshot()
	cfg := a.config.Get()

	hasGit := gc != nil && gc.HasRepositories()
	hasPom := root != "" && (scan.HasRootPom || scan.HasChildPom)
	var (
		branches []string
		current  string
		commits  []git.Commit
	)

	moduleCount := 0

	defOut := cfg.OutputDir
	if defOut == "" && root != "" {
		defOut = filepath.Join(root, "dist_jars")
	}

	return InitialStateDTO{
		ProjectRoot:      root,
		Config:           cfg,
		HasGit:           hasGit,
		HasPom:           hasPom,
		HasMaven:         maven.HasMaven(),
		Branches:         branches,
		CurrentBranch:    current,
		Commits:          commits,
		ModuleCount:      moduleCount,
		DefaultOutputDir: defOut,
	}
}

func (a *App) PickDirectory(title, defaultDir string) (string, error) {
	// Ensure default dir exists so WebView2 doesn't reject it
	if defaultDir != "" {
		os.MkdirAll(defaultDir, 0o755)
	}
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:            title,
		DefaultDirectory: defaultDir,
	})
}

func (a *App) ChangeProjectRoot(newRoot string) (InitialStateDTO, error) {
	if newRoot == "" {
		return InitialStateDTO{}, fmt.Errorf("path is empty")
	}
	abs, err := filepath.Abs(newRoot)
	if err != nil {
		return InitialStateDTO{}, err
	}
	if !project.IsValidProjectRoot(abs) {
		return InitialStateDTO{}, fmt.Errorf("鎵€閫夌洰褰曚笅鏈壘鍒?.git 鎴?pom.xml")
	}
	a.setRootDir(abs)
	a.config.Patch(map[string]any{"project_root": abs})
	return a.GetInitialState(), nil
}

func (a *App) RefreshProject() InitialStateDTO {
	root, _, _ := a.snapshot()
	if root != "" {
		// 重新扫描，刷新 hasPom / moduleCount 缓存与子仓列表。
		a.setRootDir(root)
	}
	_, _, mvn := a.snapshot()
	if mvn != nil {
		mvn.Reload()
	}
	return a.GetInitialState()
}

func (a *App) CountModules() int {
	root, _, _ := a.snapshot()
	if root == "" {
		return 0
	}
	return a.scanSnapshot().ModuleCount
}

// 鈹€鈹€鈹€ Git 鎿嶄綔 鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€

func (a *App) ListBranches() []string {
	_, gc, _ := a.snapshot()
	if gc == nil {
		return []string{}
	}
	return gc.Branches()
}

func (a *App) CurrentBranch() string {
	_, gc, _ := a.snapshot()
	if gc == nil {
		return ""
	}
	return gc.CurrentBranch()
}

type SwitchBranchResult struct {
	Success       bool         `json:"success"`
	Message       string       `json:"message"`
	PullOutput    string       `json:"pullOutput"`
	Branches      []string     `json:"branches"`
	CurrentBranch string       `json:"currentBranch"`
	Commits       []git.Commit `json:"commits"`
}

func (a *App) SwitchBranch(branch string) SwitchBranchResult {
	_, gc, _ := a.snapshot()
	if gc == nil {
		return SwitchBranchResult{Success: false, Message: "椤圭洰鏈垵濮嬪寲"}
	}
	dirty := gc.StatusPorcelain()
	if strings.TrimSpace(dirty) != "" {
		a.emitLog("工作区有未提交的更改，切换分支可能失败", "warning")
	}
	if err := gc.Checkout(branch); err != nil {
		return SwitchBranchResult{Success: false, Message: err.Error()}
	}
	var cur string
	for i := 0; i < 10; i++ {
		cur = gc.CurrentBranch()
		if cur == branch || strings.HasPrefix(cur, branch+" ") {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if cur != branch && !strings.HasPrefix(cur, branch+" ") {
		return SwitchBranchResult{Success: false, Message: "鍒囨崲鍚庡垎鏀湭鐢熸晥"}
	}
	a.config.Patch(map[string]any{"last_branch": branch})

	pullOut, _ := gc.Pull(branch)
	return SwitchBranchResult{
		Success:       true,
		Message:       "宸插垏鎹㈠埌 " + branch,
		PullOutput:    pullOut,
		Branches:      []string{},
		CurrentBranch: cur,
		Commits:       []git.Commit{},
	}
}

func (a *App) ListCommits(limit int) []git.Commit {
	_, gc, _ := a.snapshot()
	if gc == nil {
		return []git.Commit{}
	}
	commits, _ := gc.Log(limit)
	if commits == nil {
		return []git.Commit{}
	}
	return commits
}

func (a *App) StartLoadCommits(limit int) {
	_, gc, _ := a.snapshot()
	if gc == nil {
		a.emitEvent("commits:complete", "{}")
		return
	}
	if limit <= 0 {
		limit = 300
	}
	go func() {
		total := 0
		gc.LogBatches(limit, func(batch []git.Commit) {
			total += len(batch)
			payload, _ := json.Marshal(map[string]any{"commits": batch})
			a.emitEvent("commits:batch", string(payload))
		})
		payload, _ := json.Marshal(map[string]any{"total": total})
		a.emitEvent("commits:complete", string(payload))
	}()
}

func (a *App) GetChangedFiles(hash string) []string {
	_, gc, _ := a.snapshot()
	if gc == nil {
		return []string{}
	}
	files, _ := gc.ChangedFiles(hash)
	if files == nil {
		return []string{}
	}
	return files
}

func (a *App) GetCommitDiff(hash string) (git.CommitDiff, error) {
	_, gc, _ := a.snapshot()
	if gc == nil {
		return git.CommitDiff{}, fmt.Errorf("git 鏈垵濮嬪寲")
	}
	return gc.GetCommitDiff(hash, 512*1024)
}

func (a *App) GetFileDiff(hash, path string) (git.FileDiff, error) {
	_, gc, _ := a.snapshot()
	if gc == nil {
		return git.FileDiff{}, fmt.Errorf("git 鏈垵濮嬪寲")
	}
	return gc.GetFileDiff(hash, path, 1024*1024)
}

// 鈹€鈹€鈹€ Maven 鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€

func (a *App) GetModuleGraph() maven.ModuleGraph {
	_, _, mvn := a.snapshot()
	if mvn == nil {
		return maven.ModuleGraph{}
	}
	return mvn.Graph()
}

type ImpactDTO struct {
	ChangedModules   []string `json:"changedModules"`
	ChangedFiles     []string `json:"changedFiles"`
	AutoAddedModules []string `json:"autoAddedModules"`
	BuildPlan        []string `json:"buildPlan"`
}

func (a *App) ResolveImpactedModules(commits []string, smartDependency bool) ImpactDTO {
	root, gc, mvn := a.snapshot()
	_ = root
	var dto ImpactDTO
	if gc == nil || mvn == nil {
		return dto
	}
	changedFiles := map[string]struct{}{}
	for _, c := range commits {
		files, _ := gc.ChangedFiles(c)
		for _, f := range files {
			changedFiles[f] = struct{}{}
		}
	}
	filesList := make([]string, 0, len(changedFiles))
	for f := range changedFiles {
		filesList = append(filesList, f)
	}
	dto.ChangedFiles = filesList

	moduleSet := map[string]struct{}{}
	for f := range changedFiles {
		if m := mvn.FindModuleByFile(f); m != "" {
			moduleSet[m] = struct{}{}
		}
	}
	changed := make([]string, 0, len(moduleSet))
	for m := range moduleSet {
		changed = append(changed, m)
	}
	dto.ChangedModules = changed

	if smartDependency {
		plan := mvn.ExpandWithDependencies(changed)
		dto.BuildPlan = plan
		base := map[string]struct{}{}
		for _, m := range changed {
			base[m] = struct{}{}
		}
		for _, m := range plan {
			if _, ok := base[m]; !ok {
				dto.AutoAddedModules = append(dto.AutoAddedModules, m)
			}
		}
	} else {
		dto.BuildPlan = changed
	}
	return dto
}

// 鈹€鈹€鈹€ 鏋勫缓 鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€

type StartPackagingOpts struct {
	Commits         []string `json:"commits"`
	SpeedMode       string   `json:"speedMode"`
	ScopeMode       string   `json:"scopeMode"`
	SmartDependency bool     `json:"smartDependency"`
	OutputDir       string   `json:"outputDir"`
}

func (a *App) StartPackaging(opts StartPackagingOpts) error {
	a.buildMu.Lock()
	if a.buildRunning {
		a.buildMu.Unlock()
		return fmt.Errorf("已有打包任务在执行")
	}
	a.buildRunning = true
	ctx, cancel := context.WithCancel(a.ctx)
	a.buildCancel = cancel
	a.buildMu.Unlock()

	go a.runBuild(ctx, opts)
	return nil
}

func (a *App) CancelPackaging() bool {
	return a.cancelBuildIfRunning()
}

func (a *App) cancelBuildIfRunning() bool {
	a.buildMu.Lock()
	defer a.buildMu.Unlock()
	if a.buildCancel != nil {
		a.buildCancel()
		a.buildCancel = nil
		return true
	}
	return false
}

func (a *App) finishBuildState() {
	a.buildMu.Lock()
	a.buildCancel = nil
	a.buildRunning = false
	a.buildMu.Unlock()
}

func (a *App) runBuild(ctx context.Context, opts StartPackagingOpts) {
	defer a.finishBuildState()

	root, gc, mvn := a.snapshot()
	if gc == nil || mvn == nil {
		a.emitComplete(false, "椤圭洰鏈垵濮嬪寲")
		return
	}
	if !maven.HasMaven() {
		a.emitLog("未检测到 mvn 命令，请先安装 Maven 并加入 PATH。", "error")
		a.emitComplete(false, "缂哄皯 Maven")
		return
	}

	a.timing.Reset()
	a.emitLog(strings.Repeat("=", 60), "header")
	a.emitLog("开始增量打包流程", "header")
	a.emitProgress(0.05, "鍒嗘瀽鍙樻洿")

	fileSet := map[string]struct{}{}
	for _, hash := range opts.Commits {
		files, err := gc.ChangedFiles(hash)
		if err != nil {
			a.emitLog("璇诲彇鎻愪氦鍙樻洿澶辫触: "+err.Error(), "error")
			continue
		}
		for _, f := range files {
			fileSet[f] = struct{}{}
		}
	}
	if len(fileSet) == 0 {
		a.emitLog("未找到变更文件", "warning")
		a.emitComplete(false, "无变更文件")
		return
	}
	a.emitLog(fmt.Sprintf("找到 %d 个变更文件", len(fileSet)), "success")
	a.emitProgress(0.15, "瀹氫綅妯″潡")

	moduleSet := map[string]struct{}{}
	for f := range fileSet {
		if m := mvn.FindModuleByFile(f); m != "" {
			moduleSet[m] = struct{}{}
		}
	}
	if len(moduleSet) == 0 {
		a.emitLog("鏈壘鍒板彉鏇存枃浠跺搴旂殑 Maven 妯″潡", "error")
		a.emitComplete(false, "鏃犲彲鏋勫缓妯″潡")
		return
	}
	changedModules := make([]string, 0, len(moduleSet))
	for m := range moduleSet {
		changedModules = append(changedModules, m)
	}
	a.emitLog(fmt.Sprintf("变更模块 %d 个", len(changedModules)), "success")
	for _, m := range changedModules {
		a.emitLog("  鈥?"+m, "info")
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join(root, "dist_jars")
	}
	a.config.Patch(map[string]any{
		"output_dir":       outputDir,
		"build_speed":      opts.SpeedMode,
		"build_scope_mode": opts.ScopeMode,
		"smart_dependency": opts.SmartDependency,
	})

	a.emitProgress(0.25, "鍚姩 Maven")

	payloadStart, _ := json.Marshal(map[string]any{
		"modules":   changedModules,
		"speedMode": opts.SpeedMode,
		"scopeMode": opts.ScopeMode,
		"smart":     opts.SmartDependency,
		"outputDir": outputDir,
	})
	a.emitEvent("build:start", string(payloadStart))

	emitter := &wailsEmitter{app: a}
	result := mvn.BuildWithSmartRetry(ctx, maven.BuildOptions{
		Modules:         changedModules,
		SpeedMode:       opts.SpeedMode,
		ScopeMode:       opts.ScopeMode,
		SmartDependency: opts.SmartDependency,
		OutputDir:       outputDir,
	}, emitter)

	if result.Success {
		a.emitLog(strings.Repeat("=", 60), "header")
		a.emitLog(fmt.Sprintf("完成，已收集 %d 个 JAR", len(result.CollectedJars)), "success")
		if len(result.AutoAddedModules) > 0 {
			a.emitLog(fmt.Sprintf("额外构建 %d 个依赖模块", len(result.AutoAddedModules)), "warning")
		}
		a.emitProgress(1.0, "瀹屾垚")
		a.emitCompletePayload(true, "鎵撳寘鎴愬姛", result)
		return
	}
	a.emitLog("Maven 构建失败，请查看日志。", "error")
	a.emitCompletePayload(false, "鏋勫缓澶辫触", result)
}

func (a *App) GetTimingSnapshot() timing.Summary {
	return a.timing.Snapshot()
}

// 鈹€鈹€鈹€ 閰嶇疆 鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€

func (a *App) GetConfig() config.Config { return a.config.Get() }

func (a *App) PatchConfig(patch map[string]any) config.Config {
	return a.config.Patch(patch)
}

func (a *App) HasMaven() bool { return maven.HasMaven() }

func (a *App) ValidateOutputDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("璺緞涓虹┖")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	probe := filepath.Join(abs, ".write_test.tmp")
	if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
		return err
	}
	_ = os.Remove(probe)
	return nil
}

// 鈹€鈹€鈹€ Emitter锛坵ails 浜嬩欢妗ユ帴锛?鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€

type wailsEmitter struct{ app *App }

func (w *wailsEmitter) EmitLog(line, level string) { w.app.emitLog(line, level) }
func (w *wailsEmitter) EmitProgress(v float64, text string) {
	w.app.emitProgress(v, text)
}
func (w *wailsEmitter) EmitModuleDone(module string, elapsedMs int64, success bool) {
	w.app.timing.Add(module, elapsedMs, success)
	payload, _ := json.Marshal(map[string]any{
		"module":    module,
		"elapsedMs": elapsedMs,
		"success":   success,
	})
	w.app.emitEvent("build:module-done", string(payload))
}

func (a *App) emitLog(line, level string) {
	payload, _ := json.Marshal(map[string]any{"line": line, "level": level, "ts": time.Now().UnixMilli()})
	a.emitEvent("build:log", string(payload))
}

func (a *App) emitProgress(v float64, text string) {
	payload, _ := json.Marshal(map[string]any{"value": v, "text": text})
	a.emitEvent("build:progress", string(payload))
}

func (a *App) emitComplete(success bool, message string) {
	payload, _ := json.Marshal(map[string]any{"success": success, "message": message})
	a.emitEvent("build:complete", string(payload))
}

func (a *App) emitCompletePayload(success bool, message string, result maven.BuildResult) {
	payload, _ := json.Marshal(map[string]any{
		"success": success,
		"message": message,
		"result":  result,
	})
	a.emitEvent("build:complete", string(payload))
}

func (a *App) emitEvent(name, payload string) {
	if a.ctx == nil {
		return
	}
	wailsRuntime.EventsEmit(a.ctx, name, payload)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
