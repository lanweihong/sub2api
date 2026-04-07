package repository

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// spoolToTempFile 将 io.Reader 的数据写入临时文件，返回打开的文件句柄和数据大小。
// 调用方负责关闭文件并删除临时文件。
// tmpDir 为空字符串时使用系统默认临时目录。
func spoolToTempFile(body io.Reader, tmpDir string) (tmpFile *os.File, size int64, err error) {
	// 检查临时目录可用磁盘空间（最低要求 100MB）
	if err := checkDiskSpace(tmpDir, 100*1024*1024); err != nil {
		return nil, 0, err
	}

	tmpFile, err = os.CreateTemp(tmpDir, "sub2api-backup-upload-*")
	if err != nil {
		return nil, 0, fmt.Errorf("create temp file: %w", err)
	}

	size, err = io.Copy(tmpFile, body)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return nil, 0, fmt.Errorf("spool to temp file: %w", err)
	}

	// 回到文件开头以供后续读取
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return nil, 0, fmt.Errorf("seek temp file: %w", err)
	}

	return tmpFile, size, nil
}

// checkDiskSpace 检查指定目录的可用磁盘空间是否满足最低要求
func checkDiskSpace(dir string, minBytes uint64) error {
	if dir == "" {
		dir = os.TempDir()
	}
	var stat unix.Statfs_t
	if err := unix.Statfs(dir, &stat); err != nil {
		return fmt.Errorf("check disk space for %s: %w", dir, err)
	}
	available := stat.Bavail * uint64(stat.Bsize)
	if available < minBytes {
		return fmt.Errorf("insufficient disk space in %s: available %d bytes, required at least %d bytes", dir, available, minBytes)
	}
	return nil
}
