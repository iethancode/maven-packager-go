// Package history 以 JSONL 追加写方式持久化构建历史。
package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const FileName = "build_history.jsonl"

// Record 单次构建记录。
type Record struct {
	ID               string   `json:"id"`
	StartedAt        string   `json:"startedAt"`
	Branch           string   `json:"branch"`
	Success          bool     `json:"success"`
	Commits          []string `json:"commits"`
	ChangedModules   []string `json:"changedModules"`
	AutoAddedModules []string `json:"autoAddedModules"`
	BuiltModules     []string `json:"builtModules"`
	CollectedJars    []string `json:"collectedJars"`
	ElapsedMs        int64    `json:"elapsedMs"`
	SpeedMode        string   `json:"speedMode"`
	ScopeMode        string   `json:"scopeMode"`
	OutputDir        string   `json:"outputDir"`
}

// Store 负责 JSONL 读写。
type Store struct {
	mu   sync.Mutex
	path string
}

func NewStore(baseDir string) *Store {
	return &Store{path: filepath.Join(baseDir, FileName)}
}

// Append 追加一条记录。
func (s *Store) Append(r Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.ID == "" {
		r.ID = time.Now().Format("20060102-150405.000")
	}
	if r.StartedAt == "" {
		r.StartedAt = time.Now().Format(time.RFC3339)
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}

// List 按时间降序返回历史记录，最多 limit 条（limit <= 0 表示全部）。
func (s *Store) List(limit int) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Record{}, nil
		}
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 4*1024*1024)
	var records []Record
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(line, &r); err != nil {
			continue
		}
		records = append(records, r)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].StartedAt > records[j].StartedAt })
	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}
	return records, nil
}

// Get 按 ID 定位一条记录。
func (s *Store) Get(id string) (*Record, error) {
	all, err := s.List(0)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, nil
}

// Clear 清空全部历史。
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(s.path)
}
