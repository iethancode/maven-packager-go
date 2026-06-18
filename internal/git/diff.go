package git

import (
	"strings"
	"time"
)

// FileDiff 单文件 diff 单元。
type FileDiff struct {
	Path     string `json:"path"`
	Status   string `json:"status"` // A/M/D/R 等
	Diff     string `json:"diff"`
	Truncate bool   `json:"truncate"`
}

// CommitDiff 某次提交全部文件的 diff 聚合。
type CommitDiff struct {
	Hash  string     `json:"hash"`
	Files []FileDiff `json:"files"`
}

// GetCommitStatus 返回 hash 下各文件的状态（A/M/D/...）。
func (c *Client) GetCommitStatus(hash string) (map[string]string, error) {
	out, err := c.Run(20*time.Second, "show", "--name-status", "--pretty=format:", hash)
	if err != nil {
		return nil, err
	}
	res := make(map[string]string)
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		parts := strings.Fields(ln)
		if len(parts) < 2 {
			continue
		}
		status := string(parts[0][0])
		path := parts[len(parts)-1]
		res[path] = status
	}
	return res, nil
}

// GetFileDiff 单文件 diff。超过 maxBytes 则截断并标记 truncate。
func (c *Client) GetFileDiff(hash, path string, maxBytes int) (FileDiff, error) {
	if maxBytes <= 0 {
		maxBytes = 2 * 1024 * 1024
	}
	out, err := c.Run(20*time.Second, "show",
		"--format=",
		"--unified=3",
		"--no-color",
		hash, "--", path)
	if err != nil {
		return FileDiff{Path: path}, err
	}
	truncate := false
	if len(out) > maxBytes {
		out = out[:maxBytes] + "\n\n... [diff truncated: file too large] ..."
		truncate = true
	}
	return FileDiff{Path: path, Diff: out, Truncate: truncate}, nil
}

// GetCommitDiff 取一个提交的全量 diff。
func (c *Client) GetCommitDiff(hash string, maxBytesPerFile int) (CommitDiff, error) {
	files, err := c.ChangedFiles(hash)
	if err != nil {
		return CommitDiff{Hash: hash}, err
	}
	statusMap, _ := c.GetCommitStatus(hash)

	items := make([]FileDiff, 0, len(files))
	for _, f := range files {
		fd, err := c.GetFileDiff(hash, f, maxBytesPerFile)
		if err != nil {
			fd = FileDiff{Path: f, Diff: "(diff 获取失败: " + err.Error() + ")"}
		}
		if s, ok := statusMap[f]; ok {
			fd.Status = s
		}
		items = append(items, fd)
	}
	return CommitDiff{Hash: hash, Files: items}, nil
}
