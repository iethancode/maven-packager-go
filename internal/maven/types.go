// Package maven 负责 Maven 项目模型、依赖解析与构建命令生成/执行。
package maven

import (
	"path/filepath"
	"sort"
	"strings"
)

// Coord 表示 Maven 的坐标。
type Coord struct {
	GroupID    string
	ArtifactID string
}

// ModuleInfo 单个 Maven 模块信息。
type ModuleInfo struct {
	Path         string   `json:"path"`         // 相对项目根（/ 分隔）
	PomPath      string   `json:"pomPath"`      // 绝对 pom 路径
	ArtifactID   string   `json:"artifactId"`
	GroupID      string   `json:"groupId"`
	Version      string   `json:"version"`
	Name         string   `json:"name"`
	Packaging    string   `json:"packaging"`
	Dependencies []Coord  `json:"-"`             // 原始声明依赖
	InternalDeps []string `json:"internalDeps"` // 已解析的内部依赖模块路径
}

// GraphNode 前端依赖图节点。
type GraphNode struct {
	ID         string `json:"id"`         // module path
	ArtifactID string `json:"artifactId"`
	GroupID    string `json:"groupId"`
	Version    string `json:"version"`
	Packaging  string `json:"packaging"`
	Name       string `json:"name"`
}

// GraphEdge 依赖边。from -> to，from 依赖 to。
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ModuleGraph 完整依赖图。
type ModuleGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// NormalizeModulePath 把系统相关路径规范成 `a/b/c` 格式，顶层返回空串。
func NormalizeModulePath(raw string) string {
	if raw == "" {
		return ""
	}
	p := filepath.ToSlash(filepath.Clean(raw))
	p = strings.TrimSpace(p)
	if p == "" || p == "." {
		return ""
	}
	return strings.TrimPrefix(p, "./")
}

// SortedDepsKeys 返回排序后的内部依赖列表，保证构建的确定性。
func SortedDepsKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
