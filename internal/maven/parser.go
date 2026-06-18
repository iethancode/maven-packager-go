package maven

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type xmlPom struct {
	XMLName      xml.Name    `xml:"project"`
	GroupID      string      `xml:"groupId"`
	ArtifactID   string      `xml:"artifactId"`
	Version      string      `xml:"version"`
	Name         string      `xml:"name"`
	Packaging    string      `xml:"packaging"`
	Parent       *xmlParent  `xml:"parent"`
	Properties   xmlProps    `xml:"properties"`
	Dependencies xmlDepsRoot `xml:"dependencies"`
}

type xmlParent struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

type xmlDepsRoot struct {
	Deps []xmlDep `xml:"dependency"`
}

type xmlDep struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Optional   string `xml:"optional"`
}

// xmlProps 任意键值属性；采用 UnmarshalXML 自定义读取所有子元素。
type xmlProps struct {
	Map map[string]string
}

func (p *xmlProps) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if p.Map == nil {
		p.Map = map[string]string{}
	}
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			var val string
			if err := d.DecodeElement(&val, &t); err != nil {
				return err
			}
			p.Map[t.Name.Local] = strings.TrimSpace(val)
		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
	}
}

// DiscoverModulePoms 递归扫描项目根，返回所有 pom.xml 的绝对路径。
func DiscoverModulePoms(rootDir string) []string {
	skip := map[string]struct{}{
		".git": {}, ".idea": {}, ".vscode": {},
		"target": {}, "node_modules": {}, "dist": {},
	}
	var res []string
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if _, ok := skip[d.Name()]; ok {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "pom.xml" {
			res = append(res, path)
		}
		return nil
	})
	return res
}

// ParseModule 解析单个 pom.xml，占位符在本函数内部完成解析。
func ParseModule(rootDir, pomPath string) (*ModuleInfo, error) {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return nil, err
	}

	var p xmlPom
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	dec.Strict = false
	if err := dec.Decode(&p); err != nil {
		return nil, err
	}

	props := map[string]string{}
	for k, v := range p.Properties.Map {
		props[k] = v
	}

	parentGroup, parentArtifact, parentVersion := "", "", ""
	if p.Parent != nil {
		parentGroup = resolvePlaceholders(p.Parent.GroupID, props)
		parentArtifact = resolvePlaceholders(p.Parent.ArtifactID, props)
		parentVersion = resolvePlaceholders(p.Parent.Version, props)
	}
	props["parent.groupId"] = parentGroup
	props["parent.artifactId"] = parentArtifact
	props["parent.version"] = parentVersion
	props["project.parent.groupId"] = parentGroup
	props["project.parent.artifactId"] = parentArtifact
	props["project.parent.version"] = parentVersion

	artifact := resolvePlaceholders(p.ArtifactID, props)
	group := resolvePlaceholders(firstNonEmpty(p.GroupID, parentGroup), props)
	version := resolvePlaceholders(firstNonEmpty(p.Version, parentVersion), props)
	packaging := resolvePlaceholders(firstNonEmpty(p.Packaging, "jar"), props)

	props["project.groupId"] = group
	props["pom.groupId"] = group
	props["project.artifactId"] = artifact
	props["pom.artifactId"] = artifact
	props["project.version"] = version
	props["pom.version"] = version

	name := resolvePlaceholders(firstNonEmpty(p.Name, artifact), props)

	deps := make([]Coord, 0, len(p.Dependencies.Deps))
	for _, d := range p.Dependencies.Deps {
		g := resolvePlaceholders(d.GroupID, props)
		a := resolvePlaceholders(d.ArtifactID, props)
		scope := strings.ToLower(resolvePlaceholders(d.Scope, props))
		optional := strings.EqualFold(resolvePlaceholders(d.Optional, props), "true")
		if g == "" || a == "" {
			continue
		}
		if scope == "test" || optional {
			continue
		}
		deps = append(deps, Coord{GroupID: g, ArtifactID: a})
	}

	rel, _ := filepath.Rel(rootDir, filepath.Dir(pomPath))
	modulePath := NormalizeModulePath(rel)
	if modulePath == "" {
		// 顶层 pom 自己有时也是一个模块；用空字符串代表根模块。
		modulePath = "."
	}

	return &ModuleInfo{
		Path:         modulePath,
		PomPath:      pomPath,
		ArtifactID:   artifact,
		GroupID:      group,
		Version:      version,
		Name:         name,
		Packaging:    packaging,
		Dependencies: deps,
	}, nil
}

var placeholderRe = regexp.MustCompile(`\$\{([^}]+)\}`)

func resolvePlaceholders(text string, props map[string]string) string {
	if text == "" {
		return ""
	}
	cur := strings.TrimSpace(text)
	for i := 0; i < 5; i++ {
		updated := placeholderRe.ReplaceAllStringFunc(cur, func(match string) string {
			key := match[2 : len(match)-1]
			if v, ok := props[key]; ok {
				return v
			}
			return match
		})
		if updated == cur {
			break
		}
		cur = updated
	}
	return cur
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
