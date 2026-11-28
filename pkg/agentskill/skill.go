// Package agentskill 管理技能在磁盘上的那一份：一个技能是一个目录，里面有 SKILL.md 和它引用的文件。
// 技能在 Momoko 这边只存一份；起会话时把启用的技能目录经 ACP 的 additionalDirectories 交给 Agent，不复制。
package agentskill

import (
	"context"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"momoko/pkg/agentfs"
	"momoko/pkg/archive"
	"momoko/pkg/constant"
	"momoko/pkg/download"
)

const (
	// MainFile 每个技能都要有的入口文件。
	MainFile = "SKILL.md"
	// MaxFiles 一个技能最多多少个文件。
	MaxFiles = 300
	// maxText 能在编辑器里打开的单个文件大小。
	maxText = 1 << 20
)

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// ValidName 技能名（同时是目录名）只能用小写字母、数字、点、横线和下划线。
func ValidName(name string) bool { return namePattern.MatchString(name) }

// File 技能里的一个文件。Binary 的文件只保留，不在编辑器里编辑。
type File struct {
	Path    string
	Content string
	Binary  bool
}

// Store 存放技能的根目录。
type Store struct {
	Root string
}

// Dir 某个技能的目录。
func (s Store) Dir(name string) string { return filepath.Join(s.Root, name) }

// Remove 删掉一个技能的全部文件。
func (s Store) Remove(name string) { _ = os.RemoveAll(s.Dir(name)) }

// paths 列出目录里的文件（相对路径，正斜杠），SKILL.md 排最前。
func paths(dir string) []string {
	var files []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if len(files) >= MaxFiles {
			return nil
		}
		if rel, err := filepath.Rel(dir, path); err == nil {
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.SliceStable(files, func(i, j int) bool {
		if (files[i] == MainFile) != (files[j] == MainFile) {
			return files[i] == MainFile
		}
		return files[i] < files[j]
	})
	return files
}

// Read 读一个技能的全部文件；太大或不是文本的标成 Binary，不带内容。
func (s Store) Read(name string) []File {
	dir := s.Dir(name)
	var out []File
	for _, rel := range paths(dir) {
		file := File{Path: rel}
		full := filepath.Join(dir, filepath.FromSlash(rel))
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		if info.Size() > maxText {
			file.Binary = true
		} else if raw, err := os.ReadFile(full); err != nil || !agentfs.IsText(raw) {
			file.Binary = true
		} else {
			file.Content = string(raw)
		}
		out = append(out, file)
	}
	return out
}

// cleanPath 整理技能里的文件路径：正斜杠、相对、不跳出技能目录。
func cleanPath(path string) (string, bool) {
	path = strings.TrimLeft(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/"), "/")
	if path == "" || strings.HasSuffix(path, "/") {
		return "", false
	}
	for _, seg := range strings.Split(path, "/") {
		if seg == "" || seg == "." || seg == ".." || seg == ".git" {
			return "", false
		}
	}
	return path, true
}

// Write 把一组文件写成技能目录的内容：没列出来的文件删掉，标成 Binary 的保留原样。
// oldName 不为空且和 name 不同时是改名。SKILL.md 开头的 name、description 会改成和传入的一致：
// Agent 靠它们认技能，两边不能对不上。返回文件数。
func (s Store) Write(name, description, oldName string, files []File) (int, error) {
	if !ValidName(name) {
		return 0, errors.New("名称只能用小写字母、数字、点、横线和下划线")
	}
	if len(files) > MaxFiles {
		return 0, errors.New("技能里的文件太多了")
	}
	keep := map[string]File{}
	hasMain := false
	for _, f := range files {
		path, ok := cleanPath(f.Path)
		if _, dup := keep[path]; !ok || dup {
			return 0, errors.New("文件路径不能为空或重复")
		}
		if path == MainFile {
			hasMain = true
		}
		keep[path] = f
	}
	if !hasMain {
		return 0, errors.New("技能里要有一个 SKILL.md")
	}
	dir := s.Dir(name)
	if oldName != "" && oldName != name {
		if _, err := os.Stat(dir); err == nil {
			return 0, errors.New("已经有一个叫 " + name + " 的技能了")
		}
		if err := os.Rename(s.Dir(oldName), dir); err != nil && !os.IsNotExist(err) {
			return 0, errors.New("改名失败：" + err.Error())
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, errors.New("建不了技能目录：" + err.Error())
	}
	for _, rel := range paths(dir) {
		if _, ok := keep[rel]; !ok {
			_ = os.Remove(filepath.Join(dir, filepath.FromSlash(rel)))
		}
	}
	for path, f := range keep {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if f.Binary {
			if _, err := os.Stat(full); err != nil {
				return 0, errors.New(path + " 不存在，不能标成保留")
			}
			continue
		}
		content := f.Content
		if path == MainFile {
			content = SetFrontmatter(content, name, description)
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return 0, errors.New("写入 " + path + " 失败：" + err.Error())
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return 0, errors.New("写入 " + path + " 失败：" + err.Error())
		}
	}
	agentfs.PruneEmptyDirs(dir)
	return len(keep), nil
}

// Frontmatter 读 SKILL.md 开头的 name 和 description。
func Frontmatter(content string) (name, description string) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", ""
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return "", ""
	}
	for _, line := range strings.Split(content[4:4+end], "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		switch strings.TrimSpace(key) {
		case "name":
			name = value
		case "description":
			description = value
		}
	}
	return name, description
}

// SetFrontmatter 把 SKILL.md 开头的 name、description 换成给定的值，其它键保留。
func SetFrontmatter(content, name, description string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	quote := func(s string) string {
		s = strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), `"`, `'`)
		if strings.ContainsAny(s, ":#[]{}&*!|>%@`") {
			return `"` + s + `"`
		}
		return s
	}
	head := []string{"name: " + name, "description: " + quote(description)}
	body := content
	if strings.HasPrefix(content, "---\n") {
		if end := strings.Index(content[4:], "\n---"); end >= 0 {
			for _, line := range strings.Split(content[4:4+end], "\n") {
				key, _, ok := strings.Cut(line, ":")
				top := !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t")
				if ok && top && (strings.TrimSpace(key) == "name" || strings.TrimSpace(key) == "description") {
					continue
				}
				if strings.TrimSpace(line) != "" {
					head = append(head, line)
				}
			}
			body = strings.TrimPrefix(content[4+end+4:], "\n")
		}
	}
	return "---\n" + strings.Join(head, "\n") + "\n---\n" + body
}

// Found 在一个来源里找到的技能。
type Found struct {
	Dir         string
	Name        string
	Description string
}

// Find 在 root 里找技能：根目录自己带 SKILL.md 就是它；否则往下最多四层，每个带 SKILL.md 的目录算一个。
// 技能名取 SKILL.md 里的 name，不合规时用目录名（根目录用 rootName）整理出来的。
func Find(root, rootName string) []Found {
	var dirs []string
	if _, err := os.Stat(filepath.Join(root, MainFile)); err == nil {
		dirs = []string{root}
	} else {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			rel, _ := filepath.Rel(root, path)
			if rel != "." && strings.Count(filepath.ToSlash(rel), "/") >= 4 {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, MainFile)); err == nil {
				dirs = append(dirs, path)
				return filepath.SkipDir
			}
			return nil
		})
		sort.Strings(dirs)
	}
	var out []Found
	for _, dir := range dirs {
		raw, err := os.ReadFile(filepath.Join(dir, MainFile))
		if err != nil {
			continue
		}
		name, description := Frontmatter(string(raw))
		if !ValidName(name) {
			base := filepath.Base(dir)
			if dir == root && rootName != "" {
				base = rootName
			}
			name = agentfs.Slug(base)
		}
		if !ValidName(name) {
			continue
		}
		out = append(out, Found{Dir: dir, Name: name, Description: description})
	}
	return out
}

// Import 把找到的技能整份复制进来，同名的覆盖。返回文件数。
func (s Store) Import(found Found) (int, error) {
	target := s.Dir(found.Name)
	_ = os.RemoveAll(target)
	return agentfs.CopyTree(found.Dir, target, MaxFiles)
}

// maxArchive 技能压缩包的大小上限，下载的和解出来的都算。
const maxArchive = 64 << 20

// ExtractZip 把上传的技能压缩包解到 dest（dest 要已经存在）。
func ExtractZip(ctx context.Context, data []byte, dest string) error {
	if len(data) > maxArchive {
		return errors.New("压缩包太大了")
	}
	tmp, err := os.CreateTemp("", "momoko-skill-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(data)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return archive.Extract(ctx, tmp.Name(), "skill.zip", dest, maxArchive)
}

// FetchZip 经统一下载管理下载一个技能压缩包（https 链接）解到 dest。
func FetchZip(ctx context.Context, link, dest string) error {
	u, err := url.Parse(strings.TrimSpace(link))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return errors.New("压缩包链接要以 https:// 开头")
	}
	tmp, err := os.CreateTemp("", "momoko-skill-*.zip")
	if err != nil {
		return err
	}
	_ = tmp.Close()
	defer os.Remove(tmp.Name())
	d, err := download.Default.Start(ctx, download.Request{
		URL:     u.String(),
		Dest:    tmp.Name(),
		Title:   "下载技能压缩包",
		Source:  "agent.skill",
		Perm:    string(constant.AgentManage),
		MaxSize: maxArchive,
	})
	if err != nil {
		return err
	}
	if err := d.Wait(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.New("下载压缩包超时了")
		}
		return err
	}
	return archive.Extract(ctx, tmp.Name(), "skill.zip", dest, maxArchive)
}
