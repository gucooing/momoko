package servercore

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ServerManager 使用内存管理多个服务端实例。
type ServerManager struct {
	mu      sync.RWMutex
	servers map[string]*Server
}

// NewServerManager 创建一个最小可用的管理器。
func NewServerManager() *ServerManager {
	return &ServerManager{servers: make(map[string]*Server)}
}

// Shutdown 并发停止当前所有实例，超时后仅对未退出的实例强制停止。
func (m *ServerManager) Shutdown(timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultStopTimeout
	}

	servers := m.snapshotServers()
	if len(servers) == 0 {
		return nil
	}

	type shutdownResult struct {
		id  string
		err error
	}

	results := make(chan shutdownResult, len(servers)*2)
	pending := make(map[string]*Server, len(servers))
	for _, server := range servers {
		pending[server.ID()] = server
		go func(server *Server) {
			results <- shutdownResult{id: server.ID(), err: server.Stop()}
		}(server)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	forced := false
	errs := make([]error, 0)

	for len(pending) > 0 {
		select {
		case result := <-results:
			if _, ok := pending[result.id]; !ok {
				continue
			}
			delete(pending, result.id)
			if result.err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", result.id, result.err))
			}
		case <-timer.C:
			if forced {
				continue
			}
			forced = true
			for _, server := range pending {
				go func(server *Server) {
					results <- shutdownResult{id: server.ID(), err: server.ForceStop()}
				}(server)
			}
		}
	}

	return errors.Join(errs...)
}

// Create 创建并注册一个服务端实例。
func (m *ServerManager) Create(cfg ServerConfig) (*Server, error) {
	server, err := NewServer(cfg)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.servers[server.ID()]; ok {
		return nil, errors.New("服务端实例已存在")
	}

	m.servers[server.ID()] = server
	return server, nil
}

// Get 返回指定实例。
func (m *ServerManager) Get(id string) (*Server, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, ok := m.servers[id]
	if !ok {
		return nil, false
	}
	return server, true
}

// Start 启动指定实例。
func (m *ServerManager) Start(id string) error {
	server, ok := m.Get(id)
	if !ok {
		return errors.New("服务端实例不存在")
	}
	return server.Start()
}

// Stop 优雅停止指定实例。
func (m *ServerManager) Stop(id string) error {
	server, ok := m.Get(id)
	if !ok {
		return errors.New("服务端实例不存在")
	}
	return server.Stop()
}

// ForceStop 强制停止指定实例。
func (m *ServerManager) ForceStop(id string) error {
	server, ok := m.Get(id)
	if !ok {
		return errors.New("服务端实例不存在")
	}
	return server.ForceStop()
}

// Restart 优雅重启指定实例（先停再启）。
func (m *ServerManager) Restart(id string) error {
	server, ok := m.Get(id)
	if !ok {
		return errors.New("服务端实例不存在")
	}
	return server.Restart()
}

// ForceRestart 强制重启指定实例。
func (m *ServerManager) ForceRestart(id string) error {
	server, ok := m.Get(id)
	if !ok {
		return errors.New("服务端实例不存在")
	}
	return server.ForceRestart()
}

// Send 向指定实例写入一条输入。
func (m *ServerManager) Send(id, input string) error {
	server, ok := m.Get(id)
	if !ok {
		return errors.New("服务端实例不存在")
	}
	return server.Send(input)
}

// Subscribe 订阅指定实例的实时日志。
func (m *ServerManager) Subscribe(id string) (<-chan LogEntry, func(), error) {
	server, ok := m.Get(id)
	if !ok {
		return nil, nil, errors.New("服务端实例不存在")
	}
	ch, cancel := server.Subscribe()
	return ch, cancel, nil
}

// RecentLogs 返回指定实例的最近日志。
func (m *ServerManager) RecentLogs(id string) ([]LogEntry, error) {
	server, ok := m.Get(id)
	if !ok {
		return nil, errors.New("服务端实例不存在")
	}
	return server.RecentLogs(), nil
}

func (m *ServerManager) snapshotServers() []*Server {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]*Server, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	return servers
}
