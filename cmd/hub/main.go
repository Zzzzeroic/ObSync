// cmd/hub/main.go
package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	"obsync/pkg/fsops"
	"obsync/pkg/protocol"
)

// Hub 是服务器核心
type Hub struct {
	store  *fsops.Store        // 文件存储
	conns  map[string]net.Conn // 客户端连接池
	router map[protocol.Op]func(*protocol.Message) *protocol.Message
	mu     sync.RWMutex
}

func NewHub(repoPath string) *Hub {
	store := fsops.NewStore(repoPath)
	h := &Hub{
		store: store,
		conns: make(map[string]net.Conn),
	}
	h.router = map[protocol.Op]func(*protocol.Message) *protocol.Message{
		protocol.OpDiffReq:     h.handleDiff,
		protocol.OpUploadReq:   h.handleUpload,
		protocol.OpDownloadReq: h.handleDownload,
	}
	return h
}

// Listen 启动监听
func (h *Hub) Listen(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("Hub listening on %s, repo=%s", addr, h.store.Root)
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go h.handleConn(conn)
	}
}

// handleConn 处理单连接
func (h *Hub) handleConn(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)

	for {
		line, _, err := r.ReadLine()
		if err != nil {
			log.Println("Client disconnected:", err)
			return
		}

		req, err := protocol.Decode(line)
		if err != nil {
			log.Println("Decode error:", err)
			continue
		}

		// 记录客户端 ID 到连接映射
		h.mu.Lock()
		h.conns[req.From] = c
		h.mu.Unlock()

		// 路由到对应处理器
		handler, ok := h.router[req.Op]
		if !ok {
			continue
		}
		resp := handler(req)
		resp.ID = req.ID
		resp.Op = req.Op

		// 响应客户端
		if data, err := resp.Encode(); err == nil {
			c.Write(data)
		}
	}
}

// handleDiff 处理差异查询
func (h *Hub) handleDiff(req *protocol.Message) *protocol.Message {
	// 1. 解析客户端发来的 Payload（客户端文件列表）
	rawClientFiles, ok := req.Payload.(map[string]interface{})
	if !ok {
		return &protocol.Message{Error: "invalid client payload"}
	}
	clientFiles := make(map[string]fsops.FileInfo)
	for path, rawInfo := range rawClientFiles {
		info, err := protocol.ExtractFileInfo(rawInfo)
		if err != nil {
			log.Printf("跳过文件 %s: %v", path, err)
			continue
		}
		clientFiles[path] = info
	}

	// 2. 扫描服务器仓库
	serverFiles, err := h.store.ScanAll()
	if err != nil {
		return &protocol.Message{Error: err.Error()}
	}

	// 3. 计算差异
	download := []string{} // 客户端缺or旧，服务器有
	upload := []string{}   // 客户端有，服务器缺or旧

	for serverPath, serverFileInfo := range serverFiles {
		clientFileInfo, exists := clientFiles[serverPath]
		fmt.Println("serverPath:\n", serverPath)
		fmt.Println("clientFile[path]:\n", clientFiles[serverPath])
		fmt.Println("clientFileInfo:\n", clientFileInfo)
		fmt.Println("serverFile:\n", serverFileInfo)
		if !exists {
			download = append(download, serverPath)
		} else if clientFileInfo.FileHash != serverFileInfo.FileHash {
			// diff 处理两者修改时间，优先采取修改时间最近的
			if serverFileInfo.ModTime.After(clientFileInfo.ModTime) {
				upload = append(upload, serverPath) //客户端文件修改时间更新，上传客户端文件
			} else {
				download = append(download, serverPath)
			}
		}
	}

	for clientPath := range clientFiles {
		if _, exists := serverFiles[clientPath]; !exists {
			upload = append(upload, clientPath)
		}
	}

	// 4. 返回 DiffPayload
	return &protocol.Message{
		Op: protocol.OpDiffAck,
		Payload: protocol.DiffPayload{
			Download: download,
			Upload:   upload,
		},
	}
}

// handleUpload 处理文件上传
func (h *Hub) handleUpload(req *protocol.Message) *protocol.Message {
	err := h.store.WriteChunk(req.Path, req.Offset, req.Data)
	if err != nil {
		return &protocol.Message{Error: err.Error()}
	}
	// 如果是最后一块，提交文件
	if req.Offset+req.Size >= req.TotalSize {
		if err := h.store.CommitFile(req.Path); err != nil {
			return &protocol.Message{Error: err.Error()}
		}
		// 广播通知其他客户端
		//fmt.Println("broadcast log:\n", req)
		h.broadcast(&protocol.Message{
			Op:   protocol.OpNotify,
			From: req.From,
			Path: req.Path,
		})
	}
	return &protocol.Message{Path: req.Path}
}

// handleDownload 处理文件下载
func (h *Hub) handleDownload(req *protocol.Message) *protocol.Message {
	data, err := h.store.ReadChunk(req.Path, req.Offset, req.Size)
	if err == errors.New("EOF") {
		return &protocol.Message{
			Op:     protocol.OpDownloadAck,
			Path:   req.Path,
			Offset: req.Offset,
			Size:   req.Size,
			Data:   data,
		}
	} else if err != nil {
		return &protocol.Message{Error: err.Error()}
	}
	return &protocol.Message{
		Op:     protocol.OpDownloadAck,
		Path:   req.Path,
		Offset: req.Offset,
		Size:   req.Size,
		Data:   data,
	}
}

// broadcast 广播消息给除来源外的所有客户端
func (h *Hub) broadcast(msg *protocol.Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	data, _ := msg.Encode()
	for id, conn := range h.conns {
		if id != msg.From {
			conn.Write(data)
		}
	}
}

func main() {
	home, _ := os.UserHomeDir()
	repo := filepath.Join(home, "obsync-repo")
	hub := NewHub(repo)

	log.Fatal(hub.Listen(":9527"))

}
