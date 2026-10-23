package iqcheck

import (
	"bufio"
	"crypto/rand"
	"io"
	"math/big"
	"strings"
)

// readSSE 逐事件读取 Server-Sent Events 流：空行分隔事件，支持多行 data，忽略注释行；
// fn 返回错误即停止读取并返回该错误。
func readSSE(r io.Reader, fn func(event, data string) error) error {
	br := bufio.NewReaderSize(r, 64<<10)
	var (
		event string
		data  []string
	)
	dispatch := func() error {
		if len(data) == 0 {
			event = ""
			return nil
		}
		err := fn(event, strings.Join(data, "\n"))
		event = ""
		data = data[:0]
		return err
	}
	for {
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			switch {
			case line == "":
				if derr := dispatch(); derr != nil {
					return derr
				}
			case strings.HasPrefix(line, ":"):
				// 注释 / 心跳
			case strings.HasPrefix(line, "event:"):
				event = strings.TrimSpace(line[len("event:"):])
			case strings.HasPrefix(line, "data:"):
				data = append(data, strings.TrimPrefix(line[len("data:"):], " "))
			}
		}
		if err != nil {
			if err == io.EOF {
				return dispatch()
			}
			return err
		}
	}
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
