package port_forward

import (
	"context"
	"errors"
	"sync"
)

var (
	ExampleREPEAT  = errors.New("端口转发已启动")
	NotExample     = errors.New("端口转发未启用")
	NOTProtocol    = errors.New("暂未实现该协议")
	NETListenError = errors.New("端口被占用或不可用")
)

type Protocol int

const (
	TCP Protocol = iota
	UDP
)

type Option struct {
	ID            string // 唯一id
	Protocol      Protocol
	ListenAddress string // 监听地址
	ListenPort    int    // 监听端口
	TargetAddress string // 目标地址
	TargetPort    int    // 目标端口
}

type PortForward interface {
	Option() *Option
	Start() error
	Stop()
}

// 实例管理
type Manager struct {
	sync     sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	examples map[string]PortForward // 全部已启动的实例
}

func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		sync:     sync.RWMutex{},
		ctx:      ctx,
		cancel:   cancel,
		examples: make(map[string]PortForward),
	}
}

// 注册端口转发
func (m *Manager) RegisterExample(opt *Option) error {
	m.sync.RLock()
	if _, ok := m.examples[opt.ID]; ok {
		m.sync.RUnlock()
		return ExampleREPEAT
	}
	m.sync.RUnlock()

	var example PortForward
	var err error
	switch opt.Protocol {
	case TCP:
		example, err = NewTCPPortForward(opt, m.ctx)
	case UDP:
		return NOTProtocol
	default:
		return NOTProtocol
	}
	if err != nil {
		return err
	}
	m.sync.Lock()
	m.examples[opt.ID] = example
	m.sync.Unlock()
	go m.Run(example)
	return nil
}

// 卸载端口转发
func (m *Manager) UnRegisterExample(id string) {
	m.sync.RLock()
	example, ok := m.examples[id]
	if !ok {
		m.sync.RUnlock()
		return
	}
	m.sync.RUnlock()
	example.Stop()
	m.sync.Lock()
	defer m.sync.Unlock()
	delete(m.examples, id)
	return
}

// 重载端口转发
func (m *Manager) Retry(opt *Option) error {
	m.sync.Lock()
	example, ok := m.examples[opt.ID]
	if ok {
		example.Stop()
		delete(m.examples, opt.ID)
	}
	m.sync.Unlock()

	return m.RegisterExample(opt)
}

func (m *Manager) Run(example PortForward) {
	select {
	case <-m.ctx.Done():
		return
	default:
	}

	err := example.Start()
	if err != nil {
		// TODO
		return
	}
}

func (m *Manager) Running(id string) bool {
	m.sync.RLock()
	defer m.sync.RUnlock()
	_, ok := m.examples[id]
	return ok
}

func (m *Manager) Stop() {
	m.cancel()
}
