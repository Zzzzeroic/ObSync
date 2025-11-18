# ObSync — 服务端设计文档

本仓库作为 ObSync 的同步服务端（server-only），客户端同步逻辑由 Obsidian 插件实现。本文档定义服务端的功能边界、API 契约、数据模型、存储选项、安全与部署建议，供服务端实现与 Obsidian 插件开发对齐。

## 目标与边界
- 服务端职责：存储笔记与附件、提供可查询的变更流（changes feed）、处理附件上传、记录版本与冲突元数据、提供认证与管理接口。
- 客户端（Obsidian 插件）职责：文件监控与本地编辑、生成变更并调用服务端 API、在本地进行最终合并或根据服务端冲突指示进行用户交互。
- 设计原则：简单 HTTP JSON API（MVP），易于自托管、可扩展到对象存储、支持预签名上传以降低服务器压力。

## 高层流程（序列概览）
1. 插件向服务端 `POST /v1/register-device` 注册设备并获取 device_token。
2. 插件周期或手动发起同步：
   - 拉取远端变更：`GET /v1/repos/{repo}/changes?since={checkpoint}`。
   - 推送本地变更：`POST /v1/repos/{repo}/changes`（metadata + 内容或附件指向上传 URL）。
   - 对冲突的文件，服务端返回冲突条目；插件展示并由用户在客户端解决后再次推送解决后的变更。

## 推荐 API（REST / JSON）
（下面仅为 MVP 建议，具体实现可扩展为 gRPC）

- POST /v1/register-device
  - 请求: `{ "repo": "my-repo", "device_id": "laptop-1", "display_name": "Alice Laptop" }`
  - 响应: `{ "device_token": "...", "device_id":"..." }`

- GET /v1/repos/{repo}/changes?since={checkpoint}
  - 返回变更列表（按序）：每项包含 change_id, file_uid, op, base_sha, new_sha, metadata, attachments

- POST /v1/repos/{repo}/changes
  - 请求: `{ "device_token": "...", "changes": [ {...}, ... ] }`
  - 每个 change 的结构示例:
    - `{ "change_id": "dev1-0001", "file_uid": "uid-123", "op": "modify", "path": "Notes/todo.md", "base_sha": "sha1", "new_sha": "sha2", "size": 123, "attachment_refs": [ {"name":"img.png","upload_url":"..."} ] }`
  - 响应: 成功/失败与冲突信息（若与已存在变更冲突则返回冲突条目）。

- POST /v1/repos/{repo}/attachments/request-upload
  - 请求: `{ "name": "img.png", "size": 23456, "content_type": "image/png" }`
  - 响应: `{ "upload_url": "https://...", "upload_id":"..." }`（预签名 URL，用于客户端直接上传到对象存储）

- GET /v1/repos/{repo}/file/{file_uid}
  - 获取指定文件的当前内容或元信息。

- GET /v1/repos/{repo}/conflicts?device={device}
  - 列出待处理冲突项。

## 数据模型（服务端视角）
- File object: `file_uid`, `path`（历史路径列表）、`current_sha`, `size`, `last_modified`, `tombstone`。
- Change: `change_id`, `file_uid`, `op`（create/modify/delete/rename）, `base_sha`, `new_sha`, `device_id`, `timestamp`, `attachments`。
- Repo state / checkpoint: 线性序列 id 或基于向量时钟的 checkpoint（MVP 用单调递增的 change sequence）。
- Conflict record: `conflict_id`, `file_uid`, `local_change_id`, `remote_change_id`, `status`（open/resolved/manual），可附加 `merged_result_ref`。

注意：为使客户端能做合并决定，服务端应保留 `base_sha` 与全量内容或可获取内容的引用（例如文件下载 URL 或对象存储位置）。

## 服务端对合并/冲突的角色
- 方案 A（轻量）：服务端只存储变更和元数据，不进行自动合并。若检测到并发修改，标记冲突并返回给客户端，由客户端（插件）负责三方合并并提交解决方案。
- 方案 B（中等）：服务端尝试对文本文件进行三方合并（使用 diff3-like 算法）；合并失败或二进制文件则创建冲突条目并由客户端处理。

推荐：MVP 采用方案 A（简化服务器，实现低耦合），长期可选方案 B 以减轻客户端工作量。

## 附件/大文件策略
- 使用对象存储（S3/MinIO）与预签名 URL：客户端请求上传 URL，直接向对象存储上传，随后提交 attachment 引用到服务端。
- 对于小文件可直接随 change 一并上传（仅限于小于阈值，如 100KB）。

## 认证与授权
- 支持 API Token（device_token）与可选用户登录（OAuth2/JWT）组合。
- 每个设备在注册时获取单独 token；token 可被撤销以阻止设备访问。

## 部署与可用性
- 容器化：提供 `Dockerfile` 与 Kubernetes manifests（Deployment、Service、PVC）。
- 存储后端可配置为：本地文件系统（单机测试），对象存储（S3 / MinIO），或数据库（Postgres）用于元数据。
- 备份策略：对象存储自带持久性；元数据定期导出备份。

## 监控与运维
- 提供 `/healthz`, `/metrics`（Prometheus）与日志（结构化 JSON）。

## 示例交互（简短）
1. 插件向 `/v1/register-device` 注册并得到 `device_token`。
2. 插件请求 `/v1/repos/foo/changes?since=123` 获得远端变更，应用到本地或向用户展示冲突。
3. 对本地变更，插件先请求 `/attachments/request-upload` 如需上传附件，上传完成后把 change（含 attachment refs）POST 到 `/v1/repos/foo/changes`。

## 与 Obsidian 插件的契约（要点）
- 插件负责：文件检测、三方合并 UI（如果采用方案 A）、用户通知、以及将变更以服务端定义的 change 结构发送到服务端。
- 插件应实现重试逻辑、断点续传（对大附件）与本地 rollback 辅助以应对错误。

## 测试与 CI 建议
- 单元测试：API 层、存储适配器（mock S3 或 minio）、认证模块。
- 集成测试：使用 Docker Compose 启动服务端 + MinIO，运行 E2E 用例：注册设备、上传附件、并发提交冲突场景。

## 下一步建议（我可以代为完成）
- 把本文件加入仓库（已完成）。
- scaffold 服务端骨架（`cmd/server/main.go`, `internal/api`, `internal/store` 等），实现简单的 `register-device` 与 `changes` 接口（MVP）。
- 编写一个最小示例的 Obsidian 插件请求样例（仅演示如何调用 API）。

---
如需我继续，我可以：
- 立即 scaffold 服务端最小骨架并实现 `register-device` 与 `POST /v1/repos/{repo}/changes` 的接口（REST + 内存存储，用于快速迭代）。
- 或先询问你对存储偏好的选择（`local FS` / `git` / `s3`），以及是否希望服务器做合并（方案 A 或 B）。

请告诉我下一步偏好。
