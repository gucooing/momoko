// Package logtail 从文件末尾向前读取文本行（分块读，不把整个文件读进内存），
// 用于查看大日志文件的最近记录并向前翻页。
package logtail

import (
	"bytes"
	"io"
	"os"
)

const chunkSize = 64 << 10

// MaxScan 单次最多向前扫描的字节数（筛选条件很严时避免读完整个大文件）。
const MaxScan = 64 << 20

// Result 是一次读取的结果：行按从新到旧排列。
type Result struct {
	Lines []string
	// Next 继续向前读的位置（0 表示已到文件开头）
	Next int64
	// Size 文件大小
	Size int64
}

// Tail 从 before（0 为文件末尾）向前读取最多 limit 条满足 match 的行（match 为 nil 时不筛选）。
func Tail(path string, before int64, limit int, match func(line []byte) bool) (*Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := st.Size()
	end := size
	if before > 0 && before < size {
		end = before
	}
	out := &Result{Size: size}
	var carry []byte // 当前块之后、尚未结束的行片段（位于 end 之前）
	pos := end
	scanned := int64(0)
	for pos > 0 && len(out.Lines) < limit && scanned < MaxScan {
		n := int64(chunkSize)
		if pos < n {
			n = pos
		}
		pos -= n
		scanned += n
		buf := make([]byte, n)
		if _, err := f.ReadAt(buf, pos); err != nil && err != io.EOF {
			return nil, err
		}
		buf = append(buf, carry...)
		// 从后往前切行；第一段（块开头到第一个换行）可能不完整，留到下一块
		for len(out.Lines) < limit {
			i := bytes.LastIndexByte(buf, '\n')
			if i < 0 {
				break
			}
			line := bytes.TrimRight(buf[i+1:], "\r")
			buf = buf[:i]
			if len(line) == 0 {
				continue
			}
			if match == nil || match(line) {
				out.Lines = append(out.Lines, string(line))
				if len(out.Lines) == limit {
					out.Next = pos + int64(len(buf))
					return out, nil
				}
			}
		}
		carry = buf
	}
	if pos == 0 && len(out.Lines) < limit {
		line := bytes.TrimRight(carry, "\r\n")
		if len(line) > 0 && (match == nil || match(line)) {
			out.Lines = append(out.Lines, string(line))
		}
		out.Next = 0
		return out, nil
	}
	out.Next = pos + int64(len(carry))
	return out, nil
}
