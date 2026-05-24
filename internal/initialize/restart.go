package initialize

import "sync"

type restartSignal struct {
	once sync.Once
	ch   chan struct{}
}

var restart = &restartSignal{
	ch: make(chan struct{}),
}

func RequestRestart() {
	restart.once.Do(func() {
		close(restart.ch)
	})
}

func RestartRequested() <-chan struct{} {
	return restart.ch
}
