package git

import (
	"strings"
	"time"
)

// Commit 与前端约定的字段命名（保持 JSON 小写以兼容旧前端渲染逻辑）。
type Commit struct {
	Hash   string `json:"hash"`
	Repo   string `json:"repo,omitempty"`
	Author string `json:"author"`
	Date   string `json:"date"`
	Msg    string `json:"msg"`
}

const logSeparator = "^@^"

// Log 拉取完整提交历史。前端再做关键词/日期过滤。
func (c *Client) Log(limit int) ([]Commit, error) {
	args := []string{
		"log",
		"--date=format:%Y-%m-%d %H:%M:%S",
		"--pretty=format:%h" + logSeparator + "%an" + logSeparator + "%ad" + logSeparator + "%s",
	}
	if limit > 0 {
		args = append(args, "-n", itoa(limit))
	}
	out, err := c.Run(45*time.Second, args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []Commit{}, nil
	}

	lines := strings.Split(out, "\n")
	res := make([]Commit, 0, len(lines))
	for _, ln := range lines {
		parts := strings.SplitN(ln, logSeparator, 4)
		if len(parts) != 4 {
			continue
		}
		res = append(res, Commit{
			Hash:   strings.TrimSpace(parts[0]),
			Author: strings.TrimSpace(parts[1]),
			Date:   strings.TrimSpace(parts[2]),
			Msg:    strings.TrimSpace(parts[3]),
		})
	}
	return res, nil
}

// ChangedFiles 返回指定提交下的变更文件列表。
func (c *Client) ChangedFiles(hash string) ([]string, error) {
	out, err := c.Run(20*time.Second, "show", "--name-only", "--pretty=format:", hash)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	res := make([]string, 0, len(lines))
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		res = append(res, ln)
	}
	return res, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
