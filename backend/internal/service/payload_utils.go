package service

import (
	"bytes"
	"sync"
	"unicode/utf8"
)

// streamPayloadBufPool 流式响应采样缓冲池
var streamPayloadBufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 32*1024))
	},
}

// TruncateBytesWithFlag 截断字节切片，确保不在 UTF-8 多字节字符中间截断
func TruncateBytesWithFlag(data []byte, maxSize int64) ([]byte, bool) {
	if maxSize <= 0 || int64(len(data)) <= maxSize {
		return data, false
	}
	truncated := data[:maxSize]
	for len(truncated) > 0 && !utf8.Valid(truncated) {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated, true
}

// stringPtrFromBytes 将 []byte 转为 *string，nil/空返回 nil
func stringPtrFromBytes(data []byte) *string {
	if len(data) == 0 {
		return nil
	}
	s := string(data)
	return &s
}

// trimInvalidUTF8Tail 修剪尾部不完整的 UTF-8 多字节序列。
// 流式采样可能恰好截在多字节字符中间，用此函数确保结果是合法 UTF-8。
func trimInvalidUTF8Tail(data []byte) []byte {
	if utf8.Valid(data) {
		return data
	}
	for i := len(data) - 1; i >= 0 && i >= len(data)-utf8.UTFMax; i-- {
		if utf8.Valid(data[:i]) {
			return data[:i]
		}
	}
	return data
}
