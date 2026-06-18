package git

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"maven-packager-go/internal/project"
)

const workspaceCommitSeparator = "::"

type Repository struct {
	Root string
	Rel  string
}

type Workspace struct {
	RootDir string
	repos   []Repository
}

func NewWorkspace(root string) *Workspace {
	scan := project.ScanWorkspace(root)
	return NewWorkspaceFromScan(root, scan.ChildGitDirs, scan.HasRootGit)
}

// NewWorkspaceFromScan 用预扫描得到的子仓目录直接构造 Workspace，避免再次全树遍历。
// 与 app 层的 ScanWorkspace 共享同一次扫描结果。
func NewWorkspaceFromScan(root string, childGitDirs []string, hasRootGit bool) *Workspace {
	w := &Workspace{RootDir: root}
	if hasRootGit {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			absRoot = root
		}
		w.repos = []Repository{{Root: absRoot, Rel: "."}}
		return w
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}
	for _, dir := range childGitDirs {
		rel, _ := filepath.Rel(absRoot, dir)
		w.repos = append(w.repos, Repository{
			Root: dir,
			Rel:  normalizeRel(rel),
		})
	}
	sort.SliceStable(w.repos, func(i, j int) bool {
		return w.repos[i].Rel < w.repos[j].Rel
	})
	return w
}

func DiscoverRepositories(root string) []Repository {
	if root == "" {
		return nil
	}
	scan := project.ScanWorkspace(root)
	if scan.HasRootGit {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			absRoot = root
		}
		return []Repository{{Root: absRoot, Rel: "."}}
	}
	return NewWorkspaceFromScan(root, scan.ChildGitDirs, false).Repositories()
}

func (w *Workspace) HasRepositories() bool {
	return w != nil && len(w.repos) > 0
}

func (w *Workspace) Repositories() []Repository {
	if w == nil {
		return nil
	}
	out := make([]Repository, len(w.repos))
	copy(out, w.repos)
	return out
}

func (w *Workspace) Branches() []string {
	if !w.HasRepositories() {
		return []string{}
	}
	if len(w.repos) == 1 {
		return New(w.repos[0].Root).Branches()
	}
	if len(w.repos) > 10 {
		return New(w.repos[0].Root).Branches()
	}
	seen := map[string]struct{}{}
	var out []string
	results := runRepoJobs(w.repos, func(repo Repository) []string {
		return New(repo.Root).Branches()
	})
	for _, branches := range results {
		for _, branch := range branches {
			if _, ok := seen[branch]; ok {
				continue
			}
			seen[branch] = struct{}{}
			out = append(out, branch)
		}
	}
	sort.Strings(out)
	return out
}

func (w *Workspace) CurrentBranch() string {
	if !w.HasRepositories() {
		return ""
	}
	if len(w.repos) == 1 {
		return New(w.repos[0].Root).CurrentBranch()
	}
	if len(w.repos) > 10 {
		first := New(w.repos[0].Root).CurrentBranch()
		if first == "" {
			return ""
		}
		return first + " 等"
	}
	results := runRepoJobs(w.repos, func(repo Repository) string {
		return New(repo.Root).CurrentBranch()
	})
	first := ""
	for _, branch := range results {
		if branch == "" {
			continue
		}
		if first == "" {
			first = branch
			continue
		}
		if branch != first {
			return first + " 等"
		}
	}
	return first
}

func (w *Workspace) Checkout(branch string) error {
	if !w.HasRepositories() {
		return nil
	}
	checked := false
	for _, repo := range w.repos {
		c := New(repo.Root)
		if hasBranch(c.Branches(), branch) {
			if err := c.Checkout(branch); err != nil {
				return err
			}
			checked = true
		}
	}
	if !checked && len(w.repos) == 1 {
		return New(w.repos[0].Root).Checkout(branch)
	}
	return nil
}

func (w *Workspace) Pull(branch string) (string, error) {
	if !w.HasRepositories() {
		return "", nil
	}
	var out []string
	var firstErr error
	for _, repo := range w.repos {
		c := New(repo.Root)
		if !hasBranch(c.Branches(), branch) {
			continue
		}
		text, err := c.Pull(branch)
		if text != "" {
			out = append(out, "["+repo.Rel+"] "+text)
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return strings.Join(out, "\n"), firstErr
}

func (w *Workspace) StatusPorcelain() string {
	if !w.HasRepositories() {
		return ""
	}
	var out []string
	results := runRepoJobs(w.repos, func(repo Repository) string {
		text := New(repo.Root).StatusPorcelain()
		if strings.TrimSpace(text) == "" {
			return ""
		}
		return "[" + repo.Rel + "]\n" + text
	})
	for _, text := range results {
		if text != "" {
			out = append(out, text)
		}
	}
	return strings.Join(out, "\n")
}

func (w *Workspace) Log(limit int) ([]Commit, error) {
	if !w.HasRepositories() {
		return []Commit{}, nil
	}
	if len(w.repos) == 1 {
		return New(w.repos[0].Root).Log(limit)
	}

	perRepoLimit := limit
	if len(w.repos) > 10 && (perRepoLimit <= 0 || perRepoLimit > 5) {
		perRepoLimit = 5
	}
	results := runRepoJobs(w.repos, func(repo Repository) []Commit {
		commits, err := New(repo.Root).Log(perRepoLimit)
		if err != nil {
			return nil
		}
		for i := range commits {
			commits[i].Repo = repo.Rel
			commits[i].Hash = encodeCommitID(repo.Rel, commits[i].Hash)
		}
		return commits
	})

	var all []Commit
	for _, commits := range results {
		all = append(all, commits...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Date > all[j].Date
	})
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func (w *Workspace) LogBatches(limit int, emit func([]Commit)) {
	if !w.HasRepositories() || emit == nil {
		return
	}
	if len(w.repos) == 1 {
		commits, err := New(w.repos[0].Root).Log(limit)
		if err == nil && len(commits) > 0 {
			emit(commits)
		}
		return
	}

	perRepoLimit := limit
	if len(w.repos) > 10 && (perRepoLimit <= 0 || perRepoLimit > 5) {
		perRepoLimit = 5
	}
	repos := sortRepositoriesByActivity(w.repos)
	workers := 32
	if len(repos) < workers {
		workers = len(repos)
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for _, repo := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(repo Repository) {
			defer wg.Done()
			defer func() { <-sem }()
			commits, err := New(repo.Root).Log(perRepoLimit)
			if err != nil || len(commits) == 0 {
				return
			}
			for i := range commits {
				commits[i].Repo = repo.Rel
				commits[i].Hash = encodeCommitID(repo.Rel, commits[i].Hash)
			}
			emit(commits)
		}(repo)
	}
	wg.Wait()
}

func (w *Workspace) ChangedFiles(commitID string) ([]string, error) {
	repo, hash, ok := w.resolveCommit(commitID)
	if !ok {
		return []string{}, nil
	}
	files, err := New(repo.Root).ChangedFiles(hash)
	if err != nil {
		return nil, err
	}
	return prefixFiles(repo.Rel, files), nil
}

func (w *Workspace) GetCommitDiff(commitID string, maxBytesPerFile int) (CommitDiff, error) {
	repo, hash, ok := w.resolveCommit(commitID)
	if !ok {
		return CommitDiff{Hash: commitID}, nil
	}
	diff, err := New(repo.Root).GetCommitDiff(hash, maxBytesPerFile)
	diff.Hash = commitID
	for i := range diff.Files {
		diff.Files[i].Path = prefixFile(repo.Rel, diff.Files[i].Path)
	}
	return diff, err
}

func (w *Workspace) GetFileDiff(commitID, path string, maxBytes int) (FileDiff, error) {
	repo, hash, ok := w.resolveCommit(commitID)
	if !ok {
		return FileDiff{Path: path}, nil
	}
	repoPath := stripRepoPrefix(repo.Rel, path)
	diff, err := New(repo.Root).GetFileDiff(hash, repoPath, maxBytes)
	diff.Path = prefixFile(repo.Rel, diff.Path)
	return diff, err
}

func (w *Workspace) resolveCommit(commitID string) (Repository, string, bool) {
	if !w.HasRepositories() {
		return Repository{}, "", false
	}
	if len(w.repos) == 1 {
		return w.repos[0], commitID, true
	}
	rel, hash, ok := decodeCommitID(commitID)
	if !ok {
		return Repository{}, "", false
	}
	for _, repo := range w.repos {
		if repo.Rel == rel {
			return repo, hash, true
		}
	}
	return Repository{}, "", false
}

func hasBranch(branches []string, branch string) bool {
	for _, b := range branches {
		if b == branch {
			return true
		}
	}
	return false
}

func encodeCommitID(repoRel, hash string) string {
	return repoRel + workspaceCommitSeparator + hash
}

func decodeCommitID(id string) (string, string, bool) {
	parts := strings.SplitN(id, workspaceCommitSeparator, 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func prefixFiles(repoRel string, files []string) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, prefixFile(repoRel, f))
	}
	return out
}

func prefixFile(repoRel, file string) string {
	file = normalizeRel(file)
	if repoRel == "." || repoRel == "" {
		return file
	}
	return normalizeRel(filepath.ToSlash(filepath.Join(repoRel, file)))
}

func stripRepoPrefix(repoRel, file string) string {
	file = normalizeRel(file)
	if repoRel == "." || repoRel == "" {
		return file
	}
	prefix := strings.TrimSuffix(repoRel, "/") + "/"
	return strings.TrimPrefix(file, prefix)
}

func normalizeRel(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." {
		return "."
	}
	return strings.TrimPrefix(path, "./")
}

func runRepoJobs[T any](repos []Repository, fn func(Repository) T) []T {
	results := make([]T, len(repos))
	workers := 32
	if len(repos) < workers {
		workers = len(repos)
	}
	if workers <= 0 {
		return results
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, repo := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, repo Repository) {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = fn(repo)
		}(i, repo)
	}
	wg.Wait()
	return results
}

func sortRepositoriesByActivity(repos []Repository) []Repository {
	out := make([]Repository, len(repos))
	copy(out, repos)
	sort.SliceStable(out, func(i, j int) bool {
		ti := repositoryActivityTime(out[i])
		tj := repositoryActivityTime(out[j])
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return out[i].Rel < out[j].Rel
	})
	return out
}

func repositoryActivityTime(repo Repository) time.Time {
	candidates := []string{
		filepath.Join(repo.Root, ".git", "logs", "HEAD"),
		filepath.Join(repo.Root, ".git", "HEAD"),
		filepath.Join(repo.Root, ".git"),
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil {
			return info.ModTime()
		}
	}
	return time.Time{}
}
