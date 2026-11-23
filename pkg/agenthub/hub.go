// Package agenthub 把会话事件分发给打开着页面的各个连接。
// 每个连接属于一个用户，可以打开（订阅）他自己的一个会话，有 agent:admin 的连接还可以关注所有人的会话状态（Agent 管理页用）；
// 收得太慢的连接会被断开，由页面重连后按修订号补齐。连接按会话和用户建了索引，发一条消息不用扫所有连接。
package agenthub

import "sync"

// Conn 一个页面连接。
type Conn struct {
	// UserID 连接属于谁；Manage 表示他能关注所有人的会话状态（Agent 管理页），打开会话仍只能打开自己的。
	UserID string
	Manage bool

	mu     sync.Mutex
	closed bool
	out    chan []byte

	// 以下由 Hub 在 Hub.mu 里维护。
	session string
	watch   bool
}

// Out 要发给页面的消息；连接被断开时关闭。
func (c *Conn) Out() <-chan []byte { return c.out }

// Push 给这个连接发一条消息。缓冲满了说明页面收得太慢：断开它。
func (c *Conn) Push(raw []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	select {
	case c.out <- raw:
	default:
		c.closed = true
		close(c.out)
	}
}

func (c *Conn) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.out)
	}
}

type set map[*Conn]struct{}

// Hub 全部连接。
type Hub struct {
	mu        sync.Mutex
	bySession map[string]set
	byUser    map[string]set
	watchers  set
}

func New() *Hub {
	return &Hub{bySession: map[string]set{}, byUser: map[string]set{}, watchers: set{}}
}

func add(m map[string]set, key string, c *Conn) {
	if m[key] == nil {
		m[key] = set{}
	}
	m[key][c] = struct{}{}
}

func drop(m map[string]set, key string, c *Conn) {
	if s := m[key]; s != nil {
		delete(s, c)
		if len(s) == 0 {
			delete(m, key)
		}
	}
}

// Add 登记一个新连接。
func (h *Hub) Add(userID string, manage bool) *Conn {
	c := &Conn{UserID: userID, Manage: manage, out: make(chan []byte, 512)}
	h.mu.Lock()
	add(h.byUser, userID, c)
	h.mu.Unlock()
	return c
}

// Remove 连接断开。
func (h *Hub) Remove(c *Conn) {
	h.mu.Lock()
	drop(h.byUser, c.UserID, c)
	drop(h.bySession, c.session, c)
	delete(h.watchers, c)
	h.mu.Unlock()
	c.close()
}

// Subscribe 让这个连接开始接收某个会话的事件；空字符串表示不再接收。
func (h *Hub) Subscribe(c *Conn, sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	drop(h.bySession, c.session, c)
	c.session = sessionID
	if sessionID != "" {
		add(h.bySession, sessionID, c)
	}
}

// Watch 有 agent:admin 的连接关注（或不再关注）所有人的会话状态变化。
func (h *Hub) Watch(c *Conn, on bool) {
	if !c.Manage {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	c.watch = on
	if on {
		h.watchers[c] = struct{}{}
	} else {
		delete(h.watchers, c)
	}
}

// ToAll 发给所有连接（比如配置变了，页面要重读）。
func (h *Hub) ToAll(raw []byte) {
	h.mu.Lock()
	var targets []*Conn
	for _, s := range h.byUser {
		for c := range s {
			targets = append(targets, c)
		}
	}
	h.mu.Unlock()
	for _, c := range targets {
		c.Push(raw)
	}
}

// ToSession 发给打开着这个会话的连接（对话事件）。
func (h *Hub) ToSession(sessionID string, raw []byte) {
	h.mu.Lock()
	targets := make([]*Conn, 0, len(h.bySession[sessionID]))
	for c := range h.bySession[sessionID] {
		targets = append(targets, c)
	}
	h.mu.Unlock()
	for _, c := range targets {
		c.Push(raw)
	}
}

// ToSessionOrUser 会话本身变了（状态、标题、选项、删除）：发给会话主人的所有连接（会话栏）、
// 打开着这个会话的连接，和关注所有人的管理页。
func (h *Hub) ToSessionOrUser(sessionID, userID string, raw []byte) {
	h.mu.Lock()
	targets := set{}
	for c := range h.byUser[userID] {
		targets[c] = struct{}{}
	}
	for c := range h.bySession[sessionID] {
		targets[c] = struct{}{}
	}
	for c := range h.watchers {
		targets[c] = struct{}{}
	}
	h.mu.Unlock()
	for c := range targets {
		c.Push(raw)
	}
}
