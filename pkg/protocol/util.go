package protocol

import (
	"fmt"
	"obsync/pkg/fsops"
	"time"
)

// ExtractFileInfo 从 map[string]interface{} 提取 FileInfo
func ExtractFileInfo(raw interface{}) (fsops.FileInfo, error) {
	var info fsops.FileInfo

	// 第1层：断言为 map[string]interface{}
	data, ok := raw.(map[string]interface{})
	if !ok {
		return info, fmt.Errorf("value is not map[string]interface{}, got %T", raw)
	}

	// 第2层：提取 FileHash（断言为 string）
	fileHashRaw, exists := data["FileHash"]
	if !exists {
		return info, fmt.Errorf("missing 'FileHash' field")
	}
	fileHash, ok := fileHashRaw.(string)
	if !ok {
		return info, fmt.Errorf("FileHash is not string, got %T", fileHashRaw)
	}
	info.FileHash = fileHash

	// 第3层：提取 ModTime（字符串 → time.Time）
	modTimeRaw, exists := data["ModTime"]
	if !exists {
		return info, fmt.Errorf("missing 'ModTime' field")
	}
	modTimeStr, ok := modTimeRaw.(string)
	if !ok {
		return info, fmt.Errorf("ModTime is not string, got %T", modTimeRaw)
	}
	modTime, err := time.Parse(time.RFC3339Nano, modTimeStr)
	if err != nil {
		return info, fmt.Errorf("invalid ModTime format: %v", err)
	}
	info.ModTime = modTime

	return info, nil
}
