package iqcheck

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
)

var errStopSSE = errors.New("sse: stop")

const maxSSEEventSize = 8 << 20

// readSSE frames events independently of protocol semantics. It accepts LF,
// CRLF, CR, a leading BOM, comments and multiline data. EOF flushes a final
// unterminated event for relay compatibility; callers still require a terminal
// protocol event before treating EOF as success.
func readSSE(r io.Reader, fn func(event, data string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64<<10), maxSSEEventSize)
	skipLF := false
	scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if len(data) == 0 {
			return 0, nil, nil
		}
		offset := 0
		if skipLF {
			skipLF = false
			if data[0] == '\n' {
				data = data[1:]
				offset = 1
			}
		}
		if len(data) == 0 {
			return offset, nil, nil
		}
		if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
			skipLF = data[i] == '\r'
			return offset + i + 1, data[:i], nil
		}
		if atEOF {
			return offset + len(data), data, nil
		}
		return offset, nil, nil
	})
	var event string
	var data []string
	size := 0
	first := true
	dispatch := func() error {
		var err error
		if len(data) > 0 {
			err = fn(event, strings.Join(data, "\n"))
		}
		event, data, size = "", nil, 0
		return err
	}
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			line = strings.TrimPrefix(line, "\uFEFF")
			first = false
		}
		if line == "" {
			if err := dispatch(); err != nil {
				if errors.Is(err, errStopSSE) {
					return nil
				}
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		size += len(line) + 1
		if size > maxSSEEventSize {
			return errors.New("上游 SSE 事件超过大小限制")
		}
		switch field {
		case "event":
			event = value
		case "data":
			data = append(data, value)
			// This marker has no continuation; tolerate a missing blank line.
			if len(data) == 1 && strings.TrimSpace(value) == "[DONE]" {
				if err := dispatch(); err != nil {
					if errors.Is(err, errStopSSE) {
						return nil
					}
					return err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取上游事件流失败: %w", err)
	}
	err := dispatch()
	if errors.Is(err, errStopSSE) {
		return nil
	}
	return err
}

const slugAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomSlug 生成 n 位由字母数字组成的随机路径片段。
func RandomSlug(n int) string {
	if n <= 0 {
		n = 12
	}
	b := make([]byte, n)
	max := big.NewInt(int64(len(slugAlphabet)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			// crypto/rand 失败极罕见，退化为固定字符避免 panic。
			b[i] = slugAlphabet[0]
			continue
		}
		b[i] = slugAlphabet[idx.Int64()]
	}
	return string(b)
}
