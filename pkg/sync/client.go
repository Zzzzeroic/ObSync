package sync

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"obsync/pkg/fsops"
	"obsync/pkg/protocol"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
)

// Client 是同步客户端核心（抽离自原 cmd/client/main.go）
type Client struct {
	ID      string
	Repo    *fsops.Store
	HubAddr string
	conn    net.Conn
}

// NewClient 创建客户端实例
func NewClient(hubAddr, repoPath, id string) *Client {
	return &Client{
		ID:      id,
		Repo:    fsops.NewStore(repoPath),
		HubAddr: hubAddr,
	}
}

// Run 启动主循环（供外部调用）
func (c *Client) Run() error {
	conn, err := net.Dial("tcp", c.HubAddr)
	if err != nil {
		return err
	}
	c.conn = conn
	defer conn.Close()
	log.Printf("Client %s connected to %s", c.ID, c.HubAddr)

	if err := c.syncOnce(); err != nil {
		return err
	}
	return c.watchAndServe()
}

// cmd/client/main.go
func (c *Client) syncOnce() error {
	// 获取本地文件列表
	localFiles, err := c.Repo.ScanAll()
	if err != nil {
		return err
	}

	// 发送 DiffReq
	req := &protocol.Message{
		ID:      uuid.New().String(),
		Op:      protocol.OpDiffReq,
		From:    c.ID,
		Payload: localFiles,
	}
	data, _ := req.Encode()
	c.conn.Write(data)

	// 等待 DiffAck
	r := bufio.NewReader(c.conn)
	line, _, _ := r.ReadLine()
	resp, _ := protocol.Decode(line)

	// 从 Payload 解析差异列表
	diff, ok := resp.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type")
	}
	// diff["download"] 是需要下载的文件列表
	// diff["upload"]   是需要上传的文件列表（服务器返回）
	for _, path := range diff["download"].([]interface{}) {
		if err := c.downloadFile(path.(string)); err != nil {
			log.Printf("Download %s failed: %v", path, err)
		}
	}
	for _, path := range diff["upload"].([]interface{}) {
		uploadPath := filepath.Join(c.Repo.Root, path.(string))
		if err := c.uploadFile(uploadPath); err != nil {
			log.Printf("upload %s failed: %v", path, err)
		}
	}
	return nil
}

// downloadFile 下载单个文件（简化版：一次读完）
func (c *Client) downloadFile(path string) error {
	req := &protocol.Message{
		ID:     uuid.New().String(),
		Op:     protocol.OpDownloadReq,
		From:   c.ID,
		Path:   path,
		Offset: 0,
		Size:   1024 * 1024, // 1 MB，假设文件小于 1 MB
	}
	data, _ := req.Encode()
	c.conn.Write(data)

	// 等待响应
	r := bufio.NewReader(c.conn)
	line, _, _ := r.ReadLine()
	resp, _ := protocol.Decode(line)

	// 写入本地
	tmpPath := filepath.Join(c.Repo.Root, path) + ".tmp"
	os.MkdirAll(filepath.Dir(tmpPath), 0755)
	os.WriteFile(tmpPath, resp.Data, 0644)
	return c.Repo.CommitFile(path)
}

// uploadFile 上传单个文件（简化版：一次传完）
func (c *Client) uploadFile(fullPath string) error {
	path, _ := filepath.Rel(c.Repo.Root, fullPath)
	data, _ := os.ReadFile(fullPath)

	req := &protocol.Message{
		ID:        uuid.New().String(),
		Op:        protocol.OpUploadReq,
		From:      c.ID,
		Path:      path,
		Offset:    0,
		Size:      int64(len(data)),
		TotalSize: int64(len(data)),
		Data:      data,
	}
	enc, _ := req.Encode()
	c.conn.Write(enc)
	return nil
}

// watchAndServe 监听本地文件 + 接收推送
func (c *Client) watchAndServe() error {
	// 启动 fsnotify 监听
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	allSubDir, _ := c.Repo.ScanAllDir()
	for _, subDir := range allSubDir {
		watcher.Add(string(subDir))
	}

	// 启动接收协程
	go func() {
		r := bufio.NewReader(c.conn)
		for {
			line, _, _ := r.ReadLine()
			resp, _ := protocol.Decode(line)
			if resp.Op == protocol.OpNotify {
				log.Printf("Remote changed: %s", resp.Path)
				c.downloadFile(resp.Path)
			}
		}
	}()

	// 监听本地变更
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("Local changed: %s", event.Name)
				c.uploadFile(event.Name)
			}
		case err := <-watcher.Errors:
			log.Println("Watch error:", err)
		}
	}
}
