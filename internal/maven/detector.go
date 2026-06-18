package maven

import (
	"regexp"
	"strings"
)

var missingArtifactPatterns = []*regexp.Regexp{
	regexp.MustCompile(`Could not find artifact\s+([^:\s]+):([^:\s]+):`),
	regexp.MustCompile(`The POM for\s+([^:\s]+):([^:\s]+):`),
	regexp.MustCompile(`Failure to find\s+([^:\s]+):([^:\s]+):`),
	regexp.MustCompile(`Missing artifact\s+([^:\s]+):([^:\s]+):`),
}

// ExtractMissingModules 从 Maven 构建输出中抓出缺失的内部模块路径。
func (h *Handler) ExtractMissingModules(output string, existing []string) []string {
	h.EnsureLoaded()

	existingSet := map[string]struct{}{}
	for _, p := range existing {
		if np := NormalizeModulePath(p); np != "" {
			existingSet[np] = struct{}{}
		}
	}

	seen := map[string]struct{}{}
	var out []string
	for _, ln := range strings.Split(output, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		for _, re := range missingArtifactPatterns {
			m := re.FindStringSubmatch(ln)
			if m == nil {
				continue
			}
			p := h.FindModuleByCoord(m[1], m[2])
			if p == "" {
				continue
			}
			if _, ok := existingSet[p]; ok {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	return out
}
