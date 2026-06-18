// Package project 负责定位项目根目录（.git > Maven 聚合根 > 普通 pom 目录）。
package project

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// IsMavenProjectDir 判断目录下是否存在 pom.xml。
func IsMavenProjectDir(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "pom.xml"))
	return err == nil
}

type pomSkeleton struct {
	XMLName   xml.Name `xml:"project"`
	Packaging string   `xml:"packaging"`
	Modules   struct {
		Module []string `xml:"module"`
	} `xml:"modules"`
}

// IsMavenAggregatorRoot 判断目录是否为 Maven 聚合根（packaging=pom 且声明了 modules）。
func IsMavenAggregatorRoot(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "pom.xml"))
	if err != nil {
		return false
	}
	var pom pomSkeleton
	if err := xml.Unmarshal(data, &pom); err != nil {
		return false
	}
	return strings.EqualFold(pom.Packaging, "pom") && len(pom.Modules.Module) > 0
}

// IterParentDirs 返回从 startDir 起自底向上的所有祖先目录（含自身）。
func IterParentDirs(startDir string) []string {
	out := []string{}
	cur, err := filepath.Abs(startDir)
	if err != nil {
		return out
	}
	for {
		out = append(out, cur)
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return out
}

// IsValidProjectRoot .git 或 pom.xml 之一存在即认为有效。
func IsValidProjectRoot(dir string) bool {
	if dir == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "pom.xml")); err == nil {
		return true
	}
	if HasChildGitRepository(dir) || HasChildPom(dir) {
		return true
	}
	return false
}

// HasChildGitRepository 判断目录下是否存在子 Git 仓库，用于支持工作区/群组项目。
func HasChildGitRepository(dir string) bool {
	return walkUntil(dir, func(path string, d os.DirEntry) bool {
		return d.IsDir() && path != dir && exists(filepath.Join(path, ".git"))
	})
}

// HasChildPom 判断目录下是否存在子 Maven 项目。
func HasChildPom(dir string) bool {
	return walkUntil(dir, func(path string, d os.DirEntry) bool {
		return d.IsDir() && path != dir && exists(filepath.Join(path, "pom.xml"))
	})
}

func CountChildPoms(dir string) int {
	if dir == "" {
		return 0
	}
	skip := map[string]struct{}{
		".git": {}, ".idea": {}, ".vscode": {},
		"target": {}, "node_modules": {}, "dist": {}, "build": {},
	}
	count := 0
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir {
				if _, ok := skip[d.Name()]; ok {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if d.Name() == "pom.xml" {
			count++
		}
		return nil
	})
	return count
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func walkUntil(root string, match func(string, os.DirEntry) bool) bool {
	if root == "" {
		return false
	}
	skip := map[string]struct{}{
		".git": {}, ".idea": {}, ".vscode": {},
		"target": {}, "node_modules": {}, "dist": {}, "build": {},
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path != root {
			if _, ok := skip[d.Name()]; ok {
				return filepath.SkipDir
			}
		}
		if match(path, d) {
			found = true
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

// FindProjectRoot 从多个起点向上查找最合适的项目根目录。
// 优先 .git，其次 Maven 聚合根，再次普通 pom 目录。同类取最浅路径。
func FindProjectRoot(startDirs []string) string {
	seen := map[string]struct{}{}
	var gitCandidates, aggCandidates, pomCandidates []string

	for _, s := range startDirs {
		if s == "" {
			continue
		}
		for _, cur := range IterParentDirs(s) {
			key := strings.ToLower(cur)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
				gitCandidates = append(gitCandidates, cur)
				continue
			}
			if IsMavenAggregatorRoot(cur) {
				aggCandidates = append(aggCandidates, cur)
				continue
			}
			if IsMavenProjectDir(cur) {
				pomCandidates = append(pomCandidates, cur)
			}
		}
	}

	pickShallowest := func(paths []string) string {
		if len(paths) == 0 {
			return ""
		}
		dedup := map[string]struct{}{}
		uniq := make([]string, 0, len(paths))
		for _, p := range paths {
			abs, _ := filepath.Abs(p)
			if _, ok := dedup[abs]; ok {
				continue
			}
			dedup[abs] = struct{}{}
			uniq = append(uniq, abs)
		}
		sort.SliceStable(uniq, func(i, j int) bool {
			a, b := uniq[i], uniq[j]
			ai, bi := strings.Count(a, string(os.PathSeparator)), strings.Count(b, string(os.PathSeparator))
			if ai != bi {
				return ai < bi
			}
			return len(a) < len(b)
		})
		return uniq[0]
	}

	if r := pickShallowest(gitCandidates); r != "" {
		return r
	}
	if r := pickShallowest(aggCandidates); r != "" {
		return r
	}
	return pickShallowest(pomCandidates)
}

// scanSkip 是工作区扫描时跳过的目录名集合（不进入其内部）。
// 相比单次遍历版本，扩充了常见构建/IDE 残留目录，减少子群组大仓的无效下钻。
var scanSkip = map[string]struct{}{
	".git": {}, ".idea": {}, ".vscode": {},
	"target": {}, "node_modules": {}, "dist": {}, "build": {},
	"bin": {}, "logs": {}, ".gradle": {}, ".settings": {},
	"out": {}, "tmp": {}, "coverage": {}, ".checkstyle": {},
	".svn": {}, ".hg": {},
}

// ScanResult 一次扫描工作区得到的摘要信息。
type ScanResult struct {
	HasRootGit   bool     // root 自身是 git 仓库
	HasRootPom   bool     // root 下有 pom.xml
	HasChildGit  bool     // 存在子 git 仓库（工作区/群组项目）
	HasChildPom  bool     // 存在子 Maven 项目
	ModuleCount  int      // 所有 pom.xml 数量（含 root pom 与子仓内 pom）
	ChildGitDirs []string // 子 .git 所在目录绝对路径（不含 root 自身）
}

// ScanWorkspace 单次遍历工作区，同时收集子仓、pom 存在性与模块数量，
// 避免首屏对同一目录多次全树 walk（子群组项目的主要启动瓶颈）。
// 命中子 .git 后仍会下钻统计该子仓内的 pom，但名为 scanSkip 的目录不再进入。
func ScanWorkspace(root string) ScanResult {
	var res ScanResult
	if root == "" {
		return res
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return res
	}
	res.HasRootGit = exists(filepath.Join(absRoot, ".git"))
	res.HasRootPom = exists(filepath.Join(absRoot, "pom.xml"))
	if res.HasRootPom {
		res.ModuleCount = 1
	}
	if res.HasRootGit {
		// root 自身是 git 仓库，无需继续扫描子仓
		return res
	}
	scanChildDirs(absRoot, &res)
	return res
}

func scanChildDirs(dir string, res *ScanResult) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if _, ok := scanSkip[name]; ok {
			continue
		}
		child := filepath.Join(dir, name)
		if exists(filepath.Join(child, ".git")) {
			res.HasChildGit = true
			res.ChildGitDirs = append(res.ChildGitDirs, child)
		}
		if exists(filepath.Join(child, "pom.xml")) {
			res.HasChildPom = true
			res.ModuleCount++
		}
		// 继续下钻统计深层 pom；名为 .git 等的目录已在上方被跳过。
		scanChildDirs(child, res)
	}
}

// AppBaseDir 返回 exe 所在目录，用于定位默认配置文件。
func AppBaseDir() string {
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	return filepath.Dir(exe)
}
