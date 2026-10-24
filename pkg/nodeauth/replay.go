package nodeauth

import (
	"sync"
	"time"
)

// ReplayGuard 记录时间窗口内已通过校验的 (key-id, nonce)，拒绝重复使用。
// 只记录验签成功的请求，伪造请求无法占用内存。
type ReplayGuard struct {
	mu        sync.Mutex
	seen      map[string]time.Time
	lastSweep time.Time
}

func NewReplayGuard() *ReplayGuard {
	return &ReplayGuard{seen: make(map[string]time.Time)}
}

// Use 登记一次 nonce；窗口内重复出现返回 ErrReplay。
// 时间戳在 now±MaxClockSkew 内才会被接受，记录保留 2×MaxClockSkew 即可覆盖其全部有效期。
func (g *ReplayGuard) Use(keyID, nonce string, now time.Time) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if now.Sub(g.lastSweep) > time.Minute {
		for k, expiry := range g.seen {
			if !now.Before(expiry) {
				delete(g.seen, k)
			}
		}
		g.lastSweep = now
	}
	k := keyID + "\x00" + nonce
	if expiry, ok := g.seen[k]; ok && now.Before(expiry) {
		return ErrReplay
	}
	g.seen[k] = now.Add(2 * MaxClockSkew)
	return nil
}
