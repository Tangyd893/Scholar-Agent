// Gateway — ScholarAgent HTTP 入口服务
//
// 职责：
//   - 静态资源托管（Phase 2 前端）
//   - REST API（会话管理）
//   - SSE 流式推送（/api/v1/chat/stream）
//   - gRPC 客户端 → Agent Core
//   - 设备身份 Cookie 管理（device_id）
//
// Phase 1 最小范围：health 端点
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// GET /health — 健康检查
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	addr := ":" + port
	fmt.Printf("gateway listening on %s\n", addr)
	slog.Info("gateway started", "port", port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("gateway failed", "error", err)
		os.Exit(1)
	}
}
