package maven

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"maven-packager-go/internal/procutil"
)

// 速度档位
const (
	SpeedFast     = "快速模式"
	SpeedStandard = "标准模式"
	SpeedCompat   = "兼容模式"
)

// 范围档位
const (
	ScopeSafe   = "稳妥模式"
	ScopeStrict = "严格增量模式"
)

// Emitter 用于向前端推送流式数据。app.go 会注入 Wails 实现。
type Emitter interface {
	EmitLog(line string, level string)
	EmitProgress(value float64, text string)
	EmitModuleDone(module string, elapsedMs int64, success bool)
}

// HasMaven 检查 mvn 是否在 PATH 中。
func HasMaven() bool {
	_, err := exec.LookPath("mvn")
	return err == nil
}

// BuildOptions 一次构建的全部可调参数。
type BuildOptions struct {
	Modules         []string // 变更模块原始列表
	SpeedMode       string
	ScopeMode       string
	SmartDependency bool
	OutputDir       string
	// 如果为空则实时分析 Maven 输出提取模块耗时
	PreResolved []string
}

// BuildResult 构建完成的摘要。
type BuildResult struct {
	Success          bool     `json:"success"`
	BuiltModules     []string `json:"builtModules"`
	ChangedModules   []string `json:"changedModules"`
	AutoAddedModules []string `json:"autoAddedModules"`
	CollectedJars    []string `json:"collectedJars"`
	TotalMillis      int64    `json:"totalMillis"`
	LastOutput       string   `json:"-"`
}

// CommandForReactor 多模块 reactor 命令。
func (h *Handler) CommandForReactor(modules []string, speed string, preferOffline, alsoMake bool) string {
	cpu := runtime.NumCPU()
	moduleList := strings.Join(modules, ",")
	alsoMakeFlag := ""
	if alsoMake {
		alsoMakeFlag = " -am"
	}
	base := fmt.Sprintf("mvn clean package -pl %s%s -DskipTests -Dmaven.javadoc.skip=true", moduleList, alsoMakeFlag)

	switch speed {
	case SpeedFast:
		threads := min(cpu, 8)
		cmd := fmt.Sprintf("%s -T %d --batch-mode -Dmaven.compiler.useIncrementalCompilation=true "+
			"-Ddependency.skip=true -Denforcer.skip=true -Danimal.sniffer.skip=true -Dmaven.test.skip=true",
			base, threads)
		if preferOffline {
			cmd += " -o"
		}
		return cmd
	case SpeedStandard:
		threads := min(cpu, 4)
		return fmt.Sprintf("%s -T %d --batch-mode -Dmaven.compiler.useIncrementalCompilation=true -Dmaven.test.skip=true",
			base, threads)
	default:
		return fmt.Sprintf("%s --batch-mode -Dmaven.compiler.useIncrementalCompilation=false -Dmaven.test.skip=true", base)
	}
}

// CommandForSingleModule 单模块逐个构建时使用。
func (h *Handler) CommandForSingleModule(speed string, install, preferOffline bool) string {
	cpu := runtime.NumCPU()
	goal := "package"
	if install {
		goal = "install"
	}
	base := fmt.Sprintf("mvn clean %s -DskipTests -Dmaven.javadoc.skip=true", goal)

	switch speed {
	case SpeedFast:
		cmd := fmt.Sprintf("%s -T %d --batch-mode -Dmaven.compiler.useIncrementalCompilation=true "+
			"-Ddependency.skip=true -Denforcer.skip=true -Danimal.sniffer.skip=true -Dmaven.test.skip=true",
			base, min(cpu, 4))
		if preferOffline {
			cmd += " -o"
		}
		return cmd
	case SpeedStandard:
		return fmt.Sprintf("%s -T %d --batch-mode -Dmaven.compiler.useIncrementalCompilation=true -Dmaven.test.skip=true",
			base, min(cpu, 2))
	default:
		return fmt.Sprintf("%s --batch-mode -Dmaven.compiler.useIncrementalCompilation=false -Dmaven.test.skip=true", base)
	}
}

var moduleDoneRe = regexp.MustCompile(`^\[INFO\]\s+(\S+)\s+\.+\s+(SUCCESS|FAILURE|SKIPPED)\s+\[\s*([0-9.,]+)\s*(s|min)\s*\]`)

// streamCommand 流式执行 shell 命令。
func (h *Handler) streamCommand(ctx context.Context, command, cwd string, emit Emitter) (int, string, error) {
	shell, flag := shellCommand()
	cmd := exec.CommandContext(ctx, shell, flag, command)
	cmd.Dir = cwd
	procutil.HideWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return -1, "", err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return -1, "", err
	}

	var collected strings.Builder
	reader := bufio.NewReaderSize(stdout, 64*1024)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			collected.WriteString(line)
			level := classifyLine(line)
			if emit != nil {
				emit.EmitLog(strings.TrimRight(line, "\r\n"), level)
			}
			if m := moduleDoneRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil && emit != nil {
				ms := parseDurationToMillis(m[3], m[4])
				success := m[2] == "SUCCESS"
				emit.EmitModuleDone(m[1], ms, success)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}

	waitErr := cmd.Wait()
	code := 0
	if waitErr != nil {
		if ee, ok := waitErr.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			code = -1
		}
	}
	return code, collected.String(), nil
}

func parseDurationToMillis(value, unit string) int64 {
	value = strings.ReplaceAll(value, ",", ".")
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	if unit == "min" {
		return int64(v * 60 * 1000)
	}
	return int64(v * 1000)
}

func classifyLine(line string) string {
	s := strings.TrimSpace(line)
	switch {
	case strings.Contains(s, "[ERROR]"), strings.Contains(s, "BUILD FAILURE"), strings.Contains(s, "FAILURE"):
		return "error"
	case strings.Contains(s, "[WARNING]"):
		return "warning"
	case strings.Contains(s, "BUILD SUCCESS"):
		return "success"
	case strings.HasPrefix(s, "[INFO]"):
		return "info"
	default:
		return "info"
	}
}

// BuildWithSmartRetry 复刻 Python 版的多阶段重试：
// reactor 失败 → 自动补齐缺模块 → 再 reactor → 仍失败 → 逐模块 → 逐模块失败再补齐。
// 最多三轮。
func (h *Handler) BuildWithSmartRetry(ctx context.Context, opts BuildOptions, emit Emitter) BuildResult {
	start := time.Now()
	var res BuildResult
	res.ChangedModules = cloneStrings(opts.Modules)

	buildModules := opts.Modules
	if opts.SmartDependency {
		expanded := h.ExpandWithDependencies(opts.Modules)
		buildModules = expanded
		res.AutoAddedModules = diffSlice(expanded, opts.Modules)
	}

	lastOutput := ""
	var builtModules []string
	rootPom := filepath.Join(h.RootDir, "pom.xml")
	_, rootPomErr := os.Stat(rootPom)
	canReactor := rootPomErr == nil

	for attempt := 1; attempt <= 3; attempt++ {
		if canReactor {
			alsoMake := opts.ScopeMode != ScopeStrict
			cmd := h.CommandForReactor(buildModules, opts.SpeedMode, !opts.SmartDependency, alsoMake)
			emitHeader(emit, fmt.Sprintf("─ Reactor 构建 (尝试 %d) ─", attempt))
			emitInfo(emit, fmt.Sprintf("范围模式: %s, -am: %v", opts.ScopeMode, alsoMake))
			emitInfo(emit, fmt.Sprintf("命令: %s", cmd))

			code, out, err := h.streamCommand(ctx, cmd, h.RootDir, emit)
			lastOutput = out
			if err != nil {
				emitError(emit, "执行命令失败: "+err.Error())
				return res
			}
			if code == 0 {
				builtModules = buildModules
				res.Success = true
				res.BuiltModules = builtModules
				break
			}

			if !opts.SmartDependency {
				res.BuiltModules = buildModules
				break
			}
			if extra := h.ExtractMissingModules(out, buildModules); len(extra) > 0 {
				next := h.ExpandWithDependencies(append(append([]string{}, buildModules...), extra...))
				added := diffSlice(next, buildModules)
				if len(added) > 0 {
					emitWarning(emit, fmt.Sprintf("检测到缺失内部依赖，自动补充 %d 个模块后重试。", len(added)))
					for _, m := range added {
						emitInfo(emit, "  + "+m)
					}
					buildModules = next
					res.AutoAddedModules = diffSlice(next, opts.Modules)
					continue
				}
			}
			emitWarning(emit, "Reactor 构建失败，转为逐模块构建以继续补齐内部依赖。")
		}

		if canReactor && !opts.SmartDependency {
			res.BuiltModules = buildModules
			break
		}

		ordered := h.topologicalSortPublic(buildModules)
		total := len(ordered)
		var seqOutput strings.Builder
		success := true
		for i, m := range ordered {
			modDir := filepath.Join(h.RootDir, m)
			cmd := h.CommandForSingleModule(opts.SpeedMode, true, false)
			emitHeader(emit, fmt.Sprintf("[%d/%d] 构建模块: %s", i+1, total, m))
			emitInfo(emit, "命令: "+cmd)

			code, out, err := h.streamCommand(ctx, cmd, modDir, emit)
			seqOutput.WriteString(out)
			if err != nil {
				emitError(emit, "执行命令失败: "+err.Error())
				success = false
				break
			}
			if emit != nil {
				emit.EmitProgress(0.35+float64(i+1)/float64(max(total, 1))*0.45, fmt.Sprintf("已构建 %d/%d", i+1, total))
			}
			if code != 0 {
				emitError(emit, "✗ 模块构建失败: "+m)
				success = false
				break
			}
		}
		lastOutput = seqOutput.String()
		if success {
			builtModules = ordered
			res.Success = true
			res.BuiltModules = builtModules
			break
		}
		if !opts.SmartDependency {
			res.BuiltModules = ordered
			break
		}

		extra := h.ExtractMissingModules(lastOutput, ordered)
		if len(extra) == 0 {
			res.BuiltModules = ordered
			break
		}
		next := h.ExpandWithDependencies(append(append([]string{}, ordered...), extra...))
		buildModules = next
		res.AutoAddedModules = diffSlice(next, opts.Modules)
		emitWarning(emit, fmt.Sprintf("第 %d 次补齐后仍有缺口，继续自动重试。", attempt))
	}

	if res.Success {
		res.CollectedJars = h.collectChangedJars(opts.OutputDir, res.ChangedModules, emit)
	}
	res.TotalMillis = time.Since(start).Milliseconds()
	res.LastOutput = lastOutput
	return res
}

// topologicalSortPublic 提供给 builder 内部的排序封装。
func (h *Handler) topologicalSortPublic(paths []string) []string {
	h.EnsureLoaded()
	set := map[string]struct{}{}
	for _, p := range paths {
		if np := NormalizeModulePath(p); np != "" {
			set[np] = struct{}{}
		}
	}
	return h.topologicalSort(set)
}

// collectChangedJars 把变更模块的主 jar 拷贝到 outputDir。
func (h *Handler) collectChangedJars(outputDir string, changed []string, emit Emitter) []string {
	if outputDir == "" {
		return nil
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		emitError(emit, "无法创建输出目录: "+err.Error())
		return nil
	}
	var collected []string
	for _, m := range changed {
		targetDir := filepath.Join(h.RootDir, m, "target")
		stat, err := os.Stat(targetDir)
		if err != nil || !stat.IsDir() {
			emitWarning(emit, "变更模块未找到 target 目录，跳过: "+m)
			continue
		}
		entries, _ := os.ReadDir(targetDir)
		for _, ent := range entries {
			if ent.IsDir() {
				continue
			}
			name := ent.Name()
			if !strings.HasSuffix(name, ".jar") {
				continue
			}
			if strings.HasSuffix(name, "-sources.jar") || strings.HasSuffix(name, "-javadoc.jar") {
				continue
			}
			src := filepath.Join(targetDir, name)
			dst, renamed := resolveJarTarget(outputDir, m, name)
			if err := copyFile(src, dst); err != nil {
				emitError(emit, "复制 JAR 失败: "+err.Error())
				continue
			}
			collected = append(collected, dst)
			if renamed {
				emitWarning(emit, fmt.Sprintf("JAR 重名，已重命名: %s → %s", name, filepath.Base(dst)))
			} else {
				emitSuccess(emit, fmt.Sprintf("已复制: %s → %s", m, name))
			}
		}
	}
	return collected
}

func resolveJarTarget(outputDir, module, jarName string) (string, bool) {
	return filepath.Join(outputDir, jarName), false
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if info, err := os.Stat(src); err == nil {
		_ = os.Chtimes(dst, info.ModTime(), info.ModTime())
	}
	return nil
}

func shellCommand() (string, string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", "/c"
	}
	return "/bin/sh", "-c"
}

func emitHeader(e Emitter, s string) {
	if e != nil {
		e.EmitLog(s, "header")
	}
}
func emitInfo(e Emitter, s string) {
	if e != nil {
		e.EmitLog(s, "info")
	}
}
func emitWarning(e Emitter, s string) {
	if e != nil {
		e.EmitLog(s, "warning")
	}
}
func emitError(e Emitter, s string) {
	if e != nil {
		e.EmitLog(s, "error")
	}
}
func emitSuccess(e Emitter, s string) {
	if e != nil {
		e.EmitLog(s, "success")
	}
}

func cloneStrings(s []string) []string { return append([]string(nil), s...) }

func diffSlice(full, base []string) []string {
	set := map[string]struct{}{}
	for _, b := range base {
		set[b] = struct{}{}
	}
	var out []string
	for _, f := range full {
		if _, ok := set[f]; !ok {
			out = append(out, f)
		}
	}
	return out
}
