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
