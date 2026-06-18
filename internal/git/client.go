// Package git 封装项目需要的 Git 命令。保留 Python 版同款的 ^@^ 分隔符策略。
package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"maven-packager-go/internal/procutil"
)

// Client 代表某个仓库根的 Git 操作入口。
type Client struct {
	RootDir string
}

func New(root string) *Client { return &Client{RootDir: root} }

// Run 执行命令并返回 stdout。超时保护避免挂死。
func (c *Client) Run(timeout time.Duration, args ...string) (string, error) {
	if c.RootDir == "" {
		return "", errors.New("git client: root dir is empty")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = c.RootDir
	procutil.HideWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return "", err
	}
	go func() { done <- cmd.Wait() }()

	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	select {
	case err := <-done:
		if err != nil {
			return strings.TrimSpace(stdout.String()), fmt.Errorf("git %s: %w (stderr: %s)", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
		}
		return strings.TrimSpace(stdout.String()), nil
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		return "", fmt.Errorf("git %s: timed out after %s", strings.Join(args, " "), timeout)
	}
}

// Branches 列出本地分支。
func (c *Client) Branches() []string {
	out, err := c.Run(15*time.Second, "branch", "--format=%(refname:short)")
	if err != nil || out == "" {
		return []string{}
	}
	lines := strings.Split(out, "\n")
	res := make([]string, 0, len(lines))
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			res = append(res, ln)
		}
	}
	return res
}

func (c *Client) CurrentBranch() string {
	out, _ := c.Run(15*time.Second, "branch", "--show-current")
	return strings.TrimSpace(out)
}

func (c *Client) Checkout(branch string) error {
	_, err := c.Run(60*time.Second, "checkout", branch)
	return err
}

func (c *Client) Pull(branch string) (string, error) {
	return c.Run(120*time.Second, "pull", "origin", branch)
}

func (c *Client) StatusPorcelain() string {
	out, _ := c.Run(15*time.Second, "status", "--porcelain")
	return out
}
