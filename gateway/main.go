// Gateway — ScholarAgent HTTP 入口服务
//
// 职责：
//   - 静态资源托管（Phase 2 前端）
//   - REST API（会话管理）
//   - SSE 流式推送（/api/v1/chat/stream）
//   - 反向代理 gRPC → Agent Core
//   - 设备身份 Cookie 管理（device_id）
//
// Phase 1 最小范围：health + 转发 gRPC（CLI 可直接连 agent-core）
package main

import "fmt"

func main() {
	fmt.Println("gateway starting on :8080...")
	// TODO: 初始化 Gin router、gRPC client、Redis、启动 HTTP server
}
