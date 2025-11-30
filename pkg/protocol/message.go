package protocol

import "encoding/json"

type Op string

const (
	OpDiffReq     Op = "diff-req"
	OpDiffAck     Op = "diff-ack"
	OpUploadReq   Op = "upload-req"
	OpUploadAck   Op = "upload-ack"
	OpDownloadReq Op = "download-req"
	OpDownloadAck Op = "download-ack"
	OpNotify      Op = "notify"
)

// Message 是所有消息的载体
type Message struct {
	ID        string      `json:"id"`
	Op        Op          `json:"op"`
	From      string      `json:"from"`
	Path      string      `json:"path,omitempty"`
	Offset    int64       `json:"offset,omitempty"`
	Size      int64       `json:"size,omitempty"`
	TotalSize int64       `json:"totalSize,omitempty"`
	Data      []byte      `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Payload   interface{} `json:"payload,omitempty"` // ← 承载 DiffPayload
}

// DiffPayload 是 diff 请求/响应的载体
type DiffPayload struct {
	Download []string `json:"download"` // 需要下载的文件路径列表
	Upload   []string `json:"upload"`   // 需要上传的文件路径列表
}

func (m *Message) Encode() ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func Decode(b []byte) (*Message, error) {
	var m Message
	err := json.Unmarshal(b, &m)
	return &m, err
}
