package servercore

import (
	"errors"
	"sync"
)

// ServerManager 使用内存管理多个服务端实例。
type ServerManager struct {
	mu      sync.RWMutex
	servers map[string]*Server
}

// NewServerManager 创建一个最小可用的管理器。
func NewServerManager() *ServerManager {
	return &ServerManager{
		servers: make(map[string]*Server),
	}
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
func (m *ServerManager) Get(id string) (*Server, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, ok := m.servers[id]
	if !ok {
		return nil, errors.New("服务端实例不存在")
	}
	return server, nil
}

// Start 启动指定实例。
func (m *ServerManager) Start(id string) error {
	server, err := m.Get(id)
	if err != nil {
		return err
	}
	return server.Start()
}

// Stop 停止指定实例。
func (m *ServerManager) Stop(id string) error {
	server, err := m.Get(id)
	if err != nil {
		return err
	}
	return server.Stop()
}

// Restart 重启指定实例。
func (m *ServerManager) Restart(id string) error {
	server, err := m.Get(id)
	if err != nil {
		return err
	}
	return server.Restart()
}

// Send 向指定实例写入一条输入。
func (m *ServerManager) Send(id, input string) error {
	server, err := m.Get(id)
	if err != nil {
		return err
	}
	return server.Send(input)
}

// Subscribe 订阅指定实例的实时日志。
func (m *ServerManager) Subscribe(id string) (<-chan LogEntry, func(), error) {
	server, err := m.Get(id)
	if err != nil {
		return nil, nil, err
	}
	ch, cancel := server.Subscribe()
	return ch, cancel, nil
}

// RecentLogs 返回指定实例的最近日志。
func (m *ServerManager) RecentLogs(id string) ([]LogEntry, error) {
	server, err := m.Get(id)
	if err != nil {
		return nil, err
	}
	return server.RecentLogs(), nil
}
