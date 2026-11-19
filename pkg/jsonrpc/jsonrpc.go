// Package jsonrpc 是 JSON-RPC 2.0：消息格式，加上跑在一对读写流上、一行一条消息的连接。
// ACP（Agent 进程的 stdin / stdout）、MCP 的 stdio 传输都用 Conn；MCP 的 HTTP 传输只用消息格式。
package jsonrpc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// 标准错误码。
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// maxLine 一条消息最长多少字节，超过的整行丢掉：免得被刷屏的输出撑爆内存。
const maxLine = 16 << 20

// Error 是 JSON-RPC 的错误对象。
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s (%d)", e.Message, e.Code)
}

// NewError 生成一个错误对象。
func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

// ErrClosed 连接已经关闭（对端进程退出，或主动关闭）。
var ErrClosed = errors.New("jsonrpc: connection closed")

// Message 一条消息：请求（有 Method 和 ID）、通知（只有 Method）或应答（只有 ID）。
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// HasID 这条消息带不带编号。
func (m *Message) HasID() bool {
	return len(m.ID) > 0 && string(m.ID) != "null"
}

// IDKey 编号的比较键：有的对端把数字编号当字符串还回来，两种写法算同一个。
func (m *Message) IDKey() string {
	return strings.Trim(string(m.ID), `"`)
}

// Outcome 应答里的结果；错误应答返回 *Error。
func (m *Message) Outcome() (json.RawMessage, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Result, nil
}

// Request 生成一个请求；id 为 0 时是通知。
func Request(id int64, method string, params any) (*Message, error) {
	raw, err := marshalParams(params)
	if err != nil {
		return nil, err
	}
	msg := &Message{JSONRPC: "2.0", Method: method, Params: raw}
	if id != 0 {
		msg.ID = json.RawMessage(strconv.FormatInt(id, 10))
	}
	return msg, nil
}

// Reply 生成对某个请求的应答。
func Reply(id json.RawMessage, result any, rpcErr *Error) *Message {
	msg := &Message{JSONRPC: "2.0", ID: id}
	if rpcErr != nil {
		msg.Error = rpcErr
		return msg
	}
	raw, err := json.Marshal(result)
	switch {
	case err != nil:
		msg.Error = NewError(CodeInternalError, err.Error())
	case string(raw) == "null":
		msg.Result = json.RawMessage("{}")
	default:
		msg.Result = raw
	}
	return msg
}

// Parse 解一条消息；不是 JSON 对象（启动横幅、日志）时返回 nil。
func Parse(line []byte) *Message {
	line = bytes.TrimSpace(line)
	if len(line) == 0 || line[0] != '{' {
		return nil
	}
	msg := &Message{}
	if err := json.Unmarshal(line, msg); err != nil {
		return nil
	}
	return msg
}

func marshalParams(params any) (json.RawMessage, error) {
	if params == nil {
		return json.RawMessage("{}"), nil
	}
	return json.Marshal(params)
}

// Handler 处理对端发来的通知和请求。
//
// 两个方法都在同一个协程里、按消息到达的顺序依次调用，所以通知之间、通知和随后的请求之间顺序不会乱。
// 实现里不能阻塞，也不能同步地调用 Conn.Call（自己这次调用的应答排在后面，会死锁）；
// 需要等人回答的请求先返回，之后再调用 reply。
type Handler interface {
	Notification(method string, params json.RawMessage)
	// Request 的 reply 只能调用一次；err 不为空时回复错误。
	Request(method string, params json.RawMessage, reply func(result any, err *Error))
}

// Conn 一条跑在读写流上的连接，一行一条消息。
type Conn struct {
	handler Handler

	writeMu sync.Mutex
	w       io.Writer

	mu      sync.Mutex
	nextID  int64
	pending map[string]chan *Message
	err     error

	// 收到的消息先进队列，再由一个协程按顺序分发：应答、通知、请求三者之间的先后保持不变。
	queueMu sync.Mutex
	queue   []*Message
	wake    chan struct{}

	done      chan struct{}
	closeOnce sync.Once
}

// NewConn 在 r / w 上建立连接并开始读消息。r 读到末尾或出错时连接关闭。h 可以为 nil。
func NewConn(r io.Reader, w io.Writer, h Handler) *Conn {
	c := &Conn{
		handler: h,
		w:       w,
		pending: make(map[string]chan *Message),
		wake:    make(chan struct{}, 1),
		done:    make(chan struct{}),
	}
	go c.readLoop(r)
	go c.dispatchLoop()
	return c
}

// Done 在连接关闭后关闭。
func (c *Conn) Done() <-chan struct{} { return c.done }

// Err 连接关闭的原因；还没关闭时为 nil。
func (c *Conn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Close 关闭连接：还在等应答的调用都以 ErrClosed 结束。它不关闭底层的流。
func (c *Conn) Close() { c.shutdown(ErrClosed) }

func (c *Conn) shutdown(err error) {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.err = err
		pending := c.pending
		c.pending = make(map[string]chan *Message)
		c.mu.Unlock()
		close(c.done)
		for _, ch := range pending {
			close(ch)
		}
	})
}

func (c *Conn) readLoop(r io.Reader) {
	reader := bufio.NewReaderSize(r, 64*1024)
	for {
		line, err := readLine(reader)
		if msg := Parse(line); msg != nil {
			c.queueMu.Lock()
			c.queue = append(c.queue, msg)
			c.queueMu.Unlock()
			select {
			case c.wake <- struct{}{}:
			default:
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = ErrClosed
			}
			c.shutdown(err)
			return
		}
	}
}

// readLine 读一行，不含换行符；超过 maxLine 的行整行丢掉，返回空行。
func readLine(r *bufio.Reader) ([]byte, error) {
	var line []byte
	over := false
	for {
		chunk, err := r.ReadSlice('\n')
		if !over {
			line = append(line, chunk...)
			if len(line) > maxLine {
				over, line = true, nil
			}
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return line, err
	}
}

func (c *Conn) dispatchLoop() {
	for {
		c.queueMu.Lock()
		batch := c.queue
		c.queue = nil
		c.queueMu.Unlock()
		for _, msg := range batch {
			c.dispatch(msg)
		}
		if len(batch) > 0 {
			continue
		}
		select {
		case <-c.wake:
		case <-c.done:
			// 关闭前已经读到的消息照常处理完。
			c.queueMu.Lock()
			rest := c.queue
			c.queue = nil
			c.queueMu.Unlock()
			for _, msg := range rest {
				c.dispatch(msg)
			}
			return
		}
	}
}

func (c *Conn) dispatch(msg *Message) {
	switch {
	case msg.Method != "" && msg.HasID():
		id := msg.ID
		var once sync.Once
		reply := func(result any, rpcErr *Error) {
			once.Do(func() { _ = c.write(Reply(id, result, rpcErr)) })
		}
		if c.handler == nil {
			reply(nil, NewError(CodeMethodNotFound, "method not found: "+msg.Method))
			return
		}
		c.handler.Request(msg.Method, msg.Params, reply)
	case msg.Method != "":
		if c.handler != nil {
			c.handler.Notification(msg.Method, msg.Params)
		}
	case msg.HasID():
		key := msg.IDKey()
		c.mu.Lock()
		ch, ok := c.pending[key]
		if ok {
			delete(c.pending, key)
		}
		c.mu.Unlock()
		if ok {
			ch <- msg
		}
	}
}

func (c *Conn) write(msg *Message) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	select {
	case <-c.done:
		return c.closedErr()
	default:
	}
	if _, err := c.w.Write(raw); err != nil {
		c.shutdown(err)
		return err
	}
	return nil
}

func (c *Conn) closedErr() error {
	if err := c.Err(); err != nil {
		return err
	}
	return ErrClosed
}

// Call 发一个请求并等应答，result 为 nil 时不解结果。ctx 结束时返回 ctx 的错误，但不撤回已经发出的请求。
func (c *Conn) Call(ctx context.Context, method string, params, result any) error {
	c.mu.Lock()
	if c.err != nil {
		err := c.err
		c.mu.Unlock()
		return err
	}
	c.nextID++
	id := c.nextID
	key := strconv.FormatInt(id, 10)
	ch := make(chan *Message, 1)
	c.pending[key] = ch
	c.mu.Unlock()

	msg, err := Request(id, method, params)
	if err == nil {
		err = c.write(msg)
	}
	if err != nil {
		c.mu.Lock()
		delete(c.pending, key)
		c.mu.Unlock()
		return err
	}

	select {
	case reply, ok := <-ch:
		if !ok {
			return c.closedErr()
		}
		raw, err := reply.Outcome()
		if err != nil {
			return err
		}
		if result == nil || len(raw) == 0 {
			return nil
		}
		return json.Unmarshal(raw, result)
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, key)
		c.mu.Unlock()
		return ctx.Err()
	}
}

// Notify 发一个通知，不等应答。
func (c *Conn) Notify(method string, params any) error {
	msg, err := Request(0, method, params)
	if err != nil {
		return err
	}
	return c.write(msg)
}
