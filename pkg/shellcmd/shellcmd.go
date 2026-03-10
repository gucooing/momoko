package shellcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

type Options struct {
	Stdin []byte
	Dir   string
	Env   []string

	// StreamBuffer is the channel buffer size for stdout/stderr events.
	// If <= 0, a default value is used.
	StreamBuffer int
	// BlockWhenStreamBlocked uses blocking channel send for live stream events.
	// Default false (recommended) drops live chunks on backpressure to avoid deadlock.
	BlockWhenStreamBlocked bool
}

type Result struct {
	Stdout             string
	Stderr             string
	ExitCode           int
	DroppedStdoutBytes int
	DroppedStderrBytes int
}

type Session struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser

	stdoutCh chan []byte
	stderrCh chan []byte

	dropWhenBlocked bool

	stdoutBuf bytes.Buffer
	stderrBuf bytes.Buffer

	mu            sync.Mutex
	stdinClosed   bool
	pumpErr       error
	droppedStdout int
	droppedStderr int

	waitCh chan waitResult
	pumpWG sync.WaitGroup
	waited sync.Once
	final  waitResult
}

type waitResult struct {
	result Result
	err    error
}

const (
	defaultStreamBuffer = 128
)

func Start(ctx context.Context, program string, args []string, opts Options) (*Session, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, program, args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe for %q failed: %w", program, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe for %q failed: %w", program, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("create stderr pipe for %q failed: %w", program, err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %q failed: %w", program, err)
	}

	bufferSize := opts.StreamBuffer
	if bufferSize <= 0 {
		bufferSize = defaultStreamBuffer
	}
	session := &Session{
		cmd:             cmd,
		stdin:           stdin,
		stdoutCh:        make(chan []byte, bufferSize),
		stderrCh:        make(chan []byte, bufferSize),
		dropWhenBlocked: !opts.BlockWhenStreamBlocked,
		waitCh:          make(chan waitResult, 1),
	}

	session.pumpWG.Add(2)
	go session.pump(stdout, session.stdoutCh, &session.stdoutBuf, true)
	go session.pump(stderr, session.stderrCh, &session.stderrBuf, false)
	go session.wait(program)

	if len(opts.Stdin) > 0 {
		if _, err := session.Write(opts.Stdin); err != nil {
			_ = session.Kill()
			_, _ = session.Wait()
			return nil, fmt.Errorf("write initial stdin for %q failed: %w", program, err)
		}
	}

	return session, nil
}

func Run(ctx context.Context, program string, args []string, opts Options) (Result, error) {
	session, err := Start(ctx, program, args, opts)
	if err != nil {
		return Result{}, err
	}
	_ = session.CloseStdin()
	return session.Wait()
}

func (s *Session) Stdout() <-chan []byte {
	return s.stdoutCh
}

func (s *Session) Stderr() <-chan []byte {
	return s.stderrCh
}

func (s *Session) Write(input []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stdinClosed {
		return 0, errors.New("stdin already closed")
	}
	return s.stdin.Write(input)
}

func (s *Session) WriteString(input string) (int, error) {
	return s.Write([]byte(input))
}

func (s *Session) CloseStdin() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stdinClosed {
		return nil
	}
	s.stdinClosed = true
	return s.stdin.Close()
}

func (s *Session) Kill() error {
	if s.cmd.Process == nil {
		return nil
	}
	return s.cmd.Process.Kill()
}

func (s *Session) Wait() (Result, error) {
	s.waited.Do(func() {
		r, ok := <-s.waitCh
		if !ok {
			s.final = waitResult{err: errors.New("wait channel closed unexpectedly")}
			return
		}
		s.final = r
	})
	return s.final.result, s.final.err
}

func (s *Session) wait(program string) {
	waitErr := s.cmd.Wait()
	s.pumpWG.Wait()

	s.mu.Lock()
	pumpErr := s.pumpErr
	droppedStdout := s.droppedStdout
	droppedStderr := s.droppedStderr
	s.mu.Unlock()

	result := Result{
		Stdout:             s.stdoutBuf.String(),
		Stderr:             s.stderrBuf.String(),
		ExitCode:           exitCode(waitErr),
		DroppedStdoutBytes: droppedStdout,
		DroppedStderrBytes: droppedStderr,
	}

	if waitErr != nil {
		s.waitCh <- waitResult{
			result: result,
			err:    fmt.Errorf("run %q failed: %w", program, waitErr),
		}
		close(s.waitCh)
		return
	}
	if pumpErr != nil {
		s.waitCh <- waitResult{
			result: result,
			err:    fmt.Errorf("capture output for %q failed: %w", program, pumpErr),
		}
		close(s.waitCh)
		return
	}
	s.waitCh <- waitResult{result: result}
	close(s.waitCh)
}

func (s *Session) pump(r io.ReadCloser, streamCh chan []byte, dst *bytes.Buffer, isStdout bool) {
	defer s.pumpWG.Done()
	defer close(streamCh)
	defer r.Close()

	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			dst.Write(chunk)
			if s.dropWhenBlocked {
				select {
				case streamCh <- chunk:
				default:
					s.mu.Lock()
					if isStdout {
						s.droppedStdout += len(chunk)
					} else {
						s.droppedStderr += len(chunk)
					}
					s.mu.Unlock()
				}
			} else {
				streamCh <- chunk
			}
		}
		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) {
			return
		}
		s.setPumpErr(err)
		return
	}
}

func (s *Session) setPumpErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pumpErr == nil {
		s.pumpErr = err
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
