package agentfs

import (
	"context"
	"strings"
	"sync"
	"time"
)

// Index 按文件名搜一个目录（「@ 引用文件」用）。列一次目录很贵（大仓库几万个文件），
// 而输入框里每敲一个字就搜一次，所以列出来的文件留一小段时间，这段时间里同一个目录的搜索都用这一份。
type Index struct {
	ttl  time.Duration
	list func(ctx context.Context, dir string) []string

	mu      sync.Mutex
	entries map[string]*indexEntry
}

type indexEntry struct {
	ready chan struct{}
	files []string
	at    time.Time
}

// NewIndex list 列出 dir 里的全部文件（相对路径，正斜杠）。
func NewIndex(ttl time.Duration, list func(ctx context.Context, dir string) []string) *Index {
	return &Index{ttl: ttl, list: list, entries: map[string]*indexEntry{}}
}

// Search 文件名里含 keyword 的排前面，其次是路径里含的，最多 limit 条；keyword 为空时按列出的顺序给前 limit 个。
func (x *Index) Search(ctx context.Context, dir, keyword string, limit int) []string {
	files := x.files(ctx, dir)
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	var byName, byPath []string
	for _, path := range files {
		lower := strings.ToLower(path)
		base := lower[strings.LastIndexByte(lower, '/')+1:]
		switch {
		case keyword == "" || strings.Contains(base, keyword):
			byName = append(byName, path)
			if len(byName) >= limit {
				return byName
			}
		case len(byPath) < limit && strings.Contains(lower, keyword):
			byPath = append(byPath, path)
		}
	}
	out := append(byName, byPath...)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// files 这个目录的文件列表：有没过期的就用，正在列的就等它，否则列一次。顺手清掉过期的目录。
func (x *Index) files(ctx context.Context, dir string) []string {
	now := time.Now()
	x.mu.Lock()
	for key, e := range x.entries {
		if !e.at.IsZero() && now.Sub(e.at) > x.ttl {
			delete(x.entries, key)
		}
	}
	e := x.entries[dir]
	if e == nil {
		e = &indexEntry{ready: make(chan struct{})}
		x.entries[dir] = e
		x.mu.Unlock()
		files := x.list(ctx, dir)
		x.mu.Lock()
		e.files, e.at = files, time.Now()
		x.mu.Unlock()
		close(e.ready)
		return files
	}
	x.mu.Unlock()
	select {
	case <-e.ready:
		return e.files
	case <-ctx.Done():
		return nil
	}
}
