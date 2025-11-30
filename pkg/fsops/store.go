// pkg/fsops/store.go
package fsops

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Store 是文件存储抽象
type Store struct {
	Root string // 仓库根目录，如 /home/pi/sync/repo
	mu   sync.Mutex
}

// NewStore 创建存储实例
func NewStore(root string) *Store {
	os.MkdirAll(root, 0755)
	return &Store{Root: root}
}

// HashFile 计算文件 SHA256（快速判断差异）
func (s *Store) HashFile(path string) (string, error) {
	f, err := os.Open(filepath.Join(s.Root, path))
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// WriteChunk 原子写入文件块（offset=0 时创建.tmp，最后 rename）
func (s *Store) WriteChunk(path string, offset int64, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tmpPath := filepath.Join(s.Root, path) + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteAt(data, offset); err != nil {
		return err
	}
	return nil
}

// CommitFile 将 .tmp 重命名为正式文件（原子操作）
func (s *Store) CommitFile(path string) error {
	tmpPath := filepath.Join(s.Root, path) + ".tmp"
	finalPath := filepath.Join(s.Root, path)
	return os.Rename(tmpPath, finalPath)
}

// ReadChunk 读取文件块（支持断点续传）
func (s *Store) ReadChunk(path string, offset, size int64) ([]byte, error) {
	f, err := os.Open(filepath.Join(s.Root, path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := make([]byte, size)
	n, err := f.ReadAt(buf, offset)
	return buf[:n], err
}

// ScanAll 扫描仓库所有文件，返回 路径→哈希 映射
func (s *Store) ScanAll() (map[string]string, error) {
	result := make(map[string]string)
	err := filepath.Walk(s.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(s.Root, path)
		hash, _ := s.HashFile(relPath)
		result[relPath] = hash
		return nil
	})
	return result, err
}
