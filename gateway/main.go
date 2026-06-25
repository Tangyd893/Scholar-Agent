// Gateway — ScholarAgent HTTP 入口服务
//
// 职责：
//   - REST API（会话管理）
//   - SSE 流式推送（POST /api/v1/chat/stream）
//   - gRPC 客户端 → Agent Core
//   - 设备身份 Cookie 管理（device_id）
//   - 静态资源托管（Phase 2 前端）
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Tangyd893/Scholar-Agent/proto/gen/agent"
)

func main() {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	agentAddr := os.Getenv("AGENT_CORE_GRPC_ADDR")
	if agentAddr == "" {
		agentAddr = "localhost:50051"
	}

	// 连接 Agent Core gRPC
	conn, err := grpc.NewClient(agentAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("gateway: failed to connect agent-core", "addr", agentAddr, "error", err)
		os.Exit(1)
	}
	defer conn.Close()
	agentClient := pb.NewAgentCoreClient(conn)

	mux := http.NewServeMux()

	// =====================================================================
	// GET /health
	// =====================================================================
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// =====================================================================
	// POST /api/v1/chat/stream — SSE 流式问答
	// =====================================================================
	mux.HandleFunc("POST /api/v1/chat/stream", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SessionID string `json:"session_id"`
			Query     string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":{"code":"INVALID_REQUEST","message":"invalid JSON"}}`, http.StatusBadRequest)
			return
		}
		if req.Query == "" {
			http.Error(w, `{"error":{"code":"INVALID_REQUEST","message":"query is required"}}`, http.StatusBadRequest)
			return
		}
		if req.SessionID == "" {
			req.SessionID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
		}

		// SSE 响应头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			slog.Error("gateway: streaming not supported")
			return
		}

		// 调用 Agent Core gRPC
		stream, err := agentClient.Run(r.Context(), &pb.RunRequest{
			SessionId: req.SessionID,
			Query:     req.Query,
		})
		if err != nil {
			writeSSE(w, "error", fmt.Sprintf(`{"type":"error","content":"gRPC call failed: %v"}`, err))
			flusher.Flush()
			return
		}

		// 转发 StepEvent 流
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				writeSSE(w, "error", fmt.Sprintf(`{"type":"error","content":"stream error: %v"}`, err))
				flusher.Flush()
				return
			}

			data, _ := json.Marshal(map[string]interface{}{
				"type":         event.Type,
				"content":      event.Content,
				"step":         event.Step,
				"tool_args":    event.ToolArgsJson,
				"timestamp":    event.Timestamp.AsTime().Format(time.RFC3339),
			})
			writeSSE(w, "", string(data))
			flusher.Flush()
		}
	})

	// =====================================================================
	// POST /api/v1/sessions — 创建会话
	// =====================================================================
	mux.HandleFunc("POST /api/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Title string `json:"title"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session_id": sessionID,
			"title":      req.Title,
			"created_at": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// =====================================================================
	// GET /api/v1/sessions — 列出会话
	// =====================================================================
	mux.HandleFunc("GET /api/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": []interface{}{},
		})
	})

	// =====================================================================
	// 启动
	// =====================================================================
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	addr := ":" + port
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		fmt.Printf("gateway listening on %s\n", addr)
		slog.Info("gateway started", "port", port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("gateway failed", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("gateway shutting down...")
	srv.Shutdown(context.Background())
}

func writeSSE(w io.Writer, event, data string) {
	if event != "" {
		fmt.Fprintf(w, "event: %s\n", event)
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
}
