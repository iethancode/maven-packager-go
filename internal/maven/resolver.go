package maven

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Handler 聚合一个 Maven 项目的模块索引与操作。
type Handler struct {
	RootDir string

	mu            sync.Mutex
	modules       map[string]*ModuleInfo // path -> info
	coordIndex    map[Coord]string       // (group,artifact) -> path
	artifactIndex map[string][]string    // lower(artifact) -> paths
	nameIndex     map[string][]string    // lower(name/dirName) -> paths
	loaded        bool
}

func NewHandler(rootDir string) *Handler {
	return &Handler{
		RootDir:       rootDir,
		modules:       map[string]*ModuleInfo{},
		coordIndex:    map[Coord]string{},
		artifactIndex: map[string][]string{},
		nameIndex:     map[string][]string{},
	}
}

// EnsureLoaded 懒加载模块索引。
func (h *Handler) EnsureLoaded() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.loaded {
		return
	}
	h.reloadLocked()
}

// Reload 强制刷新索引。
func (h *Handler) Reload() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reloadLocked()
}

func (h *Handler) reloadLocked() {
	h.modules = map[string]*ModuleInfo{}
	h.coordIndex = map[Coord]string{}
	h.artifactIndex = map[string][]string{}
	h.nameIndex = map[string][]string{}

	poms := DiscoverModulePoms(h.RootDir)
	for _, pom := range poms {
		info, err := ParseModule(h.RootDir, pom)
		if err != nil || info == nil {
			continue
		}
		h.modules[info.Path] = info

		if info.GroupID != "" && info.ArtifactID != "" {
			if _, ok := h.coordIndex[Coord{info.GroupID, info.ArtifactID}]; !ok {
				h.coordIndex[Coord{info.GroupID, info.ArtifactID}] = info.Path
			}
		}
		if info.ArtifactID != "" {
			key := strings.ToLower(info.ArtifactID)
			h.artifactIndex[key] = appendUnique(h.artifactIndex[key], info.Path)
		}
		if info.Name != "" {
			key := strings.ToLower(info.Name)
			h.nameIndex[key] = appendUnique(h.nameIndex[key], info.Path)
		}
		dirName := strings.ToLower(filepath.Base(info.Path))
		if dirName != "" && dirName != "." {
			h.nameIndex[dirName] = appendUnique(h.nameIndex[dirName], info.Path)
		}
	}

	// 根据坐标索引补齐每个模块的 internal dependencies。
	for _, info := range h.modules {
		set := map[string]struct{}{}
		for _, d := range info.Dependencies {
			if p, ok := h.coordIndex[d]; ok && p != info.Path {
				set[p] = struct{}{}
			}
		}
		info.InternalDeps = SortedDepsKeys(set)
	}

	h.loaded = true
}

func appendUnique(arr []string, val string) []string {
	for _, v := range arr {
		if v == val {
			return arr
		}
	}
	return append(arr, val)
}

// Modules 只读返回当前模块集合（副本）。
func (h *Handler) Modules() map[string]*ModuleInfo {
	h.EnsureLoaded()
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make(map[string]*ModuleInfo, len(h.modules))
	for k, v := range h.modules {
		cp := *v
		out[k] = &cp
	}
	return out
}

// FindModuleByFile 根据仓库相对路径返回所在模块。
func (h *Handler) FindModuleByFile(relFile string) string {
	h.EnsureLoaded()

	abs := filepath.Join(h.RootDir, relFile)
	absDir := abs
	if stat, err := statOrNil(abs); err != nil || stat == nil || !stat.IsDir() {
		absDir = filepath.Dir(abs)
	}

	// 从文件所在目录向上查找最近的 pom 所属模块
	cur := absDir
	for {
		rel, err := filepath.Rel(h.RootDir, cur)
		if err != nil || strings.HasPrefix(rel, "..") {
			return ""
		}
		norm := NormalizeModulePath(rel)
		if norm == "" {
			norm = "."
		}
		if _, ok := h.modules[norm]; ok {
			return norm
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return ""
}

// FindModuleByName 通过 artifactId / name / 目录名查找模块。
func (h *Handler) FindModuleByName(name string) string {
	if name == "" {
		return ""
	}
	h.EnsureLoaded()

	// 原样命中
	if _, ok := h.modules[name]; ok {
		return name
	}
	key := strings.ToLower(strings.TrimSpace(name))
	candidates := map[string]struct{}{}
	for _, p := range h.artifactIndex[key] {
		candidates[p] = struct{}{}
	}
	for _, p := range h.nameIndex[key] {
		candidates[p] = struct{}{}
	}
	if len(candidates) == 0 {
		return ""
	}
	list := make([]string, 0, len(candidates))
	for k := range candidates {
		list = append(list, k)
	}
	sort.SliceStable(list, func(i, j int) bool {
		ai := strings.Count(list[i], "/")
		aj := strings.Count(list[j], "/")
		if ai != aj {
			return ai < aj
		}
		return list[i] < list[j]
	})
	return list[0]
}

// FindModuleByCoord 通过 groupId+artifactId 精确查找。
func (h *Handler) FindModuleByCoord(group, artifact string) string {
	h.EnsureLoaded()
	if p, ok := h.coordIndex[Coord{group, artifact}]; ok {
		return p
	}
	if len(h.artifactIndex[strings.ToLower(artifact)]) == 1 {
		return h.artifactIndex[strings.ToLower(artifact)][0]
	}
	return ""
}

// ExpandWithDependencies 沿内部依赖递归展开，返回拓扑排序后的构建列表。
func (h *Handler) ExpandWithDependencies(paths []string) []string {
	h.EnsureLoaded()
	expanded := map[string]struct{}{}
	stack := make([]string, 0, len(paths))
	for _, p := range paths {
		if np := NormalizeModulePath(p); np != "" {
			stack = append(stack, np)
		}
	}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, ok := expanded[cur]; ok {
			continue
		}
		info := h.modules[cur]
		if info == nil {
			continue
		}
		expanded[cur] = struct{}{}
		for _, d := range info.InternalDeps {
			if _, ok := expanded[d]; !ok {
				stack = append(stack, d)
			}
		}
	}
	return h.topologicalSort(expanded)
}

// topologicalSort 按依赖顺序排序所选模块子集。
func (h *Handler) topologicalSort(selected map[string]struct{}) []string {
	visited := map[string]struct{}{}
	visiting := map[string]struct{}{}
	out := make([]string, 0, len(selected))

	keys := make([]string, 0, len(selected))
	for k := range selected {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var visit func(p string)
	visit = func(p string) {
		if _, ok := visited[p]; ok {
			return
		}
		if _, ok := selected[p]; !ok {
			return
		}
		if _, ok := visiting[p]; ok {
			return
		}
		visiting[p] = struct{}{}
		info := h.modules[p]
		if info != nil {
			for _, d := range info.InternalDeps {
				visit(d)
			}
		}
		delete(visiting, p)
		visited[p] = struct{}{}
		out = append(out, p)
	}

	for _, k := range keys {
		visit(k)
	}
	return out
}

// Graph 返回依赖图。
func (h *Handler) Graph() ModuleGraph {
	h.EnsureLoaded()
	h.mu.Lock()
	defer h.mu.Unlock()

	nodes := make([]GraphNode, 0, len(h.modules))
	edges := make([]GraphEdge, 0)
	for _, m := range h.modules {
		nodes = append(nodes, GraphNode{
			ID:         m.Path,
			ArtifactID: m.ArtifactID,
			GroupID:    m.GroupID,
			Version:    m.Version,
			Packaging:  m.Packaging,
			Name:       m.Name,
		})
		for _, dep := range m.InternalDeps {
			edges = append(edges, GraphEdge{From: m.Path, To: dep})
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})
	return ModuleGraph{Nodes: nodes, Edges: edges}
}
