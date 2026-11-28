package servercore

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// 托管进程（见 host.go）与面板之间的协议：本地套接字上的帧，每帧 = 1 字节类型 + 4 字节长度（大端）+ 内容。
// 协议要前后兼容：在线更新后，新面板要接回旧版本起的托管进程。只能加新的帧类型、给 JSON 加字段。
const hostProtoVersion = 1

// 面板 → 托管进程
const (
	frameHello  byte = 'h' // JSON hostHelloReq：连上后第一帧，带口令
	frameInput  byte = 'i' // 原始键盘输入
	frameResize byte = 'r' // JSON hostResize
	frameStop   byte = 's' // 1 字节：0 正常结束（SIGTERM），1 强制（SIGKILL）
	frameClear  byte = 'c' // 清掉托管进程存着的输出
	frameQuit   byte = 'q' // 收到退出帧后回这一帧：托管进程可以走了
)

// 托管进程 → 面板
const (
	frameWelcome byte = 'H' // JSON hostWelcome：口令对了，随后是存着的输出，进程已退出时再跟一帧退出
	frameOutput  byte = 'O' // 原始输出
	frameExit    byte = 'X' // JSON hostExitMsg：进程退出了
	frameError   byte = 'E' // 文字：拒绝连接的原因，随后断开
)

// maxFrameSize 单帧上限（输出按 32 KiB 切块，留足余量）。
const maxFrameSize = 1 << 20

type hostHelloReq struct {
	Version int    `json:"v"`
	Token   string `json:"token"`
}

type hostWelcome struct {
	Version int   `json:"v"`
	PID     int   `json:"pid"`
	Start   int64 `json:"start"` // 进程启动时间（Unix 毫秒）
	Exited  bool  `json:"exited"`
}

type hostResize struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

type hostExitMsg struct {
	Code int   `json:"code"`
	At   int64 `json:"at"` // 退出时间（Unix 毫秒）
}

// hostSpec 面板起托管进程时经标准输入交给它的启动参数（环境变量里可能有密钥，不放命令行）。
type hostSpec struct {
	ID      string   `json:"id"`
	Path    string   `json:"path"`
	Args    []string `json:"args"`
	Dir     string   `json:"dir"`
	Env     []string `json:"env"` // 在托管进程自己的环境之上追加
	Cols    int      `json:"cols"`
	Rows    int      `json:"rows"`
	Socket  string   `json:"socket"`
	KeyFile string   `json:"keyFile"`
	Token   string   `json:"token"`
}

// hostReady 托管进程起好（或起不来）后在标准输出写的一行。
type hostReady struct {
	PID   int    `json:"pid,omitempty"`
	Error string `json:"error,omitempty"`
	Stage string `json:"stage,omitempty"` // 出错的阶段：hostStageSetup / hostStageStart
}

const (
	hostStageSetup = "setup" // 托管进程自己没准备好（套接字、伪终端）：面板改为直接运行
	hostStageStart = "start" // 应用起不来（命令不对等）：直接报错
)

func writeFrame(w io.Writer, typ byte, payload []byte) error {
	if len(payload) > maxFrameSize {
		return fmt.Errorf("帧太大: %d", len(payload))
	}
	buf := make([]byte, 5+len(payload))
	buf[0] = typ
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(payload)))
	copy(buf[5:], payload)
	_, err := w.Write(buf)
	return err
}

func writeJSONFrame(w io.Writer, typ byte, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return writeFrame(w, typ, payload)
}

func readFrame(r io.Reader) (byte, []byte, error) {
	var head [5]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return 0, nil, err
	}
	size := binary.BigEndian.Uint32(head[1:5])
	if size > maxFrameSize {
		return 0, nil, errors.New("帧太大")
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, err
	}
	return head[0], payload, nil
}
