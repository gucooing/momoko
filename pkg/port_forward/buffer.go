package port_forward

import "sync"

// relayBufferSize 单向数据搬运缓冲区大小。
// 64KB 在吞吐（少一次 syscall 搬更多数据）与每连接内存占用之间取平衡。
const relayBufferSize = 64 * 1024

// bufPool 复用搬运缓冲区。
// 每条 TCP 连接的两个方向、每条 UDP 会话的回程读取各需一份缓冲；
// 高并发短连接场景下由 sync.Pool 在各 P 本地缓存，避免反复分配 64KB 造成 GC 压力。
var bufPool = sync.Pool{
	New: func() any {
		return make([]byte, relayBufferSize)
	},
}

// getRelayBuf 取一份长度恒为 relayBufferSize 的缓冲。
// b[:cap(b)] 做防御性归一：即便归还时被 reslice，取出时也能还原到满长度。
func getRelayBuf() []byte {
	b := bufPool.Get().([]byte)
	return b[:cap(b)]
}

// putRelayBuf 归还缓冲。仅当底层连接不再引用该切片时方可归还。
func putRelayBuf(b []byte) {
	bufPool.Put(b)
}
