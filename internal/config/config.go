package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"maven-packager-go/internal/project"
)

const FileName = "packager_config.json"

type Config struct {
	Theme           string `json:"theme"`
	OutputDir       string `json:"output_dir"`
	BuildSpeed      string `json:"build_speed"`
	BuildScopeMode  string `json:"build_scope_mode"`
	LastBranch      string `json:"last_branch"`
	SmartDependency bool   `json:"smart_dependency"`
	ProjectRoot     string `json:"project_root"`
}

func Default() Config {
	return Config{
		Theme:           "Light",
		OutputDir:       "",
		BuildSpeed:      "快速模式",
		BuildScopeMode:  "稳妥模式",
		LastBranch:      "",
		SmartDependency: true,
		ProjectRoot:     "",
	}
}

type Manager struct {
	mu   sync.Mutex
	path string
	cur  Config
}

func NewManager() *Manager {
	m := &Manager{path: resolveConfigPath()}
	m.cur = m.loadFromDisk()
	return m
}

func (m *Manager) Path() string { return m.path }

func (m *Manager) Get() Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cur
}

func (m *Manager) Reload() Config {
	next := m.loadFromDisk()
	m.mu.Lock()
	m.cur = next
	m.mu.Unlock()
	return next
}

func (m *Manager) Set(c Config) error {
	m.mu.Lock()
	m.cur = c
	m.mu.Unlock()
	return m.saveToDisk(c)
}

func (m *Manager) Patch(patch map[string]any) Config {
	m.mu.Lock()
	raw, _ := json.Marshal(m.cur)
	var merged map[string]any
	_ = json.Unmarshal(raw, &merged)
	for k, v := range patch {
		merged[k] = v
	}
	out, _ := json.Marshal(merged)
	var next Config
	_ = json.Unmarshal(out, &next)
	m.cur = next
	snapshot := next
	m.mu.Unlock()

	_ = m.saveToDisk(snapshot)
	return snapshot
}

func (m *Manager) loadFromDisk() Config {
	cur := Default()
	data, err := os.ReadFile(m.path)
	if err != nil {
		_ = m.saveToDisk(cur)
		return cur
	}

	var loaded map[string]any
	if err := json.Unmarshal(data, &loaded); err != nil {
		_ = m.saveToDisk(cur)
		return cur
	}

	defBytes, _ := json.Marshal(cur)
	var merged map[string]any
	_ = json.Unmarshal(defBytes, &merged)
	for k, v := range loaded {
		merged[k] = v
	}
	out, _ := json.Marshal(merged)
	if err := json.Unmarshal(out, &cur); err != nil {
		cur = Default()
	}
	_ = m.saveToDisk(cur)
	return cur
}

func (m *Manager) saveToDisk(c Config) error {
	_ = os.MkdirAll(filepath.Dir(m.path), 0o755)
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.path)
}

func resolveConfigPath() string {
	if len(os.Args) > 0 && os.Args[0] != "" {
		if argPath, err := filepath.Abs(os.Args[0]); err == nil {
			argDir := filepath.Dir(argPath)
			argConfig := filepath.Join(argDir, FileName)
			if _, err := os.Stat(argConfig); err == nil {
				return argConfig
			}
			if _, err := os.Stat(argPath); err == nil {
				return argConfig
			}
		}
	}
	return filepath.Join(project.AppBaseDir(), FileName)
}
